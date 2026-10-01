package bssciservices

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// dispatchTestNow is the instant the dispatcher test clock reports.
var dispatchTestNow = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// Fixture errors returned by the dispatcher test doubles.
var (
	errTestDBConnectionFailed    = errors.New("database connection failed")
	errTestCommitFailed          = errors.New("commit failed")
	errTestSendFailed            = errors.New("send failed")
	errTestWritePayloadAmbiguous = fmt.Errorf("write payload: %w", bssci.ErrAmbiguousWrite)
	errTestMarkQueuedFailed      = errors.New("mark queued failed")
	errTestReserveDBDown         = errors.New("db down")
)

// mockLoggerForDispatch is a minimal mock implementing logger.Logger
type mockLoggerForDispatch struct{}

func (m *mockLoggerForDispatch) Debug(_ string, _ ...interface{})                           {}
func (m *mockLoggerForDispatch) Info(_ string, _ ...interface{})                            {}
func (m *mockLoggerForDispatch) Warn(_ string, _ ...interface{})                            {}
func (m *mockLoggerForDispatch) Error(_ string, _ ...interface{})                           {}
func (m *mockLoggerForDispatch) Fatal(_ string, _ ...interface{})                           {}
func (m *mockLoggerForDispatch) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLoggerForDispatch) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLoggerForDispatch) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLoggerForDispatch) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLoggerForDispatch) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLoggerForDispatch) WithField(_ string, _ interface{}) logger.Logger            { return m }

func (m *mockLoggerForDispatch) WithFields(_ map[string]interface{}) logger.Logger { return m }

// mockReserverForDispatch is a focused fake for DownlinkReserver. The
// transaction the real adapter opens is not visible here by design: the
// dispatcher only observes whether a row was durably reserved.
type mockReserverForDispatch struct {
	reserveResult        *storage.DownlinkMessage
	reserveErr           error
	reserveByQueueResult *storage.DownlinkMessage
	reserveByQueueErr    error
	reserveByQueueCalls  []reserveByQueueCall
	// reserveNextCalls counts automatic-path reservations.
	reserveNextCalls int
	// reserved records that a reservation was durably committed, which the
	// dispatcher relies on before performing any wire write.
	reserved bool
}

func (m *mockReserverForDispatch) ReserveNextPending(_ context.Context, _ int64, _ []byte,
	_ uint64,
) (*storage.DownlinkMessage, error) {
	m.reserveNextCalls++
	if m.reserveErr != nil {
		return nil, m.reserveErr
	}
	if m.reserveResult == nil {
		// Mirrors the reservation adapter: nothing pending is storage.ErrNotFound.
		return nil, fmt.Errorf("downlink reservation: reserve: %w", storage.ErrNotFound)
	}
	m.reserved = true
	return m.reserveResult, nil
}

func (m *mockReserverForDispatch) ReserveByQueueID(_ context.Context, tenantID int64, orgID uuid.UUID,
	queueID uint64, epEUI []byte, bsEUI uint64,
) (*storage.DownlinkMessage, error) {
	m.reserveByQueueCalls = append(m.reserveByQueueCalls, reserveByQueueCall{
		tenantID: tenantID, orgID: orgID, queueID: queueID, epEUI: epEUI, bsEUI: bsEUI,
	})
	if m.reserveByQueueErr != nil {
		return nil, m.reserveByQueueErr
	}
	if m.reserveByQueueResult == nil {
		return nil, storage.ErrNotFound
	}
	m.reserved = true
	return m.reserveByQueueResult, nil
}

// reserveByQueueCall captures ReservePendingDownlinkByQueueID arguments
type reserveByQueueCall struct {
	tenantID int64
	orgID    uuid.UUID
	queueID  uint64
	epEUI    []byte
	bsEUI    uint64
}

