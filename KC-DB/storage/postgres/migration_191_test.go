package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateDownlinkSentAfterExpiry: migration 000191 records when the holding
// station first reported sent a downlink already reported expired.
func validateDownlinkSentAfterExpiry(t *testing.T, db *sql.DB) {
	var dataType, nullable string
	require.NoError(t, db.QueryRow(`SELECT data_type, is_nullable FROM information_schema.columns WHERE table_name = 'downlink_queue' AND column_name = 'sent_after_expiry_at'`).Scan(&dataType, &nullable))
	assert.Equal(t, "timestamp with time zone", dataType)
	assert.Equal(t, "YES", nullable, "only a contradicted expiry records it")
}
