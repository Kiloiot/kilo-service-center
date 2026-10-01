package postgres

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedSessionRow creates a tenant, base station, and session, returning the
// tenant and session ids.
func seedSessionRow(t *testing.T, db *sqlx.DB) (tenantID, sessionID int64) {
	t.Helper()
	require.NoError(t, db.QueryRow(`
		INSERT INTO tenants (name, status, created_at, updated_at)
		VALUES ('session-update', 'active', NOW(), NOW()) RETURNING id`).Scan(&tenantID))

	var bsID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url)
		VALUES ($1, $2, 'session-update-bs', 'bssci', 'bssci://test') RETURNING id`,
		tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 9}).Scan(&bsID))

	require.NoError(t, db.QueryRow(`
		INSERT INTO basestation_sessions (
			basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id,
			status, can_resume, encoding, started_at, ended_at
		) VALUES ($1, $2, $3, $4, 5, -5, 'active', true, 'msgpack', NOW(), NOW()) RETURNING id`,
		bsID, tenantID, make([]byte, 16), make([]byte, 16)).Scan(&sessionID))
	return tenantID, sessionID
}

func TestUpdateSession_PartialUpdate(t *testing.T) {
	db := SetupTestDB(t)
	tenantID, sessionID := seedSessionRow(t, db)
	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())

	newBsOp := int64(42)
	require.NoError(t, repo.UpdateSession(testutil.TestContext(), tenantID, sessionID,
		&models.BaseStationSessionUpdateRequest{SnBsOpId: &newBsOp}))

	var bsOp, scOp int64
	var status string
	require.NoError(t, db.QueryRow(`
		SELECT sn_bs_op_id, sn_sc_op_id, status FROM basestation_sessions WHERE id = $1`,
		sessionID).Scan(&bsOp, &scOp, &status))
	assert.Equal(t, int64(42), bsOp, "the requested field changes")
	assert.Equal(t, int64(-5), scOp, "unrequested fields stay untouched")
	assert.Equal(t, "active", status)
}

func TestUpdateSession_ClearEndedAt(t *testing.T) {
	db := SetupTestDB(t)
	tenantID, sessionID := seedSessionRow(t, db)
	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())

	require.NoError(t, repo.UpdateSession(testutil.TestContext(), tenantID, sessionID,
		&models.BaseStationSessionUpdateRequest{ClearEndedAt: true}))

	var endedAt *time.Time
	require.NoError(t, db.QueryRow(`
		SELECT ended_at FROM basestation_sessions WHERE id = $1`, sessionID).Scan(&endedAt))
	assert.Nil(t, endedAt, "ClearEndedAt must null the column")
}

func TestUpdateSession_SetAndClearEndedAtConflict(t *testing.T) {
	db := SetupTestDB(t)
	tenantID, sessionID := seedSessionRow(t, db)
	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())

	now := time.Now()
	err := repo.UpdateSession(testutil.TestContext(), tenantID, sessionID,
		&models.BaseStationSessionUpdateRequest{EndedAt: &now, ClearEndedAt: true})
	require.Error(t, err, "setting and clearing ended_at at once must be rejected")

	var endedAt *time.Time
	require.NoError(t, db.QueryRow(`
		SELECT ended_at FROM basestation_sessions WHERE id = $1`, sessionID).Scan(&endedAt))
	assert.NotNil(t, endedAt, "a rejected request must not mutate the row")
}

func TestUpdateSession_EmptyUpdateRejected(t *testing.T) {
	db := SetupTestDB(t)
	tenantID, sessionID := seedSessionRow(t, db)
	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())

	err := repo.UpdateSession(testutil.TestContext(), tenantID, sessionID,
		&models.BaseStationSessionUpdateRequest{})
	require.Error(t, err, "an update with no fields must be rejected")

	require.Error(t, repo.UpdateSession(testutil.TestContext(), tenantID, sessionID, nil),
		"a nil request must be rejected")
}

func TestUpdateSession_WrongTenantIsNotFound(t *testing.T) {
	db := SetupTestDB(t)
	_, sessionID := seedSessionRow(t, db)
	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())

	newBsOp := int64(7)
	err := repo.UpdateSession(testutil.TestContext(), 999999, sessionID,
		&models.BaseStationSessionUpdateRequest{SnBsOpId: &newBsOp})
	require.Error(t, err, "a session belonging to another tenant must not be updatable")
}
