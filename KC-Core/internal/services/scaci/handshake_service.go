// Package scaciservices implements the SCACI (Service Center Application Center Interface) protocol server.
//
// handshake_service.go implements HandshakeService interface.
//
// Extracted Logic:
//   - Version negotiation (from handler_connect.go:94-128)
//   - Session resumption validation (from handler_connect.go:130-164)
//   - Connect response building (from handler_connect.go:165-205)
//
// Dependencies (injected):
//   - interfaces.SCACISessionRepository: Session persistence (REUSED, not new interface)
//   - logger.Logger: Structured logging
//
// Error Handling:
//   - Returns error tokens from errors_catalog.go (package-private constants)
//   - NO Go errors returned from public methods
//   - Transport layer resolves tokens → POSIX codes
//
// Infrastructure Reuse:
//   - errors_catalog.go: All error tokens (err*)
//   - log_messages.go: All log constants (Log*)
//   - version.go: parseSemanticVersion() helper
//   - session.go: NewSession(), CanResume(), FormatUUID() helpers
package scaciservices

import (
	"context"
	"crypto/x509"
	"encoding/binary"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // FormatEUI64 helper
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// handshakeService implements HandshakeService interface
//
// This service manages SCACI connection establishment, version negotiation,
// and session resumption per MIOTY §3.3. It delegates session persistence to
// the existing interfaces.SCACISessionRepository (14 scoped methods).
//
// Organization + Tenant Resolution:
//   - Receives *x509.Certificate from transport layer (handler_connect.go)
//   - Resolves org UUID -> tenant ID via injected org.Resolver (DIP)
//   - Strict mode (strictOrgResolution=true): Fail-closed on resolution failure
//   - Community mode (strictOrgResolution=false): Falls back to defaultTenantID
//   - Validates cross-tenant session resume attempts (fail-closed security)
//
// Version Support:
//   - Currently supports major version 1, minor version 0
//   - Future enhancement: make this configurable
//
// Session Persistence:
//   - Service validates and builds session object (with organization_id)
//   - Handler is responsible for persisting to database
//   - This separation keeps transport concerns (conn mapping) in handler layer
type handshakeService struct {
	sessionRepo         interfaces.SCACISessionRepository // REUSE existing interface
	logger              logger.Logger
	orgResolver         org.Resolver              // Org UUID -> tenant ID resolution
	defaultTenantID     int64                     // Community fallback tenant ID
	strictOrgResolution bool                      // Fail-closed on org resolution failure (production mode)
	certVerifier        scaci.CertificateVerifier // Certificate security validation
	scEui               uint64                    // Service Center EUI from config
	scVendor            string                    // SC vendor name
	scModel             string                    // SC model
	scName              string                    // SC instance name
	scSwVersion         string                    // SC software version
}

// NewHandshakeService creates a new handshake service
//
// Parameters:
//   - sessionRepo: Existing SCACISessionRepository interface (14 methods)
//   - logger: Structured logger
//   - orgResolver: Organization UUID -> tenant ID resolver
//   - defaultTenantID: Fallback tenant ID for community mode (when cert parsing fails)
//   - strictOrgResolution: If true, fail-closed on org resolution failure (production mode)
//   - certVerifier: Certificate security validator (injected via DI)
//   - scEui: Service Center EUI64
//   - scVendor: SC vendor name (for ConnectResponse metadata)
//   - scModel: SC model (for ConnectResponse metadata)
//   - scName: SC instance name (for ConnectResponse metadata)
//   - scSwVersion: SC software version (for ConnectResponse metadata)
//
// Returns:
//   - HandshakeService: Service instance implementing interface
func NewHandshakeService(
	sessionRepo interfaces.SCACISessionRepository,
	log logger.Logger,
	orgResolver org.Resolver,
	defaultTenantID int64,
	strictOrgResolution bool,
	certVerifier scaci.CertificateVerifier,
	scEui uint64,
	scVendor, scModel, scName, scSwVersion string,
) scaci.HandshakeService {
	return &handshakeService{
		sessionRepo:         sessionRepo,
		logger:              log,
		orgResolver:         orgResolver,
		defaultTenantID:     defaultTenantID,
		strictOrgResolution: strictOrgResolution,
		certVerifier:        certVerifier,
		scEui:               scEui,
		scVendor:            scVendor,
		scModel:             scModel,
		scName:              scName,
		scSwVersion:         scSwVersion,
	}
}

// ValidateConnect implements HandshakeService.ValidateConnect
//
// Extracted from handler_connect.go:33-205.
//
// Flow:
//  1. Guard against nil certificate (prevents panic on cert.Subject dereference)
//  2. Resolve tenant from certificate via injected org.Resolver
//  3. Negotiate version (§2.1-2.3): Compare major/minor, reject incompatible
//  4. Check resume attempt: If SnAcOpId/SnScOpId present, validate session
//  5. Resume validation: Reject if certificate tenant doesn't match stored session tenant
//  6. Resume or create: Either resume existing session or create new
//  7. Build response: Include negotiated version, snScUuid, metadata
//  8. Handler persists session after receiving response from this method
//
// Parameters:
//   - ctx: Request context with timeout
//   - req: Decoded Connect message from wire
//   - cert: Client certificate from TLS handshake (service resolves tenant)
//
// Returns:
//   - *Session: Session object for handler to map to connection (includes resolved tenantID)
//   - *ConnectResponse: Response to send on wire
//   - string: Error token or "" on success
func (hs *handshakeService) ValidateConnect(
	ctx context.Context,
	req *scaci.Connect,
	cert *x509.Certificate,
) (*scaci.Session, *scaci.ConnectResponse, string) {
	// Guard 1: Nil certificate check (prevents panic on cert.Subject)
	if cert == nil {
		hs.logger.ErrorContext(ctx, scaci.LogSCACINoClientCertificate)
		return nil, nil, scaci.ErrNilCertificate
	}

	// Guard 2: Nil resolver check (defensive - shouldn't happen if wired correctly)
	if hs.orgResolver == nil {
		hs.logger.ErrorContext(ctx, scaci.LogSCACIOrgResolverNotInjected)
		return nil, nil, scaci.ErrNilCertificate
	}

	// Guard 3: Certificate security validation (defense-in-depth)
	// Verify certificate expiry, key usage, and subject BEFORE tenant resolution
	if errToken := hs.certVerifier.VerifyCertificate(cert); errToken != "" {
		// Certificate validation failed - return error token
		return nil, nil, errToken
	}

	// Step 1: Resolve organization UUID + tenant ID from certificate.
	// Try certificate-based org resolution first (Kilo Cloud mode).
	orgID, tenantID, err := hs.orgResolver.ResolveCert(ctx, cert)
	if err != nil {
		// Strict mode: fail-closed on org resolution failure (production/managed cloud)
		if hs.strictOrgResolution {
			hs.logger.ErrorContext(ctx, scaci.LogSCACICertificateMappingFailed,
				"error", err,
				"certCN", cert.Subject.CommonName,
				"strictMode", true)
			return nil, nil, scaci.ErrCertificateTenantResolutionFailed
		}

		// Community mode: fall back to default tenant ID
		hs.logger.WarnContext(ctx, "Certificate org resolution failed, using community fallback",
			"error", err,
			"certCN", cert.Subject.CommonName,
			"defaultTenantID", hs.defaultTenantID)

		tenantID = hs.defaultTenantID
		orgID, err = hs.orgResolver.GetDefaultOrgForTenant(ctx, tenantID)
		if err != nil {
			hs.logger.ErrorContext(ctx, scaci.LogSCACICertificateMappingFailed,
				"error", err,
				"tenantID", tenantID)
			return nil, nil, scaci.ErrCertificateTenantResolutionFailed
		}
	}

	hs.logger.DebugContext(ctx, scaci.LogSCACIProcessingConnect,
		"tenantID", tenantID,
		"orgID", orgID.String(),
		"certCN", cert.Subject.CommonName)

	// Step 2: Version negotiation (§2.1-2.3)
	negotiatedVersion, errToken := hs.NegotiateVersion(req.Version)
	if errToken != "" {
		return nil, nil, errToken
	}

	// Step 2: Check for session resumption attempt
	// Resume requires BOTH SnAcOpId and SnScOpId (validation already done in handler)
	isResumeAttempt := req.SnAcOpId != nil && req.SnScOpId != nil

	var session *scaci.Session
	canResume := false

	if isResumeAttempt {
		// Attempt to resolve resumption
		resumeOk, errToken := hs.ResolveResume(
			ctx,
			tenantID,
			req.SnAcUUID[:],
			nil, // scUUID not needed for lookup, we validate via CheckSessionResumable
			*req.SnAcOpId,
			*req.SnScOpId,
			req.Version, // SCACI §§2.1-2.3: validate version matches stored negotiated_version
		)
		if errToken != "" {
			hs.logger.WarnContext(ctx, scaci.LogSCACIResumeFailed,
				"acEui", pkgmioty.FormatEUI64(req.AcEui),
				"reason", errToken)
			// Resume failure is not fatal - create new session instead
		} else if resumeOk {
			// Load the existing session from database via GetSessionByAcUUID
			var acUUIDBytes [16]byte
			copy(acUUIDBytes[:], req.SnAcUUID[:])

			dbSession, err := hs.sessionRepo.GetSessionByAcUUID(ctx, tenantID, acUUIDBytes)
			if err == nil && dbSession != nil {
				// Validate certificate tenant matches stored session tenant
				if dbSession.TenantID != tenantID {
					hs.logger.ErrorContext(ctx, scaci.LogSCACICrossTenantResumeRejected,
						"certTenantID", tenantID,
						"sessionTenantID", dbSession.TenantID)
					// Fail-closed: reject cross-tenant resume attempts
					return nil, nil, scaci.ErrNoActiveSession
				}

				// Reconstruct Session from database model
				session = hs.sessionFromModel(dbSession)
				session.Resumed = true
				canResume = true

				hs.logger.InfoContext(ctx, scaci.LogSCACISessionResumed,
					"acEui", pkgmioty.FormatEUI64(req.AcEui),
					"snAcUuid", scaci.FormatUUID(req.SnAcUUID))
			}
		}
	}

	// Step 3: Create new session if resume failed or not attempted
	if session == nil {
		session = scaci.NewSession(tenantID, req.AcEui, req.SnAcUUID)
		session.OrganizationID = orgID
		canResume = false

		hs.logger.InfoContext(ctx, scaci.LogSCACINewSessionCreated,
			"acEui", pkgmioty.FormatEUI64(req.AcEui),
			"snScUuid", scaci.FormatUUID(session.SnScUUID),
			"orgID", orgID.String())
	}

	// Step 4: Update session metadata from Connect message
	session.EnsureMetadata()
	if req.Vendor != nil {
		session.Metadata["vendor"] = *req.Vendor
	}
	if req.Model != nil {
		session.Metadata["model"] = *req.Model
	}
	if req.Name != nil {
		session.Metadata["name"] = *req.Name
	}
	if req.SwVersion != nil {
		session.Metadata["swVersion"] = *req.SwVersion
	}
	// SCACI §3.3.1: info is arbitrary key-value object (optional)
	// Preserve entire nested structure for KC-Web/audit consumption
	if req.Info != nil {
		session.Metadata["info"] = req.Info
	}

	// TODO: Validate software version (non-fatal per SCACI §3.3.2 - swVersion is optional)
	switch hs.scSwVersion {
	case "":
		hs.logger.WarnContext(ctx, "Software version not configured - ConnectResponse will omit swVersion field",
			"tenantID", tenantID,
			"acEui", pkgmioty.FormatEUI64(req.AcEui))
	case "dev", "dev-local":
		hs.logger.WarnContext(ctx, "Using development software version in ConnectResponse",
			"swVersion", hs.scSwVersion,
			"tenantID", tenantID,
			"acEui", pkgmioty.FormatEUI64(req.AcEui))
	}

	// Step 5: Build ConnectResponse
	// Note: Session persistence is handled by the handler layer after this returns
	// This keeps transport concerns (conn mapping, database transaction timing) separate
	resp := &scaci.ConnectResponse{
		Version:   &negotiatedVersion,
		ScEui:     hs.scEui,
		SnScUUID:  session.SnScUUID,
		SnResume:  canResume,
		Vendor:    &hs.scVendor,
		Model:     &hs.scModel,
		Name:      &hs.scName,
		SwVersion: &hs.scSwVersion,
	}

	return session, resp, ""
}

// NegotiateVersion implements HandshakeService.NegotiateVersion
//
// Extracted from handler_connect.go:94-128
//
// SCACI Version Negotiation Rules (§2.1-2.3):
//   - §2.1: Major version MUST match (reject on mismatch)
//   - §2.2: Minor version compatibility - downgrade to highest common
//   - §2.3: Patch version ignored for protocol compatibility
//
// Current Support:
//   - Major: 1 (only)
//   - Minor: 0 (highest supported)
//   - Patch: 0 (informational)
//
// Parameters:
//   - clientVersion: Version string from Connect message (e.g., "1.0.0")
//
// Returns:
//   - string: Negotiated version (e.g., "1.0.0")
//   - string: Error token (errInvalidVersionFormat, errMajorVersionUnsupported) or ""
func (hs *handshakeService) NegotiateVersion(clientVersion string) (string, string) {
	// Parse client version using existing helper
	reqMajor, reqMinor, _, err := scaci.ParseSemanticVersion(clientVersion)
	if err != nil {
		hs.logger.Error(scaci.LogSCACIInvalidVersionFormat,
			"version", clientVersion,
			"error", err)
		return "", scaci.ErrInvalidVersionFormat
	}

	// SCACI §2.1: Major version must match
	if reqMajor != scaci.SupportedMajorVersion {
		hs.logger.Error(scaci.LogSCACIMajorVersionMismatch,
			"requested", clientVersion,
			"supported_major", scaci.SupportedMajorVersion)
		return "", scaci.ErrMajorVersionUnsupported
	}

	// SCACI §2.2: Minor version compatibility
	if reqMinor > scaci.SupportedMinorVersion {
		hs.logger.Error(scaci.LogSCACIMinorVersionTooHigh,
			"requested", clientVersion,
			"supported_minor", scaci.SupportedMinorVersion)
		return "", scaci.ErrMinorVersionUnsupported
	}

	// SCACI §2.3: Ignore patch - use ProtocolVersionString from constants
	negotiatedVersion := scaci.ProtocolVersionString

	hs.logger.Info(scaci.LogSCACIVersionNegotiationOk,
		"requested", clientVersion,
		"negotiated", negotiatedVersion)

	return negotiatedVersion, ""
}

// ResolveResume implements HandshakeService.ResolveResume
//
// Validates session resumption per SCACI §3.3 and §§2.1-2.3 using the existing
// interfaces.SCACISessionRepository.CheckSessionResumable() method.
//
// This method delegates to the repository's CheckSessionResumable which:
//   - Validates UUID consistency
//   - Checks operation ID progression
//   - Ensures session is in resumable state
//   - Returns NegotiatedVersion for version mismatch validation
//
// Parameters:
//   - ctx: Request context
//   - tenantID: Tenant scope for session lookup
//   - acUUID: AC session UUID (snAcUuid from Connect)
//   - scUUID: SC session UUID (optional, not used for lookup)
//   - acOpId: Last AC operation ID
//   - scOpId: Last SC operation ID
//   - requestVersion: Protocol version from Connect message (must match stored negotiated_version)
//
// Returns:
//   - bool: True if session can be resumed
//   - string: Error token if validation fails, "" on success
func (hs *handshakeService) ResolveResume(
	ctx context.Context,
	tenantID int64,
	acUUID []byte,
	_ []byte, // scUUID not used for lookup, validated via CheckSessionResumable
	acOpId, scOpId int64,
	requestVersion string,
) (bool, string) {
	// Convert acUUID to [16]byte for repository call
	var acUUIDBytes [16]byte
	if len(acUUID) != 16 {
		hs.logger.ErrorContext(ctx, scaci.LogSCACIInvalidAcUUIDLength,
			"length", len(acUUID))
		return false, scaci.ErrSnAcUUIDZero
	}
	copy(acUUIDBytes[:], acUUID)

	// Use existing repository method to check resumability
	// Pass both acOpId and scOpId for full opId parity validation per SCACI-S.1-05/3.2-03
	resumptionInfo, err := hs.sessionRepo.CheckSessionResumable(ctx, tenantID, acUUIDBytes, acOpId, scOpId)
	if err != nil {
		hs.logger.DebugContext(ctx, scaci.LogSCACIResumeFailed,
			"error", err,
			"tenantID", tenantID)
		return false, scaci.ErrNoActiveSession
	}

	if resumptionInfo == nil {
		hs.logger.DebugContext(ctx, scaci.LogSCACINoResumableSession,
			"tenantID", tenantID)
		return false, scaci.ErrNoActiveSession
	}

	// Validate operation ID consistency
	// CheckSessionResumable already validated this
	if !resumptionInfo.CanResume {
		hs.logger.WarnContext(ctx, scaci.LogSCACISessionCannotResume,
			"acOpId", acOpId,
			"scOpId", scOpId,
			"reason", resumptionInfo.ReasonIfNotResumable)
		return false, scaci.ErrOpIdOutOfOrder
	}

	// SCACI §§2.1-2.3: Validate version consistency on resume
	// Resume attempts must match major.minor of originally negotiated version (patch ignored per §2.3)
	if resumptionInfo.NegotiatedVersion != "" {
		storedMajor, storedMinor, _, storedErr := scaci.ParseSemanticVersion(resumptionInfo.NegotiatedVersion)
		reqMajor, reqMinor, _, reqErr := scaci.ParseSemanticVersion(requestVersion)

		// Fail-closed on parse errors (should not happen if versions were validated at connect)
		if storedErr != nil || reqErr != nil {
			hs.logger.WarnContext(ctx, scaci.LogSCACIVersionMismatchOnResume,
				"storedVersion", resumptionInfo.NegotiatedVersion,
				"requestVersion", requestVersion,
				"storedParseErr", storedErr,
				"requestParseErr", reqErr,
				"tenantID", tenantID)
			return false, scaci.ErrVersionMismatchOnResume
		}

		// §2.3: Compare major.minor only, ignore patch
		if storedMajor != reqMajor || storedMinor != reqMinor {
			hs.logger.WarnContext(ctx, scaci.LogSCACIVersionMismatchOnResume,
				"storedMajor", storedMajor, "storedMinor", storedMinor,
				"reqMajor", reqMajor, "reqMinor", reqMinor,
				"tenantID", tenantID)
			return false, scaci.ErrVersionMismatchOnResume
		}
	}
	// Empty stored version (pre-migration session) -> allow resume (fail-open for backwards compat)

	return true, ""
}

// sessionFromModel converts models.SCACISession to Session
//
// Helper method to reconstruct in-memory Session from database model.
// This is needed because the repository returns models.SCACISession but
// the service layer works with Session.
//
// Parameters:
//   - dbSession: Database session model
//
// Returns:
//   - *Session: In-memory session object
func (hs *handshakeService) sessionFromModel(dbSession *models.SCACISession) *scaci.Session {
	lastSeen := dbSession.ConnectedAt
	if dbSession.LastHeartbeat != nil {
		lastSeen = *dbSession.LastHeartbeat
	}

	// Preserve metadata types from DB (includes Connect.info per SCACI §3.3.1).
	// Preserves nested objects without type-asserting to string.
	metadata := make(map[string]interface{}, len(dbSession.Metadata))
	for k, v := range dbSession.Metadata {
		metadata[k] = v
	}

	// Extract org UUID from DB model
	orgID := uuid.Nil
	if dbSession.OrganizationID != nil {
		orgID = *dbSession.OrganizationID
	}

	session := &scaci.Session{
		ID:             dbSession.ID,
		TenantID:       dbSession.TenantID,
		OrganizationID: orgID,
		AcEui:          binary.BigEndian.Uint64(dbSession.AcEUI[:]),
		SnAcUUID:       dbSession.SnAcUUID,
		SnScUUID:       dbSession.SnScUUID,
		AcOpIdMin:      0, // Not stored in DB, reset on resume
		ScOpIdMax:      0, // Not stored in DB, reset on resume
		AcOpIdCounter:  dbSession.LastOpIDAc,
		ScOpIdCounter:  dbSession.LastOpIDSc,
		State:          scaci.StateActive, // Resumed sessions are active
		Resumed:        false,             // Caller sets this to true
		Connected:      dbSession.ConnectedAt,
		LastSeen:       lastSeen,
		Metadata:       metadata,
	}

	// Initialize in-memory caches for resumed sessions (not persisted to DB)
	// PendingDeregisterOps tracks epEui for in-flight deregister operations per SCACI §3.7
	session.EnsurePendingDeregisterOps()

	return session
}
