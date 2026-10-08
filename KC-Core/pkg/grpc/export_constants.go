// Package grpc provides gRPC service constants and utilities for the KiloCenter API.
package grpc

import (
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
)

// Export format constants - re-exported from domain layer (internal/services/messages).
// Single source of truth in domain layer. These MUST NOT be defined inline in adapter files.
const (
	// ExportFormatJSON indicates JSON export format
	ExportFormatJSON = messages.ExportFormatJSON
	// ExportFormatCSV indicates CSV export format
	ExportFormatCSV = messages.ExportFormatCSV
	// NOTE: xlsx is NOT supported - do not add without implementation
)

// Content type constants for export responses - transport-specific (not in domain).
const (
	ContentTypeJSON        = "application/json"
	ContentTypeCSV         = "text/csv"
	ContentTypeOctetStream = "application/octet-stream"
)

// IsValidExportFormat checks if the given format is supported.
// Re-exports from domain layer.
func IsValidExportFormat(format string) bool {
	return messages.IsValidExportFormat(format)
}

// Response status constants for gRPC response status fields and operation tracking.
const (
	// StatusQueued indicates an operation has been queued for processing
	StatusQueued = "queued"
)

// Base station status constants.
const (
	// StatusOffline indicates a base station is offline
	StatusOffline = "offline"
	// StatusOnline indicates a base station is online
	StatusOnline = "online"
)

// Response message constants.
const (
	MsgRevokeInitiated            = "Revoke initiated for downlink message %s"
	MsgRevoked                    = "Downlink message %s revoked"
	MsgEndpointNotAttached        = "Endpoint not currently attached to any base station"
	MsgFmtBSNotConnected          = "Base station %s is not currently connected"
	MsgFmtBSHandshakeIncomplete   = "Base station %s handshake not complete"
	MsgFmtBSNotBidirectional      = "Base station %s does not support bidirectional operations"
	MsgNoSuitableBSSession        = "No suitable base station session available"
	MsgFmtDLRXQuerySendFailed     = "Failed to send query: %v"
	MsgDLRXQuerySent              = "DL RX status query sent to base station"
	MsgULTransmitQueued           = "UL data transmit operation queued for transmission"
	MsgStatusRequestFailed        = "Failed to send status request: %v"
	MsgStatusRequestSent          = "Status request sent successfully"
	MsgPingRequestSent            = "Ping request sent successfully"
	MsgCertsGenerated             = "Server certificates generated successfully"
	MsgCertsRenewed               = "Server certificates renewed successfully"
	MsgServerCertRequiredBeforeBS = "Server certificates must be generated before creating base station certificates. Generate server certificates first via the Certificates settings."
)

// Log message constants for gRPC operations.
const (
	LogBaseStationNotConnected   = "Base station not connected"
	LogStatusRequestFailed       = "Failed to send status request"
	LogPingInitiateFailed        = "Failed to initiate ping"
	LogGenerateServerCertsFailed = "generate server certificates failed"
	LogRenewServerCertsFailed    = "renew server certificates failed"
	LogReleaseManifestLoadFailed = "Failed to load release manifest"
	// LogStreamDeadlineNotCleared warns that a streaming RPC keeps the HTTP
	// server's deadlines and will be cut when they pass.
	LogStreamDeadlineNotCleared = "Streaming RPC keeps the HTTP server deadlines"
)

// fullMethodSeparator joins a service name and a method name into the
// "/package.Service/Method" path a gRPC call is served on.
const fullMethodSeparator = "/"

// Log message constants for BaseStation operations.
const (
	LogBaseStationCreating      = "Creating base station"
	LogBaseStationAlreadyExists = "Base station already exists"
)

// Log message constants for Endpoint operations.
const (
	LogEndpointAlreadyExists = "Endpoint already exists"
	LogEndpointCreateFailed  = "Failed to create endpoint"
)

// Log message constants for RBAC authorization interceptor.
const (
	LogRBACResolutionFailed   = "RBAC role resolution failed"
	LogRBACInsufficientRole   = "RBAC denied: insufficient role"
	LogRBACUnknownMethod      = "RBAC denied: method has no role requirement"
	LogRBACStreamRolesChanged = "RBAC stream ended: roles changed since it opened"
)

// Log message constants for Certificate service.
const (
	LogDownloadCertFailed         = "download certificate failed"
	LogCertIssuanceRequiresTenant = "certificate issuance requires tenant context"
	LogGenerateCertificateFailed  = "generate certificate failed"
	LogGetServerCertStatusFailed  = "get server certificate status failed"
)

