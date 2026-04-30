package config

import (
	"strings"
	"testing"
)

// minValidConfig returns a Config that passes all existing validation.
// Tests override individual fields to trigger specific CE hard-fail rules.
func minValidConfig() Config {
	return Config{
		General: GeneralConfig{
			ServerName: "test",
			TenantID:   1,
			Edition:    EditionECE, // default: enterprise
		},
		Storage: StorageConfig{
			Type: StorageTypePostgres,
			Host: "localhost",
			Port: 5432,
		},
		GRPC: GRPCConfig{
			Enabled: true,
			Port:    50051,
		},
	}
}

func TestValidate_CE_OrgEnforcementIncompatible(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.General.OrgEnforcementEnabled = true

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for CE + org_enforcement_enabled")
	}
	if !strings.Contains(err.Error(), ErrCEOrgEnforcementIncompatible) {
		t.Fatalf("expected CE org enforcement error, got: %v", err)
	}
}

func TestValidate_CE_StrictOrgResolutionIncompatible(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.Protocol.StrictOrgResolution = true

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for CE + strict_org_resolution")
	}
	if !strings.Contains(err.Error(), ErrCEStrictOrgResolutionIncompatible) {
		t.Fatalf("expected CE strict org error, got: %v", err)
	}
}

func TestValidate_CE_CertTenantMappingIncompatible(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.Protocol.SCACICertTenantMapping = true

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for CE + scaci_cert_tenant_mapping")
	}
	if !strings.Contains(err.Error(), ErrCECertTenantMappingIncompatible) {
		t.Fatalf("expected CE cert tenant mapping error, got: %v", err)
	}
}

func TestValidate_CE_ExternalOrgClaimIncompatible(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.Auth.OIDC.ExternalOrgClaim = "org.id"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for CE + external_org_claim")
	}
	if !strings.Contains(err.Error(), ErrCEExternalOrgClaimIncompatible) {
		t.Fatalf("expected CE external org claim error, got: %v", err)
	}
}

func TestValidate_CE_TenantIDRequired(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.General.TenantID = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for CE + missing tenant_id")
	}
	if !strings.Contains(err.Error(), ErrCETenantIDRequired) {
		t.Fatalf("expected CE tenant ID error, got: %v", err)
	}
}

func TestValidate_CE_ValidConfig(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.General.TenantID = 1

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected valid CE config, got error: %v", err)
	}
}

func TestValidate_ECE_AllowsOrgEnforcement(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionECE
	cfg.General.OrgEnforcementEnabled = true

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("ECE should allow org enforcement, got: %v", err)
	}
}
