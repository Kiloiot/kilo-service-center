// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/encoding"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/vmihailenco/msgpack/v5"
)

// Operation metadata keys for scaci_operation_log ResponseData/RequestData
// Most keys are now exported from constants.go for cross-module access (gRPC handlers).
// Only module-internal keys remain here.
const (
	metadataKeyCompletedAt       = "completedAt"
	metadataKeyProtocolViolation = "protocolViolation"
)

// handleRegister processes Register messages per SCACI §3.6.1-3.6.3
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Field validation (epEui, nwkKey length)
//   - Endpoint existence check
//   - Create or update endpoint
//   - MIOTY field persistence
func (s *Server) handleRegister(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Step 1: Decode register payload (transport layer)
	var req Register
	if err := msgpack.Unmarshal(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDecodeRegisterFailed,
			"opId", opId,
			"error", err)
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errInvalidRegisterPayload)
	}

	// Step 2: Record operation as pending (for resume safety) via operationRecorder
	if session.ID > 0 && s.operationRecorder != nil {
		opCtx, opCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer opCancel()

		// Record all §3.6.1 register fields except nwkKey (security exclusion)
		requestData := map[string]interface{}{
			"epEui":       req.EpEui, // Numeric uint64 for propagation lookup (lines 165-173 parse numeric types)
			"bidi":        req.Bidi,
			"preAttach":   req.PreAttach,
			"shAddr":      req.ShAddr,
			"attachCnt":   req.AttachCnt,
			"packetCnt":   req.PacketCnt,
			"dualChan":    req.DualChan,
			"repetition":  req.Repetition,
			"wideCarrOff": req.WideCarrOff,
			"longBlkDist": req.LongBlkDist,
			// nwkKey explicitly excluded for security
		}
		if err := s.operationRecorder.Record(opCtx, session, opId, CmdRegister, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordRegisterOpFailed, "error", err)
			// Continue - operation tracking is for resume, not critical path
		}
	}

	// Step 3: Delegate to EndpointService for business logic
	ctx := s.sessionContext(session)
	errToken := s.endpointSvc.Register(ctx, &req, session.TenantID)
	if errToken != "" {
		// Service returned error token - resolve and send
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errToken)
	}

	// Step 4: Update session activity
	session.UpdateLastSeen()

	// Step 5: Send RegisterResponse (transport layer)
	resp := RegisterResponse{
		BaseMessage: BaseMessage{
			Command: CmdRegisterResponse,
			OpId:    opId,
		},
	}

	// Step 6: Update operation state to acknowledged
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId, models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, "error", err)
		}
	}

	return s.SendRegisterResponse(conn, session, &resp)
}

// handleRegisterComplete processes RegisterComplete messages per SCACI §3.6.3
//
// Completes the three-way handshake for register operation and marks operation as completed
// for resume safety.
func (s *Server) handleRegisterComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIRegisterHandshakeComplete,
		"opId", opId,
		"tenantId", session.TenantID)

	session.UpdateLastSeen()

	// Update operation state to completed with context timeout
	var op *models.SCACIOperation
	if session.ID > 0 && s.operationRepo != nil {
		ctx, cancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cancel()

		// Load operation to verify it exists and get request data
		var err error
		op, err = s.operationRepo.GetOperationByOpID(ctx, session.ID, opId)
		if err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACILoadRegisterOpFailed, "opId", opId, "error", err)
			return nil // Don't fail handshake on missing operation
		}

		// Update to completed
		if err := s.operationRepo.UpdateOperationState(ctx, session.ID, opId, models.OperationStateCompleted, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, "error", err)
		}
	}

	// BSSCI §5.8.2-5.8.3: Trigger attach propagation if endpoint was registered with preAttach
	// Uses new propagationSvc + sessionSnapshotProvider pattern (avoids circular dependency)
	if op != nil && op.RequestData != nil {
		// Extract epEui from operation request data
		var epEui uint64
		if epEuiFloat, ok := op.RequestData["epEui"].(float64); ok {
			epEui = uint64(epEuiFloat)
		} else if epEuiInt, ok := op.RequestData["epEui"].(int64); ok {
			if epEuiInt >= 0 {
				epEui = uint64(epEuiInt)
			}
		} else if epEuiUint, ok := op.RequestData["epEui"].(uint64); ok {
			epEui = epEuiUint
		} else if epEuiStr, ok := op.RequestData["epEui"].(string); ok {
			// Legacy: Parse hex string for pending ops created before numeric storage fix
			if parsed, parseErr := strconv.ParseUint(epEuiStr, 16, 64); parseErr == nil {
				epEui = parsed
			}
		}

		if epEui != 0 {
			// CORRECTED: Convert EUI to bytes using BigEndian.PutUint64 (not AppendUint64)
			euiBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(euiBytes, epEui)

			// Fetch endpoint to check preAttach flag - try session tenant first, then cross-tenant fallback
			endpoint, errToken := s.endpointSvc.GetByEUI(s.sessionContext(session), session.TenantID, euiBytes)

			// Fallback: endpoint roamed from different tenant (BSSCI pattern at server.go:5197-5202)
			if errToken == ErrEndpointNotFound {
				endpoint, errToken = s.endpointSvc.GetGlobal(s.sessionContext(session), euiBytes)
			}

			if errToken == "" && endpoint != nil && endpoint.PreAttach {
				// BSSCI §3.8: attach propagate is "required for unidirectional End Points"
				// The bidi filtering belongs in the BSSCI sender (for base station capability), not here
				// Build owner context from endpoint's tenant (not SCACI session tenant)
				ownerCtx := pkgcontext.WithTenantID(context.Background(), endpoint.TenantID)

				// Resolve organization (SCACI now has orgResolver - full parity with BSSCI)
				var orgUUID uuid.UUID
				if s.orgResolver != nil {
					resolvedOrg, err := s.orgResolver.GetDefaultOrgForTenant(ownerCtx, endpoint.TenantID)
					if err == nil && resolvedOrg != uuid.Nil {
						orgUUID = resolvedOrg
						ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, orgUUID)
					}
				}

				s.logger.InfoContext(ownerCtx, LogSCACITriggeringAttachPropagation,
					"epEui", pkgmioty.FormatEUI64(epEui),
					"endpointId", endpoint.ID,
					"tenantId", session.TenantID)

				if s.propagationSvc != nil && s.sessionSnapshotProvider != nil {
					// Extract tenant/org before goroutine to avoid context cancellation
					tenantID := endpoint.TenantID
					orgID := orgUUID

					// Async propagation with owner context: Uses endpoint.TenantID (owner) not session.TenantID.
					// Fresh context avoids cancellation and ensures correct tenant attribution for roaming.
					// Intentional context.Background() for fire-and-forget async operation.
					go func(ep *models.EndPoint, tenID int64, org uuid.UUID) {
						// Create detached context with preserved metadata
						bgCtx := pkgcontext.WithTenantID(context.Background(), tenID)
						if org != uuid.Nil {
							bgCtx = pkgcontext.WithOrganizationID(bgCtx, org)
						}

						activeSessions := s.sessionSnapshotProvider.ConnectedSessionsSnapshot()
						if err := s.propagationSvc.TriggerEndpointPropagate(bgCtx, ep.ID, activeSessions); err != nil {
							s.logger.ErrorContext(bgCtx, LogSCACIAttachPropagationErrors,
								"endpoint_id", ep.ID,
								"error", err)
						}
					}(endpoint, tenantID, orgID)
				}
			}
		}
	}

	return nil // No response per spec
}

