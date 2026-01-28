package capabilities

import "errors"

var (
	ErrInvalidMemoryLimit          = errors.New("invalid memory limit")
	ErrInvalidFileDescriptorLimit  = errors.New("invalid file descriptor limit")
	ErrCapabilityNotAllowed        = errors.New("capability not allowed")
	ErrPathNotAllowed              = errors.New("path not allowed")
	ErrDomainNotAllowed            = errors.New("domain not allowed")
)
