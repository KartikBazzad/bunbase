package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/pool"
	"github.com/kartikbazzad/bunbase/functions/internal/worker"
)

// InvokeRequest represents a function invocation request
type InvokeRequest struct {
	Method     string
	Path       string
	Headers    map[string]string
	Query      map[string]string
	Body       []byte
	DeadlineMS int64
}

// InvokeResult represents the result of an invocation
type InvokeResult struct {
	Success      bool
	Status       int
	Headers      map[string]string
	Body         []byte
	Error        string
	ExecutionTime time.Duration
	IsColdStart  bool
}

// Invocation represents a pending invocation
type Invocation struct {
	ID      string
	Request *InvokeRequest
	Result  chan *InvokeResult
	Context context.Context
}

// Scheduler manages invocation queue and worker assignment
type Scheduler struct {
	pools    map[string]*pool.WorkerPool // functionID -> pool
	queues   map[string][]*Invocation     // functionID -> queue
	mu       sync.RWMutex
	logger   *logger.Logger
	stopped  bool
}

// NewScheduler creates a new scheduler
func NewScheduler(log *logger.Logger) *Scheduler {
	return &Scheduler{
		pools:  make(map[string]*pool.WorkerPool),
		queues: make(map[string][]*Invocation),
		logger: log,
	}
}

// RegisterPool registers a worker pool for a function
func (s *Scheduler) RegisterPool(functionID string, p *pool.WorkerPool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools[functionID] = p
	s.logger.Info("Registered pool for function %s", functionID)
}

// UnregisterPool unregisters a worker pool
func (s *Scheduler) UnregisterPool(functionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pools, functionID)
	delete(s.queues, functionID)
	s.logger.Info("Unregistered pool for function %s", functionID)
}

// Schedule schedules an invocation
func (s *Scheduler) Schedule(ctx context.Context, functionID string, req *InvokeRequest) (*InvokeResult, error) {
	if s.stopped {
		return nil, fmt.Errorf("scheduler is stopped")
	}

	s.mu.RLock()
	p, exists := s.pools[functionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no pool registered for function %s", functionID)
	}

	// Convert request to worker invoke payload
	invokePayload := &worker.InvokePayload{
		Method:     req.Method,
		Path:       req.Path,
		Headers:    req.Headers,
		Query:      req.Query,
		Body:       worker.EncodeBody(req.Body),
		DeadlineMS: req.DeadlineMS,
	}

	startTime := time.Now()
	isColdStart := false

	// Acquire worker
	s.logger.Debug("Acquiring worker for function %s", functionID)
	w, err := p.Acquire(ctx)
	if err != nil {
		s.logger.Error("Failed to acquire worker for function %s: %v", functionID, err)
		if err == pool.ErrMaxWorkersReached {
			// Queue invocation
			return s.queueInvocation(ctx, functionID, req)
		}
		return &InvokeResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, err
	}

	s.logger.Debug("Acquired worker %s for function %s", w.GetID(), functionID)

	// Check if this was a cold start (no warm workers available before)
	stats := p.GetStats()
	if stats.WarmWorkers == 0 && stats.BusyWorkers == 1 {
		isColdStart = true
		s.logger.Debug("Cold start detected for function %s", functionID)
	}

	// Execute invocation
	s.logger.Debug("Executing invocation on worker %s for function %s", w.GetID(), functionID)
	result, errPayload, err := w.Invoke(ctx, invokePayload)
	s.logger.Debug("Invocation completed on worker %s: success=%v, error=%v", w.GetID(), result != nil && err == nil, err)

	// Release worker
	p.Release(w)

	executionTime := time.Since(startTime)

	if err != nil {
		return &InvokeResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: executionTime,
			IsColdStart:   isColdStart,
		}, err
	}

	if errPayload != nil {
		return &InvokeResult{
			Success:       false,
			Error:         errPayload.Message,
			ExecutionTime: executionTime,
			IsColdStart:   isColdStart,
		}, nil
	}

	// Decode response body
	body, err := worker.DecodeBody(result.Body)
	if err != nil {
		return &InvokeResult{
			Success:       false,
			Error:         fmt.Sprintf("failed to decode response body: %v", err),
			ExecutionTime: executionTime,
			IsColdStart:   isColdStart,
		}, nil
	}

	return &InvokeResult{
		Success:       true,
		Status:        result.Status,
		Headers:       result.Headers,
		Body:          body,
		ExecutionTime: executionTime,
		IsColdStart:   isColdStart,
	}, nil
}

// queueInvocation queues an invocation when workers are busy
func (s *Scheduler) queueInvocation(ctx context.Context, functionID string, req *InvokeRequest) (*InvokeResult, error) {
	invocation := &Invocation{
		ID:      fmt.Sprintf("invoke-%d", time.Now().UnixNano()),
		Request: req,
		Result:  make(chan *InvokeResult, 1),
		Context: ctx,
	}

	s.mu.Lock()
	s.queues[functionID] = append(s.queues[functionID], invocation)
	queueLen := len(s.queues[functionID])
	s.mu.Unlock()

	s.logger.Debug("Queued invocation %s for function %s (queue length: %d)", invocation.ID, functionID, queueLen)

	// Process queue in background
	go s.processQueue(functionID)

	// Wait for result
	select {
	case result := <-invocation.Result:
		return result, nil
	case <-ctx.Done():
		// Remove from queue
		s.mu.Lock()
		queue := s.queues[functionID]
		for i, inv := range queue {
			if inv.ID == invocation.ID {
				s.queues[functionID] = append(queue[:i], queue[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		return nil, ctx.Err()
	}
}

// processQueue processes queued invocations
func (s *Scheduler) processQueue(functionID string) {
	s.mu.RLock()
	_, exists := s.pools[functionID]
	queue := s.queues[functionID]
	s.mu.RUnlock()

	if !exists || len(queue) == 0 {
		return
	}

	// Process invocations one at a time
	for {
		s.mu.Lock()
		if len(queue) == 0 {
			s.mu.Unlock()
			return
		}
		invocation := queue[0]
		queue = queue[1:]
		s.queues[functionID] = queue
		s.mu.Unlock()

		// Execute invocation
		result, err := s.Schedule(invocation.Context, functionID, invocation.Request)
		if err != nil {
			result = &InvokeResult{
				Success: false,
				Error:   err.Error(),
			}
		}

		// Send result
		select {
		case invocation.Result <- result:
		default:
			// Channel closed or receiver gone
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopped = true

	// Stop all pools
	for _, p := range s.pools {
		p.Stop()
	}

	// Clear queues
	s.queues = make(map[string][]*Invocation)
	s.pools = make(map[string]*pool.WorkerPool)
}
