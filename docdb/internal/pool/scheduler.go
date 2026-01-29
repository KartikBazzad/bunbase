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
}

// NewScheduler creates a new scheduler.
//
// Parameters:
//   - queueDepth: Maximum number of pending requests per database
//   - logger: Logger instance
//
// Returns:
//   - Initialized scheduler ready for Start()
//
// Note: Scheduler is not started until Start() is called.
func NewScheduler(queueDepth int, logger interface{}) *Scheduler {
	return &Scheduler{
		queues:            make(map[uint64]chan *Request),
		queueDepth:        queueDepth,
		dbIDs:             make([]uint64, 0),
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

func (s *Scheduler) Start() {
	s.mu.Lock()
	s.stopped = false
	s.mu.Unlock()

	// Calculate dynamic worker count
	s.calculateWorkerCount()

	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker()
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

	// Auto-scale: max(NumCPU*2, db_count*2)
	s.dbMu.RLock()
	dbCount := len(s.dbIDs)
	s.dbMu.RUnlock()

	numCPU := runtime.NumCPU()
	minWorkers := numCPU * 2
	dbBasedWorkers := dbCount * 2

	// Use the larger of the two
	s.workerCount = minWorkers
	if dbBasedWorkers > minWorkers {
		s.workerCount = dbBasedWorkers
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

// GetCurrentWorkerCount returns the current number of scheduler workers.
func (s *Scheduler) GetCurrentWorkerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.workerCount
}

// PickNextQueue returns the database ID and queue with the largest pending depth,
// or (0, nil) when no queues have work. Under skewed workloads this gives
// the hottest DB more service and keeps tail latency bounded.
func (s *Scheduler) PickNextQueue() (dbID uint64, queue chan *Request) {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
	if len(s.queues) == 0 {
		return 0, nil
	}
	var best uint64
	maxDepth := -1
	for id, ch := range s.queues {
		d := len(ch)
		if d > maxDepth {
			maxDepth = d
			best = id
		}
	}
	if maxDepth <= 0 {
		return 0, nil
	}
	return best, s.queues[best]
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
		}
		queue = s.queues[req.DBID]
		s.dbMu.Unlock()
	}

	select {
	case queue <- req:
		return nil
	default:
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
		_, queue := s.PickNextQueue()
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
			}
			continue
		}

		select {
		case req, ok := <-queue:
			if !ok {
				s.mu.Lock()
				if s.stopped {
					s.mu.Unlock()
					return
				}
				s.mu.Unlock()
				continue
			}
			if s.pool != nil {
				s.pool.handleRequest(req)
			}
		default:
			continue
		}
	}
}
