package resilience

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"
)

// retryableSnapshot pins the policy's output: a method joining or leaving the
// retry set changes client behaviour and must be a deliberate edit here.
var retryableSnapshot = []string{
	"/kilocenter.api.v1.CoreService/DecodePreview",
	"/kilocenter.api.v1.CoreService/ExportBaseStationMessages",
	"/kilocenter.api.v1.CoreService/GetActivityAnalytics",
	"/kilocenter.api.v1.CoreService/GetAlertSummary",
	"/kilocenter.api.v1.CoreService/GetAnalyticsOverview",
	"/kilocenter.api.v1.CoreService/GetBaseStation",
	"/kilocenter.api.v1.CoreService/GetBaseStationAvailability",
	"/kilocenter.api.v1.CoreService/GetBaseStationMessage",
	"/kilocenter.api.v1.CoreService/GetBaseStationMessageStats",
	"/kilocenter.api.v1.CoreService/GetBaseStationMessagesReceived",
	"/kilocenter.api.v1.CoreService/GetBaseStationStats",
	"/kilocenter.api.v1.CoreService/GetBlueprint",
	"/kilocenter.api.v1.CoreService/GetCEStatus",
	"/kilocenter.api.v1.CoreService/GetDLRXStatus",
	"/kilocenter.api.v1.CoreService/GetDLRXStatusQueries",
	"/kilocenter.api.v1.CoreService/GetDeviceModel",
	"/kilocenter.api.v1.CoreService/GetDiagnosticsBundle",
	"/kilocenter.api.v1.CoreService/GetDownlinkResults",
	"/kilocenter.api.v1.CoreService/GetEndPoint",
	"/kilocenter.api.v1.CoreService/GetEndPointOperations",
	"/kilocenter.api.v1.CoreService/GetEndPointStats",
	"/kilocenter.api.v1.CoreService/GetIntegration",
	"/kilocenter.api.v1.CoreService/GetManufacturer",
	"/kilocenter.api.v1.CoreService/GetMessage",
	"/kilocenter.api.v1.CoreService/GetReleaseInfo",
	"/kilocenter.api.v1.CoreService/GetScaciSession",
	"/kilocenter.api.v1.CoreService/GetScaciStatistics",
	"/kilocenter.api.v1.CoreService/GetScaciStatus",
	"/kilocenter.api.v1.CoreService/GetServerCertificateStatus",
	"/kilocenter.api.v1.CoreService/GetSignalQualityAnalytics",
	"/kilocenter.api.v1.CoreService/GetStatistics",
	"/kilocenter.api.v1.CoreService/GetSystemStatus",
	"/kilocenter.api.v1.CoreService/ListAlerts",
	"/kilocenter.api.v1.CoreService/ListAllBaseStationLocations",
	"/kilocenter.api.v1.CoreService/ListBaseStationActivity",
	"/kilocenter.api.v1.CoreService/ListBaseStationMessages",
	"/kilocenter.api.v1.CoreService/ListBaseStations",
	"/kilocenter.api.v1.CoreService/ListBlueprints",
	"/kilocenter.api.v1.CoreService/ListCEInstances",
	"/kilocenter.api.v1.CoreService/ListCapabilities",
	"/kilocenter.api.v1.CoreService/ListDeviceModels",
	"/kilocenter.api.v1.CoreService/ListDownlinkQueue",
	"/kilocenter.api.v1.CoreService/ListEndPoints",
	"/kilocenter.api.v1.CoreService/ListEndpointActivity",
	"/kilocenter.api.v1.CoreService/ListEndpointMessages",
	"/kilocenter.api.v1.CoreService/ListErrorGroups",
	"/kilocenter.api.v1.CoreService/ListEvents",
	"/kilocenter.api.v1.CoreService/ListIntegrations",
	"/kilocenter.api.v1.CoreService/ListManufacturers",
	"/kilocenter.api.v1.CoreService/ListMessages",
	"/kilocenter.api.v1.CoreService/ListScaciErrors",
	"/kilocenter.api.v1.CoreService/ListScaciQueues",
	"/kilocenter.api.v1.CoreService/ListScaciSessions",
	"/kilocenter.api.v1.CoreService/QueryDLRXStatus",
	"/kilocenter.api.v1.CoreService/SearchBaseStationMessages",
	"/kilocenter.api.v1.IdentityService/GetApiKey",
	"/kilocenter.api.v1.IdentityService/GetAuthSettings",
	"/kilocenter.api.v1.IdentityService/GetOrganization",
	"/kilocenter.api.v1.IdentityService/GetOrganizationUser",
	"/kilocenter.api.v1.IdentityService/GetProfile",
	"/kilocenter.api.v1.IdentityService/GetUser",
	"/kilocenter.api.v1.IdentityService/ListApiKeys",
	"/kilocenter.api.v1.IdentityService/ListOrganizationUsers",
	"/kilocenter.api.v1.IdentityService/ListOrganizations",
	"/kilocenter.api.v1.IdentityService/ListUserOrganizations",
	"/kilocenter.api.v1.IdentityService/ListUsers",
	"/kilocenter.api.v1.KiloCenterService/DecodePreview",
	"/kilocenter.api.v1.KiloCenterService/ExportBaseStationMessages",
	"/kilocenter.api.v1.KiloCenterService/GetActivityAnalytics",
	"/kilocenter.api.v1.KiloCenterService/GetAlertSummary",
	"/kilocenter.api.v1.KiloCenterService/GetAnalyticsOverview",
	"/kilocenter.api.v1.KiloCenterService/GetApiKey",
	"/kilocenter.api.v1.KiloCenterService/GetAuthSettings",
	"/kilocenter.api.v1.KiloCenterService/GetBaseStation",
	"/kilocenter.api.v1.KiloCenterService/GetBaseStationAvailability",
	"/kilocenter.api.v1.KiloCenterService/GetBaseStationMessage",
	"/kilocenter.api.v1.KiloCenterService/GetBaseStationMessageStats",
	"/kilocenter.api.v1.KiloCenterService/GetBaseStationMessagesReceived",
	"/kilocenter.api.v1.KiloCenterService/GetBaseStationStats",
	"/kilocenter.api.v1.KiloCenterService/GetBlueprint",
	"/kilocenter.api.v1.KiloCenterService/GetCEStatus",
	"/kilocenter.api.v1.KiloCenterService/GetDLRXStatus",
	"/kilocenter.api.v1.KiloCenterService/GetDLRXStatusQueries",
	"/kilocenter.api.v1.KiloCenterService/GetDeviceModel",
	"/kilocenter.api.v1.KiloCenterService/GetDiagnosticsBundle",
	"/kilocenter.api.v1.KiloCenterService/GetDownlinkResults",
	"/kilocenter.api.v1.KiloCenterService/GetEndPoint",
	"/kilocenter.api.v1.KiloCenterService/GetEndPointOperations",
	"/kilocenter.api.v1.KiloCenterService/GetEndPointStats",
	"/kilocenter.api.v1.KiloCenterService/GetIntegration",
	"/kilocenter.api.v1.KiloCenterService/GetManufacturer",
	"/kilocenter.api.v1.KiloCenterService/GetMessage",
	"/kilocenter.api.v1.KiloCenterService/GetOrganization",
	"/kilocenter.api.v1.KiloCenterService/GetOrganizationUser",
	"/kilocenter.api.v1.KiloCenterService/GetProfile",
	"/kilocenter.api.v1.KiloCenterService/GetReleaseInfo",
	"/kilocenter.api.v1.KiloCenterService/GetScaciSession",
	"/kilocenter.api.v1.KiloCenterService/GetScaciStatistics",
	"/kilocenter.api.v1.KiloCenterService/GetScaciStatus",
	"/kilocenter.api.v1.KiloCenterService/GetServerCertificateStatus",
	"/kilocenter.api.v1.KiloCenterService/GetSignalQualityAnalytics",
	"/kilocenter.api.v1.KiloCenterService/GetStatistics",
	"/kilocenter.api.v1.KiloCenterService/GetSystemStatus",
	"/kilocenter.api.v1.KiloCenterService/GetUser",
	"/kilocenter.api.v1.KiloCenterService/ListAlerts",
	"/kilocenter.api.v1.KiloCenterService/ListAllBaseStationLocations",
	"/kilocenter.api.v1.KiloCenterService/ListApiKeys",
	"/kilocenter.api.v1.KiloCenterService/ListBaseStationActivity",
	"/kilocenter.api.v1.KiloCenterService/ListBaseStationMessages",
	"/kilocenter.api.v1.KiloCenterService/ListBaseStations",
	"/kilocenter.api.v1.KiloCenterService/ListBlueprints",
	"/kilocenter.api.v1.KiloCenterService/ListCEInstances",
	"/kilocenter.api.v1.KiloCenterService/ListCapabilities",
	"/kilocenter.api.v1.KiloCenterService/ListDeviceModels",
	"/kilocenter.api.v1.KiloCenterService/ListDownlinkQueue",
	"/kilocenter.api.v1.KiloCenterService/ListEndPoints",
	"/kilocenter.api.v1.KiloCenterService/ListEndpointActivity",
	"/kilocenter.api.v1.KiloCenterService/ListEndpointMessages",
	"/kilocenter.api.v1.KiloCenterService/ListErrorGroups",
	"/kilocenter.api.v1.KiloCenterService/ListEvents",
	"/kilocenter.api.v1.KiloCenterService/ListIntegrations",
	"/kilocenter.api.v1.KiloCenterService/ListManufacturers",
	"/kilocenter.api.v1.KiloCenterService/ListMessages",
	"/kilocenter.api.v1.KiloCenterService/ListOrganizationUsers",
	"/kilocenter.api.v1.KiloCenterService/ListOrganizations",
	"/kilocenter.api.v1.KiloCenterService/ListScaciErrors",
	"/kilocenter.api.v1.KiloCenterService/ListScaciQueues",
	"/kilocenter.api.v1.KiloCenterService/ListScaciSessions",
	"/kilocenter.api.v1.KiloCenterService/ListUserOrganizations",
	"/kilocenter.api.v1.KiloCenterService/ListUsers",
	"/kilocenter.api.v1.KiloCenterService/QueryDLRXStatus",
	"/kilocenter.api.v1.KiloCenterService/SearchBaseStationMessages",
}

