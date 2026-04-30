package config

import (
	"strings"
	"testing"
)

// =============================================================================
// ValidateSCACIConfig Tests (SCACI §1 compliance)
// =============================================================================

func TestValidateSCACIConfig_Disabled_NoValidation(t *testing.T) {
	// When SCACI is disabled, no validation should occur
	cfg := &ProtocolConfig{
		SCACIEnabled: false,
		// All other fields intentionally invalid/empty
		SCACIHost: "",
		SCACIPort: 0,
		SCACITLS:  TLSConfig{Enabled: false},
	}

	err := ValidateSCACIConfig(cfg)
	if err != nil {
		t.Errorf("ValidateSCACIConfig() should skip validation when disabled, got error: %v", err)
	}
}

func TestValidateSCACIConfig_EmptyHost(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "",
		SCACIPort:    5001,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail with empty host")
	}
	if !strings.Contains(err.Error(), "scaci_host") {
		t.Errorf("Error should mention 'scaci_host', got: %v", err)
	}
	if !strings.Contains(err.Error(), "SCACI §1") {
		t.Errorf("Error should reference 'SCACI §1', got: %v", err)
	}
}

func TestValidateSCACIConfig_InvalidPort_Zero(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    0,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail with port 0")
	}
	if !strings.Contains(err.Error(), "scaci_port") {
		t.Errorf("Error should mention 'scaci_port', got: %v", err)
	}
}

func TestValidateSCACIConfig_InvalidPort_Negative(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    -1,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail with negative port")
	}
	if !strings.Contains(err.Error(), "scaci_port") {
		t.Errorf("Error should mention 'scaci_port', got: %v", err)
	}
}

func TestValidateSCACIConfig_InvalidPort_TooHigh(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    65536,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail with port > 65535")
	}
	if !strings.Contains(err.Error(), "scaci_port") {
		t.Errorf("Error should mention 'scaci_port', got: %v", err)
	}
	if !strings.Contains(err.Error(), "65536") {
		t.Errorf("Error should include actual port value, got: %v", err)
	}
}

func TestValidateSCACIConfig_TLSDisabled(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    5001,
		SCACITLS: TLSConfig{
			Enabled: false,
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail when TLS is disabled")
	}
	if !strings.Contains(err.Error(), "scaci_tls.enabled") {
		t.Errorf("Error should mention 'scaci_tls.enabled', got: %v", err)
	}
	if !strings.Contains(err.Error(), "mutual TLS") {
		t.Errorf("Error should mention mutual TLS requirement, got: %v", err)
	}
}

func TestValidateSCACIConfig_MissingCertFile(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    5001,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail with missing cert_file")
	}
	if !strings.Contains(err.Error(), "cert_file") {
		t.Errorf("Error should mention 'cert_file', got: %v", err)
	}
}

func TestValidateSCACIConfig_MissingKeyFile(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    5001,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail with missing key_file")
	}
	if !strings.Contains(err.Error(), "key_file") {
		t.Errorf("Error should mention 'key_file', got: %v", err)
	}
}

func TestValidateSCACIConfig_MissingCAFile(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    5001,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err == nil {
		t.Error("ValidateSCACIConfig() should fail with missing ca_file")
	}
	if !strings.Contains(err.Error(), "ca_file") {
		t.Errorf("Error should mention 'ca_file', got: %v", err)
	}
	if !strings.Contains(err.Error(), "mutual TLS") {
		t.Errorf("Error should mention mutual TLS requirement, got: %v", err)
	}
}

func TestValidateSCACIConfig_Valid(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "scaci.mioty.local",
		SCACIPort:    5001,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err != nil {
		t.Errorf("ValidateSCACIConfig() should pass with valid config, got error: %v", err)
	}
}

