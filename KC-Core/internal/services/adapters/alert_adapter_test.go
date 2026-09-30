package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	alertsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/alerts"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	alertAdapterTenant   int64 = 42
	alertAdapterEventID        = "5f0c2a1e-9c6b-4b7e-8d0a-1f2e3d4c5b6a"
	alertAdapterRecent         = 5
	alertAdapterCritical       = 2
	alertAdapterErrors         = 3
	alertAdapterWarnings       = 4
	alertAdapterTotal          = 9
)

var errAlertStoreDown = errors.New("store down")

type fakeAlertEventStore struct {
	listed     []models.AlertFilter
	counted    []models.AlertFilter
	summarized []models.AlertFilter
	events     []*models.SystemEvent
	counts     map[string]int64
	listErr    error
	countErr   error
	summaryErr error
}

func (f *fakeAlertEventStore) GetActiveAlerts(_ context.Context, filter models.AlertFilter) ([]*models.SystemEvent, error) {
	f.listed = append(f.listed, filter)
	return f.events, f.listErr
}

func (f *fakeAlertEventStore) CountActiveAlerts(_ context.Context, filter models.AlertFilter) (int64, error) {
	f.counted = append(f.counted, filter)
	return alertAdapterTotal, f.countErr
}

func (f *fakeAlertEventStore) CountAlertsBySeverity(_ context.Context, filter models.AlertFilter) (map[string]int64, error) {
	f.summarized = append(f.summarized, filter)
	return f.counts, f.summaryErr
}

func newAlertStore() *fakeAlertEventStore {
	return &fakeAlertEventStore{
		events: []*models.SystemEvent{{
			ID: alertAdapterEventID, Severity: models.EventSeverityError, Status: models.EventStatusAcknowledged,
			Title: "offline", CreatedAt: time.Unix(0, 0),
		}},
		counts: map[string]int64{
			models.EventSeverityCritical: alertAdapterCritical,
			models.EventSeverityError:    alertAdapterErrors,
			models.EventSeverityWarning:  alertAdapterWarnings,
		},
	}
}

func TestAlertStoreAdapter_ListPassesTheStatusFilterToListingAndCount(t *testing.T) {
	store := newAlertStore()
	adapter := NewAlertStoreAdapter(store, alertAdapterRecent)
	statuses := []string{models.EventStatusAcknowledged}

	alerts, total, err := adapter.List(testutil.TestContext(), alertAdapterTenant, &alertsservice.AlertFilter{Status: statuses}, 10, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(alertAdapterTotal), total)
	require.Len(t, store.listed, 1)
	assert.Equal(t, statuses, store.listed[0].Statuses)
	assert.Equal(t, "42", store.listed[0].TenantID)
	require.Len(t, store.counted, 1)
	assert.Equal(t, statuses, store.counted[0].Statuses, "the total counts the same statuses")
	assert.Zero(t, store.counted[0].Limit)
	require.Len(t, alerts, 1)
	assert.Equal(t, alertAdapterEventID, alerts[0].ID, "the event id is the alert id")
}

func TestAlertStoreAdapter_ListWithoutStatusesLeavesTheUnresolvedDefault(t *testing.T) {
	store := newAlertStore()

	_, _, err := NewAlertStoreAdapter(store, alertAdapterRecent).List(testutil.TestContext(), alertAdapterTenant, nil, 10, 0)

	require.NoError(t, err)
	assert.Empty(t, store.listed[0].Statuses)
	assert.Equal(t, alertsservice.AlertSeverities, store.listed[0].Severities)
}

func TestAlertStoreAdapter_ListReportsACountFailure(t *testing.T) {
	store := newAlertStore()
	store.countErr = errAlertStoreDown

	_, _, err := NewAlertStoreAdapter(store, alertAdapterRecent).List(testutil.TestContext(), alertAdapterTenant, nil, 10, 0)

	require.ErrorIs(t, err, errAlertStoreDown)
}

func TestAlertStoreAdapter_SummaryCountsEveryAlertSeverity(t *testing.T) {
	store := newAlertStore()

	summary, err := NewAlertStoreAdapter(store, alertAdapterRecent).GetSummary(testutil.TestContext(), alertAdapterTenant)

	require.NoError(t, err)
	assert.Equal(t, int32(alertAdapterCritical), summary.Critical)
	assert.Equal(t, int32(alertAdapterErrors), summary.Error)
	assert.Equal(t, int32(alertAdapterWarnings), summary.Warning)
	require.Len(t, summary.Recent, 1)
	assert.Equal(t, alertAdapterRecent, store.listed[0].Limit)
}

func TestAlertStoreAdapter_SummaryReadsTheListRule(t *testing.T) {
	store := newAlertStore()
	adapter := NewAlertStoreAdapter(store, alertAdapterRecent)

	_, _, err := adapter.List(testutil.TestContext(), alertAdapterTenant, nil, alertAdapterRecent, 0)
	require.NoError(t, err)
	_, err = adapter.GetSummary(testutil.TestContext(), alertAdapterTenant)
	require.NoError(t, err)

	require.Len(t, store.listed, 2)
	assert.Equal(t, store.listed[0], store.listed[1], "the recent alerts are the list's first page")
	assert.Nil(t, store.listed[1].Since, "an open alert counts however old it is")
	require.Len(t, store.summarized, 1)
	assert.Equal(t, store.counted[0], store.summarized[0], "each count is the list total of its severity")
}

func TestAlertStoreAdapter_SummaryReportsStoreFailures(t *testing.T) {
	counts := newAlertStore()
	counts.summaryErr = errAlertStoreDown
	_, err := NewAlertStoreAdapter(counts, alertAdapterRecent).GetSummary(testutil.TestContext(), alertAdapterTenant)
	require.ErrorIs(t, err, errAlertStoreDown)

	recent := newAlertStore()
	recent.listErr = errAlertStoreDown
	_, err = NewAlertStoreAdapter(recent, alertAdapterRecent).GetSummary(testutil.TestContext(), alertAdapterTenant)
	require.ErrorIs(t, err, errAlertStoreDown)
}
