package bssci

import (
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// startStatusMechanism starts periodic status requests to the base station
func (s *Server) startStatusMechanism(session *Session) {
	session.mu.Lock()
	// Guard against multiple goroutines
	if session.stopStatus != nil {
		session.mu.Unlock()
		s.logger.DebugContext(s.sessionContext(session), LogBSSCIStatusMechanismAlreadyRunningForSession,
			logger.FieldSessionIDCamel, session.ID)
		return
	}

	stopStatus := make(chan struct{})
	session.stopStatus = stopStatus
	session.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		initialDelay := time.NewTimer(s.config.statusRequestInitialDelay())
		defer initialDelay.Stop()
		select {
		case <-initialDelay.C:
		case <-stopStatus:
			s.logger.InfoContext(s.sessionContext(session), LogBSSCIStoppingStatusMechanism,
				logger.FieldBsEui, session.BaseStationEUI)
			return
		}
		if _, err := s.requestPeriodicStatus(session); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIInitialStatusRequestFailed,
				logger.FieldBsEui, session.BaseStationEUI,
				logger.FieldError, err)
		}

		ticker := time.NewTicker(s.config.statusRequestInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := s.requestPeriodicStatus(session); err != nil {
					s.logger.ErrorContext(s.sessionContext(session), LogBSSCIStatusRequestFailedInPeriodicLoop,
						logger.FieldBsEui, session.BaseStationEUI,
						logger.FieldError, err)
				}
			case <-stopStatus:
				s.logger.InfoContext(s.sessionContext(session), LogBSSCIStoppingStatusMechanism,
					logger.FieldBsEui, session.BaseStationEUI)
				return
			}
		}
	}()
}

// SendStatusRequest sends an operator's status request to the base station;
// the gRPC service calls it. The station's answer is announced as a
// basestation_status_answered event.
func (s *Server) SendStatusRequest(session interface{}) (int64, error) {
	sess, ok := session.(*Session)
	if !ok || sess == nil {
		return 0, fmt.Errorf("%s", ResolveErrorMessage(errInvalidSessionType))
	}
	opId, frame, err := s.prepareStatusRequest(sess, map[string]interface{}{metadataKeyOperatorRequested: true})
	if err != nil {
		return 0, err
	}
	if err := s.writeStatusRequest(sess, opId, frame); err != nil {
		return 0, err
	}
	return opId, nil
}

// requestPeriodicStatus sends the status mechanism's periodic request; its
// answers are samples, announced to nobody.
func (s *Server) requestPeriodicStatus(sess *Session) (int64, error) {
	opId, frame, err := s.prepareStatusRequest(sess, nil)
	if err != nil {
		return 0, err
	}
	return opId, s.writeStatusRequest(sess, opId, frame)
}

// prepareStatusRequest allocates the operation ID and persists the pending
// record with its metadata. Durable order (BSSCI rev1 §5.2 / classic §3.2):
// allocate the ID, persist the counter, persist the pending record, then
// write the frame; the counter is never rolled back on failure.
func (s *Server) prepareStatusRequest(sess *Session, metadata map[string]interface{}) (int64, map[string]interface{}, error) {
	opId, err := s.pendingOps.begin(s.sessionContext(sess), sess)
	if err != nil {
		s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToSendStatusRequest,
			logger.FieldBsEui, sess.BaseStationEUI,
			logger.FieldError, err)
		return 0, nil, err
	}

	statusRequest := map[string]interface{}{
		"command": mioty.CmdStatus,
		"opId":    opId,
	}

	// The recovery record must be durable before the operation goes on the
	// wire: an SC operation whose pending row was never persisted cannot be
	// reissued on resume. A persistence failure aborts the send, leaving only
	// a consumed-ID gap.
	if err := s.pendingOps.persist(s.safeCtx(), sess, opId, mioty.CmdStatus, statusRequest, nil, metadata); err != nil {
		s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToPersistPendingStatusOperation, logger.FieldError, err)
		return 0, nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistPendingStatusOperation), err)
	}
	return opId, statusRequest, nil
}

// writeStatusRequest writes a prepared status request, settling its pending
// record when the write fails.
func (s *Server) writeStatusRequest(sess *Session, opId int64, statusRequest map[string]interface{}) error {
	s.logger.DebugContext(s.sessionContext(sess), LogBSSCISendingStatusRequestToBaseStation,
		logger.FieldBsEui, sess.BaseStationEUI,
		logger.FieldOpID, opId)

	err := s.sendMessage(sess, statusRequest)
	if err == nil {
		return nil
	}
	s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToSendStatusRequest,
		logger.FieldBsEui, sess.BaseStationEUI,
		logger.FieldError, err)
	if errors.Is(err, ErrAmbiguousWrite) {
		// The frame may be partially on the wire: keep the pending row for
		// resume reissue with the original ID and close the transport.
		s.closeTransportAfterWriteFailure(sess, opId, err)
	} else if cleanupErr := s.pendingOps.remove(s.sessionContext(sess), sess, opId); cleanupErr != nil {
		// Nothing reached the wire; the pending row is removed so resume
		// does not reissue an operation that was never sent.
		s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToCleanupPendingOpAfterSendFailure,
			logger.FieldError, cleanupErr,
			logger.FieldOpID, opId)
	}
	return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendStatusRequest), err)
}

// announceStatusAnswer records the station's answer, received at answeredAt,
// to a pending operator status request in its activity, including one
// reissued after a resume; answers to the periodic poll and repeated answers
// are not recorded.
func (s *Server) announceStatusAnswer(session *Session, opID int64, answeredAt time.Time) {
	request, err := s.statusSvc.GetPendingOperation(session, opID)
	if err != nil || !getBoolField(request.Metadata, metadataKeyOperatorRequested, false) {
		return
	}
	s.recordStationEvent(s.sessionContext(session), session.BaseStationEUI,
		models.EventTypeBaseStationStatusAnswered, answeredAt, map[string]interface{}{models.EventDetailKeyOpID: opID})
}
