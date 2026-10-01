// Package events provides tests for the events service implementation.
package events

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Timing and fixture policy shared by the streaming tests.
const (
	// testStreamPollInterval keeps stream polls fast so tests finish quickly.
	testStreamPollInterval = 10 * time.Millisecond
	// testStreamOverlap is how far each stream read reaches back in storage order.
	testStreamOverlap = time.Second
	// testStreamBatchSize is a non-default batch size to prove it is honored.
	testStreamBatchSize = 25
	// testCloseTimeout bounds the wait for a channel to close.
	testCloseTimeout = 100 * time.Millisecond
	// testEmitTimeout bounds the wait for a stream to emit an event.
	testEmitTimeout = 200 * time.Millisecond
	// testDedupStreamTimeout bounds a stream that must span several polls.
	testDedupStreamTimeout = 300 * time.Millisecond
	// testCollectTimeout bounds the collection window for emitted events.
	testCollectTimeout = 250 * time.Millisecond
	// testEventType is the fixture event type used by streaming tests.
	testEventType = "test.event"
)

// mockEventStore implements SystemEventStore for testing.
type mockEventStore struct {
	mu            sync.Mutex
	events        []*models.SystemEvent
	err           error
	listCallCount int
	lastFilter    *EventFilter
	lastLimit     int
	lastOffset    int
	listCalled    chan struct{} // Signal channel for test synchronization
}

func newMockEventStore() *mockEventStore {
	return &mockEventStore{
		listCalled: make(chan struct{}, 10),
	}
}

// add stores events while a stream polls the store, stamping each with the
// time it was stored as the database does.
func (m *mockEventStore) add(events ...*models.SystemEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range events {
		e.StoredAt = time.Now()
	}
	m.events = append(m.events, events...)
}

// matching applies the inclusive time bounds, the storage bound and the
// newest-first order the real store applies in SQL; ties keep the order they
// were stored in.
func (m *mockEventStore) matching(filter *EventFilter) []*models.SystemEvent {
	if filter == nil {
		filter = &EventFilter{}
	}
	var filtered []*models.SystemEvent
	for _, e := range m.events {
		if filter.StartTime != nil && e.CreatedAt.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && e.CreatedAt.After(*filter.EndTime) {
			continue
		}
		if filter.StoredSince != nil && e.StoredAt.Before(*filter.StoredSince) {
			continue
		}
		filtered = append(filtered, e)
	}
	at := func(e *models.SystemEvent) time.Time { return e.CreatedAt }
	if filter.OrderBy == models.EventOrderByStored {
		at = func(e *models.SystemEvent) time.Time { return e.StoredAt }
	}
	slices.SortStableFunc(filtered, func(a, b *models.SystemEvent) int { return at(b).Compare(at(a)) })
	return filtered
}

func (m *mockEventStore) GetEvents(_ context.Context, _ int64, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listCallCount++
	m.lastFilter = filter
	m.lastLimit = limit
	m.lastOffset = offset
	// Signal that GetEvents was called (non-blocking)
	select {
	case m.listCalled <- struct{}{}:
	default:
	}
	if m.err != nil {
		return nil, m.err
	}
	matched := m.matching(filter)
	if offset >= len(matched) {
		return nil, nil
	}
	return matched[offset:min(offset+limit, len(matched))], nil
}

func (m *mockEventStore) CountEvents(_ context.Context, _ int64, filter *EventFilter) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return 0, m.err
	}
	return int64(len(m.matching(filter))), nil
}

// mockResolver implements EUIResolver for testing.
type mockResolver struct {
	bsID *int64
	epID *int64
}

func (m *mockResolver) ResolveBaseStationID(_ context.Context, _ int64, _ []byte) (*int64, error) {
	return m.bsID, nil
}

func (m *mockResolver) ResolveEndpointID(_ context.Context, _ int64, _ []byte) (*int64, error) {
	return m.epID, nil
}