// Content type constants for certificate files.
const (
	// ContentTypePEM is the MIME type for PEM-encoded certificates
	ContentTypePEM = "application/x-pem-file"
)

// =========================================================================
// gRPC metadata header constants.
// All call sites MUST import from here.
// =========================================================================

const (
	// MetadataKeyAuthorization is the gRPC metadata header for bearer token.
	MetadataKeyAuthorization = "authorization"

	// MetadataKeyTenantID is the gRPC metadata header for tenant ID (dev mode).
	MetadataKeyTenantID = "x-tenant-id"

	// MetadataKeyOrganizationID is the gRPC metadata header for organization UUID.
	// Required by fail-closed interceptor for downlink operations.
	MetadataKeyOrganizationID = "x-organization-id"

	// MetadataKeyUserID is the gRPC metadata header for user UUID.
	// Required by fail-closed interceptor for audit logging.
	MetadataKeyUserID = "x-user-id"

	// BearerPrefix is the standard Bearer token prefix.
	BearerPrefix = "Bearer "

	// Internal trust headers — injected by KC-Gateway, trusted by KC-Core in gateway mode.
	// KC-Gateway strips these from inbound client requests to prevent spoofing.

	// MetadataKeyInternalTenantID carries the gateway-validated tenant ID.
	MetadataKeyInternalTenantID = "x-kc-internal-tenant-id"

	// MetadataKeyInternalOrgID carries the gateway-validated organization UUID.
	MetadataKeyInternalOrgID = "x-kc-internal-org-id"

	// MetadataKeyInternalUserID carries the gateway-validated user UUID.
	MetadataKeyInternalUserID = "x-kc-internal-user-id"

	// MetadataKeyInternalServiceAccountID carries the gateway-validated API key
	// UUID of a caller that authenticated with a service-account key.
	MetadataKeyInternalServiceAccountID = "x-kc-internal-service-account-id"

	// MetadataKeyInternalPeerSecret carries the shared secret for peer-to-peer internal gRPC auth.
	MetadataKeyInternalPeerSecret = "x-kc-internal-peer-secret" //nolint:gosec // metadata key name, not a credential
)

// GRPCWebAllowedHeaders lists the gRPC-web specific headers for CORS allowlist.
var GRPCWebAllowedHeaders = []string{
	MetadataKeyAuthorization,
	MetadataKeyTenantID,
	MetadataKeyOrganizationID,
	MetadataKeyUserID,
	"x-grpc-web",
	"grpc-timeout",
	"grpc-encoding",
	"grpc-accept-encoding",
	"content-type",
}

// GRPCWebExposeHeaders lists the gRPC-web exposed response headers for CORS.
var GRPCWebExposeHeaders = []string{
	"grpc-status",
	"grpc-message",
	"grpc-status-details-bin",
}

// =========================================================================
// gRPC-web content type and header constants.
// =========================================================================

const (

	// ContentTypeGRPC is the native gRPC content type prefix.
	ContentTypeGRPC = "application/grpc"

	// HeaderContentType is the Content-Type header name.
	HeaderContentType = "Content-Type"

	// HeaderOrigin is the CORS origin header name.
	HeaderOrigin = "Origin"

	// HeaderVary is the cache control vary header name.
	HeaderVary = "Vary"

	// HeaderAccessControlAllowOrigin is the CORS allow origin response header.
	HeaderAccessControlAllowOrigin = "Access-Control-Allow-Origin"

	// HeaderAccessControlAllowMethods is the CORS allowed methods response header.
	HeaderAccessControlAllowMethods = "Access-Control-Allow-Methods"

	// HeaderAccessControlAllowHeaders is the CORS allowed request headers response header.
	HeaderAccessControlAllowHeaders = "Access-Control-Allow-Headers"

	// HeaderAccessControlExposeHeaders is the CORS exposed response headers header.
	HeaderAccessControlExposeHeaders = "Access-Control-Expose-Headers"

	// HeaderAccessControlAllowCredentials is the CORS credentials response header.
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"

	// HeaderAccessControlMaxAge is the CORS preflight cache lifetime response header.
	HeaderAccessControlMaxAge = "Access-Control-Max-Age"

	// HeaderValueTrue is the literal value for boolean-valued headers such as
	// Access-Control-Allow-Credentials.
	HeaderValueTrue = "true"
)
