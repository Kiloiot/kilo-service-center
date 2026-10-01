package scaciservices

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// resumedSession returns the stored session the connect resumes (SCACI
// §3.3.1), or nil when the connect starts a new session. The session is
// looked up as the connecting application center's - its organization and
// acEui - so any other application center presenting the snAcUuid finds none
// and starts a session of its own, like every other refused resume. A stored
// session of another tenant fails the connect closed.
func (hs *handshakeService) resumedSession(ctx context.Context, req *scaci.Connect, tenantID int64, orgID uuid.UUID) (*scaci.Session, string) {
	if req.SnAcOpId == nil || req.SnScOpId == nil {
		return nil, ""
	}
	ac := scaci.ApplicationCenter{TenantID: tenantID, OrganizationID: orgID, AcEui: req.AcEui}
	resumable, errToken := hs.ResolveResume(ctx, ac, req.SnAcUUID[:], nil, *req.SnAcOpId, *req.SnScOpId, req.Version)
	if errToken != "" {
		hs.logger.WarnContext(ctx, scaci.LogSCACIResumeFailed,
			logger.FieldAcEui, mioty.FormatEUI64(req.AcEui),
			logger.FieldReason, errToken)
		return nil, ""
	}
	if !resumable {
		return nil, ""
	}

	row, err := hs.sessionRepo.GetSessionByAcUUID(ctx, ac.Key(), req.SnAcUUID)
	if err != nil {
		return nil, ""
	}
	if row.TenantID != tenantID {
		hs.logger.ErrorContext(ctx, scaci.LogSCACICrossTenantResumeRejected,
			logger.FieldCertTenantID, tenantID,
			logger.FieldSessionTenantID, row.TenantID)
		return nil, scaci.ErrNoActiveSession
	}

	stored := scaci.RestoreSession(row)
	stored.Resumed = true
	stored.OpenReissueWindow(*req.SnAcOpId)
	hs.logger.InfoContext(ctx, scaci.LogSCACISessionResumed,
		logger.FieldAcEui, mioty.FormatEUI64(req.AcEui),
		logger.FieldSnAcUUID, scaci.FormatUUID(req.SnAcUUID))
	return stored, ""
}

// ResolveResume implements HandshakeService.ResolveResume
//
// Validates session resumption per SCACI §1, §3.3.1 and §§2.1-2.3: the
// session store reports whether the session can still be resumed and the
// operation ID counters and version it stored; the application center's
// operation IDs must agree with those counters (scaci.ResumeOpIDConflict) and
// its version with the negotiated one.
//
// Parameters:
//   - ctx: Request context
//   - ac: The Application Center whose session is looked up
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
	ac scaci.ApplicationCenter,
	acUUID []byte,
	_ []byte, // scUUID not used for lookup
	acOpId, scOpId int64,
	requestVersion string,
) (bool, string) {
	// Convert acUUID to [16]byte for repository call
	var acUUIDBytes [16]byte
	if len(acUUID) != 16 {
		hs.logger.ErrorContext(ctx, scaci.LogSCACIInvalidAcUUIDLength,
			logger.FieldLength, len(acUUID))
		return false, scaci.ErrSnAcUUIDZero
	}
	copy(acUUIDBytes[:], acUUID)

	resumptionInfo, err := hs.sessionRepo.CheckSessionResumable(ctx, ac.Key(), acUUIDBytes)
	if err != nil {
		hs.logger.DebugContext(ctx, scaci.LogSCACIResumeFailed,
			logger.FieldError, err,
			logger.FieldTenantID, ac.TenantID)
		return false, scaci.ErrNoActiveSession
	}

	if resumptionInfo == nil {
		hs.logger.DebugContext(ctx, scaci.LogSCACINoResumableSession,
			logger.FieldTenantID, ac.TenantID)
		return false, scaci.ErrNoActiveSession
	}

	if reason := resumeRefusal(resumptionInfo, acOpId, scOpId); reason != "" {
		hs.logger.WarnContext(ctx, scaci.LogSCACISessionCannotResume,
			logger.FieldAcOpID, acOpId,
			logger.FieldScOpID, scOpId,
			logger.FieldReason, reason)
		return false, scaci.ErrOpIdOutOfOrder
	}

	if hs.versionChangedOnResume(ctx, ac.TenantID, resumptionInfo.NegotiatedVersion, requestVersion) {
		return false, scaci.ErrVersionMismatchOnResume
	}
	return true, ""
}

// resumeRefusal is why the stored session cannot be resumed with the
// application center's operation IDs, or "" when it can (SCACI §1, §3.3.1).
func resumeRefusal(info *models.SCACISessionResumptionInfo, acOpId, scOpId int64) string {
	if !info.CanResume {
		return info.ReasonIfNotResumable
	}
	return scaci.ResumeOpIDConflict(acOpId, scOpId, scaci.OpIDPair{AC: info.LastKnownAcOpId, SC: info.LastKnownScOpId})
}

// versionChangedOnResume reports a resume whose version differs in major or
// minor from the one the session negotiated (SCACI §§2.1-2.3; patch is
// ignored). It fails closed on an unparseable version; a session stored
// without a version resumes with any.
func (hs *handshakeService) versionChangedOnResume(ctx context.Context, tenantID int64, stored, requested string) bool {
	if stored == "" {
		return false
	}
	storedMajor, storedMinor, _, storedErr := scaci.ParseSemanticVersion(stored)
	reqMajor, reqMinor, _, reqErr := scaci.ParseSemanticVersion(requested)
	if storedErr != nil || reqErr != nil {
		hs.logger.WarnContext(ctx, scaci.LogSCACIVersionMismatchOnResume,
			logger.FieldStoredVersion, stored,
			logger.FieldRequestVersion, requested,
			logger.FieldStoredParseErr, storedErr,
			logger.FieldRequestParseErr, reqErr,
			logger.FieldTenantID, tenantID)
		return true
	}
	if storedMajor != reqMajor || storedMinor != reqMinor {
		hs.logger.WarnContext(ctx, scaci.LogSCACIVersionMismatchOnResume,
			logger.FieldStoredMajor, storedMajor, logger.FieldStoredMinor, storedMinor,
			logger.FieldReqMajor, reqMajor, logger.FieldReqMinor, reqMinor,
			logger.FieldTenantID, tenantID)
		return true
	}
	return false
}
