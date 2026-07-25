package bssciservices

import (
	"errors"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

func newResumeTestService(t *testing.T, tenantID int64) (*sessionService, *mockBaseStationSessionRepo) {
	t.Helper()
	svc, mockRepo, _ := newTakeoverTestService(t, tenantID)
	return svc, mockRepo
}

func newTakeoverTestService(t *testing.T, tenantID int64) (*sessionService, *mockBaseStationSessionRepo, *mockPendingOpsStore) {
	t.Helper()
	mockRepo := newMockBaseStationSessionRepo()
	pendingOps := newMockPendingOpsStore()
	svc := NewSessionService(
		mockRepo,
		&mockBaseStationRepo{},
		pendingOps,
		&mockSystemEventStore{},
		tenantID,
		logger.NewNop(),
	).(*sessionService)
	return svc, mockRepo, pendingOps
}

func seedResumableSession(repo *mockBaseStationSessionRepo, id, tenantID int64, bsUUID, scUUID [16]byte, bsOpId, scOpId int64) *models.BaseStationSession {
	protocolVersion := mioty.MIOTYProtocolVersion
	session := &models.BaseStationSession{
		ID:              id,
		CanResume:       true,
		Encoding:        "msgpack",
		BaseStationID:   1,
		TenantID:        tenantID,
		SnBsUuid:        bsUUID,
		SnScUuid:        scUUID,
		SnBsOpId:        bsOpId,
		SnScOpId:        scOpId,
		Status:          models.SessionStatusDisconnected,
		ProtocolVersion: &protocolVersion,
		StartedAt:       time.Now().Add(-1 * time.Hour),
	}
	repo.sessions[id] = session
	return session
}

func int64Ptr(v int64) *int64 { return &v }

// TestHandleResume_DBLookup_BySnBsUuid verifies the database fallback: resume
// identity is snBsUuid scoped by tenant and base station EUI, matching only
// disconnected, resumable sessions (BSSCI §5.3.1).
func TestHandleResume_DBLookup_BySnBsUuid(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	scUUID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	bsUUID := [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}
	orgID := uuid.New()
	dbSession := seedResumableSession(mockRepo, 42, 100, bsUUID, scUUID, 1000, -500)
	dbSession.OrganizationID = &orgID

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:],
		int64Ptr(1000), int64Ptr(-500), 0x123456789ABCDEF0)

	require.Equal(t, bssci.ResumeCompatible, outcome.Disposition)
	require.NotNil(t, outcome.Previous)
	restoredSession := outcome.Previous
	assert.Equal(t, int64(42), restoredSession.DbSessionID)
	assert.Equal(t, scUUID[:], restoredSession.SessionUUID)
	assert.Equal(t, bsUUID[:], restoredSession.BsUUID)
	assert.Equal(t, int64(1000), restoredSession.LastBsOpId)
	assert.Equal(t, int64(-500), restoredSession.LastScOpId)
	assert.Equal(t, "msgpack", restoredSession.Encoding)
	assert.Equal(t, mioty.MIOTYProtocolVersion, restoredSession.NegotiatedVersion)
	assert.Equal(t, orgID, restoredSession.OrganizationID)
	assert.Equal(t, int64(100), restoredSession.ResolvedTenantID)
	assert.Equal(t, uint64(0x123456789ABCDEF0), restoredSession.BaseStationEUI)
}

// TestHandleResume_AbsentCounters verifies that absent snBsOpId/snScOpId
// constraints are not asserted (BSSCI §5.3.1 optional fields).
func TestHandleResume_AbsentCounters(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30}
	seedResumableSession(mockRepo, 43, 100, bsUUID, [16]byte{0x31}, 2000, -900)

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:], nil, nil, 0xAABB)

	require.Equal(t, bssci.ResumeCompatible, outcome.Disposition)
	require.NotNil(t, outcome.Previous)
	restoredSession := outcome.Previous
	assert.Equal(t, int64(2000), restoredSession.LastBsOpId,
		"authoritative persisted counters restored when constraints absent")
	assert.Equal(t, int64(-900), restoredSession.LastScOpId)
}

