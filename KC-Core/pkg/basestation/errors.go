package basestation

import "errors"

// Sentinel errors for base-station lookups.
var (
	errBasestationNotFound = errors.New("basestation not found")
)

// Error format strings for wrapped failures; verbs are filled at the point of failure.
const (
	errFmtGetBaseStation         = "failed to get base station: %w"
	errFmtUpdateLastSeen         = "failed to update last seen: %w"
	errFmtUpdateConnectionStatus = "failed to update connection status: %w"
	errFmtDisconnectBaseStation  = "failed to disconnect base station: %w"
	errFmtGetBasestation         = "failed to get basestation: %w"
)
