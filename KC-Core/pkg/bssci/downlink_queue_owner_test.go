package bssci

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	queueOwnerTenant   = int64(3)
	queueOwnerQueueID  = int64(910)
	queueOwnerEndpoint = uint64(0x70b3d59cd0000341)
)

// queueMetadata answers every queue metadata lookup with one operation's metadata.
type queueMetadata struct {
	StatusService
	endpointEUI uint64
	queueID     int64
	tenant      string
}

func (m queueMetadata) ExtractQueueMetadata(*Session, int64) (uint64, int64, string, *uuid.UUID) {
	return m.endpointEUI, m.queueID, m.tenant, nil
}

// ackRecorder records every queue acknowledgement the service is handed.
type ackRecorder struct {
	mqttTestDownlinkService
	acks []QueueAcknowledgement
}

func (r *ackRecorder) ProcessQueueAck(_ context.Context, _ *Session, ack QueueAcknowledgement) error {
	r.acks = append(r.acks, ack)
	return nil
}

// queueAckAudit records the tenant every queue acknowledgement is filed under.
type queueAckAudit struct {
	noopAuditLogger
	tenants []string
}

func (a *queueAckAudit) RecordQueueAck(_ context.Context, tenant string, _ *Session, _ uint64, _ int64, _ int64) error {
	a.tenants = append(a.tenants, tenant)
	return nil
}

func acknowledgeQueue(t *testing.T, recordedTenant string, resolveOwner bool) (*ackRecorder, *queueAckAudit) {
	t.Helper()
	server := NewTestServerWithMemoryStatusService(logger.NewNop(), nil, nil, 1)
	server.statusSvc = queueMetadata{
		StatusService: server.statusSvc, endpointEUI: queueOwnerEndpoint, queueID: queueOwnerQueueID, tenant: recordedTenant,
	}
	downlinks := &ackRecorder{}
	server.downlinkSvc = downlinks
	audit := &queueAckAudit{}
	server.auditLogger = audit
	if resolveOwner {
		server.tenantResolver.RegisterQueueTenant(queueOwnerQueueID, strconv.FormatInt(queueOwnerTenant, 10))
	}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{ID: "queue-station", BaseStationEUI: TestBsEui04, Encoding: EncodingJSON, DbSessionID: 1},
		Conn:                 &bsscitest.TestConn{Encoding: "json"},
	}

	require.NoError(t, server.handleDLDataQueueResponse(session, &Message{Command: mioty.CmdDLDataQueueResponse, OpId: -7}, nil))
	return downlinks, audit
}

// TestDLDataQueueResponse_FilesUnderTheQueueRowOwner: an acknowledgement whose
// operation lost its tenant is filed under the owner of the queue row, never
// under the server's default tenant.
func TestDLDataQueueResponse_FilesUnderTheQueueRowOwner(t *testing.T) {
	downlinks, audit := acknowledgeQueue(t, "", true)

	assert.Equal(t, []QueueAcknowledgement{{QueueID: queueOwnerQueueID, OwnerTenant: "3"}}, downlinks.acks)
	assert.Equal(t, []string{"3"}, audit.tenants)
}

// TestDLDataQueueResponse_UnresolvedOwnerTouchesNothing: with no owner to
// file it under, the acknowledgement updates no row and records no event.
func TestDLDataQueueResponse_UnresolvedOwnerTouchesNothing(t *testing.T) {
	downlinks, audit := acknowledgeQueue(t, "", false)

	assert.Empty(t, downlinks.acks)
	assert.Empty(t, audit.tenants)
}

// TestDLDataQueueResponse_KeepsTheRecordedTenant: the tenant recorded with the
// dlDataQue operation is the owner.
func TestDLDataQueueResponse_KeepsTheRecordedTenant(t *testing.T) {
	downlinks, audit := acknowledgeQueue(t, "5", true)

	assert.Equal(t, []QueueAcknowledgement{{QueueID: queueOwnerQueueID, OwnerTenant: "5"}}, downlinks.acks)
	assert.Equal(t, []string{"5"}, audit.tenants)
}
