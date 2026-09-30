package scaci

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// validateDLDataQueue checks the SCACI §3.10.1 fields: the endpoint, userData
// against cntDepend and packetCnt, and each payload against the radio limit
// (MIOTY radio protocol §3.6.6.3).
func (s *Server) validateDLDataQueue(ctx context.Context, req *DLDataQueue) string {
	if req.EpEui == 0 {
		return errEpEuiZero
	}
	if req.CntDepend && len(req.PacketCnt) != len(req.UserData) {
		s.logger.WarnContext(ctx, LogSCACICntDependLengthMismatch,
			logger.FieldPacketCntLen, len(req.PacketCnt),
			logger.FieldUserDataLen, len(req.UserData))
		return errCntDependMismatch
	}
	if !req.CntDepend && len(req.PacketCnt) > 0 {
		return errCntDependPacketCntOmit
	}
	// SCACI §3.10.1: "single user data entry if cntDepend is false"
	if !req.CntDepend && len(req.UserData) > 1 {
		s.logger.WarnContext(ctx, LogSCACINonCntDependMultiPayload,
			logger.FieldUserDataLen, len(req.UserData))
		return errNonCntDependMultiPayload
	}
	for i, payload := range req.UserData {
		if len(payload) > mioty.MaxDLUserDataBytes {
			s.logger.WarnContext(ctx, LogSCACIDownlinkPayloadTooLarge,
				logger.FieldEntry, i,
				logger.FieldSize, len(payload),
				logger.FieldMax, mioty.MaxDLUserDataBytes)
			return errDLPayloadTooLarge
		}
	}
	return ""
}

// requireBidirectionalEndpoint refuses a downlink for an endpoint the tenant
// does not have, or one registered without bidi, which never opens a
// downlink window.
func (s *Server) requireBidirectionalEndpoint(ctx context.Context, tenantID int64, epEUI uint64) (string, int) {
	eui := make([]byte, 8)
	binary.BigEndian.PutUint64(eui, epEUI)
	endpoint, epErrToken := s.endpointSvc.GetByEUI(ctx, tenantID, eui)
	if epErrToken == ErrEndpointNotFound {
		return epErrToken, POSIX_ENOENT
	}
	if epErrToken != "" {
		return epErrToken, POSIX_EIO
	}
	if !endpoint.Bidi {
		s.logger.WarnContext(ctx, LogSCACIDownlinkEndpointNotBidirectional,
			logger.FieldEpEui, mioty.FormatEUI64(epEUI),
			logger.FieldTenantIDCamel, tenantID)
		return errEndpointNotBidirectional, POSIX_ENOTSUP
	}
	return "", 0
}

// enqueueOrganization is the organization the downlink is enqueued under:
// the session's, else the tenant's default. Every queue row carries one,
// since dispatch derives the delivery organization from the row; uuid.Nil
// when neither is available, which refuses the downlink.
func (s *Server) enqueueOrganization(ctx context.Context, session *Session) uuid.UUID {
	if session.OrganizationID != uuid.Nil || s.orgResolver == nil {
		return session.OrganizationID
	}
	resolvedOrg, err := s.orgResolver.GetDefaultOrgForTenant(ctx, session.TenantID)
	if err != nil {
		s.logger.WarnContext(ctx, LogSCACIDownlinkOrgResolutionFailed,
			logger.FieldTenantIDCamel, session.TenantID, logger.FieldError, err)
		return uuid.Nil
	}
	return resolvedOrg
}

// queuedDownlink is the session's pending queue row of the request, with the
// SCACI §3.10.1 defaults of its optional fields; the service center queue id
// is assigned when the row is persisted.
func queuedDownlink(req *DLDataQueue, session *Session, enqueueOrg uuid.UUID, acQueID *uint64, ref string) *storage.DownlinkMessage {
	dlMsg := &storage.DownlinkMessage{
		EPEUI:          mioty.FormatEUI64(req.EpEui),
		TenantID:       strconv.FormatInt(session.TenantID, 10),
		OrganizationID: &enqueueOrg,
		Status:         bssci.DLQueueStatusPending,
		Attempts:       dlQueueInitialAttempts,
		MaxAttempts:    dlQueueMaxAttempts,
		ACQueID:        acQueID,
		ACEUI:          queuingApplicationCenter(session, acQueID),
		Ref:            ref,
		CntDepend:      req.CntDepend,
		Priority:       valueOr(req.Prio),
		Format:         valueOr(req.Format),
		ResponseExp:    valueOr(req.ResponseExp),
		ResponsePrio:   valueOr(req.ResponsePrio),
		DlWindReq:      valueOr(req.DlWindReq),
		ExpOnly:        valueOr(req.ExpOnly),
		DlRxStatQry:    valueOr(req.DlRxStatQry),
	}
	if len(req.UserData) > 0 {
		dlMsg.Payload = req.UserData[0]
	}
	if req.CntDepend {
		dlMsg.UserData = req.UserData
		dlMsg.PacketCntArray = make([]int64, len(req.PacketCnt))
		for i, counter := range req.PacketCnt {
			dlMsg.PacketCntArray[i] = int64(counter)
		}
	}
	return dlMsg
}

// queuingApplicationCenter is the EUI of the session's Application Center
// for a request that carries its queue id, which the downlink's result is
// reported to (SCACI §3.12); nil for a gRPC or MQTT request, which carries none.
func queuingApplicationCenter(session *Session, acQueID *uint64) *uint64 {
	if acQueID == nil {
		return nil
	}
	acEui := session.AcEui
	return &acEui
}

