package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a standardized application error
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"` // Internal error for logging
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// New creates a new AppError
func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NotFound creates a 404 error
func NotFound(message string) *AppError {
	return New(http.StatusNotFound, message, nil)
}

// BadRequest creates a 400 error
func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, message, nil)
}

// Internal creates a 500 error
func Internal(err error) *AppError {
	return New(http.StatusInternalServerError, "Internal Server Error", err)
}

// Unauthorized creates a 401 error
func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message, nil)
}
