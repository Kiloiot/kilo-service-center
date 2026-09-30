package scaciservices

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinkid"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// dlServiceTestLifetime is the downlink lifetime the DL service tests configure.
const dlServiceTestLifetime = 90 * time.Minute

// sequenceQueueIDs hands out a fixed sequence of service center queue ids.
type sequenceQueueIDs struct {
	ids  []int64
	next int
}

func (g *sequenceQueueIDs) NextQueueID(_ context.Context) (int64, error) {
	id := g.ids[g.next%len(g.ids)]
	g.next++
	return id, nil
}

func newTestDLService(t *testing.T, sched scheduler.DownlinkScheduler, store DownlinkStore) scaci.DLService {
	t.Helper()
	return newTestDLServiceWithIDs(t, sched, store, &sequenceQueueIDs{ids: []int64{9_000_001}})
}

func newTestDLServiceWithIDs(t *testing.T, sched scheduler.DownlinkScheduler, store DownlinkStore, ids downlinkid.Generator) scaci.DLService {
	t.Helper()
	return newRecordedDLService(t, sched, store, ids, &enqueueEvents{})
}

func newRecordedDLService(t *testing.T, sched scheduler.DownlinkScheduler, store DownlinkStore, ids downlinkid.Generator, events EnqueueRecorder) scaci.DLService {
	t.Helper()
	allocator, err := downlinkid.New(ids, downlinkid.DefaultAttempts)
	require.NoError(t, err)
	svc, err := NewDLService(sched, store, allocator, dlServiceTestLifetime, events, &mockDLLogger{})
	require.NoError(t, err)
	return svc
}

// enqueueEvents records the downlinks announced as queued.
type enqueueEvents struct {
	enqueued []*storage.DownlinkMessage
	err      error
}

func (e *enqueueEvents) RecordEnqueued(_ context.Context, downlink *storage.DownlinkMessage) error {
	e.enqueued = append(e.enqueued, downlink)
	return e.err
}

// enqueueRecordingStore persists into memory, refusing the service center
// queue ids and Application Center queue ids already taken.
type enqueueRecordingStore struct {
	DownlinkStore
	takenQueueIDs map[int64]bool
	takenACIDs    map[uint64]bool
	attempts      []int64
	lifetimes     []time.Duration
}

func (s *enqueueRecordingStore) EnqueueDownlink(_ context.Context, dl *storage.DownlinkMessage, lifetime time.Duration) (*storage.DownlinkMessage, error) {
	s.attempts = append(s.attempts, dl.QueID)
	s.lifetimes = append(s.lifetimes, lifetime)
	if s.takenQueueIDs[dl.QueID] {
		return nil, fmt.Errorf("enqueue downlink: %w", storage.ErrDownlinkQueueIDTaken)
	}
	if dl.ACQueID != nil && s.takenACIDs[*dl.ACQueID] {
		return nil, fmt.Errorf("enqueue downlink: %w", storage.ErrDuplicateKey)
	}
	stored := *dl
	return &stored, nil
}

func TestNewDLService_RejectsMissingCollaborators(t *testing.T) {
	allocator, err := downlinkid.New(&sequenceQueueIDs{ids: []int64{1}}, downlinkid.DefaultAttempts)
	require.NoError(t, err)
	sched, store, events, log := new(mockDownlinkScheduler), &enqueueRecordingStore{}, &enqueueEvents{}, &mockDLLogger{}
	cases := map[string]struct {
		sched     scheduler.DownlinkScheduler
		store     DownlinkStore
		allocator QueueIDAllocator
		events    EnqueueRecorder
		log       logger.Logger
		want      error
	}{
		"nil scheduler": {store: store, allocator: allocator, events: events, log: log, want: ErrNilDownlinkScheduler},
		"nil store":     {sched: sched, allocator: allocator, events: events, log: log, want: ErrNilDownlinkStore},
		"nil allocator": {sched: sched, store: store, events: events, log: log, want: ErrNilQueueIDAllocator},
		"nil recorder":  {sched: sched, store: store, allocator: allocator, log: log, want: ErrNilEnqueueRecorder},
		"nil logger":    {sched: sched, store: store, allocator: allocator, events: events, want: ErrNilDLServiceLogger},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc, err := NewDLService(tc.sched, tc.store, tc.allocator, dlServiceTestLifetime, tc.events, tc.log)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, svc)
		})
	}
}

func TestNewDLService_RejectsANonPositiveLifetime(t *testing.T) {
	allocator, err := downlinkid.New(&sequenceQueueIDs{ids: []int64{1}}, downlinkid.DefaultAttempts)
	require.NoError(t, err)
	svc, err := NewDLService(new(mockDownlinkScheduler), &enqueueRecordingStore{}, allocator, 0, &enqueueEvents{}, &mockDLLogger{})
	require.ErrorIs(t, err, ErrNonPositiveDownlinkLifetime)
	assert.Nil(t, svc)
}

