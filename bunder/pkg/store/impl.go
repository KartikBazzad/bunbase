package store

import (
	"github.com/kartikbazzad/bunbase/bunder/internal/data_structures"
)

type storeImpl struct {
	kv *data_structures.KVStore
}

func (s *storeImpl) Get(key []byte) []byte {
	return s.kv.Get(key)
}

func (s *storeImpl) Set(key, value []byte) error {
	return s.kv.Set(key, value)
}

func (s *storeImpl) Delete(key []byte) (bool, error) {
	return s.kv.Delete(key)
}

func (s *storeImpl) Exists(key []byte) bool {
	return s.kv.Exists(key)
}

func (s *storeImpl) Keys(pattern []byte) [][]byte {
	return s.kv.Keys(pattern)
}

func (s *storeImpl) Close() error {
	return s.kv.Close()
}

// Open opens an embedded KV store at the given path.
func Open(opts Options) (Store, error) {
	if opts.BufferPoolSize <= 0 {
		opts.BufferPoolSize = 10000
	}
	if opts.Shards <= 0 {
		opts.Shards = 256
	}
	if opts.DataPath == "" {
		opts.DataPath = "./data"
	}
	kv, err := data_structures.OpenKVStore(opts.DataPath, opts.BufferPoolSize, opts.Shards)
	if err != nil {
		return nil, err
	}
	return &storeImpl{kv: kv}, nil
}
