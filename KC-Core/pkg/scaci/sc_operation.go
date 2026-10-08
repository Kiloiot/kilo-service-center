package scaci

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// everyApplicationCenter reaches every Application Center of the tenant.
func everyApplicationCenter(*Session) bool { return true }

// applicationCenter reaches the sessions of one Application Center: its acEui
// in its organization, never another organization's of the same acEui.
func applicationCenter(ac ApplicationCenter) func(*Session) bool {
	return func(session *Session) bool { return session.ApplicationCenter() == ac }
}

// broadcastSCOperation initiates the operation, recorded for resume by
// op.record, on every connected session of the tenant that reaches selects,
// and has the holder record it for every such session without a connection
// (SCACI §1). A session that does not get it is reported; the others are
// still served.
func (s *Server) broadcastSCOperation(ctx context.Context, tenantID int64, reaches func(*Session) bool, op scOperation) error {
	targets, heldErr := s.registry.targets(ctx, tenantID, reaches, op.command, op.record)
	var failures []error
	if heldErr != nil {
		s.logger.ErrorContext(ctx, op.failedLog, logger.FieldTenantIDCamel, tenantID, logger.FieldError, heldErr)
		failures = append(failures, fmt.Errorf(errFmtBroadcastHeld, errBroadcastSessionFailed, op.command, heldErr))
	}
	for _, t := range targets {
		if err := s.initiateSCOperation(t.conn, t.session, op); err != nil {
			s.logger.ErrorContext(s.sessionContext(t.session), op.failedLog,
				logger.FieldAcEui, mioty.FormatEUI64(t.session.AcEui),
				logger.FieldError, err)
			failures = append(failures, fmt.Errorf(errFmtBroadcastSession, errBroadcastSessionFailed, op.command, t.session.ID, err))
			continue
		}
		t.session.UpdateLastSeen(s.clock.Now())
	}
	return errors.Join(failures...)
}

// scOperation is one operation the service center initiates (SCACI §3.2
// negative opIds).
type scOperation struct {
	command string
	// failedLog reports a session the operation did not reach.
	failedLog string
	// record persists the operation before its request is written.
	record OperationRecord
	// send writes the request.
	send func(conn net.Conn, session *Session, opId int64) error
}

// initiateSCOperation reserves the next SC opId, records the operation, then
// writes its request - all under the session's issue and write locks, so SC
// operations reach the application center in decrementing order, after
// anything a resume reissues (SCACI §3.2). Recording first lets a resume
// reissue the operation (§1) and lets its response find the record however
// fast it arrives. A session that already has the operation is not sent it
// again, and one whose request cannot be written once it is on record is
// left to its resume.
func (s *Server) initiateSCOperation(conn net.Conn, session *Session, op scOperation) error {
	session.issueMu.Lock()
	defer session.issueMu.Unlock()

	opId := session.NextScOpId()
	s.persistOpIDs(session)
	recording, err := op.record(session, opId)
	if err != nil || recording == RecordedBefore {
		return err
	}
	err = s.sendSCOperation(session, opId, op.command, func() error {
		return op.send(conn, session, opId)
	})
	if err == nil || recording != RecordedNew {
		return err
	}
	s.leaveToResume(conn, session, opId, op.command, err)
	return nil
}

// leaveToResume closes the connection an operation on record could not be
// written to: the session loses its connection and stays resumable, and its
// resume reissues the operation under its original opId (SCACI §1, §3.2).
func (s *Server) leaveToResume(conn net.Conn, session *Session, opId int64, command string, cause error) {
	s.logger.WarnContext(s.sessionContext(session), LogSCACIOperationLeftToResume,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui), logger.FieldOpID, opId,
		logger.FieldCommand, command, logger.FieldError, cause)
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		s.logger.WarnContext(s.sessionContext(session), LogSCACICloseConnectionFailed,
			logger.FieldRemote, conn.RemoteAddr(), logger.FieldError, err)
	}
}

// sendSCOperation writes the request of an SC operation, new or reissued,
// under the session's write lock, after registering the response it awaits
// (SCACI §3.2).
func (s *Server) sendSCOperation(session *Session, opId int64, command string, send func() error) error {
	session.WriteMu.Lock()
	defer session.WriteMu.Unlock()
	session.AwaitScResponse(opId, command)
	return send()
}
