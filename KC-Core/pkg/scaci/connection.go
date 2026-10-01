package scaci

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleConnection processes a single SCACI connection per MIOTY SCACI v1.0.0 §3
//
// Flow:
//  1. Extract client certificate for service-layer tenant resolution
//  2. Wait for Connect message (must be first, opId=0)
//  3. Thread certificate to HandshakeService (service resolves tenant)
//  4. Process subsequent messages via routing
//  5. Clean up session on disconnect
//
// Note: Tenant resolution happens in HandshakeService, NOT at transport layer
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer s.closeConnection(nil, conn)
	defer s.endConnection(conn)
	closeOnShutdown := context.AfterFunc(s.safeCtx(), func() { s.closeConnection(nil, conn) })
	defer closeOnShutdown()

	// Extract client certificate from TLS connection
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		s.logger.ErrorContext(s.safeCtx(), LogSCACIConnectionNotTLS)
		return
	}

	// A peer that stays silent cannot hold the handler: the whole connect,
	// from the TLS handshake to conCmp, must finish in time.
	establishBy := s.clock.Now().Add(s.config.ConnectionEstablishmentTimeout)
	clientCert, ok := s.handshakeTLS(tlsConn, establishBy)
	if !ok {
		return
	}

	// Session will be created after Connect message is received
	// Per spec §3.3: Connect must be first message with opId=0
	var session *Session

	// Process messages until connection closes
	for {
		select {
		case <-s.shutdown:
			return
		default:
		}
		if !s.serveFrame(conn, &session, clientCert, establishBy) {
			return
		}
	}
}

// endConnection releases the connection's session; a session that was live
// lost its connection, which is filed as its close (SCACI §1).
func (s *Server) endConnection(conn net.Conn) {
	// Read before release: a resume of the lost session starts only after release holds it.
	lostAt := s.clock.Now()
	lost := s.registry.release(s.persistContext(), conn, SessionPersistTimeout)
	if lost != nil && lost.connected() {
		s.recordSessionEvent(lost, models.EventTypeSCACISessionClosed, models.SCACISessionClosedConnectionLost, conn, lostAt)
	}
}

// handshakeTLS completes the TLS handshake by establishBy and returns the
// client certificate the handshake service resolves the tenant from.
func (s *Server) handshakeTLS(tlsConn *tls.Conn, establishBy time.Time) (*x509.Certificate, bool) {
	handshakeCtx, cancelHandshake := context.WithDeadline(s.safeCtx(), establishBy)
	err := tlsConn.HandshakeContext(handshakeCtx)
	cancelHandshake()
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogSCACITLSHandshakeFailed,
			logger.FieldRemote, tlsConn.RemoteAddr().String(), logger.FieldError, err)
		return nil, false
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		s.logger.ErrorContext(s.safeCtx(), LogSCACINoClientCertificate)
		return nil, false
	}

	clientCert := state.PeerCertificates[0]
	s.logger.InfoContext(s.safeCtx(), LogSCACIConnectionEstablished,
		logger.FieldRemote, tlsConn.RemoteAddr().String(),
		logger.FieldCertCn, clientCert.Subject.CommonName)
	return clientCert, true
}

