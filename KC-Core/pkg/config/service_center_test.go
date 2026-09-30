package config

import (
	"strings"
	"testing"
	"time"
)

// Shared fixture values for the configuration validation tests in this package.
const (
	flagEnabled  = true
	flagDisabled = false

	testSCACIHost = "scaci.mioty.local"
	testBSSCIHost = "bssci.mioty.local"
	testCertFile  = "/path/to/cert.pem"
	testKeyFile   = "/path/to/key.pem"
	testCAFile    = "/path/to/ca.pem"

	invalidPortZero    = 0
	invalidPortTooHigh = 65536
	invalidTimingValue = 0
)

// =============================================================================
// ValidateSCACIConfig Tests (SCACI §1 compliance)
// =============================================================================

func TestValidateSCACIConfig_Disabled_NoValidation(t *testing.T) {
	// When SCACI is disabled, no validation should occur
	cfg := &ProtocolConfig{
		SCACIEnabled: flagDisabled,
		// All other fields intentionally invalid/empty
		SCACIHost: "",
		SCACIPort: invalidPortZero,
		SCACITLS:  TLSConfig{Enabled: flagDisabled},
	}

	err := ValidateSCACIConfig(cfg)
	if err != nil {
		t.Errorf("ValidateSCACIConfig() should skip validation when disabled, got error: %v", err)
	}
}

