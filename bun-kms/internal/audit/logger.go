package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Logger writes audit events to an append-only log.
type Logger struct {
	mu   sync.Mutex
	file *os.File
}

// NewLogger opens an append-only audit log file.
func NewLogger(path string) (*Logger, error) {
	if path == "" {
		return &Logger{}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

// Log writes an audit event. Timestamp is set if zero.
func (l *Logger) Log(evt Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	line, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	_, err = l.file.Write(line)
	return err
}

// Close closes the audit log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}
