package docdb

import (
	"github.com/kartikbazzad/docdb/internal/types"
)

type MVCC struct {
	currentTxID uint64
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
