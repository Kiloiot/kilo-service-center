package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	sessionEventTenant        int64 = 720
	sessionEventForeignTenant int64 = 721
	sessionEventSessionID     int64 = 42
)

const (
	sessionEventAcEui    = "70-B3-D5-00-00-00-00-01"
	sessionEventRemote   = "192.0.2.10:40000"
	sessionEventPageSize = 10
	// sessionEventErrorToken stands in for a SCACI catalog token, which KC-DB never interprets.
	sessionEventErrorToken = "refusal-token"
)

func newSessionEventStore(t *testing.T) (*SCACIEventStore, context.Context) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSystemEventsTestDB(t)
	createSystemEventsTestTenant(t, db, sessionEventTenant, "Session Event Tenant")
	createSystemEventsTestTenant(t, db, sessionEventForeignTenant, "Session Event Foreign Tenant")
	ctx, cancel := context.WithTimeout(testutil.TestContext(), testContextTimeout)
	t.Cleanup(cancel)
	events := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get())
	return NewSCACIEventStore(db.DB, events, clock.SystemClock{}, logger.Get()), ctx
}

func TestRecordSessionEvent_FilesTheLifecycleUnderTheSessionTenant(t *testing.T) {
	store, ctx := newSessionEventStore(t)

	cases := []struct {
		eventType string
		severity  string
		title     string
	}{
		{models.EventTypeSCACISessionOpened, models.EventSeverityInfo, "Application center 70-B3-D5-00-00-00-00-01 connected"},
		{models.EventTypeSCACISessionResumed, models.EventSeverityInfo, "Application center 70-B3-D5-00-00-00-00-01 resumed its session"},
		{models.EventTypeSCACISessionClosed, models.EventSeverityWarning, "Application center 70-B3-D5-00-00-00-00-01 disconnected"},
	}
	for _, tc := range cases {
		require.NoError(t, store.RecordSessionEvent(ctx, &models.SCACISessionEvent{
			TenantID: sessionEventTenant, Category: models.EventCategorySCACI, EventType: tc.eventType, SessionID: sessionEventSessionID,
			AcEui: sessionEventAcEui, RemoteAddr: sessionEventRemote,
		}))
	}

	sessionID := sessionEventSessionID
	events, err := store.ListSCACIEvents(ctx, sessionEventTenant, &sessionID, "", sessionEventPageSize, 0)
	require.NoError(t, err)
	require.Len(t, events, len(cases))
	byType := map[string]*models.SystemEvent{}
	for _, e := range events {
		byType[e.EventType] = e
	}
	for _, tc := range cases {
		e := byType[tc.eventType]
		require.NotNil(t, e, tc.eventType)
		assert.Equal(t, models.EventCategorySCACI, e.Category)
		assert.Equal(t, tc.severity, e.Severity)
		assert.Equal(t, tc.title, e.Title)
		assert.Equal(t, models.SourceTypeApplicationCenter, e.SourceType)
		assert.Equal(t, sessionEventAcEui, e.SourceName)
		var details map[string]interface{}
		require.NoError(t, json.Unmarshal(e.Details, &details))
		assert.Equal(t, sessionEventRemote, details[models.EventDetailKeyRemoteAddr])
	}

	foreign, err := store.ListSCACIEvents(ctx, sessionEventForeignTenant, nil, "", sessionEventPageSize, 0)
	require.NoError(t, err)
	assert.Empty(t, foreign)
}

func TestRecordSessionEvent_RefusedConnectCarriesTheCatalogError(t *testing.T) {
	store, ctx := newSessionEventStore(t)

	require.NoError(t, store.RecordSessionEvent(ctx, &models.SCACISessionEvent{
		TenantID: sessionEventTenant, Category: models.EventCategorySCACI, EventType: models.EventTypeSCACIConnectRefused,
		ErrorToken: sessionEventErrorToken, ErrorMessage: "major version unsupported",
	}))

	events, err := store.ListSCACIEvents(ctx, sessionEventTenant, nil, models.EventTypeSCACIConnectRefused, sessionEventPageSize, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, models.EventSeverityError, e.Severity)
	assert.Equal(t, "Application center unknown connect refused", e.Title)
	assert.Equal(t, "Connect of application center unknown refused: major version unsupported", e.Description)
	var details map[string]interface{}
	require.NoError(t, json.Unmarshal(e.Details, &details))
	assert.Equal(t, sessionEventErrorToken, details[models.EventDetailKeyErrorToken])
	assert.NotContains(t, details, models.EventDetailKeySessionID, "no session existed")
}