// retryableSet returns the policy's output for every gateway service as full
// method names.
func retryableSet() map[string]bool {
	set := make(map[string]bool)
	for _, name := range retryableMethods(rpccatalog.Proxied()...) {
		set["/"+name.Service+"/"+name.Method] = true
	}
	return set
}

func TestRetryableMethods_MatchesSnapshot(t *testing.T) {
	got := make([]string, 0)
	for method := range retryableSet() {
		got = append(got, method)
	}
	sort.Strings(got)
	assert.Equal(t, retryableSnapshot, got)
}

func TestRetryableMethods_NoStreamingOverlap(t *testing.T) {
	derived := retryableSet()
	streams := 0
	for _, desc := range rpccatalog.Proxied() {
		for _, stream := range desc.Streams {
			streams++
			assert.False(t, derived[rpccatalog.FullMethod(desc, stream.StreamName)],
				"streaming method %q must not be retryable", stream.StreamName)
		}
	}
	require.NotZero(t, streams, "the descriptors must declare streams for this test to mean anything")
}

func TestRetryableMethods_OnlyIdempotentMethods(t *testing.T) {
	for method := range retryableSet() {
		short := method[strings.LastIndex(method, "/")+1:]
		assert.True(t, isRetryable(short), "method %q does not match the retry policy", method)
	}
}

