package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// The detach and propagate records share the messages table with the
// uplinks; their operation-specific fields live in the subpackets JSONB.
const (
	sqlInsertDetachMessage = `
		INSERT INTO messages (
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			rx_time, rx_duration, packet_cnt, snr, rssi, eq_snr,
			subpackets,
			dl_open, response_exp, dl_ack,
			received_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, false, false, false, $15, $16)`

	sqlInsertAttachPropagateMessage = `
		INSERT INTO messages (
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			nwk_sn_key, subpackets,
			rx_time, packet_cnt, snr, rssi,
			dl_open, response_exp, dl_ack,
			received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0, 0, 0, 0, false, false, false, $10)`

	sqlInsertDetachPropagateMessage = `
		INSERT INTO messages (
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			subpackets,
			rx_time, packet_cnt, snr, rssi,
			dl_open, response_exp, dl_ack,
			received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, 0, 0, 0, false, false, false, $9)`
)

// stampRecord fills in what a stored record lacks: its id, when it was
// received and which command it records.
func (r *MessageRepository) stampRecord(id *string, receivedAt *time.Time, command *string, defaultCommand string) {
	if *id == "" {
		*id = uuid.New().String()
	}
	if receivedAt.IsZero() {
		*receivedAt = r.clock.Now()
	}
	if *command == "" {
		*command = defaultCommand
	}
}

// storedEUI renders EUI bytes for a BYTEA column; anything but one EUI's
// worth of bytes is stored as the zero EUI.
func storedEUI(eui []byte) []byte {
	return mioty.EUI64Bytes(mioty.EUI64FromBytes(eui))
}

// CreateDetachMessage stores a detach operation (BSSCI §5.7) with its
// structured message.
func (r *MessageRepository) CreateDetachMessage(ctx context.Context, msg *mioty.DetachMessage, structuredMsg map[string]interface{}) error {
	r.stampRecord(&msg.ID, &msg.ReceivedAt, &msg.CommandType, mioty.CmdDetach)
	subpackets, err := jsonbText(true, detachSubpackets(msg, structuredMsg), errWrapMarshalSubpackets)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, sqlInsertDetachMessage,
		msg.ID, msg.TenantID, msg.OrgUUID, msg.CommandType, msg.OpId, storedEUI(msg.EpEui), storedEUI(msg.BasestationEui),
		msg.RxTime, msg.RxDuration, msg.PacketCnt, msg.SNR, msg.RSSI, msg.EqSnr,
		subpackets, msg.ReceivedAt, msg.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInsertDetachMessage, err)
	}
	return nil
}

// detachSubpackets is the subpackets document of a detach record: its
// validation context, the radio subpackets and the structured message.
func detachSubpackets(msg *mioty.DetachMessage, structuredMsg map[string]interface{}) map[string]interface{} {
	document := map[string]interface{}{
		"signature":        msg.Signature,
		"profile":          msg.Profile,
		"messageType":      msg.MessageType,
		"direction":        msg.Direction,
		"interfaceType":    msg.InterfaceType,
		"validationStatus": msg.ValidationStatus,
	}
	if msg.Subpackets != nil {
		document["snr"] = msg.Subpackets.SNR
		document["rssi"] = msg.Subpackets.RSSI
		document["frequency"] = msg.Subpackets.Frequency
		document["phase"] = msg.Subpackets.Phase
	}
	if structuredMsg != nil {
		document["rawMessage"] = structuredMsg
	}
	return document
}

// CreateAttachPropagateMessage stores an attach propagate operation
// (BSSCI §5.8) with the endpoint parameters it propagated (§5.8.1).
func (r *MessageRepository) CreateAttachPropagateMessage(ctx context.Context, msg *mioty.AttachPropagateMessage) error {
	r.stampRecord(&msg.ID, &msg.ReceivedAt, &msg.CommandType, mioty.CmdAttachPropagate)
	subpackets, err := jsonbText(true, map[string]interface{}{
		"shAddr":        msg.ShAddr,
		"bidi":          msg.Bidi,
		"lastPacketCnt": msg.LastPacketCnt,
		"dualChan":      msg.DualChan,
		"repetition":    msg.Repetition,
		"wideCarrOff":   msg.WideCarrOff,
		"longBlkDist":   msg.LongBlkDist,
	}, errWrapFailedToMarshalSubpackets)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, sqlInsertAttachPropagateMessage,
		msg.ID, msg.TenantID, msg.OrgUUID, msg.CommandType, msg.OpId, mioty.EUI64Bytes(msg.EpEui), storedEUI(msg.BasestationEui),
		msg.NwkSnKey, subpackets, msg.ReceivedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInsertAttachPropagateMessage, err)
	}
	return nil
}

// CreateDetachPropagateMessage stores a detach propagate operation (BSSCI §5.9).
func (r *MessageRepository) CreateDetachPropagateMessage(ctx context.Context, msg *mioty.DetachPropagateMessage) error {
	r.stampRecord(&msg.ID, &msg.ReceivedAt, &msg.CommandType, mioty.CmdDetachPropagate)
	subpackets, err := jsonbText(true, map[string]interface{}{
		"messageType":   msg.MessageType,
		"direction":     msg.Direction,
		"interfaceType": msg.InterfaceType,
	}, errWrapMarshalSubpackets)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, sqlInsertDetachPropagateMessage,
		msg.ID, msg.TenantID, msg.OrgUUID, msg.CommandType, msg.OpId, mioty.EUI64Bytes(msg.EpEui), storedEUI(msg.BasestationEui),
		subpackets, msg.ReceivedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInsertDetachPropagateMessage, err)
	}
	return nil
}
