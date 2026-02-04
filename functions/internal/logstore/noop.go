package logstore

import "time"

// NoopStore is a log store that discards appends and returns no logs. Used when Loki is disabled or in tests.
type NoopStore struct{}

// Append does nothing.
func (NoopStore) Append(_, _, _, _ string) error {
	return nil
}

// GetLogs returns an empty slice.
func (NoopStore) GetLogs(_ string, _ time.Time, _ int) ([]LogEntry, error) {
	return nil, nil
}
