// Package events provides tests for the events service implementation.
package events

import (
	"context"
	"testing"
	"time"

	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventStore implements SystemEventStore for testing.
type mockEventStore struct {
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

func (m *mockEventStore) List(_ context.Context, _ int64, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error) {
	m.listCallCount++
	m.lastFilter = filter
	m.lastLimit = limit
	m.lastOffset = offset
	// Signal that List was called (non-blocking)
	select {
	case m.listCalled <- struct{}{}:
	default:
	}
	if m.err != nil {
		return nil, 0, m.err
	}
	// Simulate DB filtering: only return events with CreatedAt > StartTime
	// This validates that the in-loop filter in Stream() prevents duplicates
	if filter != nil && filter.StartTime != nil {
		startTime := time.Unix(*filter.StartTime, 0)
		var filtered []*models.SystemEvent
		for _, e := range m.events {
			if e.CreatedAt.After(startTime) {
				filtered = append(filtered, e)
			}
		}
		return filtered, int64(len(filtered)), nil
	}
	return m.events, int64(len(m.events)), nil
}

func (m *mockEventStore) ListByBaseStation(_ context.Context, _ int64, _ []byte, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error) {
	m.listCallCount++
	m.lastFilter = filter
	m.lastLimit = limit
	m.lastOffset = offset
	// Signal that List was called (non-blocking)
	select {
	case m.listCalled <- struct{}{}:
	default:
	}
	if m.err != nil {
		return nil, 0, m.err
	}
	// Simulate DB filtering: only return events with CreatedAt > StartTime
	// This validates that the in-loop filter in StreamByBaseStation() prevents duplicates
	if filter != nil && filter.StartTime != nil {
		startTime := time.Unix(*filter.StartTime, 0)
		var filtered []*models.SystemEvent
		for _, e := range m.events {
			if e.CreatedAt.After(startTime) {
				filtered = append(filtered, e)
			}
		}
		return filtered, int64(len(filtered)), nil
	}
	return m.events, int64(len(m.events)), nil
}

func (m *mockEventStore) ListByEndPoint(_ context.Context, _ int64, _ []byte, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error) {
	m.listCallCount++
	m.lastFilter = filter
	m.lastLimit = limit
	m.lastOffset = offset
	// Signal that List was called (non-blocking)
	select {
	case m.listCalled <- struct{}{}:
	default:
	}
	if m.err != nil {
		return nil, 0, m.err
	}
	// Simulate DB filtering: only return events with CreatedAt > StartTime
	// This validates that the in-loop filter in StreamByEndPoint() prevents duplicates
	if filter != nil && filter.StartTime != nil {
		startTime := time.Unix(*filter.StartTime, 0)
		var filtered []*models.SystemEvent
		for _, e := range m.events {
			if e.CreatedAt.After(startTime) {
				filtered = append(filtered, e)
			}
		}
		return filtered, int64(len(filtered)), nil
	}
	return m.events, int64(len(m.events)), nil
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
	svc := New(store, time.Second, 0, log)
	require.NotNil(t, svc)
	assert.Equal(t, DefaultEventStreamBatchSize, svc.streamBatchSize)

	// Test with negative batch size - should clamp to default
	svc = New(store, time.Second, -10, log)
	require.NotNil(t, svc)
	assert.Equal(t, DefaultEventStreamBatchSize, svc.streamBatchSize)

	// Test with positive batch size - should use provided value
	svc = New(store, time.Second, 50, log)
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

	svc := New(store, time.Second, 100, &mockLogger{})
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
	svc := New(store, 10*time.Millisecond, 100, &mockLogger{})

	ctx, cancel := context.WithCancel(testutil.TestContext())

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
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected channel to close after context cancellation")
	}
}

// TestStream_EmitsEvents verifies that streaming emits events from the store.
func TestStream_EmitsEvents(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	store.events = []*models.SystemEvent{
		{ID: "1", TenantID: "1", Category: "test", EventType: "test.event", CreatedAt: now},
	}

	svc := New(store, 10*time.Millisecond, 100, &mockLogger{})
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 200*time.Millisecond)
	defer cancel()

	ch, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)

	// MUST receive at least one event - context timeout is a failure
	select {
	case event := <-ch:
		require.NotNil(t, event, "expected non-nil event")
		assert.Equal(t, "1", event.ID)
		assert.Equal(t, "test.event", event.EventType)
	case <-ctx.Done():
		t.Fatal("timed out waiting for event - stream did not emit")
	}
}

// TestStream_UsesBatchSize verifies that streaming respects batch size.
func TestStream_UsesBatchSize(t *testing.T) {
	store := newMockEventStore()
	batchSize := 25
	svc := New(store, 10*time.Millisecond, batchSize, &mockLogger{})

	ctx, cancel := context.WithTimeout(testutil.TestContext(), 200*time.Millisecond)
	defer cancel()

	_, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)

	// Wait for at least one poll with synchronization (no time.Sleep)
	select {
	case <-store.listCalled:
		// List was called
	case <-ctx.Done():
		t.Fatal("timed out waiting for store.List to be called")
	}

	// Verify batch size was used in query
	assert.Equal(t, batchSize, store.lastLimit)
}

