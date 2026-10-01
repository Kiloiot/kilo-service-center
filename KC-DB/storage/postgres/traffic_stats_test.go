package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// trafficNow is a Wednesday, so this week, this month and earlier differ.
var trafficNow = time.Date(2026, time.September, 16, 12, 0, 0, 0, time.UTC)

const (
	trafficStationOneRssi = -80.0
	trafficStationTwoRssi = -60.0
	trafficStationTwoSnr  = 20.0
	trafficDetachRssi     = -90.0
	trafficDetachSnr      = 4.5
	trafficRadioSkew      = 3 * time.Millisecond
)

// trafficFixture holds the rx times of the uplinks seeded by seedTraffic.
type trafficFixture struct {
	*uplinkStoreFixture
	repo                               *MessageRepository
	today, thisWeek, thisMonth, before time.Time
	detach                             time.Time
}

// seedTraffic stores four uplinks the way BSSCI stores them, a detach and
// the propagations the service center sends: the uplink today is heard by
// both stations, the one before this month by station two only.
func seedTraffic(t *testing.T) *trafficFixture {
	t.Helper()
	f := &trafficFixture{
		uplinkStoreFixture: newUplinkStoreFixture(t),
		today:              trafficNow.Add(-time.Hour),
		thisWeek:           trafficNow.AddDate(0, 0, -2),
		thisMonth:          trafficNow.AddDate(0, 0, -10),
		before:             trafficNow.AddDate(0, 0, -40),
		detach:             trafficNow.Add(-30 * time.Minute),
	}
	f.repo = NewMessageRepository(f.db, f.clock, logger.Get())
	f.persistUplink(t, 1, uplinkStoreBsEuiOne, f.today, trafficStationOneRssi)
	f.persistUplink(t, 1, uplinkStoreBsEuiTwo, f.today.Add(trafficRadioSkew), trafficStationTwoRssi)
	f.persistUplink(t, 2, uplinkStoreBsEuiOne, f.thisWeek, trafficStationOneRssi)
	f.persistUplink(t, 3, uplinkStoreBsEuiOne, f.thisMonth, trafficStationOneRssi)
	f.persistUplink(t, 4, uplinkStoreBsEuiTwo, f.before, trafficStationTwoRssi)
	f.storeControlPlane(t)
	f.clock.Set(trafficNow)
	return f
}

func (f *trafficFixture) persistUplink(t *testing.T, packetCnt uint32, bsEui uint64, rx time.Time, rssi float64) {
	t.Helper()
	req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, bsEui, packetCnt, []byte{byte(packetCnt)})
	withReceptionTime(req, rx.UnixNano())
	req.Message.RSSI, req.Message.BaseStations[0].Rssi = rssi, rssi
	if bsEui == uplinkStoreBsEuiTwo {
		req.Message.SNR, req.Message.BaseStations[0].Snr = trafficStationTwoSnr, trafficStationTwoSnr
	}
	f.clock.Set(rx)
	_, err := f.store.Persist(testutil.TestContext(), req)
	require.NoError(t, err)
}

// storeControlPlane stores an endpoint detach and the attach and detach
// propagations with their completions, all at station one within the hour.
func (f *trafficFixture) storeControlPlane(t *testing.T) {
	t.Helper()
	ctx := testutil.TestContext()
	station := mioty.EUI64Bytes(uplinkStoreBsEuiOne)
	require.NoError(t, f.repo.CreateDetachMessage(ctx, &mioty.DetachMessage{
		EpEui: mioty.EUI64Bytes(uplinkStoreEpEui), RxTime: f.detach.UnixNano(), PacketCnt: 5,
		SNR: trafficDetachSnr, RSSI: trafficDetachRssi, Signature: []byte{1, 2, 3, 4},
		BasestationEui: station, TenantID: uplinkStoreTenantA, ReceivedAt: f.detach,
	}, nil))
	for _, command := range []string{mioty.CmdAttachPropagate, mioty.CmdAttachPropagateComplete} {
		require.NoError(t, f.repo.CreateAttachPropagateMessage(ctx, &mioty.AttachPropagateMessage{
			CommandType: command, EpEui: uplinkStoreEpEui, NwkSnKey: make([]byte, 16),
			BasestationEui: station, TenantID: uplinkStoreTenantA, ReceivedAt: f.today,
		}))
	}
	for _, command := range []string{mioty.CmdDetachPropagate, mioty.CmdDetachPropagateComplete} {
		require.NoError(t, f.repo.CreateDetachPropagateMessage(ctx, &mioty.DetachPropagateMessage{
			CommandType: command, EpEui: uplinkStoreEpEui,
			BasestationEui: station, TenantID: uplinkStoreTenantA, ReceivedAt: f.detach,
		}))
	}
}

func assertStoredInstant(t *testing.T, want time.Time, got *time.Time, name string) {
	t.Helper()
	require.NotNil(t, got, name)
	assertInstant(t, want, *got, name)
}

