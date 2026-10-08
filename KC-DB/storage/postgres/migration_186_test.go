package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	uplinkDeliveryIndex   = "uq_scaci_op_log_uplink_delivery"
	uplinkDeliveryAcEui   = "70b3d59cd000a186"
	uplinkDeliveryMessage = "c3d4e5f6-a7b8-4c9d-8e0f-1a2b3c4d5e6f"
)

// validateSCACIUplinkDeliveryIdentity: migration 000186 keeps one outbound
// ulData record per session and stored uplink.
func validateSCACIUplinkDeliveryIdentity(t *testing.T, db *sql.DB) {
	var unique bool
	require.NoError(t, db.QueryRow(`SELECT i.indisunique FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
		WHERE c.relname = $1`, uplinkDeliveryIndex).Scan(&unique))
	assert.True(t, unique)
}

// seedSCACISession stores a session of the tenant for the operation log rows.
func (h *migrationHarness) seedSCACISession(tenantID int64) int64 {
	return h.queryInt(`INSERT INTO scaci_sessions (tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid, status)
		VALUES ($1, decode($2, 'hex'), uuid_send(gen_random_uuid()), uuid_send(gen_random_uuid()), 'disconnected') RETURNING id`,
		tenantID, uplinkDeliveryAcEui)
}

// logULData stores an outbound ulData record of the session with requestData.
func (h *migrationHarness) logULData(sessionID, tenantID, opID int64, requestData string) error {
	_, err := h.db.Exec(`INSERT INTO scaci_operation_log (session_id, tenant_id, op_id, command, direction, state, request_data)
		VALUES ($1, $2, $3, 'ulData', 'outbound', 'pending', $4::jsonb)`, sessionID, tenantID, opID, requestData)
	return err
}

// TestMigration186KeepsOneULDataPerSessionAndUplink: after 000186 the ulData
// records stored before (without sourceMessageId) stay as they are, a second
// record of one session and uplink is refused while other uplinks, sessions
// and commands are not, and the down migration lifts the rule again.
func TestMigration186KeepsOneULDataPerSessionAndUplink(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(185)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000b186")
	sessionID := h.seedSCACISession(tenantID)
	require.NoError(t, h.logULData(sessionID, tenantID, -1, `{"packetCnt": 1}`))
	require.NoError(t, h.logULData(sessionID, tenantID, -2, `{"packetCnt": 1}`))

	h.migrateTo(186)
	assert.Contains(t, h.indexDefinition(uplinkDeliveryIndex),
		"(session_id, ((request_data ->> 'sourceMessageId'::text))) WHERE (((command)::text = 'ulData'::text) AND ((direction)::text = 'outbound'::text) AND ((request_data ->> 'sourceMessageId'::text) IS NOT NULL))")
	identified := `{"packetCnt": 2, "sourceMessageId": "` + uplinkDeliveryMessage + `"}`
	require.NoError(t, h.logULData(sessionID, tenantID, -3, identified))
	assert.Error(t, h.logULData(sessionID, tenantID, -4, identified), "one ulData per session and uplink")
	require.NoError(t, h.logULData(sessionID, tenantID, -5, `{"packetCnt": 3, "sourceMessageId": "d4e5f6a7-b8c9-4d0e-8f1a-2b3c4d5e6f70"}`))
	require.NoError(t, h.logULData(h.seedSCACISession(tenantID), tenantID, -1, identified), "another session keeps its own")
	assert.Equal(t, int64(2), h.queryInt(`SELECT count(*) FROM scaci_operation_log WHERE session_id = $1 AND NOT request_data ? 'sourceMessageId'`, sessionID),
		"the records stored before stay")

	h.migrateTo(185)
	assert.Empty(t, h.indexDefinitionIfAny(uplinkDeliveryIndex))
	require.NoError(t, h.logULData(sessionID, tenantID, -6, identified), "the previous release records every attempt")

	h.exec(`DELETE FROM scaci_operation_log WHERE op_id = -6`)
	h.migrateTo(186)
	assert.NotEmpty(t, h.indexDefinitionIfAny(uplinkDeliveryIndex), "a second up restores the rule")
}