// TestStreamByBaseStation_ContextCancellation verifies base station streaming cancellation.
func TestStreamByBaseStation_ContextCancellation(t *testing.T) {
	store := newMockEventStore()
	svc := New(store, 10*time.Millisecond, 100, &mockLogger{})

	ctx, cancel := context.WithCancel(testutil.TestContext())
	bsEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	ch, err := svc.StreamByBaseStation(ctx, 1, bsEui, nil)
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Cancel context
	cancel()

	// Channel should close
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Expected channel to be closed after context cancellation")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected channel to close after context cancellation")
	}
}

// TestStreamByEndPoint_ContextCancellation verifies endpoint streaming cancellation.
func TestStreamByEndPoint_ContextCancellation(t *testing.T) {
	store := newMockEventStore()
	svc := New(store, 10*time.Millisecond, 100, &mockLogger{})

	ctx, cancel := context.WithCancel(testutil.TestContext())
	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	ch, err := svc.StreamByEndPoint(ctx, 1, epEui, nil)
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Cancel context
	cancel()

	// Channel should close
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Expected channel to be closed after context cancellation")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected channel to close after context cancellation")
	}
}

// TestConvertFilters_Nil verifies nil filter handling.
func TestConvertFilters_Nil(t *testing.T) {
	result := convertFilters(nil)
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

	result := convertFilters(input)
	require.NotNil(t, result)
	assert.Equal(t, []string{"test", "system"}, result.Categories)
	assert.Equal(t, []string{"info", "warning"}, result.Severity)
	require.NotNil(t, result.StartTime)
	require.NotNil(t, result.EndTime)
	assert.Equal(t, now.Unix(), *result.StartTime)
	assert.Equal(t, later.Unix(), *result.EndTime)
}

// TestStream_NoDuplicatesOnSameTimestamp verifies events with same CreatedAt are not duplicated.
func TestStream_NoDuplicatesOnSameTimestamp(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	// Two events with exact same timestamp
	store.events = []*models.SystemEvent{
		{ID: "1", TenantID: "1", Category: "test", EventType: "test.event", CreatedAt: now},
		{ID: "2", TenantID: "1", Category: "test", EventType: "test.event", CreatedAt: now},
	}

	svc := New(store, 10*time.Millisecond, 100, &mockLogger{})
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 300*time.Millisecond)
	defer cancel()

	ch, err := svc.Stream(ctx, 1, nil)
	require.NoError(t, err)

	// Collect all emitted event IDs
	seen := make(map[string]int)
	timeout := time.After(250 * time.Millisecond)

loop:
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				break loop
			}
			seen[event.ID]++
			// Early exit once expected count reached
			if len(seen) == 2 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	// Each event should be emitted exactly once (no duplicates)
	for id, count := range seen {
		assert.Equal(t, 1, count, "event %s should be emitted exactly once, got %d", id, count)
	}
	// Must receive all 2 events (not just "at least one")
	assert.Len(t, seen, 2, "should receive all 2 events")
}

// TestStreamByBaseStation_NoDuplicatesOnSameTimestamp verifies no duplicates in BS stream.
func TestStreamByBaseStation_NoDuplicatesOnSameTimestamp(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	store.events = []*models.SystemEvent{
		{ID: "1", TenantID: "1", Category: "test", EventType: "test.event", CreatedAt: now},
		{ID: "2", TenantID: "1", Category: "test", EventType: "test.event", CreatedAt: now},
	}

	svc := New(store, 10*time.Millisecond, 100, &mockLogger{})
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 300*time.Millisecond)
	defer cancel()

	bsEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	ch, err := svc.StreamByBaseStation(ctx, 1, bsEui, nil)
	require.NoError(t, err)

	seen := make(map[string]int)
	timeout := time.After(250 * time.Millisecond)

loop:
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				break loop
			}
			seen[event.ID]++
			// Early exit once expected count reached
			if len(seen) == 2 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	for id, count := range seen {
		assert.Equal(t, 1, count, "event %s should be emitted exactly once, got %d", id, count)
	}
	// Must receive all 2 events (not just "at least one")
	assert.Len(t, seen, 2, "should receive all 2 events")
}

// TestStreamByEndPoint_NoDuplicatesOnSameTimestamp verifies no duplicates in EP stream.
func TestStreamByEndPoint_NoDuplicatesOnSameTimestamp(t *testing.T) {
	store := newMockEventStore()
	now := time.Now()
	store.events = []*models.SystemEvent{
		{ID: "1", TenantID: "1", Category: "test", EventType: "test.event", CreatedAt: now},
		{ID: "2", TenantID: "1", Category: "test", EventType: "test.event", CreatedAt: now},
	}

	svc := New(store, 10*time.Millisecond, 100, &mockLogger{})
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 300*time.Millisecond)
	defer cancel()

	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	ch, err := svc.StreamByEndPoint(ctx, 1, epEui, nil)
	require.NoError(t, err)

	seen := make(map[string]int)
	timeout := time.After(250 * time.Millisecond)

loop:
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				break loop
			}
			seen[event.ID]++
			// Early exit once expected count reached
			if len(seen) == 2 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	for id, count := range seen {
		assert.Equal(t, 1, count, "event %s should be emitted exactly once, got %d", id, count)
	}
	// Must receive all 2 events (not just "at least one")
	assert.Len(t, seen, 2, "should receive all 2 events")
}
