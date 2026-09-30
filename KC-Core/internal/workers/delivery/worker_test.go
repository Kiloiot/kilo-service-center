package delivery

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	workerTestTenantID  = int64(42)
	workerTestBatchSize = 10
	workerTestPoll      = time.Millisecond
	workerTestRunWait   = 200 * time.Millisecond
	workerTestBase      = 2 * time.Second
	workerTestMax       = 5 * time.Minute
	// workerTestLongOutage is far beyond the attempts a capped schedule needs
	// to reach its maximum.
	workerTestLongOutage = 500
)

var (
	errWorkerTestChannelDown = errors.New("channel down")
	workerTestNow            = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
)

type parkCall struct {
	id      uuid.UUID
	channel models.DeliveryChannel
	reason  string
}

type rescheduleCall struct {
	id     uuid.UUID
	delay  time.Duration
	reason string
}

type fakeOutbox struct {
	mu          sync.Mutex
	rows        []models.MessageDeliveryRecord
	claimErr    error
	leases      []time.Duration
	delivered   []uuid.UUID
	rescheduled []rescheduleCall
	parked      []parkCall
	claims      int
}

func (o *fakeOutbox) ClaimDue(_ context.Context, _ int, lease time.Duration) ([]models.MessageDeliveryRecord, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.claims++
	o.leases = append(o.leases, lease)
	if o.claimErr != nil {
		return nil, o.claimErr
	}
	rows := o.rows
	o.rows = nil
	return rows, nil
}

func (o *fakeOutbox) MarkDelivered(_ context.Context, id uuid.UUID, _ models.DeliveryChannel) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.delivered = append(o.delivered, id)
	return nil
}

func (o *fakeOutbox) Reschedule(_ context.Context, id uuid.UUID, _ models.DeliveryChannel, delay time.Duration, reason string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.rescheduled = append(o.rescheduled, rescheduleCall{id: id, delay: delay, reason: reason})
	return nil
}

func (o *fakeOutbox) Park(_ context.Context, id uuid.UUID, channel models.DeliveryChannel, reason string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.parked = append(o.parked, parkCall{id: id, channel: channel, reason: reason})
	return nil
}

type fakeMessages struct {
	msgs map[string]*mioty.ULDataMessage
	err  error
}

func (m *fakeMessages) GetULDataMessage(_ context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	msg, ok := m.msgs[id]
	if !ok || msg.TenantID != tenantID {
		return nil, fmt.Errorf("%s: %w", id, storage.ErrNotFound)
	}
	return msg, nil
}

type fakeSCACI struct {
	err     error
	tenants []int64
	msgs    []*mioty.ULDataMessage
}

func (s *fakeSCACI) BroadcastULData(_ context.Context, tenantID int64, data *mioty.ULDataMessage) error {
	s.tenants = append(s.tenants, tenantID)
	s.msgs = append(s.msgs, data)
	return s.err
}

type mqttCall struct {
	orgUUID string
	epEUI   uint64
	bsEUI   uint64
	payload []byte
}

type fakeMQTT struct {
	err   error
	calls []mqttCall
}

func (m *fakeMQTT) PublishUplink(_ context.Context, orgUUID string, msg *mioty.ULDataMessage) error {
	m.calls = append(m.calls, mqttCall{orgUUID: orgUUID, epEUI: msg.EpEui, bsEUI: msg.BsEui, payload: msg.UserData})
	return m.err
}

// ackCall is one acknowledgement handed to the MQTT publisher.
type ackCall struct {
	orgUUID, ref string
	epEUI, queID uint64
	packetCnt    uint32
}

type fakeDownlinkAcks struct {
	err   error
	calls []ackCall
}

func (a *fakeDownlinkAcks) PublishDownlinkAcknowledged(_ context.Context, orgUUID, ref string, epEUI, queID uint64, packetCnt uint32) error {
	a.calls = append(a.calls, ackCall{orgUUID: orgUUID, ref: ref, epEUI: epEUI, queID: queID, packetCnt: packetCnt})
	return a.err
}

