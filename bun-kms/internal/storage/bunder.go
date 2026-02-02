package storage

import (
	"bytes"
	"errors"

	"github.com/kartikbazzad/bunbase/bunder/pkg/store"
)

const (
	keyPrefix    = "kms:key:"
	secretPrefix = "kms:secret:"
)

// BunderStore wraps Bunder's embeddable store with AES-GCM encryption at rest.
type BunderStore struct {
	backend   store.Store
	masterKey []byte
}

// NewBunderStore opens an embedded Bunder store at dataPath and wraps it with encryption.
func NewBunderStore(dataPath string, masterKey []byte, opts *store.Options) (Store, error) {
	if len(masterKey) != 32 {
		return nil, errors.New("master key must be 32 bytes")
	}
	if dataPath == "" {
		dataPath = "./data"
	}
	o := store.DefaultOptions(dataPath)
	if opts != nil {
		if opts.BufferPoolSize > 0 {
			o.BufferPoolSize = opts.BufferPoolSize
		}
		if opts.Shards > 0 {
			o.Shards = opts.Shards
		}
	}
	backend, err := store.Open(o)
	if err != nil {
		return nil, err
	}
	return &BunderStore{backend: backend, masterKey: masterKey}, nil
}

func (b *BunderStore) Get(key []byte) ([]byte, error) {
	raw := b.backend.Get(key)
	if raw == nil {
		return nil, nil
	}
	return DecryptValue(b.masterKey, raw)
}

func (b *BunderStore) Set(key, value []byte) error {
	enc, err := EncryptValue(b.masterKey, value)
	if err != nil {
		return err
	}
	return b.backend.Set(key, enc)
}

func (b *BunderStore) Delete(key []byte) (bool, error) {
	return b.backend.Delete(key)
}

func (b *BunderStore) KeysWithPrefix(prefix []byte) ([][]byte, error) {
	pattern := make([]byte, len(prefix)+1)
	copy(pattern, prefix)
	pattern[len(prefix)] = '*'
	return b.backend.Keys(pattern), nil
}

func (b *BunderStore) Close() error {
	return b.backend.Close()
}

// KeyStorageKey returns the storage key for a vault key name.
func KeyStorageKey(name string) []byte {
	return append([]byte(keyPrefix), name...)
}

// SecretStorageKey returns the storage key for a secret name.
func SecretStorageKey(name string) []byte {
	return append([]byte(secretPrefix), name...)
}

// KeyPrefix returns the prefix for listing keys.
func KeyPrefix() []byte {
	return []byte(keyPrefix)
}

// SecretPrefix returns the prefix for listing secrets.
func SecretPrefix() []byte {
	return []byte(secretPrefix)
}

// ParseKeyName strips the key prefix from a storage key to get the name.
func ParseKeyName(full []byte) string {
	if !bytes.HasPrefix(full, []byte(keyPrefix)) {
		return ""
	}
	return string(full[len(keyPrefix):])
}

// ParseSecretName strips the secret prefix from a storage key.
func ParseSecretName(full []byte) string {
	if !bytes.HasPrefix(full, []byte(secretPrefix)) {
		return ""
	}
	return string(full[len(secretPrefix):])
}