// mockMIOTYDownlinksForDispatch implements dispatch-specific methods for testing
type mockMIOTYDownlinksForDispatch struct {
	reserveResult        *storage.DownlinkMessage
	reserveErr           error
	reserveByQueueResult *storage.DownlinkMessage
	reserveByQueueErr    error
	reserveByQueueCalls  []reserveByQueueCall
	markQueuedErr        error
	markQueuedCalls      int
	markQueuedTxTimes    []int64                   // captured confirmation transmission times
	markQueuedOrgIDs     []*uuid.UUID              // captured confirmation organizations
	statusUpdates        []mioty.DLQueueStatus     // captured UpdateDownlinkStatus statuses
	statusUpdateOrgIDs   []*uuid.UUID              // captured release organizations
	releasedStations     []uint64                  // captured ReleaseStationReservations stations
	releaseKeeps         [][]int64                 // captured kept queue ids per release
	releasedQueues       []uint64                  // captured ReleaseStationQueue stations
	endpointReleases     []endpointRelease         // captured ReleaseEndpointAtStation calls
	released             []storage.PendingDownlink // rows every release returns
	releaseErr           error
}

// endpointRelease is one ReleaseEndpointAtStation call.
type endpointRelease struct {
	tenantID int64
	epEUI    uint64
	bsEUI    uint64
	sentBy   time.Time
}

func (m *mockMIOTYDownlinksForDispatch) ReleaseEndpointAtStation(_ context.Context, tenantID int64, epEUI, bsEUI uint64, sentBy time.Time) ([]storage.PendingDownlink, error) {
	m.endpointReleases = append(m.endpointReleases, endpointRelease{tenantID: tenantID, epEUI: epEUI, bsEUI: bsEUI, sentBy: sentBy})
	return m.releasedRows()
}

func (m *mockMIOTYDownlinksForDispatch) ReleaseStationQueue(_ context.Context, bsEUI uint64) ([]storage.PendingDownlink, error) {
	m.releasedQueues = append(m.releasedQueues, bsEUI)
	return m.releasedRows()
}

func (m *mockMIOTYDownlinksForDispatch) ReleaseStationReservations(_ context.Context, bsEUI uint64, keepQueIDs []int64) ([]storage.PendingDownlink, error) {
	m.releasedStations = append(m.releasedStations, bsEUI)
	m.releaseKeeps = append(m.releaseKeeps, keepQueIDs)
	return m.releasedRows()
}

func (m *mockMIOTYDownlinksForDispatch) releasedRows() ([]storage.PendingDownlink, error) {
	if m.releaseErr != nil {
		return nil, m.releaseErr
	}
	return m.released, nil
}

func (m *mockMIOTYDownlinksForDispatch) ReserveNextPendingDownlink(_ context.Context, _ int64, _ []byte, _ uint64) (*storage.DownlinkMessage, error) {
	if m.reserveErr != nil {
		return nil, m.reserveErr
	}
	return m.reserveResult, nil
}

func (m *mockMIOTYDownlinksForDispatch) ReservePendingDownlinkByQueueID(_ context.Context, tenantID int64, orgID uuid.UUID, queueID uint64, epEUI []byte, bsEUI uint64) (*storage.DownlinkMessage, error) {
	m.reserveByQueueCalls = append(m.reserveByQueueCalls, reserveByQueueCall{
		tenantID: tenantID, orgID: orgID, queueID: queueID, epEUI: epEUI, bsEUI: bsEUI,
	})
	if m.reserveByQueueErr != nil {
		return nil, m.reserveByQueueErr
	}
	return m.reserveByQueueResult, nil
}

// orgID parameter enables organization-scoped queue marking
func (m *mockMIOTYDownlinksForDispatch) MarkReservedAsQueued(_ context.Context, _ uint64, _ int64, _ uint64, txTime int64, _ *uint32, orgID *uuid.UUID) error {
	m.markQueuedCalls++
	m.markQueuedTxTimes = append(m.markQueuedTxTimes, txTime)
	m.markQueuedOrgIDs = append(m.markQueuedOrgIDs, orgID)
	return m.markQueuedErr
}

// Stub remaining interface methods
func (m *mockMIOTYDownlinksForDispatch) GetDownlinkQueue(_ context.Context, _ string, _ string) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockMIOTYDownlinksForDispatch) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockMIOTYDownlinksForDispatch) UpdatePendingDownlink(_ context.Context, _ int64, _ *uuid.UUID, _ []byte, _ int64, _ storage.DownlinkPatch) (*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockMIOTYDownlinksForDispatch) GetDownlinkResults(_ context.Context, _ int64, _ *uuid.UUID, _ storage.DownlinkResultFilter, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return nil, 0, nil
}

