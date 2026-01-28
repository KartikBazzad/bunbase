package docdb

import "github.com/kartikbazzad/docdb/internal/errors"

// Re-export core errors for backward compatibility
// TODO: Directly import internal/errors in v1.0
var (
	ErrDBNotOpen        = errors.ErrDBNotOpen
	ErrDocNotFound      = errors.ErrDocNotFound
	ErrDocExists        = errors.ErrDocExists
	ErrDocAlreadyExists = errors.ErrDocExists
	ErrMemoryLimit      = errors.ErrMemoryLimit
)
