package util

import "errors"

// Common errors used throughout bundoc
var (
	// Storage errors
	ErrPageNotFound    = errors.New("page not found")
	ErrPageFull        = errors.New("page is full")
	ErrInvalidPageID   = errors.New("invalid page ID")
	ErrDiskReadFailed  = errors.New("disk read failed")
	ErrDiskWriteFailed = errors.New("disk write failed")

	// Transaction errors
	ErrTxnAborted   = errors.New("transaction aborted")
	ErrTxnDeadlock  = errors.New("transaction deadlock detected")
	ErrTxnTimeout   = errors.New("transaction timeout")
	ErrTxnReadOnly  = errors.New("transaction is read-only")
	ErrTxnNotActive = errors.New("transaction is not active")

	// Query errors
	ErrInvalidQuery       = errors.New("invalid query")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrDocumentNotFound   = errors.New("document not found")

	// Database errors
	ErrDatabaseClosed  = errors.New("database is closed")
	ErrDatabaseCorrupt = errors.New("database is corrupt")

	// WAL errors
	ErrWALCorrupt     = errors.New("WAL is corrupt")
	ErrWALSegmentFull = errors.New("WAL segment is full")
)
