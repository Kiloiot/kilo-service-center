package bssci

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// VMOperation tracks pending VM operations for three-way handshake
type VMOperation struct {
	Type      string    // "activate", "deactivate", "dlData"
	Endpoint  uint64    // Endpoint EUI
	MACType   uint8     // MAC type (0-7)
	Data      []byte    // For downlink data
	Timestamp time.Time // When operation started
}

// handleVMActivate handles VM activate request from Service Center
//

func (s *Server) handleVMActivate(_ *Server, _ *Session, _ *Message, _ map[string]interface{}) error {
	// This would be initiated by Service Center, not received from base station
	// Keeping for completeness but typically won't be called
	return fmt.Errorf("%s", ResolveErrorMessage(errVMOperationSentByBS))
}

// handleVMActivateResponse handles vm.activateRsp from base station
//

func (s *Server) handleVMActivateResponse(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Extract endpoint EUI and MAC type from pending operation
	var epEui uint64
	var macType uint8
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation reads
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		var err error
		pendingOp, err = s.statusSvc.GetPendingOperation(session, msg.OpId)
		if err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
				"opId", msg.OpId, "error", err)
		}
	}
	if pendingOp != nil {
		// Convert []byte to uint64
		if len(pendingOp.Endpoint) == 8 {
			epEui = binary.BigEndian.Uint64(pendingOp.Endpoint)
		}
		// Convert int to uint8
		macType, _ = safeUint8(int64(pendingOp.MACType), "macType")
	}

	// Check result field to determine success/failure
	result, ok := data["result"].(bool)
	if !ok {
		result = false // Assume failure if no result field
	}

	if result {
		s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMActivateSucceeded,
			"bsEui", session.BaseStationEUI,
			"epEui", epEui,
			"macType", macType,
			"opId", msg.OpId)

		// Update session's active VM types
		s.mu.Lock()
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
		s.mu.Unlock()

		// Record success event
		if s.eventStore != nil && epEui != 0 {
			eventData := map[string]interface{}{
				"bsEui":   fmt.Sprintf("%016X", session.BaseStationEUI),
				"epEui":   fmt.Sprintf("%016X", epEui),
				"macType": macType,
				"opId":    msg.OpId,
				"status":  "success",
				"message": fmt.Sprintf("VM MAC type %d activated for endpoint", macType),
			}

			ctx := s.sessionContext(session)
			if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
				TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
				EventType:   EventTypeVMActivateSuccess,
				Category:    mioty.CategoryEndpoint,
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf("VM Activate Success - MAC Type %d", macType),
				Description: fmt.Sprintf("Successfully activated VM MAC type %d for endpoint %016X on base station %s", macType, epEui, session.Name),
				SourceType:  mioty.SourceTypeEndpoint,
				SourceName:  fmt.Sprintf("%016X", epEui),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Details:     func() []byte { b, _ := json.Marshal(eventData); return b }(),
			}); err != nil {
				s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordVMActivateSuccessEvent, "error", err)
			}
		}
	} else {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMActivateFailed,
			"bsEui", session.BaseStationEUI,
			"epEui", epEui,
			"macType", macType,
			"opId", msg.OpId)

		// Record failure event
		if s.eventStore != nil && epEui != 0 {
			eventData := map[string]interface{}{
				"bsEui":   fmt.Sprintf("%016X", session.BaseStationEUI),
				"epEui":   fmt.Sprintf("%016X", epEui),
				"macType": macType,
				"opId":    msg.OpId,
				"status":  "failed",
				"error":   "Base station rejected VM activate request",
			}

			ctx := s.sessionContext(session)
			if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
				TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
				EventType:   EventTypeVMActivateFailed,
				Category:    mioty.CategoryEndpoint,
				Severity:    SeverityError,
				Title:       fmt.Sprintf("VM Activate Failed - MAC Type %d", macType),
				Description: fmt.Sprintf("Failed to activate VM MAC type %d for endpoint %016X on base station %s", macType, epEui, session.Name),
				SourceType:  mioty.SourceTypeEndpoint,
				SourceName:  fmt.Sprintf("%016X", epEui),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Details:     func() []byte { b, _ := json.Marshal(eventData); return b }(),
			}); err != nil {
				s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordVMActivateFailureEvent, "error", err)
			}
		}
	}

	return nil
}

