package bssciservices

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
func (m *mockLoggerForDispatch) WithFields(_ map[string]interface{}) logger.Logger          { return m }

// mockTransactionForDispatch implements interfaces.Transaction for dispatcher tests
type mockTransactionForDispatch struct {
	miotyDownlinks *mockMIOTYDownlinksForDispatch
	committed      bool
	rolledBack     bool
	commitErr      error
}

func (m *mockTransactionForDispatch) Commit() error {
	if m.commitErr != nil {
		return m.commitErr
	}
	m.committed = true
	return nil
}

func (m *mockTransactionForDispatch) Rollback() error {
	m.rolledBack = true
	return nil
}

func (m *mockTransactionForDispatch) EndPoints() interfaces.EndpointRepository          { return nil }
func (m *mockTransactionForDispatch) BaseStations() interfaces.BaseStationRepository    { return nil }
func (m *mockTransactionForDispatch) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (m *mockTransactionForDispatch) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (m *mockTransactionForDispatch) EndPointSessions() interfaces.EndPointSessionRepository {
	return nil
}
func (m *mockTransactionForDispatch) EndPointKeys() interfaces.EndPointKeyRepository { return nil }
func (m *mockTransactionForDispatch) RoamingAgreements() interfaces.RoamingAgreementRepository {
	return nil
}
func (m *mockTransactionForDispatch) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (m *mockTransactionForDispatch) PendingOperations() interfaces.PendingOperationRepository {
	return nil
}
func (m *mockTransactionForDispatch) DLRXStatus() interfaces.DLRXStatusRepository      { return nil }
func (m *mockTransactionForDispatch) MIOTYMessages() interfaces.MIOTYMessageRepository { return nil }
func (m *mockTransactionForDispatch) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (m *mockTransactionForDispatch) Users() interfaces.UserRepository                 { return nil }
func (m *mockTransactionForDispatch) APIKeys() interfaces.APIKeyRepository             { return nil }
func (m *mockTransactionForDispatch) Integrations() interfaces.IntegrationRepository   { return nil }
func (m *mockTransactionForDispatch) Manufacturers() interfaces.ManufacturerRepository { return nil } // Blueprint catalog
func (m *mockTransactionForDispatch) DeviceModels() interfaces.DeviceModelRepository   { return nil } // Blueprint catalog
func (m *mockTransactionForDispatch) Blueprints() interfaces.BlueprintRepository       { return nil } // Blueprint catalog

func (m *mockTransactionForDispatch) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository {
	return m.miotyDownlinks
}

// Additional accessors for Transaction
func (m *mockTransactionForDispatch) Organizations() interfaces.OrganizationRepository { return nil }
func (m *mockTransactionForDispatch) GetSqlxDB() *sqlx.DB                              { return nil }
func (m *mockTransactionForDispatch) SystemEvents() interfaces.SystemEventStore        { return nil }
func (m *mockTransactionForDispatch) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (m *mockTransactionForDispatch) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (m *mockTransactionForDispatch) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }

// mockStorageForDispatch implements interfaces.Storage for dispatcher tests.
// MIOTYDownlinks() exposes the same repository the transaction wraps, matching
// production where the regular repository handles the post-send confirmation.
type mockStorageForDispatch struct {
	tx       *mockTransactionForDispatch
	dlRepo   *mockMIOTYDownlinksForDispatch
	beginErr error
}

func (m *mockStorageForDispatch) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	return m.tx, nil
}

