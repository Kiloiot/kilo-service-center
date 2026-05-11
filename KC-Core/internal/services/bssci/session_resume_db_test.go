package bssciservices

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleResume_DBLookup_ByScUuid tests database fallback with SC UUID lookup
func TestHandleResume_DBLookup_ByScUuid(t *testing.T) {
	// Setup: Create session service with mock repository
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		100, // tenant ID
		testLogger,
	).(*sessionService)

	// Setup: Create a persisted session in DB
	scUUID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	bsUUID := [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}
	orgID := uuid.New()

	protocolVersion := "1.0.0"
	dbSession := &models.BaseStationSession{
		ID:              42,
		BaseStationID:   1,
		TenantID:        100,
		SnBsUuid:        bsUUID,
		SnScUuid:        scUUID,
		SnBsOpId:        1000,
		SnScOpId:        -500,
		Status:          models.SessionStatusActive,
		CanResume:       true,
		Encoding:        "msgpack",
		ProtocolVersion: &protocolVersion, // BSSCI §4-4.5: protocol version persisted
		OrganizationID:  &orgID,
		StartedAt:       time.Now().Add(-1 * time.Hour),
	}
	mockRepo.sessions[42] = dbSession

	// Execute: Call HandleResume with SC UUID
	// Create session with matching tenant ID for DB lookup
	testSession := &bssci.Session{ResolvedTenantID: 100}
	restoredSession, err := svc.HandleResume(testSession, bsUUID[:], scUUID[:], 1001, -500, 0x123456789ABCDEF0)

	// Verify: Session restored with all fields
	require.NoError(t, err)
	require.NotNil(t, restoredSession)
	assert.Equal(t, int64(42), restoredSession.DbSessionID)
	assert.Equal(t, scUUID[:], restoredSession.SessionUUID)
	assert.Equal(t, bsUUID[:], restoredSession.BsUUID)
	assert.Equal(t, int64(1000), restoredSession.LastBsOpId)
	assert.Equal(t, int64(-500), restoredSession.LastScOpId)
	assert.Equal(t, "msgpack", restoredSession.Encoding)
	assert.Equal(t, "1.0.0", restoredSession.NegotiatedVersion) // BSSCI §4-4.5: version restored
	assert.NotEmpty(t, restoredSession.ClientVersion)           // Client version populated from DB
	assert.Equal(t, orgID, restoredSession.OrganizationID)
	assert.Equal(t, int64(100), restoredSession.ResolvedTenantID)
	assert.Equal(t, uint64(0x123456789ABCDEF0), restoredSession.BaseStationEUI)
	assert.True(t, restoredSession.IsResumed)

	// Verify: Session re-populated in in-memory map
	cachedSession, exists := svc.sessionsByUUID[string(scUUID[:])]
	assert.True(t, exists)
	assert.Equal(t, restoredSession, cachedSession)
}

// TestHandleResume_DBLookup_ByBsUuid tests fallback with BS UUID lookup
func TestHandleResume_DBLookup_ByBsUuid(t *testing.T) {
	// Setup: Create session service with mock repository
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		200, // tenant ID
		testLogger,
	).(*sessionService)

	// Setup: Create session with only BS UUID match (SC UUID mismatch)
	bsUUID := [16]byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30}
	scUUID := [16]byte{0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F, 0x40}

	dbSession := &models.BaseStationSession{
		ID:             43,
		BaseStationID:  2,
		TenantID:       200,
		SnBsUuid:       bsUUID,
		SnScUuid:       scUUID,
		SnBsOpId:       2000,
		SnScOpId:       -1000,
		Status:         models.SessionStatusActive,
		CanResume:      true,
		Encoding:       "json",
		OrganizationID: nil, // Test nil pointer handling
		StartedAt:      time.Now().Add(-30 * time.Minute),
	}
	mockRepo.sessions[43] = dbSession

	// Execute: Call HandleResume with mismatched SC UUID (forces BS UUID fallback)
	wrongScUUID := [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	testSession := &bssci.Session{ResolvedTenantID: 200}
	restoredSession, err := svc.HandleResume(testSession, bsUUID[:], wrongScUUID[:], 2001, -1000, 0xFEDCBA9876543210)

	// Verify: BS UUID fallback succeeds
	require.NoError(t, err)
	require.NotNil(t, restoredSession)
	assert.Equal(t, int64(43), restoredSession.DbSessionID)
	assert.Equal(t, int64(2000), restoredSession.LastBsOpId)
	assert.Equal(t, int64(-1000), restoredSession.LastScOpId)
	assert.Equal(t, "json", restoredSession.Encoding)
	assert.Equal(t, uuid.Nil, restoredSession.OrganizationID) // Nil pointer converted to uuid.Nil
	assert.Equal(t, int64(200), restoredSession.ResolvedTenantID)
	assert.True(t, restoredSession.IsResumed)
}