func TestRecordSessionEvent_ServerLevelRefusalIsASecurityEvent(t *testing.T) {
	store, ctx := newSessionEventStore(t)

	require.NoError(t, store.RecordSessionEvent(ctx, &models.SCACISessionEvent{
		TenantID: sessionEventTenant, Category: models.EventCategorySecurity, EventType: models.EventTypeSCACIConnectRefused,
		ErrorToken: sessionEventErrorToken,
	}))

	scaci, err := store.ListSCACIEvents(ctx, sessionEventTenant, nil, "", sessionEventPageSize, 0)
	require.NoError(t, err)
	assert.Empty(t, scaci, "a server-level refusal is not among the tenant's SCACI events")
	events, err := store.events.GetEvents(ctx, models.SystemEventFilter{
		TenantID: strconv.FormatInt(sessionEventTenant, 10), Categories: []string{models.EventCategorySecurity},
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, models.EventTypeSCACIConnectRefused, events[0].EventType)
}

const (
	refusalStorm        = 12
	refusalOtherToken   = "other-refusal-token"
	refusalOtherAcEui   = "70-B3-D5-00-00-00-00-02"
	refusalCountedTwice = 2
	refusalMovedRemote  = "192.0.2.20:40001"
)

func (s *SCACIEventStore) refusalRecords(ctx context.Context, t *testing.T) []*models.SystemEvent {
	t.Helper()
	events, err := s.ListSCACIEvents(ctx, sessionEventTenant, nil, models.EventTypeSCACIConnectRefused, refusalStorm+1, 0)
	require.NoError(t, err)
	return events
}

func refusalOf(acEui, token string) *models.SCACISessionEvent {
	return &models.SCACISessionEvent{
		TenantID: sessionEventTenant, Category: models.EventCategorySCACI, EventType: models.EventTypeSCACIConnectRefused,
		AcEui: acEui, RemoteAddr: sessionEventRemote, ErrorToken: token, ErrorMessage: token,
	}
}

func detailsOf(t *testing.T, e *models.SystemEvent) map[string]interface{} {
	t.Helper()
	var details map[string]interface{}
	require.NoError(t, json.Unmarshal(e.Details, &details))
	return details
}

// An application center retrying in a loop leaves one record per cause,
// counting the refusals from the first to the last, however they interleave.
func TestRecordSessionEvent_ARefusalStormIsOneRecordPerCause(t *testing.T) {
	store, ctx := newSessionEventStore(t)

	var wg sync.WaitGroup
	errs := make(chan error, refusalStorm)
	for range refusalStorm {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.RecordSessionEvent(ctx, refusalOf(sessionEventAcEui, sessionEventErrorToken))
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.NoError(t, store.RecordSessionEvent(ctx, refusalOf(sessionEventAcEui, refusalOtherToken)))
	require.NoError(t, store.RecordSessionEvent(ctx, refusalOf(refusalOtherAcEui, sessionEventErrorToken)))

	records := store.refusalRecords(ctx, t)
	require.Len(t, records, 3, "one record per application center and error token")
	counts := map[string]float64{}
	for _, e := range records {
		details := detailsOf(t, e)
		counts[e.SourceName+"/"+details[models.EventDetailKeyErrorToken].(string)] = details[models.EventDetailKeyCount].(float64)
		assert.NotEmpty(t, details[models.EventDetailKeyFirstSeen])
		assert.NotEmpty(t, details[models.EventDetailKeyLastSeen])
	}
	assert.Equal(t, float64(refusalStorm), counts[sessionEventAcEui+"/"+sessionEventErrorToken])
	assert.Equal(t, float64(1), counts[sessionEventAcEui+"/"+refusalOtherToken])
	assert.Equal(t, float64(1), counts[refusalOtherAcEui+"/"+sessionEventErrorToken])
}

// A recurrence reported without a remote address keeps the recorded one; a
// new address replaces it.
func TestRecordSessionEvent_ARecurrenceWithoutAnAddressKeepsTheRecordedOne(t *testing.T) {
	store, ctx := newSessionEventStore(t)
	withAddress := refusalOf(sessionEventAcEui, sessionEventErrorToken)
	withoutAddress := refusalOf(sessionEventAcEui, sessionEventErrorToken)
	withoutAddress.RemoteAddr = ""
	movedAddress := refusalOf(sessionEventAcEui, sessionEventErrorToken)
	movedAddress.RemoteAddr = refusalMovedRemote

	require.NoError(t, store.RecordSessionEvent(ctx, withAddress))
	require.NoError(t, store.RecordSessionEvent(ctx, withoutAddress))
	records := store.refusalRecords(ctx, t)
	require.Len(t, records, 1)
	assert.Equal(t, sessionEventRemote, detailsOf(t, records[0])[models.EventDetailKeyRemoteAddr])

	require.NoError(t, store.RecordSessionEvent(ctx, movedAddress))
	assert.Equal(t, refusalMovedRemote, detailsOf(t, store.refusalRecords(ctx, t)[0])[models.EventDetailKeyRemoteAddr])
}

// A cause first recorded without an address gains none from a recurrence
// without one: the key stays absent, never null.
func TestRecordSessionEvent_NoAddressStaysAbsent(t *testing.T) {
	store, ctx := newSessionEventStore(t)
	anonymous := refusalOf(sessionEventAcEui, sessionEventErrorToken)
	anonymous.RemoteAddr = ""

	require.NoError(t, store.RecordSessionEvent(ctx, anonymous))
	require.NoError(t, store.RecordSessionEvent(ctx, anonymous))

	records := store.refusalRecords(ctx, t)
	require.Len(t, records, 1)
	assert.NotContains(t, detailsOf(t, records[0]), models.EventDetailKeyRemoteAddr)
}

// A resolved cause stays closed: a new refusal opens a new record for it.
func TestRecordSessionEvent_ARefusalAfterResolutionStartsANewRecord(t *testing.T) {
	store, ctx := newSessionEventStore(t)
	require.NoError(t, store.RecordSessionEvent(ctx, refusalOf(sessionEventAcEui, sessionEventErrorToken)))
	require.NoError(t, store.RecordSessionEvent(ctx, refusalOf(sessionEventAcEui, sessionEventErrorToken)))
	first := store.refusalRecords(ctx, t)
	require.Len(t, first, 1)
	assert.Equal(t, float64(refusalCountedTwice), detailsOf(t, first[0])[models.EventDetailKeyCount])
	firstSeen := detailsOf(t, first[0])[models.EventDetailKeyFirstSeen]
	assert.NotEqual(t, firstSeen, detailsOf(t, first[0])[models.EventDetailKeyLastSeen], "the recurrence moves lastSeen only")
	_, err := store.db.ExecContext(ctx, `UPDATE system_events SET status = $1 WHERE id = $2::uuid`, models.EventStatusResolved, first[0].ID)
	require.NoError(t, err)

	require.NoError(t, store.RecordSessionEvent(ctx, refusalOf(sessionEventAcEui, sessionEventErrorToken)))

	records := store.refusalRecords(ctx, t)
	require.Len(t, records, 2)
	assert.Equal(t, float64(1), detailsOf(t, records[0])[models.EventDetailKeyCount], "the newest record starts counting again")
}

func TestRecordSessionEvent_ASupersededSessionSaysSo(t *testing.T) {
	store, ctx := newSessionEventStore(t)

	require.NoError(t, store.RecordSessionEvent(ctx, &models.SCACISessionEvent{
		TenantID: sessionEventTenant, Category: models.EventCategorySCACI, EventType: models.EventTypeSCACISessionClosed,
		SessionID: sessionEventSessionID, AcEui: sessionEventAcEui, Reason: models.SCACISessionClosedSuperseded,
	}))

	events, err := store.ListSCACIEvents(ctx, sessionEventTenant, nil, models.EventTypeSCACISessionClosed, sessionEventPageSize, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Session 42 of application center 70-B3-D5-00-00-00-00-01 was replaced by a new session of the application center", events[0].Description)
	assert.Equal(t, models.SCACISessionClosedSuperseded, detailsOf(t, events[0])[models.EventDetailKeyReason])
}

func TestRecordSessionEvent_RejectsAnotherCategory(t *testing.T) {
	store := &SCACIEventStore{}

	for _, category := range []string{"", models.EventCategoryProtocol} {
		err := store.RecordSessionEvent(testutil.TestContext(), &models.SCACISessionEvent{
			TenantID: sessionEventTenant, Category: category, EventType: models.EventTypeSCACISessionOpened,
		})
		require.ErrorIs(t, err, errTextSCACISessionEventCategory, category)
	}
}

func TestRecordSessionEvent_RejectsAnUnknownType(t *testing.T) {
	store := &SCACIEventStore{}

	err := store.RecordSessionEvent(testutil.TestContext(), &models.SCACISessionEvent{
		TenantID: sessionEventTenant, EventType: models.EventTypeSCACIError,
	})

	require.ErrorIs(t, err, errTextUnknownSCACISessionEvent)
}
