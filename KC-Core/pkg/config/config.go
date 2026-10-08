// Package config provides configuration types for KiloCenter modules
package config

import (
	"fmt"
	"time"
)

// GetMQTTBrokerURL returns the MQTT broker URL.
func GetMQTTBrokerURL(cfg MQTTConfig) string {
	protocol := MQTTSchemeTCP
	if cfg.TLS.Enabled {
		protocol = MQTTSchemeSSL
	}
	return fmt.Sprintf("%s://%s:%d", protocol, cfg.Host, cfg.Port)
}

// StorageConfig contains database configuration settings
type StorageConfig struct {
	Type                 string `mapstructure:"type"` // postgres, mysql, etc.
	Host                 string `mapstructure:"host"`
	Port                 int    `mapstructure:"port"`
	Database             string `mapstructure:"database"`
	Username             string `mapstructure:"username"`
	Password             string `mapstructure:"password"`
	SSLMode              string `mapstructure:"ssl_mode"`
	MaxConnections       int    `mapstructure:"max_connections"`
	MaxIdleConns         int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime      int    `mapstructure:"conn_max_lifetime"`  // seconds
	ConnMaxIdleTime      int    `mapstructure:"conn_max_idle_time"` // seconds
	EnableMigrations     bool   `mapstructure:"enable_migrations"`
	MessageRetentionDays int    `mapstructure:"message_retention_days"`
	ArchivalEnabled      *bool  `mapstructure:"archival_enabled"`
}

// GeneralConfig contains general server settings
type GeneralConfig struct {
	ServerName      string `mapstructure:"server_name"`
	Environment     string `mapstructure:"environment"` // development, staging, production
	LogLevel        string `mapstructure:"log_level"`
	LogFormat       string `mapstructure:"log_format"` // json, text
	HealthCheckPort int    `mapstructure:"health_check_port"`
	TenantID        int64  `mapstructure:"tenant_id"`        // Default tenant ID for single-tenant deployments
	SoftwareVersion string `mapstructure:"software_version"` // SC software version for BSSCI conRsp (§5.3.2)

	// Edition controls which feature set is enabled: "ece" (Enterprise Cloud Edition) or "ce" (Community Edition).
	// CE mode activates the federation relay client; ECE mode activates the federation ingress.
	Edition string `mapstructure:"edition"`

	// Organization resolution configuration
	OrgEnforcementEnabled bool `mapstructure:"org_enforcement_enabled"` // Strict org-based isolation (optional)
	OrgCacheTTLMinutes    int  `mapstructure:"org_cache_ttl_minutes"`   // Cache entry TTL (default: 5 minutes)
	OrgCacheMaxEntries    int  `mapstructure:"org_cache_max_entries"`   // Max cache size (default: 10,000)

	ActivityWindowHours int `mapstructure:"activity_window_hours"` // Endpoint activity threshold in hours (default: 24)
}

// TLSConfig contains TLS configuration settings
type TLSConfig struct {
	Enabled            bool     `mapstructure:"enabled"`
	CertFile           string   `mapstructure:"cert_file"`
	KeyFile            string   `mapstructure:"key_file"`
	CAFile             string   `mapstructure:"ca_file"`
	ServerName         string   `mapstructure:"server_name"`
	InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify"`
	MinVersion         string   `mapstructure:"min_version"`
	AllowedCiphers     []string `mapstructure:"allowed_ciphers"`
}

// MQTTConfig contains MQTT broker settings
type MQTTConfig struct {
	Enabled                    bool      `mapstructure:"enabled"`
	Host                       string    `mapstructure:"host"`
	Port                       int       `mapstructure:"port"`
	ClientID                   string    `mapstructure:"client_id"`
	Username                   string    `mapstructure:"username"`
	Password                   string    `mapstructure:"password"`
	TLS                        TLSConfig `mapstructure:"tls"`
	CleanSession               bool      `mapstructure:"clean_session"`
	KeepAlive                  int       `mapstructure:"keep_alive"`             // seconds
	MaxReconnectInterval       int       `mapstructure:"max_reconnect_interval"` // seconds
	TopicPrefix                string    `mapstructure:"topic_prefix"`
	EnableCommandSubscriptions bool      `mapstructure:"enable_command_subscriptions"`
}