// mockLogger implements logger.Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...interface{})                      {}
func (m *mockLogger) Info(_ string, _ ...interface{})                       {}
func (m *mockLogger) Warn(_ string, _ ...interface{})                       {}
func (m *mockLogger) Error(_ string, _ ...interface{})                      {}
func (m *mockLogger) Fatal(_ string, _ ...interface{})                      {}
func (m *mockLogger) DebugContext(_ context.Context, _ string, _ ...any)    {}
func (m *mockLogger) InfoContext(_ context.Context, _ string, _ ...any)     {}
func (m *mockLogger) WarnContext(_ context.Context, _ string, _ ...any)     {}
func (m *mockLogger) ErrorContext(_ context.Context, _ string, _ ...any)    {}
func (m *mockLogger) FatalContext(_ context.Context, _ string, _ ...any)    {}
func (m *mockLogger) WithField(_ string, _ interface{}) logger.Logger       { return m }
func (m *mockLogger) WithFields(_ map[string]interface{}) logger.Logger     { return m }
func (m *mockLogger) With(_ ...interface{}) logger.Logger                   { return m }
func (m *mockLogger) WithContext(_ context.Context, _ ...any) logger.Logger { return m }
func (m *mockLogger) SetLevel(_ string)                                     {}

// TestNew_DefaultBatchSize verifies that the constructor clamps invalid batch sizes.
func TestNew_DefaultBatchSize(t *testing.T) {
	store := newMockEventStore()
	log := &mockLogger{}

	// Test with zero batch size - should clamp to default
	svc := New(store, &mockResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), 0, log)
	require.NotNil(t, svc)
	assert.Equal(t, DefaultEventStreamBatchSize, svc.streamBatchSize)

	// Test with negative batch size - should clamp to default
	svc = New(store, &mockResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), -10, log)
	require.NotNil(t, svc)
	assert.Equal(t, DefaultEventStreamBatchSize, svc.streamBatchSize)

	// Test with positive batch size - should use provided value
	svc = New(store, &mockResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), 50, log)
	require.NotNil(t, svc)
	assert.Equal(t, 50, svc.streamBatchSize)
}

// TestList_Success verifies basic list functionality.
func TestList_Success(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	store.events = []*models.SystemEvent{
		{ID: "1", TenantID: "1", Category: "test", CreatedAt: now},
		{ID: "2", TenantID: "1", Category: "test", CreatedAt: now},
	}

	svc := New(store, &mockResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})
	ctx := testutil.TestContext()

	filters := &grpcservices.EventFilters{
		Categories: []string{"test"},
	}

	events, total, err := svc.List(ctx, 1, filters, 10, 0)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, int64(2), total)
}

// TestStream_ContextCancellation verifies that streaming stops when context is cancelled.
func TestStream_ContextCancellation(t *testing.T) {
	store := newMockEventStore()
	svc := New(store, &mockResolver{}, registrationAt{registered: true}, testStreamPollInterval, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})

	ctx, cancel := testutil.TestContextWithCancel()

	ch, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Cancel context immediately
	cancel()

	// Channel should close
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Expected channel to be closed after context cancellation")
		}
	case <-time.After(testCloseTimeout):
		t.Error("Expected channel to close after context cancellation")
	}
}

// awaitReads waits until the stream has read the store reads times; a read
// takes the store's lock before it signals, so a row added afterwards is not
// part of it.
func awaitReads(ctx context.Context, t *testing.T, store *mockEventStore, reads int) {
	t.Helper()
	for range reads {
		select {
		case <-store.listCalled:
		case <-ctx.Done():
			t.Fatal("timed out waiting for the stream to read the store")
		}
	}
}

// Reads a stream's baseline takes: an empty store answers the read of the
// newest row; with history, the rows within the overlap of it are read next.
const (
	baselineReadsEmpty       = 1
	baselineReadsWithHistory = 2
)

// collectIDs counts the IDs a stream emits over the collection window.
func collectIDs(ch <-chan *grpcservices.Event) map[string]int {
	seen := make(map[string]int)
	timeout := time.After(testCollectTimeout)
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return seen
			}
			seen[event.ID]++
		case <-timeout:
			return seen
		}
	}
}