// handleDeregister processes Deregister messages per SCACI §3.7.1
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Field validation (epEui)
//   - Endpoint existence check (tenant-scoped)
//   - Mark endpoint inactive (DetachEndpoint)
//
// NOTE: BSSCI detach propagation happens in handleDeregisterComplete, not here.
// This maintains spec-compliant three-way handshake (wait for AC confirmation).
func (s *Server) handleDeregister(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Step 1: Decode deregister payload (transport layer)
	var req Deregister
	if err := msgpack.Unmarshal(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDecodeDeregisterFailed,
			"opId", opId,
			"error", err)
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errInvalidDeregisterPayload)
	}

	// Step 2: Record operation as pending (for resume safety) via operationRecorder
	if session.ID > 0 && s.operationRecorder != nil {
		opCtx, opCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer opCancel()

		requestData := map[string]interface{}{"epEui": pkgmioty.FormatEUI64(req.EpEui)}
		if err := s.operationRecorder.Record(opCtx, session, opId, CmdDeregister, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordDeregisterOpFailed, "error", err)
			// Continue - operation tracking is for resume, not critical path
		}
	}

	// Step 3: Delegate to EndpointService for business logic
	ctx := s.sessionContext(session)
	errToken := s.endpointSvc.Deregister(ctx, req.EpEui, session.TenantID)
	if errToken != "" {
		// Map error tokens to appropriate POSIX codes
		posixCode := POSIX_EINVAL
		switch errToken {
		case errEndpointNotFound:
			posixCode = POSIX_ENOENT
		case errDatabaseError, errFailedUpdateEndpoint:
			posixCode = POSIX_EIO
		}
		return s.sendErrorWithCatalog(conn, session, opId, posixCode, errToken)
	}

	// Step 4: Cache epEui for handleDeregisterComplete cleanup per SCACI §3.7.3
	// This ensures cleanup runs even if DB lookup fails during complete phase.
	// No mutex needed: single-goroutine-per-connection model.
	session.EnsurePendingDeregisterOps()
	session.PendingDeregisterOps[opId] = req.EpEui

	// Step 5: Update session activity
	session.UpdateLastSeen()

	// Step 6: Build DeregisterResponse (transport layer)
	resp := DeregisterResponse{
		BaseMessage: BaseMessage{
			Command: CmdDeregisterResponse,
			OpId:    opId,
		},
	}

	// Step 7: Send response FIRST (must succeed before marking acknowledged)
	if err := s.SendDeregisterResponse(conn, session, &resp); err != nil {
		// On send failure, mark operation failed with metadata
		if session.ID > 0 && s.operationRepo != nil {
			failCtx, failCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
			defer failCancel()
			failMeta := map[string]interface{}{
				MetadataKeyErrorToken:  errSendDeregisterResponseFailed,
				MetadataKeyErrorDetail: err.Error(),
			}
			_ = s.operationRepo.UpdateOperationState(failCtx, session.ID, opId, models.OperationStateFailed, failMeta)
		}
		return err
	}

	// Step 8: Mark acknowledged AFTER successful send
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId, models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, "error", err)
		}
	}

	return nil
}

// handleDeregisterComplete processes DeregisterComplete messages per SCACI §3.7.3
//
// Completes the three-way handshake by marking operation completed, revoking pending downlinks,
// and triggering BSSCI detach propagation to all connected base stations.
//
// Cleanup flow:
//  1. Try in-memory cache (PendingDeregisterOps) first - always available if handleDeregister ran
//  2. Fall back to DB lookup if cache misses (resumed session or edge case)
//  3. Always execute cleanup regardless of operation log success
//  4. Delete cache entry after cleanup to prevent memory growth
func (s *Server) handleDeregisterComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIDeregisterHandshakeComplete, "opId", opId)

	session.UpdateLastSeen()

	ctx, cancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Step 1: Try in-memory cache first (no DB dependency, single-goroutine safe)
	var epEui uint64
	var euiSource string
	session.EnsurePendingDeregisterOps()
	if cached, ok := session.PendingDeregisterOps[opId]; ok {
		epEui = cached
		euiSource = "cache"
		// Delete from cache after retrieval to prevent memory growth
		delete(session.PendingDeregisterOps, opId)
	}

	// Step 2: Fall back to DB lookup if cache missed (resumed session or edge case)
	if epEui == 0 && session.ID > 0 && s.operationRepo != nil {
		op, err := s.operationRepo.GetOperationByOpID(ctx, session.ID, opId)
		if err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACILoadDeregisterOpFailed,
				"opId", opId,
				"error", err,
				"euiSource", "db_fallback_failed")
			// Continue - cleanup will be skipped but handshake shouldn't fail
		} else {
			// Extract epEui from RequestData with JSON-safe numeric handling
			// JSON unmarshaling converts uint64 to float64, so we must handle multiple types
			if val, exists := op.RequestData["epEui"]; exists {
				switch v := val.(type) {
				case uint64:
					epEui = v
				case float64:
					epEui = uint64(v) //nolint:gosec // Safe: epEui is always positive from DB
				case int64:
					epEui = uint64(v) //nolint:gosec // Safe: epEui is always positive from DB
				case string:
					// Parse hex string (EUIs stored as hex like "70B3D59CD000089B")
					if parsed, parseErr := strconv.ParseUint(v, 16, 64); parseErr == nil {
						epEui = parsed
					}
				}
			}
			if epEui != 0 {
				euiSource = "db_fallback"
			}
		}
	}

	// Step 3: Handle missing epEui as operation failure (Gap 1 fix per R4 plan)
	// Missing epEui is a protocol/state error - mark operation failed and return early.
	if epEui == 0 {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIDeregisterCleanupSkipped,
			"opId", opId,
			"reason", "no_epEui_available")

		// Mark operation failed with error metadata
		if session.ID > 0 && s.operationRepo != nil {
			failMeta := map[string]interface{}{
				MetadataKeyErrorToken:  ErrMissingEpEui,
				MetadataKeyErrorDetail: "epEui not found in cache or operation log",
			}
			if err := s.operationRepo.UpdateOperationState(ctx, session.ID, opId, models.OperationStateFailed, failMeta); err != nil {
				s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, "error", err)
			}
		}
		return nil // Don't propagate error to wire, just fail the operation
	}

	// Step 4: Execute cleanup and track results
	var revokedCount int
	var revokeErr error
	var detachErrorCount int

	var eui [8]byte
	binary.BigEndian.PutUint64(eui[:], epEui)

	s.logger.DebugContext(s.sessionContext(session), LogSCACIDeregisterCleanupStart,
		"opId", opId,
		"epEui", pkgmioty.FormatEUI64(epEui),
		"euiSource", euiSource)

	// Revoke downlinks and capture count
	revokedCount, revokeErr = s.revokeEndpointDownlinks(ctx, session.TenantID, eui)
	if revokeErr != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIRevokeDownlinksFailed, "epEui", epEui, "error", revokeErr)
	}

	// Trigger BSSCI detach propagation to base stations
	if errs := s.endpointSvc.PropagateDetachToAll(epEui); len(errs) > 0 {
		detachErrorCount = len(errs)
		s.logger.WarnContext(s.sessionContext(session), LogSCACIDetachPropagationErrors, "count", detachErrorCount)
	} else {
		s.logger.DebugContext(s.sessionContext(session), LogSCACIDetachPropagationSent, "epEui", epEui)
	}

	// Step 5: Update operation state with cleanup metadata (Gap 2 fix per R4 plan)
	// Determine cleanup status and final operation state.
	// - completed: handshake + cleanup both succeeded
	// - completed_with_warnings: handshake succeeded, cleanup had issues (revoke/detach errors)
	// - failed: reserved for protocol/state errors (handled above in Step 3)
	if session.ID > 0 && s.operationRepo != nil {
		var cleanupStatus string
		var finalState models.OperationState

		if revokeErr != nil || detachErrorCount > 0 {
			// Partial or full cleanup failure - handshake succeeded but cleanup had issues
			cleanupStatus = "partial_failure"
			finalState = models.OperationStateCompletedWithWarnings
		} else {
			cleanupStatus = "success"
			finalState = models.OperationStateCompleted
		}

		responseData := map[string]interface{}{
			MetadataKeyEpEui:            pkgmioty.FormatEUI64(epEui),
			MetadataKeyCleanupSource:    euiSource,
			MetadataKeyRevokedCount:     revokedCount,
			MetadataKeyDetachErrorCount: detachErrorCount,
			MetadataKeyCleanupStatus:    cleanupStatus,
		}

		// Include error details if revoke failed
		if revokeErr != nil {
			responseData[MetadataKeyErrorToken] = errRevokeDownlinksFailed
			responseData[MetadataKeyErrorDetail] = revokeErr.Error()
		}

		if err := s.operationRepo.UpdateOperationState(ctx, session.ID, opId, finalState, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, "error", err)
		}
	}

	return nil // No response per spec
}

