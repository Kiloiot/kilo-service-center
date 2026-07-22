package bssci

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleStatusResponse handles statusRsp from base station per MIOTY BSSCI v1.0.0 Section 3.5.2
func (s *Server) handleStatusResponse(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedStatusRspFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"data", data)

	// Payload is normalized by handleMessage before dispatch.
	// All mandatory fields guaranteed present and correctly typed.
	// Optional fields have defaults applied (nil for pointer types).

	// Extract mandatory fields and validate per BSSCI §3.17
	// Handle both int and int64 from msgpack decoding
	codeVal, hasCode := data["code"]
	if !hasCode || codeVal == nil {
		errMsg := ResolveErrorMessage(errMissingStatusCode)
		_ = s.sendError(session, msg.OpId, POSIX_EPROTO, errMsg) // Protocol error per BSSCI §2.4
		return fmt.Errorf("%s", errMsg)
	}
	var code int64
	switch v := codeVal.(type) {
	case int64:
		code = v
	case int:
		code = int64(v)
	default:
		errMsg := fmt.Sprintf("%s: code field has unexpected type %T", ResolveErrorMessage(errInvalidFieldType), v)
		_ = s.sendError(session, msg.OpId, POSIX_EPROTO, errMsg) // Protocol error per BSSCI §2.4
		return fmt.Errorf("%s", errMsg)
	}

	messageVal, hasMessage := data["message"]
	if !hasMessage || messageVal == nil {
		errMsg := ResolveErrorMessage(errMissingStatusMessage)
		_ = s.sendError(session, msg.OpId, POSIX_EPROTO, errMsg) // Protocol error per BSSCI §2.4
		return fmt.Errorf("%s", errMsg)
	}
	message, ok := messageVal.(string)
	if !ok {
		errMsg := fmt.Sprintf("%s: message field has unexpected type %T", ResolveErrorMessage(errInvalidFieldType), messageVal)
		_ = s.sendError(session, msg.OpId, POSIX_EPROTO, errMsg) // Protocol error per BSSCI §2.4
		return fmt.Errorf("%s", errMsg)
	}

	timeVal, hasTime := data["time"]
	if !hasTime || timeVal == nil {
		errMsg := ResolveErrorMessage(errMissingStatusTime)
		_ = s.sendError(session, msg.OpId, POSIX_EPROTO, errMsg) // Protocol error per BSSCI §2.4
		return fmt.Errorf("%s", errMsg)
	}
	var systemTime int64
	switch v := timeVal.(type) {
	case int64:
		systemTime = v
	case int:
		systemTime = int64(v)
	default:
		errMsg := fmt.Sprintf("%s: time field has unexpected type %T", ResolveErrorMessage(errInvalidFieldType), v)
		_ = s.sendError(session, msg.OpId, POSIX_EPROTO, errMsg) // Protocol error per BSSCI §2.4
		return fmt.Errorf("%s", errMsg)
	}

	// Extract optional fields (normalized to nil if absent)
	var dutyCycle *float64
	if dc := data["dutyCycle"]; dc != nil {
		if dcVal, ok := dc.(float64); ok {
			dutyCycle = &dcVal
		}
	}

	var uptimeSeconds *int64
	if uptime := data["uptime"]; uptime != nil {
		if uptimeVal, ok := uptime.(int64); ok {
			uptimeSeconds = &uptimeVal
		}
	}

	var temperatureCelsius *float64
	if temp := data["temp"]; temp != nil {
		if tempVal, ok := temp.(float64); ok {
			temperatureCelsius = &tempVal
		}
	}

	var cpuLoad *float64
	if cpu := data["cpuLoad"]; cpu != nil {
		if cpuVal, ok := cpu.(float64); ok {
			cpuLoad = &cpuVal
		}
	}

	var memoryLoad *float64
	if mem := data["memLoad"]; mem != nil {
		if memVal, ok := mem.(float64); ok {
			memoryLoad = &memVal
		}
	}

	var bsConfig json.RawMessage
	if config := data["config"]; config != nil {
		if configBytes, err := json.Marshal(config); err == nil {
			bsConfig = json.RawMessage(configBytes)
		}
	}

	// Parse geoLocation optional field (BSSCI §3.5.2: Numeric[3] - [Latitude, Longitude, Altitude])
	var latitude, longitude, altitude *float64
	if geoLoc := data["geoLocation"]; geoLoc != nil {
		// geoLocation can be an array of 3 numeric values or a map/struct
		switch v := geoLoc.(type) {
		case []interface{}:
			if len(v) == 3 {
				if lat, ok := toFloat64(v[0]); ok {
					latitude = &lat
				}
				if lon, ok := toFloat64(v[1]); ok {
					longitude = &lon
				}
				if alt, ok := toFloat64(v[2]); ok {
					altitude = &alt
				}
			}
		case map[string]interface{}:
			// Handle struct format: {lat: ..., lon: ..., alt: ...}
			if lat, ok := getFloat64Field(v, "lat"); ok {
				latitude = &lat
			}
			if lon, ok := getFloat64Field(v, "lon"); ok {
				longitude = &lon
			}
			if alt, ok := getFloat64Field(v, "alt"); ok {
				altitude = &alt
			}
		}
	}

	// Store status data in database
	if s.basestationRepo != nil {
		euiStr := fmt.Sprintf("%016x", session.BaseStationEUI)
		euiBytes, err := hex.DecodeString(euiStr)
		if err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecode), err)
		}

		ctx := s.sessionContext(session)
		baseStation, err := s.basestationRepo.GetByEUI(ctx, resolvedTenant(session, s.tenantID), euiBytes)
		if err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToGetBaseStationByEUI,
				"bsEui", session.BaseStationEUI,
				"error", err)
		} else {
			updates := make(map[string]interface{})

			// Store mandatory fields
			updates["status_code"] = code
			updates["status_message"] = message
			updates["system_time"] = systemTime
			if dutyCycle != nil {
				updates["duty_cycle"] = *dutyCycle
			}
			if uptimeSeconds != nil {
				updates["uptime_seconds"] = *uptimeSeconds
			}
			if temperatureCelsius != nil {
				updates["temperature_celsius"] = *temperatureCelsius
			}
			if cpuLoad != nil {
				updates["cpu_load"] = *cpuLoad
			}
			if memoryLoad != nil {
				updates["memory_load"] = *memoryLoad
			}
			if len(bsConfig) > 0 {
				updates["bs_config"] = bsConfig
			}
			// Store geoLocation atomically: all three values must be present and valid (BSSCI §3.5.2).
			// Discard 0/0/0 — base stations without a GPS fix report all-zeros.
			if latitude != nil && longitude != nil && altitude != nil &&
				!(*latitude == 0 && *longitude == 0 && *altitude == 0) &&
				*latitude >= models.LatitudeMin && *latitude <= models.LatitudeMax &&
				*longitude >= models.LongitudeMin && *longitude <= models.LongitudeMax {
				updates["latitude"] = *latitude
				updates["longitude"] = *longitude
				updates["altitude"] = *altitude
				updates["location_source"] = models.LocationSourceGPS
				updates["location_updated_at"] = time.Now()
			}

			updates["last_status_at"] = time.Now()

			err = s.basestationRepo.Update(ctx, resolvedTenant(session, s.tenantID), baseStation.ID, updates)
			if err != nil {
				s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToUpdateBaseStationStatus,
					"bsEui", session.BaseStationEUI,
					"error", err)
			} else {
				s.logger.InfoContext(s.sessionContext(session), LogBSSCIUpdatedBaseStationStatusSuccessfully,
					"bsEui", session.BaseStationEUI,
					"fieldsUpdated", len(updates))

				// Persist status history (BSSCI §3.5.2 BSSCI-3.5-HIST)
				if s.storage != nil && s.storage.MIOTYBaseStationStatus() != nil {
					tenantID := resolvedTenant(session, s.tenantID)

					statusRecord := &mioty.BaseStationStatusRecord{
						TenantID:       tenantID,
						BaseStationID:  baseStation.ID,
						BasestationEUI: euiBytes,
						OperationID:    &msg.OpId,
						StatusCode:     int(code),
						StatusMessage:  message,
						SystemTime:     systemTime,
						DutyCycle:      dutyCycle,
						UptimeSeconds:  uptimeSeconds,
						Temperature:    temperatureCelsius,
						CPULoad:        cpuLoad,
						MemoryLoad:     memoryLoad,
						Config:         bsConfig,
						Latitude:       latitude,
						Longitude:      longitude,
						Altitude:       altitude,
					}
					if err := s.storage.MIOTYBaseStationStatus().Create(ctx, statusRecord); err != nil {
						s.logger.ErrorContext(ctx, LogBSSCIFailedToPersistStatusHistory, "error", err)
					}
				}
			}
		}
	}

	// The service center completes its own SC-initiated status operation
	// (BSSCI §3.5): after the base station's statusRsp, the SC sends statusCmp
	// and finalizes the pending operation. Because the SC sends the completion
	// itself, a spec-compliant base station never returns statusCmp, so the
	// pending row must be removed here or it leaks.
	complete := map[string]interface{}{
		"command": mioty.CmdStatusComplete,
		"opId":    msg.OpId,
	}
	if err := s.sendMessage(session, complete); err != nil {
		return err
	}
	// Finalize only after the completion write succeeded; a failed remove
	// preserves the pending operation for recovery.
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			"error", err, "opId", msg.OpId)
	}
	return nil
}

