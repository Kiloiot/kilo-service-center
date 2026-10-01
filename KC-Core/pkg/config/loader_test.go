package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Rate-limit fixture values for the loader validation tests.
const (
	invalidZeroLimit   = 0
	testRequestsPerMin = 5
)

// testConfigWithoutGRPCWeb sets a gRPC port and nothing about gRPC-web.
const testConfigWithoutGRPCWeb = `
grpc:
  port: 9090
`

func writeTestConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadGateway_EnablesGRPCWebWhenTheKeyIsAbsent(t *testing.T) {
	cfg, err := LoadGateway(writeTestConfig(t, testConfigWithoutGRPCWeb))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.GRPC.Web.Enabled {
		t.Error("the gateway is the browser ingress: a config without grpc.web.enabled must still serve gRPC-web")
	}
}

func TestLoadCore_KeepsGRPCWebOffWhenTheKeyIsAbsent(t *testing.T) {
	cfg, err := Load(writeTestConfig(t, testConfigWithoutGRPCWeb))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.GRPC.Web.Enabled {
		t.Error("KC-Core leaves gRPC-web to the gateway by default")
	}
}

func TestLoad_RegistryProviderTokenFromEnv(t *testing.T) {
	// Write a minimal config YAML with token: "" (mirrors production config.yaml)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
registry_provider:
  enabled: true
  api_url: "https://api.github.com"
  token: ""
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	const testToken = "ghp_test1234567890"

	t.Setenv("KILOCENTER_REGISTRY_PROVIDER_TOKEN", testToken)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.RegistryProvider.Token != testToken {
		t.Errorf("RegistryProvider.Token = %q, want %q", cfg.RegistryProvider.Token, testToken)
	}
}

func TestLoad_RegistryProviderTokenFromEnv_NoYAMLKey(t *testing.T) {
	// Config YAML with no token key at all — env var should still work via BindEnv
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
registry_provider:
  enabled: true
  api_url: "https://api.github.com"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	const testToken = "ghp_nokey_test9876"

	t.Setenv("KILOCENTER_REGISTRY_PROVIDER_TOKEN", testToken)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.RegistryProvider.Token != testToken {
		t.Errorf("RegistryProvider.Token = %q, want %q", cfg.RegistryProvider.Token, testToken)
	}
}

func TestLoad_RegistryProviderTokenEmpty_WhenNoEnv(t *testing.T) {
	// No env var set — token should remain empty
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
registry_provider:
  enabled: true
  api_url: "https://api.github.com"
  token: ""
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	// Ensure env var is not set
	t.Setenv("KILOCENTER_REGISTRY_PROVIDER_TOKEN", "")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.RegistryProvider.Token != "" {
		t.Errorf("RegistryProvider.Token = %q, want empty", cfg.RegistryProvider.Token)
	}
}

// writeSCEUIConfig writes a minimal config file, optionally including a protocol.sc_eui value.
func writeSCEUIConfig(t *testing.T, scEUIYAML string) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := "general:\n  server_name: \"test\"\n"
	if scEUIYAML != "" {
		yaml += "protocol:\n  sc_eui: \"" + scEUIYAML + "\"\n"
	}
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return cfgPath
}

// clearSCEUIEnv unsets both Service Center EUI environment variables for the test.
func clearSCEUIEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvProtocolSCEUI, "")
	t.Setenv(EnvLegacyServiceCenterEUI, "")
}