// handleULDataResponse processes ULDataResponse messages per SCACI §3.8.2
//
// AC acknowledges receipt of UL data. SC must then send ULDataComplete to finish handshake.
func (s *Server) handleULDataResponse(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIReceivedULDataResponse,
		"opId", opId,
		"acEui", pkgmioty.FormatEUI64(session.AcEui))

	session.UpdateLastSeen()

	// Mark operation acknowledged (even if ulDataCmp send later fails)
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateULDataOpAckFailed,
				"opId", opId,
				"error", err)
		}
	}

	// Send ULDataComplete per SCACI §3.8.3 (same opId, no new counter)
	session.WriteMu.Lock()
	cmpMsg := ULDataComplete{
		BaseMessage: BaseMessage{
			Command: CmdULDataComplete,
			OpId:    opId,
		},
	}
	err := s.SendULDataComplete(conn, session, &cmpMsg)
	session.WriteMu.Unlock()

	if err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACISendULDataCompleteFailed,
			"opId", opId,
			"error", err)

		// Mark operation failed due to send error
		if session.ID > 0 && s.operationRepo != nil {
			failCtx, failCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
			defer failCancel()

			_ = s.operationRepo.UpdateOperationState(failCtx, session.ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					"errorToken":  errFailedRecordOperation,
					"errorDetail": fmt.Sprintf("Failed to send ulDataCmp: %v", err),
				})
		}

		return err
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIULDataHandshakeComplete,
		"opId", opId)

	// Mark operation completed
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkULDataOpCompleteFailed,
				"opId", opId,
				"error", err)
		}
	}

	return nil
}

// handleULDataTransmit processes UL Data Transmit messages per SCACI §3.9.1
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Validate tenant ownership of specific BS (if requested)
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Scheduler availability guard
//   - Delegation to BSSCI scheduler
//   - Error mapping (scheduler errors → SCACI tokens)
func (s *Server) handleULDataTransmit(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Step 1: Decode UL data transmit payload (transport layer)
	var req ULDataTransmit
	if err := msgpack.Unmarshal(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDecodeULDataTxFailed,
			"opId", opId,
			"error", err)
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errInvalidULDataTxPayload)
	}

	// Step 1b: §2.4 - Validate mandatory fields before any processing or operation recording
	if errToken := ValidateULDataTransmit(&req); errToken != "" {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIULDataTxValidationFailed,
			"opId", opId,
			"errorToken", errToken)
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errToken)
	}

	// Validate tenant ownership if specific BS requested
	// Note: Preference lookup for nil bsEui is handled in ULService.ScheduleULTransmit (§3.9.1)
	if req.BsEui != nil {
		bsCtx, bsCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer bsCancel()

		bsEuiBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bsEuiBytes, *req.BsEui)
		_, err := s.statusSvc.GetBaseStation(bsCtx, session.TenantID, bsEuiBytes)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				s.logger.WarnContext(s.sessionContext(session), LogSCACIBaseStationNotFoundULTx,
					"bsEui", *req.BsEui,
					"tenantId", session.TenantID)
				return s.sendErrorWithCatalog(conn, session, opId, POSIX_ENOENT, errBaseStationNotFound)
			}
			s.logger.ErrorContext(s.sessionContext(session), LogSCACILookupBaseStationFailed,
				"bsEui", *req.BsEui,
				"error", err)
			return s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errFailedVerifyBS)
		}
	}
	// If req.BsEui is nil, ULService will handle preference lookup and fallback selection

	// §2.4/§3.9.1: Normalize optional format field to default value 0
	format := uint8(0)
	if req.Format != nil {
		format = *req.Format
	}

	// Step 3: Record operation for resume safety (BEFORE scheduling)
	if session.ID > 0 && s.operationRepo != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		// Build request data (NO raw keys)
		requestData := map[string]interface{}{
			"epEui":     pkgmioty.FormatEUI64(req.EpEui),
			"shAddr":    req.ShAddr,
			"packetCnt": req.PacketCnt,
			"format":    format, // Always record normalized format (never nil)
		}
		if req.BsEui != nil {
			requestData["bsEui"] = pkgmioty.FormatEUI64(*req.BsEui)
		}
		if req.Profile != nil {
			requestData["profile"] = *req.Profile
		}
		if req.UserData != nil {
			requestData["userData"] = encoding.EncodeUserData(req.UserData)
		}

		if err := s.operationRecorder.Record(recCtx, session, opId, CmdULDataTransmit, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIRecordULTxOpFailed,
				"opId", opId,
				"error", err)
			return s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errFailedRecordOperation)
		}
	}

	// Step 4: Delegate to ULService for business logic
	// Convert ULDataTransmit to mioty.ULDataTransmit for service call
	miotyReq := &mioty.ULDataTransmit{
		EpEui:     req.EpEui,
		ShAddr:    req.ShAddr,
		PacketCnt: req.PacketCnt,
		NwkSnKey:  req.NwkSnKey,
		UserData:  req.UserData,
		BsEui:     req.BsEui,
		Profile:   req.Profile,
		Format:    &format, // Always non-nil defaulted pointer - prevents downstream nil derefs
	}

	ctx := s.sessionContext(session)
	bssciOpID, actualBsEui, errToken := s.ulSvc.ScheduleULTransmit(ctx, miotyReq, session.TenantID)
	if errToken != "" {
		// Service returned error token - map to POSIX code and send error
		posixCode := POSIX_EINVAL
		switch errToken {
		case errULTransmitNotSupported:
			posixCode = POSIX_ENOTSUP
		case errBaseStationUnavailable:
			posixCode = POSIX_EAGAIN
		case errBaseStationNotFound:
			posixCode = POSIX_ENOENT
		case errFailedRecordOperation:
			posixCode = POSIX_EIO
		}

		// Mark operation failed
		if session.ID > 0 && s.operationRepo != nil {
			errCtx, errCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
			defer errCancel()
			_ = s.operationRepo.UpdateOperationState(errCtx, session.ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					"errorToken":  errToken,
					"errorDetail": "UL transmit scheduling failed",
				})
		}

		return s.sendErrorWithCatalog(conn, session, opId, posixCode, errToken)
	}

	// Step 5: Send response (transport layer)
	resp := ULDataTransmitResponse{
		BaseMessage: mioty.BaseMessage{
			CommandType: CmdULDataTransmitResponse,
			OpId:        opId,
		},
	}

	if err := s.SendULDataTransmitResponse(conn, session, &resp); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACISendULDataTxRspFailed,
			"opId", opId,
			"error", err)
		return err
	}

	// Step 6: Mark operation acknowledged
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		responseData := map[string]interface{}{
			"bssciOpID": bssciOpID,
			"bsEui":     pkgmioty.FormatEUI64(actualBsEui),
		}

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkULTxAckFailed,
				"opId", opId,
				"error", err)
		}
	}

	// Step 7: Update session activity and log success
	session.UpdateLastSeen()

	s.logger.InfoContext(s.sessionContext(session), LogSCACIULDataTxScheduled,
		"opId", opId,
		"bssciOpID", bssciOpID,
		"epEui", req.EpEui,
		"bsEui", actualBsEui)

	return nil
}

