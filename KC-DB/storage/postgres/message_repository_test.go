package postgres

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// CreateULDataMessage Tenant Validation Tests (BSSCI §5.10.1 fix)
// Defense-in-depth: validates tenant_id at repository layer before DB write
// ============================================================================

// TestCreateULDataMessage_RejectsZeroTenant verifies storage.ErrInvalidTenantID sentinel
// is returned when tenant_id is 0 (defense against upstream bugs).
func TestCreateULDataMessage_RejectsZeroTenant(t *testing.T) {
	// Unit test - no DB needed since validation happens before any DB operation
	repo := &MessageRepository{clock: clock.SystemClock{}, db: nil} // nil DB is fine - we fail before using it
	ctx := testutil.TestContext()

	msg := &mioty.ULDataMessage{
		TenantID:  0, // Invalid - triggers validation
		EpEui:     uint64(0x0011223344556677),
		BsEui:     uint64(0x0022334455667788),
		RxTime:    time.Now().UnixNano(),
		PacketCnt: 1,
		SNR:       10.0,
		RSSI:      -50.0,
	}

	err := insertULDataMessage(ctx, repo.db, logger.Get(), msg, time.Now())

	require.Error(t, err, "insertULDataMessage must reject zero tenant_id")
	assert.ErrorIs(t, err, storage.ErrInvalidTenantID, "Error must be ErrInvalidTenantID sentinel")
	assert.Contains(t, err.Error(), "got 0", "Error message must include the invalid value")
}

// TestCreateULDataMessage_RejectsNegativeTenant verifies storage.ErrInvalidTenantID sentinel
// is returned when tenant_id is negative (defense against signed overflow bugs).
func TestCreateULDataMessage_RejectsNegativeTenant(t *testing.T) {
	// Unit test - no DB needed since validation happens before any DB operation
	repo := &MessageRepository{clock: clock.SystemClock{}, db: nil} // nil DB is fine - we fail before using it
	ctx := testutil.TestContext()

	msg := &mioty.ULDataMessage{
		TenantID:  -1, // Invalid - triggers validation
		EpEui:     uint64(0x0011223344556677),
		BsEui:     uint64(0x0022334455667788),
		RxTime:    time.Now().UnixNano(),
		PacketCnt: 1,
		SNR:       10.0,
		RSSI:      -50.0,
	}

	err := insertULDataMessage(ctx, repo.db, logger.Get(), msg, time.Now())

	require.Error(t, err, "insertULDataMessage must reject negative tenant_id")
	assert.ErrorIs(t, err, storage.ErrInvalidTenantID, "Error must be ErrInvalidTenantID sentinel")
	assert.Contains(t, err.Error(), "got -1", "Error message must include the invalid value")
}

// TestInsertULDataMessage_RequestTenantIsTheOwner verifies both tenant columns
// take the request's tenant, which the ingest resolved as the endpoint owner;
// the insert never re-derives ownership on its own.
func TestInsertULDataMessage_RequestTenantIsTheOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)

	const ownerTenantID, otherTenantID = int64(200), int64(300)
	createTestTenant(t, db, ownerTenantID, "OwnerTenant")
	createTestTenant(t, db, otherTenantID, "OtherTenant")
	epEui := uint64(0x0011223344556699)
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI:         epEui,
		Name:          "OwnedEndpoint",
		TenantID:      otherTenantID,
		OwnerTenantID: otherTenantID,
	})

	msg := &mioty.ULDataMessage{
		TenantID:    ownerTenantID,
		EpEui:       epEui,
		BsEui:       uint64(0x0022334455667788),
		CommandType: mioty.CmdULData,
		RxTime:      time.Now().UnixNano(),
		PacketCnt:   1,
	}
	require.NoError(t, insertULDataMessage(testutil.TestContext(), db, logger.Get(), msg, time.Now()))

	var tenantID, ownerTenant int64
	require.NoError(t, db.QueryRow(`SELECT tenant_id, owner_tenant_id FROM messages WHERE id = $1`, msg.ID).Scan(&tenantID, &ownerTenant))
	assert.Equal(t, ownerTenantID, tenantID)
	assert.Equal(t, ownerTenantID, ownerTenant, "the owner is the request's tenant, not a second lookup")
}