// serveFrame reads one frame and routes it; it reports false when the
// connection's read loop ends.
func (s *Server) serveFrame(conn net.Conn, session **Session, clientCert *x509.Certificate, establishBy time.Time) bool {
	if err := conn.SetReadDeadline(connectReadDeadline(*session, establishBy)); err != nil {
		s.logger.ErrorContext(s.sessionContext(*session), LogSCACIReadFrameFailed, logger.FieldError, err)
		return false
	}
	frame, err := s.codec.Read(conn)
	if err != nil {
		s.logReadEnd(*session, conn, err)
		return false
	}

	command, opId, ok := s.frameHeader(conn, *session, frame.Payload)
	if !ok {
		return true
	}
	s.logger.DebugContext(s.sessionContext(*session), LogSCACIReceivedMessage,
		logger.FieldCommand, command,
		logger.FieldOpID, opId)

	// Special handling for Connect (must be first message)
	if *session == nil && command != CmdConnect {
		s.logger.ErrorContext(s.sessionContext(*session), LogSCACIFirstMessageMustBeConnect,
			logger.FieldCommand, command)
		s.sendErrorWithCatalog(conn, *session, opId, POSIX_EINVAL, errConnectRequired)
		return false
	}

	// Route message to handler
	if err := s.routeMessage(conn, session, clientCert, command, opId, frame.Payload); err != nil {
		s.logger.ErrorContext(s.sessionContext(*session), LogSCACIHandlerError,
			logger.FieldCommand, command,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		// Handler should have sent error response
	}
	return true
}

// frameHeader decodes the command and opId every frame carries (SCACI §3.2);
// a frame without them is answered with an error and skipped.
func (s *Server) frameHeader(conn net.Conn, session *Session, payload []byte) (string, int64, bool) {
	var msg map[string]interface{}
	if err := decodePayload(payload, &msg); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDecodeFrameFailed, logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, 0, POSIX_EINVAL, errInvalidMessageFormat)
		return "", 0, false
	}

	command, ok := msg["command"].(string)
	if !ok {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIMissingCommandField)
		s.sendErrorWithCatalog(conn, session, 0, POSIX_EINVAL, errMissingCommandField)
		return "", 0, false
	}

	opId, ok := normalizeInt64(msg["opId"])
	if !ok {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIMissingOpIDField)
		s.sendErrorWithCatalog(conn, session, 0, POSIX_EINVAL, errMissingOpIdField)
		return "", 0, false
	}
	return command, opId, true
}

// connectReadDeadline bounds reads until the connect operation completes
// (SCACI §3.3); an established session reads without a deadline, liveness is
// the ping operation's job (§3.4).
func connectReadDeadline(session *Session, establishBy time.Time) time.Time {
	if session != nil && session.connected() {
		return time.Time{}
	}
	return establishBy
}

// logReadEnd reports why the connection's read loop ended.
func (s *Server) logReadEnd(session *Session, conn net.Conn, err error) {
	ctx := s.sessionContext(session)
	switch {
	case errors.Is(err, io.EOF) || s.safeCtx().Err() != nil:
		s.logger.InfoContext(ctx, LogSCACIConnectionClosed, logger.FieldRemote, conn.RemoteAddr().String())
	case errors.Is(err, os.ErrDeadlineExceeded):
		s.logger.WarnContext(ctx, LogSCACIConnectEstablishmentTimedOut, logger.FieldRemote, conn.RemoteAddr().String())
	default:
		s.logger.ErrorContext(ctx, LogSCACIReadFrameFailed, logger.FieldError, err)
	}
}

// TLSVersionName returns a human-readable TLS version string from the protocol version number.
// Per SCACI §1 TLS evidence persistence requirements.
//
// Parameters:
//   - version: TLS protocol version number from tls.ConnectionState().Version
//
// Returns:
//   - string: Human-readable version name (e.g., "TLS 1.2", "TLS 1.3")
func TLSVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return tlsNameTLS10
	case tls.VersionTLS11:
		return tlsNameTLS11
	case tls.VersionTLS12:
		return tlsNameTLS12
	case tls.VersionTLS13:
		return tlsNameTLS13
	default:
		return fmt.Sprintf(tlsNameUnknownFmt, version)
	}
}

// newFrameCodec frames every SCACI message with the MIOTYA01 header and the
// shared payload cap, bounding each write by writeTimeout
// (protocol.socket_write_timeout).
func newFrameCodec(writeTimeout time.Duration) (nettransport.FrameCodec, error) {
	return nettransport.NewFrameCodec(mioty.SCACIFrameIdentifier, dbconfig.MaxMessageSize, writeTimeout)
}
