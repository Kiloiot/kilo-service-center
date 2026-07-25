package postgres

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerminateResumableSessions verifies a fresh takeover retires exactly the
// base station's leftover resumable rows: the active row, a non-resumable
// disconnected row, another base station's resumable row, and another tenant's
// row on the same base station all stay untouched, and a repeated sweep is a
// no-op.
func TestTerminateResumableSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	const tenantID = int64(120)
	const otherTenantID = int64(121)
	const baseStationID = int64(12)
	const otherBaseStationID = int64(13)
	const baseStationEUI = uint64(0x0102030405060712)
	const otherBaseStationEUI = uint64(0x0102030405060713)

	orgID := uuid.New()
	createTestTenant(t, db, tenantID, "TestTenant120")
	createTestTenant(t, db, otherTenantID, "TestTenant121")
	createTestOrganization(t, db, orgID, tenantID, "TestOrg")
	createTestBaseStation(t, db, baseStationID, baseStationEUI, tenantID, "TestBS-RetireResumable")
	createTestBaseStation(t, db, otherBaseStationID, otherBaseStationEUI, tenantID, "TestBS-RetireResumableOther")

	cleanupSessionTestData(t, db, "TestRetireResumable%")
	defer cleanupSessionTestData(t, db, "TestRetireResumable%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	newUUIDPair := func() ([16]byte, [16]byte) {
		var bsBytes, scBytes [16]byte
		bsUUID := uuid.New()
		scUUID := uuid.New()
		copy(bsBytes[:], bsUUID[:])
		copy(scBytes[:], scUUID[:])
		return bsBytes, scBytes
	}

	createSession := func(bsID, ownerTenantID int64, connectionID string) (*models.BaseStationSession, [16]byte) {
		bsBytes, scBytes := newUUIDPair()
		created, err := repo.CreateSession(ctx, &models.BaseStationSessionCreateRequest{
			BaseStationID:  bsID,
			TenantID:       ownerTenantID,
			SnBsUuid:       bsBytes,
			SnScUuid:       scBytes,
			ConnectionId:   stringPtr(connectionID),
			RemoteAddr:     stringPtr("192.168.1.120"),
			CanResume:      true,
			OrganizationID: uuidPtr(orgID),
			Encoding:       bssci.EncodingMessagePack,
		})
		require.NoError(t, err)
		return created, bsBytes
	}

	disconnectedAt := time.Now().Add(-time.Minute)

	// Only one active row per base station is allowed, so every row is
	// disconnected before the next one is created
	leftoverA, leftoverABsUUID := createSession(baseStationID, tenantID, "TestRetireResumable-Conn-A")
	require.NoError(t, repo.MarkDisconnected(ctx, tenantID, leftoverA.ID, "TestRetireResumable-Conn-A", disconnectedAt))

	leftoverB, _ := createSession(baseStationID, tenantID, "TestRetireResumable-Conn-B")
	require.NoError(t, repo.MarkDisconnected(ctx, tenantID, leftoverB.ID, "TestRetireResumable-Conn-B", disconnectedAt))

	nonResumable, _ := createSession(baseStationID, tenantID, "TestRetireResumable-Conn-NonResumable")
	require.NoError(t, repo.MarkDisconnected(ctx, tenantID, nonResumable.ID, "TestRetireResumable-Conn-NonResumable", disconnectedAt))
	canResume := false
	require.NoError(t, repo.UpdateSession(ctx, tenantID, nonResumable.ID, &models.BaseStationSessionUpdateRequest{
		CanResume: &canResume,
	}))

	otherTenant, _ := createSession(baseStationID, otherTenantID, "TestRetireResumable-Conn-OtherTenant")
	require.NoError(t, repo.MarkDisconnected(ctx, otherTenantID, otherTenant.ID, "TestRetireResumable-Conn-OtherTenant", disconnectedAt))

	otherStation, otherStationBsUUID := createSession(otherBaseStationID, tenantID, "TestRetireResumable-Conn-OtherStation")
	require.NoError(t, repo.MarkDisconnected(ctx, tenantID, otherStation.ID, "TestRetireResumable-Conn-OtherStation", disconnectedAt))

	active, _ := createSession(baseStationID, tenantID, "TestRetireResumable-Conn-Active")

	retired, err := repo.TerminateResumableSessions(ctx, tenantID, baseStationID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{leftoverA.ID, leftoverB.ID}, retired,
		"only the base station's resumable rows are retired")

	for _, retiredID := range []int64{leftoverA.ID, leftoverB.ID} {
		row, getErr := repo.GetSessionByID(ctx, tenantID, retiredID)
		require.NoError(t, getErr)
		assert.Equal(t, models.SessionStatusTerminated, row.Status, "retired row must be terminated")
		assert.False(t, row.CanResume, "retired row must not stay resumable")
		if assert.NotNil(t, row.EndedAt) {
			assert.WithinDuration(t, disconnectedAt, *row.EndedAt, time.Second,
				"the original disconnect timestamp is preserved")
		}
	}

	orphan, err := repo.FindResumableSession(ctx, tenantID, bsEUIToBytes(baseStationEUI), leftoverABsUUID)
	require.NoError(t, err)
	assert.Nil(t, orphan, "a retired leftover must never be offered for resume")

	activeRow, err := repo.GetSessionByID(ctx, tenantID, active.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SessionStatusActive, activeRow.Status, "the active row is not touched by the sweep")
	assert.True(t, activeRow.CanResume)

	nonResumableRow, err := repo.GetSessionByID(ctx, tenantID, nonResumable.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SessionStatusDisconnected, nonResumableRow.Status,
		"a disconnected row that cannot resume is not retired")

	otherTenantRow, err := repo.GetSessionByID(ctx, otherTenantID, otherTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SessionStatusDisconnected, otherTenantRow.Status, "another tenant's row is not retired")
	assert.True(t, otherTenantRow.CanResume)

	otherStationResumable, err := repo.FindResumableSession(ctx, tenantID, bsEUIToBytes(otherBaseStationEUI), otherStationBsUUID)
	require.NoError(t, err)
	require.NotNil(t, otherStationResumable, "another base station's resumable row is not retired")
	assert.Equal(t, otherStation.ID, otherStationResumable.ID)

	repeated, err := repo.TerminateResumableSessions(ctx, tenantID, baseStationID)
	require.NoError(t, err, "a repeated sweep must not fail")
	assert.Empty(t, repeated, "nothing is left to retire")
}
