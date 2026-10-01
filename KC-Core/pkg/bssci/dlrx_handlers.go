package bssci

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Import neutral scheduler contracts
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// handleDLRXStatus handles dlRxStat from base station when endpoint reports DL reception quality
func (s *Server) handleDLRXStatus(session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Unmarshal directly into canonical MIOTY type per BSSCI §3.15.1
	var dlRxStatus mioty.DLRxStatus

	// Convert map to msgpack bytes for proper unmarshalling
	msgpackData, err := msgpack.Marshal(data)
	if err != nil {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errFailedToMarshal)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Unmarshal using canonical struct with proper field tags
	if err := msgpack.Unmarshal(msgpackData, &dlRxStatus); err != nil {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errFailedToDecode)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Set command type and operation ID
	dlRxStatus.CommandType = mioty.CmdDLRxStatus
	dlRxStatus.OpId = msg.OpId

	// Validate mandatory fields are present
	if dlRxStatus.EpEui == 0 {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxEpEui)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	if dlRxStatus.RxTime == 0 {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxTime)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Validate and extract mandatory packetCnt field per BSSCI §3.15.1
	packetCnt, hasPacketCnt := getNumericField(data, "packetCnt")
	if !hasPacketCnt {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxPacketCnt)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	// Validate packetCnt is within uint32 range (BSSCI §3.15.1 defines as unsigned)
	if packetCnt < 0 || packetCnt > math.MaxUint32 {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidDlRxPacketCnt)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	dlRxStatus.PacketCnt = uint32(packetCnt)

	// Validate and extract mandatory dlRxSnr field per BSSCI §3.15.1
	dlRxSnr, hasSnr := getFloatFieldValidated(data, "dlRxSnr")
	if !hasSnr {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxSnr)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	// Validate SNR is finite and within plausible range per BSSCI §5.15.1
	if errToken := validateFiniteFloat(dlRxSnr, mioty.DLRxSnrMinDB, mioty.DLRxSnrMaxDB); errToken != "" {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIDLRXStatusSNRValidationFailed,
			logger.FieldEpEui, dlRxStatus.EpEui,
			logger.FieldDlRxSnr, dlRxSnr,
			logger.FieldError, errToken)
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	dlRxStatus.DlRxSnr = dlRxSnr

	// Validate and extract mandatory dlRxRssi field per BSSCI §3.15.1
	dlRxRssi, hasRssi := getFloatFieldValidated(data, "dlRxRssi")
	if !hasRssi {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxRssi)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	// Validate RSSI is finite and within plausible range per BSSCI §5.15.1
	if errToken := validateFiniteFloat(dlRxRssi, mioty.DLRxRssiMinDBm, mioty.DLRxRssiMaxDBm); errToken != "" {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIDLRXStatusRSSIValidationFailed,
			logger.FieldEpEui, dlRxStatus.EpEui,
			logger.FieldDlRxRssi, dlRxRssi,
			logger.FieldError, errToken)
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	dlRxStatus.DlRxRssi = dlRxRssi

	// Create session context for tenant resolution and persistence
	ctx := s.sessionContext(session)

	s.logger.InfoContext(ctx, LogBSSCIReceivedDLRxStatFromBaseStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldEpEui, dlRxStatus.EpEui,
		logger.FieldRxTime, dlRxStatus.RxTime,
		logger.FieldPacketCnt, dlRxStatus.PacketCnt,
		logger.FieldDlRxSnr, dlRxStatus.DlRxSnr,
		logger.FieldDlRxRssi, dlRxStatus.DlRxRssi)

	// Resolve endpoint owner tenant for roaming scenarios
	tenantID, err := s.resolveEndpointTenantID(ctx, session, dlRxStatus.EpEui)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToResolveEndpointTenantForDLRXStatus,
			logger.FieldError, err,
			logger.FieldEpEui, dlRxStatus.EpEui)
		if err := s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistDLRXStatus)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Resolve organization for owner tenant
	var ownerOrgUUID uuid.UUID
	if s.orgResolver != nil {
		ownerOrgUUID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, tenantID)
		if err != nil {
			s.logger.WarnContext(ctx, LogBSSCIOrgLookupFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
			// Continue without org - ownerOrgUUID remains Nil
		}
	}

	// Build owner-scoped context with correct tenant AND organization
	ownerCtx := pkgcontext.WithTenantID(ctx, tenantID)
	if ownerOrgUUID != uuid.Nil {
		ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
	}

	// Convert endpoint EUI to bytes (used multiple times below)
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, dlRxStatus.EpEui)

	// Convert BS EUI to bytes for repository call
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

	// Check if this dlRxStat corresponds to a pending query (BSSCI §5.15 correlation)
	if s.dlrxStore != nil {
		// Correlate by tenant+endpoint only (oldest pending) per BSSCI §5.15
		// BS opId stored for audit but NOT used in WHERE clause (different namespace)
		found, err := s.dlrxStore.MarkDLRXStatusReceived(
			ownerCtx,
			tenantID,
			epEuiBytes, // Match by tenant + endpoint only
			bsEuiBytes, // Audit: which BS actually reported
			msg.OpId,   // Audit: BS opId (different namespace)
		)
		if err != nil {
			s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToCorrelateDLRXQuery,
				logger.FieldEpEui, dlRxStatus.EpEui,
				logger.FieldOpID, msg.OpId,
				logger.FieldError, err)
			// Non-fatal: Continue processing even if correlation check fails
		}

		if !found {
			s.logger.WarnContext(ownerCtx, LogBSSCIUnsolicitedDLRXStatus,
				logger.FieldEpEui, dlRxStatus.EpEui,
				logger.FieldTenant, tenantID,
				logger.FieldBsEui, session.BaseStationEUI)
			// Log-only mode: continue processing unsolicited reports
		}
	}

	// Resolve endpoint owner's organization UUID (BSSCI §5.15)
	epOwnerOrgUUID, err := s.resolveOwnerOrgUUID(ownerCtx, tenantID, epEuiBytes)
	if err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToResolveEndpointOwnerOrgForDLRXStatus,
			logger.FieldError, err,
			logger.FieldTenant, tenantID,
			logger.FieldEpEui, dlRxStatus.EpEui)
		// Continue with nil org - backward compatible, but logged
	}

	// Store DL RX status in database under endpoint owner tenant + org
	if err := s.persistDLRXStatus(ownerCtx, tenantID, epOwnerOrgUUID, session.BaseStationEUI, dlRxStatus.EpEui, dlRxStatus.RxTime, dlRxStatus.PacketCnt, dlRxStatus.DlRxSnr, dlRxStatus.DlRxRssi); err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToPersistDLRxStatus,
			logger.FieldError, err,
			logger.FieldEpEui, dlRxStatus.EpEui,
			logger.FieldRxTime, dlRxStatus.RxTime,
			logger.FieldTenantID, tenantID)
		// Continue processing even if persistence fails - don't break the protocol
	}

	s.recordDLRxStatus(ownerCtx, session, tenantID, &dlRxStatus, msg.OpId)

	// Send dlRxStatRsp to acknowledge
	response := map[string]interface{}{
		"command": mioty.CmdDLRxStatusResponse,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, response)
}

