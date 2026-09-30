package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// registrationDeletedAgo is how long before the current registration the
// endpoint's earlier registration queued its downlink.
const registrationDeletedAgo = 24 * time.Hour

// scopeTestPage is more events than any scope test stores.
const scopeTestPage = 10

func endpointFilterEUI() *[8]byte {
	eui := [8]byte{}
	copy(eui[:], mioty.EUI64Bytes(fixtureEndpointEUI))
	return &eui
}

// A downlink an earlier registration of the EUI queued stays with it: the
// queue and results listed from the current registration leave it out.
func TestDownlinkListings_StartAtTheRegistration(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	for _, queID := range []int64{860001, 860002} {
		_, err := repos.Downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], queID, nil), testDownlinkLifetime)
		require.NoError(t, err)
	}
	_, err := db.Exec(`UPDATE downlink_queue SET created_at = NOW() - $1::interval WHERE que_id = 860001`, registrationDeletedAgo.String())
	require.NoError(t, err)
	registeredAt := time.Now().Add(-time.Hour)

	queue, err := repos.DownlinkQueueReader.ListTenantQueue(ctx, 321,
		storage.DownlinkQueueFilter{EpEUI: endpointFilterEUI(), QueuedFrom: &registeredAt}, 10, 0)
	require.NoError(t, err)
	require.Len(t, queue, 1)
	assert.Equal(t, int64(860002), queue[0].QueID)

	_, err = db.Exec(`UPDATE downlink_queue SET status = 'expired' WHERE que_id IN (860001, 860002)`)
	require.NoError(t, err)
	results, _, err := repos.Downlinks.GetDownlinkResults(ctx, 321, nil,
		storage.DownlinkResultFilter{QueuedFrom: &registeredAt}, 10, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(860002), results[0].QueID)
}

// A station's timeline holds every event that names it in its details, like
// a detach propagate filed under the endpoint, not only those it is the source of.
func TestGetEvents_StationScopeFindsTheEventsNamingTheStation(t *testing.T) {
	_, db, _ := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	_, err := db.Exec(`INSERT INTO system_events (tenant_id, event_type, event_category, severity, source_type,
		source_name, title, description, data, status, occurred_at, recorded_at)
		VALUES (321, 'detach_propagate_completed', 'endpoint', 'info', 'endpoint', '70B3D5677011FF01',
		        'Endpoint detached', 'detached', '{"bsEui":"70B3D59CD00009E6","epEui":"70B3D5677011FF01"}', 'new', NOW(), NOW())`)
	require.NoError(t, err)
	store := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get())

	events, err := store.GetEvents(ctx, models.SystemEventFilter{TenantID: "321", BaseStationEUI: "70B3D59CD00009E6", Limit: scopeTestPage})
	require.NoError(t, err)
	total, err := store.CountEvents(ctx, models.SystemEventFilter{TenantID: "321", BaseStationEUI: "70B3D59CD00009E6"})
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.Equal(t, "detach_propagate_completed", events[0].EventType)
	assert.Equal(t, int64(1), total, "the count agrees with the listing")
}

// Cross-tenant guard: a tenant that registers an EUI another tenant used never
// sees that tenant's downlink queue, downlink results or events.
func TestDownlinksAndEvents_OfAReRegisteredEUIStayWithTheirTenant(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	_, err := repos.Downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 861001, nil), testDownlinkLifetime)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO system_events (tenant_id, event_type, event_category, severity, source_type,
		source_name, title, description, status, occurred_at, recorded_at)
		VALUES (321, 'dl_data_sent', 'message', 'info', 'endpoint', $1, 'Downlink sent', 'org A', 'new', NOW(), NOW())`,
		"70B3D59CD0000321")
	require.NoError(t, err)

	queue, err := repos.DownlinkQueueReader.ListTenantQueue(ctx, 322,
		storage.DownlinkQueueFilter{EpEUI: endpointFilterEUI()}, scopeTestPage, 0)
	require.NoError(t, err)
	assert.Empty(t, queue, "another tenant's queue is never listed")
	queued, err := repos.DownlinkQueueReader.CountTenantQueue(ctx, 322, storage.DownlinkQueueFilter{EpEUI: endpointFilterEUI()})
	require.NoError(t, err)
	assert.Zero(t, queued, "another tenant's queue is never counted")

	_, err = db.Exec(`UPDATE downlink_queue SET status = 'expired' WHERE que_id = 861001`)
	require.NoError(t, err)
	results, total, err := repos.Downlinks.GetDownlinkResults(ctx, 322, nil,
		storage.DownlinkResultFilter{EpEUI: mioty.EUI64Bytes(fixtureEndpointEUI)}, scopeTestPage, 0)
	require.NoError(t, err)
	assert.Empty(t, results, "another tenant's results are never listed")
	assert.Zero(t, total, "another tenant's results are never counted")
	own, _, err := repos.Downlinks.GetDownlinkResults(ctx, 321, nil,
		storage.DownlinkResultFilter{EpEUI: mioty.EUI64Bytes(fixtureEndpointEUI)}, scopeTestPage, 0)
	require.NoError(t, err)
	assert.Len(t, own, 1, "the owning tenant still lists its result")

	events, err := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get()).GetEvents(ctx,
		models.SystemEventFilter{TenantID: "322", EndpointEUI: "70B3D59CD0000321", Limit: scopeTestPage})
	require.NoError(t, err)
	assert.Empty(t, events, "another tenant's events are never listed")
}