// TestHandleResume_StaleScCounterAccepted verifies a stale (less negative)
// snScOpId is compatible: the service center is authoritative for its own
// operation IDs.
func TestHandleResume_StaleScCounterAccepted(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F, 0x50}
	seedResumableSession(mockRepo, 44, 100, bsUUID, [16]byte{0x51}, 500, -800)

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:],
		int64Ptr(400), int64Ptr(-700), 0xCCDD)

	require.Equal(t, bssci.ResumeCompatible, outcome.Disposition)
	require.NotNil(t, outcome.Previous)
	restoredSession := outcome.Previous
	assert.Equal(t, int64(-800), restoredSession.LastScOpId,
		"the authoritative persisted SC counter is restored, not the stale reported one")
}

// TestHandleResume_RequiredBsOpIdBeyondPersisted verifies rejection when the
// base station requires a minimum operation state the service center does not
// know (snBsOpId above the persisted counter).
func TestHandleResume_RequiredBsOpIdBeyondPersisted(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F, 0x70}
	seedResumableSession(mockRepo, 45, 100, bsUUID, [16]byte{0x71}, 1000, -500)

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:],
		int64Ptr(1001), int64Ptr(-500), 0xEEFF)

	assert.Equal(t, bssci.ResumeInconsistent, outcome.Disposition)
	assert.ErrorIs(t, outcome.Err, bssci.ErrResumeCounterMismatch)
}

// TestHandleResume_ClaimedScOpIdBeyondIssued verifies rejection when the base
// station claims a more negative SC operation ID than the service center ever
// issued.
func TestHandleResume_ClaimedScOpIdBeyondIssued(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8A, 0x8B, 0x8C, 0x8D, 0x8E, 0x8F, 0x90}
	seedResumableSession(mockRepo, 46, 100, bsUUID, [16]byte{0x91}, 1000, -500)

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:],
		int64Ptr(1000), int64Ptr(-501), 0xEEFF)

	assert.Equal(t, bssci.ResumeInconsistent, outcome.Disposition)
	assert.ErrorIs(t, outcome.Err, bssci.ErrResumeCounterMismatch)
}

// TestHandleResume_TerminatedNotResumable verifies terminated sessions never
// resume regardless of matching identity.
func TestHandleResume_TerminatedNotResumable(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF, 0xB0}
	session := seedResumableSession(mockRepo, 47, 100, bsUUID, [16]byte{0xB1}, 100, -50)
	session.Status = models.SessionStatusTerminated
	session.CanResume = false

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:], nil, nil, 0x1234)

	assert.Equal(t, bssci.ResumeNoMatch, outcome.Disposition, "terminated sessions must not resume")
}

// TestHandleResume_ActiveNotResumableFromDB verifies only disconnected
// sessions qualify for the database resume path.
func TestHandleResume_ActiveNotResumableFromDB(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF, 0xD0}
	session := seedResumableSession(mockRepo, 48, 100, bsUUID, [16]byte{0xD1}, 100, -50)
	session.Status = models.SessionStatusActive

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:], nil, nil, 0x1234)

	assert.Equal(t, bssci.ResumeNoMatch, outcome.Disposition, "only disconnected sessions are resumable from persistence")
}

// TestHandleResume_ShortUUID_SkipsDBLookup verifies malformed UUIDs never
// reach the database.
func TestHandleResume_ShortUUID_SkipsDBLookup(t *testing.T) {
	svc, _ := newResumeTestService(t, 100)

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, []byte{0x01, 0x02},
		int64Ptr(100), int64Ptr(-50), 0x123456789ABCDEF0)

	assert.Equal(t, bssci.ResumeNoMatch, outcome.Disposition)
}

// TestHandleResume_NilUUID_SkipsDBLookup verifies nil UUIDs never reach the
// database.
func TestHandleResume_NilUUID_SkipsDBLookup(t *testing.T) {
	svc, _ := newResumeTestService(t, 100)

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, nil,
		int64Ptr(100), int64Ptr(-50), 0x123456789ABCDEF0)

	assert.Equal(t, bssci.ResumeNoMatch, outcome.Disposition)
}

