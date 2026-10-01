// Package config provides configuration types for KiloCenter modules
package config

import "time"

// ProtocolConfig contains MIOTY protocol settings
type ProtocolConfig struct {
	// Base Station Service Center Interface (BSSCI)
	BSCIHost        string    `mapstructure:"bsci_host"`
	BSCIPort        int       `mapstructure:"bsci_port"`
	BSCIExternalURL string    `mapstructure:"bsci_external_url"` // Client-facing URL (e.g., tls://bssci.example.com:<port>); if empty, tls://bsci_host:bsci_port; a wildcard or loopback host gives stations no URL
	BSCITLS         TLSConfig `mapstructure:"bsci_tls"`
	// ManagementPort is the loopback port of the internal BSSCI management API
	// (attach/detach propagation endpoints). Each instance on a shared host
	// needs its own value.
	ManagementPort int `mapstructure:"management_port"`

	// BSSCI Security Settings
	DetachSignatureValidationEnabled bool `mapstructure:"detach_signature_validation_enabled"` // Enable cryptographic validation of detach signatures (default: true)
	// DetachValidationAPIURL and DetachValidationSharedToken removed - KC-Core now self-contained

	// Service Center to Application Center Interface
	SCACIEnabled           bool      `mapstructure:"scaci_enabled"`
	SCACIHost              string    `mapstructure:"scaci_host"`
	SCACIPort              int       `mapstructure:"scaci_port"`
	SCACITLS               TLSConfig `mapstructure:"scaci_tls"`
	SCACICertTenantMapping bool      `mapstructure:"scaci_cert_tenant_mapping"`
	StrictOrgResolution    bool      `mapstructure:"strict_org_resolution"`       // Fail-closed on org resolution failure (default: false for community builds)
	SCALogPingOperations   bool      `mapstructure:"scaci_log_ping_operations"`   // Log Ping operations to audit trail (default: true)
	SCALogStatusOperations bool      `mapstructure:"scaci_log_status_operations"` // Log Status operations to audit trail (default: true)
	// SCACIResumeMaxPendingOperations bounds the service center operations a
	// disconnected Application Center session holds for its resume (SCACI §1).
	SCACIResumeMaxPendingOperations int `mapstructure:"scaci_resume_max_pending_operations"`

	// Protocol Parameters
	AckTimeout                     int    `mapstructure:"ack_timeout"`                      // milliseconds
	ConnectionEstablishmentTimeout int    `mapstructure:"connection_establishment_timeout"` // milliseconds
	SocketWriteTimeout             int    `mapstructure:"socket_write_timeout"`             // milliseconds; bounds every BSSCI and SCACI frame write
	DuplicateWindow                int    `mapstructure:"duplicate_window"`                 // seconds
	MessageEncoding                string `mapstructure:"message_encoding"`                 // json, msgpack
	StatusRequestInterval          int    `mapstructure:"status_request_interval"`          // seconds
	StatusRequestInitialDelay      int    `mapstructure:"status_request_initial_delay"`     // seconds
	DLRXQueryTimeout               int    `mapstructure:"dlrx_query_timeout"`               // seconds
	DLRXCleanupInterval            int    `mapstructure:"dlrx_cleanup_interval"`            // seconds

	// BSCICertificatePollInterval is the base station certificate change poll interval
	BSCICertificatePollInterval time.Duration `mapstructure:"bsci_certificate_poll_interval"`

	Roaming RoamingConfig `mapstructure:"roaming"`

	// Delivery bounds the outbox worker that fans stored uplinks out to SCACI and MQTT.
	Delivery DeliveryConfig `mapstructure:"delivery"`

	// DownlinkExpiry bounds how long a downlink waits in the queue and how its expiry is swept.
	DownlinkExpiry DownlinkExpiryConfig `mapstructure:"downlink_expiry"`

	// Federation controls CE↔ECE cooperative roaming relay
	Federation FederationConfig `mapstructure:"federation"`

	// Service Center Identity - exposed via GetReleaseInfo
	SCVendor string `mapstructure:"sc_vendor"` // Service Center vendor name (e.g., "Kilo")
	SCModel  string `mapstructure:"sc_model"`  // Service Center model (e.g., "KiloCenter")

	// SCEUI is the Service Center EUI as a canonical 16-hex string. Load resolves it with
	// precedence: KILOCENTER_PROTOCOL_SC_EUI env var > explicit file value >
	// legacy SERVICE_CENTER_EUI env var > centralized default.
	SCEUI string `mapstructure:"sc_eui"`
	// SCEUIValue is the numeric form of SCEUI, derived during Load so consumers never re-parse.
	SCEUIValue uint64 `mapstructure:"-" yaml:"-"`
	// SCEUILegacyEnvUsed records that the deprecated SERVICE_CENTER_EUI variable supplied the
	// EUI; the first consumer emits the deprecation warning since Load has no logger.
	SCEUILegacyEnvUsed bool `mapstructure:"-" yaml:"-"`
}