// Stub all other Storage methods (not used by dispatcher)
func (m *mockStorageForDispatch) Close() error                                   { return nil }
func (m *mockStorageForDispatch) Ping(_ context.Context) error                   { return nil }
func (m *mockStorageForDispatch) EndPoints() interfaces.EndpointRepository       { return nil }
func (m *mockStorageForDispatch) BaseStations() interfaces.BaseStationRepository { return nil }
func (m *mockStorageForDispatch) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (m *mockStorageForDispatch) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (m *mockStorageForDispatch) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (m *mockStorageForDispatch) EndPointSessions() interfaces.EndPointSessionRepository { return nil }
func (m *mockStorageForDispatch) EndPointKeys() interfaces.EndPointKeyRepository         { return nil }
func (m *mockStorageForDispatch) RoamingAgreements() interfaces.RoamingAgreementRepository {
	return nil
}
func (m *mockStorageForDispatch) PendingOperations() interfaces.PendingOperationRepository {
	return nil
}
func (m *mockStorageForDispatch) MIOTYMessages() interfaces.MIOTYMessageRepository { return nil }
func (m *mockStorageForDispatch) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository {
	return m.dlRepo
}
func (m *mockStorageForDispatch) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (m *mockStorageForDispatch) DLRXStatus() interfaces.DLRXStatusRepository      { return nil }
func (m *mockStorageForDispatch) Users() interfaces.UserRepository                 { return nil }
func (m *mockStorageForDispatch) APIKeys() interfaces.APIKeyRepository             { return nil }
func (m *mockStorageForDispatch) Integrations() interfaces.IntegrationRepository   { return nil }
func (m *mockStorageForDispatch) Manufacturers() interfaces.ManufacturerRepository { return nil } // Blueprint catalog
func (m *mockStorageForDispatch) DeviceModels() interfaces.DeviceModelRepository   { return nil } // Blueprint catalog
func (m *mockStorageForDispatch) Blueprints() interfaces.BlueprintRepository       { return nil } // Blueprint catalog

// Additional accessors for Storage
func (m *mockStorageForDispatch) Organizations() interfaces.OrganizationRepository     { return nil }
func (m *mockStorageForDispatch) GetSqlxDB() *sqlx.DB                                  { return nil }
func (m *mockStorageForDispatch) SystemEvents() interfaces.SystemEventStore            { return nil }
func (m *mockStorageForDispatch) SCACISessions() interfaces.SCACISessionRepository     { return nil }
func (m *mockStorageForDispatch) SCACIOperations() interfaces.SCACIOperationRepository { return nil }
func (m *mockStorageForDispatch) DownlinkQueueReader() interfaces.DownlinkQueueReader  { return nil }

// reserveByQueueCall captures ReservePendingDownlinkByQueueID arguments
type reserveByQueueCall struct {
	tenantID int64
	orgID    *uuid.UUID
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
	statusUpdates        []string // captured UpdateDownlinkStatus statuses
}

// orgID parameter enables organization-scoped reservation
func (m *mockMIOTYDownlinksForDispatch) ReserveNextPendingDownlink(_ context.Context, _ int64, _ []byte, _ uint64, _ *uuid.UUID) (*storage.DownlinkMessage, error) {
	if m.reserveErr != nil {
		return nil, m.reserveErr
	}
	return m.reserveResult, nil
}

func (m *mockMIOTYDownlinksForDispatch) ReservePendingDownlinkByQueueID(_ context.Context, tenantID int64, orgID *uuid.UUID, queueID uint64, epEUI []byte, bsEUI uint64) (*storage.DownlinkMessage, error) {
	m.reserveByQueueCalls = append(m.reserveByQueueCalls, reserveByQueueCall{
		tenantID: tenantID, orgID: orgID, queueID: queueID, epEUI: epEUI, bsEUI: bsEUI,
	})
	if m.reserveByQueueErr != nil {
		return nil, m.reserveByQueueErr
	}
	return m.reserveByQueueResult, nil
}

// orgID parameter enables organization-scoped queue marking
func (m *mockMIOTYDownlinksForDispatch) MarkReservedAsQueued(_ context.Context, _ uint64, _ int64, _ uint64, _ int64, _ *uint32, _ *uuid.UUID) error {
	m.markQueuedCalls++
	return m.markQueuedErr
}

// Stub remaining interface methods
func (m *mockMIOTYDownlinksForDispatch) GetDownlinkQueue(_ context.Context, _ string, _ string) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}
func (m *mockMIOTYDownlinksForDispatch) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (m *mockMIOTYDownlinksForDispatch) GetDownlinkByPacketCnt(_ context.Context, _ string, _ string, _ uint32) (*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockMIOTYDownlinksForDispatch) GetDownlinkResults(_ context.Context, _ string, _ string, _ *uuid.UUID, _ string, _, _ *time.Time, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return nil, 0, nil
}
func (m *mockMIOTYDownlinksForDispatch) EnqueueDownlink(_ context.Context, _ *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	return nil, nil
}

// UpdateDownlinkStatus captures release-to-pending calls
func (m *mockMIOTYDownlinksForDispatch) UpdateDownlinkStatus(_ context.Context, _ string, status string, _ *uuid.UUID) error {
	m.statusUpdates = append(m.statusUpdates, status)
	return nil
}

