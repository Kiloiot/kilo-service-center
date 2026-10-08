package bssci

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// dlDataQueueRequest is one dlDataQue the service center sends (BSSCI
// §3.12.1): the rows of its table plus the tenant and organization that own
// the downlink. It carries the SendDLDataQueue arguments through the steps.
type dlDataQueueRequest struct {
	epEui        uint64
	queId        int64
	payloads     [][]byte
	cntDepend    bool
	packetCnt    []int64
	format       uint8
	prio         float32
	responseExp  bool
	responsePrio bool
	dlWindReq    bool
	expOnly      bool
	tenantID     int64
	orgID        *uuid.UUID
}

// dlDataQueueOps are the operations of one send: the dlDataQue and, when the
// downlink asks for it, the dlRxStatQry paired with it.
type dlDataQueueOps struct {
	queue   pendingFrame
	query   pendingFrame
	paired  bool
	euiKeys []byte
}

// pendingFrame is a persisted operation's frame and its id.
type pendingFrame struct {
	opID int64
	msg  map[string]interface{}
}

// SendDLDataQueue queues downlink data for an endpoint at the session's base
// station (BSSCI §3.12), a single payload or counter-dependent ones. When
// dlRxStatQry is true (the SCACI §3.10.1 hint), a BSSCI dlRxStatQry
// operation (rev1 §5.16 / classic §3.16) is paired with the queue: both
// operations are durably persisted together before either frame is written,
// and the query frame precedes the queue frame so the DL RX status query is
// scheduled for the queued downlink's transmission.
func (s *Server) SendDLDataQueue(sessionID string, epEui uint64, payloads [][]byte, queId int64,
	prio float32, cntDepend bool, packetCnt []int64, format uint8,
	responseExp bool, responsePrio bool, dlWindReq bool, expOnly bool, tenantID int64,
	orgID *uuid.UUID, dlRxStatQry bool,
) error {
	session, err := s.queueingSession(sessionID)
	if err != nil {
		return err
	}
	if err := validateQueuedUserData(payloads, cntDepend, packetCnt); err != nil {
		return err
	}
	req := dlDataQueueRequest{
		epEui: epEui, queId: queId, payloads: payloads, cntDepend: cntDepend, packetCnt: packetCnt,
		format: format, prio: prio, responseExp: responseExp, responsePrio: responsePrio,
		dlWindReq: dlWindReq, expOnly: expOnly, tenantID: tenantID, orgID: orgID,
	}
	ops, err := s.persistDLDataQueue(session, req, dlRxStatQry)
	if err != nil {
		return err
	}
	if err := s.writeDLDataQueue(session, ops); err != nil {
		return err
	}
	// Register queue-to-tenant mapping for fast result processing (BSSCI §5.14)
	s.tenantResolver.RegisterQueueTenant(queId, strconv.FormatInt(tenantID, 10))
	return nil
}

// queueingSession is the session a dlDataQue is sent over: connected
// (BSSCI §3.3) and bidirectional.
func (s *Server) queueingSession(sessionID string) (*Session, error) {
	session, exists := s.sessions.get(sessionID)
	if !exists {
		return nil, fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}
	if !session.HandshakeComplete {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIConnectHandshakeNotCompleteDL,
			logger.FieldSessionID, sessionID,
			logger.FieldBsEui, session.BaseStationEUI)
		return nil, fmt.Errorf(errFmtTokenForSession, ResolveErrorMessage(errHandshakeNotComplete), sessionID)
	}
	if !session.Bidirectional {
		return nil, fmt.Errorf("%s %s: %w", ResolveErrorMessage(errSessionNotBidirectional),
			mioty.FormatEUI64(session.BaseStationEUI), ErrSessionNotBidirectional)
	}
	return session, nil
}

// validateQueuedUserData checks userData against BSSCI §3.12.1: one entry
// per packet counter when counter-dependent, at most one otherwise (none
// queues a pure acknowledgement), each within the radio payload limit.
func validateQueuedUserData(payloads [][]byte, cntDepend bool, packetCnt []int64) error {
	if cntDepend && len(packetCnt) != len(payloads) {
		return fmt.Errorf(errFmtTokenGotUnequal,
			ResolveErrorMessage(errCounterDependentPayloadMismatch), len(packetCnt), len(payloads))
	}
	if !cntDepend && len(payloads) > 1 {
		return fmt.Errorf(errFmtTokenGotValue, ResolveErrorMessage(errNonCounterDependentMultiPayload), len(payloads))
	}
	return validatePayloadSizes(payloads)
}

