package scaciservices

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// mockOperationRepo implements interfaces.SCACIOperationRepository for testing
type mockOperationRepo struct {
	lastRequest *models.SCACIOperationRequest
	err         error
}

func (m *mockOperationRepo) RecordOperation(_ context.Context, req *models.SCACIOperationRequest) (*models.SCACIOperation, error) {
	m.lastRequest = req
	if m.err != nil {
		return nil, m.err
	}
	// Return a minimal operation record
	return &models.SCACIOperation{
		ID:          1,
		SessionID:   req.SessionID,
		TenantID:    req.TenantID,
		OpId:        req.OpId,
		Command:     req.Command,
		Direction:   req.Direction,
		RequestData: req.RequestData,
	}, nil
}

func (m *mockOperationRepo) UpdateOperationState(_ context.Context, _ int64, _ int64, _ models.OperationState, _ map[string]interface{}) error {
	return nil
}

func (m *mockOperationRepo) GetOperationByOpID(_ context.Context, _ int64, _ int64) (*models.SCACIOperation, error) {
	return nil, nil
}

func (m *mockOperationRepo) GetPendingOperations(_ context.Context, _ int64) ([]*models.SCACIOperation, error) {
	return nil, nil
}

func (m *mockOperationRepo) GetRecentOperations(_ context.Context, _ int64, _ int) ([]*models.SCACIOperation, error) {
	return nil, nil
}

func (m *mockOperationRepo) CleanupCompletedOperations(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (m *mockOperationRepo) GetTenantOperationSummary(_ context.Context, _ int64, _ int) (*models.SCACIOperationSummary, error) {
	return nil, nil
}

func (m *mockOperationRepo) UpdateOperationStateWithError(_ context.Context, _ int64, _ int64, _ models.OperationState, _ int, _ string, _ string, _ map[string]interface{}) error {
	return nil
}

func (m *mockOperationRepo) CompleteFailedOperation(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	return nil
}

func TestRecordInboundOperation(t *testing.T) {
	repo := &mockOperationRepo{}
	recorder := NewOperationRecorder(repo)

	session := &scaci.Session{
		ID:       100,
		TenantID: 42,
		AcEui:    0x123456789ABCDEF0,
	}

	opId := int64(500)
	command := "reg"
	direction := models.OperationDirectionInbound
	data := map[string]interface{}{
		"epEui": uint64(0xAABBCCDDEEFF1122), // Numeric uint64 for propagation lookup
		"bidi":  true,
	}

	err := recorder.Record(testutil.TestContext(), session, opId, command, direction, data)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Verify operation request was created correctly
	if repo.lastRequest == nil {
		t.Fatal("expected operation request to be recorded")
	}

	if repo.lastRequest.SessionID != session.ID {
		t.Errorf("expected SessionID=%d, got %d", session.ID, repo.lastRequest.SessionID)
	}

	if repo.lastRequest.TenantID != session.TenantID {
		t.Errorf("expected TenantID=%d, got %d", session.TenantID, repo.lastRequest.TenantID)
	}

	if repo.lastRequest.OpId != opId {
		t.Errorf("expected OpId=%d, got %d", opId, repo.lastRequest.OpId)
	}

	if repo.lastRequest.Command != command {
		t.Errorf("expected Command=%s, got %s", command, repo.lastRequest.Command)
	}

	if repo.lastRequest.Direction != string(direction) {
		t.Errorf("expected Direction=%s, got %s", direction, repo.lastRequest.Direction)
	}

	// Verify direction is using constant, not literal
	if repo.lastRequest.Direction != "inbound" {
		t.Errorf("expected direction='inbound', got %s", repo.lastRequest.Direction)
	}

	if len(repo.lastRequest.RequestData) != 2 {
		t.Errorf("expected 2 data fields, got %d", len(repo.lastRequest.RequestData))
	}

	if repo.lastRequest.RequestData["epEui"] != uint64(0xAABBCCDDEEFF1122) {
		t.Errorf("unexpected epEui: %v (expected uint64)", repo.lastRequest.RequestData["epEui"])
	}

	if repo.lastRequest.RequestData["bidi"] != true {
		t.Errorf("unexpected bidi: %v", repo.lastRequest.RequestData["bidi"])
	}
}

func TestRecordOutboundOperation(t *testing.T) {
	repo := &mockOperationRepo{}
	recorder := NewOperationRecorder(repo)

	session := &scaci.Session{
		ID:       200,
		TenantID: 99,
		AcEui:    0xAABBCCDDEEFF1122,
	}

	opId := int64(-600) // Negative for SC-initiated operations
	command := "ulData"
	direction := models.OperationDirectionOutbound
	data := map[string]interface{}{
		"epEui":    "1122334455667788",
		"userData": "base64encodeddata",
	}

	err := recorder.Record(testutil.TestContext(), session, opId, command, direction, data)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Verify operation request was created correctly
	if repo.lastRequest == nil {
		t.Fatal("expected operation request to be recorded")
	}

	if repo.lastRequest.OpId != opId {
		t.Errorf("expected OpId=%d, got %d", opId, repo.lastRequest.OpId)
	}

	if repo.lastRequest.Command != command {
		t.Errorf("expected Command=%s, got %s", command, repo.lastRequest.Command)
	}

	// Verify direction is using constant, not literal
	if repo.lastRequest.Direction != "outbound" {
		t.Errorf("expected direction='outbound', got %s", repo.lastRequest.Direction)
	}
}

func TestRecordOperationRepoError(t *testing.T) {
	repo := &mockOperationRepo{
		err: context.DeadlineExceeded,
	}
	recorder := NewOperationRecorder(repo)

	session := &scaci.Session{
		ID:       300,
		TenantID: 50,
	}

	err := recorder.Record(testutil.TestContext(), session, 700, "dereg", models.OperationDirectionInbound, nil)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}
}

func TestRecordOperationWithEmptyData(t *testing.T) {
	repo := &mockOperationRepo{}
	recorder := NewOperationRecorder(repo)

	session := &scaci.Session{
		ID:       400,
		TenantID: 80,
	}

	// Some operations may have no request data (e.g., dereg with just epEui in URL)
	err := recorder.Record(testutil.TestContext(), session, 800, "dereg", models.OperationDirectionInbound, nil)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if len(repo.lastRequest.RequestData) > 0 {
		t.Errorf("expected empty/nil data, got %v", repo.lastRequest.RequestData)
	}
}
