package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Integration represents an event sink configuration for forwarding events
// to external systems (HTTP webhooks, MQTT brokers, databases)
// CRUD-only implementation for API parity
type Integration struct {
	ID             int64           `db:"id" json:"id"`
	OrgID          uuid.UUID       `db:"org_id" json:"orgId"`
	TenantID       int64           `db:"tenant_id" json:"tenantId"`
	Name           string          `db:"name" json:"name"`
	Description    *string         `db:"description" json:"description,omitempty"`
	Type           string          `db:"type" json:"type"`                          // http, mqtt, database
	Config         json.RawMessage `db:"config" json:"config"`                      // Type-specific configuration
	EventFilter    json.RawMessage `db:"event_filter" json:"eventFilter,omitempty"` // Optional event filtering rules
	DeliveryFormat string          `db:"delivery_format" json:"deliveryFormat"`     // json (default)
	Status         string          `db:"status" json:"status"`                      // active, paused, disabled
	CreatedAt      time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updatedAt"`
	CreatedBy      *string         `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy      *string         `db:"updated_by" json:"updatedBy,omitempty"`
}

// Integration type constants
const (
	IntegrationTypeHTTP     = "http"
	IntegrationTypeMQTT     = "mqtt"
	IntegrationTypeDatabase = "database"
)

// Integration status constants
const (
	IntegrationStatusActive   = "active"
	IntegrationStatusPaused   = "paused"
	IntegrationStatusDisabled = "disabled"
)

// Integration delivery format constants
const (
	IntegrationDeliveryFormatJSON = "json"
)
