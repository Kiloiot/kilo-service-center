package adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const scaciAdapterTenant int64 = 401

var (
	scaciAdapterAcEUI = [8]byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x16, 0x01}
	scaciAdapterAC    = models.SCACIApplicationCenter{TenantID: scaciAdapterTenant, AcEUI: scaciAdapterAcEUI}
)

func scaciAdapterRequest(seed byte) *models.SCACISessionCreateRequest {
	req := &models.SCACISessionCreateRequest{TenantID: scaciAdapterTenant, AcEUI: scaciAdapterAcEUI, CanResume: true, NegotiatedVersion: "1.0.0"}
	for i := range req.SnAcUUID {
		req.SnAcUUID[i] = seed + byte(i)
		req.SnScUUID[i] = seed + byte(i) + 0x40
	}
	return req
}

// createFresh opens a session the way a fresh connect does (SCACI §1).
func createFresh(ctx context.Context, adapter *SCACISessionTransactionAdapter, seed byte) (*models.SCACISession, error) {
	var created *models.SCACISession
	err := adapter.Run(ctx, func(tx SCACISessionTx) error {
		if err := tx.RetirePriorSessions(ctx, scaciAdapterAC); err != nil {
			return err
		}
		session, err := tx.CreateSession(ctx, scaciAdapterRequest(seed))
		created = session
		return err
	})
	return created, err
}

// The retirement of the application center's earlier sessions and the
// insert of its new one commit together or not at all.
func TestSCACISessionTransactionAdapter_RetiresAndCreatesInOneTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := teststore.Setup(t)
	t.Cleanup(cleanup)
	ctx := testutil.TestContext()
	_, err := db.Exec(ctx, `INSERT INTO tenants (id, name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`,
		scaciAdapterTenant, "ScaciAdapterTenant")
	require.NoError(t, err)
	repos := postgres.NewRepositories(db)
	adapter := NewSCACISessionTransactionAdapter(db)

	first, err := createFresh(ctx, adapter, 0x10)
	require.NoError(t, err)
	second, err := createFresh(ctx, adapter, 0x50)
	require.NoError(t, err)

	retired, err := repos.SCACISessions.GetSessionByID(ctx, scaciAdapterTenant, first.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SCACISessionStatusTerminated, retired.Status, "the new session discards the previous one")

	err = adapter.Run(ctx, func(tx SCACISessionTx) error {
		require.NoError(t, tx.RetirePriorSessions(ctx, scaciAdapterAC))
		_, createErr := tx.CreateSession(ctx, scaciAdapterRequest(0x90))
		require.NoError(t, createErr)
		return storage.ErrInvalidInput
	})
	require.ErrorIs(t, err, storage.ErrInvalidInput)

	kept, err := repos.SCACISessions.GetSessionByID(ctx, scaciAdapterTenant, second.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SCACISessionStatusActive, kept.Status, "a failed creation retires nothing")
	_, err = repos.SCACISessions.GetSessionByAcUUID(ctx, scaciAdapterAC, scaciAdapterRequest(0x90).SnAcUUID)
	assert.ErrorIs(t, err, storage.ErrNotFound, "a failed creation inserts nothing")
}
