package bssciservices

import (
	"context"
	"errors"
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
	m.committed = true
	return m.commitErr
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

// mockStorageForDispatch implements interfaces.Storage for dispatcher tests
type mockStorageForDispatch struct {
	tx       *mockTransactionForDispatch
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
func (m *mockStorageForDispatch) MIOTYMessages() interfaces.MIOTYMessageRepository   { return nil }
func (m *mockStorageForDispatch) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
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

// mockMIOTYDownlinksForDispatch implements transaction-specific methods for testing
type mockMIOTYDownlinksForDispatch struct {
	reserveResult   *storage.DownlinkMessage
	reserveErr      error
	markQueuedErr   error
	markQueuedCalls int
}

// orgID parameter enables organization-scoped reservation
func (m *mockMIOTYDownlinksForDispatch) ReserveNextPendingDownlink(_ context.Context, _ int64, _ []byte, _ uint64, _ *uuid.UUID) (*storage.DownlinkMessage, error) {
	if m.reserveErr != nil {
		return nil, m.reserveErr
	}
	return m.reserveResult, nil
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

// orgID parameter enables organization-scoped status updates
func (m *mockMIOTYDownlinksForDispatch) UpdateDownlinkStatus(_ context.Context, _ string, _ string, _ *uuid.UUID) error {
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
	calls int
	err   error
}

func (m *mockSendFn) Send(_ string, _ uint64, _ [][]byte, _ int64, _ float32, _ bool, _ []int64, _ uint8, _, _, _, _ bool, _ int64) error {
	m.calls++
	return m.err
}

func TestDispatchIfAvailable_Success(t *testing.T) {
	dlRepo := &mockMIOTYDownlinksForDispatch{
		reserveResult: &storage.DownlinkMessage{
			QueID:       12345,
			Payload:     []byte("test payload"),
			Priority:    1.0,
			Format:      0,
			ResponseExp: true,
		},
	}
	tx := &mockTransactionForDispatch{
		miotyDownlinks: dlRepo,
	}
	storageM := &mockStorageForDispatch{tx: tx}
	sendFn := &mockSendFn{}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn.Send,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:             "test-session-123",
			BaseStationEUI: 0x1234567890ABCDEF,
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		42,         // ownerTenantID
		uuid.New(), // ownerOrgUUID
		session,
		0xAABBCCDDEEFF0011, // epEUI
		true,               // responseExp
		false,              // dlAck
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
	if !tx.committed {
		t.Error("expected transaction to be committed")
	}
}

func TestDispatchIfAvailable_NoPendingDownlinks(t *testing.T) {
	dlRepo := &mockMIOTYDownlinksForDispatch{
		reserveResult: nil, // No pending downlinks
	}
	tx := &mockTransactionForDispatch{
		miotyDownlinks: dlRepo,
	}
	storageM := &mockStorageForDispatch{tx: tx}
	sendFn := &mockSendFn{}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn.Send,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "test-session",
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		42,
		uuid.New(),
		session,
		0xAABBCCDDEEFF0011,
		true,
		false,
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
}

func TestDispatchIfAvailable_NoTenantContext(t *testing.T) {
	storageM := &mockStorageForDispatch{}
	sendFn := &mockSendFn{}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn.Send,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "test-session",
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		0, // Zero tenant ID - should skip
		uuid.New(),
		session,
		0xAABBCCDDEEFF0011,
		true,
		false,
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
}

func TestDispatchIfAvailable_TransactionBeginError(t *testing.T) {
	storageM := &mockStorageForDispatch{
		beginErr: errors.New("database connection failed"),
	}
	sendFn := &mockSendFn{}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn.Send,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "test-session",
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		42,
		uuid.New(),
		session,
		0xAABBCCDDEEFF0011,
		true,
		false,
	)

	// Should gracefully degrade, not return error (don't fail uplink)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on tx begin error")
	}
}

func TestDispatchIfAvailable_SendFunctionError(t *testing.T) {
	dlRepo := &mockMIOTYDownlinksForDispatch{
		reserveResult: &storage.DownlinkMessage{
			QueID:    12345,
			Payload:  []byte("test"),
			Priority: 1.0,
		},
	}
	tx := &mockTransactionForDispatch{
		miotyDownlinks: dlRepo,
	}
	storageM := &mockStorageForDispatch{tx: tx}
	sendFn := &mockSendFn{
		err: errors.New("send failed"),
	}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn.Send,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "test-session",
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		42,
		uuid.New(),
		session,
		0xAABBCCDDEEFF0011,
		true,
		false,
	)

	// Should gracefully degrade
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on send error")
	}
	// Transaction should have been rolled back (defer)
	if tx.committed {
		t.Error("expected transaction NOT to be committed on send error")
	}
}

func TestDispatchIfAvailable_MarkQueuedError(t *testing.T) {
	dlRepo := &mockMIOTYDownlinksForDispatch{
		reserveResult: &storage.DownlinkMessage{
			QueID:    12345,
			Payload:  []byte("test"),
			Priority: 1.0,
		},
		markQueuedErr: errors.New("mark queued failed"),
	}
	tx := &mockTransactionForDispatch{
		miotyDownlinks: dlRepo,
	}
	storageM := &mockStorageForDispatch{tx: tx}
	sendFn := &mockSendFn{}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn.Send,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "test-session",
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		42,
		uuid.New(),
		session,
		0xAABBCCDDEEFF0011,
		true,
		false,
	)

	// Should gracefully degrade
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on mark queued error")
	}
	// Transaction should NOT be committed on mark queued error
	if tx.committed {
		t.Error("expected transaction NOT to be committed on mark queued error")
	}
}

func TestDispatchIfAvailable_CommitError(t *testing.T) {
	dlRepo := &mockMIOTYDownlinksForDispatch{
		reserveResult: &storage.DownlinkMessage{
			QueID:    12345,
			Payload:  []byte("test"),
			Priority: 1.0,
		},
	}
	tx := &mockTransactionForDispatch{
		miotyDownlinks: dlRepo,
		commitErr:      errors.New("commit failed"),
	}
	storageM := &mockStorageForDispatch{tx: tx}
	sendFn := &mockSendFn{}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn.Send,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "test-session",
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		42,
		uuid.New(),
		session,
		0xAABBCCDDEEFF0011,
		true,
		false,
	)

	// Should gracefully degrade
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatched {
		t.Error("expected dispatched=false on commit error")
	}
}

func TestDispatchIfAvailable_UsesUserDataIfPresent(t *testing.T) {
	userData := [][]byte{[]byte("packet1"), []byte("packet2")}
	dlRepo := &mockMIOTYDownlinksForDispatch{
		reserveResult: &storage.DownlinkMessage{
			QueID:    12345,
			Payload:  []byte("should be ignored"),
			UserData: userData,
			Priority: 1.0,
		},
	}
	tx := &mockTransactionForDispatch{
		miotyDownlinks: dlRepo,
	}
	storageM := &mockStorageForDispatch{tx: tx}

	var capturedPayloads [][]byte
	sendFn := func(_ string, _ uint64, payloads [][]byte, _ int64, _ float32, _ bool, _ []int64, _ uint8, _, _, _, _ bool, _ int64) error {
		capturedPayloads = payloads
		return nil
	}

	dispatcher := NewDownlinkDispatcher(
		&mockLoggerForDispatch{},
		storageM,
		sendFn,
	)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "test-session",
		},
	}

	dispatched, err := dispatcher.DispatchIfAvailable(
		context.Background(),
		42,
		uuid.New(),
		session,
		0xAABBCCDDEEFF0011,
		true,
		false,
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
