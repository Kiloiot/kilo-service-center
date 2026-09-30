package events

import (
	"encoding/json"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Detail keys shared by several event families.
const (
	detailKeyEpEui        = models.EventDetailKeyEpEui
	detailKeyBsEui        = models.EventDetailKeyBsEui
	detailKeyOpID         = models.EventDetailKeyOpID
	detailKeyQueID        = models.EventDetailKeyQueID
	detailKeyOperation    = models.EventDetailKeyOperation
	detailKeyOperationID  = models.EventDetailKeyOperationID
	detailKeyCommand      = "command"
	detailKeyReason       = models.EventDetailKeyReason
	detailKeyMethod       = "method"
	detailKeyOrgID        = "orgId"
	detailKeyUserID       = "userId"
	detailKeyRole         = "role"
	detailKeyName         = "name"
	detailKeyEmail        = "email"
	detailKeyStatus       = models.EventDetailKeyStatus
	detailKeyTimestamp    = models.EventDetailKeyTimestamp
	detailKeyBSName       = models.EventDetailKeyBaseStationName
	detailKeyMacType      = models.EventDetailKeyMacType
	detailKeyMessage      = models.EventDetailKeyMessage
	detailKeyError        = models.EventDetailKeyError
	detailKeyActiveMacs   = models.EventDetailKeyActiveMacTypes
	detailKeyRxTime       = models.EventDetailKeyRxTime
	detailKeyRssi         = models.EventDetailKeyRssi
	detailKeySnr          = models.EventDetailKeySnr
	detailKeyEqSnr        = "eqSnr"
	detailKeyAttachCnt    = models.EventDetailKeyAttachCnt
	detailKeyDualChan     = "dualChan"
	detailKeyRepetition   = "repetition"
	detailKeyWideCarrOff  = "wideCarrOff"
	detailKeyLongBlkDist  = "longBlkDist"
	detailKeyShAddr       = models.EventDetailKeyShAddr
	detailKeySuccess      = models.EventDetailKeySuccess
	detailKeyTime         = models.EventDetailKeyTime
	detailKeyFailureCode  = models.EventDetailKeyFailureCode
	detailKeyTargetBS     = models.EventDetailKeyTargetBS
	detailKeyTargetBSList = models.EventDetailKeyTargetBSList
	detailKeyTargetBSCnt  = models.EventDetailKeyTargetBSCount
	detailKeyEndpointID   = models.EventDetailKeyEndpointID
	detailKeyBSID         = models.EventDetailKeyBaseStationID
	detailKeyOpType       = models.EventDetailKeyOperationType
	detailKeySessionID    = models.EventDetailKeySessionID
	detailKeyAcEui        = models.EventDetailKeyAcEui
	detailKeyRemoteAddr   = models.EventDetailKeyRemoteAddr
	detailKeyErrorToken   = models.EventDetailKeyErrorToken
	detailKeyCount        = models.EventDetailKeyCount
	detailKeyFirstSeen    = models.EventDetailKeyFirstSeen
	detailKeyLastSeen     = models.EventDetailKeyLastSeen
	detailKeyErrorCode    = "errorCode"
	detailKeyErrorMsg     = "errorMsg"
	detailKeyValidityDays = "validityDays"
	detailKeyScope        = "scope"
	detailKeyIntegration  = "integrationId"
	detailKeyType         = "type"
	detailKeyKeyID        = "keyId"
	detailKeyKeyPrefix    = "keyPrefix"
	detailKeyKeyType      = "keyType"
	detailKeyIsOrgAdmin   = "isOrgAdmin"
	detailKeyIsBSAdmin    = "isBaseStationAdmin"
	detailKeyIsEPAdmin    = "isEndpointAdmin"
	detailKeyService      = "service"
	detailKeyVersion      = "version"
	detailKeyGitCommit    = "gitCommit"
	detailKeyHost         = "host"
	detailKeyPorts        = "ports"
	detailKeyBidi         = "bidi"
	detailKeyRevealedKeys = models.EventDetailKeyRevealedKeys
	detailKeyRemovedKeys  = models.EventDetailKeyRemovedKeys
	detailKeyServiceAcct  = models.EventDetailKeyServiceAccount
	detailKeyIsOnline     = models.EventDetailKeyIsOnline
	detailKeyConnType     = models.EventDetailKeyConnectionType
	detailKeyPacketCnt    = models.EventDetailKeyPacketCnt
	detailKeyDlRxSnr      = models.EventDetailKeyDlRxSnr
	detailKeyDlRxRssi     = models.EventDetailKeyDlRxRssi
	detailKeyUserData     = models.EventDetailKeyUserData
	detailKeyTxTime       = models.EventDetailKeyTxTime
	detailKeyResult       = models.EventDetailKeyResult
)

var (
	connectivityKeys = []string{detailKeyBsEui, detailKeyBSName, detailKeyIsOnline, detailKeyConnType}
	radioAttachKeys  = []string{detailKeyEpEui, detailKeyBsEui, detailKeyAttachCnt, detailKeyRxTime, detailKeyRssi, detailKeySnr, detailKeyEqSnr, detailKeyDualChan, detailKeyRepetition, detailKeyWideCarrOff, detailKeyLongBlkDist, detailKeyOperation, detailKeyOpID}
	propagateKeys    = []string{detailKeyEpEui, detailKeyBsEui, detailKeyShAddr, detailKeySuccess, detailKeyTime, detailKeyOperation, detailKeyOperationID, detailKeyFailureCode, detailKeyReason, detailKeyOpID, detailKeyBidi, detailKeyDualChan, detailKeyRepetition, detailKeyWideCarrOff, detailKeyLongBlkDist}
	attachmentKeys   = []string{detailKeyOperationID, detailKeyOpType, detailKeyEndpointID, detailKeyEpEui, detailKeyTargetBS, detailKeyTargetBSList, detailKeyTargetBSCnt, detailKeyError}
	vmKeys           = []string{detailKeyEpEui, detailKeyBsEui, detailKeyMacType, detailKeyActiveMacs, detailKeyOpID, detailKeyMessage, detailKeyError}
	downlinkOpKeys   = []string{detailKeyEpEui, detailKeyBsEui, detailKeyQueID, detailKeyOpID, detailKeyOperation, detailKeyStatus, detailKeyBSName, detailKeyTimestamp, detailKeyUserData, detailKeyTxTime, detailKeyPacketCnt, detailKeyResult}
	securityKeys     = []string{detailKeyMethod, detailKeyReason}
	membershipKeys   = []string{detailKeyOrgID, detailKeyUserID, detailKeyRole, detailKeyIsOrgAdmin, detailKeyIsBSAdmin, detailKeyIsEPAdmin}
	organizationKeys = []string{detailKeyOrgID, detailKeyName}
	integrationKeys  = []string{detailKeyIntegration, detailKeyName, detailKeyType, detailKeyStatus, detailKeyOrgID}
	apiKeyKeys       = []string{detailKeyKeyID, detailKeyKeyPrefix, detailKeyKeyType, detailKeyOrgID, detailKeyUserID}
	catalogKeys      = []string{models.EventDetailKeyCatalogID, models.EventDetailKeyCatalogName}
	serviceKeys      = []string{detailKeyService, detailKeyVersion, detailKeyGitCommit, detailKeyHost, detailKeyPorts}
	scaciSessionKeys = []string{detailKeySessionID, detailKeyAcEui, detailKeyRemoteAddr, detailKeyReason}
)

// detailProjection lists, per event type, the detail keys the API exposes.
// Session keys, encrypted material, uplink payloads and attach signatures
// never appear here; a downlink's user data is the tenant's own queued
// payload, which its queue already shows. An event type without an entry
// projects to no details.
var detailProjection = map[string][]string{
	models.EventTypeEndpointCreated:            {detailKeyEpEui},
	models.EventTypeEndpointUpdated:            {detailKeyEpEui},
	models.EventTypeEndpointDeleted:            {detailKeyEpEui, detailKeyEndpointID},
	models.EventTypeEndpointKeysRevealed:       {detailKeyEpEui, detailKeyRevealedKeys, detailKeyServiceAcct},
	models.EventTypeEndpointKeysRemoved:        {detailKeyEpEui, detailKeyRemovedKeys, detailKeyServiceAcct},
	models.EventTypeBSRegistered:               {detailKeyBsEui},
	models.EventTypeBSUpdated:                  {detailKeyBsEui},
	models.EventTypeBSDeregistered:             {detailKeyBsEui, detailKeyBSName, detailKeyBSID},
	models.EventTypeCertificateGenerated:       {detailKeyBsEui, detailKeyValidityDays, detailKeyBSName},
	models.EventTypeCertificateServerGenerated: {detailKeyScope},
	models.EventTypeCertificateServerRenewed:   {detailKeyScope},
	models.EventTypeIntegrationCreated:         integrationKeys,
	models.EventTypeIntegrationUpdated:         integrationKeys,
	models.EventTypeIntegrationDeleted:         integrationKeys,
	models.EventTypeDownlinkQueued:             {detailKeyEpEui, detailKeyBsEui, detailKeyQueID, detailKeyOpID},
	models.EventTypeDownlinkRevokeRequested:    {detailKeyEpEui, detailKeyBsEui, detailKeyQueID},
	models.EventTypeUserCreated:                {detailKeyUserID, detailKeyEmail},
	models.EventTypeUserUpdated:                {detailKeyUserID, detailKeyEmail},
	models.EventTypeUserDeleted:                {detailKeyUserID, detailKeyEmail},
	models.EventTypeOrgCreated:                 organizationKeys,
	models.EventTypeOrgUpdated:                 organizationKeys,
	models.EventTypeOrgDeleted:                 organizationKeys,
	models.EventTypeOrgMemberAdded:             membershipKeys,
	models.EventTypeOrgMemberUpdated:           membershipKeys,
	models.EventTypeOrgMemberRemoved:           membershipKeys,
	models.EventTypeAPIKeyCreated:              apiKeyKeys,
	models.EventTypeAPIKeyDeleted:              apiKeyKeys,
	models.EventTypeManufacturerCreated:        catalogKeys,
	models.EventTypeManufacturerUpdated:        catalogKeys,
	models.EventTypeManufacturerDeleted:        catalogKeys,
	models.EventTypeDeviceModelCreated:         catalogKeys,
	models.EventTypeDeviceModelUpdated:         catalogKeys,
	models.EventTypeDeviceModelDeleted:         catalogKeys,
	models.EventTypeBlueprintCreated:           catalogKeys,
	models.EventTypeBlueprintUpdated:           catalogKeys,
	models.EventTypeBlueprintDeleted:           catalogKeys,
	models.EventTypeAuthInvalidToken:           securityKeys,
	models.EventTypeAuthAPIKeyRejected:         securityKeys,
	models.EventTypeAuthOrgContextMissing:      securityKeys,
	models.EventTypeAuthOrgResolutionFailed:    securityKeys,
	models.EventTypeAuthInternalTrustViolation: securityKeys,
	models.EventTypeAuthPermissionDenied:       securityKeys,
	models.EventTypeAuthLoginFailed:            securityKeys,
	models.EventTypeAuthOIDCExchangeFailed:     securityKeys,
	models.EventTypeServiceStarted:             serviceKeys,
	models.EventTypeSessionPendingOpDropped:    {detailKeyOpID, detailKeyOperation, detailKeyBsEui, detailKeyError},
	models.EventTypeSessionResumeRefused:       {detailKeyBsEui, detailKeyFailureCode, detailKeyMessage},
	models.EventTypeSCACIError:                 {detailKeySessionID, detailKeyCommand, detailKeyOpID, detailKeyErrorCode, detailKeyErrorMsg, detailKeyTime},
	models.EventTypeSCACISessionOpened:         scaciSessionKeys,
	models.EventTypeSCACISessionResumed:        scaciSessionKeys,
	models.EventTypeSCACISessionClosed:         scaciSessionKeys,
	models.EventTypeSCACIConnectRefused:        {detailKeySessionID, detailKeyAcEui, detailKeyRemoteAddr, detailKeyErrorToken, detailKeyMessage, detailKeyCount, detailKeyFirstSeen, detailKeyLastSeen},
	models.EventTypeEndpointAttached:           radioAttachKeys,
	models.EventTypeEndpointDetached:           radioAttachKeys,
	models.EventTypeDetachPropagateCompleted:   propagateKeys,
	models.EventTypeBaseStationOnline:          connectivityKeys,
	models.EventTypeBaseStationOffline:         connectivityKeys,
	models.EventTypeBaseStationPingSent:        {detailKeyBsEui, detailKeyOpID},
	models.EventTypeBaseStationPingAnswered:    {detailKeyBsEui, detailKeyOpID, detailKeyResult},
	models.EventTypeBaseStationStatusAnswered:  {detailKeyBsEui, detailKeyOpID},
	models.EventTypeConnectionError:            {detailKeyBsEui, detailKeyBSName, detailKeyError, detailKeyReason},
	bssci.EventTypeAttachPropagateInitiated:    attachmentKeys,
	bssci.EventTypeAttachPropagateCompleted:    propagateKeys,
	bssci.EventTypeDetachPropagateInitiated:    attachmentKeys,
	bssci.EventTypeAttachPropagateFailed:       propagateKeys,
	bssci.EventTypeDetachPropagateFailed:       propagateKeys,
	bssci.EventTypeEndpointAttachFailed:        attachmentKeys,
	bssci.EventTypeEndpointDetachFailed:        attachmentKeys,
	bssci.EventTypeAttachOperationFailed:       {detailKeyEpEui, detailKeyBsEui, detailKeyShAddr, detailKeyReason},
	bssci.EventTypeDetachOperationFailed:       {detailKeyEpEui, detailKeyReason},
	bssci.EventTypeVMActivateSuccess:           vmKeys,
	bssci.EventTypeVMActivateFailed:            vmKeys,
	bssci.EventTypeVMDeactivateSuccess:         vmKeys,
	bssci.EventTypeVMStatusReceived:            vmKeys,
	bssci.EventTypeDLRxStatusReceived:          {detailKeyEpEui, detailKeyBsEui, detailKeyBSName, detailKeyRxTime, detailKeyPacketCnt, detailKeyDlRxSnr, detailKeyDlRxRssi, detailKeyOpID, detailKeyTimestamp},
	models.EventTypeDLDataRevokeInitiated:      downlinkOpKeys,
	models.EventTypeDLDataRevoked:              downlinkOpKeys,
	models.EventTypeDLDataQueueAcknowledged:    downlinkOpKeys,
	models.EventTypeDLDataSent:                 downlinkOpKeys,
	models.EventTypeDLDataExpired:              downlinkOpKeys,
	models.EventTypeDLDataInvalid:              downlinkOpKeys,
	models.EventTypeDLDataUnknownResult:        downlinkOpKeys,
	models.EventTypeDLDataAcknowledged:         downlinkOpKeys,
	models.EventTypeDLDataEnqueued:             downlinkOpKeys,
	models.EventTypeDLDataUpdated:              downlinkOpKeys,
	models.EventTypeDLDataRequeued:             downlinkOpKeys,

	models.EventTypeCertificatePrivateKeyDownloaded: {detailKeyBsEui, detailKeyServiceAcct},
}

// emptyDetails is what an event without an allowlisted key projects to.
var emptyDetails = json.RawMessage(`{}`)

// projectDetails keeps only the allowlisted keys of an event's details; an
// unknown event type or unparsable details project to an empty object.
func projectDetails(eventType string, details json.RawMessage) json.RawMessage {
	allowed, ok := detailProjection[eventType]
	if !ok || len(details) == 0 {
		return emptyDetails
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(details, &raw); err != nil {
		return emptyDetails
	}
	projected := make(map[string]json.RawMessage, len(allowed))
	for _, key := range allowed {
		if value, present := raw[key]; present {
			projected[key] = value
		}
	}
	if len(projected) == 0 {
		return emptyDetails
	}
	out, err := json.Marshal(projected)
	if err != nil {
		return emptyDetails
	}
	return out
}
