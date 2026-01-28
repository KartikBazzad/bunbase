package pool

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kartikbazzad/docdb/internal/logger"
)

// ShutdownTimeout is the default timeout for graceful shutdown (30 seconds).
const ShutdownTimeout = 30 * time.Second

// GracefulShutdown handles graceful shutdown of the pool with signal handling.
type GracefulShutdown struct {
	pool         *Pool
	logger       *logger.Logger
	timeout      time.Duration
	shutdownCh   chan os.Signal
	mu           sync.Mutex
	shuttingDown bool
}

// NewGracefulShutdown creates a new graceful shutdown handler.
func NewGracefulShutdown(pool *Pool, log *logger.Logger) *GracefulShutdown {
	return &GracefulShutdown{
		pool:         pool,
		logger:       log,
		timeout:      ShutdownTimeout,
		shutdownCh:   make(chan os.Signal, 1),
		shuttingDown: false,
	}
}

// StartSignalHandling starts listening for shutdown signals (SIGTERM, SIGINT).
func (gs *GracefulShutdown) StartSignalHandling() {
	signal.Notify(gs.shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-gs.shutdownCh
		gs.logger.Info("Received shutdown signal: %v", sig)
		gs.Shutdown()
	}()
}

// Shutdown performs graceful shutdown with timeout.
func (gs *GracefulShutdown) Shutdown() {
	gs.mu.Lock()
	if gs.shuttingDown {
		gs.mu.Unlock()
		return
	}
	gs.shuttingDown = true
	gs.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), gs.timeout)
	defer cancel()

	gs.logger.Info("Starting graceful shutdown (timeout: %v)", gs.timeout)

	// Phase 1: Stop accepting new connections (0-5s)
	gs.pool.stopped = true
	gs.logger.Info("Stopped accepting new requests")

	// Phase 2: Drain queues and wait for in-flight transactions (5-25s)
	drainCtx, drainCancel := context.WithTimeout(ctx, 20*time.Second)
	defer drainCancel()

	if err := gs.drainQueues(drainCtx); err != nil {
		gs.logger.Warn("Queue draining incomplete: %v", err)
	}

	// Phase 3: Final sync and close (25-30s)
	syncCtx, syncCancel := context.WithTimeout(ctx, 5*time.Second)
	defer syncCancel()

	if err := gs.syncAndCloseAll(syncCtx); err != nil {
		gs.logger.Warn("Sync and close incomplete: %v", err)
	}

	gs.logger.Info("Graceful shutdown complete")
}

// drainQueues drains all request queues and waits for in-flight transactions.
func (gs *GracefulShutdown) drainQueues(ctx context.Context) error {
	gs.logger.Info("Draining request queues...")

	// Stop scheduler (this closes queues and stops workers)
	gs.pool.sched.Stop()

	// Wait for workers to finish (with timeout)
	done := make(chan struct{})
	go func() {
		gs.pool.sched.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		gs.logger.Info("All workers finished")
		return nil
	case <-ctx.Done():
		gs.logger.Warn("Timeout waiting for workers to finish")
		return ctx.Err()
	}
}

// syncAndCloseAll syncs all database files and closes them.
func (gs *GracefulShutdown) syncAndCloseAll(ctx context.Context) error {
	gs.logger.Info("Syncing and closing all databases...")

	done := make(chan struct{})
	go func() {
		for _, db := range gs.pool.dbs {
			// Sync WAL and data files
			if db != nil {
				db.Close()
			}
		}
		gs.pool.catalog.Close()
		close(done)
	}()

	select {
	case <-done:
		gs.logger.Info("All databases synced and closed")
		return nil
	case <-ctx.Done():
		gs.logger.Warn("Timeout syncing and closing databases")
		return ctx.Err()
	}
}
