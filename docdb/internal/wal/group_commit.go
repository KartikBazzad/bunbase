package wal

import (
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/logger"
)

// GroupCommit batches WAL records and performs single fsync per batch.
//
// This dramatically reduces fsync overhead by grouping multiple transactions
// into a single sync operation.
//
// Group commit strategies:
//   - FsyncAlways: fsync on every write (no batching)
//   - FsyncGroup: batch records, flush on timer or size limit (default)
//   - FsyncInterval: flush at fixed time intervals
//   - FsyncNone: never fsync (for benchmarks only)
type GroupCommit struct {
	mu     sync.Mutex
	file   FileHandle
	config *config.FsyncConfig
	logger *logger.Logger
	mode   config.FsyncMode

	buffer     [][]byte
	bufferSize uint64
	batchSize  int

	flushTimer *time.Timer
	stopCh     chan struct{}
	wg         sync.WaitGroup

	stats GroupCommitStats

	// OnFsync callback is called after each fsync with the duration
	OnFsync func(duration time.Duration)
}

// GroupCommitStats tracks group commit performance metrics.
type GroupCommitStats struct {
	TotalBatches    uint64
	TotalRecords    uint64
	AvgBatchSize    float64
	AvgBatchLatency time.Duration
	MaxBatchSize    int
	MaxBatchLatency time.Duration
	LastFlushTime   time.Time
}

// FileHandle abstracts file operations for group commit.
type FileHandle interface {
	Write(p []byte) (n int, err error)
	Sync() error
}

// NewGroupCommit creates a new group commit manager.
//
// Parameters:
//   - file: File handle to write to
//   - cfg: Fsync configuration
//   - logger: Logger instance
//
// Returns:
//   - Initialized GroupCommit manager ready to Start()
func NewGroupCommit(file FileHandle, cfg *config.FsyncConfig, log *logger.Logger) *GroupCommit {
	return &GroupCommit{
		file:       file,
		config:     cfg,
		logger:     log,
		mode:       cfg.Mode,
		buffer:     make([][]byte, 0, cfg.MaxBatchSize),
		batchSize:  cfg.MaxBatchSize,
		flushTimer: time.NewTimer(time.Duration(cfg.IntervalMS) * time.Millisecond),
		stopCh:     make(chan struct{}),
		stats:      GroupCommitStats{},
	}
}

// Start begins the group commit background flusher.
func (gc *GroupCommit) Start() {
	gc.wg.Add(1)
	go gc.flushLoop()
}

// Stop gracefully shuts down the group commit manager.
func (gc *GroupCommit) Stop() {
	close(gc.stopCh)
	gc.flushTimer.Stop()
	gc.wg.Wait()

	// Flush remaining records on shutdown
	gc.mu.Lock()
	if len(gc.buffer) > 0 {
		gc.flushUnsafe()
	}
	gc.mu.Unlock()
}

// Write adds a record to the group commit buffer.
//
// For FsyncGroup mode, records are buffered and flushed periodically.
// For FsyncAlways mode, records are flushed immediately.
// For FsyncNone mode, records are buffered but never synced.
func (gc *GroupCommit) Write(record []byte) error {
	switch gc.mode {
	case config.FsyncAlways:
		// Immediate sync, no batching
		gc.mu.Lock()
		if _, err := gc.file.Write(record); err != nil {
			gc.mu.Unlock()
			return err
		}
		fsyncStart := time.Now()
		if err := gc.file.Sync(); err != nil {
			gc.mu.Unlock()
			return err
		}
		fsyncDuration := time.Since(fsyncStart)
		gc.mu.Unlock()
		if gc.OnFsync != nil {
			gc.OnFsync(fsyncDuration)
		}

		return nil

	case config.FsyncGroup, config.FsyncInterval:
		// Buffer for group commit
		gc.mu.Lock()

		gc.buffer = append(gc.buffer, record)
		gc.bufferSize += uint64(len(record))

		// Check if we should flush immediately
		shouldFlush := false
		if gc.mode == config.FsyncGroup && len(gc.buffer) >= gc.batchSize {
			shouldFlush = true
		}

		gc.mu.Unlock()

		if shouldFlush {
			// Trigger immediate flush
			gc.flushTimer.Reset(0)
		}

		return nil

	case config.FsyncNone:
		// Write without syncing (for benchmarks)
		gc.mu.Lock()
		if _, err := gc.file.Write(record); err != nil {
			gc.mu.Unlock()
			return err
		}
		gc.mu.Unlock()

		return nil

	default:
		return fmt.Errorf("unknown fsync mode: %d", gc.mode)
	}
}

// Sync forces an immediate flush of buffered records.
func (gc *GroupCommit) Sync() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.flushUnsafe()
}

// flushUnsafe performs the actual flush (must hold mu).
func (gc *GroupCommit) flushUnsafe() error {
	if len(gc.buffer) == 0 {
		return nil
	}

	startTime := time.Now()

	// Write all buffered records
	for _, rec := range gc.buffer {
		if _, err := gc.file.Write(rec); err != nil {
			return err
		}
	}

	// Sync based on mode
	if gc.mode == config.FsyncGroup || gc.mode == config.FsyncInterval {
		fsyncStart := time.Now()
		if err := gc.file.Sync(); err != nil {
			return err
		}
		fsyncDuration := time.Since(fsyncStart)
		if gc.OnFsync != nil {
			gc.OnFsync(fsyncDuration)
		}
	}

	// Update stats
	batchSize := len(gc.buffer)
	batchLatency := time.Since(startTime)

	gc.stats.TotalBatches++
	gc.stats.TotalRecords += uint64(batchSize)
	gc.stats.AvgBatchSize = float64(gc.stats.TotalRecords) / float64(gc.stats.TotalBatches)

	if batchSize > gc.stats.MaxBatchSize {
		gc.stats.MaxBatchSize = batchSize
	}
	if batchLatency > gc.stats.MaxBatchLatency {
		gc.stats.MaxBatchLatency = batchLatency
	}

	// Update average latency (exponential moving average)
	if gc.stats.AvgBatchLatency == 0 {
		gc.stats.AvgBatchLatency = batchLatency
	} else {
		alpha := 0.1 // Smoothing factor
		gc.stats.AvgBatchLatency = time.Duration(float64(gc.stats.AvgBatchLatency)*(1-alpha) + float64(batchLatency)*alpha)
	}

	gc.stats.LastFlushTime = time.Now()

	// Clear buffer
	gc.buffer = gc.buffer[:0]
	gc.bufferSize = 0

	return nil
}

// flushLoop is the background goroutine that periodically flushes buffers.
func (gc *GroupCommit) flushLoop() {
	defer gc.wg.Done()

	for {
		select {
		case <-gc.stopCh:
			return

		case <-gc.flushTimer.C:
			gc.mu.Lock()
			gc.flushUnsafe()
			gc.mu.Unlock()

			// Reset timer for next interval
			gc.flushTimer.Reset(time.Duration(gc.config.IntervalMS) * time.Millisecond)
		}
	}
}

// GetStats returns group commit performance statistics.
func (gc *GroupCommit) GetStats() GroupCommitStats {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.stats
}

// ResetStats clears all group commit statistics.
func (gc *GroupCommit) ResetStats() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.stats = GroupCommitStats{}
}
