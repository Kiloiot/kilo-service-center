// Package bssci implements the MIOTY Base Station Service Center Interface (BSSCI) v1.0.0
package bssci

// BSSCI Log Message Catalog
//
// Purpose: Centralized catalog of all BSSCI log messages for consistency and maintainability.
//
// Usage Pattern:
//   s.logger.Info(LogBSSCIAttachHandshakeComplete, zap.String("epEui", FormatEUI64(eui)))
//
// Naming Convention:
//   - All constants use LogBSSCI* prefix for discoverability
//   - Organized into subsystems matching BSSCI protocol operations
//
// Migration Stats:
//   - 276 unique messages (273 base + 3 propagation pause/reset tokens)
//   - Organized into 12 subsystems
//
// Subsystems:
//   1. Server Lifecycle & Connection (32 constants)
//   2. Attach Operations (18 constants)
//   3. Detach Operations (13 constants)
//   4. Uplink Data Operations (16 constants)
//   5. Uplink Transmit Operations (15 constants)
//   6. Downlink Queue Operations (50 constants)
//   7. Downlink Result Operations (20 constants)
//   8. Status Operations (14 constants)
//   9. VM Handler Operations (35 constants)
//  10. Error Handling & Persistence (49 constants)
//  11. Type Conversion & Field Validation (5 constants)
//  12. Propagation Reconciliation (13 constants)

