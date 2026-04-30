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
	ErrNotFound    = errors.New("resource not found")
	ErrDuplicate   = errors.New("resource already exists")
	ErrDatabase    = errors.New("database operation failed")
	ErrTransaction = errors.New("transaction failed")

	// Validation errors
	ErrInvalidInput  = errors.New("invalid input")
	ErrInvalidEUI    = errors.New("invalid EUI format")
	ErrMissingField  = errors.New("required field missing")
	ErrInvalidFormat = errors.New("invalid format")

	// MIOTY Protocol errors
	ErrInvalidCommand      = errors.New("invalid MIOTY command")
	ErrInvalidOperation    = errors.New("invalid MIOTY operation")
	ErrProtocolViolation   = errors.New("MIOTY protocol violation")
	ErrDeduplication       = errors.New("duplicate message detected")
	ErrSessionNotFound     = errors.New("BSSCI session not found")
	ErrEndpointNotAttached = errors.New("endpoint not attached")

	// Authentication/Authorization errors
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrTokenExpired = errors.New("token expired")
	ErrInvalidToken = errors.New("invalid token")

	// Network/Connection errors
	ErrConnection   = errors.New("connection failed")
	ErrTimeout      = errors.New("operation timed out")
	ErrDisconnected = errors.New("disconnected")

	// Business logic errors
	ErrConflict      = errors.New("resource conflict")
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrRateLimited   = errors.New("rate limited")
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
