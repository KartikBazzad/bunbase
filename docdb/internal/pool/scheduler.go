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
	"sync"
)

// Scheduler manages request distribution and worker pool.
//
// It implements:
//   - Per-database request queues
//   - Round-robin scheduling across databases
//   - Worker pool for parallel execution
//   - Queue depth enforcement (backpressure)
//
// Thread Safety: All methods are thread-safe.
type Scheduler struct {
	queues     map[uint64]chan *Request // Per-DB request queues
	queueDepth int                      // Maximum requests per queue (backpressure)
	currentDB  uint64                   // Current database being serviced (round-robin)
	dbIDs      []uint64                 // List of active database IDs
	dbMu       sync.RWMutex             // Protects dbIDs map and list
	pool       *Pool                    // Parent pool for request handling
	logger     interface{}              // Logger (interface to avoid circular dep)
	stopped    bool                     // True if scheduler is shutting down
	wg         sync.WaitGroup           // Worker pool wait group
	mu         sync.Mutex               // Protects stopped flag
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
		queues:     make(map[uint64]chan *Request),
		queueDepth: queueDepth,
		dbIDs:      make([]uint64, 0),
		logger:     logger,
		stopped:    false,
	}
}

func (s *Scheduler) SetPool(pool *Pool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pool = pool
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	s.stopped = false
	s.mu.Unlock()

	for i := 0; i < 4; i++ {
		s.wg.Add(1)
		go s.worker()
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

		s.dbMu.RLock()
		if len(s.dbIDs) == 0 {
			s.dbMu.RUnlock()
			// Small delay to avoid busy-waiting when no databases exist
			// This also allows the stopped flag to be checked more frequently
			s.mu.Lock()
			if s.stopped {
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
			continue
		}

		s.currentDB = (s.currentDB + 1) % uint64(len(s.dbIDs))
		dbID := s.dbIDs[s.currentDB]
		queue := s.queues[dbID]
		s.dbMu.RUnlock()

		select {
		case req, ok := <-queue:
			if !ok {
				// Queue was closed, check if we should exit
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
