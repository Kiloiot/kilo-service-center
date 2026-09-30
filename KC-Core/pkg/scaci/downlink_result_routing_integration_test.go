package scaci_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// The downlinks of the routing scenario: both organizations' Application
// Centers assign the same queue id, and the service center gives each
// downlink its own.
const (
	routingEpEui      = uint64(0x70B3D5677011150B)
	routingACQueID    = uint64(5)
	routingOwnerFirst = int64(9001)
	routingClaimant   = int64(9002)
	routingOwnerLater = int64(9003)
)

// synchronousWork runs background work inline, so a delivery has happened
// when the reporter returns.
type synchronousWork struct{}

func (synchronousWork) Go(ctx context.Context, fn func(context.Context)) { fn(ctx) }

// noResultEvents records no downlink result events.
type noResultEvents struct{}

func (noResultEvents) RecordDLResult(context.Context, string, *bssci.Session, *mioty.DLDataResult) error {
	return nil
}

func (noResultEvents) RecordQueueExpiry(context.Context, *storage.DownlinkMessage) error { return nil }

func (noResultEvents) RecordDownlinkAcknowledged(context.Context, *storage.DownlinkMessage, uint32) error {
	return nil
}

// storesDownlinkAs has the harness store the organization's next downlink
// under the service center queue id queID.
func (h *organizationHarness) storesDownlinkAs(orgID uuid.UUID, queID int64) {
	h.downlinks.On("EnqueueDownlink", mock.Anything, mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
		return dl.OrganizationID != nil && *dl.OrganizationID == orgID
	})).Return(&storage.DownlinkMessage{QueID: queID}, nil).Once()
}

// queueDownlink runs a dlDataQue of the routing endpoint under acQueID
// (SCACI §3.10).
func (c *acClient) queueDownlink(opID int64, acQueID uint64) {
	c.t.Helper()
	c.send(scaci.DLDataQueue{
		BaseMessage: mioty.BaseMessage{CommandType: scaci.CmdDLDataQueue, OpId: opID},
		EpEui:       routingEpEui, QueId: acQueID, UserData: mioty.DownlinkUserData{{0x01}},
	})
	c.expect(scaci.CmdDLDataQueueResponse)
	c.send(scaci.DLDataQueueComplete{BaseMessage: mioty.BaseMessage{CommandType: scaci.CmdDLDataQueueComplete, OpId: opID}})
}

// expiredDownlink is the organization's downlink the service center stored
// under queID, queued with routingACQueID by the Application Center acEui.
func expiredDownlink(orgID uuid.UUID, queID int64, acEui uint64) *storage.DownlinkMessage {
	acQueID := routingACQueID
	return &storage.DownlinkMessage{
		TenantID: strconv.FormatInt(harnessTenant, 10), OrganizationID: &orgID,
		QueID: queID, ACQueID: &acQueID, ACEUI: &acEui, EPEUI: mioty.FormatEUI64(routingEpEui),
	}
}

// The result of a downlink reaches the Application Center that queued it and
// no other, even when Application Centers of two organizations of the tenant
// reuse the same queue id over time, and when they share the acEui (SCACI
// §3.10.1, §3.12.1).
func TestSCACIServer_DownlinkResultsReachOnlyTheQueuingApplicationCenter(t *testing.T) {
	for name, claimantAcEui := range map[string]uint64{"another acEui": harnessAcEui + 1, "the same acEui": harnessAcEui} {
		t.Run(name, func(t *testing.T) { assertResultsReachOnlyTheirQueuer(t, claimantAcEui) })
	}
}

