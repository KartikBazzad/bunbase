package mvcc

import (
	"sync"
)

// IsolationLevel defines the transaction isolation level
type IsolationLevel int

const (
	ReadUncommitted IsolationLevel = iota // Not recommended, allows dirty reads
	ReadCommitted                         // Default - read only committed data
	RepeatableRead                        // Repeatable reads, prevents phantom reads
	Serializable                          // Full serializability
)

// Snapshot captures the state of the database at a specific point in time.
// It is used to ensure REPEATABLE READ or SNAPSHOT ISOLATION.
type Snapshot struct {
	Timestamp      Timestamp      // The logical time when the snapshot was taken
	MaxTxnID       uint64         // The highest Transaction ID generated at snapshot time
	ActiveTxns     []uint64       // List of transactions that were active (uncommitted) at start
	AbortedTxns    []uint64       // List of transactions known to be aborted at start
	IsolationLevel IsolationLevel // The consistency level required by this snapshot
	mu             sync.RWMutex
}

// SnapshotManager manages database snapshots
type SnapshotManager struct {
	versionMgr      *VersionManager
	activeSnapshots map[Timestamp]*Snapshot
	abortedTxns     map[uint64]bool // Global aborted transactions
	activeTxns      map[uint64]bool
	maxTxnID        uint64 // Highest allocated transaction ID
	mu              sync.RWMutex
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(vm *VersionManager) *SnapshotManager {
	return &SnapshotManager{
		versionMgr:      vm,
		activeSnapshots: make(map[Timestamp]*Snapshot),
		abortedTxns:     make(map[uint64]bool),
		activeTxns:      make(map[uint64]bool),
	}
}

// BeginSnapshot creates a new snapshot with the given isolation level
func (sm *SnapshotManager) BeginSnapshot(txnID uint64, level IsolationLevel) *Snapshot {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update max txn ID
	if txnID > sm.maxTxnID {
		sm.maxTxnID = txnID
	}

	// Get current timestamp
	ts := sm.versionMgr.NewTimestamp()

	// Copy active transactions to slice
	// Pre-allocate to avoid re-allocations
	activeTxns := make([]uint64, 0, len(sm.activeTxns))
	for txn := range sm.activeTxns {
		activeTxns = append(activeTxns, txn)
	}

	// Copy aborted transactions to slice
	abortedTxns := make([]uint64, 0, len(sm.abortedTxns))
	for txn := range sm.abortedTxns {
		abortedTxns = append(abortedTxns, txn)
	}

	// Create snapshot
	snapshot := &Snapshot{
		Timestamp:      ts,
		MaxTxnID:       sm.maxTxnID,
		ActiveTxns:     activeTxns,
		AbortedTxns:    abortedTxns,
		IsolationLevel: level,
	}

	// Register snapshot
	sm.activeSnapshots[ts] = snapshot

	// Mark transaction as active
	sm.activeTxns[txnID] = true

	return snapshot
}

// CommitTransaction marks a transaction as committed
func (sm *SnapshotManager) CommitTransaction(txnID uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Removed from active, not added to aborted -> Implicitly committed
	delete(sm.activeTxns, txnID)
}

// AbortTransaction marks a transaction as aborted
func (sm *SnapshotManager) AbortTransaction(txnID uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.abortedTxns[txnID] = true
	delete(sm.activeTxns, txnID)
}

// ReleaseSnapshot releases a snapshot when it's no longer needed
func (sm *SnapshotManager) ReleaseSnapshot(snapshot *Snapshot) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.activeSnapshots, snapshot.Timestamp)
}

// GetOldestActiveSnapshot returns the timestamp of the oldest active snapshot
// Used for garbage collection
func (sm *SnapshotManager) GetOldestActiveSnapshot() Timestamp {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.activeSnapshots) == 0 {
		return sm.versionMgr.GetCurrentTimestamp()
	}

	oldest := Timestamp(^uint64(0)) // Max uint64
	for ts := range sm.activeSnapshots {
		if ts < oldest {
			oldest = ts
		}
	}

	return oldest
}

// contains checks if a value exists in a slice
// Optimized for small datasets (linear scan is faster than map lookup for small n)
func contains(slice []uint64, val uint64) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// IsVisible checks if a specific data version is visible to this snapshot.
//
// Visibility Rules:
// 1. **Future Versions**: Not visible. (Version.Timestamp > Snapshot.Timestamp)
// 2. **Active Transactions**: Not visible. (TxnID is in ActiveTxns list)
// 3. **Aborted Transactions**: Not visible. (TxnID is in AbortedTxns list)
// 4. **Own Writes**: Implicitly visible (handled by TransactionManager Read-Your-Own-Writes).
// 5. **Committed Past**: Visible.
//
// Consistency Levels:
// - **ReadUncommitted**: Everything is visible (dirty reads allowed).
// - **ReadCommitted/RepeatableRead**: Obeys the standard rules above.
func (s *Snapshot) IsVisible(version *Version) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Version created after snapshot started?
	if version.Timestamp > s.Timestamp {
		return false
	}

	// Note: We also use TxnID for ordering check relative to MaxTxnID
	if version.TxnID > s.MaxTxnID {
		return false
	}

	switch s.IsolationLevel {
	case ReadUncommitted:
		return true

	case ReadCommitted, RepeatableRead, Serializable:
		// 1. Was it active when snapshot started?
		if contains(s.ActiveTxns, version.TxnID) {
			return false
		}
		// 2. Was it aborted?
		if contains(s.AbortedTxns, version.TxnID) {
			return false
		}
		// 3. Implicitly committed
		return true

	default:
		return false
	}
}

// GetVisibleVersion finds the appropriate version for this snapshot
func (s *Snapshot) GetVisibleVersion(head *Version) *Version {
	current := head

	for current != nil {
		if s.IsVisible(current) {
			return current
		}
		current = current.Next
	}

	return nil
}
