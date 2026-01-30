// Package pool implements request scheduling for database pool.
//
// Scheduler provides:
//   - Per-DB request queues (depth = QueueDepth)
//   - Round-robin scheduling across databases
//   - Worker pool (configurable number of workers)
//   - Backpressure signaling (queue full error)
//
// Scheduling Algorithm:
//  1. Each database gets its own queue
//  2. Round-robin picks next database to service
//  3. Workers execute requests sequentially per-database
//  4. QueueFull returned if queue at capacity
//
// Fairness Guarantee:
//   - Round-robin ensures all databases get serviced
//   - Per-DB queue prevents starvation of single database
//   - No database can monopolize workers
//
// Thread Safety: Scheduler is thread-safe.
package pool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/panjf2000/ants/v2"
)

// Scheduler manages request distribution and worker pool.
//
// It implements:
//   - Per-database request queues
//   - Round-robin scheduling across databases
//   - Worker pool for parallel execution
//   - Queue depth enforcement (backpressure)
//   - Dynamic worker scaling
//
// Thread Safety: All methods are thread-safe.
type Scheduler struct {
	queues            map[uint64]chan *Request // Per-DB request queues
	queueDepth        int                      // Maximum requests per queue (backpressure)
	currentDB         uint64                   // Current database being serviced (round-robin)
	dbIDs             []uint64                 // List of active database IDs
	dbMu              sync.RWMutex             // Protects dbIDs map and list
	pool              *Pool                    // Parent pool for request handling
	logger            interface{}              // Logger (interface to avoid circular dep)
	stopped           bool                     // True if scheduler is shutting down
	wg                sync.WaitGroup           // Worker pool wait group
	mu                sync.Mutex               // Protects stopped flag
	workerCount       int                      // Current number of workers
	maxWorkers        int                      // Maximum workers (auto-tuning cap)
	configuredWorkers int                      // Configured worker count (0 = auto)
	workerExpiry      time.Duration            // Idle goroutine expiry for ants
	preAlloc          bool                     // Pre-allocate ants worker queue
	antsPool          *ants.Pool               // Ants goroutine pool (nil until Start)

	pickTotalNs uint64 // Total time spent in PickNextQueue (nanoseconds)
	pickCount   uint64 // Number of PickNextQueue calls

	// depths: approximate per-DB queue depth for lock-free PickNextQueue scan.
	// Updated on Enqueue (+1) and when worker pops (-1). Protected by dbMu when creating.
	depths map[uint64]*atomic.Int32

	// maxTotalQueued: global cap on total requests queued across all DBs (0 = disabled).
	// totalQueued: current total; incremented on successful enqueue, decremented when worker pops.
	maxTotalQueued int
	totalQueued    atomic.Int32
}

// NewScheduler creates a new scheduler.
//
// Parameters:
//   - queueDepth: Maximum number of pending requests per database
//   - maxTotalQueued: Global cap on total queued requests across all DBs (0 = disabled)
//   - logger: Logger instance
//
// Returns:
//   - Initialized scheduler ready for Start()
//
// Note: Scheduler is not started until Start() is called.
func NewScheduler(queueDepth, maxTotalQueued int, logger interface{}) *Scheduler {
	return &Scheduler{
		queues:            make(map[uint64]chan *Request),
		queueDepth:        queueDepth,
		maxTotalQueued:    maxTotalQueued,
		dbIDs:             make([]uint64, 0),
		depths:            make(map[uint64]*atomic.Int32),
		logger:            logger,
		stopped:           false,
		workerCount:       0,
		maxWorkers:        256, // Default max
		configuredWorkers: 0,   // 0 = auto-scale
	}
}

func (s *Scheduler) SetPool(pool *Pool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pool = pool
}

func (s *Scheduler) SetWorkerConfig(workerCount int, maxWorkers int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configuredWorkers = workerCount
	if maxWorkers > 0 {
		s.maxWorkers = maxWorkers
	}
}

