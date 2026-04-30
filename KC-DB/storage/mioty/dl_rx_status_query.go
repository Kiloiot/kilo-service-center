// Package mioty contains shared MIOTY protocol types
package mioty

import (
	"time"

	"github.com/google/uuid"
)

// DLRXStatusQuery represents a DL RX status query tracking record (BSSCI §5.15 correlation)
// This tracks dlRxStatQry requests to correlate with incoming dlRxStat responses
type DLRXStatusQuery struct {
	ID             int64      `db:"id"`
	TenantID       int64      `db:"tenant_id"`
	OrganizationID *uuid.UUID `db:"org_uuid"`     // Organization UUID for multi-tenant isolation (nullable)
	EpEui          []byte     `db:"ep_eui"`       // 8 bytes - endpoint EUI
	BsEui          []byte     `db:"bs_eui"`       // 8 bytes - base station EUI
	OpId           int64      `db:"op_id"`        // SC operation ID (negative)
	Status         string     `db:"status"`       // "pending"|"received"|"timeout"
	RequestedAt    time.Time  `db:"requested_at"` // When dlRxStatQry was sent
	ReceivedAt     *time.Time `db:"received_at"`  // When dlRxStat was received (nil if not received)
}
