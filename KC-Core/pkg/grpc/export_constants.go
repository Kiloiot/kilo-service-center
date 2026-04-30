// Package grpc provides gRPC service constants and utilities for the KiloCenter API.
package grpc

import (
	"github.com/kilocenter/KC-Core/internal/services/messages"
)

// Export format constants - re-exported from domain layer (internal/services/messages).
// Single source of truth in domain layer. These MUST NOT be defined inline in adapter files.
const (
	// ExportFormatJSON indicates JSON export format
	ExportFormatJSON = messages.ExportFormatJSON
	// ExportFormatCSV indicates CSV export format
	ExportFormatCSV = messages.ExportFormatCSV
	// NOTE: xlsx is NOT supported - do not add without implementation

	// ExportMaxLimit is the maximum number of records for export operations
	ExportMaxLimit = messages.ExportMaxLimit
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
	// StatusRevokeInitiated indicates a revoke operation has started
	StatusRevokeInitiated = "revoke_initiated"

	// StatusQueued indicates an operation has been queued for processing
	StatusQueued = "queued"

	// StatusSuccess indicates an operation completed successfully
	StatusSuccess = "success"
)

// Base station status constants.
const (
	// StatusOffline indicates a base station is offline
	StatusOffline = "offline"
)

// Certificate type constants.
const (
	// CertTypeCA is the CA certificate type
	CertTypeCA = "ca"
	// CertTypeClient is the client certificate type
	CertTypeClient = "client"
	// CertTypeKey is the private key type
	CertTypeKey = "key"
)

// Response message constants.
const (
	MsgRevokeInitiated            = "Revoke initiated for downlink message %s"
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
	LogStatusRequestFailed       = "Failed to send status request"
	LogPingInitiateFailed        = "Failed to initiate ping"
	LogGenerateServerCertsFailed = "generate server certificates failed"
	LogRenewServerCertsFailed    = "renew server certificates failed"
	LogReleaseManifestLoadFailed = "Failed to load release manifest"
)

// Log message constants for SystemStatus service.
const (
	LogSystemStatusBSStatsFailed      = "Failed to fetch base station stats"
	LogSystemStatusEPCountFailed      = "Failed to fetch endpoint count"
	LogSystemStatusMsgStatsFailed     = "Failed to fetch message stats"
	LogSystemStatusMetricsFetchFailed = "Failed to fetch system status metrics"
	LogSystemStatusCalled             = "GetSystemStatus called"
	LogSystemStatusManifestLoadFailed = "Failed to load release manifest for system status"
	LogSystemStatusHealthCheckFailed  = "Failed to fetch service health statuses"
)

// Log message constants for BaseStation operations.
const (
	LogBaseStationCreating = "Creating base station"
)

// Log message constants for Endpoint operations.
const (
	LogEndpointAlreadyExists = "Endpoint already exists"
	LogEndpointCreateFailed  = "Failed to create endpoint"
)

// Log message constants for RBAC authorization interceptor.
const (
	LogRBACMissingContext     = "RBAC denied: missing org or user context"
	LogRBACResolutionFailed   = "RBAC role resolution failed"
	LogRBACInsufficientRole   = "RBAC denied: insufficient role"
	LogRBACInactiveMembership = "RBAC denied: inactive membership"
)