// orgID parameter enables organization-scoped result updates
func (m *mockMIOTYDownlinksForDispatch) UpdateDownlinkResult(_ context.Context, _ int64, _ string, _ *int64, _ *uint32, _ []byte, _ []byte, _ string, _ *uuid.UUID) error {
	return nil
}
func (m *mockMIOTYDownlinksForDispatch) UpdateDownlinkBaseStation(_ context.Context, _ uint64, _ string, _ uint64) error {
	return nil
}
func (m *mockMIOTYDownlinksForDispatch) RevokeDownlink(_ context.Context, _ int64, _ string) error {
	return nil
}

// mockSendFn tracks calls to SendDLDataQueue
type mockSendFn struct {
	calls       int
	err         error
	dlRxStatQry []bool // captured dlRxStatQry flag per call
	// txCommittedAtSend records whether the reservation transaction had been
	// committed at the moment the wire send ran
	tx                *mockTransactionForDispatch
	txCommittedAtSend []bool
}

func (m *mockSendFn) Send(_ string, _ uint64, _ [][]byte, _ int64, _ float32, _ bool, _ []int64, _ uint8, _, _, _, _ bool, _ int64, dlRxStatQry bool) error {
	m.calls++
	m.dlRxStatQry = append(m.dlRxStatQry, dlRxStatQry)
	if m.tx != nil {
		m.txCommittedAtSend = append(m.txCommittedAtSend, m.tx.committed)
	}
	return m.err
}

func newDispatchFixture(dl *storage.DownlinkMessage) (*mockMIOTYDownlinksForDispatch, *mockTransactionForDispatch, *mockStorageForDispatch, *mockSendFn) {
	dlRepo := &mockMIOTYDownlinksForDispatch{
		reserveResult:        dl,
		reserveByQueueResult: dl,
	}
	tx := &mockTransactionForDispatch{miotyDownlinks: dlRepo}
	storageM := &mockStorageForDispatch{tx: tx, dlRepo: dlRepo}
	sendFn := &mockSendFn{tx: tx}
	return dlRepo, tx, storageM, sendFn
}

func testDispatchSession() *bssci.Session {
	return &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:             "test-session-123",
			BaseStationEUI: 0x1234567890ABCDEF,
		},
	}
}

func TestDispatchIfAvailable_Success(t *testing.T) {
	dlRepo, tx, storageM, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID:       12345,
		Payload:     []byte("test payload"),
		Priority:    1.0,
		Format:      0,
		ResponseExp: true,
	})

	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

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
	if !tx.committed {
		t.Error("expected reservation transaction to be committed")
	}
	// The reservation transaction must be closed before any wire write
	if len(sendFn.txCommittedAtSend) != 1 || !sendFn.txCommittedAtSend[0] {
		t.Error("expected reservation transaction committed BEFORE the wire send")
	}
}

func TestDispatchIfAvailable_DlRxStatQryPassthrough(t *testing.T) {
	for _, want := range []bool{true, false} {
		_, _, storageM, sendFn := newDispatchFixture(&storage.DownlinkMessage{
			QueID:       7,
			Payload:     []byte("p"),
			DlRxStatQry: want,
		})
		dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

		dispatched, err := dispatcher.DispatchIfAvailable(
			context.Background(), 42, uuid.New(), testDispatchSession(),
			0xAABBCCDDEEFF0011, false, false)
		if err != nil || !dispatched {
			t.Fatalf("dlRxStatQry=%v: unexpected result dispatched=%v err=%v", want, dispatched, err)
		}
		if len(sendFn.dlRxStatQry) != 1 || sendFn.dlRxStatQry[0] != want {
			t.Errorf("expected sendFn dlRxStatQry=%v, got %v", want, sendFn.dlRxStatQry)
		}
	}
}

func TestDispatchIfAvailable_NoPendingDownlinks(t *testing.T) {
	_, tx, storageM, sendFn := newDispatchFixture(nil)
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false when no pending downlinks")
	}
	if sendFn.calls != 0 {
		t.Errorf("expected 0 sendFn calls, got %d", sendFn.calls)
	}
	if !tx.rolledBack {
		t.Error("expected empty reservation transaction to be rolled back")
	}
}