func (m *mockMIOTYDownlinksForDispatch) EnqueueDownlink(_ context.Context, _ *storage.DownlinkMessage, _ time.Duration) (*storage.DownlinkMessage, error) {
	return nil, nil
}

// UpdateDownlinkStatus captures release-to-pending calls
func (m *mockMIOTYDownlinksForDispatch) UpdateDownlinkStatus(_ context.Context, _ string, status mioty.DLQueueStatus, orgID *uuid.UUID) error {
	m.statusUpdates = append(m.statusUpdates, status)
	m.statusUpdateOrgIDs = append(m.statusUpdateOrgIDs, orgID)
	return nil
}

func (m *mockMIOTYDownlinksForDispatch) UpdateDownlinkResult(_ context.Context, tenantID int64, _ uint64, result *mioty.DLDataResult) (*storage.DownlinkMessage, error) {
	return &storage.DownlinkMessage{QueID: int64(result.QueId), EPEUI: mioty.FormatEUI64(result.EpEui), TenantID: strconv.FormatInt(tenantID, 10)}, nil
}

func (m *mockMIOTYDownlinksForDispatch) FailQueuedDownlink(_ context.Context, queID int64, tenantID int64, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return &storage.DownlinkMessage{QueID: queID, TenantID: strconv.FormatInt(tenantID, 10)}, nil
}

func (m *mockMIOTYDownlinksForDispatch) UpdateDownlinkBaseStation(context.Context, uint64, int64, uint64) error {
	return nil
}

func (m *mockMIOTYDownlinksForDispatch) RevokeDownlink(context.Context, storage.DownlinkRevocation) (bool, error) {
	return true, nil
}

func (m *mockMIOTYDownlinksForDispatch) ExpireRevokedDownlink(context.Context, storage.DownlinkRevocation) (*storage.DownlinkMessage, bool, error) {
	return nil, false, nil
}

// mockSendFn tracks calls to SendDLDataQueue
type mockSendFn struct {
	calls       int
	err         error
	dlRxStatQry []bool       // captured dlRxStatQry flag per call
	sentOrgIDs  []*uuid.UUID // captured delivery organization per call
	// reservedAtSend records whether the row had been durably reserved at the
	// moment the wire send ran
	reserver       *mockReserverForDispatch
	reservedAtSend []bool
}

func (m *mockSendFn) Send(_ string, _ uint64, _ [][]byte, _ int64, _ float32, _ bool, _ []int64, _ uint8, _, _, _, _ bool, _ int64, orgID *uuid.UUID, dlRxStatQry bool) error {
	m.calls++
	m.dlRxStatQry = append(m.dlRxStatQry, dlRxStatQry)
	m.sentOrgIDs = append(m.sentOrgIDs, orgID)
	if m.reserver != nil {
		m.reservedAtSend = append(m.reservedAtSend, m.reserver.reserved)
	}
	return m.err
}

// newDispatchFixture builds the dispatcher doubles around one queue row. A row
// without an organization is given one, mirroring the repository invariant
// (enqueue rejects ownerless rows and the column is NOT NULL); tests that pin
// the ownerless-row guard construct the row explicitly.
func newDispatchFixture(dl *storage.DownlinkMessage) (*mockMIOTYDownlinksForDispatch, *mockReserverForDispatch, *mockSendFn) {
	if dl != nil && dl.OrganizationID == nil {
		org := uuid.New()
		dl.OrganizationID = &org
	}
	dlRepo := &mockMIOTYDownlinksForDispatch{}
	reserver := &mockReserverForDispatch{
		reserveResult:        dl,
		reserveByQueueResult: dl,
	}
	sendFn := &mockSendFn{reserver: reserver}
	return dlRepo, reserver, sendFn
}

