package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/audit"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/storage"
)

type KeyType string

const (
	KeyTypeAES256   KeyType = "aes-256"
	KeyTypeRSA2048  KeyType = "rsa-2048"
	KeyTypeECDSA256 KeyType = "ecdsa-p256"
)

type KeyVersion struct {
	Version   int
	Material  []byte
	CreatedAt time.Time
}

type Key struct {
	Name      string
	Type      KeyType
	CreatedAt time.Time
	Versions  []KeyVersion
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type Vault struct {
	mu    sync.RWMutex
	keys  map[string]*Key
	store storage.Store
	audit audit.Store
}

func NewVault() *Vault {
	return &Vault{
		keys: map[string]*Key{},
	}
}

func NewVaultWithStore(store storage.Store, auditLog audit.Store) (*Vault, error) {
	v := &Vault{
		keys:  map[string]*Key{},
		store: store,
		audit: auditLog,
	}
	if store != nil {
		if err := v.loadKeys(); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (v *Vault) loadKeys() error {
	if v.store == nil {
		return nil
	}
	keyList, err := v.store.KeysWithPrefix(storage.KeyPrefix())
	if err != nil {
		return err
	}
	for _, fullKey := range keyList {
		name := storage.ParseKeyName(fullKey)
		if name == "" {
			continue
		}
		val, err := v.store.Get(fullKey)
		if err != nil || val == nil {
			continue
		}
		var key Key
		if err := json.Unmarshal(val, &key); err != nil {
			continue
		}
		v.keys[name] = &key
	}
	return nil
}

func (v *Vault) persistKey(name string, key *Key) error {
	if v.store == nil {
		return nil
	}
	data, err := json.Marshal(key)
	if err != nil {
		return err
	}
	return v.store.Set(storage.KeyStorageKey(name), data)
}

func (v *Vault) CreateKey(name string, keyType KeyType) (*Key, error) {
	if name == "" {
		return nil, errors.New("name required")
	}
	if keyType == "" {
		keyType = KeyTypeAES256
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, exists := v.keys[name]; exists {
		return nil, errors.New("key already exists")
	}

	var material []byte
	var err error
	switch keyType {
	case KeyTypeAES256:
		material = make([]byte, 32)
		if _, err = io.ReadFull(rand.Reader, material); err != nil {
			return nil, err
		}
	case KeyTypeRSA2048:
		material, err = GenerateRSA2048()
		if err != nil {
			return nil, err
		}
	case KeyTypeECDSA256:
		material, err = GenerateECDSA256()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	now := time.Now().UTC()
	key := &Key{
		Name:      name,
		Type:      keyType,
		CreatedAt: now,
		Versions: []KeyVersion{
			{Version: 1, Material: material, CreatedAt: now},
		},
	}
	v.keys[name] = key
	if err := v.persistKey(name, key); err != nil {
		delete(v.keys, name)
		return nil, err
	}
	if v.audit != nil {
		_ = v.audit.Log(audit.Event{Operation: audit.KeyCreated, Resource: name, Success: true})
	}
	return cloneKeyMetadata(key), nil
}

func (v *Vault) GetKey(name string) (*Key, error) {
	v.mu.RLock()
	key, exists := v.keys[name]
	v.mu.RUnlock()
	if !exists {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.KeyGet, Resource: name, Success: false, Message: "key not found"})
		}
		return nil, errors.New("key not found")
	}
	if key.RevokedAt != nil {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.KeyGet, Resource: name, Success: false, Message: "key revoked"})
		}
		return nil, errors.New("key revoked")
	}
	if v.audit != nil {
		_ = v.audit.Log(audit.Event{Operation: audit.KeyGet, Resource: name, Success: true})
	}
	return cloneKeyMetadata(key), nil
}

func (v *Vault) RotateKey(name string) (*Key, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	key, exists := v.keys[name]
	if !exists {
		return nil, errors.New("key not found")
	}
	if key.RevokedAt != nil {
		return nil, errors.New("key revoked")
	}
	var material []byte
	var err error
	switch key.Type {
	case KeyTypeAES256:
		material = make([]byte, 32)
		if _, err = io.ReadFull(rand.Reader, material); err != nil {
			return nil, err
		}
	case KeyTypeRSA2048:
		material, err = GenerateRSA2048()
		if err != nil {
			return nil, err
		}
	case KeyTypeECDSA256:
		material, err = GenerateECDSA256()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", key.Type)
	}
	now := time.Now().UTC()
	key.Versions = append(key.Versions, KeyVersion{
		Version:   len(key.Versions) + 1,
		Material:  material,
		CreatedAt: now,
	})
	if err := v.persistKey(name, key); err != nil {
		key.Versions = key.Versions[:len(key.Versions)-1]
		return nil, err
	}
	if v.audit != nil {
		_ = v.audit.Log(audit.Event{Operation: audit.KeyRotated, Resource: name, Success: true})
	}
	return cloneKeyMetadata(key), nil
}

