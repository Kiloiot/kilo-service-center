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

// validEventCategories is the set of categories the system_events CHECK accepts.
var validEventCategories = map[string]bool{
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

// IsValidEventCategory reports whether category is one the system_events CHECK accepts.
func IsValidEventCategory(category string) bool {
	return validEventCategories[category]
}

// ============================================================================
// Event Types - Centralized Definitions (shared by KC-Core services)
// ============================================================================

// Service lifecycle event types (category: system)
const (
	EventTypeServiceStarted = "service.started"
	EventTypeServiceStopped = "service.stopped"
)

// Session recovery event types (category: bssci)
const (
	// EventTypeSessionPendingOpDropped records a persisted pending operation
	// that could not be rebuilt for session resume and was skipped so the
	// remaining operations could still be reissued.
	EventTypeSessionPendingOpDropped = "session.pending_operation_dropped"
	// EventTypeSessionResumeRefused records a base station refusing the
	// session resume its service center offered; the session was retired so
	// the station's next connect starts a new one.
	EventTypeSessionResumeRefused = "session.resume_refused"
)

// Endpoint CRUD event types (category: endpoint)
const (
	EventTypeEndpointCreated = "endpoint.created"
	EventTypeEndpointUpdated = "endpoint.updated"
	EventTypeEndpointDeleted = "endpoint.deleted"
)

// EventTypeEndpointKeysRevealed records an operator reading an endpoint's
// keys in clear (category: audit); the keys themselves never enter the event.
const EventTypeEndpointKeysRevealed = "endpoint.keys_revealed"

// EventTypeEndpointKeysRemoved records an operator clearing a stored endpoint
// key (category: audit); the key itself never enters the event.
const EventTypeEndpointKeysRemoved = "endpoint.keys_removed"

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
	// EventTypeUserPasswordChanged records a user's password set by the user or reset by an admin.
	EventTypeUserPasswordChanged = "user.password_changed"
	// EventTypeUserRegistered records a user creating their own account.
	EventTypeUserRegistered = "user.registered"
	EventTypeOrgCreated     = "organization.created"
	EventTypeOrgUpdated     = "organization.updated"
	EventTypeOrgDeleted     = "organization.deleted"
	// EventTypeOrgMemberAdded records a user joining an organization.
	EventTypeOrgMemberAdded = "organization.member_added"
	// EventTypeOrgMemberUpdated records a membership role or permission change.
	EventTypeOrgMemberUpdated = "organization.member_updated"
	// EventTypeOrgMemberRemoved records a user leaving an organization.
	EventTypeOrgMemberRemoved = "organization.member_removed"
	// EventTypeDownlinkQueued records an operator queuing a downlink through the API.
	EventTypeDownlinkQueued = "downlink.queued"
	// EventTypeDownlinkRevokeRequested records an operator asking a base station to revoke a downlink.
	EventTypeDownlinkRevokeRequested = "downlink.revoke_requested"
	// EventTypeDownlinkUpdated records an operator editing a pending downlink.
	EventTypeDownlinkUpdated = "downlink.updated"
)

