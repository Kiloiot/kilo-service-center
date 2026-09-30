package postgres

import (
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

const (
	pendingUpdateTenant   = int64(620)
	pendingUpdateOther    = int64(621)
	pendingUpdateEpEUI    = uint64(0x70B3D59CD0000077)
	pendingUpdateOtherEp  = uint64(0x70B3D59CD0000078)
	pendingUpdatePriority = float32(0.9)
	pendingUpdateFormat   = uint8(5)
)

func TestUpdatePendingDownlink_RewritesOnlyPendingRows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	createTestTenant(t, sqlxDB, pendingUpdateTenant, "PendingUpdateTenant")
	createTestTenant(t, sqlxDB, pendingUpdateOther, "PendingUpdateOther")
	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: pendingUpdateEpEUI, Name: "PendingUpdate", TenantID: pendingUpdateTenant})
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: pendingUpdateOtherEp, Name: "PendingUpdateOther", TenantID: pendingUpdateTenant})
	epEUI := euiBytes(pendingUpdateEpEUI)

	pendingID := insertDownlink(t, db, DownlinkInsertParams{EpEUI: pendingUpdateEpEUI, TenantID: pendingUpdateTenant, Status: mioty.DLQueueStatusPending})
	queuedID := insertDownlink(t, db, DownlinkInsertParams{EpEUI: pendingUpdateEpEUI, TenantID: pendingUpdateTenant, Status: mioty.DLQueueStatusQueued})
	repo := NewRepositories(db).Downlinks

	patch := storage.DownlinkPatch{
		Payloads: [][]byte{{0x01, 0x02}, {0x03}}, Priority: pendingUpdatePriority, CntDepend: true, PacketCnt: []int64{10, 11},
		Format: pendingUpdateFormat, ResponseExp: true, DlRxStatQry: true,
	}
	updated, err := repo.UpdatePendingDownlink(t.Context(), pendingUpdateTenant, nil, epEUI, pendingID, patch)
	require.NoError(t, err)
	assert.Equal(t, pendingID, updated.QueID)
	assert.Equal(t, mioty.DLQueueStatusPending, updated.Status)
	assert.Equal(t, pendingUpdatePriority, updated.Priority)
	assert.True(t, updated.CntDepend)
	assert.Equal(t, []int64{10, 11}, updated.PacketCntArray)
	assert.Equal(t, pendingUpdateFormat, updated.Format)
	assert.True(t, updated.ResponseExp)
	assert.True(t, updated.DlRxStatQry)
	assert.Equal(t, [][]byte{{0x01, 0x02}, {0x03}}, updated.UserData)

	_, err = repo.UpdatePendingDownlink(t.Context(), pendingUpdateTenant, nil, epEUI, queuedID, patch)
	assert.ErrorIs(t, err, storage.ErrDownlinkNotPending, "a row a base station already holds cannot be edited")

	_, err = repo.UpdatePendingDownlink(t.Context(), pendingUpdateOther, nil, epEUI, pendingID, patch)
	assert.ErrorIs(t, err, storage.ErrDownlinkNotFound, "another tenant cannot see the row")

	foreignOrg := uuid.New()
	_, err = repo.UpdatePendingDownlink(t.Context(), pendingUpdateTenant, &foreignOrg, epEUI, pendingID, patch)
	assert.ErrorIs(t, err, storage.ErrDownlinkNotFound, "an organization that does not own the row cannot edit it")
}

func euiBytes(eui uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, eui)
	return b
}

// A queue id belongs to one endpoint: another endpoint of the same tenant
// must neither change the row nor learn that the queue id exists.
func TestUpdatePendingDownlink_WrongEndpointLeavesTheRowUntouched(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	createTestTenant(t, sqlxDB, pendingUpdateTenant, "PendingUpdateTenant")
	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: pendingUpdateEpEUI, Name: "PendingUpdate", TenantID: pendingUpdateTenant})
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: pendingUpdateOtherEp, Name: "PendingUpdateOther", TenantID: pendingUpdateTenant})
	repo := NewRepositories(db).Downlinks

	pendingID := insertDownlink(t, db, DownlinkInsertParams{EpEUI: pendingUpdateEpEUI, TenantID: pendingUpdateTenant, Status: mioty.DLQueueStatusPending})
	tenant := strconv.FormatInt(pendingUpdateTenant, 10)
	before := queueRow(t, repo, pendingUpdateEpEUI, tenant, pendingID)

	patch := storage.DownlinkPatch{
		Payloads: [][]byte{{0x0F, 0x0E}, {0x0D}}, Priority: pendingUpdatePriority, CntDepend: true, PacketCnt: []int64{7, 8},
		Format: pendingUpdateFormat, ResponseExp: true, ResponsePrio: true, DlWindReq: true, ExpOnly: true, DlRxStatQry: true,
	}
	_, err := repo.UpdatePendingDownlink(t.Context(), pendingUpdateTenant, nil, euiBytes(pendingUpdateOtherEp), pendingID, patch)
	assert.ErrorIs(t, err, storage.ErrDownlinkNotFound, "a foreign endpoint's queue id reads as missing, not as not-pending")

	after := queueRow(t, repo, pendingUpdateEpEUI, tenant, pendingID)
	assert.Equal(t, before.Payload, after.Payload)
	assert.Equal(t, before.Priority, after.Priority)
	assert.Equal(t, before.CntDepend, after.CntDepend)
	assert.Equal(t, before.PacketCntArray, after.PacketCntArray)
	assert.Equal(t, before.Format, after.Format)
	assert.Equal(t, before.ResponseExp, after.ResponseExp)
	assert.Equal(t, before.ResponsePrio, after.ResponsePrio)
	assert.Equal(t, before.DlWindReq, after.DlWindReq)
	assert.Equal(t, before.ExpOnly, after.ExpOnly)
	assert.Equal(t, before.DlRxStatQry, after.DlRxStatQry)
	assert.Equal(t, before.UserData, after.UserData)
	assert.Equal(t, before.Status, after.Status)
	assert.True(t, before.UpdatedAt.Equal(after.UpdatedAt), "updated_at must not move")

	updated, err := repo.UpdatePendingDownlink(t.Context(), pendingUpdateTenant, nil, euiBytes(pendingUpdateEpEUI), pendingID, patch)
	require.NoError(t, err, "the owning endpoint still edits its row")
	assert.True(t, updated.ResponsePrio)
}

func queueRow(t *testing.T, repo *MIOTYDownlinkRepository, epEUI uint64, tenant string, queID int64) *storage.DownlinkMessage {
	t.Helper()
	rows, err := repo.GetDownlinkQueue(t.Context(), mioty.FormatEUI64(epEUI), tenant)
	require.NoError(t, err)
	for _, row := range rows {
		if row.QueID == queID {
			return row
		}
	}
	t.Fatalf("queue row %d not found for endpoint %s", queID, mioty.FormatEUI64(epEUI))
	return nil
}
