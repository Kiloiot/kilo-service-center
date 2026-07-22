package bssciservices

import (
	"errors"
	"testing"
	"time"

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
	mockRepo := newMockBaseStationSessionRepo()
	svc := NewSessionService(
		mockRepo,
		&mockBaseStationRepo{},
		&mockSystemEventStore{},
		tenantID,
		logger.NewNop(),
	).(*sessionService)
	return svc, mockRepo
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
