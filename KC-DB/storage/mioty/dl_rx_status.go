// Package mioty contains shared MIOTY protocol types
package mioty

import (
	"time"

	"github.com/google/uuid"
)

// DLRXStatus represents a DL reception status report from an endpoint (BSSCI §3.15)
type DLRXStatus struct {
	ID             int64      `db:"id"`
	TenantID       int64      `db:"tenant_id"`
	OrganizationID *uuid.UUID `db:"org_uuid"` // Organization UUID for multi-tenant isolation (nullable)
	EpEui          []byte     `db:"ep_eui"`   // 8 bytes
	BsEui          []byte     `db:"bs_eui"`   // 8 bytes
	RxTime         int64      `db:"rx_time"`  // Unix UTC ns
	PacketCnt      uint32     `db:"packet_cnt"`
	DlRxSnr        float64    `db:"dl_rx_snr"`  // dB
	DlRxRssi       float64    `db:"dl_rx_rssi"` // dBm
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}