func TestValidateSCACIConfig_EmptyHost(t *testing.T) {
	cfg := &ProtocolConfig{
		SCACIEnabled: flagEnabled,
		SCACIHost:    "",
		SCACIPort:    DefaultProtocolSCACIPort,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  testKeyFile,
			CAFile:   testCAFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    invalidPortZero,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  testKeyFile,
			CAFile:   testCAFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    -1,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  testKeyFile,
			CAFile:   testCAFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    invalidPortTooHigh,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  testKeyFile,
			CAFile:   testCAFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    DefaultProtocolSCACIPort,
		SCACITLS: TLSConfig{
			Enabled: flagDisabled,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    DefaultProtocolSCACIPort,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: "",
			KeyFile:  testKeyFile,
			CAFile:   testCAFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    DefaultProtocolSCACIPort,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  "",
			CAFile:   testCAFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    DefaultProtocolSCACIPort,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  testKeyFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    testSCACIHost,
		SCACIPort:    DefaultProtocolSCACIPort,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  testKeyFile,
			CAFile:   testCAFile,
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
		SCACIEnabled: flagEnabled,
		SCACIHost:    "192.168.1.100",
		SCACIPort:    DefaultProtocolSCACIPort,
		SCACITLS: TLSConfig{
			Enabled:  flagEnabled,
			CertFile: testCertFile,
			KeyFile:  testKeyFile,
			CAFile:   testCAFile,
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
				SCACIEnabled: flagEnabled,
				SCACIHost:    testSCACIHost,
				SCACIPort:    tt.port,
				SCACITLS: TLSConfig{
					Enabled:  flagEnabled,
					CertFile: testCertFile,
					KeyFile:  testKeyFile,
					CAFile:   testCAFile,
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
		BSCIPort: DefaultProtocolBSCIPort,
		BSCITLS:  TLSConfig{Enabled: flagEnabled},
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
		BSCIHost: testBSSCIHost,
		BSCIPort: invalidPortZero,
		BSCITLS:  TLSConfig{Enabled: flagEnabled},
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
		BSCIHost:       testBSSCIHost,
		BSCIPort:       DefaultProtocolBSCIPort,
		ManagementPort: DefaultProtocolManagementPort,
		BSCITLS:        TLSConfig{Enabled: flagDisabled},
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
		BSCIHost:                       testBSSCIHost,
		BSCIPort:                       DefaultProtocolBSCIPort,
		ManagementPort:                 DefaultProtocolManagementPort,
		BSCITLS:                        TLSConfig{Enabled: flagEnabled},
		AckTimeout:                     DefaultProtocolAckTimeout,
		ConnectionEstablishmentTimeout: DefaultProtocolConnectionEstablishmentTimeout,
		SocketWriteTimeout:             DefaultProtocolSocketWriteTimeout,
		StatusRequestInterval:          DefaultProtocolStatusRequestInterval,
		StatusRequestInitialDelay:      DefaultProtocolStatusRequestInitialDelay,
		DLRXQueryTimeout:               DefaultProtocolDLRXQueryTimeout,
		DLRXCleanupInterval:            DefaultProtocolDLRXCleanupInterval,
		DuplicateWindow:                DefaultProtocolDuplicateWindow,
		Delivery: DeliveryConfig{
			PollInterval: DefaultProtocolDeliveryPollInterval,
			BatchSize:    DefaultProtocolDeliveryBatchSize,
			MaxBackoff:   DefaultProtocolDeliveryMaxBackoff,
			RetryBackoff: DefaultProtocolDeliveryRetryBackoff,
		},
		DownlinkExpiry:                  defaultDownlinkExpiry(),
		BSCICertificatePollInterval:     DefaultProtocolCertificatePollInterval,
		SCACIResumeMaxPendingOperations: DefaultProtocolSCACIResumeMaxPendingOperations,
	}

	err := ValidateServiceCenterConfig(cfg)
	if err != nil {
		t.Errorf("ValidateServiceCenterConfig() should pass with valid config, got error: %v", err)
	}
}

func TestValidateServiceCenterConfig_TimingBounds(t *testing.T) {
	base := func() *ProtocolConfig {
		return &ProtocolConfig{
			BSCIHost:                       testBSSCIHost,
			BSCIPort:                       DefaultProtocolBSCIPort,
			ManagementPort:                 DefaultProtocolManagementPort,
			BSCITLS:                        TLSConfig{Enabled: flagEnabled},
			AckTimeout:                     DefaultProtocolAckTimeout,
			ConnectionEstablishmentTimeout: DefaultProtocolConnectionEstablishmentTimeout,
			SocketWriteTimeout:             DefaultProtocolSocketWriteTimeout,
			StatusRequestInterval:          DefaultProtocolStatusRequestInterval,
			StatusRequestInitialDelay:      DefaultProtocolStatusRequestInitialDelay,
			DLRXQueryTimeout:               DefaultProtocolDLRXQueryTimeout,
			DLRXCleanupInterval:            DefaultProtocolDLRXCleanupInterval,
			DuplicateWindow:                DefaultProtocolDuplicateWindow,
			Delivery: DeliveryConfig{
				PollInterval: DefaultProtocolDeliveryPollInterval,
				BatchSize:    DefaultProtocolDeliveryBatchSize,
				MaxBackoff:   DefaultProtocolDeliveryMaxBackoff,
				RetryBackoff: DefaultProtocolDeliveryRetryBackoff,
			},
			DownlinkExpiry:                  defaultDownlinkExpiry(),
			BSCICertificatePollInterval:     DefaultProtocolCertificatePollInterval,
			SCACIResumeMaxPendingOperations: DefaultProtocolSCACIResumeMaxPendingOperations,
		}
	}

	if err := ValidateServiceCenterConfig(base()); err != nil {
		t.Fatalf("the unmodified base config must validate, got: %v", err)
	}

	zeroAck := base()
	zeroAck.AckTimeout = invalidTimingValue
	if err := ValidateServiceCenterConfig(zeroAck); err == nil {
		t.Error("zero ack_timeout must fail validation")
	}

	zeroEstablish := base()
	zeroEstablish.ConnectionEstablishmentTimeout = invalidTimingValue
	if err := ValidateServiceCenterConfig(zeroEstablish); err == nil {
		t.Error("zero connection_establishment_timeout must fail validation")
	}

	zeroWrite := base()
	zeroWrite.SocketWriteTimeout = invalidTimingValue
	if err := ValidateServiceCenterConfig(zeroWrite); err == nil {
		t.Error("zero socket_write_timeout must fail validation")
	}

	zeroStatusInterval := base()
	zeroStatusInterval.StatusRequestInterval = invalidTimingValue
	if err := ValidateServiceCenterConfig(zeroStatusInterval); err == nil {
		t.Error("zero status_request_interval must fail validation")
	}

	zeroDLRXTimeout := base()
	zeroDLRXTimeout.DLRXQueryTimeout = invalidTimingValue
	if err := ValidateServiceCenterConfig(zeroDLRXTimeout); err == nil {
		t.Error("zero dlrx_query_timeout must fail validation")
	}

	zeroDLRXCleanup := base()
	zeroDLRXCleanup.DLRXCleanupInterval = invalidTimingValue
	if err := ValidateServiceCenterConfig(zeroDLRXCleanup); err == nil {
		t.Error("zero dlrx_cleanup_interval must fail validation")
	}

	negativeWindow := base()
	negativeWindow.DuplicateWindow = -1
	if err := ValidateServiceCenterConfig(negativeWindow); err == nil {
		t.Error("negative duplicate_window must fail validation")
	}

	zeroPoll := base()
	zeroPoll.BSCICertificatePollInterval = invalidTimingValue
	if err := ValidateServiceCenterConfig(zeroPoll); err == nil {
		t.Error("zero bsci_certificate_poll_interval must fail validation")
	}

	capBelowBase := base()
	capBelowBase.Delivery.MaxBackoff = capBelowBase.Delivery.RetryBackoff / 2
	if err := ValidateServiceCenterConfig(capBelowBase); err == nil {
		t.Error("a delivery max_backoff below retry_backoff must fail validation")
	}

	for name, window := range map[string]time.Duration{
		"negative":                   -time.Millisecond,
		"the whole duplicate window": time.Duration(DefaultProtocolDuplicateWindow) * time.Second,
	} {
		cfg := base()
		cfg.Delivery.ReceptionWindow = window
		if err := ValidateServiceCenterConfig(cfg); err == nil {
			t.Errorf("a %s delivery reception_window must fail validation", name)
		}
	}

	cacheWithoutTTL := base()
	cacheWithoutTTL.Roaming = RoamingConfig{CacheEnabled: flagEnabled, CacheMaxSize: DefaultProtocolRoamingCacheMaxSize}
	if err := ValidateServiceCenterConfig(cacheWithoutTTL); err == nil {
		t.Error("an enabled roaming cache without a TTL must fail validation")
	}

	for name, unset := range map[string]func(*DownlinkExpiryConfig){
		"lifetime":       func(c *DownlinkExpiryConfig) { c.Lifetime = invalidTimingValue },
		"sweep_interval": func(c *DownlinkExpiryConfig) { c.SweepInterval = invalidTimingValue },
		"batch_size":     func(c *DownlinkExpiryConfig) { c.BatchSize = invalidTimingValue },
	} {
		cfg := base()
		unset(&cfg.DownlinkExpiry)
		if err := ValidateServiceCenterConfig(cfg); err == nil {
			t.Errorf("zero downlink_expiry.%s must fail validation", name)
		}
	}

	unboundedResume := base()
	unboundedResume.SCACIResumeMaxPendingOperations = invalidTimingValue
	if err := ValidateServiceCenterConfig(unboundedResume); err == nil {
		t.Error("a zero scaci_resume_max_pending_operations must fail validation")
	}
}

func defaultDownlinkExpiry() DownlinkExpiryConfig {
	return DownlinkExpiryConfig{
		Lifetime:      DefaultProtocolDownlinkLifetime,
		SweepInterval: DefaultProtocolDownlinkExpirySweepInterval,
		BatchSize:     DefaultProtocolDownlinkExpiryBatchSize,
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
			host:     testBSSCIHost,
			port:     DefaultProtocolBSCIPort,
			expected: "tls://bssci.mioty.local:5000",
		},
		{
			name:     "IP address",
			host:     "10.0.0.1",
			port:     DefaultProtocolBSCIPort,
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
