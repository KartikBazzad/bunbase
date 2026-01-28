package wal

import (
	"sync"

	"github.com/kartikbazzad/docdb/internal/logger"
)

// CheckpointManager tracks checkpoint state for bounded recovery.
//
// Checkpoints are written to WAL every N MB to bound recovery time.
// During recovery, we can start from the last checkpoint instead of
// replaying from the beginning of the WAL.
type CheckpointManager struct {
	mu                  sync.Mutex
	intervalBytes       uint64 // Create checkpoint every N bytes
	autoCreate          bool   // Automatically create checkpoints
	maxCheckpoints      int    // Maximum checkpoints to keep
	logger              *logger.Logger
	lastCheckpoint      uint64 // Last checkpoint transaction ID
	checkpointCount     int    // Number of checkpoints created
	walSizeAtCheckpoint uint64 // WAL size when last checkpoint was created
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager(intervalMB uint64, autoCreate bool, maxCheckpoints int, log *logger.Logger) *CheckpointManager {
	return &CheckpointManager{
		intervalBytes:       intervalMB * 1024 * 1024,
		autoCreate:          autoCreate,
		maxCheckpoints:      maxCheckpoints,
		logger:              log,
		walSizeAtCheckpoint: 0,
	}
}

// ShouldCreateCheckpoint returns true if a checkpoint should be created
// based on WAL size since last checkpoint.
func (cm *CheckpointManager) ShouldCreateCheckpoint(currentWALSize uint64) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.autoCreate || cm.intervalBytes == 0 {
		return false
	}

	// If no checkpoint has been created yet, check if we've exceeded interval
	if cm.walSizeAtCheckpoint == 0 {
		return currentWALSize >= cm.intervalBytes
	}

	// Check if we've written enough since last checkpoint
	sizeSinceLastCheckpoint := currentWALSize - cm.walSizeAtCheckpoint
	return sizeSinceLastCheckpoint >= cm.intervalBytes
}

// RecordCheckpoint records that a checkpoint was created at the given
// transaction ID and WAL size.
func (cm *CheckpointManager) RecordCheckpoint(txID uint64, walSize uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.lastCheckpoint = txID
	cm.walSizeAtCheckpoint = walSize
	cm.checkpointCount++

	cm.logger.Debug("Checkpoint recorded: tx_id=%d, wal_size=%d, count=%d", txID, walSize, cm.checkpointCount)
}

// GetLastCheckpoint returns the last checkpoint transaction ID.
func (cm *CheckpointManager) GetLastCheckpoint() uint64 {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.lastCheckpoint
}

// UpdateSize updates the tracked WAL size for checkpoint interval calculation.
func (cm *CheckpointManager) UpdateSize(walSize uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	// We track incremental size, so we don't need to update here
	// The ShouldCreateCheckpoint method uses currentWALSize directly
}

// Reset resets checkpoint tracking (used during recovery).
func (cm *CheckpointManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.lastCheckpoint = 0
	cm.walSizeAtCheckpoint = 0
	cm.checkpointCount = 0
}
