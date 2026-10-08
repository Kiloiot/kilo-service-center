package postgres

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// The Application Centers and the endpoint of the migration 184 fixture.
const (
	originEndpoint      = "70B3D59CD0000184"
	originQueuer        = "70b3d59cd00a0184"
	originOtherQueuer   = "70b3d59cd00b0184"
	originStationEUI    = "70b3d59cd000184a"
	originOpID          = int64(18400)
	originSessionUUID   = byte(0x84)
	originSessionOffset = byte(0x40)
)

// validateDownlinkOrigin: migration 000184 gives downlink_queue a nullable
// ac_eui of exactly eight bytes, set only beside an Application Center queue id.
func validateDownlinkOrigin(t *testing.T, db *sql.DB) {
	var dataType, nullable string
	require.NoError(t, db.QueryRow(`SELECT data_type, is_nullable FROM information_schema.columns
		WHERE table_name = 'downlink_queue' AND column_name = 'ac_eui'`).Scan(&dataType, &nullable))
	assert.Equal(t, "bytea", dataType)
	assert.Equal(t, "YES", nullable)
	for name, definition := range map[string]string{
		"downlink_queue_ac_eui_length":   "CHECK (((ac_eui IS NULL) OR (octet_length(ac_eui) = 8)))",
		"downlink_queue_ac_eui_queue_id": "CHECK (((ac_eui IS NULL) OR (ac_que_id IS NOT NULL)))",
	} {
		var got string
		require.NoError(t, db.QueryRow(`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = $1`, name).Scan(&got))
		assert.Equal(t, definition, got, name)
	}
}

// seedApplicationCenterSession stores a session of the Application Center
// acEUI in the organization with the status, and returns its id.
func (h *migrationHarness) seedApplicationCenterSession(tenantID int64, orgID uuid.UUID, acEUI, status string, seed byte) int64 {
	h.t.Helper()
	uuidOf := func(offset byte) []byte {
		id := make([]byte, 16)
		for i := range id {
			id[i] = seed + offset + byte(i)
		}
		return id
	}
	var id int64
	require.NoError(h.t, h.db.QueryRow(`INSERT INTO scaci_sessions (tenant_id, organization_id, ac_eui, sn_ac_uuid, sn_sc_uuid, status)
		VALUES ($1, $2, decode($3, 'hex'), $4, $5, $6) RETURNING id`,
		tenantID, orgID, acEUI, uuidOf(0), uuidOf(originSessionOffset), status).Scan(&id))
	return id
}

// recordQueueOperation stores the inbound dlDataQue record of the session
// with the request data.
func (h *migrationHarness) recordQueueOperation(tenantID, sessionID int64, requestData map[string]interface{}) {
	h.t.Helper()
	encoded, err := json.Marshal(requestData)
	require.NoError(h.t, err)
	h.exec(`INSERT INTO scaci_operation_log (session_id, tenant_id, op_id, command, direction, request_data)
		VALUES ($1, $2, $3, 'dlDataQue', 'inbound', $4)`, sessionID, tenantID, originOpID, string(encoded))
}

