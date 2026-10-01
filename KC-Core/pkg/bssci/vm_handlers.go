package bssci

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleVMActivate handles VM activate request from Service Center
//

func (s *Server) handleVMActivate(_ *Session, _ *Message, _ map[string]interface{}) error {
	// This would be initiated by Service Center, not received from base station
	// Keeping for completeness but typically won't be called
	return fmt.Errorf("%s", ResolveErrorMessage(errVMOperationSentByBS))
}

// handleVMActivateResponse handles vm.activateRsp from base station
//

func (s *Server) handleVMActivateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Extract endpoint EUI and MAC type from pending operation
	var epEui uint64
	var macType uint8
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation reads
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		pendingOp = s.pendingOperationOrWarn(session, msg.OpId)
	}
	if pendingOp != nil {
		// Convert []byte to uint64
		if len(pendingOp.Endpoint) == 8 {
			epEui = binary.BigEndian.Uint64(pendingOp.Endpoint)
		}
		// Convert int to uint8
		macType, _ = safeUint8(int64(pendingOp.MACType))
	}

	// Check result field to determine success/failure
	result, ok := data["result"].(bool)
	if !ok {
		result = false // Assume failure if no result field
	}

	if result {
		s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMActivateSucceeded,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldEpEui, epEui,
			logger.FieldMacType, macType,
			logger.FieldOpID, msg.OpId)

		// Update session's active VM types
		session.mu.Lock()
		if session.ActiveVMTypes == nil {
			session.ActiveVMTypes = make(map[uint64][]uint8)
		}
		// Add MAC type if not already present
		found := false
		for _, t := range session.ActiveVMTypes[epEui] {
			if t == macType {
				found = true
				break
			}
		}
		if !found {
			session.ActiveVMTypes[epEui] = append(session.ActiveVMTypes[epEui], macType)
		}
		session.mu.Unlock()

		// Record success event
		if s.eventStore != nil && epEui != 0 {
			eventData := map[string]interface{}{
				models.EventDetailKeyBsEui:   mioty.FormatEUI64(session.BaseStationEUI),
				models.EventDetailKeyEpEui:   mioty.FormatEUI64(epEui),
				models.EventDetailKeyMacType: macType,
				models.EventDetailKeyOpID:    msg.OpId,
				eventDataKeyStatus:           eventStatusSuccess,
				models.EventDetailKeyMessage: fmt.Sprintf(eventMsgFmtVMActivated, macType),
			}

			s.recordVMEvent(session, eventData, LogBSSCIFailedToRecordVMActivateSuccessEvent, &models.SystemEvent{
				TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
				EventType:   EventTypeVMActivateSuccess,
				Category:    mioty.CategoryEndpoint,
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf(eventTitleFmtVMActivateSuccess, macType),
				Description: fmt.Sprintf(eventDescFmtVMActivateSuccess, macType, mioty.FormatEUI64(epEui), session.Name),
				SourceType:  mioty.SourceTypeEndpoint,
				SourceName:  mioty.FormatEUI64(epEui),
				CreatedAt:   s.clock.Now(),
				UpdatedAt:   s.clock.Now(),
			})
		}
	} else {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMActivateFailed,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldEpEui, epEui,
			logger.FieldMacType, macType,
			logger.FieldOpID, msg.OpId)

		// Record failure event
		if s.eventStore != nil && epEui != 0 {
			eventData := map[string]interface{}{
				models.EventDetailKeyBsEui:   mioty.FormatEUI64(session.BaseStationEUI),
				models.EventDetailKeyEpEui:   mioty.FormatEUI64(epEui),
				models.EventDetailKeyMacType: macType,
				models.EventDetailKeyOpID:    msg.OpId,
				eventDataKeyStatus:           eventStatusFailed,
				models.EventDetailKeyError:   eventErrVMActivateRejected,
			}

			s.recordVMEvent(session, eventData, LogBSSCIFailedToRecordVMActivateFailureEvent, &models.SystemEvent{
				TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
				EventType:   EventTypeVMActivateFailed,
				Category:    mioty.CategoryEndpoint,
				Severity:    SeverityError,
				Title:       fmt.Sprintf(eventTitleFmtVMActivateFailed, macType),
				Description: fmt.Sprintf(eventDescFmtVMActivateFailed, macType, mioty.FormatEUI64(epEui), session.Name),
				SourceType:  mioty.SourceTypeEndpoint,
				SourceName:  mioty.FormatEUI64(epEui),
				CreatedAt:   s.clock.Now(),
				UpdatedAt:   s.clock.Now(),
			})
		}
	}

	return nil
}

