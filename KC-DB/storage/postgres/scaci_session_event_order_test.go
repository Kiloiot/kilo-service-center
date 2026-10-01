package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const sessionEventStepApart = time.Second

// A session's steps are listed in the order they happened, whatever order
// their writes reached the store in, and carry the moments they happened.
func TestRecordSessionEvent_ListsTheLifecycleInTheOrderItHappened(t *testing.T) {
	store, ctx := newSessionEventStore(t)
	openedAt := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	closedAt := openedAt.Add(sessionEventStepApart)
	step := func(eventType string, at time.Time) *models.SCACISessionEvent {
		return &models.SCACISessionEvent{
			TenantID: sessionEventTenant, Category: models.EventCategorySCACI, EventType: eventType,
			SessionID: sessionEventSessionID, AcEui: sessionEventAcEui, OccurredAt: at,
		}
	}

	require.NoError(t, store.RecordSessionEvent(ctx, step(models.EventTypeSCACISessionClosed, closedAt)))
	require.NoError(t, store.RecordSessionEvent(ctx, step(models.EventTypeSCACISessionOpened, openedAt)))

	sessionID := sessionEventSessionID
	events, err := store.ListSCACIEvents(ctx, sessionEventTenant, &sessionID, "", sessionEventPageSize, 0)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, models.EventTypeSCACISessionClosed, events[0].EventType, "newest first")
	assert.True(t, events[0].CreatedAt.Equal(closedAt), "closed at %s", events[0].CreatedAt)
	assert.Equal(t, models.EventTypeSCACISessionOpened, events[1].EventType)
	assert.True(t, events[1].CreatedAt.Equal(openedAt), "opened at %s", events[1].CreatedAt)
	assert.True(t, events[1].UpdatedAt.After(closedAt), "the store stamps when it recorded the step")
}
