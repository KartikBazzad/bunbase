package errors

import (
	"errors"
	"syscall"
)

// ErrorCategory represents the category of an error for retry logic.
type ErrorCategory int

const (
	ErrorTransient  ErrorCategory = iota // Temporary errors - retry with backoff
	ErrorPermanent                       // Permanent errors - no retry
	ErrorCritical                        // System-level errors - alert immediately
	ErrorValidation                      // Data validation errors - no retry
	ErrorNetwork                         // Network-related - retry with backoff
)

// Classifier categorizes errors for intelligent retry logic.
type Classifier struct{}

// NewClassifier creates a new error classifier.
func NewClassifier() *Classifier {
	return &Classifier{}
}

// Classify determines the category of an error.
func (c *Classifier) Classify(err error) ErrorCategory {
	if err == nil {
		return ErrorPermanent // Should not happen, but safe default
	}

	// Check for system-level errors
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		switch sysErr {
		case syscall.EAGAIN, syscall.ENOMEM, syscall.ETIMEDOUT:
			return ErrorTransient
		case syscall.ENOENT, syscall.EINVAL, syscall.EEXIST:
			return ErrorPermanent
		case syscall.EIO, syscall.ENOSPC:
			return ErrorCritical
		}
	}

	// Check for known DocDB errors
	switch err {
	case ErrCorruptRecord, ErrCRCMismatch, ErrInvalidJSON:
		return ErrorValidation
	case ErrFileOpen, ErrFileWrite, ErrFileSync, ErrFileRead:
		// File errors could be transient (EAGAIN) or permanent (ENOENT)
		// We'll treat them as potentially transient for retry logic
		return ErrorTransient
	case ErrPayloadTooLarge, ErrMemoryLimit:
		return ErrorPermanent
	case ErrDocNotFound, ErrDocExists, ErrDBNotOpen:
		return ErrorPermanent
	case ErrPoolStopped, ErrQueueFull:
		return ErrorPermanent
	}

	// Default: treat as permanent (no retry)
	return ErrorPermanent
}

// ShouldRetry returns true if the error category indicates retry is appropriate.
func (c *Classifier) ShouldRetry(category ErrorCategory) bool {
	return category == ErrorTransient || category == ErrorNetwork
}

// IsCritical returns true if the error requires immediate attention.
func (c *Classifier) IsCritical(category ErrorCategory) bool {
	return category == ErrorCritical
}