func mustDispatcher(t *testing.T, log logger.Logger, reserver DownlinkReserver, queue DownlinkConfirmer, sendFn SendDLQueueFunc) bssci.DownlinkDispatcher {
	t.Helper()
	dispatcher, err := NewDownlinkDispatcher(log, reserver, queue, &windowClaims{}, sendFn, testutil.NewFakeClock(dispatchTestNow))
	if err != nil {
		t.Fatalf("NewDownlinkDispatcher: %v", err)
	}
	return dispatcher
}

func testDispatchSession() *bssci.Session {
	return &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:             "test-session-123",
			BaseStationEUI: 0x1234567890ABCDEF,
		},
		Bidirectional: true,
	}
}

// TestDispatchIfAvailable_UnidirectionalStationReservesNothing: a downlink
// window reported by a base station that cannot transmit downlinks leaves the
// queue untouched for a bidirectional station hearing the endpoint.
func TestDispatchIfAvailable_UnidirectionalStationReservesNothing(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{QueID: 12346, Payload: []byte{0x01}})
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)
	session := testDispatchSession()
	session.Bidirectional = false

	dispatched, err := dispatcher.DispatchIfAvailable(testutil.TestContext(), 42, session, 0xAABBCCDDEEFF0011, dispatchTestMessageID, true)

	if err != nil || dispatched {
		t.Fatalf("expected no dispatch and no error, got dispatched=%v err=%v", dispatched, err)
	}
	if reserver.reserveNextCalls != 0 {
		t.Errorf("a unidirectional station must reserve nothing, got %d reservations", reserver.reserveNextCalls)
	}
	if sendFn.calls != 0 || len(dlRepo.statusUpdates) != 0 {
		t.Errorf("nothing is sent or released, got %d sends and %v status updates", sendFn.calls, dlRepo.statusUpdates)
	}
}

func TestDispatchIfAvailable_Success(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID:       12345,
		Payload:     []byte("test payload"),
		Priority:    1.0,
		Format:      0,
		ResponseExp: true,
	})

	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dispatched {
		t.Error("expected dispatched=true")
	}
	if sendFn.calls != 1 {
		t.Errorf("expected 1 sendFn call, got %d", sendFn.calls)
	}
	if dlRepo.markQueuedCalls != 1 {
		t.Errorf("expected 1 markQueued call, got %d", dlRepo.markQueuedCalls)
	}
	if len(dlRepo.markQueuedTxTimes) != 1 || dlRepo.markQueuedTxTimes[0] != dispatchTestNow.UnixNano() {
		t.Errorf("the queued time comes from the injected clock, got %v", dlRepo.markQueuedTxTimes)
	}
	if !reserver.reserved {
		t.Error("expected the queue row to be durably reserved")
	}
	// The reservation must be durable before any wire write
	if len(sendFn.reservedAtSend) != 1 || !sendFn.reservedAtSend[0] {
		t.Error("expected the row reserved BEFORE the wire send")
	}
}

func TestDispatchIfAvailable_DlRxStatQryPassthrough(t *testing.T) {
	for _, want := range []bool{true, false} {
		dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
			QueID:       7,
			Payload:     []byte("p"),
			DlRxStatQry: want,
		})
		dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

		dispatched, err := dispatcher.DispatchIfAvailable(
			testutil.TestContext(), 42, testDispatchSession(),
			0xAABBCCDDEEFF0011, dispatchTestMessageID, false,
		)
		if err != nil || !dispatched {
			t.Fatalf("dlRxStatQry=%v: unexpected result dispatched=%v err=%v", want, dispatched, err)
		}
		if len(sendFn.dlRxStatQry) != 1 || sendFn.dlRxStatQry[0] != want {
			t.Errorf("expected sendFn dlRxStatQry=%v, got %v", want, sendFn.dlRxStatQry)
		}
	}
}

func TestDispatchIfAvailable_NoPendingDownlinks(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(nil)
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false when no pending downlinks")
	}
	if sendFn.calls != 0 {
		t.Errorf("expected 0 sendFn calls, got %d", sendFn.calls)
	}
	if reserver.reserved {
		t.Error("expected no reservation when nothing is pending")
	}
}