// handleULDataTransmitComplete processes UL Data Transmit Complete messages per SCACI §3.9.3
//
// AC completes the three-way handshake after receiving ulDataTxRsp.
// No response per spec.
func (s *Server) handleULDataTransmitComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIProcessingULDataTxCmp,
		"opId", opId,
		"acEui", pkgmioty.FormatEUI64(session.AcEui))

	// Update session activity
	session.UpdateLastSeen()

	// Mark operation completed
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completionData := map[string]interface{}{
			"completedAt": time.Now().UTC().Format(time.RFC3339Nano),
		}

		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, completionData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkULTxOpCompleteFailed,
				"opId", opId,
				"error", err)
		}
	}

	// No response per SCACI §3.9.3
	return nil
}

// DLDataQueueCoreResult holds the results from processDLDataQueueCore
// This struct is returned by the core logic to allow callers (socket handler, gRPC path)
// to handle the results appropriately for their I/O model.
type DLDataQueueCoreResult struct {
	QueID    uint64 // Queue ID from persistence layer
	BsEui    uint64 // Base station EUI that will transmit
	StoredID int64  // Database ID of the persisted downlink record
}

// processDLDataQueueCore is the single-source core logic for dlDataQue (SCACI §3.10).
// Called by handleDLDataQueue (socket path) and QueueDownlinkInternal (gRPC path).
//
// This function performs:
//   - Endpoint existence validation
//   - Duplicate queue ID check
//   - Field extraction with defaults
//   - Storage persistence with organization context
//   - Operation recording (if session.ID > 0)
//   - BSSCI scheduler coordination
//   - Status transitions (pending → queued or failed)
//
// Parameters:
//   - ctx: Context with timeout
//   - session: Session containing tenantID, orgID, and ID for operation recording
//   - opId: Operation ID for this request (from AC or generated for internal)
//   - req: Parsed DLDataQueue request (already unmarshaled)
//
// Returns:
//   - result: Contains queId, bsEui, storedID on success
//   - errToken: Error token (empty string on success)
//   - posixCode: POSIX error code (only valid if errToken is non-empty)
func (s *Server) processDLDataQueueCore(
	ctx context.Context,
	session *Session,
	opId int64,
	req *DLDataQueue,
) (result *DLDataQueueCoreResult, errToken string, posixCode int) {
	if session == nil {
		return nil, errNoActiveSession, POSIX_EINVAL
	}

	// Validate mandatory fields per SCACI §3.10.1
	if req.EpEui == 0 {
		return nil, errEpEuiZero, POSIX_EINVAL
	}
	if req.QueId == 0 {
		return nil, errQueIDZero, POSIX_EINVAL
	}

	// Validate counter-dependent consistency
	if req.CntDepend && len(req.PacketCnt) != len(req.UserData) {
		s.logger.WarnContext(ctx, LogSCACICntDependLengthMismatch,
			"packetCntLen", len(req.PacketCnt),
			"userDataLen", len(req.UserData))
		return nil, errCntDependMismatch, POSIX_EINVAL
	}
	if !req.CntDepend && len(req.PacketCnt) > 0 {
		return nil, errCntDependPacketCntOmit, POSIX_EINVAL
	}
	// SCACI §3.10.1: "single user data entry if cntDepend is false"
	if !req.CntDepend && len(req.UserData) > 1 {
		s.logger.WarnContext(ctx, LogSCACINonCntDependMultiPayload,
			"userDataLen", len(req.UserData))
		return nil, errNonCntDependMultiPayload, POSIX_EINVAL
	}

	// Validate individual payload sizes (MIOTY radio protocol §4.3.2)
	for i, payload := range req.UserData {
		if len(payload) > mioty.MaxDLUserDataBytes {
			s.logger.WarnContext(ctx, "Downlink payload exceeds maximum size",
				"entry", i,
				"size", len(payload),
				"max", mioty.MaxDLUserDataBytes)
			return nil, errDLPayloadTooLarge, POSIX_EINVAL
		}
	}

	// Check if endpoint exists for this tenant (using service layer)
	eui := make([]byte, 8)
	binary.BigEndian.PutUint64(eui, req.EpEui)
	_, epErrToken := s.endpointSvc.GetByEUI(ctx, session.TenantID, eui)
	if epErrToken != "" {
		posixCode := POSIX_EIO
		if epErrToken == ErrEndpointNotFound {
			posixCode = POSIX_ENOENT
		}
		return nil, epErrToken, posixCode
	}

	// Guard against duplicate queue IDs via storage check
	tenantStr := strconv.FormatInt(session.TenantID, 10)
	existing, err := s.dlSvc.GetDownlinkByQueueID(ctx, req.QueId, tenantStr)
	if err == nil && existing != nil {
		s.logger.WarnContext(ctx, LogSCACIDuplicateQueIDDetected,
			"queId", req.QueId,
			"tenantId", session.TenantID)
		return nil, errQueIDExists, POSIX_EEXIST
	}

	// Extract optional fields with defaults per SCACI §3.10.1
	prio := float32(0)
	if req.Prio != nil {
		prio = *req.Prio
	}
	format := uint8(0)
	if req.Format != nil {
		format = *req.Format
	}
	responseExp := false
	if req.ResponseExp != nil {
		responseExp = *req.ResponseExp
	}
	responsePrio := false
	if req.ResponsePrio != nil {
		responsePrio = *req.ResponsePrio
	}
	dlWindReq := false
	if req.DlWindReq != nil {
		dlWindReq = *req.DlWindReq
	}
	expOnly := false
	if req.ExpOnly != nil {
		expOnly = *req.ExpOnly
	}
	dlRxStatQry := false
	if req.DlRxStatQry != nil {
		dlRxStatQry = *req.DlRxStatQry
	}

	// Convert PacketCnt from []uint32 to []int64 for storage compatibility
	var packetCntArray []int64
	if req.CntDepend && len(req.PacketCnt) > 0 {
		packetCntArray = make([]int64, len(req.PacketCnt))
		for i, cnt := range req.PacketCnt {
			packetCntArray[i] = int64(cnt)
		}
	}

	// Payload: first entry from UserData (matches gRPC path)
	var dlPayload []byte
	if len(req.UserData) > 0 {
		dlPayload = req.UserData[0]
	}

	// UserData: full slice if CntDepend, nil otherwise
	var userData [][]byte
	if req.CntDepend {
		userData = req.UserData
	}

	// Validate queue ID fits in int64 (BSSCI §5.12)
	if req.QueId > math.MaxInt64 {
		s.logger.WarnContext(ctx, LogSCACIQueueIDOutOfRange,
			"queId", req.QueId,
			"maxAllowed", math.MaxInt64)
		return nil, errQueueIDOutOfRange, POSIX_ERANGE
	}

	// Extract organization ID from session (nil pointer if not set)
	var orgID *uuid.UUID
	if session.OrganizationID != uuid.Nil {
		orgID = &session.OrganizationID
	}

	// Build storage.DownlinkMessage with organization context
	dlMsg := &storage.DownlinkMessage{
		EPEUI:          fmt.Sprintf("%016X", req.EpEui),
		TenantID:       tenantStr,
		OrganizationID: orgID,
		Payload:        dlPayload,
		Priority:       prio,
		Status:         bssci.DLQueueStatusPending,
		Attempts:       0,
		MaxAttempts:    3,
		QueID:          int64(req.QueId), //nolint:gosec // Safe: QueId validated above
		CntDepend:      req.CntDepend,
		PacketCntArray: packetCntArray,
		Format:         format,
		ResponseExp:    responseExp,
		ResponsePrio:   responsePrio,
		DlWindReq:      dlWindReq,
		ExpOnly:        expOnly,
		DlRxStatQry:    dlRxStatQry,
		UserData:       userData,
	}

	// Persist to downlink queue with timeout context
	persistCtx, persistCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer persistCancel()

	stored, err := s.dlSvc.EnqueueDownlink(persistCtx, dlMsg)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidInput) {
			s.logger.WarnContext(ctx, LogSCACIInvalidDownlinkPayload,
				"queId", req.QueId,
				"error", err)
			return nil, errInvalidDLDataQuePayload, POSIX_EINVAL
		}
		s.logger.ErrorContext(ctx, LogSCACIEnqueueDownlinkFailed, "error", err)
		return nil, errFailedPersistDownlink, POSIX_EIO
	}

	// Record operation before scheduler call (guard session.ID + nil check)
	if session.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer recCancel()

		// Serialize UserData for operation logging
		userDataSerialized, _ := json.Marshal(req.UserData)
		requestData := map[string]interface{}{
			"epEui":        fmt.Sprintf("%016X", req.EpEui),
			"queId":        stored.QueID,
			"userData":     string(userDataSerialized),
			"cntDepend":    req.CntDepend,
			"packetCnt":    packetCntArray,
			"prio":         prio,
			"format":       format,
			"responseExp":  responseExp,
			"responsePrio": responsePrio,
			"dlWindReq":    dlWindReq,
			"expOnly":      expOnly,
			"dlRxStatQry":  dlRxStatQry,
			"tenantId":     session.TenantID,
			"orgId":        session.OrganizationID.String(),
		}
		if err := s.operationRecorder.Record(recCtx, session, opId, CmdDLDataQueue, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(ctx, LogSCACIRecordOperationFailed, "error", err)
		}
	}

	// Call DL service to coordinate with BSSCI scheduler
	queuedQueId, bsEui, schedErrToken := s.dlSvc.QueueDownlink(ctx, req, session.TenantID)
	if schedErrToken != "" {
		// Map error token to POSIX code
		var schedPosixCode int
		switch schedErrToken {
		case errSchedulerUnavailable:
			schedPosixCode = POSIX_ENOTSUP
		case errBaseStationUnavailable:
			schedPosixCode = POSIX_EAGAIN
		case errFailedRecordOperation:
			schedPosixCode = POSIX_EIO
		default:
			schedPosixCode = POSIX_EINVAL
		}

		// Revert downlink status - scheduler failed
		// Use orgID (nil-safe pointer) instead of &session.OrganizationID which would filter on uuid.Nil when org not set
		statusCtx, statusCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer statusCancel()
		if err := s.dlSvc.UpdateDownlinkStatus(statusCtx, strconv.FormatInt(stored.QueID, 10), bssci.DLQueueStatusFailed, orgID); err != nil {
			s.logger.WarnContext(ctx, LogSCACIUpdateDownlinkStatusFailed, "error", err)
		}

		// Mark operation as failed
		if session.ID > 0 && s.operationRepo != nil {
			failCtx, failCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
			defer failCancel()
			_ = s.operationRepo.UpdateOperationState(failCtx, session.ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					"errorToken":  schedErrToken,
					"errorDetail": "Scheduler failed",
				})
		}

		return nil, schedErrToken, schedPosixCode
	}

	// Update status to queued after successful BSSCI coordination
	// Use orgID (nil-safe pointer) instead of &session.OrganizationID which would filter on uuid.Nil when org not set
	statusCtx, statusCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer statusCancel()
	if err := s.dlSvc.UpdateDownlinkStatus(statusCtx, strconv.FormatInt(stored.QueID, 10), bssci.DLQueueStatusQueued, orgID); err != nil {
		s.logger.WarnContext(ctx, LogSCACIUpdateDownlinkStatusQueued, "error", err)
		// Non-critical error, continue
	}

	s.logger.InfoContext(ctx, LogSCACIDLDataQueueProcessed,
		"opId", opId,
		"epEui", pkgmioty.FormatEUI64(req.EpEui),
		"queId", queuedQueId,
		"bsEui", pkgmioty.FormatEUI64(bsEui))

	return &DLDataQueueCoreResult{
		QueID:    queuedQueId,
		BsEui:    bsEui,
		StoredID: stored.ID,
	}, "", 0
}