// A stream delivers the events recorded after it opened, once each, and not
// the history stored before it.
func TestStream_EmitsOnlyEventsStoredAfterOpen(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	store.add(&models.SystemEvent{ID: "old", TenantID: "1", EventType: testEventType, CreatedAt: now.Add(-time.Minute)})

	svc := New(store, &mockResolver{}, registrationAt{registered: true}, testStreamPollInterval, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})
	ctx, cancel := testutil.TestContextWithTimeout(testDedupStreamTimeout)
	defer cancel()

	ch, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)
	awaitReads(ctx, t, store, baselineReadsWithHistory)
	store.add(&models.SystemEvent{ID: "new", TenantID: "1", EventType: testEventType, CreatedAt: now})

	assert.Equal(t, map[string]int{"new": 1}, collectIDs(ch))
}

// TestStream_UsesBatchSize verifies that streaming respects batch size.
func TestStream_UsesBatchSize(t *testing.T) {
	store := newMockEventStore()
	batchSize := testStreamBatchSize
	svc := New(store, &mockResolver{}, registrationAt{registered: true}, testStreamPollInterval, testStreamOverlap, streamwake.NewSignal(), batchSize, &mockLogger{})

	ctx, cancel := testutil.TestContextWithTimeout(testEmitTimeout)
	defer cancel()

	_, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)

	// Wait for at least one poll with synchronization (no time.Sleep)
	select {
	case <-store.listCalled:
		// GetEvents was called
	case <-ctx.Done():
		t.Fatal("timed out waiting for store.GetEvents to be called")
	}

	// Verify batch size was used in query
	assert.Equal(t, batchSize, store.lastLimit)
}

// TestConvertFilters_Nil verifies nil filter handling.
func TestConvertFilters_Nil(t *testing.T) {
	result, err := convertFilters(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestConvertFilters_WithValues verifies filter conversion.
func TestConvertFilters_WithValues(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	input := &grpcservices.EventFilters{
		Categories: []string{"test", "system"},
		Severity:   []string{"info", "warning"},
		StartTime:  &now,
		EndTime:    &later,
	}

	result, err := convertFilters(input)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []string{"test", "system"}, result.Categories)
	assert.Equal(t, []string{"info", "warning"}, result.Severity)
	require.NotNil(t, result.StartTime)
	require.NotNil(t, result.EndTime)
	assert.Equal(t, now, *result.StartTime)
	assert.Equal(t, later, *result.EndTime)
}

// More events than a batch holds, stored within one poll interval, all stream once.
func TestStream_DeliversABurstLargerThanABatch(t *testing.T) {
	const burst = 2*testStreamBatchSize + testStreamBatchSize/2
	store := newMockEventStore()
	svc := New(store, &mockResolver{}, registrationAt{registered: true}, testStreamPollInterval, testStreamOverlap, streamwake.NewSignal(), testStreamBatchSize, &mockLogger{})
	ctx, cancel := testutil.TestContextWithTimeout(testDedupStreamTimeout)
	defer cancel()

	ch, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)
	awaitReads(ctx, t, store, baselineReadsEmpty)
	now := time.Now()
	want := map[string]int{}
	events := make([]*models.SystemEvent, burst)
	for i := range events {
		id := strconv.Itoa(i)
		events[i] = &models.SystemEvent{ID: id, TenantID: "1", EventType: testEventType, CreatedAt: now.Add(time.Duration(i) * time.Millisecond)}
		want[id] = 1
	}
	store.add(events...)

	assert.Equal(t, want, collectIDs(ch))
}

// TestStream_NoDuplicatesOnSameTimestamp verifies events with same CreatedAt are each emitted once.
func TestStream_NoDuplicatesOnSameTimestamp(t *testing.T) {
	store := newMockEventStore()
	svc := New(store, &mockResolver{}, registrationAt{registered: true}, testStreamPollInterval, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})
	ctx, cancel := testutil.TestContextWithTimeout(testDedupStreamTimeout)
	defer cancel()

	ch, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)
	awaitReads(ctx, t, store, baselineReadsEmpty)
	now := time.Now()
	store.add(
		&models.SystemEvent{ID: "1", TenantID: "1", EventType: testEventType, CreatedAt: now},
		&models.SystemEvent{ID: "2", TenantID: "1", EventType: testEventType, CreatedAt: now},
	)

	assert.Equal(t, map[string]int{"1": 1, "2": 1}, collectIDs(ch))
}

