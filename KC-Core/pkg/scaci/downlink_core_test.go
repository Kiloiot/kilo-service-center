package scaci

import (
	"fmt"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Queue ids used by the downlink core tests: the Application Center's id and
// the service center id the DL service persists the row under.
const (
	coreTenantID      int64  = 1
	coreACQueID       uint64 = 42
	coreInternalQueID int64  = 7_300_000_000_000_001
	coreBsEUI         uint64 = 0x1122334455667788
	coreEpEUI         uint64 = 0x70B3D59CD00009E7
	coreQueuerAcEui   uint64 = 0x70B3D59CD0000A42
)

func coreTestServer(dl DLService) *Server {
	return coreTestServerFor(dl, &models.EndPoint{Bidi: true})
}

// coreTestServerFor serves every endpoint lookup with endpoint.
func coreTestServerFor(dl DLService, endpoint *models.EndPoint) *Server {
	endpoints := new(MockEndpointService)
	endpoints.On("GetByEUI", mock.Anything, mock.Anything, mock.Anything).Return(endpoint, "")
	return &Server{
		registry:    newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:       testFrameCodec,
		commands:    mustTestCommandRegistry(),
		clock:       clock.SystemClock{},
		logger:      testLogger(),
		dlSvc:       dl,
		endpointSvc: endpoints,
		config:      &Config{},
	}
}

// applicationQueueIDOf is the Application Center queue id a socket request carries.
func applicationQueueIDOf(queID uint64) *uint64 {
	return &queID
}

func dispatchedUnder(queID int64) interface{} {
	return mock.MatchedBy(func(req *mioty.DLDataQueue) bool {
		return req != nil && req.QueId == uint64(queID)
	})
}

// TestProcessDLDataQueueCore_StoresApplicationQueueIDBesideServiceCenterID
// pins the per-tenant queue id model (SCACI §3.10.1): the Application Center's
// queId is persisted as its own id, the row itself is keyed by the service
// center id the DL service assigns, the base station is handed that service
// center id, and the core never looks the Application Center id up across
// tenants before the insert.
func TestProcessDLDataQueueCore_StoresApplicationQueueIDBesideServiceCenterID(t *testing.T) {
	orgID := uuid.New()
	mockDL := new(MockDLService)
	mockDL.On("EnqueueDownlink", mock.Anything, queuedByTheSession()).
		Return(&storage.DownlinkMessage{ID: 42, QueID: coreInternalQueID}, nil)
	mockDL.On("QueueDownlink", mock.Anything, dispatchedUnder(coreInternalQueID), coreTenantID, orgID).
		Return(DownlinkQueueOutcome{QueID: uint64(coreInternalQueID), BsEui: coreBsEUI}, "")
	server := coreTestServer(mockDL)

	result, errToken, posixCode := server.processDLDataQueueCore(testutil.TestContext(),
		&Session{TenantID: coreTenantID, OrganizationID: orgID, AcEui: coreQueuerAcEui}, 101, &DLDataQueue{
			EpEui:    coreEpEUI,
			QueId:    coreACQueID,
			UserData: [][]byte{{0x01, 0x02, 0x03}},
		}, applicationQueueIDOf(coreACQueID), "")
	require.Empty(t, errToken)
	require.Zero(t, posixCode)
	require.NotNil(t, result)
	assert.Equal(t, uint64(coreInternalQueID), result.QueID)
	assert.Equal(t, coreBsEUI, result.BsEui)
	mockDL.AssertExpectations(t)
}

// TestProcessDLDataQueueCore_DuplicateApplicationQueueIDIsEEXIST pins that
// an Application Center queue id already queued in the same tenant is
// refused with EEXIST (SCACI §3.10.1), classified from the persistence result
// rather than a pre-insert lookup.
func TestProcessDLDataQueueCore_DuplicateApplicationQueueIDIsEEXIST(t *testing.T) {
	orgID := uuid.New()
	mockDL := new(MockDLService)
	mockDL.On("EnqueueDownlink", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("enqueue downlink: %w", storage.ErrDuplicateKey))
	server := coreTestServer(mockDL)

	result, errToken, posixCode := server.processDLDataQueueCore(testutil.TestContext(),
		&Session{TenantID: coreTenantID, OrganizationID: orgID}, 102, &DLDataQueue{
			EpEui:    coreEpEUI,
			QueId:    coreACQueID,
			UserData: [][]byte{{0x01}},
		}, applicationQueueIDOf(coreACQueID), "")
	assert.Nil(t, result)
	assert.Equal(t, errQueIDExists, errToken)
	assert.Equal(t, POSIX_EEXIST, posixCode)
	mockDL.AssertNotCalled(t, "QueueDownlink", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestProcessDLDataQueueCore_DispatchFailureStillAccepted pins that a
// persisted downlink is accepted whatever the immediate dispatch reports: the
// row stays pending for the dlOpen dispatch path, so failing the request
// would make a retrying caller queue it twice.
func TestProcessDLDataQueueCore_DispatchFailureStillAccepted(t *testing.T) {
	for _, schedToken := range []string{errSchedulerUnavailable, errFailedRecordOperation, errDownlinkNotFound} {
		t.Run(schedToken, func(t *testing.T) {
			orgID := uuid.New()
			mockDL := new(MockDLService)
			mockDL.On("EnqueueDownlink", mock.Anything, mock.Anything).
				Return(&storage.DownlinkMessage{ID: 43, QueID: coreInternalQueID}, nil)
			mockDL.On("QueueDownlink", mock.Anything, dispatchedUnder(coreInternalQueID), coreTenantID, orgID).
				Return(DownlinkQueueOutcome{}, schedToken)
			opRepo := new(MockSCACIOperationRepository)
			recorder := new(MockOperationRecorder)
			recorder.On("Record", mock.Anything, mock.Anything, int64(103), CmdDLDataQueue,
				models.OperationDirectionInbound, mock.Anything).Return(nil)
			server := coreTestServer(mockDL)
			server.operationRepo = opRepo
			server.operationRecorder = recorder

			result, errToken, posixCode := server.processDLDataQueueCore(testutil.TestContext(),
				&Session{ID: 9, TenantID: coreTenantID, OrganizationID: orgID}, 103, &DLDataQueue{
					EpEui:    coreEpEUI,
					QueId:    coreACQueID,
					UserData: [][]byte{{0xAA}},
				}, applicationQueueIDOf(coreACQueID), "")
			require.Empty(t, errToken)
			require.Zero(t, posixCode)
			require.NotNil(t, result)
			assert.True(t, result.Deferred)
			assert.Equal(t, bssci.DLQueueStatusPending, result.Status())
			assert.Equal(t, uint64(coreInternalQueID), result.QueID)
			opRepo.AssertNotCalled(t, "UpdateOperationState", mock.Anything, mock.Anything, mock.Anything,
				models.OperationStateFailed, mock.Anything)
		})
	}
}

// TestQueueDownlinkInternal_ReturnsServiceCenterQueueID pins the gRPC and
// MQTT view: an internal request carries no Application Center id, the row is
// persisted without one and with the MQTT command's ref, and the caller
// receives the service center id, pending while no base station can take it
// (SCACI §3.10).
func TestQueueDownlinkInternal_ReturnsServiceCenterQueueID(t *testing.T) {
	const commandRef = "order-17"
	orgID := uuid.New()
	mockDL := new(MockDLService)
	mockDL.On("EnqueueDownlink", mock.Anything, mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
		return dl != nil && dl.ACQueID == nil && dl.ACEUI == nil && dl.Status == bssci.DLQueueStatusPending &&
			dl.OrganizationID != nil && *dl.OrganizationID == orgID && dl.Ref == commandRef
	})).Return(&storage.DownlinkMessage{ID: 45, QueID: coreInternalQueID}, nil)
	mockDL.On("QueueDownlink", mock.Anything, dispatchedUnder(coreInternalQueID), coreTenantID, orgID).
		Return(DownlinkQueueOutcome{QueID: uint64(coreInternalQueID), Deferred: true}, "")
	server := coreTestServer(mockDL)

	result, err := server.QueueDownlinkInternal(testutil.TestContext(), coreTenantID, &orgID, &mioty.DLDataQueue{
		EpEui:    coreEpEUI,
		UserData: [][]byte{{0x01}},
	}, commandRef)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, uint64(coreInternalQueID), result.QueID)
	assert.Zero(t, result.BsEui)
	assert.Equal(t, bssci.DLQueueStatusPending, result.Status)
	mockDL.AssertExpectations(t)
}

