package docdb

import (
	"errors"
)

var (
	ErrDBNotOpen        = errors.New("database not open")
	ErrDocNotFound      = errors.New("document not found")
	ErrDocAlreadyExists = errors.New("document already exists")
	ErrMemoryLimit      = errors.New("memory limit exceeded")
)
