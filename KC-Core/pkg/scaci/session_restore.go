package scaci

import (
	"encoding/binary"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// RestoreSession rebuilds a persisted session (SCACI §3.3): identity, tenant,
// organization, operation ID counters and metadata. It is not connected; the
// connect operation that resumes it completes on a new connection.
func RestoreSession(row *models.SCACISession) *Session {
	lastSeen := row.ConnectedAt
	if row.LastHeartbeat != nil {
		lastSeen = *row.LastHeartbeat
	}
	metadata := make(map[string]interface{}, len(row.Metadata))
	for k, v := range row.Metadata {
		metadata[k] = v
	}
	orgID := uuid.Nil
	if row.OrganizationID != nil {
		orgID = *row.OrganizationID
	}
	session := &Session{
		ID:             row.ID,
		TenantID:       row.TenantID,
		OrganizationID: orgID,
		AcEui:          binary.BigEndian.Uint64(row.AcEUI[:]),
		SnAcUUID:       row.SnAcUUID,
		SnScUUID:       row.SnScUUID,
		State:          StateConnecting,
		Connected:      row.ConnectedAt,
		LastSeen:       lastSeen,
		Metadata:       metadata,
	}
	session.ops.restore(OpIDPair{AC: row.LastOpIDAc, SC: row.LastOpIDSc})
	return session
}
