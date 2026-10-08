package events

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func TestProjectDetails_KeepsOnlyAllowlistedKeys(t *testing.T) {
	raw := json.RawMessage(`{"epEui":"70B3D59CD0000002","bsEui":"70B3D59CD0000001","nwkSnKey":[1,2,3],"encryptedKey":"x","shAddr":5,"success":true,"opId":-4}`)
	got := projectDetails(bssci.EventTypeAttachPropagateCompleted, raw)
	assert.JSONEq(t, `{"epEui":"70B3D59CD0000002","bsEui":"70B3D59CD0000001","shAddr":5,"success":true,"opId":-4}`, string(got))
}

func TestProjectDetails_DropsAttachSignaturesAndPayloads(t *testing.T) {
	attach := json.RawMessage(`{"epEui":"AA","attachCnt":3,"nonce":[9],"sign":[8],"rssi":-70,"subpackets":[{}]}`)
	assert.JSONEq(t, `{"epEui":"AA","attachCnt":3,"rssi":-70}`, string(projectDetails(models.EventTypeEndpointAttached, attach)))

	vm := json.RawMessage(`{"epEui":"AA","macType":1,"userData":"ff","data":"00","opId":2}`)
	assert.JSONEq(t, `{"epEui":"AA","macType":1,"opId":2}`, string(projectDetails(bssci.EventTypeVMActivateSuccess, vm)))
}

func TestProjectDetails_RefusedConnectKeepsItsCatalogError(t *testing.T) {
	raw := json.RawMessage(`{"acEui":"70-B3-D5-00-00-00-00-01","remoteAddr":"192.0.2.10:40000","errorToken":"scaci.error.x","message":"refused","count":3,"firstSeen":"t0","lastSeen":"t2","cert":"pem"}`)
	assert.JSONEq(t, `{"acEui":"70-B3-D5-00-00-00-00-01","remoteAddr":"192.0.2.10:40000","errorToken":"scaci.error.x","message":"refused","count":3,"firstSeen":"t0","lastSeen":"t2"}`,
		string(projectDetails(models.EventTypeSCACIConnectRefused, raw)))
}

// A connectivity event and a propagate as migration 179 leaves them project
// their camelCase details, the keys the writers use.
func TestProjectDetails_ReadsMigratedCamelCaseRows(t *testing.T) {
	online := json.RawMessage(`{"baseStationName":"tims base","bsEui":"70B3D59CD00179A1","connectionType":"bssci","isOnline":true,"sessionID":"s-1"}`)
	assert.JSONEq(t, `{"baseStationName":"tims base","bsEui":"70B3D59CD00179A1","connectionType":"bssci","isOnline":true}`,
		string(projectDetails(models.EventTypeBaseStationOnline, online)))

	propagate := json.RawMessage(`{"operationId":"attach-7","operationType":"attach","endpointId":7,"epEui":"AA","targetBs":"none","targetBsList":[],"targetBsCount":0}`)
	assert.JSONEq(t, string(propagate), string(projectDetails(bssci.EventTypeAttachPropagateInitiated, propagate)))
}

// Every key the browser can receive is camelCase.
func TestDetailProjection_ExposesOnlyCamelCaseKeys(t *testing.T) {
	camelCase := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	for eventType, keys := range detailProjection {
		for _, key := range keys {
			assert.Regexp(t, camelCase, key, "event type %q exposes key %q", eventType, key)
		}
	}
}

func TestProjectDetails_UnknownTypeAndBadInputProjectToEmpty(t *testing.T) {
	assert.Equal(t, `{}`, string(projectDetails("something.new", json.RawMessage(`{"secret":1}`))))
	assert.Equal(t, `{}`, string(projectDetails(models.EventTypeEndpointCreated, nil)))
	assert.Equal(t, `{}`, string(projectDetails(models.EventTypeEndpointCreated, json.RawMessage(`not json`))))
	assert.Equal(t, `{}`, string(projectDetails(models.EventTypeEndpointCreated, json.RawMessage(`{"unrelated":true}`))))
}

func TestDetailProjection_CoversEveryModelEventType(t *testing.T) {
	// Types written without details stay out of the map on purpose.
	undetailed := map[string]bool{
		models.EventTypeServiceStopped:   true,
		models.EventTypeMigrationApplied: true,
		models.EventTypeMigrationFailed:  true,
	}
	for _, eventType := range []string{
		models.EventTypeEndpointCreated, models.EventTypeEndpointUpdated, models.EventTypeEndpointDeleted,
		models.EventTypeBSRegistered, models.EventTypeBSUpdated, models.EventTypeBSDeregistered,
		models.EventTypeUserCreated, models.EventTypeUserUpdated, models.EventTypeUserDeleted,
		models.EventTypeOrgCreated, models.EventTypeOrgUpdated, models.EventTypeOrgDeleted,
		models.EventTypeOrgMemberAdded, models.EventTypeOrgMemberUpdated, models.EventTypeOrgMemberRemoved,
		models.EventTypeDownlinkQueued, models.EventTypeDownlinkRevokeRequested,
		models.EventTypeAPIKeyCreated, models.EventTypeAPIKeyDeleted,
		models.EventTypeManufacturerCreated, models.EventTypeManufacturerUpdated, models.EventTypeManufacturerDeleted,
		models.EventTypeDeviceModelCreated, models.EventTypeDeviceModelUpdated, models.EventTypeDeviceModelDeleted,
		models.EventTypeBlueprintCreated, models.EventTypeBlueprintUpdated, models.EventTypeBlueprintDeleted,
		models.EventTypeIntegrationCreated, models.EventTypeIntegrationUpdated, models.EventTypeIntegrationDeleted,
		models.EventTypeCertificateGenerated, models.EventTypeCertificateServerGenerated, models.EventTypeCertificateServerRenewed,
		models.EventTypeCertificatePrivateKeyDownloaded,
		models.EventTypeAuthInvalidToken, models.EventTypeAuthAPIKeyRejected, models.EventTypeAuthOrgContextMissing,
		models.EventTypeAuthOrgResolutionFailed, models.EventTypeAuthInternalTrustViolation, models.EventTypeAuthPermissionDenied,
		models.EventTypeAuthLoginFailed, models.EventTypeAuthOIDCExchangeFailed,
		models.EventTypeServiceStarted, models.EventTypeSessionPendingOpDropped, models.EventTypeSessionResumeRefused, models.EventTypeSCACIError,
		models.EventTypeEndpointAttached, models.EventTypeEndpointDetached, models.EventTypeDetachPropagateCompleted,
		models.EventTypeBaseStationOnline, models.EventTypeBaseStationOffline, models.EventTypeConnectionError,
		models.EventTypeBaseStationPingSent, models.EventTypeBaseStationPingAnswered, models.EventTypeBaseStationStatusAnswered,
		models.EventTypeSCACISessionOpened, models.EventTypeSCACISessionResumed, models.EventTypeSCACISessionClosed,
		models.EventTypeSCACIConnectRefused,
	} {
		if undetailed[eventType] {
			continue
		}
		_, ok := detailProjection[eventType]
		assert.True(t, ok, "event type %q has no detail projection", eventType)
	}
}