// TestHandleResume_NoDBMatch_ReturnNil verifies unknown UUIDs fall back to a
// fresh session.
func TestHandleResume_NoDBMatch_ReturnNil(t *testing.T) {
	svc, _ := newResumeTestService(t, 100)

	unknownUUID := [16]byte{0xDE, 0xAD, 0xBE, 0xEF}
	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, unknownUUID[:],
		int64Ptr(100), int64Ptr(-50), 0x123456789ABCDEF0)

	assert.Equal(t, bssci.ResumeNoMatch, outcome.Disposition)
}

// TestHandleResume_TenantIsolation verifies a session persisted under another
// tenant never resumes, even with a matching UUID.
func TestHandleResume_TenantIsolation(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF, 0xF0}
	seedResumableSession(mockRepo, 49, 500, bsUUID, [16]byte{0xF1}, 5000, -2500)

	wrongTenantSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	wrongOutcome := svc.HandleResume(testutil.TestContext(), wrongTenantSession, bsUUID[:],
		int64Ptr(5000), int64Ptr(-2500), 0x1122334455667788)
	assert.Equal(t, bssci.ResumeNoMatch, wrongOutcome.Disposition, "cross-tenant UUID collisions must not resume")

	correctTenantSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 500,
		},
	}
	correctOutcome := svc.HandleResume(testutil.TestContext(), correctTenantSession, bsUUID[:],
		int64Ptr(5000), int64Ptr(-2500), 0x1122334455667788)
	require.Equal(t, bssci.ResumeCompatible, correctOutcome.Disposition)
	require.NotNil(t, correctOutcome.Previous)
	assert.Equal(t, int64(500), correctOutcome.Previous.ResolvedTenantID)
}

func TestHydrateSessionFromDB_OrganizationIDNilSafety(t *testing.T) {
	svc, _ := newResumeTestService(t, 700)

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
				CanResume:      true,
				Encoding:       "json",
				OrganizationID: tc.orgID,
				BaseStationID:  4,
				TenantID:       700,
				SnBsUuid:       [16]byte{},
				SnScUuid:       [16]byte{},
				SnBsOpId:       4000,
				SnScOpId:       -2000,
				Status:         models.SessionStatusDisconnected,
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
	svc, mockRepo := newResumeTestService(t, 800)

	// Seed repository with existing session lacking protocol_version (legacy row)
	scUUID := [16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
	bsUUID := [16]byte{0x0A, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0xFF, 0xEE, 0xDD, 0xCC, 0xBB, 0xAA}
	mockRepo.sessions[99] = &models.BaseStationSession{
		ID:            99,
		CanResume:     true,
		Encoding:      "msgpack",
		BaseStationID: 10,
		TenantID:      800,
		SnBsUuid:      bsUUID,
		SnScUuid:      scUUID,
		SnBsOpId:      3000,
		SnScOpId:      -1500,
		Status:        models.SessionStatusDisconnected,
	}

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "resume-connection-1",
			ResolvedTenantID:  800,
			SessionUUID:       append([]byte(nil), scUUID[:]...),
			BsUUID:            append([]byte(nil), bsUUID[:]...),
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
		},
	}

	err := svc.PersistSession(testutil.TestContext(), session, nil, true, nil)
	require.NoError(t, err)

	// Verify repository row now has protocol_version populated and the resume
	// activation restored the active, resumable state
	stored := mockRepo.sessions[99]
	require.NotNil(t, stored)
	if assert.NotNil(t, stored.ProtocolVersion, "protocol_version should be persisted on resume") {
		assert.Equal(t, mioty.MIOTYProtocolVersion, *stored.ProtocolVersion)
	}
	assert.Equal(t, models.SessionStatusActive, stored.Status, "resume activation sets active status")
	assert.True(t, stored.CanResume, "resume activation keeps the session resumable")
	assert.Nil(t, stored.EndedAt, "resume activation clears ended_at")
	if assert.NotNil(t, stored.ConnectionId) {
		assert.Equal(t, "resume-connection-1", *stored.ConnectionId)
	}
}