// MonitoringConfig contains monitoring and metrics settings
type MonitoringConfig struct {
	MetricsEnabled    bool    `mapstructure:"metrics_enabled"`
	MetricsPort       int     `mapstructure:"metrics_port"`
	MetricsPath       string  `mapstructure:"metrics_path"`
	TracingEnabled    bool    `mapstructure:"tracing_enabled"`
	TracingEndpoint   string  `mapstructure:"tracing_endpoint"`
	TracingSampleRate float64 `mapstructure:"tracing_sample_rate"`
}

// AlertConfig contains alert service configuration
type AlertConfig struct {
	RecentAlertsLimit int `mapstructure:"recent_alerts_limit"` // Max recent alerts in summary (default: 5)
}

// CertificateConfig contains certificate generation settings.
type CertificateConfig struct {
	CertGenPath        string `mapstructure:"certgen_path"`         // Path to certgen binary
	CertsDir           string `mapstructure:"certs_dir"`            // Directory for permanent certificates
	TempDir            string `mapstructure:"temp_dir"`             // Directory for temporary certificate downloads
	ServerValidityDays int    `mapstructure:"server_validity_days"` // Validity period for generated server certificates (days, default: 365)
	// ServerNames are further DNS names and IP addresses the server certificate carries.
	ServerNames        []string `mapstructure:"server_names"`
	CleanupIntervalMin int      `mapstructure:"cleanup_interval_min"` // How often expired certificate bundles are removed (minutes, default: 15)
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Enabled                 bool             `mapstructure:"enabled"`
	Port                    int              `mapstructure:"port"`
	Host                    string           `mapstructure:"host"`                   // Bind address (empty = all interfaces, "127.0.0.1" = loopback only)
	InternalTrustEnabled    bool             `mapstructure:"internal_trust_enabled"` // Trust gateway identity headers (disable auth + gRPC-web in KC-Core)
	EnableTLS               bool             `mapstructure:"enable_tls"`
	TLSCert                 string           `mapstructure:"tls_cert"`
	TLSKey                  string           `mapstructure:"tls_key"`
	StreamPollInterval      time.Duration    `mapstructure:"stream_poll_interval"`        // Fallback read of the streaming RPCs
	StreamBatchSize         int              `mapstructure:"stream_batch_size"`           // Batch size for streaming message responses
	StreamOverlap           time.Duration    `mapstructure:"stream_overlap"`              // How far each stream read reaches back in storage order
	CountCacheTTL           time.Duration    `mapstructure:"count_cache_ttl"`             // TTL for cached event COUNT(*) results
	Web                     GRPCWebConfig    `mapstructure:"web"`                         // gRPC-web config
	HTTP                    HTTPServerConfig `mapstructure:"http"`                        // HTTP server timeouts
	EnableReflection        bool             `mapstructure:"enable_reflection"`           // Enable gRPC reflection service (default: true)
	EnableHealth            bool             `mapstructure:"enable_health"`               // Enable gRPC health service (default: true)
	RBACRoleCacheTTLSeconds int              `mapstructure:"rbac_role_cache_ttl_seconds"` // TTL for cached RBAC role lookups (seconds, default: 30)
}

