// Package docdb implements worker pool for LogicalDB (v0.4).
//
// WorkerPool manages a fixed number of workers that pull tasks,
// lock partitions, execute, unlock, and send results.
package docdb

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/metrics"
	"github.com/kartikbazzad/docdb/internal/types"
)

// WorkerPool manages workers that execute tasks on partitions.
//
// Workers are NOT bound to partitions. They pull tasks from a queue,
// lock the task's partition, execute, unlock, and send results.
type WorkerPool interface {
	// Submit submits a task to the worker pool.
	Submit(task *Task)

	// Start starts the worker pool.
	Start()

	// Stop stops the worker pool and waits for workers to finish.
	Stop()

	// WorkerCount returns the current number of workers.
	WorkerCount() int
}

// workerPoolImpl implements WorkerPool.
type workerPoolImpl struct {
	mu          sync.Mutex
	taskQueue   chan *Task
	workers     []*worker
	workerCount int
	stopped     bool
	wg          sync.WaitGroup
	logger      *logger.Logger
	db          *LogicalDB // Reference to LogicalDB for partition access
}

// worker represents a single worker goroutine.
type worker struct {
	id        int
	taskQueue chan *Task
	db        *LogicalDB
	logger    *logger.Logger
	wg        *sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewWorkerPool creates a new worker pool for a LogicalDB.
func NewWorkerPool(db *LogicalDB, cfg *config.LogicalDBConfig, log *logger.Logger) WorkerPool {
	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = 1024
	}

	return &workerPoolImpl{
		taskQueue:   make(chan *Task, queueSize),
		workerCount: workerCount,
		logger:      log,
		db:          db,
	}
}

func (wp *workerPoolImpl) Submit(task *Task) {
	wp.mu.Lock()
	stopped := wp.stopped
	wp.mu.Unlock()

	if stopped {
		task.ResultCh <- &Result{
			Status: types.StatusError,
			Error:  ErrPoolStopped,
		}
		return
	}

	select {
	case wp.taskQueue <- task:
		// Record queue depth after submission (Phase C.5)
		wp.recordQueueDepth()
	default:
		// Queue full - send backpressure error
		task.ResultCh <- &Result{
			Status: types.StatusError,
			Error:  ErrQueueFull,
		}
		// Record queue depth even when full
		wp.recordQueueDepth()
	}
}

// recordQueueDepth records the current queue depth for all partitions.
// Since partitions share the worker pool queue, all partitions have the same depth.
func (wp *workerPoolImpl) recordQueueDepth() {
	queueDepth := len(wp.taskQueue)
	// Record for all partitions (they share the same queue)
	if wp.db != nil && wp.db.partitions != nil {
		for _, partition := range wp.db.partitions {
			if partition != nil {
				metrics.SetPartitionQueueDepth(wp.db.dbName, strconv.Itoa(partition.ID()), queueDepth)
			}
		}
	}
}

func (wp *workerPoolImpl) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.stopped || len(wp.workers) > 0 {
		return
	}

	wp.workers = make([]*worker, wp.workerCount)
	for i := 0; i < wp.workerCount; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		w := &worker{
			id:        i,
			taskQueue: wp.taskQueue,
			db:        wp.db,
			logger:    wp.logger,
			wg:        &wp.wg,
			ctx:       ctx,
			cancel:    cancel,
		}
		wp.workers[i] = w
		wp.wg.Add(1)
		go w.run()
	}

	wp.logger.Info("Worker pool started: %d workers", wp.workerCount)
}

func (wp *workerPoolImpl) Stop() {
	wp.mu.Lock()
	wp.stopped = true
	workers := wp.workers
	wp.workers = nil
	close(wp.taskQueue)
	wp.mu.Unlock()

	// Cancel all worker contexts
	for _, w := range workers {
		w.cancel()
	}

	// Wait for all workers to finish
	wp.wg.Wait()

	wp.logger.Info("Worker pool stopped")
}

func (wp *workerPoolImpl) WorkerCount() int {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	return wp.workerCount
}

func (w *worker) run() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case task, ok := <-w.taskQueue:
			if !ok {
				return
			}
			w.executeTask(task)
		}
	}
}

func (w *worker) executeTask(task *Task) {
	// Get partition
	partition := w.db.getPartition(task.PartitionID)
	if partition == nil {
		task.ResultCh <- &Result{
			Status: types.StatusError,
			Error:  ErrInvalidPartition,
		}
		return
	}

	// Reads are lock-free: use snapshot and do not hold partition.mu (spec: "Unlimited readers")
	if task.Op == types.OpRead {
		result := w.db.executeReadOnPartitionLockFree(partition, task.Collection, task.DocID)
		task.ResultCh <- result
		return
	}

	// Writes: lock partition (exactly one writer at a time)
	lockStart := time.Now()
	partition.mu.Lock()
	lockWait := time.Since(lockStart)
	metrics.RecordPartitionLockWait(w.db.Name(), strconv.Itoa(partition.ID()), lockWait)
	checkSingleWriter(partition, task.Op)
	result := w.db.executeOnPartition(partition, task)
	partition.mu.Unlock()

	task.ResultCh <- result
}
