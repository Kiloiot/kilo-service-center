package postgres

import (
	"fmt"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarkDisconnected_RetiredSessionStaysRetired verifies a session retired by
// TerminateSession cannot be flipped back to disconnected and resumable by its
// own connection's later cleanup, while an active session that simply loses its
// connection still becomes resumable.
func TestMarkDisconnected_RetiredSessionStaysRetired(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	const tenantID = int64(110)
	const baseStationID = int64(11)
	const baseStationEUI = uint64(0x0102030405060711)

	orgID := uuid.New()
	createTestTenant(t, db, tenantID, "TestTenant110")
	createTestOrganization(t, db, orgID, tenantID, "TestOrg")
	createTestBaseStation(t, db, baseStationID, baseStationEUI, tenantID, "TestBS-ResumeGuard")

	cleanupSessionTestData(t, db, "TestResumeGuard%")
	defer cleanupSessionTestData(t, db, "TestResumeGuard%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	staleBsUUID := uuid.New()
	staleScUUID := uuid.New()
	var staleBsBytes, staleScBytes [16]byte
	copy(staleBsBytes[:], staleBsUUID[:])
	copy(staleScBytes[:], staleScUUID[:])

	stale, err := repo.CreateSession(ctx, &models.BaseStationSessionCreateRequest{
		BaseStationID:  baseStationID,
		TenantID:       tenantID,
		SnBsUuid:       staleBsBytes,
		SnScUuid:       staleScBytes,
		ConnectionId:   stringPtr("TestResumeGuard-Conn-Stale"),
		RemoteAddr:     stringPtr("192.168.1.110"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingMessagePack,
	})
	require.NoError(t, err)

	// A fresh connect discards the stale session before creating its own row
	require.NoError(t, repo.TerminateSession(ctx, tenantID, stale.ID))

	// The stale connection's own cleanup runs afterwards with its original connection ID
	require.NoError(t, repo.MarkDisconnected(ctx, tenantID, stale.ID, "TestResumeGuard-Conn-Stale", time.Now()))

	retired, err := repo.GetSessionByID(ctx, tenantID, stale.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SessionStatusTerminated, retired.Status, "terminated session must stay terminated")
	assert.False(t, retired.CanResume, "terminated session must stay non-resumable")

	orphan, err := repo.FindResumableSession(ctx, tenantID, bsEUIToBytes(baseStationEUI), staleBsBytes)
	require.NoError(t, err)
	assert.Nil(t, orphan, "terminated session must never be offered for resume")

	freshBsUUID := uuid.New()
	freshScUUID := uuid.New()
	var freshBsBytes, freshScBytes [16]byte
	copy(freshBsBytes[:], freshBsUUID[:])
	copy(freshScBytes[:], freshScUUID[:])

	fresh, err := repo.CreateSession(ctx, &models.BaseStationSessionCreateRequest{
		BaseStationID:  baseStationID,
		TenantID:       tenantID,
		SnBsUuid:       freshBsBytes,
		SnScUuid:       freshScBytes,
		ConnectionId:   stringPtr("TestResumeGuard-Conn-Fresh"),
		RemoteAddr:     stringPtr("192.168.1.111"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingMessagePack,
	})
	require.NoError(t, err)

	require.NoError(t, repo.MarkDisconnected(ctx, tenantID, fresh.ID, "TestResumeGuard-Conn-Fresh", time.Now()))

	resumable, err := repo.FindResumableSession(ctx, tenantID, bsEUIToBytes(baseStationEUI), freshBsBytes)
	require.NoError(t, err)
	require.NotNil(t, resumable, "an active session losing its connection must stay resumable")
	assert.Equal(t, fresh.ID, resumable.ID)
	assert.Equal(t, models.SessionStatusDisconnected, resumable.Status)
	assert.True(t, resumable.CanResume)
	assert.NotNil(t, resumable.EndedAt)
}

// TestActivateSessionIfResumable_ClaimsOnlyDisconnectedResumableRow verifies the
// conditional resume activation: exactly the disconnected, resumable row is
// claimed, and every other state matches zero rows and stays untouched, so two
// connections resuming the same session cannot both activate it.
func TestActivateSessionIfResumable_ClaimsOnlyDisconnectedResumableRow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	const tenantID = int64(112)

	orgID := uuid.New()
	createTestTenant(t, db, tenantID, "TestTenant112")
	createTestOrganization(t, db, orgID, tenantID, "TestOrg112")

	cleanupSessionTestData(t, db, "TestResumeClaim%")
	defer cleanupSessionTestData(t, db, "TestResumeClaim%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	tests := []struct {
		name        string
		prepare     func(t *testing.T, sessionID int64, connectionID string)
		wantClaimed bool
		wantStatus  models.BaseStationSessionStatus
	}{
		{
			name: "disconnected and resumable",
			prepare: func(t *testing.T, sessionID int64, connectionID string) {
				require.NoError(t, repo.MarkDisconnected(ctx, tenantID, sessionID, connectionID, time.Now()))
			},
			wantClaimed: true,
			wantStatus:  models.SessionStatusActive,
		},
		{
			name:        "already activated by another connection",
			prepare:     func(_ *testing.T, _ int64, _ string) {},
			wantClaimed: false,
			wantStatus:  models.SessionStatusActive,
		},
		{
			name: "retired",
			prepare: func(t *testing.T, sessionID int64, connectionID string) {
				require.NoError(t, repo.MarkDisconnected(ctx, tenantID, sessionID, connectionID, time.Now()))
				require.NoError(t, repo.TerminateSession(ctx, tenantID, sessionID))
			},
			wantClaimed: false,
			wantStatus:  models.SessionStatusTerminated,
		},
		{
			name: "disconnected but not resumable",
			prepare: func(t *testing.T, sessionID int64, connectionID string) {
				require.NoError(t, repo.MarkDisconnected(ctx, tenantID, sessionID, connectionID, time.Now()))
				notResumable := false
				require.NoError(t, repo.UpdateSession(ctx, tenantID, sessionID, &models.BaseStationSessionUpdateRequest{
					CanResume: &notResumable,
				}))
			},
			wantClaimed: false,
			wantStatus:  models.SessionStatusDisconnected,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A base station per case: the partial unique index allows one active row each
			baseStationID := int64(120 + i)
			baseStationEUI := uint64(0x0102030405060720) + uint64(i)
			createTestBaseStation(t, db, baseStationID, baseStationEUI, tenantID, fmt.Sprintf("TestBS-ResumeClaim-%d", i))

			bsUUID := uuid.New()
			scUUID := uuid.New()
			var bsBytes, scBytes [16]byte
			copy(bsBytes[:], bsUUID[:])
			copy(scBytes[:], scUUID[:])

			originalConnID := fmt.Sprintf("TestResumeClaim-Conn-%d-Original", i)
			created, err := repo.CreateSession(ctx, &models.BaseStationSessionCreateRequest{
				BaseStationID:  baseStationID,
				TenantID:       tenantID,
				SnBsUuid:       bsBytes,
				SnScUuid:       scBytes,
				ConnectionId:   stringPtr(originalConnID),
				RemoteAddr:     stringPtr("192.168.1.112"),
				CanResume:      true,
				OrganizationID: uuidPtr(orgID),
				Encoding:       bssci.EncodingMessagePack,
			})
			require.NoError(t, err)

			tt.prepare(t, created.ID, originalConnID)

			claimConnID := fmt.Sprintf("TestResumeClaim-Conn-%d-Claim", i)
			activeStatus := models.SessionStatusActive
			canResume := true
			protocolVersion := "1.0.0"
			bsOpID := int64(11)
			scOpID := int64(-11)
			claimed, err := repo.ActivateSessionIfResumable(ctx, tenantID, created.ID, &models.BaseStationSessionUpdateRequest{
				Status:          &activeStatus,
				CanResume:       &canResume,
				ClearEndedAt:    true,
				SnBsOpId:        &bsOpID,
				SnScOpId:        &scOpID,
				ConnectionId:    stringPtr(claimConnID),
				RemoteAddr:      stringPtr("192.168.1.113"),
				Encoding:        stringPtr(bssci.EncodingMessagePack),
				ProtocolVersion: &protocolVersion,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantClaimed, claimed)

			stored, err := repo.GetSessionByID(ctx, tenantID, created.ID)
			require.NoError(t, err)
			require.NotNil(t, stored)
			assert.Equal(t, tt.wantStatus, stored.Status)

			if !tt.wantClaimed {
				require.NotNil(t, stored.ConnectionId)
				assert.Equal(t, originalConnID, *stored.ConnectionId,
					"a refused claim must not overwrite the row's connection identity")
				assert.Equal(t, int64(0), stored.SnBsOpId, "a refused claim must not touch the counters")
				assert.Equal(t, int64(0), stored.SnScOpId, "a refused claim must not touch the counters")
				return
			}

			require.NotNil(t, stored.ConnectionId)
			assert.Equal(t, claimConnID, *stored.ConnectionId)
			assert.True(t, stored.CanResume)
			assert.Nil(t, stored.EndedAt, "the claim clears the end timestamp")
			assert.Equal(t, bsOpID, stored.SnBsOpId)
			assert.Equal(t, scOpID, stored.SnScOpId)
		})
	}
}