// GRPCWebConfig holds gRPC-web and CORS configuration
type GRPCWebConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`           // Preflight cache seconds
	AllowAllOrigins  bool     `mapstructure:"allow_all_origins"` // Dev only - MUST be false when AllowCredentials is true in production
	AllowedMethods   []string `mapstructure:"allowed_methods"`   // CORS allowed methods
}

// HTTPServerConfig holds HTTP server timeout configuration
type HTTPServerConfig struct {
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// RedisConfig contains Redis settings
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	PoolSize int    `mapstructure:"pool_size"`
}

// AuthConfig contains authentication and authorization settings
type AuthConfig struct {
	Enabled      bool   `mapstructure:"enabled"`       // Enable JWT authentication
	JWKSEndpoint string `mapstructure:"jwks_endpoint"` // External JWKS URL for key discovery (mutually exclusive with LocalLoginEnabled)
	Issuer       string `mapstructure:"issuer"`        // Expected token issuer (reused for local login if JWKSEndpoint empty)
	Audience     string `mapstructure:"audience"`      // Expected audience claim (reused for local login if JWKSEndpoint empty)
	TenantClaim  string `mapstructure:"tenant_claim"`  // JWT claim containing tenant_id
	Algorithm    string `mapstructure:"algorithm"`     // JWT signing algorithm (default: RS256 for JWKS, HS256 for local)

	// Local authentication settings
	RegistrationEnabled bool          `mapstructure:"registration_enabled"`  // Enable self-service account registration (requires LocalLoginEnabled)
	LocalLoginEnabled   bool          `mapstructure:"local_login_enabled"`   // Enable local email/password login (mutually exclusive with JWKSEndpoint)
	HMACSecret          string        `mapstructure:"hmac_secret"`           // HMAC signing key for local JWT tokens (min 32 bytes)
	AccessTokenTTL      time.Duration `mapstructure:"access_token_ttl"`      // Access token lifetime (default: 15m)
	RefreshTokenTTL     time.Duration `mapstructure:"refresh_token_ttl"`     // Refresh token lifetime (default: 24h)
	RefreshTokenEnabled bool          `mapstructure:"refresh_token_enabled"` // Enable refresh token rotation (requires LocalLoginEnabled)
	LoginURL            string        `mapstructure:"login_url"`             // Custom login page URL (for UI redirect)
	LoginLabel          string        `mapstructure:"login_label"`           // Login button label (default: "Sign In")
	LoginRedirect       bool          `mapstructure:"login_redirect"`        // Redirect unauthenticated requests to LoginURL
	LogoutURL           string        `mapstructure:"logout_url"`            // Post-logout redirect URL

	// External authentication provider settings
	OIDC   OIDCProviderConfig   `mapstructure:"oidc"`   // OIDC provider configuration
	OAuth2 OAuth2ProviderConfig `mapstructure:"oauth2"` // OAuth2 PKCE provider configuration
}

// OIDCProviderConfig contains OIDC (OpenID Connect) provider settings.
type OIDCProviderConfig struct {
	Enabled                 bool          `mapstructure:"enabled"`                   // Enable OIDC authentication
	ProviderURL             string        `mapstructure:"provider_url"`              // OIDC provider base URL (e.g., https://auth.example.com)
	ClientID                string        `mapstructure:"client_id"`                 // OAuth2 client ID
	ClientSecret            string        `mapstructure:"client_secret"`             // OAuth2 client secret
	RedirectURL             string        `mapstructure:"redirect_url"`              // Registered redirect URL (must match provider config)
	Scopes                  []string      `mapstructure:"scopes"`                    // Requested scopes (default: openid, email, profile)
	LoginLabel              string        `mapstructure:"login_label"`               // Login button label (default: "Sign in with SSO")
	LoginRedirect           bool          `mapstructure:"login_redirect"`            // Auto-redirect to provider on unauthenticated access
	LogoutURL               string        `mapstructure:"logout_url"`                // Post-logout redirect URL
	RegistrationEnabled     bool          `mapstructure:"registration_enabled"`      // Allow new user creation from OIDC claims
	RegistrationCallbackURL string        `mapstructure:"registration_callback_url"` // POST callback URL for new user provisioning
	AssumeEmailVerified     bool          `mapstructure:"assume_email_verified"`     // Skip email_verified claim check
	StateTTL                time.Duration `mapstructure:"state_ttl"`                 // State parameter TTL in Redis (default: 5m)
	NonceTTL                time.Duration `mapstructure:"nonce_ttl"`                 // Nonce TTL in Redis (default: 5m)
	ExternalOrgClaim        string        `mapstructure:"external_org_claim"`        // JWT claim path for external org ID mapping
}

// OAuth2ProviderConfig contains OAuth2 (with PKCE) provider settings.
type OAuth2ProviderConfig struct {
	Enabled                 bool          `mapstructure:"enabled"`                   // Enable OAuth2 authentication
	AuthorizeURL            string        `mapstructure:"authorize_url"`             // Authorization endpoint URL
	TokenURL                string        `mapstructure:"token_url"`                 // Token endpoint URL
	UserInfoURL             string        `mapstructure:"userinfo_url"`              // UserInfo endpoint URL
	ClientID                string        `mapstructure:"client_id"`                 // OAuth2 client ID
	ClientSecret            string        `mapstructure:"client_secret"`             // OAuth2 client secret (optional if PublicClient=true)
	PublicClient            bool          `mapstructure:"public_client"`             // PKCE-only public client (no client_secret required)
	RedirectURL             string        `mapstructure:"redirect_url"`              // Registered redirect URL (must match provider config)
	Scopes                  []string      `mapstructure:"scopes"`                    // Requested scopes (default: email, profile)
	LoginLabel              string        `mapstructure:"login_label"`               // Login button label (default: "Sign in with Provider")
	LoginRedirect           bool          `mapstructure:"login_redirect"`            // Auto-redirect to provider on unauthenticated access
	LogoutURL               string        `mapstructure:"logout_url"`                // Post-logout redirect URL
	RegistrationEnabled     bool          `mapstructure:"registration_enabled"`      // Allow new user creation from userinfo
	RegistrationCallbackURL string        `mapstructure:"registration_callback_url"` // POST callback URL for new user provisioning
	AssumeEmailVerified     bool          `mapstructure:"assume_email_verified"`     // Skip email verification requirement
	StateTTL                time.Duration `mapstructure:"state_ttl"`                 // State/verifier TTL in Redis (default: 5m)
	PKCEMethod              string        `mapstructure:"pkce_method"`               // PKCE code challenge method (default: S256)
	UserIDClaim             string        `mapstructure:"user_id_claim"`             // Claim containing user ID (default: sub)
	EmailClaim              string        `mapstructure:"email_claim"`               // Claim containing email (default: email)
}

// RegistryProviderConfig contains settings for blueprint registry submission.
type RegistryProviderConfig struct {
	Enabled       bool   `mapstructure:"enabled"`        // Enable registry submission
	AuthMode      string `mapstructure:"auth_mode"`      // "token" (static PAT) or "github-app" (GitHub App installation tokens)
	APIURL        string `mapstructure:"api_url"`        // API base URL (required when enabled)
	Token         string `mapstructure:"token"`          // Personal access token for API auth (auth_mode=token)
	Owner         string `mapstructure:"owner"`          // Repository owner (org or user)
	Repo          string `mapstructure:"repo"`           // Repository name
	BaseBranch    string `mapstructure:"base_branch"`    // Target branch for submission requests
	BranchPrefix  string `mapstructure:"branch_prefix"`  // Branch name prefix
	BlueprintPath string `mapstructure:"blueprint_path"` // Directory for blueprint files
	FileExtension string `mapstructure:"file_extension"` // Blueprint file extension
	HTTPTimeout   int    `mapstructure:"http_timeout"`   // HTTP client timeout in seconds

	// GitHub App authentication (auth_mode=github-app)
	GitHubAppID             int64  `mapstructure:"github_app_id"`              // GitHub App ID
	GitHubAppInstallationID int64  `mapstructure:"github_app_installation_id"` // GitHub App Installation ID
	GitHubAppPrivateKey     string `mapstructure:"github_app_private_key"`     // PEM-encoded private key
}

// StatusConfig holds the timeout shared by the service-status probes.
type StatusConfig struct {
	Timeout time.Duration `mapstructure:"timeout"`
}

// Config represents the complete configuration structure
type Config struct {
	General          GeneralConfig          `mapstructure:"general"`
	Protocol         ProtocolConfig         `mapstructure:"protocol"`
	Storage          StorageConfig          `mapstructure:"storage"`
	MQTT             MQTTConfig             `mapstructure:"mqtt"`
	Monitoring       MonitoringConfig       `mapstructure:"monitoring"`
	GRPC             GRPCConfig             `mapstructure:"grpc"`
	Auth             AuthConfig             `mapstructure:"auth"`
	Redis            RedisConfig            `mapstructure:"redis"`
	Alerts           AlertConfig            `mapstructure:"alerts"`
	Certificates     CertificateConfig      `mapstructure:"certificates"`
	RegistryProvider RegistryProviderConfig `mapstructure:"registry_provider"` // Blueprint registry submission
	Status           StatusConfig           `mapstructure:"status"`            // Service health check configuration
	Gateway          GatewayConfig          `mapstructure:"gateway"`           // Gateway-specific settings (upstream, health port)
	InternalAuth     InternalAuthConfig     `mapstructure:"internal_auth"`     // Peer-to-peer gRPC auth (shared secret)
	Identity         IdentityConfig         `mapstructure:"identity"`          // KC-Identity service connection
}

// InternalAuthConfig holds shared-secret peer authentication for internal gRPC services.
type InternalAuthConfig struct {
	PeerSecret string `mapstructure:"peer_secret"` // Shared secret for peer-to-peer gRPC auth; required unless every internal hop is on a loopback address
}

// IdentityConfig holds KC-Identity service connection settings.
type IdentityConfig struct {
	Address string `mapstructure:"address"` // KC-Identity gRPC address (e.g., "localhost:50052")
}

// GatewayConfig holds KC-Gateway-specific settings.
type GatewayConfig struct {
	UpstreamAddress string                  `mapstructure:"upstream_address"` // KC-Core internal gRPC address
	IdentityAddress string                  `mapstructure:"identity_address"` // KC-Identity gRPC address
	HealthPort      int                     `mapstructure:"health_port"`      // Gateway health endpoint port
	Resilience      GatewayResilienceConfig `mapstructure:"resilience"`       // Upstream resilience settings
	RateLimit       GatewayRateLimitConfig  `mapstructure:"rate_limit"`       // Registration rate limiting
}

// GatewayRateLimitConfig holds per-IP rate limiting for sensitive endpoints.
type GatewayRateLimitConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	RequestsPerMin  int           `mapstructure:"requests_per_min"`
	Burst           int           `mapstructure:"burst"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// GatewayResilienceConfig holds upstream resilience settings for the gateway.
type GatewayResilienceConfig struct {
	DialTimeout        time.Duration `mapstructure:"dial_timeout"`         // Upstream dial timeout (default: 5s)
	RPCTimeout         time.Duration `mapstructure:"rpc_timeout"`          // Per-RPC deadline (default: 30s)
	MaxRetries         uint          `mapstructure:"max_retries"`          // Max retry attempts for idempotent RPCs (default: 3)
	RetryBackoff       time.Duration `mapstructure:"retry_backoff"`        // Initial retry backoff (default: 100ms)
	RetryMaxBackoff    time.Duration `mapstructure:"retry_max_backoff"`    // Maximum retry backoff (default: 1s)
	CBMaxRequests      uint32        `mapstructure:"cb_max_requests"`      // Half-open state max requests (default: 3)
	CBInterval         time.Duration `mapstructure:"cb_interval"`          // Breaker closed-state interval (default: 60s)
	CBTimeout          time.Duration `mapstructure:"cb_timeout"`           // Breaker open-to-half-open timeout (default: 30s)
	CBFailureThreshold uint32        `mapstructure:"cb_failure_threshold"` // Consecutive failures to trip breaker (default: 5)
}
