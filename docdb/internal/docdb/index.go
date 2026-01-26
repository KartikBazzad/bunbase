package docdb

import (
	"sync"

	"github.com/kartikbazzad/docdb/internal/types"
)

const (
	DefaultNumShards = 256
)

type IndexShard struct {
	mu   sync.RWMutex
	data map[uint64]*types.DocumentVersion
}

func NewIndexShard() *IndexShard {
	return &IndexShard{
		data: make(map[uint64]*types.DocumentVersion),
	}
}

func (s *IndexShard) Get(docID uint64, snapshotTxID uint64) (*types.DocumentVersion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	version, exists := s.data[docID]
	if !exists {
		return nil, false
	}

	if s.isVisible(version, snapshotTxID) {
		return version, true
	}

	return nil, false
}

func (s *IndexShard) Set(version *types.DocumentVersion) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[version.ID] = version
}

func (s *IndexShard) Delete(docID uint64, txID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version, exists := s.data[docID]; exists {
		version.DeletedTxID = &txID
	}
}

func (s *IndexShard) IsVisible(version *types.DocumentVersion, snapshotTxID uint64) bool {
	return s.isVisible(version, snapshotTxID)
}

func (s *IndexShard) isVisible(version *types.DocumentVersion, snapshotTxID uint64) bool {
	if version.CreatedTxID > snapshotTxID {
		return false
	}

	if version.DeletedTxID != nil && *version.DeletedTxID <= snapshotTxID {
		return false
	}

	return true
}

func (s *IndexShard) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *IndexShard) Snapshot() map[uint64]*types.DocumentVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[uint64]*types.DocumentVersion, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}

	return snapshot
}

type Index struct {
	shards []*IndexShard
}

func NewIndex() *Index {
	return NewIndexWithShards(DefaultNumShards)
}

func NewIndexWithShards(numShards int) *Index {
	shards := make([]*IndexShard, numShards)
	for i := range shards {
		shards[i] = NewIndexShard()
	}

	return &Index{
		shards: shards,
	}
}

func (idx *Index) getShard(docID uint64) *IndexShard {
	return idx.shards[docID%uint64(len(idx.shards))]
}

func (idx *Index) Get(docID uint64, snapshotTxID uint64) (*types.DocumentVersion, bool) {
	shard := idx.getShard(docID)
	return shard.Get(docID, snapshotTxID)
}

func (idx *Index) Set(version *types.DocumentVersion) {
	shard := idx.getShard(version.ID)
	shard.Set(version)
}

func (idx *Index) Delete(docID uint64, txID uint64) {
	shard := idx.getShard(docID)
	shard.Delete(docID, txID)
}

func (idx *Index) Size() int {
	total := 0
	for _, shard := range idx.shards {
		total += shard.Size()
	}
	return total
}

func (idx *Index) ForEach(fn func(docID uint64, version *types.DocumentVersion)) {
	for _, shard := range idx.shards {
		snapshot := shard.Snapshot()
		for docID, version := range snapshot {
			fn(docID, version)
		}
	}
}
