package interfaces

import "errors"

// ErrRecordNotFound indicates a query returned no matching rows.
// Repository implementations should wrap this sentinel for errors.Is checks.
var ErrRecordNotFound = errors.New("record not found")
