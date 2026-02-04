package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logstore"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/worker"
)

var (
	ErrPoolStopped       = fmt.Errorf("pool is stopped")
	ErrMaxWorkersReached = fmt.Errorf("max workers reached")
	ErrNoWorkers         = fmt.Errorf("no workers available")
)

// WorkerPool manages workers for a function version
type WorkerPool struct {
	functionID    string
	version       string
	bundlePath    string
	warm          []worker.Worker
	busy          []worker.Worker
	maxWorkers    int
	warmWorkers   int
	idleTimeout   time.Duration
	mu            sync.RWMutex
	logger        *logger.Logger
	cfg           *config.WorkerConfig
	workerScript  string
	initScript    string
	env           map[string]string
	stopped       bool
	cleanupTicker *time.Ticker
	cleanupStop   chan struct{}
	logStore      logstore.Store // optional; when set, workers persist logs here
}

// NewPool creates a new worker pool
func NewPool(functionID, version, bundlePath string, cfg *config.WorkerConfig, workerScript string, initScript string, env map[string]string, log *logger.Logger) *WorkerPool {
	p := &WorkerPool{
		functionID:   functionID,
		version:      version,
		bundlePath:   bundlePath,
		maxWorkers:   cfg.MaxWorkersPerFunction,
		warmWorkers:  cfg.WarmWorkersPerFunction,
		idleTimeout:  cfg.IdleTimeout,
		logger:       log,
		cfg:          cfg,
		workerScript: workerScript,
		initScript:   initScript,
		env:          env,
		warm:         make([]worker.Worker, 0),
		busy:         make([]worker.Worker, 0),
		cleanupStop:  make(chan struct{}),
	}

	// Start cleanup goroutine
	p.cleanupTicker = time.NewTicker(30 * time.Second)
	go p.cleanupIdleWorkers()

	return p
}

// SetLogStore sets the log store for persisting function logs. Optional; call after NewPool to enable.
func (p *WorkerPool) SetLogStore(store logstore.Store) {
	p.logStore = store
}

// createWorker creates a new worker instance based on runtime configuration
func (p *WorkerPool) createWorker() worker.Worker {
	// Determine runtime from config (default to "bun" for backward compatibility)
	runtime := "bun"
	if p.cfg != nil && p.cfg.Runtime != "" {
		runtime = p.cfg.Runtime
	}

	var w worker.Worker
	switch runtime {
	case "quickjs", "quickjs-ng":
		w = worker.NewQuickJSWorker(p.functionID, p.version, p.bundlePath, p.logger)
		if qw, ok := w.(*worker.QuickJSWorker); ok {
			if p.cfg != nil && p.cfg.Capabilities != nil {
				qw.SetCapabilities(p.cfg.Capabilities)
			}
			if p.logStore != nil {
				qw.SetLogStore(p.logStore)
			}
		}
		return w
	case "bun":
		fallthrough
	default:
		return worker.NewBunWorker(p.functionID, p.version, p.bundlePath, p.logger)
	}
}

// Acquire gets a warm worker or spawns a new one
func (p *WorkerPool) Acquire(ctx context.Context) (worker.Worker, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return nil, ErrPoolStopped
	}

	// Try to get a warm worker
	if len(p.warm) > 0 {
		w := p.warm[0]
		p.warm = p.warm[1:]
		p.busy = append(p.busy, w)
		p.logger.Debug("Acquired warm worker %s for function %s", w.GetID(), p.functionID)
		return w, nil
	}

	// Check if we can spawn a new worker
	totalWorkers := len(p.warm) + len(p.busy)
	if totalWorkers >= p.maxWorkers {
		return nil, ErrMaxWorkersReached
	}

	// Spawn new worker
	p.logger.Info("Spawning new worker for function %s (cold start)", p.functionID)
	w := p.createWorker()
	if err := w.Spawn(p.cfg, p.workerScript, p.initScript, p.env); err != nil {
		p.logger.Error("Failed to spawn worker for function %s: %v", p.functionID, err)
		return nil, fmt.Errorf("failed to spawn worker: %w", err)
	}

	p.busy = append(p.busy, w)
	p.logger.Info("Successfully spawned worker %s for function %s", w.GetID(), p.functionID)
	return w, nil
}