// seedQueuedRow stores a pending downlink of the endpoint under the service
// center queue id, as the schemas before migration 000160 held it.
func (h *migrationHarness) seedQueuedRow(tenantID int64, orgID uuid.UUID, queID int64) {
	h.exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at)
		VALUES (decode($1, 'hex'), $2, $3, '\x01', 'pending', $4, NULL)`, originEndpoint, tenantID, orgID, queID)
}

// downlinkOrigin is the acEui migration 184 stored for the row, "" for none.
func (h *migrationHarness) downlinkOrigin(queID int64) string {
	h.t.Helper()
	var acEUI []byte
	require.NoError(h.t, h.db.QueryRow(`SELECT ac_eui FROM downlink_queue WHERE que_id = $1`, queID).Scan(&acEUI))
	return hex.EncodeToString(acEUI)
}

// TestMigration184RestoresTheOriginOfPendingDownlinks: an upgrade from
// schema 000142, the one Kilo Cloud runs, gives a pending downlink an Application
// Center queued the acEui its dlDataQue record names, from the old queId
// record of a session its Application Center has since resumed; the downlink
// keeps the queue id that Application Center knows it by. A gRPC or MQTT
// downlink, a row whose records name two Application Centers and a row whose
// record belongs to another organization get no origin. On the current
// schema a scQueId record outweighs an older queId record naming another
// Application Center.
func TestMigration184RestoresTheOriginOfPendingDownlinks(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(142)
	tenantID, _ := h.seedTenantAndBaseStation(originStationEUI)
	owner, other := h.seedOrganization(tenantID), h.seedOrganization(tenantID)
	queuedBefore := h.seedApplicationCenterSession(tenantID, owner, originQueuer, "disconnected", originSessionUUID)
	resumed := h.seedApplicationCenterSession(tenantID, owner, originQueuer, "active", originSessionUUID+1)
	otherQueuer := h.seedApplicationCenterSession(tenantID, owner, originOtherQueuer, "disconnected", originSessionUUID+2)

	const scaciRow, grpcRow, ambiguousRow, foreignRow, currentRow = 500, 501, 502, 503, 18401
	for _, queID := range []int64{scaciRow, grpcRow, ambiguousRow} {
		h.seedQueuedRow(tenantID, owner, queID)
	}
	h.seedQueuedRow(tenantID, other, foreignRow)
	legacyRecord := func(queID int64) map[string]interface{} {
		return map[string]interface{}{"epEui": originEndpoint, "queId": queID}
	}
	h.recordQueueOperation(tenantID, queuedBefore, legacyRecord(scaciRow))
	h.recordQueueOperation(tenantID, queuedBefore, legacyRecord(ambiguousRow))
	h.recordQueueOperation(tenantID, otherQueuer, legacyRecord(ambiguousRow))
	h.recordQueueOperation(tenantID, queuedBefore, legacyRecord(foreignRow))

	h.migrateTo(183)
	h.exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, ac_que_id, earliest_at)
		VALUES (decode($1, 'hex'), $2, $3, '\x01', 'pending', $4, 5, NULL)`, originEndpoint, tenantID, owner, currentRow)
	h.recordQueueOperation(tenantID, resumed, map[string]interface{}{
		"epEui": originEndpoint, "queId": "5", "scQueId": strconv.FormatInt(currentRow, 10),
	})
	h.recordQueueOperation(tenantID, otherQueuer, legacyRecord(5))

	h.migrateTo(184)
	validateDownlinkOrigin(t, h.db)
	assertMigratedOrigins := func() {
		assert.Equal(t, originQueuer, h.downlinkOrigin(scaciRow), "the resumed Application Center that queued it")
		assert.Equal(t, originQueuer, h.downlinkOrigin(currentRow), "the scQueId record wins over an older queId record")
		for name, queID := range map[string]int64{"gRPC or MQTT": grpcRow, "ambiguous": ambiguousRow, "another organization": foreignRow} {
			assert.Empty(t, h.downlinkOrigin(queID), "%s: no Application Center is guessed", name)
		}
	}
	assertMigratedOrigins()

	logger.Initialize("error", "json")
	downlinks := NewRepositories(&DB{clock: clock.SystemClock{}, conn: h.db, sqlxDB: sqlx.NewDb(h.db, "postgres"), log: logger.Get()}).Downlinks
	pending, err := downlinks.GetDownlinkByQueueID(t.Context(), scaciRow, strconv.FormatInt(tenantID, 10))
	require.NoError(t, err)
	require.NotNil(t, pending.ACEUI)
	require.NotNil(t, pending.ACQueID)
	assert.Equal(t, originQueuer, mioty.FormatEUI64Lower(*pending.ACEUI), "the result reaches the queuing Application Center")
	assert.Equal(t, uint64(scaciRow), *pending.ACQueID, "under the queue id it assigned")
	internal, err := downlinks.GetDownlinkByQueueID(t.Context(), grpcRow, strconv.FormatInt(tenantID, 10))
	require.NoError(t, err)
	assert.Nil(t, internal.ACEUI, "a gRPC or MQTT downlink reaches no Application Center")

	h.migrateTo(183)
	assert.False(t, h.columnExists("downlink_queue", "ac_eui"))
	h.migrateTo(184)
	assertMigratedOrigins()
}

// TestMigration184RefusesAnOriginWithoutApplicationQueueID: a row carries the
// Application Center that queued it only beside the queue id it assigned,
// and only as an eight-byte EUI.
func TestMigration184RefusesAnOriginWithoutApplicationQueueID(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(184)
	tenantID, _ := h.seedTenantAndBaseStation(originStationEUI)
	owner := h.seedOrganization(tenantID)
	insert := func(queID int64, acQueID interface{}, acEUI string) error {
		_, err := h.db.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, ac_que_id, ac_eui, earliest_at)
			VALUES (decode($1, 'hex'), $2, $3, '\x01', 'pending', $4, $5, decode($6, 'hex'), NULL)`,
			originEndpoint, tenantID, owner, queID, acQueID, acEUI)
		return err
	}

	assert.ErrorContains(t, insert(18410, nil, originQueuer), "downlink_queue_ac_eui_queue_id")
	assert.ErrorContains(t, insert(18411, 7, "70b3d59c"), "downlink_queue_ac_eui_length")
	assert.NoError(t, insert(18412, 7, originQueuer))
}
