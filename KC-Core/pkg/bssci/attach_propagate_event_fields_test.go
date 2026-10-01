package bssci_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// attPrpCommitStorage lets the attach-propagate transaction commit against
// the capturing repositories.
type attPrpCommitStorage struct {
	*capturingStorage
}

func (s attPrpCommitStorage) BeginTx(context.Context) (bssci.AttachTx, error) {
	return attPrpCommitTx{endpoints: s.endpointRepo}, nil
}

type attPrpCommitTx struct {
	endpoints *attPrpTestEndpointRepo
}

func (t attPrpCommitTx) EndPoints() interfaces.EndpointRepository { return t.endpoints }
func (t attPrpCommitTx) EndPointSessions() interfaces.EndPointSessionRepository {
	return attPrpNoActiveSession{}
}
func (attPrpCommitTx) Commit() error   { return nil }
func (attPrpCommitTx) Rollback() error { return nil }

// attPrpNoActiveSession serves the endpoint-session calls the transaction makes.
type attPrpNoActiveSession struct {
	interfaces.EndPointSessionRepository
}

func (attPrpNoActiveSession) GetActive(context.Context, string) (*models.EndPointSession, error) {
	return nil, nil
}

func (attPrpNoActiveSession) Create(context.Context, *models.EndPointSession) error { return nil }

// The success event and the persisted attPrpCmp row report the short address,
// packet counter and endpoint owner the attPrp actually carried, read from the
// recovery record the live send path wrote (not only from its JSON round
// trip). The base station serves another tenant, so a completion that lost the
// owner would be recorded for the station's tenant.
func TestAttachPropagateCompletionReportsTheFieldsThatWereSent(t *testing.T) {
	const (
		tenantID      = int64(100)
		servingTenant = int64(7)
		epEui         = uint64(0x70B3D5677011150A)
		bsEui         = uint64(0x70B3D59CD00009E6)
		shortAddr     = uint16(0x1505)
		lastPacketCnt = uint32(42)
	)

	var modelEUI models.EUI
	binary.BigEndian.PutUint64(modelEUI[:], epEui)
	msgRepo := &capturingMIOTYMessageRepo{}
	endpointRepo := &attPrpTestEndpointRepo{endpoints: map[uint64]*models.EndPoint{
		epEui: {ID: 1001, EUI: modelEUI, TenantID: tenantID},
	}}
	storage := attPrpCommitStorage{&capturingStorage{miotyMessages: msgRepo, endpointRepo: endpointRepo}}
	eventStore := &capturingEventStore{}

	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(log, eventStore)
	server := bssci.NewTestServer(log, storage, eventStore, servingTenant,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	conn := &attPrpTestConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "attprp-fields-session",
			BaseStationEUI:    bsEui,
			Encoding:          bssci.EncodingMessagePack,
			HandshakeComplete: true,
			ResolvedTenantID:  servingTenant,
			DbSessionID:       1,
		},
		Conn:          conn,
		Bidirectional: true,
	}
	server.RegisterSession(session)

	nwkSnKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	require.NoError(t, server.SendAttachPropagate(session.ID, epEui, nwkSnKey, shortAddr, true, lastPacketCnt, false, 0, false, false))

	frames := conn.Frames()
	require.Len(t, frames, 1)
	var sent map[string]interface{}
	require.NoError(t, msgpack.Unmarshal(frames[0][mioty.FrameHeaderSize:], &sent))
	opID, ok := sent["opId"].(int64)
	require.True(t, ok, "opId is encoded as int64, got %T", sent["opId"])

	response := map[string]interface{}{"command": mioty.CmdAttachPropagateResponse, "opId": opID, "result": int64(0)}
	require.NoError(t, server.CallHandleAttachPropagateResponse(session,
		&bssci.Message{Command: mioty.CmdAttachPropagateResponse, OpId: opID, Data: response}, response))

	var propagated *models.SystemEvent
	for _, event := range eventStore.CapturedEvents() {
		if event.EventType == mioty.CmdAttachPropagate {
			propagated = event
		}
	}
	require.NotNil(t, propagated, "the attach propagate success event is recorded")
	assert.Contains(t, propagated.Description, fmt.Sprintf("%04X", shortAddr), "the event names the short address that was sent, in hex as the Add End Point form takes it")

	var completion *mioty.AttachPropagateMessage
	for _, row := range msgRepo.GetMessages() {
		if row.CommandType == mioty.CmdAttachPropagateComplete {
			completion = row
		}
	}
	require.NotNil(t, completion, "the attPrpCmp row is persisted")
	assert.Equal(t, shortAddr, completion.ShAddr)
	assert.Equal(t, lastPacketCnt, completion.LastPacketCnt)
	assert.Equal(t, tenantID, completion.TenantID, "the completion is recorded for the endpoint owner")
}
