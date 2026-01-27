// Package docdb implements MVCC-lite (Multi-Version Concurrency Control).
//
// MVCC-lite provides:
//   - Snapshot-based reads: Readers see consistent view
//   - Transaction IDs: Monotonically increasing
//   - Versioned documents: Track document history
//   - Visibility rules: Determine which version reader sees
//
// This is "MVCC-lite" because:
//   - No long-running transactions (short-lived only)
//   - No read-your-writes detection (single writer)
//   - Simple snapshot semantics
//
// Concurrency Model:
//   - Writers: Serialized (one at a time)
//   - Readers: Never block writers (read from snapshot)
//   - Readers: Can read concurrently (different shards)
//   - Write conflicts: "Last commit wins" (no conflict detection)
package docdb

import (
	"github.com/kartikbazzad/docdb/internal/types"
)

// MVCC manages transaction IDs and version visibility.
//
// It provides:
//   - Transaction ID generation (monotonically increasing)
//   - Snapshot calculation (current visible state)
//   - Version creation (for documents)
//   - Visibility checks (determining which version to return)
//
// Transaction IDs are used to:
//   - Order operations
//   - Determine read snapshots
//   - Track document versions
//
// Thread Safety: Not thread-safe, should be protected by caller.
type MVCC struct {
	currentTxID uint64 // Next transaction ID to assign
}

func NewMVCC() *MVCC {
	return &MVCC{
		currentTxID: 1,
	}
}

func (m *MVCC) NextTxID() uint64 {
	txID := m.currentTxID
	m.currentTxID++
	return txID
}

func (m *MVCC) SetCurrentTxID(txID uint64) {
	m.currentTxID = txID
}

func (m *MVCC) CurrentSnapshot() uint64 {
	return m.currentTxID - 1
}

func (m *MVCC) IsVisible(version *types.DocumentVersion, snapshotTxID uint64) bool {
	if version.CreatedTxID > snapshotTxID {
		return false
	}

	if version.DeletedTxID != nil && *version.DeletedTxID <= snapshotTxID {
		return false
	}

	return true
}

func (m *MVCC) CreateVersion(docID uint64, txID uint64, offset uint64, length uint32) *types.DocumentVersion {
	return &types.DocumentVersion{
		ID:          docID,
		CreatedTxID: txID,
		DeletedTxID: nil,
		Offset:      offset,
		Length:      length,
	}
}

func (m *MVCC) UpdateVersion(version *types.DocumentVersion, newTxID uint64, newOffset uint64, newLength uint32) *types.DocumentVersion {
	return &types.DocumentVersion{
		ID:          version.ID,
		CreatedTxID: newTxID,
		DeletedTxID: nil,
		Offset:      newOffset,
		Length:      newLength,
	}
}

func (m *MVCC) DeleteVersion(docID uint64, txID uint64) *types.DocumentVersion {
	return &types.DocumentVersion{
		ID:          docID,
		CreatedTxID: txID,
		DeletedTxID: &txID,
		Offset:      0,
		Length:      0,
	}
}
