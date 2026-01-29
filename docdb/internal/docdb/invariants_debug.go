//go:build debug

package docdb

import (
	"fmt"

	"github.com/kartikbazzad/docdb/internal/types"
)

// checkSingleWriter verifies that we are in the write path (op is a write type).
// Call only from executeTask after acquiring partition.mu for writes.
// Panics if op is a read (invariant: writes hold partition lock).
func checkSingleWriter(partition *Partition, op types.OperationType) {
	switch op {
	case types.OpCreate, types.OpUpdate, types.OpDelete, types.OpPatch,
		types.OpCreateCollection, types.OpDeleteCollection:
		return
	case types.OpRead:
		panic(fmt.Sprintf("docdb invariant: checkSingleWriter called for read op on partition %d", partition.ID()))
	default:
		panic(fmt.Sprintf("docdb invariant: unknown op type %d on partition %d", op, partition.ID()))
	}
}

// checkSnapshotMonotonic verifies snapshot txID is not in the future (<= max visible).
// Panics if snapshotTxID > mvcc.MaxVisibleTxID().
func checkSnapshotMonotonic(mvcc *MVCC, snapshotTxID uint64) {
	max := mvcc.MaxVisibleTxID()
	if snapshotTxID > max {
		panic(fmt.Sprintf("docdb invariant: snapshot txID %d > max visible %d", snapshotTxID, max))
	}
}

// checkRecoveryCommittedOnly verifies we only apply committed transactions during recovery.
// Panics if txID is not in committed set.
func checkRecoveryCommittedOnly(txID uint64, committed map[uint64]bool) {
	if !committed[txID] {
		panic(fmt.Sprintf("docdb invariant: recovery applying uncommitted txID %d", txID))
	}
}

// checkQuerySnapshotConsistent verifies query uses a valid snapshot when partitions exist.
// Panics if we have partitions but snapshot is zero (bug).
func checkQuerySnapshotConsistent(snapshotTxID uint64, partitionCount int) {
	if partitionCount > 1 && snapshotTxID == 0 {
		panic("docdb invariant: query with partitions but snapshot txID 0")
	}
}
