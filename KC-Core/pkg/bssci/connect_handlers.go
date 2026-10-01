package bssci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgversion "github.com/Kiloiot/kilo-service-center/pkg/version"
)

// defaultOrgForSessionTenant resolves the default organization for the
// session's current tenant (community fallback path); uuid.Nil when
// unresolvable.
func (s *Server) defaultOrgForSessionTenant(ctx context.Context, session *Session) uuid.UUID {
	if s.orgResolver == nil {
		return uuid.Nil
	}
	orgID, err := s.orgResolver.GetDefaultOrgForTenant(ctx, session.ResolvedTenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToResolveDefaultOrgForBSSCISession,
			logger.FieldError, err,
			logger.FieldTenantID, session.ResolvedTenantID)
		return uuid.Nil
	}
	return orgID
}

// formatTenantID formats the server's tenant ID as a string for database operations
//
// This helper centralizes tenant ID formatting to ensure consistency across
// all event logging and database persistence operations.
//
// Returns:
//   - Tenant ID formatted as decimal string (e.g., "42")
func (s *Server) formatTenantID() string {
	return fmt.Sprintf("%d", s.tenantID)
}

// handleConnect handles the connect operation per BSSCI specification
func (s *Server) handleConnect(session *Session, msg *Message, data map[string]interface{}) error {
	// Connect messages are only valid while no connect operation is in flight
	if session.ConnectState != ConnectStateAwaitingConnect {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingCommandBeforeHandshake,
			logger.FieldCommand, msg.Command,
			logger.FieldConnectState, int(session.ConnectState),
			logger.FieldOpID, msg.OpId)
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidHandshakeState)
	}

	// BSSCI-3.2-03: Connect operation MUST use opId=0
	if msg.OpId != 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidConnectOperationID,
			logger.FieldOpIDSnake, msg.OpId)
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidConnectOpId)
	}

	if rejection := s.readConnectRequest(session, data); rejection != nil {
		return s.rejectConnect(session, msg.OpId, rejection.Posix, rejection.Token)
	}

	// Registration and tenant authorization precede the connect response
	// (§5.3.2): an unregistered or unauthorized base station is rejected via
	// the error/errorAck sequence before any conRsp is offered
	baseStation, admitted := s.admitStation(s.sessionContext(session), session)
	if !admitted {
		return s.rejectConnect(session, msg.OpId, POSIX_EPERM, errBaseStationNotRegistered)
	}
	session.pendingBaseStation = baseStation

	if fix, hasFix := parseGeoFix(data["geoLocation"]); hasFix {
		session.GeoLocation = []float64{fix.latitude, fix.longitude, fix.altitude}
	}

	if rejection := s.negotiateSession(s.sessionContext(session), session, data); rejection != nil {
		return s.rejectConnect(session, msg.OpId, rejection.Posix, rejection.Token)
	}

	// The session stays provisional until conCmp: live-session map and
	// resumable-index registration happen in handleConnectComplete
	s.logger.InfoContext(s.safeCtx(), LogBSSCIBaseStationConnected,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldName, session.Name,
		logger.FieldClientVersion, session.ClientVersion,
		logger.FieldNegotiatedVersion, session.NegotiatedVersion,
		logger.FieldResumed, session.IsResumed)

	return s.sendConnectResponse(session, msg.OpId)
}

// readConnectRequest validates the con fields and records them on the
// provisional session (BSSCI rev1 §5.3.1).
func (s *Server) readConnectRequest(session *Session, data map[string]interface{}) *CatalogError {
	// version is mandatory in the connect message (BSSCI rev1 §5.3.1;
	// message metadata declares it Required)
	version, hasVersion := data["version"].(string)
	if !hasVersion {
		return NewCatalogError(errMandatoryFieldMissing, POSIX_EPROTO)
	}
	bsEUIRaw, hasBsEUI := data["bsEui"]
	if !hasBsEUI {
		return NewCatalogError(errMissingBsEui, POSIX_EPROTO)
	}
	selectedVersion, rejection := s.negotiateConnectVersion(version)
	if rejection != nil {
		return rejection
	}

	// bsEui is a full-range unsigned EUI-64 (BSSCI §5.3.1); values above
	// INT64_MAX are valid
	bsEUI, euiErr := coerceUint64(bsEUIRaw)
	if euiErr != nil {
		return NewCatalogError(errInvalidBsEui, POSIX_EPROTO)
	}

	session.BaseStationEUI = bsEUI
	session.ClientVersion = version             // Raw BS-provided version for audit
	session.NegotiatedVersion = selectedVersion // Negotiated version carried in conRsp (BSSCI §5.3.2)
	s.readStationDescription(session, data)
	return readBidiFlag(session, data)
}