// TestMarkDisconnected_StaleConnectionDoesNotTouchNewerSession verifies the
// stale-cleanup guard: after a reconnect activates connection B, the deferred
// cleanup of replaced connection A matches zero rows and B stays active.
func TestMarkDisconnected_StaleConnectionDoesNotTouchNewerSession(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 900)

	scUUID := [16]byte{0x10, 0x20, 0x30}
	bsUUID := [16]byte{0x40, 0x50, 0x60}
	dbSession := seedResumableSession(mockRepo, 77, 900, bsUUID, scUUID, 10, -5)

	// Connection B resumed and activated the session
	sessionB := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "connection-b",
			ResolvedTenantID:  900,
			DbSessionID:       77,
			SessionUUID:       append([]byte(nil), scUUID[:]...),
			BsUUID:            append([]byte(nil), bsUUID[:]...),
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
		},
	}
	require.NoError(t, svc.PersistSession(testutil.TestContext(), sessionB, nil, true, nil))
	require.Equal(t, models.SessionStatusActive, dbSession.Status)

	// Connection A's deferred cleanup runs after B took over
	sessionA := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:               "connection-a",
			ResolvedTenantID: 900,
			DbSessionID:      77,
		},
	}
	require.NoError(t, svc.MarkDisconnected(testutil.TestContext(), sessionA))

	// B remains active and resumable state is untouched by A's cleanup
	assert.Equal(t, models.SessionStatusActive, dbSession.Status,
		"stale cleanup must not disconnect the newer connection's session")
	assert.Nil(t, dbSession.EndedAt)

	// B's own later cleanup does transition the session
	require.NoError(t, svc.MarkDisconnected(testutil.TestContext(), sessionB))
	assert.Equal(t, models.SessionStatusDisconnected, dbSession.Status)
	assert.True(t, dbSession.CanResume)
	assert.NotNil(t, dbSession.EndedAt)
}

// TestHandleResume_InfrastructureFailure verifies a resumable-session lookup
// failure yields ResumeInfrastructureFailure so the connect is rejected rather
// than silently degraded into a fresh session that strands the old state.
func TestHandleResume_InfrastructureFailure(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)
	mockRepo.findErr = errors.New("database unavailable")

	bsUUID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 100,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:], nil, nil, 0x1234)

	assert.Equal(t, bssci.ResumeInfrastructureFailure, outcome.Disposition)
	require.Error(t, outcome.Err)
	assert.Nil(t, outcome.Previous, "no session is handed back on an infrastructure failure")
}

// TestHandleResume_VersionIncompatible verifies a resumable session persisted
// under an incompatible negotiated version is rejected as inconsistent so its
// stale state can be terminated (BSSCI rev1 §4.3).
func TestHandleResume_VersionIncompatible(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 100)

	bsUUID := [16]byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30}
	session := seedResumableSession(mockRepo, 55, 100, bsUUID, [16]byte{0x31}, 100, -50)
	incompatible := "2.0.0"
	session.ProtocolVersion = &incompatible

	testSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID:  100,
			NegotiatedVersion: "1.0.0",
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), testSession, bsUUID[:], nil, nil, 0x1234)

	assert.Equal(t, bssci.ResumeInconsistent, outcome.Disposition)
	require.NotNil(t, outcome.Previous, "the stale session is returned for termination")
}

// TestHydrateSessionNullVersionStaysEmpty verifies a legacy NULL
// protocol_version hydrates as empty rather than a fabricated version:
// resumeVersionCompatible treats empty as "cannot assert incompatibility" and
// the resume activation backfills the newly selected version.
func TestHydrateSessionNullVersionStaysEmpty(t *testing.T) {
	svc, _ := newResumeTestService(t, 700)

	dbSession := &models.BaseStationSession{
		ID:            46,
		CanResume:     true,
		Encoding:      "json",
		BaseStationID: 4,
		TenantID:      700,
		Status:        models.SessionStatusDisconnected,
		StartedAt:     time.Now(),
		// ProtocolVersion deliberately nil (legacy row)
	}

	result := svc.hydrateSessionFromDB(dbSession, 0x1234567890ABCDEF)

	assert.Empty(t, result.NegotiatedVersion,
		"a NULL persisted version must stay empty until compatibility evaluation")
	assert.Empty(t, result.ClientVersion)
	assert.True(t, resumeVersionCompatible(result.NegotiatedVersion, "1.0.0"),
		"an empty persisted version cannot assert incompatibility")
}

