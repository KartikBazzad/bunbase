package wal

import "errors"

var (
	ErrPayloadTooLarge = errors.New("payload exceeds maximum size")
	ErrCorruptRecord   = errors.New("corrupt record: invalid length or format")
	ErrCRCMismatch     = errors.New("CRC mismatch")
	ErrFileOpen        = errors.New("failed to open WAL file")
	ErrFileWrite       = errors.New("failed to write WAL file")
	ErrFileSync        = errors.New("failed to sync WAL file")
	ErrFileRead        = errors.New("failed to read WAL file")
)
