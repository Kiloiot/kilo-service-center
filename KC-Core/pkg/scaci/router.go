package scaci

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	// Shared MIOTY helpers (FormatEUI64, EPStatus)
	// Organization resolver for propagation context
	// BSSCI §5.8-5.8.3 attach propagation contracts
	// Import neutral scheduler contracts

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// SublayerHandler processes one SCACI §4 sublayer command (e.g. "rc.dir").
type SublayerHandler func(conn net.Conn, session *Session, opId int64, payload []byte) error

// RegisterSublayerHandler installs the handler for a sublayer command. The
// community server registers none; an edition that implements a sublayer
// registers its handlers before Start.
func (s *Server) RegisterSublayerHandler(command string, handler SublayerHandler) error {
	idx := strings.IndexByte(command, '.')
	if idx <= 0 {
		return fmt.Errorf("%s: %q", GetErrorDefinition(errUnsupportedSublayerPrefix).Message, command)
	}
	if _, allowed := allowedSublayerPrefixes[command[:idx]]; !allowed {
		return fmt.Errorf("%s: %q", GetErrorDefinition(errUnsupportedSublayerPrefix).Message, command)
	}
	s.sublayerHandlers[command] = handler
	return nil
}

// allowedSublayerPrefixes contains spec-defined sublayer prefixes per SCACI §4.
// Community edition has no handlers; managed/cloud may register rc.* handlers.
var allowedSublayerPrefixes = map[string]struct{}{"rc": {}}

// routeMessage dispatches messages to appropriate handlers per SCACI §3
//
// # This function is the central dispatcher for all SCACI operations
//
// Design:
//   - Connect handler receives certificate (service resolves tenant)
//   - Other handlers use session.TenantID (already resolved)
//
// Parameters:
//   - cert: Client certificate (only used by Connect handler)
//   - payload: Raw MessagePack payload from frame (handlers decode directly)
func (s *Server) routeMessage(conn net.Conn, session **Session, cert *x509.Certificate, command string, opId int64, payload []byte) error {
	// SCACI §3.3-01: Connect must complete before other operations
	if *session != nil && (*session).currentState() == StateConnecting {
		if command != CmdConnect && command != CmdConnectComplete {
			s.sendErrorWithCatalog(conn, *session, opId, POSIX_EINVAL, errConnectWaitingCmp)
			return nil
		}
	}

	// SCACI §3.3.2: opId=0 is reserved for connect handshake only
	// Allowed inbound: CmdConnect, CmdConnectComplete, CmdErrorAck (echoes triggering opId)
	// CmdConnectResponse is SC→AC only, so inbound conRsp with opId=0 is rejected
	if opId == 0 {
		if command != CmdConnect && command != CmdConnectComplete && command != CmdErrorAck {
			// Nil-safe session pointer for sendErrorWithCatalog
			var sess *Session
			if session != nil {
				sess = *session
			}
			s.sendErrorWithCatalog(conn, sess, opId, POSIX_EINVAL, ErrOpIDZeroReserved)
			return nil // Reject but keep connection open (recoverable error)
		}
	}

	// SCACI §3.2: Validate opId sign matches command initiator
	// Must occur BEFORE monotonicity validation and opId persistence
	spec, known := s.commands.lookup(command)
	if errToken := validateCommandOpIDSign(spec, known, opId); errToken != "" {
		var sess *Session
		if session != nil {
			sess = *session
		}
		s.sendErrorWithCatalog(conn, sess, opId, POSIX_EINVAL, errToken)
		return nil // Reject, do NOT persist opId counters
	}

	// A response is only valid for an SC operation that awaits it (SCACI §3.2):
	// the service center completes only operations it initiated.
	if known && spec.Answers != "" && *session != nil && !(*session).SettleScResponse(opId, spec.Answers) {
		s.logger.WarnContext(s.sessionContext(*session), LogSCACIUnsolicitedResponse,
			logger.FieldCommand, command, logger.FieldOpID, opId)
		s.sendErrorWithCatalog(conn, *session, opId, POSIX_EPROTO, errUnsolicitedResponse)
		return nil
	}

	// Validate AC opId monotonicity (skip Connect/ConnectComplete/ErrorAck and handshake completions)
	// ErrorAck echoes the opId from the original error message (SCACI §3.14.2)
	// Handshake completions reuse their request's opId per MIOTY SCACI three-way handshake pattern
	if *session != nil && opId > 0 &&
		command != CmdConnect && command != CmdConnectComplete && command != CmdErrorAck &&
		!s.commands.handshakeCompletion(command) {
		if err := (*session).AcceptAcOpId(opId); err != nil {
			s.sendErrorWithCatalog(conn, *session, opId, POSIX_EINVAL, errOpIdOutOfOrder, err.Error())
			return nil
		}
		s.persistOpIDs(*session)
	}

	// SCACI §4 Sublayer Guard: Handle dotted sublayer commands
	// Runs after opId validation, before handler switch
	if idx := strings.IndexByte(command, '.'); idx > 0 {
		prefix := command[:idx]

		// Extract session safely (double pointer pattern per lines 871-873)
		var sess *Session
		if session != nil {
			sess = *session
		}

		// Build log fields (nil-safe: only add acEui when session exists)
		logFields := []interface{}{logger.FieldCommand, command, logger.FieldPrefix, prefix}
		if sess != nil {
			logFields = append(logFields, "acEui", mioty.FormatEUI64(sess.AcEui))
		}

		// Case 1: Unknown prefix (not in spec allowlist)
		if _, allowed := allowedSublayerPrefixes[prefix]; !allowed {
			s.logger.WarnContext(s.sessionContext(*session), LogSCACIUnsupportedSublayerPrefix, logFields...)
			s.sendErrorWithCatalog(conn, sess, opId, POSIX_ENOTSUP, errUnsupportedSublayerPrefix, command)
			return nil
		}

		// Case 2: Allowed prefix - check if handler exists for the full command
		handler, hasHandler := s.sublayerHandlers[command]
		if !hasHandler {
			// No handler registered for this command
			s.logger.WarnContext(s.sessionContext(*session), LogSCACIUnsupportedSublayerPrefix, logFields...)
			s.sendErrorWithCatalog(conn, sess, opId, POSIX_ENOTSUP, errUnsupportedSublayerPrefix, command)
			return nil
		}

		// Case 3: Handler exists - dispatch directly and return
		// (bypasses switch to avoid double-error from default case)
		s.logger.DebugContext(s.sessionContext(*session), LogSCACISublayerHandlerInvoked, logFields...)
		return handler(conn, sess, opId, payload)
	}

	if !known || spec.Route == nil {
		s.logger.WarnContext(s.sessionContext(*session), LogSCACIUnknownCommand, logger.FieldCommand, command)
		s.sendErrorWithCatalog(conn, *session, opId, POSIX_ENOTSUP, errUnsupportedCommand)
		return nil
	}
	return spec.Route(s, conn, session, cert, opId, payload)
}

// persistOpIDs persists a snapshot of the session's operation ID counter pair
// in one write (SCACI §3.2) so a resume sees both counters of the same moment.
// Writes run concurrently and may land out of order; the store never moves a
// counter back.
func (s *Server) persistOpIDs(session *Session) {
	if session.ID <= 0 {
		return
	}
	pair := session.OpIDs()
	s.persistDetached(session, SessionPersistTimeout, func(ctx context.Context) {
		if err := s.sessionPersistence.PersistOpIDs(ctx, session, pair); err != nil {
			s.logger.ErrorContext(ctx, LogSCACIPersistOpIDsPairFailed, logger.FieldSessionID, session.ID, logger.FieldError, err)
		}
	})
}
