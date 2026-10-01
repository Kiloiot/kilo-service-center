package bssci

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// handleMessage routes messages to appropriate handlers
func (s *Server) handleMessage(session *Session, msg *Message, data map[string]interface{}) error {
	// BSSCI §2.4: Normalize incoming payload to validate fields and detect unknown fields
	// Forward compatibility per §2.4-01
	// Issue #3-4 Fix: Only normalize BS→SC commands (inbound) to avoid validating our own SC→BS responses
	ctx := s.sessionContext(session)

	// Only normalize inbound BS→SC and bidirectional commands (skip SC→BS responses and unknown commands for safety)
	if s.commands.shouldNormalize(msg.Command) {
		normalizedData, err := normalizePayload(ctx, s.logger, msg.Command, data)
		if err != nil {
			// Normalization failed (mandatory field missing, invalid type, etc.)
			// Send error response per BSSCI protocol
			s.logger.ErrorContext(ctx, LogBSSCIPayloadNormalizationFailed,
				logger.FieldCommand, msg.Command,
				logger.FieldOpID, msg.OpId,
				logger.FieldError, err)

			// BSSCI §2.4: All normalization errors are protocol violations (EPROTO)
			// Per spec: "Missing mandatory fields or present optional fields with invalid values
			// must be considered a protocol error"
			errToken := errInvalidMessageFormat
			switch {
			case errors.Is(err, ErrMandatoryFieldMissing):
				errToken = errMandatoryFieldMissing
			case errors.Is(err, ErrInvalidFieldType):
				errToken = errInvalidFieldType
			case errors.Is(err, ErrResponseExpRequiresDlOpen):
				errToken = errResponseExpRequiresDlOpen
			case errors.Is(err, ErrConditionalRuleFailed):
				errToken = errConditionalRuleFailed
			}

			// During the connect handshake a normalization failure enters the
			// unified error sequence: the error replaces conRsp/conCmp and
			// awaits errorAck (§5.17), and a failed error write terminates the
			// connection. After activation it is a per-operation error.
			if !session.HandshakeComplete {
				return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errToken)
			}
			if sendErr := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken)); sendErr != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToSendErrorResponse, logger.FieldError, sendErr)
			}
			return nil // Error sent via protocol, don't close connection
		}
		// Use normalized data for handler (unknown fields removed, types coerced)
		data = normalizedData
	}

	// BSSCI-3.3-03: until the connect operation completes only its own
	// messages and the error exchange are legal (an inbound conRsp included,
	// since only the service center sends it); afterwards its messages are
	// unexpected on the active session
	role := s.commands.role(msg.Command)
	if !session.HandshakeComplete && role != roleHandshake && role != roleErrorExchange {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingCommandBeforeHandshake,
			logger.FieldCommand, msg.Command,
			logger.FieldConnectState, int(session.ConnectState),
			logger.FieldOpID, msg.OpId,
			logger.FieldBsEui, session.BaseStationEUI)
		// Route through the unified reject so the error is followed by an
		// errorAck exchange (§5.17) and a failed write terminates cleanly
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errCommandBeforeHandshake)
	}
	if session.HandshakeComplete && role == roleHandshake {
		// The error replaces the unexpected message's sequence (rev1 §5.17);
		// the active session and its connect state stay untouched
		s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingConnectMessageOnActiveSession,
			logger.FieldCommand, msg.Command,
			logger.FieldConnectState, int(session.ConnectState),
			logger.FieldOpID, msg.OpId,
			logger.FieldBsEui, session.BaseStationEUI)
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidHandshakeState)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, logger.FieldError, err)
		}
		return nil
	}

	// Check for sublayer prefixes (BSSCI-4-02)
	// Allow both rc.* (remote control) and vm.* (virtual machine) sublayers
	if strings.Contains(msg.Command, ".") {
		// Only allow known sublayer prefixes
		if !strings.HasPrefix(msg.Command, sublayerPrefixRC) && !strings.HasPrefix(msg.Command, sublayerPrefixVM) {
			// Send error for unsupported sublayer
			if err := s.sendError(session, msg.OpId, POSIX_ENOTSUP, ResolveErrorMessage(errUnsupportedSublayerPrefix)); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, logger.FieldError, err)
			}
			return nil // Don't close connection
		}
	}

	// Direction enforcement (BSSCI §5.5/§5.11-5.16): a command the service
	// center itself sends (DirectionSCtoBS) must never be processed inbound.
	// A base station sending e.g. statusCmp or dlDataQueCmp is a protocol
	// violation. The vm.* sublayer is exempt here - its enforcement lands with
	// the ECE-reserved VM work. Connect-stage SCtoBS handling (conRsp) is
	// already covered by the pre-handshake gate above.
	if !strings.HasPrefix(msg.Command, sublayerPrefixVM) {
		if s.commands.isServiceCenterCommand(msg.Command) {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingInboundServiceCenterCommand,
				logger.FieldCommand, msg.Command,
				logger.FieldOpID, msg.OpId,
				logger.FieldBsEui, session.BaseStationEUI)
			if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInboundServiceCenterCommand)); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, logger.FieldError, err)
			}
			return nil // Don't close connection
		}
	}

	handler, ok := s.commands.inboundHandler(msg.Command)
	if !ok {
		// Send BSSCI error message instead of closing socket (BSSCI-4-01)
		if err := s.sendError(session, msg.OpId, POSIX_ENOTSUP, ResolveErrorMessage(errUnsupportedCommand)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, logger.FieldError, err)
		}
		return nil // Don't close connection
	}

	return handler(s, session, msg, data)
}