// TestDispatchIfAvailable_EmptyQueueIsNotAnError pins that a dlOpen uplink
// with nothing pending is the normal case: no ERROR is logged for it.
func TestDispatchIfAvailable_EmptyQueueIsNotAnError(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(nil)
	log := bsscitest.NewRecordingLogger()
	dispatcher := mustDispatcher(t, log, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false when no pending downlinks")
	}
	if errs := log.AllAtLeast("ERROR"); len(errs) != 0 {
		t.Errorf("an empty queue must not log an error, got %q", errs[0].Message)
	}
	if len(log.FilterMessage(bssci.LogDispatcherNoPending)) != 1 {
		t.Error("an empty queue is reported as nothing pending")
	}
}

func TestDispatchIfAvailable_NoTenantContext(t *testing.T) {
	reserver := &mockReserverForDispatch{}
	dlRepo := &mockMIOTYDownlinksForDispatch{}
	sendFn := &mockSendFn{}
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 0, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false when no tenant context")
	}
	if sendFn.calls != 0 {
		t.Errorf("expected 0 sendFn calls, got %d", sendFn.calls)
	}
	if reserver.reserveNextCalls != 0 {
		t.Error("no reservation may run without a tenant")
	}
}

func TestDispatchIfAvailable_TransactionBeginError(t *testing.T) {
	reserver := &mockReserverForDispatch{reserveErr: errTestDBConnectionFailed}
	dlRepo := &mockMIOTYDownlinksForDispatch{}
	sendFn := &mockSendFn{}
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	// Should gracefully degrade, not return error (don't fail uplink)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on tx begin error")
	}
}

func TestDispatchIfAvailable_ReservationCommitError(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	reserver.reserveErr = errTestCommitFailed
	reserver.reserveResult = nil
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on reservation commit error")
	}
	// Nothing may reach the wire when the reservation never became durable
	if sendFn.calls != 0 {
		t.Errorf("expected 0 sendFn calls, got %d", sendFn.calls)
	}
}

func TestDispatchIfAvailable_SendFunctionError_ReleasesToPending(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		ID: 9, QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	sendFn.err = errTestSendFailed
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)

	if err == nil {
		t.Fatal("expected error on definite send failure")
	}
	if dispatched {
		t.Error("expected dispatched=false on send error")
	}
	// Reservation was durable and the definite pre-write failure released the
	// row back to pending for at-least-once retry
	if !reserver.reserved {
		t.Error("expected the row reserved before send")
	}
	if len(dlRepo.statusUpdates) != 1 || dlRepo.statusUpdates[0] != bssci.DLQueueStatusPending {
		t.Errorf("expected release to pending, got %v", dlRepo.statusUpdates)
	}
	if dlRepo.markQueuedCalls != 0 {
		t.Errorf("expected no markQueued call, got %d", dlRepo.markQueuedCalls)
	}
}

func TestDispatchIfAvailable_AmbiguousSendError_StaysReserved(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		ID: 9, QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	sendFn.err = errTestWritePayloadAmbiguous
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)

	if !errors.Is(err, bssci.ErrAmbiguousWrite) {
		t.Fatalf("expected ambiguous-write error, got %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on ambiguous send")
	}
	// An uncertain send must never return the row to plain pending: a retry
	// would mint a replacement operation ID for a possibly delivered frame
	if len(dlRepo.statusUpdates) != 0 {
		t.Errorf("expected row to stay reserved, got status updates %v", dlRepo.statusUpdates)
	}
	if dlRepo.markQueuedCalls != 0 {
		t.Errorf("expected no markQueued call, got %d", dlRepo.markQueuedCalls)
	}
}

func TestDispatchIfAvailable_MarkQueuedError_ReportsDispatched(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	dlRepo.markQueuedErr = errTestMarkQueuedFailed
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	// The send happened: the dispatch is reported and the row stays reserved
	// until the idempotent dlDataQueRsp confirmation repairs it
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dispatched {
		t.Error("expected dispatched=true when the send succeeded")
	}
	if len(dlRepo.statusUpdates) != 0 {
		t.Errorf("expected no release, got status updates %v", dlRepo.statusUpdates)
	}
}

