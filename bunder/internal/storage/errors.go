package storage

import "errors"

// Storage layer errors.
var (
	ErrInvalidPageID = errors.New("invalid page ID")
	ErrPageNotFound  = errors.New("page not found")
	ErrPageFull      = errors.New("buffer pool full")
	ErrDiskRead      = errors.New("disk read failed")
	ErrDiskWrite     = errors.New("disk write failed")
)
