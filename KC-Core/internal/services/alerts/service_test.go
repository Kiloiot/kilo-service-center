package alerts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const alertServiceTenant int64 = 9

const alertServiceErrors = 4

type fakeAlertStore struct {
	filters []*AlertFilter
}

func (f *fakeAlertStore) List(_ context.Context, _ int64, filter *AlertFilter, _, _ int) ([]*Alert, int64, error) {
	f.filters = append(f.filters, filter)
	return []*Alert{{ID: "event-1", Status: models.EventStatusResolved}}, 1, nil
}

func (f *fakeAlertStore) GetSummary(context.Context, int64) (*AlertSummary, error) {
	return &AlertSummary{Error: alertServiceErrors}, nil
}

func TestList_PassesKnownStatusesToTheStore(t *testing.T) {
	store := &fakeAlertStore{}

	alerts, _, err := New(store, logger.NewNop()).List(testutil.TestContext(), alertServiceTenant,
		&grpcservices.AlertFilters{Status: []string{models.EventStatusResolved}}, 10, 0)

	require.NoError(t, err)
	require.Len(t, store.filters, 1)
	assert.Equal(t, []string{models.EventStatusResolved}, store.filters[0].Status)
	assert.Equal(t, "event-1", alerts[0].ID)
}

func TestList_RejectsAnUnknownStatusBeforeTheStore(t *testing.T) {
	store := &fakeAlertStore{}

	_, _, err := New(store, logger.NewNop()).List(testutil.TestContext(), alertServiceTenant,
		&grpcservices.AlertFilters{Status: []string{"active"}}, 10, 0)

	require.ErrorIs(t, err, ErrInvalidAlertStatus)
	assert.Empty(t, store.filters)
}

func TestList_RejectsASeverityOutsideTheAlertSeveritiesBeforeTheStore(t *testing.T) {
	for _, severity := range []string{models.EventSeverityInfo, "fatal"} {
		store := &fakeAlertStore{}

		_, _, err := New(store, logger.NewNop()).List(testutil.TestContext(), alertServiceTenant,
			&grpcservices.AlertFilters{Severity: []string{severity}}, 10, 0)

		require.ErrorIs(t, err, ErrInvalidAlertSeverity, severity)
		assert.Empty(t, store.filters, severity)
	}
}

func TestList_PassesEachAlertSeverityToTheStore(t *testing.T) {
	for _, severity := range AlertSeverities {
		store := &fakeAlertStore{}

		_, _, err := New(store, logger.NewNop()).List(testutil.TestContext(), alertServiceTenant,
			&grpcservices.AlertFilters{Severity: []string{severity}}, 10, 0)

		require.NoError(t, err, severity)
		assert.Equal(t, []string{severity}, store.filters[0].Severity)
	}
}

func TestGetSummary_CarriesTheErrorCount(t *testing.T) {
	summary, err := New(&fakeAlertStore{}, logger.NewNop()).GetSummary(testutil.TestContext(), alertServiceTenant)

	require.NoError(t, err)
	assert.Equal(t, int32(alertServiceErrors), summary.Error)
}
