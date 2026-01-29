// Package wal implements per-partition WAL for v0.4.
//
// Each partition has its own WAL file (p{n}.wal) with partition-local LSN.
package wal

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

// PartitionWAL represents a WAL writer for a single partition (v0.4).
//
// Each partition has:
//   - Its own WAL file (p{n}.wal)
//   - Partition-local monotonic LSN
//   - Its own group commit controller
//   - Its own checkpoint manager
type PartitionWAL struct {
	mu            sync.Mutex
	partitionID   int
	file          *os.File
	path          string
	lsn           uint64 // Partition-local monotonic LSN
	size          uint64
	maxSize       uint64
	fsyncConfig   *config.FsyncConfig
	groupCommit   *GroupCommit
	logger        *logger.Logger
	rotator       *Rotator
	isClosed      bool
	checkpointMgr *PartitionCheckpointManager
	retryCtrl     *errors.RetryController
	classifier    *errors.Classifier
	errorTracker  *errors.ErrorTracker
	onFsync       func(duration time.Duration) // Callback for fsync metrics
}

// NewPartitionWAL creates a new partition WAL.
func NewPartitionWAL(partitionID int, path string, maxSize uint64, walCfg *config.WALConfig, log *logger.Logger) *PartitionWAL {
	useFsync := walCfg.FsyncOnCommit
	if walCfg.Fsync.Mode == config.FsyncAlways {
		useFsync = true
	} else if walCfg.Fsync.Mode == config.FsyncNone {
		useFsync = false
	}

	pw := &PartitionWAL{
		partitionID:  partitionID,
		path:         path,
		maxSize:      maxSize,
		fsyncConfig:  &walCfg.Fsync,
		logger:       log,
		rotator:      NewRotator(path, maxSize, useFsync, log),
		retryCtrl:    errors.NewRetryController(),
		classifier:   errors.NewClassifier(),
		errorTracker: errors.NewErrorTracker(),
	}

	// Initialize checkpoint manager for this partition
	pw.checkpointMgr = NewPartitionCheckpointManager(partitionID, walCfg.Checkpoint, log)

	return pw
}

// Open opens the partition WAL file.
func (pw *PartitionWAL) Open() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(pw.path), 0755); err != nil {
		return errors.ErrFileOpen
	}

	file, err := os.OpenFile(pw.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.ErrFileOpen
	}

	pw.file = file
	pw.size = getWalSize(file)
	pw.isClosed = false

	// Initialize group commit if configured
	if pw.fsyncConfig != nil && (pw.fsyncConfig.Mode == config.FsyncGroup || pw.fsyncConfig.Mode == config.FsyncInterval) {
		pw.groupCommit = NewGroupCommit(file, pw.fsyncConfig, pw.logger)
		// Set fsync callback if configured
		if pw.onFsync != nil {
			pw.groupCommit.OnFsync = pw.onFsync
		}
		pw.groupCommit.Start()
	}

	// Load last LSN from checkpoint or file
	lastLSN := pw.checkpointMgr.GetLastLSN()
	if lastLSN > 0 {
		pw.lsn = lastLSN
	} else {
		// If no checkpoint, scan file to find last LSN (simplified - full implementation would parse records)
		pw.lsn = 0
	}

	return nil
}

// Write writes a WAL record with v0.4 format (LSN, TxID, Op, DocID, PayloadLen, PayloadCRC, Payload).
func (pw *PartitionWAL) Write(txID uint64, dbID uint64, collection string, docID uint64, opType types.OperationType, payload []byte) error {
	return pw.retryCtrl.Retry(func() error {
		pw.mu.Lock()
		defer pw.mu.Unlock()

		if pw.file == nil {
			err := errors.ErrFileWrite
			category := pw.classifier.Classify(err)
			pw.errorTracker.RecordError(err, category)
			return err
		}

		// Increment LSN (invariant: strictly monotonic)
		prevLSN := pw.lsn
		pw.lsn++
		lsn := pw.lsn
		checkLSNMonotonic(prevLSN, lsn)

		// Encode v0.4 record format
		record, err := EncodeRecordV4(lsn, txID, dbID, collection, docID, opType, payload)
		if err != nil {
			category := pw.classifier.Classify(err)
			pw.errorTracker.RecordError(err, category)
			return err
		}

		// Write via group commit or directly
		if pw.groupCommit != nil {
			if err := pw.groupCommit.Write(record); err != nil {
				err = errors.ErrFileWrite
				category := pw.classifier.Classify(err)
				pw.errorTracker.RecordError(err, category)
				return err
			}
		} else {
			if _, err := pw.file.Write(record); err != nil {
				err = errors.ErrFileWrite
				category := pw.classifier.Classify(err)
				pw.errorTracker.RecordError(err, category)
				return err
			}
			if pw.fsyncConfig != nil && pw.fsyncConfig.Mode == config.FsyncAlways {
				fsyncStart := time.Now()
				if err := pw.file.Sync(); err != nil {
					err = errors.ErrFileWrite
					category := pw.classifier.Classify(err)
					pw.errorTracker.RecordError(err, category)
					return err
				}
				fsyncDuration := time.Since(fsyncStart)
				if pw.onFsync != nil {
					pw.onFsync(fsyncDuration)
				}
			}
		}

		pw.size += uint64(len(record))

		// Check rotation
		if pw.maxSize > 0 && pw.size >= pw.maxSize {
			if err := pw.rotate(); err != nil {
				pw.logger.Warn("Failed to rotate partition WAL: %v", err)
			}
		}

		return nil
	}, pw.classifier)
}

