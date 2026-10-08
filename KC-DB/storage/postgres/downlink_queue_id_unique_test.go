package postgres

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// TestEnqueueDownlink_DuplicateQueueIDIsSentinel pins that a second row with
// the same service center que_id surfaces as storage.ErrDownlinkQueueIDTaken
// rather than a raw driver error, so the id allocator can draw a fresh one.
func TestEnqueueDownlink_DuplicateQueueIDIsSentinel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	const tenantID = int64(304)
	createTestTenant(t, sqlxDB, tenantID, "DuplicateQueueIDTenant")
	orgID := uuid.New()
	_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, 'dup-queue-org')`, orgID, tenantID)
	require.NoError(t, err)

	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}

	message := func() *storage.DownlinkMessage {
		return &storage.DownlinkMessage{
			EPEUI:          "0102030405060797",
			TenantID:       "304",
			OrganizationID: &orgID,
			Payload:        []byte{0x42},
			Priority:       1,
			Status:         mioty.DLQueueStatusPending,
			QueID:          900003,
		}
	}
	_, err = NewRepositories(db).Downlinks.EnqueueDownlink(t.Context(), message(), testDownlinkLifetime)
	require.NoError(t, err)

	_, err = NewRepositories(db).Downlinks.EnqueueDownlink(t.Context(), message(), testDownlinkLifetime)
	require.ErrorIs(t, err, storage.ErrDownlinkQueueIDTaken)
	require.NotErrorIs(t, err, storage.ErrDuplicateKey)
}
