// Package scaci log message constants
//
// This file centralizes all SCACI protocol log messages to ensure consistency,
// enable future localization, and prevent string duplication across handlers.
//
// Usage Pattern:
//
//	s.logger.Error(LogSCACITLSHandshakeFailed, zap.Error(err))
//	s.logger.Info(LogSCACISessionResumed, zap.Int64("sessionId", session.ID))
//	s.logger.Debug(LogSCACIReceivedMessage, zap.String("command", cmd), zap.Int64("opId", id))
//
// Naming Convention:
//   - All constants prefixed with LogSCACI
//   - Grouped by subsystem (Server, TLS, Connect, Register, etc.)
//   - Use present tense for events ("Received", "Processing")
//   - Use past tense for outcomes ("Failed", "Completed")
//
// Structured Context:
//   - Log messages are static strings
//   - Dynamic context passed via zap fields (zap.String, zap.Int64, zap.Error, etc.)
//   - NEVER embed data in the message constant itself
//
// Deduplication:
//   - 127 unique messages extracted from 136 total log occurrences
//   - Shared messages (e.g., "Failed to mark operation completed") reused across operations
package scaci

// SCACI log message constants for structured logging
const (
	// ========================================================================
	// Server Lifecycle Messages (3 constants)
	// ========================================================================

	LogSCACIServerListening           = "SCACI server listening"
	LogSCACIServerStopping            = "Stopping SCACI server..."
	LogSCACIServerStopped             = "SCACI server stopped"
	LogSCACICertsNotFound             = "SCACI certificates not found, listener deferred until certificates are generated"
	LogSCACICertsDetected             = "SCACI certificates detected, starting TLS listener"
	LogSCACIDeferredListenerFailed    = "Failed to start deferred SCACI TLS listener"
	LogSCACIDeferredListenerCancelled = "SCACI deferred listener polling cancelled"

	// ========================================================================
	// TLS & Connection Messages (7 constants)
	// ========================================================================

	LogSCACIConnectionNotTLS         = "Connection is not TLS (should never happen)"
	LogSCACITLSHandshakeFailed       = "TLS handshake failed"
	LogSCACINoClientCertificate      = "No client certificate provided (mutual TLS required)"
	LogSCACICertificateMappingFailed = "Failed to map certificate to tenant"
	LogSCACIConnectionEstablished    = "SCACI connection established"
	LogSCACIConnectionClosed         = "SCACI connection closed"
	LogSCACIAcceptConnectionFailed   = "Failed to accept connection"

	// ========================================================================
	// Tenant Mapping Messages (7 constants)
	// ========================================================================

	LogSCACITenantMappedFromCN        = "Tenant mapped from CN"
	LogSCACITenantMappedFromNumericCN = "Tenant mapped from numeric CN"
	LogSCACITenantMappedFromSAN       = "Tenant mapped from SAN"
	LogSCACICertMappingFailedStrict   = "Certificate tenant mapping failed (strict mode)"
	LogSCACIUsingFallbackTenantUnsafe = "Using fallback tenant ID - UNSAFE FOR PRODUCTION"
	LogSCACIOrgResolverNotInjected    = "SCACI: org resolver not injected into handshake service"
	LogSCACICrossTenantResumeRejected = "SCACI: resume rejected due to certificate tenant mismatch"
	LogSCACIOrgEnforcementNilUUID     = "Organization enforcement enabled but session has nil org UUID"

	// ========================================================================
	// Certificate Validation Messages (5 constants)
	// ========================================================================

	LogSCACICertNotYetValid       = "Certificate not yet valid (NotBefore is in the future)"
	LogSCACICertExpired           = "Certificate expired (NotAfter is in the past)"
	LogSCACICertMissingClientAuth = "Certificate missing ClientAuth extended key usage"
	LogSCACICertInvalidSubject    = "Certificate has invalid subject (missing CN and Organization)"
	LogSCACICertValidationPassed  = "Certificate validation passed"

	// ========================================================================
	// Message Framing & Dispatch (9 constants)
	// ========================================================================

	LogSCACIReceivedMessage           = "Received SCACI message"
	LogSCACIReadFrameFailed           = "Failed to read frame"
	LogSCACIDecodeMessagePackFailed   = "Failed to decode MessagePack"
	LogSCACIMissingCommandField       = "Missing or invalid 'command' field"
	LogSCACIMissingOpIDField          = "Missing or invalid 'opId' field"
	LogSCACIFirstMessageMustBeConnect = "First message must be Connect"
	LogSCACIHandlerError              = "Handler error"
	LogSCACIUnknownCommand            = "Unknown SCACI command"
	LogSCACIUnsupportedSublayerPrefix = "SCACI sublayer prefix not supported" // §4 sublayer guard

	// ========================================================================
	// Error Response Handling (10 constants) - SCACI §3.14
	// ========================================================================

	LogSCACIMarshalErrorFailed           = "Failed to marshal error message"
	LogSCACISendErrorFailed              = "Failed to send error message"
	LogSCACIErrorMessageSent             = "Sent error message"
	LogSCACIResponseSent                 = "Sent SCACI response"
	LogSCACIErrorReceived                = "SCACI error"
	LogSCACIInvalidPayload               = "Invalid message payload"
	LogSCACIReceivedInboundError         = "Received error message from AC"
	LogSCACIPersistInboundErrorFailed    = "Failed to persist inbound error"
	LogSCACICompleteErrorHandshakeFailed = "Failed to complete error handshake"
	LogSCACIPersistOutboundErrorFailed   = "Failed to persist outbound error"

	// ========================================================================
	// Connect Operation (14 constants)
	// ========================================================================

	LogSCACIProcessingConnect       = "Processing Connect message"
	LogSCACIDecodeConnectFailed     = "Failed to decode Connect message"
	LogSCACIConnectOpIDMustBeZero   = "Connect opId must be 0"
	LogSCACIConnectResumeFields     = "Connect resume fields"
	LogSCACIInvalidVersionFormat    = "Invalid version format"
	LogSCACIMajorVersionMismatch    = "Major version mismatch"
	LogSCACIMinorVersionTooHigh     = "Minor version too high"
	LogSCACIVersionNegotiationOk    = "Version negotiation successful"
	LogSCACISessionResumed          = "Session resumed"
	LogSCACIResumeFailed            = "Resume failed: opId mismatch"
	LogSCACISessionCannotResume     = "Session cannot be resumed"
	LogSCACIVersionMismatchOnResume = "Version mismatch on session resume"
	LogSCACINoResumableSession      = "No resumable session found"
	LogSCACINewSessionCreated       = "New session created"
	LogSCACIConnectComplete         = "Connect complete - session active"
	LogSCACIUpdateSessionFailed     = "Failed to update session"
	LogSCACIPersistSessionFailed    = "Failed to persist session"
	LogSCACIConnectCmpNonZeroOpID   = "ConnectComplete with non-zero opId, terminating"

	// ========================================================================
	// Register Operation (9 constants)
	// ========================================================================

	LogSCACIDecodeRegisterFailed      = "Failed to decode register payload"
	LogSCACIDatabaseErrorRegister     = "Database error during register"
	LogSCACICreateEndpointFailed      = "Failed to create endpoint"
	LogSCACIEndpointCreated           = "Created new endpoint via SCACI register"
	LogSCACIUpdateEndpointFailed      = "Failed to update endpoint fields"
	LogSCACIEndpointRegistered        = "Endpoint registered via SCACI"
	LogSCACIRecordRegisterOpFailed    = "Failed to record register operation"
	LogSCACIRegisterHandshakeComplete = "Register handshake complete"
	LogSCACILoadRegisterOpFailed      = "Failed to load register operation"

	// ========================================================================
	// BSSCI Attach Propagation Integration (2 constants)
	// BSSCI §5.8-5.8.3: Automatic attach propagation for preAttach endpoints
	// ========================================================================

	LogSCACITriggeringAttachPropagation = "Triggering attach propagation for preAttach endpoint"
	LogSCACIAttachPropagationErrors     = "Attach propagation encountered errors"

	// ========================================================================
	// Deregister Operation (11 constants)
	// ========================================================================

	LogSCACIDecodeDeregisterFailed      = "Failed to decode deregister payload"
	LogSCACIEndpointNotFoundDeregister  = "Endpoint not found for deregister"
	LogSCACIDatabaseErrorDeregister     = "Database error during deregister"
	LogSCACIRecordDeregisterOpFailed    = "Failed to record deregister operation"
	LogSCACIDetachEndpointFailed        = "Failed to detach endpoint"
	LogSCACIEndpointDeregistered        = "Endpoint deregistered"
	LogSCACIDeregisterHandshakeComplete = "Deregister handshake complete"
	LogSCACILoadDeregisterOpFailed      = "Failed to load deregister operation"
	LogSCACIDetachPropagationErrors     = "Detach propagation had errors"
	LogSCACIDetachPropagationSent       = "Detach propagation sent to all base stations"
	LogSCACIRevokeDownlinksFailed       = "Failed to revoke downlinks"
	LogSCACIDeregisterCleanupStart      = "Starting deregister cleanup"
	LogSCACIDeregisterCleanupSkipped    = "Deregister cleanup skipped"

	// ========================================================================
	// UL Data Operations (10 constants)
	// ========================================================================

	LogSCACISendULDataFailed           = "Failed to send UL data to AC session"
	LogSCACIReceivedULDataResponse     = "Received UL data response from AC"
	LogSCACIUpdateULDataOpAckFailed    = "Failed to update UL data operation to acknowledged"
	LogSCACISendULDataCompleteFailed   = "Failed to send UL data complete"
	LogSCACIULDataHandshakeComplete    = "UL data handshake complete"
	LogSCACIMarkULDataOpCompleteFailed = "Failed to mark UL data operation completed"
	LogSCACIRecordULDataOpFailed       = "Failed to record UL data operation"
	LogSCACISendULToACFailed           = "Failed to send UL to AC"
	LogSCACIUnexpectedULDataCmp        = "Received unexpected ulDataCmp from AC"
	LogSCACIUnexpectedULDataTxRsp      = "Received unexpected ulDataTxRsp from AC"

	// ========================================================================
	// UL Transmit Operations (12 constants)
	// ========================================================================

	LogSCACIDecodeULDataTxFailed     = "Failed to decode ulDataTx payload"
	LogSCACIBaseStationNotFoundULTx  = "Base station not found for UL transmit"
	LogSCACILookupBaseStationFailed  = "Failed to lookup base station"
	LogSCACIScheduleULTransmitFailed = "Failed to schedule UL transmit"
	LogSCACIScheduleULTxFailed       = "Failed to schedule UL transmit" // Alias
	LogSCACIULTransmitNotSupported   = "UL data transmit not supported (scheduler not configured)"
	LogSCACIULDataTxNotSupported     = "UL data transmit not supported (scheduler not configured)" // Alias
	LogSCACIULTransmitScheduled      = "UL data transmit scheduled"
	LogSCACIULDataTxScheduled        = "UL data transmit scheduled" // Alias
	LogSCACIProcessingULDataTxCmp    = "Processing ulDataTxCmp"
	LogSCACIMarkULTxAckFailed        = "Failed to mark UL transmit acknowledged"
	LogSCACISendULDataTxRspFailed    = "Failed to send ulDataTxRsp"
	LogSCACIRecordULTxOpFailed       = "Failed to record UL transmit operation"
	LogSCACIMarkULTxOpCompleteFailed = "Failed to mark UL transmit operation completed"
	LogSCACIBaseStationUnavailable   = "Base station unavailable"
	LogSCACIPreferenceLookupFailed   = "Failed to lookup preferred base station for endpoint"
	LogSCACIUsingPreferredBS         = "Using endpoint's last-attached base station preference"

	// ========================================================================
	// DL Queue Operations (12 constants)
	// ========================================================================

	LogSCACIUnmarshalDLDataQueueFailed     = "Failed to unmarshal DLDataQueue"
	LogSCACIEndpointNotFoundDLQueue        = "Endpoint not found for downlink queue"
	LogSCACICounterLengthMismatch          = "Counter-dependent length mismatch"
	LogSCACICntDependLengthMismatch        = "Counter-dependent length mismatch" // Alias
	LogSCACINonCntDependMultiPayload       = "Non-counter-dependent downlink has multiple userData entries"
	LogSCACIDuplicateQueueID               = "Duplicate queue ID detected"
	LogSCACIDuplicateQueIDDetected         = "Duplicate queue ID detected" // Alias
	LogSCACIInvalidDownlinkPayload         = "Invalid downlink payload"
	LogSCACIInvalidAcUUIDLength            = "Invalid AC UUID length"
	LogSCACIQueueIDOutOfRange              = "Queue ID exceeds maximum value"
	LogSCACIInvalidQueueIDFromDB           = "Invalid negative queue ID from database"
	LogSCACIEnqueueDownlinkFailed          = "Failed to enqueue downlink"
	LogSCACIUpdateDLStatusQueuedFailed     = "Failed to update downlink status to queued"
	LogSCACIUpdateDownlinkStatusQueued     = "Failed to update downlink status to queued" // Alias
	LogSCACIUpdateDownlinkStatusFailed     = "Failed to update downlink status to failed"
	LogSCACIDLDataQueueProcessed           = "DLDataQueue processed"
	LogSCACIDLDataQueueFailed              = "DLDataQueue failed" // Internal path error logging
	LogSCACIDLQueueHandshakeComplete       = "DLDataQueue handshake complete"
	LogSCACIDLDataQueueHandshakeComplete   = "DLDataQueue handshake complete" // Alias
	LogSCACIQueryDLQueueFailed             = "Failed to query downlink queue"
	LogSCACIDownlinkSchedulerNotConfigured = "Downlink scheduler not configured"
	LogSCACIDLQueueServiceInvoked          = "DL queue service invoked"

	// ========================================================================
	// DL Revoke Operations (11 constants)
	// ========================================================================

	LogSCACIProcessingDLDataRevoke          = "Processing DLDataRevoke"
	LogSCACIRevokingDownlinks               = "Revoking downlinks for endpoint"
	LogSCACIRevokingDownlinksForEndpoint    = "Revoking downlinks for endpoint" // Alias
	LogSCACINoPendingDownlinks              = "No pending downlinks to revoke"
	LogSCACINoPendingDownlinksToRevoke      = "No pending downlinks to revoke" // Alias
	LogSCACIRevokeDownlinkFailed            = "Failed to revoke downlink"
	LogSCACIRevokeDownlinkItemFailed        = "Failed to revoke downlink" // Alias (per-item)
	LogSCACIDLRevokeSuccessful              = "DL revoke successful"
	LogSCACIDownlinkRevocationComplete      = "Downlink revocation complete"
	LogSCACIDLDataRevokeInitiated           = "DLDataRevoke initiated"
	LogSCACIRecordDLRevokeOpFailed          = "Failed to record DLDataRevoke operation"
	LogSCACIRecordDLDataRevokeOpFailed      = "Failed to record DLDataRevoke operation" // Alias
	LogSCACIDLRevokeHandshakeComplete       = "DLDataRevoke handshake complete"
	LogSCACIDLDataRevokeHandshakeComplete   = "DLDataRevoke handshake complete" // Alias
	LogSCACIUnmarshalDLDataRevokeFailed     = "Failed to unmarshal DLDataRevoke"
	LogSCACIStorageNoRevokeSupport          = "Storage does not support RevokeDownlink"
	LogSCACIStorageNotAvailableRevoke       = "Storage not available for downlink revocation"
	LogSCACIStorageNotAvailableForRevoke    = "Storage not available for downlink revocation" // Alias
	LogSCACIDownlinkNotFoundForPacketCnt    = "Downlink not found for packet counter"
	LogSCACILookupDownlinkByPacketCntFailed = "Failed to lookup downlink by packet counter"
	LogSCACIQueueEntryNotFoundInBSSCI       = "Queue entry not found in BSSCI"
	LogSCACIBaseStationUnavailableRevoke    = "Base station unavailable"
	LogSCACIQueryDownlinkQueueFailed        = "Failed to query downlink queue"

	// ========================================================================
	// DL Result Operations (11 constants)
	// ========================================================================

	LogSCACIReceivedDLResultResponse       = "Received DL data result response from AC"
	LogSCACIReceivedDLDataResultResponse   = "Received DL data result response from AC" // Alias
	LogSCACIUpdateDLResultOpAckFailed      = "Failed to update DL result operation to acknowledged"
	LogSCACISendDLResultCompleteFailed     = "Failed to send DL result complete"
	LogSCACISendDLResultToACFailed         = "Failed to send DL result to AC"
	LogSCACIRecordDLResultOpFailed         = "Failed to record DL result operation"
	LogSCACIUpdateDLResultOpCompleteFailed = "Failed to update DL result operation to completed"
	LogSCACIUnexpectedDLResultComplete     = "Received unexpected DL result complete from AC (protocol violation)"
	LogSCACIDownlinkNotFoundPacketCounter  = "Downlink not found for packet counter"
	LogSCACILookupDownlinkByCounterFailed  = "Failed to lookup downlink by packet counter"
	LogSCACIUpdateDLStatusFailedFailed     = "Failed to update downlink status to failed"
	LogSCACIQueueEntryNotFoundBSSCI        = "Queue entry not found in BSSCI"

	// ========================================================================
	// Ping Operations (4 constants)
	// ========================================================================

	LogSCACIProcessingPing         = "Processing Ping"
	LogSCACIProcessingPingResponse = "Processing PingResponse from AC"
	LogSCACIPingHandshakeComplete  = "Ping handshake complete"
	LogSCACISendKeepaliveFailed    = "Failed to send keepalive ping"

	// ========================================================================
	// Connect/Ping Operation Recording (§3.3/§3.4 audit trail)
	// ========================================================================

	LogSCACIRecordConnectOpFailed    = "Failed to record connect operation"
	LogSCACIRecordConnectRspOpFailed = "Failed to record connect response state update"
	LogSCACIRecordConnectCmpOpFailed = "Failed to record connect complete state update"
	LogSCACIRecordPingOpFailed       = "Failed to record ping operation"
	LogSCACIRecordPingRspOpFailed    = "Failed to record ping response state update"
	LogSCACIRecordPingCmpOpFailed    = "Failed to record ping complete state update"

	// ========================================================================
	// Status Operations (10 constants)
	// ========================================================================

	LogSCACIProcessingStatus              = "Processing Status request"
	LogSCACIQueryBaseStationsStatusFailed = "Failed to query base stations for status"
	LogSCACIStatusResponsePrepared        = "Status response prepared"
	LogSCACIStatusHandshakeComplete       = "Status handshake complete"
	LogSCACIEPStatusHandshakeStub         = "EPStatus handshake complete (stub)"
	LogSCACIEPStatusHandshakeComplete     = "EPStatus handshake complete (stub)" // Alias
	LogSCACIRecordStatusOpFailed          = "Failed to record Status operation"
	LogSCACIRecordStatusRspOpFailed       = "Failed to record Status response state"
	LogSCACIRecordStatusCmpOpFailed       = "Failed to record Status complete state"
	LogSCACIStatusDependencyFailed        = "Status dependency query failed, returning degraded status"

	// ========================================================================
	// Shared Persistence Operations (9 constants)
	// ========================================================================

	LogSCACIPersistACOpIdFailed         = "Failed to persist AC opId"
	LogSCACIPersistSCOpIdFailed         = "Failed to persist SC opId"
	LogSCACIPersistOpIDsPairFailed      = "Failed to persist opId pair atomically"
	LogSCACIPersistHeartbeatFailed      = "Failed to persist session heartbeat"
	LogSCACIRecordOperationFailed       = "Failed to record operation"
	LogSCACIUpdateOperationStateFailed  = "Failed to update operation state"
	LogSCACILookupEndpointFailed        = "Failed to lookup endpoint"
	LogSCACIMarkOperationCompleteFailed = "Failed to mark operation completed"
	LogSCACIReceivedErrorAck            = "Received error acknowledgement"

	// ========================================================================
	// Pending Operation Replay (SCACI §1 Session Resumption)
	// ========================================================================

	LogSCACIGetPendingOpsFailed       = "Failed to get pending operations for replay"
	LogSCACIReplayingPendingOp        = "Replaying pending operation after session resume"
	LogSCACIReplayOpFailed            = "Failed to replay pending operation"
	LogSCACIUnknownReplayCommand      = "Unknown command type for replay, skipping"
	LogSCACISkipNonReplayable         = "Skipping non-replayable command"
	LogSCACICrossTenantReplayRejected = "Cross-tenant replay rejected"
	LogSCACIReplayUserDataCorrupted   = "Replay aborted due to corrupted userData"

	// ========================================================================
	// Assembly Validation (SCACI §§2.4, 2.5, 3.9.1, 3.12.1, 3.13.1)
	// ========================================================================

	LogSCACIDLResultValidationFailed = "SCACI DLDataResult validation failed"
	LogSCACIEPStatusValidationFailed = "SCACI EPStatus validation failed"

	// ========================================================================
	// EPStatus Broadcast & Lifecycle (SCACI §3.13)
	// ========================================================================

	LogSCACISendEPStatusFailed           = "Failed to send EPStatus to AC"
	LogSCACIRecordEPStatusOpFailed       = "Failed to record EPStatus operation"
	LogSCACINoActiveACsForTenant         = "No active ACs for tenant to broadcast EPStatus"
	LogSCACIEPStatusResponseReceived     = "Received EPStatus response from AC"
	LogSCACIOperationStateUpdateFailed   = "Failed to update operation state"
	LogSCACIReplayingEPStatus            = "Replaying EPStatus operation after session resume"
	LogSCACIReplayEPStatusInvalidData    = "Invalid data in stored EPStatus operation"
	LogSCACIReplayEPStatusFieldDecodeErr = "Failed to decode field in EPStatus replay"
	LogSCACIConnectRspValidationFailed   = "SCACI ConnectResponse validation failed"
	LogSCACIStatusRspValidationFailed    = "SCACI StatusResponse validation failed"
	LogSCACIULDataValidationFailed       = "SCACI ULData validation failed"
	LogSCACIULDataTxValidationFailed     = "SCACI ULDataTransmit validation failed" // §2.4/§3.9.1 mandatory field enforcement
	LogSCACIErrorMsgValidationFailed     = "SCACI Error message validation failed"
	LogSCACICommandMismatch              = "SCACI message command mismatch"
	LogSCACIRecordEventFailed            = "Failed to record SCACI error event"
	LogSCACISentErrorAck                 = "Sent error acknowledgment"
)
