// Package config provides configuration types and loading for KiloCenter modules.
package config

import "time"

const (
	// StorageTypePostgres is the supported storage type identifier.
	StorageTypePostgres = "postgres"

	// DefaultStorageType is the default storage type when unspecified.
	DefaultStorageType = StorageTypePostgres

	// DefaultStorageUsername is the default database username for Postgres.
	DefaultStorageUsername = "postgres"

	// Auth-prefixed constants for local authentication.
	// These use Auth prefix to avoid collisions with existing configuration.

	// AuthDefaultLoginLabel is the default label for the login button.
	AuthDefaultLoginLabel = "Sign In"

	// AuthDefaultIssuer is the default JWT issuer for local login tokens.
	AuthDefaultIssuer = "kilocenter"

	// AuthDefaultAudience is the default JWT audience for local login tokens.
	AuthDefaultAudience = "kilocenter-api"

	// AuthHMACSecretMinLength is the minimum length in bytes for HMAC signing keys.
	// 32 bytes (256 bits) provides adequate security for HMAC-SHA256/512.
	AuthHMACSecretMinLength = 32

	// AuthPasswordMinLength is the minimum password length in characters (runes).
	// 8 characters is a baseline for acceptable password security.
	AuthPasswordMinLength = 8

	// AuthPasswordMaxLength is the maximum password length in characters (runes).
	// 128 characters prevents DoS via overly large payloads while allowing strong passwords.
	AuthPasswordMaxLength = 128

	// PBKDF2 password hashing constants.
	// PBKDF2-SHA512 with 64-byte key length (matches auth_password.go implementation).
	// OWASP 2024 recommendation for PBKDF2-SHA512 is 210,000+ iterations.
	// 120,000 is acceptable minimum; tune based on hardware benchmarks.

	// AuthPBKDF2Iterations is the iteration count for PBKDF2-SHA512.
	// OWASP 2024 recommendation for PBKDF2-SHA512 is 210,000+ iterations.
	AuthPBKDF2Iterations = 210000

	// AuthPBKDF2SaltLength is the salt length in bytes (16 bytes = 128 bits).
	AuthPBKDF2SaltLength = 16

	// AuthPBKDF2KeyLength is the derived key length in bytes.
	// MUST be 64 bytes to match SHA-512 output and existing password hashes.
	AuthPBKDF2KeyLength = 64

	// AuthEmailMaxLength is the maximum email address length per RFC 5321.
	// 254 characters is the SMTP path limit.
	AuthEmailMaxLength = 254

	// AuthNameMaxLength is the maximum length for first/last name fields.
	AuthNameMaxLength = 256

	// AuthCompanyMaxLength is the maximum length for company name field.
	AuthCompanyMaxLength = 256

	// AuthEmailRegexPattern is the RFC 5322 simplified email validation pattern.
	// Validates basic email format: local-part@domain.tld
	// Pattern allows alphanumerics plus common special chars in local part,
	// standard domain format with 2+ char TLD.
	// Rationale: Simplified pattern avoids overly restrictive validation while
	// catching obvious formatting errors. Does not attempt full RFC compliance.
	AuthEmailRegexPattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`

	// =========================================================================
	// External auth provider constants
	// =========================================================================

	// AuthDefaultStateTTLString is the default TTL for OIDC/OAuth2 state parameters.
	// Used by v.SetDefault in loader.go (parsed as time.Duration).
	// Must stay consistent with AuthDefaultStateTTL time.Duration constant.
	AuthDefaultStateTTLString = "5m"

	// AuthDefaultNonceTTLString is the default TTL for OIDC nonce parameters.
	// Used by v.SetDefault in loader.go (parsed as time.Duration).
	// Must stay consistent with AuthDefaultNonceTTL time.Duration constant.
	AuthDefaultNonceTTLString = "5m"

	// AuthDefaultPKCEMethod is the default PKCE code challenge method.
	// S256 (SHA-256) is recommended over "plain" for security.
	AuthDefaultPKCEMethod = "S256"

	// AuthDefaultUserIDClaim is the default JWT claim for user identifier.
	// "sub" (subject) is the standard OIDC/JWT claim for user ID.
	AuthDefaultUserIDClaim = "sub"

	// AuthDefaultEmailClaim is the default claim for user email address.
	AuthDefaultEmailClaim = "email"

	// AuthOIDCLoginLabel is the default login button label for OIDC providers.
	AuthOIDCLoginLabel = "Sign in with SSO"

	// AuthOAuth2LoginLabel is the default login button label for OAuth2 providers.
	AuthOAuth2LoginLabel = "Sign in with Provider"

	// AuthRedisKeyPrefixOIDC is the Redis key prefix for OIDC state/nonce storage.
	AuthRedisKeyPrefixOIDC = "auth:oidc:"

	// AuthRedisKeyPrefixOAuth2 is the Redis key prefix for OAuth2 state/verifier storage.
	AuthRedisKeyPrefixOAuth2 = "auth:oauth2:"

	// AuthScopeOpenID is the OIDC required scope for ID token.
	AuthScopeOpenID = "openid"

	// AuthScopeEmail is the scope requesting email claim.
	AuthScopeEmail = "email"

	// AuthScopeProfile is the scope requesting profile claims (name, etc.).
	AuthScopeProfile = "profile"

	// DefaultAuthRegistrationEnabled defaults to false so deployments without
	// explicit local_login_enabled=true do not accidentally expose registration.
	DefaultAuthRegistrationEnabled = false

	// DefaultGatewayRateLimitEnabled enables registration rate limiting by default.
	DefaultGatewayRateLimitEnabled = true

	// DefaultGatewayRateLimitRequestsPerMin is the per-IP requests per minute for rate-limited endpoints.
	DefaultGatewayRateLimitRequestsPerMin = 5

	// DefaultGatewayRateLimitBurst is the per-IP burst allowance for rate-limited endpoints.
	DefaultGatewayRateLimitBurst = 3

	// DefaultGatewayRateLimitCleanupStr is the stale entry cleanup interval.
	DefaultGatewayRateLimitCleanupStr = "10m"

	// DefaultGatewayUpstreamAddress is the default KC-Core internal gRPC address.
	DefaultGatewayUpstreamAddress = "localhost:50051"

	// DefaultGatewayIdentityAddress is the default KC-Identity gRPC address.
	DefaultGatewayIdentityAddress = "localhost:50052"

	// DefaultGatewayHealthPort is the default port for the gateway health endpoint.
	DefaultGatewayHealthPort = 8087

	// DefaultIdentityAddress is the default KC-Identity gRPC address for KC-Core.
	DefaultIdentityAddress = "localhost:50052"

	// DefaultInternalAuthPeerSecret is the default peer secret: none, which
	// only an internal hop on a loopback address may run without.
	DefaultInternalAuthPeerSecret = ""

	// InternalAuthPeerSecretMinLength is the shortest peer secret an internal
	// gRPC hop other hosts reach accepts (openssl rand -hex 32 gives 64).
	InternalAuthPeerSecretMinLength = 32

	// LoopbackHostname names this machine in a bind or dial address.
	LoopbackHostname = "localhost"
)

// =========================================================================
// Gateway Resilience Defaults
// =========================================================================

const (
	// DefaultGatewayResilienceDialTimeoutStr is the upstream gRPC dial timeout.
	DefaultGatewayResilienceDialTimeoutStr = "5s"

	// DefaultGatewayResilienceRPCTimeoutStr is the per-RPC deadline for proxied calls.
	DefaultGatewayResilienceRPCTimeoutStr = "30s"

	// DefaultGatewayResilienceMaxRetries is the max retry attempts for idempotent RPCs.
	DefaultGatewayResilienceMaxRetries = 3

	// DefaultGatewayResilienceRetryBackoffStr is the initial retry backoff.
	DefaultGatewayResilienceRetryBackoffStr = "100ms"

	// DefaultGatewayResilienceRetryMaxBackoffStr is the maximum retry backoff.
	DefaultGatewayResilienceRetryMaxBackoffStr = "1s"

	// DefaultGatewayResilienceCBMaxRequests is the half-open state max probe requests.
	DefaultGatewayResilienceCBMaxRequests = 3

	// DefaultGatewayResilienceCBIntervalStr is the closed-state failure counting interval.
	DefaultGatewayResilienceCBIntervalStr = "60s"

	// DefaultGatewayResilienceCBTimeoutStr is the open-to-half-open recovery timeout.
	DefaultGatewayResilienceCBTimeoutStr = "30s"

	// DefaultGatewayResilienceCBFailureThreshold is the consecutive failures to trip the breaker.
	DefaultGatewayResilienceCBFailureThreshold = 5
)

// Time duration constants (cannot be const, must be var for time arithmetic)
var (
	// AuthRedisConnectTimeout is the timeout for Redis connection attempts.
	// 5 seconds provides reasonable balance between responsiveness and reliability.
	AuthRedisConnectTimeout = 5 * time.Second
)

// Slice defaults (slices cannot be constants in Go)
var (
	// AuthDefaultOIDCScopes is the default set of scopes for OIDC providers.
	// openid is required, email and profile provide user identity info.
	AuthDefaultOIDCScopes = []string{AuthScopeOpenID, AuthScopeEmail, AuthScopeProfile}

	// AuthDefaultOAuth2Scopes is the default set of scopes for OAuth2 providers.
	// email and profile provide user identity info (no openid for non-OIDC).
	AuthDefaultOAuth2Scopes = []string{AuthScopeEmail, AuthScopeProfile}
)

// =========================================================================
// General Defaults
// =========================================================================

const (
	// DefaultGeneralServerName is the default server name for identification.
	DefaultGeneralServerName = "KiloCenter"

	// DefaultGeneralEnvironment is the default runtime environment.
	DefaultGeneralEnvironment = "development"

	// EnvironmentProduction is the production environment selector.
	EnvironmentProduction = "production"

	// ProbeLoopbackHost is the host same-process status probes dial when a
	// listener is bound to a wildcard address.
	ProbeLoopbackHost = "localhost"

	// Bind addresses that mean "every interface".
	wildcardHostIPv4 = "0.0.0.0"
	wildcardHostIPv6 = "::"

	// DefaultGeneralLogLevel is the default logging verbosity.
	DefaultGeneralLogLevel = "info"

	// DefaultGeneralLogFormat is the default log output format.
	DefaultGeneralLogFormat = "json"

	// DefaultGeneralHealthCheckPort is the default port for health endpoints.
	DefaultGeneralHealthCheckPort = 8086

	// DefaultGeneralTenantID is the default tenant ID for single-tenant mode.
	DefaultGeneralTenantID = 1

	// DefaultGeneralOrgEnforcementEnabled controls strict org-based isolation.
	DefaultGeneralOrgEnforcementEnabled = false

	// DefaultGeneralOrgCacheTTLMinutes is the TTL for org cache entries.
	DefaultGeneralOrgCacheTTLMinutes = 5

	// DefaultGeneralOrgCacheMaxEntries is the maximum org cache size.
	DefaultGeneralOrgCacheMaxEntries = 10000

	// DefaultEndpointActivityWindowHours is the threshold for considering an endpoint "active".
	// Endpoints seen within this window are reported as "active" in the API.
	DefaultEndpointActivityWindowHours = 24
)

// =========================================================================
// Edition Constants
// =========================================================================

const (
	// EditionECE identifies the Enterprise Cloud Edition feature set.
	EditionECE = "ece"
	// EditionCommunity identifies the Community Edition feature set (federation relay enabled).
	EditionCommunity = "ce"

	// MaxPortNumber is the highest valid TCP/UDP port.
	MaxPortNumber = 65535

	// MQTTSchemeTCP is the plain MQTT broker URL scheme.
	MQTTSchemeTCP = "tcp"
	// MQTTSchemeSSL is the TLS MQTT broker URL scheme.
	MQTTSchemeSSL = "ssl"

	// TLSVersionValue12 selects TLS 1.2 as the configured minimum version.
	TLSVersionValue12 = "1.2"
	// TLSVersionValue13 selects TLS 1.3 as the configured minimum version.
	TLSVersionValue13 = "1.3"

	// URLSchemeTLSFmt renders the BSSCI service center URL.
	URLSchemeTLSFmt = "tls://%s:%d"
	// URL schemes stripped when reading the host of the external BSSCI URL.
	urlSchemePrefixTLS = "tls://"
	urlSchemePrefixTCP = "tcp://"
	// DefaultEdition is the edition used when none is configured.
	DefaultEdition = EditionCommunity

	// EditionLabelCommunity is the human-readable label for Community Edition.
	EditionLabelCommunity = "Community Edition"
	// EditionLabelEnterprise is the human-readable label for Enterprise Cloud Edition.
	EditionLabelEnterprise = "Enterprise Cloud Edition"
)

// IsCommunityEdition returns true when the configured edition is Community Edition.
func IsCommunityEdition(edition string) bool {
	return edition == EditionCommunity
}

// EditionLabel returns the canonical human-readable label for an edition code.
func EditionLabel(edition string) string {
	switch edition {
	case EditionECE:
		return EditionLabelEnterprise
	case EditionCommunity:
		return EditionLabelCommunity
	default:
		return EditionLabelCommunity
	}
}

// =========================================================================
// Federation Defaults
// =========================================================================

const (
	// DefaultFederationEnabled is false; operators must explicitly enable the federation relay.
	DefaultFederationEnabled = false
	// DefaultFederationHeartbeatIntervalSeconds is the interval between CE heartbeats on the relay stream.
	DefaultFederationHeartbeatIntervalSeconds = 60
	// DefaultFederationReconnectMaxBackoffSeconds is the maximum reconnect delay in seconds.
	DefaultFederationReconnectMaxBackoffSeconds = 300
	// DefaultFederationIngressGRPCPort is the port for the ECE federation-ingress gRPC server.
	DefaultFederationIngressGRPCPort = 50055
	// DefaultFederationRevocationPollIntervalSeconds is the interval at which the ingress service
	// polls the DB to detect revoked CE instances.
	DefaultFederationRevocationPollIntervalSeconds = 10
)

// =========================================================================
// Federation TLS Defaults
// =========================================================================

const (
	// DefaultFederationTLSEnabled controls TLS on the CE→ECE relay connection.
	// Disabled by default; operators should enable for production deployments.
	DefaultFederationTLSEnabled = false

	// DefaultFederationTLSInsecureSkipVerify controls certificate verification on the relay connection.
	// Must be false in production.
	DefaultFederationTLSInsecureSkipVerify = false
)

// =========================================================================
// Protocol BSSCI/SCACI Defaults (loader.go:85-110)
// =========================================================================

const (
	// DefaultProtocolBSCIHost is the BSSCI listener bind address.
	DefaultProtocolBSCIHost = "0.0.0.0"

	// DefaultProtocolBSCIPort is the BSSCI listener port.
	DefaultProtocolBSCIPort = 5000

	// DefaultProtocolManagementPort is the loopback port of the internal BSSCI
	// management API.
	DefaultProtocolManagementPort = 8081

	// DefaultProtocolBSCITLSEnabled controls TLS for BSSCI connections.
	DefaultProtocolBSCITLSEnabled = true

	// DefaultProtocolBSCITLSCertFile is the path to the BSSCI TLS certificate.
	DefaultProtocolBSCITLSCertFile = "certificates/server.crt"

	// DefaultProtocolBSCITLSKeyFile is the path to the BSSCI TLS private key.
	DefaultProtocolBSCITLSKeyFile = "certificates/server.key"

	// DefaultProtocolBSCITLSCAFile is the path to the BSSCI CA certificate.
	DefaultProtocolBSCITLSCAFile = "certificates/ca.crt"

	// DefaultProtocolBSCITLSMinVersion is the minimum TLS version for BSSCI.
	// BSSCI mandates TLS with mutual certificates and sets no floor, so 1.2 admits every conforming base station; operators raise it.
	DefaultProtocolBSCITLSMinVersion = "1.2"

	// DefaultProtocolSCACIEnabled controls whether SCACI server is started.
	DefaultProtocolSCACIEnabled = true

	// DefaultProtocolSCACIPort is the SCACI listener port.
	DefaultProtocolSCACIPort = 5001

	// DefaultProtocolSCACITLSMinVersion is the minimum TLS version for SCACI.
	// TLS 1.3 per MIOTY Security Guide v1.1; may override to "1.2" for legacy ACs.
	DefaultProtocolSCACITLSMinVersion = "1.3"

	// DefaultProtocolSCACILogPingOps controls logging of SCACI Ping operations.
	DefaultProtocolSCACILogPingOps = true

	// DefaultProtocolSCACILogStatusOps controls logging of SCACI Status operations.
	DefaultProtocolSCACILogStatusOps = true

	// DefaultProtocolSCACIResumeMaxPendingOperations is how many service center
	// operations a disconnected Application Center session holds for its resume.
	DefaultProtocolSCACIResumeMaxPendingOperations = 10000

	// SCACIDashboardDefaultWindow is the lookback the SCACI statistics and
	// status RPCs use when the request carries no range.
	SCACIDashboardDefaultWindow = 24 * time.Hour

	// DiagnosticsMaxBundleBytes caps the diagnostics archive a server admin can download.
	DiagnosticsMaxBundleBytes = 8 << 20
	// DiagnosticsResponseHeadroomBytes covers the response fields that travel
	// with the archive.
	DiagnosticsResponseHeadroomBytes = 64 << 10
	// GatewayUpstreamMaxResponseBytes is the largest response KC-Gateway
	// accepts from an upstream; it carries a full diagnostics bundle.
	GatewayUpstreamMaxResponseBytes = DiagnosticsMaxBundleBytes + DiagnosticsResponseHeadroomBytes
	// DiagnosticsMaxEvents bounds the newest events the bundle carries.
	DiagnosticsMaxEvents = 500
	// DiagnosticsMaxSessions bounds the base station sessions listed in the bundle.
	DiagnosticsMaxSessions = 200
	// DiagnosticsGenerationTimeout bounds the reads behind one bundle.
	DiagnosticsGenerationTimeout = 30 * time.Second
	// DiagnosticsArchiveFilename is the download name of the bundle.
	DiagnosticsArchiveFilename = "kilocenter-diagnostics.zip"
	// DiagnosticsContentType is the media type of the bundle.
	DiagnosticsContentType = "application/zip"

	// DefaultProtocolSCVendor is the Service Center vendor name.
	DefaultProtocolSCVendor = "Kilo"

	// DefaultProtocolSCModel is the Service Center model identifier.
	DefaultProtocolSCModel = "KiloCenter"

	// DefaultProtocolSCEUI is the canonical Service Center EUI ("KC" ASCII prefix + unique ID)
	// as a plain 16-hex string.
	DefaultProtocolSCEUI = "4B43000000000001"

	// ConfigKeyProtocolSCEUI is the config file key for the Service Center EUI.
	ConfigKeyProtocolSCEUI = "protocol.sc_eui"

	// EnvProtocolSCEUI is the environment variable that overrides protocol.sc_eui.
	EnvProtocolSCEUI = "KILOCENTER_PROTOCOL_SC_EUI"

	// EnvLegacyServiceCenterEUI is the deprecated Service Center EUI environment variable.
	// It accepts base-0 syntax (decimal or 0x-prefixed) and is consulted only when neither
	// EnvProtocolSCEUI nor an explicit protocol.sc_eui file value is present.
	EnvLegacyServiceCenterEUI = "SERVICE_CENTER_EUI"

	// DefaultProtocolAckTimeout is the acknowledgement timeout in milliseconds.
	DefaultProtocolAckTimeout = 5000

	// DefaultProtocolConnectionEstablishmentTimeout bounds a fresh connection before its con arrives, in milliseconds.
	DefaultProtocolConnectionEstablishmentTimeout = 30000

	// DefaultProtocolSocketWriteTimeout bounds every BSSCI and SCACI frame write, in milliseconds.
	DefaultProtocolSocketWriteTimeout = 10000

	// DefaultProtocolStatusRequestInterval is how often the SC polls a base station for status, in seconds.
	DefaultProtocolStatusRequestInterval = 30
	// DefaultProtocolStatusRequestInitialDelay delays the first status poll after connect, in seconds.
	DefaultProtocolStatusRequestInitialDelay = 5
	// DefaultProtocolDLRXQueryTimeout expires an unanswered dlRxStatQry after this many seconds.
	DefaultProtocolDLRXQueryTimeout = 300
	// DefaultProtocolDLRXCleanupInterval is the dlRxStatQry expiry sweep cadence, in seconds.
	DefaultProtocolDLRXCleanupInterval = 60

	// DefaultProtocolDuplicateWindow is the duplicate detection window in seconds.
	DefaultProtocolDuplicateWindow = 300

	// DefaultProtocolDeliveryPollInterval is how often the delivery worker claims due outbox rows.
	DefaultProtocolDeliveryPollInterval = time.Second
	// DefaultProtocolDeliveryBatchSize is the number of outbox rows claimed per poll.
	DefaultProtocolDeliveryBatchSize = 50
	// DefaultProtocolDeliveryRetryBackoff is the base of the exponential retry backoff.
	DefaultProtocolDeliveryRetryBackoff = 2 * time.Second
	// DefaultProtocolDeliveryMaxBackoff caps the wait between retries of a transiently failing delivery.
	DefaultProtocolDeliveryMaxBackoff = 5 * time.Minute
	// DefaultProtocolDeliveryReceptionWindow is how long a new uplink waits for the receptions of
	// the other base stations to merge into it before it is delivered (SCACI §3.8.1 baseStations).
	DefaultProtocolDeliveryReceptionWindow = 500 * time.Millisecond

	// DefaultProtocolDownlinkLifetime is how long a queued downlink waits for a downlink window before it expires.
	DefaultProtocolDownlinkLifetime = 24 * time.Hour
	// DefaultProtocolDownlinkExpirySweepInterval is how often overdue downlinks are expired and
	// reported; one indexed statement over the pending rows, so an Application Center learns of
	// an expiry within seconds.
	DefaultProtocolDownlinkExpirySweepInterval = 5 * time.Second
	// DefaultProtocolDownlinkExpiryBatchSize is the number of downlinks one sweep statement expires.
	DefaultProtocolDownlinkExpiryBatchSize = 100

	// DefaultProtocolRoamingCacheEnabled caches endpoint owners for roaming decisions.
	DefaultProtocolRoamingCacheEnabled = true
	// DefaultProtocolRoamingCacheTTL is how long a cached endpoint owner is trusted.
	DefaultProtocolRoamingCacheTTL = 5 * time.Minute
	// DefaultProtocolRoamingCacheMaxSize bounds the number of cached endpoint owners.
	DefaultProtocolRoamingCacheMaxSize = 10000
	// DefaultProtocolRoamingEnableAuditTrail records roaming attach and detach events.
	DefaultProtocolRoamingEnableAuditTrail = true

	// DefaultProtocolCertificatePollInterval is the base station certificate change poll interval.
	DefaultProtocolCertificatePollInterval = 10 * time.Second
)

// =========================================================================
// Storage Defaults (loader.go:112-124)
// NOTE: DefaultStorageType and DefaultStorageUsername already exist above.
// =========================================================================

const (
	// DefaultStorageHost is the default database host.
	DefaultStorageHost = "localhost"

	// DefaultStoragePort is the default PostgreSQL port.
	DefaultStoragePort = 5432

	// DefaultStorageDatabase is the default database name.
	DefaultStorageDatabase = "kilocenter"

	// DefaultStoragePassword is the default database password (empty for dev).
	DefaultStoragePassword = ""

	// DefaultStorageSSLMode is the default PostgreSQL SSL mode.
	DefaultStorageSSLMode = "disable"

	// DefaultStorageMaxConnections is the maximum connection pool size.
	DefaultStorageMaxConnections = 25

	// DefaultStorageMaxIdleConns is the maximum idle connections in the pool.
	DefaultStorageMaxIdleConns = 5

	// DefaultStorageConnMaxLifetime is the max connection lifetime in seconds.
	DefaultStorageConnMaxLifetime = 300

	// DefaultStorageConnMaxIdleTime is the max connection idle time in seconds.
	DefaultStorageConnMaxIdleTime = 60

	// DefaultStorageEnableMigrations controls automatic migration on startup.
	DefaultStorageEnableMigrations = true
)

// =========================================================================
// MQTT Defaults (loader.go:126-136)
// =========================================================================

const (
	// DefaultMQTTEnabled controls whether MQTT publishing is active.
	DefaultMQTTEnabled = false

	// DefaultMQTTHost is the default MQTT broker host.
	DefaultMQTTHost = "localhost"

	// DefaultMQTTPort is the default MQTT broker port.
	DefaultMQTTPort = 1883

	// DefaultMQTTClientID is the default MQTT client identifier.
	DefaultMQTTClientID = "kilocenter"

	// DefaultMQTTKeepAlive is the MQTT keep-alive interval in seconds.
	DefaultMQTTKeepAlive = 60

	// DefaultMQTTCleanSession controls MQTT session persistence.
	DefaultMQTTCleanSession = true

	// DefaultMQTTMaxReconnectInterval is the max reconnect wait in seconds.
	DefaultMQTTMaxReconnectInterval = 300

	// DefaultMQTTTopicPrefix is the MQTT topic namespace prefix.
	DefaultMQTTTopicPrefix = "mioty"

	// DefaultMQTTEnableCommandSubscriptions controls inbound MQTT command/down subscriber.
	DefaultMQTTEnableCommandSubscriptions = false

	// DefaultMQTTTLSEnabled controls TLS for MQTT connections.
	DefaultMQTTTLSEnabled = false

	// DefaultMQTTTLSInsecureSkipVerify skips TLS certificate verification.
	DefaultMQTTTLSInsecureSkipVerify = false
)

// =========================================================================
// WebGUI Defaults (loader.go:155-159)
// =========================================================================

// =========================================================================
// Monitoring Defaults (loader.go:161-166)
// =========================================================================

const (
	// DefaultMonitoringMetricsEnabled controls Prometheus metrics export.
	DefaultMonitoringMetricsEnabled = true

	// DefaultMonitoringMetricsPort is the metrics server port.
	// Uses 9091 to avoid conflict with gRPC default port (9090).
	DefaultMonitoringMetricsPort = 9091

	// DefaultMonitoringMetricsPath is the metrics endpoint path.
	DefaultMonitoringMetricsPath = "/metrics"

	// DefaultMonitoringTracingEnabled controls distributed tracing.
	DefaultMonitoringTracingEnabled = false

	// DefaultMonitoringTracingSampleRate is the tracing sample ratio.
	DefaultMonitoringTracingSampleRate = 0.1
)

// =========================================================================
// gRPC Defaults (loader.go:168-175)
// =========================================================================

const (
	// DefaultGRPCEnabled controls whether gRPC server is started.
	DefaultGRPCEnabled = true

	// DefaultGRPCPort is the gRPC server port.
	DefaultGRPCPort = 50051

	// DefaultGRPCHost is the gRPC server bind address.
	// Empty string binds to all interfaces (backward-compatible default).
	DefaultGRPCHost = ""

	// DefaultGRPCInternalTrustEnabled controls gateway trust mode.
	// When true, KC-Core trusts identity headers from gateway and disables its own auth/gRPC-web.
	DefaultGRPCInternalTrustEnabled = false

	// DefaultGRPCStreamPollInterval is the fallback read of the streaming RPCs; stored rows wake them at once.
	DefaultGRPCStreamPollInterval = "5s"

	// DefaultGRPCStreamBatchSize is the streaming response batch size.
	DefaultGRPCStreamBatchSize = 100

	// DefaultGRPCStreamOverlap is how far each stream read reaches back behind
	// the newest row it read: the longest a writer may take to commit a row
	// after the database stamped it.
	DefaultGRPCStreamOverlap = "10s"

	// DefaultGRPCCountCacheTTL is how long unary event COUNT(*) results are cached.
	DefaultGRPCCountCacheTTL = "10s"

	// CountCacheComputeTimeout bounds a shared event COUNT(*), which no caller's deadline covers.
	CountCacheComputeTimeout = 30 * time.Second

	// DefaultRBACRoleCacheTTLSeconds is the default TTL for cached RBAC role lookups.
	DefaultRBACRoleCacheTTLSeconds = 30
)

// =========================================================================
// gRPC-web Defaults
// =========================================================================

const (
	// DefaultGRPCEnableReflection controls gRPC reflection service.
	DefaultGRPCEnableReflection = true

	// DefaultGRPCEnableHealth controls gRPC health service.
	DefaultGRPCEnableHealth = true

	// DefaultGatewayGRPCWebEnabled is true: KC-Gateway is the browser ingress,
	// so a gateway configuration that omits grpc.web.enabled still serves it.
	DefaultGatewayGRPCWebEnabled = true

	// DefaultGRPCWebEnabled is false: KC-Gateway terminates gRPC-web; a self-hosted
	// deployment that talks to KC-Core directly enables it explicitly.
	DefaultGRPCWebEnabled = false

	// DefaultGRPCWebAllowCredentials controls CORS credentials for gRPC-web.
	DefaultGRPCWebAllowCredentials = true

	// DefaultGRPCWebAllowAllOrigins is false by default for security.
	// Set to true only in development with explicit config.
	DefaultGRPCWebAllowAllOrigins = false

	// GRPCWebDefaultMaxAgeSeconds is the CORS preflight cache duration.
	GRPCWebDefaultMaxAgeSeconds = 3600

	// HTTPDefaultReadTimeoutString is the HTTP read timeout duration string.
	HTTPDefaultReadTimeoutString = "30s"

	// HTTPDefaultWriteTimeoutString is the HTTP write timeout duration string.
	HTTPDefaultWriteTimeoutString = "30s"

	// HTTPDefaultIdleTimeoutString is the HTTP idle timeout duration string.
	HTTPDefaultIdleTimeoutString = "120s"
)

// gRPC-web slice defaults (slices cannot be constants in Go)
var (
	// GRPCWebDefaultAllowedMethods is the default CORS methods for gRPC-web.
	GRPCWebDefaultAllowedMethods = []string{"POST", "OPTIONS"}

	// DefaultGRPCWebAllowedOrigins is the default allowed origins (empty = use AllowAllOrigins).
	DefaultGRPCWebAllowedOrigins = []string{}

	// DefaultGRPCWebAllowedHeaders is the default allowed headers (empty = use grpc constants).
	DefaultGRPCWebAllowedHeaders = []string{}

	// DefaultGRPCWebExposeHeaders is the default exposed headers (empty = use grpc constants).
	DefaultGRPCWebExposeHeaders = []string{}
)

// =========================================================================
// Auth Additional Defaults (loader.go:177-244)
// NOTE: Existing Auth* constants (AuthDefaultLoginLabel, AuthDefaultOIDCScopes,
// AuthDefaultOAuth2Scopes, AuthDefaultStateTTLString, AuthDefaultNonceTTLString,
// AuthDefaultPKCEMethod, AuthDefaultUserIDClaim, AuthDefaultEmailClaim,
// AuthOIDCLoginLabel, AuthOAuth2LoginLabel) are defined above - DO NOT REDEFINE.
// =========================================================================

const (
	// DefaultAuthEnabled controls JWT authentication.
	DefaultAuthEnabled = false

	// DefaultAuthJWKSEndpoint is the external JWKS URL (empty = local auth).
	DefaultAuthJWKSEndpoint = ""

	// DefaultAuthIssuer is the JWT issuer for validation (empty = no issuer check).
	DefaultAuthIssuer = ""

	// DefaultAuthAudience is the JWT audience for validation (empty = no audience check).
	DefaultAuthAudience = ""

	// DefaultAuthTenantClaim is the JWT claim for tenant ID.
	DefaultAuthTenantClaim = "urn:zitadel:iam:org:project:id"

	// DefaultAuthAlgorithm is the JWT signing algorithm.
	DefaultAuthAlgorithm = "RS256"

	// DefaultAuthLocalLoginEnabled controls local email/password auth.
	DefaultAuthLocalLoginEnabled = false

	// DefaultAuthHMACSecret is the HMAC key (empty = must be configured).
	DefaultAuthHMACSecret = ""

	// DefaultAuthAccessTokenTTL is the access token lifetime string.
	DefaultAuthAccessTokenTTL = "15m"

	// DefaultAuthRefreshTokenTTL is the refresh token lifetime string.
	DefaultAuthRefreshTokenTTL = "24h"

	// DefaultAuthRefreshTokenEnabled controls refresh token rotation.
	DefaultAuthRefreshTokenEnabled = false

	// DefaultAuthLoginURL is the custom login page URL (empty = built-in).
	DefaultAuthLoginURL = ""

	// DefaultAuthLoginRedirect controls redirect to login on 401.
	DefaultAuthLoginRedirect = true

	// DefaultAuthLogoutURL is the post-logout redirect URL.
	DefaultAuthLogoutURL = ""

	// DefaultAuthOIDCEnabled controls OIDC provider.
	DefaultAuthOIDCEnabled = false

	// DefaultAuthOIDCProviderURL is the OIDC provider base URL.
	DefaultAuthOIDCProviderURL = ""

	// DefaultAuthOIDCClientID is the OIDC client ID.
	DefaultAuthOIDCClientID = ""

	// DefaultAuthOIDCClientSecret is the OIDC client secret.
	DefaultAuthOIDCClientSecret = ""

	// DefaultAuthOIDCRedirectURL is the OIDC redirect URL.
	DefaultAuthOIDCRedirectURL = ""

	// DefaultAuthOIDCLoginRedirect controls auto-redirect to OIDC provider.
	DefaultAuthOIDCLoginRedirect = false

	// DefaultAuthOIDCLogoutURL is the OIDC post-logout URL.
	DefaultAuthOIDCLogoutURL = ""

	// DefaultAuthOIDCRegistrationEnabled controls OIDC user creation.
	DefaultAuthOIDCRegistrationEnabled = false

	// DefaultAuthOIDCRegistrationCallbackURL is the OIDC registration callback.
	DefaultAuthOIDCRegistrationCallbackURL = ""

	// DefaultAuthOIDCAssumeEmailVerified skips email verification check.
	DefaultAuthOIDCAssumeEmailVerified = false

	// DefaultAuthOIDCExternalOrgClaim is the external org ID claim path.
	DefaultAuthOIDCExternalOrgClaim = ""

	// DefaultAuthOAuth2Enabled controls OAuth2 provider.
	DefaultAuthOAuth2Enabled = false

	// DefaultAuthOAuth2AuthorizeURL is the OAuth2 authorize endpoint.
	DefaultAuthOAuth2AuthorizeURL = ""

	// DefaultAuthOAuth2TokenURL is the OAuth2 token endpoint.
	DefaultAuthOAuth2TokenURL = ""

	// DefaultAuthOAuth2UserinfoURL is the OAuth2 userinfo endpoint.
	DefaultAuthOAuth2UserinfoURL = ""

	// DefaultAuthOAuth2ClientID is the OAuth2 client ID.
	DefaultAuthOAuth2ClientID = ""

	// DefaultAuthOAuth2ClientSecret is the OAuth2 client secret.
	DefaultAuthOAuth2ClientSecret = ""

	// DefaultAuthOAuth2PublicClient controls PKCE-only mode.
	DefaultAuthOAuth2PublicClient = false

	// DefaultAuthOAuth2RedirectURL is the OAuth2 redirect URL.
	DefaultAuthOAuth2RedirectURL = ""

	// DefaultAuthOAuth2LoginRedirect controls auto-redirect to OAuth2 provider.
	DefaultAuthOAuth2LoginRedirect = false

	// DefaultAuthOAuth2LogoutURL is the OAuth2 post-logout URL.
	DefaultAuthOAuth2LogoutURL = ""

	// DefaultAuthOAuth2RegistrationEnabled controls OAuth2 user creation.
	DefaultAuthOAuth2RegistrationEnabled = false

	// DefaultAuthOAuth2AssumeEmailVerified skips email verification check.
	DefaultAuthOAuth2AssumeEmailVerified = false
)

// =========================================================================
// Redis Defaults (loader.go:246-251)
// =========================================================================

const (
	// DefaultRedisHost is the default Redis host.
	DefaultRedisHost = "localhost"

	// DefaultRedisPort is the default Redis port.
	DefaultRedisPort = 6379

	// DefaultRedisPassword is the default Redis password (empty for no auth).
	DefaultRedisPassword = ""

	// DefaultRedisDatabase is the default Redis database number.
	DefaultRedisDatabase = 0

	// DefaultRedisPoolSize is the default Redis connection pool size.
	DefaultRedisPoolSize = 10
)

// =========================================================================
// Alerts Defaults (loader.go:253-255)
// =========================================================================

const (
	// DefaultAlertsRecentAlertsLimit is the max recent alerts in summary.
	DefaultAlertsRecentAlertsLimit = 5
)

// =========================================================================
// Analytics Defaults (loader.go:257-260)
// =========================================================================

// =========================================================================
// Certificate Defaults
// =========================================================================

const (
	// DefaultCertificatesCertGenPath is the path to the certgen binary.
	// Relative to KC-Core working directory.
	DefaultCertificatesCertGenPath = "./certgen"

	// DefaultCertificatesCertsDir is the directory for permanent certificates.
	DefaultCertificatesCertsDir = "certificates"

	// DefaultCertificatesTempDir is the directory for temporary certificate downloads.
	DefaultCertificatesTempDir = "/tmp/kilocenter-certs"

	// DefaultCertificatesServerValidityDays is the default validity period for generated server certificates.
	DefaultCertificatesServerValidityDays = 365

	// DefaultCertificatesCleanupIntervalMin is how often, in minutes, expired certificate bundles are removed.
	DefaultCertificatesCleanupIntervalMin = 15

	// DefaultCertificatesHostname is the fallback hostname for server certificate generation.
	DefaultCertificatesHostname = "localhost"
)

// =========================================================================
// Status Monitoring Defaults
// =========================================================================

const (
	// DefaultStatusTimeoutString is the default timeout for health check requests.
	DefaultStatusTimeoutString = "5s"
)

// =========================================================================
// Registry Provider Defaults (Blueprint submission to external registries)
// =========================================================================

const (
	// DefaultRegistryProviderEnabled disables registry submission by default.
	DefaultRegistryProviderEnabled = false

	// DefaultRegistryProviderAPIURL is empty; must be configured when enabled.
	// Set to the registry provider API base URL.
	DefaultRegistryProviderAPIURL = ""

	// DefaultRegistryProviderBaseBranch is the target branch for submission requests.
	DefaultRegistryProviderBaseBranch = "main"

	// DefaultRegistryProviderBranchPrefix prefixes new branch names.
	DefaultRegistryProviderBranchPrefix = "blueprint/"

	// DefaultRegistryProviderBlueprintPath is the path prefix for blueprint files.
	// Empty because manufacturer directories sit at the registry root.
	DefaultRegistryProviderBlueprintPath = ""

	// DefaultRegistryProviderFileExtension is the blueprint file extension.
	DefaultRegistryProviderFileExtension = ".json"

	// DefaultRegistryProviderHTTPTimeout is the HTTP client timeout in seconds.
	DefaultRegistryProviderHTTPTimeout = 30

	// DefaultRegistryProviderToken is empty; set via KILOCENTER_REGISTRY_PROVIDER_TOKEN env var.
	DefaultRegistryProviderToken = ""

	// DefaultRegistryProviderAuthMode uses static token for backward compatibility.
	DefaultRegistryProviderAuthMode = "token"
)
