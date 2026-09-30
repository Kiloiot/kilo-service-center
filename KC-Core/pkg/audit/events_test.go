package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	testTenant  = int64(9)
	testUser    = "user-1"
	testEpID    = int64(77)
	testBsID    = int64(78)
	testDetails = "value"
)

var (
	testNow    = time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	errStorage = errors.New("storage down")
)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testNow }

var _ clock.Clock = fixedClock{}

type captureWriter struct {
	events []*models.SystemEvent
	err    error
}

func (w *captureWriter) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	if w.err != nil {
		return w.err
	}
	w.events = append(w.events, event)
	return nil
}

func TestEmitAudit_RendersAuditEventFromClock(t *testing.T) {
	writer := &captureWriter{}
	emitter, err := NewEmitter(writer, fixedClock{})
	require.NoError(t, err)
	source := uuid.New()
	epID := testEpID

	err = emitter.EmitAudit(testutil.TestContext(), Event{
		TenantID:   testTenant,
		SourceID:   &source,
		Category:   models.EventCategoryEndpoint,
		EventType:  models.EventTypeEndpointCreated,
		Title:      models.EventTitleEndpointCreated,
		SourceName: "ep",
		UserID:     testUser,
		EndpointID: &epID,
		Details:    map[string]any{"key": testDetails},
	})
	require.NoError(t, err)
	require.Len(t, writer.events, 1)
	got := writer.events[0]
	assert.Equal(t, "9", got.TenantID)
	assert.Equal(t, models.EventCategoryEndpoint, got.Category, "endpoint lifecycle stays in the endpoint category")
	assert.Equal(t, models.EventSeverityInfo, got.Severity)
	assert.Equal(t, models.SourceTypeAPI, got.SourceType, "the source type defaults to the API")
	assert.Equal(t, &source, got.SourceID)
	assert.Equal(t, testUser, got.UserID)
	assert.Equal(t, &epID, got.EndpointID)
	assert.JSONEq(t, `{"key":"value"}`, string(got.Details))
	assert.Equal(t, testNow, got.CreatedAt)
	assert.Equal(t, testNow, got.UpdatedAt)
}

func TestEmitAudit_KeepsAnExplicitSourceType(t *testing.T) {
	writer := &captureWriter{}
	emitter, err := NewEmitter(writer, fixedClock{})
	require.NoError(t, err)
	require.NoError(t, emitter.EmitAudit(testutil.TestContext(),
		Event{TenantID: testTenant, SourceType: models.SourceTypeSystem, EventType: models.EventTypeCertificateServerRenewed}))
	require.Len(t, writer.events, 1)
	assert.Equal(t, models.SourceTypeSystem, writer.events[0].SourceType)
}

func TestEmitAudit_RejectsAnEventWithoutATenant(t *testing.T) {
	for _, tenant := range []int64{0, -1} {
		writer := &captureWriter{}
		emitter, err := NewEmitter(writer, fixedClock{})
		require.NoError(t, err)
		err = emitter.EmitAudit(testutil.TestContext(), Event{TenantID: tenant, EventType: models.EventTypeCertificateServerRenewed})
		assert.ErrorIs(t, err, ErrTenantRequired, "tenant %d", tenant)
		assert.Empty(t, writer.events, "nothing is written for tenant %d", tenant)
	}
}

func TestEmitAudit_KeepsARemovedSubjectInTheDetails(t *testing.T) {
	epID, bsID := testEpID, testBsID
	cases := []struct {
		event     Event
		detailKey string
		id        int64
	}{
		{Event{EventType: models.EventTypeEndpointDeleted, EndpointID: &epID}, models.EventDetailKeyEndpointID, epID},
		{Event{EventType: models.EventTypeBSDeregistered, BaseStationID: &bsID}, models.EventDetailKeyBaseStationID, bsID},
	}
	for _, tc := range cases {
		writer := &captureWriter{}
		emitter, err := NewEmitter(writer, fixedClock{})
		require.NoError(t, err)
		tc.event.TenantID = testTenant
		tc.event.Details = map[string]any{"key": testDetails}
		require.NoError(t, emitter.EmitAudit(testutil.TestContext(), tc.event))
		require.Len(t, writer.events, 1)
		got := writer.events[0]
		assert.Nil(t, got.EndpointID, "%s: the deleted row cannot be linked", tc.event.EventType)
		assert.Nil(t, got.BasestationID, "%s: the deleted row cannot be linked", tc.event.EventType)
		var details map[string]any
		require.NoError(t, json.Unmarshal(got.Details, &details))
		assert.EqualValues(t, tc.id, details[tc.detailKey], "%s keeps the removed row's id", tc.event.EventType)
		assert.Equal(t, testDetails, details["key"], "%s keeps the caller's details", tc.event.EventType)
		assert.Len(t, tc.event.Details, 1, "%s: the caller's details are not mutated", tc.event.EventType)
	}
}