const (
	// ========================================================================
	// Server Lifecycle & Connection Messages (32 constants)
	// ========================================================================

	// LogBSSCIServerListening indicates the BSSCI server has started and is listening for connections
	LogBSSCIServerListening = "BSSCI server listening"
	// LogBSSCIServerStopping indicates the BSSCI server is shutting down
	LogBSSCIServerStopping = "Stopping BSSCI server"
	// LogBSSCINewConnection indicates a new BSSCI client connection
	LogBSSCINewConnection = "new connection"
	// LogBSSCIConnectionEstablished indicates successful BSSCI connection establishment
	LogBSSCIConnectionEstablished = "connection established"
	// LogBSSCIConnectionClosed indicates a BSSCI connection has been closed
	LogBSSCIConnectionClosed = "connection closed"
	// LogBSSCIReceivedMessageDebug logs raw BSSCI message content for debugging
	LogBSSCIReceivedMessageDebug = "received message debug"
	// LogBSSCIHandshakeInitiated indicates BSSCI handshake has started
	LogBSSCIHandshakeInitiated = "handshake initiated"
	// LogBSSCIHandshakeComplete indicates BSSCI handshake has completed successfully
	LogBSSCIHandshakeComplete = "handshake complete"
	// LogBSSCIRejectingCommandBeforeHandshake logs when a command is rejected due to incomplete handshake
	LogBSSCIRejectingCommandBeforeHandshake = "Rejecting command before handshake complete"
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
	// LogBSSCIReusingExistingActiveSession indicates reusing an active session
	LogBSSCIReusingExistingActiveSession = "Reusing existing active session"
	// LogBSSCITerminatedStaleSession indicates a stale session was terminated before creating new session
	LogBSSCITerminatedStaleSession = "Terminated stale session"
	// LogBSSCIFailedToTerminateStaleSession indicates failure to terminate stale session
	LogBSSCIFailedToTerminateStaleSession = "Failed to terminate stale session"
	// LogBSSCIFailedToTerminateSession indicates failure to terminate session on disconnect
	LogBSSCIFailedToTerminateSession = "Failed to terminate session"
	// LogBSSCISessionTerminated indicates session was successfully terminated
	LogBSSCISessionTerminated = "Session terminated"
	// LogBSSCIFailedToDeletePendingOperations indicates failure to delete pending operations
	LogBSSCIFailedToDeletePendingOperations = "Failed to delete pending operations"
	// LogBSSCIDeletedPendingOperations indicates pending operations were successfully deleted
	LogBSSCIDeletedPendingOperations = "Deleted pending operations"
	// LogBSSCIStaleSessionDetectedDuringResume indicates a stale session was found during resume attempt
	LogBSSCIStaleSessionDetectedDuringResume = "Stale session detected during resume"
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
	// LogBSSCIOperationIDBackwards is a log message constant
	LogBSSCIOperationIDBackwards = "Operation ID backwards"
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
	// LogBSSCISendingBSSCIError is a log message constant
	LogBSSCISendingBSSCIError = "Sending BSSCI error"
	// LogBSSCIPersistedPendingOperation is a log message constant
	LogBSSCIPersistedPendingOperation = "Persisted pending operation"
	// LogBSSCIRemovedPendingOperation is a log message constant
	LogBSSCIRemovedPendingOperation = "Removed pending operation"
	// LogBSSCIUpdatedPendingOperationMetadata is a log message constant
	LogBSSCIUpdatedPendingOperationMetadata = "Updated pending operation metadata"

	// ========================================================================
	// Attach Operations (18 constants)
	// ========================================================================

	// LogBSSCIAttachHandshakeInitiated is a log message constant
	LogBSSCIAttachHandshakeInitiated = "Attach handshake initiated"
	// LogBSSCIAttachHandshakeComplete is a log message constant
	LogBSSCIAttachHandshakeComplete = "Attach handshake complete"
	// LogBSSCIAttachOperationCompletedSuccessfully is a log message constant
	LogBSSCIAttachOperationCompletedSuccessfully = "Attach operation completed successfully"
	// LogBSSCIReceivedAttCmpWithoutPendingAttach is a log message constant
	LogBSSCIReceivedAttCmpWithoutPendingAttach = "Received attCmp without pending attach operation" //nolint:gosec // G101: false positive - attCmp is BSSCI protocol command
	// LogBSSCIAttachInitiated is a log message constant
	LogBSSCIAttachInitiated = "attach initiated"
	// LogBSSCIReceivedAttachResponse is a log message constant
	LogBSSCIReceivedAttachResponse = "Received attach response"
	// LogBSSCIReceivedAttachComplete is a log message constant
	LogBSSCIReceivedAttachComplete = "Received attach complete"
	// LogBSSCISendingAttachPropagate is a log message constant
	LogBSSCISendingAttachPropagate = "Sending attach propagate"
	// LogBSSCISendingAttachPropagateComplete is a log message constant
	LogBSSCISendingAttachPropagateComplete = "Sending attach propagate complete"
	// LogBSSCINoConnectedBaseStationsForAttachPropagate is a log message constant
	LogBSSCINoConnectedBaseStationsForAttachPropagate = "No connected base stations available for attach propagate"
	// LogBSSCIAttachCounterReplay is a log message constant for replay attack detection
	LogBSSCIAttachCounterReplay = "Attach counter replay detected - incoming value not monotonic"
	// LogBSSCIAttachPropagateQueued is a log message constant
	LogBSSCIAttachPropagateQueued = "Attach propagate queued for sending"
	// LogBSSCIAttachPropagateRspReceived is a log message constant
	LogBSSCIAttachPropagateRspReceived = "Attach propagate response received"
	// LogBSSCIAttachPropagateCmpReceived is a log message constant
	LogBSSCIAttachPropagateCmpReceived = "Attach propagate complete received"
	// LogBSSCIUpdatedEndpointWithAttachPropagateInfo is a log message constant
	LogBSSCIUpdatedEndpointWithAttachPropagateInfo = "Updated endpoint with attach propagate info"
	// LogBSSCIFailedToPersistAttachOperation is a log message constant
	LogBSSCIFailedToPersistAttachOperation = "Failed to persist attach operation"
	// LogBSSCIFailedToPersistAttachPropagateOperation is a log message constant
	LogBSSCIFailedToPersistAttachPropagateOperation = "Failed to persist attach propagate operation"
	// LogBSSCIFailedToRemovePendingAttachOperation is a log message constant
	LogBSSCIFailedToRemovePendingAttachOperation = "Failed to remove pending attach operation"
	// LogBSSCIFailedToRecordAttachEvent is a log message constant
	LogBSSCIFailedToRecordAttachEvent = "Failed to record attach event"
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
	// Detach Operations (13 constants)
	// ========================================================================

	// LogBSSCIDetachHandshakeInitiated is a log message constant
	LogBSSCIDetachHandshakeInitiated = "Detach handshake initiated"
	// LogBSSCIDetachHandshakeComplete is a log message constant
	LogBSSCIDetachHandshakeComplete = "Detach handshake complete"
	// LogBSSCIDetachOperationCompleted is a log message constant
	LogBSSCIDetachOperationCompleted = "detach operation completed"
	// LogBSSCIReceivedDetCmpWithoutPendingDetach is a log message constant
	LogBSSCIReceivedDetCmpWithoutPendingDetach = "Received detCmp without pending detach operation" //nolint:gosec // G101: false positive - detCmp is BSSCI protocol command
	// LogBSSCIReceivedDetachResponse is a log message constant
	LogBSSCIReceivedDetachResponse = "Received detach response"
	// LogBSSCIReceivedDetachComplete is a log message constant
	LogBSSCIReceivedDetachComplete = "Received detach complete"
	// LogBSSCISendingDetachPropagate is a log message constant
	LogBSSCISendingDetachPropagate = "Sending detach propagate"
	// LogBSSCISendingDetachPropagateComplete is a log message constant
	LogBSSCISendingDetachPropagateComplete = "Sending detach propagate complete"
	// LogBSSCINoConnectedBaseStationsForDetachPropagate is a log message constant
	LogBSSCINoConnectedBaseStationsForDetachPropagate = "No connected base stations available for detach propagate"
	// LogBSSCIUpdatedEndpointWithDetachInfo is a log message constant
	LogBSSCIUpdatedEndpointWithDetachInfo = "Updated endpoint with detach info"
	// LogBSSCIFailedToPersistDetachOperation is a log message constant
	LogBSSCIFailedToPersistDetachOperation = "Failed to persist detach operation"
	// LogBSSCIFailedToRecordDetachEvent is a log message constant
	LogBSSCIFailedToRecordDetachEvent = "Failed to record detach event"
	// LogBSSCIFailedToPersistDetachMessage is a log message constant
	LogBSSCIFailedToPersistDetachMessage = "Failed to persist detach message to mioty_messages table"

	// ========================================================================
	// Uplink Data Operations (19 constants)
	// ========================================================================

	// LogBSSCIUplinkDataReceivedFirstReception is a log message constant
	LogBSSCIUplinkDataReceivedFirstReception = "Uplink data received (first reception)"
	// LogBSSCIUplinkDataDuplicate is a log message constant
	LogBSSCIUplinkDataDuplicate = "Duplicate uplink data"
	// LogBSSCIReceivedULDataCompletion is a log message constant
	LogBSSCIReceivedULDataCompletion = "Received UL data completion"
	// LogBSSCIULDataHandshakeComplete is a log message constant
	LogBSSCIULDataHandshakeComplete = "UL data handshake complete"
	// LogBSSCIReceivedStringUserData is a log message constant
	LogBSSCIReceivedStringUserData = "Received string userData, skipping (expected byte array or Numeric[])"
	// LogBSSCIUnknownUserDataType is a log message constant
	LogBSSCIUnknownUserDataType = "Unknown userData type, using empty payload"
	// LogBSSCIUnsupportedUserDataElementType is a log message constant
	LogBSSCIUnsupportedUserDataElementType = "Unsupported userData element type, using zero"
	// LogBSSCIStoringUplinkMessage is a log message constant
	LogBSSCIStoringUplinkMessage = "Storing uplink message"
	// LogBSSCIFailedToStoreUplinkMessage is a log message constant
	LogBSSCIFailedToStoreUplinkMessage = "Failed to store uplink message"
	// LogBSSCIFailedToPersistULDataOperation is a log message constant
	LogBSSCIFailedToPersistULDataOperation = "Failed to persist UL data operation"
	// LogBSSCIEndpointLookupReturnedNil is a log message constant
	LogBSSCIEndpointLookupReturnedNil = "Endpoint lookup returned nil without error, using session tenant"
	// LogBSSCIEndpointNotProvisionedForULData is a log message constant
	LogBSSCIEndpointNotProvisionedForULData = "Endpoint not provisioned for UL data, using session tenant"
	// LogBSSCIFailedToCheckUplinkDeduplication is a log message constant
	LogBSSCIFailedToCheckUplinkDeduplication = "Failed to check uplink deduplication"
	// LogBSSCIFailedToPublishUplinkToMQTT is a log message constant
	LogBSSCIFailedToPublishUplinkToMQTT = "Failed to publish uplink to MQTT"
	// LogBSSCIFailedToRecordUplinkEvent is a log message constant
	LogBSSCIFailedToRecordUplinkEvent = "Failed to record uplink event"
	// LogBSSCIFailedToRemovePendingULDataOperation is a log message constant
	LogBSSCIFailedToRemovePendingULDataOperation = "Failed to remove pending UL data operation"
	// LogBSSCIFailedToResolveEndpointTenant is a log message constant
	LogBSSCIFailedToResolveEndpointTenant = "Failed to resolve endpoint tenant for UL data"
	// LogBSSCIFailedToDecryptUserData is a log message constant
	LogBSSCIFailedToDecryptUserData = "Failed to decrypt user data"
	// LogBSSCIMessageStoreNotAvailable is a log message constant
	LogBSSCIMessageStoreNotAvailable = "Message store not available"
	// LogBSSCICreateULDataMessageFailed is logged when CreateULDataMessage fails for first reception
	LogBSSCICreateULDataMessageFailed = "bssci.create_uldata_message_failed"
	// LogBSSCIUpdateULDataBaseStationsFailed is logged when UpdateULDataBaseStations fails for duplicate
	LogBSSCIUpdateULDataBaseStationsFailed = "bssci.update_uldata_basestations_failed"
	// LogBSSCIGetDLRXStatusFailed is logged when batch DL RX status lookup fails per SCACI §3.8.1
	LogBSSCIGetDLRXStatusFailed = "bssci.get_dlrx_status_failed"
	// LogULDataTenantResolutionFailed is logged when tenant resolution fails and falls back to serving tenant
	LogULDataTenantResolutionFailed = "UL data tenant resolution failed, using serving tenant"

	// ========================================================================
	// Uplink Transmit Operations (16 constants)
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
	// LogBSSCIDecryptionFailedTryingBase64Decode is a log message constant
	LogBSSCIDecryptionFailedTryingBase64Decode = "Decryption failed for legacy key, trying base64 decode"
	// LogBSSCIFailedToPersistULDataTxOperation is a log message constant
	LogBSSCIFailedToPersistULDataTxOperation = "Failed to persist ulDataTx operation"
	// LogBSSCIFailedToRemovePendingULTransmitOperation is a log message constant
	LogBSSCIFailedToRemovePendingULTransmitOperation = "Failed to remove pending UL transmit operation from database"
	// LogBSSCIFailedToCreateULTransmitEvent is a log message constant
	LogBSSCIFailedToCreateULTransmitEvent = "Failed to create UL transmit event"

	// ========================================================================
	// Downlink Queue Operations (54 constants)
	// ========================================================================

	// LogBSSCIReceivedDLDataQueRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataQueRspFromBaseStation = "Received dlDataQueRsp from base station"
	// LogBSSCIDLDataQueueOperationCompleted is a log message constant
	LogBSSCIDLDataQueueOperationCompleted = "DL data queue operation completed"
	// LogBSSCIReceivedDLRxStatFromBaseStation is a log message constant
	LogBSSCIReceivedDLRxStatFromBaseStation = "Received dlRxStat from base station"
	// LogBSSCIReceivedDLRxStatRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLRxStatRspFromBaseStation = "Received dlRxStatRsp from base station"
	// LogBSSCIReceivedDLRxStatQryRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLRxStatQryRspFromBaseStation = "Received dlRxStatQryRsp from base station"
	// LogBSSCIDLRxStatusOperationCompleted is a log message constant
	LogBSSCIDLRxStatusOperationCompleted = "DL RX status operation completed"
	// LogBSSCIDLRxStatusQueryOperationCompleted is a log message constant
	LogBSSCIDLRxStatusQueryOperationCompleted = "DL RX status query operation completed"
	// LogBSSCISentDLRxStatQryToBaseStation is a log message constant
	LogBSSCISentDLRxStatQryToBaseStation = "Sent dlRxStatQry to base station"
	// LogBSSCIPersistedDLRxStatus is a log message constant
	LogBSSCIPersistedDLRxStatus = "Persisted DL RX status"
	// LogBSSCIReceivedDLDataRevRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataRevRspFromBaseStation = "Received dlDataRevRsp from base station"
	// LogBSSCIDLDataRevokeOperationCompleted is a log message constant
	LogBSSCIDLDataRevokeOperationCompleted = "DL data revoke operation completed"
	// LogBSSCISentDLDataRevToBaseStation is a log message constant
	LogBSSCISentDLDataRevToBaseStation = "Sent dlDataRev to base station"
	// LogBSSCIInvalidQueueIDInRevokeResponse is a log message constant
	LogBSSCIInvalidQueueIDInRevokeResponse = "Invalid queue ID in revoke response"
	// LogBSSCINoQueueIDInRevokeComplete is a log message constant
	LogBSSCINoQueueIDInRevokeComplete = "No queue ID in revoke complete"
	// LogBSSCICannotResolveTenantForRevoke is a log message constant
	LogBSSCICannotResolveTenantForRevoke = "Cannot resolve tenant for revoke"
	// LogBSSCICannotResolveTenantForRevokeComplete is a log message constant
	LogBSSCICannotResolveTenantForRevokeComplete = "Cannot resolve tenant for revoke complete"
	// LogBSSCIInvalidTenantIDFormat is a log message constant
	LogBSSCIInvalidTenantIDFormat = "Invalid tenant ID format"
	// LogBSSCIUpdatedDownlinkResult is a log message constant
	LogBSSCIUpdatedDownlinkResult = "Updated downlink result"
	// LogBSSCIUpdatedDownlinkBaseStationOwnership is a log message constant
	LogBSSCIUpdatedDownlinkBaseStationOwnership = "Updated downlink base station ownership"
	// LogBSSCIFailedToUpdateDownlinkResult is a log message constant
	LogBSSCIFailedToUpdateDownlinkResult = "Failed to update downlink result"
	// LogBSSCIFailedToUpdateDownlinkBaseStationOwnership is a log message constant
	LogBSSCIFailedToUpdateDownlinkBaseStationOwnership = "Failed to update downlink base station ownership"
	// LogBSSCIFailedToUpdateDownlinkAsRevoked is a log message constant
	LogBSSCIFailedToUpdateDownlinkAsRevoked = "Failed to update downlink as revoked"
	// LogBSSCIFailedToClearDownlinkBaseStationOwnership is a log message constant
	LogBSSCIFailedToClearDownlinkBaseStationOwnership = "Failed to clear downlink base station ownership"
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
	// LogBSSCIFailedToCheckDLRxStatQuery is a log message constant (downlink_handlers.go:~1186)
	LogBSSCIFailedToCheckDLRxStatQuery = "Failed to check DL RX status query state"
	// LogBSSCIUnsolicitedDLRxStatus is a log message constant (downlink_handlers.go:~1194)
	LogBSSCIUnsolicitedDLRxStatus = "Received unsolicited dlRxStat (no pending query)"
	// LogBSSCIFailedToMarkDLRxStatQueryComplete is a log message constant (downlink_handlers.go:~1432)
	LogBSSCIFailedToMarkDLRxStatQueryComplete = "Failed to mark DL RX status query complete"
	// LogBSSCIFailedToGetPendingOperation is a log message constant (BSSCI §§5.11-5.12.3 Gap 1)
	LogBSSCIFailedToGetPendingOperation = "Failed to get pending operation from StatusService"
	// LogBSSCIFailedToRemovePendingOperationFromDB is a log message constant
	LogBSSCIFailedToRemovePendingOperationFromDB = "Failed to remove pending operation from database"
	// LogBSSCIFailedToRemovePendingOperationAfterSendFailure is a log message constant
	LogBSSCIFailedToRemovePendingOperationAfterSendFailure = "Failed to remove pending operation from database after send failure"
	// LogBSSCIFailedToSendErrorFrameForMissingEpEui is a log message constant
	LogBSSCIFailedToSendErrorFrameForMissingEpEui = "failed to send error frame for missing epEui"
	// LogBSSCIFailedToSendErrorFrameForMissingQueID is a log message constant
	LogBSSCIFailedToSendErrorFrameForMissingQueID = "failed to send error frame for missing queId"
	// LogBSSCIFailedToSendErrorFrameForMissingResult is a log message constant
	LogBSSCIFailedToSendErrorFrameForMissingResult = "failed to send error frame for missing result"
	// LogBSSCIFailedToSendErrorFrameForInvalidResult is a log message constant
	LogBSSCIFailedToSendErrorFrameForInvalidResult = "failed to send error frame for invalid result enum"
	// LogBSSCIFailedToSendErrorFrameForUnknownQueueID is a log message constant
	LogBSSCIFailedToSendErrorFrameForUnknownQueueID = "failed to send error frame for unknown queue ID"
	// LogBSSCIFailedToSendErrorFrameForMissingConditionalFields is a log message constant
	LogBSSCIFailedToSendErrorFrameForMissingConditionalFields = "failed to send error frame for missing conditional fields"
	// LogBSSCIFailedToSendErrorFrameForInvalidConditionalFields is a log message constant
	LogBSSCIFailedToSendErrorFrameForInvalidConditionalFields = "failed to send error frame for invalid conditional fields"
	// LogBSSCIFailedToSendErrorFrameForDatabaseError is a log message constant
	LogBSSCIFailedToSendErrorFrameForDatabaseError = "failed to send error frame for database error"
	// LogBSSCIFailedToSendErrorFrame is a log message constant
	LogBSSCIFailedToSendErrorFrame = "failed to send error frame"
	// LogBSSCIFailedToBroadcastDLResultToSCACI is a log message constant
	LogBSSCIFailedToBroadcastDLResultToSCACI = "Failed to broadcast DL result to SCACI"
	// LogBSSCICannotResolveTenantForDownlinkResult is a log message constant
	LogBSSCICannotResolveTenantForDownlinkResult = "Cannot resolve tenant for downlink result"
	// LogBSSCIQueueIDOutOfRange is a log message constant
	LogBSSCIQueueIDOutOfRange = "Queue ID out of range for int64 conversion"
	// LogBSSCINegativeQueueIDInMetadata is a log message constant
	LogBSSCINegativeQueueIDInMetadata = "Negative queue ID found in operation metadata"
	// LogBSSCIInvalidQueueIDForUpdate is a log message constant
	LogBSSCIInvalidQueueIDForUpdate = "Invalid queue ID for base station ownership update"
	// LogBSSCIUpdatedBaseStationBidiCapability is a log message constant
	LogBSSCIUpdatedBaseStationBidiCapability = "Updated base station bidi capability"
	// LogBSSCIFailedToUpdateBaseStationBidiCapability is a log message constant
	LogBSSCIFailedToUpdateBaseStationBidiCapability = "Failed to update base station bidi capability"
	// LogBSSCIFailedToQueryDownlinkQueue is a log message constant
	LogBSSCIFailedToQueryDownlinkQueue = "Failed to query downlink queue"
	// LogBSSCIFailedToMarkDownlinkAsSent is a log message constant
	LogBSSCIFailedToMarkDownlinkAsSent = "Failed to mark downlink as sent"
	// LogBSSCIFailedToEnqueueDownlink is logged when downlink enqueue operation fails
	LogBSSCIFailedToEnqueueDownlink = "Failed to enqueue downlink"
	// LogBSSCIEnqueuedDownlink is logged when downlink successfully enqueued
	LogBSSCIEnqueuedDownlink = "Downlink enqueued"
	// LogBSSCIProcessingRevokeResponse is logged when processing revoke response from base station
	LogBSSCIProcessingRevokeResponse = "Processing DL Data Revoke Response from base station"
	// LogBSSCIStorageUnavailableForRevoke is logged when storage unavailable for revoke persistence
	LogBSSCIStorageUnavailableForRevoke = "Storage unavailable - cannot persist revoke operation"

	// ========================================================================
	// Downlink Result Operations (23 constants)
	// ========================================================================

	// LogBSSCIReceivedDLDataResFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataResFromBaseStation = "Received dlDataRes from base station"
	// LogBSSCIReceivedDLDataResRspFromBaseStation is a log message constant
	LogBSSCIReceivedDLDataResRspFromBaseStation = "Received dlDataResRsp from base station"
	// LogBSSCIDLDataResultOperationCompleted is a log message constant
	LogBSSCIDLDataResultOperationCompleted = "DL data result operation completed"
	// LogBSSCISendingDLDataResultResponse is a log message constant
	LogBSSCISendingDLDataResultResponse = "Sending dlDataResRsp to base station"
	// LogBSSCISendingDLDataResultComplete is a log message constant
	LogBSSCISendingDLDataResultComplete = "Sending dlDataResCmp to base station"
	// LogBSSCIProcessingDLDataResult is a log message constant
	LogBSSCIProcessingDLDataResult = "Processing dlDataRes"
	// LogBSSCIValidatedDLDataResultFields is a log message constant
	LogBSSCIValidatedDLDataResultFields = "Validated dlDataRes fields"
	// LogBSSCIFailedToValidateDLDataResultFields is a log message constant
	LogBSSCIFailedToValidateDLDataResultFields = "Failed to validate dlDataRes fields"
	// LogBSSCIFailedToPersistDLDataResOperation is a log message constant
	LogBSSCIFailedToPersistDLDataResOperation = "Failed to persist dlDataRes operation"
	// LogBSSCIFailedToRemovePendingDLResultOperation is a log message constant
	LogBSSCIFailedToRemovePendingDLResultOperation = "Failed to remove pending DL result operation"
	// LogBSSCIFailedToStoreDLResultInDatabase is a log message constant
	LogBSSCIFailedToStoreDLResultInDatabase = "Failed to store DL result in database"
	// LogBSSCIStoredDLResultInDatabase is a log message constant
	LogBSSCIStoredDLResultInDatabase = "Stored DL result in database"
	// LogBSSCIFailedToFindQueueEntryForResult is a log message constant
	LogBSSCIFailedToFindQueueEntryForResult = "Failed to find queue entry for result"
	// LogBSSCIQueueEntryNotFoundForResult is a log message constant
	LogBSSCIQueueEntryNotFoundForResult = "Queue entry not found for result"
	// LogBSSCIFailedToNotifySCACIOfDLResult is a log message constant
	LogBSSCIFailedToNotifySCACIOfDLResult = "Failed to notify SCACI of DL result"
	// LogBSSCINotifiedSCACIOfDLResult is a log message constant
	LogBSSCINotifiedSCACIOfDLResult = "Notified SCACI of DL result"
	// LogBSSCIDLResultCounterMismatch is a log message constant
	LogBSSCIDLResultCounterMismatch = "DL result counter mismatch"
	// LogBSSCIDLResultMissingRequiredField is a log message constant
	LogBSSCIDLResultMissingRequiredField = "DL result missing required field"
	// LogBSSCIDLResultInvalidEnumValue is a log message constant
	LogBSSCIDLResultInvalidEnumValue = "DL result invalid enum value"
	// LogBSSCIDLResultProcessingComplete is a log message constant
	LogBSSCIDLResultProcessingComplete = "DL result processing complete"
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
	// Status Operations (14 constants)
	// ========================================================================

	// LogBSSCISendingStatusRequestToBaseStation is a log message constant
	LogBSSCISendingStatusRequestToBaseStation = "Sending status request to base station"
	// LogBSSCIReceivedStatusRspFromBaseStation is a log message constant
	LogBSSCIReceivedStatusRspFromBaseStation = "Received statusRsp from base station"
	// LogBSSCIStatusOperationCompleted is a log message constant
	LogBSSCIStatusOperationCompleted = "Status operation completed"
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
	// LogBSSCIFailedToRemovePendingStatusOperation is a log message constant
	LogBSSCIFailedToRemovePendingStatusOperation = "Failed to remove pending status operation"
	// LogBSSCIFailedToUpdateBaseStationStatus is a log message constant
	LogBSSCIFailedToUpdateBaseStationStatus = "Failed to update base station status"
	// LogBSSCIFailedToPersistStatusHistory is a log message constant
	LogBSSCIFailedToPersistStatusHistory = "Failed to persist status history"
	// LogBSSCIFailedToSendStatusRequest is a log message constant
	LogBSSCIFailedToSendStatusRequest = "Failed to send status request"
	// LogBSSCIFailedToCleanupPendingOpAfterSendFailure is a log message constant
	LogBSSCIFailedToCleanupPendingOpAfterSendFailure = "Failed to cleanup pending operation after send failure"

	// ========================================================================
	// VM Handler Operations (35 constants)
	// ========================================================================

	// LogBSSCIVMUplinkDataReceived is a log message constant
	LogBSSCIVMUplinkDataReceived = "VM uplink data received"
	// LogBSSCIStoringVMUplinkMessage is a log message constant
	LogBSSCIStoringVMUplinkMessage = "Storing VM uplink message"
	// LogBSSCIVMStatusReceived is a log message constant
	LogBSSCIVMStatusReceived = "VM status received"
	// LogBSSCIVMStatusResponseMissingMacTypesField is a log message constant
	LogBSSCIVMStatusResponseMissingMacTypesField = "VM status response missing macTypes field"
	// LogBSSCIVMActivateSucceeded is a log message constant
	LogBSSCIVMActivateSucceeded = "VM activate succeeded"
	// LogBSSCIVMActivateFailed is a log message constant
	LogBSSCIVMActivateFailed = "VM activate failed"

	// ========================================================================
	// Additional Server Lifecycle constants (21 constants)
	// ========================================================================

	// LogBSSCIServerStarted is a log message constant
	LogBSSCIServerStarted = "BSSCI server started"
	// LogBSSCIFailedToAcceptConnection is a log message constant
	LogBSSCIFailedToAcceptConnection = "failed to accept connection"
	// LogBSSCIFailedToReadHeader is a log message constant
	LogBSSCIFailedToReadHeader = "failed to read header"
	// LogBSSCIInvalidProtocolIdentifier is a log message constant
	LogBSSCIInvalidProtocolIdentifier = "invalid protocol identifier"
	// LogBSSCIFailedToReadPayload is a log message constant
	LogBSSCIFailedToReadPayload = "failed to read payload"
	// LogBSSCIFailedToDecodeMessagePack is a log message constant
	LogBSSCIFailedToDecodeMessagePack = "failed to decode MessagePack"
	// LogBSSCIMissingCommandField is a log message constant
	LogBSSCIMissingCommandField = "missing command field"
	// LogBSSCIInvalidOpIDType is a log message constant
	LogBSSCIInvalidOpIDType = "Invalid opId type"
	// LogBSSCIInvalidServerProtocolVersion is a log message constant
	LogBSSCIInvalidServerProtocolVersion = "Invalid server protocol version"
	// LogBSSCIMinorVersionMismatch is a log message constant
	LogBSSCIMinorVersionMismatch = "Minor version mismatch"
	// LogBSSCIMinorVersionNegotiatedDown is logged when a base station requests a newer minor version and the session continues at the service center's selected version (BSSCI §5.3.2)
	LogBSSCIMinorVersionNegotiatedDown = "Minor version newer than service center, negotiating down"
	// LogBSSCIBaseStationConnected is a log message constant
	LogBSSCIBaseStationConnected = "Base Station connected"
	// LogBSSCIBaseStationFoundInDatabase is a log message constant
	LogBSSCIBaseStationFoundInDatabase = "Base Station found in database, updating status"
	// LogBSSCIBaseStationNotFoundInDatabase is a log message constant
	LogBSSCIBaseStationNotFoundInDatabase = "Base Station not found in database - rejecting connection"
	// LogBSSCIBaseStationConnectionCompletedSuccessfully is a log message constant
	LogBSSCIBaseStationConnectionCompletedSuccessfully = "Base Station connection completed successfully"
	// LogBSSCIBaseStationDisconnectedStatusOffline is a log message constant
	LogBSSCIBaseStationDisconnectedStatusOffline = "Base Station disconnected, status updated to offline"
	// LogBSSCIFailedToUpdateOfflineStatus is a log message constant
	LogBSSCIFailedToUpdateOfflineStatus = "Failed to update offline status"
	// LogBSSCIFailedToInitializeKeyEncryptor is a log message constant
	LogBSSCIFailedToInitializeKeyEncryptor = "Failed to initialize key encryptor, keys will be stored unencrypted"
	// LogBSSCIFailedToHandleMessage is a log message constant
	LogBSSCIFailedToHandleMessage = "failed to handle message"
	// LogBSSCIInvalidConnectOperationID is a log message constant
	LogBSSCIInvalidConnectOperationID = "Invalid connect operation ID"

	// LogBSSCIDebuggingCheckingSessionsForAttachPropagate is a log message constant
	LogBSSCIDebuggingCheckingSessionsForAttachPropagate = "DEBUGGING: Checking sessions for attach propagate"
	// LogBSSCIDebuggingSessionStatus is a log message constant
	LogBSSCIDebuggingSessionStatus = "DEBUGGING: Session status"
	// LogBSSCIDebugFullAttachPropagateMessage is a log message constant
	LogBSSCIDebugFullAttachPropagateMessage = "DEBUG: Full attach propagate message"
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
	// LogBSSCIDuplicateMessageReceived is a log message constant
	LogBSSCIDuplicateMessageReceived = "Duplicate message received"
	// LogBSSCIDownlinkWindowOpen is a log message constant
	LogBSSCIDownlinkWindowOpen = "Downlink window open"
	// LogBSSCIDownlinkDispatched is logged when auto-dispatch successfully sends a downlink
	LogBSSCIDownlinkDispatched = "Downlink auto-dispatched on dlOpen"
	// LogBSSCIDownlinkDispatchError is logged when auto-dispatch fails
	LogBSSCIDownlinkDispatchError = "Downlink auto-dispatch error"
	// LogDispatcherNoTenant is logged when dispatcher skips due to missing tenant context
	LogDispatcherNoTenant = "Downlink dispatcher skipping: no tenant context"
	// LogDispatcherTxBeginFailed is logged when transaction start fails
	LogDispatcherTxBeginFailed = "Downlink dispatcher: transaction begin failed"
	// LogDispatcherQueryFailed is logged when pending downlink query fails
	LogDispatcherQueryFailed = "Downlink dispatcher: query failed"
	// LogDispatcherNoPending is logged when no pending downlinks found (debug level)
	LogDispatcherNoPending = "Downlink dispatcher: no pending downlinks"
	// LogDispatcherSendFailed is logged when SendDLDataQueue fails
	LogDispatcherSendFailed = "Downlink dispatcher: send failed"
	// LogDispatcherMarkSentFailed is logged when marking downlink as queued fails
	LogDispatcherMarkSentFailed = "Downlink dispatcher: mark queued failed"
	// LogDispatcherTxCommitFailed is logged when transaction commit fails
	LogDispatcherTxCommitFailed = "Downlink dispatcher: commit failed"
	// LogDispatcherSuccess is logged when downlink successfully dispatched
	LogDispatcherSuccess = "Downlink dispatcher: success"

	// LogBSSCIDeduplicationError is a log message constant
	LogBSSCIDeduplicationError = "Deduplication error"
	// LogBSSCICompletingThreeWayHandshakeForError is a log message constant
	LogBSSCICompletingThreeWayHandshakeForError = "Completing three-way handshake for error operation"
	// LogBSSCIErrorOperationHandshakeCompletedDatabaseNotUpdated is a log message constant
	LogBSSCIErrorOperationHandshakeCompletedDatabaseNotUpdated = "Error operation handshake completed, database state NOT updated"
	// LogBSSCIBaseStationReportedError is a log message constant
	LogBSSCIBaseStationReportedError = "Base Station reported error"
	// LogBSSCIDatabaseSessionCreated is a log message constant
	LogBSSCIDatabaseSessionCreated = "Database session created"
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
	// LogBSSCIErrorIteratingPendingOperationsRows is a log message constant
	LogBSSCIErrorIteratingPendingOperationsRows = "Error iterating pending operations rows"
	// LogBSSCILoadedUserDataFromMetadataForULDataTx is a log message constant
	LogBSSCILoadedUserDataFromMetadataForULDataTx = "Loaded userData from metadata for ulDataTx operation"
	// LogBSSCIFailedToCreateDatabaseSession is a log message constant
	LogBSSCIFailedToCreateDatabaseSession = "Failed to create database session"
	// LogBSSCIFailedToUpdateDatabaseSession is a log message constant
	LogBSSCIFailedToUpdateDatabaseSession = "Failed to update database session"
	// LogBSSCIFailedToFindExistingActiveSessionAfterConflict is a log message constant
	LogBSSCIFailedToFindExistingActiveSessionAfterConflict = "Failed to find existing active session after conflict"
	// LogBSSCIFailedToSendErrorMessageToBaseStation is a log message constant
	LogBSSCIFailedToSendErrorMessageToBaseStation = "Failed to send error message to base station"
	// LogBSSCIFailedToSendErrorAck is a log message constant
	LogBSSCIFailedToSendErrorAck = "Failed to send errorAck"
	// LogBSSCIErrorAckSent is a log message constant for successful errorAck transmission (BSSCI §5.17.2)
	LogBSSCIErrorAckSent = "Sent errorAck to base station for failed operation"
	// LogBSSCIAcknowledgingErrorForUnknownOperation is a log message constant per BSSCI §5.17
	LogBSSCIAcknowledgingErrorForUnknownOperation = "Acknowledging error for unknown operation per BSSCI §5.17"
	// LogBSSCIFailedToSendCompletionMessageForErrorOperation is a log message constant
	LogBSSCIFailedToSendCompletionMessageForErrorOperation = "Failed to send completion message for error operation"
	// LogBSSCIFailedToStoreMessage is a log message constant
	LogBSSCIFailedToStoreMessage = "Failed to store message"
	// LogBSSCIFailedToStoreULDataCompletionEvent is a log message constant
	LogBSSCIFailedToStoreULDataCompletionEvent = "Failed to store UL data completion event"
	// LogBSSCIFailedToStoreGeneratedShortAddress is a log message constant
	LogBSSCIFailedToStoreGeneratedShortAddress = "Failed to store generated short address"
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
	// LogBSSCIFailedToScanPendingOperationRow is a log message constant
	LogBSSCIFailedToScanPendingOperationRow = "Failed to scan pending operation row"
	// LogBSSCIFailedToUnmarshalOperationData is a log message constant
	LogBSSCIFailedToUnmarshalOperationData = "Failed to unmarshal operation data"
	// LogBSSCIFailedToUnmarshalMetadata is a log message constant
	LogBSSCIFailedToUnmarshalMetadata = "Failed to unmarshal metadata"
	// LogBSSCIFailedToReconstituteDLDataQueSkipping is a log message constant
	LogBSSCIFailedToReconstituteDLDataQueSkipping = "Failed to reconstitute dlDataQue, skipping"
	// LogBSSCIFailedToReconstituteDLDataRevSkipping is a log message constant
	LogBSSCIFailedToReconstituteDLDataRevSkipping = "Failed to reconstitute dlDataRev, skipping"
	// LogBSSCIFailedToReconstituteULDataTxSkipping is a log message constant
	LogBSSCIFailedToReconstituteULDataTxSkipping = "Failed to reconstitute ulDataTx, skipping"
	// LogBSSCIFailedToReissuePendingOperation is a log message constant
	LogBSSCIFailedToReissuePendingOperation = "Failed to reissue pending operation"
	// LogBSSCIFailedToRemovePendingOperation is a log message constant
	LogBSSCIFailedToRemovePendingOperation = "Failed to remove pending operation"
	// LogBSSCIFailedToRemovePendingOperationAfterErrorAck is a log message constant
	LogBSSCIFailedToRemovePendingOperationAfterErrorAck = "Failed to remove pending operation after errorAck"
	// LogBSSCIFailedToDeletePendingOperationFromDatabase is a log message constant
	LogBSSCIFailedToDeletePendingOperationFromDatabase = "Failed to delete pending operation from database"
	// LogBSSCIFailedToPersistFailureMetadata is a log message constant
	LogBSSCIFailedToPersistFailureMetadata = "Failed to persist failure metadata"
	// LogBSSCIFailedToUpdateEndpointStatus is a log message constant
	LogBSSCIFailedToUpdateEndpointStatus = "Failed to update endpoint status"
	// LogBSSCIFailedToUpdateEndpointWithAttachInfo is a log message constant
	LogBSSCIFailedToUpdateEndpointWithAttachInfo = "Failed to update endpoint with attach info"
	// LogBSSCIFailedToUpdateEndpointWithAttachPropagateInfo is a log message constant
	LogBSSCIFailedToUpdateEndpointWithAttachPropagateInfo = "Failed to update endpoint with attach propagate info"
	// LogBSSCIFailedToUpdateEndpointWithDetachInfo is a log message constant
	LogBSSCIFailedToUpdateEndpointWithDetachInfo = "Failed to update endpoint with detach info"
	// LogBSSCIFailedToUpdateEndpointAttachmentState is a log message constant
	LogBSSCIFailedToUpdateEndpointAttachmentState = "Failed to update endpoint attachment state"
	// LogBSSCIFailedToUpdateEndpointAttachMetadata is a log message constant
	LogBSSCIFailedToUpdateEndpointAttachMetadata = "Failed to update endpoint attach metadata"
	// LogBSSCIFailedToUpdateEndpointDetachmentState is a log message constant
	LogBSSCIFailedToUpdateEndpointDetachmentState = "Failed to update endpoint detachment state"
	// LogBSSCIFailedToUpdateEndpointDetachState is a log message constant
	LogBSSCIFailedToUpdateEndpointDetachState = "Failed to update endpoint detach state"
	// LogBSSCIFailedToUpdateEndpointShortAddress is a log message constant
	LogBSSCIFailedToUpdateEndpointShortAddress = "Failed to update endpoint short address"
	// LogBSSCIFailedToUpdateConnectionStatus is a log message constant
	LogBSSCIFailedToUpdateConnectionStatus = "Failed to update connection status"
	// LogBSSCIFailedToUpdateLastSeen is a log message constant
	LogBSSCIFailedToUpdateLastSeen = "Failed to update last seen"
	// LogBSSCIFailedToEncryptNetworkKeyStoringUnencrypted is a log message constant
	LogBSSCIFailedToEncryptNetworkKeyStoringUnencrypted = "Failed to encrypt network key, storing unencrypted"
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
	// LogBSSCIFailedToCreateAttachInitiatedEvent is a log message constant
	LogBSSCIFailedToCreateAttachInitiatedEvent = "Failed to create attach initiated event"
	// LogBSSCIFailedToCreateAttachFailedEvent is a log message constant
	LogBSSCIFailedToCreateAttachFailedEvent = "Failed to create attach failed event"
	// LogBSSCIFailedToCreateAttachmentEvent is a log message constant
	LogBSSCIFailedToCreateAttachmentEvent = "Failed to create attachment event"
	// LogBSSCIFailedToCreateBaseStationAttachmentEvent is a log message constant
	LogBSSCIFailedToCreateBaseStationAttachmentEvent = "Failed to create base station attachment event"
	// LogBSSCIFailedToCreateBasestationAttachInitiatedEvent is a log message constant
	LogBSSCIFailedToCreateBasestationAttachInitiatedEvent = "Failed to create basestation attach initiated event"
	// LogBSSCIFailedToCreateBasestationDetachInitiatedEvent is a log message constant
	LogBSSCIFailedToCreateBasestationDetachInitiatedEvent = "Failed to create basestation detach initiated event"
	// LogBSSCIFailedToCreateBaseStationDetachmentEvent is a log message constant
	LogBSSCIFailedToCreateBaseStationDetachmentEvent = "Failed to create base station detachment event"
	// LogBSSCIFailedToCreateDetachEvent is a log message constant
	LogBSSCIFailedToCreateDetachEvent = "Failed to create detach event"
	// LogBSSCIFailedToCreateDetachInitiatedEvent is a log message constant
	LogBSSCIFailedToCreateDetachInitiatedEvent = "Failed to create detach initiated event"
	// LogBSSCIFailedToCreateDetachFailedEvent is a log message constant
	LogBSSCIFailedToCreateDetachFailedEvent = "Failed to create detach failed event"
	// LogBSSCIFailedToCreateDetachmentEvent is a log message constant
	LogBSSCIFailedToCreateDetachmentEvent = "Failed to create detachment event"
	// LogBSSCIFailedToCreateEvent is a log message constant
	LogBSSCIFailedToCreateEvent = "Failed to create event"
	// LogBSSCIFailedToCreateULDataEvent is a log message constant
	LogBSSCIFailedToCreateULDataEvent = "Failed to create UL data event"
	// LogBSSCIFailedToClearPersistedPendingOperation is a log message constant
	LogBSSCIFailedToClearPersistedPendingOperation = "Failed to clear persisted pending operation"
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
	// LogBSSCIVMDownlinkDataAccepted is a log message constant
	LogBSSCIVMDownlinkDataAccepted = "VM downlink data accepted"
	// LogBSSCIVMDownlinkDataRejected is a log message constant
	LogBSSCIVMDownlinkDataRejected = "VM downlink data rejected"
	// LogBSSCIVMDownlinkDataOperationCompleted is a log message constant
	LogBSSCIVMDownlinkDataOperationCompleted = "VM downlink data operation completed"
	// LogBSSCIFailedToRecordVMDeactivateSuccessEvent is a log message constant
	LogBSSCIFailedToRecordVMDeactivateSuccessEvent = "Failed to record VM deactivate success event"
	// LogBSSCIFailedToRecordVMDownlinkAcceptedEvent is a log message constant
	LogBSSCIFailedToRecordVMDownlinkAcceptedEvent = "Failed to record VM downlink accepted event"
	// LogBSSCIFailedToRemovePendingVMOpAfterSendFailure is a log message constant
	LogBSSCIFailedToRemovePendingVMOpAfterSendFailure = "Failed to remove pending operation from database after send failure"
	// LogBSSCIFailedToUpdatePendingOperationMetadata is a log message constant
	LogBSSCIFailedToUpdatePendingOperationMetadata = "Failed to update pending operation metadata"
	// LogBSSCIFailedToRecordVMStatusEvent is a log message constant
	LogBSSCIFailedToRecordVMStatusEvent = "Failed to record VM status event"
	// LogBSSCIFailedToCheckVMUplinkDeduplication is a log message constant
	LogBSSCIFailedToCheckVMUplinkDeduplication = "Failed to check VM uplink deduplication"
	// LogBSSCIFailedToRecordVMUplinkDataEvent is a log message constant
	LogBSSCIFailedToRecordVMUplinkDataEvent = "Failed to record VM uplink data event"
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
	// LogBSSCIFailedToStoreVMUplinkMessage is a log message constant
	LogBSSCIFailedToStoreVMUplinkMessage = "Failed to store VM uplink message"
	// LogBSSCIFailedToValidateVMStatusResponse is a log message constant
	LogBSSCIFailedToValidateVMStatusResponse = "Failed to validate VM status response"
	// LogBSSCIFailedToProcessVMUplinkData is a log message constant
	LogBSSCIFailedToProcessVMUplinkData = "Failed to process VM uplink data"
	// LogBSSCIFailedToSendVMCommand is a log message constant
	LogBSSCIFailedToSendVMCommand = "Failed to send VM command"
	// LogBSSCIVMOperationTimeout is a log message constant
	LogBSSCIVMOperationTimeout = "VM operation timeout"

	// ========================================================================
	// Type Conversion & Field Validation (5 constants)
	// ========================================================================

	// LogBSSCIIntegerOverflowInDeduplication is logged when an integer overflow occurs during deduplication field conversion
	LogBSSCIIntegerOverflowInDeduplication = "Integer overflow in deduplication context"
	// LogBSSCIIntegerOverflowInMacTypeParsing is logged when an integer overflow occurs during MAC type parsing
	LogBSSCIIntegerOverflowInMacTypeParsing = "Integer overflow in macType parsing"
	// LogBSSCIIntegerOverflowInLogging is logged when an integer overflow occurs during logging field conversion
	LogBSSCIIntegerOverflowInLogging = "Integer overflow in logging field conversion"
	// LogBSSCIIntegerOverflowInEventPayload is logged when an integer overflow occurs during event payload conversion
	LogBSSCIIntegerOverflowInEventPayload = "Integer overflow in event payload conversion"
	// LogBSSCIDebugGetNumericField is logged for debugging numeric field extraction from MessagePack data
	LogBSSCIDebugGetNumericField = "Debug: getNumericField extraction"

	// ========================================================================
	// Multi-Tenant Roaming & Organization Resolution (3 constants)
	// ========================================================================

	// LogBSSCIMissingTenantInMetadata is logged when tenant ID is missing from pending operation metadata
	LogBSSCIMissingTenantInMetadata = "Missing tenantId in pending operation metadata, falling back to session tenant"
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
	// Propagation Reconciliation (17 constants)
	// ========================================================================

	// LogBSSCIPropagationStarted is logged when automatic propagation reconciliation is initiated on base station connect
	LogBSSCIPropagationStarted = "bssci.propagation.started"
	// LogBSSCIPropagationSkippedNoChanges is logged when reconciliation skipped due to no new endpoints since last sync
	LogBSSCIPropagationSkippedNoChanges = "bssci.propagation.skipped_no_changes"
	// LogBSSCIPropagationSkippedRetryBackoff is logged when reconciliation skipped due to retry backoff window
	LogBSSCIPropagationSkippedRetryBackoff = "bssci.propagation.skipped_retry_backoff"
	// LogBSSCIPropagationCompleted is logged when automatic propagation reconciliation finishes successfully
	LogBSSCIPropagationCompleted = "bssci.propagation.completed"
	// LogBSSCIPropagationEndpointFailed is logged when a single endpoint fails during reconciliation batch
	LogBSSCIPropagationEndpointFailed = "bssci.propagation.endpoint_failed"
	// LogBSSCIPropagationFailed is logged when automatic propagation reconciliation encounters a fatal error
	LogBSSCIPropagationFailed = "bssci.propagation.failed"
	// LogBSSCIPropagationRetryScheduled is logged when exponential backoff is scheduled after propagation error
	LogBSSCIPropagationRetryScheduled = "bssci.propagation.retry_scheduled"
	// LogBSSCIPropagationLookupFailed is logged when base station propagation state lookup fails
	LogBSSCIPropagationLookupFailed = "bssci.propagation.bs_lookup_failed"
	// LogBSSCIPropagationBatchComplete is logged when a batch of endpoints completes propagation processing
	LogBSSCIPropagationBatchComplete = "bssci.propagation.batch_complete"
	// LogBSSCIPropagationPausedMaxRetries is logged when automatic propagation is paused after exceeding maximum retry attempts
	LogBSSCIPropagationPausedMaxRetries = "bssci.propagation.paused_max_retries"
	// LogBSSCIPropagationResetManually is logged when propagation reconciliation is manually reset by an administrator
	LogBSSCIPropagationResetManually = "bssci.propagation.reset_manually"
	// LogBSSCIPropagationCoolDownNotImplemented is logged when the auto cool-down feature is unavailable
	LogBSSCIPropagationCoolDownNotImplemented = "bssci.propagation.cool_down_not_implemented"
	// LogBSSCIResumeReconcileOps is logged when reconciliation operations are prioritized during session resume
	LogBSSCIResumeReconcileOps = "bssci.resume.reconcile_ops"
	// LogBSSCIFailedToPersistAttachPropagateMessage is logged when attach propagate message persistence fails
	LogBSSCIFailedToPersistAttachPropagateMessage = "Failed to persist attach propagate message"
	// LogBSSCIAutomaticPropagationFailedAfterOTAAttach is logged when automatic propagation fails after OTA attach completion
	LogBSSCIAutomaticPropagationFailedAfterOTAAttach = "Automatic propagation failed after OTA attach"
	// LogBSSCIBaseStationReconciliationFailed is logged when base station reconciliation fails after handshake
	LogBSSCIBaseStationReconciliationFailed = "Base station reconciliation failed after handshake"
	// LogBSSCIEndpointNotFoundForAttachPropagation is logged when endpoint cannot be found for attach propagation
	LogBSSCIEndpointNotFoundForAttachPropagation = "Endpoint not found for attach propagation"

	// ========================================================================
	// Outbound Message Validation (BSSCI §2.5) (5 constants)
	// ========================================================================

	// LogBSSCIOutboundValidationFailed is logged when outbound message validation fails
	LogBSSCIOutboundValidationFailed = "bssci.outbound.validation.failed"
	// LogBSSCIUnknownOutboundCommand is logged when outbound command is not in specification catalog
	LogBSSCIUnknownOutboundCommand = "bssci.outbound.unknown_command"
	// LogBSSCIExtraOutboundFields is logged when outbound message contains non-specification fields
	LogBSSCIExtraOutboundFields = "bssci.outbound.extra_fields"
	// LogBSSCIOutboundDisallowedField is logged when outbound message contains disallowed field
	LogBSSCIOutboundDisallowedField = "bssci.outbound.disallowed_field"
	// LogBSSCIMissingMandatoryField is logged when outbound message missing mandatory non-optional field
	LogBSSCIMissingMandatoryField = "bssci.outbound.missing_mandatory_field"
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

	// LogBSSCIUnsupportedOperationTypeStartEvent is logged when logStartEvent receives an unexpected operation type
	LogBSSCIUnsupportedOperationTypeStartEvent = "unsupported operation type for start event"

	// ========================================================================
	// MQTT Event Publishing (8 constants)
	// ========================================================================

	// LogBSSCIFailedToPublishAttachEventToMQTT is logged when MQTT attach event publish fails
	LogBSSCIFailedToPublishAttachEventToMQTT = "Failed to publish attach event to MQTT"
	// LogBSSCIFailedToPublishDetachEventToMQTT is logged when MQTT detach event publish fails
	LogBSSCIFailedToPublishDetachEventToMQTT = "Failed to publish detach event to MQTT"
	// LogBSSCIFailedToPublishDLResultToMQTT is logged when MQTT downlink result event publish fails
	LogBSSCIFailedToPublishDLResultToMQTT = "Failed to publish DL result event to MQTT"
	// LogBSSCIMQTTPublishSkippedOrgUnresolved is logged when MQTT publish is skipped due to unresolved organization
	LogBSSCIMQTTPublishSkippedOrgUnresolved = "MQTT publish skipped: organization unresolved"

	// MQTTEventKeyUplink identifies uplink events in structured log fields
	MQTTEventKeyUplink = "uplink"
	// MQTTEventKeyAttach identifies attach events in structured log fields
	MQTTEventKeyAttach = "attach"
	// MQTTEventKeyDetach identifies detach events in structured log fields
	MQTTEventKeyDetach = "detach"
	// MQTTEventKeyDownlinkResult identifies downlink result events in structured log fields
	MQTTEventKeyDownlinkResult = "downlink_result"

	// LogBSSCICertsNotFound is logged when TLS certificates are not yet available at startup
	LogBSSCICertsNotFound = "BSSCI certificates not found, listener deferred until certificates are generated"
	// LogBSSCICertsDetected is logged when deferred certificate polling finds certificates
	LogBSSCICertsDetected = "BSSCI certificates detected, starting TLS listener"
	// LogBSSCIDeferredListenerFailed is logged when the deferred TLS listener fails to start
	LogBSSCIDeferredListenerFailed = "Failed to start deferred BSSCI TLS listener"
	// LogBSSCIDeferredListenerCancelled is logged when the deferred listener polling is cancelled
	LogBSSCIDeferredListenerCancelled = "BSSCI deferred listener polling cancelled"

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

	// LogBSSCIUplinkDeduplicationError is logged when the deduplicator returns an error during ingest.
	LogBSSCIUplinkDeduplicationError = "Uplink deduplication error"
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
	// LogBSSCIFailedToPersistUplinkMessage is logged when CreateULDataMessage fails.
	LogBSSCIFailedToPersistUplinkMessage = "Failed to persist uplink message"
	// LogBSSCIFailedToMarshalBaseStationsForDuplicateUpdate is logged when JSON-marshaling base_stations fails on the duplicate path.
	LogBSSCIFailedToMarshalBaseStationsForDuplicateUpdate = "Failed to marshal base_stations for duplicate update"
	// LogBSSCIFailedToUpdateBaseStationsForDuplicate is logged when UpdateULDataBaseStations fails on the duplicate path.
	LogBSSCIFailedToUpdateBaseStationsForDuplicate = "Failed to update base_stations for duplicate"
	// LogBSSCIFailedToBroadcastUplinkToSCACI is logged when the SCACI broadcaster returns an error during ingest.
	LogBSSCIFailedToBroadcastUplinkToSCACI = "Failed to broadcast uplink to SCACI"
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
	// LogBSSCIClosingBSSCISessionDueToEUIChange is logged when closing BSSCI session due to EUI change.
	LogBSSCIClosingBSSCISessionDueToEUIChange = "Closing BSSCI session due to EUI change"
	// LogBSSCIDetachFromUnknownEndpoint is logged when detach from unknown endpoint.
	LogBSSCIDetachFromUnknownEndpoint = "Detach from unknown endpoint"
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
	// LogBSSCIFailedToCloseConnectionDuringEUIChangeCleanup is logged when failed to close connection during EUI change cleanup.
	LogBSSCIFailedToCloseConnectionDuringEUIChangeCleanup = "Failed to close connection during EUI change cleanup"
	// LogBSSCIFailedToCloseListener is logged when failed to close listener.
	LogBSSCIFailedToCloseListener = "Failed to close listener"
	// LogBSSCIFailedToCommitAttachTransaction is logged when failed to commit attach transaction.
	LogBSSCIFailedToCommitAttachTransaction = "Failed to commit attach transaction"
	// LogBSSCIFailedToCreateEndpointSession is logged when failed to create endpoint session.
	LogBSSCIFailedToCreateEndpointSession = "Failed to create endpoint session"
	// LogBSSCIFailedToDeriveSessionKey is logged when failed to derive session key.
	LogBSSCIFailedToDeriveSessionKey = "Failed to derive session key"
	// LogBSSCIFailedToEncryptSessionKey is logged when failed to encrypt session key.
	LogBSSCIFailedToEncryptSessionKey = "Failed to encrypt session key"
	// LogBSSCIFailedToEnqueueRelayUplink is logged when failed to enqueue relay uplink.
	LogBSSCIFailedToEnqueueRelayUplink = "Failed to enqueue relay uplink"
	// LogBSSCIFailedToLoadEndpointSession is logged when failed to load endpoint session.
	LogBSSCIFailedToLoadEndpointSession = "Failed to load endpoint session"
	// LogBSSCIFailedToMarshalConnectInfo is logged when failed to marshal connect info.
	LogBSSCIFailedToMarshalConnectInfo = "Failed to marshal connect info"
	// LogBSSCIFailedToNormalizeAttachSubpackets is logged when failed to normalize attach subpackets.
	LogBSSCIFailedToNormalizeAttachSubpackets = "Failed to normalize attach subpackets"
	// LogBSSCIFailedToNormalizeDetachMetadataOnResumeUsingRawData is logged when failed to normalize detach metadata on resume, using raw data.
	LogBSSCIFailedToNormalizeDetachMetadataOnResumeUsingRawData = "Failed to normalize detach metadata on resume, using raw data"
	// LogBSSCIFailedToNormalizeDetachSubpackets is logged when failed to normalize detach subpackets.
	LogBSSCIFailedToNormalizeDetachSubpackets = "Failed to normalize detach subpackets"
	// LogBSSCIFailedToPersistDetachPropagateMessage is logged when failed to persist detach propagate message.
	LogBSSCIFailedToPersistDetachPropagateMessage = "Failed to persist detach propagate message"
	// LogBSSCIFailedToPersistEncoding is logged when failed to persist encoding.
	LogBSSCIFailedToPersistEncoding = "Failed to persist encoding"
	// LogBSSCIFailedToPersistSession is logged when failed to persist session.
	LogBSSCIFailedToPersistSession = "Failed to persist session"
	// LogBSSCIFailedToRecordPendingAttachOperation is logged when failed to record pending attach operation.
	LogBSSCIFailedToRecordPendingAttachOperation = "Failed to record pending attach operation"
	// LogBSSCIFailedToRecordPendingOperation is logged when failed to record pending operation.
	LogBSSCIFailedToRecordPendingOperation = "Failed to record pending operation"
	// LogBSSCIFailedToRecordRoamingAttach is logged when failed to record roaming attach.
	LogBSSCIFailedToRecordRoamingAttach = "Failed to record roaming attach"
	// LogBSSCIFailedToRecordRoamingDetach is logged when failed to record roaming detach.
	LogBSSCIFailedToRecordRoamingDetach = "Failed to record roaming detach"
	// LogBSSCIFailedToResolveDefaultOrgForBSSCISession is logged when failed to resolve default org for BSSCI session.
	LogBSSCIFailedToResolveDefaultOrgForBSSCISession = "Failed to resolve default org for BSSCI session"
	// LogBSSCIFailedToResolveOrganizationForEndpointOwner is logged when failed to resolve organization for endpoint owner.
	LogBSSCIFailedToResolveOrganizationForEndpointOwner = "Failed to resolve organization for endpoint owner"
	// LogBSSCIFailedToResolveOrganizationForUnknownEndpointOwner is logged when failed to resolve organization for unknown endpoint owner.
	LogBSSCIFailedToResolveOrganizationForUnknownEndpointOwner = "Failed to resolve organization for unknown endpoint owner"
	// LogBSSCIFailedToSendCatalogError is logged when failed to send catalog error.
	LogBSSCIFailedToSendCatalogError = "Failed to send catalog error"
	// LogBSSCIFailedToSendErrorResponse is logged when failed to send error response.
	LogBSSCIFailedToSendErrorResponse = "Failed to send error response"
	// LogBSSCIFailedToSetReadDeadline is logged when failed to set read deadline.
	LogBSSCIFailedToSetReadDeadline = "Failed to set read deadline"
	// LogBSSCIFailedToTerminateDBSessionDuringEUIChange is logged when failed to terminate DB session during EUI change.
	LogBSSCIFailedToTerminateDBSessionDuringEUIChange = "Failed to terminate DB session during EUI change"
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
)
