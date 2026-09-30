// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// handleConnect processes Connect messages per SCACI §3.3
//
// Handler threads certificate to HandshakeService; service layer resolves
// org UUID to tenant ID via org.Resolver.
//
// Connect Flow (Three-way handshake):
//  1. AC → SC: Connect (opId=0, version, acEui, snAcUuid, optional resume fields)
//  2. SC → AC: ConnectResponse (opId=0, negotiated version, scEui, snResume, snScUuid)
//  3. AC → SC: ConnectComplete (opId=0)
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Validate transport requirements (opId==0, mandatory fields, resume field pairing)
//   - Thread certificate to service (does NOT resolve tenant/org)
//   - Map session to connection (transport concern)
//   - Send response frame
//   - Persist session with organization_id
//
// Service Responsibilities (Business Logic):
//   - Organization + tenant resolution from certificate (via org.Resolver)
//   - Community fallback to default org (when cert parsing fails)
//   - Version negotiation, session resumption, session creation
func (s *Server) handleConnect(conn net.Conn, session **Session, cert *x509.Certificate, opId int64, payload []byte) error {
	// Pre-session logging: Session does not exist until Connect handshake
	// completes, so these sites use s.safeCtx() (a plain value-free context).
	// Certificate CN is logged as an explicit field; tenant resolution happens
	// in HandshakeService.
	certCN := unknownCertCN
	if cert != nil && cert.Subject.CommonName != "" {
		certCN = cert.Subject.CommonName
	}
	s.logger.DebugContext(s.safeCtx(), LogSCACIProcessingConnect, logger.FieldCertCN, certCN)

	// Step 1: Decode Connect message from payload (transport layer)
	var req Connect
	if err := decodePayload(payload, &req); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogSCACIDecodeConnectFailed, logger.FieldError, err)
		errToken := decodeFailureToken(err, errInvalidConnectFormat, errInvalidConnectFormat)
		s.recordConnectRefused(conn, nil, cert, 0, errToken)
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errToken)
		return nil
	}

	// Step 2: Validate transport-layer requirements per SCACI §3.3

	// SCACI §3.3-02: Connect MUST use opId == OpIDConnect (0)
	if opId != OpIDConnect {
		s.logger.ErrorContext(s.safeCtx(), LogSCACIConnectOpIDMustBeZero, logger.FieldOpID, opId)
		s.recordConnectRefused(conn, nil, cert, req.AcEui, errConnectOpIdMustBeZero)
		s.refuseConnection(conn, nil, opId, errConnectOpIdMustBeZero)
		return errInvalidConnectOpId
	}

	// SCACI §3.3.1-01: Validate mandatory fields via sessionValidator
	if errToken := s.sessionValidator.ValidateConnectFields(&req); errToken != "" {
		s.recordConnectRefused(conn, nil, cert, req.AcEui, errToken)
		s.refuseConnection(conn, nil, opId, errToken)
		return fmt.Errorf(errFmtConnectValidationFailed, errToken)
	}

	// Debug: Log resume field state
	s.logger.DebugContext(s.safeCtx(), LogSCACIConnectResumeFields,
		logger.FieldHasSnAcOpId, req.SnAcOpId != nil,
		logger.FieldHasSnScOpId, req.SnScOpId != nil)

	// Step 3: Delegate to HandshakeService for business logic
	// Service handles: tenant resolution, version negotiation, session resumption, session creation, metadata
	ctx := s.safeCtx()
	newSession, resp, errToken := s.handshakeSvc.ValidateConnect(ctx, &req, cert)
	if errToken != "" {
		// Service returned error token - sendErrorWithCatalog handles POSIXCode resolution
		// (version errors have POSIXCode=POSIX_ENOTSUP in catalog; others use default)
		s.recordConnectRefused(conn, nil, cert, req.AcEui, errToken)
		s.refuseConnection(conn, nil, opId, errToken)
		return fmt.Errorf(errFmtHandshakeValidationFailed, errToken)
	}

	// Enforce organization context in strict mode
	// When org_enforcement_enabled=true, reject sessions with nil organization UUID
	if s.config.OrgEnforcementEnabled && newSession.OrganizationID == uuid.Nil {
		s.logger.WarnContext(s.sessionContext(newSession), LogSCACIOrgEnforcementNilUUID,
			logger.FieldAcEui, newSession.AcEui,
			logger.FieldOrgEnforcementEnabled, true)
		s.recordConnectRefused(conn, newSession, cert, req.AcEui, errOrgHeaderRequired)
		s.refuseConnection(conn, nil, opId, errOrgHeaderRequired)
		return errOrgUUIDNotResolved
	}

	// Store negotiated version in session for persistence in handleConnectComplete (SCACI §§2.1-2.3)
	if resp.Version != nil {
		newSession.NegotiatedVersion = *resp.Version
	} else {
		newSession.NegotiatedVersion = ProtocolVersionString
	}

	// Step 4b: Persist fresh sessions synchronously before operation logging (§3.3-04 audit trail)
	// Fresh sessions (ID == 0) must have a real DB ID assigned BEFORE operationRecorder.Record
	// so that Connect audit rows contain real session IDs. Resumed sessions already have ID > 0.
	if newSession.ID == 0 && !newSession.Resumed && s.sessionPersistence != nil {
		// Capture TLS state for sync persistence
		tlsConn := conn.(*tls.Conn)
		state := tlsConn.ConnectionState()

		var certFingerprint, certSubject string
		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			hash := sha256.Sum256(cert.Raw)
			certFingerprint = hex.EncodeToString(hash[:])
			certSubject = cert.Subject.String()
		}
		remoteAddr := conn.RemoteAddr().String()
		tlsVersion := TLSVersionName(state.Version)
		cipherSuite := tls.CipherSuiteName(state.CipherSuite)

		// Synchronous persistence to get real session ID for audit logging
		syncCtx, syncCancel := context.WithTimeout(s.sessionContext(newSession), ConnectPersistTimeout)
		id, err := s.sessionPersistence.PersistConnectSync(syncCtx, newSession, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, newSession.NegotiatedVersion)
		syncCancel()

		if err != nil {
			s.logger.ErrorContext(s.sessionContext(newSession), LogSCACIPersistSessionFailed, logger.FieldError, err)
			s.recordConnectRefused(conn, newSession, cert, req.AcEui, ErrInternalError)
			s.refuseConnection(conn, newSession, opId, ErrInternalError)
			return fmt.Errorf(errFmtPersistConnectSyncFailed, err)
		}
		newSession.ID = id
	}

	// Only a session whose row exists takes the connection over from the
	// previous one of its application center.
	adopted, replaced := s.registry.adopt(s.sessionContext(newSession), conn, newSession)
	if !adopted {
		return s.refuseUnheldResume(conn, newSession, opId)
	}
	s.recordSuperseded(replaced)
	*session = newSession

	// Step 4c: Record Connect request for audit trail (§3.3)
	// Connect is always logged (unlike Ping which is configurable)
	// Enriched payload per §3.3.1: vendor/model/name/swVersion/info + resume fields
	if newSession.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(newSession), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		requestData := map[string]interface{}{
			"version":  req.Version,
			"acEui":    mioty.FormatEUI64(req.AcEui),
			"snAcUuid": hex.EncodeToString(req.SnAcUUID[:]),
			"resumed":  newSession.Resumed,
		}
		// Add optional AC metadata fields per SCACI §3.3.1
		if req.Vendor != nil {
			requestData["vendor"] = *req.Vendor
		}
		if req.Model != nil {
			requestData["model"] = *req.Model
		}
		if req.Name != nil {
			requestData["name"] = *req.Name
		}
		if req.SwVersion != nil {
			requestData["swVersion"] = *req.SwVersion
		}
		// Preserve nested info object for KC-Web consumption
		if req.Info != nil {
			requestData["info"] = req.Info
		}
		// Include resume fields when present (resume attempt indicators)
		if req.SnAcOpId != nil {
			requestData["snAcOpId"] = *req.SnAcOpId
		}
		if req.SnScOpId != nil {
			requestData["snScOpId"] = *req.SnScOpId
		}
		if err := s.operationRecorder.Record(recCtx, newSession, opId, CmdConnect, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(newSession), LogSCACIRecordConnectOpFailed, logger.FieldError, err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	// Step 5: Send ConnectResponse (transport layer)
	// Add BaseMessage fields required by wire protocol
	resp.BaseMessage = BaseMessage{
		Command: CmdConnectResponse,
		OpId:    opId,
	}

	if err := s.SendConnectResponse(conn, *session, resp); err != nil {
		return err
	}

	// Step 5b: Update ConnectResponse state for audit trail (§3.3)
	// State transition: pending → acknowledged (row created in Step 4b)
	if newSession.ID > 0 && s.operationRepo != nil {
		rspCtx, rspCancel := context.WithTimeout(s.sessionContext(newSession), dbconfig.DefaultQueryTimeout)
		defer rspCancel()

		responseData := map[string]interface{}{
			"version":  resp.Version,
			"scEui":    mioty.FormatEUI64(resp.ScEui),
			"snScUuid": hex.EncodeToString(resp.SnScUUID[:]),
			"snResume": resp.SnResume,
		}
		if err := s.operationRepo.UpdateOperationState(rspCtx, newSession.ID, opId, models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(newSession), LogSCACIRecordConnectRspOpFailed, logger.FieldError, err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	return nil
}

// handleConnectComplete processes ConnectComplete messages per SCACI §3.3.3
//
// This completes the three-way handshake for Connect operation.
// The AC acknowledges receipt of ConnectResponse.
//
// No response is sent per spec - handshake is complete.
func (s *Server) handleConnectComplete(conn net.Conn, session *Session, opId int64) error {
	completedAt := s.clock.Now()
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Per SCACI §3.3, the entire connect handshake must use opId=OpIDConnect (0)
	if opId != OpIDConnect {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIConnectCmpNonZeroOpID,
			logger.FieldOpID, opId,
			logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errConCmpOpIDMustBeZero)
		return conn.Close()
	}
	// The connect operation completes once (SCACI §3.3).
	if session.connected() {
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EPROTO, errUnsolicitedResponse)
		return nil
	}

	// Capture TLS version and cipher suite per SCACI §1 evidence requirements
	tlsConn := conn.(*tls.Conn)
	state := tlsConn.ConnectionState()
	tlsVersion := TLSVersionName(state.Version)
	cipherSuite := tls.CipherSuiteName(state.CipherSuite)

	if !s.admitConnected(conn, session, completedAt, tlsVersion, cipherSuite) {
		s.recordConnectRefused(conn, session, nil, session.AcEui, errNoActiveSession)
		return conn.Close()
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIConnectComplete,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui),
		logger.FieldResumed, session.Resumed)

	// Filed before the audit write, so it reaches the log close to the moment it carries.
	s.recordSessionEvent(session, liveEventType(session), "", conn, completedAt)
	s.recordConnectComplete(session, opId, tlsVersion, cipherSuite)
	s.reissueHeld(conn, session)
	return nil
}
