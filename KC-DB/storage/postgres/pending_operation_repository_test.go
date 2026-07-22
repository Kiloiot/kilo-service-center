package postgres

import (
	"encoding/json"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetBySession_NullMetadataScans verifies rows persisted without metadata
// (a nullable column) load during session resume instead of failing the scan.
func TestGetBySession_NullMetadataScans(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping container test in short mode")
	}
	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	ctx := testutil.TestContext()

	var tenantID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO tenants (name, status, created_at, updated_at)
		VALUES ('pending-null-md', 'active', NOW(), NOW()) RETURNING id`).Scan(&tenantID))

	var bsID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url)
		VALUES ($1, $2, 'pending-null-md-bs', 'bssci', 'bssci://test') RETURNING id`,
		tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 7}).Scan(&bsID))

	var sessionID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestation_sessions (
			basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id,
			status, can_resume, encoding, started_at
		) VALUES ($1, $2, $3, $4, 0, 0, 'disconnected', true, 'msgpack', NOW()) RETURNING id`,
		bsID, tenantID, make([]byte, 16), make([]byte, 16)).Scan(&sessionID))

	repo := NewPendingOperationRepository(db, logger.NewNop())
	require.NoError(t, repo.Create(ctx, &interfaces.PendingOperationRequest{
		SessionID:     sessionID,
		OperationID:   -1,
		OperationType: "status",
		OperationData: json.RawMessage(`{"command":"status","opId":-1}`),
	}), "create pending operation without metadata")

	ops, err := repo.GetBySession(ctx, sessionID)
	require.NoError(t, err, "NULL metadata must scan during resume load")
	require.Len(t, ops, 1)
	assert.Nil(t, ops[0].Metadata, "absent metadata loads as nil")
	assert.Equal(t, int64(-1), ops[0].OperationID)
}