// TestQueueDownlinkInternal_ExhaustedServiceCenterIDsIsAPersistenceFailure
// pins that running out of service center queue ids is the service center's
// failure (EIO), never an Application Center duplicate.
func TestQueueDownlinkInternal_ExhaustedServiceCenterIDsIsAPersistenceFailure(t *testing.T) {
	orgID := uuid.New()
	mockDL := new(MockDLService)
	mockDL.On("EnqueueDownlink", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("attempts exhausted: %w", storage.ErrDownlinkQueueIDTaken))
	server := coreTestServer(mockDL)

	result, err := server.QueueDownlinkInternal(testutil.TestContext(), coreTenantID, &orgID, &mioty.DLDataQueue{
		EpEui:    coreEpEUI,
		UserData: [][]byte{{0x01}},
	}, "")
	require.Nil(t, result)
	var queueErr *DLDataQueueError
	require.ErrorAs(t, err, &queueErr)
	assert.Equal(t, errFailedPersistDownlink, queueErr.Token)
	assert.Equal(t, POSIX_EIO, queueErr.POSIX)
}

// TestHandleDLDataQueue_AcceptsApplicationQueueIDZero pins SCACI §3.10.1
// l.509: the Application Center assigns any 64-bit queue id, zero included,
// and the downlink is stored under it beside the service center's own id.
func TestHandleDLDataQueue_AcceptsApplicationQueueIDZero(t *testing.T) {
	mockDL := new(MockDLService)
	var stored *storage.DownlinkMessage
	mockDL.On("EnqueueDownlink", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		stored = args.Get(1).(*storage.DownlinkMessage)
	}).Return(&storage.DownlinkMessage{QueID: coreInternalQueID}, nil)
	mockDL.On("QueueDownlink", mock.Anything, mock.Anything, int64(coreTenantID), mock.Anything).
		Return(DownlinkQueueOutcome{QueID: uint64(coreInternalQueID), Deferred: true}, "")
	server := coreTestServer(mockDL)
	payload, err := msgpack.Marshal(&DLDataQueue{
		BaseMessage: mioty.BaseMessage{CommandType: CmdDLDataQueue, OpId: 104},
		EpEui:       coreEpEUI,
		QueId:       0,
		UserData:    [][]byte{{0x01}},
	})
	require.NoError(t, err)
	conn := &mockConn{}

	require.NoError(t, server.handleDLDataQueue(conn, &Session{TenantID: coreTenantID, OrganizationID: uuid.New(), State: StateActive}, 104, payload))

	var rsp DLDataQueueResponse
	require.NoError(t, decodeResponse(conn.written, &rsp))
	assert.Equal(t, CmdDLDataQueueResponse, rsp.CommandType, "an Application Center queue id of zero is accepted")
	require.NotNil(t, stored)
	require.NotNil(t, stored.ACQueID, "the downlink keeps the Application Center's queue id zero")
	assert.Zero(t, *stored.ACQueID)
}