// TestPersistSessionFreshTakeoverDeletesStalePendingOperations verifies a fresh
// session that retires a stale active session also removes that session's
// pending operations: a surviving row is reissued if the retired session is
// ever resumed (BSSCI §3 "new session starts, discarding state").
func TestPersistSessionFreshTakeoverDeletesStalePendingOperations(t *testing.T) {
	svc, mockRepo, pendingOps := newTakeoverTestService(t, 900)

	staleScUUID := [16]byte{0x01, 0x02, 0x03}
	staleBsUUID := [16]byte{0x04, 0x05, 0x06}
	stale := seedResumableSession(mockRepo, 61, 900, staleBsUUID, staleScUUID, 10, -5)
	stale.Status = models.SessionStatusActive
	staleConnID := "connection-a"
	stale.ConnectionId = &staleConnID
	pendingOps.seed(61, -1, -2)
	pendingOps.seed(62, -7)

	baseStation := &basestation.BaseStation{ID: 1, TenantID: 900, EUI: [8]byte{0x70, 0xB3, 0xD5}}
	freshSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "connection-b",
			ResolvedTenantID:  900,
			SessionUUID:       []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
			BsUUID:            []byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30},
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
		},
	}

	require.NoError(t, svc.PersistSession(testutil.TestContext(), freshSession, baseStation, false, nil))

	assert.Equal(t, models.SessionStatusTerminated, stale.Status, "the stale session is retired")
	assert.Empty(t, pendingOps.operationIDs(61),
		"a retired session must leave no pending operations to reissue")
	assert.Equal(t, []int64{-7}, pendingOps.operationIDs(62),
		"only the retired session's operations are removed")
	require.NotZero(t, freshSession.DbSessionID)
	assert.Empty(t, pendingOps.operationIDs(freshSession.DbSessionID),
		"the new session starts without pending operations")
}

// TestPersistSessionFreshTakeoverAbortsWhenPendingDeleteFails verifies the
// activation aborts when the stale session's pending operations cannot be
// removed, so no new session is created on top of half-retired state.
func TestPersistSessionFreshTakeoverAbortsWhenPendingDeleteFails(t *testing.T) {
	svc, mockRepo, pendingOps := newTakeoverTestService(t, 900)

	stale := seedResumableSession(mockRepo, 63, 900, [16]byte{0x31}, [16]byte{0x41}, 10, -5)
	stale.Status = models.SessionStatusActive
	pendingOps.seed(63, -1)
	pendingOps.deleteErr = errors.New("database unavailable")

	baseStation := &basestation.BaseStation{ID: 1, TenantID: 900}
	freshSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "connection-b",
			ResolvedTenantID:  900,
			SessionUUID:       []byte{0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F, 0x60},
			BsUUID:            []byte{0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F, 0x70},
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
		},
	}

	err := svc.PersistSession(testutil.TestContext(), freshSession, baseStation, false, nil)

	require.Error(t, err)
	assert.Zero(t, freshSession.DbSessionID, "no session row is created when the retirement is incomplete")
	assert.Len(t, mockRepo.sessions, 1, "only the retired session row exists")
}

func TestMarkDisconnected_RetiredSessionStaysRetired(t *testing.T) {
	svc, mockRepo := newResumeTestService(t, 910)

	connectionID := "connection-a"
	protocolVersion := mioty.MIOTYProtocolVersion
	dbSession := &models.BaseStationSession{
		ID:              91,
		BaseStationID:   1,
		TenantID:        910,
		Status:          models.SessionStatusActive,
		CanResume:       true,
		ConnectionId:    &connectionID,
		Encoding:        "msgpack",
		ProtocolVersion: &protocolVersion,
		StartedAt:       time.Now().Add(-1 * time.Hour),
	}
	mockRepo.sessions[dbSession.ID] = dbSession

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:               connectionID,
			ResolvedTenantID: 910,
			DbSessionID:      dbSession.ID,
		},
	}

	require.NoError(t, svc.TerminateSession(testutil.TestContext(), session))
	require.Equal(t, models.SessionStatusTerminated, dbSession.Status)

	require.NoError(t, svc.MarkDisconnected(testutil.TestContext(), session))

	assert.Equal(t, models.SessionStatusTerminated, dbSession.Status,
		"a retired session must not be resurrected by its own connection's cleanup")
	assert.False(t, dbSession.CanResume,
		"a retired session must not become resumable again")
}

