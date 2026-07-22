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

	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
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
	certCN := "unknown"
	if cert != nil && cert.Subject.CommonName != "" {
		certCN = cert.Subject.CommonName
	}
	s.logger.DebugContext(s.safeCtx(), LogSCACIProcessingConnect, "certCN", certCN)

	// Step 1: Decode Connect message from payload (transport layer)
	var req Connect
	if err := msgpack.Unmarshal(payload, &req); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogSCACIDecodeConnectFailed, "error", err)
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errInvalidConnectFormat)
	}

	// Step 2: Validate transport-layer requirements per SCACI §3.3

	// SCACI §3.3-02: Connect MUST use opId == OpIDConnect (0)
	if opId != OpIDConnect {
		s.logger.ErrorContext(s.safeCtx(), LogSCACIConnectOpIDMustBeZero, "opId", opId)
		_ = s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errConnectOpIdMustBeZero)
		_ = conn.Close()
		return fmt.Errorf("invalid connect opId")
	}

	// SCACI §3.3.1-01: Validate mandatory fields via sessionValidator
	if errToken := s.sessionValidator.ValidateConnectFields(&req); errToken != "" {
		_ = s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errToken)
		_ = conn.Close()
		return fmt.Errorf("connect validation failed: %s", errToken)
	}

	// Debug: Log resume field state
	s.logger.DebugContext(s.safeCtx(), LogSCACIConnectResumeFields,
		"hasSnAcOpId", req.SnAcOpId != nil,
		"hasSnScOpId", req.SnScOpId != nil)

	// Step 3: Delegate to HandshakeService for business logic
	// Service handles: tenant resolution, version negotiation, session resumption, session creation, metadata
	ctx := s.safeCtx()
	newSession, resp, errToken := s.handshakeSvc.ValidateConnect(ctx, &req, cert)
	if errToken != "" {
		// Service returned error token - sendErrorWithCatalog handles POSIXCode resolution
		// (version errors have POSIXCode=POSIX_ENOTSUP in catalog; others use default)
		_ = s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errToken)
		_ = conn.Close()
		return fmt.Errorf("handshake validation failed: %s", errToken)
	}

	// Enforce organization context in strict mode
	// When org_enforcement_enabled=true, reject sessions with nil organization UUID
	if s.config.OrgEnforcementEnabled && newSession.OrganizationID == uuid.Nil {
		s.logger.WarnContext(s.sessionContext(newSession), LogSCACIOrgEnforcementNilUUID,
			"acEui", newSession.AcEui,
			"orgEnforcementEnabled", true)
		_ = s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errOrgHeaderRequired)
		_ = conn.Close()
		return fmt.Errorf("org enforcement: organization UUID required but not resolved from certificate")
	}

	// Step 4: Update session mapping (transport concern)
	// Map connection → session for future message routing
	s.sessionsMu.Lock()
	if newSession.Resumed {
		// Remove old connection mapping BEFORE assigning new one
		for oldConn, sess := range s.sessions {
			if sess.ID == newSession.ID && oldConn != conn {
				delete(s.sessions, oldConn)
				break
			}
		}
	}
	s.sessions[conn] = newSession
	s.sessionsMu.Unlock()
	*session = newSession

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
			s.logger.ErrorContext(s.sessionContext(newSession), LogSCACIPersistSessionFailed, "error", err)
			_ = s.sendErrorWithCatalog(conn, newSession, opId, POSIX_EINVAL, ErrInternalError)
			_ = conn.Close()
			return fmt.Errorf("persist connect sync failed: %w", err)
		}
		newSession.ID = id
		// Mark as already persisted so handleConnectComplete doesn't re-create
		newSession.SyncPersisted = true
	}

	// Step 4c: Record Connect request for audit trail (§3.3)
	// Connect is always logged (unlike Ping which is configurable)
	// Enriched payload per §3.3.1: vendor/model/name/swVersion/info + resume fields
	if newSession.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(newSession), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		requestData := map[string]interface{}{
			"version":  req.Version,
			"acEui":    pkgmioty.FormatEUI64(req.AcEui),
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
			s.logger.WarnContext(s.sessionContext(newSession), LogSCACIRecordConnectOpFailed, "error", err)
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
			"scEui":    pkgmioty.FormatEUI64(resp.ScEui),
			"snScUuid": hex.EncodeToString(resp.SnScUUID[:]),
			"snResume": resp.SnResume,
		}
		if err := s.operationRepo.UpdateOperationState(rspCtx, newSession.ID, opId, models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(newSession), LogSCACIRecordConnectRspOpFailed, "error", err)
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
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Per SCACI §3.3, the entire connect handshake must use opId=OpIDConnect (0)
	if opId != OpIDConnect {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIConnectCmpNonZeroOpID,
			"opId", opId,
			"acEui", pkgmioty.FormatEUI64(session.AcEui))
		_ = s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errConCmpOpIDMustBeZero)
		return conn.Close()
	}

	// Capture TLS state before goroutine
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

	// Capture TLS version and cipher suite per SCACI §1 evidence requirements
	tlsVersion := TLSVersionName(state.Version)
	cipherSuite := tls.CipherSuiteName(state.CipherSuite)

	// Transition to active
	s.sessionsMu.Lock()
	session.State = StateActive
	session.UpdateLastSeen()

	// Session already contains all necessary data
	s.sessionsMu.Unlock()

	s.logger.DebugContext(s.sessionContext(session), LogSCACIConnectComplete,
		"acEui", pkgmioty.FormatEUI64(session.AcEui),
		"resumed", session.Resumed)

	// Update ConnectComplete state for audit trail (§3.3)
	// State transition: acknowledged → completed (completes three-way handshake)
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completeData := map[string]interface{}{
			"acEui":       pkgmioty.FormatEUI64(session.AcEui),
			"resumed":     session.Resumed,
			"tlsVersion":  tlsVersion,
			"cipherSuite": cipherSuite,
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId, models.OperationStateCompleted, completeData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordConnectCmpOpFailed, "error", err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	// Persist async - delegate to SessionPersistence service
	// Use session.NegotiatedVersion set during handleConnect (SCACI §§2.1-2.3)
	// Skip if session was sync-persisted in handleConnect (SyncPersisted flag set for fresh sessions)
	if !session.SyncPersisted {
		s.sessionPersistence.PersistConnectAsync(s.sessionContext(session), session, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, session.NegotiatedVersion)
	}

	// SCACI §1: Replay pending operations on successful session resume
	// Only SC-originated operations (negative opIds) are replayed: CmdULData and CmdDLDataResult
	if session.Resumed {
		go s.replayPendingOperations(conn, session)
	}

	return nil
}
