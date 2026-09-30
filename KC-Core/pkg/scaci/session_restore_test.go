package scaci

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// A restored session carries the identity, the operation ID counters and the
// Connect metadata of its row, including the arbitrary info object (SCACI
// §3.3.1), and waits for a connect operation to activate it.
func TestRestoreSession_KeepsIdentityCountersAndNestedMetadata(t *testing.T) {
	orgID := uuid.New()
	heartbeat := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	info := map[string]interface{}{
		"serialNo":     12345,
		"capabilities": []interface{}{"downlink", "multicast"},
		"nested":       map[string]interface{}{"deep": "value"},
	}
	row := &models.SCACISession{
		ID:             123,
		TenantID:       42,
		OrganizationID: &orgID,
		AcEUI:          [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22},
		SnAcUUID:       [16]byte{0x01, 0x02, 0x03},
		SnScUUID:       [16]byte{0x11, 0x12, 0x13},
		ConnectedAt:    heartbeat.Add(-time.Hour),
		LastHeartbeat:  &heartbeat,
		LastOpIDAc:     100,
		LastOpIDSc:     -100,
		Metadata:       map[string]interface{}{"vendor": "TestVendor", "info": info},
	}

	session := RestoreSession(row)

	assert.Equal(t, int64(123), session.ID)
	assert.Equal(t, int64(42), session.TenantID)
	assert.Equal(t, orgID, session.OrganizationID)
	assert.Equal(t, uint64(0xAABBCCDDEEFF1122), session.AcEui)
	assert.Equal(t, UUID16(row.SnAcUUID), session.SnAcUUID)
	assert.Equal(t, UUID16(row.SnScUUID), session.SnScUUID)
	assert.Equal(t, OpIDPair{AC: 100, SC: -100}, session.OpIDs())
	assert.Equal(t, StateConnecting, session.State, "active only once conCmp completes the connect operation (SCACI §3.3)")
	assert.False(t, session.Resumed)
	assert.Equal(t, heartbeat, session.LastSeen)
	assert.Equal(t, "TestVendor", session.Metadata["vendor"])
	assert.Equal(t, info, session.Metadata["info"])
}

func TestRestoreSession_WithoutMetadataOrOrganization(t *testing.T) {
	session := RestoreSession(&models.SCACISession{ID: 456, TenantID: 1, ConnectedAt: time.Now()})

	assert.NotNil(t, session.Metadata)
	assert.Empty(t, session.Metadata)
	assert.Equal(t, uuid.Nil, session.OrganizationID)
	assert.Equal(t, session.Connected, session.LastSeen)
}