// getFloat64Field is now defined in server.go to avoid duplication

// startStatusMechanism starts periodic status requests to the base station
func (s *Server) startStatusMechanism(session *Session) {
	session.mu.Lock()
	// Guard against multiple goroutines
	if session.stopStatus != nil {
		session.mu.Unlock()
		s.logger.DebugContext(s.sessionContext(session), LogBSSCIStatusMechanismAlreadyRunningForSession,
			"sessionId", session.ID)
		return
	}

	stopStatus := make(chan struct{})
	session.stopStatus = stopStatus
	session.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.statusRequestInterval())
		defer ticker.Stop()

		time.Sleep(s.statusRequestInitialDelay())
		if _, err := s.SendStatusRequest(session); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIInitialStatusRequestFailed,
				"bsEui", session.BaseStationEUI,
				"error", err)
		}

		for {
			select {
			case <-ticker.C:
				if _, err := s.SendStatusRequest(session); err != nil {
					s.logger.ErrorContext(s.sessionContext(session), LogBSSCIStatusRequestFailedInPeriodicLoop,
						"bsEui", session.BaseStationEUI,
						"error", err)
				}
			case <-stopStatus:
				s.logger.InfoContext(s.sessionContext(session), LogBSSCIStoppingStatusMechanism,
					"bsEui", session.BaseStationEUI)
				return
			}
		}
	}()
}

