package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

const RSAKeySize = 2048

// GenerateRSA2048 generates a new RSA-2048 key pair and returns the private key as PKCS#8 PEM bytes.
func GenerateRSA2048() ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, RSAKeySize)
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

// SignRSA signs digest (expected SHA-256 hash) with the private key stored in material (PKCS#8 PEM).
func SignRSA(material []byte, digest []byte) ([]byte, error) {
	if len(digest) != sha256.Size {
		return nil, errors.New("digest must be 32 bytes (SHA-256)")
	}
	priv, err := parseRSAPrivateKey(material)
	if err != nil {
		return nil, err
	}
	return rsa.SignPKCS1v15(rand.Reader, priv, 0, digest)
}

// VerifyRSA verifies a PKCS#1 v1.5 signature. publicKey is the PEM-encoded public key.
func VerifyRSA(publicKey []byte, digest []byte, signature []byte) error {
	if len(digest) != sha256.Size {
		return errors.New("digest must be 32 bytes (SHA-256)")
	}
	pub, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return err
	}
	return rsa.VerifyPKCS1v15(pub, 0, digest, signature)
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return priv, nil
}

// RSAPublicKeyFromPrivate returns PEM-encoded public key from private key material.
func RSAPublicKeyFromPrivate(material []byte) ([]byte, error) {
	priv, err := parseRSAPrivateKey(material)
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

func parseRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return pub, nil
}
