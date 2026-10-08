package postgres

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// validateDownlinkCommandRef: migration 000187 gives downlink_queue a
// nullable ref bounded like the MQTT command handler bounds it.
func validateDownlinkCommandRef(t *testing.T, db *sql.DB) {
	var dataType, nullable string
	require.NoError(t, db.QueryRow(`SELECT data_type, is_nullable FROM information_schema.columns
		WHERE table_name = 'downlink_queue' AND column_name = 'ref'`).Scan(&dataType, &nullable))
	assert.Equal(t, "text", dataType)
	assert.Equal(t, "YES", nullable)
	var definition string
	require.NoError(t, db.QueryRow(`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = 'downlink_queue_ref_length'`).Scan(&definition))
	assert.Equal(t, "CHECK (((ref IS NULL) OR ((octet_length(ref) >= 1) AND (octet_length(ref) <= 128))))", definition)
}

// TestMigration187BoundsTheRefLikeTheCommandHandler: the column accepts a
// ref of storage.MaxDownlinkRefBytes and refuses a longer or an empty one.
func TestMigration187BoundsTheRefLikeTheCommandHandler(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(187)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000187a")
	orgID := h.seedOrganization(tenantID)
	insert := func(queID int64, ref string) error {
		_, err := h.db.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at, ref)
			VALUES (decode('70b3d59cd0000187', 'hex'), $1, $2, '\x01', 'pending', $3, NULL, $4)`, tenantID, orgID, queID, ref)
		return err
	}

	require.NoError(t, insert(187001, strings.Repeat("r", storage.MaxDownlinkRefBytes)))
	require.Error(t, insert(187002, strings.Repeat("r", storage.MaxDownlinkRefBytes+1)))
	require.Error(t, insert(187003, ""), "no ref is NULL, never an empty one")
}