// TestHandleResume_ShortUUID_SkipsDBLookup tests that short UUIDs don't trigger DB queries
func TestHandleResume_ShortUUID_SkipsDBLookup(t *testing.T) {
	// Setup: Create session service
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		300,
		testLogger,
	).(*sessionService)

	// Execute: Call HandleResume with short UUIDs (< 16 bytes)
	shortUUID := []byte{0x01, 0x02, 0x03} // Only 3 bytes
	testSession := &bssci.Session{ResolvedTenantID: 300}
	restoredSession, err := svc.HandleResume(testSession, shortUUID, shortUUID, 100, -50, 0x123456789ABCDEF0)

	// Verify: Returns nil (no panic, no DB query)
	require.NoError(t, err)
	assert.Nil(t, restoredSession)
}

// TestHandleResume_NilUUID_SkipsDBLookup tests that nil UUIDs are handled gracefully
func TestHandleResume_NilUUID_SkipsDBLookup(t *testing.T) {
	// Setup
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		400,
		testLogger,
	).(*sessionService)

	// Execute: Call HandleResume with nil UUIDs
	testSession := &bssci.Session{ResolvedTenantID: 400}
	restoredSession, err := svc.HandleResume(testSession, nil, nil, 100, -50, 0x123456789ABCDEF0)

	// Verify: Returns nil cleanly (no panic)
	require.NoError(t, err)
	assert.Nil(t, restoredSession)
}

// TestHandleResume_NoDBMatch_ReturnNil tests fallback to fresh session
func TestHandleResume_NoDBMatch_ReturnNil(t *testing.T) {
	// Setup: Empty repository
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		500,
		testLogger,
	).(*sessionService)

	// Execute: Try to resume non-existent session
	unknownUUID := [16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}
	testSession := &bssci.Session{ResolvedTenantID: 500}
	restoredSession, err := svc.HandleResume(testSession, unknownUUID[:], unknownUUID[:], 100, -50, 0x123456789ABCDEF0)

	// Verify: Returns nil for fresh session fallback
	require.NoError(t, err)
	assert.Nil(t, restoredSession)
}

// TestHandleResume_UsesServerTenantID tests tenant isolation
func TestHandleResume_UsesServerTenantID(t *testing.T) {
	// Setup: Repository with session for tenant 600
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	// Create service with tenant 999 (different from session's tenant)
	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		999, // Service tenant ID
		testLogger,
	).(*sessionService)

	scUUID := [16]byte{0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F}
	bsUUID := [16]byte{0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F}

	dbSession := &models.BaseStationSession{
		ID:             44,
		BaseStationID:  3,
		TenantID:       600, // Different from service tenant
		SnBsUuid:       bsUUID,
		SnScUuid:       scUUID,
		SnBsOpId:       3000,
		SnScOpId:       -1500,
		Status:         models.SessionStatusActive,
		CanResume:      true,
		Encoding:       "msgpack",
		OrganizationID: nil,
		StartedAt:      time.Now().Add(-2 * time.Hour),
	}
	mockRepo.sessions[44] = dbSession

	// Execute: Resume session - use tenant 600 to match DB session, not service's 999
	testSession := &bssci.Session{ResolvedTenantID: 600}
	restoredSession, err := svc.HandleResume(testSession, bsUUID[:], scUUID[:], 3001, -1500, 0xABCDEF0123456789)

	// Verify: Session restored with DB's tenant ID (not service's default)
	require.NoError(t, err)
	require.NotNil(t, restoredSession)
	assert.Equal(t, int64(600), restoredSession.ResolvedTenantID) // From DB, not service.tenantID
}

