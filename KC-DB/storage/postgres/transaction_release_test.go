package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// rollbackTestTenant owns every fixture of the transaction release tests.
const rollbackTestTenant = int64(830)

// installRejectingLegacyArchive creates a preserved pre-000139 archive whose
// updates always fail, so an EUI rename fails at its legacy-archive step.
func installRejectingLegacyArchive(t *testing.T, db *sqlx.DB, epEUI, bsEUI []byte) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE messages_archive_pre000139 (id BIGINT PRIMARY KEY, ep_eui BYTEA, bs_eui BYTEA)`,
		`CREATE FUNCTION reject_legacy_archive_update() RETURNS trigger AS $$
		 BEGIN RAISE EXCEPTION 'legacy archive rejects updates'; END $$ LANGUAGE plpgsql`,
		`CREATE TRIGGER reject_legacy_archive_update BEFORE UPDATE ON messages_archive_pre000139
		 FOR EACH ROW EXECUTE FUNCTION reject_legacy_archive_update()`,
	} {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}
	_, err := db.Exec(`INSERT INTO messages_archive_pre000139 (id, ep_eui, bs_eui) VALUES (1, $1, $2)`, epEUI, bsEUI)
	require.NoError(t, err)
}

// assertNoConnectionHeld proves a failed operation released its transaction
// instead of leaving it open until the request context ends.
func assertNoConnectionHeld(t *testing.T, db *sqlx.DB) {
	t.Helper()
	assert.Zero(t, db.Stats().InUse, "a failed transaction must roll back and release its connection")
}

func createRollbackBaseStation(t *testing.T, repo *BaseStationRepository, eui models.EUI, name string) {
	t.Helper()
	require.NoError(t, repo.Create(testutil.TestContext(), &models.BaseStation{
		EUI:              eui,
		TenantID:         rollbackTestTenant,
		Name:             name,
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}))
}

func TestBaseStationUpdateEUI_RollsBackWhenTheNewEUIIsTaken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	createTestTenant(t, db, rollbackTestTenant, "EUI Rollback Tenant")
	repo := NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
	taken := models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x83, 0x00, 0x01}
	renamed := models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x83, 0x00, 0x02}
	createRollbackBaseStation(t, repo, taken, "rollback-taken")
	createRollbackBaseStation(t, repo, renamed, "rollback-renamed")

	_, err := repo.UpdateEUI(testutil.TestContext(), rollbackTestTenant, renamed[:], taken[:])
	require.ErrorIs(t, err, storage.ErrAlreadyExists)
	assertNoConnectionHeld(t, db)
}

func TestBaseStationUpdateEUI_RollsBackWhenTheLegacyArchiveFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	createTestTenant(t, db, rollbackTestTenant, "EUI Rollback Tenant")
	repo := NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
	oldEUI := models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x83, 0x00, 0x03}
	newEUI := models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x83, 0x00, 0x04}
	createRollbackBaseStation(t, repo, oldEUI, "rollback-legacy")
	installRejectingLegacyArchive(t, db, make([]byte, 8), oldEUI[:])

	_, err := repo.UpdateEUI(testutil.TestContext(), rollbackTestTenant, oldEUI[:], newEUI[:])
	require.Error(t, err)
	assertNoConnectionHeld(t, db)

	var stored []byte
	require.NoError(t, db.Get(&stored, `SELECT bs_eui FROM basestations WHERE name = 'rollback-legacy'`))
	assert.Equal(t, oldEUI[:], stored, "the rename must not be committed")
}

func TestEndPointUpdateWithEUI_RollsBackWhenTheLegacyArchiveFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	createTestTenant(t, db, rollbackTestTenant, "EUI Rollback Tenant")
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	oldEUI := models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0x00, 0x83, 0x00, 0x05}
	endpoint := &models.EndPoint{
		EUI:      oldEUI,
		Name:     "rollback-endpoint",
		TenantID: rollbackTestTenant,
		EPClass:  "A",
		NwkSnKey: make([]byte, 16),
		Tags:     map[string]string{},
	}
	require.NoError(t, repo.Create(ctx, endpoint))
	installRejectingLegacyArchive(t, db, oldEUI[:], make([]byte, 8))

	reloaded, err := repo.GetByEUI(ctx, rollbackTestTenant, oldEUI[:])
	require.NoError(t, err)
	reloaded.EUI = models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0x00, 0x83, 0x00, 0x06}
	_, err = repo.UpdateWithEUI(ctx, rollbackTestTenant, oldEUI[:], reloaded)
	require.Error(t, err)
	assertNoConnectionHeld(t, db)

	var stored []byte
	require.NoError(t, db.Get(&stored, `SELECT ep_eui FROM endpoints WHERE name = 'rollback-endpoint'`))
	assert.Equal(t, oldEUI[:], stored, "the rename must not be committed")
}

func TestOrganizationDelete_RollsBackWhenTheOrganizationIsMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	createTestTenant(t, db, rollbackTestTenant, "EUI Rollback Tenant")
	repo := NewOrganizationRepository(db, logger.Get())

	err := repo.Delete(testutil.TestContext(), uuid.MustParse("99999999-9999-9999-9999-999999999999"), rollbackTestTenant)
	require.ErrorIs(t, err, storage.ErrNotFound)
	assertNoConnectionHeld(t, db)
}
