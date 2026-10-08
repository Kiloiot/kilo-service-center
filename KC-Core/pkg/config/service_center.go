package config

import (
	"fmt"
	"net"
	"slices"
	"strings"
	"time"
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
		return errBSSCIHostRequired
	}

	if cfg.BSCIPort < 1 || cfg.BSCIPort > MaxPortNumber {
		return fmt.Errorf(errFmtBSSCIPortOutOfRange, cfg.BSCIPort)
	}

	if cfg.ManagementPort < 1 || cfg.ManagementPort > MaxPortNumber {
		return fmt.Errorf(errFmtManagementPortOutOfRange, cfg.ManagementPort)
	}

	if !cfg.BSCITLS.Enabled {
		return errBSSCITLSMustBeEnabled
	}

	if cfg.AckTimeout <= 0 {
		return fmt.Errorf(errFmtAckTimeoutNotPositive, cfg.AckTimeout)
	}

	if cfg.ConnectionEstablishmentTimeout <= 0 {
		return fmt.Errorf(errFmtConnEstablishTimeoutNotPositive, cfg.ConnectionEstablishmentTimeout)
	}

	if cfg.SocketWriteTimeout <= 0 {
		return fmt.Errorf(errFmtSocketWriteTimeoutNotPositive, cfg.SocketWriteTimeout)
	}

	if cfg.StatusRequestInterval <= 0 {
		return fmt.Errorf(errFmtStatusRequestIntervalNotPositive, cfg.StatusRequestInterval)
	}

	if cfg.StatusRequestInitialDelay < 0 {
		return fmt.Errorf(errFmtStatusRequestInitialDelayNegative, cfg.StatusRequestInitialDelay)
	}

	if cfg.DLRXQueryTimeout <= 0 {
		return fmt.Errorf(errFmtDLRXQueryTimeoutNotPositive, cfg.DLRXQueryTimeout)
	}

	if cfg.DLRXCleanupInterval <= 0 {
		return fmt.Errorf(errFmtDLRXCleanupIntervalNotPositive, cfg.DLRXCleanupInterval)
	}

	if cfg.DuplicateWindow <= 0 {
		return fmt.Errorf(errFmtDuplicateWindowNotPositive, cfg.DuplicateWindow)
	}

	if cfg.BSCICertificatePollInterval <= 0 {
		return fmt.Errorf(errFmtCertPollIntervalNotPositive, cfg.BSCICertificatePollInterval)
	}

	if cfg.Delivery.PollInterval <= 0 || cfg.Delivery.BatchSize <= 0 || cfg.Delivery.RetryBackoff <= 0 ||
		cfg.Delivery.MaxBackoff < cfg.Delivery.RetryBackoff {
		return fmt.Errorf(errFmtDeliveryConfigInvalid, cfg.Delivery.PollInterval, cfg.Delivery.BatchSize, cfg.Delivery.RetryBackoff, cfg.Delivery.MaxBackoff)
	}

	if cfg.Delivery.ReceptionWindow < 0 || cfg.Delivery.ReceptionWindow >= time.Duration(cfg.DuplicateWindow)*time.Second {
		return fmt.Errorf(errFmtDeliveryReceptionWindowInvalid, cfg.Delivery.ReceptionWindow, cfg.DuplicateWindow)
	}

	if cfg.DownlinkExpiry.Lifetime <= 0 || cfg.DownlinkExpiry.SweepInterval <= 0 || cfg.DownlinkExpiry.BatchSize <= 0 {
		return fmt.Errorf(errFmtDownlinkExpiryConfigInvalid, cfg.DownlinkExpiry.Lifetime, cfg.DownlinkExpiry.SweepInterval, cfg.DownlinkExpiry.BatchSize)
	}

	if len(cfg.DownlinkExpiry.RevokeNotHeldCodes) == 0 || slices.ContainsFunc(cfg.DownlinkExpiry.RevokeNotHeldCodes, func(code int) bool { return code <= 0 }) {
		return fmt.Errorf(errFmtRevokeNotHeldCodesInvalid, cfg.DownlinkExpiry.RevokeNotHeldCodes)
	}

	if cfg.SCACIResumeMaxPendingOperations <= 0 {
		return fmt.Errorf(errFmtSCACIResumeLimitNotPositive, cfg.SCACIResumeMaxPendingOperations)
	}

	if cfg.Roaming.CacheEnabled && (cfg.Roaming.CacheTTL <= 0 || cfg.Roaming.CacheMaxSize <= 0) {
		return fmt.Errorf(errFmtRoamingCacheConfigInvalid, cfg.Roaming.CacheTTL, cfg.Roaming.CacheMaxSize)
	}

	return nil
}

// GetServiceCenterURL returns the BSSCI URL base stations connect to (BSSCI §1):
// BSCIExternalURL when configured, else tls://BSCIHost:BSCIPort. It is empty
// when that address is a wildcard or loopback one, which no base station can
// reach; protocol.bsci_external_url fixes it.
func GetServiceCenterURL(cfg *ProtocolConfig) string {
	if cfg.BSCIExternalURL != "" {
		if !isStationReachable(ExternalBSSCIHost(cfg)) {
			return ""
		}
		return cfg.BSCIExternalURL
	}
	if !isStationReachable(cfg.BSCIHost) {
		return ""
	}
	return fmt.Sprintf(URLSchemeTLSFmt, cfg.BSCIHost, cfg.BSCIPort)
}

// StoredServiceCenterURL is how a base station record stores a service center
// URL: nil, stored as NULL, when GetServiceCenterURL knows none.
func StoredServiceCenterURL(url string) *string {
	if url == "" {
		return nil
	}
	return &url
}

// isStationReachable reports whether a base station on another machine can
// reach host.
func isStationReachable(host string) bool {
	return !IsWildcardHost(host) && !isLoopbackHost(host)
}

// ExternalBSSCIHost is the host of the configured external BSSCI URL, or empty
// when none is configured or it names the wildcard address.
func ExternalBSSCIHost(cfg *ProtocolConfig) string {
	raw := cfg.BSCIExternalURL
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(strings.TrimPrefix(raw, urlSchemePrefixTLS), urlSchemePrefixTCP)
	host := raw
	if h, _, err := net.SplitHostPort(raw); err == nil && h != "" {
		host = h
	}
	if IsWildcardHost(host) {
		return ""
	}
	return host
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
		return errSCACIHostRequired
	}

	if cfg.SCACIPort < 1 || cfg.SCACIPort > MaxPortNumber {
		return fmt.Errorf(errFmtSCACIPortOutOfRange, cfg.SCACIPort)
	}

	if !cfg.SCACITLS.Enabled {
		return errSCACITLSMustBeEnabled
	}

	// Validate TLS files are specified when TLS enabled
	if cfg.SCACITLS.CertFile == "" {
		return errSCACICertFileRequired
	}
	if cfg.SCACITLS.KeyFile == "" {
		return errSCACIKeyFileRequired
	}
	if cfg.SCACITLS.CAFile == "" {
		return errSCACICAFileRequired
	}

	return nil
}
