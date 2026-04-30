package config

import (
	"errors"
	"fmt"
)

// ValidateServiceCenterConfig enforces BSSCI §1 compliance.
// Validates that the Service Center has a fixed network location as required by the MIOTY specification.
//
// Validation rules:
//   - Host: Non-empty string (allows DNS names like bssci.mioty.local and IP addresses)
//   - Port: Range [1, 65535]
//   - TLS: Must be enabled (fail-fast on disabled, BSSCI requires mutual TLS)
//
// Returns error if any validation fails. Server must Fatal on validation failure
// to enforce BSSCI §1 "fixed network location" requirement.
func ValidateServiceCenterConfig(cfg *ProtocolConfig) error {
	if cfg.BSCIHost == "" {
		return errors.New("protocol.bsci_host required for BSSCI §1 compliance (non-empty string)")
	}

	if cfg.BSCIPort < 1 || cfg.BSCIPort > 65535 {
		return fmt.Errorf("protocol.bsci_port must be in range [1,65535], got %d", cfg.BSCIPort)
	}

	if !cfg.BSCITLS.Enabled {
		return errors.New("protocol.bsci_tls.enabled must be true (BSSCI requires mutual TLS)")
	}

	return nil
}

// GetServiceCenterURL returns the canonical BSSCI URL per BSSCI §1.
// Format: tls://host:port
//
// This URL is provided to base stations during certificate generation
// and must match the actual network location where the Service Center listens.
//
// Priority:
//  1. BSCIExternalURL if configured (client-facing URL, e.g., tls://bssci.example.com:5000)
//  2. Falls back to tls://BSCIHost:BSCIPort (useful when BSCIHost is not 0.0.0.0)
//
// Supports both DNS names and IP addresses per BSSCI §1 requirements.
func GetServiceCenterURL(cfg *ProtocolConfig) string {
	if cfg.BSCIExternalURL != "" {
		return cfg.BSCIExternalURL
	}
	return fmt.Sprintf("tls://%s:%d", cfg.BSCIHost, cfg.BSCIPort)
}

// ValidateSCACIConfig enforces SCACI §1 compliance.
// Validates fixed network location and mutual TLS requirements.
//
// Validation rules:
//   - SCACIEnabled: If false, skip validation (SCACI disabled)
//   - Host: Non-empty string when enabled (SCACI §1 fixed network location)
//   - Port: Range [1, 65535]
//   - TLS: Must be enabled (SCACI requires mutual TLS)
//   - TLS files: cert_file, key_file, ca_file all required
//
// Returns error if validation fails. Server must Fatal on error
// to enforce SCACI §1 "fixed network location" requirement.
func ValidateSCACIConfig(cfg *ProtocolConfig) error {
	if !cfg.SCACIEnabled {
		return nil // SCACI disabled, skip validation
	}

	if cfg.SCACIHost == "" {
		return errors.New("protocol.scaci_host required when SCACI enabled (SCACI §1 fixed network location)")
	}

	if cfg.SCACIPort < 1 || cfg.SCACIPort > 65535 {
		return fmt.Errorf("protocol.scaci_port must be in range [1,65535], got %d", cfg.SCACIPort)
	}

	if !cfg.SCACITLS.Enabled {
		return errors.New("protocol.scaci_tls.enabled must be true (SCACI requires mutual TLS)")
	}

	// Validate TLS files are specified when TLS enabled
	if cfg.SCACITLS.CertFile == "" {
		return errors.New("protocol.scaci_tls.cert_file required when SCACI TLS enabled")
	}
	if cfg.SCACITLS.KeyFile == "" {
		return errors.New("protocol.scaci_tls.key_file required when SCACI TLS enabled")
	}
	if cfg.SCACITLS.CAFile == "" {
		return errors.New("protocol.scaci_tls.ca_file required for mutual TLS")
	}

	return nil
}
