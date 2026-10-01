package scaci

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// QueueDownlinkInternal provides gRPC/internal access to SCACI dlDataQue flow.
//
// This method creates a synthetic session for internal calls and delegates to
// processDLDataQueueCore - the same single-source logic used by socket handlers.
// This ensures consistent validation, persistence, operation recording, and
// scheduler coordination for both SCACI socket and gRPC paths.
//
// Parameters:
//   - ctx: Context with timeout (inherits from gRPC request)
//   - tenantID: Authenticated tenant ID from gRPC context
//   - orgID: Organization UUID for audit trail (may be nil)
//   - req: MIOTY DLDataQueue request from gRPC
//   - command: The ref and deadline of an MQTT command, stored with the downlink; zero for none
//
// Returns:
//   - *DLDataQueueResult: Result containing queId, bsEui, opId, status
//   - error: Non-nil if operation failed
func (s *Server) QueueDownlinkInternal(
	ctx context.Context,
	tenantID int64,
	orgID *uuid.UUID,
	req *mioty.DLDataQueue,
	command storage.DownlinkCommand,
) (*DLDataQueueResult, error) {
	// Create synthetic session for internal calls
	var orgUUID uuid.UUID
	if orgID != nil {
		orgUUID = *orgID
	}
	session := &Session{
		TenantID:       tenantID,
		OrganizationID: orgUUID,
		// ID is 0 - skip operation recording for internal paths (no persistent session)
		// conn is nil - marks this as internal call (no socket I/O)
	}

	// Allocate positive internal opId (matches AC semantics per §3.2)
	// For internal paths, we use time-based opId since there's no AC counter
	opId := s.clock.Now().UnixNano() // Positive, unique per call

	dlReq := &DLDataQueue{
		EpEui:        req.EpEui,
		UserData:     req.UserData,
		Prio:         req.Prio,
		CntDepend:    req.CntDepend,
		PacketCnt:    req.PacketCnt,
		Format:       req.Format,
		ResponseExp:  req.ResponseExp,
		ResponsePrio: req.ResponsePrio,
		DlWindReq:    req.DlWindReq,
		ExpOnly:      req.ExpOnly,
		DlRxStatQry:  req.DlRxStatQry,
	}

	// Internal callers carry no Application Center queue id: the service
	// center's own id identifies the downlink to them.
	result, errToken, posixCode := s.processDLDataQueueCore(ctx, session, opId, dlReq, nil, command)
	if errToken != "" {
		s.logger.WarnContext(ctx, LogSCACIDLDataQueueFailed,
			logger.FieldOpID, opId,
			logger.FieldTenantIDCamel, tenantID,
			logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
			logger.FieldErrorToken, errToken,
			logger.FieldPath, downlinkPathInternal)
		return nil, &DLDataQueueError{Token: errToken, POSIX: posixCode}
	}

	s.logger.InfoContext(ctx, LogSCACIDLDataQueueProcessed,
		logger.FieldOpID, opId,
		logger.FieldTenantIDCamel, tenantID,
		logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
		logger.FieldQueID, result.QueID,
		logger.FieldBsEui, mioty.FormatEUI64(result.BsEui),
		logger.FieldPath, downlinkPathInternal)

	return &DLDataQueueResult{
		QueID:  result.QueID,
		BsEui:  result.BsEui,
		OpID:   opId,
		Status: result.Status(),
	}, nil
}

// DLDataQueueCoreResult holds the results from processDLDataQueueCore
// This struct is returned by the core logic to allow callers (socket handler, gRPC path)
// to handle the results appropriately for their I/O model.
type DLDataQueueCoreResult struct {
	QueID    uint64 // Service center queue ID the downlink was persisted under
	BsEui    uint64 // Base station EUI that will transmit; zero while deferred
	Deferred bool   // Not handed to a base station yet; the row waits for the next downlink window
}

// Status reports the queue row state the caller may surface: queued once a
// base station holds the downlink, pending while delivery is deferred.
func (r *DLDataQueueCoreResult) Status() mioty.DLQueueStatus {
	if r.Deferred {
		return bssci.DLQueueStatusPending
	}
	return bssci.DLQueueStatusQueued
}

// processDLDataQueueCore is the single-source core logic for dlDataQue (SCACI §3.10).
// Called by handleDLDataQueue (socket path) and QueueDownlinkInternal (gRPC path).
//
// It validates the request and the endpoint, persists the downlink under a
// service center queue id beside acQueID, the Application Center's queue id
// (any 64-bit value, SCACI §3.10.1; nil for a request without one) and the
// ref and deadline of the MQTT command that queued it (zero for none), records
// the operation of a persistent session, and hands the row to the BSSCI
// scheduler. Once the row is persisted the downlink is accepted; when it
// cannot be handed to a base station now it stays pending for the
// endpoint's next downlink window and the result is Deferred.
//
// Returns:
//   - result: Contains queId, bsEui on success
//   - errToken: Error token (empty string on success)
//   - posixCode: POSIX error code (only valid if errToken is non-empty)
func (s *Server) processDLDataQueueCore(
	ctx context.Context,
	session *Session,
	opId int64,
	req *DLDataQueue,
	acQueID *uint64,
	command storage.DownlinkCommand,
) (result *DLDataQueueCoreResult, errToken string, posixCode int) {
	if session == nil {
		return nil, errNoActiveSession, POSIX_EINVAL
	}
	if errToken := s.validateDLDataQueue(ctx, req); errToken != "" {
		return nil, errToken, POSIX_EINVAL
	}
	if errToken, posixCode := s.requireBidirectionalEndpoint(ctx, session.TenantID, req.EpEui); errToken != "" {
		return nil, errToken, posixCode
	}
	enqueueOrg := s.enqueueOrganization(ctx, session)
	if enqueueOrg == uuid.Nil {
		return nil, errDownlinkOrgUnresolved, POSIX_EINVAL
	}
	dlMsg := queuedDownlink(req, session, enqueueOrg, acQueID, command)
	queID, errToken, posixCode := s.persistQueuedDownlink(ctx, dlMsg)
	if errToken != "" {
		return nil, errToken, posixCode
	}
	s.recordDLDataQueue(ctx, session, opId, req, dlMsg, queID)
	outcome := s.dispatchQueuedDownlink(ctx, session.TenantID, opId, req, queID, enqueueOrg)
	return &DLDataQueueCoreResult{
		QueID:    outcome.QueID,
		BsEui:    outcome.BsEui,
		Deferred: outcome.Deferred,
	}, "", 0
}
