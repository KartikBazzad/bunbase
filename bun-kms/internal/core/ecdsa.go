package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
)

// GenerateECDSA256 generates a new ECDSA P-256 key pair and returns the private key as PKCS#8 PEM bytes.
func GenerateECDSA256() ([]byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: b}
	return pem.EncodeToMemory(block), nil
}

// SignECDSA signs digest (expected SHA-256 hash) with the ECDSA private key (PKCS#8 PEM).
func SignECDSA(material []byte, digest []byte) ([]byte, error) {
	if len(digest) != sha256.Size {
		return nil, errors.New("digest must be 32 bytes (SHA-256)")
	}
	priv, err := parseECDSAPrivateKey(material)
	if err != nil {
		return nil, err
	}
	r, s, err := ecdsa.Sign(rand.Reader, priv, digest)
	if err != nil {
		return nil, err
	}
	// P-256: r and s are 32 bytes each when padded
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sig[32-len(rBytes):], rBytes)
	copy(sig[64-len(sBytes):], sBytes)
	return sig, nil
}

// VerifyECDSA verifies an ECDSA signature. publicKey is PEM-encoded. Signature is r||s (32+32 bytes for P-256).
func VerifyECDSA(publicKey []byte, digest []byte, signature []byte) error {
	if len(digest) != sha256.Size {
		return errors.New("digest must be 32 bytes (SHA-256)")
	}
	if len(signature) != 64 {
		return errors.New("signature must be 64 bytes (r||s)")
	}
	pub, err := parseECDSAPublicKey(publicKey)
	if err != nil {
		return err
	}
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:64])
	if ecdsa.Verify(pub, digest, r, s) {
		return nil
	}
	return errors.New("signature verification failed")
}

func parseECDSAPrivateKey(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an ECDSA private key")
	}
	return priv, nil
}

// ECDSAPublicKeyFromPrivate returns PEM-encoded public key from private key material.
func ECDSAPublicKeyFromPrivate(material []byte) ([]byte, error) {
	priv, err := parseECDSAPrivateKey(material)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: b}
	return pem.EncodeToMemory(block), nil
}

func parseECDSAPublicKey(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("not an ECDSA public key")
	}
	return pub, nil
}