func TestDispatchIfAvailable_UsesUserDataIfPresent(t *testing.T) {
	userData := [][]byte{[]byte("packet1"), []byte("packet2")}
	dlRepo, reserver, _ := newDispatchFixture(&storage.DownlinkMessage{
		QueID:    12345,
		Payload:  []byte("should be ignored"),
		UserData: userData,
		Priority: 1.0,
	})

	var capturedPayloads [][]byte
	sendFn := func(_ string, _ uint64, payloads [][]byte, _ int64, _ float32, _ bool, _ []int64, _ uint8, _, _, _, _ bool, _ int64, _ *uuid.UUID, _ bool) error {
		capturedPayloads = payloads
		return nil
	}

	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dispatched {
		t.Error("expected dispatched=true")
	}

	// Verify UserData was used, not Payload
	if len(capturedPayloads) != 2 {
		t.Fatalf("expected 2 payloads (from UserData), got %d", len(capturedPayloads))
	}
	if string(capturedPayloads[0]) != "packet1" {
		t.Errorf("expected first payload 'packet1', got '%s'", string(capturedPayloads[0]))
	}
}

func TestDispatchQueue_Success(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 777, Payload: []byte("test"), Priority: 1.0, DlRxStatQry: true,
	})
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)
	enqueueOrg := uuid.New()

	dispatched, err := dispatcher.DispatchQueue(
		testutil.TestContext(), 42, enqueueOrg, testDispatchSession(), 777, 0xAABBCCDDEEFF0011,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dispatched {
		t.Error("expected dispatched=true")
	}
	if sendFn.calls != 1 || !sendFn.dlRxStatQry[0] {
		t.Errorf("expected 1 send with dlRxStatQry=true, got calls=%d flags=%v", sendFn.calls, sendFn.dlRxStatQry)
	}
	if dlRepo.markQueuedCalls != 1 {
		t.Errorf("expected 1 markQueued call, got %d", dlRepo.markQueuedCalls)
	}
	// Exact-match reservation arguments, scoped to the enqueuing organization
	if len(reserver.reserveByQueueCalls) != 1 {
		t.Fatalf("expected 1 reserve-by-queue call, got %d", len(reserver.reserveByQueueCalls))
	}
	call := reserver.reserveByQueueCalls[0]
	if call.tenantID != 42 || call.queueID != 777 || call.bsEUI != 0x1234567890ABCDEF {
		t.Errorf("unexpected reservation args: %+v", call)
	}
	if call.orgID != enqueueOrg {
		t.Errorf("reservation must be scoped to the enqueuing organization, got %v", call.orgID)
	}
}

// TestDispatchQueue_IgnoresSessionOrganization pins the roaming fix: the base
// station session's organization belongs to the serving station's owner, which
// under roaming is a different tenant entirely. The exact-queue reservation
// must be scoped to the enqueuing organization, and the delivery must carry
// the row's organization.
func TestDispatchQueue_IgnoresSessionOrganization(t *testing.T) {
	rowOrg := uuid.New()
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 88, Payload: []byte("x"), OrganizationID: &rowOrg,
	})
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	session := testDispatchSession()
	session.OrganizationID = uuid.New() // foreign serving base station's organization

	dispatched, err := dispatcher.DispatchQueue(
		testutil.TestContext(), 42, rowOrg, session, 88, 0xAABBCCDDEEFF0011,
	)

	if err != nil || !dispatched {
		t.Fatalf("dispatch failed: dispatched=%v err=%v", dispatched, err)
	}
	if len(reserver.reserveByQueueCalls) != 1 || reserver.reserveByQueueCalls[0].orgID != rowOrg {
		t.Errorf("reservation must use the enqueuing organization, got %+v", reserver.reserveByQueueCalls)
	}
	if len(sendFn.sentOrgIDs) != 1 || sendFn.sentOrgIDs[0] == nil || *sendFn.sentOrgIDs[0] != rowOrg {
		t.Errorf("the delivery must carry the row's organization, got %v", sendFn.sentOrgIDs)
	}
}

func TestDispatchQueue_NoMatchingPendingRow(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(nil)
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchQueue(
		testutil.TestContext(), 42, uuid.New(), testDispatchSession(), 777, 0xAABBCCDDEEFF0011,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false when no matching pending row")
	}
	if sendFn.calls != 0 {
		t.Errorf("expected 0 sendFn calls, got %d", sendFn.calls)
	}
	if dlRepo.markQueuedCalls != 0 {
		t.Errorf("expected 0 markQueued calls, got %d", dlRepo.markQueuedCalls)
	}
}

