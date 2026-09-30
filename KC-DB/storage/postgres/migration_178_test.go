package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	unknownURLStationEUI       = "70b3d59cd0017801"
	sqlInsertStationWithoutURL = `INSERT INTO basestations (bs_eui, name, tenant_id, connection_type)
		VALUES (decode($1, 'hex'), $2, $3, 'bssci')`
)

func (h *migrationHarness) bssciURLCheckExists() bool {
	return h.queryInt(`SELECT count(*) FROM pg_constraint
		WHERE conrelid = 'basestations'::regclass AND conname = 'check_bssci_config'`) == 1
}

// TestMigration178AcceptsABSSCIStationWithoutServiceCenterURL: before 178 a
// BSSCI station without a URL is refused, after it the row is stored. The down
// migration keeps the check dropped while such a row exists and restores it
// once every BSSCI station has a URL.
func TestMigration178AcceptsABSSCIStationWithoutServiceCenterURL(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(177)
	tenantID, _ := h.seedTenantAndBaseStation(backfillStationEUI)

	_, err := h.db.Exec(sqlInsertStationWithoutURL, unknownURLStationEUI, "bs-"+unknownURLStationEUI, tenantID)
	require.Error(t, err, "177 refuses a BSSCI station without a URL")
	assert.Contains(t, err.Error(), "check_bssci_config")

	h.migrateTo(178)
	h.exec(sqlInsertStationWithoutURL, unknownURLStationEUI, "bs-"+unknownURLStationEUI, tenantID)
	validateBSSCIServiceCenterURLOptional(t, h.db)

	h.migrateTo(177)
	assert.False(t, h.bssciURLCheckExists(), "the check stays dropped while a BSSCI station has no URL")
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM basestations WHERE service_center_url IS NULL`),
		"the down migration keeps the station")

	h.migrateTo(178)
	h.exec(`DELETE FROM basestations WHERE service_center_url IS NULL`)
	h.migrateTo(177)
	assert.True(t, h.bssciURLCheckExists(), "the check returns once every BSSCI station has a URL")
}