// handleDLDataQueue processes DLDataQueue messages per SCACI §3.10.1
//
// AC queues downlink data for SC to transmit to endpoint.
// This handler delegates to processDLDataQueueCore for single-source business logic,
// then handles socket-specific I/O (response send, operation state updates).
func (s *Server) handleDLDataQueue(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Unmarshal request (transport layer responsibility)
	var req DLDataQueue
	if err := msgpack.Unmarshal(payload, &req); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIUnmarshalDLDataQueueFailed, "error", err)
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errMalformedPayload)
	}

	// Delegate to single-source core logic
	ctx := s.sessionContext(session)
	result, errToken, posixCode := s.processDLDataQueueCore(ctx, session, opId, &req)
	if errToken != "" {
		return s.sendErrorWithCatalog(conn, session, opId, posixCode, errToken)
	}

	// Success - send response (transport layer)
	resp := DLDataQueueResponse{
		BaseMessage: mioty.BaseMessage{
			CommandType: CmdDLDataQueueResponse,
			OpId:        opId,
		},
	}

	if err := s.SendDLDataQueueResponse(conn, session, &resp); err != nil {
		return err
	}

	// Mark operation as acknowledged with queId/bsEui metadata (socket path only)
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		responseData := map[string]interface{}{
			"queId":          fmt.Sprintf("%d", result.QueID),
			"bsEui":          pkgmioty.FormatEUI64(result.BsEui),
			"status":         bssci.DLQueueStatusQueued,
			"acknowledgedAt": time.Now().UTC().Format(time.RFC3339Nano),
		}
		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(ctx, LogSCACIUpdateOperationStateFailed, "error", err)
		}
	}

	session.UpdateLastSeen()

	return nil
}