func TestDispatchQueue_ReservationError(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(nil)
	reserver.reserveByQueueErr = errTestReserveDBDown
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchQueue(
		testutil.TestContext(), 42, uuid.New(), testDispatchSession(), 777, 0xAABBCCDDEEFF0011,
	)

	if err == nil {
		t.Fatal("expected reservation error to propagate")
	}
	if dispatched {
		t.Error("expected dispatched=false on reservation error")
	}
	if sendFn.calls != 0 {
		t.Errorf("expected 0 sendFn calls, got %d", sendFn.calls)
	}
}

// TestDispatchQueue_FailsClosedWithoutOrganization pins the exact-queue guard:
// a request that cannot name the enqueuing organization must not reserve
// anything.
func TestDispatchQueue_FailsClosedWithoutOrganization(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{QueID: 7, Payload: []byte("x")})
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchQueue(
		testutil.TestContext(), 42, uuid.Nil, testDispatchSession(), 7, 0xAABBCCDDEEFF0011,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched || sendFn.calls != 0 || len(reserver.reserveByQueueCalls) != 0 {
		t.Error("nothing may run without an organization")
	}
}

// TestDispatchIfAvailable_RowOrganizationFlowsThroughLifecycle asserts the
// reserved row's organization is the one the send and the confirmation carry.
func TestDispatchIfAvailable_RowOrganizationFlowsThroughLifecycle(t *testing.T) {
	rowOrg := uuid.New()
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 5, Payload: []byte("x"), OrganizationID: &rowOrg,
	})
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)

	if err != nil || !dispatched {
		t.Fatalf("dispatch failed: dispatched=%v err=%v", dispatched, err)
	}
	if len(sendFn.sentOrgIDs) != 1 || sendFn.sentOrgIDs[0] == nil || *sendFn.sentOrgIDs[0] != rowOrg {
		t.Errorf("the delivery must carry the row's organization, got %v", sendFn.sentOrgIDs)
	}
	if len(dlRepo.markQueuedOrgIDs) != 1 || dlRepo.markQueuedOrgIDs[0] == nil || *dlRepo.markQueuedOrgIDs[0] != rowOrg {
		t.Errorf("confirmation must carry the row's organization, got %v", dlRepo.markQueuedOrgIDs)
	}
}

// TestDispatchReserved_RejectsOwnerlessRow asserts a row that comes back with
// no organization is neither sent nor mutated: enqueue and the schema forbid
// ownerless rows, so one appearing here means the invariant was bypassed and
// the dispatcher stops with zero writes, leaving the row reserved.
func TestDispatchReserved_RejectsOwnerlessRow(t *testing.T) {
	dlRepo := &mockMIOTYDownlinksForDispatch{}
	ownerless := &storage.DownlinkMessage{QueID: 9, Payload: []byte("x")}
	reserver := &mockReserverForDispatch{
		reserveResult:        ownerless,
		reserveByQueueResult: ownerless,
	}
	sendFn := &mockSendFn{reserver: reserver}
	dispatcher := mustDispatcher(t, &mockLoggerForDispatch{}, reserver, dlRepo, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		testutil.TestContext(), 42, testDispatchSession(),
		0xAABBCCDDEEFF0011, dispatchTestMessageID, true,
	)

	if !errors.Is(err, bssci.ErrDispatchOrgMismatch) {
		t.Fatalf("expected ErrDispatchOrgMismatch, got %v", err)
	}
	if dispatched || sendFn.calls != 0 {
		t.Error("an ownerless row must never be sent")
	}
	if len(dlRepo.statusUpdates) != 0 {
		t.Errorf("an ownerless row must not be mutated, got status updates %v", dlRepo.statusUpdates)
	}
	if dlRepo.markQueuedCalls != 0 {
		t.Errorf("an ownerless row must not be confirmed, got %d MarkReservedAsQueued calls", dlRepo.markQueuedCalls)
	}
}