// fakeDownlinks holds the acknowledged downlinks by id.
type fakeDownlinks struct {
	rows map[int64]*storage.DownlinkMessage
}

func (d *fakeDownlinks) GetAcknowledgedDownlink(_ context.Context, tenantID, downlinkID int64) (*storage.DownlinkMessage, error) {
	row, ok := d.rows[downlinkID]
	if !ok || row.TenantID != strconv.FormatInt(tenantID, 10) {
		return nil, fmt.Errorf("%d: %w", downlinkID, storage.ErrNotFound)
	}
	return row, nil
}

type fakeEvents struct {
	events []*models.SystemEvent
}

func (e *fakeEvents) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	e.events = append(e.events, event)
	return nil
}

func newStoredMessage(id uuid.UUID, orgUUID string) *mioty.ULDataMessage {
	msg := &mioty.ULDataMessage{
		ID:        id.String(),
		TenantID:  workerTestTenantID,
		EpEui:     0x70B3D5677011150A,
		BsEui:     0x70B3D59CD00009E6,
		PacketCnt: 7,
		UserData:  []byte{0x42},
	}
	if orgUUID != "" {
		msg.OrgUUID = &orgUUID
	}
	return msg
}

func newRow(id uuid.UUID, channel models.DeliveryChannel, attempts int) models.MessageDeliveryRecord {
	return models.MessageDeliveryRecord{
		MessageID:     id,
		Channel:       channel,
		OwnerTenantID: workerTestTenantID,
		Status:        models.DeliveryStatusPending,
		Attempts:      attempts,
	}
}

func newTestRetryPolicy(t *testing.T) RetryPolicy {
	t.Helper()
	policy, err := NewRetryPolicy(workerTestBase, workerTestMax)
	require.NoError(t, err)
	return policy
}

type workerFixture struct {
	outbox    *fakeOutbox
	messages  *fakeMessages
	downlinks *fakeDownlinks
	scaci     *fakeSCACI
	mqtt      *fakeMQTT
	acks      *fakeDownlinkAcks
	events    *fakeEvents
	deps      Dependencies
	cfg       Config
	worker    *Worker
}

func newWorkerFixture(t *testing.T, rows ...models.MessageDeliveryRecord) *workerFixture {
	t.Helper()
	f := &workerFixture{
		outbox:    &fakeOutbox{rows: rows},
		messages:  &fakeMessages{msgs: map[string]*mioty.ULDataMessage{}},
		downlinks: &fakeDownlinks{rows: map[int64]*storage.DownlinkMessage{}},
		scaci:     &fakeSCACI{},
		mqtt:      &fakeMQTT{},
		acks:      &fakeDownlinkAcks{},
		events:    &fakeEvents{},
	}
	f.deps = Dependencies{
		Outbox:    f.outbox,
		Outcomes:  f.outbox,
		Messages:  f.messages,
		Downlinks: f.downlinks,
		Channels:  Channels{SCACI: f.scaci, MQTT: f.mqtt, DownlinkAcks: f.acks},
		Events:    f.events,
		Clock:     testutil.NewFakeClock(workerTestNow),
		Logger:    logger.NewNop(),
	}
	f.cfg = Config{PollInterval: workerTestPoll, BatchSize: workerTestBatchSize, Retry: newTestRetryPolicy(t)}
	f.rebuild(t)
	return f
}

// rebuild constructs the worker from the fixture's current dependencies.
func (f *workerFixture) rebuild(t *testing.T) {
	t.Helper()
	worker, err := NewWorker(f.deps, f.cfg)
	require.NoError(t, err)
	f.worker = worker
}

