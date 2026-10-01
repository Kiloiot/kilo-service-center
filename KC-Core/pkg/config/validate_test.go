package config

import (
	"fmt"
	"strings"
	"testing"
)

// testExternalOrgClaim is the OIDC claim path fixture used by edition-gating tests.
const testExternalOrgClaim = "org.id"

// testCustomEdition is an edition neither CE nor ECE rules cover.
const testCustomEdition = "custom"

// testUnsetSetting is a numeric setting left at zero.
const testUnsetSetting = 0

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
			Port: DefaultStoragePort,
		},
		GRPC: GRPCConfig{
			Enabled: flagEnabled,
			Port:    DefaultGRPCPort,
		},
		Certificates: defaultCertificateConfig(),
	}
}

// defaultCertificateConfig returns the certificate settings the loader defaults to.
func defaultCertificateConfig() CertificateConfig {
	return CertificateConfig{
		CertGenPath:        DefaultCertificatesCertGenPath,
		CertsDir:           DefaultCertificatesCertsDir,
		TempDir:            DefaultCertificatesTempDir,
		ServerValidityDays: DefaultCertificatesServerValidityDays,
		CleanupIntervalMin: DefaultCertificatesCleanupIntervalMin,
	}
}

// TestValidate_CertificateSettings refuses every certificate setting the
// certificate service cannot run with; the loader's defaults are the only
// fallback, so a setting that is present must be usable.
func TestValidate_CertificateSettings(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*CertificateConfig)
		wantErr string
	}{
		{name: "cleanup interval zero", mutate: func(c *CertificateConfig) { c.CleanupIntervalMin = testUnsetSetting }, wantErr: ErrCertificatesCleanupIntervalPositive},
		{name: "server validity zero", mutate: func(c *CertificateConfig) { c.ServerValidityDays = testUnsetSetting }, wantErr: ErrCertificatesServerValidityDaysPositive},
		{name: "certgen path empty", mutate: func(c *CertificateConfig) { c.CertGenPath = "" }, wantErr: ErrCertificatesCertGenPathRequired},
		{name: "certs dir empty", mutate: func(c *CertificateConfig) { c.CertsDir = "" }, wantErr: ErrCertificatesCertsDirRequired},
		{name: "temp dir empty", mutate: func(c *CertificateConfig) { c.TempDir = "" }, wantErr: ErrCertificatesTempDirRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := minValidConfig()
			tc.mutate(&cfg.Certificates)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestValidate_StrictOrgResolutionRequiresCertTenantMapping holds for every
// edition with a SCACI listener: strict resolution cannot resolve a peer
// without mapping its certificate to a tenant.
func TestSCACIListener_StrictOrgResolutionRequiresCertTenantMapping(t *testing.T) {
	for _, edition := range []string{testCustomEdition, EditionECE} {
		cfg := minValidConfig()
		cfg.General.Edition = edition
		cfg.Protocol.SCACIEnabled = flagEnabled
		cfg.Protocol.StrictOrgResolution = flagEnabled

		err := cfg.validateSCACIListener()
		if err == nil || !strings.Contains(err.Error(), ErrStrictOrgResolutionRequiresCertTenantMapping) {
			t.Fatalf("edition %q: expected %q, got: %v", edition, ErrStrictOrgResolutionRequiresCertTenantMapping, err)
		}

		cfg.Protocol.SCACIEnabled = flagDisabled
		if err := cfg.validateSCACIListener(); err != nil {
			t.Fatalf("edition %q without a SCACI listener: %v", edition, err)
		}
	}
}

func TestValidate_CE_OrgEnforcementIncompatible(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.General.OrgEnforcementEnabled = flagEnabled

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
	cfg.Auth.OIDC.ExternalOrgClaim = testExternalOrgClaim

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for CE + external_org_claim")
	}
	if !strings.Contains(err.Error(), ErrCEExternalOrgClaimIncompatible) {
		t.Fatalf("expected CE external org claim error, got: %v", err)
	}
}

// TestValidate_PlatformTenantIDRequired: every edition files operator and
// audit events under the platform tenant, so none may run without it.
func TestValidate_PlatformTenantIDRequired(t *testing.T) {
	for _, edition := range []string{EditionCommunity, EditionECE} {
		cfg := minValidConfig()
		cfg.General.Edition = edition
		cfg.General.TenantID = 0

		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), ErrPlatformTenantIDRequired) {
			t.Fatalf("edition %s without tenant_id: got %v, want the platform tenant error", edition, err)
		}
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

// eceConfig is an ECE configuration every binary accepts apart from
// general.org_enforcement_enabled, which it sets to orgEnforcement.
func eceConfig(orgEnforcement bool) string {
	return fmt.Sprintf("general:\n  edition: %q\n  org_enforcement_enabled: %t\n"+
		"protocol:\n  strict_org_resolution: true\n  scaci_cert_tenant_mapping: true\n"+
		"internal_auth:\n  peer_secret: %q\n", EditionECE, orgEnforcement, testPeerSecret)
}

// ECE identifies base stations and Application Centers by their organization
// certificates, so KC-Core and KC-Gateway, which act on organization
// enforcement, refuse to start without it. KC-Identity never reads the
// setting and starts either way.
func TestLoad_ECE_OrgEnforcementRequiredByCoreAndGatewayOnly(t *testing.T) {
	for name, load := range map[string]func(string) (*Config, error){"KC-Core": Load, "KC-Gateway": LoadGateway} {
		_, err := load(writeTestConfig(t, eceConfig(false)))
		if err == nil || !strings.Contains(err.Error(), ErrECEOrgEnforcementRequired) {
			t.Errorf("%s: expected %q, got: %v", name, ErrECEOrgEnforcementRequired, err)
		}
		if _, err := load(writeTestConfig(t, eceConfig(true))); err != nil {
			t.Errorf("%s: ECE with organization enforcement must load, got: %v", name, err)
		}
	}
	if _, err := LoadIdentity(writeTestConfig(t, eceConfig(false))); err != nil {
		t.Errorf("KC-Identity must load ECE without organization enforcement, got: %v", err)
	}
}

// eceServiceConfig is an ECE configuration without a protocol block, the
// shape Kilo Cloud gives KC-Identity and KC-Gateway: SCACI is left at its
// default, enabled.
func eceServiceConfig() string {
	return fmt.Sprintf("general:\n  edition: %q\n  org_enforcement_enabled: true\n"+
		"internal_auth:\n  peer_secret: %q\n", EditionECE, testPeerSecret)
}

// Only KC-Core runs the SCACI listener, so only KC-Core refuses an ECE
// configuration whose listener cannot tell the organizations' Application
// Centers apart; KC-Identity and KC-Gateway load the same configuration.
func TestLoad_ECE_SCACIListenerRulesApplyToCoreOnly(t *testing.T) {
	path := writeTestConfig(t, eceServiceConfig())
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), ErrECEStrictOrgResolutionRequired) {
		t.Errorf("KC-Core: expected %q, got: %v", ErrECEStrictOrgResolutionRequired, err)
	}
	for name, load := range map[string]func(string) (*Config, error){"KC-Identity": LoadIdentity, "KC-Gateway": LoadGateway} {
		if _, err := load(path); err != nil {
			t.Errorf("%s runs no SCACI listener and must load, got: %v", name, err)
		}
	}
}

