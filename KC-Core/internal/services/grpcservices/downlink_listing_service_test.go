package grpcservices

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

const downlinkTenant = int64(1)

var (
	downlinkEndpoint   = [8]byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x05}
	endpointRegistered = time.Date(2026, 9, 28, 12, 47, 16, 0, time.UTC)
)

// recordingQueue records the filters the listings run with.
type recordingQueue struct {
	DownlinkResultsStore
	queueFilter  *storage.DownlinkQueueFilter
	resultFilter *storage.DownlinkResultFilter
}

func (q *recordingQueue) CountTenantQueue(_ context.Context, _ int64, filter storage.DownlinkQueueFilter) (int64, error) {
	q.queueFilter = &filter
	return 0, nil
}

func (q *recordingQueue) ListTenantQueue(_ context.Context, _ int64, filter storage.DownlinkQueueFilter, _, _ int) ([]*storage.DownlinkMessage, error) {
	q.queueFilter = &filter
	return nil, nil
}

func (q *recordingQueue) GetDownlinkResults(_ context.Context, _ int64, _ *uuid.UUID, filter storage.DownlinkResultFilter, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	q.resultFilter = &filter
	return nil, 0, nil
}

// registrations answers one registration time, or none.
type registrations struct {
	at    time.Time
	found bool
}

func (r registrations) Start(context.Context, int64, []byte, *time.Time) (*time.Time, bool, error) {
	if !r.found {
		return nil, false, nil
	}
	return &r.at, true, nil
}

// An endpoint's downlink queue and results start at its registration, so a
// deleted registration's downlinks stay with it.
func TestDownlinkListings_OfAnEndpointStartAtItsRegistration(t *testing.T) {
	queue := &recordingQueue{}
	svc := NewDownlinkListingService(queue, queue, registrations{at: endpointRegistered, found: true})
	eui := downlinkEndpoint

	_, _, err := svc.ListDownlinkQueue(testutil.TestContext(), downlinkTenant, storage.DownlinkQueueFilter{EpEUI: &eui}, 10, 0)
	require.NoError(t, err)
	_, _, err = svc.GetDownlinkResults(testutil.TestContext(), downlinkTenant, nil, storage.DownlinkResultFilter{EpEUI: eui[:]}, 10, 0)
	require.NoError(t, err)

	require.NotNil(t, queue.queueFilter.QueuedFrom)
	assert.Equal(t, endpointRegistered, *queue.queueFilter.QueuedFrom)
	require.NotNil(t, queue.resultFilter.QueuedFrom)
	assert.Equal(t, endpointRegistered, *queue.resultFilter.QueuedFrom)
}

// An EUI the tenant has not registered lists no downlinks at all.
func TestDownlinkListings_OfAnUnregisteredEndpointAreEmpty(t *testing.T) {
	queue := &recordingQueue{}
	svc := NewDownlinkListingService(queue, queue, registrations{})
	eui := downlinkEndpoint

	entries, total, err := svc.ListDownlinkQueue(testutil.TestContext(), downlinkTenant, storage.DownlinkQueueFilter{EpEUI: &eui}, 10, 0)

	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.Zero(t, total)
	assert.Nil(t, queue.queueFilter, "the queue is not read for an endpoint the tenant does not have")
}
