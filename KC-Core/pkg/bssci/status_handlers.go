package bssci

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// statusReport is a base station's statusRsp (BSSCI v1.0.0 §3.5.2): the
// mandatory code, message and time, and the optional readings it reported.
type statusReport struct {
	code          int64
	message       string
	systemTime    int64
	dutyCycle     *float64
	uptimeSeconds *int64
	temperature   *float64
	cpuLoad       *float64
	memoryLoad    *float64
	config        json.RawMessage
	geo           geoReport
	fix           geoFix
}

// handleStatusResponse handles statusRsp from base station per MIOTY BSSCI v1.0.0 Section 3.5.2
func (s *Server) handleStatusResponse(session *Session, msg *Message, data map[string]interface{}) error {
	answeredAt := s.clock.Now()
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedStatusRspFromBaseStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldData, data)

	report, err := s.readStatusReport(session, msg.OpId, data)
	if err != nil {
		return err
	}
	if err := s.persistStatusReport(session, msg.OpId, report); err != nil {
		return err
	}
	s.announceStatusAnswer(session, msg.OpId, answeredAt)
	return s.completeStatusOperation(session, msg.OpId)
}

// readStatusReport validates the mandatory fields (BSSCI §3.17) of a payload
// handleMessage has normalized, refusing the operation on the first one
// missing or mistyped, and reads the optional ones.
func (s *Server) readStatusReport(session *Session, opID int64, data map[string]interface{}) (statusReport, error) {
	code, err := s.statusInteger(session, opID, data["code"], errMissingStatusCode, errFmtCodeFieldUnexpectedType)
	if err != nil {
		return statusReport{}, err
	}
	message, err := s.statusMessage(session, opID, data["message"])
	if err != nil {
		return statusReport{}, err
	}
	systemTime, err := s.statusInteger(session, opID, data["time"], errMissingStatusTime, errFmtTimeFieldUnexpectedType)
	if err != nil {
		return statusReport{}, err
	}
	report := statusReport{
		code:          code,
		message:       message,
		systemTime:    systemTime,
		dutyCycle:     optionalReading[float64](data, "dutyCycle"),
		uptimeSeconds: optionalReading[int64](data, "uptime"),
		temperature:   optionalReading[float64](data, "temp"),
		cpuLoad:       optionalReading[float64](data, "cpuLoad"),
		memoryLoad:    optionalReading[float64](data, "memLoad"),
		config:        optionalConfig(data["config"]),
	}
	report.fix, report.geo = parseGeoReport(data["geoLocation"])
	return report, nil
}

// statusInteger reads a mandatory integer field; msgpack decodes it as int or int64.
func (s *Server) statusInteger(session *Session, opID int64, value interface{}, missing, wrongTypeFmt string) (int64, error) {
	switch v := value.(type) {
	case nil:
		return 0, s.refuseStatus(session, opID, ResolveErrorMessage(missing))
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, s.refuseStatus(session, opID, fmt.Sprintf(wrongTypeFmt, ResolveErrorMessage(errInvalidFieldType), v))
	}
}

// statusMessage reads the mandatory message field.
func (s *Server) statusMessage(session *Session, opID int64, value interface{}) (string, error) {
	if value == nil {
		return "", s.refuseStatus(session, opID, ResolveErrorMessage(errMissingStatusMessage))
	}
	message, ok := value.(string)
	if !ok {
		return "", s.refuseStatus(session, opID, fmt.Sprintf(errFmtMessageFieldUnexpectedType, ResolveErrorMessage(errInvalidFieldType), value))
	}
	return message, nil
}

// refuseStatus answers a malformed statusRsp with a protocol error (BSSCI
// §2.4); the refusal is returned whether or not the error frame was written.
func (s *Server) refuseStatus(session *Session, opID int64, errMsg string) error {
	if err := s.sendError(session, opID, POSIX_EPROTO, errMsg); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorResponse, logger.FieldError, err)
	}
	return fmt.Errorf("%s", errMsg)
}

// optionalReading is an optional field of the reported type, nil when absent or of another type.
func optionalReading[T float64 | int64](data map[string]interface{}, key string) *T {
	value, ok := data[key].(T)
	if !ok {
		return nil
	}
	return &value
}

