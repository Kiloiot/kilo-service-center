package diagnostics

import "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"

// EffectiveConfig is the explicit, non-secret view of the configuration the
// bundle carries: no passwords, DSNs, key or certificate paths, tokens or
// JWT secrets, and no payload-bearing settings.
type EffectiveConfig struct {
	General    GeneralView    `json:"general"`
	Storage    StorageView    `json:"storage"`
	Protocol   ProtocolView   `json:"protocol"`
	GRPC       GRPCView       `json:"grpc"`
	MQTT       MQTTView       `json:"mqtt"`
	Auth       AuthView       `json:"auth"`
	Monitoring MonitoringView `json:"monitoring"`
}

// GeneralView projects config.GeneralConfig.
type GeneralView struct {
	ServerName            string `json:"serverName"`
	Environment           string `json:"environment"`
	LogLevel              string `json:"logLevel"`
	Edition               string `json:"edition"`
	OrgEnforcementEnabled bool   `json:"orgEnforcementEnabled"`
	ActivityWindowHours   int    `json:"activityWindowHours"`
	TenantID              int64  `json:"tenantId"`
}

// StorageView projects config.StorageConfig without credentials.
type StorageView struct {
	Type                 string `json:"type"`
	Host                 string `json:"host"`
	Port                 int    `json:"port"`
	Database             string `json:"database"`
	SSLMode              string `json:"sslMode"`
	MaxConnections       int    `json:"maxConnections"`
	MessageRetentionDays int    `json:"messageRetentionDays"`
	ArchivalEnabled      *bool  `json:"archivalEnabled"`
}

// ProtocolView projects config.ProtocolConfig without certificate paths.
type ProtocolView struct {
	BSCIHost                  string `json:"bsciHost"`
	BSCIPort                  int    `json:"bsciPort"`
	BSCIExternalURL           string `json:"bsciExternalUrl"`
	BSCITLSMinVersion         string `json:"bsciTlsMinVersion"`
	SCACIEnabled              bool   `json:"scaciEnabled"`
	SCACIHost                 string `json:"scaciHost"`
	SCACIPort                 int    `json:"scaciPort"`
	SCACITLSMinVersion        string `json:"scaciTlsMinVersion"`
	SCACICertTenantMapping    bool   `json:"scaciCertTenantMapping"`
	StrictOrgResolution       bool   `json:"strictOrgResolution"`
	AckTimeoutMs              int    `json:"ackTimeoutMs"`
	DuplicateWindowSeconds    int    `json:"duplicateWindowSeconds"`
	MessageEncoding           string `json:"messageEncoding"`
	StatusRequestIntervalSecs int    `json:"statusRequestIntervalSeconds"`
	RoamingEnabled            bool   `json:"roamingEnabled"`
	FederationEnabled         bool   `json:"federationEnabled"`
	SCVendor                  string `json:"scVendor"`
	SCModel                   string `json:"scModel"`
}

// GRPCView projects config.GRPCConfig.
type GRPCView struct {
	Enabled              bool   `json:"enabled"`
	Host                 string `json:"host"`
	Port                 int    `json:"port"`
	InternalTrustEnabled bool   `json:"internalTrustEnabled"`
	TLSEnabled           bool   `json:"tlsEnabled"`
	WebEnabled           bool   `json:"webEnabled"`
}

// MQTTView projects config.MQTTConfig without credentials.
type MQTTView struct {
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	TopicPrefix string `json:"topicPrefix"`
	TLSEnabled  bool   `json:"tlsEnabled"`
}

