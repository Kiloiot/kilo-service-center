// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"github.com/kilocenter/KC-Core/pkg/bssci"
)

// Error catalog tokens for SCACI protocol operations
// These tokens enable centralized error messaging and localization support
const (
	// Session and connection errors
	errNoActiveSession  = "scaci.error.no_active_session"
	ErrNoActiveSession  = errNoActiveSession // Exported for service layer
	errOpIdOutOfOrder   = "scaci.error.op_id_out_of_order"
	ErrOpIdOutOfOrder   = errOpIdOutOfOrder // Exported for service layer
	errOpIDSignMismatch = "scaci.error.op_id_sign_mismatch"

	// Protocol/framing errors
	errInvalidMessageFormat      = "scaci.error.invalid_message_format"
	errMissingCommandField       = "scaci.error.missing_command_field"
	errMissingOpIdField          = "scaci.error.missing_op_id_field"
	errConnectRequired           = "scaci.error.connect_required"
	errConnectWaitingCmp         = "scaci.error.connect_waiting_cmp"
	errUnsupportedCommand        = "scaci.error.unsupported_command"
	errProtocolViolationULCmp    = "scaci.error.protocol_violation_ul_cmp"
	errProtocolViolationULRsp    = "scaci.error.protocol_violation_ul_rsp"
	errProtocolViolationDLResCmp = "scaci.error.protocol_violation_dl_res_cmp"

	// Validation errors - General
	errInvalidPayload           = "scaci.error.invalid_payload"
	errInvalidRegisterPayload   = "scaci.error.invalid_register_payload"
	ErrInvalidRegisterPayload   = errInvalidRegisterPayload // Exported for service layer (SCACI §3.6.1)
	errInvalidDeregisterPayload = "scaci.error.invalid_deregister_payload"
	errInvalidULDataTxPayload   = "scaci.error.invalid_ul_data_tx_payload"
	errInvalidDLDataQuePayload  = "scaci.error.invalid_dl_data_que_payload"
	errInvalidDLDataRevPayload  = "scaci.error.invalid_dl_data_rev_payload"
	errMalformedPayload         = "scaci.error.malformed_payload"
	errMissingEpEui             = "scaci.error.missing_ep_eui"
	ErrMissingEpEui             = errMissingEpEui // Exported for service layer
	errEpEuiZero                = "scaci.error.ep_eui_zero"
	errQueIDZero                = "scaci.error.que_id_zero"
	errUserDataEmpty            = "scaci.error.user_data_empty"
	errDatabaseError            = "scaci.error.database_error"
	ErrDatabaseError            = errDatabaseError // Exported for service layer

	// Connect operation errors
	errInvalidConnectFormat    = "scaci.error.invalid_connect_format"
	errConnectOpIdMustBeZero   = "scaci.error.connect_op_id_must_be_zero"
	errMissingVersion          = "scaci.error.missing_version"
	ErrMissingVersion          = errMissingVersion // Exported for service layer
	errMissingAcEui            = "scaci.error.missing_ac_eui"
	ErrMissingAcEui            = errMissingAcEui // Exported for service layer
	errSnAcUUIDZero            = "scaci.error.sn_ac_uuid_zero"
	ErrSnAcUUIDZero            = errSnAcUUIDZero // Exported for service layer
	errSnScOpIdRequired        = "scaci.error.sn_sc_op_id_required"
	ErrSnScOpIdRequired        = errSnScOpIdRequired // Exported for service layer
	errSnAcOpIdRequired        = "scaci.error.sn_ac_op_id_required"
	ErrSnAcOpIdRequired        = errSnAcOpIdRequired // Exported for service layer
	errInvalidVersionFormat    = "scaci.error.invalid_version_format"
	ErrInvalidVersionFormat    = errInvalidVersionFormat // Exported for service layer
	errMajorVersionUnsupported = "scaci.error.major_version_unsupported"
	ErrMajorVersionUnsupported = errMajorVersionUnsupported // Exported for service layer
	errMinorVersionUnsupported = "scaci.error.minor_version_unsupported"
	ErrMinorVersionUnsupported = errMinorVersionUnsupported // Exported for service layer

	// Version mismatch on session resume per SCACI §§2.1-2.3
	// Resume attempts must use the same version as the original connect
	errVersionMismatchOnResume = "scaci.error.version_mismatch_on_resume"
	ErrVersionMismatchOnResume = errVersionMismatchOnResume // Exported for service layer (SCACI §§2.1-2.3)

	errConCmpOpIDMustBeZero = "scaci.error.con_cmp_op_id_must_be_zero"

	// opId=0 reservation error (§3.3.2)
	errOpIDZeroReserved = "scaci.error.op_id_zero_reserved"
	ErrOpIDZeroReserved = errOpIDZeroReserved // Exported for service layer (§3.3.2)

	// Internal error (generic fallback)
	errInternalError = "scaci.error.internal_error"
	ErrInternalError = errInternalError // Exported for service layer

	// TLS certificate / tenant mapping errors
	errNilCertificate        = "scaci.error.nil_certificate"
	ErrNilCertificate        = errNilCertificate // Exported for service layer
	errCertTenantIDRequired  = "scaci.error.cert_tenant_id_required"
	ErrCertTenantIDRequired  = errCertTenantIDRequired // Exported for service layer
	errCertNotYetValid       = "scaci.error.cert_not_yet_valid"
	ErrCertNotYetValid       = errCertNotYetValid // Exported for service layer
	errCertExpired           = "scaci.error.cert_expired"
	ErrCertExpired           = errCertExpired // Exported for service layer
	errCertMissingClientAuth = "scaci.error.cert_missing_client_auth"
	ErrCertMissingClientAuth = errCertMissingClientAuth // Exported for service layer
	errCertInvalidSubject    = "scaci.error.cert_invalid_subject"
	ErrCertInvalidSubject    = errCertInvalidSubject // Exported for service layer

	// Organization resolution from certificate
	errCertificateTenantResolutionFailed = "scaci.error.cert_tenant_resolution_failed"
	ErrCertificateTenantResolutionFailed = errCertificateTenantResolutionFailed // Exported for service layer

	// Organization header enforcement
	errOrgHeaderRequired = "scaci.error.org_header_required"
	ErrOrgHeaderRequired = errOrgHeaderRequired // Exported for service layer

	// Register operation errors
	errInvalidNwkKeyLength   = "scaci.error.invalid_nwk_key_length"
	ErrInvalidNwkKeyLength   = errInvalidNwkKeyLength // Exported for service layer
	errInvalidNwkSnKeyLength = "scaci.error.invalid_nwk_sn_key_length"
	errFailedCreateEndpoint  = "scaci.error.failed_create_endpoint"
	ErrFailedCreateEndpoint  = errFailedCreateEndpoint // Exported for service layer
	errFailedUpdateEndpoint  = "scaci.error.failed_update_endpoint"
	ErrFailedUpdateEndpoint  = errFailedUpdateEndpoint // Exported for service layer

	// Deregister operation errors
	errEndpointNotFound             = "scaci.error.endpoint_not_found"
	ErrEndpointNotFound             = errEndpointNotFound // Exported for service layer
	errSendDeregisterResponseFailed = "scaci.error.send_deregister_response_failed"
	ErrSendDeregisterResponseFailed = errSendDeregisterResponseFailed // Exported for service layer (§3.7.2)
	errRevokeDownlinksFailed        = "scaci.error.revoke_downlinks_failed"
	ErrRevokeDownlinksFailed        = errRevokeDownlinksFailed // Exported for service layer (§3.7.3)

	// UL Data Transmit operation errors
	errBaseStationNotFound    = "scaci.error.base_station_not_found"
	ErrBaseStationNotFound    = errBaseStationNotFound // Exported for service layer
	errFailedVerifyBS         = "scaci.error.failed_verify_base_station"
	errULTransmitNotSupported = "scaci.error.ul_transmit_not_supported"
	ErrULTransmitNotSupported = errULTransmitNotSupported // Exported for service layer
	errFailedRecordOperation  = "scaci.error.failed_record_operation"
	ErrFailedRecordOperation  = errFailedRecordOperation // Exported for service layer

	// Sublayer prefix guard errors (§4)
	errUnsupportedSublayerPrefix = "scaci.error.unsupported_sublayer_prefix"
	ErrUnsupportedSublayerPrefix = errUnsupportedSublayerPrefix // Exported for service layer (§4)
)

