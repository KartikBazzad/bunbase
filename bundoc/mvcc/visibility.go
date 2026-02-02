package mvcc

import (
	"fmt"
	"sync"
	"time"
)

// VisibilityChecker encapsulates the logic for determining which version of data
// should be returned to a transaction based on its snapshot.
type VisibilityChecker struct {
	snapshotMgr *SnapshotManager
	mu          sync.RWMutex
}

// NewVisibilityChecker creates a new visibility checker
func NewVisibilityChecker(sm *SnapshotManager) *VisibilityChecker {
	return &VisibilityChecker{
		snapshotMgr: sm,
	}
}

// CheckVisibility determines if a version is visible to a snapshot
func (vc *VisibilityChecker) CheckVisibility(snapshot *Snapshot, version *Version) bool {
	return snapshot.IsVisible(version)
}

// GetVisibleData retrieves the visible version data for a snapshot
func (vc *VisibilityChecker) GetVisibleData(snapshot *Snapshot, versionChain *Version) ([]byte, error) {
	visibleVersion := snapshot.GetVisibleVersion(versionChain)
	if visibleVersion == nil {
		return nil, fmt.Errorf("no visible version found")
	}
	return visibleVersion.Data, nil
}

// GarbageCollector is a background service that periodically cleans up
// old data versions that are no longer visible to any active snapshot.
//
// Optimized for:
// - Low overhead (background processing).
// - Batch processing (checking oldest active snapshot).
type GarbageCollector struct {
	snapshotMgr *SnapshotManager
	gcInterval  time.Duration
	running     bool
	stopChan    chan struct{}
	mu          sync.Mutex
}

// NewGarbageCollector creates a new garbage collector
func NewGarbageCollector(sm *SnapshotManager, gcInterval time.Duration) *GarbageCollector {
	return &GarbageCollector{
		snapshotMgr: sm,
		gcInterval:  gcInterval,
		running:     false,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the garbage collection background process
func (gc *GarbageCollector) Start() {
	gc.mu.Lock()
	if gc.running {
		gc.mu.Unlock()
		return
	}
	gc.running = true
	gc.mu.Unlock()

	go gc.run()
}

// Stop stops the garbage collection background process
func (gc *GarbageCollector) Stop() {
	gc.mu.Lock()
	if !gc.running {
		gc.mu.Unlock()
		return
	}
	gc.running = false
	gc.mu.Unlock()

	close(gc.stopChan)
}

// run executes the garbage collection loop
func (gc *GarbageCollector) run() {
	ticker := time.NewTicker(gc.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gc.performGC()
		case <-gc.stopChan:
			return
		}
	}
}

// performGC performs a garbage collection cycle
func (gc *GarbageCollector) performGC() {
	// Get the oldest active snapshot
	oldestSnapshot := gc.snapshotMgr.GetOldestActiveSnapshot()

	// In a real implementation, we would iterate through all version chains
	// and call GarbageCollect on each one
	// For now, this is a placeholder that demonstrates the concept

	_ = oldestSnapshot // Would be used to clean up old versions
}

// ManualGC performs a manual garbage collection on a version chain
func (gc *GarbageCollector) ManualGC(versionChain *Version) *Version {
	oldestSnapshot := gc.snapshotMgr.GetOldestActiveSnapshot()
	return GarbageCollect(versionChain, oldestSnapshot)
}

// GetStats returns garbage collection statistics
func (gc *GarbageCollector) GetStats() GCStats {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	return GCStats{
		Running:  gc.running,
		Interval: gc.gcInterval,
	}
}

// GCStats contains garbage collection statistics
type GCStats struct {
	Running  bool
	Interval time.Duration
}
