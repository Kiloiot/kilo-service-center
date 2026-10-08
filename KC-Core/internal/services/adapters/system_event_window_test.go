package adapters

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	windowTestTenant   = int64(7)
	windowTestLimit    = 10
	windowTestBatch    = 10
	windowTestPoll     = time.Minute
	windowTestOverlap  = time.Second
	windowTestDelivery = time.Second
)

// windowTestAt is a moment between two whole seconds.
var windowTestAt = time.Date(2026, time.September, 29, 12, 40, 35, 321186000, time.UTC)

// windowRecorder is an event store that records the filters it is read with
// and serves its events inside their bounds, newest first.
type windowRecorder struct {
	interfaces.SystemEventStore
	mu      sync.Mutex
	events  []*models.SystemEvent
	filters []models.SystemEventFilter
}

func (r *windowRecorder) GetEvents(_ context.Context, filter models.SystemEventFilter) ([]*models.SystemEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filters = append(r.filters, filter)
	var inWindow []*models.SystemEvent
	for i := len(r.events) - 1; i >= 0; i-- {
		at, stored := r.events[i].CreatedAt, r.events[i].StoredAt
		if (filter.Since == nil || !at.Before(*filter.Since)) && (filter.Until == nil || !at.After(*filter.Until)) &&
			(filter.StoredSince == nil || !stored.Before(*filter.StoredSince)) {
			inWindow = append(inWindow, r.events[i])
		}
	}
	if filter.Offset >= len(inWindow) {
		return nil, nil
	}
	return inWindow[filter.Offset:min(filter.Offset+filter.Limit, len(inWindow))], nil
}

func (r *windowRecorder) CountEvents(ctx context.Context, filter models.SystemEventFilter) (int64, error) {
	events, err := r.GetEvents(ctx, filter)
	return int64(len(events)), err
}

func (r *windowRecorder) store(event *models.SystemEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *windowRecorder) recorded() []models.SystemEventFilter {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]models.SystemEventFilter(nil), r.filters...)
}

// A listing's window reaches the store exactly as requested, not widened to
// whole seconds.
func TestEventListing_ReadsTheRequestedWindowExactly(t *testing.T) {
	store := &windowRecorder{}
	svc := eventsservice.New(NewSystemEventStoreAdapter(store), nil, nil, windowTestPoll, windowTestOverlap, streamwake.NewSignal(), 0, logger.NewNop())
	end := windowTestAt.Add(time.Millisecond)

	_, _, err := svc.List(testutil.TestContext(), windowTestTenant, &grpcservices.EventFilters{StartTime: &windowTestAt, EndTime: &end}, windowTestLimit, 0)
	require.NoError(t, err)

	require.NotEmpty(t, store.recorded())
	for _, filter := range store.recorded() {
		require.NotNil(t, filter.Since)
		require.NotNil(t, filter.Until)
		assert.True(t, windowTestAt.Equal(*filter.Since), "since %s, want %s", filter.Since, windowTestAt)
		assert.True(t, end.Equal(*filter.Until), "until %s, want %s", filter.Until, end)
	}
}

// A stream reads the store in storage order from the overlap behind the
// newest event it read, and leaves the occurrence window to its filters.
func TestEventStream_ReadsTheStoreInStorageOrder(t *testing.T) {
	store := &windowRecorder{}
	wake := streamwake.NewSignal()
	svc := eventsservice.New(NewSystemEventStoreAdapter(store), nil, nil, windowTestPoll, windowTestOverlap, wake, windowTestBatch, logger.NewNop())
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()
	stream, err := svc.Stream(ctx, windowTestTenant, nil)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(store.recorded()) > 0 }, windowTestDelivery, time.Millisecond, "the baseline read")

	store.store(&models.SystemEvent{ID: "stored", TenantID: "7", CreatedAt: windowTestAt, StoredAt: windowTestAt})
	wake.Notify()
	select {
	case event := <-stream:
		assert.Equal(t, "stored", event.ID)
	case <-time.After(windowTestDelivery):
		require.FailNow(t, "the stored event is not delivered")
	}
	store.store(&models.SystemEvent{ID: "late", TenantID: "7", CreatedAt: windowTestAt.Add(-time.Hour), StoredAt: windowTestAt.Add(-windowTestOverlap / 2)})
	wake.Notify()
	select {
	case event := <-stream:
		assert.Equal(t, "late", event.ID, "an event stored late within the overlap reaches the stream")
	case <-time.After(windowTestDelivery):
		require.FailNow(t, "the late event is not delivered")
	}

	last := store.recorded()[len(store.recorded())-1]
	assert.Equal(t, models.EventOrderByStored, last.OrderBy)
	require.NotNil(t, last.StoredSince)
	assert.True(t, windowTestAt.Add(-windowTestOverlap).Equal(*last.StoredSince), "since %s", last.StoredSince)
	assert.Nil(t, last.Since, "the stream sets no occurrence bound of its own")
	assert.Nil(t, last.Until)
}
