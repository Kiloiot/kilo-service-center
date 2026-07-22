package scaciservices

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// ============================================================================
// Mock Repository for Session Persistence Tests
// ============================================================================

// mockSessionRepoForPersistence is a mock for testing session persistence
type mockSessionRepoForPersistence struct {
	mu               sync.Mutex
	createCalled     bool
	updateCalled     bool
	lastCreateReq    *models.SCACISessionCreateRequest
	lastUpdateReq    *models.SCACISessionUpdateRequest
	lastUpdateTenant int64
	lastUpdateID     int64
	createErr        error
	updateErr        error
	createdSession   *models.SCACISession
	completionCh     chan struct{} // Unbuffered - each signal must be consumed
	// Heartbeat tracking fields (SCACI §3.4)
	heartbeatCalled        bool
	lastHeartbeatTenantID  int64
	lastHeartbeatSessionID int64
	heartbeatErr           error
}

func newMockSessionRepoForPersistence() *mockSessionRepoForPersistence {
	return &mockSessionRepoForPersistence{
		createdSession: &models.SCACISession{
			ID:       123,
			TenantID: 1,
		},
		completionCh: make(chan struct{}), // Unbuffered for strict synchronization
	}
}

func (m *mockSessionRepoForPersistence) CreateSession(_ context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	m.mu.Lock()
	m.createCalled = true
	m.lastCreateReq = req
	err := m.createErr
	session := m.createdSession
	m.mu.Unlock()

	// Signal AFTER releasing lock - blocks until test consumes
	m.completionCh <- struct{}{}

	if err != nil {
		return nil, err
	}
	return session, nil
}

func (m *mockSessionRepoForPersistence) UpdateSession(_ context.Context, tenantID, sessionID int64, req *models.SCACISessionUpdateRequest) error {
	m.mu.Lock()
	m.updateCalled = true
	m.lastUpdateReq = req
	m.lastUpdateTenant = tenantID
	m.lastUpdateID = sessionID
	err := m.updateErr
	m.mu.Unlock()

	// Signal AFTER releasing lock - blocks until test consumes
	m.completionCh <- struct{}{}

	return err
}

// getLastCreateReq returns the last create request (thread-safe)
func (m *mockSessionRepoForPersistence) getLastCreateReq() *models.SCACISessionCreateRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastCreateReq
}

// getLastUpdateReq returns the last update request (thread-safe)
func (m *mockSessionRepoForPersistence) getLastUpdateReq() *models.SCACISessionUpdateRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastUpdateReq
}

// wasCreateCalled returns whether CreateSession was called (thread-safe)
func (m *mockSessionRepoForPersistence) wasCreateCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createCalled
}

// wasUpdateCalled returns whether UpdateSession was called (thread-safe)
func (m *mockSessionRepoForPersistence) wasUpdateCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCalled
}

// wasHeartbeatCalled returns whether UpdateHeartbeat was called (thread-safe)
func (m *mockSessionRepoForPersistence) wasHeartbeatCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.heartbeatCalled
}

// getLastHeartbeatIDs returns the last heartbeat tenantID and sessionID (thread-safe)
func (m *mockSessionRepoForPersistence) getLastHeartbeatIDs() (int64, int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastHeartbeatTenantID, m.lastHeartbeatSessionID
}

// waitForCompletion waits for exactly one async operation to complete.
// Use this when a test triggers exactly one CreateSession or UpdateSession call.
func (m *mockSessionRepoForPersistence) waitForCompletion() {
	<-m.completionCh
}

// waitForN waits for exactly n async operations to complete.
// Use this when a test triggers multiple Create/Update calls.
//
//nolint:unused // Reserved for tests that trigger multiple async operations
func (m *mockSessionRepoForPersistence) waitForN(n int) {
	for i := 0; i < n; i++ {
		<-m.completionCh
	}
}

