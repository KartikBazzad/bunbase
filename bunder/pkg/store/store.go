// Package store provides an embeddable key-value store for use by other BunBase services.
package store

// Store is the interface for an embeddable KV store (Get, Set, Delete, Exists, Keys, Close).
type Store interface {
	Get(key []byte) []byte
	Set(key, value []byte) error
	Delete(key []byte) (bool, error)
	Exists(key []byte) bool
	Keys(pattern []byte) [][]byte
	Close() error
}

// Options configures the embedded store.
type Options struct {
	DataPath       string
	BufferPoolSize int
	Shards         int
}

// DefaultOptions returns sensible defaults.
func DefaultOptions(dataPath string) Options {
	return Options{
		DataPath:       dataPath,
		BufferPoolSize: 10000,
		Shards:         256,
	}
}
