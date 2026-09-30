package scaci

import (
	"context"
	"net"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// admitConnected completes the connect operation of the session on conn at
// completedAt through the registry, which records a resume on the session's
// row. A session issues nothing new until what it held is reissued (SCACI §1,
// §3.2), so it takes its issue gate, which reissueHeld opens. A session
// superseded on conn, or a resumed session no longer held, is refused.
func (s *Server) admitConnected(conn net.Conn, session *Session, completedAt time.Time, tlsVersion, cipherSuite string) bool {
	session.issueMu.Lock()
	ctx, cancel := context.WithTimeout(s.sessionContext(session), ConnectPersistTimeout)
	defer cancel()
	err := s.registry.complete(ctx, conn, session, completedAt, tlsVersion, cipherSuite)
	if err == nil {
		return true
	}
	session.issueMu.Unlock()
	s.logger.WarnContext(s.sessionContext(session), LogSCACIConnectCompleteRefused,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui), logger.FieldSessionID, session.ID, logger.FieldError, err)
	return false
}

// recordConnectComplete moves the connect operation's audit row from
// acknowledged to completed (SCACI §3.3). The audit trail is not on the
// critical path, so a failure is only logged.
func (s *Server) recordConnectComplete(session *Session, opId int64, tlsVersion, cipherSuite string) {
	if session.ID <= 0 || s.operationRepo == nil {
		return
	}
	cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer cmpCancel()

	completeData := map[string]interface{}{
		"acEui":       mioty.FormatEUI64(session.AcEui),
		"resumed":     session.Resumed,
		"tlsVersion":  tlsVersion,
		"cipherSuite": cipherSuite,
	}
	if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId, models.OperationStateCompleted, completeData); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordConnectCmpOpFailed, logger.FieldError, err)
	}
}

// reissueHeld reissues, in the background, the SC operations not completed
// before the connection loss and those recorded while the session was held,
// during a connection loss or its connect operation (SCACI §1, §3.3), then
// activates the session and opens its issue gate.
func (s *Server) reissueHeld(conn net.Conn, session *Session) {
	s.runTracked(func() {
		defer session.issueMu.Unlock()
		s.replayPendingOperations(conn, session)
		session.setState(StateActive)
	})
}

// refuseUnheldResume refuses the resume of a session that is no longer held:
// its state was discarded, so there is nothing left to resume (SCACI §1).
func (s *Server) refuseUnheldResume(conn net.Conn, session *Session, opId int64) error {
	s.logger.WarnContext(s.sessionContext(session), LogSCACIResumedSessionNotHeld,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui), logger.FieldSessionID, session.ID)
	s.recordConnectRefused(conn, session, nil, session.AcEui, errNoActiveSession)
	s.refuseConnection(conn, nil, opId, errNoActiveSession)
	return errResumedSessionNotHeld
}
