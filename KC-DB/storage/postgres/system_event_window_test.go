package postgres

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// Both bounds of an event window are inclusive: an event that occurred at a
// bound is listed and counted, whether the bound falls on a whole second or
// between two.
func TestEventWindow_IncludesTheEventsAtItsBounds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSystemEventsTestDB(t)
	const tenantID = int64(820)
	createSystemEventsTestTenant(t, db, tenantID, "InclusiveWindowTest")
	store := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	for _, bound := range []struct {
		name string
		id   string
		at   time.Time
	}{
		{"whole second", "00000000-0000-0000-0000-000000000821", time.Date(2026, time.September, 29, 12, 40, 35, 0, time.UTC)},
		{"between seconds", "00000000-0000-0000-0000-000000000822", time.Date(2026, time.September, 29, 12, 41, 35, 321186000, time.UTC)},
		{"a microsecond past a second", "00000000-0000-0000-0000-000000000823", time.Date(2026, time.September, 29, 12, 42, 35, 1000, time.UTC)},
	} {
		name, at := bound.name, bound.at
		insertTiedEvent(t, db, tenantID, at, tiedEvent{id: bound.id})
		filter := models.SystemEventFilter{TenantID: strconv.FormatInt(tenantID, 10), Since: &at, Until: &at, Limit: testEventQueryLimit}

		events, err := store.GetEvents(ctx, filter)
		require.NoError(t, err, name)
		require.Len(t, events, 1, name)
		assert.Equal(t, bound.id, events[0].ID, name)
		count, err := store.CountEvents(ctx, filter)
		require.NoError(t, err, name)
		assert.Equal(t, int64(1), count, name)
	}
}
