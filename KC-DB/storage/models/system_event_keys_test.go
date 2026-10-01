package models

import (
	"regexp"
	"testing"
)

// Event details reach the browser as written, so every key a writer names
// through these constants is camelCase.
func TestEventDetailKeysAreCamelCase(t *testing.T) {
	camelCase := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	for _, key := range []string{
		EventDetailKeyOpID, EventDetailKeyQueID, EventDetailKeyEpEui, EventDetailKeyBsEui, EventDetailKeyBaseStationEui,
		EventDetailKeyBaseStationName, EventDetailKeyOperation, EventDetailKeyOperationID, EventDetailKeyOperationType,
		EventDetailKeyEndpointID, EventDetailKeyRevealedKeys, EventDetailKeyRemovedKeys, EventDetailKeyServiceAccount,
		EventDetailKeyBaseStationID, EventDetailKeyStatus, EventDetailKeyTimestamp, EventDetailKeyTime, EventDetailKeyRxTime,
		EventDetailKeyTxTime, EventDetailKeyUserData, EventDetailKeyResult, EventDetailKeyPacketCnt, EventDetailKeyDlRxSnr,
		EventDetailKeyDlRxRssi, EventDetailKeyRssi, EventDetailKeySnr, EventDetailKeyAttachCnt, EventDetailKeyShAddr,
		EventDetailKeyShortAddr, EventDetailKeyMacType, EventDetailKeyActiveMacTypes, EventDetailKeyMessage, EventDetailKeyError,
		EventDetailKeyReason, EventDetailKeySuccess, EventDetailKeyFailureCode, EventDetailKeyTenantID, EventDetailKeyTargetBS,
		EventDetailKeyTargetBSList, EventDetailKeyTargetBSCount, EventDetailKeyIsOnline, EventDetailKeyConnectionType,
		EventDetailKeySessionID, EventDetailKeyAcEui, EventDetailKeyRemoteAddr, EventDetailKeyErrorToken, EventDetailKeyCount,
		EventDetailKeyFirstSeen, EventDetailKeyLastSeen,
	} {
		if !camelCase.MatchString(key) {
			t.Errorf("event detail key %q is not camelCase", key)
		}
	}
}