// handleVMActivateComplete handles vm.activateCmp from base station
//

func (s *Server) handleVMActivateComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Clean up pending operation from database
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOperation,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldOpID, msg.OpId,
			logger.FieldError, err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMActivateOperationCompleted,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	return nil
}

// handleVMDeactivateResponse handles vm.deactivateRsp from base station
//

func (s *Server) handleVMDeactivateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Extract endpoint EUI and MAC type from pending operation
	var epEui uint64
	var macType uint8
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation reads
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		pendingOp = s.pendingOperationOrWarn(session, msg.OpId)
	}
	if pendingOp != nil {
		// Convert []byte to uint64
		if len(pendingOp.Endpoint) == 8 {
			epEui = binary.BigEndian.Uint64(pendingOp.Endpoint)
		}
		// Convert int to uint8
		macType, _ = safeUint8(int64(pendingOp.MACType))
	}

	// Check result field
	result, ok := data["result"].(bool)
	if !ok {
		result = false
	}

	if result {
		s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMDeactivateSucceeded,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldEpEui, epEui,
			logger.FieldMacType, macType,
			logger.FieldOpID, msg.OpId)

		// Remove MAC type from session's active VM types
		session.mu.Lock()
		if session.ActiveVMTypes != nil {
			activeTypes := session.ActiveVMTypes[epEui]
			newTypes := []uint8{}
			for _, t := range activeTypes {
				if t != macType {
					newTypes = append(newTypes, t)
				}
			}
			session.ActiveVMTypes[epEui] = newTypes
		}
		session.mu.Unlock()

		// Record success event
		if s.eventStore != nil && epEui != 0 {
			eventData := map[string]interface{}{
				models.EventDetailKeyBsEui:   mioty.FormatEUI64(session.BaseStationEUI),
				models.EventDetailKeyEpEui:   mioty.FormatEUI64(epEui),
				models.EventDetailKeyMacType: macType,
				models.EventDetailKeyOpID:    msg.OpId,
				eventDataKeyStatus:           eventStatusSuccess,
				models.EventDetailKeyMessage: fmt.Sprintf(eventMsgFmtVMDeactivated, macType),
			}

			s.recordVMEvent(session, eventData, LogBSSCIFailedToRecordVMDeactivateSuccessEvent, &models.SystemEvent{
				TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
				EventType:   EventTypeVMDeactivateSuccess,
				Category:    mioty.CategoryEndpoint,
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf(eventTitleFmtVMDeactivateSuccess, macType),
				Description: fmt.Sprintf(eventDescFmtVMDeactivateSuccess, macType, mioty.FormatEUI64(epEui), session.Name),
				SourceType:  mioty.SourceTypeEndpoint,
				SourceName:  mioty.FormatEUI64(epEui),
				CreatedAt:   s.clock.Now(),
				UpdatedAt:   s.clock.Now(),
			})
		}
	} else {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMDeactivateFailed,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldEpEui, epEui,
			logger.FieldMacType, macType,
			logger.FieldOpID, msg.OpId)
	}

	return nil
}

// handleVMDeactivateComplete handles vm.deactivateCmp from base station
//

func (s *Server) handleVMDeactivateComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Clean up pending operation from database
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOperation,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldOpID, msg.OpId,
			logger.FieldError, err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMDeactivateOperationCompleted,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	return nil
}

// handleVMStatusResponse handles vm.statusRsp from base station
//