// rotate rotates the partition WAL file.
func (pw *PartitionWAL) rotate() error {
	if pw.rotator == nil {
		return nil
	}

	newPath, err := pw.rotator.Rotate()
	if err != nil {
		return err
	}

	// Close old file
	if pw.file != nil {
		pw.file.Close()
	}

	// Open new file
	file, err := os.OpenFile(newPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	pw.file = file
	pw.path = newPath
	pw.size = 0

	// Reinitialize group commit if needed
	if pw.groupCommit != nil {
		pw.groupCommit.Stop()
		pw.groupCommit = NewGroupCommit(file, pw.fsyncConfig, pw.logger)
		if pw.onFsync != nil {
			pw.groupCommit.OnFsync = pw.onFsync
		}
		pw.groupCommit.Start()
	}

	return nil
}

// Close closes the partition WAL. Syncs the file before closing so replay after reopen sees all data.
func (pw *PartitionWAL) Close() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.isClosed {
		return nil
	}

	if pw.groupCommit != nil {
		pw.groupCommit.Stop()
		pw.groupCommit = nil
	}

	if pw.file != nil {
		_ = pw.file.Sync()
		pw.file.Close()
		pw.file = nil
	}

	pw.isClosed = true
	return nil
}

// Size returns the current WAL size.
func (pw *PartitionWAL) Size() uint64 {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.size
}

// CurrentLSN returns the current LSN.
func (pw *PartitionWAL) CurrentLSN() uint64 {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.lsn
}

// GetCheckpointManager returns the checkpoint manager for this partition.
func (pw *PartitionWAL) GetCheckpointManager() *PartitionCheckpointManager {
	return pw.checkpointMgr
}

// SetFsyncCallback sets the callback function to be called after each fsync with the duration.
func (pw *PartitionWAL) SetFsyncCallback(callback func(duration time.Duration)) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.onFsync = callback
	if pw.groupCommit != nil {
		pw.groupCommit.OnFsync = callback
	}
}

// SetNextLSN sets the next LSN to use (e.g. after recovery).
// The next Write() will use nextLSN; typically call with max replayed LSN so the next write uses max+1.
func (pw *PartitionWAL) SetNextLSN(nextLSN uint64) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.lsn = nextLSN
}

// PartitionCheckpointManager manages checkpoints for a single partition.
type PartitionCheckpointManager struct {
	mu                  sync.Mutex
	partitionID         int
	checkpointPath      string
	intervalBytes       uint64
	autoCreate          bool
	maxCheckpoints      int
	logger              *logger.Logger
	lastLSN             uint64
	checkpointCount     int
	walSizeAtCheckpoint uint64
}

// NewPartitionCheckpointManager creates a new partition checkpoint manager.
func NewPartitionCheckpointManager(partitionID int, cfg config.CheckpointConfig, log *logger.Logger) *PartitionCheckpointManager {
	return &PartitionCheckpointManager{
		partitionID:         partitionID,
		checkpointPath:      "", // Will be set when checkpoint directory is known
		intervalBytes:       cfg.IntervalMB * 1024 * 1024,
		autoCreate:          cfg.AutoCreate,
		maxCheckpoints:      cfg.MaxCheckpoints,
		logger:              log,
		walSizeAtCheckpoint: 0,
	}
}

// SetCheckpointPath sets the checkpoint directory path.
func (pcm *PartitionCheckpointManager) SetCheckpointPath(checkpointDir string) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	pcm.checkpointPath = filepath.Join(checkpointDir, fmt.Sprintf("p%d.chk", pcm.partitionID))
}

// ShouldCreateCheckpoint returns true if a checkpoint should be created.
func (pcm *PartitionCheckpointManager) ShouldCreateCheckpoint(currentWALSize uint64) bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	if !pcm.autoCreate || pcm.intervalBytes == 0 {
		return false
	}

	if pcm.walSizeAtCheckpoint == 0 {
		return currentWALSize >= pcm.intervalBytes
	}

	sizeSinceLastCheckpoint := currentWALSize - pcm.walSizeAtCheckpoint
	return sizeSinceLastCheckpoint >= pcm.intervalBytes
}

// WriteCheckpoint writes a checkpoint for this partition.
func (pcm *PartitionCheckpointManager) WriteCheckpoint(lsn uint64, walSize uint64) error {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	if pcm.checkpointPath == "" {
		return fmt.Errorf("checkpoint path not set")
	}

	// Write checkpoint file (simplified - full implementation would write structured data)
	// Format: LSN (8 bytes) | WALSize (8 bytes)
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data[0:], lsn)
	binary.LittleEndian.PutUint64(data[8:], walSize)

	if err := os.WriteFile(pcm.checkpointPath, data, 0644); err != nil {
		return err
	}

	pcm.lastLSN = lsn
	pcm.walSizeAtCheckpoint = walSize
	pcm.checkpointCount++

	pcm.logger.Debug("Partition checkpoint written: partition=%d, lsn=%d, wal_size=%d", pcm.partitionID, lsn, walSize)
	return nil
}

// GetLastLSN returns the last checkpointed LSN.
func (pcm *PartitionCheckpointManager) GetLastLSN() uint64 {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	if pcm.checkpointPath == "" {
		return 0
	}

	// Read checkpoint file
	data, err := os.ReadFile(pcm.checkpointPath)
	if err != nil {
		return 0
	}

	if len(data) < 8 {
		return 0
	}

	return binary.LittleEndian.Uint64(data[0:])
}
