package pool

import (
	"sync"
)

type Scheduler struct {
	queues     map[uint64]chan *Request
	queueDepth int
	currentDB  uint64
	dbIDs      []uint64
	dbMu       sync.RWMutex
	pool       *Pool
	logger     interface{}
	stopped    bool
	wg         sync.WaitGroup
	mu         sync.Mutex
}

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

	s.wg.Wait()
}

func (s *Scheduler) Enqueue(req *Request) error {
	s.dbMu.RLock()
	queue, exists := s.queues[req.DBID]
	s.dbMu.RUnlock()

	if !exists {
		s.dbMu.Lock()
		if _, exists := s.queues[req.DBID]; !exists {
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
			continue
		}

		s.currentDB = (s.currentDB + 1) % uint64(len(s.dbIDs))
		dbID := s.dbIDs[s.currentDB]
		queue := s.queues[dbID]
		s.dbMu.RUnlock()

		select {
		case req := <-queue:
			if s.pool != nil {
				s.pool.handleRequest(req)
			}
		default:
			continue
		}
	}
}
