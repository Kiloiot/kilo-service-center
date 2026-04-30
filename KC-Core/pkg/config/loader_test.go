package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestInternalTrustStartupFailsWithGRPCWeb(t *testing.T) {
	cfg := &Config{
		General: GeneralConfig{ServerName: "test"},
		Storage: StorageConfig{Type: "postgres", Host: "localhost", Port: 5432},
		GRPC: GRPCConfig{
			InternalTrustEnabled: true,
			Web:                  GRPCWebConfig{Enabled: true},
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
		General: GeneralConfig{ServerName: "test"},
		Storage: StorageConfig{Type: "postgres", Host: "localhost", Port: 5432},
	}
}

func TestValidate_RegistrationRequiresLocalLogin(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Auth.RegistrationEnabled = true
	cfg.Auth.LocalLoginEnabled = false

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
	cfg.Auth.Enabled = true
	cfg.Auth.RegistrationEnabled = true
	cfg.Auth.LocalLoginEnabled = true
	cfg.Auth.HMACSecret = "this-is-a-secret-that-is-at-least-32-bytes-long!"

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error for valid registration config, got: %v", err)
	}
}

func TestValidate_RateLimitRequestsPerMinPositive(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Gateway.RateLimit.Enabled = true
	cfg.Gateway.RateLimit.RequestsPerMin = 0

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
	cfg.Gateway.RateLimit.Enabled = true
	cfg.Gateway.RateLimit.RequestsPerMin = 5
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
	cfg.Gateway.RateLimit.Enabled = true
	cfg.Gateway.RateLimit.RequestsPerMin = 5
	cfg.Gateway.RateLimit.Burst = 3
	cfg.Gateway.RateLimit.CleanupInterval = 0

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
	cfg.Gateway.RateLimit.Enabled = false

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error when rate limiting disabled, got: %v", err)
	}
}
