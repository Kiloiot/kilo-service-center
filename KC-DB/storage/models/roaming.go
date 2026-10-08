// Package models - Roaming support structures
package models

import (
	"encoding/json"
	"time"
)

// RoamingEvent represents a roaming activity in the audit trail
type RoamingEvent struct {
	ID              int64           `db:"id" json:"id"`
	EventType       string          `db:"event_type" json:"eventType"`              // 'attach', 'detach', 'handover'
	EpEUI           []byte          `db:"ep_eui" json:"epEui"`                      // Endpoint EUI
	OwnerTenantID   int64           `db:"owner_tenant_id" json:"ownerTenantId"`     // Original owner
	ServingTenantID int64           `db:"serving_tenant_id" json:"servingTenantId"` // Current serving tenant
	FromBsEUI       []byte          `db:"from_bs_eui" json:"fromBsEui,omitempty"`   // Source base station (for handover)
	ToBsEUI         []byte          `db:"to_bs_eui" json:"toBsEui,omitempty"`       // Destination base station
	Reason          string          `db:"reason" json:"reason,omitempty"`           // Reason for roaming event
	Metadata        json.RawMessage `db:"metadata" json:"metadata,omitempty"`       // Additional event data
	CreatedAt       time.Time       `db:"created_at" json:"createdAt"`
}

// RoamingEventType constants for event classification
const (
	RoamingEventAttach   = "attach"
	RoamingEventDetach   = "detach"
	RoamingEventHandover = "handover"
)
