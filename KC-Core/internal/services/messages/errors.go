// Package messages provides message listing service implementation for gRPC layer.
package messages

import "errors"

// Domain-layer sentinel errors for message operations.
// Adapters return these; handlers map to gRPC tokens.

// ErrExportUnsupportedFormat indicates an unsupported export format was requested.
var ErrExportUnsupportedFormat = errors.New("unsupported export format")

// ErrMessageNotOwnedByBaseStation indicates a message that does not belong to the requested base station.
var ErrMessageNotOwnedByBaseStation = errors.New("message does not belong to requested base station")

// Operation sentinels wrapping message store failures.
var (
	// ErrGetMessage wraps store failures while loading a message.
	ErrGetMessage = errors.New("get message")
	// ErrListMessages wraps store failures while listing messages.
	ErrListMessages = errors.New("list messages")
	// ErrListBaseStationMessages wraps store failures while listing base station messages.
	ErrListBaseStationMessages = errors.New("list base station messages")
	// ErrGetMessageStats wraps store failures while computing message statistics.
	ErrGetMessageStats = errors.New("get message stats")
	// ErrSearchMessages wraps store failures while searching messages.
	ErrSearchMessages = errors.New("search messages")
	// ErrExportMessages wraps store failures while exporting messages.
	ErrExportMessages = errors.New("export messages")
)