// TestBroadcastDLDataResult_ReportsApplicationQueueID pins that the dlDataRes
// sent to Application Centers carries the queue id they assigned, never the
// service center id the base station reported (SCACI §3.12.1).
func TestBroadcastDLDataResult_ReportsApplicationQueueID(t *testing.T) {
	conn := &mockConn{}
	server := coreTestServer(new(MockDLService))
	server.registry.sessions[conn] = &Session{TenantID: coreTenantID, AcEui: originQueuerAcEui, State: StateActive}

	require.NoError(t, server.BroadcastDLDataResult(testutil.TestContext(), ApplicationCenter{TenantID: coreTenantID, AcEui: originQueuerAcEui}, coreACQueID, &mioty.DLDataResult{
		EpEui:  coreEpEUI,
		QueId:  uint64(coreInternalQueID),
		Result: mioty.ResultExpired,
	}))

	var sent DLDataResult
	require.NoError(t, decodeResponse(conn.written, &sent))
	assert.Equal(t, coreACQueID, sent.QueID)
}

// TestProcessDLDataQueueCore_RefusesUnidirectionalEndpoint: an endpoint
// registered without bidi never opens a downlink window, so every ingress
// (SCACI, gRPC, MQTT share this core) refuses the downlink before persisting it.
func TestProcessDLDataQueueCore_RefusesUnidirectionalEndpoint(t *testing.T) {
	mockDL := new(MockDLService)
	server := coreTestServerFor(mockDL, &models.EndPoint{Bidi: false})

	result, errToken, posixCode := server.processDLDataQueueCore(testutil.TestContext(),
		&Session{TenantID: coreTenantID, OrganizationID: uuid.New()}, 104, &DLDataQueue{
			EpEui:    coreEpEUI,
			QueId:    coreACQueID,
			UserData: [][]byte{{0x01}},
		}, applicationQueueIDOf(coreACQueID), "")

	assert.Nil(t, result)
	assert.Equal(t, errEndpointNotBidirectional, errToken)
	assert.Equal(t, POSIX_ENOTSUP, posixCode)
	mockDL.AssertNotCalled(t, "EnqueueDownlink", mock.Anything, mock.Anything)
	mockDL.AssertNotCalled(t, "QueueDownlink", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestProcessDLDataQueueCore_AcceptsEveryUnsigned64BitApplicationQueueID:
// SCACI §3.10.1 makes the queId a 64-bit numeric the Application Center
// chooses, so an id at or above 2^63 is queued under exactly that id.
func TestProcessDLDataQueueCore_AcceptsEveryUnsigned64BitApplicationQueueID(t *testing.T) {
	const beyondSigned = uint64(1)<<63 + 5
	orgID := uuid.New()
	mockDL := new(MockDLService)
	mockDL.On("EnqueueDownlink", mock.Anything, mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
		return dl != nil && dl.ACQueID != nil && *dl.ACQueID == beyondSigned
	})).Return(&storage.DownlinkMessage{ID: 43, QueID: coreInternalQueID}, nil)
	mockDL.On("QueueDownlink", mock.Anything, dispatchedUnder(coreInternalQueID), coreTenantID, orgID).
		Return(DownlinkQueueOutcome{QueID: uint64(coreInternalQueID), BsEui: coreBsEUI}, "")
	server := coreTestServer(mockDL)

	result, errToken, posixCode := server.processDLDataQueueCore(testutil.TestContext(),
		&Session{TenantID: coreTenantID, OrganizationID: orgID}, 103, &DLDataQueue{
			EpEui:    coreEpEUI,
			QueId:    beyondSigned,
			UserData: [][]byte{{0x01}},
		}, applicationQueueIDOf(beyondSigned), "")
	require.Empty(t, errToken)
	require.Zero(t, posixCode)
	require.NotNil(t, result)
	mockDL.AssertExpectations(t)
}