// handleDLDataQueueComplete processes DLDataQueueComplete messages per SCACI §3.10.3
func (s *Server) handleDLDataQueueComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Mark operation as completed (guard session.ID)
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completedAt := time.Now().UTC().Format(time.RFC3339Nano)
		completionData := map[string]interface{}{
			"completedAt": completedAt,
			"status":      bssci.DLQueueStatusQueued,
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, completionData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, "error", err)
		}
	}

	session.UpdateLastSeen()

	s.logger.DebugContext(s.sessionContext(session), LogSCACIDLDataQueueHandshakeComplete,
		"opId", opId)

	// No response per SCACI §3.10.3
	return nil
}

// handleDLDataRevoke processes DLDataRevoke messages per SCACI §3.11 DL Data Revoke
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Validate mandatory fields (epEui, packetCnt)
//   - Resolve packetCnt → queId via database lookup
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Scheduler availability guard
//   - Delegation to BSSCI scheduler (RevokeDownlink)
//   - Error mapping (scheduler errors → SCACI tokens)
//
// Three-way handshake flow:
//  1. AC sends DLDataRevoke (this handler)
//  2. SC sends DLDataRevokeResponse
//  3. AC sends DLDataRevokeComplete (handled by handleDLDataRevokeComplete)
//
// POSIX error mapping per SCACI §3.9:
//   - POSIX_ENOENT (2): Queue entry not found
//   - POSIX_EAGAIN (11): Base station temporarily unavailable
//   - POSIX_ENOTSUP (95): Downlink scheduler not configured
func (s *Server) handleDLDataRevoke(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Step 1: Decode MessagePack payload (transport layer)
	var req DLDataRevoke
	if err := msgpack.Unmarshal(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIUnmarshalDLDataRevokeFailed, "error", err)
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errInvalidDLDataRevPayload)
	}

	s.logger.InfoContext(s.sessionContext(session), LogSCACIProcessingDLDataRevoke,
		"opId", opId,
		"packetCnt", req.PacketCnt,
		"epEui", pkgmioty.FormatEUI64(req.EpEui))

	// Step 2: Validate mandatory fields per SCACI §3.11.1
	if req.EpEui == 0 {
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errMissingEpEui)
	}
	if req.PacketCnt == 0 {
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errMissingPacketCnt)
	}

	// Step 3: Resolve packetCnt → queId via database lookup (transport-layer resolution)
	// SCACI uses packetCnt as identifier, but BSSCI uses queId
	tenantIDStr := strconv.FormatInt(session.TenantID, 10)
	epEuiHex := fmt.Sprintf("%016X", req.EpEui)

	lookupCtx, lookupCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer lookupCancel()

	downlink, err := s.dlSvc.GetDownlinkByPacketCnt(lookupCtx, tenantIDStr, epEuiHex, req.PacketCnt)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIDownlinkNotFoundForPacketCnt,
				"packetCnt", req.PacketCnt,
				"epEui", pkgmioty.FormatEUI64(req.EpEui),
				"tenantId", session.TenantID)

			// Mark operation as failed
			if session.ID > 0 && s.operationRepo != nil {
				failCtx, failCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
				defer failCancel()
				_ = s.operationRepo.UpdateOperationState(failCtx, session.ID, opId,
					models.OperationStateFailed, map[string]interface{}{
						"errorToken": errDownlinkNotFound,
						"packetCnt":  req.PacketCnt,
						"posix_code": POSIX_ENOENT,
					})
			}

			return s.sendErrorWithCatalog(conn, session, opId, POSIX_ENOENT, errDownlinkNotFound)
		}

		s.logger.ErrorContext(s.sessionContext(session), LogSCACILookupDownlinkByPacketCntFailed,
			"packetCnt", req.PacketCnt,
			"error", err)

		// Mark operation as failed
		if session.ID > 0 && s.operationRepo != nil {
			failCtx, failCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
			defer failCancel()
			_ = s.operationRepo.UpdateOperationState(failCtx, session.ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					"errorToken":  errDatabaseError,
					"errorDetail": err.Error(),
					"packetCnt":   req.PacketCnt,
					"posix_code":  POSIX_EIO,
				})
		}

		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errDatabaseError)
	}

	// Derived queue ID from database lookup
	// Guard against negative values (database constraint should prevent this)
	if downlink.QueID < 0 {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIInvalidQueueIDFromDB,
			"queId", downlink.QueID)
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errDatabaseError)
	}
	queId := uint64(downlink.QueID)

	// Step 4: Record operation before BSSCI coordination (for session resume)
	if session.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		requestData := map[string]interface{}{
			"packetCnt":  req.PacketCnt,
			"epEui":      pkgmioty.FormatEUI64(req.EpEui),
			"queId":      queId, // derived from DB
			"receivedAt": time.Now().UTC().Format(time.RFC3339Nano),
		}
		if err := s.operationRecorder.Record(recCtx, session, opId, CmdDLDataRevoke, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordDLDataRevokeOpFailed, "error", err)
		}
	}

	session.UpdateLastSeen()

	// Step 5: Delegate to DLService for business logic
	ctx := s.sessionContext(session)
	bsEui, errToken := s.dlSvc.RevokeDownlink(ctx, queId, session.TenantID)
	if errToken != "" {
		// Service returned error token - map to POSIX code and send error
		posixCode := POSIX_EINVAL
		switch errToken {
		case errDownlinkNotFound:
			posixCode = POSIX_ENOENT
		case errSchedulerUnavailable:
			posixCode = POSIX_ENOTSUP
		case errDatabaseError:
			posixCode = POSIX_EIO
		}

		// Mark operation as failed
		if session.ID > 0 && s.operationRepo != nil {
			failCtx, failCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
			defer failCancel()
			_ = s.operationRepo.UpdateOperationState(failCtx, session.ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					"errorToken":  errToken,
					"errorDetail": "DL revoke failed",
					"packetCnt":   req.PacketCnt,
					"queId":       queId,
					"posix_code":  posixCode,
				})
		}

		return s.sendErrorWithCatalog(conn, session, opId, posixCode, errToken)
	}

	// Step 6: Mark operation as acknowledged (BSSCI coordination initiated)
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		ackData := map[string]interface{}{
			"packetCnt":      req.PacketCnt,
			"queId":          queId,
			"bsEui":          pkgmioty.FormatEUI64(bsEui),
			"acknowledgedAt": time.Now().UTC().Format(time.RFC3339Nano),
		}
		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, ackData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, "error", err)
		}
	}

	// Step 7: Send response per SCACI §3.11.2 (transport layer)
	resp := DLDataRevokeResponse{
		BaseMessage: BaseMessage{
			Command: CmdDLDataRevokeResponse,
			OpId:    opId,
		},
	}

	s.logger.InfoContext(s.sessionContext(session), LogSCACIDLDataRevokeInitiated,
		"opId", opId,
		"packetCnt", req.PacketCnt,
		"queId", queId,
		"bsEui", pkgmioty.FormatEUI64(bsEui))

	return s.SendDLDataRevokeResponse(conn, session, &resp)
}