// ErrBaseStationTenantMismatch indicates the requested base station belongs to a different tenant.
// This error is imported from the BSSCI package and maps to POSIX_ENOENT when surfaced via SCACI.
var ErrBaseStationTenantMismatch = bssci.ErrBaseStationTenantMismatch

const (
	// DL Data Queue operation errors (SCACI §3.10)
	errDLPayloadTooLarge = "scaci.error.dl_payload_too_large"
	// ErrDLPayloadTooLarge is exported for gRPC error mapping.
	ErrDLPayloadTooLarge        = errDLPayloadTooLarge
	errCntDependMismatch        = "scaci.error.cnt_depend_mismatch"
	errCntDependPacketCntOmit   = "scaci.error.cnt_depend_packet_cnt_omit"
	errNonCntDependMultiPayload = "scaci.error.non_cnt_depend_multi_payload"
	errQueIDExists              = "scaci.error.que_id_exists"
	errQueueIDOutOfRange        = "scaci.error.queue_id_out_of_range"
	errFailedPersistDownlink    = "scaci.error.failed_persist_downlink"

	// ErrCntDependMismatch indicates a mismatch between cntDepend flag and packet count array (SCACI §3.10.2).
	ErrCntDependMismatch = errCntDependMismatch
	// ErrCntDependPacketCntOmit indicates packetCnt was provided when cntDepend=false (SCACI §3.10.2).
	ErrCntDependPacketCntOmit = errCntDependPacketCntOmit
	// ErrNonCntDependMultiPayload indicates multiple payloads with counter-independent mode (SCACI §3.10.3).
	ErrNonCntDependMultiPayload = errNonCntDependMultiPayload
	// ErrQueIDExists indicates the requested queue ID already exists (SCACI §3.10.4).
	ErrQueIDExists = errQueIDExists
	// ErrQueueIDOutOfRange indicates the queue ID is outside valid range (SCACI §3.10.4).
	ErrQueueIDOutOfRange = errQueueIDOutOfRange
	// ErrFailedPersistDownlink indicates a database persistence failure for downlink (SCACI §3.10.5).
	ErrFailedPersistDownlink = errFailedPersistDownlink

	// DL Data Revoke operation errors
	errMissingPacketCnt = "scaci.error.missing_packet_cnt"
	errDownlinkNotFound = "scaci.error.downlink_not_found"

	// ErrDownlinkNotFound indicates the requested downlink queue entry was not found
	ErrDownlinkNotFound     = errDownlinkNotFound // Exported for service layer
	errSchedulerUnavailable = "scaci.error.scheduler_unavailable"

	// ErrSchedulerUnavailable indicates the downlink scheduler is temporarily unavailable
	ErrSchedulerUnavailable   = errSchedulerUnavailable // Exported for service layer
	errBaseStationUnavailable = "scaci.error.base_station_unavailable"

	// ErrBaseStationUnavailable indicates the requested base station is not available for downlink
	ErrBaseStationUnavailable = errBaseStationUnavailable // Exported for service layer

	// ============================================================================
	// Assembly Validation Errors (SCACI §§2.5, 3.12.1, 3.13.1)
	// ============================================================================

	// DLDataResult assembly validation (§3.12.1)
	errDLDataResultMissingResult     = "scaci.error.dlresult_missing_result"
	errDLDataResultInvalidResultEnum = "scaci.error.dlresult_invalid_result_enum"
	errDLDataResultOpIDNotNegative   = "scaci.error.dlresult_opid_not_negative" // SC-originated must be negative
	errDLDataResultSentMissingBsEui  = "scaci.error.dlresult_sent_missing_bs_eui"
	errDLDataResultSentInvalidBsEui  = "scaci.error.dlresult_sent_invalid_bs_eui" // bsEui zero (invalid EUI64)
	errDLDataResultSentMissingTxTime = "scaci.error.dlresult_sent_missing_tx_time"
	errDLDataResultSentMissingPktCnt = "scaci.error.dlresult_sent_missing_packet_cnt"

	// EPStatus assembly validation (§3.13.1)
	errEPStatusMissingStatus     = "scaci.error.epstatus_missing_status"
	errEPStatusInvalidStatusEnum = "scaci.error.epstatus_invalid_status_enum"
	errEPStatusMissingAttachCnt  = "scaci.error.epstatus_missing_attach_cnt"
	errEPStatusMissingNonce      = "scaci.error.epstatus_missing_nonce"
	errEPStatusMissingSign       = "scaci.error.epstatus_missing_sign"

	// ConnectResponse assembly validation (§3.3.2)
	errConnectResponseMissingScEui    = "scaci.error.conrsp_missing_sc_eui"
	errConnectResponseMissingSnScUUID = "scaci.error.conrsp_missing_sn_sc_uuid"

	// StatusResponse assembly validation (§3.5.2)
	errStatusResponseMissingMessage = "scaci.error.status_missing_message"
	errStatusResponseMissingTime    = "scaci.error.status_missing_time"

	// ULData assembly validation (§3.8.1)
	errULDataMissingBaseStations     = "scaci.error.uldata_missing_base_stations"
	errULDataMissingEpEui            = "scaci.error.uldata_missing_ep_eui"
	errULDataInvalidEpEui            = "scaci.error.uldata_invalid_ep_eui"
	errULDataInvalidBsEui            = "scaci.error.uldata_invalid_bs_eui"
	errULDataMissingRxTime           = "scaci.error.uldata_missing_rx_time"
	errULDataInvalidRxTime           = "scaci.error.uldata_invalid_rx_time"
	errULDataMissingSnr              = "scaci.error.uldata_missing_snr"
	errULDataInvalidSnr              = "scaci.error.uldata_invalid_snr"
	errULDataMissingRssi             = "scaci.error.uldata_missing_rssi"
	errULDataInvalidRssi             = "scaci.error.uldata_invalid_rssi"
	errULDataInvalidEqSnr            = "scaci.error.uldata_invalid_eq_snr"
	errULDataInvalidRxDuration       = "scaci.error.uldata_invalid_rx_duration"
	errULDataSubpacketLengthMismatch = "scaci.error.uldata_subpacket_length_mismatch"
	// ErrULDataInvalidEqSnr is exported for use by validators in assembly validation per §3.8.1
	ErrULDataInvalidEqSnr = errULDataInvalidEqSnr

	// Error message assembly validation (§3.14.1)
	errErrorMissingCode    = "scaci.error.error_missing_code"
	errErrorMissingMessage = "scaci.error.error_missing_message"
)