// BSSCI downlink lifecycle event types (category: message), written by the
// BSSCI audit logger and the revoke handler and projected by the events service.
const (
	EventTypeDLDataQueueAcknowledged = "dl_data_queue_acknowledged"
	EventTypeDLDataSent              = "dl_data_sent"
	// EventTypeDLDataSentAfterExpiry records the station holding a downlink reporting it sent after it was reported expired.
	EventTypeDLDataSentAfterExpiry = "dl_data_sent_after_expiry"
	EventTypeDLDataExpired         = "dl_data_expired"
	EventTypeDLDataInvalid         = "dl_data_invalid"
	EventTypeDLDataUnknownResult   = "dl_data_unknown_result"
	EventTypeDLDataRevokeInitiated = "dl_data_revoke_initiated"
	EventTypeDLDataRevoked         = "dl_data_revoked"
	// EventTypeDLDataAcknowledged records an uplink's dlAck for a transmitted downlink (BSSCI §3.10.1).
	EventTypeDLDataAcknowledged = "dl_data_acknowledged"
	// EventTypeDLDataEnqueued records a downlink entering the service center queue (SCACI §3.10).
	EventTypeDLDataEnqueued = "dl_data_enqueued"
	// EventTypeDLDataUpdated records a pending downlink rewritten before any base station holds it.
	EventTypeDLDataUpdated = "dl_data_updated"
	// EventTypeDLDataRequeued records a downlink a base station let go returning to the queue.
	EventTypeDLDataRequeued = "dl_data_requeued"
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

// Certificate event types: a station's certificate is filed under the
// basestation category, the server's under audit.
const (
	EventTypeCertificateGenerated       = "certificate.generated"
	EventTypeCertificateServerGenerated = "certificate.server_generated"
	EventTypeCertificateServerRenewed   = "certificate.server_renewed"
)

// EventTypeCertificatePrivateKeyDownloaded records an operator taking a
// station's private key (category: audit); the key itself never enters the event.
const EventTypeCertificatePrivateKeyDownloaded = "certificate.private_key_downloaded"

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

// Protocol error event types (category: scaci)
const (
	// EventTypeSCACIError marks a recorded SCACI protocol error.
	EventTypeSCACIError = "scaciErr"
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

// Base station ping and status event types (category: basestation, BSSCI §5.4, §5.5)
const (
	// EventTypeBaseStationPingSent records the service center writing a ping to a base station.
	EventTypeBaseStationPingSent = "basestation_ping_sent"
	// EventTypeBaseStationPingAnswered records a base station answering the service center's ping with pingRsp.
	EventTypeBaseStationPingAnswered = "basestation_ping_answered"
	// EventTypeBaseStationStatusAnswered records a base station answering an operator's status request with statusRsp.
	EventTypeBaseStationStatusAnswered = "basestation_status_answered"
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
	// EventTitleEndpointCreated is the title for endpoint creation events.
	EventTitleEndpointCreated = "Endpoint created"
	// EventTitleEndpointUpdated is the title for endpoint update events.
	EventTitleEndpointUpdated = "Endpoint updated"
	// EventTitleEndpointDeleted is the title for endpoint deletion events.
	EventTitleEndpointDeleted = "Endpoint deleted"
	// EventTitleEndpointKeysRevealed is the title for endpoint key reveal events.
	EventTitleEndpointKeysRevealed = "Endpoint keys revealed"
	// EventTitleEndpointKeysRemoved is the title for endpoint key removal events.
	EventTitleEndpointKeysRemoved = "Endpoint keys removed"
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
	// EventTitleUserPasswordChanged is the title for password change events.
	EventTitleUserPasswordChanged = "Password changed"
	// EventTitleUserRegistered is the title for self-registration events.
	EventTitleUserRegistered = "User registered"
	// EventTitleOrgCreated is the title for organization creation events.
	EventTitleOrgCreated = "Organization created"
	// EventTitleOrgUpdated is the title for organization update events.
	EventTitleOrgUpdated = "Organization updated"
	// EventTitleOrgDeleted is the title for organization deletion events.
	EventTitleOrgDeleted = "Organization deleted"
	// EventTitleOrgMemberAdded is the title for membership creation events.
	EventTitleOrgMemberAdded = "Organization member added"
	// EventTitleOrgMemberUpdated is the title for membership update events.
	EventTitleOrgMemberUpdated = "Organization member updated"
	// EventTitleOrgMemberRemoved is the title for membership removal events.
	EventTitleOrgMemberRemoved = "Organization member removed"
	// EventTitleDownlinkQueued is the title for operator-queued downlink events.
	EventTitleDownlinkQueued = "Downlink queued"
	// EventTitleDownlinkRevokeRequested is the title for operator downlink revoke requests.
	EventTitleDownlinkRevokeRequested = "Downlink revoke requested"
	// EventTitleDownlinkUpdated is the title for pending downlink edits.
	EventTitleDownlinkUpdated = "Downlink updated"
	// EventTitleSCACIError is the title format for SCACI protocol error events.
	EventTitleSCACIError = "SCACI Error: %s (opId=%d)"
	// EventDescriptionSCACIError is the description format for SCACI protocol error events.
	EventDescriptionSCACIError = "SCACI operation '%s' failed with error %d: %s"
	// EventTitleEndpointAttachedViaBS is the title format for endpoint attach events with BS context.
	EventTitleEndpointAttachedViaBS = "Endpoint %s attached via BS %s"
	// EventTitleEndpointDetachedViaBS is the title format for endpoint detach events with BS context.
	EventTitleEndpointDetachedViaBS = "Endpoint %s detached via BS %s"
	// EventTitlePendingOperationDropped is the title format for a pending
	// operation skipped during session resume (operation type, operation ID).
	EventTitlePendingOperationDropped = "Pending operation %s %d dropped during session resume"
	// EventTitleSessionResumeRefused is the title format for a base station
	// refusing an offered session resume (base station EUI).
	EventTitleSessionResumeRefused = "Base station %s refused resuming its session"
	// EventTitleEndpointDetachedFromBS is the title format for detach propagate events.
	EventTitleEndpointDetachedFromBS = "Endpoint %s detached from BS %s"
	// EventTitleDLDataQueued is the title of a downlink a base station accepted into its queue.
	EventTitleDLDataQueued = "Downlink queued at base station"
	// EventTitleDLDataSent is the title of a downlink a base station transmitted.
	EventTitleDLDataSent = "Downlink sent"
	// EventTitleDLDataSentAfterExpiry is the title of a "sent" a base station reported for a downlink already reported expired.
	EventTitleDLDataSentAfterExpiry = "Downlink reported sent after it expired"
	// EventTitleDLDataExpired is the title of a downlink that expired untransmitted.
	EventTitleDLDataExpired = "Downlink expired"
	// EventTitleDLDataInvalid is the title of a downlink a base station refused as invalid.
	EventTitleDLDataInvalid = "Downlink refused as invalid"
	// EventTitleDLDataUnknownResult is the title of a downlink result outside the specification.
	EventTitleDLDataUnknownResult = "Downlink result unknown"
	// EventTitleDLDataRevoke is the title of a revoke sent to the base station holding a downlink.
	EventTitleDLDataRevoke = "Downlink revoke sent to base station"
	// EventTitleDLDataRevoked is the title of a downlink a base station revoked.
	EventTitleDLDataRevoked = "Downlink revoked"
	// EventTitleDLDataAcknowledged is the title of a downlink the endpoint acknowledged (BSSCI §3.10.1 dlAck).
	EventTitleDLDataAcknowledged = "Downlink acknowledged by endpoint"
	// EventTitleDLDataEnqueued is the title of a downlink the service center queued.
	EventTitleDLDataEnqueued = "Downlink queued at service center"
	// EventTitleDLDataUpdated is the title of a pending downlink rewritten in the queue.
	EventTitleDLDataUpdated = "Pending downlink updated"
	// EventTitleDLDataRequeued is the title of a downlink returned to the queue by the base station holding it.
	EventTitleDLDataRequeued = "Downlink returned to the queue"
	// Event acknowledgement statuses persisted in system_events.status.
	// EventStatusNew marks an unhandled event.
	// EventStatusAcknowledged marks an event an operator has seen.
	// EventStatusResolved marks a closed event.
	EventStatusNew          = "new"
	EventStatusAcknowledged = "acknowledged"
	EventStatusResolved     = "resolved"

	// EventTitleDLRxStatus is the title of the reception status an endpoint reported for a downlink.
	EventTitleDLRxStatus = "Downlink reception reported"

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
	// EventTitleCertificatePrivateKeyDownloaded is the title for station private key downloads.
	EventTitleCertificatePrivateKeyDownloaded = "Base station private key downloaded"

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
	// EventDescriptionEndpointKeysRevealed is the description format for endpoint key reveal events (EUI, key names).
	EventDescriptionEndpointKeysRevealed = "Keys of endpoint %s revealed: %s"
	// EventDescriptionEndpointKeysRemoved is the description format for endpoint key removal events (EUI, key names).
	EventDescriptionEndpointKeysRemoved = "Keys of endpoint %s removed: %s"
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
	// EventDescriptionUserPasswordChanged is the description format for password change events.
	EventDescriptionUserPasswordChanged = "Password of user %s changed"
	// EventDescriptionUserRegistered is the description format for self-registration events.
	EventDescriptionUserRegistered = "User %s registered"
	// EventDescriptionOrgCreated is the description format for organization creation events.
	EventDescriptionOrgCreated = "Organization %s created"
	// EventDescriptionOrgUpdated is the description format for organization update events.
	EventDescriptionOrgUpdated = "Organization %s updated"
	// EventDescriptionOrgDeleted is the description format for organization deletion events.
	EventDescriptionOrgDeleted = "Organization %s deleted"
	// EventDescriptionOrgMemberAdded is the description format for membership creation events.
	EventDescriptionOrgMemberAdded = "User %s added to organization %s as %s"
	// EventDescriptionOrgMemberUpdated is the description format for membership update events.
	EventDescriptionOrgMemberUpdated = "Membership of user %s in organization %s updated"
	// EventDescriptionOrgMemberRemoved is the description format for membership removal events.
	EventDescriptionOrgMemberRemoved = "User %s removed from organization %s"
	// EventDescriptionDownlinkQueued is the description format for operator-queued downlink events.
	EventDescriptionDownlinkQueued = "Downlink %d queued for endpoint %s"
	// EventDescriptionDownlinkRevokeRequested is the description format for operator downlink revoke requests.
	EventDescriptionDownlinkRevokeRequested = "Revoke of downlink %d requested for endpoint %s"
	// EventDescriptionDownlinkUpdated is the description format for pending downlink edits.
	EventDescriptionDownlinkUpdated = "Downlink %d updated for endpoint %s"

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
	// EventDescriptionCertificatePrivateKeyDownloadedFmt names the station whose private key was downloaded.
	EventDescriptionCertificatePrivateKeyDownloadedFmt = "Private key of base station %s downloaded"

	// Migration event description formats.
	EventDescriptionMigrationAppliedFmt = "Migration applied to version %d on %s"
	EventDescriptionMigrationFailedFmt  = "Migration failed on %s: %s"
)

// ============================================================================
// Event Detail Keys
// ============================================================================

// JSON keys of the details/data document persisted with a system event and
// of the pending-operation metadata the BSSCI handlers resume from. Event
// details reach the browser as written, so every key is camelCase.
const (
	EventDetailKeyOpID            = "opId"
	EventDetailKeyQueID           = "queId"
	EventDetailKeyEpEui           = "epEui"
	EventDetailKeyBsEui           = "bsEui"
	EventDetailKeyBaseStationEui  = "baseStationEui"
	EventDetailKeyBaseStationName = "baseStationName"
	EventDetailKeyOperation       = "operation"
	EventDetailKeyOperationID     = "operationId"
	EventDetailKeyOperationType   = "operationType"
	EventDetailKeyEndpointID      = "endpointId"
	EventDetailKeyRevealedKeys    = "revealedKeys"
	EventDetailKeyRemovedKeys     = "removedKeys"
	EventDetailKeyServiceAccount  = "serviceAccountId"
	EventDetailKeyBaseStationID   = "baseStationId"
	EventDetailKeyStatus          = "status"
	EventDetailKeyTimestamp       = "timestamp"
	EventDetailKeyTime            = "time"
	EventDetailKeyRxTime          = "rxTime"
	EventDetailKeyTxTime          = "txTime"
	EventDetailKeyUserData        = "userData"
	EventDetailKeyResult          = "result"
	EventDetailKeyPacketCnt       = "packetCnt"
	EventDetailKeyDlRxSnr         = "dlRxSnr"
	EventDetailKeyDlRxRssi        = "dlRxRssi"
	EventDetailKeyRssi            = "rssi"
	EventDetailKeySnr             = "snr"
	EventDetailKeyAttachCnt       = "attachCnt"
	EventDetailKeyShAddr          = "shAddr"
	EventDetailKeyShortAddr       = "shortAddr"
	EventDetailKeyMacType         = "macType"
	EventDetailKeyActiveMacTypes  = "activeMacTypes"
	EventDetailKeyMessage         = "message"
	EventDetailKeyError           = "error"
	EventDetailKeyReason          = "reason"
	EventDetailKeySuccess         = "success"
	EventDetailKeyFailureCode     = "failureCode"
	EventDetailKeyTenantID        = "tenantID"
	EventDetailKeyTargetBS        = "targetBs"
	EventDetailKeyTargetBSList    = "targetBsList"
	EventDetailKeyTargetBSCount   = "targetBsCount"
	EventDetailKeyIsOnline        = "isOnline"
	EventDetailKeyConnectionType  = "connectionType"
)

// ============================================================================
// Event Severities
// ============================================================================

// Event severity constants for log level classification.
const (
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
	// SourceTypeSystem identifies events from system sources.
	SourceTypeSystem = "system"
	// SourceTypeAPI identifies events from API sources.
	SourceTypeAPI = "api"
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
	// UserEmail is read, never written: the acting user's email, resolved only
	// when that user belongs or belonged to an organization of the event's tenant.
	UserEmail string `json:"user_email" db:"-"`

	// Event content
	Title       string          `json:"title" db:"title"`
	Description string          `json:"description" db:"description"`
	Details     json.RawMessage `json:"details" db:"data"` // Maps to 'data' JSONB column

	// Processing state
	Status string `json:"status" db:"status"` // EventStatus*; stored as EventStatusNew when empty

	// Tags and timestamps
	Tags      []string  `json:"tags" db:"tags"`              // Maps to 'tags' HSTORE column
	CreatedAt time.Time `json:"created_at" db:"occurred_at"` // Maps to 'occurred_at' column
	UpdatedAt time.Time `json:"updated_at" db:"recorded_at"` // Maps to 'recorded_at' column
	// StoredAt is when the database stored the event, or last moved it forward
	// in time, by the database clock; the live event stream reads in this order.
	StoredAt time.Time `json:"-" db:"stored_at"`
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

// ErrorGroupFilter selects the failures ListErrorGroups groups: events of
// the categories at or above the listed severities inside [From, To], plus,
// when IncludeSCACIFailures is set, the failed SCACI operations.
type ErrorGroupFilter struct {
	TenantID             int64
	Categories           []string
	Severities           []string
	EventTypePrefixes    []string
	IncludeSCACIFailures bool
	From                 time.Time
	To                   time.Time
	Limit                int
	Offset               int
}

// EventErrorGroup is one grouped failure of a bucket.
type EventErrorGroup struct {
	EventType  string
	Code       string
	Message    string
	SourceName string
	FirstSeen  time.Time
	LastSeen   time.Time
	Count      int64
	LastOpID   string
}

// EventOrderByStored sorts a listing by when the database stored each event.
const EventOrderByStored = "stored_at"

// SystemEventFilter defines filters for querying events
type SystemEventFilter struct {
	TenantID       string
	EventTypes     []string
	Categories     []string
	Severities     []string
	SourceTypes    []string
	SourceID       *uuid.UUID
	BaseStationID  *int64
	EndpointID     *int64
	BaseStationEUI string // Fallback: filter by source_name ILIKE for BS events
	EndpointEUI    string // Fallback: filter by source_name ILIKE for EP events
	Status         []string
	Since          *time.Time
	Until          *time.Time
	SearchText     string
	OpID           *int64     // Matches the opId carried in the event data
	StoredSince    *time.Time // Keeps the events the database stored at or after it
	Limit          int
	Offset         int
	OrderBy        string // "occurred_at", "severity", etc.
	OrderDirection string // "asc" or "desc"
}

// AlertFilter defines filters specifically for alerts. Without Statuses the
// alerts are the unresolved ones.
type AlertFilter struct {
	TenantID   string
	Severities []string // only warning, error, critical
	Categories []string
	Statuses   []string // EventStatus* values
	Since      *time.Time
	Limit      int
	Offset     int
}