// SetAntsOptions sets options for the ants goroutine pool (WorkerExpiry, PreAlloc).
// Call before Start(). Defaults: WorkerExpiry=1s, PreAlloc=false.
func (s *Scheduler) SetAntsOptions(workerExpiry time.Duration, preAlloc bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workerExpiry = workerExpiry
	s.preAlloc = preAlloc
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	s.stopped = false
	s.mu.Unlock()

	// Calculate dynamic worker count
	s.calculateWorkerCount()

	expiry := s.workerExpiry
	if expiry <= 0 {
		expiry = time.Second
	}
	opts := []ants.Option{
		ants.WithExpiryDuration(expiry),
		ants.WithPreAlloc(s.preAlloc),
		ants.WithPanicHandler(func(v any) {
			if l, ok := s.logger.(*logger.Logger); ok {
				l.Error("scheduler worker panic: %v", v)
			}
		}),
	}
	antsPool, err := ants.NewPool(s.workerCount, opts...)
	if err != nil {
		// Fallback: start goroutines without ants (e.g. invalid size)
		for i := 0; i < s.workerCount; i++ {
			s.wg.Add(1)
			go s.worker()
		}
		return
	}
	s.antsPool = antsPool
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		_ = s.antsPool.Submit(func() { s.worker() })
	}
}

func (s *Scheduler) calculateWorkerCount() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If worker count is explicitly configured, use it
	if s.configuredWorkers > 0 {
		s.workerCount = s.configuredWorkers
		// Cap at max workers
		if s.workerCount > s.maxWorkers {
			s.workerCount = s.maxWorkers
		}
		return
	}

	// Auto-scale: more workers when many DBs to drain queues faster and avoid collapse
	s.dbMu.RLock()
	dbCount := len(s.dbIDs)
	s.dbMu.RUnlock()

	numCPU := runtime.NumCPU()
	minWorkers := numCPU * 2
	multiplier := 2
	if dbCount > 10 {
		multiplier = 4 // 20 DBs -> 80 workers to reduce queue buildup
	}
	dbBasedWorkers := dbCount * multiplier

	// Use the larger of the two
	s.workerCount = minWorkers
	if dbBasedWorkers > minWorkers {
		s.workerCount = dbBasedWorkers
	}

	// When multi-writer is on (maxWorkers > 1), use a higher minimum so many DBs get enough drain capacity
	// (dbCount is 0 at Start(), so dbBasedWorkers doesn't help until DBs register)
	if s.maxWorkers > 1 && s.workerCount < 32 {
		s.workerCount = 32
	}

	// Cap at max workers
	if s.workerCount > s.maxWorkers {
		s.workerCount = s.maxWorkers
	}

	// Ensure at least 4 workers (minimum for reasonable concurrency)
	if s.workerCount < 4 {
		s.workerCount = 4
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	ap := s.antsPool
	s.stopped = true
	s.mu.Unlock()

	// Close all queues to signal workers to exit
	s.dbMu.Lock()
	for _, queue := range s.queues {
		close(queue)
	}
	s.queues = make(map[uint64]chan *Request)
	s.dbIDs = make([]uint64, 0)
	s.dbMu.Unlock()

	s.wg.Wait()
	if ap != nil {
		_ = ap.ReleaseTimeout(3 * time.Second)
		s.mu.Lock()
		s.antsPool = nil
		s.mu.Unlock()
	}
}

// GetQueueDepthStats returns current queue depth for all databases.
func (s *Scheduler) GetQueueDepthStats() map[uint64]int {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()

	stats := make(map[uint64]int)
	for dbID, queue := range s.queues {
		stats[dbID] = len(queue)
	}
	return stats
}

// GetAvgQueueDepth returns the average queue depth across all databases.
func (s *Scheduler) GetAvgQueueDepth() float64 {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()

	if len(s.queues) == 0 {
		return 0
	}

	total := 0
	for _, queue := range s.queues {
		total += len(queue)
	}
	return float64(total) / float64(len(s.queues))
}

// GetPickStats returns time spent selecting queues (for bottleneck profiling).
func (s *Scheduler) GetPickStats() map[string]interface{} {
	totalNs := atomic.LoadUint64(&s.pickTotalNs)
	count := atomic.LoadUint64(&s.pickCount)
	avgNs := uint64(0)
	if count > 0 {
		avgNs = totalNs / count
	}
	return map[string]interface{}{
		"pick_total_ns": totalNs,
		"pick_count":    count,
		"pick_avg_ns":   avgNs,
	}
}

// GetCurrentWorkerCount returns the current number of scheduler workers.
func (s *Scheduler) GetCurrentWorkerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.workerCount
}

// GetAntsStats returns ants pool metrics when using ants (Running, Waiting, Free, Cap).
// Returns nil when not using ants.
func (s *Scheduler) GetAntsStats() map[string]interface{} {
	s.mu.Lock()
	ap := s.antsPool
	s.mu.Unlock()
	if ap == nil {
		return nil
	}
	return map[string]interface{}{
		"running_workers": ap.Running(),
		"waiting_tasks":   ap.Waiting(),
		"free_workers":   ap.Free(),
		"pool_capacity":  ap.Cap(),
	}
}