func TestWorker_DeliversSCACIAndMarksRow(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannelSCACI, 1))
	f.messages.msgs[id.String()] = newStoredMessage(id, "")

	f.worker.DrainOnce(testutil.TestContext())

	require.Equal(t, []int64{workerTestTenantID}, f.scaci.tenants)
	assert.Equal(t, id.String(), f.scaci.msgs[0].ID)
	assert.Equal(t, []uuid.UUID{id}, f.outbox.delivered)
	assert.Empty(t, f.outbox.parked)
	assert.Empty(t, f.outbox.rescheduled)
	assert.Empty(t, f.mqtt.calls)
	assert.Equal(t, []time.Duration{workerTestMax}, f.outbox.leases, "a claimed row stays hidden for the longest retry delay")
}

func TestWorker_DeliversMQTTWithOwnerOrg(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	org := uuid.New().String()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannelMQTT, 1))
	f.messages.msgs[id.String()] = newStoredMessage(id, org)

	f.worker.DrainOnce(testutil.TestContext())

	require.Len(t, f.mqtt.calls, 1)
	assert.Equal(t, org, f.mqtt.calls[0].orgUUID)
	assert.Equal(t, uint64(0x70B3D5677011150A), f.mqtt.calls[0].epEUI)
	assert.Equal(t, uint64(0x70B3D59CD00009E6), f.mqtt.calls[0].bsEUI)
	assert.Equal(t, []byte{0x42}, f.mqtt.calls[0].payload)
	assert.Equal(t, []uuid.UUID{id}, f.outbox.delivered)
	assert.Empty(t, f.scaci.tenants)
}

// The worker hands the stored uplink to SCACI as it is: the stored duplicate
// flag (receptions merged into the message) is not rewritten for the wire.
func TestWorker_SCACIDeliveryLeavesTheStoredMessageUnchanged(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannelSCACI, 1))
	stored := newStoredMessage(id, "")
	merged := true
	stored.Duplicate = &merged
	stored.PacketCntReused = false
	f.messages.msgs[id.String()] = stored

	f.worker.DrainOnce(testutil.TestContext())

	require.Len(t, f.scaci.msgs, 1)
	assert.Same(t, stored, f.scaci.msgs[0])
	require.NotNil(t, stored.Duplicate)
	assert.True(t, *stored.Duplicate, "the stored message keeps its own duplicate flag")
	assert.Equal(t, []uuid.UUID{id}, f.outbox.delivered)
}

// The acknowledged downlink of the fixture: its row id, queue id, ref, window
// and the organization that queued it.
const (
	workerTestDownlinkID = int64(77)
	workerTestQueID      = int64(918001)
	workerTestRef        = "order-17"
	workerTestWindow     = int64(41)
)

var workerTestQueuingOrg = uuid.MustParse("5c1b4a0e-0000-4000-8000-000000000918")

// newAckRow is the acknowledgement row of the fixture's downlink.
func newAckRow(id uuid.UUID) models.MessageDeliveryRecord {
	row := newRow(id, models.DeliveryChannelMQTTDownlinkAck, 1)
	downlinkID := workerTestDownlinkID
	row.AcknowledgedDownlinkID = &downlinkID
	return row
}

// newAcknowledgedDownlink is the fixture's downlink as the queue stores it.
func newAcknowledgedDownlink() *storage.DownlinkMessage {
	org := workerTestQueuingOrg
	return &storage.DownlinkMessage{
		ID: workerTestDownlinkID, QueID: workerTestQueID, EPEUI: "70B3D5677011150A", TenantID: "42",
		OrganizationID: &org, Ref: workerTestRef, TransmissionPacketCnt: workerTestWindow,
	}
}

