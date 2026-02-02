// Package concurrency provides sharded maps and page latches for Bunder concurrency.
package concurrency

import (
	"hash/fnv"
	"sync"
)

// ShardedMap is a key-value map sharded by FNV-1a hash of the key to reduce lock contention.
// Default 256 shards; each shard has its own RWMutex. Get returns a copy of the value.
type ShardedMap struct {
	shards []*shard
	n      int
}

type shard struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewShardedMap creates a sharded map with n shards (e.g. 256).
func NewShardedMap(n int) *ShardedMap {
	if n <= 0 {
		n = 256
	}
	shards := make([]*shard, n)
	for i := range shards {
		shards[i] = &shard{data: make(map[string][]byte)}
	}
	return &ShardedMap{shards: shards, n: n}
}

// hash returns shard index for key (FNV-1a).
func (m *ShardedMap) hash(key []byte) uint32 {
	h := fnv.New32a()
	_, _ = h.Write(key)
	return h.Sum32() % uint32(m.n)
}

func (m *ShardedMap) shardForKey(key []byte) *shard {
	return m.shards[m.hash(key)]
}

// Get returns the value for key, or nil if not set.
func (m *ShardedMap) Get(key []byte) []byte {
	s := m.shardForKey(key)
	s.mu.RLock()
	v, ok := s.data[string(key)]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	// Return copy so caller cannot mutate
	out := make([]byte, len(v))
	copy(out, v)
	return out
}

// Set sets key to value.
func (m *ShardedMap) Set(key, value []byte) {
	s := m.shardForKey(key)
	s.mu.Lock()
	if s.data == nil {
		s.data = make(map[string][]byte)
	}
	s.data[string(key)] = append([]byte(nil), value...)
	s.mu.Unlock()
}

// Delete removes key. Returns true if the key was present.
func (m *ShardedMap) Delete(key []byte) bool {
	s := m.shardForKey(key)
	s.mu.Lock()
	_, ok := s.data[string(key)]
	if ok {
		delete(s.data, string(key))
	}
	s.mu.Unlock()
	return ok
}

// Exists returns true if key is set.
func (m *ShardedMap) Exists(key []byte) bool {
	s := m.shardForKey(key)
	s.mu.RLock()
	_, ok := s.data[string(key)]
	s.mu.RUnlock()
	return ok
}

// Keys returns all keys (across all shards). Use for KEYS * only; expensive.
func (m *ShardedMap) Keys() [][]byte {
	var out [][]byte
	for _, s := range m.shards {
		s.mu.RLock()
		for k := range s.data {
			out = append(out, []byte(k))
		}
		s.mu.RUnlock()
	}
	return out
}

// Count returns the total number of keys (expensive).
func (m *ShardedMap) Count() int {
	n := 0
	for _, s := range m.shards {
		s.mu.RLock()
		n += len(s.data)
		s.mu.RUnlock()
	}
	return n
}