// PickNextQueue returns the database ID and queue with the largest pending depth,
// or (0, nil) when no queues have work. Uses per-DB atomic depth counters so the
// hot-path scan does not hold dbMu, reducing contention under many DBs.
func (s *Scheduler) PickNextQueue() (dbID uint64, queue chan *Request) {
	s.dbMu.RLock()
	n := len(s.dbIDs)
	if n == 0 {
		s.dbMu.RUnlock()
		return 0, nil
	}
	dbIDsCopy := make([]uint64, n)
	copy(dbIDsCopy, s.dbIDs)
	s.dbMu.RUnlock()

	var best uint64
	maxDepth := int32(-1)
	for _, id := range dbIDsCopy {
		depth := s.depths[id]
		if depth == nil {
			continue
		}
		d := depth.Load()
		if d > maxDepth {
			maxDepth = d
			best = id
		}
	}
	if maxDepth <= 0 {
		return 0, nil
	}

	s.dbMu.RLock()
	queue = s.queues[best]
	s.dbMu.RUnlock()
	return best, queue
}

// decrementDepth decrements the approximate depth for dbID (call after popping from queue).
func (s *Scheduler) decrementDepth(dbID uint64) {
	s.dbMu.RLock()
	d := s.depths[dbID]
	s.dbMu.RUnlock()
	if d != nil {
		d.Add(-1)
	}
}

func (s *Scheduler) Enqueue(req *Request) error {
	s.mu.Lock()
	stopped := s.stopped
	s.mu.Unlock()

	if stopped {
		return ErrPoolStopped
	}

	s.dbMu.RLock()
	queue, exists := s.queues[req.DBID]
	s.dbMu.RUnlock()

	if !exists {
		s.dbMu.Lock()
		// Check again after acquiring write lock
		if _, exists := s.queues[req.DBID]; !exists {
			// Check if stopped before creating new queue
			s.mu.Lock()
			if s.stopped {
				s.mu.Unlock()
				s.dbMu.Unlock()
				return ErrPoolStopped
			}
			s.mu.Unlock()
			s.queues[req.DBID] = make(chan *Request, s.queueDepth)
			s.dbIDs = append(s.dbIDs, req.DBID)
			s.depths[req.DBID] = &atomic.Int32{}
		}
		queue = s.queues[req.DBID]
		s.dbMu.Unlock()
	}

	// Global backpressure: reserve a slot so total queued does not exceed maxTotalQueued.
	if s.maxTotalQueued > 0 {
		newVal := s.totalQueued.Add(1)
		if newVal > int32(s.maxTotalQueued) {
			s.totalQueued.Add(-1)
			return ErrQueueFull
		}
	}

	select {
	case queue <- req:
		s.dbMu.RLock()
		d := s.depths[req.DBID]
		s.dbMu.RUnlock()
		if d != nil {
			d.Add(1)
		}
		return nil
	default:
		if s.maxTotalQueued > 0 {
			s.totalQueued.Add(-1)
		}
		return ErrQueueFull
	}
}

func (s *Scheduler) worker() {
	defer s.wg.Done()

	for {
		s.mu.Lock()
		stopped := s.stopped
		s.mu.Unlock()

		if stopped {
			return
		}

		// Prefer the queue with the most pending work (fairness under skewed workloads)
		pickStart := time.Now()
		dbID, queue := s.PickNextQueue()
		atomic.AddUint64(&s.pickTotalNs, uint64(time.Since(pickStart).Nanoseconds()))
		atomic.AddUint64(&s.pickCount, 1)
		if queue == nil {
			s.dbMu.RLock()
			empty := len(s.dbIDs) == 0
			s.dbMu.RUnlock()
			if empty {
				// No databases yet; allow stopped check and avoid busy spin
				s.mu.Lock()
				if s.stopped {
					s.mu.Unlock()
					return
				}
				s.mu.Unlock()
				// Yield to avoid tight spinning when no databases exist
				time.Sleep(10 * time.Millisecond)
			} else {
				// No work available, yield to avoid busy-waiting
				time.Sleep(1 * time.Millisecond)
			}
			continue
		}

		// Block on channel read (no default case) to avoid busy-waiting
		req, ok := <-queue
		if !ok {
			s.mu.Lock()
			if s.stopped {
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
			continue
		}
		if s.maxTotalQueued > 0 {
			s.totalQueued.Add(-1)
		}
		s.decrementDepth(dbID)
		if s.pool != nil {
			s.pool.handleRequest(req)
		}
	}
}