// An acknowledgement row publishes the acknowledged downlink to the
// organization that queued it, with its ref and window, and never loads the
// uplink that carried it: that uplink may belong to another organization.
func TestWorker_PublishesTheAcknowledgementToTheQueuingOrganization(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newAckRow(id))
	f.downlinks.rows[workerTestDownlinkID] = newAcknowledgedDownlink()
	f.messages.err = errors.New("the uplink is not read")

	f.worker.DrainOnce(testutil.TestContext())

	assert.Equal(t, []ackCall{{
		orgUUID: workerTestQueuingOrg.String(), ref: workerTestRef, epEUI: 0x70B3D5677011150A,
		queID: uint64(workerTestQueID), packetCnt: uint32(workerTestWindow),
	}}, f.acks.calls)
	assert.Equal(t, []uuid.UUID{id}, f.outbox.delivered)
	assert.Empty(t, f.mqtt.calls, "no uplink is published for an acknowledgement")
	assert.Empty(t, f.scaci.tenants, "no Application Center is told of an acknowledgement")
}

// A broker outage leaves the acknowledgement pending for its next attempt.
func TestWorker_AcknowledgementOutageIsRescheduled(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newAckRow(id))
	f.downlinks.rows[workerTestDownlinkID] = newAcknowledgedDownlink()
	f.acks.err = errWorkerTestChannelDown

	f.worker.DrainOnce(testutil.TestContext())

	assert.Len(t, f.acks.calls, 1)
	assert.Empty(t, f.outbox.delivered)
	assert.Empty(t, f.outbox.parked)
	require.Len(t, f.outbox.rescheduled, 1)
	assert.Contains(t, f.outbox.rescheduled[0].reason, errWorkerTestChannelDown.Error())
}

func TestWorker_TransientFailureIsRescheduled(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannelSCACI, 1))
	f.messages.msgs[id.String()] = newStoredMessage(id, "")
	f.scaci.err = errWorkerTestChannelDown

	f.worker.DrainOnce(testutil.TestContext())

	assert.Empty(t, f.outbox.delivered, "a failed attempt must leave the row pending")
	assert.Empty(t, f.outbox.parked)
	assert.Empty(t, f.events.events)
	require.Len(t, f.outbox.rescheduled, 1)
	assert.Equal(t, workerTestBase, f.outbox.rescheduled[0].delay, "the first retry waits the base backoff")
	assert.Contains(t, f.outbox.rescheduled[0].reason, errWorkerTestChannelDown.Error())
}

func TestWorker_TransientFailureIsNeverParked(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannelSCACI, workerTestLongOutage))
	f.messages.msgs[id.String()] = newStoredMessage(id, "")
	f.scaci.err = errWorkerTestChannelDown

	f.worker.DrainOnce(testutil.TestContext())

	assert.Empty(t, f.outbox.parked, "a channel outage is retried however long it lasts")
	assert.Empty(t, f.events.events)
	require.Len(t, f.outbox.rescheduled, 1)
	assert.Equal(t, workerTestMax, f.outbox.rescheduled[0].delay, "a long outage is retried at the capped backoff")
}

func TestWorker_LoadFailureRetries(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannelSCACI, 1))
	f.messages.err = errors.New("db down")

	f.worker.DrainOnce(testutil.TestContext())

	assert.Empty(t, f.scaci.tenants)
	assert.Empty(t, f.outbox.delivered)
	assert.Empty(t, f.outbox.parked)
	assert.Len(t, f.outbox.rescheduled, 1, "a database outage is transient")
}

