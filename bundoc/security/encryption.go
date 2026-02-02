package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	KeySize   = 32 // AES-256
	NonceSize = 12
	TagSize   = 16
	Overhead  = NonceSize + TagSize
)

// Encryptor handles AES-GCM encryption
type Encryptor struct {
	aead cipher.AEAD
}

// NewEncryptor creates a new encryptor with the given key
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{aead: aead}, nil
}

// EncryptBlock encrypts a data block.
// Returns: [Nonce (12)] + [Ciphertext (N)] + [Tag (16)]
// Total size = len(plaintext) + Overhead
func (e *Encryptor) EncryptBlock(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends result to destination.
	// We start with nonce.
	ciphertext := e.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBlock decrypts a data block.
// Expects: [Nonce (12)] + [Ciphertext (N)] + [Tag (16)]
func (e *Encryptor) DecryptBlock(data []byte) ([]byte, error) {
	if len(data) < Overhead {
		return nil, fmt.Errorf("data too short")
	}

	nonce := data[:NonceSize]
	ciphertext := data[NonceSize:]

	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateKey generates a random 32-byte key
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}