func (s *Server) handleVMStatusResponse(session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Extract endpoint EUI from pending operation
	var epEui uint64
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation reads
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		pendingOp = s.pendingOperationOrWarn(session, msg.OpId)
	}
	if pendingOp != nil {
		if len(pendingOp.Endpoint) == 8 {
			epEui = binary.BigEndian.Uint64(pendingOp.Endpoint)
		}
	}

	// Extract macTypes array from response
	macTypesRaw, ok := data["macTypes"].([]interface{})
	if !ok {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMStatusResponseMissingMacTypesField,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldEpEui, epEui)
		return nil
	}

	macTypes := []uint8{}
	for _, mt := range macTypesRaw {
		if macType, ok := mt.(float64); ok {
			macTypeSafe, errToken := safeUint8(int64(macType))
			if errToken != "" {
				s.logger.WarnContext(s.sessionContext(session), LogBSSCIIntegerOverflowInMacTypeParsing,
					logger.FieldField, logger.FieldMacType,
					logger.FieldError, ResolveErrorMessage(errToken))
				continue // Skip invalid macType
			}
			macTypes = append(macTypes, macTypeSafe)
		}
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMStatusReceived,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldEpEui, epEui,
		logger.FieldActiveMacTypes, macTypes,
		logger.FieldOpID, msg.OpId)

	// Update session's active VM types
	session.mu.Lock()
	if session.ActiveVMTypes == nil {
		session.ActiveVMTypes = make(map[uint64][]uint8)
	}
	session.ActiveVMTypes[epEui] = macTypes
	session.mu.Unlock()

	// Record status event
	if s.eventStore != nil && epEui != 0 {
		eventData := map[string]interface{}{
			models.EventDetailKeyBsEui:          mioty.FormatEUI64(session.BaseStationEUI),
			models.EventDetailKeyEpEui:          mioty.FormatEUI64(epEui),
			models.EventDetailKeyActiveMacTypes: macTypes,
			models.EventDetailKeyOpID:           msg.OpId,
		}

		s.recordVMEvent(session, eventData, LogBSSCIFailedToRecordVMStatusEvent, &models.SystemEvent{
			TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
			EventType:   EventTypeVMStatusReceived,
			Category:    mioty.CategoryEndpoint,
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf(eventTitleFmtVMStatus, len(macTypes)),
			Description: fmt.Sprintf(eventDescFmtVMStatus, mioty.FormatEUI64(epEui), len(macTypes), session.Name),
			SourceType:  mioty.SourceTypeEndpoint,
			SourceName:  mioty.FormatEUI64(epEui),
			CreatedAt:   s.clock.Now(),
			UpdatedAt:   s.clock.Now(),
		})
	}

	// Remove from pending operations
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation removal
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOperation,
			logger.FieldOpID, msg.OpId, logger.FieldError, err)
	}

	return nil
}

// Helper functions for Service Center to send VM commands

// SendVMActivate sends a VM activate command to a base station
func (s *Server) SendVMActivate(sessionID string, epEui uint64, macType uint8) error {
	session, exists := s.sessions.get(sessionID)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	// Create VM activate message per BSSCI spec
	vmActivate := map[string]interface{}{
		"command": mioty.CmdVMActivate,
		"epEui":   epEui,
		"macType": macType,
		"opId":    opId,
	}

	// Create EUI bytes for database storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Persist operation to database
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdVMActivate, vmActivate, euiBytes, map[string]interface{}{
		"epEui":   epEui,
		"macType": macType,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMActivateOperation,
			logger.FieldSessionID, sessionID,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return err
	}

	if err := s.sendMessage(session, vmActivate); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMActivate), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMActivateCommand,
		logger.FieldSessionID, sessionID,
		logger.FieldEpEui, epEui,
		logger.FieldMacType, macType,
		logger.FieldOpID, opId)

	return nil
}

// SendVMDeactivate sends a VM deactivate command to a base station
func (s *Server) SendVMDeactivate(sessionID string, epEui uint64, macType uint8) error {
	session, exists := s.sessions.get(sessionID)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	// Create VM deactivate message per BSSCI spec
	vmDeactivate := map[string]interface{}{
		"command": mioty.CmdVMDeactivate,
		"epEui":   epEui,
		"macType": macType,
		"opId":    opId,
	}

	// Create EUI bytes for database storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Persist operation to database
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdVMDeactivate, vmDeactivate, euiBytes, map[string]interface{}{
		"epEui":   epEui,
		"macType": macType,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMDeactivateOperation,
			logger.FieldSessionID, sessionID,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return err
	}

	if err := s.sendMessage(session, vmDeactivate); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMDeactivate), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMDeactivateCommand,
		logger.FieldSessionID, sessionID,
		logger.FieldEpEui, epEui,
		logger.FieldMacType, macType,
		logger.FieldOpID, opId)

	return nil
}