// handleVMActivateComplete handles vm.activateCmp from base station
//

func (s *Server) handleVMActivateComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Clean up pending operation from database
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOperation,
			"sessionID", session.DbSessionID,
			"opId", msg.OpId,
			"error", err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMActivateOperationCompleted,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	return nil
}

// handleVMDeactivateResponse handles vm.deactivateRsp from base station
//

func (s *Server) handleVMDeactivateResponse(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Extract endpoint EUI and MAC type from pending operation
	var epEui uint64
	var macType uint8
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation reads
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		var err error
		pendingOp, err = s.statusSvc.GetPendingOperation(session, msg.OpId)
		if err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
				"opId", msg.OpId, "error", err)
		}
	}
	if pendingOp != nil {
		// Convert []byte to uint64
		if len(pendingOp.Endpoint) == 8 {
			epEui = binary.BigEndian.Uint64(pendingOp.Endpoint)
		}
		// Convert int to uint8
		macType, _ = safeUint8(int64(pendingOp.MACType), "macType")
	}

	// Check result field
	result, ok := data["result"].(bool)
	if !ok {
		result = false
	}

	if result {
		s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMDeactivateSucceeded,
			"bsEui", session.BaseStationEUI,
			"epEui", epEui,
			"macType", macType,
			"opId", msg.OpId)

		// Remove MAC type from session's active VM types
		s.mu.Lock()
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
		s.mu.Unlock()

		// Record success event
		if s.eventStore != nil && epEui != 0 {
			eventData := map[string]interface{}{
				"bsEui":   fmt.Sprintf("%016X", session.BaseStationEUI),
				"epEui":   fmt.Sprintf("%016X", epEui),
				"macType": macType,
				"opId":    msg.OpId,
				"status":  "success",
				"message": fmt.Sprintf("VM MAC type %d deactivated for endpoint", macType),
			}

			ctx := s.sessionContext(session)
			if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
				TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
				EventType:   EventTypeVMDeactivateSuccess,
				Category:    mioty.CategoryEndpoint,
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf("VM Deactivate Success - MAC Type %d", macType),
				Description: fmt.Sprintf("Successfully deactivated VM MAC type %d for endpoint %016X on base station %s", macType, epEui, session.Name),
				SourceType:  mioty.SourceTypeEndpoint,
				SourceName:  fmt.Sprintf("%016X", epEui),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Details:     func() []byte { b, _ := json.Marshal(eventData); return b }(),
			}); err != nil {
				s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordVMDeactivateSuccessEvent, "error", err)
			}
		}
	} else {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIVMDeactivateFailed,
			"bsEui", session.BaseStationEUI,
			"epEui", epEui,
			"macType", macType,
			"opId", msg.OpId)
	}

	return nil
}

// handleVMDeactivateComplete handles vm.deactivateCmp from base station
//

func (s *Server) handleVMDeactivateComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Clean up pending operation from database
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOperation,
			"sessionID", session.DbSessionID,
			"opId", msg.OpId,
			"error", err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMDeactivateOperationCompleted,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	return nil
}

// handleVMStatusResponse handles vm.statusRsp from base station
//