// TestPersistSessionFreshTakeoverRetiresLeftoverResumableSession verifies a
// fresh connect discards the base station's leftover resumable session and its
// pending operations, so a later resume is never handed that state, and that a
// second fresh connect finds nothing left to discard (BSSCI §3).
func TestPersistSessionFreshTakeoverRetiresLeftoverResumableSession(t *testing.T) {
	svc, mockRepo, pendingOps := newTakeoverTestService(t, 920)

	leftoverBsUUID := [16]byte{0x71, 0x72, 0x73}
	leftover := seedResumableSession(mockRepo, 71, 920, leftoverBsUUID, [16]byte{0x81}, 20, -9)
	leftoverConnID := "connection-a"
	leftover.ConnectionId = &leftoverConnID
	pendingOps.seed(71, -11, -12)
	pendingOps.seed(72, -13)

	baseStation := &basestation.BaseStation{ID: 1, TenantID: 920, EUI: [8]byte{0x70, 0xB3, 0xD5}}
	freshSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "connection-b",
			ResolvedTenantID:  920,
			SessionUUID:       []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
			BsUUID:            []byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30},
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
		},
	}

	require.NoError(t, svc.PersistSession(testutil.TestContext(), freshSession, baseStation, false, nil))

	assert.Equal(t, models.SessionStatusTerminated, leftover.Status, "the leftover resumable session is retired")
	assert.False(t, leftover.CanResume, "a retired session must not stay resumable")
	assert.Empty(t, pendingOps.operationIDs(71),
		"a retired session must leave no pending operations to reissue")
	assert.Equal(t, []int64{-13}, pendingOps.operationIDs(72),
		"only the retired session's operations are removed")

	resumeAttempt := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ResolvedTenantID: 920,
		},
	}
	outcome := svc.HandleResume(testutil.TestContext(), resumeAttempt, leftoverBsUUID[:], nil, nil, 0x70B3D5)
	assert.Equal(t, bssci.ResumeNoMatch, outcome.Disposition,
		"a discarded leftover must never be offered for resume")

	secondSession := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "connection-c",
			ResolvedTenantID:  920,
			SessionUUID:       []byte{0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F, 0x40},
			BsUUID:            []byte{0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F, 0x50},
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
		},
	}
	require.NoError(t, svc.PersistSession(testutil.TestContext(), secondSession, baseStation, false, nil),
		"a second fresh connect must not fail on already-discarded state")
	assert.NotEqual(t, freshSession.DbSessionID, secondSession.DbSessionID)
}

