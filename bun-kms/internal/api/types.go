package api

// CreateKeyRequest is the payload for creating a new key
type CreateKeyRequest struct {
	Name string `json:"name"`
	Type string `json:"type"` // aes-256, rsa-2048, ecdsa-p256
}

// EncryptRequest payload
type EncryptRequest struct {
	Plaintext string `json:"plaintext"` // Base64 encoded or raw string? Let's say string for MVP, or []byte json
}

// DecryptRequest payload
type DecryptRequest struct {
	Ciphertext string `json:"ciphertext"` // Base64 encoded
}

// SecretRequest payload
type SecretRequest struct {
	Value string `json:"value"`
}
