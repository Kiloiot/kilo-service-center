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

// RoamingPolicy represents the policy configuration for a tenant
type RoamingPolicy struct {
	MaxRoamingEndpoints int                    `json:"maxRoamingEndpoints,omitempty"`
	AllowedPartners     []int64                `json:"allowedPartners,omitempty"`
	BlockedEndpoints    []string               `json:"blockedEndpoints,omitempty"`
	RateLimits          map[string]int         `json:"rateLimits,omitempty"`
	BillingMode         string                 `json:"billingMode,omitempty"` // "per-message", "flat-rate", etc.
	CustomRules         map[string]interface{} `json:"customRules,omitempty"`
}

// RoamingEndpointInfo represents roaming endpoint information in base station sessions
type RoamingEndpointInfo struct {
	EUI           string `json:"eui"`
	OwnerTenantID int64  `json:"ownerTenantId"`
	AttachedAt    string `json:"attachedAt"`
}

// RoamingStatistics represents aggregated roaming metrics
type RoamingStatistics struct {
	TenantID          int64  `db:"tenant_id" json:"tenantId"`
	TenantName        string `db:"tenant_name" json:"tenantName"`
	OwnedRoamingOut   int    `db:"owned_roaming_out" json:"ownedRoamingOut"`     // Owned endpoints roaming elsewhere
	VisitingRoamingIn int    `db:"visiting_roaming_in" json:"visitingRoamingIn"` // Visiting endpoints from other tenants
	RoamingEvents24h  int    `db:"roaming_events_24h" json:"roamingEvents24h"`
	RoamingEnabled    bool   `db:"roaming_enabled" json:"roamingEnabled"`
	PartnerCount      int    `db:"partner_count" json:"partnerCount"`
}

// TenantRoamingConfig represents tenant-level roaming configuration
type TenantRoamingConfig struct {
	RoamingEnabled  bool            `db:"roaming_enabled" json:"roamingEnabled"`
	RoamingPartners json.RawMessage `db:"roaming_partners" json:"roamingPartners"` // Array of partner tenant IDs
	RoamingPolicy   json.RawMessage `db:"roaming_policy" json:"roamingPolicy"`     // Policy object
}
