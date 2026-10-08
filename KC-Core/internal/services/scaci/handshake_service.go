// Package scaciservices implements the SCACI (Service Center Application Center Interface) protocol server.
//
// handshake_service.go implements the HandshakeService interface: it returns
// error tokens from the SCACI error catalog, never Go errors, and the
// transport layer resolves the tokens to POSIX codes.
package scaciservices

import (
	"context"
	"crypto/x509"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	pkgversion "github.com/Kiloiot/kilo-service-center/pkg/version"
)

// SessionFactory creates a new SCACI session; it is injected so the
// composition root supplies the crypto-backed implementation and tests can
// force entropy failure without package globals.
type SessionFactory func(tenantID int64, acEui uint64, snAcUUID scaci.UUID16) (*scaci.Session, error)

// handshakeService establishes SCACI connections, negotiates the version and
// resumes sessions (SCACI §3.3). The client certificate resolves to an
// organization and tenant; strict resolution fails closed, community mode
// falls back to the default tenant's organization. The handler persists the
// session this service validates.
type handshakeService struct {
	sessionRepo         SessionResumeReader
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
	newSession          SessionFactory            // Session creation with injected entropy
}

// NewHandshakeService creates a new handshake service
//
// Parameters:
//   - sessionRepo: the stored sessions a resume looks up
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
	sessionRepo SessionResumeReader,
	log logger.Logger,
	orgResolver org.Resolver,
	defaultTenantID int64,
	strictOrgResolution bool,
	certVerifier scaci.CertificateVerifier,
	scEui uint64,
	scVendor, scModel, scName, scSwVersion string,
	newSession SessionFactory,
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
		newSession:          newSession,
	}
}

// ValidateConnect implements HandshakeService.ValidateConnect (SCACI §3.3):
// it resolves the certificate's organization and tenant, negotiates the
// version (§§2.1-2.3), resumes the stored session the application center
// presents when it is that application center's, or starts a new one, and
// builds the connect response. The handler persists the session afterwards.
func (hs *handshakeService) ValidateConnect(
	ctx context.Context,
	req *scaci.Connect,
	cert *x509.Certificate,
) (*scaci.Session, *scaci.ConnectResponse, string) {
	orgID, tenantID, errToken := hs.resolveOrganization(ctx, cert)
	if errToken != "" {
		return nil, nil, errToken
	}
	hs.logger.DebugContext(ctx, scaci.LogSCACIProcessingConnect,
		logger.FieldTenantID, tenantID,
		logger.FieldOrgID, orgID.String(),
		logger.FieldCertCN, cert.Subject.CommonName)

	negotiatedVersion, errToken := hs.NegotiateVersion(ctx, req.Version)
	if errToken != "" {
		return nil, nil, errToken
	}

	session, errToken := hs.resumedSession(ctx, req, tenantID, orgID)
	if errToken != "" {
		return nil, nil, errToken
	}
	canResume := session != nil
	if !canResume {
		if session, errToken = hs.startSession(ctx, req, tenantID, orgID); errToken != "" {
			return nil, nil, errToken
		}
	}

	recordConnectMetadata(session, req)
	hs.warnSoftwareVersion(ctx, tenantID, req.AcEui)

	return session, &scaci.ConnectResponse{
		Version:   &negotiatedVersion,
		ScEui:     hs.scEui,
		SnScUUID:  session.SnScUUID,
		SnResume:  canResume,
		Vendor:    &hs.scVendor,
		Model:     &hs.scModel,
		Name:      &hs.scName,
		SwVersion: &hs.scSwVersion,
	}, ""
}

// resolveOrganization verifies the client certificate and resolves the
// organization and tenant it belongs to.
func (hs *handshakeService) resolveOrganization(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, string) {
	if cert == nil {
		hs.logger.ErrorContext(ctx, scaci.LogSCACINoClientCertificate)
		return uuid.Nil, 0, scaci.ErrNilCertificate
	}
	if hs.orgResolver == nil {
		hs.logger.ErrorContext(ctx, scaci.LogSCACIOrgResolverNotInjected)
		return uuid.Nil, 0, scaci.ErrNilCertificate
	}
	// Certificate expiry, key usage and subject are verified before the tenant is resolved.
	if errToken := hs.certVerifier.VerifyCertificate(ctx, cert); errToken != "" {
		return uuid.Nil, 0, errToken
	}
	orgID, tenantID, err := hs.orgResolver.ResolveCert(ctx, cert)
	if err != nil {
		return hs.defaultOrganization(ctx, cert, err)
	}
	return orgID, tenantID, ""
}

// CertificateTenant implements HandshakeService.CertificateTenant. The
// community fallback to the default tenant does not apply: a certificate that
// maps to no organization resolves to no tenant.
func (hs *handshakeService) CertificateTenant(ctx context.Context, cert *x509.Certificate) (int64, bool) {
	if cert == nil || hs.orgResolver == nil {
		return 0, false
	}
	_, tenantID, err := hs.orgResolver.ResolveCert(ctx, cert)
	if err != nil || tenantID <= 0 {
		return 0, false
	}
	return tenantID, true
}

