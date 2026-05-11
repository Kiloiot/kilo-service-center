package bssci

// Error catalog tokens for BSSCI protocol operations
// These tokens enable centralized error messaging and spec traceability
const (
	// Session and connection errors
	errSessionNotFound          = "bssci.error.session_not_found"
	errHandshakeNotComplete     = "bssci.error.handshake_not_complete"
	errSessionNil               = "bssci.error.session_nil"
	errInvalidSessionType       = "bssci.error.invalid_session_type"
	errCannotSendPing           = "bssci.error.cannot_send_ping"
	errBaseStationNotRegistered = "bssci.error.base_station_not_registered"
	errResumeCounterMismatch    = "bssci.error.resume_counter_mismatch"
	errInvalidResumeCounters    = "bssci.error.invalid_resume_counters"

	// Protocol/framing errors
	errInvalidMessageFormat  = "bssci.error.invalid_message_format"
	errFailedToEncode        = "bssci.error.failed_to_encode"
	errFailedToMarshal       = "bssci.error.failed_to_marshal"
	errFailedToMarshalMeta   = "bssci.error.failed_to_marshal_metadata"
	errFailedToWrite         = "bssci.error.failed_to_write"
	errFailedToWriteHeader   = "bssci.error.failed_to_write_header"
	errFailedToWritePayload  = "bssci.error.failed_to_write_payload"
	errFailedToDecode        = "bssci.error.failed_to_decode"
	errFailedToDecodePayload = "bssci.error.failed_to_decode_payload"
	errPayloadTooLarge       = "bssci.error.payload_too_large"

	// Version negotiation errors (BSSCI §2.1-2.3)
	errInvalidVersionFormat    = "bssci.error.invalid_version_format"
	errInvalidMajorVersion     = "bssci.error.invalid_major_version"
	errInvalidMinorVersion     = "bssci.error.invalid_minor_version"
	errInvalidPatchVersion     = "bssci.error.invalid_patch_version"
	errUnsupportedMajorVersion = "bssci.error.unsupported_major_version"
	errUnsupportedMinorVersion = "bssci.error.unsupported_minor_version"

	// Message normalization errors (BSSCI §2.4)
	errNormalizationFailed   = "bssci.error.normalization_failed"
	errMandatoryFieldMissing = "bssci.error.mandatory_field_missing"
	errInvalidFieldType      = "bssci.error.invalid_field_type"
	errConditionalRuleFailed = "bssci.error.conditional_rule_failed"

	// Operation ID validation errors (BSSCI §3.2)
	errInvalidOperationID          = "bssci.error.invalid_operation_id"
	errOperationIDNotPositive      = "bssci.error.operation_id_not_positive"
	errOperationIDBackwards        = "bssci.error.operation_id_backwards"
	errOperationIDIncreasing       = "bssci.error.operation_id_increasing"
	errSCOperationIDMustBeNegative = "bssci.error.sc_operation_id_must_be_negative"
	errOperationIDValidation       = "bssci.error.operation_id_validation"
	errInvalidConnectOpId          = "bssci.error.invalid_connect_op_id"
	errConnectOpIDNotZero          = "bssci.error.connect_op_id_not_zero"
	errInvalidConnectCompleteOpId  = "bssci.error.invalid_connect_complete_op_id"

	// Connect operation errors (BSSCI §3.3)
	errMissingBsEui            = "bssci.error.missing_bs_eui"
	errInvalidBsEui            = "bssci.error.invalid_bs_eui"
	errInvalidUUIDByteType     = "bssci.error.invalid_uuid_byte_type"
	errInvalidUUIDLength       = "bssci.error.invalid_uuid_length"
	errUnsupportedUUIDType     = "bssci.error.unsupported_uuid_type"
	errUUIDDataNil             = "bssci.error.uuid_data_nil"
	errNoConnectedBaseStations = "bssci.error.no_connected_base_stations"
	errFailedToLoadCA          = "bssci.error.failed_to_load_ca"
	errFailedToParseCA         = "bssci.error.failed_to_parse_ca"
	errFailedToLoadTLS         = "bssci.error.failed_to_load_tls"
	errFailedToStartTLS        = "bssci.error.failed_to_start_tls"
	errCommandBeforeHandshake  = "bssci.error.command_before_handshake"
	errMissingProtocolVersion  = "bssci.error.missing_protocol_version"
	errMissingBidiFlag         = "bssci.error.missing_bidi_flag"

	// Attach operation errors (BSSCI §3.6)
	errMissingEpEui                = "bssci.error.missing_ep_eui"
	errAttachOperationFailed       = "bssci.error.attach_operation_failed"
	errAttachPropagateFailed       = "bssci.error.attach_propagate_failed"
	errPropagationBroadcastFailure = "bssci.error.propagation_broadcast_failure"
	errEndpointAttachFailed        = "bssci.error.endpoint_attach_failed"
	errInvalidEndpointEUIFormat    = "bssci.error.invalid_endpoint_eui_format"
	errBaseStationNotBidirectional = "bssci.error.base_station_not_bidirectional"
	errNoPendingAttachOperation    = "bssci.error.no_pending_attach_operation"
	errInvalidNwkSnKeyLength       = "bssci.error.invalid_nwk_sn_key_length"
	errBaseStationEUIOutOfRange    = "bssci.error.base_station_eui_out_of_range"

	// Detach operation errors (BSSCI §3.7)
	errDetachOperationFailed    = "bssci.error.detach_operation_failed"
	errDetachPropagateFailed    = "bssci.error.detach_propagate_failed"
	errEndpointDetachFailed     = "bssci.error.endpoint_detach_failed"
	errNoPendingDetachOperation = "bssci.error.no_pending_detach_operation"
	errDetachSignatureInvalid   = "bssci.error.detach_signature_invalid" // Unknown endpoint signature validation
	errEndpointNotFound         = "bssci.error.endpoint_not_found"       // Endpoint not found during detach validation
	errSignatureMismatch        = "bssci.error.signature_mismatch"

	// Downlink adapter errors (BSSCI §5.13)
	errDLQueueNilResult = "bssci.error.dl_queue_nil_result"

	// Downlink payload size (MIOTY radio protocol §4.3.2)
	errDLPayloadTooLarge = "bssci.error.dl_payload_too_large"

	// Downlink data queue errors (BSSCI §3.9)
	errInvalidQueueID                  = "bssci.error.invalid_queue_id"
	errQueueIDNotFound                 = "bssci.error.queue_id_not_found"
	errMissingQueID                    = "bssci.error.missing_que_id"
	errCannotResolveTenantForQueue     = "bssci.error.cannot_resolve_tenant_for_queue"
	errInvalidPayloadCount             = "bssci.error.invalid_payload_count"
	errCounterDependentPayloadMismatch = "bssci.error.counter_dependent_payload_mismatch"
	errNonCounterDependentMultiPayload = "bssci.error.non_counter_dependent_multi_payload"
	errMissingPacketCnt                = "bssci.error.missing_packet_cnt"
	errMissingResultField              = "bssci.error.missing_result_field"
	errMissingTxTimeOrPacketCnt        = "bssci.error.missing_txtime_or_packetcnt"
	errUnexpectedTxTimeOrPacketCnt     = "bssci.error.unexpected_txtime_or_packetcnt"
	errInvalidTenantIDFormat           = "bssci.error.invalid_tenant_id_format"
	errQueueIDOutOfRange               = "bssci.error.queue_id_out_of_range"
	errNegativeQueueIDInMetadata       = "bssci.error.negative_queue_id_in_metadata"
	errInvalidPacketCounter            = "bssci.error.invalid_packet_counter"

	// DL Data Revoke errors (BSSCI §3.10)
	errCannotSendDlDataRev    = "bssci.error.cannot_send_dl_data_rev"
	errFailedToSendDlDataRev  = "bssci.error.failed_to_send_dl_data_rev"
	errMissingEpEuiInMetadata = "bssci.error.missing_ep_eui_in_metadata"
	errMissingQueIDInMetadata = "bssci.error.missing_que_id_in_metadata"

	// DL RX Status errors (BSSCI §3.11)
	errFailedToPersistDLRXStatus = "bssci.error.failed_to_persist_dl_rx_status"
	errFailedToSendDlRxStatQry   = "bssci.error.failed_to_send_dl_rx_stat_qry"
	errInvalidResultValue        = "bssci.error.invalid_result_value"
	errMissingDlRxEpEui          = "bssci.error.missing_dl_rx_ep_eui"
	errMissingDlRxTime           = "bssci.error.missing_dl_rx_time"
	errMissingDlRxPacketCnt      = "bssci.error.missing_dl_rx_packet_cnt"
	errInvalidDlRxPacketCnt      = "bssci.error.invalid_dl_rx_packet_cnt"
	errMissingDlRxSnr            = "bssci.error.missing_dl_rx_snr"
	errInvalidDlRxSnr            = "bssci.error.invalid_dl_rx_snr"
	errMissingDlRxRssi           = "bssci.error.missing_dl_rx_rssi"
	errInvalidDlRxRssi           = "bssci.error.invalid_dl_rx_rssi"
	errUnexpectedDlRxStatus      = "bssci.error.unexpected_dl_rx_status"

	// Database and persistence errors
	errDatabaseError          = "bssci.error.database_error"
	errDatabaseNotAvailable   = "bssci.error.database_not_available"
	errDatabaseUpdateFailed   = "bssci.error.database_update_failed"
	errDeduplicationError     = "bssci.error.deduplication_error"
	errPacketCounterCollision = "bssci.error.packet_counter_collision"

	// Version and protocol compatibility errors
	errVersionIncompatible          = "bssci.error.version_incompatible"
	errSCOperationIDMustNotIncrease = "bssci.error.sc_operation_id_must_not_increase"

	// Propagation and operation failures
	errOperationFailed          = "bssci.error.operation_failed"
	errOperationNotFound        = "bssci.error.operation_not_found"
	errPropagateFailed          = "bssci.error.propagate_failed"
	errEndpointOperationFailed  = "bssci.error.endpoint_operation_failed"
	errFailedToSendError        = "bssci.error.failed_to_send_error"
	errBaseStationReportedError = "bssci.error.base_station_reported_error"
	errFailedToEnqueueDownlink  = "bssci.error.failed_to_enqueue_downlink"

	// Variable MAC (VM) operation errors
	errVMOperationSentByBS      = "bssci.error.vm_operation_sent_by_bs"
	errMissingMacType           = "bssci.error.missing_mac_type"
	errMissingRxTime            = "bssci.error.missing_rx_time"
	errFailedToSendVMActivate   = "bssci.error.failed_to_send_vm_activate"
	errFailedToSendVMDeactivate = "bssci.error.failed_to_send_vm_deactivate"
	errFailedToSendVMStatus     = "bssci.error.failed_to_send_vm_status"
	errFailedToSendVMDlData     = "bssci.error.failed_to_send_vm_dl_data"

	// Uplink transmit errors (BSSCI §3.8)
	errNwkSnKeyInvalidLength              = "bssci.error.nwk_sn_key_invalid_length"
	errBaseStationTenantMismatch          = "bssci.error.base_station_tenant_mismatch"
	errFailedToSendULDataTransmitComplete = "bssci.error.failed_to_send_ul_data_transmit_complete"

	// Session lookup errors (DL RX status query, roaming support)
	errSessionNotReady               = "bssci.error.session_not_ready"
	errSessionNotBidirectional       = "bssci.error.session_not_bidirectional"
	errFailedToEncryptKey            = "bssci.error.failed_to_encrypt_key"
	errFailedToDecryptKey            = "bssci.error.failed_to_decrypt_key"
	errMissingEncryptedKey           = "bssci.error.missing_encrypted_key"
	errFailedToDecodeUserData        = "bssci.error.failed_to_decode_user_data"
	errFailedToSendULTransmit        = "bssci.error.failed_to_send_ul_transmit"
	errMissingAttachCnt              = "bssci.error.missing_attach_cnt"
	errMissingNonce                  = "bssci.error.missing_nonce"
	errMissingSign                   = "bssci.error.missing_sign"
	errInvalidSignature              = "bssci.error.invalid_signature"
	errInvalidAttachCntRange         = "bssci.error.invalid_attach_cnt_range"
	errInvalidNonceElement           = "bssci.error.invalid_nonce_element"
	errInvalidSignElement            = "bssci.error.invalid_sign_element"
	errAttachCounterNotMonotonic     = "bssci.error.attach_counter_not_monotonic"
	errInvalidSnrValue               = "bssci.error.invalid_snr_value"
	errInvalidRssiValue              = "bssci.error.invalid_rssi_value"
	errInvalidEqSnrValue             = "bssci.error.invalid_eqsnr_value"
	errEndpointNotProvisioned        = "bssci.error.endpoint_not_provisioned"
	errRoamingNotAllowed             = "bssci.error.roaming_not_allowed"
	errResponseExpRequiresDlOpen     = "bssci.error.response_exp_requires_dl_open"
	errRxTimeExceedsMaximum          = "bssci.error.rx_time_exceeds_maximum"
	errInvalidRxTimeFormat           = "bssci.error.invalid_rx_time_format"
	errInvalidRxTimeValue            = "bssci.error.invalid_rx_time_value"
	errFailedToPersistULData         = "bssci.error.failed_to_persist_ul_data"
	errFailedToResolveEndpointTenant = "bssci.error.failed_to_resolve_endpoint_tenant"
	errUnsupportedSublayerPrefix     = "bssci.error.unsupported_sublayer_prefix"
	errUnsupportedCommand            = "bssci.error.unsupported_command"

	// Propagate operation errors (BSSCI §5.17 error handling)
	errAttachPropagateRejected = "bssci.error.attach_propagate_rejected"
	errDetachPropagateRejected = "bssci.error.detach_propagate_rejected"

	// Status operation errors
	errFailedToSendStatusRequest = "bssci.error.failed_to_send_status_request"
	errMissingStatusCode         = "bssci.error.missing_status_code"
	errMissingStatusMessage      = "bssci.error.missing_status_message"
	errMissingStatusTime         = "bssci.error.missing_status_time"

	// Management HTTP API errors (BSSCI §3.6 attach/§3.7 detach propagation)
	// Exported for use by pkg/management HTTP handlers
	ErrMgmtMethodNotAllowed        = "bssci.mgmt.method_not_allowed"
	ErrMgmtSessionIDRequired       = "bssci.mgmt.session_id_required"
	ErrMgmtInvalidRequestBody      = "bssci.mgmt.invalid_request_body"
	ErrMgmtInvalidNwkSnKeyEncoding = "bssci.mgmt.invalid_nwk_sn_key_encoding"
	ErrMgmtInvalidNwkSnKeyLength   = "bssci.mgmt.invalid_nwk_sn_key_length"
	ErrMgmtAttachPropagateFailed   = "bssci.mgmt.attach_propagate_failed"
	ErrMgmtDetachPropagateFailed   = "bssci.mgmt.detach_propagate_failed"
	ErrMgmtJSONEncodeFailed        = "bssci.mgmt.json_encode_failed"
	ErrMgmtFailedToBindAddress     = "bssci.mgmt.failed_to_bind_address"
	ErrMgmtServerFailed            = "bssci.mgmt.server_failed"
	ErrMgmtInvalidTenantContext    = "bssci.mgmt.invalid_tenant_context"
	ErrMgmtNoConnectedSessions     = "bssci.mgmt.no_connected_sessions"
	ErrMgmtSessionLookupFailed     = "bssci.mgmt.session_lookup_failed"

	// Type conversion and validation errors
	errInvalidFieldValue = "bssci.error.invalid_field_value"

	// Organization context enforcement
	errOrgContextMissing = "bssci.error.org_context_missing"

	// Float validation errors (BSSCI §5.15.1)
	errInvalidFloatNaN = "bssci.err.invalid_float_nan"
	errInvalidFloatInf = "bssci.err.invalid_float_inf"
	errFloatOutOfRange = "bssci.err.float_out_of_range"

	// Outbound message validation errors (BSSCI §2.5)
	errOutboundMissingCommand        = "bssci.error.outbound_missing_command"
	errOutboundMissingField          = "bssci.error.outbound_missing_field"
	errOutboundMissingMandatoryField = "bssci.error.outbound_missing_mandatory_field"
	errOutboundExtraField            = "bssci.error.outbound_extra_field"
	errOutboundUnknownCommand        = "bssci.error.outbound_unknown_command"
	errOutboundMarshalFailed         = "bssci.error.outbound_marshal_failed"
	errOutboundInvalidFieldType      = "bssci.error.outbound_invalid_field_type"

	// Unknown/generic errors
	errUnknownError = "bssci.error.unknown"
)

