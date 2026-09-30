package bssci

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// reclaimStationReservations returns to pending the downlinks the base
// station holds that this session cannot complete. A session that is not
// resumed discards the previous session's state (BSSCI §1): the station no
// longer holds the downlinks it had queued, and only the rows behind the
// dlDataQue operations reissued on resume can still be confirmed by a
// dlDataQueRsp.
func (s *Server) reclaimStationReservations(ctx context.Context, session *Session, reissued []*PendingOperation) {
	if _, err := s.downlinkReclaimer.ReclaimReservations(ctx, session.BaseStationEUI, reissuedQueueIDs(reissued)); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToReclaimReservations,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
	}
	if session.IsResumed {
		return
	}
	if _, err := s.downlinkReclaimer.ReclaimDiscardedQueue(ctx, session.BaseStationEUI); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToReclaimDiscardedQueue,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
	}
}

// reclaimEndpointQueue returns to pending the owner's downlinks for the
// endpoint that the base station held when it took the attach propagate
// issued at propagatedAt: the station discards them with the attachment the
// propagate replaces (BSSCI §3.8), so they are dispatched again at the
// endpoint's next downlink window.
func (s *Server) reclaimEndpointQueue(ctx context.Context, session *Session, ownerTenantID int64, epEUI uint64, propagatedAt time.Time) {
	if _, err := s.downlinkReclaimer.ReclaimEndpointQueue(ctx, ownerTenantID, epEUI, session.BaseStationEUI, propagatedAt); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToReclaimEndpointQueue,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldEpEui, epEUI,
			logger.FieldError, err)
	}
}

// reissuedQueueIDs lists the service center queue ids of the dlDataQue
// operations among the reissued pending operations.
func reissuedQueueIDs(ops []*PendingOperation) []int64 {
	var queIDs []int64
	for _, op := range ops {
		if op.OperationType != mioty.CmdDLDataQueue || op.Metadata == nil {
			continue
		}
		if queID, err := coerceInt64(op.Metadata[models.EventDetailKeyQueID]); err == nil {
			queIDs = append(queIDs, queID)
		}
	}
	return queIDs
}