func TestEmitAudit_LinksALiveSubject(t *testing.T) {
	writer := &captureWriter{}
	emitter, err := NewEmitter(writer, fixedClock{})
	require.NoError(t, err)
	bsID := testBsID
	require.NoError(t, emitter.EmitAudit(testutil.TestContext(),
		Event{TenantID: testTenant, EventType: models.EventTypeCertificateGenerated, BaseStationID: &bsID}))
	require.Len(t, writer.events, 1)
	assert.Equal(t, &bsID, writer.events[0].BasestationID)
	assert.JSONEq(t, `null`, string(writer.events[0].Details), "a live subject is linked, not copied")
}

func TestEmitAudit_FilesTheEventUnderTheCallersCategory(t *testing.T) {
	cases := map[string]string{
		models.EventCategoryEndpoint:    models.EventCategoryEndpoint,
		models.EventCategoryBaseStation: models.EventCategoryBaseStation,
		"":                              models.EventCategoryAudit,
	}
	for requested, filed := range cases {
		writer := &captureWriter{}
		emitter, err := NewEmitter(writer, fixedClock{})
		require.NoError(t, err)
		require.NoError(t, emitter.EmitAudit(testutil.TestContext(),
			Event{TenantID: testTenant, Category: requested, EventType: models.EventTypeEndpointCreated, UserID: testUser}))
		require.Len(t, writer.events, 1)
		assert.Equal(t, filed, writer.events[0].Category, "requested category %q", requested)
		assert.Equal(t, testUser, writer.events[0].UserID, "the acting user stays on the record")
	}
}