// Exported error constants for service layer implementations
// These reference the package-private constants above and are needed by
// internal/services/bssci implementations that use package-private tokens
const (
	// Version negotiation errors (used by SessionService)
	ErrVersionIncompatible     = errVersionIncompatible
	ErrUnsupportedMajorVersion = errUnsupportedMajorVersion
	ErrUnsupportedMinorVersion = errUnsupportedMinorVersion
	ErrInvalidVersionFormat    = errInvalidVersionFormat
	ErrInvalidMajorVersion     = errInvalidMajorVersion
	ErrInvalidMinorVersion     = errInvalidMinorVersion
	ErrInvalidPatchVersion     = errInvalidPatchVersion

	// Connection errors (used by ConnectionService)
	ErrBaseStationNotRegistered = errBaseStationNotRegistered

	// Operation errors (used by StatusService)
	ErrOperationNotFound = errOperationNotFound

	// Downlink errors (used by DownlinkService)
	ErrDLQueueNilResult            = errDLQueueNilResult
	ErrFailedToEnqueueDownlink     = errFailedToEnqueueDownlink
	ErrInvalidQueueID              = errInvalidQueueID              // BSSCI §5.13 revoke validation
	ErrCannotResolveTenantForQueue = errCannotResolveTenantForQueue // BSSCI §5.13 tenant routing
	ErrDatabaseUpdateFailed        = errDatabaseUpdateFailed        // BSSCI §5.13 persistence errors
	ErrInvalidPacketCounter        = errInvalidPacketCounter        // BSSCI §3.12 counter-dependent packet validation

	// DL RX status validation errors (BSSCI §3.11, additional hardening)
	ErrInvalidDlRxSnr  = errInvalidDlRxSnr  // dlRxSnr must be in uint32 range
	ErrInvalidDlRxRssi = errInvalidDlRxRssi // dlRxRssi must be in uint32 range

	// Propagation broadcast errors (used by propagation service)
	ErrPropagationBroadcastFailure = errPropagationBroadcastFailure

	// Propagate-rejection errors (used by handlePropagateResponseFailure)
	ErrAttachPropagateFailed = errAttachPropagateFailed
	ErrDetachPropagateFailed = errDetachPropagateFailed

	// Note: ErrBaseStationTenantMismatch is now defined in ul_transmit.go as a proper error sentinel
)

