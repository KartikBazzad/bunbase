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
	DefaultNumShards = 512 // Number of index shards (tunable for performance) - increased from 256 for better concurrency
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

// CollectionIndex manages the index for a single collection.
type CollectionIndex struct {
	mu     sync.RWMutex
	shards []*IndexShard
}

func NewCollectionIndex() *CollectionIndex {
	return NewCollectionIndexWithShards(DefaultNumShards)
}

func NewCollectionIndexWithShards(numShards int) *CollectionIndex {
	shards := make([]*IndexShard, numShards)
	for i := range shards {
		shards[i] = NewIndexShard()
	}

	return &CollectionIndex{
		shards: shards,
	}
}

func (ci *CollectionIndex) getShard(docID uint64) *IndexShard {
	return ci.shards[docID%uint64(len(ci.shards))]
}

func (ci *CollectionIndex) Get(docID uint64, snapshotTxID uint64) (*types.DocumentVersion, bool) {
	shard := ci.getShard(docID)
	return shard.Get(docID, snapshotTxID)
}

func (ci *CollectionIndex) Set(version *types.DocumentVersion) {
	shard := ci.getShard(version.ID)
	shard.Set(version)
}

func (ci *CollectionIndex) Delete(docID uint64, txID uint64) {
	shard := ci.getShard(docID)
	shard.Delete(docID, txID)
}

func (ci *CollectionIndex) Size() int {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	total := 0
	for _, shard := range ci.shards {
		total += shard.Size()
	}
	return total
}

func (ci *CollectionIndex) ForEach(fn func(docID uint64, version *types.DocumentVersion)) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	for _, shard := range ci.shards {
		snapshot := shard.Snapshot()
		for docID, version := range snapshot {
			fn(docID, version)
		}
	}
}

func (ci *CollectionIndex) LiveCount() int {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	count := 0
	for _, shard := range ci.shards {
		count += shard.LiveCount()
	}
	return count
}

func (ci *CollectionIndex) TombstonedCount() int {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	count := 0
	for _, shard := range ci.shards {
		count += shard.TombstonedCount()
	}
	return count
}

type Index struct {
	mu          sync.RWMutex
	collections map[string]*CollectionIndex
}

func NewIndex() *Index {
	return &Index{
		collections: make(map[string]*CollectionIndex),
	}
}

// getOrCreateCollectionIndex gets or creates the index for a collection.
func (idx *Index) getOrCreateCollectionIndex(collection string) *CollectionIndex {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if collection == "" {
		collection = DefaultCollection
	}

	if ci, exists := idx.collections[collection]; exists {
		return ci
	}

	ci := NewCollectionIndex()
	idx.collections[collection] = ci
	return ci
}

// getCollectionIndex gets the index for a collection (returns nil if not found).
func (idx *Index) getCollectionIndex(collection string) *CollectionIndex {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if collection == "" {
		collection = DefaultCollection
	}

	return idx.collections[collection]
}

func (idx *Index) Get(collection string, docID uint64, snapshotTxID uint64) (*types.DocumentVersion, bool) {
	ci := idx.getCollectionIndex(collection)
	if ci == nil {
		return nil, false
	}
	return ci.Get(docID, snapshotTxID)
}

func (idx *Index) Set(collection string, version *types.DocumentVersion) {
	ci := idx.getOrCreateCollectionIndex(collection)
	ci.Set(version)
}

func (idx *Index) Delete(collection string, docID uint64, txID uint64) {
	ci := idx.getCollectionIndex(collection)
	if ci == nil {
		return
	}
	ci.Delete(docID, txID)
}

func (idx *Index) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	total := 0
	for _, ci := range idx.collections {
		total += ci.Size()
	}
	return total
}

func (idx *Index) ForEach(collection string, fn func(docID uint64, version *types.DocumentVersion)) {
	ci := idx.getCollectionIndex(collection)
	if ci == nil {
		return
	}
	ci.ForEach(fn)
}

func (idx *Index) ForEachCollection(fn func(collection string, ci *CollectionIndex)) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	for name, ci := range idx.collections {
		fn(name, ci)
	}
}

func (idx *Index) LiveCount(collection string) int {
	ci := idx.getCollectionIndex(collection)
	if ci == nil {
		return 0
	}
	return ci.LiveCount()
}

func (idx *Index) TombstonedCount(collection string) int {
	ci := idx.getCollectionIndex(collection)
	if ci == nil {
		return 0
	}
	return ci.TombstonedCount()
}

func (idx *Index) TotalLiveCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	count := 0
	for _, ci := range idx.collections {
		count += ci.LiveCount()
	}
	return count
}

func (idx *Index) TotalTombstonedCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	count := 0
	for _, ci := range idx.collections {
		count += ci.TombstonedCount()
	}
	return count
}

func (idx *Index) LastCompaction() time.Time {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// CollectionIndex doesn't track compaction time per collection
	// This is a placeholder - actual implementation would track per collection
	return time.Time{}
}

// GetCollectionNames returns all collection names that have indexes.
func (idx *Index) GetCollectionNames() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	names := make([]string, 0, len(idx.collections))
	for name := range idx.collections {
		names = append(names, name)
	}
	return names
}