func TestWorker_PermanentFailuresParkAtOnce(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		channel models.DeliveryChannel
		prepare func(f *workerFixture, id uuid.UUID)
		cause   error
	}{
		{"unknown channel", models.DeliveryChannel("smoke"), func(f *workerFixture, id uuid.UUID) {
			f.messages.msgs[id.String()] = newStoredMessage(id, "")
		}, errUnknownChannel},
		{"message missing", models.DeliveryChannelSCACI, func(*workerFixture, uuid.UUID) {}, storage.ErrNotFound},
		{"MQTT without owner organization", models.DeliveryChannelMQTT, func(f *workerFixture, id uuid.UUID) {
			f.messages.msgs[id.String()] = newStoredMessage(id, "")
		}, errChannelNotConfigured},
		{"channel not configured", models.DeliveryChannelSCACI, func(f *workerFixture, id uuid.UUID) {
			f.messages.msgs[id.String()] = newStoredMessage(id, "")
			f.deps.Channels = Channels{}
		}, errChannelNotConfigured},
		{"acknowledgement without a publisher", models.DeliveryChannelMQTTDownlinkAck, func(f *workerFixture, _ uuid.UUID) {
			f.downlinks.rows[workerTestDownlinkID] = newAcknowledgedDownlink()
			f.deps.Channels.DownlinkAcks = nil
		}, errChannelNotConfigured},
		{"acknowledged downlink missing", models.DeliveryChannelMQTTDownlinkAck, func(*workerFixture, uuid.UUID) {}, storage.ErrNotFound},
		{"acknowledged downlink of another tenant", models.DeliveryChannelMQTTDownlinkAck, func(f *workerFixture, _ uuid.UUID) {
			foreign := newAcknowledgedDownlink()
			foreign.TenantID = "43"
			f.downlinks.rows[workerTestDownlinkID] = foreign
		}, storage.ErrNotFound},
		{"acknowledged downlink without organization", models.DeliveryChannelMQTTDownlinkAck, func(f *workerFixture, _ uuid.UUID) {
			orphan := newAcknowledgedDownlink()
			orphan.OrganizationID = nil
			f.downlinks.rows[workerTestDownlinkID] = orphan
		}, errChannelNotConfigured},
		{"acknowledged downlink without a wire queue id", models.DeliveryChannelMQTTDownlinkAck, func(f *workerFixture, _ uuid.UUID) {
			unqueued := newAcknowledgedDownlink()
			unqueued.QueID = 0
			f.downlinks.rows[workerTestDownlinkID] = unqueued
		}, errUnpublishableAck},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id := uuid.New()
			row := newRow(id, tc.channel, 1)
			if tc.channel == models.DeliveryChannelMQTTDownlinkAck {
				row = newAckRow(id)
			}
			f := newWorkerFixture(t, row)
			tc.prepare(f, id)
			f.rebuild(t)

			f.worker.DrainOnce(testutil.TestContext())

			require.Len(t, f.outbox.parked, 1, "no retry can fix a permanent failure")
			assert.Equal(t, id, f.outbox.parked[0].id)
			assert.Contains(t, f.outbox.parked[0].reason, tc.cause.Error())
			assert.Empty(t, f.outbox.rescheduled)
			assert.Empty(t, f.outbox.delivered)
		})
	}
}

func TestWorker_ParkRecordsOperatorEvent(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannel("smoke"), 1))
	f.messages.msgs[id.String()] = newStoredMessage(id, "")

	f.worker.DrainOnce(testutil.TestContext())

	require.Len(t, f.events.events, 1)
	event := f.events.events[0]
	assert.Equal(t, eventTypeDeliveryParked, event.EventType)
	assert.Equal(t, models.EventCategoryMessage, event.Category)
	assert.Equal(t, "42", event.TenantID)
	assert.Equal(t, id.String(), event.SourceName)
	assert.Contains(t, event.Description, errUnknownChannel.Error())
	assert.Equal(t, workerTestNow, event.CreatedAt, "the event is stamped by the injected clock")
	assert.Equal(t, workerTestNow, event.UpdatedAt)
}

func TestWorker_ClaimFailureDeliversNothing(t *testing.T) {
	t.Parallel()
	f := newWorkerFixture(t)
	f.outbox.claimErr = errors.New("db down")

	f.worker.DrainOnce(testutil.TestContext())

	assert.Empty(t, f.scaci.tenants)
	assert.Empty(t, f.outbox.delivered)
}

func TestWorker_RunStopsOnCancel(t *testing.T) {
	t.Parallel()
	f := newWorkerFixture(t)
	stop := f.worker.Start(testutil.TestContext())
	time.Sleep(workerTestPoll * 5)
	stop()
	f.outbox.mu.Lock()
	defer f.outbox.mu.Unlock()
	assert.GreaterOrEqual(t, f.outbox.claims, 1, "Run must poll the outbox")
}

