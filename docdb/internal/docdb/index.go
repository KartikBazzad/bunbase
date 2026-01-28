// Package docdb implements sharded in-memory index for document lookups.
//
// The index is split into multiple shards to reduce lock contention.
// Each shard is protected by its own RWMutex, allowing concurrent reads
// across different shards while still providing thread-safety.
//
// Sharding Strategy: Document ID is hashed using modulo operation.
//
//	shard_id = doc_id % num_shards
//
// This ensures documents are distributed evenly across shards.
package docdb

import (
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/types"
)

const (
	DefaultNumShards = 256 // Number of index shards (tunable for performance)
)

// IndexShard manages a subset of document versions.
//
// Each shard is protected by its own RWMutex to enable:
// - Concurrent reads from different goroutines
// - Serialized writes within a shard
// - High throughput with reduced contention
//
// Thread Safety: All methods are thread-safe via mu.
type IndexShard struct {
	mu   sync.RWMutex                      // Protects shard data
	data map[uint64]*types.DocumentVersion // Document versions in this shard
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

// LiveCount returns the number of live (non-deleted) documents in the index.
func (s *IndexShard) LiveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, v := range s.data {
		if v.DeletedTxID == nil {
			count++
		}
	}

	return count
}

// TombstonedCount returns the number of deleted documents in the index.
func (s *IndexShard) TombstonedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, v := range s.data {
		if v.DeletedTxID != nil {
			count++
		}
	}

	return count
}

// LastCompaction returns the time of the last compaction.
func (s *IndexShard) LastCompaction() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return time.Time{}
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
	mu     sync.RWMutex
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
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	total := 0
	for _, shard := range idx.shards {
		total += shard.Size()
	}
	return total
}

func (idx *Index) ForEach(fn func(docID uint64, version *types.DocumentVersion)) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	for _, shard := range idx.shards {
		snapshot := shard.Snapshot()
		for docID, version := range snapshot {
			fn(docID, version)
		}
	}
}

func (idx *Index) LiveCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	count := 0
	for _, shard := range idx.shards {
		count += shard.LiveCount()
	}

	return count
}

func (idx *Index) TombstonedCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	count := 0
	for _, shard := range idx.shards {
		count += shard.TombstonedCount()
	}

	return count
}

func (idx *Index) LastCompaction() time.Time {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var latest time.Time
	for _, shard := range idx.shards {
		shardLatest := shard.LastCompaction()
		if shardLatest.After(latest) {
			latest = shardLatest
		}
	}

	return latest
}

