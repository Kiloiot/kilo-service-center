// Package mocks provides test doubles for storage interfaces.
package mocks

import "errors"

// ErrMockNotConfiguredMsg is the message for unconfigured mock functions.
const ErrMockNotConfiguredMsg = "mock function not configured"

// ErrMockNotConfigured is returned when a mock function pointer is nil.
var ErrMockNotConfigured = errors.New(ErrMockNotConfiguredMsg)
