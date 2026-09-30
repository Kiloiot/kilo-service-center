package bssci

import (
	"context"
	"encoding/binary"
	"math"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// revokeOperation is the downlink a service center dlDataRev names: its
// endpoint, its queue id and the tenant its revocation is recorded under.
type revokeOperation struct {
	endpointEUI uint64
	queueID     int64
	tenant      string
}

// revokeOperationOf reads the downlink the dlDataRev opID names from its
// recovery record; the queue id is zero when the record names none. The
// tenant is the one recorded with the operation, else the station's, else
// the server default.
func (s *Server) revokeOperationOf(session *Session, opID int64) revokeOperation {
	pendingOp, err := s.statusSvc.GetPendingOperation(session, opID)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
			logger.FieldOpID, opID, logger.FieldError, err)
	}
	var target revokeOperation
	if pendingOp == nil {
		return target
	}
	if len(pendingOp.Endpoint) == 8 {
		target.endpointEUI = binary.BigEndian.Uint64(pendingOp.Endpoint)
	}
	if pendingOp.Metadata == nil {
		return target
	}
	target.queueID = s.recordedQueueID(session, opID, pendingOp.Metadata[models.EventDetailKeyQueID])
	switch recorded, ok := pendingOp.Metadata[models.EventDetailKeyTenantID].(string); {
	case ok:
		target.tenant = recorded
	case session.ResolvedTenantID != 0:
		target.tenant = strconv.FormatInt(session.ResolvedTenantID, 10)
	default:
		target.tenant = s.formatTenantID()
	}
	return target
}

// recordedQueueID reads a recovery record's queue id, which an in-memory
// record holds as an unsigned or signed integer and a reloaded one as a
// float; zero when it is missing or beyond the signed range.
func (s *Server) recordedQueueID(session *Session, opID int64, recorded interface{}) int64 {
	switch queueID := recorded.(type) {
	case int64:
		return queueID
	case float64:
		return int64(queueID)
	case uint64:
		if queueID > math.MaxInt64 {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIQueueIDOutOfRange,
				logger.FieldOpID, opID,
				logger.FieldQueID, queueID,
				logger.FieldReason, reasonQueueIDOverflow)
			return 0
		}
		return int64(queueID)
	}
	return 0
}

// recordRevocation files a revocation in the owner's events; a downlink that
// had already ended keeps its outcome as the only one.
func (s *Server) recordRevocation(ctx context.Context, session *Session, target revokeOperation, opID int64, revoked bool) {
	if !revoked || target.endpointEUI == 0 {
		return
	}
	if err := s.auditLogger.RecordDLRevokeResponse(ctx, target.tenant, session, target.endpointEUI, target.queueID, opID); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordDLDataRevokedEvent, logger.FieldError, err)
	}
}

// revokeUnheldDownlink ends the downlink of a dlDataRev the base station
// answered with error (BSSCI §3.17): the station does not hold it, so it will
// never be transmitted, and it ends revoked as the revoke asked (§3.13), with
// the revocation recorded as a confirmed one is. A downlink that already
// ended, such as one the expiry sweep expired, keeps its outcome.
func (s *Server) revokeUnheldDownlink(ctx context.Context, session *Session, opID int64, code int, message string) {
	target := s.revokeOperationOf(session, opID)
	if target.queueID <= 0 {
		return
	}
	revoked, err := s.downlinkSvc.ProcessRevokeRefusal(ctx, session, RevokeRefusal{
		QueueID: target.queueID, EndpointEUI: target.endpointEUI, Code: code, Message: message,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToRevokeRefusedDownlink,
			logger.FieldQueID, target.queueID,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
		return
	}
	s.recordRevocation(ctx, session, target, opID, revoked)
}