func TestDispatchIfAvailable_NoTenantContext(t *testing.T) {
	storageM := &mockStorageForDispatch{}
	sendFn := &mockSendFn{}
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 0, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false when no tenant context")
	}
	if sendFn.calls != 0 {
		t.Errorf("expected 0 sendFn calls, got %d", sendFn.calls)
	}
}

func TestDispatchIfAvailable_TransactionBeginError(t *testing.T) {
	storageM := &mockStorageForDispatch{beginErr: errors.New("database connection failed")}
	sendFn := &mockSendFn{}
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

	// Should gracefully degrade, not return error (don't fail uplink)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on tx begin error")
	}
}

func TestDispatchIfAvailable_ReservationCommitError(t *testing.T) {
	_, tx, storageM, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	tx.commitErr = errors.New("commit failed")
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

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
	dlRepo, tx, storageM, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		ID: 9, QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	sendFn.err = errors.New("send failed")
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

	if err == nil {
		t.Fatal("expected error on definite send failure")
	}
	if dispatched {
		t.Error("expected dispatched=false on send error")
	}
	// Reservation was durable (committed) and the definite pre-write failure
	// released the row back to pending for at-least-once retry
	if !tx.committed {
		t.Error("expected reservation transaction committed before send")
	}
	if len(dlRepo.statusUpdates) != 1 || dlRepo.statusUpdates[0] != bssci.DLQueueStatusPending {
		t.Errorf("expected release to pending, got %v", dlRepo.statusUpdates)
	}
	if dlRepo.markQueuedCalls != 0 {
		t.Errorf("expected no markQueued call, got %d", dlRepo.markQueuedCalls)
	}
}

func TestDispatchIfAvailable_AmbiguousSendError_StaysReserved(t *testing.T) {
	dlRepo, _, storageM, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		ID: 9, QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	sendFn.err = fmt.Errorf("write payload: %w", bssci.ErrAmbiguousWrite)
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

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
	dlRepo, _, storageM, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 12345, Payload: []byte("test"), Priority: 1.0,
	})
	dlRepo.markQueuedErr = errors.New("mark queued failed")
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

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
	_, _, storageM, _ := newDispatchFixture(&storage.DownlinkMessage{
		QueID:    12345,
		Payload:  []byte("should be ignored"),
		UserData: userData,
		Priority: 1.0,
	})

	var capturedPayloads [][]byte
	sendFn := func(_ string, _ uint64, payloads [][]byte, _ int64, _ float32, _ bool, _ []int64, _ uint8, _, _, _, _ bool, _ int64, _ bool) error {
		capturedPayloads = payloads
		return nil
	}

	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn)

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(), 42, uuid.New(), testDispatchSession(),
		0xAABBCCDDEEFF0011, true, false)

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
	dlRepo, _, storageM, sendFn := newDispatchFixture(&storage.DownlinkMessage{
		QueID: 777, Payload: []byte("test"), Priority: 1.0, DlRxStatQry: true,
	})
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchQueue(
		context.Background(), 42, uuid.New(), testDispatchSession(), 777, 0xAABBCCDDEEFF0011)

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
	// Exact-match reservation arguments
	if len(dlRepo.reserveByQueueCalls) != 1 {
		t.Fatalf("expected 1 reserve-by-queue call, got %d", len(dlRepo.reserveByQueueCalls))
	}
	call := dlRepo.reserveByQueueCalls[0]
	if call.tenantID != 42 || call.queueID != 777 || call.bsEUI != 0x1234567890ABCDEF {
		t.Errorf("unexpected reservation args: %+v", call)
	}
}

func TestDispatchQueue_NoMatchingPendingRow(t *testing.T) {
	dlRepo, _, storageM, sendFn := newDispatchFixture(nil)
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchQueue(
		context.Background(), 42, uuid.New(), testDispatchSession(), 777, 0xAABBCCDDEEFF0011)

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
	dlRepo, _, storageM, sendFn := newDispatchFixture(nil)
	dlRepo.reserveByQueueErr = errors.New("db down")
	dispatcher := NewDownlinkDispatcher(&mockLoggerForDispatch{}, storageM, sendFn.Send)

	dispatched, err := dispatcher.DispatchQueue(
		context.Background(), 42, uuid.New(), testDispatchSession(), 777, 0xAABBCCDDEEFF0011)

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