// handleDLRXStatusResponse handles dlRxStatRsp from base station
func (s *Server) handleDLRXStatusResponse(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIReceivedDLRxStatRspFromBaseStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Send dlRxStatCmp to complete the three-way handshake
	complete := map[string]interface{}{
		"command": mioty.CmdDLRxStatusComplete,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, complete)
}

// handleDLRXStatusComplete handles dlRxStatCmp from base station
func (s *Server) handleDLRXStatusComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIDLRxStatusOperationCompleted,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	return nil
}

// SendDLRXStatusQuery sends a DL RX status query to request endpoint DL reception quality
func (s *Server) SendDLRXStatusQuery(sessionID string, epEui uint64) error {
	session, exists := s.sessions.get(sessionID)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the correlation and recovery records, then write
	// the frame. The counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	// Create dlRxStatQry message per MIOTY spec Section 5.16.1
	msg := map[string]interface{}{
		"command": mioty.CmdDLRxStatusQuery,
		"opId":    opId,
		"epEui":   epEui,
	}

	// Create EUI bytes for storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// The correlation row and the recovery record must both be durable before
	// the frame is written; either persistence failure aborts the send.
	if err := s.persistDLRXQueryCorrelation(session, opId, epEui, euiBytes); err != nil {
		return err
	}
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdDLRxStatusQuery, msg, euiBytes, nil); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLRxStatQryOperation,
			logger.FieldSessionID, sessionID,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return err
	}

	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationAfterSendFailure,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlRxStatQry), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentDLRxStatQryToBaseStation,
		logger.FieldSessionID, sessionID,
		logger.FieldEpEui, epEui,
		logger.FieldOpID, opId)

	return nil
}

