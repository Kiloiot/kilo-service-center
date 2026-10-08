// Package errors provides centralized error definitions for KiloCenter
// All modules should use these error definitions instead of creating their own
package errors

import (
	"errors"
	"fmt"
)

// Domain Errors - Use these throughout the application
var (
	// Database errors
	ErrNotFound  = errors.New("resource not found")
	ErrDuplicate = errors.New("resource already exists")
	ErrDatabase  = errors.New("database operation failed")

	// Validation errors
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidEUI   = errors.New("invalid EUI format")
	ErrMissingField = errors.New("required field missing")
)

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted context
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
