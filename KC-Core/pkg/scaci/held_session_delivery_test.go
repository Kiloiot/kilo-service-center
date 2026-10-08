package scaci

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/workers/delivery"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Delivery worker settings of the outbox tests: one row per drain.
const (
	outboxTestPoll    = time.Second
	outboxTestBatch   = 1
	outboxTestBackoff = time.Second
	outboxTestMax     = time.Minute
	// outboxTestAttempt is the attempt the claimed row is on.
	outboxTestAttempt = 1
)

// outboxFake is the uplink delivery outbox holding one SCACI row of heldTenant.
type outboxFake struct {
	rows        []models.MessageDeliveryRecord
	delivered   []uuid.UUID
	rescheduled []uuid.UUID
}

func (o *outboxFake) ClaimDue(context.Context, int, time.Duration) ([]models.MessageDeliveryRecord, error) {
	rows := o.rows
	o.rows = nil
	return rows, nil
}

func (o *outboxFake) MarkDelivered(_ context.Context, id uuid.UUID, _ models.DeliveryChannel) error {
	o.delivered = append(o.delivered, id)
	return nil
}

func (o *outboxFake) Reschedule(_ context.Context, id uuid.UUID, _ models.DeliveryChannel, _ time.Duration, _ string) error {
	o.rescheduled = append(o.rescheduled, id)
	return nil
}

func (o *outboxFake) Park(context.Context, uuid.UUID, models.DeliveryChannel, string) error {
	return nil
}

// storedUplinks serves the stored uplink every outbox row points to.
type storedUplinks struct{ uplink *mioty.ULDataMessage }

func (s storedUplinks) GetULDataMessage(context.Context, string, int64) (*mioty.ULDataMessage, error) {
	return s.uplink, nil
}

// noDownlinks holds no acknowledged downlink: these outboxes carry only uplinks.
type noDownlinks struct{}

func (noDownlinks) GetAcknowledgedDownlink(context.Context, int64, int64) (*storage.DownlinkMessage, error) {
	return nil, storage.ErrNotFound
}

type noEvents struct{}

func (noEvents) CreateEvent(context.Context, *models.SystemEvent) error { return nil }

// deliverThroughOutbox runs one drain of the delivery worker over a single
// SCACI row of heldTenant, with the server as the SCACI channel.
func deliverThroughOutbox(t *testing.T, server *Server) (*outboxFake, uuid.UUID) {
	t.Helper()
	id := uuid.New()
	outbox := &outboxFake{rows: []models.MessageDeliveryRecord{{MessageID: id, Channel: models.DeliveryChannelSCACI, OwnerTenantID: heldTenant, Attempts: outboxTestAttempt}}}
	retry, err := delivery.NewRetryPolicy(outboxTestBackoff, outboxTestMax)
	require.NoError(t, err)
	worker, err := delivery.NewWorker(delivery.Dependencies{
		Outbox:    outbox,
		Outcomes:  outbox,
		Messages:  storedUplinks{uplink: broadcastULDataFixture()},
		Downlinks: noDownlinks{},
		Channels:  delivery.Channels{SCACI: server},
		Events:    noEvents{},
		Clock:     clock.SystemClock{},
		Logger:    logger.NewNop(),
	}, delivery.Config{PollInterval: outboxTestPoll, BatchSize: outboxTestBatch, Retry: retry})
	require.NoError(t, err)
	worker.DrainOnce(testutil.TestContext())
	return outbox, id
}

// Recording the uplink for a disconnected session that stays resumable is its
// delivery to that session: the outbox row is done (SCACI §1).
func TestOutbox_UplinkRecordedForHeldSessionIsDelivered(t *testing.T) {
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))

	outbox, id := deliverThroughOutbox(t, server)

	assert.Equal(t, []uuid.UUID{id}, outbox.delivered)
	assert.Equal(t, []heldRecord{{sessionID: heldSessionID, command: CmdULData}}, holder.recorded())
}

// A held uplink that cannot be recorded is a failed attempt, retried like a
// failed send, so the resume does not miss it.
func TestOutbox_UnrecordableHeldUplinkIsRetried(t *testing.T) {
	holder := newHolderFake()
	holder.failWith = errHeldTestStore
	server := newHeldTestServer(holder)
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))

	outbox, id := deliverThroughOutbox(t, server)

	assert.Empty(t, outbox.delivered, "a held session that did not get the uplink leaves the row pending")
	assert.Equal(t, []uuid.UUID{id}, outbox.rescheduled)
}

// Two Application Centers and two attempts through the outbox: A is
// connected, B is held and its first record fails. The retry records the
// uplink for B only, A keeps the one operation it received, and B's resume
// reissues its one operation under its opId (SCACI §1, §3.2).
func TestOutbox_RetryRecordsTheUplinkOnceForEverySession(t *testing.T) {
	server := newHeldTestServer(newHolderFake())
	ledger := newOperationLedger()
	server.operationRecorder, server.operationRepo = ledger, ledger
	connA := &mockConn{}
	server.registry.sessions[connA] = &Session{ID: heldOtherACID, TenantID: heldTenant, AcEui: heldOtherAcEui, State: StateActive}
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	ledger.fail(heldSessionID, errHeldTestStore)

	first, id := deliverThroughOutbox(t, server)
	ledger.heal(heldSessionID)
	second, _ := deliverThroughOutbox(t, server)

	assert.Equal(t, []uuid.UUID{id}, first.rescheduled, "B did not get the uplink: the row stays pending")
	assert.Len(t, second.delivered, 1, "the retry reaches every session")
	opIDsA := ledger.uplinkOpIDs(heldOtherACID, broadcastULDataTestMessageID)
	opIDsB := ledger.uplinkOpIDs(heldSessionID, broadcastULDataTestMessageID)
	require.Len(t, opIDsA, 1, "no second operation for A")
	require.Len(t, opIDsB, 1)
	assert.Equal(t, opIDsA, writtenOpIDs(t, connA.written), "A received the uplink once")

	resumed := &mockConn{}
	server.replayPendingOperations(resumed, heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	assert.Equal(t, opIDsB, writtenOpIDs(t, resumed.written), "B's resume reissues its one operation")
}