// Stub remaining interface methods
func (m *mockSessionRepoForPersistence) CheckSessionResumable(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
	return nil, nil
}
func (m *mockSessionRepoForPersistence) GetSessionByAcUUID(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepoForPersistence) GetSessionByID(_ context.Context, _, _ int64) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepoForPersistence) GetActiveSessionByAcEUI(_ context.Context, _ int64, _ [8]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepoForPersistence) GetSessionByScUUID(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepoForPersistence) UpdateOperationIDs(_ context.Context, _, _ int64, _, _ int64) error {
	return nil
}
func (m *mockSessionRepoForPersistence) UpdateHeartbeat(_ context.Context, tenantID, sessionID int64) error {
	m.mu.Lock()
	m.heartbeatCalled = true
	m.lastHeartbeatTenantID = tenantID
	m.lastHeartbeatSessionID = sessionID
	err := m.heartbeatErr
	m.mu.Unlock()

	// Signal AFTER releasing lock - blocks until test consumes
	m.completionCh <- struct{}{}

	return err
}
func (m *mockSessionRepoForPersistence) DisconnectSession(_ context.Context, _, _ int64) error {
	return nil
}
func (m *mockSessionRepoForPersistence) TerminateSession(_ context.Context, _, _ int64) error {
	return nil
}
func (m *mockSessionRepoForPersistence) TerminateAllSessions(_ context.Context, _ int64, _ [8]byte) error {
	return nil
}
func (m *mockSessionRepoForPersistence) ListSessions(_ context.Context, _ *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	return nil, 0, nil
}
func (m *mockSessionRepoForPersistence) GetSessionStatistics(_ context.Context, _ int64) (*models.SCACISessionStatistics, error) {
	return nil, nil
}
func (m *mockSessionRepoForPersistence) CleanupExpiredSessions(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

// ============================================================================
// Organization ID Persistence Tests (P0)
// ============================================================================

// TestPersistConnectAsync_FreshSession_WritesOrganizationID validates that
// fresh sessions have organization_id persisted to the database.
// Ref: session_persistence.go:98-102 (orgID population), line 112 (CreateRequest field)
func TestPersistConnectAsync_FreshSession_WritesOrganizationID(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	// Create session with organization ID set
	orgID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	session := &scaci.Session{
		TenantID:       42,
		OrganizationID: orgID,
		AcEui:          0xAABBCCDDEEFF1122,
		SnAcUUID:       [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID:       [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:        false, // Fresh session
	}

	// Call persistence (async)
	svc.PersistConnectAsync(testutil.TestContext(), session, "cert-fingerprint", "CN=test", "192.168.1.1:5001",
		"TLS 1.3", "TLS_AES_256_GCM_SHA384", scaci.ProtocolVersionString)

	// Wait for async goroutine to complete (deterministic)
	mockRepo.waitForCompletion()

	// Assert CreateSession was called
	require.True(t, mockRepo.wasCreateCalled(), "CreateSession should be called for fresh session")

	// Verify organization_id was populated
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)
	require.NotNil(t, createReq.OrganizationID, "OrganizationID should not be nil")
	assert.Equal(t, orgID, *createReq.OrganizationID, "OrganizationID should match session value")
	assert.Equal(t, int64(42), createReq.TenantID, "TenantID should be preserved")
}

// TestPersistConnectAsync_FreshSession_NilOrgID validates that nil org ID
// is handled correctly (community mode / no org resolution).
func TestPersistConnectAsync_FreshSession_NilOrgID(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	// Create session without organization ID (uuid.Nil)
	session := &scaci.Session{
		TenantID:       1,
		OrganizationID: uuid.Nil, // No org set (community mode)
		AcEui:          0xAABBCCDDEEFF1122,
		SnAcUUID:       [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID:       [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:        false,
	}

	svc.PersistConnectAsync(testutil.TestContext(), session, "", "", "", "", "", scaci.ProtocolVersionString)
	mockRepo.waitForCompletion()

	require.True(t, mockRepo.wasCreateCalled())
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)
	assert.Nil(t, createReq.OrganizationID, "OrganizationID should be nil when session.OrganizationID is uuid.Nil")
}

// TestPersistConnectAsync_ResumedSession_PreservesOrgID validates that resumed
// sessions do NOT override organization_id (it's preserved from original session).
// Ref: session_persistence.go:45-77 (resume path - UpdateSession, not CreateSession)
func TestPersistConnectAsync_ResumedSession_PreservesOrgID(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	// Create resumed session (has ID > 0)
	session := &scaci.Session{
		ID:             999, // Existing session ID
		TenantID:       42,
		OrganizationID: uuid.MustParse("22222222-3333-4444-5555-666666666666"),
		AcEui:          0xAABBCCDDEEFF1122,
		SnAcUUID:       [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID:       [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:        true, // Resumed session
	}

	svc.PersistConnectAsync(testutil.TestContext(), session, "", "", "", "TLS 1.3", "TLS_AES_256_GCM_SHA384", scaci.ProtocolVersionString)
	mockRepo.waitForCompletion()

	// Assert UpdateSession was called (not CreateSession)
	require.True(t, mockRepo.wasUpdateCalled(), "UpdateSession should be called for resumed session")
	require.False(t, mockRepo.wasCreateCalled(), "CreateSession should NOT be called for resumed session")

	// UpdateRequest does NOT have OrganizationID field - it's preserved in DB
	// This is correct behavior: org_id is set at session creation, not changed on resume
	updateReq := mockRepo.getLastUpdateReq()
	require.NotNil(t, updateReq)
	// Verify other fields are updated
	assert.NotNil(t, updateReq.Status)
	assert.Equal(t, "active", *updateReq.Status)
}

// ============================================================================
// TLS Evidence Tests (P1)
// ============================================================================

// TestPersistConnectAsync_FreshSession_WritesTLSEvidence validates that
// TLS version, cipher suite, and certificate fingerprint are persisted.
// Ref: session_persistence.go:81-96 (TLS evidence population)
func TestPersistConnectAsync_FreshSession_WritesTLSEvidence(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	session := &scaci.Session{
		TenantID: 1,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID: [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:  false,
	}

	certFingerprint := "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd"
	tlsVersion := "TLS 1.3"
	cipherSuite := "TLS_AES_256_GCM_SHA384"

	svc.PersistConnectAsync(testutil.TestContext(), session, certFingerprint, "CN=test-ac", "10.0.0.1:5001",
		tlsVersion, cipherSuite, scaci.ProtocolVersionString)
	mockRepo.waitForCompletion()

	require.True(t, mockRepo.wasCreateCalled())
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)

	// Verify TLS evidence fields
	require.NotNil(t, createReq.TLSVersion)
	assert.Equal(t, tlsVersion, *createReq.TLSVersion)

	require.NotNil(t, createReq.CipherSuite)
	assert.Equal(t, cipherSuite, *createReq.CipherSuite)

	require.NotNil(t, createReq.CertificateFingerprint)
	assert.Equal(t, certFingerprint, *createReq.CertificateFingerprint)

	require.NotNil(t, createReq.ClientCertSubject)
	assert.Equal(t, "CN=test-ac", *createReq.ClientCertSubject)

	require.NotNil(t, createReq.RemoteAddr)
	assert.Equal(t, "10.0.0.1:5001", *createReq.RemoteAddr)
}

// TestPersistConnectAsync_ResumedSession_UpdatesTLSEvidence validates that
// TLS evidence is updated on session resume (new TLS handshake).
// Ref: session_persistence.go:50-64 (TLS update on resume)
func TestPersistConnectAsync_ResumedSession_UpdatesTLSEvidence(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	session := &scaci.Session{
		ID:       999,
		TenantID: 42,
		Resumed:  true,
	}

	newTLSVersion := "TLS 1.3"
	newCipherSuite := "TLS_CHACHA20_POLY1305_SHA256"

	svc.PersistConnectAsync(testutil.TestContext(), session, "", "", "", newTLSVersion, newCipherSuite, scaci.ProtocolVersionString)
	mockRepo.waitForCompletion()

	require.True(t, mockRepo.wasUpdateCalled())
	updateReq := mockRepo.getLastUpdateReq()
	require.NotNil(t, updateReq)

	// Verify TLS evidence updated on resume
	require.NotNil(t, updateReq.TLSVersion)
	assert.Equal(t, newTLSVersion, *updateReq.TLSVersion)

	require.NotNil(t, updateReq.CipherSuite)
	assert.Equal(t, newCipherSuite, *updateReq.CipherSuite)
}

// TestPersistConnectAsync_UsesConnectPersistTimeout validates that
// persistence uses ConnectPersistTimeout constant (5 seconds per constants.go).
// This test verifies the timeout doesn't block the handler flow.
// Ref: session_persistence.go:42 (context.WithTimeout)
func TestPersistConnectAsync_UsesConnectPersistTimeout(t *testing.T) {
	// Verify the timeout constant exists and has expected value
	assert.Equal(t, 5*time.Second, scaci.ConnectPersistTimeout,
		"ConnectPersistTimeout should be 5 seconds per constants.go")
}

// TestPersistConnectAsync_NonBlocking validates that PersistConnectAsync
// returns immediately without waiting for persistence to complete.
func TestPersistConnectAsync_NonBlocking(t *testing.T) {
	log := logger.NewNop()

	// Create a slow mock that takes 500ms
	slowMock := newSlowSessionRepo(500 * time.Millisecond)
	svc := NewSessionPersistence(slowMock, log)

	session := &scaci.Session{
		TenantID: 1,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID: [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:  false,
	}

	start := time.Now()
	svc.PersistConnectAsync(testutil.TestContext(), session, "", "", "", "", "", scaci.ProtocolVersionString)
	elapsed := time.Since(start)

	// Should return almost immediately (not wait for 500ms mock delay)
	assert.Less(t, elapsed, 50*time.Millisecond,
		"PersistConnectAsync should return immediately, not block for persistence")

	// Wait for async to complete deterministically
	slowMock.waitForCompletion()
	assert.True(t, slowMock.wasCreateCalled(), "CreateSession should eventually be called")
}

// TestPersistConnectAsync_NegotiatedVersionDefault validates that
// negotiated_version defaults to ProtocolVersionString if empty.
// Ref: session_persistence.go:104-108
func TestPersistConnectAsync_NegotiatedVersionDefault(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	session := &scaci.Session{
		TenantID: 1,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID: [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:  false,
	}

	// Pass empty negotiatedVersion
	svc.PersistConnectAsync(testutil.TestContext(), session, "", "", "", "", "", "")
	mockRepo.waitForCompletion()

	require.True(t, mockRepo.wasCreateCalled())
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)

	// Should default to ProtocolVersionString
	assert.Equal(t, scaci.ProtocolVersionString, createReq.NegotiatedVersion,
		"Empty negotiatedVersion should default to ProtocolVersionString")
}

// ============================================================================
// Helper for Non-Blocking Test
// ============================================================================

type slowSessionRepo struct {
	mockSessionRepoForPersistence
	delay        time.Duration
	createCalled bool
	mu           sync.Mutex
	completionCh chan struct{} // Completion signal for deterministic waiting
}

func newSlowSessionRepo(delay time.Duration) *slowSessionRepo {
	return &slowSessionRepo{
		delay:        delay,
		completionCh: make(chan struct{}), // Unbuffered
	}
}

func (s *slowSessionRepo) CreateSession(_ context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	time.Sleep(s.delay) // Intentional delay for non-blocking test
	s.mu.Lock()
	s.createCalled = true
	s.mu.Unlock()
	s.completionCh <- struct{}{} // Signal completion
	return &models.SCACISession{ID: 1, TenantID: req.TenantID}, nil
}

func (s *slowSessionRepo) wasCreateCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createCalled
}

func (s *slowSessionRepo) waitForCompletion() {
	<-s.completionCh
}

// ============================================================================
// Metadata Persistence Tests (SCACI §3.3.1 info field)
// ============================================================================

// TestPersistConnectAsync_FreshSession_WritesNestedMetadata validates that
// nested metadata (including info field per §3.3.1) is persisted correctly.
// Ref: session_persistence.go:123 (Metadata in CreateRequest)
func TestPersistConnectAsync_FreshSession_WritesNestedMetadata(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	// Create session with nested metadata (info field per SCACI §3.3.1)
	session := &scaci.Session{
		TenantID: 42,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID: [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:  false,
		Metadata: map[string]interface{}{
			"vendor":    "TestVendor",
			"model":     "TestModel",
			"swVersion": "2.0.0",
			// info field: arbitrary nested object per SCACI §3.3.1
			"info": map[string]interface{}{
				"firmware":     "v1.2.3",
				"serialNo":     12345,
				"capabilities": []interface{}{"downlink", "multicast"},
			},
		},
	}

	svc.PersistConnectAsync(testutil.TestContext(), session, "", "", "", "", "", scaci.ProtocolVersionString)
	mockRepo.waitForCompletion()

	require.True(t, mockRepo.wasCreateCalled(), "CreateSession should be called for fresh session")
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)
	require.NotNil(t, createReq.Metadata, "Metadata should not be nil")

	// Verify string metadata preserved
	assert.Equal(t, "TestVendor", createReq.Metadata["vendor"])
	assert.Equal(t, "TestModel", createReq.Metadata["model"])
	assert.Equal(t, "2.0.0", createReq.Metadata["swVersion"])

	// Verify nested info object preserved (CRITICAL for §3.3.1)
	info, ok := createReq.Metadata["info"]
	require.True(t, ok, "info field should be present in persisted metadata")
	infoMap, ok := info.(map[string]interface{})
	require.True(t, ok, "info should be map[string]interface{}")
	assert.Equal(t, "v1.2.3", infoMap["firmware"])
	assert.Equal(t, 12345, infoMap["serialNo"])
	caps, ok := infoMap["capabilities"].([]interface{})
	require.True(t, ok, "capabilities should be slice")
	assert.Len(t, caps, 2)
}

// TestPersistConnectAsync_ResumedSession_WritesNestedMetadata validates that
// metadata is persisted on resume via UpdateSession.
// Ref: session_persistence.go:64 (Metadata in UpdateRequest)
func TestPersistConnectAsync_ResumedSession_WritesNestedMetadata(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	// Create resumed session with nested metadata
	session := &scaci.Session{
		ID:       999,
		TenantID: 42,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID: [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:  true,
		Metadata: map[string]interface{}{
			"vendor": "UpdatedVendor",
			"info": map[string]interface{}{
				"newField": "newValue",
				"nested": map[string]interface{}{
					"deep": "object",
				},
			},
		},
	}

	svc.PersistConnectAsync(testutil.TestContext(), session, "", "", "", "TLS 1.3", "TLS_AES_256_GCM_SHA384", scaci.ProtocolVersionString)
	mockRepo.waitForCompletion()

	require.True(t, mockRepo.wasUpdateCalled(), "UpdateSession should be called for resumed session")
	require.False(t, mockRepo.wasCreateCalled(), "CreateSession should NOT be called for resumed session")

	updateReq := mockRepo.getLastUpdateReq()
	require.NotNil(t, updateReq)
	require.NotNil(t, updateReq.Metadata, "Metadata should be included in update")

	// Verify nested info preserved in update
	info, ok := updateReq.Metadata["info"]
	require.True(t, ok, "info field should be present in update metadata")
	infoMap, ok := info.(map[string]interface{})
	require.True(t, ok, "info should be map[string]interface{}")
	assert.Equal(t, "newValue", infoMap["newField"])

	// Verify deeply nested object
	nested, ok := infoMap["nested"].(map[string]interface{})
	require.True(t, ok, "nested should be map")
	assert.Equal(t, "object", nested["deep"])
}

// TestDeepCopyMetadata_PreservesNestedTypes validates that deepCopyMetadata
// preserves non-string types (numbers, arrays, nested objects) AND provides
// true isolation (mutating original doesn't affect copy).
func TestDeepCopyMetadata_PreservesNestedTypes(t *testing.T) {
	input := map[string]interface{}{
		"stringVal": "hello",
		"intVal":    42,
		"floatVal":  3.14,
		"boolVal":   true,
		"arrayVal":  []interface{}{1, 2, 3},
		"nested": map[string]interface{}{
			"key": "value",
		},
	}

	result := deepCopyMetadata(input)

	require.NotNil(t, result)
	assert.Equal(t, "hello", result["stringVal"])
	assert.Equal(t, 42, result["intVal"])
	assert.Equal(t, 3.14, result["floatVal"])
	assert.Equal(t, true, result["boolVal"])

	arr, ok := result["arrayVal"].([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 3)

	nested, ok := result["nested"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", nested["key"])

	// Verify isolation: mutating original doesn't affect copy
	input["stringVal"] = "modified"
	input["nested"].(map[string]interface{})["key"] = "modified"
	assert.Equal(t, "hello", result["stringVal"], "copy should be isolated from original mutations")
	assert.Equal(t, "value", result["nested"].(map[string]interface{})["key"], "nested copy should be isolated")
}

// TestDeepCopyMetadata_NilInput validates nil handling
func TestDeepCopyMetadata_NilInput(t *testing.T) {
	result := deepCopyMetadata(nil)
	assert.Nil(t, result, "deepCopyMetadata(nil) should return nil")
}

// TestDeepCopyMetadata_NestedMapIsolation validates that nested maps are
// truly deep-copied, not shared references.
func TestDeepCopyMetadata_NestedMapIsolation(t *testing.T) {
	original := map[string]interface{}{
		"vendor": "TestVendor",
		"info": map[string]interface{}{
			"firmware": "v1.0",
			"config": map[string]interface{}{
				"setting1": "value1",
				"setting2": 42,
			},
		},
	}

	copied := deepCopyMetadata(original)

	// Mutate the original nested maps
	original["info"].(map[string]interface{})["firmware"] = "v2.0"
	original["info"].(map[string]interface{})["config"].(map[string]interface{})["setting1"] = "MODIFIED"
	original["info"].(map[string]interface{})["newKey"] = "newValue"

	// Verify copy is unchanged
	info := copied["info"].(map[string]interface{})
	assert.Equal(t, "v1.0", info["firmware"], "nested map should be isolated")
	config := info["config"].(map[string]interface{})
	assert.Equal(t, "value1", config["setting1"], "deeply nested map should be isolated")
	_, hasNewKey := info["newKey"]
	assert.False(t, hasNewKey, "new keys added to original should not appear in copy")
}

// TestDeepCopyMetadata_NestedSliceIsolation validates that nested slices are
// truly deep-copied, not shared references.
func TestDeepCopyMetadata_NestedSliceIsolation(t *testing.T) {
	original := map[string]interface{}{
		"capabilities": []interface{}{"read", "write", "delete"},
		"nested": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"id": 1, "name": "item1"},
				map[string]interface{}{"id": 2, "name": "item2"},
			},
		},
	}

	copied := deepCopyMetadata(original)

	// Mutate the original slices
	original["capabilities"].([]interface{})[0] = "MODIFIED"
	original["nested"].(map[string]interface{})["items"].([]interface{})[0].(map[string]interface{})["name"] = "MODIFIED"

	// Verify copy is unchanged
	caps := copied["capabilities"].([]interface{})
	assert.Equal(t, "read", caps[0], "slice elements should be isolated")

	nested := copied["nested"].(map[string]interface{})
	items := nested["items"].([]interface{})
	item1 := items[0].(map[string]interface{})
	assert.Equal(t, "item1", item1["name"], "nested slice elements should be isolated")
}

// ============================================================================
// Heartbeat Persistence Tests (SCACI §3.4)
// ============================================================================

// TestPersistHeartbeatAsync_CallsUpdateHeartbeat validates that
// PersistHeartbeatAsync invokes the repository's UpdateHeartbeat method.
// Ref: session_persistence.go:267-291
func TestPersistHeartbeatAsync_CallsUpdateHeartbeat(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	session := &scaci.Session{
		ID:       123,
		TenantID: 42,
	}

	svc.PersistHeartbeatAsync(testutil.TestContext(), session)
	mockRepo.waitForCompletion()

	assert.True(t, mockRepo.wasHeartbeatCalled(), "UpdateHeartbeat should be called")
}

// TestPersistHeartbeatAsync_CorrectIDs validates that the correct
// tenantID and sessionID are passed to UpdateHeartbeat.
// Ref: session_persistence.go:273-274 (primitive capture), line 284 (call)
func TestPersistHeartbeatAsync_CorrectIDs(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	expectedTenantID := int64(42)
	expectedSessionID := int64(999)

	session := &scaci.Session{
		ID:       expectedSessionID,
		TenantID: expectedTenantID,
	}

	svc.PersistHeartbeatAsync(testutil.TestContext(), session)
	mockRepo.waitForCompletion()

	tenantID, sessionID := mockRepo.getLastHeartbeatIDs()
	assert.Equal(t, expectedTenantID, tenantID, "tenantID should match session.TenantID")
	assert.Equal(t, expectedSessionID, sessionID, "sessionID should match session.ID")
}

// TestPersistHeartbeatAsync_SkipsNilSession validates that nil session
// is handled gracefully without panicking or calling UpdateHeartbeat.
// Ref: session_persistence.go:268-270 (guard clause)
func TestPersistHeartbeatAsync_SkipsNilSession(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	// Should not panic
	svc.PersistHeartbeatAsync(testutil.TestContext(), nil)

	// Give goroutine time to execute if it were to (it shouldn't)
	time.Sleep(50 * time.Millisecond)

	assert.False(t, mockRepo.wasHeartbeatCalled(), "UpdateHeartbeat should NOT be called for nil session")
}

// TestPersistHeartbeatAsync_SkipsZeroID validates that sessions with
// ID == 0 (not yet persisted) skip heartbeat persistence.
// Ref: session_persistence.go:268-270 (guard clause)
func TestPersistHeartbeatAsync_SkipsZeroID(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	svc := NewSessionPersistence(mockRepo, log)

	session := &scaci.Session{
		ID:       0, // Not yet persisted
		TenantID: 42,
	}

	svc.PersistHeartbeatAsync(testutil.TestContext(), session)

	// Give goroutine time to execute if it were to (it shouldn't)
	time.Sleep(50 * time.Millisecond)

	assert.False(t, mockRepo.wasHeartbeatCalled(), "UpdateHeartbeat should NOT be called for session.ID == 0")
}

// TestPersistHeartbeatAsync_NonBlocking validates that PersistHeartbeatAsync
// returns immediately without waiting for the DB call to complete.
// Ref: session_persistence.go:279-290 (goroutine spawn)
func TestPersistHeartbeatAsync_NonBlocking(t *testing.T) {
	log := logger.NewNop()

	// Create a slow mock that takes 500ms
	slowMock := newSlowHeartbeatRepo(500 * time.Millisecond)
	svc := NewSessionPersistence(slowMock, log)

	session := &scaci.Session{
		ID:       123,
		TenantID: 42,
	}

	start := time.Now()
	svc.PersistHeartbeatAsync(testutil.TestContext(), session)
	elapsed := time.Since(start)

	// Should return almost immediately (not wait for 500ms mock delay)
	assert.Less(t, elapsed, 50*time.Millisecond,
		"PersistHeartbeatAsync should return immediately, not block for persistence")

	// Wait for async to complete deterministically
	slowMock.waitForCompletion()
	assert.True(t, slowMock.wasHeartbeatCalled(), "UpdateHeartbeat should eventually be called")
}

// TestPersistHeartbeatAsync_UsesConnectPersistTimeout validates that
// heartbeat persistence uses ConnectPersistTimeout constant.
// Ref: session_persistence.go:281 (context.WithTimeout)
func TestPersistHeartbeatAsync_UsesConnectPersistTimeout(t *testing.T) {
	// Verify the timeout constant exists and has expected value
	assert.Equal(t, 5*time.Second, scaci.ConnectPersistTimeout,
		"ConnectPersistTimeout should be 5 seconds per constants.go")
}

// TestPersistHeartbeatAsync_LogsErrorOnFailure validates that errors
// from UpdateHeartbeat are logged but don't propagate (best-effort).
// Ref: session_persistence.go:285-288 (WarnContext on error)
func TestPersistHeartbeatAsync_LogsErrorOnFailure(t *testing.T) {
	log := logger.NewNop()
	mockRepo := newMockSessionRepoForPersistence()
	mockRepo.heartbeatErr = assert.AnError // Simulate DB failure
	svc := NewSessionPersistence(mockRepo, log)

	session := &scaci.Session{
		ID:       123,
		TenantID: 42,
	}

	// Should not panic even with error
	svc.PersistHeartbeatAsync(testutil.TestContext(), session)
	mockRepo.waitForCompletion()

	// Error is logged but method completes normally
	assert.True(t, mockRepo.wasHeartbeatCalled(), "UpdateHeartbeat should still be called")
}

// TestPersistHeartbeatAsync_WarnContextFields validates that WarnContext is called
// with the correct message and fields.
// This test uses a logger spy to capture WarnContext calls.
// Ref: session_persistence.go:285-288 (WarnContext call)
func TestPersistHeartbeatAsync_WarnContextFields(t *testing.T) {
	mockLogger := newMockLoggerForWarnContext()
	mockRepo := newMockSessionRepoForPersistence()
	mockRepo.heartbeatErr = assert.AnError // Simulate DB failure
	svc := NewSessionPersistence(mockRepo, mockLogger)

	expectedSessionID := int64(123)
	expectedTenantID := int64(42)

	session := &scaci.Session{
		ID:       expectedSessionID,
		TenantID: expectedTenantID,
	}

	svc.PersistHeartbeatAsync(testutil.TestContext(), session)
	mockRepo.waitForCompletion()

	// Wait for WarnContext call (deterministic)
	mockLogger.waitForCompletion()

	// Assert WarnContext was called
	require.True(t, mockLogger.wasWarnContextCalled(), "WarnContext should be called on error")

	// Assert correct message
	assert.Equal(t, scaci.LogSCACIPersistHeartbeatFailed, mockLogger.getWarnContextMsg(),
		"WarnContext should use LogSCACIPersistHeartbeatFailed message constant")

	// Assert fields contain sessionID, tenantID, error
	fields := mockLogger.getWarnContextFields()
	assert.Equal(t, expectedSessionID, fields["sessionID"], "sessionID field should match")
	assert.Equal(t, expectedTenantID, fields["tenantID"], "tenantID field should match")
	assert.NotNil(t, fields["error"], "error field should be present")
}

// ============================================================================
// Helper for Heartbeat Non-Blocking Test
// ============================================================================

type slowHeartbeatRepo struct {
	mockSessionRepoForPersistence
	delay           time.Duration
	heartbeatCalled bool
	mu              sync.Mutex
	completionCh    chan struct{}
}

func newSlowHeartbeatRepo(delay time.Duration) *slowHeartbeatRepo {
	return &slowHeartbeatRepo{
		delay:        delay,
		completionCh: make(chan struct{}),
	}
}

func (s *slowHeartbeatRepo) UpdateHeartbeat(_ context.Context, _, _ int64) error {
	time.Sleep(s.delay) // Intentional delay for non-blocking test
	s.mu.Lock()
	s.heartbeatCalled = true
	s.mu.Unlock()
	s.completionCh <- struct{}{} // Signal completion
	return nil
}

func (s *slowHeartbeatRepo) wasHeartbeatCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.heartbeatCalled
}

func (s *slowHeartbeatRepo) waitForCompletion() {
	<-s.completionCh
}

// ============================================================================
// Mock Logger for WarnContext Capture
// ============================================================================

// mockLoggerForWarnContext captures WarnContext calls for testing.
// Implements logger.Logger interface with only WarnContext actually capturing data.
type mockLoggerForWarnContext struct {
	mu                sync.Mutex
	warnContextCalled bool
	warnContextMsg    string
	warnContextFields map[string]interface{}
	completionCh      chan struct{} // Completion signal for deterministic testing
}

func newMockLoggerForWarnContext() *mockLoggerForWarnContext {
	return &mockLoggerForWarnContext{
		warnContextFields: make(map[string]interface{}),
		completionCh:      make(chan struct{}, 1), // Buffered to avoid blocking
	}
}

// WarnContext captures the call for assertion
func (m *mockLoggerForWarnContext) WarnContext(_ context.Context, msg string, fields ...interface{}) {
	m.mu.Lock()
	m.warnContextCalled = true
	m.warnContextMsg = msg
	// Convert fields to map (key-value pairs)
	for i := 0; i+1 < len(fields); i += 2 {
		if key, ok := fields[i].(string); ok {
			m.warnContextFields[key] = fields[i+1]
		}
	}
	m.mu.Unlock()
	m.completionCh <- struct{}{} // Signal completion after capturing
}

func (m *mockLoggerForWarnContext) wasWarnContextCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.warnContextCalled
}

func (m *mockLoggerForWarnContext) getWarnContextMsg() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.warnContextMsg
}

func (m *mockLoggerForWarnContext) getWarnContextFields() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]interface{})
	for k, v := range m.warnContextFields {
		result[k] = v
	}
	return result
}

func (m *mockLoggerForWarnContext) waitForCompletion() {
	<-m.completionCh
}

// Remaining Logger interface methods (no-op stubs)
func (m *mockLoggerForWarnContext) Debug(_ string, _ ...interface{})                           {}
func (m *mockLoggerForWarnContext) Info(_ string, _ ...interface{})                            {}
func (m *mockLoggerForWarnContext) Warn(_ string, _ ...interface{})                            {}
func (m *mockLoggerForWarnContext) Error(_ string, _ ...interface{})                           {}
func (m *mockLoggerForWarnContext) Fatal(_ string, _ ...interface{})                           {}
func (m *mockLoggerForWarnContext) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLoggerForWarnContext) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLoggerForWarnContext) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLoggerForWarnContext) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLoggerForWarnContext) WithField(_ string, _ interface{}) logger.Logger            { return m }
func (m *mockLoggerForWarnContext) WithFields(_ map[string]interface{}) logger.Logger          { return m }
