package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// sqlInsertULData stores one ulData row; the owning tenant fills both tenant columns.
const sqlInsertULData = `
	INSERT INTO messages (
		id, tenant_id, owner_tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
		rx_time, packet_cnt, snr, rssi, user_data,
		dl_open, response_exp, dl_ack,
		rx_duration, eq_snr, profile, mode, format, subpackets,
		base_stations, duplicate, packet_cnt_reused,
		received_at, processed_at,
		decoded_payload, blueprint_type_eui, blueprint_version_id, decode_status, decode_error_code
	) VALUES (
		$1, $2, $2, $3, $4, $5, $6, $7,
		$8, $9, $10, $11, $12,
		$13, $14, $15,
		$16, $17, $18, $19, $20, $21,
		$22, $23, $24,
		$25, $26,
		$27, $28, $29, $30, $31
	)`

// decodedPayloadPrefixBytes bounds the bytes of an undecodable decoded payload logged for diagnosis.
const decodedPayloadPrefixBytes = 64

// insertULDataMessage writes one uplink row; the message's tenant is the owner and fills both tenant columns.
func insertULDataMessage(ctx context.Context, exec uplinkExecutor, log logger.Logger, msg *mioty.ULDataMessage, now time.Time) error {
	if msg.TenantID <= 0 {
		return fmt.Errorf(errFmtInsertULDataMessageGot, storage.ErrInvalidTenantID, msg.TenantID)
	}
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = now
	}
	subpackets, err := jsonbText(msg.Subpackets != nil, msg.Subpackets, errWrapSerializeSubpackets)
	if err != nil {
		return err
	}
	baseStations, err := jsonbText(len(msg.BaseStations) > 0, msg.BaseStations, errWrapSerializeBaseStations)
	if err != nil {
		return err
	}
	decoded := decodedPayloadColumns(log, msg)
	var format uint8 // BSSCI §5.10.1: an absent format is 0
	if msg.Format != nil {
		format = *msg.Format
	}
	duplicate := msg.Duplicate != nil && *msg.Duplicate
	var blueprintTypeEUI interface{} // NULL or exactly one EUI per the column's check constraint
	if len(msg.BlueprintTypeEUI) == dbconfig.EUISize {
		blueprintTypeEUI = msg.BlueprintTypeEUI
	}
	_, err = exec.ExecContext(ctx, sqlInsertULData,
		msg.ID, msg.TenantID, msg.OrgUUID, msg.CommandType, msg.OpId, mioty.EUI64Bytes(msg.EpEui), mioty.EUI64Bytes(msg.BsEui),
		msg.RxTime, msg.PacketCnt, msg.SNR, msg.RSSI, msg.UserData,
		msg.DlOpen, msg.ResponseExp, msg.DlAck,
		msg.RxDuration, msg.EqSnr, msg.Profile, msg.Mode, format, subpackets,
		baseStations, duplicate, msg.PacketCntReused,
		msg.ReceivedAt, msg.ProcessedAt,
		decoded.payload, blueprintTypeEUI, msg.BlueprintVersionID, decoded.status, decoded.errorCode,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInsertULDataMessage, err)
	}
	return nil
}

// jsonbText renders a JSONB column value as text: lib/pq sends []byte in
// binary format, which PostgreSQL rejects for JSONB, while *string goes as
// text. An absent value is SQL NULL.
func jsonbText(present bool, value interface{}, errWrap string) (*string, error) {
	if !present {
		return nil, nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrap, err)
	}
	s := string(b)
	return &s, nil
}

// decodedColumns are the blueprint decode columns of an uplink row.
type decodedColumns struct {
	payload           *string
	status, errorCode string
}

// decodedPayloadColumns renders the decoded payload as JSONB text; a payload
// that is not JSON is stored as NULL with a failed decode status, so a
// success status never comes without its payload.
func decodedPayloadColumns(log logger.Logger, msg *mioty.ULDataMessage) decodedColumns {
	columns := decodedColumns{status: msg.DecodeStatus, errorCode: msg.DecodeErrorCode}
	if len(msg.DecodedPayload) == 0 {
		return columns
	}
	if json.Valid(msg.DecodedPayload) {
		s := string(msg.DecodedPayload)
		columns.payload = &s
		return columns
	}
	prefix := msg.DecodedPayload[:min(len(msg.DecodedPayload), decodedPayloadPrefixBytes)]
	log.Warn(logMsgDecodedPayloadInvalidJSON,
		logger.FieldEpEui, msg.EpEui, logger.FieldOpID, msg.OpId, logger.FieldDecodeStatus, msg.DecodeStatus,
		logger.FieldLength, len(msg.DecodedPayload), logger.FieldPayloadPrefix, prefix)
	columns.status = mioty.DecodeStatusFailed
	columns.errorCode = mioty.ErrInvalidJSONPayload
	return columns
}

// updateEndpointFromULData refreshes the endpoint's last-seen radio state;
// a missing endpoint row is an error because a message must never outlive
// the registration it belongs to. Only an over-the-air attach resets the
// counter (radio protocol §3.6.5.3), so a late telegram never rewinds it.
func updateEndpointFromULData(ctx context.Context, exec uplinkExecutor, msg *mioty.ULDataMessage) error {
	query := `
		UPDATE endpoints
		SET
			last_seen_at = NOW(),
			last_rx_time = $1,
			last_rx_duration = $2,
			packet_cnt = GREATEST(packet_cnt, $3),
			last_packet_cnt = GREATEST(last_packet_cnt, $3),
			last_snr = $4,
			last_rssi = $5,
			last_eq_snr = $6,
			last_profile = $7,
			last_mode = $8,
			last_user_data = $9,
			last_format_id = $10,
			last_dl_open = $11,
			last_response_exp = $12,
			last_dl_ack = $13,
			updated_at = NOW()
		WHERE ep_eui = $14 AND owner_tenant_id = $15`

	result, err := exec.ExecContext(
		ctx, query,
		msg.RxTime, msg.RxDuration, msg.PacketCnt,
		msg.SNR, msg.RSSI, msg.EqSnr,
		msg.Profile, msg.Mode, msg.UserData, msg.Format,
		msg.DlOpen, msg.ResponseExp, msg.DlAck,
		mioty.EUI64Bytes(msg.EpEui), msg.TenantID,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateEndpointFromULData, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", errWrapUpdateEndpointFromULData, storage.ErrNotFound)
	}

	return nil
}
