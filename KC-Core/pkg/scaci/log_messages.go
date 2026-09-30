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
package scaci

// SCACI log message constants for structured logging
const (
	// ========================================================================
	// Server Lifecycle Messages
	// ========================================================================

	LogSCACIServerStopping = "Stopping SCACI server..."
	LogSCACIServerStopped  = "SCACI server stopped"

	// ========================================================================
	// TLS & Connection Messages
	// ========================================================================

	LogSCACIConnectionNotTLS         = "Connection is not TLS (should never happen)"
	LogSCACITLSHandshakeFailed       = "TLS handshake failed"
	LogSCACINoClientCertificate      = "No client certificate provided (mutual TLS required)"
	LogSCACICertificateMappingFailed = "Failed to map certificate to tenant"
	LogSCACIConnectionEstablished    = "SCACI connection established"
	LogSCACIConnectionClosed         = "SCACI connection closed"
	LogSCACICloseAfterWriteFailed    = "Closing SCACI connection after a failed frame write reported an error"
	LogSCACICloseConnectionFailed    = "Failed to close SCACI connection"

	LogSCACIConnectEstablishmentTimedOut  = "SCACI connection closed: the connect operation did not complete in time"
	LogSCACISupersededConnectionClosed    = "SCACI connection closed: a newer connection took over its application center session"
	LogSCACIMarkSessionDisconnectedFailed = "Failed to mark SCACI session disconnected"
	LogSCACIReconciledAbandonedSessions   = "SCACI sessions left live by a previous process are disconnected and resumable again"

	// ========================================================================
	// Tenant Mapping Messages
	// ========================================================================

	LogSCACIOrgResolverNotInjected    = "SCACI: org resolver not injected into handshake service"
	LogSCACICrossTenantResumeRejected = "SCACI: resume rejected due to certificate tenant mismatch"
	LogSCACIOrgEnforcementNilUUID     = "Organization enforcement enabled but session has nil org UUID"

	// ========================================================================
	// Certificate Validation Messages
	// ========================================================================

	LogSCACICertNotYetValid       = "Certificate not yet valid (NotBefore is in the future)"
	LogSCACICertExpired           = "Certificate expired (NotAfter is in the past)"
	LogSCACICertMissingClientAuth = "Certificate missing ClientAuth extended key usage"
	LogSCACICertInvalidSubject    = "Certificate has invalid subject (missing CN and Organization)"
	LogSCACICertValidationPassed  = "Certificate validation passed"

	// ========================================================================
	// Message Framing & Dispatch
	// ========================================================================

	LogSCACIReceivedMessage           = "Received SCACI message"
	LogSCACIReadFrameFailed           = "Failed to read frame"
	LogSCACIDecodeFrameFailed         = "Failed to decode frame payload"
	LogSCACIMissingCommandField       = "Missing or invalid 'command' field"
	LogSCACIMissingOpIDField          = "Missing or invalid 'opId' field"
	LogSCACIFirstMessageMustBeConnect = "First message must be Connect"
	LogSCACIHandlerError              = "Handler error"
	LogSCACIUnknownCommand            = "Unknown SCACI command"
	LogSCACIUnsupportedSublayerPrefix = "SCACI sublayer prefix not supported" // §4 sublayer guard

	// ========================================================================
	// Error Response Handling - SCACI §3.14
	// ========================================================================

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
	// Connect Operation
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
	LogSCACISessionCreateFailed     = "Session creation failed"
	LogSCACIConnectComplete         = "Connect complete - session active"
	LogSCACIUpdateSessionFailed     = "Failed to update session"
	LogSCACIPersistSessionFailed    = "Failed to persist session"
	LogSCACIConnectCmpNonZeroOpID   = "ConnectComplete with non-zero opId, terminating"

	// Reasons ResumeOpIDConflict gives for a resume whose operation IDs
	// contradict the stored counters (SCACI §3.3.1).
	resumeReasonFmtACOpIDUnknown     = "AC opId not known to the service center: required=%d, known=%d"
	resumeReasonFmtSCOpIDNeverIssued = "SC opId not issued by the service center: provided=%d, issued down to %d"

	// ========================================================================
	// Register Operation
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
	// BSSCI Attach Propagation Integration
	// BSSCI §5.8-5.8.3: Automatic attach propagation for preAttach endpoints
	// ========================================================================

	LogSCACITriggeringAttachPropagation = "Triggering attach propagation for preAttach endpoint"
	LogSCACIAttachPropagationErrors     = "Attach propagation encountered errors"

	// ========================================================================
	// Deregister Operation
	// ========================================================================

	LogSCACIDecodeDeregisterFailed       = "Failed to decode deregister payload"
	LogSCACIEndpointNotFoundDeregister   = "Endpoint not found for deregister"
	LogSCACIDatabaseErrorDeregister      = "Database error during deregister"
	LogSCACIRecordDeregisterOpFailed     = "Failed to record deregister operation"
	LogSCACIDetachEndpointFailed         = "Failed to detach endpoint"
	LogSCACIEndpointDeregistered         = "Endpoint deregistered"
	LogSCACIDeregisterHandshakeComplete  = "Deregister handshake complete"
	LogSCACILoadDeregisterOpFailed       = "Failed to load deregister operation"
	LogSCACIDetachPropagationErrors      = "Detach propagation had errors"
	LogSCACIUnexpectedRegisterComplete   = "regCmp for an operation that is not a reg answered with regRsp"
	LogSCACIUnexpectedDeregisterComplete = "deregCmp for an operation that is not a dereg answered with deregRsp"
	LogSCACIDetachPropagationSent        = "Detach propagation sent to all base stations"
	LogSCACIRevokeDownlinksFailed        = "Failed to revoke downlinks"
	LogSCACIDeregisterCleanupStart       = "Starting deregister cleanup"

	// ========================================================================
	// UL Data Operations
	// ========================================================================

	LogSCACISendULDataFailed           = "Failed to send UL data to AC session"
	LogSCACIUplinkAlreadyRecorded      = "Uplink already recorded for the AC session; its original operation stands"
	LogSCACIOperationLeftToResume      = "Recorded operation could not be sent; connection closed so the session resume reissues it"
	LogSCACIReceivedULDataResponse     = "Received UL data response from AC"
	LogSCACIUpdateULDataOpAckFailed    = "Failed to update UL data operation to acknowledged"
	LogSCACISendULDataCompleteFailed   = "Failed to send UL data complete"
	LogSCACIULDataHandshakeComplete    = "UL data handshake complete"
	LogSCACIMarkULDataOpCompleteFailed = "Failed to mark UL data operation completed"
	LogSCACIUnexpectedULDataCmp        = "Received unexpected ulDataCmp from AC"
	LogSCACIUnexpectedEPStatusCmp      = "Received unexpected epStatCmp from AC"
	LogSCACIUnsolicitedResponse        = "Received a response to no outstanding Service Center operation"
	LogSCACISendEPStatusCompleteFailed = "Failed to send EP status complete"
	LogSCACIUnexpectedULDataTxRsp      = "Received unexpected ulDataTxRsp from AC"

	// ========================================================================
	// UL Transmit Operations
	// ========================================================================

	LogSCACIDecodeULDataTxFailed     = "Failed to decode ulDataTx payload"
	LogSCACIBaseStationNotFoundULTx  = "Base station not found for UL transmit"
	LogSCACILookupBaseStationFailed  = "Failed to lookup base station"
	LogSCACIScheduleULTxFailed       = "Failed to schedule UL transmit"                            // Alias
	LogSCACIULDataTxNotSupported     = "UL data transmit not supported (scheduler not configured)" // Alias
	LogSCACIULDataTxScheduled        = "UL data transmit scheduled"                                // Alias
	LogSCACIProcessingULDataTxCmp    = "Processing ulDataTxCmp"
	LogSCACIMarkULTxAckFailed        = "Failed to mark UL transmit acknowledged"
	LogSCACISendULDataTxRspFailed    = "Failed to send ulDataTxRsp"
	LogSCACIRecordULTxOpFailed       = "Failed to record UL transmit operation"
	LogSCACIMarkULTxOpCompleteFailed = "Failed to mark UL transmit operation completed"
	LogSCACIPreferenceLookupFailed   = "Failed to lookup preferred base station for endpoint"
	LogSCACIUsingPreferredBS         = "Using endpoint's last-attached base station preference"

	// ========================================================================
	// DL Queue Operations
	// ========================================================================

	LogSCACIUnmarshalDLDataQueueFailed       = "Failed to unmarshal DLDataQueue"
	LogSCACICntDependLengthMismatch          = "Counter-dependent length mismatch" // Alias
	LogSCACINonCntDependMultiPayload         = "Non-counter-dependent downlink has multiple userData entries"
	LogSCACIDuplicateQueIDDetected           = "Duplicate queue ID detected" // Alias
	LogSCACIInvalidDownlinkPayload           = "Invalid downlink payload"
	LogSCACIDownlinkPayloadTooLarge          = "Downlink payload exceeds maximum size"
	LogSCACIInvalidAcUUIDLength              = "Invalid AC UUID length"
	LogSCACIInvalidQueueIDFromDB             = "Invalid negative queue ID from database"
	LogSCACIEnqueueDownlinkFailed            = "Failed to enqueue downlink"
	LogSCACIRecordEnqueuedEventFailed        = "Downlink queued without its event: the event could not be recorded"
	LogSCACIPersistDrainTimedOut             = "Session persistence drain timed out during shutdown"
	LogSCACIDownlinkOrgResolutionFailed      = "Failed to resolve downlink organization for tenant"
	LogSCACIDLDataQueueProcessed             = "DLDataQueue processed"
	LogSCACIDLDataQueueDeferred              = "DLDataQueue deferred to the endpoint's next downlink window: no connected bidirectional station serves it"
	LogSCACIDLDataQueueDispatchFailed        = "DLDataQueue persisted but not dispatched; it waits for the next downlink window"
	LogSCACIDLDataQueueFailed                = "DLDataQueue failed" // Internal path error logging
	LogSCACIDownlinkEndpointNotBidirectional = "DLDataQueue refused: the endpoint is not bidirectional"
	LogSCACIDLDataQueueHandshakeComplete     = "DLDataQueue handshake complete" // Alias
	LogSCACIDLQueueServiceInvoked            = "DL queue service invoked"

	// ========================================================================
	// DL Revoke Operations
	// ========================================================================

	LogSCACIProcessingDLDataRevoke          = "Processing DLDataRevoke"
	LogSCACIRevokingDownlinksForEndpoint    = "Revoking downlinks for endpoint" // Alias
	LogSCACINoPendingDownlinksToRevoke      = "No pending downlinks to revoke"  // Alias
	LogSCACIRevokeDownlinkFailed            = "Failed to revoke downlink"
	LogSCACIRevokeDownlinkItemFailed        = "Failed to revoke downlink" // Alias (per-item)
	LogSCACIDLRevokeSuccessful              = "DL revoke successful"
	LogSCACIDownlinkRevocationComplete      = "Downlink revocation complete"
	LogSCACIDLDataRevokeInitiated           = "DLDataRevoke initiated"
	LogSCACIRecordDLDataRevokeOpFailed      = "Failed to record DLDataRevoke operation" // Alias
	LogSCACIDLDataRevokeHandshakeComplete   = "DLDataRevoke handshake complete"         // Alias
	LogSCACIUnmarshalDLDataRevokeFailed     = "Failed to unmarshal DLDataRevoke"
	LogSCACIStorageNotAvailableRevoke       = "Storage not available for downlink revocation"
	LogSCACIDownlinkNotFoundForPacketCnt    = "No downlink is scheduled for the packet counter; the revoke has nothing to revoke"
	LogSCACILookupDownlinkByPacketCntFailed = "Failed to lookup downlink by packet counter"
	LogSCACIQueryDownlinkQueueFailed        = "Failed to query downlink queue"

	// ========================================================================
	// DL Result Operations
	// ========================================================================

	LogSCACIReceivedDLDataResultResponse         = "Received DL data result response from AC" // Alias
	LogSCACIUpdateDLResultOpAckFailed            = "Failed to update DL result operation to acknowledged"
	LogSCACISendDLResultCompleteFailed           = "Failed to send DL result complete"
	LogSCACISendDLResultToACFailed               = "Failed to send DL result to AC"
	LogSCACIDLResultNotQueuedByApplicationCenter = "DL result not sent to Application Centers: the downlink was not queued by one"
	LogSCACIDLResultQueuerUnknown                = "DL result not sent: the downlink names no Application Center that queued it"
	LogSCACIResumedSessionNotHeld                = "Resumed session is no longer held for resumption; closing the connection"
	LogSCACIConnectCompleteRefused               = "Connect operation cannot complete for its session; closing the connection"
	LogSCACIUpdateDLResultOpCompleteFailed       = "Failed to update DL result operation to completed"
	LogSCACIUnexpectedDLResultComplete           = "Received unexpected DL result complete from AC (protocol violation)"

	// ========================================================================
	// Ping Operations
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
	LogSCACIRecordSessionEventFailed = "Failed to record application center session event"
	LogSCACIRecordPingOpFailed       = "Failed to record ping operation"
	LogSCACIRecordPingRspOpFailed    = "Failed to record ping response state update"
	LogSCACIRecordPingCmpOpFailed    = "Failed to record ping complete state update"

	// ========================================================================
	// Status Operations
	// ========================================================================

	LogSCACIProcessingStatus          = "Processing Status request"
	LogSCACIStatusResponsePrepared    = "Status response prepared"
	LogSCACIStatusHandshakeComplete   = "Status handshake complete"
	LogSCACIEPStatusHandshakeComplete = "EPStatus handshake complete"
	LogSCACIRecordStatusOpFailed      = "Failed to record Status operation"
	LogSCACIRecordStatusRspOpFailed   = "Failed to record Status response state"
	LogSCACIRecordStatusCmpOpFailed   = "Failed to record Status complete state"
	LogSCACIStatusDependencyFailed    = "Status dependency query failed, returning degraded status"

	// ========================================================================
	// Shared Persistence Operations
	// ========================================================================

	LogSCACIPersistOpIDsPairFailed      = "Failed to persist opId pair atomically"
	LogSCACIPersistHeartbeatFailed      = "Failed to persist session heartbeat"
	LogSCACIRecordOperationFailed       = "Failed to record operation"
	LogSCACIUpdateOperationStateFailed  = "Failed to update operation state"
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
	LogSCACIEPStatusResponseReceived     = "Received EPStatus response from AC"
	LogSCACIOperationStateUpdateFailed   = "Failed to update operation state"
	LogSCACIReplayingEPStatus            = "Replaying EPStatus operation after session resume"
	LogSCACIReplayEPStatusInvalidData    = "Invalid data in stored EPStatus operation"
	LogSCACIReplayEPStatusFieldDecodeErr = "Failed to decode field in EPStatus replay"
	LogSCACIReplayULDataFieldDecodeErr   = "Failed to decode field in ulData replay"
	LogSCACIConnectRspValidationFailed   = "SCACI ConnectResponse validation failed"
	LogSCACIStatusRspValidationFailed    = "SCACI StatusResponse validation failed"
	LogSCACIULDataValidationFailed       = "SCACI ULData validation failed"
	LogSCACIULDataTxValidationFailed     = "SCACI ULDataTransmit validation failed" // §2.4/§3.9.1 mandatory field enforcement
	LogSCACIErrorMsgValidationFailed     = "SCACI Error message validation failed"
	LogSCACICommandMismatch              = "SCACI message command mismatch"
	LogSCACIRecordEventFailed            = "Failed to record SCACI error event"
	LogSCACISentErrorAck                 = "Sent error acknowledgment"
	// LogSCACICertOrgResolutionFallback is logged when certificate org resolution fails and community fallback applies.
	LogSCACICertOrgResolutionFallback = "Certificate org resolution failed, using community fallback"
	// LogSCACISoftwareVersionNotConfigured is logged when the ConnectResponse omits swVersion because none is configured.
	LogSCACISoftwareVersionNotConfigured = "Software version not configured - ConnectResponse will omit swVersion field"
	// LogSCACIUsingDevelopmentSoftwareVersion is logged when a development software version is used in the ConnectResponse.
	LogSCACIUsingDevelopmentSoftwareVersion = "Using development software version in ConnectResponse"
	// LogSCACISublayerHandlerInvoked is logged when a registered sublayer handler dispatches a message.
	LogSCACISublayerHandlerInvoked = "SCACI sublayer handler invoked"
)