func (v *Vault) RevokeKey(name string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	key, exists := v.keys[name]
	if !exists {
		return errors.New("key not found")
	}
	if key.RevokedAt != nil {
		return errors.New("key already revoked")
	}
	now := time.Now().UTC()
	key.RevokedAt = &now
	if err := v.persistKey(name, key); err != nil {
		key.RevokedAt = nil
		return err
	}
	if v.audit != nil {
		_ = v.audit.Log(audit.Event{Operation: audit.KeyRevoked, Resource: name, Success: true})
	}
	return nil
}

func (v *Vault) Encrypt(name string, plaintext, aad []byte) ([]byte, int, []byte, error) {
	v.mu.RLock()
	key, exists := v.keys[name]
	v.mu.RUnlock()
	if !exists {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataEncrypted, Resource: name, Success: false, Message: "key not found"})
		}
		return nil, 0, nil, errors.New("key not found")
	}
	if key.RevokedAt != nil {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataEncrypted, Resource: name, Success: false, Message: "key revoked"})
		}
		return nil, 0, nil, errors.New("key revoked")
	}
	if len(key.Versions) == 0 {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataEncrypted, Resource: name, Success: false, Message: "no key versions"})
		}
		return nil, 0, nil, errors.New("no key versions available")
	}
	if key.Type != KeyTypeAES256 {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataEncrypted, Resource: name, Success: false, Message: "key type does not support encrypt"})
		}
		return nil, 0, nil, errors.New("key type does not support encrypt")
	}
	version := key.Versions[len(key.Versions)-1]
	block, err := aes.NewCipher(version.Material)
	if err != nil {
		return nil, 0, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, 0, nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)
	if v.audit != nil {
		_ = v.audit.Log(audit.Event{Operation: audit.DataEncrypted, Resource: name, Success: true})
	}
	return ciphertext, version.Version, nonce, nil
}

func (v *Vault) Decrypt(name string, version int, nonce, ciphertext, aad []byte) ([]byte, error) {
	v.mu.RLock()
	key, exists := v.keys[name]
	v.mu.RUnlock()
	if !exists {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataDecrypted, Resource: name, Success: false, Message: "key not found"})
		}
		return nil, errors.New("key not found")
	}
	if key.RevokedAt != nil {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataDecrypted, Resource: name, Success: false, Message: "key revoked"})
		}
		return nil, errors.New("key revoked")
	}
	if version <= 0 || version > len(key.Versions) {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataDecrypted, Resource: name, Success: false, Message: "invalid key version"})
		}
		return nil, errors.New("invalid key version")
	}
	if key.Type != KeyTypeAES256 {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataDecrypted, Resource: name, Success: false, Message: "key type does not support decrypt"})
		}
		return nil, errors.New("key type does not support decrypt")
	}
	material := key.Versions[version-1].Material
	block, err := aes.NewCipher(material)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		if v.audit != nil {
			_ = v.audit.Log(audit.Event{Operation: audit.DataDecrypted, Resource: name, Success: false, Message: "decrypt failed"})
		}
		return nil, errors.New("decrypt failed")
	}
	if v.audit != nil {
		_ = v.audit.Log(audit.Event{Operation: audit.DataDecrypted, Resource: name, Success: true})
	}
	return plaintext, nil
}

func cloneKeyMetadata(key *Key) *Key {
	versions := make([]KeyVersion, len(key.Versions))
	for i, kv := range key.Versions {
		versions[i] = KeyVersion{Version: kv.Version, CreatedAt: kv.CreatedAt}
	}
	var revokedAt *time.Time
	if key.RevokedAt != nil {
		t := *key.RevokedAt
		revokedAt = &t
	}
	return &Key{
		Name:      key.Name,
		Type:      key.Type,
		CreatedAt: key.CreatedAt,
		Versions:  versions,
		RevokedAt: revokedAt,
	}
}
