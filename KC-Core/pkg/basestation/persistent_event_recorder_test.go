package basestation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	recorderDefaultTenant = "1"
	recorderStationTenant = int64(4)
	recorderOperator      = "be8a02c9-1470-4314-8a86-48712761aca9"
	recorderStationEUI    = "70B3D59CD00009E6"
	recorderPingOpID      = int64(-41)
)

var errEventStoreDown = errors.New("event store unavailable")

var recorderOccurredAt = time.Date(2026, time.September, 29, 12, 40, 35, 321186000, time.UTC)

type capturedEvents struct {
	events []*models.SystemEvent
	err    error
}

func (c *capturedEvents) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	c.events = append(c.events, event)
	return c.err
}

func recordedDetails(t *testing.T, event *models.SystemEvent) map[string]interface{} {
	t.Helper()
	var details map[string]interface{}
	require.NoError(t, json.Unmarshal(event.Details, &details))
	return details
}

// A ping the operator sent is stored for the station's tenant, names the
// operator and the station, and carries the ping's opId.
func TestRecordEvent_PingSentNamesStationOperatorAndOpID(t *testing.T) {
	store := &capturedEvents{}
	recorder := NewPersistentEventRecorder(logger.NewNop(), store, recorderDefaultTenant)
	ctx := pkgcontext.WithUserID(pkgcontext.WithTenantID(testutil.TestContext(), recorderStationTenant), recorderOperator)

	require.NoError(t, recorder.RecordEvent(ctx, testStationEUI, models.EventTypeBaseStationPingSent, recorderOccurredAt,
		map[string]interface{}{models.EventDetailKeyOpID: recorderPingOpID}))

	require.Len(t, store.events, 1)
	event := store.events[0]
	assert.Equal(t, "4", event.TenantID)
	assert.Equal(t, recorderOperator, event.UserID)
	assert.Equal(t, models.EventCategoryBaseStation, event.Category)
	assert.Equal(t, recorderStationEUI, event.SourceName)
	assert.Equal(t, titleBSPingSent, event.Title)
	assert.Contains(t, event.Description, "-41")
	details := recordedDetails(t, event)
	assert.Equal(t, recorderStationEUI, details[models.EventDetailKeyBsEui])
	assert.EqualValues(t, recorderPingOpID, details[models.EventDetailKeyOpID])
}

// A station's answer carries no operator and reads with its result.
func TestRecordEvent_PingAnsweredCarriesResult(t *testing.T) {
	store := &capturedEvents{}
	recorder := NewPersistentEventRecorder(logger.NewNop(), store, recorderDefaultTenant)
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), recorderStationTenant)

	require.NoError(t, recorder.RecordEvent(ctx, testStationEUI, models.EventTypeBaseStationPingAnswered, recorderOccurredAt,
		map[string]interface{}{models.EventDetailKeyOpID: recorderPingOpID, models.EventDetailKeyResult: 0}))

	require.Len(t, store.events, 1)
	event := store.events[0]
	assert.Empty(t, event.UserID)
	assert.Equal(t, titleBSPingAnswered, event.Title)
	assert.Contains(t, event.Description, recorderStationEUI)
	details := recordedDetails(t, event)
	assert.EqualValues(t, 0, details[models.EventDetailKeyResult])
}

// A station's answer to an operator's status request reads with its opId.
func TestRecordEvent_StatusAnsweredNamesStationAndOpID(t *testing.T) {
	store := &capturedEvents{}
	recorder := NewPersistentEventRecorder(logger.NewNop(), store, recorderDefaultTenant)
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), recorderStationTenant)

	require.NoError(t, recorder.RecordEvent(ctx, testStationEUI, models.EventTypeBaseStationStatusAnswered, recorderOccurredAt,
		map[string]interface{}{models.EventDetailKeyOpID: recorderPingOpID, models.EventDetailKeyBaseStationName: testStationName}))

	require.Len(t, store.events, 1)
	event := store.events[0]
	assert.Equal(t, models.EventCategoryBaseStation, event.Category)
	assert.Equal(t, titleBSStatusAnswered, event.Title)
	assert.Contains(t, event.Description, `"`+testStationName+`"`)
	assert.Contains(t, event.Description, recorderStationEUI)
	assert.Contains(t, event.Description, "-41")
	details := recordedDetails(t, event)
	assert.EqualValues(t, recorderPingOpID, details[models.EventDetailKeyOpID])
}

// Ping events read with the station's name and EUI, as its connectivity
// events do.
func TestRecordEvent_PingEventsNameTheStation(t *testing.T) {
	for _, eventType := range []string{models.EventTypeBaseStationPingSent, models.EventTypeBaseStationPingAnswered} {
		t.Run(eventType, func(t *testing.T) {
			store := &capturedEvents{}
			recorder := NewPersistentEventRecorder(logger.NewNop(), store, recorderDefaultTenant)

			require.NoError(t, recorder.RecordEvent(testutil.TestContext(), testStationEUI, eventType, recorderOccurredAt, map[string]interface{}{
				models.EventDetailKeyOpID: recorderPingOpID, models.EventDetailKeyBaseStationName: testStationName}))

			require.Len(t, store.events, 1)
			assert.Contains(t, store.events[0].Description, `"`+testStationName+`"`)
			assert.Contains(t, store.events[0].Description, recorderStationEUI)
			assert.Contains(t, store.events[0].Description, "-41")
		})
	}
}

// Connectivity events keep naming the station, and without a tenant in ctx
// the configured tenant owns the event.
func TestRecordEvent_OnlineNamesStationForDefaultTenant(t *testing.T) {
	store := &capturedEvents{}
	recorder := NewPersistentEventRecorder(logger.NewNop(), store, recorderDefaultTenant)

	require.NoError(t, recorder.RecordEvent(testutil.TestContext(), testStationEUI, models.EventTypeBaseStationOnline, recorderOccurredAt,
		map[string]interface{}{models.EventDetailKeyBaseStationName: testStationName}))

	require.Len(t, store.events, 1)
	assert.Equal(t, recorderDefaultTenant, store.events[0].TenantID)
	assert.Equal(t, models.EventSeverityInfo, store.events[0].Severity)
	assert.Contains(t, store.events[0].Title, testStationName)
}

// A failed write is logged, never returned: the action it records stands.
func TestRecordEvent_StoreFailureIsNotReturned(t *testing.T) {
	store := &capturedEvents{err: errEventStoreDown}
	recorder := NewPersistentEventRecorder(logger.NewNop(), store, recorderDefaultTenant)

	assert.NoError(t, recorder.RecordEvent(testutil.TestContext(), testStationEUI, models.EventTypeBaseStationOffline, recorderOccurredAt, nil))
}

// The event occurred when its caller observed the exchange, however late the
// recording reaches the store; the store stamps when it was recorded.
func TestRecordEvent_StoresTheSuppliedOccurrenceTime(t *testing.T) {
	store := &capturedEvents{}
	recorder := NewPersistentEventRecorder(logger.NewNop(), store, recorderDefaultTenant)

	require.NoError(t, recorder.RecordEvent(testutil.TestContext(), testStationEUI, models.EventTypeBaseStationPingSent, recorderOccurredAt,
		map[string]interface{}{models.EventDetailKeyOpID: recorderPingOpID}))

	require.Len(t, store.events, 1)
	assert.True(t, recorderOccurredAt.Equal(store.events[0].CreatedAt),
		"occurred_at %s, want %s", store.events[0].CreatedAt, recorderOccurredAt)
	assert.True(t, store.events[0].UpdatedAt.IsZero(), "recorded_at is the store's to stamp")
}