// Release returns a worker to the warm pool
func (p *WorkerPool) Release(w worker.Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find worker in busy list
	for i, bw := range p.busy {
		if bw.GetID() == w.GetID() {
			// Remove from busy
			p.busy = append(p.busy[:i], p.busy[i+1:]...)

			// Check if worker is still healthy
			if !w.HealthCheck() {
				p.logger.Warn("Worker %s failed health check, terminating", w.GetID())
				w.Terminate()
				return
			}

			// Add to warm pool if we need more warm workers
			if len(p.warm) < p.warmWorkers {
				p.warm = append(p.warm, w)
				p.logger.Debug("Released worker %s to warm pool", w.GetID())
			} else {
				// Too many warm workers, terminate this one
				p.logger.Debug("Too many warm workers, terminating %s", w.GetID())
				w.Terminate()
			}
			return
		}
	}

	p.logger.Warn("Attempted to release worker %s that is not in busy list", w.GetID())
}

// Terminate kills a worker (e.g., on error)
func (p *WorkerPool) TerminateWorker(w worker.Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from busy
	for i, bw := range p.busy {
		if bw.GetID() == w.GetID() {
			p.busy = append(p.busy[:i], p.busy[i+1:]...)
			break
		}
	}

	// Remove from warm
	for i, ww := range p.warm {
		if ww.GetID() == w.GetID() {
			p.warm = append(p.warm[:i], p.warm[i+1:]...)
			break
		}
	}

	w.Terminate()
}

// cleanupIdleWorkers periodically terminates idle workers
func (p *WorkerPool) cleanupIdleWorkers() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.mu.Lock()
			now := time.Now()
			var toTerminate []worker.Worker

			// Check warm workers
			for i := len(p.warm) - 1; i >= 0; i-- {
				w := p.warm[i]
				if now.Sub(w.GetLastUsed()) > p.idleTimeout {
					toTerminate = append(toTerminate, w)
					p.warm = append(p.warm[:i], p.warm[i+1:]...)
				}
			}

			// Terminate idle workers
			for _, w := range toTerminate {
				p.logger.Debug("Terminating idle worker %s", w.GetID())
				w.Terminate()
			}
			p.mu.Unlock()

		case <-p.cleanupStop:
			return
		}
	}
}

// Stop stops the pool and terminates all workers
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}

	p.stopped = true
	p.cleanupTicker.Stop()
	close(p.cleanupStop)

	// Copy workers to avoid holding lock during termination
	warmWorkers := make([]worker.Worker, len(p.warm))
	copy(warmWorkers, p.warm)
	busyWorkers := make([]worker.Worker, len(p.busy))
	copy(busyWorkers, p.busy)

	p.warm = nil
	p.busy = nil
	p.mu.Unlock()

	// Terminate all workers (without holding lock)
	p.logger.Info("Terminating %d warm and %d busy workers for function %s", len(warmWorkers), len(busyWorkers), p.functionID)

	// Terminate workers in parallel with timeout
	done := make(chan bool, len(warmWorkers)+len(busyWorkers))

	for _, w := range warmWorkers {
		go func(w worker.Worker) {
			w.Terminate()
			done <- true
		}(w)
	}
	for _, w := range busyWorkers {
		go func(w worker.Worker) {
			w.Terminate()
			done <- true
		}(w)
	}

	// Wait for all terminations with timeout
	timeout := time.After(3 * time.Second)
	completed := 0
	total := len(warmWorkers) + len(busyWorkers)

	if total == 0 {
		p.logger.Info("No workers to terminate for function %s", p.functionID)
		return
	}

	for completed < total {
		select {
		case <-done:
			completed++
		case <-timeout:
			p.logger.Warn("Worker termination timeout (%d/%d completed), continuing shutdown", completed, total)
			// Don't wait longer - Terminate() has its own timeout
			break
		}
	}

	p.logger.Info("Worker pool stopped for function %s", p.functionID)
}

// GetStats returns pool statistics
func (p *WorkerPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		FunctionID:   p.functionID,
		Version:      p.version,
		WarmWorkers:  len(p.warm),
		BusyWorkers:  len(p.busy),
		MaxWorkers:   p.maxWorkers,
		TotalWorkers: len(p.warm) + len(p.busy),
	}
}

// PoolStats represents pool statistics
type PoolStats struct {
	FunctionID   string
	Version      string
	WarmWorkers  int
	BusyWorkers  int
	MaxWorkers   int
	TotalWorkers int
}