// valueOr is the value of an optional field, the zero value when absent.
func valueOr[T any](field *T) T {
	var zero T
	if field == nil {
		return zero
	}
	return *field
}

// persistQueuedDownlink stores the downlink and returns its service center
// queue id for the wire.
func (s *Server) persistQueuedDownlink(ctx context.Context, dlMsg *storage.DownlinkMessage) (uint64, string, int) {
	persistCtx, persistCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer persistCancel()
	stored, err := s.dlSvc.EnqueueDownlink(persistCtx, dlMsg)
	switch {
	case errors.Is(err, storage.ErrDuplicateKey):
		s.logger.WarnContext(ctx, LogSCACIDuplicateQueIDDetected,
			logger.FieldQueID, valueOr(dlMsg.ACQueID),
			logger.FieldTenantIDCamel, dlMsg.TenantID)
		return 0, errQueIDExists, POSIX_EEXIST
	case errors.Is(err, storage.ErrInvalidInput):
		s.logger.WarnContext(ctx, LogSCACIInvalidDownlinkPayload,
			logger.FieldQueID, valueOr(dlMsg.ACQueID),
			logger.FieldError, err)
		return 0, errInvalidDLDataQuePayload, POSIX_EINVAL
	case err != nil:
		s.logger.ErrorContext(ctx, LogSCACIEnqueueDownlinkFailed, logger.FieldError, err)
		return 0, errFailedPersistDownlink, POSIX_EIO
	}
	queID, valid := stored.WireQueueID()
	if !valid {
		s.logger.ErrorContext(ctx, LogSCACIInvalidQueueIDFromDB, logger.FieldQueID, stored.QueID)
		return 0, errFailedPersistDownlink, POSIX_EIO
	}
	return queID, "", 0
}

// recordDLDataQueue records the dlDataQue of a persistent session with the
// service center queue id queID the downlink was stored under, as its audit
// trail; the stored row names the Application Center the result goes to.
func (s *Server) recordDLDataQueue(ctx context.Context, session *Session, opId int64, req *DLDataQueue, dlMsg *storage.DownlinkMessage, queID uint64) {
	if session.ID <= 0 || s.operationRecorder == nil || dlMsg.ACQueID == nil {
		return
	}
	userDataSerialized, err := json.Marshal(req.UserData)
	if err != nil {
		s.logger.WarnContext(ctx, LogSCACIRecordOperationFailed, logger.FieldOpID, opId, logger.FieldError, err)
		return
	}
	recCtx, recCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer recCancel()
	requestData := map[string]interface{}{
		"epEui": dlMsg.EPEUI,
		"queId": operationLogUint64(*dlMsg.ACQueID),
		models.OperationRequestKeyServiceCenterQueueID: operationLogUint64(queID),
		"userData":     string(userDataSerialized),
		"cntDepend":    dlMsg.CntDepend,
		"packetCnt":    dlMsg.PacketCntArray,
		"prio":         dlMsg.Priority,
		"format":       dlMsg.Format,
		"responseExp":  dlMsg.ResponseExp,
		"responsePrio": dlMsg.ResponsePrio,
		"dlWindReq":    dlMsg.DlWindReq,
		"expOnly":      dlMsg.ExpOnly,
		"dlRxStatQry":  dlMsg.DlRxStatQry,
		"tenantId":     session.TenantID,
		"orgId":        dlMsg.OrganizationID.String(),
	}
	if err := s.operationRecorder.Record(recCtx, session, opId, CmdDLDataQueue, models.OperationDirectionInbound, requestData); err != nil {
		s.logger.WarnContext(ctx, LogSCACIRecordOperationFailed, logger.FieldError, err)
	}
}

// dispatchQueuedDownlink hands the persisted row to the BSSCI scheduler by
// its service center queue id, scoped to the organization it was enqueued
// under. A scheduler failure defers the downlink: the row stays pending and
// the dlOpen dispatch delivers it, so failing the request would invite a
// retry that delivers it twice.
func (s *Server) dispatchQueuedDownlink(ctx context.Context, tenantID, opId int64, req *DLDataQueue, queID uint64, enqueueOrg uuid.UUID) DownlinkQueueOutcome {
	dispatchReq := *req
	dispatchReq.QueId = queID
	outcome, schedErrToken := s.dlSvc.QueueDownlink(ctx, &dispatchReq, tenantID, enqueueOrg)
	if schedErrToken != "" {
		s.logger.WarnContext(ctx, LogSCACIDLDataQueueDispatchFailed,
			logger.FieldOpID, opId,
			logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
			logger.FieldQueID, queID,
			logger.FieldErrorToken, schedErrToken)
		outcome = DownlinkQueueOutcome{QueID: queID, Deferred: true}
	}
	if outcome.Deferred {
		s.logger.InfoContext(ctx, LogSCACIDLDataQueueDeferred,
			logger.FieldOpID, opId,
			logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
			logger.FieldQueID, outcome.QueID)
		return outcome
	}
	s.logger.InfoContext(ctx, LogSCACIDLDataQueueProcessed,
		logger.FieldOpID, opId,
		logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
		logger.FieldQueID, outcome.QueID,
		logger.FieldBsEui, mioty.FormatEUI64(outcome.BsEui))
	return outcome
}
