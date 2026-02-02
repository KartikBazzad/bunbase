package security

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"hash"
	"strings"
)

// SCRAM Constants
const (
	ScramIterCount = 4096
	ScramSaltLen   = 16
)

// GenerateSalt creates a random salt
func GenerateSalt() (string, error) {
	b := make([]byte, ScramSaltLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// ScramCredentials holds the stored auth data
type ScramCredentials struct {
	Salt      string
	StoredKey string // Base64 encoded ClientKey (H(SaltedPassword)) xor ClientProof? No.
	// StoredKey = H(ClientKey)
	// ServerKey = HMAC(SaltedPassword, "Server Key")
	// SaltedPassword = PBKDF2(password, salt, iter)
	// ClientKey = HMAC(SaltedPassword, "Client Key")
	ServerKey  string // Base64 encoded
	Iterations int
}

// GenerateCredentials computes the SCRAM secrets for a password
func GenerateCredentials(password, salt string, iterations int) (ScramCredentials, error) {
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return ScramCredentials{}, err
	}

	saltedPassword := PBKDF2([]byte(password), saltBytes, iterations, 32, sha256.New)
	clientKey := computeHMAC(saltedPassword, []byte("Client Key"))
	storedKey := computeHash(clientKey)
	serverKey := computeHMAC(saltedPassword, []byte("Server Key"))

	return ScramCredentials{
		Salt:       salt,
		StoredKey:  base64.StdEncoding.EncodeToString(storedKey),
		ServerKey:  base64.StdEncoding.EncodeToString(serverKey),
		Iterations: iterations,
	}, nil
}

// VerifyClientProof verifies the proof sent by the client
func VerifyClientProof(storedKeyB64, authMessage, clientProofB64 string) bool {
	storedKey, _ := base64.StdEncoding.DecodeString(storedKeyB64)
	clientProof, _ := base64.StdEncoding.DecodeString(clientProofB64)

	clientSignature := computeHMAC(storedKey, []byte(authMessage))
	clientKey := xorBytes(clientProof, clientSignature)
	recoveredStoredKey := computeHash(clientKey)

	return bytes.Equal(storedKey, recoveredStoredKey)
}

// -- Primitives --

func computeHMAC(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func computeHash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

func xorBytes(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	res := make([]byte, n)
	for i := 0; i < n; i++ {
		res[i] = a[i] ^ b[i]
	}
	return res
}

// PBKDF2 implements Password-Based Key Derivation Function 2 (RFC 2898)
// Minimal implementation for SHA-256 to allow zero-dependency
func PBKDF2(password, salt []byte, iter, keyLen int, h func() hash.Hash) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var buf []byte
	dk := make([]byte, 0, numBlocks*hashLen)
	U := make([]byte, hashLen)

	for block := 1; block <= numBlocks; block++ {
		// U_1 = PRF(password, salt || INT_32_BE(block))
		prf.Reset()
		prf.Write(salt)
		buf = make([]byte, 4)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf)
		U = prf.Sum(U[:0])

		// T_block = U_1
		blockKey := make([]byte, len(U))
		copy(blockKey, U)

		// U_2 through U_c
		for i := 2; i <= iter; i++ {
			prf.Reset()
			prf.Write(U)
			U = prf.Sum(U[:0])

			// T_block ^= U_i
			for k := 0; k < len(U); k++ {
				blockKey[k] ^= U[k]
			}
		}
		dk = append(dk, blockKey...)
	}
	return dk[:keyLen]
}

// ComputeClientProof generates the proof for the client to send to server
func ComputeClientProof(password, salt string, iterations int, authMessage string) (string, error) {
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return "", err
	}

	saltedPassword := PBKDF2([]byte(password), saltBytes, iterations, 32, sha256.New)
	clientKey := computeHMAC(saltedPassword, []byte("Client Key"))
	storedKey := computeHash(clientKey)
	clientSignature := computeHMAC(storedKey, []byte(authMessage))
	clientProof := xorBytes(clientKey, clientSignature)

	return base64.StdEncoding.EncodeToString(clientProof), nil
}

// ParseSCRAMMessage parses a minimal SCRAM message "n=user,r=nonce"
// Simplification: We assume standard client-first-message format
func ParseSCRAMMessage(msg string) map[string]string {
	parts := strings.Split(msg, ",")
	res := make(map[string]string)
	for _, part := range parts {
		if len(part) > 2 && part[1] == '=' {
			key := string(part[0])
			val := part[2:]
			res[key] = val
		}
	}
	return res
}