// ErrorDefinition maps error tokens to messages, spec sections, severity, and POSIX codes
// This enables error tracking and evidence generation for SCACI v1.0.0
type ErrorDefinition struct {
	Token       string // Error token (e.g., "scaci.error.no_active_session")
	Message     string // Human-readable message
	SpecSection string // SCACI spec reference (e.g., "§3.3", "§3.8.2")
	Severity    string // "error", "warning", "protocol_violation"
	POSIXCode   int    // POSIX error code for wire protocol (0 = use caller-provided code)
}

// errorDefinitions maps error tokens to full metadata
// Future enhancement: support multiple languages via locale-specific definitions
var errorDefinitions = map[string]ErrorDefinition{
	// Session and connection errors
	errNoActiveSession: {
		Token:       "scaci.error.no_active_session",
		Message:     "No active session",
		SpecSection: "§3.3",
		Severity:    "error",
	},

	// Protocol/framing errors
	errInvalidMessageFormat: {
		Token:       "scaci.error.invalid_message_format",
		Message:     "Invalid message format",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errMissingCommandField: {
		Token:       "scaci.error.missing_command_field",
		Message:     "Missing command field",
		SpecSection: "§3.2",
		Severity:    "error",
	},
	errMissingOpIdField: {
		Token:       "scaci.error.missing_op_id_field",
		Message:     "Missing opId field",
		SpecSection: "§3.2",
		Severity:    "error",
	},
	errConnectRequired: {
		Token:       "scaci.error.connect_required",
		Message:     "Connect required as first message",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errConnectWaitingCmp: {
		Token:       "scaci.error.connect_waiting_cmp",
		Message:     "Connect must complete (waiting for conCmp)",
		SpecSection: "§3.3.3",
		Severity:    "error",
	},
	errUnsupportedCommand: {
		Token:       "scaci.error.unsupported_command",
		Message:     "Unsupported command",
		SpecSection: "§3.2",
		Severity:    "error",
	},
	errProtocolViolationULCmp: {
		Token:       "scaci.error.protocol_violation_ul_cmp",
		Message:     "Protocol violation: Service Center issues ulDataCmp",
		SpecSection: "§3.8.3",
		Severity:    "protocol_violation",
	},
	errProtocolViolationULRsp: {
		Token:       "scaci.error.protocol_violation_ul_rsp",
		Message:     "Protocol violation: Service Center issues ulDataTxRsp",
		SpecSection: "§3.9.2",
		Severity:    "protocol_violation",
	},
	errProtocolViolationDLResCmp: {
		Token:       "scaci.error.protocol_violation_dl_res_cmp",
		Message:     "Protocol violation: Application Center sends txDataResCmp (SC should send it)",
		SpecSection: "§3.12.3",
		Severity:    "protocol_violation",
	},
	errOpIdOutOfOrder: {
		Token:       "scaci.error.op_id_out_of_order",
		Message:     "Operation ID out of sequence",
		SpecSection: "§3.2",
		Severity:    "error",
	},
	errOpIDSignMismatch: {
		Token:       "scaci.error.op_id_sign_mismatch",
		Message:     "Operation ID sign does not match command initiator",
		SpecSection: "§3.2",
		Severity:    "error",
		POSIXCode:   POSIX_EINVAL,
	},

	// Validation errors - General
	errInvalidPayload: {
		Token:       "scaci.error.invalid_payload",
		Message:     "Invalid message payload",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errInvalidRegisterPayload: {
		Token:       "scaci.error.invalid_register_payload",
		Message:     "Invalid register payload",
		SpecSection: "§3.6.1",
		Severity:    "error",
	},
	errInvalidDeregisterPayload: {
		Token:       "scaci.error.invalid_deregister_payload",
		Message:     "Invalid deregister payload",
		SpecSection: "§3.7.1",
		Severity:    "error",
	},
	errInvalidULDataTxPayload: {
		Token:       "scaci.error.invalid_ul_data_tx_payload",
		Message:     "Invalid UL data transmit payload",
		SpecSection: "§3.9.1",
		Severity:    "error",
	},
	errInvalidDLDataQuePayload: {
		Token:       "scaci.error.invalid_dl_data_que_payload",
		Message:     "Malformed request payload",
		SpecSection: "§3.10.1",
		Severity:    "error",
	},
	errInvalidDLDataRevPayload: {
		Token:       "scaci.error.invalid_dl_data_rev_payload",
		Message:     "Invalid DL data revoke format",
		SpecSection: "§3.11.1",
		Severity:    "error",
	},
	errMalformedPayload: {
		Token:       "scaci.error.malformed_payload",
		Message:     "Malformed request payload",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errMissingEpEui: {
		Token:       "scaci.error.missing_ep_eui",
		Message:     "Missing endpoint EUI",
		SpecSection: "§3.6.1",
		Severity:    "error",
		POSIXCode:   POSIX_EINVAL,
	},
	errEpEuiZero: {
		Token:       "scaci.error.ep_eui_zero",
		Message:     "Endpoint EUI must be non-zero",
		SpecSection: "§3.6.1",
		Severity:    "error",
	},
	errQueIDZero: {
		Token:       "scaci.error.que_id_zero",
		Message:     "Queue ID must be non-zero",
		SpecSection: "§3.10.1",
		Severity:    "error",
	},
	errUserDataEmpty: {
		Token:       "scaci.error.user_data_empty",
		Message:     "User data must not be empty",
		SpecSection: "§3.10.1",
		Severity:    "error",
	},
	errDatabaseError: {
		Token:       "scaci.error.database_error",
		Message:     "Database error",
		SpecSection: "§3",
		Severity:    "error",
		POSIXCode:   POSIX_EIO,
	},

	// Connect operation errors
	errInvalidConnectFormat: {
		Token:       "scaci.error.invalid_connect_format",
		Message:     "Invalid Connect message format",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errConnectOpIdMustBeZero: {
		Token:       "scaci.error.connect_op_id_must_be_zero",
		Message:     "Connect must use opId 0",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errMissingVersion: {
		Token:       "scaci.error.missing_version",
		Message:     "Missing version",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errMissingAcEui: {
		Token:       "scaci.error.missing_ac_eui",
		Message:     "Missing acEui",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errSnAcUUIDZero: {
		Token:       "scaci.error.sn_ac_uuid_zero",
		Message:     "snAcUuid cannot be zero",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errSnScOpIdRequired: {
		Token:       "scaci.error.sn_sc_op_id_required",
		Message:     "snScOpId required when snAcOpId is present",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errSnAcOpIdRequired: {
		Token:       "scaci.error.sn_ac_op_id_required",
		Message:     "snAcOpId required when snScOpId is present",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errInvalidVersionFormat: {
		Token:       "scaci.error.invalid_version_format",
		Message:     "Invalid version format",
		SpecSection: "§2",
		Severity:    "error",
	},
	errMajorVersionUnsupported: {
		Token:       "scaci.error.major_version_unsupported",
		Message:     "Major version not supported",
		SpecSection: "§2.1",
		Severity:    "error",
		POSIXCode:   POSIX_ENOTSUP, // 95 - operation not supported (version incompatible)
	},
	errMinorVersionUnsupported: {
		Token:       "scaci.error.minor_version_unsupported",
		Message:     "Minor version not supported",
		SpecSection: "§2.2",
		Severity:    "error",
		POSIXCode:   POSIX_ENOTSUP, // 95 - operation not supported (version incompatible)
	},
	errVersionMismatchOnResume: {
		Token:       "scaci.error.version_mismatch_on_resume",
		Message:     "Version mismatch on session resume: must match originally negotiated version",
		SpecSection: "§2.1-2.3",
		Severity:    "error",
		POSIXCode:   POSIX_ENOTSUP, // 95 - operation not supported (version incompatible)
	},
	errConCmpOpIDMustBeZero: {
		Token:       "scaci.error.con_cmp_op_id_must_be_zero",
		Message:     "conCmp must use opId=0 per SCACI §3.3",
		SpecSection: "§3.3.3",
		Severity:    "error",
	},
	errOpIDZeroReserved: {
		Token:       "scaci.error.op_id_zero_reserved",
		Message:     "opId=0 is reserved for connect handshake",
		SpecSection: "§3.3.2",
		Severity:    "error",
		POSIXCode:   POSIX_EINVAL, // 22 - invalid argument (opId reserved)
	},
	errInternalError: {
		Token:       "scaci.error.internal_error",
		Message:     "Internal server error",
		SpecSection: "§3",
		Severity:    "error",
		POSIXCode:   POSIX_EINVAL, // 22 - generic internal error
	},

	// TLS certificate / tenant mapping errors
	errNilCertificate: {
		Token:       "scaci.error.nil_certificate",
		Message:     "No certificate provided",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errCertTenantIDRequired: {
		Token:       "scaci.error.cert_tenant_id_required",
		Message:     "Certificate CN/SAN must contain tenant ID (e.g., 'tenant-123')",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errCertNotYetValid: {
		Token:       "scaci.error.cert_not_yet_valid",
		Message:     "Certificate not yet valid (current time before NotBefore)",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errCertExpired: {
		Token:       "scaci.error.cert_expired",
		Message:     "Certificate expired (current time after NotAfter)",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errCertMissingClientAuth: {
		Token:       "scaci.error.cert_missing_client_auth",
		Message:     "Certificate missing ClientAuth extended key usage (required for mutual TLS)",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errCertInvalidSubject: {
		Token:       "scaci.error.cert_invalid_subject",
		Message:     "Certificate has invalid subject (missing CN and Organization)",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errCertificateTenantResolutionFailed: {
		Token:       "scaci.error.cert_tenant_resolution_failed",
		Message:     "Failed to resolve tenant from certificate organization claim",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errOrgHeaderRequired: {
		Token:       "scaci.error.org_header_required",
		Message:     "Organization header required in strict mode (X-Organization-ID cannot be nil)",
		SpecSection: "§3.1",
		Severity:    "error",
	},

	// Register operation errors
	errInvalidNwkKeyLength: {
		Token:       "scaci.error.invalid_nwk_key_length",
		Message:     "Network key must be exactly 16 bytes",
		SpecSection: "§3.6.1",
		Severity:    "error",
		POSIXCode:   POSIX_EINVAL,
	},
	errInvalidNwkSnKeyLength: {
		Token:       "scaci.error.invalid_nwk_sn_key_length",
		Message:     "Network session key must be exactly 16 bytes",
		SpecSection: "§3.6.1",
		Severity:    "error",
	},
	errFailedCreateEndpoint: {
		Token:       "scaci.error.failed_create_endpoint",
		Message:     "Failed to create endpoint",
		SpecSection: "§3.6.2",
		Severity:    "error",
		POSIXCode:   POSIX_EIO,
	},
	errFailedUpdateEndpoint: {
		Token:       "scaci.error.failed_update_endpoint",
		Message:     "Failed to update endpoint",
		SpecSection: "§3.6.2",
		Severity:    "error",
		POSIXCode:   POSIX_EIO,
	},

	// Deregister operation errors
	errEndpointNotFound: {
		Token:       "scaci.error.endpoint_not_found",
		Message:     "Endpoint not found",
		SpecSection: "§3.7.1",
		Severity:    "error",
	},
	errSendDeregisterResponseFailed: {
		Token:       errSendDeregisterResponseFailed,
		Message:     "Failed to send deregister response",
		SpecSection: "§3.7.2",
		Severity:    "error",
		POSIXCode:   POSIX_EIO,
	},
	errRevokeDownlinksFailed: {
		Token:       errRevokeDownlinksFailed,
		Message:     "Failed to revoke pending downlinks during deregister cleanup",
		SpecSection: "§3.7.3",
		Severity:    "warning",
		POSIXCode:   POSIX_EIO,
	},

	// UL Data Transmit operation errors
	errBaseStationNotFound: {
		Token:       "scaci.error.base_station_not_found",
		Message:     "Base station not found",
		SpecSection: "§3.9.1",
		Severity:    "error",
		POSIXCode:   POSIX_ENOENT,
	},
	errFailedVerifyBS: {
		Token:       "scaci.error.failed_verify_base_station",
		Message:     "Failed to verify base station",
		SpecSection: "§3.9.1",
		Severity:    "error",
		POSIXCode:   POSIX_EIO,
	},
	errULTransmitNotSupported: {
		Token:       "scaci.error.ul_transmit_not_supported",
		Message:     "UL data transmit not supported",
		SpecSection: "§3.9",
		Severity:    "error",
		POSIXCode:   POSIX_ENOTSUP,
	},
	errFailedRecordOperation: {
		Token:       "scaci.error.failed_record_operation",
		Message:     "Failed to record operation",
		SpecSection: "§3.9.2",
		Severity:    "error",
		POSIXCode:   POSIX_EIO,
	},

	// DL Data Queue operation errors
	errDLPayloadTooLarge: {
		Token:       "scaci.error.dl_payload_too_large",
		Message:     "Downlink payload exceeds 200-byte maximum (MIOTY radio protocol §4.3.2)",
		SpecSection: "§3.10.1",
		Severity:    "error",
		POSIXCode:   POSIX_EINVAL,
	},
	errCntDependMismatch: {
		Token:       "scaci.error.cnt_depend_mismatch",
		Message:     "Counter-dependent mismatch: packet count length must equal user data length",
		SpecSection: "§3.10.1",
		Severity:    "error",
	},
	errCntDependPacketCntOmit: {
		Token:       "scaci.error.cnt_depend_packet_cnt_omit",
		Message:     "Counter-dependent=false requires packet counter omitted",
		SpecSection: "§3.10.1",
		Severity:    "error",
	},
	errNonCntDependMultiPayload: {
		Token:       "scaci.error.non_cnt_depend_multi_payload",
		Message:     "Non-counter-dependent downlink must have single userData entry",
		SpecSection: "§3.10.1",
		Severity:    "error",
		POSIXCode:   POSIX_EINVAL,
	},
	errQueIDExists: {
		Token:       "scaci.error.que_id_exists",
		Message:     "Queue ID already exists",
		SpecSection: "§3.10.1",
		Severity:    "error",
	},
	errQueueIDOutOfRange: {
		Token:       "scaci.error.queue_id_out_of_range",
		Message:     "Queue ID exceeds maximum value (2^63-1)",
		SpecSection: "§3.10.1",
		Severity:    "error",
	},
	errFailedPersistDownlink: {
		Token:       "scaci.error.failed_persist_downlink",
		Message:     "Failed to persist downlink",
		SpecSection: "§3.10.2",
		Severity:    "error",
	},

	// DL Data Revoke operation errors
	errMissingPacketCnt: {
		Token:       "scaci.error.missing_packet_cnt",
		Message:     "Missing packet counter",
		SpecSection: "§3.11.1",
		Severity:    "error",
	},
	errDownlinkNotFound: {
		Token:       "scaci.error.downlink_not_found",
		Message:     "Downlink not found for packet counter",
		SpecSection: "§3.11.1",
		Severity:    "error",
	},
	errSchedulerUnavailable: {
		Token:       "scaci.error.scheduler_unavailable",
		Message:     "Downlink scheduler not available",
		SpecSection: "§3.11",
		Severity:    "error",
	},
	errBaseStationUnavailable: {
		Token:       "scaci.error.base_station_unavailable",
		Message:     "Base station temporarily unavailable",
		SpecSection: "§3.11",
		Severity:    "error",
	},

	// Sublayer prefix guard (§4)
	errUnsupportedSublayerPrefix: {
		Token:       errUnsupportedSublayerPrefix,
		Message:     "Unsupported sublayer prefix in command",
		SpecSection: "§4",
		Severity:    "error",
		POSIXCode:   POSIX_ENOTSUP, // 95 - operation not supported (sublayer unavailable)
	},

	// ============================================================================
	// Assembly Validation Errors (SCACI §§2.5, 3.12.1, 3.13.1)
	// ============================================================================

	// DLDataResult assembly validation (§3.12.1)
	// Note: queId uses existing errQueIDZero; no duplicate token needed
	errDLDataResultMissingResult: {
		Token:       errDLDataResultMissingResult,
		Message:     "DLDataResult missing result",
		SpecSection: "§3.12.1",
		Severity:    "error",
	},
	errDLDataResultInvalidResultEnum: {
		Token:       errDLDataResultInvalidResultEnum,
		Message:     "DLDataResult result must be sent/expired/invalid/revoked",
		SpecSection: "§3.12.1",
		Severity:    "error",
	},
	errDLDataResultOpIDNotNegative: {
		Token:       errDLDataResultOpIDNotNegative,
		Message:     "SC-originated DLDataResult opId must be negative",
		SpecSection: "§3.12.1",
		Severity:    "protocol_violation",
		POSIXCode:   POSIX_EPROTO,
	},
	errDLDataResultSentMissingBsEui: {
		Token:       errDLDataResultSentMissingBsEui,
		Message:     "DLDataResult with result=sent requires bsEui",
		SpecSection: "§3.12.1",
		Severity:    "error",
	},
	errDLDataResultSentInvalidBsEui: {
		Token:       errDLDataResultSentInvalidBsEui,
		Message:     "DLDataResult bsEui must be valid EUI64 (non-zero)",
		SpecSection: "§3.12.1",
		Severity:    "error",
	},
	errDLDataResultSentMissingTxTime: {
		Token:       errDLDataResultSentMissingTxTime,
		Message:     "DLDataResult with result=sent requires txTime",
		SpecSection: "§3.12.1",
		Severity:    "error",
	},
	errDLDataResultSentMissingPktCnt: {
		Token:       errDLDataResultSentMissingPktCnt,
		Message:     "DLDataResult with result=sent requires packetCnt",
		SpecSection: "§3.12.1",
		Severity:    "error",
	},

	// EPStatus assembly validation (§3.13.1)
	errEPStatusMissingStatus: {
		Token:       errEPStatusMissingStatus,
		Message:     "EPStatus missing epStatus",
		SpecSection: "§3.13.1",
		Severity:    "error",
	},
	errEPStatusInvalidStatusEnum: {
		Token:       errEPStatusInvalidStatusEnum,
		Message:     "EPStatus epStatus must be attached/detached",
		SpecSection: "§3.13.1",
		Severity:    "error",
	},
	// OTA field requirements per §3.13.1:
	// - attached: attachCnt, nonce, sign required
	// - detached: sign required
	errEPStatusMissingAttachCnt: {
		Token:       errEPStatusMissingAttachCnt,
		Message:     "EPStatus missing attachCnt (required for attached status)",
		SpecSection: "§3.13.1",
		Severity:    "error",
		POSIXCode:   POSIX_EPROTO,
	},
	errEPStatusMissingNonce: {
		Token:       errEPStatusMissingNonce,
		Message:     "EPStatus missing nonce (required for attached status)",
		SpecSection: "§3.13.1",
		Severity:    "error",
		POSIXCode:   POSIX_EPROTO,
	},
	errEPStatusMissingSign: {
		Token:       errEPStatusMissingSign,
		Message:     "EPStatus missing sign (required for OTA status)",
		SpecSection: "§3.13.1",
		Severity:    "error",
		POSIXCode:   POSIX_EPROTO,
	},

	// ConnectResponse assembly validation (§3.3.2)
	errConnectResponseMissingScEui: {
		Token:       errConnectResponseMissingScEui,
		Message:     "ConnectResponse missing scEui",
		SpecSection: "§3.3.2",
		Severity:    "error",
	},
	errConnectResponseMissingSnScUUID: {
		Token:       errConnectResponseMissingSnScUUID,
		Message:     "ConnectResponse missing snScUuid",
		SpecSection: "§3.3.2",
		Severity:    "error",
	},

	// StatusResponse assembly validation (§3.5.2)
	errStatusResponseMissingMessage: {
		Token:       errStatusResponseMissingMessage,
		Message:     "StatusResponse missing message",
		SpecSection: "§3.5.2",
		Severity:    "error",
	},
	errStatusResponseMissingTime: {
		Token:       errStatusResponseMissingTime,
		Message:     "StatusResponse missing time",
		SpecSection: "§3.5.2",
		Severity:    "error",
	},

	// ULData assembly validation (§3.8.1)
	errULDataMissingBaseStations: {
		Token:       errULDataMissingBaseStations,
		Message:     "ULData missing baseStations",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataMissingEpEui: {
		Token:       errULDataMissingEpEui,
		Message:     "ULData missing epEui",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataInvalidEpEui: {
		Token:       errULDataInvalidEpEui,
		Message:     "ULData epEui must be non-zero EUI64",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataInvalidBsEui: {
		Token:       errULDataInvalidBsEui,
		Message:     "ULData baseStation bsEui must be non-zero EUI64",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataMissingRxTime: {
		Token:       errULDataMissingRxTime,
		Message:     "ULData baseStation missing rxTime",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataInvalidRxTime: {
		Token:       errULDataInvalidRxTime,
		Message:     "ULData baseStation rxTime must be positive Unix ns",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataMissingSnr: {
		Token:       errULDataMissingSnr,
		Message:     "ULData baseStation missing snr",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataInvalidSnr: {
		Token:       errULDataInvalidSnr,
		Message:     "ULData baseStation snr contains invalid value (NaN/Inf)",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataMissingRssi: {
		Token:       errULDataMissingRssi,
		Message:     "ULData baseStation missing rssi",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataInvalidRssi: {
		Token:       errULDataInvalidRssi,
		Message:     "ULData baseStation rssi contains invalid value (NaN/Inf)",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataInvalidEqSnr: {
		Token:       errULDataInvalidEqSnr,
		Message:     "ULData baseStation eqSnr contains invalid value (NaN/Inf); 0 and nil are valid",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataInvalidRxDuration: {
		Token:       errULDataInvalidRxDuration,
		Message:     "ULData baseStation rxDuration must be non-negative if present",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errULDataSubpacketLengthMismatch: {
		Token:       errULDataSubpacketLengthMismatch,
		Message:     "ULData baseStation subpackets arrays must have equal length (snr, rssi, frequency)",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},

	// Error message assembly validation (§3.14.1)
	errErrorMissingCode: {
		Token:       errErrorMissingCode,
		Message:     "Error message missing code (code=0 is invalid for Error)",
		SpecSection: "§3.14.1",
		Severity:    "error",
	},
	errErrorMissingMessage: {
		Token:       errErrorMissingMessage,
		Message:     "Error message missing message field",
		SpecSection: "§3.14.1",
		Severity:    "error",
	},
}

// GetErrorDefinition returns the full error metadata for a token
// Use this for new code that needs spec traceability
func GetErrorDefinition(token string) ErrorDefinition {
	if def, ok := errorDefinitions[token]; ok {
		return def
	}
	// Return default definition for unknown tokens
	return ErrorDefinition{
		Token:       token,
		Message:     token,
		SpecSection: "unknown",
		Severity:    "error",
	}
}
