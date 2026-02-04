package logstore

import "time"

// LogEntry represents a single function log line.
type LogEntry struct {
	FunctionID   string    `json:"function_id"`
	InvocationID string    `json:"invocation_id"`
	Level        string    `json:"level"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

// Store is the interface for persisting and querying function logs.
type Store interface {
	Append(functionID, invocationID, level, message string) error
	GetLogs(functionID string, since time.Time, limit int) ([]LogEntry, error)
}