// handleDLDataRevokeComplete processes DLDataRevokeComplete messages per SCACI §3.11.3
//
// This handler completes the three-way handshake for DL Data Revoke. By the time this
// message arrives, the BSSCI layer has already sent dlDataRev to the base station and
// received dlDataRevRsp. The AC is now acknowledging receipt of our DLDataRevokeResponse.
//
// This handler:
//  1. Marks the operation as completed in the operation log
//  2. Updates session last-seen timestamp
//  3. No response is sent per SCACI §3.11.3
func (s *Server) handleDLDataRevokeComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Mark operation as completed (guard session.ID)
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completedAt := time.Now().UTC().Format(time.RFC3339Nano)
		completionData := map[string]interface{}{
			"completedAt": completedAt,
			"status":      ResultRevoked,
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, completionData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, "error", err)
		}
	}

	session.UpdateLastSeen()

	s.logger.DebugContext(s.sessionContext(session), LogSCACIDLDataRevokeHandshakeComplete,
		"opId", opId)

	// No response per SCACI §3.11.3
	return nil
}

// handleEPStatusResponse processes EPStatusResponse per SCACI §3.13.2
// This is AC's acknowledgement of SC's EPStatus message
func (s *Server) handleEPStatusResponse(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	ctx := s.sessionContext(session)
	s.logger.DebugContext(ctx, LogSCACIEPStatusResponseReceived, "opId", opId)

	session.UpdateLastSeen()

	// Mark operation as acknowledged (pending -> acknowledged)
	if s.operationRepo != nil && session.ID > 0 {
		opCtx, opCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer opCancel()

		responseData := map[string]interface{}{"status": "ack"}
		if err := s.operationRepo.UpdateOperationState(opCtx, session.ID, opId,
			models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(ctx, LogSCACIOperationStateUpdateFailed,
				"opId", opId, "state", models.OperationStateAcknowledged, "error", err)
		}
	}

	// Persist opId pair atomically per §3.2
	s.persistOpIDsPair(session.ID, session.TenantID, session.AcOpIdCounter, session.ScOpIdCounter)

	return nil // No SC response to Response message
}

// handleEPStatusComplete processes EPStatusComplete messages per SCACI §3.13.3
//
// EPStatus is SC-initiated (SC sends EPStatus to AC, AC responds with EPStatusResponse, AC completes with EPStatusComplete)
// This handler receives the Complete from AC and marks the operation as completed.
func (s *Server) handleEPStatusComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	ctx := s.sessionContext(session)
	s.logger.DebugContext(ctx, LogSCACIEPStatusHandshakeComplete, "opId", opId)

	session.UpdateLastSeen()

	// Mark operation as completed with metadata
	if s.operationRepo != nil && session.ID > 0 {
		opCtx, opCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer opCancel()

		responseData := map[string]interface{}{
			"status":      "completed",
			"completedAt": time.Now().UTC().Format(time.RFC3339),
		}
		if err := s.operationRepo.UpdateOperationState(opCtx, session.ID, opId,
			models.OperationStateCompleted, responseData); err != nil {
			s.logger.WarnContext(ctx, LogSCACIOperationStateUpdateFailed,
				"opId", opId, "state", models.OperationStateCompleted, "error", err)
		}
	}

	// Persist opId pair atomically per §3.2
	s.persistOpIDsPair(session.ID, session.TenantID, session.AcOpIdCounter, session.ScOpIdCounter)

	return nil // No response per SCACI §3.13.3
}

// revokeEndpointDownlinks revokes all pending/scheduled downlinks for an endpoint
//
// This helper is called during deregister complete to clean up the downlink queue.
// It queries all pending/scheduled downlinks for the tenant/endpoint and marks them
// as revoked using the existing RevokeDownlink method.
//
// Parameters:
//   - ctx: Context with timeout for database operations
//   - tenantID: Tenant identifier for isolation
//   - epEui: Endpoint EUI as [8]byte array
//
// Returns:
//   - int: Number of downlinks successfully revoked
//   - error: Non-nil if downlink revocation fails (partial count still returned)
func (s *Server) revokeEndpointDownlinks(ctx context.Context, tenantID int64, epEui [8]byte) (int, error) {
	// Defensive nil guard (unreachable under validated startup per constructor validation)
	// Allows graceful degradation if constructor validation is bypassed in tests
	if s.dlSvc == nil {
		s.logger.WarnContext(ctx, LogSCACIStorageNotAvailableRevoke,
			"tenantId", tenantID,
			"epEui", hex.EncodeToString(epEui[:]))
		return 0, nil // Graceful no-op, don't fail deregister handshake
	}

	// Convert endpoint EUI to hex string for GetDownlinkQueue
	deviceEUI := hex.EncodeToString(epEui[:])
	tenantIDStr := strconv.FormatInt(tenantID, 10)

	s.logger.DebugContext(ctx, LogSCACIRevokingDownlinksForEndpoint,
		"tenantId", tenantID,
		"epEui", deviceEUI)

	// Query all pending/scheduled downlinks for this endpoint
	downlinks, err := s.dlSvc.GetDownlinkQueue(ctx, deviceEUI, tenantIDStr)
	if err != nil {
		s.logger.ErrorContext(ctx, LogSCACIQueryDownlinkQueueFailed,
			"epEui", deviceEUI,
			"error", err)
		return 0, fmt.Errorf("failed to query downlink queue: %w", err)
	}

	if len(downlinks) == 0 {
		s.logger.DebugContext(ctx, LogSCACINoPendingDownlinksToRevoke, "epEui", deviceEUI)
		return 0, nil
	}

	// Revoke each downlink message via DLService
	var revokeErrors []error
	revokedCount := 0

	for _, dl := range downlinks {
		if err := s.dlSvc.RevokeDownlinkByID(ctx, dl.QueID, tenantIDStr); err != nil {
			s.logger.WarnContext(ctx, LogSCACIRevokeDownlinkItemFailed,
				"queId", dl.QueID,
				"epEui", deviceEUI,
				"error", err)
			revokeErrors = append(revokeErrors, err)
		} else {
			revokedCount++
		}
	}

	s.logger.InfoContext(ctx, LogSCACIDownlinkRevocationComplete,
		"epEui", deviceEUI,
		"total", len(downlinks),
		"revoked", revokedCount,
		"failed", len(revokeErrors))

	// Return count and aggregated error if any revocations failed
	if len(revokeErrors) > 0 {
		return revokedCount, fmt.Errorf("failed to revoke %d/%d downlinks", len(revokeErrors), len(downlinks))
	}

	return revokedCount, nil
}

