package bssciservices

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// mockEventStore implements interfaces.SystemEventStore for testing
type mockEventStore struct {
	lastEvent *models.SystemEvent
	err       error
}

func (m *mockEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	m.lastEvent = event
	return m.err
}

func (m *mockEventStore) GetEvents(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetActiveAlerts(_ context.Context, _ interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return nil, nil
}

func (m *mockEventStore) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}

func (m *mockEventStore) CountEvents(_ context.Context, _ interfaces.SystemEventFilter) (int64, error) {
	return 0, nil
}

func (m *mockEventStore) CountActiveAlerts(_ context.Context, _ interfaces.AlertFilter) (int64, error) {
	return 0, nil
}

func TestRecordQueueAck(t *testing.T) {
	store := &mockEventStore{}
	logger := NewAuditLogger(store)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: 0x123456789ABCDEF0,
		},
		UserProvidedName: "test-basestation",
	}

	tenant := "42"
	epEui := uint64(0xAABBCCDDEEFF1122)
	queueID := int64(999888)
	opId := int64(-12345)

	err := logger.RecordQueueAck(testutil.TestContext(), tenant, session, epEui, queueID, opId)
	if err != nil {
		t.Fatalf("RecordQueueAck failed: %v", err)
	}

	// Verify event was created
	if store.lastEvent == nil {
		t.Fatal("expected event to be created")
	}

	// Verify event fields per BSSCI §5.12
	if store.lastEvent.TenantID != tenant {
		t.Errorf("expected TenantID=%s, got %s", tenant, store.lastEvent.TenantID)
	}

	if store.lastEvent.EventType != "dl_data_queue_acknowledged" {
		t.Errorf("expected EventType=dl_data_queue_acknowledged, got %s", store.lastEvent.EventType)
	}

	if store.lastEvent.Category != "message" {
		t.Errorf("expected Category=message, got %s", store.lastEvent.Category)
	}

	if store.lastEvent.Severity != "info" {
		t.Errorf("expected Severity=info, got %s", store.lastEvent.Severity)
	}

	if store.lastEvent.SourceType != "endpoint" {
		t.Errorf("expected SourceType=endpoint, got %s", store.lastEvent.SourceType)
	}

	expectedSourceName := "AABBCCDDEEFF1122"
	if store.lastEvent.SourceName != expectedSourceName {
		t.Errorf("expected SourceName=%s, got %s", expectedSourceName, store.lastEvent.SourceName)
	}

	// Verify title includes base station name
	if store.lastEvent.Title != "DL Data Queued - test-basestation" {
		t.Errorf("unexpected Title: %s", store.lastEvent.Title)
	}
}

func TestRecordDLResultSent(t *testing.T) {
	store := &mockEventStore{}
	logger := NewAuditLogger(store)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: 0x1111222233334444,
		},
		UserProvidedName: "bs-success",
	}

	tenant := "100"
	result := &mioty.DLDataResult{
		QueId:  uint64(555666),
		EpEui:  uint64(0x9988776655443322),
		Result: mioty.ResultSent, // "sent"
	}

	err := logger.RecordDLResult(testutil.TestContext(), tenant, session, result)
	if err != nil {
		t.Fatalf("RecordDLResult failed: %v", err)
	}

	// Verify event was created
	if store.lastEvent == nil {
		t.Fatal("expected event to be created")
	}

	// Verify event fields for successful transmission per BSSCI §5.14
	if store.lastEvent.EventType != "dl_data_sent" {
		t.Errorf("expected EventType=dl_data_sent, got %s", store.lastEvent.EventType)
	}

	if store.lastEvent.Severity != "info" {
		t.Errorf("expected Severity=info for sent, got %s", store.lastEvent.Severity)
	}

	if store.lastEvent.Title != "DL Data Sent - bs-success" {
		t.Errorf("unexpected Title: %s", store.lastEvent.Title)
	}

	expectedSourceName := "9988776655443322"
	if store.lastEvent.SourceName != expectedSourceName {
		t.Errorf("expected SourceName=%s, got %s", expectedSourceName, store.lastEvent.SourceName)
	}
}

func TestRecordDLResultExpired(t *testing.T) {
	store := &mockEventStore{}
	logger := NewAuditLogger(store)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: 0x5555666677778888,
		},
		UserProvidedName: "bs-expired",
	}

	tenant := "200"
	result := &mioty.DLDataResult{
		QueId:  uint64(777888),
		EpEui:  uint64(0x1122334455667788),
		Result: mioty.ResultExpired, // "expired"
	}

	err := logger.RecordDLResult(testutil.TestContext(), tenant, session, result)
	if err != nil {
		t.Fatalf("RecordDLResult failed: %v", err)
	}

	// Verify event was created
	if store.lastEvent == nil {
		t.Fatal("expected event to be created")
	}

	// Verify event fields for expired transmission per BSSCI §5.14
	if store.lastEvent.EventType != "dl_data_expired" {
		t.Errorf("expected EventType=dl_data_expired, got %s", store.lastEvent.EventType)
	}

	if store.lastEvent.Severity != "warning" {
		t.Errorf("expected Severity=warning for expired, got %s", store.lastEvent.Severity)
	}

	if store.lastEvent.Title != "DL Data Expired - bs-expired" {
		t.Errorf("unexpected Title: %s", store.lastEvent.Title)
	}

	expectedSourceName := "1122334455667788"
	if store.lastEvent.SourceName != expectedSourceName {
		t.Errorf("expected SourceName=%s, got %s", expectedSourceName, store.lastEvent.SourceName)
	}
}

func TestRecordDLResultInvalid(t *testing.T) {
	store := &mockEventStore{}
	logger := NewAuditLogger(store)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: 0x9999AAAABBBBCCCC,
		},
		UserProvidedName: "bs-invalid",
	}

	tenant := "300"
	result := &mioty.DLDataResult{
		QueId:  uint64(888999),
		EpEui:  uint64(0x2233445566778899),
		Result: mioty.ResultInvalid, // "invalid"
	}

	err := logger.RecordDLResult(testutil.TestContext(), tenant, session, result)
	if err != nil {
		t.Fatalf("RecordDLResult failed: %v", err)
	}

	// Verify event was created
	if store.lastEvent == nil {
		t.Fatal("expected event to be created")
	}

	// Verify event fields for invalid transmission per BSSCI §5.14
	if store.lastEvent.EventType != "dl_data_invalid" {
		t.Errorf("expected EventType=dl_data_invalid, got %s", store.lastEvent.EventType)
	}

	if store.lastEvent.Severity != "error" {
		t.Errorf("expected Severity=error for invalid, got %s", store.lastEvent.Severity)
	}

	if store.lastEvent.Title != "DL Data Invalid - bs-invalid" {
		t.Errorf("unexpected Title: %s", store.lastEvent.Title)
	}

	expectedSourceName := "2233445566778899"
	if store.lastEvent.SourceName != expectedSourceName {
		t.Errorf("expected SourceName=%s, got %s", expectedSourceName, store.lastEvent.SourceName)
	}
}

func TestRecordQueueAckStoreError(t *testing.T) {
	store := &mockEventStore{
		err: context.DeadlineExceeded,
	}
	logger := NewAuditLogger(store)

	session := &bssci.Session{
		UserProvidedName: "test-bs",
	}

	err := logger.RecordQueueAck(testutil.TestContext(), "1", session, 123, 456, -789)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}
}
