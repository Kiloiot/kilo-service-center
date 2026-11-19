// Package bssci implements the MIOTY Base Station Service Center Interface (BSSCI) v1.0.0
package bssci

// BSSCI Log Message Catalog
//
// Purpose: Centralized catalog of all BSSCI log messages for consistency and maintainability.
//
// Usage Pattern:
//   s.logger.Info(LogBSSCINewConnection, zap.String("epEui", FormatEUI64(eui)))
//
// Naming Convention:
//   - All constants use LogBSSCI* prefix for discoverability
//   - Organized into subsystems matching BSSCI protocol operations

const (
	// ========================================================================
	// Server Lifecycle & Connection Messages
	// ========================================================================

	// LogBSSCINewConnection indicates a new BSSCI client connection
	LogBSSCINewConnection = "new connection"
	// LogBSSCIReceivedMessageDebug logs raw BSSCI message content for debugging
	LogBSSCIReceivedMessageDebug = "received message debug"
	// LogBSSCIRejectingCommandBeforeHandshake logs when a command is rejected due to incomplete handshake
	LogBSSCIRejectingCommandBeforeHandshake = "Rejecting command before handshake complete"
	// LogBSSCIRejectingConnectMessageOnActiveSession logs a connect-operation message received after the handshake completed
	LogBSSCIRejectingConnectMessageOnActiveSession = "Rejecting connect-operation message on an active session"
	// LogBSSCIRejectingInboundServiceCenterCommand logs when a base station sends a service-center-initiated command
	LogBSSCIRejectingInboundServiceCenterCommand = "Rejecting inbound service-center-initiated command from base station"
	// LogBSSCIDLRXQueryExpirySweepFailed logs a failed dlRxStatQry expiry sweep
	LogBSSCIDLRXQueryExpirySweepFailed = "DL RX status query expiry sweep failed"
	// LogBSSCIDLRXQueriesExpired logs the count of expired dlRxStatQry queries
	LogBSSCIDLRXQueriesExpired = "Expired stale DL RX status queries"
	// LogBSSCIConnectHandshakeNotComplete logs when connect handshake is incomplete
	LogBSSCIConnectHandshakeNotComplete = "Connect handshake not complete for session"
	// LogBSSCIConnectHandshakeNotCompleteDL logs when connect handshake is incomplete for downlink
	LogBSSCIConnectHandshakeNotCompleteDL = "Connect handshake not complete for downlink operation"
	// LogBSSCIStartingNewSession indicates a new BSSCI session is starting
	LogBSSCIStartingNewSession = "Starting new session"
	// LogBSSCIResumingPreviousSession indicates a previous BSSCI session is being resumed
	LogBSSCIResumingPreviousSession = "Resuming previous session"
	// LogBSSCIResumingSessionWithPendingOps indicates session resume with pending operations
	LogBSSCIResumingSessionWithPendingOps = "Resuming session with pending operations"
	// LogBSSCITerminatedStaleSession indicates a stale session was terminated before creating new session
	LogBSSCITerminatedStaleSession = "Terminated stale session"
	// LogBSSCIFailedToTerminateStaleSession indicates failure to terminate stale session
	LogBSSCIFailedToTerminateStaleSession = "Failed to terminate stale session"
	// LogBSSCIFailedToTerminateResumableSessions indicates failure to retire the base station's leftover resumable sessions
	LogBSSCIFailedToTerminateResumableSessions = "Failed to terminate leftover resumable sessions"
	// LogBSSCIDisplacedLiveSessionForBaseStation indicates a newly activated session displaced a live session of the same base station
	LogBSSCIDisplacedLiveSessionForBaseStation = "Displaced live session for base station"
	// LogBSSCIFailedToCloseDisplacedSessionConnection indicates failure to close the displaced session's connection
	LogBSSCIFailedToCloseDisplacedSessionConnection = "Failed to close displaced session connection"
	// LogBSSCIResumeAlreadyClaimed is logged when the resumable session was activated or retired by another connection before this one could claim it
	LogBSSCIResumeAlreadyClaimed = "Resume rejected: session already claimed by another connection"
	// LogBSSCIFailedToTerminateSession indicates failure to terminate session on disconnect
	LogBSSCIFailedToTerminateSession = "Failed to terminate session"
	// LogBSSCIFailedToMarkSessionDisconnected indicates a lost connection's session could not be handed back resumable
	LogBSSCIFailedToMarkSessionDisconnected = "Failed to mark session disconnected"
	// LogBSSCIFailedToDeletePendingOperations indicates failure to delete pending operations
	LogBSSCIFailedToDeletePendingOperations = "Failed to delete pending operations"
	// LogBSSCIDeletedPendingOperations indicates pending operations were successfully deleted
	LogBSSCIDeletedPendingOperations = "Deleted pending operations"
	// LogBSSCIInvalidEncodingInDatabase indicates invalid encoding value stored in database
	LogBSSCIInvalidEncodingInDatabase = "Invalid encoding in database"
	// LogBSSCINoPendingOperationsToResume indicates no pending operations during resume
	LogBSSCINoPendingOperationsToResume = "No pending operations to resume"
	// LogBSSCIProcessingPendingOperationForResume indicates processing pending operation
	LogBSSCIProcessingPendingOperationForResume = "Processing pending operation for resume"
	// LogBSSCIAttachPropagateDebug logs attach propagate debugging information
	LogBSSCIAttachPropagateDebug = "Attach propagate debug: repetition flag set"
	// LogBSSCIStartedStatusMechanismForBaseStation indicates status mechanism started
	LogBSSCIStartedStatusMechanismForBaseStation = "Started status mechanism for base station"
	// LogBSSCISendingPingToBaseStation indicates ping being sent to base station
	LogBSSCISendingPingToBaseStation = "Sending ping to base station"
	// LogBSSCIReceivedPingResponse is a log message constant
	LogBSSCIReceivedPingResponse = "Received ping response"
	// LogBSSCIStationEventNotRecorded is logged when a protocol exchange could not be put in the station's activity
	LogBSSCIStationEventNotRecorded = "Failed to record the base station activity event"
	// LogBSSCIOperationIDBackwards is a log message constant
	LogBSSCIOperationIDBackwards = "Operation ID backwards"
	// LogBSSCIOperationNotOpen is logged when a completion names no open base station operation
	LogBSSCIOperationNotOpen = "Completion does not name an open base station operation"
	// LogBSSCIOperationIDNotPositive is a log message constant
	LogBSSCIOperationIDNotPositive = "Operation ID not positive for BS-initiated operation"
	// LogBSSCIOperationIDValidationFailed is a log message constant
	LogBSSCIOperationIDValidationFailed = "Operation ID validation failed"
	// LogBSSCISCOperationIDMustNotIncrease is a log message constant
	LogBSSCISCOperationIDMustNotIncrease = "SC operation ID must not increase"
	// LogBSSCIOpIDFieldNotFound is a log message constant
	LogBSSCIOpIDFieldNotFound = "opId field not found in message"
	// LogBSSCIPayloadTooLarge is a log message constant
	LogBSSCIPayloadTooLarge = "payload too large"
	// LogBSSCIVersionIncompatible is a log message constant
	LogBSSCIVersionIncompatible = "Version incompatible"
	// LogBSSCIReceivedErrorAckFromBaseStation is a log message constant
	LogBSSCIReceivedErrorAckFromBaseStation = "Received errorAck from base station"
	// LogBSSCIFailedToResolveOwnerOrgForDLRxQuery is a log message constant
	LogBSSCIFailedToResolveOwnerOrgForDLRxQuery = "Failed to resolve endpoint owner org for DL RX query tracking"
	// LogBSSCICertIdentityDefaultOrgLookupFailed is a log message constant
	LogBSSCICertIdentityDefaultOrgLookupFailed = "Failed to resolve default organization for certificate identity"
	// LogBSSCICertIdentityRejectedStrictMode is a log message constant
	LogBSSCICertIdentityRejectedStrictMode = "Certificate identity resolution failed; closing connection (org enforcement enabled)"
	// LogBSSCICertFingerprintMismatch is a log message constant
	LogBSSCICertFingerprintMismatch = "Presented certificate does not match the registered fingerprint"
	// LogBSSCICertFingerprintPinned is logged when a registered station's blank fingerprint is pinned to its certificate
	LogBSSCICertFingerprintPinned = "Pinned the registered base station's certificate fingerprint"
	// LogBSSCICertExpiryRecorded is logged when a bound certificate's expiry is stored for its station
	LogBSSCICertExpiryRecorded = "Recorded the base station certificate's expiry"
	// LogBSSCICertExpiryBackfillFailed is logged when a bound certificate's expiry could not be stored
	LogBSSCICertExpiryBackfillFailed = "Failed to record the base station certificate's expiry"
	// LogBSSCIStationCertificateRefused is logged when a connecting station's client certificate does not belong to it
	LogBSSCIStationCertificateRefused = "Base station refused: its client certificate does not belong to the station it claims"
	// LogBSSCICertSubjectEUIMismatch is a log message constant
	LogBSSCICertSubjectEUIMismatch = "Certificate subject EUI does not match the connect bsEui"
	// LogBSSCIUnsolicitedErrorAck is a log message constant
	LogBSSCIUnsolicitedErrorAck = "Ignoring errorAck with no matching sent error"
	// LogBSSCIClosingConnectionAfterWriteFailure is a log message constant
	LogBSSCIClosingConnectionAfterWriteFailure = "Closing connection after ambiguous frame write; pending operations preserved for resume"
	// LogBSSCISendingBSSCIError is a log message constant
	LogBSSCISendingBSSCIError = "Sending BSSCI error"
	// LogBSSCIPersistedPendingOperation is a log message constant
	LogBSSCIPersistedPendingOperation = "Persisted pending operation"
	// LogBSSCIUpdatedPendingOperationMetadata is a log message constant
	LogBSSCIUpdatedPendingOperationMetadata = "Updated pending operation metadata"

	// ========================================================================
	// Attach Operations
	// ========================================================================

	// LogBSSCIAttachOperationCompletedSuccessfully is a log message constant
	LogBSSCIAttachOperationCompletedSuccessfully = "Attach operation completed successfully"
	// LogBSSCIReceivedAttCmpWithoutPendingAttach is a log message constant
	LogBSSCIReceivedAttCmpWithoutPendingAttach = "Received attCmp without pending attach operation" //nolint:gosec // G101: false positive - attCmp is BSSCI protocol command
	// LogBSSCISendingAttachPropagate is a log message constant
	LogBSSCISendingAttachPropagate = "Sending attach propagate"
	// LogBSSCISendingAttachPropagateComplete is a log message constant
	LogBSSCISendingAttachPropagateComplete = "Sending attach propagate complete"
	// LogBSSCINoConnectedBaseStationsForAttachPropagate is a log message constant
	LogBSSCINoConnectedBaseStationsForAttachPropagate = "No connected base stations available for attach propagate"
	// LogBSSCIAttachCounterReplay is a log message constant for replay attack detection
	LogBSSCIAttachCounterReplay = "Attach counter replay detected - incoming value not monotonic"
	// LogBSSCIUpdatedEndpointWithAttachPropagateInfo is a log message constant
	LogBSSCIUpdatedEndpointWithAttachPropagateInfo = "Updated endpoint with attach propagate info"
	// LogBSSCIFailedToSendAttachPropagateComplete is a log message constant
	LogBSSCIFailedToSendAttachPropagateComplete = "Failed to send attach propagate complete"
	// LogBSSCIFailedToSendDetachPropagateComplete is the symmetric token used
	// when the Service Center cannot deliver the detach-propagate completion
	// message to the base station.
	LogBSSCIFailedToSendDetachPropagateComplete = "Failed to send detach propagate complete"
	// LogBSSCIFailedToCreateCompletionEvent is a log message constant
	LogBSSCIFailedToCreateCompletionEvent = "Failed to create attach propagate completion event"
	// LogBSSCIFailedToPersistAttachPropagateComplete is a log message constant
	LogBSSCIFailedToPersistAttachPropagateComplete = "Failed to persist attach propagate complete"

	// ========================================================================
	// Detach Operations
	// ========================================================================

	// LogBSSCIReceivedDetCmpWithoutPendingDetach is a log message constant
	LogBSSCIReceivedDetCmpWithoutPendingDetach = "Received detCmp without pending detach operation" //nolint:gosec // G101: false positive - detCmp is BSSCI protocol command
	// LogBSSCISendingDetachPropagate is a log message constant
	LogBSSCISendingDetachPropagate = "Sending detach propagate"
	// LogBSSCISendingDetachPropagateComplete is a log message constant
	LogBSSCISendingDetachPropagateComplete = "Sending detach propagate complete"
	// LogBSSCINoConnectedBaseStationsForDetachPropagate is a log message constant
	LogBSSCINoConnectedBaseStationsForDetachPropagate = "No connected base stations available for detach propagate"
	// LogBSSCIUpdatedEndpointWithDetachInfo is a log message constant
	LogBSSCIUpdatedEndpointWithDetachInfo = "Updated endpoint with detach info"
	// LogBSSCIFailedToPersistDetachMessage is a log message constant
	LogBSSCIFailedToPersistDetachMessage = "Failed to persist detach message to mioty_messages table"

	// ========================================================================
	// Uplink Data Operations
	// ========================================================================

	// LogBSSCIReceivedULDataCompletion is a log message constant
	LogBSSCIReceivedULDataCompletion = "Received UL data completion"
	// LogBSSCIReceivedStringUserData is a log message constant
	LogBSSCIReceivedStringUserData = "Received string userData, skipping (expected byte array or Numeric[])"
	// LogBSSCIUnknownUserDataType is a log message constant
	LogBSSCIUnknownUserDataType = "Unknown userData type, using empty payload"
	// LogBSSCIUnsupportedUserDataElementType is a log message constant
	LogBSSCIUnsupportedUserDataElementType = "Unsupported userData element type, using zero"
	// LogBSSCIFailedToResolveEndpointTenant is a log message constant
	LogBSSCIFailedToResolveEndpointTenant = "Failed to resolve endpoint tenant for UL data"

	// ========================================================================
	// Uplink Transmit Operations
	// ========================================================================

	// LogBSSCIBaseStationDoesNotSupportBidi is a log message constant
	LogBSSCIBaseStationDoesNotSupportBidi = "Base station does not support bidirectional communication"
	// LogBSSCIBaseStationTenantMismatch is a log message constant
	LogBSSCIBaseStationTenantMismatch = "base station tenant mismatch - cannot schedule UL transmit"
	// LogBSSCISendingULDataTransmit is a log message constant
	LogBSSCISendingULDataTransmit = "Sending UL data transmit"
	// LogBSSCISendingULDataTransmitComplete is a log message constant
	LogBSSCISendingULDataTransmitComplete = "Sending UL data transmit complete"
	// LogBSSCIReceivedULDataTransmitResponse is a log message constant
	LogBSSCIReceivedULDataTransmitResponse = "Received UL data transmit response"
	// LogBSSCIReceivedULDataTransmitCompletion is a log message constant
	LogBSSCIReceivedULDataTransmitCompletion = "Received UL data transmit completion"
	// LogBSSCIULDataTransmitThreeWayHandshakeCompleted is a log message constant
	LogBSSCIULDataTransmitThreeWayHandshakeCompleted = "UL data transmit three-way handshake completed"
	// LogBSSCIULDataTransmitResponseResult is a log message constant
	LogBSSCIULDataTransmitResponseResult = "UL data transmit response result"
	// LogBSSCINoPendingOperationForULTransmitComplete is a log message constant
	LogBSSCINoPendingOperationForULTransmitComplete = "No pending operation found for UL data transmit completion"
	// LogBSSCIUserDataMissingFromBothSources is a log message constant
	LogBSSCIUserDataMissingFromBothSources = "userData missing from both metadata and PendingOperation, using empty data"
	// LogBSSCIUserDataMissingFromMetadata is a log message constant
	LogBSSCIUserDataMissingFromMetadata = "userData missing from metadata, using PendingOperation.Data"
	// LogBSSCIFailedToEncryptNetworkSessionKey is a log message constant
	LogBSSCIFailedToEncryptNetworkSessionKey = "Failed to encrypt network session key"
	// LogBSSCIFailedToPersistULDataTxOperation is a log message constant
	LogBSSCIFailedToPersistULDataTxOperation = "Failed to persist ulDataTx operation"
	// LogBSSCIFailedToRemovePendingULTransmitOperation is a log message constant
	LogBSSCIFailedToRemovePendingULTransmitOperation = "Failed to remove pending UL transmit operation from database"

	// ========================================================================
	// Downlink Queue Operations
	// ========================================================================

	// LogBSSCIReceivedDLDataQueRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataQueRspFromBaseStation = "Received dlDataQueRsp from base station"
	// LogBSSCIReceivedDLRxStatFromBaseStation is a log message constant
	LogBSSCIReceivedDLRxStatFromBaseStation = "Received dlRxStat from base station"
	// LogBSSCIReceivedDLRxStatRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLRxStatRspFromBaseStation = "Received dlRxStatRsp from base station"
	// LogBSSCIReceivedDLRxStatQryRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLRxStatQryRspFromBaseStation = "Received dlRxStatQryRsp from base station"
	// LogBSSCIDLRxStatusOperationCompleted is a log message constant
	LogBSSCIDLRxStatusOperationCompleted = "DL RX status operation completed"
	// LogBSSCISentDLRxStatQryToBaseStation is a log message constant
	LogBSSCISentDLRxStatQryToBaseStation = "Sent dlRxStatQry to base station"
	// LogBSSCIPersistedDLRxStatus is a log message constant
	LogBSSCIPersistedDLRxStatus = "Persisted DL RX status"
	// LogBSSCIReceivedDLDataRevRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataRevRspFromBaseStation = "Received dlDataRevRsp from base station"
	// LogBSSCISentDLDataRevToBaseStation is a log message constant
	LogBSSCISentDLDataRevToBaseStation = "Sent dlDataRev to base station"
	// LogBSSCIInvalidQueueIDInRevokeResponse is a log message constant
	LogBSSCIInvalidQueueIDInRevokeResponse = "Invalid queue ID in revoke response"
	// LogBSSCICannotResolveTenantForRevoke is a log message constant
	LogBSSCICannotResolveTenantForRevoke = "Cannot resolve tenant for revoke"
	// LogBSSCIInvalidTenantIDFormat is a log message constant
	LogBSSCIInvalidTenantIDFormat = "Invalid tenant ID format"
	// LogBSSCIUpdatedDownlinkResult is a log message constant
	LogBSSCIUpdatedDownlinkResult = "Updated downlink result"
	// LogBSSCIFailedToUpdateDownlinkResult is a log message constant
	LogBSSCIFailedToUpdateDownlinkResult = "Failed to update downlink result"
	// LogBSSCIFailedToUpdateDownlinkAsRevoked is a log message constant
	LogBSSCIFailedToUpdateDownlinkAsRevoked = "Failed to update downlink as revoked"
	// LogBSSCIFailedToPersistDLDataQueOperation is a log message constant
	LogBSSCIFailedToPersistDLDataQueOperation = "Failed to persist dlDataQue operation"
	// LogBSSCIFailedToPersistDLDataRevOperation is a log message constant
	LogBSSCIFailedToPersistDLDataRevOperation = "Failed to persist dlDataRev operation"
	// LogBSSCIFailedToPersistDLRxStatQryOperation is a log message constant
	LogBSSCIFailedToPersistDLRxStatQryOperation = "Failed to persist dlRxStatQry operation"
	// LogBSSCIFailedToPersistDLRxStatus is a log message constant
	LogBSSCIFailedToPersistDLRxStatus = "Failed to persist DL RX status"
	// LogBSSCIMessageStoreNotAvailableForDLRxStatus is a log message constant
	LogBSSCIMessageStoreNotAvailableForDLRxStatus = "Message store not available for DL RX status persistence"
	// LogBSSCIFailedToRecordDLDataQueueAcknowledgedEvent is a log message constant
	LogBSSCIFailedToRecordDLDataQueueAcknowledgedEvent = "Failed to record dl data queue acknowledged event"
	// LogBSSCIFailedToRecordDLDataResultEvent is a log message constant
	LogBSSCIFailedToRecordDLDataResultEvent = "Failed to record dl data result event"
	// LogBSSCIFailedToRecordDLDataRevokedEvent is a log message constant
	LogBSSCIFailedToRecordDLDataRevokedEvent = "Failed to record dl data revoked event"
	// LogBSSCIFailedToRecordDLDataRevokeInitiatedEvent is a log message constant
	LogBSSCIFailedToRecordDLDataRevokeInitiatedEvent = "Failed to record dl data revoke initiated event"
	// LogBSSCIFailedToRecordDLRxStatusEvent is a log message constant
	LogBSSCIFailedToRecordDLRxStatusEvent = "Failed to record dl rx status event"
	// LogBSSCIFailedToPersistDLRxStatQueryTracking is a log message constant (downlink_handlers.go:~1315)
	LogBSSCIFailedToPersistDLRxStatQueryTracking = "Failed to persist DL RX status query tracking"
	// LogBSSCIFailedToGetPendingOperation is a log message constant (BSSCI §§5.11-5.12.3 Gap 1)
	LogBSSCIFailedToGetPendingOperation = "Failed to get pending operation from StatusService"
	// LogBSSCIFailedToRemovePendingOperationFromDB is a log message constant
	LogBSSCIFailedToRemovePendingOperationFromDB = "Failed to remove pending operation from database"
	// LogBSSCIFailedToRemovePendingOperationAfterSendFailure is a log message constant
	LogBSSCIFailedToRemovePendingOperationAfterSendFailure = "Failed to remove pending operation from database after send failure"
	// LogBSSCIFailedToSendErrorFrame is a log message constant
	LogBSSCIFailedToSendErrorFrame = "failed to send error frame"
	// LogBSSCIFailedToBroadcastDLResultToSCACI is a log message constant
	LogBSSCIFailedToBroadcastDLResultToSCACI = "Failed to broadcast DL result to SCACI"
	// LogBSSCICannotResolveTenantForDownlinkResult is a log message constant
	LogBSSCICannotResolveTenantForDownlinkResult = "Cannot resolve tenant for downlink result"
	// LogBSSCIQueueIDOutOfRange is a log message constant
	LogBSSCIQueueIDOutOfRange = "Queue ID out of range for int64 conversion"
	// LogBSSCIUpdatedBaseStationBidiCapability is a log message constant
	LogBSSCIUpdatedBaseStationBidiCapability = "Updated base station bidi capability"
	// LogBSSCIFailedToUpdateBaseStationBidiCapability is a log message constant
	LogBSSCIFailedToUpdateBaseStationBidiCapability = "Failed to update base station bidi capability"
	// LogBSSCIProcessingRevokeResponse is logged when processing revoke response from base station
	LogBSSCIProcessingRevokeResponse = "Processing DL Data Revoke Response from base station"

	// ========================================================================
	// Downlink Result Operations
	// ========================================================================

	// LogBSSCIReceivedDLDataResFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataResFromBaseStation = "Received dlDataRes from base station"
	// LogBSSCIReceivedDLDataResRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataResRspFromBaseStation = "Received dlDataResRsp from base station"
	// LogBSSCIDLDataResultOperationCompleted is a log message constant
	LogBSSCIDLDataResultOperationCompleted = "DL data result operation completed"
	// LogBSSCIUnexpectedFieldsInDLDataResRsp is logged when dlDataResRsp contains non-canonical fields
	LogBSSCIUnexpectedFieldsInDLDataResRsp = "Unexpected fields in dlDataResRsp - spec violation"
	// LogBSSCIMissingCommandFieldInResponse is logged when command field missing from response
	LogBSSCIMissingCommandFieldInResponse = "Missing command field in response message"
	// LogBSSCIMissingOpIDFieldInResponse is logged when opId field missing from response
	LogBSSCIMissingOpIDFieldInResponse = "Missing opId field in response message"

	// LogBSSCISessionNotFound is logged when base station session not found for DL operations
	LogBSSCISessionNotFound = "Base station session not found"
	// LogBSSCISessionNotReady is logged when session handshake not complete
	LogBSSCISessionNotReady = "Base station session not ready"
	// LogBSSCISessionNotBidirectional is logged when session does not support bidirectional operations
	LogBSSCISessionNotBidirectional = "Base station session not bidirectional"

	// ========================================================================
	// Status Operations
	// ========================================================================

	// LogBSSCISendingStatusRequestToBaseStation is a log message constant
	LogBSSCISendingStatusRequestToBaseStation = "Sending status request to base station"
	// LogBSSCIReceivedStatusRspFromBaseStation is a log message constant
	LogBSSCIReceivedStatusRspFromBaseStation = "Received statusRsp from base station"
	// LogBSSCIStatusMechanismAlreadyRunningForSession is a log message constant
	LogBSSCIStatusMechanismAlreadyRunningForSession = "Status mechanism already running for session"
	// LogBSSCIStoppingStatusMechanism is a log message constant
	LogBSSCIStoppingStatusMechanism = "Stopping status mechanism"
	// LogBSSCIInitialStatusRequestFailed is a log message constant
	LogBSSCIInitialStatusRequestFailed = "Initial status request failed"
	// LogBSSCIStatusRequestFailedInPeriodicLoop is a log message constant
	LogBSSCIStatusRequestFailedInPeriodicLoop = "Status request failed in periodic loop"
	// LogBSSCIUpdatedBaseStationStatusSuccessfully is a log message constant
	LogBSSCIUpdatedBaseStationStatusSuccessfully = "Updated base station status successfully"
	// LogBSSCIFailedToGetBaseStationByEUI is a log message constant
	LogBSSCIFailedToGetBaseStationByEUI = "Failed to get base station by EUI"
	// LogBSSCIFailedToPersistPendingStatusOperation is a log message constant
	LogBSSCIFailedToPersistPendingStatusOperation = "Failed to persist pending status operation"
	// LogBSSCIFailedToUpdateBaseStationStatus is a log message constant
	LogBSSCIFailedToUpdateBaseStationStatus = "Failed to update base station status"
	// LogBSSCIFailedToPersistStatusHistory is a log message constant
	LogBSSCIFailedToPersistStatusHistory = "Failed to persist status history"
	// LogBSSCIFailedToSendStatusRequest is a log message constant
	LogBSSCIFailedToSendStatusRequest = "Failed to send status request"
	// LogBSSCIFailedToCleanupPendingOpAfterSendFailure is a log message constant
	LogBSSCIFailedToCleanupPendingOpAfterSendFailure = "Failed to cleanup pending operation after send failure"

	// ========================================================================
	// VM Handler Operations
	// ========================================================================

	// LogBSSCIVMStatusReceived is a log message constant
	LogBSSCIVMStatusReceived = "VM status received"
	// LogBSSCIVMStatusResponseMissingMacTypesField is a log message constant
	LogBSSCIVMStatusResponseMissingMacTypesField = "VM status response missing macTypes field"
	// LogBSSCIVMActivateSucceeded is a log message constant
	LogBSSCIVMActivateSucceeded = "VM activate succeeded"
	// LogBSSCIVMActivateFailed is a log message constant
	LogBSSCIVMActivateFailed = "VM activate failed"

	// ========================================================================
	// Additional Server Lifecycle constants
	// ========================================================================

	// LogBSSCIFailedToReadFrame is logged when a frame cannot be read from the connection.
	LogBSSCIFailedToReadFrame = "failed to read frame"
	// LogBSSCIInvalidProtocolIdentifier is a log message constant
	LogBSSCIInvalidProtocolIdentifier = "invalid protocol identifier"
	// LogBSSCIFailedToDecodeMessagePack is a log message constant
	LogBSSCIFailedToDecodeMessagePack = "failed to decode MessagePack"
	// LogBSSCIMissingCommandField is a log message constant
	LogBSSCIMissingCommandField = "missing command field"
	// LogBSSCIInvalidOpIDType is a log message constant
	LogBSSCIInvalidOpIDType = "Invalid opId type"
	// LogBSSCIMinorVersionMismatch is a log message constant
	LogBSSCIMinorVersionMismatch = "Minor version mismatch"
	// LogBSSCIMinorVersionNegotiatedDown is logged when a base station requests a newer minor version and the session continues at the service center's selected version (BSSCI §5.3.2)
	LogBSSCIMinorVersionNegotiatedDown = "Minor version newer than service center, negotiating down"
	// LogBSSCIBaseStationConnected is a log message constant
	LogBSSCIBaseStationConnected = "Base Station connected"
	// LogBSSCIBaseStationNotFoundInDatabase is a log message constant
	LogBSSCIBaseStationNotFoundInDatabase = "Base Station not found in database - rejecting connection"
	// LogBSSCIBaseStationConnectionCompletedSuccessfully is a log message constant
	LogBSSCIBaseStationConnectionCompletedSuccessfully = "Base Station connection completed successfully"
	// LogBSSCIBaseStationDisconnectedStatusOffline is a log message constant
	LogBSSCIBaseStationDisconnectedStatusOffline = "Base Station disconnected, status updated to offline"
	// LogBSSCIFailedToUpdateOfflineStatus is a log message constant
	LogBSSCIFailedToUpdateOfflineStatus = "Failed to update offline status"
	// LogBSSCIFailedToHandleMessage is a log message constant
	LogBSSCIFailedToHandleMessage = "failed to handle message"
	// LogBSSCIInvalidConnectOperationID is a log message constant
	LogBSSCIInvalidConnectOperationID = "Invalid connect operation ID"

	// LogBSSCIStationlessPropagationOwnerUnresolved is logged when a propagation no base station can receive names an endpoint whose owner cannot be resolved, so its failure event is recorded for no tenant.
	LogBSSCIStationlessPropagationOwnerUnresolved = "Propagation without connected base stations: endpoint owner not resolved, failure event not recorded"
	// LogBSSCIEndPointAttachRequestWithFullTelemetry is a log message constant
	LogBSSCIEndPointAttachRequestWithFullTelemetry = "End Point attach request with full telemetry"
	// LogBSSCIEndPointDetachRequestWithTelemetry is a log message constant
	LogBSSCIEndPointDetachRequestWithTelemetry = "End Point detach request with telemetry"
	// LogBSSCIEndpointAttachedToBaseStation is a log message constant
	LogBSSCIEndpointAttachedToBaseStation = "Endpoint attached to base station"
	// LogBSSCIEndpointDetachedFromBaseStation is a log message constant
	LogBSSCIEndpointDetachedFromBaseStation = "Endpoint detached from base station"
	// LogBSSCIEndpointNotProvisionedForAttach is a log message constant
	LogBSSCIEndpointNotProvisionedForAttach = "Endpoint not provisioned for attach"
	// LogBSSCIEndpointNotFoundForDetachCompletion is a log message constant
	LogBSSCIEndpointNotFoundForDetachCompletion = "Endpoint not found for detach completion"
	// LogBSSCIEndpointNotFoundInDatabaseForAttachPropagate is a log message constant
	LogBSSCIEndpointNotFoundInDatabaseForAttachPropagate = "Endpoint not found in database for attach propagate"
	// LogBSSCIEndpointNotFoundInDatabaseForDetachPropagate is a log message constant
	LogBSSCIEndpointNotFoundInDatabaseForDetachPropagate = "Endpoint not found in database for detach propagate"
	// LogBSSCIInvalidNetworkKeyLengthForAttach is a log message constant
	LogBSSCIInvalidNetworkKeyLengthForAttach = "Invalid network key length for attach"
	// LogBSSCIAttachPropagateResponseReceived is a log message constant
	LogBSSCIAttachPropagateResponseReceived = "Attach propagate response received"
	// LogBSSCIAttachPropagateAcceptedByBaseStation is a log message constant
	LogBSSCIAttachPropagateAcceptedByBaseStation = "Attach propagate accepted by base station"
	// LogBSSCIAttachPropagateRejectedByBaseStation is a log message constant
	LogBSSCIAttachPropagateRejectedByBaseStation = "Attach propagate rejected by base station"
	// LogBSSCIAttachPropagateCompleted is a log message constant
	LogBSSCIAttachPropagateCompleted = "Attach propagate completed"
	// LogBSSCICannotAttachPropagateToNonBidiBaseStation is a log message constant
	LogBSSCICannotAttachPropagateToNonBidiBaseStation = "Cannot attach propagate to non-bidirectional base station for bidirectional endpoint"
	// LogBSSCIDetachPropagateResponseReceived is a log message constant
	LogBSSCIDetachPropagateResponseReceived = "Detach propagate response received"
	// LogBSSCIDetachPropagateAcceptedByBaseStation is a log message constant
	LogBSSCIDetachPropagateAcceptedByBaseStation = "Detach propagate accepted by base station"
	// LogBSSCIDetachPropagateRejectedByBaseStation is a log message constant
	LogBSSCIDetachPropagateRejectedByBaseStation = "Detach propagate rejected by base station"
	// LogBSSCIDetachPropagateCompleted is a log message constant
	LogBSSCIDetachPropagateCompleted = "Detach propagate completed"
	// LogBSSCIFailedToPersistDetachPropagateComplete is logged when detPrpCmp persistence fails
	LogBSSCIFailedToPersistDetachPropagateComplete = "Failed to persist detach propagate complete message"
	// LogBSSCIDownlinkWindowOpen is a log message constant
	LogBSSCIDownlinkWindowOpen = "Downlink window open"
	// LogBSSCIDownlinkDispatched is logged when auto-dispatch successfully sends a downlink
	LogBSSCIDownlinkDispatched = "Downlink auto-dispatched on dlOpen"
	// LogBSSCIDownlinkDispatchError is logged when auto-dispatch fails
	LogBSSCIDownlinkDispatchError = "Downlink auto-dispatch error"
	// LogDispatcherNoTenant is logged when dispatcher skips due to missing tenant context
	LogDispatcherNoTenant = "Downlink dispatcher skipping: no tenant context"
	// LogBSSCIServingStationUnavailable is logged when the station serving an endpoint cannot take a downlink now
	LogBSSCIServingStationUnavailable = "Downlink deferred: the base station serving the endpoint is not connected or not bidirectional"
	// LogDispatcherDiscardedQueueReclaimed is logged when a fresh session returns the station's queued downlinks to pending
	LogDispatcherDiscardedQueueReclaimed = "Downlinks queued at a base station returned to pending: its new session discarded them"
	// LogDispatcherDiscardedRevocationsExpired is logged when a fresh session ends expired the overdue downlinks its station was asked to drop
	LogDispatcherDiscardedRevocationsExpired = "Overdue downlinks a base station was asked to drop expired: its new session discarded them"
	// LogDispatcherEndpointQueueReclaimed is logged when an attach propagate returns the endpoint's downlinks a station held to pending
	LogDispatcherEndpointQueueReclaimed = "Endpoint downlinks queued at a base station returned to pending: its attach propagate discarded them"
	// LogDispatcherWindowAlreadyClaimed is logged when another reception already holds the telegram's downlink window
	LogDispatcherWindowAlreadyClaimed = "Downlink dispatcher skipping: another reception holds the telegram's downlink window"
	// LogDispatcherWindowClaimFailed is logged when the telegram's downlink window could not be claimed
	LogDispatcherWindowClaimFailed = "Downlink dispatcher could not claim the telegram's downlink window"
	// LogDispatcherWindowReleaseFailed is logged when an unused downlink window could not be given back
	LogDispatcherWindowReleaseFailed = "Downlink dispatcher could not release the telegram's unused downlink window"
	// LogDispatcherStationNotBidirectional is logged when a downlink window arrives through a unidirectional base station
	LogDispatcherStationNotBidirectional = "Downlink dispatcher skipping: the base station is not bidirectional"
	// LogDispatcherNoOrganization is logged when dispatch is refused because the
	// owner organization is unresolved; dispatch fails closed rather than
	// selecting a queue row without an organization scope.
	LogDispatcherNoOrganization = "Downlink dispatcher skipping: no organization context"
	// LogDispatcherOrgMismatch is logged when a reserved row does not belong to
	// the organization the dispatch ran for.
	LogDispatcherOrgMismatch = "Downlink dispatcher aborting: reserved row not owned by requesting organization"
	// LogDispatcherQueryFailed is logged when pending downlink query fails
	LogDispatcherQueryFailed = "Downlink dispatcher: query failed"
	// LogDispatcherNoPending is logged when no pending downlinks found (debug level)
	LogDispatcherNoPending = "Downlink dispatcher: no pending downlinks"
	// LogDispatcherSendFailed is logged when SendDLDataQueue fails
	LogDispatcherSendFailed = "Downlink dispatcher: send failed"
	// LogDispatcherMarkSentFailed is logged when marking downlink as queued fails
	LogDispatcherMarkSentFailed = "Downlink dispatcher: mark queued failed"
	// LogDispatcherReleaseFailed is logged when releasing a reservation back to pending fails
	LogDispatcherReleaseFailed = "Downlink dispatcher: failed to release reservation to pending"
	// LogBSSCIFailedToConfirmDownlinkQueued is a log message constant
	LogBSSCIFailedToConfirmDownlinkQueued = "Failed to confirm downlink queue row as queued after dlDataQueRsp"
	// LogBSSCIFailedToRevokeRefusedDownlink is logged when a base station's refusal of a dlDataRev cannot be recorded
	LogBSSCIFailedToRevokeRefusedDownlink = "Failed to record a base station's refusal of a dlDataRev"
	// LogBSSCIRevokeRefusedByStation is logged when a base station answers a dlDataRev with error
	LogBSSCIRevokeRefusedByStation = "Base station refused a dlDataRev"
	// LogBSSCIRevokeRefusalKeepsDownlinkInFlight is logged when a refusal's code does not say the station lacks the downlink
	LogBSSCIRevokeRefusalKeepsDownlinkInFlight = "dlDataRev refusal does not say the base station lacks the downlink; it stays in flight until a result or the station's next session"
	// LogBSSCIFailedToListStationRevocations is logged when a connected station cannot be asked again to drop its overdue downlinks
	LogBSSCIFailedToListStationRevocations = "Failed to list the overdue downlinks a connected base station is asked to drop"
	// LogBSSCIFailedToResendRevocation is logged when an overdue downlink cannot be revoked again at its reconnected station
	LogBSSCIFailedToResendRevocation = "Failed to ask a reconnected base station again to drop an overdue downlink"
	// LogBSSCIRevokeAlreadyInFlight is logged when a downlink is not asked for again because its dlDataRev awaits the station's answer
	LogBSSCIRevokeAlreadyInFlight = "dlDataRev not repeated: the base station has not answered the one in flight"
	// LogBSSCIFailedToFailRejectedDownlink is logged when a downlink its base station rejected cannot be failed
	LogBSSCIFailedToFailRejectedDownlink = "Failed to mark a downlink its base station rejected as failed"
	// LogBSSCIQueueOwnerUnresolved is logged when no tenant owns the downlink a dlDataQue answer refers to
	LogBSSCIQueueOwnerUnresolved = "Cannot resolve the tenant owning a queued downlink"
	// LogDispatcherSuccess is logged when downlink successfully dispatched
	LogDispatcherSuccess = "Downlink dispatcher: success"
	// LogDispatcherReservationsReclaimed is logged when a base station's orphaned reservations return to pending
	LogDispatcherReservationsReclaimed = "Downlink dispatcher: returned orphaned base station reservations to pending"
	// LogBSSCIFailedToReclaimReservations is logged when a base station's orphaned reservations could not be released
	LogBSSCIFailedToReclaimReservations = "Failed to return orphaned base station downlink reservations to pending"
	// LogBSSCIFailedToRecordEndpointAck is logged when an uplink's dlAck could not be recorded on its downlink
	LogBSSCIFailedToRecordEndpointAck = "Failed to record the endpoint acknowledgement of a downlink"
	// LogBSSCIDownlinkAcknowledgedByEndpoint is logged when an uplink's dlAck acknowledges a transmitted downlink
	LogBSSCIDownlinkAcknowledgedByEndpoint = "Downlink acknowledged by the endpoint"
	// LogBSSCIEndpointAckWithoutDownlink is logged when an uplink's dlAck matches no unacknowledged transmitted downlink
	LogBSSCIEndpointAckWithoutDownlink = "Endpoint acknowledgement matches no transmitted downlink"
	// LogBSSCIFailedToRecordDownlinkAckEvent is logged when the event of a downlink its endpoint acknowledged could not be recorded
	LogBSSCIFailedToRecordDownlinkAckEvent = "Failed to record the event of a downlink acknowledged by the endpoint"
	// LogBSSCIServingStationUnknown is logged when a downlink waits because no base station heard or attached its endpoint yet
	LogBSSCIServingStationUnknown = "No base station heard or attached the endpoint yet; the downlink waits for its next downlink window"
	// LogBSSCIResultForFinishedDownlink is logged when a dlDataRes names a downlink that already ended, whose outcome stays
	LogBSSCIResultForFinishedDownlink = "Result for a downlink that already ended leaves its outcome unchanged"
	// LogBSSCISentResultForExpiredDownlink is logged when a station reports a downlink sent that was already reported expired
	LogBSSCISentResultForExpiredDownlink = "Base station reported a downlink sent after it was reported expired; the expiry stands"
	// LogBSSCIRevokeAnswerForDownlinkNotHeld is logged when a station answers a revoke of a downlink it no longer holds
	LogBSSCIRevokeAnswerForDownlinkNotHeld = "Revoke answered for a downlink that already ended or another base station holds"
	// LogBSSCIFailedToReclaimDiscardedQueue is logged when the downlinks a fresh session discarded could not be released
	LogBSSCIFailedToReclaimDiscardedQueue = "Failed to return the downlinks a base station discarded to pending"
	// LogBSSCIFailedToListPendingDownlinks is logged when a connected station cannot be sent the downlinks it serves
	LogBSSCIFailedToListPendingDownlinks = "Failed to list the pending downlinks for a connected base station"
	// LogBSSCIServingStationLookupFailedOnConnect is logged when a connected station's claim to serve an endpoint cannot be decided
	LogBSSCIServingStationLookupFailedOnConnect = "Failed to decide whether the connected base station serves the endpoint of a pending downlink"
	// LogBSSCIFailedToDispatchOnConnect is logged when a pending downlink could not be sent to the connected station that serves it
	LogBSSCIFailedToDispatchOnConnect = "Failed to send a pending downlink to the connected base station that serves its endpoint"
	// LogBSSCIFailedToReclaimEndpointQueue is logged when the downlinks an attach propagate discarded could not be released
	LogBSSCIFailedToReclaimEndpointQueue = "Failed to return the endpoint's downlinks a base station discarded on attach propagate to pending"

	// LogBSSCIErrorFinalizedOperation is logged when a base station error finalizes a tracked service center operation
	LogBSSCIErrorFinalizedOperation = "Base station error finalized the service center operation"
	// LogBSSCIBaseStationReportedError is a log message constant
	LogBSSCIBaseStationReportedError = "Base Station reported error"
	// LogBSSCIDatabaseSessionCreated is a log message constant
	LogBSSCIDatabaseSessionCreated = "Database session created"
	// LogBSSCIReconciledAbandonedSessions logs sessions a previous process left active that were made resumable
	LogBSSCIReconciledAbandonedSessions = "Sessions left active by a previous process are resumable again"
	// LogBSSCIDatabaseSessionUpdated is a log message constant
	LogBSSCIDatabaseSessionUpdated = "Database session updated"
	// LogBSSCIDatabaseNotAvailableForPendingOpPersistence is a log message constant
	LogBSSCIDatabaseNotAvailableForPendingOpPersistence = "Database not available or no session ID for pending operation persistence"
	// LogBSSCIDatabaseNotAvailableForPendingOpsLoad is a log message constant
	LogBSSCIDatabaseNotAvailableForPendingOpsLoad = "Database not available or no session ID for pending operations load"
	// LogBSSCIDatabaseNotAvailableForPendingOpUpdate is a log message constant
	LogBSSCIDatabaseNotAvailableForPendingOpUpdate = "Database not available or no session ID for pending operation update"
	// LogBSSCILoadedPendingOperationsFromDatabase is a log message constant
	LogBSSCILoadedPendingOperationsFromDatabase = "Loaded pending operations from database"
	// LogBSSCILoadedUserDataFromMetadataForULDataTx is a log message constant
	LogBSSCILoadedUserDataFromMetadataForULDataTx = "Loaded userData from metadata for ulDataTx operation"
	// LogBSSCIFailedToUpdateDatabaseSession is a log message constant
	LogBSSCIFailedToUpdateDatabaseSession = "Failed to update database session"
	// LogBSSCIFailedToSendErrorMessageToBaseStation is a log message constant
	LogBSSCIFailedToSendErrorMessageToBaseStation = "Failed to send error message to base station"
	// LogBSSCIFailedToSendErrorAck is a log message constant
	LogBSSCIFailedToSendErrorAck = "Failed to send errorAck"
	// LogBSSCIFailedToForwardULDataToSCACI is a log message constant
	LogBSSCIFailedToForwardULDataToSCACI = "Failed to forward UL data to SCACI"
	// LogBSSCIFailedToPersistPendingOperationMigrationNeeded is a log message constant
	LogBSSCIFailedToPersistPendingOperationMigrationNeeded = "Failed to persist pending operation (migration 050 may be needed)"
	// LogBSSCIFailedToPersistPendingOperation is a log message constant
	LogBSSCIFailedToPersistPendingOperation = "Failed to persist pending operation"
	// LogBSSCIFailedToQueryPendingOperationsFromDatabase is a log message constant
	LogBSSCIFailedToQueryPendingOperationsFromDatabase = "Failed to query pending operations from database"
	// LogBSSCIFailedToLoadPendingOperationsForSessionResume is a log message constant
	LogBSSCIFailedToLoadPendingOperationsForSessionResume = "Failed to load pending operations for session resume"
	// LogBSSCIFailedToUnmarshalOperationData is a log message constant
	LogBSSCIFailedToUnmarshalOperationData = "Failed to unmarshal operation data"
	// LogBSSCIFailedToUnmarshalMetadata is a log message constant
	LogBSSCIFailedToUnmarshalMetadata = "Failed to unmarshal metadata"
	// LogBSSCIFailedToReconstitutePendingOperation indicates a persisted
	// operation could not be semantically rebuilt; the resume is rejected.
	LogBSSCIFailedToReconstitutePendingOperation = "Failed to reconstitute pending operation, rejecting resume"
	// LogBSSCIFailedToReissuePendingOperation is a log message constant
	LogBSSCIFailedToReissuePendingOperation = "Failed to reissue pending operation"
	// LogBSSCIFailedToRemovePendingOperation is a log message constant
	LogBSSCIFailedToRemovePendingOperation = "Failed to remove pending operation"
	// LogBSSCIFailedToRemovePendingOperationAfterErrorAck is a log message constant
	LogBSSCIFailedToRemovePendingOperationAfterErrorAck = "Failed to remove pending operation after errorAck"
	// LogBSSCIFailedToPersistFailureMetadata is a log message constant
	LogBSSCIFailedToPersistFailureMetadata = "Failed to persist failure metadata"
	// LogBSSCIFailedToUpdateEndpointWithDetachInfo is a log message constant
	LogBSSCIFailedToUpdateEndpointWithDetachInfo = "Failed to update endpoint with detach info"
	// LogBSSCIFailedToUpdateEndpointAttachmentState is a log message constant
	LogBSSCIFailedToUpdateEndpointAttachmentState = "Failed to update endpoint attachment state"
	// LogBSSCIFailedToUpdateEndpointAttachMetadata is a log message constant
	LogBSSCIFailedToUpdateEndpointAttachMetadata = "Failed to update endpoint attach metadata"
	// LogBSSCIFailedToRestartPacketCounter is logged when an over-the-air attach cannot restart the endpoint packet counter.
	LogBSSCIFailedToRestartPacketCounter = "Failed to restart endpoint packet counter"
	// LogBSSCIFailedToLockAttachCounter is logged when an over-the-air attach cannot lock the endpoint's attach counter.
	LogBSSCIFailedToLockAttachCounter = "Failed to lock endpoint attach counter"
	// LogBSSCIFailedToUpdateEndpointDetachState is a log message constant
	LogBSSCIFailedToUpdateEndpointDetachState = "Failed to update endpoint detach state"
	// LogBSSCIFailedToUpdateConnectionStatus is a log message constant
	LogBSSCIFailedToUpdateConnectionStatus = "Failed to update connection status"
	// LogBSSCIFailedToUpdateLastSeen is a log message constant
	LogBSSCIFailedToUpdateLastSeen = "Failed to update last seen"
	// LogBSSCIFailedToExtractSnBsUUID is a log message constant
	LogBSSCIFailedToExtractSnBsUUID = "Failed to extract snBsUuid"
	// LogBSSCIFailedToCreateFailureEvent is a log message constant
	LogBSSCIFailedToCreateFailureEvent = "Failed to create failure event"
	// LogBSSCIFailedToCreateBaseStationFailureEvent is a log message constant
	LogBSSCIFailedToCreateBaseStationFailureEvent = "Failed to create base station failure event"
	// LogBSSCIFailedToCreateNonBidiAttachFailureEvent is a log message constant
	LogBSSCIFailedToCreateNonBidiAttachFailureEvent = "Failed to create non-bidirectional attach failure event"
	// LogBSSCIFailedToCreateAttachEvent is a log message constant
	LogBSSCIFailedToCreateAttachEvent = "Failed to create attach event"
	// LogBSSCIFailedToCreateAttachFailedEvent is a log message constant
	LogBSSCIFailedToCreateAttachFailedEvent = "Failed to create attach failed event"
	// LogBSSCIFailedToCreateAttachmentEvent is a log message constant
	LogBSSCIFailedToCreateAttachmentEvent = "Failed to create attachment event"
	// LogBSSCIFailedToCreateDetachEvent is a log message constant
	LogBSSCIFailedToCreateDetachEvent = "Failed to create detach event"
	// LogBSSCIFailedToCreateDetachFailedEvent is a log message constant
	LogBSSCIFailedToCreateDetachFailedEvent = "Failed to create detach failed event"
	// LogBSSCIFailedToCreateDetachmentEvent is a log message constant
	LogBSSCIFailedToCreateDetachmentEvent = "Failed to create detachment event"
	// LogBSSCIFailedToClearPersistedPendingOperation is a log message constant
	LogBSSCIFailedToClearPersistedPendingOperation = "Failed to clear persisted pending operation"
	// LogBSSCIFailedToRecordDroppedPendingOperation is a log message constant
	LogBSSCIFailedToRecordDroppedPendingOperation = "Failed to record dropped pending operation event"
	// LogBSSCIFailedToUpdateSessionCounters is a log message constant
	LogBSSCIFailedToUpdateSessionCounters = "Failed to update session counters"

	// LogBSSCIDetachOperationCompletedSuccessfully indicates successful detach operation completion
	LogBSSCIDetachOperationCompletedSuccessfully = "Detach operation completed successfully"
	// LogBSSCIFailedToRemovePendingOperationFromDatabase indicates database cleanup failure
	LogBSSCIFailedToRemovePendingOperationFromDatabase = "Failed to remove pending operation from database"
	// LogBSSCIFailedToRecordVMActivateSuccessEvent is a log message constant
	LogBSSCIFailedToRecordVMActivateSuccessEvent = "Failed to record VM activate success event"
	// LogBSSCIFailedToRecordVMActivateFailureEvent is a log message constant
	LogBSSCIFailedToRecordVMActivateFailureEvent = "Failed to record VM activate failure event"
	// LogBSSCIFailedToRemovePendingVMOperation is a log message constant
	LogBSSCIFailedToRemovePendingVMOperation = "Failed to remove pending operation from database"
	// LogBSSCIVMActivateOperationCompleted is a log message constant
	LogBSSCIVMActivateOperationCompleted = "VM activate operation completed"
	// LogBSSCIVMDeactivateSucceeded is a log message constant
	LogBSSCIVMDeactivateSucceeded = "VM deactivate succeeded"
	// LogBSSCIVMDeactivateFailed is a log message constant
	LogBSSCIVMDeactivateFailed = "VM deactivate failed"
	// LogBSSCIVMDeactivateOperationCompleted is a log message constant
	LogBSSCIVMDeactivateOperationCompleted = "VM deactivate operation completed"
	// LogBSSCIFailedToRecordVMDeactivateSuccessEvent is a log message constant
	LogBSSCIFailedToRecordVMDeactivateSuccessEvent = "Failed to record VM deactivate success event"
	// LogBSSCIFailedToRemovePendingVMOpAfterSendFailure is a log message constant
	LogBSSCIFailedToRemovePendingVMOpAfterSendFailure = "Failed to remove pending operation from database after send failure"
	// LogBSSCIFailedToUpdatePendingOperationMetadata is a log message constant
	LogBSSCIFailedToUpdatePendingOperationMetadata = "Failed to update pending operation metadata"
	// LogBSSCIFailedToRecordVMStatusEvent is a log message constant
	LogBSSCIFailedToRecordVMStatusEvent = "Failed to record VM status event"
	// LogBSSCIFailedToPersistVMActivateOperation is a log message constant
	LogBSSCIFailedToPersistVMActivateOperation = "Failed to persist VM activate operation"
	// LogBSSCISentVMActivateCommand is a log message constant
	LogBSSCISentVMActivateCommand = "Sent VM activate command"
	// LogBSSCIFailedToPersistVMDeactivateOperation is a log message constant
	LogBSSCIFailedToPersistVMDeactivateOperation = "Failed to persist VM deactivate operation"
	// LogBSSCISentVMDeactivateCommand is a log message constant
	LogBSSCISentVMDeactivateCommand = "Sent VM deactivate command"
	// LogBSSCIFailedToPersistVMStatusOperation is a log message constant
	LogBSSCIFailedToPersistVMStatusOperation = "Failed to persist VM status operation"
	// LogBSSCISentVMStatusRequest is a log message constant
	LogBSSCISentVMStatusRequest = "Sent VM status request"
	// LogBSSCIFailedToPersistVMDownlinkDataOperation is a log message constant
	LogBSSCIFailedToPersistVMDownlinkDataOperation = "Failed to persist VM downlink data operation"
	// LogBSSCISentVMDownlinkData is a log message constant
	LogBSSCISentVMDownlinkData = "Sent VM downlink data"

	// ========================================================================
	// Type Conversion & Field Validation
	// ========================================================================

	// LogBSSCIIntegerOverflowInMacTypeParsing is logged when an integer overflow occurs during MAC type parsing
	LogBSSCIIntegerOverflowInMacTypeParsing = "Integer overflow in macType parsing"

	// ========================================================================
	// Multi-Tenant Roaming & Organization Resolution
	// ========================================================================

	// LogBSSCIMissingTenantInMetadata is logged when tenant ID is missing from pending operation metadata
	LogBSSCIMissingTenantInMetadata = "Missing tenantId in pending operation metadata, falling back to session tenant"
	// LogBSSCIInvalidAttachPropagateRecordField is logged when an attach propagate recovery record field is missing or out of range
	LogBSSCIInvalidAttachPropagateRecordField = "Attach propagate recovery record field is missing or out of range"
	// LogBSSCIEndpointNotFoundForPropagate is logged when endpoint cannot be found for attach propagate completion
	LogBSSCIEndpointNotFoundForPropagate = "Endpoint not found for attach propagate completion - skipping DB updates"
	// LogBSSCIOrgLookupFailed is logged when default organization lookup fails for a tenant during attach propagate
	LogBSSCIOrgLookupFailed = "Failed to lookup default organization for tenant during attach propagate"
	// LogBSSCIEndpointNotFound is logged when endpoint lookup fails during tenant resolution
	LogBSSCIEndpointNotFound = "Endpoint not found during tenant resolution - falling back to session tenant"
	// LogBSSCIResolvedRoamingEndpointTenant is logged when endpoint tenant is resolved via roaming lookup during downlink operations
	LogBSSCIResolvedRoamingEndpointTenant = "Resolved roaming endpoint tenant via database lookup"
	// LogBSSCIEUIPrecisionLoss is logged when EUI value exceeds float64 safe integer range (>2^53) during type conversion
	LogBSSCIEUIPrecisionLoss = "EUI precision loss - value exceeds float64 safe integer range"
	// LogBSSCINumericPrecisionLoss is logged when a non-EUI numeric field value exceeds the exact integer range of its wire float representation
	LogBSSCINumericPrecisionLoss = "Numeric precision loss - value exceeds exact float integer range"

	// ========================================================================
	// Propagation Reconciliation
	// ========================================================================

	// LogBSSCIFailedToPersistAttachPropagateMessage is logged when attach propagate message persistence fails
	LogBSSCIFailedToPersistAttachPropagateMessage = "Failed to persist attach propagate message"
	// LogBSSCIFailedToDecideOverTheAirAttach is logged when a completed over-the-air attach could not be recorded
	LogBSSCIFailedToDecideOverTheAirAttach = "Failed to record the over-the-air attach of an endpoint"
	// LogBSSCIAutomaticPropagationFailedAfterOTAAttach is logged when automatic propagation fails after OTA attach completion
	LogBSSCIAutomaticPropagationFailedAfterOTAAttach = "Automatic propagation failed after OTA attach"
	// LogBSSCIDetachPropagationFailedAfterOTADetach is logged when a station cannot be told to drop an endpoint that detached over the air elsewhere
	LogBSSCIDetachPropagationFailedAfterOTADetach = "Detach propagation failed after OTA detach"
	// LogBSSCIBaseStationReconciliationFailed is logged when base station reconciliation fails after handshake
	LogBSSCIBaseStationReconciliationFailed = "Base station reconciliation failed after handshake"

	// ========================================================================
	// Outbound Message Validation (BSSCI §2.5)
	// ========================================================================

	// LogBSSCIOutboundValidationFailed is logged when outbound message validation fails
	LogBSSCIOutboundValidationFailed = "bssci.outbound.validation.failed"
	// LogBSSCIOutboundDisallowedField is logged when outbound message contains disallowed field
	LogBSSCIOutboundDisallowedField = "bssci.outbound.disallowed_field"
	// LogBSSCIOutboundMissingMandatoryFieldText is logged when outbound message missing mandatory field
	LogBSSCIOutboundMissingMandatoryFieldText = "bssci.outbound.missing_mandatory"

	// ========================================================================
	// SCACI EPStatus Forwarding (SCACI §3.13)
	// ========================================================================

	// LogBSSCIEPStatusForwardFailed is logged when EPStatus forwarding to SCACI fails after attach/detach
	LogBSSCIEPStatusForwardFailed = "Failed to forward EPStatus to SCACI after attach/detach completion"

	// ========================================================================
	// Endpoint Attachment Service
	// ========================================================================

	// ========================================================================
	// MQTT Event Publishing
	// ========================================================================

	// LogBSSCIFailedToPublishDLResultToMQTT is logged when MQTT downlink result event publish fails
	LogBSSCIFailedToPublishDLResultToMQTT = "Failed to publish DL result event to MQTT"
	// LogBSSCIMQTTPublishSkippedOrgUnresolved is logged when MQTT publish is skipped due to unresolved organization
	LogBSSCIMQTTPublishSkippedOrgUnresolved = "MQTT publish skipped: organization unresolved"

	// MQTTEventKeyDownlinkResult identifies downlink result events in structured log fields
	MQTTEventKeyDownlinkResult = "downlink_result"

	// Service-layer Propagation Operations live in internal/services/bssci/propagation_service.go.

	// LogBSSCISkippingPropagationDueToTenantMismatch is logged when ATT-03 filters out a session
	// that belongs to a different tenant than the endpoint owner.
	LogBSSCISkippingPropagationDueToTenantMismatch = "Skipping propagation due to tenant mismatch"
	// LogBSSCIFailedToPropagateToSession is logged when a per-session propagate send fails.
	LogBSSCIFailedToPropagateToSession = "Failed to propagate to session"
	// LogBSSCIEndpointPropagationCompleted summarizes a TriggerEndpointPropagate fan-out.
	LogBSSCIEndpointPropagationCompleted = "Endpoint propagation completed"
	// LogBSSCIStartingBaseStationReconciliation marks the start of reconcile-on-connect.
	LogBSSCIStartingBaseStationReconciliation = "Starting base station reconciliation"
	// LogBSSCIReconciliationPropagateFailed is logged when a reconciliation per-endpoint send fails.
	LogBSSCIReconciliationPropagateFailed = "Reconciliation propagate failed"
	// LogBSSCIBaseStationReconciliationCompleted summarizes a reconcile-on-connect run.
	LogBSSCIBaseStationReconciliationCompleted = "Base station reconciliation completed"

	// Service-layer Uplink Ingest Pipeline lives in internal/services/bssci/uplink_ingest_service.go.

	// LogBSSCIUplinkPacketCounterCollision is logged when a packet counter repeats inside the window with different content.
	LogBSSCIUplinkPacketCounterCollision = "Uplink packet counter collision"
	// LogBSSCIDuplicateUplinkReceived is logged when dedup classifies an uplink as a duplicate.
	LogBSSCIDuplicateUplinkReceived = "Duplicate uplink received"
	// LogBSSCIUplinkFirstReception is logged on the first reception of a unique uplink.
	LogBSSCIUplinkFirstReception = "Uplink first reception"
	// LogBSSCIEndpointNotFoundDuringIngestTenantResolution is logged when tenant resolution can't find the endpoint.
	LogBSSCIEndpointNotFoundDuringIngestTenantResolution = "Endpoint not found during ingest tenant resolution"
	// LogBSSCIRoamingDetectionFailedDuringIngest is logged when the roaming service errors during tenant resolution.
	LogBSSCIRoamingDetectionFailedDuringIngest = "Roaming detection failed during ingest"
	// LogBSSCIRoamingEndpointUplink is logged when ingest classifies an uplink as a roaming reception.
	LogBSSCIRoamingEndpointUplink = "Roaming endpoint uplink"
	// LogBSSCIRoamingNotAllowed is logged when a detected roam is rejected because
	// the owning and serving tenants have no roaming agreement.
	LogBSSCIRoamingNotAllowed = "Roaming not allowed"
	// LogBSSCIFailedToBeginAttachPropagateTransaction prefixes attach-propagate
	// persistence failures that occur before any work is done.
	LogBSSCIFailedToBeginAttachPropagateTransaction = "failed to begin attach propagate transaction"
	// LogBSSCIFailedToCommitAttachPropagateTransaction prefixes attach-propagate
	// persistence failures where the work succeeded but the commit did not.
	LogBSSCIFailedToCommitAttachPropagateTransaction = "failed to commit attach propagate transaction"
	// LogBSSCIFailedToResolveOrganizationForUplink is logged when org lookup fails for an uplink.
	LogBSSCIFailedToResolveOrganizationForUplink = "Failed to resolve organization for uplink"
	// LogBSSCIBlueprintResolutionFailed is logged when blueprint resolution fails during ingest.
	LogBSSCIBlueprintResolutionFailed = "Blueprint resolution failed"
	// LogBSSCIBlueprintDecodeError is logged when the blueprint decoder returns an error.
	LogBSSCIBlueprintDecodeError = "Blueprint decode error"
	// LogBSSCIFailedToFetchEndpointForBlueprintDecode is logged when the endpoint lookup for blueprint decode fails.
	LogBSSCIFailedToFetchEndpointForBlueprintDecode = "Failed to fetch endpoint for blueprint decode"
	// LogBSSCIFailedToFetchDLRXStatus is logged when DL RX status fetch fails during ingest.
	LogBSSCIFailedToFetchDLRXStatus = "Failed to fetch DL RX status"
	// LogBSSCIFailedToPersistUplinkMessage is logged when the uplink store rejects a persist.
	LogBSSCIFailedToPersistUplinkMessage = "Failed to persist uplink message"
	// LogBSSCIMQTTUplinkPublishSkippedOrgUnresolved is logged when MQTT publish is skipped due to unresolved owner-org UUID.
	LogBSSCIMQTTUplinkPublishSkippedOrgUnresolved = "MQTT uplink publish skipped: org unresolved"

	// ========================================================================
	// Server core (KC-Core/pkg/bssci/server.go)
	// ========================================================================

	// LogBSSCIAttachSignatureValidationFailed is logged when attach signature validation failed.
	LogBSSCIAttachSignatureValidationFailed = "Attach signature validation failed"
	// LogBSSCICertOrgResolutionFailedUsingCommunityFallback is logged when BSSCI cert org resolution failed, using community fallback.
	LogBSSCICertOrgResolutionFailedUsingCommunityFallback = "BSSCI cert org resolution failed, using community fallback"
	// LogBSSCICertOrgResolutionSucceeded is logged when BSSCI cert org resolution succeeded.
	LogBSSCICertOrgResolutionSucceeded = "BSSCI cert org resolution succeeded"
	// LogBSSCISessionNoPeerCertUsingDefaults is logged when BSSCI session no peer cert, using defaults.
	LogBSSCISessionNoPeerCertUsingDefaults = "BSSCI session no peer cert, using defaults"
	// LogBSSCISessionOrgTenantResolved is logged when BSSCI session org/tenant resolved.
	LogBSSCISessionOrgTenantResolved = "BSSCI session org/tenant resolved"
	// LogBSSCIClosingRetiredStationSession is logged when the session of a base station whose EUI changed or that was deleted is closed.
	LogBSSCIClosingRetiredStationSession = "Closing the BSSCI session of a base station whose EUI changed or that was deleted"
	// LogBSSCIDetachFromUnknownEndpoint is logged when detach from unknown endpoint.
	LogBSSCIDetachFromUnknownEndpoint = "Detach from unknown endpoint"
	// LogBSSCIDetachOwnerLookupFailed is logged when the endpoint owner lookup fails on a detach.
	LogBSSCIDetachOwnerLookupFailed = "Detach owner lookup failed"
	// LogBSSCIAttachOwnerLookupFailed is logged when the endpoint owner lookup fails on an attach.
	LogBSSCIAttachOwnerLookupFailed = "Attach owner lookup failed"
	// LogBSSCIPropagateOwnerLookupFailed is logged when the endpoint owner lookup fails before an attach or detach propagate.
	LogBSSCIPropagateOwnerLookupFailed = "Propagate owner lookup failed"
	// LogBSSCIDetachSignatureValidationFailed is logged when detach signature validation failed.
	LogBSSCIDetachSignatureValidationFailed = "Detach signature validation failed"
	// LogBSSCIDetachValidatorNotConfiguredUsingSessionTenantForUnknownEndpoint is logged when detach validator not configured - using session tenant for unknown endpoint.
	LogBSSCIDetachValidatorNotConfiguredUsingSessionTenantForUnknownEndpoint = "Detach validator not configured - using session tenant for unknown endpoint"
	// LogBSSCIDetectedMessageEncoding is logged when detected message encoding.
	LogBSSCIDetectedMessageEncoding = "Detected message encoding"
	// LogBSSCIDispositionResolutionFailedRejectingUplink is logged when disposition resolution failed; rejecting uplink.
	LogBSSCIDispositionResolutionFailedRejectingUplink = "Disposition resolution failed; rejecting uplink"
	// LogBSSCIFailedToBeginTransaction is logged when failed to begin transaction.
	LogBSSCIFailedToBeginTransaction = "Failed to begin transaction"
	// LogBSSCIFailedToCheckSessionResume is logged when failed to check session resume.
	LogBSSCIFailedToCheckSessionResume = "Failed to check session resume"
	// LogBSSCIFailedToCloseConnection is logged when failed to close connection.
	LogBSSCIFailedToCloseConnection = "Failed to close connection"
	// LogBSSCIFailedToCloseRetiredStationConnection is logged when the connection of a retired station session cannot be closed.
	LogBSSCIFailedToCloseRetiredStationConnection = "Failed to close the connection of a base station whose EUI changed or that was deleted"
	// LogBSSCIFailedToCommitAttachTransaction is logged when failed to commit attach transaction.
	LogBSSCIFailedToCommitAttachTransaction = "Failed to commit attach transaction"
	// LogBSSCIFailedToCreateEndpointSession is logged when failed to create endpoint session.
	LogBSSCIFailedToCreateEndpointSession = "Failed to create endpoint session"
	// LogBSSCIFailedToDeriveSessionKey is logged when failed to derive session key.
	LogBSSCIFailedToDeriveSessionKey = "Failed to derive session key"
	// LogBSSCIFailedToEnqueueRelayUplink is logged when failed to enqueue relay uplink.
	LogBSSCIFailedToEnqueueRelayUplink = "Failed to enqueue relay uplink"
	// LogBSSCIFailedToLoadEndpointSession is logged when failed to load endpoint session.
	LogBSSCIFailedToLoadEndpointSession = "Failed to load endpoint session"
	// LogBSSCIFailedToMarshalConnectInfo is logged when failed to marshal connect info.
	LogBSSCIFailedToMarshalConnectInfo = "Failed to marshal connect info"
	// LogBSSCIFailedToNormalizeDetachMetadataOnResumeUsingRawData is logged when failed to normalize detach metadata on resume, using raw data.
	LogBSSCIFailedToNormalizeDetachMetadataOnResumeUsingRawData = "Failed to normalize detach metadata on resume, using raw data"
	// LogBSSCIFailedToPersistDetachPropagateMessage is logged when failed to persist detach propagate message.
	LogBSSCIFailedToPersistDetachPropagateMessage = "Failed to persist detach propagate message"
	// LogBSSCIFailedToPersistEncoding is logged when failed to persist encoding.
	LogBSSCIFailedToPersistEncoding = "Failed to persist encoding"
	// LogBSSCIFailedToPersistSession is logged when failed to persist session.
	LogBSSCIFailedToPersistSession = "Failed to persist session"
	// LogBSSCIFailedToRecordPendingAttachOperation is logged when failed to record pending attach operation.
	LogBSSCIFailedToRecordPendingAttachOperation = "Failed to record pending attach operation"
	// LogBSSCIFailedToRecordRoamingAttach is logged when failed to record roaming attach.
	LogBSSCIFailedToRecordRoamingAttach = "Failed to record roaming attach"
	// LogBSSCIFailedToRecordRoamingDetach is logged when failed to record roaming detach.
	LogBSSCIFailedToRecordRoamingDetach = "Failed to record roaming detach"
	// LogBSSCIFailedToResolveDefaultOrgForBSSCISession is logged when failed to resolve default org for BSSCI session.
	LogBSSCIFailedToResolveDefaultOrgForBSSCISession = "Failed to resolve default org for BSSCI session"
	// LogBSSCIFailedToResolveOrganizationForEndpointOwner is logged when failed to resolve organization for endpoint owner.
	LogBSSCIFailedToResolveOrganizationForEndpointOwner = "Failed to resolve organization for endpoint owner"
	// LogBSSCIFailedToSendCatalogError is logged when failed to send catalog error.
	LogBSSCIFailedToSendCatalogError = "Failed to send catalog error"

	// LogBSSCIVMStatusCompleteNotSupported rejects a VM status complete in the community edition.
	LogBSSCIVMStatusCompleteNotSupported = "VM status complete not supported in community edition"
	// LogBSSCIVMDLDataNotSupported rejects VM downlink data in the community edition.
	LogBSSCIVMDLDataNotSupported = "VM downlink data not supported in community edition"
	// LogBSSCIVMDLDataResponseNotSupported rejects a VM downlink data response in the community edition.
	LogBSSCIVMDLDataResponseNotSupported = "VM downlink data response not supported in community edition"
	// LogBSSCIVMDLDataCompleteNotSupported rejects a VM downlink data complete in the community edition.
	LogBSSCIVMDLDataCompleteNotSupported = "VM downlink data complete not supported in community edition"

	// LogBSSCIUnknownFieldDropped notes a dropped unknown field per the BSSCI §2.4 forward-compatibility rule.
	LogBSSCIUnknownFieldDropped = "Unknown field in message - dropping for forward compatibility"
	// LogBSSCIFailedToSendErrorResponse is logged when failed to send error response.
	LogBSSCIFailedToSendErrorResponse = "Failed to send error response"
	// LogBSSCIFailedToSetReadDeadline is logged when failed to set read deadline.
	LogBSSCIFailedToSetReadDeadline = "Failed to set read deadline"
	// LogBSSCIFailedToTerminateRetiredStationSession is logged when the DB session of a retired station cannot be terminated.
	LogBSSCIFailedToTerminateRetiredStationSession = "Failed to terminate the DB session of a base station whose EUI changed or that was deleted"
	// LogBSSCIFailedToUpdateEndpointDetachTelemetry is logged when failed to update endpoint detach telemetry.
	LogBSSCIFailedToUpdateEndpointDetachTelemetry = "Failed to update endpoint detach telemetry"
	// LogBSSCIFailedToUpdateEndpointSession is logged when failed to update endpoint session.
	LogBSSCIFailedToUpdateEndpointSession = "Failed to update endpoint session"
	// LogBSSCIFailedToUpdateRadioMetrics is logged when failed to update radio metrics.
	LogBSSCIFailedToUpdateRadioMetrics = "Failed to update radio metrics"
	// LogBSSCIFailedToUpdateSessionRoaming is logged when failed to update session roaming.
	LogBSSCIFailedToUpdateSessionRoaming = "Failed to update session roaming"
	// LogBSSCIFailedToUpdateSessionRoamingForDetach is logged when failed to update session roaming for detach.
	LogBSSCIFailedToUpdateSessionRoamingForDetach = "Failed to update session roaming for detach"
	// LogBSSCINoPeerCertAndFailedToResolveDefaultOrgForBSSCISession is logged when no peer cert and failed to resolve default org for BSSCI session.
	LogBSSCINoPeerCertAndFailedToResolveDefaultOrgForBSSCISession = "No peer cert and failed to resolve default org for BSSCI session"
	// LogBSSCINormalizedDetachMetadataOnResume is logged when normalized detach metadata on resume.
	LogBSSCINormalizedDetachMetadataOnResume = "Normalized detach metadata on resume"
	// LogBSSCIPayloadNormalizationFailed is logged when payload normalization failed.
	LogBSSCIPayloadNormalizationFailed = "Payload normalization failed"
	// LogBSSCIRoamingEndpointAttaching is logged when roaming endpoint attaching.
	LogBSSCIRoamingEndpointAttaching = "Roaming endpoint attaching"
	// LogBSSCIRoamingEndpointDetaching is logged when roaming endpoint detaching.
	LogBSSCIRoamingEndpointDetaching = "Roaming endpoint detaching"
	// LogBSSCIRoamingValidationFailed is logged when roaming validation failed.
	LogBSSCIRoamingValidationFailed = "Roaming validation failed"
	// LogBSSCIRoamingValidationFailedDuringDetach is logged when roaming validation failed during detach.
	LogBSSCIRoamingValidationFailedDuringDetach = "Roaming validation failed during detach"
	// LogBSSCISoftwareVersionNotConfiguredConnectResponseWillOmitSwVersionField is logged when software version not configured - ConnectResponse will omit swVersion field.
	LogBSSCISoftwareVersionNotConfiguredConnectResponseWillOmitSwVersionField = "Software version not configured - ConnectResponse will omit swVersion field"
	// LogBSSCITLSHandshakeFailed is logged when TLS handshake failed.
	LogBSSCITLSHandshakeFailed = "TLS handshake failed"
	// LogBSSCIUnknownEndpointDetachSignatureInvalid is logged when unknown endpoint detach signature invalid.
	LogBSSCIUnknownEndpointDetachSignatureInvalid = "Unknown endpoint detach signature invalid"
	// LogBSSCIUnknownEndpointDetachSignatureValidatedSuccessfully is logged when unknown endpoint detach signature validated successfully.
	LogBSSCIUnknownEndpointDetachSignatureValidatedSuccessfully = "Unknown endpoint detach signature validated successfully"
	// LogBSSCIUnknownEndpointDetachSignatureValidationFailed is logged when unknown endpoint detach signature validation failed.
	LogBSSCIUnknownEndpointDetachSignatureValidationFailed = "Unknown endpoint detach signature validation failed"
	// LogBSSCIUnknownEndpointNotFoundDuringDetachValidation is logged when unknown endpoint not found during detach validation.
	LogBSSCIUnknownEndpointNotFoundDuringDetachValidation = "Unknown endpoint not found during detach validation"
	// LogBSSCIUplinkIngestFailed is logged when uplink ingest failed.
	LogBSSCIUplinkIngestFailed = "Uplink ingest failed"
	// LogBSSCIUsingDevelopmentSoftwareVersionInConnectResponse is logged when using development software version in ConnectResponse.
	LogBSSCIUsingDevelopmentSoftwareVersionInConnectResponse = "Using development software version in ConnectResponse"
	// LogBSSCIResumeRejectedVersionIncompatible is logged when a resume is rejected because the persisted negotiated version is incompatible with the selected version.
	LogBSSCIResumeRejectedVersionIncompatible = "Resume rejected: persisted negotiated version incompatible with the selected version"
	// LogBSSCIResumeRejectedBsOpIDBeyondPersisted is logged when a resume is rejected because the required BS operation ID is beyond the persisted state.
	LogBSSCIResumeRejectedBsOpIDBeyondPersisted = "Resume rejected: required BS operation ID beyond persisted state"
	// LogBSSCIResumeRejectedScOpIDBeyondIssued is logged when a resume is rejected because the claimed SC operation ID is beyond the issued state.
	LogBSSCIResumeRejectedScOpIDBeyondIssued = "Resume rejected: claimed SC operation ID beyond issued state"
	// LogBSSCIResumeAcceptedStaleBsCounter is logged when a resume is accepted with a stale BS counter.
	LogBSSCIResumeAcceptedStaleBsCounter = "Resume accepted with stale BS counter (SC is authoritative)"
	// LogBSSCIResumeRefusedByBaseStation is logged when a base station refuses the offered session resume and the session is retired.
	LogBSSCIResumeRefusedByBaseStation = "Base station refused the offered session resume; retiring the session"
	// LogBSSCIFailedToRecordRefusedResume is logged when the refused-resume system event cannot be stored.
	LogBSSCIFailedToRecordRefusedResume = "Failed to record refused session resume event"
	// LogBSSCIFailedToMarshalPendingOperation is logged when a pending operation cannot be marshaled for persistence.
	LogBSSCIFailedToMarshalPendingOperation = "Failed to marshal pending operation"
	// LogBSSCIFailedToMarshalPendingOperationMetadata is logged when pending operation metadata cannot be marshaled for persistence.
	LogBSSCIFailedToMarshalPendingOperationMetadata = "Failed to marshal pending operation metadata"
	// LogBSSCIDLRXStatusSNRValidationFailed is logged when the dlRxSnr value in a DL RX status report fails validation.
	LogBSSCIDLRXStatusSNRValidationFailed = "DL RX status SNR validation failed"
	// LogBSSCIDLRXStatusRSSIValidationFailed is logged when the dlRxRssi value in a DL RX status report fails validation.
	LogBSSCIDLRXStatusRSSIValidationFailed = "DL RX status RSSI validation failed"
	// LogBSSCIFailedToResolveEndpointTenantForDLRXStatus is logged when the endpoint tenant cannot be resolved for a DL RX status report.
	LogBSSCIFailedToResolveEndpointTenantForDLRXStatus = "Failed to resolve endpoint tenant for DL RX status"
	// LogBSSCIFailedToCorrelateDLRXQuery is logged when a DL RX status report cannot be correlated with its pending query.
	LogBSSCIFailedToCorrelateDLRXQuery = "Failed to correlate DL RX query"
	// LogBSSCIUnsolicitedDLRXStatus is logged when a DL RX status report arrives without a pending query.
	LogBSSCIUnsolicitedDLRXStatus = "Unsolicited DL RX status (no pending query)"
	// LogBSSCIFailedToResolveEndpointOwnerOrgForDLRXStatus is logged when the endpoint owner organization cannot be resolved for DL RX status persistence.
	LogBSSCIFailedToResolveEndpointOwnerOrgForDLRXStatus = "Failed to resolve endpoint owner org for DL RX status persistence"
)
