package scaci

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Recording is what recording a service center operation for a session left
// on record.
type Recording int

const (
	// NotRecorded means nothing a resume could reissue is on record, so the
	// request is sent and a failed write is a failed delivery.
	NotRecorded Recording = iota
	// RecordedNew means the operation is on record under the opId just reserved;
	// the session's resume reissues it until it completes (SCACI §1).
	RecordedNew
	// RecordedBefore means the session already has the operation under its
	// original opId (SCACI §3.2), so it is not issued again.
	RecordedBefore
)

// OperationRecord records a service center operation for a session under
// the opId reserved for it.
type OperationRecord func(session *Session, opId int64) (Recording, error)

// recordForResume records an SC operation a resume must be able to reissue;
// an operation that cannot be recorded must not be sent.
func (s *Server) recordForResume(ctx context.Context, command string, requestData map[string]interface{}) OperationRecord {
	return func(session *Session, opId int64) (Recording, error) {
		if session.ID <= 0 {
			return NotRecorded, nil
		}
		recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dbconfig.DefaultQueryTimeout)
		defer cancel()
		if err := s.operationRecorder.Record(recCtx, session, opId, command, models.OperationDirectionOutbound, requestData); err != nil {
			return NotRecorded, fmt.Errorf("%w: %w", errRecordSCOperation, err)
		}
		return RecordedNew, nil
	}
}

// recordUplinkForResume records the ulData operation delivering the stored
// uplink sourceMessageID for resume, once per session: a session that has it
// keeps its original operation, so a retried delivery never reaches it again
// as a new, non-duplicate operation (SCACI §3.2, §3.8.1).
func (s *Server) recordUplinkForResume(ctx context.Context, sourceMessageID string, requestData map[string]interface{}) OperationRecord {
	return func(session *Session, opId int64) (Recording, error) {
		if session.ID <= 0 {
			return NotRecorded, nil
		}
		recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dbconfig.DefaultQueryTimeout)
		defer cancel()
		operation, recorded, err := s.operationRecorder.EnsureUplinkOperation(recCtx, session, opId, sourceMessageID, requestData)
		if err != nil {
			return NotRecorded, fmt.Errorf("%w: %w", errRecordSCOperation, err)
		}
		if recorded {
			return RecordedNew, nil
		}
		s.logger.DebugContext(s.sessionContext(session), LogSCACIUplinkAlreadyRecorded,
			logger.FieldMessageID, sourceMessageID, logger.FieldOpID, operation.OpId, logger.FieldState, operation.State)
		return RecordedBefore, nil
	}
}