// optionalConfig is the reported base station configuration as JSON, nil when absent.
func optionalConfig(config interface{}) json.RawMessage {
	if config == nil {
		return nil
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil
	}
	return encoded
}

// persistStatusReport stores the report on the station and in its status
// history; a failed store is logged and never refuses the station's answer.
func (s *Server) persistStatusReport(session *Session, opID int64, report statusReport) error {
	if s.basestationRepo == nil {
		return nil
	}
	euiBytes, err := hex.DecodeString(mioty.FormatEUI64(session.BaseStationEUI))
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecode), err)
	}

	ctx := s.sessionContext(session)
	tenantID := resolvedTenant(session, s.tenantID)
	baseStation, err := s.basestationRepo.GetByEUI(ctx, tenantID, euiBytes)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToGetBaseStationByEUI,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
		return nil
	}

	updates := s.statusUpdates(report, baseStation)
	if err := s.basestationRepo.Update(ctx, tenantID, baseStation.ID, updates); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateBaseStationStatus,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
		return nil
	}
	s.logger.InfoContext(ctx, LogBSSCIUpdatedBaseStationStatusSuccessfully,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldFieldsUpdated, len(updates))
	s.recordStatusHistory(ctx, tenantID, baseStation.ID, euiBytes, opID, report)
	return nil
}

// statusUpdates are the base station columns the report writes.
func (s *Server) statusUpdates(report statusReport, baseStation *models.BaseStation) map[string]interface{} {
	updates := map[string]interface{}{
		"status_code":    report.code,
		"status_message": report.message,
		"system_time":    report.systemTime,
	}
	setReading(updates, "duty_cycle", report.dutyCycle)
	setReading(updates, "uptime_seconds", report.uptimeSeconds)
	setReading(updates, "temperature_celsius", report.temperature)
	setReading(updates, "cpu_load", report.cpuLoad)
	setReading(updates, "memory_load", report.memoryLoad)
	if len(report.config) > 0 {
		updates["bs_config"] = report.config
	}
	now := s.clock.Now()
	for column, value := range statusLocationUpdates(report.geo, report.fix, baseStation, now) {
		updates[column] = value
	}
	updates["last_status_at"] = now
	return updates
}

// setReading writes a reported reading's column; an unreported one keeps its stored value.
func setReading[T float64 | int64](updates map[string]interface{}, column string, value *T) {
	if value != nil {
		updates[column] = *value
	}
}

// recordStatusHistory persists the report in the station's status history (BSSCI §3.5.2 BSSCI-3.5-HIST).
func (s *Server) recordStatusHistory(ctx context.Context, tenantID, baseStationID int64, euiBytes []byte, opID int64, report statusReport) {
	if s.bsStatusStore == nil {
		return
	}
	record := &mioty.BaseStationStatusRecord{
		TenantID:       tenantID,
		BaseStationID:  baseStationID,
		BasestationEUI: euiBytes,
		OperationID:    &opID,
		StatusCode:     int(report.code),
		StatusMessage:  report.message,
		SystemTime:     report.systemTime,
		DutyCycle:      report.dutyCycle,
		UptimeSeconds:  report.uptimeSeconds,
		Temperature:    report.temperature,
		CPULoad:        report.cpuLoad,
		MemoryLoad:     report.memoryLoad,
		Config:         report.config,
	}
	if report.geo == geoFixed {
		record.Latitude, record.Longitude, record.Altitude = &report.fix.latitude, &report.fix.longitude, &report.fix.altitude
	}
	if err := s.bsStatusStore.Create(ctx, record); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToPersistStatusHistory, logger.FieldError, err)
	}
}

// completeStatusOperation sends statusCmp and finalizes the pending operation
// (BSSCI §3.5). The service center completes its own operation, so a
// spec-compliant base station never returns statusCmp and the pending row
// must be removed here or it leaks; a failed remove keeps it for recovery.
func (s *Server) completeStatusOperation(session *Session, opID int64) error {
	complete := map[string]interface{}{
		"command": mioty.CmdStatusComplete,
		"opId":    opID,
	}
	if err := s.sendMessage(session, complete); err != nil {
		return err
	}
	if err := s.pendingOps.remove(s.sessionContext(session), session, opID); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			logger.FieldError, err, logger.FieldOpID, opID)
	}
	return nil
}
