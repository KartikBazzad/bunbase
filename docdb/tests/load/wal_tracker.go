package load

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WALSample represents a WAL size measurement at a point in time.
type WALSample struct {
	Timestamp time.Time
	SizeBytes uint64
}

// WALTracker tracks WAL size growth over time.
type WALTracker struct {
	mu        sync.RWMutex
	samples   []WALSample
	walDir    string
	dbName    string
	startTime time.Time
}

// NewWALTracker creates a new WAL tracker.
func NewWALTracker(walDir, dbName string) *WALTracker {
	return &WALTracker{
		samples:   make([]WALSample, 0),
		walDir:    walDir,
		dbName:    dbName,
		startTime: time.Now(),
	}
}

// Sample records the current WAL size.
func (wt *WALTracker) Sample() error {
	size, err := wt.getTotalWALSize()
	if err != nil {
		return fmt.Errorf("failed to get WAL size: %w", err)
	}

	wt.mu.Lock()
	defer wt.mu.Unlock()

	wt.samples = append(wt.samples, WALSample{
		Timestamp: time.Now(),
		SizeBytes: size,
	})

	return nil
}

// getTotalWALSize calculates the total WAL size for the partitioned layout.
// Per-database WAL lives under {walDir}/{dbName}/ with per-partition files
// p0.wal, p0.wal.1, p1.wal, etc. We sum sizes of all partition WAL files
// (names starting with "p" and containing ".wal"); subdirectories (e.g. checkpoints) are ignored.
func (wt *WALTracker) getTotalWALSize() (uint64, error) {
	dbWALDir := filepath.Join(wt.walDir, wt.dbName)
	entries, err := os.ReadDir(dbWALDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read WAL dir %s: %w", dbWALDir, err)
	}

	var totalSize uint64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Partition WAL files: p0.wal, p0.wal.1, p1.wal, etc. (prefix "p" and contains ".wal")
		if len(name) < 2 || name[0] != 'p' {
			continue
		}
		if !strings.Contains(name, ".wal") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		totalSize += uint64(info.Size())
	}
	return totalSize, nil
}

// GetSamples returns all WAL size samples.
func (wt *WALTracker) GetSamples() []WALSample {
	wt.mu.RLock()
	defer wt.mu.RUnlock()

	result := make([]WALSample, len(wt.samples))
	copy(result, wt.samples)
	return result
}

// GetGrowthRate calculates the WAL growth rate in bytes per second.
func (wt *WALTracker) GetGrowthRate() (float64, error) {
	wt.mu.RLock()
	defer wt.mu.RUnlock()

	if len(wt.samples) < 2 {
		return 0, fmt.Errorf("need at least 2 samples to calculate growth rate")
	}

	first := wt.samples[0]
	last := wt.samples[len(wt.samples)-1]

	timeDiff := last.Timestamp.Sub(first.Timestamp).Seconds()
	if timeDiff <= 0 {
		return 0, fmt.Errorf("invalid time difference")
	}

	sizeDiff := float64(last.SizeBytes) - float64(first.SizeBytes)
	growthRate := sizeDiff / timeDiff

	return growthRate, nil
}

// GetSummary returns summary statistics about WAL growth.
func (wt *WALTracker) GetSummary() WALGrowthSummary {
	wt.mu.RLock()
	defer wt.mu.RUnlock()

	if len(wt.samples) == 0 {
		return WALGrowthSummary{}
	}

	first := wt.samples[0]
	last := wt.samples[len(wt.samples)-1]

	var maxSize uint64
	for _, sample := range wt.samples {
		if sample.SizeBytes > maxSize {
			maxSize = sample.SizeBytes
		}
	}

	timeDiff := last.Timestamp.Sub(first.Timestamp).Seconds()
	var growthRate float64
	if timeDiff > 0 {
		sizeDiff := float64(last.SizeBytes) - float64(first.SizeBytes)
		growthRate = sizeDiff / timeDiff
	}

	return WALGrowthSummary{
		InitialSizeBytes:      first.SizeBytes,
		FinalSizeBytes:        last.SizeBytes,
		MaxSizeBytes:          maxSize,
		GrowthRateBytesPerSec: growthRate,
		SampleCount:           len(wt.samples),
		DurationSeconds:       timeDiff,
	}
}

// WALGrowthSummary contains summary statistics about WAL growth.
type WALGrowthSummary struct {
	InitialSizeBytes      uint64
	FinalSizeBytes        uint64
	MaxSizeBytes          uint64
	GrowthRateBytesPerSec float64
	SampleCount           int
	DurationSeconds       float64
}

// Reset clears all samples.
func (wt *WALTracker) Reset() {
	wt.mu.Lock()
	defer wt.mu.Unlock()

	wt.samples = wt.samples[:0]
	wt.startTime = time.Now()
}
