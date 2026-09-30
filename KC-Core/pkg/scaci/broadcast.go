package scaci

import (
	"context"
	"net"

	// Shared MIOTY helpers (FormatEUI64, EPStatus)
	// Organization resolver for propagation context
	// BSSCI §5.8-5.8.3 attach propagation contracts
	// Import neutral scheduler contracts

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/encoding"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// buildFallbackBaseStationReception creates a single-BS reception array from legacy ULDataMessage fields
// Deep-copies optional fields and sanitizes per SCACI §3.8.1
func buildFallbackBaseStationReception(data *mioty.ULDataMessage) []mioty.BaseStationReception {
	bs := mioty.BaseStationReception{
		BsEui:  data.BsEui,
		RxTime: data.RxTime,
		Snr:    data.SNR,
		Rssi:   data.RSSI,
	}

	if data.RxDuration != nil {
		copied := *data.RxDuration
		bs.RxDuration = &copied
	}
	if data.EqSnr != nil {
		copied := *data.EqSnr
		bs.EqSnr = &copied
	}
	if data.Profile != nil {
		copied := *data.Profile
		bs.Profile = &copied
	}
	if data.Mode != nil {
		copied := *data.Mode
		bs.Mode = &copied
	}
	if data.Subpackets != nil {
		bs.Subpackets = &mioty.Subpackets{
			SNR:       append([]float64{}, data.Subpackets.SNR...),
			RSSI:      append([]float64{}, data.Subpackets.RSSI...),
			Frequency: append([]int64{}, data.Subpackets.Frequency...),
			Phase:     append([]float64{}, data.Subpackets.Phase...),
		}
	}

	return []mioty.BaseStationReception{bs}
}

// wireUserData returns the userData to put on the wire. SCACI §3.8.1 makes
// userData mandatory and allows it to be empty (control-only telegrams are
// empty data, §3.8), so an absent payload is sent as an empty one.
func wireUserData(userData []byte) []byte {
	return append([]byte{}, userData...)
}

// BroadcastULData forwards the stored uplink data to every connected
// Application Center of the tenant (SCACI §3.8). A telegram is delivered once
// per session: a retried delivery reaches only the sessions that do not have
// it yet, so its SCACI duplicate flag reports a reused packet counter, not the
// receptions merged into the stored message nor a repeated attempt (§3.8.1).
//
// Returns:
//   - error: one entry per session that did not receive the uplink
func (s *Server) BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error {
	if data == nil {
		return errNilULDataMessage
	}
	if data.ID == "" {
		return errULDataMessageNotStored
	}

	baseStations := data.BaseStations
	if len(baseStations) == 0 {
		baseStations = buildFallbackBaseStationReception(data)
	}
	// Force responseExp=false when dlOpen=false per SCACI §3.8.1 semantics
	responseExp := data.ResponseExp && data.DlOpen
	userData := wireUserData(data.UserData)
	duplicate := data.PacketCntReused

	requestData := map[string]interface{}{
		MetadataKeyEpEui: mioty.FormatEUI64(data.EpEui),
		"packetCnt":      data.PacketCnt,
		"dlOpen":         data.DlOpen,
		"responseExp":    responseExp,
		"dlAck":          data.DlAck,
		"baseStations":   buildBaseStationsRequestData(baseStations),
		"duplicate":      duplicate,
		"userData":       encoding.EncodeUserData(userData),
	}
	if data.Format != nil {
		requestData["format"] = *data.Format
	}

	return s.broadcastSCOperation(ctx, tenantID, everyApplicationCenter, scOperation{
		command:   CmdULData,
		failedLog: LogSCACISendULDataFailed,
		record:    s.recordUplinkForResume(ctx, data.ID, requestData),
		send: func(conn net.Conn, session *Session, opId int64) error {
			return s.SendULData(conn, session, &ULData{
				BaseMessage:  BaseMessage{Command: CmdULData, OpId: opId},
				EpEui:        data.EpEui,
				BaseStations: baseStations,
				PacketCnt:    data.PacketCnt,
				UserData:     userData,
				Format:       data.Format,
				DlOpen:       data.DlOpen,
				ResponseExp:  responseExp,
				DlAck:        data.DlAck,
				Duplicate:    &duplicate,
			})
		},
	})
}

// BroadcastDLDataResult forwards a DL data result to queuer, the Application
// Center that queued the downlink (SCACI §3.12), under acQueID, the queue id
// it assigned (SCACI §3.12.1). A queuer whose session is held for resume
// receives it when it resumes; another organization's Application Center of
// the same acEui never does.
//
// Returns:
//   - error: one entry per session that did not receive the result
func (s *Server) BroadcastDLDataResult(ctx context.Context, queuer ApplicationCenter, acQueID uint64, result *mioty.DLDataResult) error {
	if result == nil {
		return errNilDLDataResult
	}
	return s.broadcastSCOperation(ctx, queuer.TenantID, applicationCenter(queuer), scOperation{
		command:   CmdDLDataResult,
		failedLog: LogSCACISendDLResultToACFailed,
		record:    s.recordForResume(ctx, CmdDLDataResult, dlDataResultRequestData(result, acQueID)),
		send: func(conn net.Conn, session *Session, opId int64) error {
			return s.SendDLDataResult(conn, session, &DLDataResult{
				BaseMessage: BaseMessage{Command: CmdDLDataResult, OpId: opId},
				EpEui:       result.EpEui,
				QueID:       acQueID,
				Result:      result.Result,
				TxTime:      result.TxTime,
				PacketCnt:   result.PacketCnt,
				BsEui:       result.BsEui,
			})
		},
	})
}

// EPStatusData holds the data for an EPStatus broadcast
// This struct is used as input to BroadcastEPStatus and mirrors the EPStatus message fields.
// BSSCI→SCACI integration uses this to pass attach/detach status data.
type EPStatusData struct {
	EpEui      uint64
	EpStatus   string            // EPStatusAttached or EPStatusDetached
	AttachCnt  *uint32           // OTA attach only
	Nonce      *mioty.Numeric4   // OTA attach only
	Sign       *mioty.Numeric4   // OTA attach/detach only
	Snr        *float64          // OTA attach/detach only
	Rssi       *float64          // OTA attach/detach only
	EqSnr      *float64          // Optional
	Subpackets *mioty.Subpackets // Optional
}

// BroadcastEPStatus forwards an endpoint status to every connected
// Application Center of the tenant (SCACI §3.13).
//
// Returns:
//   - error: one entry per session that did not receive the status
func (s *Server) BroadcastEPStatus(ctx context.Context, tenantID int64, data *EPStatusData) error {
	if data == nil {
		return errNilEPStatusData
	}

	return s.broadcastSCOperation(ctx, tenantID, everyApplicationCenter, scOperation{
		command:   CmdEPStatus,
		failedLog: LogSCACISendEPStatusFailed,
		record:    s.recordForResume(ctx, CmdEPStatus, epStatusRequestData(data)),
		send: func(conn net.Conn, session *Session, opId int64) error {
			return s.SendEPStatus(conn, session, &EPStatus{
				BaseMessage: BaseMessage{Command: CmdEPStatus, OpId: opId},
				EpEui:       data.EpEui,
				EpStatus:    data.EpStatus,
				AttachCnt:   data.AttachCnt,
				Nonce:       data.Nonce,
				Sign:        data.Sign,
				Snr:         data.Snr,
				Rssi:        data.Rssi,
				EqSnr:       data.EqSnr,
				Subpackets:  data.Subpackets,
			})
		},
	})
}
