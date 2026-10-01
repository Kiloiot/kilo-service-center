package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	analyticsTestTenant  int64 = 7
	analyticsTestStation       = "70-B3-D5-9C-D0-00-09-E6"
)

// Figures the fake message store reports.
const (
	firstDayMessages  = 12
	secondDayMessages = 30
	lowestSNR         = 1
	highestSNR        = 20
	stationMessages   = 7
)

type fakeMessageStore struct {
	overview *Stats
	days     []DailyActivity
	signal   *SignalStats
	stations []BaseStationSignalStats
	err      error
	windows  [][2]time.Time
	tenants  []int64
}

func (f *fakeMessageStore) record(tenantID int64, start, end time.Time) {
	f.tenants = append(f.tenants, tenantID)
	f.windows = append(f.windows, [2]time.Time{start, end})
}

func (f *fakeMessageStore) GetAnalyticsOverview(_ context.Context, tenantID int64, start, end time.Time) (*Stats, error) {
	f.record(tenantID, start, end)
	return f.overview, f.err
}

func (f *fakeMessageStore) GetDailyActivity(_ context.Context, tenantID int64, start, end time.Time) ([]DailyActivity, error) {
	f.record(tenantID, start, end)
	return f.days, f.err
}

func (f *fakeMessageStore) GetSignalQualityStats(_ context.Context, tenantID int64, start, end time.Time) (*SignalStats, error) {
	f.record(tenantID, start, end)
	return f.signal, f.err
}

func (f *fakeMessageStore) GetSignalQualityByBaseStation(_ context.Context, tenantID int64, start, end time.Time) ([]BaseStationSignalStats, error) {
	f.record(tenantID, start, end)
	return f.stations, f.err
}

// analyticsNow is the injected clock's time.
var analyticsNow = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

type fixedClock time.Time

func (c fixedClock) Now() time.Time { return time.Time(c) }

func newTestService(store *fakeMessageStore) *Service {
	return New(store, logger.NewNop(), fixedClock(analyticsNow))
}

func TestGetActivity_TotalsComeFromTheWholeWindowAndSlotsKeepTheirOrder(t *testing.T) {
	day1 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)
	store := &fakeMessageStore{
		overview: &Stats{TotalMessages: 42, ActiveEndpoints: 5, ActiveBaseStations: 2},
		days: []DailyActivity{
			{Day: day1, MessageCount: firstDayMessages, UniqueEndpoints: 3},
			{Day: day2, MessageCount: secondDayMessages, UniqueEndpoints: 4},
		},
	}
	start, end := day1, day2.AddDate(0, 0, 1)

	activity, err := newTestService(store).GetActivity(testutil.TestContext(), analyticsTestTenant, &start, &end, GranularityDay)

	require.NoError(t, err)
	assert.Equal(t, int64(42), activity.TotalMessages)
	assert.Equal(t, int64(5), activity.UniqueEndpoints)
	assert.Equal(t, int64(2), activity.UniqueBaseStations)
	require.Len(t, activity.Slots, 2)
	assert.Equal(t, day1, activity.Slots[0].Slot)
	assert.Equal(t, int64(4), activity.Slots[1].EndpointCount)
	assert.Equal(t, start, activity.StartTime)
	assert.Equal(t, end, activity.EndTime)
	assert.Equal(t, []int64{analyticsTestTenant, analyticsTestTenant}, store.tenants)
}

func TestGetActivity_OpenWindowEndsNowAndLooksBackAWeek(t *testing.T) {
	store := &fakeMessageStore{overview: &Stats{}}

	activity, err := newTestService(store).GetActivity(testutil.TestContext(), analyticsTestTenant, nil, nil, "")

	require.NoError(t, err)
	assert.Equal(t, defaultActivityWindow, activity.EndTime.Sub(activity.StartTime))
	assert.Equal(t, analyticsNow, activity.EndTime, "an open window ends at the injected clock's now")
}

func TestGetActivity_RejectsGranularityTheStoreCannotAggregate(t *testing.T) {
	store := &fakeMessageStore{overview: &Stats{}}

	_, err := newTestService(store).GetActivity(testutil.TestContext(), analyticsTestTenant, nil, nil, "hour")

	require.ErrorIs(t, err, ErrUnsupportedGranularity)
	assert.Empty(t, store.tenants)
}

func TestGetActivity_StoreFailureIsWrapped(t *testing.T) {
	store := &fakeMessageStore{err: errors.New("down")}

	_, err := newTestService(store).GetActivity(testutil.TestContext(), analyticsTestTenant, nil, nil, "")

	require.ErrorIs(t, err, errActivityTotals)
}

func TestGetSignalQuality_ReportsMediansAndPerStationQuality(t *testing.T) {
	store := &fakeMessageStore{
		signal: &SignalStats{AvgRSSI: -80, MinRSSI: -110, MaxRSSI: -50, MedianRSSI: -82, AvgSNR: 9, MinSNR: lowestSNR, MaxSNR: highestSNR, MedianSNR: 8},
		stations: []BaseStationSignalStats{
			{EUI: analyticsTestStation, AvgRSSI: -79, AvgSNR: 10, MessageCount: stationMessages},
		},
	}

	quality, err := newTestService(store).GetSignalQuality(testutil.TestContext(), analyticsTestTenant, nil, nil)

	require.NoError(t, err)
	assert.InDelta(t, -82, quality.MedianRSSI, 0.0001)
	assert.InDelta(t, 8, quality.MedianSNR, 0.0001)
	assert.Equal(t, [2]float64{-110, -50}, quality.RSSIRange)
	require.Len(t, quality.ByBaseStation, 1)
	assert.Equal(t, analyticsTestStation, quality.ByBaseStation[0].EUI)
	assert.Equal(t, int64(stationMessages), quality.ByBaseStation[0].MessageCount)
	assert.Equal(t, defaultAnalyticsWindow, quality.EndTime.Sub(quality.StartTime))
}

func TestGetOverview_ReportsTheActiveBaseStations(t *testing.T) {
	store := &fakeMessageStore{overview: &Stats{TotalMessages: 3, ActiveEndpoints: 2, ActiveBaseStations: 1}}

	overview, err := newTestService(store).GetOverview(testutil.TestContext(), analyticsTestTenant, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(1), overview.ActiveBaseStations)
	assert.Equal(t, int64(2), overview.ActiveEndpoints)
}
