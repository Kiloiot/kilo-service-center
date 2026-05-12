package models

import (
	"encoding/json"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// ============================================================================
// Event Categories - Single Source of Truth (matches DB constraint)
// ============================================================================

// Event category constants - aliases to mioty.* protocol-layer constants (single source of truth)
// KC-Core uses mioty.* directly; system_event.go aliases for API/app-layer usage.
const (
	// EventCategoryEndpoint is the category for endpoint CRUD and attach/detach events.
	EventCategoryEndpoint = mioty.CategoryEndpoint
	// EventCategoryBaseStation is the category for base station CRUD and connect/disconnect events.
	EventCategoryBaseStation = mioty.CategoryBaseStation
	// EventCategoryMessage is the category for UL/DL message traffic (excluded from dashboard).
	EventCategoryMessage = mioty.CategoryMessage
	// EventCategorySystem is the category for service lifecycle and background job events.
	EventCategorySystem = mioty.CategorySystem
)

// App-level event category constants (not in mioty.*).
const (
	// EventCategorySecurity is the category for auth failures and access violations.
	EventCategorySecurity = "security"
	// EventCategoryRoaming is the category for cross-tenant roaming events.
	EventCategoryRoaming = "roaming"
	// EventCategoryError is the category for system errors.
	EventCategoryError = "error"
	// EventCategoryAudit is the category for admin actions (user/org CRUD).
	EventCategoryAudit = "audit"
	// EventCategoryProtocol is the category for generic BSSCI/SCACI protocol events.
	EventCategoryProtocol = "protocol"
	// EventCategorySCACI is the category for SCACI-specific events.
	EventCategorySCACI = "scaci"
	// EventCategoryBSSCI is the category for BSSCI-specific events.
	EventCategoryBSSCI = "bssci"
	// EventCategorySession is the category for protocol sessions and realtime streaming.
	EventCategorySession = "session"
)

// ValidEventCategories provides the complete set of valid categories for validation
var ValidEventCategories = map[string]bool{
	EventCategorySecurity:    true,
	EventCategoryEndpoint:    true,
	EventCategoryBaseStation: true,
	EventCategoryMessage:     true,
	EventCategorySystem:      true,
	EventCategoryRoaming:     true,
	EventCategoryError:       true,
	EventCategoryAudit:       true,
	EventCategoryProtocol:    true,
	EventCategorySCACI:       true,
	EventCategoryBSSCI:       true,
	EventCategorySession:     true,
}

// ============================================================================
// Event Types - Centralized Definitions (shared by KC-Core services)
// ============================================================================

// Service lifecycle event types (category: system)
const (
	EventTypeServiceStarted = "service.started"
	EventTypeServiceStopped = "service.stopped"
)

// Endpoint CRUD event types (category: endpoint)
const (
	EventTypeEndpointCreated = "endpoint.created"
	EventTypeEndpointUpdated = "endpoint.updated"
	EventTypeEndpointDeleted = "endpoint.deleted"
)

// Base station CRUD event types (category: basestation)
const (
	EventTypeBSRegistered   = "basestation.registered"
	EventTypeBSUpdated      = "basestation.updated"
	EventTypeBSDeregistered = "basestation.deregistered"
)

// User/Org CRUD event types (category: audit)
const (
	EventTypeUserCreated = "user.created"
	EventTypeUserUpdated = "user.updated"
	EventTypeUserDeleted = "user.deleted"
	EventTypeOrgCreated  = "organization.created"
	EventTypeOrgUpdated  = "organization.updated"
	EventTypeOrgDeleted  = "organization.deleted"
)

// API key event types (category: audit)
const (
	EventTypeAPIKeyCreated = "api_key.created"
	EventTypeAPIKeyDeleted = "api_key.deleted"
)

// Integration event types (category: audit)
const (
	EventTypeIntegrationCreated = "integration.created"
	EventTypeIntegrationUpdated = "integration.updated"
	EventTypeIntegrationDeleted = "integration.deleted"
)

// Certificate event types (category: audit)
const (
	EventTypeCertificateGenerated       = "certificate.generated"
	EventTypeCertificateServerGenerated = "certificate.server_generated"
	EventTypeCertificateServerRenewed   = "certificate.server_renewed"
)

// Migration event types (category: system)
const (
	EventTypeMigrationApplied = "migration.applied"
	EventTypeMigrationFailed  = "migration.failed"
)

// Security event types (category: security)
const (
	EventTypeAuthInvalidToken           = "auth.invalid_token"    //nolint:gosec // G101: event type identifier, not a credential
	EventTypeAuthAPIKeyRejected         = "auth.api_key_rejected" //nolint:gosec // G101: event type identifier, not a credential
	EventTypeAuthOrgContextMissing      = "auth.organization_context_missing"
	EventTypeAuthOrgResolutionFailed    = "auth.organization_resolution_failed"
	EventTypeAuthInternalTrustViolation = "auth.internal_trust_violation"
	EventTypeAuthPermissionDenied       = "auth.permission_denied"
	EventTypeAuthLoginFailed            = "auth.login_failed"
	EventTypeAuthOIDCExchangeFailed     = "auth.oidc_exchange_failed"
	EventTypeAuthOAuth2ExchangeFailed   = "auth.oauth2_exchange_failed"
)

// Realtime event types (category: endpoint, basestation, bssci)
// NOTE: Underscore format preserved for backward compatibility with existing DB rows
const (
	EventTypeEndpointAttached         = "endpoint_attached"
	EventTypeEndpointDetached         = "endpoint_detached"
	EventTypeBaseStationOnline        = "basestation_online"
	EventTypeBaseStationOffline       = "basestation_offline"
	EventTypeDetachPropagateCompleted = "detach_propagate_completed"
	EventTypeConnectionError          = "connection_error"
)

// ============================================================================
// Event Titles - Centralized Strings (no literals in code)
// ============================================================================

// Service lifecycle title/description format strings.
const (
	// EventTitleServiceStartedFmt formats a service startup title with service name and hostname.
	EventTitleServiceStartedFmt = "%s started on %s"
	// EventTitleServiceStoppedFmt formats a service shutdown title with service name and hostname.
	EventTitleServiceStoppedFmt = "%s stopped on %s"
	// EventDescriptionServiceStartedFmt formats a startup description with service name, version, and listen info.
	EventDescriptionServiceStartedFmt = "%s v%s started — %s"
	// EventDescriptionServiceStoppedFmt formats a shutdown description with service name and version.
	EventDescriptionServiceStoppedFmt = "%s v%s stopped gracefully"
)

// Event title constants for human-readable event descriptions.
const (
	// EventTitleServiceStarted is the title for service startup events.
	EventTitleServiceStarted = "Service started"
	// EventTitleServiceStopped is the title for service shutdown events.
	EventTitleServiceStopped = "Service stopped"
	// EventTitleEndpointAttached is the title for endpoint attach events.
	EventTitleEndpointAttached = "Endpoint attached"
	// EventTitleEndpointDetached is the title for endpoint detach events.
	EventTitleEndpointDetached = "Endpoint detached"
	// EventTitleBaseStationOnline is the title for base station online events.
	EventTitleBaseStationOnline = "Base station online"
	// EventTitleBaseStationOffline is the title for base station offline events.
	EventTitleBaseStationOffline = "Base station offline"
	// EventTitleEndpointCreated is the title for endpoint creation events.
	EventTitleEndpointCreated = "Endpoint created"
	// EventTitleEndpointUpdated is the title for endpoint update events.
	EventTitleEndpointUpdated = "Endpoint updated"
	// EventTitleEndpointDeleted is the title for endpoint deletion events.
	EventTitleEndpointDeleted = "Endpoint deleted"
	// EventTitleBSRegistered is the title for base station registration events.
	EventTitleBSRegistered = "Base station registered"
	// EventTitleBSUpdated is the title for base station update events.
	EventTitleBSUpdated = "Base station updated"
	// EventTitleBSDeregistered is the title for base station deregistration events.
	EventTitleBSDeregistered = "Base station deregistered"
	// EventTitleUserCreated is the title for user creation events.
	EventTitleUserCreated = "User created"
	// EventTitleUserUpdated is the title for user update events.
	EventTitleUserUpdated = "User updated"
	// EventTitleUserDeleted is the title for user deletion events.
	EventTitleUserDeleted = "User deleted"
	// EventTitleOrgCreated is the title for organization creation events.
	EventTitleOrgCreated = "Organization created"
	// EventTitleOrgUpdated is the title for organization update events.
	EventTitleOrgUpdated = "Organization updated"
	// EventTitleOrgDeleted is the title for organization deletion events.
	EventTitleOrgDeleted = "Organization deleted"
	// EventTitleSCACIError is the title format for SCACI protocol error events.
	EventTitleSCACIError = "SCACI Error: %s (opId=%d)"
	// EventDescriptionSCACIError is the description format for SCACI protocol error events.
	EventDescriptionSCACIError = "SCACI operation '%s' failed with error %d: %s"
	// EventTitleEndpointAttachedViaBS is the title format for endpoint attach events with BS context.
	EventTitleEndpointAttachedViaBS = "Endpoint %s attached via BS %s"
	// EventTitleEndpointDetachedViaBS is the title format for endpoint detach events with BS context.
	EventTitleEndpointDetachedViaBS = "Endpoint %s detached via BS %s"
	// EventTitleEndpointDetachedFromBS is the title format for detach propagate events.
	EventTitleEndpointDetachedFromBS = "Endpoint %s detached from BS %s"
	// EventTitleDLDataQueued is the title format for downlink data queue events.
	EventTitleDLDataQueued = "DL Data Queued - %s"
	// EventTitleDLDataSent is the title format for downlink data sent events.
	EventTitleDLDataSent = "DL Data Sent - %s"
	// EventTitleDLDataExpired is the title format for downlink data expired events.
	EventTitleDLDataExpired = "DL Data Expired - %s"
	// EventTitleDLDataInvalid is the title format for downlink data invalid events.
	EventTitleDLDataInvalid = "DL Data Invalid - %s"
	// EventTitleDLDataUnknownResult is the title format for downlink data unknown result events.
	EventTitleDLDataUnknownResult = "DL Data Unknown Result - %s"
	// EventTitleDLDataRevoke is the title format for downlink data revoke events.
	EventTitleDLDataRevoke = "DL Data Revoke - %s"
	// EventTitleDLDataRevoked is the title format for downlink data revoked events.
	EventTitleDLDataRevoked = "DL Data Revoked - %s"
	// EventTitleDLRxStatus is the title format for downlink RX status events.
	EventTitleDLRxStatus = "DL RX Status - %s"

	// API key event titles.
	EventTitleAPIKeyCreated = "API key created" //nolint:gosec // G101: UI title string, not a credential
	EventTitleAPIKeyDeleted = "API key deleted" //nolint:gosec // G101: UI title string, not a credential

	// Integration event titles.
	EventTitleIntegrationCreated = "Integration created"
	EventTitleIntegrationUpdated = "Integration updated"
	EventTitleIntegrationDeleted = "Integration deleted"

	// Certificate event titles.
	EventTitleCertificateGenerated       = "Certificate generated"
	EventTitleCertificateServerGenerated = "Server certificates generated"
	EventTitleCertificateServerRenewed   = "Server certificates renewed"

	// Migration event titles.
	EventTitleMigrationApplied = "Database migration applied"
	EventTitleMigrationFailed  = "Database migration failed"

	// Security event titles.
	EventTitleAuthInvalidToken           = "Invalid authentication token"
	EventTitleAuthAPIKeyRejected         = "API key rejected"
	EventTitleAuthOrgContextMissing      = "Organization context missing"
	EventTitleAuthOrgResolutionFailed    = "Organization resolution failed"
	EventTitleAuthInternalTrustViolation = "Internal trust violation"
	EventTitleAuthPermissionDenied       = "Permission denied"
	EventTitleAuthLoginFailed            = "Login failed"
	EventTitleAuthOIDCExchangeFailed     = "OIDC exchange failed"
	EventTitleAuthOAuth2ExchangeFailed   = "OAuth2 exchange failed"
)

// ============================================================================
// Event Description Format Strings (CRUD events only)
// ============================================================================

// Event description format strings for CRUD events.
const (
	// EventDescriptionEndpointCreated is the description format for endpoint creation events.
	EventDescriptionEndpointCreated = "Endpoint %s created"
	// EventDescriptionEndpointUpdated is the description format for endpoint update events.
	EventDescriptionEndpointUpdated = "Endpoint %s updated"
	// EventDescriptionEndpointDeleted is the description format for endpoint deletion events.
	EventDescriptionEndpointDeleted = "Endpoint %s deleted"
	// EventDescriptionBSRegistered is the description format for base station registration events.
	EventDescriptionBSRegistered = "Base station %s registered"
	// EventDescriptionBSUpdated is the description format for base station update events.
	EventDescriptionBSUpdated = "Base station %s updated"
	// EventDescriptionBSDeleted is the description format for base station deletion events.
	EventDescriptionBSDeleted = "Base station %s deleted"
	// EventDescriptionUserCreated is the description format for user creation events.
	EventDescriptionUserCreated = "User %s created"
	// EventDescriptionUserUpdated is the description format for user update events.
	EventDescriptionUserUpdated = "User %s updated"
	// EventDescriptionUserDeleted is the description format for user deletion events.
	EventDescriptionUserDeleted = "User %s deleted"
	// EventDescriptionOrgCreated is the description format for organization creation events.
	EventDescriptionOrgCreated = "Organization %s created"
	// EventDescriptionOrgUpdated is the description format for organization update events.
	EventDescriptionOrgUpdated = "Organization %s updated"
	// EventDescriptionOrgDeleted is the description format for organization deletion events.
	EventDescriptionOrgDeleted = "Organization %s deleted"

	// API key event description formats.
	EventDescriptionAPIKeyCreatedFmt = "API key '%s' created for organization %s" //nolint:gosec // G101: UI description template, not a credential
	EventDescriptionAPIKeyDeletedFmt = "API key '%s' deleted"                     //nolint:gosec // G101: UI description template, not a credential

	// Integration event description formats.
	EventDescriptionIntegrationCreatedFmt = "Integration '%s' created"
	EventDescriptionIntegrationUpdatedFmt = "Integration '%s' updated"
	EventDescriptionIntegrationDeletedFmt = "Integration '%s' deleted"

	// Certificate event description formats.
	EventDescriptionCertificateGeneratedFmt       = "Certificate generated for base station %s"
	EventDescriptionCertificateServerGeneratedFmt = "Server TLS certificates generated for %s"
	EventDescriptionCertificateServerRenewedFmt   = "Server TLS certificates renewed for %s"

	// Migration event description formats.
	EventDescriptionMigrationAppliedFmt = "Migration applied to version %d on %s"
	EventDescriptionMigrationFailedFmt  = "Migration failed on %s: %s"
)

// ============================================================================
// Event Severities
// ============================================================================

// Event severity constants for log level classification.
const (
	// EventSeverityDebug is the severity for debug-level events.
	EventSeverityDebug = "debug"
	// EventSeverityInfo is the severity for informational events.
	EventSeverityInfo = "info"
	// EventSeverityWarning is the severity for warning events.
	EventSeverityWarning = "warning"
	// EventSeverityError is the severity for error events.
	EventSeverityError = "error"
	// EventSeverityCritical is the severity for critical events.
	EventSeverityCritical = "critical"
)

// ============================================================================
// Source Types
// ============================================================================

// Source type constants - aliases to mioty.* protocol-layer constants (single source of truth).
const (
	// SourceTypeBaseStation identifies events from base station sources.
	SourceTypeBaseStation = mioty.SourceTypeBaseStation
	// SourceTypeEndpoint identifies events from endpoint sources.
	SourceTypeEndpoint = mioty.SourceTypeEndpoint
	// SourceTypeServiceCenter identifies events from service center sources.
	SourceTypeServiceCenter = mioty.SourceTypeServiceCenter
)

// App-level source type constants (not in mioty.*).
const (
	// SourceTypeApplicationCenter identifies events from application center sources.
	SourceTypeApplicationCenter = "application_center"
	// SourceTypeSystem identifies events from system sources.
	SourceTypeSystem = "system"
	// SourceTypeAPI identifies events from API sources.
	SourceTypeAPI = "api"
	// SourceTypeClient identifies events from client sources.
	SourceTypeClient = "client"
)

// SystemEvent represents a system-level event.
// Fields match the schema in migrations/008_system_events.up.sql and 021_add_status_to_system_events.up.sql.
type SystemEvent struct {
	ID       string `json:"id" db:"id"`
	TenantID string `json:"tenant_id" db:"tenant_id"`

	// Event classification
	EventType string `json:"event_type" db:"event_type"`   // e.g., endpoint.created, basestation.registered (see EventType* constants)
	Category  string `json:"category" db:"event_category"` // security, endpoint, basestation, message, system, etc. (see EventCategory* constants)
	Severity  string `json:"severity" db:"severity"`       // debug, info, warning, error, critical (see EventSeverity* constants)
	EventCode string `json:"event_code" db:"event_code"`   // Machine-readable event code

	// Source information
	SourceType string     `json:"source_type" db:"source_type"` // endpoint, basestation, service_center, api, client (see SourceType* constants)
	SourceID   *uuid.UUID `json:"source_id" db:"source_id"`     // UUID of the source entity (org UUIDs only — use EndpointID/BasestationID for asset scoping)
	SourceName string     `json:"source_name" db:"source_name"` // Human-readable identifier (EUI, username) for display and fallback filtering

	// Entity correlation (for scoped activity feeds)
	EndpointID    *int64 `json:"endpoint_id" db:"endpoint_id"`       // FK to endpoints.id for endpoint-scoped queries
	BasestationID *int64 `json:"basestation_id" db:"basestation_id"` // FK to basestations.id for BS-scoped queries
	MessageID     *int64 `json:"message_id" db:"message_id"`         // FK to mioty_messages.id
	UserID        string `json:"user_id" db:"user_id"`               // User ID for audit trail

	// Event content
	Title       string          `json:"title" db:"title"`
	Description string          `json:"description" db:"description"`
	Details     json.RawMessage `json:"details" db:"data"` // Maps to 'data' JSONB column

	// Processing state
	Status string `json:"status" db:"status"` // new, active, acknowledged, resolved, ignored

	// Tags and timestamps
	Tags      []string  `json:"tags" db:"tags"`              // Maps to 'tags' HSTORE column
	CreatedAt time.Time `json:"created_at" db:"occurred_at"` // Maps to 'occurred_at' column
	UpdatedAt time.Time `json:"updated_at" db:"recorded_at"` // Maps to 'recorded_at' column
}

// SystemEventStats represents system event statistics
type SystemEventStats struct {
	TenantID           string           `json:"tenant_id"`
	TotalEvents        int64            `json:"total_events"`
	EventsByType       map[string]int64 `json:"events_by_type"`
	EventsBySeverity   map[string]int64 `json:"events_by_severity"`
	EventsByStatus     map[string]int64 `json:"events_by_status"`
	NewEvents          int64            `json:"new_events"`
	AcknowledgedEvents int64            `json:"acknowledged_events"`
	ResolvedEvents     int64            `json:"resolved_events"`
}
