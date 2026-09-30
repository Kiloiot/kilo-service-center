package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// columnComment reads the comment on a column of the public schema.
func (h *migrationHarness) columnComment(table, column string) string {
	h.t.Helper()
	var comment sql.NullString
	require.NoError(h.t, h.db.QueryRow(`SELECT col_description(c.oid, a.attnum)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid
		WHERE n.nspname = 'public' AND c.relname = $1 AND a.attname = $2`, table, column).Scan(&comment))
	return comment.String
}

// The recorded time survives the rename to attachment_changed_at, which now
// also records an attach decision that kept the status. The downgrade renames
// it back with its earlier comment, and its guard refuses while another
// column already holds the earlier name.
func TestMigration170RenamesTheAttachmentChangeTime(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(169)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000a170")
	detachTime := statusChangeDetachTime
	decided := h.seedEndpoint(tenantID, "70b3d5677011a170", "detached", &detachTime)
	h.exec(`UPDATE endpoints SET status_changed_at = to_timestamp($1) WHERE id = $2`, statusChangeDetachUnix, decided)

	h.migrateTo(170)

	assert.False(t, h.columnExists("endpoints", "status_changed_at"), "the column is renamed")
	assert.Equal(t, statusChangeDetachUnix, h.queryInt(`SELECT extract(epoch FROM attachment_changed_at)::bigint FROM endpoints WHERE id = $1`, decided),
		"the recorded time survives the rename")
	assert.Contains(t, h.columnComment("endpoints", "attachment_changed_at"), "attached the endpoint or changed its ep_status")

	h.exec(`ALTER TABLE endpoints ADD COLUMN status_changed_at TIMESTAMPTZ`)
	err := h.m.Migrate(169)
	require.ErrorContains(t, err, "endpoints.status_changed_at already exists")
	require.NoError(t, h.m.Force(170))
	h.exec(`ALTER TABLE endpoints DROP COLUMN status_changed_at`)

	h.migrateTo(169)

	assert.False(t, h.columnExists("endpoints", "attachment_changed_at"), "the downgrade renames the column back")
	assert.Equal(t, statusChangeDetachUnix, h.queryInt(`SELECT extract(epoch FROM status_changed_at)::bigint FROM endpoints WHERE id = $1`, decided))
	assert.Contains(t, h.columnComment("endpoints", "status_changed_at"), "last changed ep_status")
	h.migrateTo(170)
}