// TestDLService_EnqueueDownlink_CarriesTheConfiguredLifetime: every ingress
// enqueues through this service, so the configured lifetime bounds every
// downlink's wait for a window (protocol.downlink_expiry.lifetime).
func TestDLService_EnqueueDownlink_CarriesTheConfiguredLifetime(t *testing.T) {
	store := &enqueueRecordingStore{}
	svc := newTestDLService(t, new(mockDownlinkScheduler), store)
	orgID := uuid.New()

	_, err := svc.EnqueueDownlink(testutil.TestContext(), &storage.DownlinkMessage{
		EPEUI: "70B3D59CD00009E7", TenantID: "1", OrganizationID: &orgID, Status: mioty.DLQueueStatusPending,
	})

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{dlServiceTestLifetime}, store.lifetimes)
}

// TestDLService_EnqueueDownlink_AssignsServiceCenterQueueID pins that every
// downlink is persisted under a service center queue id drawn by the
// allocator, drawing again while the id is taken, and that the Application
// Center's id travels beside it untouched.
func TestDLService_EnqueueDownlink_AssignsServiceCenterQueueID(t *testing.T) {
	store := &enqueueRecordingStore{takenQueueIDs: map[int64]bool{101: true}}
	svc := newTestDLServiceWithIDs(t, new(mockDownlinkScheduler), store, &sequenceQueueIDs{ids: []int64{101, 102}})
	acID := uint64(42)
	orgID := uuid.New()

	stored, err := svc.EnqueueDownlink(testutil.TestContext(), &storage.DownlinkMessage{
		EPEUI: "70B3D59CD00009E7", TenantID: "1", OrganizationID: &orgID,
		Status: mioty.DLQueueStatusPending, ACQueID: &acID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(102), stored.QueID)
	require.NotNil(t, stored.ACQueID)
	assert.Equal(t, acID, *stored.ACQueID)
	assert.Equal(t, []int64{101, 102}, store.attempts)
}

// Every ingress queues through EnqueueDownlink, an Application Center's
// dlDataQue, the web API and MQTT alike, so the downlink it stores is
// announced there once, and a refused one is not.
func TestDLService_EnqueueDownlink_AnnouncesTheQueuedDownlink(t *testing.T) {
	events := &enqueueEvents{}
	svc := newRecordedDLService(t, new(mockDownlinkScheduler), &enqueueRecordingStore{takenACIDs: map[uint64]bool{42: true}},
		&sequenceQueueIDs{ids: []int64{301, 302}}, events)
	orgID := uuid.New()

	stored, err := svc.EnqueueDownlink(testutil.TestContext(), &storage.DownlinkMessage{
		EPEUI: "70B3D59CD00009E7", TenantID: "1", OrganizationID: &orgID, Status: mioty.DLQueueStatusPending,
	})
	require.NoError(t, err)
	assert.Equal(t, []*storage.DownlinkMessage{stored}, events.enqueued, "the stored downlink, under its service center queue id")

	duplicate := uint64(42)
	_, err = svc.EnqueueDownlink(testutil.TestContext(), &storage.DownlinkMessage{ACQueID: &duplicate})
	require.ErrorIs(t, err, storage.ErrDuplicateKey)
	assert.Len(t, events.enqueued, 1, "a refused downlink is not announced")
}

// The downlink stays queued when its announcement fails.
func TestDLService_EnqueueDownlink_StaysQueuedWhenItsEventFails(t *testing.T) {
	events := &enqueueEvents{err: errors.New("event store down")}
	svc := newRecordedDLService(t, new(mockDownlinkScheduler), &enqueueRecordingStore{}, &sequenceQueueIDs{ids: []int64{401}}, events)
	orgID := uuid.New()

	stored, err := svc.EnqueueDownlink(testutil.TestContext(), &storage.DownlinkMessage{
		EPEUI: "70B3D59CD00009E7", TenantID: "1", OrganizationID: &orgID, Status: mioty.DLQueueStatusPending,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(401), stored.QueID)
	assert.Len(t, events.enqueued, 1)
}

// TestDLService_EnqueueDownlink_ApplicationQueueIDDuplicateIsNotRetried pins
// that an Application Center id already in flight in its organization surfaces as the
// duplicate it is instead of burning fresh service center ids.
func TestDLService_EnqueueDownlink_ApplicationQueueIDDuplicateIsNotRetried(t *testing.T) {
	store := &enqueueRecordingStore{takenACIDs: map[uint64]bool{42: true}}
	svc := newTestDLServiceWithIDs(t, new(mockDownlinkScheduler), store, &sequenceQueueIDs{ids: []int64{201, 202}})
	acID := uint64(42)

	_, err := svc.EnqueueDownlink(testutil.TestContext(), &storage.DownlinkMessage{ACQueID: &acID})
	require.ErrorIs(t, err, storage.ErrDuplicateKey)
	assert.Len(t, store.attempts, 1)
}