func TestEmitAudit_ReportsFailures(t *testing.T) {
	var nilEmitter *Emitter
	assert.ErrorIs(t, nilEmitter.EmitAudit(testutil.TestContext(), Event{}), ErrEmitterNotConfigured)
	_, err := NewEmitter(nil, fixedClock{})
	assert.ErrorIs(t, err, ErrNilEventWriter)
	_, err = NewEmitter(&captureWriter{}, nil)
	assert.ErrorIs(t, err, ErrNilClock)

	failing, err := NewEmitter(&captureWriter{err: errStorage}, fixedClock{})
	require.NoError(t, err)
	assert.ErrorIs(t, failing.EmitAudit(testutil.TestContext(), Event{TenantID: testTenant, EventType: models.EventTypeUserCreated}), errStorage)

	unmarshalable, err := NewEmitter(&captureWriter{}, fixedClock{})
	require.NoError(t, err)
	err = unmarshalable.EmitAudit(testutil.TestContext(), Event{TenantID: testTenant, Details: map[string]any{"fn": func() {}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), errWrapMarshalDetails)
}

type stubEmitter struct {
	events []Event
	err    error
}

func (e *stubEmitter) EmitAudit(_ context.Context, ev Event) error {
	e.events = append(e.events, ev)
	return e.err
}

type countingDrops struct{ byType map[string]int }

func (c *countingDrops) Inc(eventType string) { c.byType[eventType]++ }

type captureLogger struct {
	logger.Logger
	errors []string
}

func (l *captureLogger) ErrorContext(_ context.Context, msg string, _ ...any) {
	l.errors = append(l.errors, msg)
}

func TestRecorder_WritesTheEvent(t *testing.T) {
	emitter := &stubEmitter{}
	drops := &countingDrops{byType: map[string]int{}}
	log := &captureLogger{Logger: logger.NewNop()}
	recorder, err := NewRecorder(emitter, log, drops)
	require.NoError(t, err)

	recorder.Record(testutil.TestContext(), Event{TenantID: testTenant, EventType: models.EventTypeEndpointDeleted})

	require.Len(t, emitter.events, 1)
	assert.Empty(t, drops.byType)
	assert.Empty(t, log.errors)
}

func TestRecorder_ReportsADroppedEventAsAnIntegrityFailure(t *testing.T) {
	emitter := &stubEmitter{err: errStorage}
	drops := &countingDrops{byType: map[string]int{}}
	log := &captureLogger{Logger: logger.NewNop()}
	recorder, err := NewRecorder(emitter, log, drops)
	require.NoError(t, err)

	recorder.Record(testutil.TestContext(), Event{TenantID: testTenant, EventType: models.EventTypeBSDeregistered})
	recorder.Record(testutil.TestContext(), Event{TenantID: testTenant, EventType: models.EventTypeBSDeregistered})

	assert.Equal(t, map[string]int{models.EventTypeBSDeregistered: 2}, drops.byType, "every drop is counted by event type")
	assert.Equal(t, []string{LogEventDropped, LogEventDropped}, log.errors, "every drop is logged as an error")
}

func TestNewRecorder_RequiresEveryCollaborator(t *testing.T) {
	drops := &countingDrops{byType: map[string]int{}}
	_, err := NewRecorder(nil, logger.NewNop(), drops)
	assert.ErrorIs(t, err, ErrNilEmitter)
	_, err = NewRecorder(&stubEmitter{}, nil, drops)
	assert.ErrorIs(t, err, ErrNilLogger)
	_, err = NewRecorder(&stubEmitter{}, logger.NewNop(), nil)
	assert.ErrorIs(t, err, ErrNilDropCounter)
}

func TestPrometheusDropCounter_CountsByEventType(t *testing.T) {
	reg := prometheus.NewRegistry()
	drops, err := NewPrometheusDropCounter(reg)
	require.NoError(t, err)

	drops.Inc(models.EventTypeEndpointDeleted)
	drops.Inc(models.EventTypeEndpointDeleted)
	drops.Inc(models.EventTypeCertificateServerRenewed)

	assert.InDelta(t, 2, promtestutil.ToFloat64(drops.dropped.WithLabelValues(models.EventTypeEndpointDeleted)), 0)
	assert.InDelta(t, 1, promtestutil.ToFloat64(drops.dropped.WithLabelValues(models.EventTypeCertificateServerRenewed)), 0)

	_, err = NewPrometheusDropCounter(reg)
	assert.Error(t, err, "the counter registers once per registry")
	_, err = NewPrometheusDropCounter(nil)
	assert.ErrorIs(t, err, ErrNilRegisterer)
}

func TestRecorder_NamesTheActingUser(t *testing.T) {
	const actor = "0b0d7c1e-2f5c-4c3a-9c2e-1f8a6f0a1d2e"
	emitter := &stubEmitter{}
	recorder, err := NewRecorder(emitter, logger.NewNop(), &countingDrops{byType: map[string]int{}})
	require.NoError(t, err)
	ctx := pkgcontext.WithUserID(testutil.TestContext(), actor)

	recorder.Record(ctx, Event{TenantID: testTenant, EventType: models.EventTypeEndpointDeleted})
	recorder.Record(ctx, Event{TenantID: testTenant, EventType: models.EventTypeUserCreated, UserID: testUser})
	recorder.Record(pkgcontext.WithUserID(testutil.TestContext(), "not-a-user-id"), Event{TenantID: testTenant})
	recorder.Record(testutil.TestContext(), Event{TenantID: testTenant})

	require.Len(t, emitter.events, 4)
	assert.Equal(t, actor, emitter.events[0].UserID, "the acting user comes from the request context")
	assert.Equal(t, testUser, emitter.events[1].UserID, "an explicit user is kept")
	assert.Empty(t, emitter.events[2].UserID, "a malformed user id is not recorded")
	assert.Empty(t, emitter.events[3].UserID, "a call without a user records none")
}

func TestRecorder_NamesTheActingServiceAccount(t *testing.T) {
	keyID := uuid.MustParse("5b1f0c1e-3f4e-4d1a-9f2b-7c0d6e8a9b10")
	emitter := &stubEmitter{}
	recorder, err := NewRecorder(emitter, logger.NewNop(), &countingDrops{byType: map[string]int{}})
	require.NoError(t, err)
	details := map[string]any{models.EventDetailKeyEpEui: "70B3D56770111505"}

	recorder.Record(pkgcontext.WithServiceAccountID(testutil.TestContext(), keyID), Event{TenantID: testTenant, Details: details})

	require.Len(t, emitter.events, 1)
	assert.Empty(t, emitter.events[0].UserID)
	assert.Equal(t, keyID.String(), emitter.events[0].Details[models.EventDetailKeyServiceAccount])
	assert.Equal(t, "70B3D56770111505", emitter.events[0].Details[models.EventDetailKeyEpEui])
	assert.NotContains(t, details, models.EventDetailKeyServiceAccount, "the caller's details are not modified")
}
