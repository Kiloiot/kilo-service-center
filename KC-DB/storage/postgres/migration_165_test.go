package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigration165ClaimsDownlinkWindowsPerTelegram(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(164)

	h.migrateTo(165)

	for _, table := range []string{"messages", "messages_archive"} {
		assert.True(t, h.columnExists(table, "dl_window_claimed"), "%s.dl_window_claimed", table)
	}

	h.migrateTo(164)
	for _, table := range []string{"messages", "messages_archive"} {
		assert.False(t, h.columnExists(table, "dl_window_claimed"), "%s.dl_window_claimed is dropped", table)
	}
	h.migrateTo(165)
}
