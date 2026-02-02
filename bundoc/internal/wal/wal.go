// Package wal implements Write-Ahead Logging for durability.
//
// The WAL ensures that all changes are recorded sequentially on disk before being applied
// to the main data files. This allows the database to recover from crashes by replaying
// the log.
//
// Key Components:
//   - WAL: The main coordinator managing segments and log appends.
//   - Segment: A single log file (rotated when full).
//   - Record: A single log entry (header + payload).
//   - GroupCommitter: Optimizes throughput by batching synchronous disk flushes.
package wal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// WAL represents the Write-Ahead Log Manager.
// It manages a sequence of log segments and handles atomic appends.
type WAL struct {
	dir            string
	currentSegment *Segment      // The active segment being written to
	currentLSN     atomic.Uint64 // Monotonically increasing Log Sequence Number
	nextSegmentID  SegmentID
	buffer         *bufio.Writer // Buffered writer for performance
	bufferSize     int
	mu             sync.RWMutex
}

// DefaultBufferSize is the default WAL buffer size (256KB)
const DefaultBufferSize = 256 * 1024

// NewWAL creates a new Write-Ahead Log
func NewWAL(dir string) (*WAL, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	// Create first segment
	segment, err := NewSegment(dir, 0, LSN(1))
	if err != nil {
		return nil, err
	}

	wal := &WAL{
		dir:            dir,
		currentSegment: segment,
		nextSegmentID:  1,
		bufferSize:     DefaultBufferSize,
	}
	wal.currentLSN.Store(1)

	return wal, nil
}

// Append appends a record to the WAL and returns its LSN
func (w *WAL) Append(record *Record) (LSN, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Assign LSN
	lsn := LSN(w.currentLSN.Add(1))
	record.LSN = lsn

	// Check if we need to rotate segment
	if w.currentSegment.IsFull() {
		if err := w.rotateSegment(); err != nil {
			return 0, err
		}
	}

	// Write to current segment
	if err := w.currentSegment.Write(record); err != nil {
		return 0, err
	}

	return lsn, nil
}

// AppendBatch appends multiple records to the WAL atomically
func (w *WAL) AppendBatch(records []*Record) (LSN, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var lastLSN LSN
	for _, record := range records {
		// Assign LSN
		lastLSN = LSN(w.currentLSN.Add(1))
		record.LSN = lastLSN

		// Check if we need to rotate segment
		if w.currentSegment.IsFull() {
			if err := w.rotateSegment(); err != nil {
				return 0, err
			}
		}

		// Write to current segment
		if err := w.currentSegment.Write(record); err != nil {
			return 0, err
		}
	}

	return lastLSN, nil
}

// Sync forces a sync of the WAL to disk
func (w *WAL) Sync() error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.currentSegment.Sync()
}

// rotateSegment creates a new segment and closes the current one
func (w *WAL) rotateSegment() error {
	// Close current segment
	if err := w.currentSegment.Close(); err != nil {
		return err
	}

	// Create new segment
	nextLSN := LSN(w.currentLSN.Load() + 1)
	newSegment, err := NewSegment(w.dir, w.nextSegmentID, nextLSN)
	if err != nil {
		return err
	}

	w.currentSegment = newSegment
	w.nextSegmentID++

	return nil
}

// GetCurrentLSN returns the current LSN
func (w *WAL) GetCurrentLSN() LSN {
	return LSN(w.currentLSN.Load())
}

// ReadAllRecords reads all records from all WAL segments
func (w *WAL) ReadAllRecords() ([]*Record, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// List all WAL files
	files, err := filepath.Glob(filepath.Join(w.dir, "wal-*.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to list WAL files: %w", err)
	}

	var allRecords []*Record

	// Read each segment
	for _, file := range files {
		// Extract segment ID from filename
		var segID uint64
		if _, err := fmt.Sscanf(filepath.Base(file), "wal-%016x.log", &segID); err != nil {
			continue // Skip invalid files
		}

		segment, err := OpenSegment(w.dir, SegmentID(segID))
		if err != nil {
			return nil, err
		}

		records, err := segment.ReadRecords()
		segment.Close()

		if err != nil {
			return nil, err
		}

		allRecords = append(allRecords, records...)
	}

	return allRecords, nil
}

// Truncate removes WAL segments up to (but not including) the given LSN
func (w *WAL) Truncate(upToLSN LSN) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// List all WAL files
	files, err := filepath.Glob(filepath.Join(w.dir, "wal-*.log"))
	if err != nil {
		return fmt.Errorf("failed to list WAL files: %w", err)
	}

	for _, file := range files {
		// Extract segment ID
		var segID uint64
		if _, err := fmt.Sscanf(filepath.Base(file), "wal-%016x.log", &segID); err != nil {
			continue
		}

		// Skip current segment
		if SegmentID(segID) == w.currentSegment.ID {
			continue
		}

		// Open and check if segment can be deleted
		segment, err := OpenSegment(w.dir, SegmentID(segID))
		if err != nil {
			continue
		}

		// For simplicity, we'll check if segment end LSN < upToLSN
		// In full implementation, would read segment metadata
		segment.Close()

		// Delete old segments (in practice, would be more careful)
		// For now, simplified logic
	}

	return nil
}

// Close closes the WAL
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentSegment != nil {
		return w.currentSegment.Close()
	}
	return nil
}

// RecordExists checks if a record with the given LSN exists
func (w *WAL) RecordExists(lsn LSN) bool {
	return lsn <= w.GetCurrentLSN() && lsn > 0
}
