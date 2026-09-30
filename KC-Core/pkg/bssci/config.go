package bssci

import (
	"time"
	// Shared MIOTY helpers (FormatEUI64, EPStatus)
)

// Config holds server configuration
type Config struct {
	ListenAddr                       string
	TLSCert                          string
	TLSKey                           string
	TLSCACert                        string
	TLSMinVersion                    string // "1.2" or "1.3", parsed by config.ParseTLSMinVersion
	ServiceCenterEUI                 uint64
	Vendor                           string
	Model                            string
	Name                             string
	SoftwareVersion                  string
	OrgEnforcementEnabled            bool   // Require valid X-Organization-ID in gRPC metadata
	MessageEncoding                  string // BSSCI Section 1: default message encoding (json/msgpack)
	DetachSignatureValidationEnabled bool   // BSSCI §5.7.1: Enable detach signature validation
	// OperationAckTimeout bounds how long the service center waits for the
	// next handshake message (conCmp or errorAck) after sending conRsp or a
	// connect-stage error (states AwaitingConnectComplete/AwaitingConnectErrorAck).
	OperationAckTimeout time.Duration
	// ConnectionEstablishmentTimeout bounds a freshly accepted connection
	// before the base station sends its con (state AwaitingConnect), so an
	// idle socket cannot hold resources indefinitely.
	ConnectionEstablishmentTimeout time.Duration
	// SocketWriteTimeout bounds every frame write so a base station that
	// stops reading cannot hold a sender; NewServer refuses one that is not
	// positive.
	SocketWriteTimeout time.Duration
	// CertificatePollInterval is the certificate change poll interval
	CertificatePollInterval time.Duration
	// StatusRequestInterval is how often the SC polls a base station for status
	StatusRequestInterval time.Duration
	// StatusRequestInitialDelay delays the first status poll after connect
	StatusRequestInitialDelay time.Duration
	// DLRXQueryTimeout expires an unanswered dlRxStatQry after this duration
	DLRXQueryTimeout time.Duration
	// DLRXCleanupInterval is the dlRxStatQry expiry sweep cadence
	DLRXCleanupInterval time.Duration
}

// operationAckTimeout returns the configured handshake wait bound, falling
// back to the package default when unset.
func (c *Config) operationAckTimeout() time.Duration {
	if c != nil && c.OperationAckTimeout > 0 {
		return c.OperationAckTimeout
	}
	return defaultOperationAckTimeout
}

// connectionEstablishmentTimeout returns the configured bound for a freshly
// accepted connection to send its con, falling back to the package default.
func (c *Config) connectionEstablishmentTimeout() time.Duration {
	if c != nil && c.ConnectionEstablishmentTimeout > 0 {
		return c.ConnectionEstablishmentTimeout
	}
	return defaultConnectionEstablishmentTimeout
}

// statusRequestInterval returns the configured status poll interval, falling
// back to the package default when unset.
func (c *Config) statusRequestInterval() time.Duration {
	if c != nil && c.StatusRequestInterval > 0 {
		return c.StatusRequestInterval
	}
	return defaultStatusRequestInterval
}

// statusRequestInitialDelay returns the configured delay before the first
// status poll, falling back to the package default when unset.
func (c *Config) statusRequestInitialDelay() time.Duration {
	if c != nil && c.StatusRequestInitialDelay > 0 {
		return c.StatusRequestInitialDelay
	}
	return defaultStatusRequestInitialDelay
}

// dlrxQueryTimeout returns the configured dlRxStatQry expiry, falling back to
// the package default when unset.
func (c *Config) dlrxQueryTimeout() time.Duration {
	if c != nil && c.DLRXQueryTimeout > 0 {
		return c.DLRXQueryTimeout
	}
	return defaultDLRXQueryTimeout
}

// dlrxCleanupInterval returns the configured dlRxStatQry expiry sweep cadence,
// falling back to the package default when unset.
func (c *Config) dlrxCleanupInterval() time.Duration {
	if c != nil && c.DLRXCleanupInterval > 0 {
		return c.DLRXCleanupInterval
	}
	return defaultDLRXCleanupInterval
}