// defaultOrganization is the default tenant's organization a certificate
// that resolves to none falls back to in community mode; strict resolution
// fails closed.
func (hs *handshakeService) defaultOrganization(ctx context.Context, cert *x509.Certificate, resolveErr error) (uuid.UUID, int64, string) {
	if hs.strictOrgResolution {
		hs.logger.ErrorContext(ctx, scaci.LogSCACICertificateMappingFailed,
			logger.FieldError, resolveErr,
			logger.FieldCertCN, cert.Subject.CommonName,
			logger.FieldStrictMode, true)
		return uuid.Nil, 0, scaci.ErrCertificateTenantResolutionFailed
	}

	hs.logger.WarnContext(ctx, scaci.LogSCACICertOrgResolutionFallback,
		logger.FieldError, resolveErr,
		logger.FieldCertCN, cert.Subject.CommonName,
		logger.FieldDefaultTenantID, hs.defaultTenantID)

	orgID, err := hs.orgResolver.GetDefaultOrgForTenant(ctx, hs.defaultTenantID)
	if err != nil {
		hs.logger.ErrorContext(ctx, scaci.LogSCACICertificateMappingFailed,
			logger.FieldError, err,
			logger.FieldTenantID, hs.defaultTenantID)
		return uuid.Nil, 0, scaci.ErrCertificateTenantResolutionFailed
	}
	return orgID, hs.defaultTenantID, ""
}

// startSession creates a new session of the organization's application
// center. No session may exist with a predictable identifier, so an entropy
// failure rejects the connect.
func (hs *handshakeService) startSession(ctx context.Context, req *scaci.Connect, tenantID int64, orgID uuid.UUID) (*scaci.Session, string) {
	session, err := hs.newSession(tenantID, req.AcEui, req.SnAcUUID)
	if err != nil {
		hs.logger.ErrorContext(ctx, scaci.LogSCACISessionCreateFailed, logger.FieldError, err)
		return nil, scaci.ErrSessionEntropyFailure
	}
	session.OrganizationID = orgID

	hs.logger.InfoContext(ctx, scaci.LogSCACINewSessionCreated,
		logger.FieldAcEui, mioty.FormatEUI64(req.AcEui),
		logger.FieldSnScUUID, scaci.FormatUUID(session.SnScUUID),
		logger.FieldOrgID, orgID.String())
	return session, ""
}

// recordConnectMetadata keeps the connect metadata of SCACI §3.3.1 on the
// session; info is an arbitrary object and is kept whole.
func recordConnectMetadata(session *scaci.Session, req *scaci.Connect) {
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
	if req.Info != nil {
		session.Metadata["info"] = req.Info
	}
}

// warnSoftwareVersion warns when the service center's swVersion (SCACI
// §3.3.2, optional) is unset or a development build.
func (hs *handshakeService) warnSoftwareVersion(ctx context.Context, tenantID int64, acEui uint64) {
	switch hs.scSwVersion {
	case "":
		hs.logger.WarnContext(ctx, scaci.LogSCACISoftwareVersionNotConfigured,
			logger.FieldTenantID, tenantID,
			logger.FieldAcEui, mioty.FormatEUI64(acEui))
	case pkgversion.DevVersion, pkgversion.DevLocalVersion:
		hs.logger.WarnContext(ctx, scaci.LogSCACIUsingDevelopmentSoftwareVersion,
			logger.FieldSwVersion, hs.scSwVersion,
			logger.FieldTenantID, tenantID,
			logger.FieldAcEui, mioty.FormatEUI64(acEui))
	}
}

// NegotiateVersion implements HandshakeService.NegotiateVersion
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
func (hs *handshakeService) NegotiateVersion(ctx context.Context, clientVersion string) (string, string) {
	// Parse client version using existing helper
	reqMajor, reqMinor, _, err := scaci.ParseSemanticVersion(clientVersion)
	if err != nil {
		hs.logger.ErrorContext(ctx, scaci.LogSCACIInvalidVersionFormat,
			logger.FieldVersion, clientVersion,
			logger.FieldError, err)
		return "", scaci.ErrInvalidVersionFormat
	}

	// SCACI §2.1: Major version must match
	if reqMajor != scaci.SupportedMajorVersion {
		hs.logger.ErrorContext(ctx, scaci.LogSCACIMajorVersionMismatch,
			logger.FieldRequested, clientVersion,
			logger.FieldSupportedMajor, scaci.SupportedMajorVersion)
		return "", scaci.ErrMajorVersionUnsupported
	}

	// SCACI §2.2: Minor version compatibility
	if reqMinor > scaci.SupportedMinorVersion {
		hs.logger.ErrorContext(ctx, scaci.LogSCACIMinorVersionTooHigh,
			logger.FieldRequested, clientVersion,
			logger.FieldSupportedMinor, scaci.SupportedMinorVersion)
		return "", scaci.ErrMinorVersionUnsupported
	}

	// SCACI §2.3: Ignore patch - use ProtocolVersionString from constants
	negotiatedVersion := scaci.ProtocolVersionString

	hs.logger.InfoContext(ctx, scaci.LogSCACIVersionNegotiationOk,
		logger.FieldRequested, clientVersion,
		logger.FieldNegotiated, negotiatedVersion)

	return negotiatedVersion, ""
}
