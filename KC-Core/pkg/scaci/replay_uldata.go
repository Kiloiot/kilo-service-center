package scaci

import (
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/encoding"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// replayULData reconstructs and resends ulData from stored RequestData
// Storage format: map[string]interface{} with epEui as hex string (FormatEUI64)
func (s *Server) replayULData(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	data := op.RequestData

	epEui, ok := extractEUIFromJSON(data, MetadataKeyEpEui)
	if !ok {
		return errMissingStoredEpEui
	}
	userData, err := s.replayUserData(session, op)
	if err != nil {
		return err
	}

	packetCnt, _ := extractInt64FromJSON(data, "packetCnt")
	dlOpen, _ := data["dlOpen"].(bool)
	responseExp, _ := data["responseExp"].(bool)
	dlAck, _ := data["dlAck"].(bool)
	duplicate, _ := data["duplicate"].(bool)

	msg := ULData{
		BaseMessage: BaseMessage{
			Command: CmdULData,
			OpId:    op.OpId, // Preserve original opId for replay
		},
		EpEui:        epEui,
		BaseStations: s.replayReceptions(session, op.OpId, data),
		PacketCnt:    uint32(packetCnt), // #nosec G115 - SCACI §3.8 defines PacketCnt as 32-bit
		UserData:     userData,
		DlOpen:       dlOpen,
		ResponseExp:  responseExp,
		DlAck:        dlAck,
		Duplicate:    &duplicate,
	}

	// Extract format if present (stored as float64 from JSON, needs to be uint8)
	if formatVal, ok := extractInt64FromJSON(data, "format"); ok {
		formatU8 := uint8(formatVal) // #nosec G115 - SCACI §3.8 defines Format as 8-bit
		msg.Format = &formatU8
	}

	return s.SendULData(conn, session, &msg)
}

// replayUserData decodes the recorded userData of an ulData; a record that
// cannot be decoded is corrupt and is not reissued (SCACI §1).
func (s *Server) replayUserData(session *Session, op *models.SCACIOperation) ([]byte, error) {
	recorded, _ := op.RequestData["userData"].(string)
	if recorded == "" {
		return wireUserData(nil), nil
	}
	userData, err := encoding.DecodeUserData(recorded)
	if err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIReplayUserDataCorrupted,
			logger.FieldOpID, op.OpId, logger.FieldStoredValue, recorded, logger.FieldError, err)
		return nil, fmt.Errorf(errFmtDecodeUserDataForReplay, err)
	}
	return wireUserData(userData), nil
}

// replayReceptions restores the base station receptions of an ulData record.
func (s *Server) replayReceptions(session *Session, opId int64, data map[string]interface{}) []mioty.BaseStationReception {
	recorded, _ := data["baseStations"].([]interface{})
	var baseStations []mioty.BaseStationReception
	for _, item := range recorded {
		if reception, ok := item.(map[string]interface{}); ok {
			baseStations = append(baseStations, s.replayReception(session, opId, reception))
		}
	}
	return baseStations
}

// replayReception restores one base station reception as
// buildBaseStationsRequestData recorded it (SCACI §3.8.1). Subpackets are
// optional, so a record whose subpackets cannot be decoded is replayed without
// them, as an EPStatus replay does.
func (s *Server) replayReception(session *Session, opId int64, recorded map[string]interface{}) mioty.BaseStationReception {
	bs := mioty.BaseStationReception{}
	bs.BsEui, _ = extractEUIFromJSON(recorded, metadataKeyBsEui)
	bs.RxTime, _ = extractInt64FromJSON(recorded, "rxTime")
	bs.Snr, _ = recorded["snr"].(float64)
	bs.Rssi, _ = recorded["rssi"].(float64)
	if rxDuration, ok := extractInt64FromJSON(recorded, "rxDuration"); ok {
		bs.RxDuration = &rxDuration
	}
	bs.EqSnr = recordedFloat(recorded, "eqSnr")
	bs.DlRxSnr = recordedFloat(recorded, "dlRxSnr")
	bs.DlRxRssi = recordedFloat(recorded, "dlRxRssi")
	bs.Profile = recordedString(recorded, "profile")
	bs.Mode = recordedString(recorded, "mode")
	if raw, ok := recorded[metadataKeySubpackets]; ok && raw != nil {
		subpackets, err := reconstructSubpackets(raw)
		if err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIReplayULDataFieldDecodeErr,
				logger.FieldOpID, opId, logger.FieldField, metadataKeySubpackets, logger.FieldError, err)
		}
		bs.Subpackets = subpackets
	}
	return bs
}

// recordedFloat returns an optional number of a recorded reception.
func recordedFloat(recorded map[string]interface{}, key string) *float64 {
	if v, ok := recorded[key].(float64); ok {
		return &v
	}
	return nil
}

// recordedString returns an optional text of a recorded reception.
func recordedString(recorded map[string]interface{}, key string) *string {
	if v, ok := recorded[key].(string); ok {
		return &v
	}
	return nil
}
