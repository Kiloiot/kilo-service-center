package config

import "errors"

// Sentinel errors returned by this package; callers match them with errors.Is.
var (
	errSCACICAFileRequired   = errors.New("protocol.scaci_tls.ca_file required for mutual TLS")
	errSCACIKeyFileRequired  = errors.New("protocol.scaci_tls.key_file required when SCACI TLS enabled")
	errSCACICertFileRequired = errors.New("protocol.scaci_tls.cert_file required when SCACI TLS enabled")
	errSCACITLSMustBeEnabled = errors.New("protocol.scaci_tls.enabled must be true (SCACI requires mutual TLS)")
	errSCACIHostRequired     = errors.New("protocol.scaci_host required when SCACI enabled (SCACI §1 fixed network location)")
	errBSSCITLSMustBeEnabled = errors.New("protocol.bsci_tls.enabled must be true (BSSCI requires mutual TLS)")
	errBSSCIHostRequired     = errors.New("protocol.bsci_host required for BSSCI §1 compliance (non-empty string)")
)

// Error format strings shared by this package's failure paths; verbs are filled at the point of failure.
const (
	errFmtSCACIPortOutOfRange               = "protocol.scaci_port must be in range [1,65535], got %d"
	errFmtSCACIResumeLimitNotPositive       = "protocol.scaci_resume_max_pending_operations must be positive, got %d"
	errFmtCertPollIntervalNotPositive       = "protocol.bsci_certificate_poll_interval must be a positive duration, got %s"
	errFmtDeliveryConfigInvalid             = "protocol.delivery poll_interval, batch_size and retry_backoff must be positive and max_backoff at least retry_backoff, got %v, %d, %v, %v"
	errFmtDeliveryReceptionWindowInvalid    = "protocol.delivery.reception_window must be at least 0 and shorter than protocol.duplicate_window, got %v and %ds"
	errFmtDuplicateWindowNotPositive        = "protocol.duplicate_window must be positive seconds, got %d"
	errFmtDownlinkExpiryConfigInvalid       = "protocol.downlink_expiry lifetime, sweep_interval and batch_size must be positive, got %v, %v, %d"
	errFmtRoamingCacheConfigInvalid         = "protocol.roaming cache_ttl and cache_max_size must be positive when cache_enabled, got %v, %d"
	errFmtDLRXCleanupIntervalNotPositive    = "protocol.dlrx_cleanup_interval must be positive seconds, got %d"
	errFmtDLRXQueryTimeoutNotPositive       = "protocol.dlrx_query_timeout must be positive seconds, got %d"
	errFmtStatusRequestInitialDelayNegative = "protocol.status_request_initial_delay must not be negative, got %d"
	errFmtStatusRequestIntervalNotPositive  = "protocol.status_request_interval must be positive seconds, got %d"
	errFmtConnEstablishTimeoutNotPositive   = "protocol.connection_establishment_timeout must be positive milliseconds, got %d"
	errFmtSocketWriteTimeoutNotPositive     = "protocol.socket_write_timeout must be positive milliseconds, got %d"
	errFmtAckTimeoutNotPositive             = "protocol.ack_timeout must be positive milliseconds, got %d"
	errFmtBSSCIPortOutOfRange               = "protocol.bsci_port must be in range [1,65535], got %d"
	errFmtManagementPortOutOfRange          = "protocol.management_port must be in range [1,65535], got %d"
	errFmtUnsupportedTLSVersion             = "unsupported TLS version: %s (supported: 1.2, 1.3, or blank for default)"
)
