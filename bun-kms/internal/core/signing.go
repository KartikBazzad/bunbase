package core

import (
	"crypto/sha256"
	"errors"
)

// Sign signs digest (must be SHA-256, 32 bytes) with the key's latest version. Only RSA-2048 and ECDSA-P256 are supported.
func (v *Vault) Sign(name string, digest []byte) ([]byte, error) {
	v.mu.RLock()
	key, exists := v.keys[name]
	v.mu.RUnlock()
	if !exists {
		return nil, errors.New("key not found")
	}
	if key.RevokedAt != nil {
		return nil, errors.New("key revoked")
	}
	if len(key.Versions) == 0 {
		return nil, errors.New("no key versions available")
	}
	if len(digest) != sha256.Size {
		return nil, errors.New("digest must be 32 bytes (SHA-256)")
	}
	version := key.Versions[len(key.Versions)-1]
	switch key.Type {
	case KeyTypeRSA2048:
		return SignRSA(version.Material, digest)
	case KeyTypeECDSA256:
		return SignECDSA(version.Material, digest)
	default:
		return nil, errors.New("key type does not support signing")
	}
}

// Verify verifies a signature for the given key name and digest. For RSA, publicKey is derived from the key's latest version.
func (v *Vault) Verify(name string, digest []byte, signature []byte) error {
	v.mu.RLock()
	key, exists := v.keys[name]
	v.mu.RUnlock()
	if !exists {
		return errors.New("key not found")
	}
	if key.RevokedAt != nil {
		return errors.New("key revoked")
	}
	if len(key.Versions) == 0 {
		return errors.New("no key versions available")
	}
	version := key.Versions[len(key.Versions)-1]
	switch key.Type {
	case KeyTypeRSA2048:
		pub, err := RSAPublicKeyFromPrivate(version.Material)
		if err != nil {
			return err
		}
		return VerifyRSA(pub, digest, signature)
	case KeyTypeECDSA256:
		pub, err := ECDSAPublicKeyFromPrivate(version.Material)
		if err != nil {
			return err
		}
		return VerifyECDSA(pub, digest, signature)
	default:
		return errors.New("key type does not support verification")
	}
}