func assertResultsReachOnlyTheirQueuer(t *testing.T, claimantAcEui uint64) {
	h := newOrganizationHarness(t)
	h.endpoints.On("GetByEUI", mock.Anything, mock.Anything, mock.Anything).Return(&models.EndPoint{Bidi: true}, "")
	h.downlinks.On("QueueDownlink", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(scaci.DownlinkQueueOutcome{Deferred: true}, "")
	reporter, err := bssciservices.NewDownlinkResultReporter(h.server, bssciservices.DownlinkResultsWithoutMQTT{},
		noResultEvents{}, synchronousWork{}, logger.NewNop())
	require.NoError(t, err)
	ctx := testutil.TestContext()
	owner := h.dialAs(t, h.ownerCert, harnessAcEui)
	owner.connect(harnessUUID(harnessUUIDFirst))
	claimant := h.dialAs(t, h.claimCert, claimantAcEui)
	claimant.connect(harnessUUID(harnessUUIDSecond))

	h.storesDownlinkAs(h.owner, routingOwnerFirst)
	owner.queueDownlink(harnessFirstAcOpID, routingACQueID)
	h.storesDownlinkAs(h.claimant, routingClaimant)
	claimant.queueDownlink(harnessFirstAcOpID, routingACQueID)

	require.NoError(t, reporter.ReportExpiredInQueue(ctx, expiredDownlink(h.owner, routingOwnerFirst, harnessAcEui)))
	assert.Equal(t, int64(routingACQueID), intField(t, owner.expect(scaci.CmdDLDataResult), "queId"), "the owner gets its result")
	claimant.assertQuiet(offlineQuietPeriod)

	h.storesDownlinkAs(h.owner, routingOwnerLater)
	owner.queueDownlink(harnessFirstAcOpID+1, routingACQueID)
	require.NoError(t, reporter.ReportExpiredInQueue(ctx, expiredDownlink(h.claimant, routingClaimant, claimantAcEui)))
	assert.Equal(t, int64(routingACQueID), intField(t, claimant.expect(scaci.CmdDLDataResult), "queId"), "the claimant gets its result")
	owner.assertQuiet(offlineQuietPeriod)
}

// refuseQueueRecords makes the operation log refuse every dlDataQue record,
// as an audit write that fails after the downlink was stored.
func (h *organizationHarness) refuseQueueRecords(t *testing.T) {
	t.Helper()
	ctx := testutil.TestContext()
	_, err := h.db.Exec(ctx, `CREATE FUNCTION refuse_queue_record() RETURNS trigger LANGUAGE plpgsql
		AS $$ BEGIN RAISE EXCEPTION 'dlDataQue record refused'; END $$`)
	require.NoError(t, err)
	_, err = h.db.Exec(ctx, `CREATE TRIGGER refuse_queue_record BEFORE INSERT ON scaci_operation_log
		FOR EACH ROW WHEN (NEW.command = 'dlDataQue') EXECUTE FUNCTION refuse_queue_record()`)
	require.NoError(t, err)
}

// A downlink whose dlDataQue record the operation log refused still reports
// its result to the Application Center that queued it, under the queue id it
// assigned: the stored downlink names that Application Center (SCACI §3.12).
func TestSCACIServer_DownlinkResultRoutedWhenTheQueueRecordFails(t *testing.T) {
	h := newOrganizationHarness(t)
	h.endpoints.On("GetByEUI", mock.Anything, mock.Anything, mock.Anything).Return(&models.EndPoint{Bidi: true}, "")
	h.downlinks.On("QueueDownlink", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(scaci.DownlinkQueueOutcome{Deferred: true}, "")
	var stored *storage.DownlinkMessage
	h.downlinks.On("EnqueueDownlink", mock.Anything, mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
		stored = dl
		return true
	})).Return(&storage.DownlinkMessage{QueID: routingOwnerFirst}, nil).Once()
	h.refuseQueueRecords(t)
	reporter, err := bssciservices.NewDownlinkResultReporter(h.server, bssciservices.DownlinkResultsWithoutMQTT{},
		noResultEvents{}, synchronousWork{}, logger.NewNop())
	require.NoError(t, err)
	owner := h.dialAs(t, h.ownerCert, harnessAcEui)
	owner.connect(harnessUUID(harnessUUIDFirst))

	owner.queueDownlink(harnessFirstAcOpID, routingACQueID)
	require.NotNil(t, stored)
	stored.QueID = routingOwnerFirst
	require.NoError(t, reporter.ReportExpiredInQueue(testutil.TestContext(), stored))

	assert.Equal(t, int64(routingACQueID), intField(t, owner.expect(scaci.CmdDLDataResult), "queId"), "the queuer gets its result")
}
