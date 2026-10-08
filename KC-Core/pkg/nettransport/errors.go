package nettransport

import "errors"

// Sentinels callers match with errors.Is; the wrapping formats below add the
// numbers a log line needs.
var (
	ErrHeaderTooShort          = errors.New("header too short")
	ErrInvalidFrameIdentifier  = errors.New("invalid frame identifier")
	ErrPayloadTooLarge         = errors.New("payload size too large")
	ErrShortWrite              = errors.New("short write")
	ErrWriteTimeoutRequired    = errors.New("frame codec has no write timeout")
	ErrUnsupportedCipherSuite  = errors.New("unsupported cipher suite")
	ErrInvalidCertPollInterval = errors.New("certificate poll interval must be positive")
	ErrListenerAlreadyStarted  = errors.New("listener already started")
	ErrListenerStopped         = errors.New("listener stopped")
)

const (
	errFmtHeaderTooShort          = "%w: need %d bytes, got %d"
	errFmtInvalidIdentifier       = "%w: expected %q, got %q"
	errFmtPayloadTooLarge         = "%w: %d bytes exceeds maximum %d bytes"
	errFmtReadPayload             = "read frame payload: %w"
	errFmtShortWrite              = "%w: wrote %d of %d bytes"
	errFmtWriteTimeoutNotPositive = "%w: %s is not positive"
	errFmtSetWriteDeadline        = "set write deadline: %w"
	errFmtClearWriteDeadline      = "clear write deadline: %w"
	errFmtLoadServerCertificate   = "failed to load server certificate: %w"
	errFmtReadCACertificate       = "failed to read CA certificate: %w"
	errFmtNoValidCACertificates   = "no valid CA certificates found in %s"
	errFmtTLSMinVersion           = "TLS configuration: %w"
	errFmtUnsupportedCipher       = "%w: %s"
	errFmtTLSConfig               = "TLS configuration failed: %w"
	errFmtListen                  = "failed to create TLS listener on %s: %w"
)