func (s *Server) handleVMStatusResponse(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Extract endpoint EUI from pending operation
	var epEui uint64
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation reads
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		var err error
		pendingOp, err = s.statusSvc.GetPendingOperation(session, msg.OpId)
		if err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
				"opId", msg.OpId, "error", err)
		}
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
			"bsEui", session.BaseStationEUI,
			"epEui", epEui)
		return nil
	}

	macTypes := []uint8{}
	for _, mt := range macTypesRaw {
		if macType, ok := mt.(float64); ok {
			macTypeSafe, errToken := safeUint8(int64(macType), "macType")
			if errToken != "" {
				s.logger.WarnContext(s.sessionContext(session), LogBSSCIIntegerOverflowInMacTypeParsing,
					"field", "macType",
					"error", ResolveErrorMessage(errToken))
				continue // Skip invalid macType
			}
			macTypes = append(macTypes, macTypeSafe)
		}
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIVMStatusReceived,
		"bsEui", session.BaseStationEUI,
		"epEui", epEui,
		"activeMacTypes", macTypes,
		"opId", msg.OpId)

	// Update session's active VM types
	s.mu.Lock()
	if session.ActiveVMTypes == nil {
		session.ActiveVMTypes = make(map[uint64][]uint8)
	}
	session.ActiveVMTypes[epEui] = macTypes
	s.mu.Unlock()

	// Record status event
	if s.eventStore != nil && epEui != 0 {
		eventData := map[string]interface{}{
			"bsEui":          fmt.Sprintf("%016X", session.BaseStationEUI),
			"epEui":          fmt.Sprintf("%016X", epEui),
			"activeMacTypes": macTypes,
			"opId":           msg.OpId,
		}

		ctx := s.sessionContext(session)
		if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
			EventType:   EventTypeVMStatusReceived,
			Category:    mioty.CategoryEndpoint,
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf("VM Status - %d Active MAC Types", len(macTypes)),
			Description: fmt.Sprintf("Endpoint %016X has %d active VM MAC types on base station %s", epEui, len(macTypes), session.Name),
			SourceType:  mioty.SourceTypeEndpoint,
			SourceName:  fmt.Sprintf("%016X", epEui),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Details:     func() []byte { b, _ := json.Marshal(eventData); return b }(),
		}); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordVMStatusEvent, "error", err)
		}
	}

	// Remove from pending operations
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation removal
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOperation,
			"opId", msg.OpId, "error", err)
	}

	return nil
}

// Helper functions for Service Center to send VM commands

// SendVMActivate sends a VM activate command to a base station
func (s *Server) SendVMActivate(sessionID string, epEui uint64, macType uint8) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

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
	if err := s.persistPendingOperation(session, opId, mioty.CmdVMActivate, vmActivate, euiBytes, map[string]interface{}{
		"epEui":   epEui,
		"macType": macType,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMActivateOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
	}

	// Send message with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, vmActivate); err != nil {
		// CRITICAL: Rollback operation ID on send failure
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()

		// Clean up pending operation from database on send failure
		// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
		if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMActivate), err)
	}

	// Success - persist counter to DB for session resume
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(session), session); err != nil {
		s.logger.Error(LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", sessionID,
			"opId", opId)
		// Don't fail the operation - message was sent successfully
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMActivateCommand,
		"sessionID", sessionID,
		"epEui", epEui,
		"macType", macType,
		"opId", opId)

	return nil
}

// SendVMDeactivate sends a VM deactivate command to a base station
func (s *Server) SendVMDeactivate(sessionID string, epEui uint64, macType uint8) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

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
	if err := s.persistPendingOperation(session, opId, mioty.CmdVMDeactivate, vmDeactivate, euiBytes, map[string]interface{}{
		"epEui":   epEui,
		"macType": macType,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMDeactivateOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
	}

	// Send message with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, vmDeactivate); err != nil {
		// CRITICAL: Rollback operation ID on send failure
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()

		// Clean up pending operation from database on send failure
		// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
		if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMDeactivate), err)
	}

	// Success - persist counter to DB for session resume
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(session), session); err != nil {
		s.logger.Error(LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", sessionID,
			"opId", opId)
		// Don't fail the operation - message was sent successfully
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMDeactivateCommand,
		"sessionID", sessionID,
		"epEui", epEui,
		"macType", macType,
		"opId", opId)

	return nil
}

// SendVMStatus sends a VM status request to a base station
func (s *Server) SendVMStatus(sessionID string, epEui uint64) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

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
	if err := s.persistPendingOperation(session, opId, mioty.CmdVMStatus, vmStatus, euiBytes, map[string]interface{}{
		"epEui": epEui,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMStatusOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
	}

	// Send message with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, vmStatus); err != nil {
		// CRITICAL: Rollback operation ID on send failure
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()

		// Clean up pending operation from database on send failure
		// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
		if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMStatus), err)
	}

	// Success - persist counter to DB for session resume
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(session), session); err != nil {
		s.logger.Error(LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", sessionID,
			"opId", opId)
		// Don't fail the operation - message was sent successfully
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMStatusRequest,
		"sessionID", sessionID,
		"epEui", epEui,
		"opId", opId)

	return nil
}

