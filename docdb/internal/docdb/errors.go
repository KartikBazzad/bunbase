package docdb

import (
	"errors"

	docdberrors "github.com/kartikbazzad/docdb/internal/errors"
)

// Re-export core errors for backward compatibility
// TODO: Directly import internal/errors in v1.0
var (
	ErrDBNotOpen        = docdberrors.ErrDBNotOpen
	ErrDocNotFound      = docdberrors.ErrDocNotFound
	ErrDocExists        = docdberrors.ErrDocExists
	ErrDocAlreadyExists = docdberrors.ErrDocExists
	ErrMemoryLimit      = docdberrors.ErrMemoryLimit
)

// v0.4 partition and worker pool errors
var (
	ErrPoolStopped      = errors.New("worker pool is stopped")
	ErrQueueFull        = errors.New("task queue is full")
	ErrInvalidPartition = errors.New("invalid partition ID")
)