// AuthView projects config.AuthConfig without secrets or endpoints.
type AuthView struct {
	Enabled             bool   `json:"enabled"`
	LocalLoginEnabled   bool   `json:"localLoginEnabled"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
	RefreshTokenEnabled bool   `json:"refreshTokenEnabled"`
	Issuer              string `json:"issuer"`
	Audience            string `json:"audience"`
	OIDCEnabled         bool   `json:"oidcEnabled"`
	OAuth2Enabled       bool   `json:"oauth2Enabled"`
}

// MonitoringView projects config.MonitoringConfig.
type MonitoringView struct {
	MetricsEnabled bool `json:"metricsEnabled"`
	MetricsPort    int  `json:"metricsPort"`
	TracingEnabled bool `json:"tracingEnabled"`
}

// ProjectConfig copies the allowlisted fields; nothing else leaves the process.
func ProjectConfig(cfg *config.Config) EffectiveConfig {
	if cfg == nil {
		return EffectiveConfig{}
	}
	return EffectiveConfig{
		General: GeneralView{
			ServerName: cfg.General.ServerName, Environment: cfg.General.Environment, LogLevel: cfg.General.LogLevel,
			Edition: cfg.General.Edition, OrgEnforcementEnabled: cfg.General.OrgEnforcementEnabled,
			ActivityWindowHours: cfg.General.ActivityWindowHours, TenantID: cfg.General.TenantID,
		},
		Storage: StorageView{
			Type: cfg.Storage.Type, Host: cfg.Storage.Host, Port: cfg.Storage.Port, Database: cfg.Storage.Database,
			SSLMode: cfg.Storage.SSLMode, MaxConnections: cfg.Storage.MaxConnections,
			MessageRetentionDays: cfg.Storage.MessageRetentionDays, ArchivalEnabled: cfg.Storage.ArchivalEnabled,
		},
		Protocol: ProtocolView{
			BSCIHost: cfg.Protocol.BSCIHost, BSCIPort: cfg.Protocol.BSCIPort, BSCIExternalURL: cfg.Protocol.BSCIExternalURL,
			BSCITLSMinVersion: cfg.Protocol.BSCITLS.MinVersion, SCACIEnabled: cfg.Protocol.SCACIEnabled,
			SCACIHost: cfg.Protocol.SCACIHost, SCACIPort: cfg.Protocol.SCACIPort, SCACITLSMinVersion: cfg.Protocol.SCACITLS.MinVersion,
			SCACICertTenantMapping: cfg.Protocol.SCACICertTenantMapping, StrictOrgResolution: cfg.Protocol.StrictOrgResolution,
			AckTimeoutMs: cfg.Protocol.AckTimeout, DuplicateWindowSeconds: cfg.Protocol.DuplicateWindow,
			MessageEncoding: cfg.Protocol.MessageEncoding, StatusRequestIntervalSecs: cfg.Protocol.StatusRequestInterval,
			RoamingEnabled: cfg.Protocol.Roaming.Enabled, FederationEnabled: cfg.Protocol.Federation.Enabled,
			SCVendor: cfg.Protocol.SCVendor, SCModel: cfg.Protocol.SCModel,
		},
		GRPC: GRPCView{
			Enabled: cfg.GRPC.Enabled, Host: cfg.GRPC.Host, Port: cfg.GRPC.Port, InternalTrustEnabled: cfg.GRPC.InternalTrustEnabled,
			TLSEnabled: cfg.GRPC.EnableTLS, WebEnabled: cfg.GRPC.Web.Enabled,
		},
		MQTT: MQTTView{Enabled: cfg.MQTT.Enabled, Host: cfg.MQTT.Host, Port: cfg.MQTT.Port, TopicPrefix: cfg.MQTT.TopicPrefix, TLSEnabled: cfg.MQTT.TLS.Enabled},
		Auth: AuthView{
			Enabled: cfg.Auth.Enabled, LocalLoginEnabled: cfg.Auth.LocalLoginEnabled, RegistrationEnabled: cfg.Auth.RegistrationEnabled,
			RefreshTokenEnabled: cfg.Auth.RefreshTokenEnabled, Issuer: cfg.Auth.Issuer, Audience: cfg.Auth.Audience,
			OIDCEnabled: cfg.Auth.OIDC.Enabled, OAuth2Enabled: cfg.Auth.OAuth2.Enabled,
		},
		Monitoring: MonitoringView{MetricsEnabled: cfg.Monitoring.MetricsEnabled, MetricsPort: cfg.Monitoring.MetricsPort, TracingEnabled: cfg.Monitoring.TracingEnabled},
	}
}
