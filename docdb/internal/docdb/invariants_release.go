//go:build !debug

package docdb

import (
	"github.com/kartikbazzad/docdb/internal/types"
)

func checkSingleWriter(partition *Partition, op types.OperationType) {
	_ = partition
	_ = op
}

func checkSnapshotMonotonic(mvcc *MVCC, snapshotTxID uint64) {
	_ = mvcc
	_ = snapshotTxID
}

func checkRecoveryCommittedOnly(txID uint64, committed map[uint64]bool) {
	_ = txID
	_ = committed
}

func checkQuerySnapshotConsistent(snapshotTxID uint64, partitionCount int) {
	_ = snapshotTxID
	_ = partitionCount
}
