package docdb

// Concurrent write to same document: undefined but safe
// Behavior: "Last commit wins"
// - No conflict detection
// - Both versions are persisted in WAL
// - Index shows last committed version
// - Non-deterministic across restarts but correct for v0
//
// This is safe because:
// - ACID properties are maintained
// - Reads see last committed version
// - No silent data loss (both versions in WAL)
// - Deterministic if writes are serialized

import (
	"errors"
)

var (
	ErrDBNotOpen        = errors.New("database not open")
	ErrDocNotFound      = errors.New("document not found")
	ErrDocAlreadyExists = errors.New("document already exists")
	ErrMemoryLimit      = errors.New("memory limit exceeded")
)