// SendStatusRequest sends a status request to the base station
// This method is exported for use by the gRPC service
func (s *Server) SendStatusRequest(session interface{}) (int64, error) {
	// Type assert the session interface back to *Session
	sess, ok := session.(*Session)
	if !ok || sess == nil {
		return 0, fmt.Errorf("%s", ResolveErrorMessage(errInvalidSessionType))
	}

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	sess.mu.Lock()
	sess.LastScOpId--
	opId := sess.LastScOpId
	sess.mu.Unlock()

	statusRequest := map[string]interface{}{
		"command": mioty.CmdStatus,
		"opId":    opId,
	}

	s.logger.DebugContext(s.sessionContext(sess), LogBSSCISendingStatusRequestToBaseStation,
		"bsEui", sess.BaseStationEUI,
		"opId", opId)

	// The recovery record must be durable before the operation goes on the
	// wire: an SC operation whose pending row was never persisted cannot be
	// reissued on resume. A persistence failure rolls the operation ID back
	// and aborts the send.
	if err := s.persistPendingOperation(sess, opId, mioty.CmdStatus, statusRequest, nil, nil); err != nil {
		s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToPersistPendingStatusOperation, "error", err)
		sess.mu.Lock()
		sess.LastScOpId++
		sess.mu.Unlock()
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistPendingStatusOperation), err)
	}

	// Send message with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(sess, statusRequest); err != nil {
		// CRITICAL: Rollback operation ID on send failure
		sess.mu.Lock()
		sess.LastScOpId++
		sess.mu.Unlock()

		s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToSendStatusRequest,
			"bsEui", sess.BaseStationEUI,
			"error", err)
		// Clean up pending operation since sending failed
		if cleanupErr := s.removePendingOperation(sess, opId); cleanupErr != nil {
			s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToCleanupPendingOpAfterSendFailure,
				"error", cleanupErr,
				"opId", opId)
		}
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendStatusRequest), err)
	}

	// Success - persist counter to DB for session resume
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(sess), sess); err != nil {
		s.logger.ErrorContext(s.sessionContext(sess), LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", sess.ID,
			"opId", opId)
		// Don't fail the operation - message was sent successfully
	}

	return opId, nil
}