func TestValidateSCACIConfig_Valid_IPAddress(t *testing.T) {
	// SCACI §1 allows both DNS names and IP addresses
	cfg := &ProtocolConfig{
		SCACIEnabled: true,
		SCACIHost:    "192.168.1.100",
		SCACIPort:    5001,
		SCACITLS: TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
			CAFile:   "/path/to/ca.pem",
		},
	}

	err := ValidateSCACIConfig(cfg)
	if err != nil {
		t.Errorf("ValidateSCACIConfig() should accept IP address as host, got error: %v", err)
	}
}

func TestValidateSCACIConfig_Valid_BoundaryPorts(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"minimum valid port", 1},
		{"maximum valid port", 65535},
		{"common SCACI port", 5001},
		{"alternative port", 8443},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProtocolConfig{
				SCACIEnabled: true,
				SCACIHost:    "scaci.mioty.local",
				SCACIPort:    tt.port,
				SCACITLS: TLSConfig{
					Enabled:  true,
					CertFile: "/path/to/cert.pem",
					KeyFile:  "/path/to/key.pem",
					CAFile:   "/path/to/ca.pem",
				},
			}

			err := ValidateSCACIConfig(cfg)
			if err != nil {
				t.Errorf("ValidateSCACIConfig() should accept port %d, got error: %v", tt.port, err)
			}
		})
	}
}

// =============================================================================
// ValidateServiceCenterConfig Tests (BSSCI §1 compliance)
// =============================================================================

func TestValidateServiceCenterConfig_EmptyHost(t *testing.T) {
	cfg := &ProtocolConfig{
		BSCIHost: "",
		BSCIPort: 5000,
		BSCITLS:  TLSConfig{Enabled: true},
	}

	err := ValidateServiceCenterConfig(cfg)
	if err == nil {
		t.Error("ValidateServiceCenterConfig() should fail with empty host")
	}
	if !strings.Contains(err.Error(), "bsci_host") {
		t.Errorf("Error should mention 'bsci_host', got: %v", err)
	}
}

func TestValidateServiceCenterConfig_InvalidPort(t *testing.T) {
	cfg := &ProtocolConfig{
		BSCIHost: "bssci.mioty.local",
		BSCIPort: 0,
		BSCITLS:  TLSConfig{Enabled: true},
	}

	err := ValidateServiceCenterConfig(cfg)
	if err == nil {
		t.Error("ValidateServiceCenterConfig() should fail with port 0")
	}
	if !strings.Contains(err.Error(), "bsci_port") {
		t.Errorf("Error should mention 'bsci_port', got: %v", err)
	}
}

func TestValidateServiceCenterConfig_TLSDisabled(t *testing.T) {
	cfg := &ProtocolConfig{
		BSCIHost: "bssci.mioty.local",
		BSCIPort: 5000,
		BSCITLS:  TLSConfig{Enabled: false},
	}

	err := ValidateServiceCenterConfig(cfg)
	if err == nil {
		t.Error("ValidateServiceCenterConfig() should fail when TLS is disabled")
	}
	if !strings.Contains(err.Error(), "mutual TLS") {
		t.Errorf("Error should mention mutual TLS requirement, got: %v", err)
	}
}

func TestValidateServiceCenterConfig_Valid(t *testing.T) {
	cfg := &ProtocolConfig{
		BSCIHost: "bssci.mioty.local",
		BSCIPort: 5000,
		BSCITLS:  TLSConfig{Enabled: true},
	}

	err := ValidateServiceCenterConfig(cfg)
	if err != nil {
		t.Errorf("ValidateServiceCenterConfig() should pass with valid config, got error: %v", err)
	}
}

// =============================================================================
// GetServiceCenterURL Tests
// =============================================================================

func TestGetServiceCenterURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "DNS name",
			host:     "bssci.mioty.local",
			port:     5000,
			expected: "tls://bssci.mioty.local:5000",
		},
		{
			name:     "IP address",
			host:     "10.0.0.1",
			port:     5000,
			expected: "tls://10.0.0.1:5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProtocolConfig{
				BSCIHost: tt.host,
				BSCIPort: tt.port,
			}

			result := GetServiceCenterURL(cfg)
			if result != tt.expected {
				t.Errorf("GetServiceCenterURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}
