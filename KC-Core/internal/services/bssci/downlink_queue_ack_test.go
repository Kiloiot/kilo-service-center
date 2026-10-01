package bssciservices

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

var errHolderStoreDown = errors.New("downlink queue down")

type holderWrite struct {
	queID    uint64
	tenantID int64
	bsEUI    uint64
}

type queuedConfirmation struct {
	queID    uint64
	tenantID int64
	bsEUI    uint64
	txTime   int64
	orgID    *uuid.UUID
}

// holderRecorder records both writes of a queue acknowledgement and fails
// the holder write when told to.
type holderRecorder struct {
	holders   []holderWrite
	confirms  []queuedConfirmation
	holderErr error
}

func (r *holderRecorder) UpdateDownlinkBaseStation(_ context.Context, queID uint64, tenantID int64, bsEUI uint64) error {
	r.holders = append(r.holders, holderWrite{queID: queID, tenantID: tenantID, bsEUI: bsEUI})
	return r.holderErr
}

func (r *holderRecorder) MarkReservedAsQueued(_ context.Context, queID uint64, tenantID int64, bsEUI uint64, txTime int64, _ *uint32, orgID *uuid.UUID) error {
	r.confirms = append(r.confirms, queuedConfirmation{queID: queID, tenantID: tenantID, bsEUI: bsEUI, txTime: txTime, orgID: orgID})
	return nil
}

func newQueueAckService(t *testing.T, holders DownlinkHolderWriter) bssci.DownlinkService {
	t.Helper()
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: NewTenantResolver(nil), Outcomes: &mockMIOTYDownlinksForDispatch{}, Holders: holders,
		Results: newReporterFixture(t).reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	}, newRevokeAnswers(t, logger.NewNop(), NewTenantResolver(nil), &mockMIOTYDownlinksForDispatch{}, newReporterFixture(t).reporter))
	require.NoError(t, err)
	return svc
}

// TestProcessQueueAck_RecordsTheHolderAndConfirmsTheRow pins BSSCI §3.12: the
// station that answered dlDataQueRsp holds the owner's downlink, and the
// reserved row is confirmed queued under the organization it was enqueued in.
func TestProcessQueueAck_RecordsTheHolderAndConfirmsTheRow(t *testing.T) {
	holders := &holderRecorder{}
	org := uuid.New()

	require.NoError(t, newQueueAckService(t, holders).ProcessQueueAck(testutil.TestContext(), reporterStation(),
		bssci.QueueAcknowledgement{QueueID: reporterQueueID, OwnerTenant: "3", OrganizationID: &org}))

	assert.Equal(t, []holderWrite{{queID: uint64(reporterQueueID), tenantID: 3, bsEUI: reporterStationEUI}}, holders.holders)
	assert.Equal(t, []queuedConfirmation{{queID: uint64(reporterQueueID), tenantID: 3, bsEUI: reporterStationEUI,
		txTime: dispatchTestNow.UnixNano(), orgID: &org}}, holders.confirms)
}

// TestProcessQueueAck_ConfirmsTheRowWhenTheHolderWriteFails: the two writes
// are independent, so a failed holder write is reported without skipping the
// confirmation.
func TestProcessQueueAck_ConfirmsTheRowWhenTheHolderWriteFails(t *testing.T) {
	holders := &holderRecorder{holderErr: errHolderStoreDown}

	err := newQueueAckService(t, holders).ProcessQueueAck(testutil.TestContext(), reporterStation(),
		bssci.QueueAcknowledgement{QueueID: reporterQueueID, OwnerTenant: "3"})

	require.ErrorIs(t, err, errRecordDownlinkHolder)
	require.ErrorIs(t, err, errHolderStoreDown)
	assert.Len(t, holders.confirms, 1)
}

func TestProcessQueueAck_RejectsAnAcknowledgementNamingNoOwnedDownlink(t *testing.T) {
	for name, ack := range map[string]bssci.QueueAcknowledgement{
		"no queue id":   {QueueID: 0, OwnerTenant: "3"},
		"invalid owner": {QueueID: reporterQueueID, OwnerTenant: "tenant-3"},
	} {
		t.Run(name, func(t *testing.T) {
			holders := &holderRecorder{}

			require.Error(t, newQueueAckService(t, holders).ProcessQueueAck(testutil.TestContext(), reporterStation(), ack))

			assert.Empty(t, holders.holders)
			assert.Empty(t, holders.confirms)
		})
	}
}
