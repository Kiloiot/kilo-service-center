// Package messages provides message listing service implementation for gRPC layer.
package messages

import "errors"

// Domain-layer sentinel errors for message operations.
// Adapters return these; handlers map to gRPC tokens.

// ErrExportUnsupportedFormat indicates an unsupported export format was requested.
var ErrExportUnsupportedFormat = errors.New("unsupported export format")