// handleDLDataResultResponse processes DLDataResultResponse messages per SCACI §3.12.2
//
// AC acknowledges receipt of DL data result. SC must then send DLDataResultComplete to finish handshake.
func (s *Server) handleDLDataResultResponse(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIReceivedDLDataResultResponse,
		"opId", opId,
		"acEui", pkgmioty.FormatEUI64(session.AcEui))

	session.UpdateLastSeen()

	// Mark operation acknowledged
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateDLResultOpAckFailed,
				"opId", opId,
				"error", err)
		}
	}

	// Send DLDataResultComplete per SCACI §3.12.3 (same opId, no new counter)
	session.WriteMu.Lock()
	cmpMsg := DLDataResultComplete{
		BaseMessage: BaseMessage{
			Command: CmdDLDataResultComplete,
			OpId:    opId,
		},
	}
	err := s.SendDLDataResultComplete(conn, session, &cmpMsg)
	session.WriteMu.Unlock()

	if err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACISendDLResultCompleteFailed,
			"opId", opId,
			"error", err)

		// Mark operation failed with catalog-aligned error metadata
		if session.ID > 0 && s.operationRepo != nil {
			failCtx, failCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
			defer failCancel()

			responseData := map[string]interface{}{
				MetadataKeyErrorToken:  errFailedRecordOperation,
				MetadataKeyErrorDetail: err.Error(),
			}
			_ = s.operationRepo.UpdateOperationState(failCtx, session.ID, opId,
				models.OperationStateFailed, responseData)
		}
		return err
	}

	// Mark operation completed after successful send
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		responseData := map[string]interface{}{
			metadataKeyCompletedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateDLResultOpCompleteFailed,
				"opId", opId,
				"error", err)
		}
	}

	return nil
}

// handleDLDataResultComplete processes DLDataResultComplete messages per SCACI §3.12.3
//
// AC should NOT send this message (SC sends txDataResCmp to AC, not vice-versa).
// This handler logs the protocol violation but gracefully marks operation completed.
func (s *Server) handleDLDataResultComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	// Log protocol violation for compliance evidence
	s.logger.WarnContext(s.sessionContext(session), LogSCACIUnexpectedDLResultComplete,
		"opId", opId,
		"acEui", pkgmioty.FormatEUI64(session.AcEui),
		"errorToken", errProtocolViolationDLResCmp)

	session.UpdateLastSeen()

	// Mark operation completed gracefully despite violation
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		responseData := map[string]interface{}{
			metadataKeyCompletedAt:       time.Now().UTC().Format(time.RFC3339),
			metadataKeyProtocolViolation: "AC sent txDataResCmp (should only receive from SC)",
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateDLResultOpCompleteFailed,
				"opId", opId,
				"error", err)
		}
	}

	return nil // No response per spec
}

// handleInboundError processes error messages received from the Application Center (SCACI §3.14).
// AC sends error when it cannot process an SC-initiated operation.
// SC must acknowledge with errorAck to complete the error handshake.
func (s *Server) handleInboundError(conn net.Conn, session *Session, opId int64, payload []byte) error {
	ctx := s.sessionContext(session)

	// Decode the error message
	var errMsg Error
	if err := msgpack.Unmarshal(payload, &errMsg); err != nil {
		s.logger.ErrorContext(ctx, LogSCACIInvalidPayload,
			"command", CmdError,
			"opId", opId,
			"error", err)
		return nil // Cannot respond to malformed error
	}

	// Validate inbound error message per SCACI §3.14.1
	// AC must send non-zero code and non-empty message
	if validationErr := ValidateError(&errMsg); validationErr != "" {
		s.logger.ErrorContext(ctx, LogSCACIErrorMsgValidationFailed,
			"command", CmdError,
			"opId", opId,
			"errorToken", validationErr,
			"receivedCode", errMsg.Code,
			"receivedMessage", errMsg.Message)
		// Respond with error per §3.14 - invalid error messages are protocol violations
		return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, validationErr)
	}

	s.logger.WarnContext(ctx, LogSCACIReceivedInboundError,
		"opId", opId,
		"acEui", pkgmioty.FormatEUI64(session.AcEui),
		"posixCode", errMsg.Code,
		"message", errMsg.Message)

	session.UpdateLastSeen()

	// Record the inbound error via ErrorRecorder if available
	if s.errorRecorder != nil {
		if err := s.errorRecorder.RecordInboundError(ctx, session, opId, errMsg.Code, errMsg.Message); err != nil {
			s.logger.WarnContext(ctx, LogSCACIPersistInboundErrorFailed,
				"opId", opId,
				"error", err)
		}
	}

	// Send errorAck to complete the error handshake per §3.14
	return s.sendErrorAck(conn, session, opId)
}

// handleErrorAck processes errorAck messages received from the Application Center (SCACI §3.14).
// AC sends errorAck after receiving an error from SC, completing the error handshake.
func (s *Server) handleErrorAck(_ net.Conn, session *Session, opId int64) error {
	ctx := s.sessionContext(session)

	s.logger.DebugContext(ctx, LogSCACIReceivedErrorAck,
		"opId", opId,
		"acEui", pkgmioty.FormatEUI64(session.AcEui))

	session.UpdateLastSeen()

	// Complete the error handshake via ErrorRecorder if available
	if s.errorRecorder != nil {
		if err := s.errorRecorder.CompleteErrorHandshake(ctx, session, opId); err != nil {
			s.logger.WarnContext(ctx, LogSCACICompleteErrorHandshakeFailed,
				"opId", opId,
				"error", err)
		}
	}

	return nil // No response per spec - errorAck completes the sequence
}