// SendVMStatus sends a VM status request to a base station
func (s *Server) SendVMStatus(sessionID string, epEui uint64) error {
	session, exists := s.sessions.get(sessionID)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	// Create VM status message per BSSCI spec
	vmStatus := map[string]interface{}{
		"command": mioty.CmdVMStatus,
		"epEui":   epEui,
		"opId":    opId,
	}

	// Create EUI bytes for database storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Persist operation to database
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdVMStatus, vmStatus, euiBytes, map[string]interface{}{
		"epEui": epEui,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMStatusOperation,
			logger.FieldSessionID, sessionID,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return err
	}

	if err := s.sendMessage(session, vmStatus); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMStatus), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMStatusRequest,
		logger.FieldSessionID, sessionID,
		logger.FieldEpEui, epEui,
		logger.FieldOpID, opId)

	return nil
}

// SendVMDownlinkData sends downlink data via VM sub-channel
func (s *Server) SendVMDownlinkData(sessionID string, epEui uint64, macType uint8, userData []byte) error {
	session, exists := s.sessions.get(sessionID)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	// Create VM downlink data message per BSSCI spec
	// TxTime is optional - only set when needed
	var txTime *int64
	ts := s.clock.Now().UnixMilli()
	if ts > 0 {
		txTime = &ts
	}

	vmDlData := &mioty.VMDLData{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdVMDLData,
			OpId:        opId,
		},
		EpEui:    epEui,
		MACType:  macType,
		UserData: userData,
		TxTime:   txTime, // Optional pointer per types.go:662
	}

	// Create EUI bytes for database storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Convert struct to map for persistPendingOperation
	vmDlDataMap := map[string]interface{}{
		"command":  mioty.CmdVMDLData,
		"opId":     opId,
		"epEui":    epEui,
		"macType":  macType,
		"userData": userData,
	}
	if txTime != nil {
		vmDlDataMap["txTime"] = *txTime
	}

	// Persist operation to database
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdVMDLData, vmDlDataMap, euiBytes, map[string]interface{}{
		"epEui":   epEui,
		"macType": macType,
		"data":    userData,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMDownlinkDataOperation,
			logger.FieldSessionID, sessionID,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return err
	}

	if err := s.sendMessage(session, vmDlData); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMDlData), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMDownlinkData,
		logger.FieldSessionID, sessionID,
		logger.FieldEpEui, epEui,
		logger.FieldMacType, macType,
		logger.FieldDataLen, len(userData),
		logger.FieldOpID, opId)

	return nil
}

// handleVMDeactivate is a stub - deactivate is initiated by Service Center
//

func (s *Server) handleVMDeactivate(_ *Session, _ *Message, _ map[string]interface{}) error {
	return fmt.Errorf("%s", ResolveErrorMessage(errVMOperationSentByBS))
}

// handleVMStatus is a stub - status request is initiated by Service Center
//

func (s *Server) handleVMStatus(_ *Session, _ *Message, _ map[string]interface{}) error {
	return fmt.Errorf("%s", ResolveErrorMessage(errVMOperationSentByBS))
}

// ============================================================================
// Community Edition VM Stubs
// ============================================================================
// The following handlers provide clean scaffolding for VM operations.
// Community edition rejects these commands with "unsupported" errors.
// VM/Recon operations are not supported in the community edition.

// handleVMStatusComplete handles VM status complete message from base station
// Community edition: Emits catalog error to base station, does not clean up pending operations
func (s *Server) handleVMStatusComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMStatusCompleteNotSupported,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	s.sendCatalogError(session, msg.OpId, catalogErr)

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}

// handleVMDLData handles VM downlink data request (SC-initiated)
// Community edition: Emits catalog error to base station, does not clean up pending operations
func (s *Server) handleVMDLData(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMDLDataNotSupported,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	s.sendCatalogError(session, msg.OpId, catalogErr)

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}

// handleVMDLDataResponse handles VM downlink data response from base station
// Community edition: Emits catalog error to base station, does not clean up pending operations
func (s *Server) handleVMDLDataResponse(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMDLDataResponseNotSupported,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	s.sendCatalogError(session, msg.OpId, catalogErr)

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}

// handleVMDLDataComplete handles VM downlink data complete message from base station
// Community edition: Emits catalog error to base station, does not clean up pending operations
func (s *Server) handleVMDLDataComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMDLDataCompleteNotSupported,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	s.sendCatalogError(session, msg.OpId, catalogErr)

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}