// TestPersistSessionFreshTakeoverAbortsOnIncompleteRetirement verifies the
// activation aborts when the leftover resumable state cannot be fully
// discarded, so no new session row is created on top of reissuable state.
func TestPersistSessionFreshTakeoverRetirementFailures(t *testing.T) {
	testCases := []struct {
		name            string
		arrange         func(repo *mockBaseStationSessionRepo, pendingOps *mockPendingOpsStore)
		wantAbort       bool
		leftoverStatus  models.BaseStationSessionStatus
		leftoverPending []int64
	}{
		{
			// A row left resumable would be reissued on a later resume, so the
			// activation must not proceed.
			name: "retirement fails",
			arrange: func(repo *mockBaseStationSessionRepo, _ *mockPendingOpsStore) {
				repo.terminateResumableErr = errors.New("database unavailable")
			},
			wantAbort:       true,
			leftoverStatus:  models.SessionStatusDisconnected,
			leftoverPending: []int64{-11},
		},
		{
			// The row is already retired, so its leftover operations are unreachable
			// and failing to delete them must not deny the new session.
			name: "pending operation delete fails",
			arrange: func(_ *mockBaseStationSessionRepo, pendingOps *mockPendingOpsStore) {
				pendingOps.deleteErr = errors.New("database unavailable")
			},
			wantAbort:       false,
			leftoverStatus:  models.SessionStatusTerminated,
			leftoverPending: []int64{-11},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, mockRepo, pendingOps := newTakeoverTestService(t, 930)

			leftover := seedResumableSession(mockRepo, 73, 930, [16]byte{0x91}, [16]byte{0xA1}, 20, -9)
			pendingOps.seed(73, -11)
			tc.arrange(mockRepo, pendingOps)

			baseStation := &basestation.BaseStation{ID: 1, TenantID: 930}
			freshSession := &bssci.Session{
				ProtocolSessionState: bssci.ProtocolSessionState{
					ID:                "connection-b",
					ResolvedTenantID:  930,
					SessionUUID:       []byte{0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF, 0xC0},
					BsUUID:            []byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF, 0xD0},
					NegotiatedVersion: mioty.MIOTYProtocolVersion,
					Encoding:          "msgpack",
				},
			}

			err := svc.PersistSession(testutil.TestContext(), freshSession, baseStation, false, nil)

			if tc.wantAbort {
				require.Error(t, err)
				assert.Zero(t, freshSession.DbSessionID, "no session row is created when a row stays resumable")
				assert.Len(t, mockRepo.sessions, 1, "only the leftover session row exists")
			} else {
				require.NoError(t, err)
				assert.NotZero(t, freshSession.DbSessionID, "the new session is created once the row is retired")
				assert.Len(t, mockRepo.sessions, 2, "the leftover row and the new session row exist")
			}
			assert.Equal(t, tc.leftoverStatus, leftover.Status)
			assert.Equal(t, tc.leftoverPending, pendingOps.operationIDs(73),
				"the leftover operations stay until they can be discarded")
		})
	}
}

// newResumeClaimConnection builds a resuming connection for the seeded row: the
// SC session UUID is the resume identity PersistSession looks the row up by.
func newResumeClaimConnection(connectionID string, tenantID int64, bsUUID, scUUID [16]byte) *bssci.Session {
	return &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                connectionID,
			ResolvedTenantID:  tenantID,
			SessionUUID:       append([]byte(nil), scUUID[:]...),
			BsUUID:            append([]byte(nil), bsUUID[:]...),
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
			IsResumed:         true,
		},
	}
}

// TestPersistSessionResumeClaimIsExclusive verifies the conditional resume
// activation: the first connection claims the disconnected row, and a second
// connection that passed HandleResume for the same row is refused instead of
// reactivating it and reissuing its pending operations a second time.
func TestPersistSessionResumeClaimIsExclusive(t *testing.T) {
	const tenantID = int64(920)
	bsUUID := [16]byte{0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF, 0xC0}
	scUUID := [16]byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF, 0xB0}

	svc, mockRepo := newResumeTestService(t, tenantID)
	row := seedResumableSession(mockRepo, 92, tenantID, bsUUID, scUUID, 4000, -2000)

	winner := newResumeClaimConnection("connection-a", tenantID, bsUUID, scUUID)
	winner.DbSessionID = row.ID
	require.NoError(t, svc.PersistSession(testutil.TestContext(), winner, nil, true, nil))
	require.Equal(t, models.SessionStatusActive, row.Status, "the first claimant activates the row")
	require.Equal(t, row.ID, winner.DbSessionID)

	loser := newResumeClaimConnection("connection-b", tenantID, bsUUID, scUUID)
	loser.DbSessionID = row.ID

	err := svc.PersistSession(testutil.TestContext(), loser, nil, true, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, bssci.ErrResumeAlreadyClaimed)
	assert.Zero(t, loser.DbSessionID,
		"the refused connection must own no row, so its teardown cannot retire the claimant's session")
	assert.Equal(t, models.SessionStatusActive, row.Status, "the claimant stays active")
	assert.True(t, row.CanResume, "the claimant stays resumable")
	if assert.NotNil(t, row.ConnectionId) {
		assert.Equal(t, "connection-a", *row.ConnectionId, "the claimant keeps the row's connection identity")
	}
}

