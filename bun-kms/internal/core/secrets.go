package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/audit"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/storage"
)

type SecretRecord struct {
	Name       string
	Ciphertext []byte
	CreatedAt  time.Time
}

type SecretStore struct {
	mu        sync.RWMutex
	masterKey []byte
	secrets   map[string]SecretRecord
	store     storage.Store
	audit     audit.Store
}

func NewSecretStore(masterKey []byte) (*SecretStore, error) {
	if len(masterKey) != 32 {
		return nil, errors.New("master key must be 32 bytes")
	}
	return &SecretStore{
		masterKey: masterKey,
		secrets:   map[string]SecretRecord{},
	}, nil
}

// NewSecretStoreWithStore creates a secret store backed by the given store (optional) and loads existing secrets.
// When store is nil, the store is in-memory only. auditLog is optional; when non-nil, operations are logged.
func NewSecretStoreWithStore(masterKey []byte, store storage.Store, auditLog audit.Store) (*SecretStore, error) {
	s, err := NewSecretStore(masterKey)
	if err != nil {
		return nil, err
	}
	s.store = store
	s.audit = auditLog
	if store != nil {
		if err := s.loadSecrets(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *SecretStore) loadSecrets() error {
	if s.store == nil {
		return nil
	}
	keyList, err := s.store.KeysWithPrefix(storage.SecretPrefix())
	if err != nil {
		return err
	}
	for _, fullKey := range keyList {
		name := storage.ParseSecretName(fullKey)
		if name == "" {
			continue
		}
		val, err := s.store.Get(fullKey)
		if err != nil || val == nil {
			continue
		}
		var rec SecretRecord
		if err := json.Unmarshal(val, &rec); err != nil {
			continue
		}
		s.secrets[name] = rec
	}
	return nil
}

func (s *SecretStore) persistSecret(name string, rec SecretRecord) error {
	if s.store == nil {
		return nil
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return s.store.Set(storage.SecretStorageKey(name), data)
}

func (s *SecretStore) Put(name string, value []byte) (SecretRecord, error) {
	if name == "" {
		return SecretRecord{}, errors.New("name required")
	}
	ciphertext, err := s.encrypt(value)
	if err != nil {
		return SecretRecord{}, err
	}
	record := SecretRecord{
		Name:       name,
		Ciphertext: ciphertext,
		CreatedAt:  time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[name] = record
	if err := s.persistSecret(name, record); err != nil {
		delete(s.secrets, name)
		return SecretRecord{}, err
	}
	if s.audit != nil {
		_ = s.audit.Log(audit.Event{Operation: audit.SecretPut, Resource: name, Success: true})
	}
	return record, nil
}

func (s *SecretStore) Get(name string) ([]byte, SecretRecord, error) {
	s.mu.RLock()
	record, exists := s.secrets[name]
	s.mu.RUnlock()
	if !exists {
		if s.audit != nil {
			_ = s.audit.Log(audit.Event{Operation: audit.SecretGet, Resource: name, Success: false, Message: "secret not found"})
		}
		return nil, SecretRecord{}, errors.New("secret not found")
	}
	plaintext, err := s.decrypt(record.Ciphertext)
	if err != nil {
		if s.audit != nil {
			_ = s.audit.Log(audit.Event{Operation: audit.SecretGet, Resource: name, Success: false, Message: "decrypt failed"})
		}
		return nil, SecretRecord{}, err
	}
	if s.audit != nil {
		_ = s.audit.Log(audit.Event{Operation: audit.SecretGet, Resource: name, Success: true})
	}
	return plaintext, record, nil
}

func (s *SecretStore) encrypt(value []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, value, nil)
	return append(nonce, ciphertext...), nil
}

func (s *SecretStore) decrypt(blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(blob) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce := blob[:gcm.NonceSize()]
	ciphertext := blob[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decrypt failed")
	}
	return plaintext, nil
}
