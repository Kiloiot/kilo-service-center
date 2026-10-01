package postgres

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const deviceEUIEventTenant = int64(901)

// Device timelines match the EUIs an event's details name whatever form the
// writer gave them: lowercase, dashed or numeric, for the station and the endpoint.
func TestGetEvents_DeviceScopesMatchTheEUIsAWriterGaveInAnyForm(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSystemEventsTestDB(t)
	createSystemEventsTestTenant(t, db, deviceEUIEventTenant, "DeviceEUIEvents")
	store := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	for _, details := range []string{
		`{"bsEui":"70b3d59cd00009e6","epEui":"70-b3-d5-67-70-11-ff-01"}`,
		`{"bsEui":8121069422560414182,"epEui":8121069422560414182}`,
	} {
		require.NoError(t, store.CreateEvent(ctx, &models.SystemEvent{
			TenantID: "901", EventType: "detach_propagate_completed", Category: "endpoint", Severity: "info",
			SourceType: "service_center", SourceName: "writer", Title: "Endpoint detached",
			Details: json.RawMessage(details),
		}))
	}

	for _, filter := range []models.SystemEventFilter{
		{TenantID: "901", BaseStationEUI: "70B3D59CD00009E6"},
		{TenantID: "901", BaseStationEUI: "70-b3-d5-9c-d0-00-09-e6"},
	} {
		filter.Limit = scopeTestPage
		events, err := store.GetEvents(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, events, 2, "the station's timeline holds both events for filter %q", filter.BaseStationEUI)
	}
	endpoint, err := store.GetEvents(ctx, models.SystemEventFilter{TenantID: "901", EndpointEUI: "70B3D5677011FF01", Limit: scopeTestPage})
	require.NoError(t, err)
	require.Len(t, endpoint, 1)

	var stored map[string]string
	require.NoError(t, json.Unmarshal(endpoint[0].Details, &stored))
	assert.Equal(t, map[string]string{"bsEui": "70B3D59CD00009E6", "epEui": "70B3D5677011FF01"}, stored,
		"the details keep one canonical form")
}

func TestCanonicalEventDetails_KeepsWhatIsNotADeviceEUI(t *testing.T) {
	for _, details := range []string{
		`[1,2]`,
		`{"bsEui":"not-an-eui","epEui":true}`,
		`{"bsEui":"70B3D59CD00009E6","opId":-7}`,
	} {
		assert.JSONEq(t, details, string(canonicalEventDetails([]byte(details))))
	}
}