// persistDLRXQueryCorrelation durably records the dl_rx_status_queries
// correlation row for an outgoing dlRxStatQry (BSSCI rev1 §5.16 / classic
// §3.16) under the endpoint owner's tenant. Correlation persistence failure is
// a pre-write failure: the caller must not put the query on the wire, because
// the eventual dlRxStat report could not be attributed. Owner-organization
// resolution failure alone stays non-fatal (nil org, backward compatible).
func (s *Server) persistDLRXQueryCorrelation(session *Session, opId int64, epEui uint64, euiBytes []byte) error {
	if s.dlrxStore == nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLRxStatQueryTracking,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldOpID, opId,
			logger.FieldEpEui, epEui)
		return NewCatalogError(errFailedToPersistDLRxCorrelation, POSIX_EPROTO)
	}

	// A roaming endpoint's query belongs to its owner, not to the station's
	// tenant, so the owner's dlRxStat finds it (BSSCI §5.15).
	tenantID, err := s.resolveEndpointTenantID(s.sessionContext(session), session, epEui)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistDLRxCorrelation), err)
	}

	// Resolve endpoint owner's organization UUID (BSSCI §5.15)
	epOwnerOrgUUID, err := s.resolveOwnerOrgUUID(s.sessionContext(session), tenantID, euiBytes)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToResolveOwnerOrgForDLRxQuery,
			logger.FieldError, err,
			logger.FieldTenant, tenantID,
			logger.FieldEpEui, epEui)
		// Continue with nil org - backward compatible, but logged
	}

	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

	if err := s.dlrxStore.CreateDLRXStatusQuery(s.sessionContext(session), tenantID, epOwnerOrgUUID, euiBytes, bsEuiBytes, opId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLRxStatQueryTracking,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldOpID, opId,
			logger.FieldEpEui, epEui,
			logger.FieldError, err)
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistDLRxCorrelation), err)
	}

	return nil
}

// handleDLRXStatusQueryResponse handles dlRxStatQryRsp from base station
func (s *Server) handleDLRXStatusQueryResponse(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedDLRxStatQryRspFromBaseStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// The service center completes its own SC-initiated dlRxStatQry operation
	// (BSSCI §3.16): it sends dlRxStatQryCmp and finalizes the pending
	// operation. A spec-compliant base station never returns dlRxStatQryCmp,
	// so the pending row is removed here or it leaks.
	complete := map[string]interface{}{
		"command": mioty.CmdDLRxStatusQueryComplete,
		"opId":    msg.OpId,
	}
	if err := s.sendMessage(session, complete); err != nil {
		return err
	}
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			logger.FieldError, err, logger.FieldOpID, msg.OpId)
	}
	return nil
}

// resolveOwnerOrgUUID fetches the endpoint owner's organization UUID
// Returns nil UUID (backward compatible) if endpoint not found or orgResolver unavailable
func (s *Server) resolveOwnerOrgUUID(ctx context.Context, tenantID int64, epEuiBytes []byte) (*uuid.UUID, error) {
	// Community edition or no org resolver - return nil (backward compatible)
	if s.orgResolver == nil {
		return nil, nil
	}

	// Fetch endpoint to verify it exists and belongs to this tenant
	if s.endpointRepo == nil {
		return nil, errEndpointRepositoryNotAvailable
	}

	endpoint, err := s.endpointRepo.GetByEUI(ctx, tenantID, epEuiBytes)
	if err != nil {
		return nil, fmt.Errorf(errFmtFetchEndpoint, err)
	}
	if endpoint == nil {
		return nil, errDownlinkEndpointNotFound
	}

	// Resolve org UUID for the endpoint's tenant (which may differ from BS tenant in roaming)
	orgUUID, err := s.orgResolver.GetDefaultOrgForTenant(ctx, endpoint.TenantID)
	if err != nil {
		return nil, fmt.Errorf(errFmtResolveOrgForEndpointOwnerTenant, endpoint.TenantID, err)
	}

	// Return pointer to UUID (nil if uuid.Nil for community mode)
	if orgUUID == uuid.Nil {
		return nil, nil
	}
	return &orgUUID, nil
}