// sendMessage sends a message to the Base Station
func (s *Server) sendMessage(session *Session, msg interface{}) error {
	// Wrap to *Message for consistent RawPayload capture
	outMsg, err := s.wrapOutboundMessage(msg)
	if err != nil {
		return fmt.Errorf(errFmtWrapOutboundMessage, err)
	}

	// BSSCI §2.5.1: Validate outbound message fields before encoding.
	// Validation inspects a projection only; failures are returned to the
	// caller - an invalid outbound frame must never trigger sending a protocol
	// error in its place.
	projection, err := outboundValidationProjection(outMsg.Data)
	if err != nil {
		return fmt.Errorf(errFmtOutboundValidationFailed, err)
	}
	if err := s.validateOutboundMessage(session, projection); err != nil {
		var catalogErr *CatalogError
		if errors.As(err, &catalogErr) {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIOutboundValidationFailed,
				logger.FieldErrorTokenSnake, catalogErr.Token,
				logger.FieldCommand, outMsg.Command,
				logger.FieldBsEuiSnake, session.BaseStationEUI)
		}
		return fmt.Errorf(errFmtOutboundValidationFailed, err)
	}

	// Encode the original payload (typed or map) using the negotiated encoding
	// (BSSCI Section 1); uint64 fields survive exactly
	payload, err := encodeMessage(outMsg.Data, session.Encoding)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToEncode), err)
	}

	// Capture encoded payload for forensic analysis
	outMsg.RawPayload = payload

	encoded, err := s.frames.Encode(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errPayloadTooLarge), err)
	}

	// A write that failed after bytes may have reached the wire is ambiguous:
	// the connection's framing can no longer be trusted and callers must close
	// it and rely on session resume for recovery (BSSCI rev1 §5.3.1 / classic
	// §3.3.1). A connection closed before the frame was written in full cannot
	// have delivered it and leaves no framing to protect: a definite failure.
	session.mu.Lock()
	defer session.mu.Unlock()
	if err := s.frames.Send(session.Conn, encoded); err != nil {
		if errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToWritePayload), err)
		}
		return fmt.Errorf("%s: %w: %w", ResolveErrorMessage(errFailedToWritePayload), ErrAmbiguousWrite, err)
	}
	return nil
}

// ErrAmbiguousWrite reports a frame write that failed after bytes may have
// reached the wire. The transport framing can no longer be trusted; callers
// close the connection and rely on session resume (with the original
// operation IDs) for recovery instead of retrying on the same connection.
var ErrAmbiguousWrite = errors.New("ambiguous frame write")

// ErrDispatchOrgMismatch reports a reserved downlink row owned by a different
// organization than the dispatch ran for. The row is released; the send never
// happens.
var ErrDispatchOrgMismatch = errors.New("reserved downlink not owned by requesting organization")

// closeTransportAfterWriteFailure closes the session transport after an
// ambiguous frame write. Persisted pending operations are deliberately
// preserved: the disconnect path marks the session resumable and resume
// reissues the operations with their original IDs.
func (s *Server) closeTransportAfterWriteFailure(session *Session, opId int64, cause error) {
	s.logger.ErrorContext(s.sessionContext(session), LogBSSCIClosingConnectionAfterWriteFailure,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, opId,
		logger.FieldError, cause)
	if session.Conn == nil {
		return
	}
	if err := session.Conn.Close(); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToCloseConnection,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldOpID, opId,
			logger.FieldError, err)
	}
}

// rejectConnect fails the connect operation per BSSCI §5.17: an error frame
// replaces the normal response, the session enters AwaitingConnectErrorAck,
// and the connection stays open until the base station acknowledges with
// errorAck or the handshake read deadline expires. The connection is never
// closed immediately after sending the error.
func (s *Server) rejectConnect(session *Session, opId int64, posix int, errToken string) error {
	if err := s.sendError(session, opId, posix, ResolveErrorMessage(errToken)); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, logger.FieldError, err)
		// Error frame could not be written - the connection is unusable
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errToken), err)
	}
	session.ConnectState = ConnectStateAwaitingConnectErrorAck
	return nil
}

