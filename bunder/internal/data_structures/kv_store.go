// Package data_structures provides the KV store and Redis-like List, Set, and Hash types.
package data_structures

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/kartikbazzad/bunbase/bunder/internal/concurrency"
	"github.com/kartikbazzad/bunbase/bunder/internal/storage"
)

// ErrClosed is returned when an operation is attempted on a closed KV store.
var ErrClosed = errors.New("kv store closed")

// KVStore is the primary key-value store: sharded in-memory map plus persistent B+Tree.
// OpenKVStore creates data.db, ensures pages 0–1 (meta, freelist), and loads the freelist.
type KVStore struct {
	mu         sync.RWMutex
	data       *concurrency.ShardedMap
	pool       *storage.BufferPool
	pager      *storage.Pager
	freelist   *storage.FreeList
	btree      *storage.BTree
	rootPageID storage.PageID
	closed     bool
}

// OpenKVStore opens or creates a KV store at dataPath (creates data/data.db, ensures pages 0–1, loads freelist).
func OpenKVStore(dataPath string, bufferPoolSize, shards int) (*KVStore, error) {
	if bufferPoolSize <= 0 {
		bufferPoolSize = 10000
	}
	if shards <= 0 {
		shards = 256
	}
	dbPath := filepath.Join(dataPath, "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	pager, err := storage.NewPager(dbPath)
	if err != nil {
		return nil, err
	}
	if err := pager.EnsurePages(2); err != nil {
		pager.Close()
		return nil, err
	}
	pool := storage.NewBufferPool(bufferPoolSize, pager)
	freelist := storage.NewFreeList(pager, pool)
	if err := freelist.Load(); err != nil {
		pool.Close()
		return nil, err
	}
	// Use pager for new pages (BTree will use pool which uses pager); freelist for reuse
	btree, err := storage.NewBTree(pool)
	if err != nil {
		pool.Close()
		return nil, err
	}
	kv := &KVStore{
		data:       concurrency.NewShardedMap(shards),
		pool:       pool,
		pager:      pager,
		freelist:   freelist,
		btree:      btree,
		rootPageID: btree.GetRootID(),
	}
	return kv, nil
}

// Get returns the value for key, or nil if not found.
func (kv *KVStore) Get(key []byte) []byte {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if kv.closed {
		return nil
	}
	return kv.data.Get(key)
}

// Set sets key to value.
func (kv *KVStore) Set(key, value []byte) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return ErrClosed
	}
	kv.data.Set(key, value)
	if kv.btree != nil {
		return kv.btree.Put(key, value)
	}
	return nil
}

// Delete removes key. Returns true if the key was present.
func (kv *KVStore) Delete(key []byte) (bool, error) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return false, ErrClosed
	}
	ok := kv.data.Delete(key)
	if ok && kv.btree != nil {
		_ = kv.btree.Delete(key)
	}
	return ok, nil
}

// Exists returns true if key is set.
func (kv *KVStore) Exists(key []byte) bool {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if kv.closed {
		return false
	}
	return kv.data.Exists(key)
}

// Keys returns all keys matching the pattern. Pattern supports * (any) and ? (single char); empty = all.
func (kv *KVStore) Keys(pattern []byte) [][]byte {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if kv.closed {
		return nil
	}
	all := kv.data.Keys()
	if len(pattern) == 0 {
		return all
	}
	re := patternToRegex(pattern)
	var out [][]byte
	for _, k := range all {
		if re.Match(k) {
			out = append(out, k)
		}
	}
	return out
}

func patternToRegex(pattern []byte) *regexp.Regexp {
	var b bytes.Buffer
	b.WriteString("^")
	for _, c := range pattern {
		switch c {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteString("$")
	re, _ := regexp.Compile(b.String())
	return re
}

// Count returns the number of keys.
func (kv *KVStore) Count() int {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if kv.closed {
		return 0
	}
	return kv.data.Count()
}

// Close closes the store and flushes to disk.
func (kv *KVStore) Close() error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.closed {
		return nil
	}
	kv.closed = true
	if kv.freelist != nil {
		_ = kv.freelist.Persist(kv.pool)
	}
	return kv.pool.Close()
}