// persistDLRXStatus stores a DL RX status report in the database under the endpoint owner tenant
func (s *Server) persistDLRXStatus(ctx context.Context, tenantID int64, ownerOrgUUID *uuid.UUID, bsEui uint64, epEui uint64, rxTime int64, packetCnt uint32, dlRxSnr float64, dlRxRssi float64) error {
	// Convert EUI to bytes
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, epEui)

	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, bsEui)

	// Build canonical DLRXStatus using resolved endpoint owner tenant + org (BSSCI §5.15)
	status := &mioty.DLRXStatus{
		TenantID:       tenantID,
		OrganizationID: ownerOrgUUID, // Endpoint owner's org, not BS owner's
		EpEui:          epEuiBytes,
		BsEui:          bsEuiBytes,
		RxTime:         rxTime,
		PacketCnt:      packetCnt,
		DlRxSnr:        dlRxSnr,
		DlRxRssi:       dlRxRssi,
	}

	// Persist through the DL RX status store with session context
	if s.dlrxStore != nil {
		if err := s.dlrxStore.CreateDLRXStatus(ctx, status); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistDLRXStatus), err)
		}

		s.logger.DebugContext(ctx, LogBSSCIPersistedDLRxStatus,
			logger.FieldEpEui, epEui,
			logger.FieldRxTime, rxTime,
			logger.FieldDlRxSnr, dlRxSnr,
			logger.FieldDlRxRssi, dlRxRssi,
			logger.FieldTenantID, tenantID)
	} else {
		s.logger.WarnContext(ctx, LogBSSCIMessageStoreNotAvailableForDLRxStatus,
			logger.FieldEpEui, epEui,
			logger.FieldRxTime, rxTime)
	}

	return nil
}

// recordDLRxStatus files the reception status an endpoint reported under its
// owner (roaming tenant isolation), naming the station that relayed it.
func (s *Server) recordDLRxStatus(ownerCtx context.Context, session *Session, tenantID int64, status *mioty.DLRxStatus, opID int64) {
	if s.eventStore == nil {
		return
	}
	station := IdentifyStation(ownerCtx, s.basestationRepo, tenantID, session.BaseStationEUI)
	eventData := map[string]interface{}{
		models.EventDetailKeyBsEui:     station.EUI,
		models.EventDetailKeyEpEui:     mioty.FormatEUI64(status.EpEui),
		models.EventDetailKeyRxTime:    status.RxTime,
		models.EventDetailKeyPacketCnt: status.PacketCnt,
		models.EventDetailKeyDlRxSnr:   status.DlRxSnr,
		models.EventDetailKeyDlRxRssi:  status.DlRxRssi,
		models.EventDetailKeyOpID:      opID,
		models.EventDetailKeyTimestamp: s.clock.Now().Format(time.RFC3339),
	}
	if station.Name != "" {
		eventData[models.EventDetailKeyBaseStationName] = station.Name
	}
	details, err := json.Marshal(eventData)
	if err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToRecordDLRxStatusEvent, logger.FieldError, err)
		return
	}
	if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:  strconv.FormatInt(tenantID, 10),
		EventType: EventTypeDLRxStatusReceived,
		Category:  models.EventCategoryMessage,
		Severity:  models.EventSeverityInfo,
		Title:     models.EventTitleDLRxStatus,
		Description: fmt.Sprintf(eventDescFmtDLReceptionReported,
			mioty.FormatEUI64(status.EpEui), station.Label(), status.DlRxSnr, status.DlRxRssi),
		SourceType:    mioty.SourceTypeEndpoint,
		SourceName:    mioty.FormatEUI64(status.EpEui),
		BasestationID: station.ID,
		CreatedAt:     s.clock.Now(),
		UpdatedAt:     s.clock.Now(),
		Details:       details,
	}); err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToRecordDLRxStatusEvent, logger.FieldError, err)
	}
}