// sendError sends a BSSCI error message (BSSCI-4-01, BSSCI-4-02)
func (s *Server) sendError(session *Session, opId int64, code int, message string) error {
	// Use canonical Error struct (BSSCI §4-4.5)
	errorMsg := mioty.Error{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdError,
			OpId:        opId,
		},
		Code:    code,
		Message: message,
	}

	// Use context.Background() if s.ctx is nil (e.g., in tests)
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background() // context-root: lifecycle-fallback
	}
	s.logger.WarnContext(ctx, LogBSSCISendingBSSCIError,
		logger.FieldOpID, opId,
		logger.FieldCode, code,
		logger.FieldMessage, message)

	if err := s.sendMessage(session, errorMsg); err != nil {
		return err
	}

	// The base station will answer with errorAck (BSSCI rev1 §5.17 / classic
	// §3.17). Record the expectation so handleErrorAck only acts on
	// acknowledgements this service center actually solicited. Plain
	// rejections are ack-only; sendErrorReplacingOperation registers the
	// finalizing disposition for errors that replace a pending SC operation.
	session.registerPendingErrorAck(opId, errorAckAckOnly)
	return nil
}

// sendErrorReplacingOperation sends an error frame that replaces the normal
// response/completion sequence of a known pending SC operation (BSSCI rev1
// §5.17 / classic §3.17). The base station's errorAck then completes that
// operation, so the errorAck is registered with the finalizing disposition.
func (s *Server) sendErrorReplacingOperation(session *Session, opId int64, code int, message string) error {
	if err := s.sendError(session, opId, code, message); err != nil {
		return err
	}
	session.registerPendingErrorAck(opId, errorAckFinalizePendingOperation)
	return nil
}

// sendCatalogError answers opId with the catalog error, resolving its token
// through ResolveErrorMessage. No caller can act on a failed error frame, so
// the failure is logged with the station, session and opId.
func (s *Server) sendCatalogError(session *Session, opId int64, err *CatalogError) {
	sendErr := s.sendError(session, opId, err.Posix, ResolveErrorMessage(err.Token))
	if sendErr == nil {
		return
	}
	s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToSendCatalogError,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldSessionID, session.ID,
		logger.FieldOpID, opId,
		logger.FieldErrorToken, err.Token,
		logger.FieldError, sendErr)
}

// sequenceOperation applies the operation ID rules of rev1 §5.2 / classic
// §3.2 by the command's role and records the base station's open operations:
// a new base-station operation needs a positive ID above every earlier one, a
// completion must name an open operation, and a response to a service-center
// operation carries a negative ID the service center has issued.
func (s *Server) sequenceOperation(session *Session, command string, opId int64) string {
	role := s.commands.role(command)
	if role == roleErrorExchange && opId > 0 {
		session.endBaseStationOperation(opId)
	}
	if !role.sequenced() {
		return ""
	}

	// Connect operation must use ID 0 (BSSCI-3.2-03)
	if session.BaseStationEUI == 0 && opId != 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidConnectOperationID,
			logger.FieldOpIDSnake, opId)
		return errConnectOpIDNotZero
	}

	switch role {
	case roleOpens:
		if opId <= 0 {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIOperationIDNotPositive, logger.FieldOpIDSnake, opId)
			return errOperationIDNotPositive
		}
		if !session.startBaseStationOperation(opId) {
			lastBsOpID, _ := session.OperationCounters()
			s.logger.WarnContext(s.safeCtx(), LogBSSCIOperationIDBackwards,
				logger.FieldOpIDSnake, opId,
				logger.FieldLastBsOpID, lastBsOpID)
			return errOperationIDBackwards
		}
	case roleCloses:
		if opId <= 0 {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIOperationIDNotPositive, logger.FieldOpIDSnake, opId)
			return errOperationIDNotPositive
		}
		if !session.completeBaseStationOperation(opId) {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIOperationNotOpen,
				logger.FieldCommand, command,
				logger.FieldOpIDSnake, opId)
			return errOperationNotOpen
		}
	default:
		return s.sequenceServiceCenterResponse(session, opId)
	}
	return ""
}

// sequenceServiceCenterResponse checks a response to a service-center
// operation: its ID is negative and was issued already (IDs between the
// newest issued one and zero belong to pending operations).
func (s *Server) sequenceServiceCenterResponse(session *Session, opId int64) string {
	if opId >= 0 {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCISCOperationIDMustNotIncrease,
			logger.FieldOpID, opId)
		return errSCOperationIDMustBeNegative
	}
	// A session that has issued nothing yet accepts any negative ID.
	_, lastScOpID := session.OperationCounters()
	if lastScOpID != 0 && opId < lastScOpID {
		s.logger.WarnContext(s.safeCtx(), LogBSSCISCOperationIDMustNotIncrease,
			logger.FieldOpIDSnake, opId,
			logger.FieldLastScOpID, lastScOpID)
		return errOperationIDIncreasing
	}
	return ""
}
