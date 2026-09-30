package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// sessionOwnershipFixture seeds one base station per scenario row.
type sessionOwnershipFixture struct {
	repo     *BaseStationSessionRepository
	tenantID int64
	orgID    uuid.UUID
}

func (f sessionOwnershipFixture) create(t *testing.T, bsID int64, connectionID string, owner models.EUI) *models.BaseStationSession {
	t.Helper()
	bsUUID, scUUID := uuid.New(), uuid.New()
	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:  bsID,
		TenantID:       f.tenantID,
		ConnectionId:   stringPtr(connectionID),
		CanResume:      true,
		OrganizationID: uuidPtr(f.orgID),
		Encoding:       mioty.EncodingMessagePack,
		ScEui:          owner,
	}
	copy(req.SnBsUuid[:], bsUUID[:])
	copy(req.SnScUuid[:], scUUID[:])
	created, err := f.repo.CreateSession(testutil.TestContext(), req)
	require.NoError(t, err)
	return created
}

// A restarting service center hands back resumable exactly the rows its
// previous process left active (and ownerless legacy rows), never another
// service center's rows or rows that already ended.
func TestDisconnectAbandonedSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	const tenantID = int64(140)
	thisSC := models.EUI{0x4B, 0x43, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	otherSC := models.EUI{0x4B, 0x43, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	orgID := uuid.New()
	createTestTenant(t, db, tenantID, "TestTenant140")
	createTestOrganization(t, db, orgID, tenantID, "TestOrg140")
	for i := int64(0); i < 4; i++ {
		createTestBaseStation(t, db, 40+i, 0x0102030405060740+uint64(i), tenantID, "TestBS-Abandoned")
	}
	cleanupSessionTestData(t, db, "TestAbandoned%")
	defer cleanupSessionTestData(t, db, "TestAbandoned%")

	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())
	f := sessionOwnershipFixture{repo: repo, tenantID: tenantID, orgID: orgID}
	ctx := testutil.TestContext()

	own := f.create(t, 40, "TestAbandoned-Own", thisSC)
	foreign := f.create(t, 41, "TestAbandoned-Foreign", otherSC)
	legacy := f.create(t, 42, "TestAbandoned-Legacy", thisSC)
	_, err := db.ExecContext(ctx, `UPDATE basestation_sessions SET sc_eui = NULL WHERE id = $1`, legacy.ID)
	require.NoError(t, err)
	ended := f.create(t, 43, "TestAbandoned-Ended", thisSC)
	endedAt := time.Now().Add(-time.Hour)
	require.NoError(t, repo.MarkDisconnected(ctx, tenantID, ended.ID, "TestAbandoned-Ended", endedAt))

	reconciled, err := repo.DisconnectAbandonedSessions(ctx, thisSC)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{own.ID, legacy.ID}, reconciled)

	for _, id := range []int64{own.ID, legacy.ID} {
		row, getErr := repo.GetSessionByID(ctx, tenantID, id)
		require.NoError(t, getErr)
		assert.True(t, row.CanResumeSession(), "an abandoned session is resumable again")
		assert.NotNil(t, row.EndedAt)
	}
	var legacyOwner []byte
	require.NoError(t, db.GetContext(ctx, &legacyOwner, `SELECT sc_eui FROM basestation_sessions WHERE id = $1`, legacy.ID))
	assert.Equal(t, thisSC[:], legacyOwner, "an ownerless row is claimed by the reconciling service center")

	foreignRow, err := repo.GetSessionByID(ctx, tenantID, foreign.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SessionStatusActive, foreignRow.Status, "another service center's session is not touched")

	endedRow, err := repo.GetSessionByID(ctx, tenantID, ended.ID)
	require.NoError(t, err)
	if assert.NotNil(t, endedRow.EndedAt) {
		assert.WithinDuration(t, endedAt, *endedRow.EndedAt, time.Second, "an ended session keeps its end time")
	}

	again, err := repo.DisconnectAbandonedSessions(ctx, thisSC)
	require.NoError(t, err)
	assert.Empty(t, again, "a repeated reconciliation finds nothing")
}
