package scaciservices

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testServiceCenterEUI              = uint64(0x4B43000000000178)
	testExpectedConnectPersistTimeout = 5 * time.Second
	// persistenceSignalBuffer lets the synchronous calls signal the mock's
	// completion channel without a concurrent reader.
	persistenceSignalBuffer = 4
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
	lastUpdateReq    *models.SCACISessionResume
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
	// writes lists the lifecycle and counter writes in order.
	writes []string
}

func newMockSessionRepoForPersistence() *mockSessionRepoForPersistence {
	return &mockSessionRepoForPersistence{
		createdSession: &models.SCACISession{
			ID:       123,
			TenantID: 1,
		},
		completionCh: make(chan struct{}, persistenceSignalBuffer),
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

func (m *mockSessionRepoForPersistence) ResumeSession(_ context.Context, tenantID, sessionID int64, req *models.SCACISessionResume) error {
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
func (m *mockSessionRepoForPersistence) getLastUpdateReq() *models.SCACISessionResume {
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

// wasUpdateCalled returns whether ResumeSession was called (thread-safe)
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
// Use this when a test triggers exactly one CreateSession or ResumeSession call.
func (m *mockSessionRepoForPersistence) waitForCompletion() {
	<-m.completionCh
}

// Stub remaining interface methods
func (m *mockSessionRepoForPersistence) CheckSessionResumable(_ context.Context, _ models.SCACIApplicationCenter, _ [16]byte) (*models.SCACISessionResumptionInfo, error) {
	return nil, nil
}

func (m *mockSessionRepoForPersistence) GetSessionByAcUUID(_ context.Context, _ models.SCACIApplicationCenter, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}

func (m *mockSessionRepoForPersistence) GetSessionByID(_ context.Context, _, _ int64) (*models.SCACISession, error) {
	return nil, nil
}

func (m *mockSessionRepoForPersistence) GetSessionByScUUID(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}

func (m *mockSessionRepoForPersistence) UpdateOperationIDs(_ context.Context, tenantID, sessionID int64, acOpID, scOpID int64) error {
	m.noteRowWrite(fmt.Sprintf("opIDs %d/%d %d/%d", tenantID, sessionID, acOpID, scOpID))
	return nil
}

func (m *mockSessionRepoForPersistence) MarkSessionDisconnected(_ context.Context, tenantID, sessionID int64) error {
	m.noteRowWrite(fmt.Sprintf("disconnected %d/%d", tenantID, sessionID))
	return nil
}

func (m *mockSessionRepoForPersistence) noteRowWrite(write string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writes = append(m.writes, write)
}

func (m *mockSessionRepoForPersistence) rowWrites() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.writes...)
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

func (m *mockSessionRepoForPersistence) TerminateSession(_ context.Context, tenantID, sessionID int64) error {
	m.noteRowWrite(fmt.Sprintf("terminated %d/%d", tenantID, sessionID))
	return nil
}

func (m *mockSessionRepoForPersistence) ListSessions(_ context.Context, _ *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	return nil, 0, nil
}

func (m *mockSessionRepoForPersistence) GetSessionStatistics(_ context.Context, _ int64) (*models.SCACISessionStatistics, error) {
	return nil, nil
}

// creationTx runs a session creation against the mock: the retirement is
// recorded, the insert goes to the mock repository.
type creationTx struct {
	repo    *mockSessionRepoForPersistence
	retired []models.SCACIApplicationCenter
}

func (c *creationTx) Run(_ context.Context, fn func(SessionCreationTx) error) error { return fn(c) }

func (c *creationTx) RetirePriorSessions(_ context.Context, ac models.SCACIApplicationCenter) error {
	c.retired = append(c.retired, ac)
	return nil
}

func (c *creationTx) CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	if len(c.retired) == 0 {
		return nil, errCreatedBeforeRetirement
	}
	return c.repo.CreateSession(ctx, req)
}

// errCreatedBeforeRetirement fails a creation that did not retire the prior
// sessions first.
var errCreatedBeforeRetirement = errors.New("session created before the prior sessions were retired")

func newTestPersistence(t *testing.T, repo *mockSessionRepoForPersistence) *SessionRows {
	t.Helper()
	persistence, err := NewSessionRows(&creationTx{repo: repo}, repo, repo, testServiceCenterEUI)
	require.NoError(t, err)
	return persistence
}

// A fresh session retires the earlier sessions of its organization's
// application center in the transaction that creates it (SCACI §1); a session
// without an organization retires those without one.
func TestPersistConnectSync_RetiresThePriorSessionsFirst(t *testing.T) {
	orgID := uuid.New()
	for name, tc := range map[string]struct {
		org  uuid.UUID
		want *uuid.UUID
	}{
		"organization": {org: orgID, want: &orgID},
		"none":         {org: uuid.Nil, want: nil},
	} {
		t.Run(name, func(t *testing.T) {
			repo := newMockSessionRepoForPersistence()
			creations := &creationTx{repo: repo}
			persistence, err := NewSessionRows(creations, repo, repo, testServiceCenterEUI)
			require.NoError(t, err)
			session := &scaci.Session{TenantID: 42, OrganizationID: tc.org, AcEui: 0xAABBCCDDEEFF1122}

			_, err = persistence.PersistConnectSync(testutil.TestContext(), session, "", "", "", "", "", scaci.ProtocolVersionString)
			repo.waitForCompletion()

			require.NoError(t, err)
			assert.Equal(t, []models.SCACIApplicationCenter{{TenantID: 42, OrganizationID: tc.want, AcEUI: [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}}}, creations.retired)
		})
	}
}

// A created and a resumed session are both owned by this service center, the
// ownership its startup reconciliation is scoped by.
func TestSessionRows_RecordTheOwningServiceCenter(t *testing.T) {
	repo := newMockSessionRepoForPersistence()
	persistence := newTestPersistence(t, repo)
	owner := models.EUI(mioty.EUI64(testServiceCenterEUI).ToBytes())

	_, err := persistence.PersistConnectSync(testutil.TestContext(), &scaci.Session{TenantID: 42, AcEui: 0xAABBCCDDEEFF1122},
		"", "", "", "", "", scaci.ProtocolVersionString)
	require.NoError(t, err)
	require.NoError(t, persistence.PersistResume(testutil.TestContext(), &scaci.Session{ID: 7, TenantID: 42, Resumed: true}, "", ""))

	assert.Equal(t, owner, repo.lastCreateReq.ScEui)
	assert.Equal(t, owner, repo.lastUpdateReq.ScEui)
}

func TestNewSessionRows_RefusesAMissingStore(t *testing.T) {
	repo := newMockSessionRepoForPersistence()
	_, err := NewSessionRows(nil, repo, repo, testServiceCenterEUI)
	require.ErrorIs(t, err, errMissingSessionPersistenceDependency)
	_, err = NewSessionRows(&creationTx{repo: repo}, nil, repo, testServiceCenterEUI)
	require.ErrorIs(t, err, errMissingSessionPersistenceDependency)
	_, err = NewSessionRows(&creationTx{repo: repo}, repo, nil, testServiceCenterEUI)
	require.ErrorIs(t, err, errMissingSessionPersistenceDependency)
}

// SessionRows is the one writer of a session's row: the loss of the
// connection, the operation ID counters and the end of resumability reach
// the store for the session's tenant and ID.
func TestSessionRows_WritesTheSessionRow(t *testing.T) {
	repo := newMockSessionRepoForPersistence()
	rows := newTestPersistence(t, repo)
	session := &scaci.Session{ID: 7, TenantID: 42}
	ctx := testutil.TestContext()

	require.NoError(t, rows.PersistDisconnect(ctx, session))
	require.NoError(t, rows.PersistOpIDs(ctx, session, scaci.OpIDPair{AC: 3, SC: -4}))
	require.NoError(t, rows.EndResumability(ctx, session))

	assert.Equal(t, []string{"disconnected 42/7", "opIDs 42/7 3/-4", "terminated 42/7"}, repo.rowWrites())
}

// ============================================================================
// Organization ID Persistence Tests (P0)
// ============================================================================

// TestPersistConnectSync_WritesOrganizationID validates that fresh sessions
// have organization_id persisted to the database via the sync path.
func TestPersistConnectSync_WritesOrganizationID(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

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

	// The mock signals on an unbuffered channel; consume it concurrently so
	// the synchronous call does not deadlock.
	done := make(chan struct{})
	go func() { mockRepo.waitForCompletion(); close(done) }()
	id, err := svc.PersistConnectSync(testutil.TestContext(), session, "cert-fingerprint", "CN=test", "192.168.1.1:5001",
		"TLS 1.3", "TLS_AES_256_GCM_SHA384", scaci.ProtocolVersionString)
	<-done
	require.NoError(t, err)
	assert.Equal(t, int64(123), id, "sync persistence returns the DB-assigned session ID")

	// Assert CreateSession was called
	require.True(t, mockRepo.wasCreateCalled(), "CreateSession should be called for fresh session")

	// Verify organization_id was populated
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)
	require.NotNil(t, createReq.OrganizationID, "OrganizationID should not be nil")
	assert.Equal(t, orgID, *createReq.OrganizationID, "OrganizationID should match session value")
	assert.Equal(t, int64(42), createReq.TenantID, "TenantID should be preserved")
}

// TestPersistConnectSync_NilOrgID validates that nil org ID is handled
// correctly (community mode / no org resolution).
func TestPersistConnectSync_NilOrgID(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

	// Create session without organization ID (uuid.Nil)
	session := &scaci.Session{
		TenantID:       1,
		OrganizationID: uuid.Nil, // No org set (community mode)
		AcEui:          0xAABBCCDDEEFF1122,
		SnAcUUID:       [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID:       [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:        false,
	}

	done := make(chan struct{})
	go func() { mockRepo.waitForCompletion(); close(done) }()
	_, err := svc.PersistConnectSync(testutil.TestContext(), session, "", "", "", "", "", scaci.ProtocolVersionString)
	<-done
	require.NoError(t, err)

	require.True(t, mockRepo.wasCreateCalled())
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)
	assert.Nil(t, createReq.OrganizationID, "OrganizationID should be nil when session.OrganizationID is uuid.Nil")
}

// TestPersistResume_PreservesOrgID validates that resumed sessions do
// NOT override organization_id (it's preserved from the original session row).
func TestPersistResume_PreservesOrgID(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

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

	require.NoError(t, svc.PersistResume(testutil.TestContext(), session, "TLS 1.3", "TLS_AES_256_GCM_SHA384"))
	mockRepo.waitForCompletion()

	// Assert ResumeSession was called (not CreateSession)
	require.True(t, mockRepo.wasUpdateCalled(), "ResumeSession should be called for resumed session")
	require.False(t, mockRepo.wasCreateCalled(), "CreateSession should NOT be called for resumed session")

	// The resume carries no organization: org_id is set at session creation, not changed on resume
	require.NotNil(t, mockRepo.getLastUpdateReq())
}

// ============================================================================
// TLS Evidence Tests (P1)
// ============================================================================

// TestPersistConnectSync_WritesTLSEvidence validates that TLS version,
// cipher suite, and certificate fingerprint are persisted.
func TestPersistConnectSync_WritesTLSEvidence(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

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

	done := make(chan struct{})
	go func() { mockRepo.waitForCompletion(); close(done) }()
	_, err := svc.PersistConnectSync(testutil.TestContext(), session, certFingerprint, "CN=test-ac", "10.0.0.1:5001",
		tlsVersion, cipherSuite, scaci.ProtocolVersionString)
	<-done
	require.NoError(t, err)

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

// TestPersistResume_UpdatesTLSEvidence validates that TLS evidence is
// updated on session resume (new TLS handshake).
func TestPersistResume_UpdatesTLSEvidence(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

	session := &scaci.Session{
		ID:       999,
		TenantID: 42,
		Resumed:  true,
	}

	newTLSVersion := "TLS 1.3"
	newCipherSuite := "TLS_CHACHA20_POLY1305_SHA256"

	require.NoError(t, svc.PersistResume(testutil.TestContext(), session, newTLSVersion, newCipherSuite))
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

// TestConnectPersistTimeoutConstant validates the ConnectPersistTimeout
// constant bounding both persistence paths (5 seconds per constants.go).
func TestConnectPersistTimeoutConstant(t *testing.T) {
	// Verify the timeout constant exists and has expected value
	assert.Equal(t, testExpectedConnectPersistTimeout, scaci.ConnectPersistTimeout,
		"ConnectPersistTimeout should be 5 seconds per constants.go")
}

// TestPersistResume_RejectsFreshSession validates the resume-only
// contract: a session without a persisted ID is rejected without touching
// the repository (fresh sessions go through PersistConnectSync).
func TestPersistResume_RejectsFreshSession(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

	session := &scaci.Session{TenantID: 1, Resumed: false}
	require.ErrorIs(t, svc.PersistResume(testutil.TestContext(), session, "", ""), errResumeRequiresPersistedSession)

	assert.False(t, mockRepo.wasCreateCalled(), "a misused resume path must never create a session row")
	assert.False(t, mockRepo.wasUpdateCalled(), "a fresh session must not be updated as a resume")
}

// TestPersistConnectSync_NegotiatedVersionDefault validates that
// negotiated_version defaults to ProtocolVersionString if empty.
func TestPersistConnectSync_NegotiatedVersionDefault(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

	session := &scaci.Session{
		TenantID: 1,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID: [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		Resumed:  false,
	}

	// Pass empty negotiatedVersion
	done := make(chan struct{})
	go func() { mockRepo.waitForCompletion(); close(done) }()
	_, err := svc.PersistConnectSync(testutil.TestContext(), session, "", "", "", "", "", "")
	<-done
	require.NoError(t, err)

	require.True(t, mockRepo.wasCreateCalled())
	createReq := mockRepo.getLastCreateReq()
	require.NotNil(t, createReq)

	// Should default to ProtocolVersionString
	assert.Equal(t, scaci.ProtocolVersionString, createReq.NegotiatedVersion,
		"Empty negotiatedVersion should default to ProtocolVersionString")
}

// ============================================================================
// Metadata Persistence Tests (SCACI §3.3.1 info field)
// ============================================================================

// TestPersistConnectSync_WritesNestedMetadata validates that nested metadata
// (including info field per §3.3.1) is persisted correctly.
func TestPersistConnectSync_WritesNestedMetadata(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

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
				"firmware":     testFirmwareVersion,
				"serialNo":     12345,
				"capabilities": []interface{}{"downlink", "multicast"},
			},
		},
	}

	done := make(chan struct{})
	go func() { mockRepo.waitForCompletion(); close(done) }()
	_, err := svc.PersistConnectSync(testutil.TestContext(), session, "", "", "", "", "", scaci.ProtocolVersionString)
	<-done
	require.NoError(t, err)

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
	assert.Equal(t, testFirmwareVersion, infoMap["firmware"])
	assert.Equal(t, 12345, infoMap["serialNo"])
	caps, ok := infoMap["capabilities"].([]interface{})
	require.True(t, ok, "capabilities should be slice")
	assert.Len(t, caps, 2)
}

// TestPersistResume_WritesNestedMetadata validates that metadata is
// persisted on resume via ResumeSession.
func TestPersistResume_WritesNestedMetadata(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

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

	require.NoError(t, svc.PersistResume(testutil.TestContext(), session, "TLS 1.3", "TLS_AES_256_GCM_SHA384"))
	mockRepo.waitForCompletion()

	require.True(t, mockRepo.wasUpdateCalled(), "ResumeSession should be called for resumed session")
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
			"firmware": testFirmwareVersionInitial,
			"config": map[string]interface{}{
				"setting1": "value1",
				"setting2": 42,
			},
		},
	}

	copied := deepCopyMetadata(original)

	// Mutate the original nested maps
	original["info"].(map[string]interface{})["firmware"] = testFirmwareVersionUpdated
	original["info"].(map[string]interface{})["config"].(map[string]interface{})["setting1"] = "MODIFIED"
	original["info"].(map[string]interface{})["newKey"] = "newValue"

	// Verify copy is unchanged
	info := copied["info"].(map[string]interface{})
	assert.Equal(t, testFirmwareVersionInitial, info["firmware"], "nested map should be isolated")
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

// TestPersistHeartbeat_CallsUpdateHeartbeat validates that
// PersistHeartbeat invokes the repository's UpdateHeartbeat method.
// Ref: session_persistence.go:267-291
func TestPersistHeartbeat_CallsUpdateHeartbeat(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

	session := &scaci.Session{
		ID:       123,
		TenantID: 42,
	}

	require.NoError(t, svc.PersistHeartbeat(testutil.TestContext(), session))
	mockRepo.waitForCompletion()

	assert.True(t, mockRepo.wasHeartbeatCalled(), "UpdateHeartbeat should be called")
}

// TestPersistHeartbeat_CorrectIDs validates that the correct
// tenantID and sessionID are passed to UpdateHeartbeat.
// Ref: session_persistence.go:273-274 (primitive capture), line 284 (call)
func TestPersistHeartbeat_CorrectIDs(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

	expectedTenantID := int64(42)
	expectedSessionID := int64(999)

	session := &scaci.Session{
		ID:       expectedSessionID,
		TenantID: expectedTenantID,
	}

	require.NoError(t, svc.PersistHeartbeat(testutil.TestContext(), session))
	mockRepo.waitForCompletion()

	tenantID, sessionID := mockRepo.getLastHeartbeatIDs()
	assert.Equal(t, expectedTenantID, tenantID, "tenantID should match session.TenantID")
	assert.Equal(t, expectedSessionID, sessionID, "sessionID should match session.ID")
}

// TestPersistHeartbeat_SkipsZeroID validates that sessions with
// ID == 0 (not yet persisted) skip heartbeat persistence.
// Ref: session_persistence.go:268-270 (guard clause)
func TestPersistHeartbeat_SkipsZeroID(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	svc := newTestPersistence(t, mockRepo)

	session := &scaci.Session{
		ID:       0, // Not yet persisted
		TenantID: 42,
	}

	require.NoError(t, svc.PersistHeartbeat(testutil.TestContext(), session))

	assert.False(t, mockRepo.wasHeartbeatCalled(), "UpdateHeartbeat should NOT be called for session.ID == 0")
}

// TestPersistHeartbeat_ReturnsStoreFailure validates that the store's error
// reaches the caller, which decides how to report it.
// Ref: session_persistence.go:285-288 (WarnContext on error)
func TestPersistHeartbeat_ReturnsStoreFailure(t *testing.T) {
	mockRepo := newMockSessionRepoForPersistence()
	mockRepo.heartbeatErr = assert.AnError // Simulate DB failure
	svc := newTestPersistence(t, mockRepo)

	session := &scaci.Session{
		ID:       123,
		TenantID: 42,
	}

	require.ErrorIs(t, svc.PersistHeartbeat(testutil.TestContext(), session), assert.AnError)
	assert.True(t, mockRepo.wasHeartbeatCalled(), "UpdateHeartbeat should still be called")
}