func TestBaseStationMessageStats_CountsTheUplinksTheStationReceived(t *testing.T) {
	f := seedTraffic(t)
	ctx := testutil.TestContext()

	one, err := f.repo.GetBaseStationMessageStats(ctx, uplinkStoreTenantA, mioty.EUI64Bytes(uplinkStoreBsEuiOne), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), one.TotalMessages, "propagations and the detach are not uplinks")
	assert.Equal(t, int64(1), one.TotalEndpoints)
	assert.InDelta(t, trafficStationOneRssi, one.AvgRSSI, 1e-9, "propagations carry no radio values")
	assert.Equal(t, int64(1), one.MessagesToday)
	assert.Equal(t, int64(2), one.MessagesThisWeek)
	assert.Equal(t, int64(3), one.MessagesThisMonth)
	assertStoredInstant(t, f.thisMonth, one.FirstMessageAt, "first uplink")
	assertStoredInstant(t, f.today, one.LastMessageAt, "last uplink")

	two, err := f.repo.GetBaseStationMessageStats(ctx, uplinkStoreTenantA, mioty.EUI64Bytes(uplinkStoreBsEuiTwo), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), two.TotalMessages, "an uplink counts at every station that received it")
	assert.InDelta(t, trafficStationTwoRssi, two.AvgRSSI, 1e-9, "a station averages its own receptions")
	assert.InDelta(t, trafficStationTwoSnr, two.AvgSNR, 1e-9)
	assert.Equal(t, int64(1), two.MessagesThisMonth, "the earlier uplink predates this month")
	assertStoredInstant(t, f.before, two.FirstMessageAt, "first uplink")
	assertStoredInstant(t, f.today, two.LastMessageAt, "last uplink")

	since, until := trafficNow.AddDate(0, 0, -3), trafficNow
	windowed, err := f.repo.GetBaseStationMessageStats(ctx, uplinkStoreTenantA, mioty.EUI64Bytes(uplinkStoreBsEuiOne), &since, &until)
	require.NoError(t, err)
	assert.Equal(t, int64(2), windowed.TotalMessages)
	assertStoredInstant(t, f.thisWeek, windowed.FirstMessageAt, "first uplink in the window")
}

func TestBaseStationEndpointCountsAndLastSeen_CountOnlyUplinks(t *testing.T) {
	f := seedTraffic(t)
	ctx := testutil.TestContext()
	endpoint := mioty.FormatEUI64(uplinkStoreEpEui)

	counts, err := f.repo.GetBaseStationEndpointCounts(ctx, uplinkStoreTenantA, mioty.EUI64Bytes(uplinkStoreBsEuiOne), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{endpoint: 3}, counts)
	counts, err = f.repo.GetBaseStationEndpointCounts(ctx, uplinkStoreTenantA, mioty.EUI64Bytes(uplinkStoreBsEuiTwo), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{endpoint: 2}, counts)

	lastSeen, err := f.repo.GetBaseStationLastSeen(ctx, uplinkStoreTenantA, mioty.EUI64Bytes(uplinkStoreBsEuiOne))
	require.NoError(t, err)
	assertStoredInstant(t, f.today, lastSeen, "the latest uplink, not the later detach propagation")
}

func TestEndpointMessageStats_CountOnlyUplinks(t *testing.T) {
	f := seedTraffic(t)
	analytics := NewMessageAnalyticsRepository(f.db)

	stats, err := analytics.GetMessageStatsByEndpointSince(testutil.TestContext(), uplinkStoreEpEui, uplinkStoreTenantA, trafficNow.AddDate(0, 0, -5))
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalCount, "the detach and the propagations are not uplinks")
	assert.InDelta(t, trafficStationOneRssi, stats.AvgRSSI, 1e-9)
	assertStoredInstant(t, f.thisWeek, stats.FirstSeen, "first uplink")
	assertStoredInstant(t, f.today, stats.LastSeen, "last uplink")
	require.NotNil(t, stats.ActiveDays)
	assert.Equal(t, 2, *stats.ActiveDays)

	all, err := analytics.GetMessageStatsByEndpoint(testutil.TestContext(), uplinkStoreEpEui, uplinkStoreTenantA)
	require.NoError(t, err)
	assert.Equal(t, int64(4), all.TotalCount)
	assert.Equal(t, int64(2), all.UniqueEndpoints, "both stations were primary receivers")
	assertStoredInstant(t, f.before, all.FirstSeen, "first uplink")
}

func TestAnalyticsOverview_CountsOnlyFramesReceivedFromEndpoints(t *testing.T) {
	f := seedTraffic(t)
	ctx := testutil.TestContext()
	analytics := NewMessageAnalyticsRepository(f.db)
	start, end := trafficNow.Add(-24*time.Hour), trafficNow

	overview, err := analytics.GetAnalyticsOverview(ctx, uplinkStoreTenantA, start, end)
	require.NoError(t, err)
	assert.Equal(t, int64(2), overview.TotalMessages, "the uplink and the detach; propagations are sent, not received")
	require.NotNil(t, overview.AvgRSSI)
	assert.InDelta(t, (trafficStationOneRssi+trafficDetachRssi)/2, *overview.AvgRSSI, 1e-9)

	quality, err := analytics.GetSignalQualityStats(ctx, uplinkStoreTenantA, start, end)
	require.NoError(t, err)
	assert.Equal(t, int64(2), quality.TotalMessages)
	assert.InDelta(t, trafficDetachRssi, quality.MinRSSI, 1e-9)
	assert.InDelta(t, trafficStationOneRssi, quality.MaxRSSI, 1e-9, "no propagation's zero rssi")

	counts, err := analytics.GetMessageCountsByEndpoint(ctx, uplinkStoreTenantA, start, end)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{mioty.FormatEUI64(uplinkStoreEpEui): 2}, counts)
}
