package types

import "github.com/kartikbazzad/docdb/internal/errors"

// Re-export core errors for backward compatibility
// TODO: Directly import internal/errors in v1.0
var (
	ErrInvalidJSON = errors.ErrInvalidJSON
	ErrDocExists   = errors.ErrDocExists
	ErrDocNotFound = errors.ErrDocNotFound
	ErrDBNotOpen   = errors.ErrDBNotOpen
	ErrMemoryLimit = errors.ErrMemoryLimit
)