// ErrorDefinition maps error tokens to messages, spec sections, and severity
// This enables spec traceability and structured error reporting for BSSCI v1.0.0
type ErrorDefinition struct {
	Token       string // Error token (e.g., "bssci.error.session_not_found")
	Message     string // Human-readable message
	SpecSection string // BSSCI spec reference (e.g., "§3.3", "§3.6.1")
	Severity    string // "error", "warning", "protocol_violation"
}

// errorDefinitions maps error tokens to full metadata
var errorDefinitions = map[string]ErrorDefinition{
	// Session and connection errors
	errSessionNotFound: {
		Token:       "bssci.error.session_not_found",
		Message:     "Session not found",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errHandshakeNotComplete: {
		Token:       "bssci.error.handshake_not_complete",
		Message:     "Handshake not complete",
		SpecSection: "§3.3.3",
		Severity:    "protocol_violation",
	},
	errSessionNil: {
		Token:       "bssci.error.session_nil",
		Message:     "Invalid session: session is nil",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errInvalidSessionType: {
		Token:       "bssci.error.invalid_session_type",
		Message:     "Invalid session type or nil session",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errCannotSendPing: {
		Token:       "bssci.error.cannot_send_ping",
		Message:     "Cannot send ping: handshake not complete",
		SpecSection: "§3.4",
		Severity:    "protocol_violation",
	},
	errBaseStationNotRegistered: {
		Token:       "bssci.error.base_station_not_registered",
		Message:     "Base station not registered",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errResumeCounterMismatch: {
		Token:       "bssci.error.resume_counter_mismatch",
		Message:     "Resume rejected: operation counters do not match persisted session state",
		SpecSection: "§5.2",
		Severity:    "protocol_violation",
	},
	errInvalidResumeCounters: {
		Token:       "bssci.error.invalid_resume_counters",
		Message:     "Resume counters invalid: mutual presence or sign constraints violated",
		SpecSection: "§5.3.1",
		Severity:    "protocol_violation",
	},

	// Protocol/framing errors
	errInvalidMessageFormat: {
		Token:       "bssci.error.invalid_message_format",
		Message:     "Invalid message format",
		SpecSection: "§3.1",
		Severity:    "protocol_violation",
	},
	errFailedToEncode: {
		Token:       "bssci.error.failed_to_encode",
		Message:     "Failed to encode message",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errFailedToMarshal: {
		Token:       "bssci.error.failed_to_marshal",
		Message:     "Failed to marshal message",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errFailedToMarshalMeta: {
		Token:       "bssci.error.failed_to_marshal_metadata",
		Message:     "Failed to marshal metadata",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errFailedToWrite: {
		Token:       "bssci.error.failed_to_write",
		Message:     "Failed to write",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errFailedToWriteHeader: {
		Token:       "bssci.error.failed_to_write_header",
		Message:     "Failed to write header",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errFailedToWritePayload: {
		Token:       "bssci.error.failed_to_write_payload",
		Message:     "Failed to write payload",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errFailedToDecode: {
		Token:       "bssci.error.failed_to_decode",
		Message:     "Failed to decode",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errFailedToDecodePayload: {
		Token:       "bssci.error.failed_to_decode_payload",
		Message:     "Failed to decode payload",
		SpecSection: "§3.1",
		Severity:    "error",
	},
	errPayloadTooLarge: {
		Token:       "bssci.error.payload_too_large",
		Message:     "Payload exceeds maximum size (4GB for uint32 length field)",
		SpecSection: "§3.1",
		Severity:    "error",
	},

	// Version negotiation errors (BSSCI §2.1-2.3)
	errInvalidVersionFormat: {
		Token:       "bssci.error.invalid_version_format",
		Message:     "Invalid version format",
		SpecSection: "§2.1",
		Severity:    "protocol_violation",
	},
	errInvalidMajorVersion: {
		Token:       "bssci.error.invalid_major_version",
		Message:     "Invalid major version",
		SpecSection: "§2.1",
		Severity:    "protocol_violation",
	},
	errInvalidMinorVersion: {
		Token:       "bssci.error.invalid_minor_version",
		Message:     "Invalid minor version",
		SpecSection: "§2.1",
		Severity:    "protocol_violation",
	},
	errInvalidPatchVersion: {
		Token:       "bssci.error.invalid_patch_version",
		Message:     "Invalid patch version",
		SpecSection: "§2.1",
		Severity:    "protocol_violation",
	},
	errUnsupportedMajorVersion: {
		Token:       "bssci.error.unsupported_major_version",
		Message:     "Unsupported major version",
		SpecSection: "§2.1",
		Severity:    "protocol_violation",
	},
	errUnsupportedMinorVersion: {
		Token:       "bssci.error.unsupported_minor_version",
		Message:     "Unsupported minor version",
		SpecSection: "§2.1",
		Severity:    "protocol_violation",
	},

	// Message normalization errors (BSSCI §2.4)
	errNormalizationFailed: {
		Token:       "bssci.error.normalization_failed",
		Message:     "Message normalization failed",
		SpecSection: "§2.4",
		Severity:    "protocol_violation",
	},
	errMandatoryFieldMissing: {
		Token:       "bssci.error.mandatory_field_missing",
		Message:     "Mandatory field missing",
		SpecSection: "§2.4",
		Severity:    "protocol_violation",
	},
	errInvalidFieldType: {
		Token:       "bssci.error.invalid_field_type",
		Message:     "Invalid field type",
		SpecSection: "§2.4",
		Severity:    "protocol_violation",
	},
	errConditionalRuleFailed: {
		Token:       "bssci.error.conditional_rule_failed",
		Message:     "Conditional field requirement failed",
		SpecSection: "§2.4",
		Severity:    "protocol_violation",
	},

	// Operation ID validation errors (BSSCI §3.2)
	errInvalidOperationID: {
		Token:       "bssci.error.invalid_operation_id",
		Message:     "Invalid operation ID",
		SpecSection: "§3.2",
		Severity:    "protocol_violation",
	},
	errOperationIDNotPositive: {
		Token:       "bssci.error.operation_id_not_positive",
		Message:     "Base station operation ID must be positive",
		SpecSection: "§3.2",
		Severity:    "protocol_violation",
	},
	errOperationIDBackwards: {
		Token:       "bssci.error.operation_id_backwards",
		Message:     "Base station operation ID must not go backwards",
		SpecSection: "§3.2",
		Severity:    "protocol_violation",
	},
	errSCOperationIDMustBeNegative: {
		Token:       "bssci.error.sc_operation_id_must_be_negative",
		Message:     "Service Center operation ID must be negative",
		SpecSection: "§3.2-02",
		Severity:    "protocol_violation",
	},
	errOperationIDValidation: {
		Token:       "bssci.error.operation_id_validation",
		Message:     "Operation ID validation failed",
		SpecSection: "§3.2",
		Severity:    "protocol_violation",
	},
	errInvalidConnectOpId: {
		Token:       "bssci.error.invalid_connect_op_id",
		Message:     "Invalid connect opId",
		SpecSection: "§3.3.1",
		Severity:    "protocol_violation",
	},
	errConnectOpIDNotZero: {
		Token:       "bssci.error.connect_op_id_not_zero",
		Message:     "Connect operation must use opId 0",
		SpecSection: "§3.3.1",
		Severity:    "protocol_violation",
	},
	errInvalidConnectCompleteOpId: {
		Token:       "bssci.error.invalid_connect_complete_op_id",
		Message:     "Invalid opId for connect complete",
		SpecSection: "§3.3.3",
		Severity:    "protocol_violation",
	},

	// Connect operation errors (BSSCI §3.3)
	errMissingBsEui: {
		Token:       "bssci.error.missing_bs_eui",
		Message:     "Missing bsEui",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errInvalidBsEui: {
		Token:       "bssci.error.invalid_bs_eui",
		Message:     "Invalid bsEui: negative value not allowed",
		SpecSection: "§3.3.1",
		Severity:    "error",
	},
	errInvalidUUIDByteType: {
		Token:       "bssci.error.invalid_uuid_byte_type",
		Message:     "Invalid UUID byte type",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errInvalidUUIDLength: {
		Token:       "bssci.error.invalid_uuid_length",
		Message:     "UUID must be 16 bytes",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errUnsupportedUUIDType: {
		Token:       "bssci.error.unsupported_uuid_type",
		Message:     "Unsupported UUID type",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errNoConnectedBaseStations: {
		Token:       "bssci.error.no_connected_base_stations",
		Message:     "No connected base stations available",
		SpecSection: "§3",
		Severity:    "error",
	},
	errFailedToLoadCA: {
		Token:       "bssci.error.failed_to_load_ca",
		Message:     "Failed to load CA certificate",
		SpecSection: "§1",
		Severity:    "error",
	},
	errFailedToParseCA: {
		Token:       "bssci.error.failed_to_parse_ca",
		Message:     "Failed to parse CA certificate",
		SpecSection: "§1",
		Severity:    "error",
	},
	errFailedToLoadTLS: {
		Token:       "bssci.error.failed_to_load_tls",
		Message:     "Failed to load TLS certificate",
		SpecSection: "§1",
		Severity:    "error",
	},
	errFailedToStartTLS: {
		Token:       "bssci.error.failed_to_start_tls",
		Message:     "Failed to start TLS listener",
		SpecSection: "§1",
		Severity:    "error",
	},
	errCommandBeforeHandshake: {
		Token:       "bssci.error.command_before_handshake",
		Message:     "Operation not allowed before connect handshake completes",
		SpecSection: "§3.3",
		Severity:    "protocol_violation",
	},
	errMissingProtocolVersion: {
		Token:       "bssci.error.missing_protocol_version",
		Message:     "Protocol version must be provided by Base Station",
		SpecSection: "§5.3",
		Severity:    "error",
	},
	errMissingBidiFlag: {
		Token:       "bssci.error.missing_bidi_flag",
		Message:     "Bidirectional capability flag (bidi) must be explicitly declared",
		SpecSection: "§5.3",
		Severity:    "error",
	},

	// Attach operation errors (BSSCI §3.6)
	errMissingEpEui: {
		Token:       "bssci.error.missing_ep_eui",
		Message:     "Missing epEui",
		SpecSection: "§3.6.1",
		Severity:    "error",
	},
	errAttachOperationFailed: {
		Token:       "bssci.error.attach_operation_failed",
		Message:     "Attach operation failed",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	errAttachPropagateFailed: {
		Token:       "bssci.error.attach_propagate_failed",
		Message:     "Attach propagate failed",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	errPropagationBroadcastFailure: {
		Token:       "bssci.error.propagation_broadcast_failure",
		Message:     "Propagation broadcast encountered failures",
		SpecSection: "§5.8.3",
		Severity:    "error",
	},
	errEndpointAttachFailed: {
		Token:       "bssci.error.endpoint_attach_failed",
		Message:     "Endpoint attach failed",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	errInvalidEndpointEUIFormat: {
		Token:       "bssci.error.invalid_endpoint_eui_format",
		Message:     "Invalid endpoint EUI format",
		SpecSection: "§3.6.1",
		Severity:    "error",
	},
	errBaseStationNotBidirectional: {
		Token:       "bssci.error.base_station_not_bidirectional",
		Message:     "Base station not bidirectional",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	errNoPendingAttachOperation: {
		Token:       "bssci.error.no_pending_attach_operation",
		Message:     "No pending attach operation",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	errInvalidNwkSnKeyLength: {
		Token:       "bssci.error.invalid_nwk_sn_key_length",
		Message:     "Network session key must be exactly 16 bytes",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	errBaseStationEUIOutOfRange: {
		Token:       "bssci.error.base_station_eui_out_of_range",
		Message:     "Base station EUI exceeds signed 64-bit integer range",
		SpecSection: "§3.1 (field encoding rules), §5.6.1 (attach telemetry)",
		Severity:    "protocol_violation",
	},

	// Detach operation errors (BSSCI §3.7)
	errDetachOperationFailed: {
		Token:       "bssci.error.detach_operation_failed",
		Message:     "Detach operation failed",
		SpecSection: "§3.7",
		Severity:    "error",
	},
	errDetachPropagateFailed: {
		Token:       "bssci.error.detach_propagate_failed",
		Message:     "Detach propagate failed",
		SpecSection: "§3.7",
		Severity:    "error",
	},
	errAttachPropagateRejected: {
		Token:       "bssci.error.attach_propagate_rejected",
		Message:     "Base station rejected attach propagate operation",
		SpecSection: "§5.17",
		Severity:    "error",
	},
	errDetachPropagateRejected: {
		Token:       "bssci.error.detach_propagate_rejected",
		Message:     "Base station rejected detach propagate operation",
		SpecSection: "§5.17",
		Severity:    "error",
	},
	errEndpointDetachFailed: {
		Token:       "bssci.error.endpoint_detach_failed",
		Message:     "Endpoint detach failed",
		SpecSection: "§3.7",
		Severity:    "error",
	},
	errNoPendingDetachOperation: {
		Token:       "bssci.error.no_pending_detach_operation",
		Message:     "No pending detach operation",
		SpecSection: "§3.7",
		Severity:    "error",
	},
	errDetachSignatureInvalid: {
		Token:       "bssci.error.detach_signature_invalid",
		Message:     "Unknown endpoint detach signature validation failed",
		SpecSection: "§5.7",
		Severity:    "error",
	},
	errEndpointNotFound: {
		Token:       "bssci.error.endpoint_not_found",
		Message:     "Unknown endpoint not found in database",
		SpecSection: "§5.7",
		Severity:    "error",
	},
	errSignatureMismatch: {
		Token:       "bssci.error.signature_mismatch",
		Message:     "Cryptographic signature verification failed",
		SpecSection: "§5.7.1",
		Severity:    "error",
	},

	// Downlink adapter errors (BSSCI §5.13)
	errDLQueueNilResult: {
		Token:       "bssci.error.dl_queue_nil_result",
		Message:     "QueueDownlinkInternal returned nil result",
		SpecSection: "§5.13",
		Severity:    "error",
	},

	// Downlink payload size (MIOTY radio protocol §4.3.2)
	errDLPayloadTooLarge: {
		Token:       "bssci.error.dl_payload_too_large",
		Message:     "Downlink payload exceeds 200-byte maximum (MIOTY radio protocol §4.3.2)",
		SpecSection: "§4.3.2",
		Severity:    "error",
	},

	// Downlink data queue errors (BSSCI §3.9)
	errInvalidQueueID: {
		Token:       "bssci.error.invalid_queue_id",
		Message:     "Invalid queue ID",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errQueueIDNotFound: {
		Token:       "bssci.error.queue_id_not_found",
		Message:     "Queue ID not found",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errMissingQueID: {
		Token:       "bssci.error.missing_que_id",
		Message:     "Missing queId",
		SpecSection: "§3.9.1",
		Severity:    "error",
	},
	errCannotResolveTenantForQueue: {
		Token:       "bssci.error.cannot_resolve_tenant_for_queue",
		Message:     "Cannot resolve tenant for queue",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errInvalidPayloadCount: {
		Token:       "bssci.error.invalid_payload_count",
		Message:     "Invalid payload count",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errCounterDependentPayloadMismatch: {
		Token:       "bssci.error.counter_dependent_payload_mismatch",
		Message:     "Counter-dependent mode payload mismatch",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errNonCounterDependentMultiPayload: {
		Token:       "bssci.error.non_counter_dependent_multi_payload",
		Message:     "Non-counter-dependent mode requires exactly 1 payload",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errMissingPacketCnt: {
		Token:       "bssci.error.missing_packet_cnt",
		Message:     "Missing packetCnt",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errMissingResultField: {
		Token:       "bssci.error.missing_result_field",
		Message:     "Missing result field in dlDataRes",
		SpecSection: "§3.14.1",
		Severity:    "protocol_violation",
	},
	errMissingTxTimeOrPacketCnt: {
		Token:       "bssci.error.missing_txtime_or_packetcnt",
		Message:     "txTime and packetCnt are required when result='sent'",
		SpecSection: "§3.14.1",
		Severity:    "protocol_violation",
	},
	errUnexpectedTxTimeOrPacketCnt: {
		Token:       "bssci.error.unexpected_txtime_or_packetcnt",
		Message:     "txTime and packetCnt must not be present when result is not 'sent'",
		SpecSection: "§3.14.1",
		Severity:    "protocol_violation",
	},
	errInvalidTenantIDFormat: {
		Token:       "bssci.error.invalid_tenant_id_format",
		Message:     "Invalid tenant ID format",
		SpecSection: "§3",
		Severity:    "error",
	},
	errQueueIDOutOfRange: {
		Token:       "bssci.error.queue_id_out_of_range",
		Message:     "Queue ID exceeds maximum safe value for int64 conversion",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errNegativeQueueIDInMetadata: {
		Token:       "bssci.error.negative_queue_id_in_metadata",
		Message:     "Negative queue ID found in operation metadata",
		SpecSection: "§3.9",
		Severity:    "error",
	},
	errInvalidPacketCounter: {
		Token:       "bssci.error.invalid_packet_counter",
		Message:     "Packet counter value must be in uint32 range (0 to 4294967295)",
		SpecSection: "§3.12",
		Severity:    "protocol_violation",
	},

	// DL Data Revoke errors (BSSCI §3.10)
	errCannotSendDlDataRev: {
		Token:       "bssci.error.cannot_send_dl_data_rev",
		Message:     "Cannot send dlDataRev: handshake not complete",
		SpecSection: "§3.10",
		Severity:    "protocol_violation",
	},
	errFailedToSendDlDataRev: {
		Token:       "bssci.error.failed_to_send_dl_data_rev",
		Message:     "Failed to send dlDataRev",
		SpecSection: "§3.10",
		Severity:    "error",
	},
	errMissingEpEuiInMetadata: {
		Token:       "bssci.error.missing_ep_eui_in_metadata",
		Message:     "Missing epEui in metadata",
		SpecSection: "§3.10",
		Severity:    "error",
	},
	errMissingQueIDInMetadata: {
		Token:       "bssci.error.missing_que_id_in_metadata",
		Message:     "Missing queId in metadata",
		SpecSection: "§3.10",
		Severity:    "error",
	},

	// DL RX Status errors (BSSCI §3.11)
	errFailedToPersistDLRXStatus: {
		Token:       "bssci.error.failed_to_persist_dl_rx_status",
		Message:     "Failed to persist DL RX status",
		SpecSection: "§3.11",
		Severity:    "error",
	},
	errFailedToSendDlRxStatQry: {
		Token:       "bssci.error.failed_to_send_dl_rx_stat_qry",
		Message:     "Failed to send dlRxStatQry",
		SpecSection: "§3.11",
		Severity:    "error",
	},
	errInvalidResultValue: {
		Token:       "bssci.error.invalid_result_value",
		Message:     "Invalid result value",
		SpecSection: "§3.11",
		Severity:    "error",
	},
	errMissingDlRxEpEui: {
		Token:       "bssci.error.missing_dl_rx_ep_eui",
		Message:     "Missing epEui in dlRxStat",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errMissingDlRxTime: {
		Token:       "bssci.error.missing_dl_rx_time",
		Message:     "Missing rxTime in dlRxStat",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errMissingDlRxPacketCnt: {
		Token:       "bssci.error.missing_dl_rx_packet_cnt",
		Message:     "Missing packetCnt in dlRxStat",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errInvalidDlRxPacketCnt: {
		Token:       "bssci.error.invalid_dl_rx_packet_cnt",
		Message:     "Invalid packetCnt in dlRxStat",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errMissingDlRxSnr: {
		Token:       "bssci.error.missing_dl_rx_snr",
		Message:     "Missing dlRxSnr in dlRxStat",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errInvalidDlRxSnr: {
		Token:       "bssci.error.invalid_dl_rx_snr",
		Message:     "dlRxSnr value must be in uint32 range (0 to 4294967295)",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errMissingDlRxRssi: {
		Token:       "bssci.error.missing_dl_rx_rssi",
		Message:     "Missing dlRxRssi in dlRxStat",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errInvalidDlRxRssi: {
		Token:       "bssci.error.invalid_dl_rx_rssi",
		Message:     "dlRxRssi value must be in uint32 range (0 to 4294967295)",
		SpecSection: "§3.11",
		Severity:    "protocol_violation",
	},
	errUnexpectedDlRxStatus: {
		Token:       "bssci.error.unexpected_dl_rx_status",
		Message:     "Received dlRxStat with no pending query",
		SpecSection: "§5.15",
		Severity:    "protocol_violation",
	},

	// Database and persistence errors
	errDatabaseError: {
		Token:       "bssci.error.database_error",
		Message:     "Database error",
		SpecSection: "§4",
		Severity:    "error",
	},
	errDatabaseNotAvailable: {
		Token:       "bssci.error.database_not_available",
		Message:     "Database not available",
		SpecSection: "§4",
		Severity:    "error",
	},
	errDatabaseUpdateFailed: {
		Token:       "bssci.error.database_update_failed",
		Message:     "Database update failed",
		SpecSection: "§4",
		Severity:    "error",
	},
	errDeduplicationError: {
		Token:       "bssci.error.deduplication_error",
		Message:     "Deduplication error",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errPacketCounterCollision: {
		Token:       "bssci.error.packet_counter_collision",
		Message:     "Packet counter collision detected during deduplication",
		SpecSection: "§3.8",
		Severity:    "error",
	},

	// Version and protocol compatibility errors
	errVersionIncompatible: {
		Token:       "bssci.error.version_incompatible",
		Message:     "Version incompatible",
		SpecSection: "§2.1",
		Severity:    "protocol_violation",
	},
	errUUIDDataNil: {
		Token:       "bssci.error.uuid_data_nil",
		Message:     "UUID data is nil",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	errSCOperationIDMustNotIncrease: {
		Token:       "bssci.error.sc_operation_id_must_not_increase",
		Message:     "Service center operation ID must not increase",
		SpecSection: "§3.2",
		Severity:    "protocol_violation",
	},

	// Propagation and operation failures
	errOperationFailed: {
		Token:       "bssci.error.operation_failed",
		Message:     "Operation failed",
		SpecSection: "§3",
		Severity:    "error",
	},
	errPropagateFailed: {
		Token:       "bssci.error.propagate_failed",
		Message:     "Propagate failed",
		SpecSection: "§3",
		Severity:    "error",
	},
	errEndpointOperationFailed: {
		Token:       "bssci.error.endpoint_operation_failed",
		Message:     "Endpoint operation failed",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	errFailedToSendError: {
		Token:       "bssci.error.failed_to_send_error",
		Message:     "Failed to send error",
		SpecSection: "§4",
		Severity:    "error",
	},
	errBaseStationReportedError: {
		Token:       "bssci.error.base_station_reported_error",
		Message:     "Base station reported error",
		SpecSection: "§4",
		Severity:    "error",
	},

	// Variable MAC (VM) operation errors
	errVMOperationSentByBS: {
		Token:       "bssci.error.vm_operation_sent_by_bs",
		Message:     "VM operation should be sent by Service Center, not received from Base Station",
		SpecSection: "§3",
		Severity:    "protocol_violation",
	},
	errMissingMacType: {
		Token:       "bssci.error.missing_mac_type",
		Message:     "Missing macType in VM uplink data",
		SpecSection: "§3",
		Severity:    "error",
	},
	errMissingRxTime: {
		Token:       "bssci.error.missing_rx_time",
		Message:     "Missing rxTime in VM uplink data",
		SpecSection: "§3",
		Severity:    "error",
	},
	errFailedToSendVMActivate: {
		Token:       "bssci.error.failed_to_send_vm_activate",
		Message:     "Failed to send VM activate",
		SpecSection: "§3",
		Severity:    "error",
	},
	errFailedToSendVMDeactivate: {
		Token:       "bssci.error.failed_to_send_vm_deactivate",
		Message:     "Failed to send VM deactivate",
		SpecSection: "§3",
		Severity:    "error",
	},
	errFailedToSendVMStatus: {
		Token:       "bssci.error.failed_to_send_vm_status",
		Message:     "Failed to send VM status",
		SpecSection: "§3",
		Severity:    "error",
	},
	errFailedToSendVMDlData: {
		Token:       "bssci.error.failed_to_send_vm_dl_data",
		Message:     "Failed to send VM downlink data",
		SpecSection: "§3",
		Severity:    "error",
	},

	// Uplink transmit errors
	errNwkSnKeyInvalidLength: {
		Token:       "bssci.error.nwk_sn_key_invalid_length",
		Message:     "nwkSnKey must be exactly 16 bytes",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errBaseStationTenantMismatch: {
		Token:       "bssci.error.base_station_tenant_mismatch",
		Message:     "Base station belongs to different tenant - cannot schedule UL transmit",
		SpecSection: "§5.11 (UL data transmit), multi-tenant isolation",
		Severity:    "error",
	},
	errSessionNotReady: {
		Token:       "bssci.error.session_not_ready",
		Message:     "Base station session not ready - handshake incomplete",
		SpecSection: "§3.3 (connect), §5.16 (DL RX status query)",
		Severity:    "error",
	},
	errSessionNotBidirectional: {
		Token:       "bssci.error.session_not_bidirectional",
		Message:     "Base station does not support bidirectional operations",
		SpecSection: "§5.16 (DL RX status query), bidirectional capability check",
		Severity:    "error",
	},
	errFailedToSendULDataTransmitComplete: {
		Token:       "bssci.error.failed_to_send_ul_data_transmit_complete",
		Message:     "Failed to send UL data transmit completion",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errFailedToEncryptKey: {
		Token:       "bssci.error.failed_to_encrypt_key",
		Message:     "Failed to encrypt key",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errFailedToDecryptKey: {
		Token:       "bssci.error.failed_to_decrypt_key",
		Message:     "Failed to decrypt key",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errMissingEncryptedKey: {
		Token:       "bssci.error.missing_encrypted_key",
		Message:     "Encrypted key missing from metadata",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errFailedToDecodeUserData: {
		Token:       "bssci.error.failed_to_decode_user_data",
		Message:     "Failed to decode userData",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errFailedToSendULTransmit: {
		Token:       "bssci.error.failed_to_send_ul_transmit",
		Message:     "Failed to send UL transmit to BS",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errMissingAttachCnt: {
		Token:       "bssci.error.missing_attach_cnt",
		Message:     "Missing mandatory field: attachCnt",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errMissingNonce: {
		Token:       "bssci.error.missing_nonce",
		Message:     "Missing or invalid mandatory field: nonce (must be 4 bytes)",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errMissingSign: {
		Token:       "bssci.error.missing_sign",
		Message:     "Missing or invalid mandatory field: sign (must be 4 bytes)",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errInvalidSignature: {
		Token:       "bssci.error.invalid_signature",
		Message:     "Attach signature validation failed",
		SpecSection: "§3.6.1/3.7.1",
		Severity:    "protocol_violation",
	},
	errInvalidAttachCntRange: {
		Token:       "bssci.error.invalid_attach_cnt_range",
		Message:     "attachCnt must be 24-bit unsigned (0 to 16777215)",
		SpecSection: "§5.6.1",
		Severity:    "error",
	},
	errInvalidNonceElement: {
		Token:       "bssci.error.invalid_nonce_element",
		Message:     "nonce array elements must be integers 0-255",
		SpecSection: "§5.6.1",
		Severity:    "error",
	},
	errInvalidSignElement: {
		Token:       "bssci.error.invalid_sign_element",
		Message:     "sign array elements must be integers 0-255",
		SpecSection: "§5.6.1",
		Severity:    "error",
	},
	errAttachCounterNotMonotonic: {
		Token:       "bssci.error.attach_counter_not_monotonic",
		Message:     "attachCnt must be greater than previous value (replay protection)",
		SpecSection: "§5.6.1",
		Severity:    "security",
	},
	errInvalidSnrValue: {
		Token:       "bssci.error.invalid_snr_value",
		Message:     "Missing or invalid mandatory field: snr (must be numeric)",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errInvalidRssiValue: {
		Token:       "bssci.error.invalid_rssi_value",
		Message:     "Missing or invalid mandatory field: rssi (must be numeric)",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errInvalidEqSnrValue: {
		Token:       "bssci.error.invalid_eqsnr_value",
		Message:     "Invalid value type for optional field: eqSnr (must be numeric)",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errEndpointNotProvisioned: {
		Token:       "bssci.error.endpoint_not_provisioned",
		Message:     "Endpoint not provisioned",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errRoamingNotAllowed: {
		Token:       "bssci.error.roaming_not_allowed",
		Message:     "Roaming not allowed between tenants",
		SpecSection: "§3.8 (roaming)",
		Severity:    "error",
	},
	errResponseExpRequiresDlOpen: {
		Token:       "bssci.error.response_exp_requires_dl_open",
		Message:     "responseExp flag requires dlOpen flag per MIOTY Radio Protocol §3.6.5.1",
		SpecSection: "§5.10.1",
		Severity:    "protocol_violation",
	},
	errRxTimeExceedsMaximum: {
		Token:       "bssci.error.rx_time_exceeds_maximum",
		Message:     "rxTime value exceeds maximum",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errInvalidRxTimeFormat: {
		Token:       "bssci.error.invalid_rx_time_format",
		Message:     "Invalid rxTime format",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errInvalidRxTimeValue: {
		Token:       "bssci.error.invalid_rx_time_value",
		Message:     "Invalid rxTime value: must be positive",
		SpecSection: "§3.8",
		Severity:    "protocol_violation",
	},
	errFailedToPersistULData: {
		Token:       "bssci.error.failed_to_persist_ul_data",
		Message:     "Failed to persist UL data",
		SpecSection: "§3.8",
		Severity:    "error",
	},
	errFailedToResolveEndpointTenant: {
		Token:       "bssci.error.failed_to_resolve_endpoint_tenant",
		Message:     "Failed to resolve endpoint tenant for UL data - database error",
		SpecSection: "§3.8.1",
		Severity:    "error",
	},
	errUnsupportedSublayerPrefix: {
		Token:       "bssci.error.unsupported_sublayer_prefix",
		Message:     "Unsupported sublayer prefix in command",
		SpecSection: "§3.1",
		Severity:    "protocol_violation",
	},
	errUnsupportedCommand: {
		Token:       "bssci.error.unsupported_command",
		Message:     "Unsupported command",
		SpecSection: "§3.1",
		Severity:    "protocol_violation",
	},

	// Status operation errors
	errFailedToSendStatusRequest: {
		Token:       "bssci.error.failed_to_send_status_request",
		Message:     "Failed to send status request",
		SpecSection: "§3.5",
		Severity:    "error",
	},
	errMissingStatusCode: {
		Token:       "bssci.error.missing_status_code",
		Message:     "Missing mandatory field: code",
		SpecSection: "§3.5.2",
		Severity:    "protocol_violation",
	},
	errMissingStatusMessage: {
		Token:       "bssci.error.missing_status_message",
		Message:     "Missing mandatory field: message",
		SpecSection: "§3.5.2",
		Severity:    "protocol_violation",
	},
	errMissingStatusTime: {
		Token:       "bssci.error.missing_status_time",
		Message:     "Missing mandatory field: time",
		SpecSection: "§3.5.2",
		Severity:    "protocol_violation",
	},

	// Management HTTP API errors
	ErrMgmtMethodNotAllowed: {
		Token:       "bssci.mgmt.method_not_allowed",
		Message:     "Method not allowed",
		SpecSection: "§3.6,§3.7", // Attach/detach propagation endpoints
		Severity:    "error",
	},
	ErrMgmtSessionIDRequired: {
		Token:       "bssci.mgmt.session_id_required",
		Message:     "session_id query parameter required",
		SpecSection: "§3.3", // Session management
		Severity:    "error",
	},
	ErrMgmtInvalidRequestBody: {
		Token:       "bssci.mgmt.invalid_request_body",
		Message:     "Invalid request body",
		SpecSection: "§3.6,§3.7",
		Severity:    "error",
	},
	ErrMgmtInvalidNwkSnKeyEncoding: {
		Token:       "bssci.mgmt.invalid_nwk_sn_key_encoding",
		Message:     "Invalid nwkSnKey encoding",
		SpecSection: "§3.8.1", // Network session key requirements
		Severity:    "error",
	},
	ErrMgmtInvalidNwkSnKeyLength: {
		Token:       "bssci.mgmt.invalid_nwk_sn_key_length",
		Message:     "nwkSnKey must be exactly 16 bytes",
		SpecSection: "§3.8.1", // BSSCI-3.8.1-01: key length validation
		Severity:    "protocol_violation",
	},
	ErrMgmtAttachPropagateFailed: {
		Token:       "bssci.mgmt.attach_propagate_failed",
		Message:     "Failed to send attach propagate",
		SpecSection: "§3.6",
		Severity:    "error",
	},
	ErrMgmtDetachPropagateFailed: {
		Token:       "bssci.mgmt.detach_propagate_failed",
		Message:     "Failed to send detach propagate",
		SpecSection: "§3.7",
		Severity:    "error",
	},
	ErrMgmtJSONEncodeFailed: {
		Token:       "bssci.mgmt.json_encode_failed",
		Message:     "Failed to encode JSON response",
		SpecSection: "§3.1", // Message encoding
		Severity:    "error",
	},
	ErrMgmtFailedToBindAddress: {
		Token:       "bssci.mgmt.failed_to_bind_address",
		Message:     "Failed to bind to management API address",
		SpecSection: "N/A", // Infrastructure, not protocol
		Severity:    "error",
	},
	ErrMgmtServerFailed: {
		Token:       "bssci.mgmt.server_failed",
		Message:     "Management HTTP server failed",
		SpecSection: "N/A",
		Severity:    "error",
	},
	ErrMgmtInvalidTenantContext: {
		Token:       "bssci.mgmt.invalid_tenant_context",
		Message:     "Cannot resolve tenant context for operation",
		SpecSection: "§3.2", // Operation context
		Severity:    "error",
	},
	ErrMgmtNoConnectedSessions: {
		Token:       "bssci.mgmt.no_connected_sessions",
		Message:     "No connected base station sessions available",
		SpecSection: "§3.3",
		Severity:    "error",
	},
	ErrMgmtSessionLookupFailed: {
		Token:       "bssci.mgmt.session_lookup_failed",
		Message:     "Failed to look up session state",
		SpecSection: "§3.3",
		Severity:    "error",
	},

	// Type conversion and validation errors
	errInvalidFieldValue: {
		Token:       "bssci.error.invalid_field_value",
		Message:     "Field value out of valid range for MIOTY type encoding",
		SpecSection: "§3.1 (field encoding rules), Table 3-1 (data type limits)",
		Severity:    "protocol_violation",
	},

	// Organization context enforcement
	errOrgContextMissing: {
		Token:       "bssci.error.org_context_missing",
		Message:     "Organization context required in strict mode (gRPC metadata missing X-Organization-ID)",
		SpecSection: "§3.1",
		Severity:    "error",
	},

	// Float validation errors (BSSCI §5.15.1)
	errInvalidFloatNaN: {
		Token:       "bssci.err.invalid_float_nan",
		Message:     "Float field contains NaN",
		SpecSection: "§5.15.1",
		Severity:    "protocol_violation",
	},
	errInvalidFloatInf: {
		Token:       "bssci.err.invalid_float_inf",
		Message:     "Float field contains infinity",
		SpecSection: "§5.15.1",
		Severity:    "protocol_violation",
	},
	errFloatOutOfRange: {
		Token:       "bssci.err.float_out_of_range",
		Message:     "Float field outside valid range",
		SpecSection: "§5.15.1",
		Severity:    "protocol_violation",
	},

	// Outbound message validation errors (BSSCI §2.5)
	errOutboundMissingCommand: {
		Token:       "bssci.error.outbound_missing_command",
		Message:     "outbound message missing command field",
		SpecSection: "§2.5.1",
		Severity:    "protocol_violation",
	},
	errOutboundMissingField: {
		Token:       "bssci.error.outbound_missing_field",
		Message:     "outbound message missing mandatory field",
		SpecSection: "§2.5.2",
		Severity:    "protocol_violation",
	},
	errOutboundMissingMandatoryField: {
		Token:       "bssci.error.outbound_missing_mandatory_field",
		Message:     "outbound message missing mandatory non-optional field per specification",
		SpecSection: "§4.5, §5",
		Severity:    "protocol_violation",
	},
	errOutboundExtraField: {
		Token:       "bssci.error.outbound_extra_field",
		Message:     "outbound message contains non-specification field",
		SpecSection: "§2.5.3",
		Severity:    "protocol_violation",
	},
	errOutboundUnknownCommand: {
		Token:       "bssci.error.outbound_unknown_command",
		Message:     "outbound command not in specification catalog",
		SpecSection: "§2.5",
		Severity:    "protocol_violation",
	},
	errOutboundMarshalFailed: {
		Token:       "bssci.error.outbound_marshal_failed",
		Message:     "outbound message failed type conversion for validation",
		SpecSection: "§2.5",
		Severity:    "internal_error",
	},
	errOutboundInvalidFieldType: {
		Token:       "bssci.error.outbound_invalid_field_type",
		Message:     "outbound message field has invalid type (expected numeric type for opId)",
		SpecSection: "§2.5.1",
		Severity:    "protocol_violation",
	},

	// Unknown/generic errors
	errUnknownError: {
		Token:       "bssci.error.unknown",
		Message:     "Unknown error",
		SpecSection: "§4",
		Severity:    "error",
	},
}

// GetErrorDefinition retrieves the error definition for a given token
// Returns a default definition if token not found
func GetErrorDefinition(token string) ErrorDefinition {
	if def, ok := errorDefinitions[token]; ok {
		return def
	}
	// Return default for unknown tokens
	return ErrorDefinition{
		Token:       token,
		Message:     "Unknown error",
		SpecSection: "§4",
		Severity:    "error",
	}
}

// ResolveErrorMessage returns the human-readable message for an error token
func ResolveErrorMessage(token string) string {
	return GetErrorDefinition(token).Message
}
