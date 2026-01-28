package wal

import "github.com/kartikbazzad/docdb/internal/errors"

// Re-export WAL errors for backward compatibility
// TODO: Directly import internal/errors in v1.0
var (
	ErrPayloadTooLarge = errors.ErrPayloadTooLarge
	ErrCorruptRecord   = errors.ErrCorruptRecord
	ErrCRCMismatch     = errors.ErrCRCMismatch
	ErrFileOpen        = errors.ErrFileOpen
	ErrFileWrite       = errors.ErrFileWrite
	ErrFileSync        = errors.ErrFileSync
	ErrFileRead        = errors.ErrFileRead
)
