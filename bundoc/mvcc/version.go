// Package mvcc implements Multi-Version Concurrency Control (MVCC) for Bundoc.
//
// It provides:
// - Version Chains: Linked lists of data versions for each record.
// - Snapshots: Consistent views of the database for transactions.
// - Visibility Rules: Logic to determine which version is visible to a transaction.
// - Garbage Collection: Cleanup of old versions that are no longer visible.
package mvcc

import (
	"sync/atomic"
	"time"
)

// Timestamp represents a unique, monotonically increasing point in time.
type Timestamp uint64

// Version represents a single historical state of a record.
// Versions are linked in a reverse-chronological chain (newest first).
type Version struct {
	Timestamp Timestamp // Creation time of this version
	Data      []byte    // The actual data content
	TxnID     uint64    // ID of the transaction that created this version
	Next      *Version  // Pointer to the previous (older) version
}

// VersionManager manages timestamps and version chains
type VersionManager struct {
	currentTimestamp atomic.Uint64
}

// NewVersionManager creates a new version manager
func NewVersionManager() *VersionManager {
	vm := &VersionManager{}
	// Initialize with current Unix nanosecond timestamp
	vm.currentTimestamp.Store(uint64(time.Now().UnixNano()))
	return vm
}

// NewTimestamp generates a new unique timestamp
func (vm *VersionManager) NewTimestamp() Timestamp {
	// Atomically increment and return
	ts := vm.currentTimestamp.Add(1)
	return Timestamp(ts)
}

// GetCurrentTimestamp returns the current timestamp without incrementing
func (vm *VersionManager) GetCurrentTimestamp() Timestamp {
	return Timestamp(vm.currentTimestamp.Load())
}

// CreateVersion creates a new version with the given data
func (vm *VersionManager) CreateVersion(data []byte, txnID uint64) *Version {
	return &Version{
		Timestamp: vm.NewTimestamp(),
		Data:      data,
		TxnID:     txnID,
		Next:      nil,
	}
}

// AddVersion adds a new version to the front of a version chain
func (vm *VersionManager) AddVersion(head *Version, newVersion *Version) *Version {
	newVersion.Next = head
	return newVersion
}

// FindVersion finds the appropriate version for a given snapshot
// Returns the most recent version that is visible to the snapshot
func FindVersion(head *Version, snapshot *Snapshot) *Version {
	current := head

	for current != nil {
		// Check if this version is visible to the snapshot
		if snapshot.IsVisible(current) {
			return current
		}
		current = current.Next
	}

	return nil // No visible version found
}

// GarbageCollect removes old versions that are no longer needed
// Keeps versions that might be visible to active snapshots
func GarbageCollect(head *Version, oldestActiveSnapshot Timestamp) *Version {
	if head == nil {
		return nil
	}

	// Keep the head
	current := head

	// Traverse and remove old versions
	for current.Next != nil {
		// If the next version is older than the oldest active snapshot
		// and we already have a newer committed version, we can remove it
		if current.Next.Timestamp < oldestActiveSnapshot {
			// Skip the next version (remove it from chain)
			current.Next = current.Next.Next
		} else {
			current = current.Next
		}
	}

	return head
}

// CountVersions counts the number of versions in a version chain
func CountVersions(head *Version) int {
	count := 0
	current := head
	for current != nil {
		count++
		current = current.Next
	}
	return count
}

// CopyData creates a deep copy of version data
func CopyData(data []byte) []byte {
	if data == nil {
		return nil
	}
	copy := make([]byte, len(data))
	for i, b := range data {
		copy[i] = b
	}
	return copy
}