func TestValidate_ECE_AllowsOrgEnforcement(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionECE
	cfg.General.OrgEnforcementEnabled = flagEnabled

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("ECE should allow org enforcement, got: %v", err)
	}
}

func TestSCACIListener_ECEOrgIsolation(t *testing.T) {
	cases := []struct {
		name          string
		scaciEnabled  bool
		strictOrg     bool
		certMapping   bool
		wantErrSubstr string
	}{
		{name: "scaci enabled without either flag", scaciEnabled: flagEnabled, wantErrSubstr: ErrECEStrictOrgResolutionRequired},
		{name: "scaci enabled with cert mapping only", scaciEnabled: flagEnabled, certMapping: flagEnabled, wantErrSubstr: ErrECEStrictOrgResolutionRequired},
		{name: "scaci enabled with strict resolution only", scaciEnabled: flagEnabled, strictOrg: flagEnabled, wantErrSubstr: ErrStrictOrgResolutionRequiresCertTenantMapping},
		{name: "scaci enabled with both flags", scaciEnabled: flagEnabled, strictOrg: flagEnabled, certMapping: flagEnabled},
		{name: "scaci disabled without either flag", scaciEnabled: flagDisabled},
		{name: "scaci disabled with both flags", scaciEnabled: flagDisabled, strictOrg: flagEnabled, certMapping: flagEnabled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := minValidConfig()
			cfg.General.Edition = EditionECE
			cfg.Protocol.SCACIEnabled = tc.scaciEnabled
			cfg.Protocol.StrictOrgResolution = tc.strictOrg
			cfg.Protocol.SCACICertTenantMapping = tc.certMapping

			err := cfg.validateSCACIListener()
			if tc.wantErrSubstr == "" {
				if err != nil {
					t.Fatalf("expected valid ECE config, got: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Fatalf("expected %q, got: %v", tc.wantErrSubstr, err)
			}
		})
	}
}

func TestValidate_CE_SCACIEnabledWithoutStrictFlags(t *testing.T) {
	cfg := minValidConfig()
	cfg.General.Edition = EditionCommunity
	cfg.Protocol.SCACIEnabled = flagEnabled

	if err := cfg.Validate(); err != nil {
		t.Fatalf("CE with SCACI keeps its default-tenant isolation, got: %v", err)
	}
}