// RoamingConfig contains multi-tenant roaming settings
type RoamingConfig struct {
	Enabled          bool          `mapstructure:"enabled"`            // Enable roaming support (default: false)
	CacheEnabled     bool          `mapstructure:"cache_enabled"`      // Enable ownership cache (default: true)
	CacheTTL         time.Duration `mapstructure:"cache_ttl"`          // Cache TTL (default: 5m)
	CacheMaxSize     int           `mapstructure:"cache_max_size"`     // Max cache entries (default: 10000)
	EnableAuditTrail bool          `mapstructure:"enable_audit_trail"` // Record roaming events (default: true)
}

// DeliveryConfig tunes the message delivery outbox worker.
type DeliveryConfig struct {
	PollInterval time.Duration `mapstructure:"poll_interval"` // how often due rows are claimed (default: 1s)
	BatchSize    int           `mapstructure:"batch_size"`    // rows claimed per poll (default: 50)
	RetryBackoff time.Duration `mapstructure:"retry_backoff"` // base of the exponential backoff (default: 2s)
	MaxBackoff   time.Duration `mapstructure:"max_backoff"`   // cap of the backoff between retries (default: 5m)
	// ReceptionWindow is how long a new uplink waits for the receptions of the
	// other base stations before it is delivered (default: 500ms).
	ReceptionWindow time.Duration `mapstructure:"reception_window"`
}

// DownlinkExpiryConfig tunes the downlink lifetime and the sweep that expires overdue downlinks.
type DownlinkExpiryConfig struct {
	Lifetime      time.Duration `mapstructure:"lifetime"`       // how long a queued downlink waits for a downlink window (default: 24h)
	SweepInterval time.Duration `mapstructure:"sweep_interval"` // how often overdue downlinks are expired and reported (default: 5s)
	BatchSize     int           `mapstructure:"batch_size"`     // downlinks expired per statement (default: 100)
	// RevokeNotHeldCodes are the POSIX codes of a dlDataRev error answer that
	// say the base station does not hold the downlink, so it ends as revoked
	// or, when its lifetime ended, expired; any other refusal leaves it in
	// flight (default: [2], ENOENT).
	RevokeNotHeldCodes []int `mapstructure:"revoke_not_held_codes"`
}

// FederationTLSConfig holds TLS settings for the CE→ECE relay transport.
type FederationTLSConfig struct {
	// Enabled activates mutual TLS on the federation relay gRPC connection.
	Enabled bool `mapstructure:"enabled"`
	// CertFile is the path to the client certificate presented to ECE.
	CertFile string `mapstructure:"cert_file"`
	// KeyFile is the path to the client private key.
	KeyFile string `mapstructure:"key_file"`
	// CAFile is the path to the CA certificate used to verify the ECE server certificate.
	CAFile string `mapstructure:"ca_file"`
	// InsecureSkipVerify disables server certificate verification (development only).
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

// FederationConfig controls CE↔ECE cooperative roaming.
// CE instances use this to configure the outbound relay stream to an ECE endpoint.
// ECE federation-ingress uses the synthetic_bs_eui to stamp relayed uplinks.
type FederationConfig struct {
	// Enabled activates the federation relay client (CE) or ingress acceptor (ECE federation binary).
	Enabled bool `mapstructure:"enabled"`
	// ECEEndpoint is the gRPC address of the ECE federation-ingress service (CE mode only).
	ECEEndpoint string `mapstructure:"ece_endpoint"`
	// HeartbeatInterval controls how often CE sends heartbeats on the Connect stream.
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	// ReconnectMaxBackoff limits the exponential reconnect delay.
	ReconnectMaxBackoff time.Duration `mapstructure:"reconnect_max_backoff"`
	// SyntheticBsEUI is the reserved BS EUI written into tenant-visible message records for
	// federation-relayed uplinks. It must never appear in the base_stations table.
	// Configured on the ECE federation-ingress; ignored in CE mode.
	SyntheticBsEUI string `mapstructure:"synthetic_bs_eui"`
	// IngressGRPCPort is the port the ECE federation-ingress gRPC server listens on.
	IngressGRPCPort int `mapstructure:"ingress_grpc_port"`
	// RevocationPollInterval controls how often the ingress service checks the DB for
	// revoked CE instances. Defaults to 10s.
	RevocationPollInterval time.Duration `mapstructure:"revocation_poll_interval"`
	// TLS configures mutual TLS for the CE→ECE relay transport (CE mode only).
	TLS FederationTLSConfig `mapstructure:"tls"`
}
