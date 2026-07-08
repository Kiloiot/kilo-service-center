// Package config provides configuration types and loading for KiloCenter modules.
// This exported loader can be used by other modules.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load loads configuration from file and environment variables.
// Exported for use by other modules.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config name and paths
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/kilocenter")
	}

	// Set environment variable prefix
	v.SetEnvPrefix("KILOCENTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicit binding for nested keys with underscores in parent names.
	// Viper's AutomaticEnv cannot distinguish "registry_provider.token" from
	// "registry.provider.token" when mapping to KILOCENTER_REGISTRY_PROVIDER_TOKEN.
	_ = v.BindEnv("registry_provider.token", "KILOCENTER_REGISTRY_PROVIDER_TOKEN")
	_ = v.BindEnv("registry_provider.owner", "KILOCENTER_REGISTRY_PROVIDER_OWNER")
	_ = v.BindEnv("registry_provider.repo", "KILOCENTER_REGISTRY_PROVIDER_REPO")
	_ = v.BindEnv("registry_provider.github_app_id", "KILOCENTER_REGISTRY_PROVIDER_GITHUB_APP_ID")
	_ = v.BindEnv("registry_provider.github_app_installation_id", "KILOCENTER_REGISTRY_PROVIDER_GITHUB_APP_INSTALLATION_ID")
	_ = v.BindEnv("registry_provider.github_app_private_key", "KILOCENTER_REGISTRY_PROVIDER_GITHUB_APP_PRIVATE_KEY")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf(ErrConfigReadFmt, err)
		}
		// Config file not found; use defaults and environment
	}

	// Set defaults
	setDefaults(v)

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf(ErrConfigUnmarshalFmt, err)
	}

	// Apply issuer/audience defaults ONLY for local auth (not JWKS)
	// When using external JWKS, issuer/audience come from the IdP config
	if cfg.Auth.Enabled && cfg.Auth.LocalLoginEnabled && cfg.Auth.JWKSEndpoint == "" {
		if cfg.Auth.Issuer == "" {
			cfg.Auth.Issuer = AuthDefaultIssuer
		}
		if cfg.Auth.Audience == "" {
			cfg.Auth.Audience = AuthDefaultAudience
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults sets ALL default configuration values.
// Contains all defaults from internal/config/config.go plus bootstrap defaults.
func setDefaults(v *viper.Viper) {
	// General defaults
	v.SetDefault("general.server_name", DefaultGeneralServerName)
	v.SetDefault("general.environment", DefaultGeneralEnvironment)
	v.SetDefault("general.log_level", DefaultGeneralLogLevel)
	v.SetDefault("general.log_format", DefaultGeneralLogFormat)
	v.SetDefault("general.health_check_port", DefaultGeneralHealthCheckPort)
	v.SetDefault("general.tenant_id", DefaultGeneralTenantID)
	v.SetDefault("general.edition", DefaultEdition)

	// Organization resolution defaults
	v.SetDefault("general.org_enforcement_enabled", DefaultGeneralOrgEnforcementEnabled)
	v.SetDefault("general.org_cache_ttl_minutes", DefaultGeneralOrgCacheTTLMinutes)
	v.SetDefault("general.org_cache_max_entries", DefaultGeneralOrgCacheMaxEntries)
	v.SetDefault("general.activity_window_hours", DefaultEndpointActivityWindowHours)

	// Protocol defaults - BSSCI
	v.SetDefault("protocol.bsci_host", DefaultProtocolBSCIHost)
	v.SetDefault("protocol.bsci_port", DefaultProtocolBSCIPort)
	v.SetDefault("protocol.bsci_tls.enabled", DefaultProtocolBSCITLSEnabled)
	v.SetDefault("protocol.bsci_tls.cert_file", DefaultProtocolBSCITLSCertFile)
	v.SetDefault("protocol.bsci_tls.key_file", DefaultProtocolBSCITLSKeyFile)
	v.SetDefault("protocol.bsci_tls.ca_file", DefaultProtocolBSCITLSCAFile)
	v.SetDefault("protocol.bsci_tls.min_version", DefaultProtocolBSCITLSMinVersion)
	v.SetDefault("protocol.scaci_enabled", DefaultProtocolSCACIEnabled)
	v.SetDefault("protocol.scaci_port", DefaultProtocolSCACIPort)
	v.SetDefault("protocol.scaci_tls.min_version", DefaultProtocolSCACITLSMinVersion)
	v.SetDefault("protocol.scaci_log_ping_operations", DefaultProtocolSCACILogPingOps)
	v.SetDefault("protocol.scaci_log_status_operations", DefaultProtocolSCACILogStatusOps)
	// Service Center Identity
	v.SetDefault("protocol.sc_vendor", DefaultProtocolSCVendor)
	v.SetDefault("protocol.sc_model", DefaultProtocolSCModel)
	v.SetDefault("protocol.max_retransmissions", DefaultProtocolMaxRetransmissions)
	v.SetDefault("protocol.ack_timeout", DefaultProtocolAckTimeout)
	v.SetDefault("protocol.duplicate_window", DefaultProtocolDuplicateWindow)

	// Federation relay defaults
	v.SetDefault("protocol.federation.enabled", DefaultFederationEnabled)
	v.SetDefault("protocol.federation.heartbeat_interval", fmt.Sprintf("%ds", DefaultFederationHeartbeatIntervalSeconds))
	v.SetDefault("protocol.federation.outbox_max_size", DefaultFederationOutboxMaxSize)
	v.SetDefault("protocol.federation.reconnect_max_backoff", fmt.Sprintf("%ds", DefaultFederationReconnectMaxBackoffSeconds))
	v.SetDefault("protocol.federation.ingress_grpc_port", DefaultFederationIngressGRPCPort)
	v.SetDefault("protocol.federation.revocation_poll_interval", fmt.Sprintf("%ds", DefaultFederationRevocationPollIntervalSeconds))
	v.SetDefault("protocol.federation.tls.enabled", DefaultFederationTLSEnabled)
	v.SetDefault("protocol.federation.tls.insecure_skip_verify", DefaultFederationTLSInsecureSkipVerify)

	// Propagation reconciliation defaults (string format for time.Duration per Viper convention)
	v.SetDefault("protocol.propagation.batch_size", DefaultProtocolPropBatchSize)
	v.SetDefault("protocol.propagation.inter_batch_delay", DefaultProtocolPropInterBatchDelay)
	v.SetDefault("protocol.propagation.max_retries", DefaultProtocolPropMaxRetries)
	v.SetDefault("protocol.propagation.retry_backoff", DefaultProtocolPropRetryBackoff)
	v.SetDefault("protocol.propagation.cool_down", DefaultProtocolPropCoolDown)

	// Storage defaults
	v.SetDefault("storage.type", DefaultStorageType)
	v.SetDefault("storage.host", DefaultStorageHost)
	v.SetDefault("storage.port", DefaultStoragePort)
	v.SetDefault("storage.database", DefaultStorageDatabase)
	v.SetDefault("storage.username", DefaultStorageUsername)
	v.SetDefault("storage.password", DefaultStoragePassword)
	v.SetDefault("storage.ssl_mode", DefaultStorageSSLMode)
	v.SetDefault("storage.max_connections", DefaultStorageMaxConnections)
	v.SetDefault("storage.max_idle_conns", DefaultStorageMaxIdleConns)
	v.SetDefault("storage.conn_max_lifetime", DefaultStorageConnMaxLifetime)
	v.SetDefault("storage.conn_max_idle_time", DefaultStorageConnMaxIdleTime)
	v.SetDefault("storage.enable_migrations", DefaultStorageEnableMigrations)

	// MQTT defaults
	v.SetDefault("mqtt.enabled", DefaultMQTTEnabled)
	v.SetDefault("mqtt.host", DefaultMQTTHost)
	v.SetDefault("mqtt.port", DefaultMQTTPort)
	v.SetDefault("mqtt.client_id", DefaultMQTTClientID)
	v.SetDefault("mqtt.keep_alive", DefaultMQTTKeepAlive)
	v.SetDefault("mqtt.clean_session", DefaultMQTTCleanSession)
	v.SetDefault("mqtt.max_reconnect_interval", DefaultMQTTMaxReconnectInterval)
	v.SetDefault("mqtt.topic_prefix", DefaultMQTTTopicPrefix)
	v.SetDefault("mqtt.enable_command_subscriptions", DefaultMQTTEnableCommandSubscriptions)
	v.SetDefault("mqtt.tls.enabled", DefaultMQTTTLSEnabled)
	v.SetDefault("mqtt.tls.insecure_skip_verify", DefaultMQTTTLSInsecureSkipVerify)

	// WebGUI defaults
	v.SetDefault("web_gui.enabled", DefaultWebGUIEnabled)
	v.SetDefault("web_gui.port", DefaultWebGUIPort)
	v.SetDefault("web_gui.host", DefaultWebGUIHost)
	v.SetDefault("web_gui.static_path", DefaultWebGUIStaticPath)

	// Monitoring defaults
	v.SetDefault("monitoring.metrics_enabled", DefaultMonitoringMetricsEnabled)
	v.SetDefault("monitoring.metrics_port", DefaultMonitoringMetricsPort)
	v.SetDefault("monitoring.metrics_path", DefaultMonitoringMetricsPath)
	v.SetDefault("monitoring.tracing_enabled", DefaultMonitoringTracingEnabled)
	v.SetDefault("monitoring.tracing_sample_rate", DefaultMonitoringTracingSampleRate)

	// gRPC defaults
	v.SetDefault("grpc.enabled", DefaultGRPCEnabled)
	v.SetDefault("grpc.port", DefaultGRPCPort)
	v.SetDefault("grpc.host", DefaultGRPCHost)
	v.SetDefault("grpc.internal_trust_enabled", DefaultGRPCInternalTrustEnabled)
	v.SetDefault("grpc.tls_enabled", DefaultGRPCTLSEnabled)
	v.SetDefault("grpc.max_recv_msg_size", DefaultGRPCMaxRecvMsgSize)
	v.SetDefault("grpc.max_send_msg_size", DefaultGRPCMaxSendMsgSize)
	v.SetDefault("grpc.stream_poll_interval", DefaultGRPCStreamPollInterval)
	v.SetDefault("grpc.stream_batch_size", DefaultGRPCStreamBatchSize)
	v.SetDefault("grpc.count_cache_ttl", DefaultGRPCCountCacheTTL)
	v.SetDefault("grpc.rbac_role_cache_ttl_seconds", DefaultRBACRoleCacheTTLSeconds)

	// gRPC-web defaults
	v.SetDefault("grpc.enable_reflection", DefaultGRPCEnableReflection)
	v.SetDefault("grpc.enable_health", DefaultGRPCEnableHealth)
	v.SetDefault("grpc.web.enabled", DefaultGRPCWebEnabled)
	v.SetDefault("grpc.web.allowed_origins", DefaultGRPCWebAllowedOrigins)
	v.SetDefault("grpc.web.allowed_headers", DefaultGRPCWebAllowedHeaders)
	v.SetDefault("grpc.web.expose_headers", DefaultGRPCWebExposeHeaders)
	v.SetDefault("grpc.web.allow_credentials", DefaultGRPCWebAllowCredentials)
	v.SetDefault("grpc.web.max_age", GRPCWebDefaultMaxAgeSeconds)
	v.SetDefault("grpc.web.allow_all_origins", DefaultGRPCWebAllowAllOrigins)
	v.SetDefault("grpc.web.enable_websockets", DefaultGRPCWebEnableWebsockets)
	v.SetDefault("grpc.web.allowed_methods", GRPCWebDefaultAllowedMethods)

	// HTTP server timeout defaults
	v.SetDefault("grpc.http.read_timeout", HTTPDefaultReadTimeoutString)
	v.SetDefault("grpc.http.write_timeout", HTTPDefaultWriteTimeoutString)
	v.SetDefault("grpc.http.idle_timeout", HTTPDefaultIdleTimeoutString)

	// Auth defaults
	v.SetDefault("auth.enabled", DefaultAuthEnabled)
	v.SetDefault("auth.jwks_endpoint", DefaultAuthJWKSEndpoint)
	v.SetDefault("auth.issuer", DefaultAuthIssuer)     // Empty default; validate against issuer when set
	v.SetDefault("auth.audience", DefaultAuthAudience) // Empty default; validate against audience when set
	v.SetDefault("auth.tenant_claim", DefaultAuthTenantClaim)
	v.SetDefault("auth.algorithm", DefaultAuthAlgorithm)

	// Local authentication defaults
	v.SetDefault("auth.registration_enabled", DefaultAuthRegistrationEnabled)
	v.SetDefault("auth.local_login_enabled", DefaultAuthLocalLoginEnabled)
	v.SetDefault("auth.hmac_secret", DefaultAuthHMACSecret)
	v.SetDefault("auth.access_token_ttl", DefaultAuthAccessTokenTTL)
	v.SetDefault("auth.refresh_token_ttl", DefaultAuthRefreshTokenTTL)
	v.SetDefault("auth.refresh_token_enabled", DefaultAuthRefreshTokenEnabled)
	v.SetDefault("auth.login_url", DefaultAuthLoginURL)
	v.SetDefault("auth.login_label", AuthDefaultLoginLabel) // Existing constant
	v.SetDefault("auth.login_redirect", DefaultAuthLoginRedirect)
	v.SetDefault("auth.logout_url", DefaultAuthLogoutURL)

	// Bootstrap defaults (no-op unless explicitly configured)
	v.SetDefault("auth.bootstrap.enabled", DefaultAuthBootstrapEnabled)
	v.SetDefault("auth.bootstrap.initial_admin_email", DefaultAuthBootstrapEmail)
	v.SetDefault("auth.bootstrap.password_hash", DefaultAuthBootstrapPasswordHash)
	v.SetDefault("auth.bootstrap.tenant_id", DefaultAuthBootstrapTenantID)
	v.SetDefault("auth.bootstrap.organization_name", DefaultAuthBootstrapOrgName)
	v.SetDefault("auth.bootstrap.create_org", DefaultAuthBootstrapCreateOrg)
	v.SetDefault("auth.bootstrap.create_membership", DefaultAuthBootstrapCreateMembership)

	// External authentication provider defaults
	v.SetDefault("auth.ui_callback_url", DefaultAuthUICallbackURL)

	// OIDC provider defaults
	v.SetDefault("auth.oidc.enabled", DefaultAuthOIDCEnabled)
	v.SetDefault("auth.oidc.provider_url", DefaultAuthOIDCProviderURL)
	v.SetDefault("auth.oidc.client_id", DefaultAuthOIDCClientID)
	v.SetDefault("auth.oidc.client_secret", DefaultAuthOIDCClientSecret)
	v.SetDefault("auth.oidc.redirect_url", DefaultAuthOIDCRedirectURL)
	v.SetDefault("auth.oidc.scopes", AuthDefaultOIDCScopes)   // Existing constant
	v.SetDefault("auth.oidc.login_label", AuthOIDCLoginLabel) // Existing constant
	v.SetDefault("auth.oidc.login_redirect", DefaultAuthOIDCLoginRedirect)
	v.SetDefault("auth.oidc.logout_url", DefaultAuthOIDCLogoutURL)
	v.SetDefault("auth.oidc.registration_enabled", DefaultAuthOIDCRegistrationEnabled)
	v.SetDefault("auth.oidc.registration_callback_url", DefaultAuthOIDCRegistrationCallbackURL)
	v.SetDefault("auth.oidc.assume_email_verified", DefaultAuthOIDCAssumeEmailVerified)
	v.SetDefault("auth.oidc.state_ttl", AuthDefaultStateTTLString) // Existing constant
	v.SetDefault("auth.oidc.nonce_ttl", AuthDefaultNonceTTLString) // Existing constant
	v.SetDefault("auth.oidc.external_org_claim", DefaultAuthOIDCExternalOrgClaim)

	// OAuth2 provider defaults
	v.SetDefault("auth.oauth2.enabled", DefaultAuthOAuth2Enabled)
	v.SetDefault("auth.oauth2.authorize_url", DefaultAuthOAuth2AuthorizeURL)
	v.SetDefault("auth.oauth2.token_url", DefaultAuthOAuth2TokenURL)
	v.SetDefault("auth.oauth2.userinfo_url", DefaultAuthOAuth2UserinfoURL)
	v.SetDefault("auth.oauth2.client_id", DefaultAuthOAuth2ClientID)
	v.SetDefault("auth.oauth2.client_secret", DefaultAuthOAuth2ClientSecret)
	v.SetDefault("auth.oauth2.public_client", DefaultAuthOAuth2PublicClient)
	v.SetDefault("auth.oauth2.redirect_url", DefaultAuthOAuth2RedirectURL)
	v.SetDefault("auth.oauth2.scopes", AuthDefaultOAuth2Scopes)   // Existing constant
	v.SetDefault("auth.oauth2.login_label", AuthOAuth2LoginLabel) // Existing constant
	v.SetDefault("auth.oauth2.login_redirect", DefaultAuthOAuth2LoginRedirect)
	v.SetDefault("auth.oauth2.logout_url", DefaultAuthOAuth2LogoutURL)
	v.SetDefault("auth.oauth2.registration_enabled", DefaultAuthOAuth2RegistrationEnabled)
	v.SetDefault("auth.oauth2.assume_email_verified", DefaultAuthOAuth2AssumeEmailVerified)
	v.SetDefault("auth.oauth2.state_ttl", AuthDefaultStateTTLString)  // Existing constant
	v.SetDefault("auth.oauth2.pkce_method", AuthDefaultPKCEMethod)    // Existing constant
	v.SetDefault("auth.oauth2.user_id_claim", AuthDefaultUserIDClaim) // Existing constant
	v.SetDefault("auth.oauth2.email_claim", AuthDefaultEmailClaim)    // Existing constant

	// Redis defaults (required when OIDC or OAuth2 enabled)
	v.SetDefault("redis.host", DefaultRedisHost)
	v.SetDefault("redis.port", DefaultRedisPort)
	v.SetDefault("redis.password", DefaultRedisPassword)
	v.SetDefault("redis.database", DefaultRedisDatabase)
	v.SetDefault("redis.pool_size", DefaultRedisPoolSize)

	// Alert service defaults
	v.SetDefault("alerts.summary_lookback_hours", DefaultAlertsSummaryLookbackHours)
	v.SetDefault("alerts.recent_alerts_limit", DefaultAlertsRecentAlertsLimit)

	// Analytics service defaults
	v.SetDefault("analytics.default_window_hours", DefaultAnalyticsWindowHours)
	v.SetDefault("analytics.activity_window_days", DefaultAnalyticsActivityDays)
	v.SetDefault("analytics.top_endpoints_limit", DefaultAnalyticsTopEndpointsLimit)

	// Certificate service defaults
	v.SetDefault("certificates.certgen_path", DefaultCertificatesCertGenPath)
	v.SetDefault("certificates.certs_dir", DefaultCertificatesCertsDir)
	v.SetDefault("certificates.temp_dir", DefaultCertificatesTempDir)
	v.SetDefault("certificates.cleanup_interval_min", DefaultCertificatesCleanupIntervalMin)
	v.SetDefault("certificates.server_validity_days", DefaultCertificatesServerValidityDays)

	// Registry provider defaults
	v.SetDefault("registry_provider.enabled", DefaultRegistryProviderEnabled)
	v.SetDefault("registry_provider.api_url", DefaultRegistryProviderAPIURL)
	v.SetDefault("registry_provider.base_branch", DefaultRegistryProviderBaseBranch)
	v.SetDefault("registry_provider.branch_prefix", DefaultRegistryProviderBranchPrefix)
	v.SetDefault("registry_provider.blueprint_path", DefaultRegistryProviderBlueprintPath)
	v.SetDefault("registry_provider.file_extension", DefaultRegistryProviderFileExtension)
	v.SetDefault("registry_provider.http_timeout", DefaultRegistryProviderHTTPTimeout)
	v.SetDefault("registry_provider.schema_url", DefaultRegistryProviderSchemaURL)
	v.SetDefault("registry_provider.token", DefaultRegistryProviderToken)
	v.SetDefault("registry_provider.auth_mode", DefaultRegistryProviderAuthMode)

	// Status monitoring defaults
	v.SetDefault("status.timeout", DefaultStatusTimeoutString)

	// Gateway defaults
	v.SetDefault("gateway.upstream_address", DefaultGatewayUpstreamAddress)
	v.SetDefault("gateway.identity_address", DefaultGatewayIdentityAddress)
	v.SetDefault("gateway.health_port", DefaultGatewayHealthPort)

	// Gateway rate limit defaults
	v.SetDefault("gateway.rate_limit.enabled", DefaultGatewayRateLimitEnabled)
	v.SetDefault("gateway.rate_limit.requests_per_min", DefaultGatewayRateLimitRequestsPerMin)
	v.SetDefault("gateway.rate_limit.burst", DefaultGatewayRateLimitBurst)
	v.SetDefault("gateway.rate_limit.cleanup_interval", DefaultGatewayRateLimitCleanupStr)

	// Gateway resilience defaults
	v.SetDefault("gateway.resilience.dial_timeout", DefaultGatewayResilienceDialTimeoutStr)
	v.SetDefault("gateway.resilience.rpc_timeout", DefaultGatewayResilienceRPCTimeoutStr)
	v.SetDefault("gateway.resilience.max_retries", DefaultGatewayResilienceMaxRetries)
	v.SetDefault("gateway.resilience.retry_backoff", DefaultGatewayResilienceRetryBackoffStr)
	v.SetDefault("gateway.resilience.retry_max_backoff", DefaultGatewayResilienceRetryMaxBackoffStr)
	v.SetDefault("gateway.resilience.cb_max_requests", DefaultGatewayResilienceCBMaxRequests)
	v.SetDefault("gateway.resilience.cb_interval", DefaultGatewayResilienceCBIntervalStr)
	v.SetDefault("gateway.resilience.cb_timeout", DefaultGatewayResilienceCBTimeoutStr)
	v.SetDefault("gateway.resilience.cb_failure_threshold", DefaultGatewayResilienceCBFailureThreshold)

	// Internal auth defaults
	v.SetDefault("internal_auth.peer_secret", DefaultInternalAuthPeerSecret)

	// Identity service defaults
	v.SetDefault("identity.address", DefaultIdentityAddress)
}
