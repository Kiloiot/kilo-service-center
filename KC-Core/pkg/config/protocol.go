// Package config provides configuration types for KiloCenter modules
package config

import "time"

// ProtocolConfig contains MIOTY protocol settings
type ProtocolConfig struct {
	// Base Station Service Center Interface (BSSCI)
	BSCIHost        string    `mapstructure:"bsci_host"`
	BSCIPort        int       `mapstructure:"bsci_port"`
	BSCIExternalURL string    `mapstructure:"bsci_external_url"` // Client-facing URL (e.g., tls://bssci.example.com:5000); if empty, falls back to tls://bsci_host:bsci_port
	BSCITLS         TLSConfig `mapstructure:"bsci_tls"`

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

	// Protocol Parameters
	MaxRetransmissions int    `mapstructure:"max_retransmissions"`
	AckTimeout         int    `mapstructure:"ack_timeout"`      // milliseconds
	DuplicateWindow    int    `mapstructure:"duplicate_window"` // seconds
	MessageEncoding    string `mapstructure:"message_encoding"` // json, msgpack

	// Automatic Propagation Reconciliation
	Propagation PropagationConfig `mapstructure:"propagation"`

	Roaming RoamingConfig `mapstructure:"roaming"`

	// Federation controls CE↔ECE cooperative roaming relay
	Federation FederationConfig `mapstructure:"federation"`

	// Service Center Identity - exposed via GetReleaseInfo
	SCVendor string `mapstructure:"sc_vendor"` // Service Center vendor name (e.g., "Kilo")
	SCModel  string `mapstructure:"sc_model"`  // Service Center model (e.g., "KiloCenter")
}

// PropagationConfig contains automatic endpoint propagation settings
type PropagationConfig struct {
	BatchSize       int           `mapstructure:"batch_size"`        // Endpoints per batch (default: 500)
	InterBatchDelay time.Duration `mapstructure:"inter_batch_delay"` // Delay between batches (default: 100ms)
	MaxRetries      int           `mapstructure:"max_retries"`       // Max retry attempts before pause (default: 3)
	RetryBackoff    time.Duration `mapstructure:"retry_backoff"`     // Initial backoff duration (default: 30s)
	CoolDown        time.Duration `mapstructure:"cool_down"`         // Auto-reset after pause (default: 2h, 0=disabled)
}

// RoamingConfig contains multi-tenant roaming settings
type RoamingConfig struct {
	Enabled          bool          `mapstructure:"enabled"`            // Enable roaming support (default: false)
	CacheEnabled     bool          `mapstructure:"cache_enabled"`      // Enable ownership cache (default: true)
	CacheTTL         time.Duration `mapstructure:"cache_ttl"`          // Cache TTL (default: 5m)
	CacheMaxSize     int           `mapstructure:"cache_max_size"`     // Max cache entries (default: 10000)
	EnableAuditTrail bool          `mapstructure:"enable_audit_trail"` // Record roaming events (default: true)
	EnableMetrics    bool          `mapstructure:"enable_metrics"`     // Track roaming metrics (default: true)
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
	// OutboxMaxSize caps the number of unacknowledged relay records in the durable outbox.
	OutboxMaxSize int `mapstructure:"outbox_max_size"`
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
