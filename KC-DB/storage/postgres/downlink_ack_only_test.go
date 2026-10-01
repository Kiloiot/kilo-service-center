package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// TestDownlinkQueue_AcknowledgementOnlyPersists pins that a pure
// acknowledgement downlink (SCACI §3.10: "If user data is empty, a pure
// acknowledgement downlink is queued") persists on enqueue and on the pending
// edit path, carrying an empty payload.
func TestDownlinkQueue_AcknowledgementOnlyPersists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	const tenantID = int64(311)
	createTestTenant(t, sqlxDB, tenantID, "AckOnlyTenant")
	orgID := uuid.New()
	_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, 'ack-only-org')`, orgID, tenantID)
	require.NoError(t, err)

	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	downlinks := NewRepositories(db).Downlinks

	epEUIText := "0102030405060311"
	stored, err := downlinks.EnqueueDownlink(t.Context(), &storage.DownlinkMessage{
		EPEUI:          epEUIText,
		TenantID:       "311",
		OrganizationID: &orgID,
		Status:         mioty.DLQueueStatusPending,
		QueID:          910311,
	}, testDownlinkLifetime)
	require.NoError(t, err, "an acknowledgement-only downlink must persist")

	var payload []byte
	require.NoError(t, sqlxDB.Get(&payload, `SELECT payload FROM downlink_queue WHERE id = $1`, stored.ID))
	assert.NotNil(t, payload)
	assert.Empty(t, payload)

	epEUI := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x03, 0x11}
	_, err = downlinks.UpdatePendingDownlink(t.Context(), tenantID, &orgID, epEUI, 910311, storage.DownlinkPatch{Priority: 2})
	require.NoError(t, err, "editing a pending downlink into an acknowledgement must persist")
}
