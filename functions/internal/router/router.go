package router

import (
	"fmt"
	"sync"

	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/metadata"
	"github.com/kartikbazzad/bunbase/functions/internal/pool"
	"github.com/kartikbazzad/bunbase/functions/internal/scheduler"
)

var (
	ErrFunctionNotFound = fmt.Errorf("function not found")
	ErrNotDeployed      = fmt.Errorf("function not deployed")
)

// Router routes function invocations to appropriate pools
type Router struct {
	metadata  *metadata.Store
	scheduler *scheduler.Scheduler
	pools     map[string]*pool.WorkerPool // functionID -> pool
	mu        sync.RWMutex
	logger    *logger.Logger
}

// NewRouter creates a new router
func NewRouter(meta *metadata.Store, sched *scheduler.Scheduler, log *logger.Logger) *Router {
	return &Router{
		metadata:  meta,
		scheduler: sched,
		pools:     make(map[string]*pool.WorkerPool),
		logger:    log,
	}
}

// ResolveFunction resolves a function name or ID to a function
func (r *Router) ResolveFunction(nameOrID string) (*metadata.Function, error) {
	// Try as ID first
	fn, err := r.metadata.GetFunctionByID(nameOrID)
	if err == nil {
		return fn, nil
	}

	// Try as name
	fn, err = r.metadata.GetFunctionByName(nameOrID)
	if err == nil {
		return fn, nil
	}

	return nil, ErrFunctionNotFound
}

// GetPool gets the worker pool for a function
func (r *Router) GetPool(functionID string) (*pool.WorkerPool, error) {
	r.mu.RLock()
	p, exists := r.pools[functionID]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no pool found for function %s", functionID)
	}

	return p, nil
}

// RegisterPool registers a worker pool for a function
func (r *Router) RegisterPool(functionID string, p *pool.WorkerPool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pools[functionID]; exists {
		r.logger.Debug("Pool already exists for function %s, replacing", functionID)
	}
	
	r.pools[functionID] = p
	r.scheduler.RegisterPool(functionID, p)
	r.logger.Info("Registered pool for function %s", functionID)
}

// UnregisterPool unregisters a worker pool
func (r *Router) UnregisterPool(functionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.pools, functionID)
	r.scheduler.UnregisterPool(functionID)
	r.logger.Info("Unregistered pool for function %s", functionID)
}

// EnsurePool ensures a pool exists for a function (creates if needed)
func (r *Router) EnsurePool(functionID string, p *pool.WorkerPool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pools[functionID]; !exists {
		r.pools[functionID] = p
		r.scheduler.RegisterPool(functionID, p)
		r.logger.Info("Created pool for function %s", functionID)
	}
}

// Route routes an invocation request to the appropriate pool
func (r *Router) Route(nameOrID string) (*metadata.Function, *pool.WorkerPool, error) {
	// Resolve function
	fn, err := r.ResolveFunction(nameOrID)
	if err != nil {
		return nil, nil, err
	}

	// Check if deployed
	if fn.Status != metadata.FunctionStatusDeployed {
		return nil, nil, ErrNotDeployed
	}

	// Get pool
	p, err := r.GetPool(fn.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("function %s has no active pool", fn.ID)
	}

	return fn, p, nil
}

// ListPools returns all registered pools
func (r *Router) ListPools() map[string]*pool.WorkerPool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*pool.WorkerPool)
	for k, v := range r.pools {
		result[k] = v
	}
	return result
}
