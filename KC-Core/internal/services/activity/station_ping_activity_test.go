package activity

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/adapters"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	pingActivityTenant   int64 = 7301
	pingActivityOpID     int64 = -23
	pingActivityOperator       = "be8a02c9-1470-4314-8a86-48712761aca9"
	pingActivityPoll           = time.Second
	pingActivityOverlap        = time.Second
)

var (
	pingActivityStation = [8]byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x73, 0x01}
	pingActivityOther   = [8]byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x73, 0x02}
)

// The station's activity feed returns the pings recorded through the station
// event recorder, and only that station's.
func TestListBaseStationActivity_ReturnsRecordedPings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := testsupport.SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	_, err := db.Exec(`INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW()) ON CONFLICT (id) DO NOTHING`,
		pingActivityTenant, "ping-activity-"+strconv.FormatInt(pingActivityTenant, 10), t.Name())
	require.NoError(t, err)

	log := logger.NewNop()
	stations := postgres.NewBaseStationRepository(db, clock.SystemClock{}, log)
	eventStore := postgres.NewSystemEventStore(db.DB, clock.SystemClock{}, log)
	recorder := basestation.NewPersistentEventRecorder(log, eventStore, strconv.FormatInt(pingActivityTenant, 10))
	stationCtx := pkgcontext.WithTenantID(testutil.TestContext(), pingActivityTenant)
	operatorCtx := pkgcontext.WithUserID(stationCtx, pingActivityOperator)

	require.NoError(t, recorder.RecordEvent(operatorCtx, pingActivityStation, models.EventTypeBaseStationPingSent, time.Now(),
		map[string]interface{}{models.EventDetailKeyOpID: pingActivityOpID}))
	require.NoError(t, recorder.RecordEvent(stationCtx, pingActivityStation, models.EventTypeBaseStationPingAnswered, time.Now(),
		map[string]interface{}{models.EventDetailKeyOpID: pingActivityOpID}))
	require.NoError(t, recorder.RecordEvent(operatorCtx, pingActivityOther, models.EventTypeBaseStationPingSent, time.Now(),
		map[string]interface{}{models.EventDetailKeyOpID: pingActivityOpID}))

	events := eventsservice.New(adapters.NewSystemEventStoreAdapter(eventStore),
		adapters.NewEUIResolver(stations, nil), nil, pingActivityPoll, pingActivityOverlap, streamwake.NewSignal(), 0, log)
	svc := New(events, noMessages{}, log)
	ctx := authz.WithRoles(testutil.TestContextWithTenant(pingActivityTenant), authz.Roles{BaseStationManager: true})

	result, err := svc.ListBaseStationActivity(ctx, pingActivityTenant, pingActivityStation[:], nil, 0, "")

	require.NoError(t, err)
	byType := map[string]map[string]any{}
	for _, item := range result.Items {
		require.NotNil(t, item.Event, "the feed holds only the station's events")
		var details map[string]any
		require.NoError(t, json.Unmarshal(item.Event.Data, &details))
		byType[item.Event.EventType] = details
		if item.Event.EventType == models.EventTypeBaseStationPingSent {
			assert.Equal(t, pingActivityOperator, item.Event.UserID, "the ping names the operator who sent it")
		}
	}
	require.Len(t, byType, 2, "both pings of the station and none of the other station")
	for _, eventType := range []string{models.EventTypeBaseStationPingSent, models.EventTypeBaseStationPingAnswered} {
		require.Contains(t, byType, eventType)
		assert.EqualValues(t, pingActivityOpID, byType[eventType][models.EventDetailKeyOpID])
		assert.Contains(t, byType[eventType], models.EventDetailKeyBsEui)
	}
	assert.Equal(t, int64(2), result.TotalCount)
}