// persistDLDataQueue allocates the operation ids and durably records the
// operations before any frame is written (BSSCI rev1 §5.2 / classic §3.2):
// the counter once for the whole pair, then the pending records. IDs are
// never rolled back. The query id is allocated first so its pending row
// precedes the queue row in reissue order.
func (s *Server) persistDLDataQueue(session *Session, req dlDataQueueRequest, dlRxStatQry bool) (dlDataQueueOps, error) {
	ops := dlDataQueueOps{paired: dlRxStatQry, euiKeys: make([]byte, 8)}
	binary.BigEndian.PutUint64(ops.euiKeys, req.epEui)
	if ops.paired {
		ops.query.opID = session.NextScOpID()
	}
	ops.queue.opID = session.NextScOpID()
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(session), session); err != nil {
		return ops, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistSessionCounters), err)
	}
	ops.queue.msg = dlDataQueueMessage(ops.queue.opID, req)
	metadata := dlDataQueueMetadata(session.BaseStationEUI, req)
	if !ops.paired {
		err := s.pendingOps.persist(s.safeCtx(), session, ops.queue.opID, mioty.CmdDLDataQueue, ops.queue.msg, ops.euiKeys, metadata)
		return ops, s.persistFailure(session, ops, err)
	}
	ops.query.msg = map[string]interface{}{
		"command": mioty.CmdDLRxStatusQuery,
		"opId":    ops.query.opID,
		"epEui":   req.epEui,
	}
	if err := s.persistDLRXQueryCorrelation(session, ops.query.opID, req.epEui, ops.euiKeys); err != nil {
		return ops, err
	}
	pair := []*PendingOperation{
		s.pendingOps.build(ops.query.opID, mioty.CmdDLRxStatusQuery, ops.query.msg, ops.euiKeys, nil),
		s.pendingOps.build(ops.queue.opID, mioty.CmdDLDataQueue, ops.queue.msg, ops.euiKeys, metadata),
	}
	return ops, s.persistFailure(session, ops, s.pendingOps.persistBatch(s.safeCtx(), session, pair))
}

// persistFailure logs a failed persistence of the operations; nil passes through.
func (s *Server) persistFailure(session *Session, ops dlDataQueueOps, err error) error {
	if err == nil {
		return nil
	}
	fields := []interface{}{logger.FieldSessionID, session.ID, logger.FieldOpID, ops.queue.opID, logger.FieldError, err}
	if ops.paired {
		fields = append(fields, logger.FieldQryOpID, ops.query.opID)
	}
	s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLDataQueOperation, fields...)
	return err
}

// writeDLDataQueue writes the query frame, then the queue frame. A query
// failure aborts the pair, so the queue frame is never written to a possibly
// corrupt connection. A frame that may be partly on the wire keeps its
// pending rows for the resume reissue and closes the transport; the rows of
// frames that never reached the wire are removed.
func (s *Server) writeDLDataQueue(session *Session, ops dlDataQueueOps) error {
	if ops.paired {
		if err := s.sendMessage(session, ops.query.msg); err != nil {
			s.settleUnsentFrame(session, ops.query.opID, err, ops.query.opID, ops.queue.opID)
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlRxStatQry), err)
		}
	}
	if err := s.sendMessage(session, ops.queue.msg); err != nil {
		s.settleUnsentFrame(session, ops.queue.opID, err, ops.queue.opID)
		return err
	}
	return nil
}

// settleUnsentFrame handles the write failure of the frame of failedOpID: an
// ambiguous write closes the transport and keeps every pending row for the
// resume reissue with the original ids; otherwise nothing reached the wire
// and the rows of unsentOpIDs are removed.
func (s *Server) settleUnsentFrame(session *Session, failedOpID int64, err error, unsentOpIDs ...int64) {
	if errors.Is(err, ErrAmbiguousWrite) {
		s.closeTransportAfterWriteFailure(session, failedOpID, err)
		return
	}
	for _, opID := range unsentOpIDs {
		if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opID); cleanupErr != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToClearPersistedPendingOperation,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opID,
				logger.FieldError, cleanupErr)
		}
	}
}

// dlDataQueueMessage is the dlDataQue frame (BSSCI §3.12.1): cntDepend is
// always present, packetCnt only with it, and the optional fields only when
// set.
func dlDataQueueMessage(opID int64, req dlDataQueueRequest) map[string]interface{} {
	msg := map[string]interface{}{
		"command":   mioty.CmdDLDataQueue,
		"opId":      opID,
		"epEui":     req.epEui,
		"queId":     req.queId,
		"userData":  buildDLDataQueUserData(req.payloads, req.cntDepend),
		"prio":      req.prio,
		"cntDepend": req.cntDepend,
	}
	if req.cntDepend {
		counters := make([]interface{}, len(req.packetCnt))
		for i, counter := range req.packetCnt {
			counters[i] = counter
		}
		msg["packetCnt"] = counters
	}
	if req.format > 0 {
		msg["format"] = req.format
	}
	if req.responseExp {
		msg["responseExp"] = true
	}
	if req.responsePrio {
		msg["responsePrio"] = true
	}
	if req.dlWindReq {
		msg["dlWindReq"] = true
	}
	if req.expOnly {
		msg["expOnly"] = true
	}
	return msg
}

// dlDataQueueMetadata is what a resumed session needs to rebuild the frame
// (reconstitueDLDataQueMessage) and to confirm the downlink under its owner.
func dlDataQueueMetadata(bsEUI uint64, req dlDataQueueRequest) map[string]interface{} {
	encodedPayloads := make([]string, len(req.payloads))
	for i, payload := range req.payloads {
		encodedPayloads[i] = base64.StdEncoding.EncodeToString(payload)
	}
	metadata := map[string]interface{}{
		"bsEui":        bsEUI,
		"epEui":        req.epEui,
		"queId":        req.queId,
		"prio":         req.prio,
		"payloads":     encodedPayloads,
		"cntDepend":    req.cntDepend,
		"packetCnt":    req.packetCnt,
		"format":       req.format,
		"responseExp":  req.responseExp,
		"responsePrio": req.responsePrio,
		"dlWindReq":    req.dlWindReq,
		"expOnly":      req.expOnly,
		// A string, so a JSON round trip cannot turn the tenant id into a float64.
		"tenantID": strconv.FormatInt(req.tenantID, 10),
	}
	if req.orgID != nil {
		// The dlDataQueRsp repair after a crash stays organization-scoped.
		metadata["organizationID"] = req.orgID.String()
	}
	return metadata
}
