package audit

import "time"

// Operation is the type of audited operation.
type Operation string

const (
	KeyCreated    Operation = "key_created"
	KeyRotated    Operation = "key_rotated"
	KeyRevoked    Operation = "key_revoked"
	KeyGet        Operation = "key_get"
	DataEncrypted Operation = "data_encrypted"
	DataDecrypted Operation = "data_decrypted"
	SecretPut     Operation = "secret_put"
	SecretGet     Operation = "secret_get"
)

// Event is a single audit log entry.
type Event struct {
	Timestamp time.Time         `json:"timestamp"`
	Operation Operation         `json:"operation"`
	Actor     string            `json:"actor,omitempty"`
	Resource  string            `json:"resource"`
	Success   bool              `json:"success"`
	Message   string            `json:"message,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
