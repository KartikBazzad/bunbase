package security

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// EventType defines the category of audit event
type EventType string

const (
	EventLoginSuccess EventType = "LOGIN_SUCCESS"
	EventLoginFailure EventType = "LOGIN_FAILURE"
	EventUserCreated  EventType = "USER_CREATED"
	EventUserUpdated  EventType = "USER_UPDATED"
	EventUserDeleted  EventType = "USER_DELETED"
	EventAccessDenied EventType = "ACCESS_DENIED"
	EventSystemStart  EventType = "SYSTEM_START"
)

// AuditEvent represents a single loggable security event
type AuditEvent struct {
	Timestamp time.Time              `json:"ts"`
	Type      EventType              `json:"type"`
	User      string                 `json:"user,omitempty"`
	RemoteIP  string                 `json:"ip,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// AuditLogger handles writing audit events
type AuditLogger struct {
	file *os.File
	mu   sync.Mutex
}

// NewAuditLogger creates a new logger writing to the specified path
func NewAuditLogger(path string) (*AuditLogger, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	return &AuditLogger{
		file: file,
	}, nil
}

// Log records an event
func (l *AuditLogger) Log(evtType EventType, user, ip string, details map[string]interface{}) {
	if l.file == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	event := AuditEvent{
		Timestamp: time.Now().UTC(),
		Type:      evtType,
		User:      user,
		RemoteIP:  ip,
		Details:   details,
	}

	encoder := json.NewEncoder(l.file)
	if err := encoder.Encode(event); err != nil {
		// Fallback to stderr if audit log fails (critical)
		fmt.Fprintf(os.Stderr, "CRITICAL: Failed to write audit log: %v\n", err)
	}
}

// Close closes the log file
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// DiscardLogger returns a logger that writes nowhere (for testing/default)
func DiscardLogger() *AuditLogger {
	return &AuditLogger{
		file: nil, // Handled in Log check? No, need to fix Log
	}
}