// SendVMDownlinkData sends downlink data via VM sub-channel
func (s *Server) SendVMDownlinkData(sessionID string, epEui uint64, macType uint8, userData []byte) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

	// Create VM downlink data message per BSSCI spec
	// TxTime is optional - only set when needed
	var txTime *int64
	ts := time.Now().UnixNano() / 1e6 // Unix milliseconds
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
	if err := s.persistPendingOperation(session, opId, mioty.CmdVMDLData, vmDlDataMap, euiBytes, map[string]interface{}{
		"epEui":   epEui,
		"macType": macType,
		"data":    userData,
	}); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistVMDownlinkDataOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
	}

	// Send message with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, vmDlData); err != nil {
		// CRITICAL: Rollback operation ID on send failure
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()

		// Clean up pending operation from database on send failure
		// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
		if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingVMOpAfterSendFailure,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendVMDlData), err)
	}

	// Success - persist counter to DB for session resume
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(session), session); err != nil {
		s.logger.Error(LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", sessionID,
			"opId", opId)
		// Don't fail the operation - message was sent successfully
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentVMDownlinkData,
		"sessionID", sessionID,
		"epEui", epEui,
		"macType", macType,
		"dataLen", len(userData),
		"opId", opId)

	return nil
}

// handleVMDeactivate is a stub - deactivate is initiated by Service Center
//

func (s *Server) handleVMDeactivate(_ *Server, _ *Session, _ *Message, _ map[string]interface{}) error {
	return fmt.Errorf("%s", ResolveErrorMessage(errVMOperationSentByBS))
}

// handleVMStatus is a stub - status request is initiated by Service Center
//

func (s *Server) handleVMStatus(_ *Server, _ *Session, _ *Message, _ map[string]interface{}) error {
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
func (s *Server) handleVMStatusComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), "VM status complete not supported in community edition",
		"eui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	if err := s.sendCatalogError(session, msg.OpId, catalogErr); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), "Failed to send catalog error",
			"opId", msg.OpId,
			"error", err)
	}

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}

// handleVMDLData handles VM downlink data request (SC-initiated)
// Community edition: Emits catalog error to base station, does not clean up pending operations
func (s *Server) handleVMDLData(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), "VM downlink data not supported in community edition",
		"eui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	if err := s.sendCatalogError(session, msg.OpId, catalogErr); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), "Failed to send catalog error",
			"opId", msg.OpId,
			"error", err)
	}

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}

// handleVMDLDataResponse handles VM downlink data response from base station
// Community edition: Emits catalog error to base station, does not clean up pending operations
func (s *Server) handleVMDLDataResponse(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), "VM downlink data response not supported in community edition",
		"eui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	if err := s.sendCatalogError(session, msg.OpId, catalogErr); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), "Failed to send catalog error",
			"opId", msg.OpId,
			"error", err)
	}

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}

// handleVMDLDataComplete handles VM downlink data complete message from base station
// Community edition: Emits catalog error to base station, does not clean up pending operations
func (s *Server) handleVMDLDataComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}
	s.logger.WarnContext(s.sessionContext(session), "VM downlink data complete not supported in community edition",
		"eui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Emit catalog error to base station per BSSCI protocol
	// Use POSIX_ENOSYS (38) to match existing catalog token for unsupported commands
	catalogErr := NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS)
	if err := s.sendCatalogError(session, msg.OpId, catalogErr); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), "Failed to send catalog error",
			"opId", msg.OpId,
			"error", err)
	}

	// Return error for internal tracking
	// Known limitation: pending operations are not cleaned up for unsupported VM commands.
	return fmt.Errorf("%s", ResolveErrorMessage(errUnsupportedCommand))
}