func TestLoad_SCEUIDefault(t *testing.T) {
	clearSCEUIEnv(t)
	cfg, err := Load(writeSCEUIConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Protocol.SCEUI != DefaultProtocolSCEUI {
		t.Errorf("SCEUI = %q, want %q", cfg.Protocol.SCEUI, DefaultProtocolSCEUI)
	}
	if cfg.Protocol.SCEUIValue != 0x4B43000000000001 {
		t.Errorf("SCEUIValue = %#016x, want 0x4B43000000000001", cfg.Protocol.SCEUIValue)
	}
	if cfg.Protocol.SCEUILegacyEnvUsed {
		t.Error("SCEUILegacyEnvUsed = true, want false for default")
	}
}

func TestLoad_RoamingAndDeliveryDefaults(t *testing.T) {
	clearSCEUIEnv(t)
	cfg, err := Load(writeSCEUIConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	roaming := cfg.Protocol.Roaming
	if roaming.CacheEnabled != DefaultProtocolRoamingCacheEnabled || roaming.CacheTTL != DefaultProtocolRoamingCacheTTL ||
		roaming.CacheMaxSize != DefaultProtocolRoamingCacheMaxSize || roaming.EnableAuditTrail != DefaultProtocolRoamingEnableAuditTrail {
		t.Errorf("protocol.roaming = %+v, want the documented defaults", roaming)
	}
	if cfg.Protocol.Delivery.MaxBackoff != DefaultProtocolDeliveryMaxBackoff {
		t.Errorf("protocol.delivery.max_backoff = %v, want %v", cfg.Protocol.Delivery.MaxBackoff, DefaultProtocolDeliveryMaxBackoff)
	}
	if cfg.Protocol.Delivery.ReceptionWindow != DefaultProtocolDeliveryReceptionWindow {
		t.Errorf("protocol.delivery.reception_window = %v, want %v", cfg.Protocol.Delivery.ReceptionWindow, DefaultProtocolDeliveryReceptionWindow)
	}
	if cfg.Protocol.DownlinkExpiry.Lifetime != DefaultProtocolDownlinkLifetime ||
		cfg.Protocol.DownlinkExpiry.SweepInterval != DefaultProtocolDownlinkExpirySweepInterval ||
		cfg.Protocol.DownlinkExpiry.BatchSize != DefaultProtocolDownlinkExpiryBatchSize ||
		!slices.Equal(cfg.Protocol.DownlinkExpiry.RevokeNotHeldCodes, DefaultProtocolDownlinkRevokeNotHeldCodes) {
		t.Errorf("protocol.downlink_expiry = %+v, want the documented defaults", cfg.Protocol.DownlinkExpiry)
	}
	if cfg.Protocol.SCACIResumeMaxPendingOperations != DefaultProtocolSCACIResumeMaxPendingOperations {
		t.Errorf("protocol.scaci_resume_max_pending_operations = %d, want %d",
			cfg.Protocol.SCACIResumeMaxPendingOperations, DefaultProtocolSCACIResumeMaxPendingOperations)
	}
}

// TestLoad_RevokeNotHeldCodesFromEnvironment: an operator names the codes of
// its stations as a comma-separated environment variable.
func TestLoad_RevokeNotHeldCodesFromEnvironment(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv("KILOCENTER_PROTOCOL_DOWNLINK_EXPIRY_REVOKE_NOT_HELD_CODES", "2,3")
	cfg, err := Load(writeSCEUIConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if want := []int{2, 3}; !slices.Equal(cfg.Protocol.DownlinkExpiry.RevokeNotHeldCodes, want) {
		t.Errorf("revoke_not_held_codes = %v, want %v", cfg.Protocol.DownlinkExpiry.RevokeNotHeldCodes, want)
	}
}

func TestLoad_SCEUIFromFile(t *testing.T) {
	clearSCEUIEnv(t)
	cfg, err := Load(writeSCEUIConfig(t, "CA-FE-CA-FE-CA-FE-CA-FE"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Protocol.SCEUIValue != 0xCAFECAFECAFECAFE {
		t.Errorf("SCEUIValue = %#016x, want 0xCAFECAFECAFECAFE", cfg.Protocol.SCEUIValue)
	}
	if cfg.Protocol.SCEUI != "CAFECAFECAFECAFE" {
		t.Errorf("SCEUI = %q, want canonical %q", cfg.Protocol.SCEUI, "CAFECAFECAFECAFE")
	}
}

func TestLoad_SCEUIModernEnvWinsOverFile(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvProtocolSCEUI, "0102030405060708")
	cfg, err := Load(writeSCEUIConfig(t, "CAFECAFECAFECAFE"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Protocol.SCEUIValue != 0x0102030405060708 {
		t.Errorf("SCEUIValue = %#016x, want 0x0102030405060708", cfg.Protocol.SCEUIValue)
	}
}

func TestLoad_SCEUILegacyEnvDecimal(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvLegacyServiceCenterEUI, "1234567890")
	cfg, err := Load(writeSCEUIConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Protocol.SCEUIValue != 1234567890 {
		t.Errorf("SCEUIValue = %d, want 1234567890", cfg.Protocol.SCEUIValue)
	}
	if cfg.Protocol.SCEUI != "00000000499602D2" {
		t.Errorf("SCEUI = %q, want canonical %q", cfg.Protocol.SCEUI, "00000000499602D2")
	}
	if !cfg.Protocol.SCEUILegacyEnvUsed {
		t.Error("SCEUILegacyEnvUsed = false, want true")
	}
}

func TestLoad_SCEUILegacyEnvHexHighBit(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvLegacyServiceCenterEUI, "0xCAFECAFECAFECAFE")
	cfg, err := Load(writeSCEUIConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Protocol.SCEUIValue != 0xCAFECAFECAFECAFE {
		t.Errorf("SCEUIValue = %#016x, want 0xCAFECAFECAFECAFE", cfg.Protocol.SCEUIValue)
	}
	if !cfg.Protocol.SCEUILegacyEnvUsed {
		t.Error("SCEUILegacyEnvUsed = false, want true")
	}
}

func TestLoad_SCEUIMalformedModernEnvFails(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvProtocolSCEUI, "not-an-eui")
	t.Setenv(EnvLegacyServiceCenterEUI, "0x4B43000000000001")
	if _, err := Load(writeSCEUIConfig(t, "CAFECAFECAFECAFE")); err == nil {
		t.Fatal("expected load failure for malformed modern env value, got nil")
	}
}

func TestLoad_SCEUIMalformedFileFails(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvLegacyServiceCenterEUI, "0x4B43000000000001")
	if _, err := Load(writeSCEUIConfig(t, "ZZZZ")); err == nil {
		t.Fatal("expected load failure for malformed file value, got nil")
	}
}

func TestLoad_SCEUIMalformedLegacyFails(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvLegacyServiceCenterEUI, "banana")
	if _, err := Load(writeSCEUIConfig(t, "")); err == nil {
		t.Fatal("expected load failure for malformed legacy env value, got nil")
	}
}

func TestLoad_SCEUILegacyIgnoredWhenModernEnvSet(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvProtocolSCEUI, "CAFECAFECAFECAFE")
	t.Setenv(EnvLegacyServiceCenterEUI, "banana")
	cfg, err := Load(writeSCEUIConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Protocol.SCEUIValue != 0xCAFECAFECAFECAFE {
		t.Errorf("SCEUIValue = %#016x, want 0xCAFECAFECAFECAFE", cfg.Protocol.SCEUIValue)
	}
	if cfg.Protocol.SCEUILegacyEnvUsed {
		t.Error("SCEUILegacyEnvUsed = true, want false when modern env supplies the value")
	}
}

func TestLoad_SCEUILegacyIgnoredWhenFileSet(t *testing.T) {
	clearSCEUIEnv(t)
	t.Setenv(EnvLegacyServiceCenterEUI, "1234567890")
	cfg, err := Load(writeSCEUIConfig(t, "0102030405060708"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Protocol.SCEUIValue != 0x0102030405060708 {
		t.Errorf("SCEUIValue = %#016x, want 0x0102030405060708", cfg.Protocol.SCEUIValue)
	}
	if cfg.Protocol.SCEUILegacyEnvUsed {
		t.Error("SCEUILegacyEnvUsed = true, want false when file supplies the value")
	}
}

func TestInternalTrustStartupFailsWithGRPCWeb(t *testing.T) {
	cfg := &Config{
		General: GeneralConfig{ServerName: "test", TenantID: DefaultGeneralTenantID},
		Storage: StorageConfig{Type: "postgres", Host: "localhost", Port: DefaultStoragePort},
		GRPC: GRPCConfig{
			InternalTrustEnabled: flagEnabled,
			Web:                  GRPCWebConfig{Enabled: flagEnabled},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for internal_trust_enabled + web.enabled")
	}
	if !strings.Contains(err.Error(), "internal_trust") {
		t.Errorf("expected error about internal trust conflict, got: %v", err)
	}
}

// validBaseConfig returns a minimal Config that passes general and storage validation.
func validBaseConfig() *Config {
	return &Config{
		General:      GeneralConfig{ServerName: "test", TenantID: DefaultGeneralTenantID},
		Storage:      StorageConfig{Type: "postgres", Host: "localhost", Port: DefaultStoragePort},
		Certificates: defaultCertificateConfig(),
	}
}

func TestValidate_RegistrationRequiresLocalLogin(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Auth.RegistrationEnabled = flagEnabled
	cfg.Auth.LocalLoginEnabled = flagDisabled

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when registration_enabled=true without local_login_enabled")
	}
	if !strings.Contains(err.Error(), "registration_enabled requires local_login_enabled") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_RegistrationWithLocalLoginPasses(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Auth.Enabled = flagEnabled
	cfg.Auth.RegistrationEnabled = flagEnabled
	cfg.Auth.LocalLoginEnabled = flagEnabled
	cfg.Auth.HMACSecret = "this-is-a-secret-that-is-at-least-32-bytes-long!"

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error for valid registration config, got: %v", err)
	}
}

func TestValidate_RateLimitRequestsPerMinPositive(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Gateway.RateLimit.Enabled = flagEnabled
	cfg.Gateway.RateLimit.RequestsPerMin = invalidZeroLimit

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for requests_per_min=0")
	}
	if !strings.Contains(err.Error(), "requests_per_min must be positive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_RateLimitBurstPositive(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Gateway.RateLimit.Enabled = flagEnabled
	cfg.Gateway.RateLimit.RequestsPerMin = testRequestsPerMin
	cfg.Gateway.RateLimit.Burst = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for burst=0")
	}
	if !strings.Contains(err.Error(), "burst must be positive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_RateLimitCleanupIntervalPositive(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Gateway.RateLimit.Enabled = flagEnabled
	cfg.Gateway.RateLimit.RequestsPerMin = testRequestsPerMin
	cfg.Gateway.RateLimit.Burst = 3
	cfg.Gateway.RateLimit.CleanupInterval = invalidZeroLimit

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for cleanup_interval=0")
	}
	if !strings.Contains(err.Error(), "cleanup_interval must be positive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_RateLimitDisabledSkipsValidation(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Gateway.RateLimit.Enabled = flagDisabled

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error when rate limiting disabled, got: %v", err)
	}
}

func TestLoad_RegistryProviderBlueprintPathDefaultsEmpty(t *testing.T) {
	// Manufacturer directories sit at the registry root, so no path prefix is added.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
registry_provider:
  enabled: true
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	t.Setenv("KILOCENTER_REGISTRY_PROVIDER_BLUEPRINT_PATH", "")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.RegistryProvider.BlueprintPath != "" {
		t.Errorf("RegistryProvider.BlueprintPath = %q, want empty", cfg.RegistryProvider.BlueprintPath)
	}
}

func TestLoad_RegistryProviderBlueprintPathFromEnv(t *testing.T) {
	// A registry that nests blueprints under a prefix is supported without a rebuild.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
registry_provider:
  enabled: true
  blueprint_path: ""
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	const wantPath = "blueprints/"

	t.Setenv("KILOCENTER_REGISTRY_PROVIDER_BLUEPRINT_PATH", wantPath)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.RegistryProvider.BlueprintPath != wantPath {
		t.Errorf("RegistryProvider.BlueprintPath = %q, want %q", cfg.RegistryProvider.BlueprintPath, wantPath)
	}
}