func TestRetryableMethods_Membership(t *testing.T) {
	derived := retryableSet()
	assert.True(t, derived["/kilocenter.api.v1.CoreService/GetEndPoint"])
	assert.True(t, derived["/kilocenter.api.v1.CoreService/ListEndPoints"])
	assert.True(t, derived["/kilocenter.api.v1.CoreService/DecodePreview"])
	assert.False(t, derived["/kilocenter.api.v1.CoreService/CreateEndPoint"])
	assert.False(t, derived["/kilocenter.api.v1.CoreService/DeleteEndPoint"])
	assert.False(t, derived["/kilocenter.api.v1.CoreService/StreamEvents"])
}

func TestBuildRetryServiceConfig_ValidJSON(t *testing.T) {
	cfg := config.GatewayResilienceConfig{
		MaxRetries:      testMaxRetries,
		RetryBackoff:    testRetryBackoff,
		RetryMaxBackoff: testRetryMaxBackoff,
	}
	sc, err := BuildRetryServiceConfig(cfg, rpccatalog.Proxied()...)
	require.NoError(t, err)
	var parsed serviceConfig
	require.NoError(t, json.Unmarshal([]byte(sc), &parsed))
	require.Len(t, parsed.MethodConfig, 1)
	assert.Len(t, parsed.MethodConfig[0].Name, len(retryableSnapshot))
	assert.Contains(t, parsed.MethodConfig[0].Name, methodName{Service: "kilocenter.api.v1.CoreService", Method: "GetEndPoint"})
	assert.Equal(t, int(testMaxRetries)+1, parsed.MethodConfig[0].RetryPolicy.MaxAttempts)
	assert.Equal(t, []string{statusCodeUnavailable}, parsed.MethodConfig[0].RetryPolicy.RetryableStatusCodes)
}