func TestSeveritiesFor_OutcomeNarrowsSeverities(t *testing.T) {
	got, err := severitiesFor(grpcservices.EventOutcomeFailure, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{models.EventSeverityError, models.EventSeverityCritical}, got)

	got, err = severitiesFor(grpcservices.EventOutcomeSuccess, []string{models.EventSeverityWarning, models.EventSeverityError})
	require.NoError(t, err)
	assert.Equal(t, []string{models.EventSeverityWarning}, got, "severities outside the outcome are dropped")

	got, err = severitiesFor("", []string{models.EventSeverityCritical})
	require.NoError(t, err)
	assert.Equal(t, []string{models.EventSeverityCritical}, got)

	_, err = severitiesFor("sideways", nil)
	assert.ErrorIs(t, err, ErrInvalidOutcome)

	_, err = severitiesFor(grpcservices.EventOutcomeSuccess, []string{models.EventSeverityError})
	assert.ErrorIs(t, err, ErrOutcomeSeverityMismatch)
}

func TestConvertFilters_MapsLogFilters(t *testing.T) {
	opID := int64(-5)
	filter, err := convertFilters(&grpcservices.EventFilters{OpID: &opID, EpEUI: "AA", BsEUI: "BB", Search: "boot", Outcome: grpcservices.EventOutcomeFailure})
	require.NoError(t, err)
	assert.Equal(t, &opID, filter.OpID)
	assert.Equal(t, "AA", filter.EndpointEUI)
	assert.Equal(t, "BB", filter.BaseStationEUI)
	assert.Equal(t, "boot", filter.Search)
	assert.ElementsMatch(t, []string{models.EventSeverityError, models.EventSeverityCritical}, filter.Severity)
}

// unknownEventType stands in for an event type the projection has never heard of.
const unknownEventType = "vendor.custom"

func TestList_ProjectsDetailsPerEventType(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	store.events = []*models.SystemEvent{
		{ID: "1", TenantID: "1", EventType: models.EventTypeEndpointCreated, CreatedAt: now, Details: json.RawMessage(`{"epEui":"AA","nwkSnKey":"secret"}`)},
		{ID: "2", TenantID: "1", EventType: unknownEventType, CreatedAt: now, Details: json.RawMessage(`{"anything":1}`)},
	}

	svc := New(store, &mockResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})
	events, _, err := svc.List(testutil.TestContext(), 1, &grpcservices.EventFilters{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.JSONEq(t, `{"epEui":"AA"}`, string(events[0].Data), "only allowlisted keys reach the API")
	assert.JSONEq(t, `{}`, string(events[1].Data), "an unknown event type exposes no details")
}

// testActorID stands in for the user an audit event records.
const testActorID = "7a1d9c52-4f0e-4b8e-9d7a-3f2c1b0e5d44"

// testActorEmail stands in for the email the store resolves for testActorID.
const testActorEmail = "operator@tenant.example"

func TestList_CarriesActingUser(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	store.events = []*models.SystemEvent{
		{ID: "1", TenantID: "1", EventType: models.EventTypeDownlinkQueued, Category: models.EventCategoryAudit, UserID: testActorID, UserEmail: testActorEmail, CreatedAt: now},
		{ID: "2", TenantID: "1", EventType: models.EventTypeServiceStarted, Category: models.EventCategorySystem, CreatedAt: now},
	}

	svc := New(store, &mockResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})
	events, _, err := svc.List(testutil.TestContext(), 1, &grpcservices.EventFilters{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, testActorID, events[0].UserID, "an operator action names its actor")
	assert.Equal(t, testActorEmail, events[0].UserEmail, "and the actor's email the store resolved")
	assert.Empty(t, events[1].UserID, "a service-raised event has no actor")
	assert.Empty(t, events[1].UserEmail)
}
