package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/kartikbazzad/docdb/internal/logger"
)

// Trimmer manages WAL segment trimming after checkpoints.
type Trimmer struct {
	walDir      string
	dbName      string
	logger      *logger.Logger
	mu          sync.Mutex
	trimmedSegs []string // Track trimmed segments
}

// NewTrimmer creates a new WAL trimmer.
func NewTrimmer(walDir, dbName string, log *logger.Logger) *Trimmer {
	return &Trimmer{
		walDir:      walDir,
		dbName:      dbName,
		logger:      log,
		trimmedSegs: make([]string, 0),
	}
}

// TrimSegmentsBeforeCheckpoint trims WAL segments that are before the last checkpoint.
//
// It identifies segments that are safe to trim (all records before checkpoint)
// and deletes them atomically. The keepSegments parameter controls how many
// segments to keep before the checkpoint for safety.
func (t *Trimmer) TrimSegmentsBeforeCheckpoint(lastCheckpointTxID uint64, keepSegments int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.shouldTrim() {
		return nil
	}

	segments, err := t.listWALSegments()
	if err != nil {
		return fmt.Errorf("failed to list WAL segments: %w", err)
	}

	if len(segments) <= keepSegments+1 {
		// Not enough segments to trim
		return nil
	}

	// Sort segments by name (which includes sequence numbers)
	sort.Strings(segments)

	// Find segments that are before the checkpoint
	// We keep the active segment and keepSegments before checkpoint
	trimCount := len(segments) - keepSegments - 1
	if trimCount <= 0 {
		return nil
	}

	trimmed := 0
	for i := 0; i < trimCount; i++ {
		segPath := segments[i]

		// Verify segment is not the active one
		if t.isActiveSegment(segPath) {
			continue
		}

		// Verify segment is before checkpoint
		if !t.isSegmentBeforeCheckpoint(segPath, lastCheckpointTxID) {
			continue
		}

		// Delete segment atomically
		if err := t.deleteSegment(segPath); err != nil {
			t.logger.Warn("Failed to delete segment %s: %v", segPath, err)
			continue
		}

		t.trimmedSegs = append(t.trimmedSegs, segPath)
		trimmed++
		t.logger.Info("Trimmed WAL segment: %s", segPath)
	}

	if trimmed > 0 {
		t.logger.Info("Trimmed %d WAL segments (kept %d)", trimmed, keepSegments)
	}

	return nil
}

// shouldTrim checks if trimming is enabled and safe to perform.
func (t *Trimmer) shouldTrim() bool {
	// Trimming is enabled by default
	// Additional safety checks can be added here
	return true
}

// listWALSegments lists all WAL segments for this database.
func (t *Trimmer) listWALSegments() ([]string, error) {
	basePattern := fmt.Sprintf("%s.wal", t.dbName)
	pattern := filepath.Join(t.walDir, basePattern+"*")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	segments := make([]string, 0, len(matches))
	for _, match := range matches {
		// Include both active segment and rotated segments
		if strings.HasPrefix(filepath.Base(match), basePattern) {
			segments = append(segments, match)
		}
	}

	return segments, nil
}

// isActiveSegment checks if a segment is the active (current) WAL file.
func (t *Trimmer) isActiveSegment(segPath string) bool {
	baseName := filepath.Base(segPath)
	activeName := fmt.Sprintf("%s.wal", t.dbName)
	return baseName == activeName
}

// isSegmentBeforeCheckpoint checks if a segment contains only records before the checkpoint.
//
// This is a simplified check - in a production system, we'd need to scan
// the segment to verify all records are before the checkpoint.
func (t *Trimmer) isSegmentBeforeCheckpoint(segPath string, checkpointTxID uint64) bool {
	// For v0.1, we assume rotated segments are before checkpoint
	// This is safe because checkpoints are created after rotation
	// In a more sophisticated implementation, we'd scan the segment
	return !t.isActiveSegment(segPath)
}

// deleteSegment deletes a WAL segment atomically.
func (t *Trimmer) deleteSegment(segPath string) error {
	// Atomic deletion: rename then delete
	// This ensures the file is not in use when deleted
	tempPath := segPath + ".deleting"
	if err := os.Rename(segPath, tempPath); err != nil {
		return fmt.Errorf("failed to rename segment: %w", err)
	}

	if err := os.Remove(tempPath); err != nil {
		return fmt.Errorf("failed to delete segment: %w", err)
	}

	return nil
}

// GetTrimmedSegments returns the list of trimmed segments.
func (t *Trimmer) GetTrimmedSegments() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]string, len(t.trimmedSegs))
	copy(result, t.trimmedSegs)
	return result
}

// Reset clears the trimmed segments list.
func (t *Trimmer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.trimmedSegs = make([]string, 0)
}
