package errors

import (
	"errors"
)

// File I/O errors - used by both WAL and data files
var (
	// ErrInvalidJSON is returned when payload is not valid UTF-8 JSON.
	// The message intentionally mentions forbidden prefixes used by the shell
	// parser (raw:, hex:) so tests and users get a clear hint while the error
	// value itself remains stable.
	ErrInvalidJSON = errors.New("payload must be valid JSON (forbidden prefixes: raw: hex:)")

	// ErrDocExists is returned when creating a document that already exists
	ErrDocExists = errors.New("document already exists")

	// ErrDocNotFound is returned when reading/deleting non-existent document
	ErrDocNotFound = errors.New("document not found")

	// ErrDBNotOpen is returned when operating on closed database
	ErrDBNotOpen = errors.New("database not open")

	// ErrMemoryLimit is returned when memory limit is exceeded
	ErrMemoryLimit = errors.New("memory limit exceeded")

	// ErrPayloadTooLarge is returned when payload exceeds maximum size
	ErrPayloadTooLarge = errors.New("payload exceeds maximum size")

	// ErrCorruptRecord is returned when WAL record has invalid format
	ErrCorruptRecord = errors.New("corrupt record: invalid length or format")

	// ErrCRCMismatch is returned when CRC32 checksum doesn't match
	ErrCRCMismatch = errors.New("CRC mismatch")

	// ErrFileOpen is returned when WAL file cannot be opened
	ErrFileOpen = errors.New("failed to open file")

	// ErrFileWrite is returned when WAL file cannot be written
	ErrFileWrite = errors.New("failed to write file")

	// ErrFileSync is returned when WAL file cannot be synced
	ErrFileSync = errors.New("failed to sync file")

	// ErrFileRead is returned when WAL file cannot be read
	ErrFileRead = errors.New("failed to read file")

	// ErrPoolStopped is returned when pool is stopping down
	ErrPoolStopped = errors.New("pool is stopped")

	// ErrQueueFull is returned when request queue is at capacity
	ErrQueueFull = errors.New("request queue is full")

	// ErrInvalidRequestID is returned when request ID is invalid
	ErrInvalidRequestID = errors.New("invalid request ID")

	// ErrFrameTooLarge is returned when IPC frame exceeds maximum size
	ErrFrameTooLarge = errors.New("frame size exceeds maximum")

	// ErrDBNotActive is returned when database is not in active state
	ErrDBNotActive = errors.New("database is not active")

	// ErrUnknownOperation is returned when operation type is invalid
	ErrUnknownOperation = errors.New("unknown operation type")

	// Collection errors
	ErrCollectionNotFound = errors.New("collection not found")
	ErrCollectionExists   = errors.New("collection already exists")
	ErrCollectionNotEmpty = errors.New("collection is not empty")

	// Path-based update errors
	ErrInvalidPath   = errors.New("invalid JSON path")
	ErrNotJSONObject = errors.New("document is not a JSON object")
	ErrInvalidPatch  = errors.New("invalid patch operations")
)