// TestHandleResume_TenantIsolation verifies sessions are isolated by tenant ID
func TestHandleResume_TenantIsolation(t *testing.T) {
	// Setup: Create session service for tenant 800
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		800, // Service default tenant
		testLogger,
	).(*sessionService)

	// Setup: Create session in DB for tenant 900 (different from service)
	scUUID := [16]byte{0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7A, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F}
	bsUUID := [16]byte{0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8A, 0x8B, 0x8C, 0x8D, 0x8E, 0x8F}

	dbSession := &models.BaseStationSession{
		ID:             45,
		BaseStationID:  5,
		TenantID:       900, // Tenant 900 owns this session
		SnBsUuid:       bsUUID,
		SnScUuid:       scUUID,
		SnBsOpId:       5000,
		SnScOpId:       -2500,
		Status:         models.SessionStatusActive,
		CanResume:      true,
		Encoding:       "msgpack",
		OrganizationID: nil,
		StartedAt:      time.Now().Add(-1 * time.Hour),
	}
	mockRepo.sessions[45] = dbSession

	// Execute: Attempt to resume with wrong tenant (800 instead of 900)
	wrongTenantSession := &bssci.Session{ResolvedTenantID: 800}
	restoredSession, err := svc.HandleResume(wrongTenantSession, bsUUID[:], scUUID[:], 5001, -2500, 0x1122334455667788)

	// Verify: Tenant isolation - should NOT find session from different tenant
	require.NoError(t, err)
	assert.Nil(t, restoredSession, "Should not resume session from different tenant")

	// Execute: Attempt to resume with correct tenant (900)
	correctTenantSession := &bssci.Session{ResolvedTenantID: 900}
	restoredSession, err = svc.HandleResume(correctTenantSession, bsUUID[:], scUUID[:], 5001, -2500, 0x1122334455667788)

	// Verify: Correct tenant can resume the session
	require.NoError(t, err)
	require.NotNil(t, restoredSession, "Should resume session with correct tenant")
	assert.Equal(t, int64(45), restoredSession.DbSessionID)
	assert.Equal(t, int64(900), restoredSession.ResolvedTenantID)
}

// TestHydrateSessionFromDB_OrganizationIDNilSafety tests nil pointer conversion
func TestHydrateSessionFromDB_OrganizationIDNilSafety(t *testing.T) {
	// Setup
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		700,
		testLogger,
	).(*sessionService)

	// Test cases: nil vs. non-nil OrganizationID
	testCases := []struct {
		name           string
		orgID          *uuid.UUID
		expectedResult uuid.UUID
	}{
		{
			name:           "Nil pointer converts to uuid.Nil",
			orgID:          nil,
			expectedResult: uuid.Nil,
		},
		{
			name:           "Non-nil pointer preserved",
			orgID:          func() *uuid.UUID { id := uuid.New(); return &id }(),
			expectedResult: func() uuid.UUID { id := uuid.New(); return id }(), // Will be overwritten below
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.orgID != nil {
				tc.expectedResult = *tc.orgID
			}

			dbSession := &models.BaseStationSession{
				ID:             45,
				BaseStationID:  4,
				TenantID:       700,
				SnBsUuid:       [16]byte{},
				SnScUuid:       [16]byte{},
				SnBsOpId:       4000,
				SnScOpId:       -2000,
				Status:         models.SessionStatusActive,
				CanResume:      true,
				Encoding:       "json",
				OrganizationID: tc.orgID,
				StartedAt:      time.Now(),
			}

			// Execute
			result := svc.hydrateSessionFromDB(dbSession, 0x1234567890ABCDEF)

			// Verify
			assert.Equal(t, tc.expectedResult, result.OrganizationID)
		})
	}
}

// TestPersistSessionResumeUpdatesProtocolVersion ensures resume path persists negotiated version to repository
func TestPersistSessionResumeUpdatesProtocolVersion(t *testing.T) {
	mockRepo := newMockBaseStationSessionRepo()
	mockBsRepo := &mockBaseStationRepo{}
	mockEventStore := &mockSystemEventStore{}
	testLogger := logger.NewNop()

	svc := NewSessionService(
		mockRepo,
		mockBsRepo,
		mockEventStore,
		800,
		testLogger,
	).(*sessionService)

	// Seed repository with existing session lacking protocol_version (legacy row)
	scUUID := [16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
	bsUUID := [16]byte{0x0A, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0xFF, 0xEE, 0xDD, 0xCC, 0xBB, 0xAA}
	mockRepo.sessions[99] = &models.BaseStationSession{
		ID:            99,
		BaseStationID: 10,
		TenantID:      800,
		SnBsUuid:      bsUUID,
		SnScUuid:      scUUID,
		SnBsOpId:      3000,
		SnScOpId:      -1500,
		Status:        models.SessionStatusActive,
		CanResume:     true,
		Encoding:      "msgpack",
	}

	session := &bssci.Session{
		ResolvedTenantID:  800,
		SessionUUID:       append([]byte(nil), scUUID[:]...),
		BsUUID:            append([]byte(nil), bsUUID[:]...),
		NegotiatedVersion: mioty.MIOTYProtocolVersion,
		Encoding:          "msgpack",
	}

	err := svc.PersistSession(context.Background(), session, nil, true, nil)
	require.NoError(t, err)

	// Verify repository row now has protocol_version populated
	stored := mockRepo.sessions[99]
	require.NotNil(t, stored)
	if assert.NotNil(t, stored.ProtocolVersion, "protocol_version should be persisted on resume") {
		assert.Equal(t, mioty.MIOTYProtocolVersion, *stored.ProtocolVersion)
	}
}