// TestPersistSessionResumeRefusedWhenRowNotResumable verifies every non-claimable
// state of the row refuses activation and leaves it byte-for-byte as it was.
func TestPersistSessionResumeRefusedWhenRowNotResumable(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.BaseStationSession)
	}{
		{
			name: "activated by another connection",
			mutate: func(row *models.BaseStationSession) {
				row.Status = models.SessionStatusActive
			},
		},
		{
			name: "retired as inconsistent resume",
			mutate: func(row *models.BaseStationSession) {
				row.Status = models.SessionStatusTerminated
				row.CanResume = false
			},
		},
		{
			name: "resumability revoked while disconnected",
			mutate: func(row *models.BaseStationSession) {
				row.CanResume = false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const tenantID = int64(930)
			bsUUID := [16]byte{0xC1, 0xC2, 0xC3}
			scUUID := [16]byte{0xD1, 0xD2, 0xD3}

			svc, mockRepo := newResumeTestService(t, tenantID)
			row := seedResumableSession(mockRepo, 93, tenantID, bsUUID, scUUID, 4000, -2000)
			claimantConnID := "connection-a"
			row.ConnectionId = &claimantConnID
			tt.mutate(row)
			before := *row

			session := newResumeClaimConnection("connection-b", tenantID, bsUUID, scUUID)
			session.DbSessionID = row.ID

			err := svc.PersistSession(testutil.TestContext(), session, nil, true, nil)

			require.Error(t, err)
			assert.ErrorIs(t, err, bssci.ErrResumeAlreadyClaimed)
			assert.Zero(t, session.DbSessionID, "the refused connection must own no row")
			assert.Equal(t, before.Status, row.Status)
			assert.Equal(t, before.CanResume, row.CanResume)
			assert.Equal(t, before.ConnectionId, row.ConnectionId)
			assert.Equal(t, before.SnScOpId, row.SnScOpId)
			assert.Equal(t, before.SnBsOpId, row.SnBsOpId)
		})
	}
}

// TestPersistSessionResumeRefusedAfterFreshTakeoverDiscardedRow verifies the two
// gaps close together: a fresh connect that discards the row a resume claimant
// already accepted leaves that claimant unable to activate it, so the discarded
// pending operations are never reissued.
func TestPersistSessionResumeRefusedAfterFreshTakeoverDiscardedRow(t *testing.T) {
	svc, mockRepo, pendingOps := newTakeoverTestService(t, 940)

	bsUUID := [16]byte{0xE1, 0xE2, 0xE3}
	scUUID := [16]byte{0xF1, 0xF2, 0xF3}
	row := seedResumableSession(mockRepo, 94, 940, bsUUID, scUUID, 20, -9)
	pendingOps.seed(94, -11)

	claimant := newResumeClaimConnection("connection-a", 940, bsUUID, scUUID)
	outcome := svc.HandleResume(testutil.TestContext(), claimant, bsUUID[:], nil, nil, 0x70B3D5)
	require.Equal(t, bssci.ResumeCompatible, outcome.Disposition)

	baseStation := &basestation.BaseStation{ID: 1, TenantID: 940}
	fresh := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "connection-b",
			ResolvedTenantID:  940,
			SessionUUID:       []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x11},
			BsUUID:            []byte{0x11, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x12},
			NegotiatedVersion: mioty.MIOTYProtocolVersion,
			Encoding:          "msgpack",
		},
	}
	require.NoError(t, svc.PersistSession(testutil.TestContext(), fresh, baseStation, false, nil))

	claimant.DbSessionID = row.ID
	err := svc.PersistSession(testutil.TestContext(), claimant, nil, true, nil)

	require.ErrorIs(t, err, bssci.ErrResumeAlreadyClaimed)
	assert.Zero(t, claimant.DbSessionID)
	assert.Equal(t, models.SessionStatusTerminated, row.Status)
	assert.Empty(t, pendingOps.operationIDs(94))
}