// negotiateConnectVersion selects the version the service center will speak;
// the conRsp carries it and the base station agrees via conCmp or rejects
// (BSSCI-2.1-01, BSSCI-2.2-02, rev1 §4.2, §5.3.2).
func (s *Server) negotiateConnectVersion(version string) (string, *CatalogError) {
	selectedVersion, negErr := s.versionNegotiator.Negotiate(s.safeCtx(), version)
	if negErr == nil {
		return selectedVersion, nil
	}
	var catErr *CatalogError
	if errors.As(negErr, &catErr) {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIVersionIncompatible,
			logger.FieldClientVersionSnake, version,
			logger.FieldError, catErr.Token)
		return "", catErr
	}
	return "", NewCatalogError(errVersionIncompatible, POSIX_EPROTO)
}

// readStationDescription records the optional descriptive con fields.
func (s *Server) readStationDescription(session *Session, data map[string]interface{}) {
	session.Vendor = getStringField(data, "vendor", "")
	session.Model = getStringField(data, "model", "")
	session.Name = getStringField(data, "name", "")
	session.SoftwareVersion = getStringField(data, "swVersion", "")

	// BSSCI §5.3: Capture connect info object for session persistence
	infoValue, hasInfo := data["info"]
	if !hasInfo {
		return
	}
	infoJSON, err := json.Marshal(infoValue)
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToMarshalConnectInfo, logger.FieldError, err)
		return
	}
	session.ConnectInfo = infoJSON
}

// readBidiFlag records the bidi flag, which must be explicitly declared
// (BSSCI §5.3).
func readBidiFlag(session *Session, data map[string]interface{}) *CatalogError {
	bidi, ok := data["bidi"].(bool)
	if !ok {
		return NewCatalogError(errMissingBidiFlag, POSIX_EPROTO)
	}
	session.Bidirectional = bidi
	return nil
}

// sendConnectResponse offers the negotiated session in conRsp (BSSCI §5.3.2).
func (s *Server) sendConnectResponse(session *Session, opID int64) error {
	var sessionUUID mioty.SessionUUID
	copy(sessionUUID[:], session.SessionUUID)
	s.warnUnreleasedSoftwareVersion(session, sessionUUID)

	response := mioty.ConnectResponse{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdConnectResponse,
			OpId:        opID,
		},
		Version:   session.NegotiatedVersion, // SC canonical version, not client echo
		ScEui:     s.config.ServiceCenterEUI,
		Vendor:    &s.config.Vendor,
		Model:     &s.config.Model,
		Name:      &s.config.Name,
		SwVersion: &s.config.SoftwareVersion,
		SnResume:  session.IsResumed,
		SnScUuid:  sessionUUID,
	}

	if err := s.sendMessage(session, response); err != nil {
		session.ConnectState = ConnectStateTerminal
		return err
	}
	session.ConnectState = ConnectStateAwaitingConnectComplete
	return nil
}

// warnUnreleasedSoftwareVersion flags a conRsp whose optional swVersion is
// missing or a development build (non-fatal per BSSCI §5.3.2).
func (s *Server) warnUnreleasedSoftwareVersion(session *Session, sessionUUID mioty.SessionUUID) {
	switch s.config.SoftwareVersion {
	case "":
		s.logger.WarnContext(s.safeCtx(), LogBSSCISoftwareVersionNotConfiguredConnectResponseWillOmitSwVersionField,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldSessionUUID, sessionUUID)
	case pkgversion.DevVersion, pkgversion.DevLocalVersion:
		s.logger.WarnContext(s.safeCtx(), LogBSSCIUsingDevelopmentSoftwareVersionInConnectResponse,
			logger.FieldSwVersion, s.config.SoftwareVersion,
			logger.FieldBsEui, session.BaseStationEUI)
	}
}