// Log message constants for Certificate service.
const (
	LogDownloadCertFailed      = "download certificate failed"
	LogCertTempDirCreateFailed = "Failed to create temp directory"
	LogCertInvalidEUI          = "Invalid EUI format"
	LogCertInvalidValidityDays = "Invalid validity days"
	LogCertDirectoryInfo       = "Certificate directory"
	LogCertGeneratorNotFound   = "Certificate generator not found"
	LogCertGeneratorPathInfo   = "Certificate generator path"
	LogCertGenerationRequested = "certificate generation requested"
	LogCertGenerationFailed    = "Certificate generation failed"
	LogCertParseFailed         = "Failed to parse certificate"
	LogCertExpiryUpdateFailed  = "Failed to update certificate expiry"
	LogCertCleanupFailed       = "Failed to cleanup temp directory"
	LogCertConfigMissingPath   = "Certificate generator path not configured and default not found"

	// Directory operations
	LogCertDirCreateFailed  = "Failed to create certificate directory"
	LogCertDirRemoveFailed  = "Failed to remove certificate directory"
	LogCertsDirCreateFailed = "Failed to create certificates directory"

	// CA certificate operations
	LogCertCACertReadFailed = "Failed to read existing CA certificate"
	LogCertCACertCopyFailed = "Failed to copy CA certificate"
	LogCertCACertCopied     = "Copied existing CA certificate"
	LogCertCAKeyReadFailed  = "Failed to read existing CA key"
	LogCertCAKeyCopyFailed  = "Failed to copy CA key"

	// Certificate generation execution
	LogCertGenerationExecuting = "Executing certificate generation"
	LogCertGenerationStdout    = "Certificate generation stdout"
	LogCertGenerationStderr    = "Certificate generation stderr"
	LogCertGenerationSuccess   = "Certificate generation successful"
	LogCertGeneratedWithExpiry = "Certificate generated with expiry"
	LogCertInfoWriteFailed     = "Failed to write certificate info"
	LogCertPersistenceSkipped  = "Certificate persistence skipped"

	// Certificate reading/parsing
	LogCertReadFailed          = "failed to read certificate"
	LogCertUnmarshalFailed     = "Failed to unmarshal certificate info"
	LogCertFileReadFailed      = "Failed to read certificate file"
	LogCertPEMBlockParseFailed = "Failed to parse PEM block"
	LogCertCertsPathInfo       = "KC-Core certificates path"

	// Server certificate operations
	LogServerCertGenRequested     = "server certificate generation requested"
	LogServerCertGenExecuting     = "Executing server certificate generation"
	LogServerCertGenFailed        = "Server certificate generation failed"
	LogServerCertGenSuccess       = "Server certificate generation successful"
	LogServerCertRenewalRequested = "server certificate renewal requested"

	// Cleanup operations
	LogCertTempDirRemoveFailed    = "Failed to remove temp certificate directory"
	LogCertExpiredDirRemoveFailed = "Failed to remove expired certificate directory"

	// Stored certificate retrieval
	LogCertBSNotFound = "Base station not found for certificate retrieval"

	// Startup hints
	LogCertConfigMissingPathHint = "ensure service is started from kilocenter-modules/ directory or set certificates.certgen_path in config"
)

// File extension constants.
const (
	// ExtPEM is the standard PEM certificate file extension
	ExtPEM = ".pem"
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
	// ContentTypeGRPCWebPrefix is the gRPC-web Content-Type prefix for routing detection.
	ContentTypeGRPCWebPrefix = "application/grpc-web"

	// ContentTypeGRPCWeb is the gRPC-web binary protobuf content type.
	ContentTypeGRPCWeb = "application/grpc-web+proto"

	// ContentTypeGRPCWebText is the gRPC-web text (base64) content type.
	ContentTypeGRPCWebText = "application/grpc-web-text+proto"

	// ContentTypeGRPC is the native gRPC content type prefix.
	ContentTypeGRPC = "application/grpc"

	// ContentTypePlainText is the plain text content type (used by http.NotFound).
	ContentTypePlainText = "text/plain; charset=utf-8"

	// HeaderContentType is the Content-Type header name.
	HeaderContentType = "Content-Type"

	// HeaderGRPCWeb is the gRPC-web marker header name.
	HeaderGRPCWeb = "x-grpc-web"

	// HeaderGRPCStatus is the gRPC status response header.
	HeaderGRPCStatus = "grpc-status"

	// HeaderOrigin is the CORS origin header name.
	HeaderOrigin = "Origin"

	// HeaderVary is the cache control vary header name.
	HeaderVary = "Vary"

	// HeaderAccessControlAllowOrigin is the CORS allow origin response header.
	HeaderAccessControlAllowOrigin = "Access-Control-Allow-Origin"

	// HeaderAccessControlRequestMethod is the CORS preflight request method header.
	HeaderAccessControlRequestMethod = "Access-Control-Request-Method"
)