// blockingSCACI holds every broadcast until released.
type blockingSCACI struct {
	entered chan struct{}
	release chan struct{}
}

func (b *blockingSCACI) BroadcastULData(context.Context, int64, *mioty.ULDataMessage) error {
	b.entered <- struct{}{}
	<-b.release
	return nil
}

func TestWorker_StopWaitsForTheBatchInFlight(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	f := newWorkerFixture(t, newRow(id, models.DeliveryChannelSCACI, 1))
	f.messages.msgs[id.String()] = newStoredMessage(id, "")
	scaci := &blockingSCACI{entered: make(chan struct{}), release: make(chan struct{})}
	f.deps.Channels.SCACI = scaci
	f.rebuild(t)

	stop := f.worker.Start(testutil.TestContext())
	<-scaci.entered
	stopped := make(chan struct{})
	go func() {
		stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		close(scaci.release)
		t.Fatal("stop returned while a delivery was in flight")
	case <-time.After(workerTestRunWait):
	}
	close(scaci.release)
	<-stopped

	f.outbox.mu.Lock()
	defer f.outbox.mu.Unlock()
	assert.Equal(t, []uuid.UUID{id}, f.outbox.delivered, "the batch in flight finishes and records its outcome")
}

func TestNewWorker_RejectsMissingCollaboratorsAndConfig(t *testing.T) {
	t.Parallel()
	f := newWorkerFixture(t)
	withoutClock := f.deps
	withoutClock.Clock = nil
	_, err := NewWorker(withoutClock, f.cfg)
	assert.ErrorIs(t, err, errMissingDependency)

	withoutEvents := f.deps
	withoutEvents.Events = nil
	_, err = NewWorker(withoutEvents, f.cfg)
	assert.ErrorIs(t, err, errMissingDependency)

	withoutDownlinks := f.deps
	withoutDownlinks.Downlinks = nil
	_, err = NewWorker(withoutDownlinks, f.cfg)
	assert.ErrorIs(t, err, errMissingDependency)

	_, err = NewWorker(f.deps, Config{PollInterval: workerTestPoll, BatchSize: workerTestBatchSize})
	assert.ErrorIs(t, err, errInvalidConfig, "a worker without a retry policy cannot lease rows")
}

func TestRetryPolicy_DoublesUpToTheCap(t *testing.T) {
	t.Parallel()
	policy := newTestRetryPolicy(t)
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, workerTestBase},
		{2, 2 * workerTestBase},
		{3, 4 * workerTestBase},
		{8, 128 * workerTestBase},
		{9, workerTestMax},
		{workerTestLongOutage, workerTestMax},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, policy.Delay(tc.attempt), "attempt %d", tc.attempt)
	}
	assert.Equal(t, workerTestMax, policy.Lease())
}

func TestRetryPolicy_ClassifiesFailures(t *testing.T) {
	t.Parallel()
	policy := newTestRetryPolicy(t)
	for _, permanent := range []error{errUnknownChannel, errChannelNotConfigured, errUnpublishableAck,
		fmt.Errorf("%w: %w", errLoadMessage, storage.ErrNotFound), fmt.Errorf("%w: %w", errLoadDownlink, storage.ErrNotFound)} {
		assert.True(t, policy.Permanent(permanent), "%v is permanent", permanent)
	}
	for _, transient := range []error{errWorkerTestChannelDown, fmt.Errorf("%w: %w", errLoadMessage, errWorkerTestChannelDown)} {
		assert.False(t, policy.Permanent(transient), "%v is transient", transient)
	}
}

func TestNewRetryPolicy_RejectsInvertedBounds(t *testing.T) {
	t.Parallel()
	_, err := NewRetryPolicy(workerTestMax, workerTestBase)
	assert.ErrorIs(t, err, errInvalidRetryPolicy)
	_, err = NewRetryPolicy(0, workerTestMax)
	assert.ErrorIs(t, err, errInvalidRetryPolicy)
}
