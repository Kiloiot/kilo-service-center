package postgres

import (
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

func TestEventOrdering_OnlyAllowListedColumnsAndDirections(t *testing.T) {
	column, direction, err := eventOrdering(models.SystemEventFilter{})
	require.NoError(t, err)
	assert.Equal(t, "occurred_at DESC", column+" "+direction, "newest first by default")

	column, direction, err = eventOrdering(models.SystemEventFilter{OrderBy: "Severity", OrderDirection: "ASC"})
	require.NoError(t, err)
	assert.Equal(t, "severity ASC", column+" "+direction)

	for _, filter := range []models.SystemEventFilter{
		{OrderBy: "occurred_at; DROP TABLE system_events"},
		{OrderBy: "data->>'secret'"},
		{OrderDirection: "desc, (SELECT 1)"},
	} {
		_, _, err := eventOrdering(filter)
		assert.ErrorIs(t, err, errTextUnsortableEvents, filter)
	}
}

func TestEventOrderClause_BreaksTiesInTheSortDirection(t *testing.T) {
	assert.Equal(t, "occurred_at DESC, recorded_at DESC, id DESC", eventOrderClause("occurred_at", "DESC"))
	assert.Equal(t, "recorded_at ASC, id ASC", eventOrderClause("recorded_at", "ASC"))
}

// tiedEvent is an event that occurred at the shared moment and was recorded
// recordedStep milliseconds after it.
type tiedEvent struct {
	id           string
	recordedStep int
}

func insertTiedEvent(t *testing.T, db *sqlx.DB, tenantID int64, occurredAt time.Time, event tiedEvent) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO system_events (
			id, tenant_id, event_type, event_category, severity,
			source_type, source_name, title, description,
			data, status, occurred_at, recorded_at
		) VALUES ($1::uuid, $2, 'basestation_ping_sent', 'basestation', 'info', 'basestation', 'test', $1, 'Tied event', '{}', 'new', $3, $4)
	`, event.id, tenantID, occurredAt, occurredAt.Add(time.Duration(event.recordedStep)*time.Millisecond))
	require.NoError(t, err)
}

// Events that occurred at the same moment list in the order they were
// recorded, then by id, in the direction the listing sorts.
func TestGetEvents_EqualOccurrenceOrdersByRecordedThenID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSystemEventsTestDB(t)
	const tenantID = int64(810)
	createSystemEventsTestTenant(t, db, tenantID, "TieOrderTest")
	occurredAt := time.Date(2026, time.September, 29, 12, 40, 35, 321186000, time.UTC)
	first := tiedEvent{"00000000-0000-0000-0000-000000000001", 0}
	second := tiedEvent{"00000000-0000-0000-0000-000000000002", 1}
	thirdLowID := tiedEvent{"00000000-0000-0000-0000-000000000003", 2}
	thirdHighID := tiedEvent{"00000000-0000-0000-0000-000000000004", 2}
	last := tiedEvent{"00000000-0000-0000-0000-000000000005", 3}
	for _, event := range []tiedEvent{thirdLowID, first, last, second, thirdHighID} {
		insertTiedEvent(t, db, tenantID, occurredAt, event)
	}
	store := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get())

	for direction, want := range map[string][]tiedEvent{
		"desc": {last, thirdHighID, thirdLowID, second, first},
		"asc":  {first, second, thirdLowID, thirdHighID, last},
	} {
		events, err := store.GetEvents(testutil.TestContext(), models.SystemEventFilter{
			TenantID: strconv.FormatInt(tenantID, 10), OrderDirection: direction, Limit: testEventQueryLimit,
		})
		require.NoError(t, err)
		got := make([]string, 0, len(events))
		for _, event := range events {
			got = append(got, event.ID)
		}
		wantIDs := make([]string, 0, len(want))
		for _, event := range want {
			wantIDs = append(wantIDs, event.id)
		}
		assert.Equal(t, wantIDs, got, direction)
	}
}

func TestGetEvents_RefusesAnOrderingOutsideTheAllowList(t *testing.T) {
	store := &SystemEventStore{}

	_, err := store.GetEvents(testutil.TestContext(), models.SystemEventFilter{TenantID: "1", OrderBy: "tenant_id, title"})

	require.ErrorIs(t, err, errTextUnsortableEvents)
}
