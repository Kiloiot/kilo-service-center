package bssci

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// queueRevokeAudit records the downlinks it is told were revoked in the queue.
type queueRevokeAudit struct {
	noopAuditLogger
	revoked []*storage.DownlinkMessage
	err     error
}

func (a *queueRevokeAudit) RecordQueueRevoked(_ context.Context, downlink *storage.DownlinkMessage) error {
	a.revoked = append(a.revoked, downlink)
	return a.err
}

// revokedInQueue is the row a revoke in the queue leaves behind.
var revokedInQueue = &storage.DownlinkMessage{
	EPEUI: revokeEndpointEUI, TenantID: "3", Status: mioty.DLQueueStatusRevoked, QueID: int64(revokeQueueID),
}

// Every revoke source reaches the queue through RevokeDownlink: an
// Application Center's dlDataRev and its deregistration (SCACI §3.11, §3.6)
// and an operator's revoke name the owner's organization and endpoint. A
// downlink revoked in the queue is announced under its owner, so every open
// view of the endpoint's queue shows it revoked.
func TestRevokeDownlink_ARevokeInTheQueueIsAnnounced(t *testing.T) {
	org := uuid.New()
	endpoint := uint64(0x70B3D59CD0000341)
	for name, ref := range map[string]scheduler.DownlinkRef{
		"application center dlDataRev":        {TenantID: revokeOwnerTenant, QueID: revokeQueueID, OrganizationID: &org, EpEUI: &endpoint},
		"application center deregistration":   {TenantID: revokeOwnerTenant, QueID: revokeQueueID, OrganizationID: &org, EpEUI: &endpoint},
		"operator revoke through the web API": {TenantID: revokeOwnerTenant, QueID: revokeQueueID, OrganizationID: &org, EpEUI: &endpoint},
	} {
		t.Run(name, func(t *testing.T) {
			server, _ := newRevokeServer(t, queueRowStore{row: revokedInQueue}, &pendingRevocations{pending: true})
			audit := &queueRevokeAudit{}
			server.auditLogger = audit

			bsEui, err := server.RevokeDownlink(testutil.TestContext(), ref)

			require.NoError(t, err)
			assert.Zero(t, bsEui)
			assert.Equal(t, []*storage.DownlinkMessage{revokedInQueue}, audit.revoked,
				"the downlink revoked in the queue is announced once")
		})
	}
}

// A downlink a base station holds is revoked there; its revocation is
// announced when the station confirms it, not by the queue.
func TestRevokeDownlink_ARevokeSentToTheStationIsNotAnnouncedByTheQueue(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{row: heldDownlink(mioty.DLQueueStatusQueued)}, &pendingRevocations{})
	audit := &queueRevokeAudit{}
	server.auditLogger = audit
	registerRoamingStation(server)

	_, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)

	require.NoError(t, err)
	assert.Empty(t, audit.revoked)
}

// The revoke stands when its announcement cannot be recorded: the caller is
// answered as for any revoke in the queue.
func TestRevokeDownlink_ARevokeInTheQueueStandsWhenItsEventFails(t *testing.T) {
	for name, fault := range map[string]struct {
		queue queueRowStore
		audit *queueRevokeAudit
	}{
		"the revoked row cannot be read": {queue: queueRowStore{err: errors.New("read failed")}, audit: &queueRevokeAudit{}},
		"the event cannot be written":    {queue: queueRowStore{row: revokedInQueue}, audit: &queueRevokeAudit{err: errors.New("write failed")}},
	} {
		t.Run(name, func(t *testing.T) {
			server, _ := newRevokeServer(t, fault.queue, &pendingRevocations{pending: true})
			server.auditLogger = fault.audit

			bsEui, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)

			require.NoError(t, err)
			assert.Zero(t, bsEui)
		})
	}
}
