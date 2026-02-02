package storage

// Store is the persistence interface for keys and secrets (encrypted at rest by the implementation).
type Store interface {
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	Delete(key []byte) (bool, error)
	KeysWithPrefix(prefix []byte) ([][]byte, error)
	Close() error
}
