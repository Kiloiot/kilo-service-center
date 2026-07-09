package postgres

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MessageRepository implements message storage operations
// MessageRepository implements interfaces.MIOTYMessageRepository
type MessageRepository struct {
	db *sqlx.DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// euiToBytes encodes a uint64 EUI-64 as 8 big-endian bytes for BYTEA columns.
func euiToBytes(eui uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, eui)
	return b
}

// euiFromBytes decodes an 8-byte big-endian BYTEA EUI-64 into uint64 (0 if malformed).
func euiFromBytes(b []byte) uint64 {
	if len(b) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// CreateULDataMessage stores a new UL Data message in the database
func (r *MessageRepository) CreateULDataMessage(ctx context.Context, msg *mioty.ULDataMessage) error {
	// Validate tenant_id before any DB operation - defense against upstream bugs
	if msg.TenantID <= 0 {
		return fmt.Errorf("CreateULDataMessage: %w (got %d)", storage.ErrInvalidTenantID, msg.TenantID)
	}

	// Generate ID if not set
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Set timestamps
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = time.Now()
	}

	// Serialize subpackets to JSONB text format for lib/pq compatibility.
	// lib/pq sends []byte as binary format which PostgreSQL rejects for JSONB columns;
	// passing *string guarantees text format.
	var subpacketsStr *string
	if msg.Subpackets != nil {
		b, err := json.Marshal(msg.Subpackets)
		if err != nil {
			return fmt.Errorf("serialize subpackets: %w", err)
		}
		if json.Valid(b) {
			s := string(b)
			subpacketsStr = &s
		}
	}

	// Serialize base_stations to JSONB text format per SCACI §3.8.1
	var baseStationsStr *string
	if len(msg.BaseStations) > 0 {
		b, err := json.Marshal(msg.BaseStations)
		if err != nil {
			return fmt.Errorf("serialize base_stations: %w", err)
		}
		if json.Valid(b) {
			s := string(b)
			baseStationsStr = &s
		}
	}

	// Apply BSSCI §5.10.1 default: format defaults to 0 when absent
	var formatVal uint8
	if msg.Format != nil {
		formatVal = *msg.Format
	}

	// Get owner_tenant_id from endpoint for roaming support.
	// Convert uint64 EUI to 8-byte big-endian for bytea WHERE clause.
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, msg.EpEui)

	var ownerTenantID int64
	err := r.db.GetContext(ctx, &ownerTenantID,
		`SELECT COALESCE(owner_tenant_id, tenant_id) FROM endpoints WHERE ep_eui = $1 LIMIT 1`,
		epEuiBytes)
	if err != nil {
		// If endpoint not found, use the serving tenant as owner
		ownerTenantID = msg.TenantID
	}

	// Sanitize DecodedPayload and convert to *string for lib/pq JSONB text format.
	// PostgreSQL rejects []byte as binary format for JSONB columns;
	// passing *string guarantees text format. Empty/invalid JSON becomes SQL NULL.
	var decodedPayloadStr *string
	decodeStatus := msg.DecodeStatus
	decodeErrorCode := msg.DecodeErrorCode
	if len(msg.DecodedPayload) > 0 {
		if json.Valid(msg.DecodedPayload) {
			s := string(msg.DecodedPayload)
			decodedPayloadStr = &s
		} else {
			// Log metadata + hex prefix for debugging, store as NULL
			hexPrefix := msg.DecodedPayload
			if len(hexPrefix) > 64 {
				hexPrefix = hexPrefix[:64]
			}
			log.Printf("WARNING: decoded_payload invalid JSON - storing NULL. "+
				"ep_eui=%d op_id=%d decode_status=%s len=%d hex_prefix=%x",
				msg.EpEui, msg.OpId, msg.DecodeStatus, len(msg.DecodedPayload), hexPrefix)
			// Normalize status to prevent inconsistency (success status with NULL payload)
			decodeStatus = mioty.DecodeStatusFailed
			decodeErrorCode = mioty.ErrInvalidJSONPayload
		}
	}

	// Resolve duplicate flag: first reception is always false
	duplicateVal := false
	if msg.Duplicate != nil {
		duplicateVal = *msg.Duplicate
	}

	query := `
		INSERT INTO messages (
			id, tenant_id, owner_tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			rx_time, packet_cnt, snr, rssi, user_data,
			dl_open, response_exp, dl_ack,
			rx_duration, eq_snr, profile, mode, format, subpackets,
			base_stations, duplicate,
			received_at, processed_at,
			decoded_payload, blueprint_type_eui, blueprint_version_id, decode_status, decode_error_code
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16,
			$17, $18, $19, $20, $21, $22,
			$23, $24,
			$25, $26,
			$27, $28, $29, $30, $31
		)`

	// blueprint_type_eui must be NULL or exactly 8 bytes per DB check constraint
	var blueprintTypeEUIParam interface{}
	if len(msg.BlueprintTypeEUI) == 8 {
		blueprintTypeEUIParam = msg.BlueprintTypeEUI
	}

	_, err = r.db.ExecContext(ctx, query,
		msg.ID, msg.TenantID, ownerTenantID, msg.OrgUUID, msg.CommandType, msg.OpId, euiToBytes(msg.EpEui), euiToBytes(msg.BsEui),
		msg.RxTime, msg.PacketCnt, msg.SNR, msg.RSSI, msg.UserData,
		msg.DlOpen, msg.ResponseExp, msg.DlAck,
		msg.RxDuration, msg.EqSnr, msg.Profile, msg.Mode, formatVal, subpacketsStr,
		baseStationsStr, duplicateVal,
		msg.ReceivedAt, msg.ProcessedAt,
		decodedPayloadStr, blueprintTypeEUIParam, msg.BlueprintVersionID, decodeStatus, decodeErrorCode,
	)

	if err != nil {
		return fmt.Errorf("insert UL data message: %w", err)
	}

	// Also update the endpoint's last seen fields
	err = r.updateEndpointFromULData(ctx, msg)
	if err != nil {
		log.Printf("Warning: failed to update endpoint from UL data: %v", err)
	}

	return nil
}

// UpdateULDataBaseStations merges multi-BS reception data on duplicate detection per SCACI §3.8.1
// Called when a duplicate message is received from another base station within the deduplication window.
// Updates the existing message row with the merged base_stations JSONB array.
func (r *MessageRepository) UpdateULDataBaseStations(
	ctx context.Context,
	tenantID int64,
	epEui uint64,
	packetCnt uint32,
	rxTimeMin int64,
	baseStationsJSON []byte,
) error {
	// Update the most recent matching message within the deduplication window
	// Uses subquery to find the specific row to update (Postgres-specific pattern)
	query := `
		UPDATE messages
		SET base_stations = $1,
			duplicate = true,
			updated_at = NOW()
		WHERE id = (
			SELECT id FROM messages
			WHERE tenant_id = $2
			  AND ep_eui = $3
			  AND packet_cnt = $4
			  AND rx_time >= $5
			ORDER BY rx_time DESC
			LIMIT 1
		)`

	// Convert to *string for lib/pq JSONB text format compatibility
	var baseStationsStr *string
	if len(baseStationsJSON) > 0 && json.Valid(baseStationsJSON) {
		s := string(baseStationsJSON)
		baseStationsStr = &s
	}

	result, err := r.db.ExecContext(ctx, query,
		baseStationsStr, tenantID, euiToBytes(epEui), packetCnt, rxTimeMin)
	if err != nil {
		return fmt.Errorf("update base_stations: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		// No row updated - this can happen at dedup window boundaries
		// Return sentinel error - caller logs with context (no logging in repo layer)
		return ErrNoMatchingMessage
	}
	return nil
}

// CreateAttachMessage stores a new attach message in the messages table
// Per BSSCI §5.6 Attach operation and messages schema (migration 047)
// Schema alignment: Retargeted from mioty_messages → messages per migration 047
func (r *MessageRepository) CreateAttachMessage(ctx context.Context, msg *mioty.AttachMessage, structuredMsg map[string]interface{}) error {
	// Generate UUID if not set (messages table uses UUID PRIMARY KEY)
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = now
	}

	// Set command type default
	if msg.CommandType == "" {
		msg.CommandType = mioty.CmdAttach
	}

	// Convert endpoint EUI bytes to uint64 for BIGINT column
	var epEui uint64
	if len(msg.EpEui) == 8 {
		epEui = binary.BigEndian.Uint64(msg.EpEui)
	}

	// Convert basestation EUI bytes to uint64 for BIGINT column
	var bsEui uint64
	if len(msg.BasestationEui) == 8 {
		bsEui = binary.BigEndian.Uint64(msg.BasestationEui)
	}

	// Build subpackets JSONB with attach-specific fields and structured message
	subpacketsMap := map[string]interface{}{
		"signature":     msg.Signature,
		"profile":       msg.Profile,
		"messageType":   msg.MessageType,
		"direction":     msg.Direction,
		"interfaceType": msg.InterfaceType,
	}
	// Merge in existing subpackets struct if present
	if msg.Subpackets != nil {
		subpacketsMap["snr"] = msg.Subpackets.SNR
		subpacketsMap["rssi"] = msg.Subpackets.RSSI
		subpacketsMap["frequency"] = msg.Subpackets.Frequency
		subpacketsMap["phase"] = msg.Subpackets.Phase
	}
	// Merge in structured message if present
	if structuredMsg != nil {
		subpacketsMap["rawMessage"] = structuredMsg
	}
	subpacketsBytes, err := json.Marshal(subpacketsMap)
	if err != nil {
		return fmt.Errorf("marshal subpackets: %w", err)
	}
	// Convert to *string for lib/pq JSONB text format compatibility
	subpacketsStr := string(subpacketsBytes)

	// Insert into messages table (canonical table per migration 047)
	// Column mapping: ep_eui/bs_eui as BIGINT, command_type, op_id, subpackets JSONB
	// Note: AttachMessage struct doesn't have OrgUUID - use nil
	query := `
		INSERT INTO messages (
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			rx_time, rx_duration, packet_cnt, snr, rssi, eq_snr,
			subpackets,
			dl_open, response_exp, dl_ack,
			received_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err = r.db.ExecContext(ctx, query,
		msg.ID, msg.TenantID, nil, msg.CommandType, msg.OpId, euiToBytes(epEui), euiToBytes(bsEui),
		msg.RxTime, msg.RxDuration, msg.PacketCnt, msg.SNR, msg.RSSI, msg.EqSnr,
		&subpacketsStr,
		false, false, false, // dl_open, response_exp, dl_ack - defaults for attach messages
		msg.ReceivedAt, msg.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("insert attach message: %w", err)
	}

	return nil
}

// CreateDetachMessage stores a new detach message in the messages table
// Per BSSCI §5.7 Detach operation and messages schema (migration 047)
// Schema alignment: Retargeted from mioty_messages → messages per migration 047
func (r *MessageRepository) CreateDetachMessage(ctx context.Context, msg *mioty.DetachMessage, structuredMsg map[string]interface{}) error {
	// Generate UUID if not set (messages table uses UUID PRIMARY KEY)
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = now
	}

	// Set command type default
	if msg.CommandType == "" {
		msg.CommandType = mioty.CmdDetach
	}

	// Convert endpoint EUI bytes to uint64 for BIGINT column
	var epEui uint64
	if len(msg.EpEui) == 8 {
		epEui = binary.BigEndian.Uint64(msg.EpEui)
	}

	// Convert basestation EUI bytes to uint64 for BIGINT column
	var bsEui uint64
	if len(msg.BasestationEui) == 8 {
		bsEui = binary.BigEndian.Uint64(msg.BasestationEui)
	}

	// Build subpackets JSONB with detach-specific fields and structured message
	subpacketsMap := map[string]interface{}{
		"signature":     msg.Signature,
		"profile":       msg.Profile,
		"messageType":   msg.MessageType,
		"direction":     msg.Direction,
		"interfaceType": msg.InterfaceType,
	}
	// Merge in existing subpackets struct if present
	if msg.Subpackets != nil {
		subpacketsMap["snr"] = msg.Subpackets.SNR
		subpacketsMap["rssi"] = msg.Subpackets.RSSI
		subpacketsMap["frequency"] = msg.Subpackets.Frequency
		subpacketsMap["phase"] = msg.Subpackets.Phase
	}
	// Merge in structured message if present
	if structuredMsg != nil {
		subpacketsMap["rawMessage"] = structuredMsg
	}
	subpacketsBytes, err := json.Marshal(subpacketsMap)
	if err != nil {
		return fmt.Errorf("marshal subpackets: %w", err)
	}
	// Convert to *string for lib/pq JSONB text format compatibility
	subpacketsStr := string(subpacketsBytes)

	// Insert into messages table (canonical table per migration 047)
	// Column mapping: ep_eui/bs_eui as BIGINT, command_type, op_id, subpackets JSONB
	query := `
		INSERT INTO messages (
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			rx_time, rx_duration, packet_cnt, snr, rssi, eq_snr,
			subpackets,
			dl_open, response_exp, dl_ack,
			received_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err = r.db.ExecContext(ctx, query,
		msg.ID, msg.TenantID, msg.OrgUUID, msg.CommandType, msg.OpId, euiToBytes(epEui), euiToBytes(bsEui),
		msg.RxTime, msg.RxDuration, msg.PacketCnt, msg.SNR, msg.RSSI, msg.EqSnr,
		&subpacketsStr,
		false, false, false, // dl_open, response_exp, dl_ack - defaults for detach messages
		msg.ReceivedAt, msg.ProcessedAt,
	)

	if err != nil {
		return fmt.Errorf("insert detach message: %w", err)
	}

	return nil
}

// CreateAttachPropagateMessage stores a new attach propagate message in the messages table
// Per BSSCI §5.8-5.8.3 Attach Propagate operation
// Uses canonical messages table (migration 047) with propagate-specific fields in subpackets JSONB
func (r *MessageRepository) CreateAttachPropagateMessage(ctx context.Context, msg *mioty.AttachPropagateMessage) error {
	// Generate UUID if not set (messages.id is UUID PRIMARY KEY DEFAULT gen_random_uuid())
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = now
	}

	// Set command type default
	if msg.CommandType == "" {
		msg.CommandType = mioty.CmdAttachPropagate
	}

	// Convert basestation EUI bytes to uint64 for BIGINT column
	var bsEui uint64
	if len(msg.BasestationEui) == 8 {
		bsEui = binary.BigEndian.Uint64(msg.BasestationEui)
	}

	// Build subpackets JSONB with propagate-specific fields per BSSCI §5.8.1
	subpackets := map[string]interface{}{
		"shAddr":        msg.ShAddr,
		"bidi":          msg.Bidi,
		"lastPacketCnt": msg.LastPacketCnt,
		"dualChan":      msg.DualChan,
		"repetition":    msg.Repetition,
		"wideCarrOff":   msg.WideCarrOff,
		"longBlkDist":   msg.LongBlkDist,
	}
	subpacketsBytes, err := json.Marshal(subpackets)
	if err != nil {
		return fmt.Errorf("failed to marshal subpackets: %w", err)
	}
	// Convert to *string for lib/pq JSONB text format compatibility
	subpacketsStr := string(subpacketsBytes)

	// Insert into messages table (canonical table per migration 047)
	// Uses messages schema: UUID id, BIGINT ep_eui/bs_eui, subpackets JSONB for propagate fields
	query := `
		INSERT INTO messages (
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			nwk_sn_key, subpackets,
			rx_time, packet_cnt, snr, rssi,
			dl_open, response_exp, dl_ack,
			received_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9,
			$10, $11, $12, $13,
			$14, $15, $16,
			$17
		)`

	_, err = r.db.ExecContext(ctx, query,
		msg.ID, msg.TenantID, msg.OrgUUID, msg.CommandType, msg.OpId, euiToBytes(msg.EpEui), euiToBytes(bsEui),
		msg.NwkSnKey, &subpacketsStr,
		0, 0, 0.0, 0.0, // rx_time, packet_cnt, snr, rssi - defaults for propagate messages
		false, false, false, // dl_open, response_exp, dl_ack - defaults for propagate messages
		msg.ReceivedAt,
	)
	if err != nil {
		return fmt.Errorf("insert attach propagate message: %w", err)
	}

	return nil
}

// CreateDetachPropagateMessage stores a new detach propagate message in the messages table
// Per BSSCI §5.9-5.9.3 Detach Propagate operation and messages schema (migration 047)
// Schema alignment: Retargeted from mioty_messages → messages per migration 047
func (r *MessageRepository) CreateDetachPropagateMessage(ctx context.Context, msg *mioty.DetachPropagateMessage) error {
	// Generate UUID if not set (messages table uses UUID PRIMARY KEY)
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = now
	}

	// Set command type default
	if msg.CommandType == "" {
		msg.CommandType = mioty.CmdDetachPropagate
	}

	// Convert basestation EUI bytes to uint64 for BIGINT column
	var bsEui uint64
	if len(msg.BasestationEui) == 8 {
		bsEui = binary.BigEndian.Uint64(msg.BasestationEui)
	}

	// Build subpackets JSONB with detach propagate-specific fields
	subpackets := map[string]interface{}{
		"messageType":   msg.MessageType,
		"direction":     msg.Direction,
		"interfaceType": msg.InterfaceType,
	}
	subpacketsBytes, err := json.Marshal(subpackets)
	if err != nil {
		return fmt.Errorf("marshal subpackets: %w", err)
	}
	// Convert to *string for lib/pq JSONB text format compatibility
	subpacketsStr := string(subpacketsBytes)

	// Insert into messages table (canonical table per migration 047)
	// Column mapping: ep_eui/bs_eui as BIGINT, command_type, op_id, subpackets JSONB
	query := `
		INSERT INTO messages (
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			subpackets,
			rx_time, packet_cnt, snr, rssi,
			dl_open, response_exp, dl_ack,
			received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err = r.db.ExecContext(ctx, query,
		msg.ID, msg.TenantID, msg.OrgUUID, msg.CommandType, msg.OpId, euiToBytes(msg.EpEui), euiToBytes(bsEui),
		&subpacketsStr,
		0, 0, 0.0, 0.0, // rx_time, packet_cnt, snr, rssi - defaults for propagate messages
		false, false, false, // dl_open, response_exp, dl_ack - defaults for propagate messages
		msg.ReceivedAt,
	)
	if err != nil {
		return fmt.Errorf("insert detach propagate message: %w", err)
	}

	return nil
}

// updateEndpointFromULData updates the endpoint table with latest UL Data information
func (r *MessageRepository) updateEndpointFromULData(ctx context.Context, msg *mioty.ULDataMessage) error {
	query := `
		UPDATE endpoints 
		SET 
			last_seen_at = NOW(),
			last_rx_time = $1,
			last_rx_duration = $2,
			packet_cnt = $3,
			last_packet_cnt = $3,
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
		WHERE ep_eui = $14 AND tenant_id = $15`

	// Convert uint64 EpEui to 8-byte big-endian for bytea column match
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, msg.EpEui)

	result, err := r.db.ExecContext(ctx, query,
		msg.RxTime, msg.RxDuration, msg.PacketCnt,
		msg.SNR, msg.RSSI, msg.EqSnr,
		msg.Profile, msg.Mode, msg.UserData, msg.Format,
		msg.DlOpen, msg.ResponseExp, msg.DlAck,
		epEuiBytes, msg.TenantID,
	)

	if err != nil {
		return fmt.Errorf("update endpoint from UL data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Endpoint doesn't exist - this is normal for new endpoints
		return nil
	}

	return nil
}

// GetULDataMessage retrieves a single UL Data message by ID
func (r *MessageRepository) GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error) {
	var msg mioty.ULDataMessage
	var subpacketsJSON []byte
	var baseStationsJSON []byte
	var duplicate bool
	// Use nullable types for columns that may be NULL
	var decodedPayloadJSON []byte
	var decodeStatus, decodeErrorCode sql.NullString

	query := `
		SELECT
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			rx_time, packet_cnt, snr, rssi, user_data,
			dl_open, response_exp, dl_ack,
			rx_duration, eq_snr, profile, mode, format, subpackets,
			base_stations, duplicate,
			received_at, processed_at,
			decoded_payload, blueprint_type_eui, blueprint_version_id, decode_status, decode_error_code
		FROM messages
		WHERE id = $1 AND tenant_id = $2`

	var epEuiB, bsEuiB []byte
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&msg.ID, &msg.TenantID, &msg.OrgUUID, &msg.CommandType, &msg.OpId, &epEuiB, &bsEuiB,
		&msg.RxTime, &msg.PacketCnt, &msg.SNR, &msg.RSSI, &msg.UserData,
		&msg.DlOpen, &msg.ResponseExp, &msg.DlAck,
		&msg.RxDuration, &msg.EqSnr, &msg.Profile, &msg.Mode, &msg.Format, &subpacketsJSON,
		&baseStationsJSON, &duplicate,
		&msg.ReceivedAt, &msg.ProcessedAt,
		&decodedPayloadJSON, &msg.BlueprintTypeEUI, &msg.BlueprintVersionID, &decodeStatus, &decodeErrorCode,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get UL data message: %w", err)
	}
	msg.EpEui = euiFromBytes(epEuiB)
	msg.BsEui = euiFromBytes(bsEuiB)

	// Assign nullable fields
	if decodedPayloadJSON != nil {
		msg.DecodedPayload = decodedPayloadJSON
	}
	if decodeStatus.Valid {
		msg.DecodeStatus = decodeStatus.String
	}
	if decodeErrorCode.Valid {
		msg.DecodeErrorCode = decodeErrorCode.String
	}

	// Deserialize subpackets if present
	if subpacketsJSON != nil {
		var subpackets mioty.Subpackets
		if err := json.Unmarshal(subpacketsJSON, &subpackets); err != nil {
			return nil, fmt.Errorf("deserialize subpackets: %w", err)
		}
		msg.Subpackets = &subpackets
	}

	// Deserialize base_stations JSONB if present
	if baseStationsJSON != nil {
		var baseStations []mioty.BaseStationReception
		if err := json.Unmarshal(baseStationsJSON, &baseStations); err != nil {
			return nil, fmt.Errorf("deserialize base_stations: %w", err)
		}
		msg.BaseStations = baseStations
	}
	msg.Duplicate = &duplicate

	return &msg, nil
}

// GetDetachMessage retrieves a specific detach message by ID and tenant
// FIX-5: Implements full detach message retrieval with org_uuid support
// Schema alignment: Retargeted from mioty_messages → messages per migration 047
func (r *MessageRepository) GetDetachMessage(ctx context.Context, id string, tenantID int64) (*mioty.DetachMessage, error) {
	var msg mioty.DetachMessage
	var subpacketsJSON []byte
	var epEuiB, bsEuiB []byte

	// Query from canonical messages table (migration 047)
	// Column mapping: op_id (not operation_id), bs_eui (not basestation_eui), ep_eui as BIGINT
	query := `
		SELECT
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			rx_time, packet_cnt, snr, rssi,
			rx_duration, eq_snr, subpackets,
			received_at, processed_at
		FROM messages
		WHERE id = $1 AND tenant_id = $2 AND command_type = $3`

	err := r.db.QueryRowContext(ctx, query, id, tenantID, mioty.CmdDetach).Scan(
		&msg.ID, &msg.TenantID, &msg.OrgUUID, &msg.CommandType, &msg.OpId, &epEuiB, &bsEuiB,
		&msg.RxTime, &msg.PacketCnt, &msg.SNR, &msg.RSSI,
		&msg.RxDuration, &msg.EqSnr, &subpacketsJSON,
		&msg.ReceivedAt, &msg.ProcessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("detach message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get detach message: %w", err)
	}

	// ep_eui / bs_eui stored as BYTEA(8)
	if len(epEuiB) == 8 {
		msg.EpEui = epEuiB
	}
	if len(bsEuiB) == 8 {
		msg.BasestationEui = bsEuiB
	}

	// Deserialize subpackets if present - extract detach-specific fields
	if subpacketsJSON != nil {
		var subpacketsMap map[string]interface{}
		if err := json.Unmarshal(subpacketsJSON, &subpacketsMap); err != nil {
			return nil, fmt.Errorf("deserialize subpackets: %w", err)
		}
		// Extract signature from subpackets (stored as []interface{} from JSON)
		if sig, ok := subpacketsMap["signature"].([]interface{}); ok {
			msg.Signature = make([]byte, len(sig))
			for i, v := range sig {
				if f, ok := v.(float64); ok {
					msg.Signature[i] = byte(f)
				}
			}
		}
		// Extract profile (stored as *string in struct)
		if profile, ok := subpacketsMap["profile"].(string); ok {
			msg.Profile = &profile
		}
		// Extract messageType, direction, interfaceType (all string types in struct)
		if mt, ok := subpacketsMap["messageType"].(string); ok {
			msg.MessageType = mt
		}
		if dir, ok := subpacketsMap["direction"].(string); ok {
			msg.Direction = dir
		}
		if it, ok := subpacketsMap["interfaceType"].(string); ok {
			msg.InterfaceType = it
		}
		// Build Subpackets struct for SNR/RSSI/Frequency/Phase arrays
		var subpackets mioty.Subpackets
		if snrArr, ok := subpacketsMap["snr"].([]interface{}); ok {
			subpackets.SNR = make([]float64, len(snrArr))
			for i, v := range snrArr {
				if f, ok := v.(float64); ok {
					subpackets.SNR[i] = f
				}
			}
		}
		if rssiArr, ok := subpacketsMap["rssi"].([]interface{}); ok {
			subpackets.RSSI = make([]float64, len(rssiArr))
			for i, v := range rssiArr {
				if f, ok := v.(float64); ok {
					subpackets.RSSI[i] = f
				}
			}
		}
		if freqArr, ok := subpacketsMap["frequency"].([]interface{}); ok {
			subpackets.Frequency = make([]int64, len(freqArr))
			for i, v := range freqArr {
				if f, ok := v.(float64); ok {
					subpackets.Frequency[i] = int64(f)
				}
			}
		}
		if phaseArr, ok := subpacketsMap["phase"].([]interface{}); ok {
			subpackets.Phase = make([]float64, len(phaseArr))
			for i, v := range phaseArr {
				if f, ok := v.(float64); ok {
					subpackets.Phase[i] = f
				}
			}
		}
		// Only set if any subpacket arrays were populated
		if len(subpackets.SNR) > 0 || len(subpackets.RSSI) > 0 ||
			len(subpackets.Frequency) > 0 || len(subpackets.Phase) > 0 {
			msg.Subpackets = &subpackets
		}
	}

	return &msg, nil
}

// ListULDataMessages retrieves messages with filtering and pagination
// Only returns actual uplink data messages (ulData), not propagate messages
func (r *MessageRepository) ListULDataMessages(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	// Build WHERE clause - filter for ulData only, exclude propagate messages
	whereClauses := []string{"tenant_id = $1", "command_type = $2"}
	args := []interface{}{filter.TenantID, mioty.CmdULData}
	argCount := 3

	if filter.EpEui != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("ep_eui = $%d", argCount))
		args = append(args, euiToBytes(*filter.EpEui))
		argCount++
	}

	if filter.BsEui != nil {
		// Include messages where BS is primary receiver OR secondary receiver in base_stations JSONB
		bsContainment := fmt.Sprintf(`[{"bsEui":%d}]`, *filter.BsEui)
		whereClauses = append(whereClauses, fmt.Sprintf("(bs_eui = $%d OR base_stations @> $%d::jsonb)", argCount, argCount+1))
		args = append(args, euiToBytes(*filter.BsEui), bsContainment)
		argCount += 2
	}

	if filter.StartTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rx_time >= $%d", argCount))
		args = append(args, filter.StartTime.UnixNano())
		argCount++
	}

	if filter.EndTime != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rx_time <= $%d", argCount))
		args = append(args, filter.EndTime.UnixNano())
		argCount++
	}

	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		// Search across ep_eui, bs_eui, and hex-encoded user_data (case-insensitive)
		searchClause := fmt.Sprintf("(encode(ep_eui, 'hex') ILIKE $%d OR encode(bs_eui, 'hex') ILIKE $%d OR encode(user_data, 'hex') ILIKE $%d)", argCount, argCount+1, argCount+2)
		whereClauses = append(whereClauses, searchClause)
		searchPattern := "%" + *filter.SearchTerm + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argCount += 3
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total matching records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM messages WHERE %s", whereSQL)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count messages: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
			rx_time, packet_cnt, snr, rssi, user_data,
			dl_open, response_exp, dl_ack,
			rx_duration, eq_snr, profile, mode, format, subpackets,
			base_stations, duplicate,
			received_at, processed_at,
			decoded_payload, blueprint_type_eui, blueprint_version_id, decode_status, decode_error_code
		FROM messages
		WHERE %s
		ORDER BY rx_time DESC
		LIMIT $%d OFFSET $%d`,
		whereSQL, argCount, argCount+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list messages: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in message repository: %v", err)
		}
	}()

	messages := make([]*mioty.ULDataMessage, 0)
	for rows.Next() {
		var msg mioty.ULDataMessage
		var subpacketsJSON []byte
		var baseStationsJSON []byte
		var duplicate bool
		// Use nullable types for columns that may be NULL
		var decodedPayloadJSON []byte
		var decodeStatus, decodeErrorCode sql.NullString

		var epEuiB, bsEuiB []byte
		err := rows.Scan(
			&msg.ID, &msg.TenantID, &msg.OrgUUID, &msg.CommandType, &msg.OpId, &epEuiB, &bsEuiB,
			&msg.RxTime, &msg.PacketCnt, &msg.SNR, &msg.RSSI, &msg.UserData,
			&msg.DlOpen, &msg.ResponseExp, &msg.DlAck,
			&msg.RxDuration, &msg.EqSnr, &msg.Profile, &msg.Mode, &msg.Format, &subpacketsJSON,
			&baseStationsJSON, &duplicate,
			&msg.ReceivedAt, &msg.ProcessedAt,
			&decodedPayloadJSON, &msg.BlueprintTypeEUI, &msg.BlueprintVersionID, &decodeStatus, &decodeErrorCode,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan message: %w", err)
		}
		msg.EpEui = euiFromBytes(epEuiB)
		msg.BsEui = euiFromBytes(bsEuiB)
		// Assign decoded payload if not NULL
		if decodedPayloadJSON != nil {
			msg.DecodedPayload = decodedPayloadJSON
		}
		// Assign nullable string fields
		if decodeStatus.Valid {
			msg.DecodeStatus = decodeStatus.String
		}
		if decodeErrorCode.Valid {
			msg.DecodeErrorCode = decodeErrorCode.String
		}

		// Deserialize base_stations JSONB if present
		if baseStationsJSON != nil {
			var baseStations []mioty.BaseStationReception
			if err := json.Unmarshal(baseStationsJSON, &baseStations); err != nil {
				fmt.Printf("Warning: failed to deserialize base_stations: %v\n", err)
			} else {
				msg.BaseStations = baseStations
			}
		}
		msg.Duplicate = &duplicate

		// Deserialize subpackets if present
		if subpacketsJSON != nil {
			var subpackets mioty.Subpackets
			if err := json.Unmarshal(subpacketsJSON, &subpackets); err != nil {
				// Log warning but continue
				fmt.Printf("Warning: failed to deserialize subpackets: %v\n", err)
			} else {
				msg.Subpackets = &subpackets
			}
		}

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, total, nil
}

// GetMessageStatsByBaseStation returns message statistics for a specific base station.
// Includes messages where the BS is a secondary receiver in base_stations JSONB.
func (r *MessageRepository) GetMessageStatsByBaseStation(ctx context.Context, bsEui uint64, tenantID int64) (*mioty.MessageStats, error) {
	query := fmt.Sprintf(`
		WITH matched AS (
			SELECT m.id, m.ep_eui,
				(elem->>'rssi')::double precision AS bs_rssi,
				(elem->>'snr')::double precision AS bs_snr
			FROM messages m,
				jsonb_array_elements(m.base_stations) AS elem
			WHERE m.tenant_id = $1
				AND (elem->>'bsEui') = '%d'
			UNION ALL
			SELECT id, ep_eui, rssi, snr
			FROM messages
			WHERE tenant_id = $1 AND bs_eui = $2
				AND (base_stations IS NULL OR jsonb_array_length(base_stations) = 0)
		)
		SELECT
			COUNT(*) as total_count,
			COUNT(DISTINCT ep_eui) as unique_endpoints,
			COALESCE(AVG(bs_rssi), 0) as avg_rssi,
			COALESCE(AVG(bs_snr), 0) as avg_snr
		FROM matched
	`, bsEui)

	var stats mioty.MessageStats
	err := r.db.QueryRowContext(ctx, query, tenantID, euiToBytes(bsEui)).Scan(
		&stats.TotalCount,
		&stats.UniqueEndpoints,
		&stats.AvgRSSI,
		&stats.AvgSNR,
	)

	if err != nil {
		return nil, fmt.Errorf("get message stats by base station: %w", err)
	}

	return &stats, nil
}

// GetExtendedMessageStatsByBaseStation returns comprehensive message statistics including min/max values and time ranges.
// Includes messages where the BS is a secondary receiver in base_stations JSONB.
func (r *MessageRepository) GetExtendedMessageStatsByBaseStation(ctx context.Context, bsEui uint64, tenantID int64) (*mioty.MessageStats, error) {
	query := fmt.Sprintf(`
		WITH matched AS (
			SELECT m.id, m.ep_eui, m.received_at,
				(elem->>'rssi')::double precision AS bs_rssi,
				(elem->>'snr')::double precision AS bs_snr
			FROM messages m,
				jsonb_array_elements(m.base_stations) AS elem
			WHERE m.tenant_id = $1
				AND (elem->>'bsEui') = '%d'
			UNION ALL
			SELECT id, ep_eui, received_at, rssi, snr
			FROM messages
			WHERE tenant_id = $1 AND bs_eui = $2
				AND (base_stations IS NULL OR jsonb_array_length(base_stations) = 0)
		)
		SELECT
			COUNT(*) as total_count,
			COUNT(DISTINCT ep_eui) as unique_endpoints,
			COALESCE(AVG(bs_rssi), 0) as avg_rssi,
			COALESCE(AVG(bs_snr), 0) as avg_snr,
			MIN(bs_rssi) as min_rssi,
			MAX(bs_rssi) as max_rssi,
			MIN(bs_snr) as min_snr,
			MAX(bs_snr) as max_snr,
			MIN(received_at) as first_seen,
			MAX(received_at) as last_seen,
			COUNT(DISTINCT DATE(received_at)) as active_days
		FROM matched
	`, bsEui)

	var stats mioty.MessageStats
	var minRSSI, maxRSSI, minSNR, maxSNR sql.NullFloat64
	var firstSeen, lastSeen sql.NullTime
	var activeDays sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, tenantID, euiToBytes(bsEui)).Scan(
		&stats.TotalCount,
		&stats.UniqueEndpoints,
		&stats.AvgRSSI,
		&stats.AvgSNR,
		&minRSSI,
		&maxRSSI,
		&minSNR,
		&maxSNR,
		&firstSeen,
		&lastSeen,
		&activeDays,
	)

	if err != nil {
		return nil, fmt.Errorf("get extended message stats by base station: %w", err)
	}

	// Set nullable fields
	if minRSSI.Valid {
		stats.MinRSSI = &minRSSI.Float64
	}
	if maxRSSI.Valid {
		stats.MaxRSSI = &maxRSSI.Float64
	}
	if minSNR.Valid {
		stats.MinSNR = &minSNR.Float64
	}
	if maxSNR.Valid {
		stats.MaxSNR = &maxSNR.Float64
	}
	if firstSeen.Valid {
		stats.FirstSeen = &firstSeen.Time
	}
	if lastSeen.Valid {
		stats.LastSeen = &lastSeen.Time
	}
	if activeDays.Valid {
		days := int(activeDays.Int64)
		stats.ActiveDays = &days
	}

	return &stats, nil
}

// GetMessageStatsByEndpoint returns message statistics for a specific endpoint
func (r *MessageRepository) GetMessageStatsByEndpoint(ctx context.Context, epEui uint64, tenantID int64) (*mioty.MessageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COUNT(DISTINCT bs_eui) as unique_basestations,
			COALESCE(AVG(rssi), 0) as avg_rssi,
			COALESCE(AVG(snr), 0) as avg_snr
		FROM messages
		WHERE ep_eui = $1 AND tenant_id = $2
	`

	var stats mioty.MessageStats
	err := r.db.QueryRowContext(ctx, query, euiToBytes(epEui), tenantID).Scan(
		&stats.TotalCount,
		&stats.UniqueEndpoints, // This will actually be unique base stations for endpoints
		&stats.AvgRSSI,
		&stats.AvgSNR,
	)

	if err != nil {
		return nil, fmt.Errorf("get message stats by endpoint: %w", err)
	}

	return &stats, nil
}

// GetOverallStats returns overall message statistics
func (r *MessageRepository) GetOverallStats(ctx context.Context, tenantID int64) (*mioty.MessageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COUNT(DISTINCT ep_eui) as unique_endpoints,
			COALESCE(AVG(rssi), 0) as avg_rssi,
			COALESCE(AVG(snr), 0) as avg_snr
		FROM messages
		WHERE tenant_id = $1
	`

	var stats mioty.MessageStats
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TotalCount,
		&stats.UniqueEndpoints,
		&stats.AvgRSSI,
		&stats.AvgSNR,
	)

	if err != nil {
		return nil, fmt.Errorf("get overall stats: %w", err)
	}

	return &stats, nil
}

// AnalyticsOverviewStats represents overview statistics for analytics

// GetAnalyticsOverview returns overview analytics within a time range
func (r *MessageRepository) GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.AnalyticsOverviewStats, error) {
	query := `
		SELECT
			COUNT(*) as total_messages,
			COUNT(DISTINCT ep_eui) as active_endpoints,
			COUNT(DISTINCT bs_eui) as active_basestations,
			AVG(rssi) as avg_rssi,
			AVG(snr) as avg_snr,
			MIN(received_at) as first_message,
			MAX(received_at) as last_message
		FROM messages
		WHERE tenant_id = $1
		AND received_at BETWEEN $2 AND $3
	`

	var stats mioty.AnalyticsOverviewStats
	err := r.db.QueryRowContext(ctx, query, tenantID, startTime, endTime).Scan(
		&stats.TotalMessages,
		&stats.ActiveEndpoints,
		&stats.ActiveBaseStations,
		&stats.AvgRSSI,
		&stats.AvgSNR,
		&stats.FirstMessage,
		&stats.LastMessage,
	)

	if err != nil {
		return nil, fmt.Errorf("get analytics overview: %w", err)
	}

	return &stats, nil
}

// HourlyActivity represents hourly message activity

// GetHourlyActivity returns hourly message counts within a time range
func (r *MessageRepository) GetHourlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.HourlyActivity, error) {
	query := `
		SELECT
			DATE_TRUNC('hour', received_at) as hour,
			COUNT(*) as message_count
		FROM messages
		WHERE tenant_id = $1
		AND received_at BETWEEN $2 AND $3
		GROUP BY hour
		ORDER BY hour
	`

	var activities []mioty.HourlyActivity
	err := r.db.SelectContext(ctx, &activities, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get hourly activity: %w", err)
	}

	return activities, nil
}

// DailyActivity represents daily message activity

// GetDailyActivity returns daily activity statistics within a time range
func (r *MessageRepository) GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.DailyActivity, error) {
	query := `
		SELECT
			DATE(received_at) as day,
			COUNT(*) as message_count,
			COUNT(DISTINCT ep_eui) as unique_endpoints,
			COUNT(DISTINCT bs_eui) as unique_basestations
		FROM messages
		WHERE tenant_id = $1
		AND received_at BETWEEN $2 AND $3
		GROUP BY day
		ORDER BY day
	`

	var activities []mioty.DailyActivity
	err := r.db.SelectContext(ctx, &activities, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get daily activity: %w", err)
	}

	return activities, nil
}

// EndpointActivity represents activity for a specific endpoint

// GetTopEndpointsByActivity returns the most active endpoints within a time range
func (r *MessageRepository) GetTopEndpointsByActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time, limit int) ([]mioty.EndpointActivity, error) {
	query := `
		SELECT
			ep_eui,
			COUNT(*) as message_count,
			MAX(received_at) as last_seen
		FROM messages
		WHERE tenant_id = $1
		AND received_at BETWEEN $2 AND $3
		GROUP BY ep_eui
		ORDER BY message_count DESC
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("get top endpoints by activity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var endpoints []mioty.EndpointActivity
	for rows.Next() {
		var ep mioty.EndpointActivity
		var epEuiB []byte
		if err := rows.Scan(&epEuiB, &ep.MessageCount, &ep.LastSeen); err != nil {
			return nil, fmt.Errorf("scan top endpoint: %w", err)
		}
		ep.EUI = euiFromBytes(epEuiB)
		endpoints = append(endpoints, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top endpoints: %w", err)
	}

	return endpoints, nil
}

// SignalQualityStats represents overall signal quality statistics

// GetSignalQualityStats returns signal quality statistics within a time range
func (r *MessageRepository) GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.SignalQualityStats, error) {
	query := `
		SELECT
			COALESCE(AVG(rssi), 0) as avg_rssi,
			COALESCE(MIN(rssi), 0) as min_rssi,
			COALESCE(MAX(rssi), 0) as max_rssi,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rssi), 0) as median_rssi,
			COALESCE(AVG(snr), 0) as avg_snr,
			COALESCE(MIN(snr), 0) as min_snr,
			COALESCE(MAX(snr), 0) as max_snr,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY snr), 0) as median_snr,
			COUNT(*) as total_messages
		FROM messages
		WHERE tenant_id = $1
		AND received_at BETWEEN $2 AND $3
	`

	var stats mioty.SignalQualityStats
	err := r.db.QueryRowContext(ctx, query, tenantID, startTime, endTime).Scan(
		&stats.AvgRSSI,
		&stats.MinRSSI,
		&stats.MaxRSSI,
		&stats.MedianRSSI,
		&stats.AvgSNR,
		&stats.MinSNR,
		&stats.MaxSNR,
		&stats.MedianSNR,
		&stats.TotalMessages,
	)

	if err != nil {
		return nil, fmt.Errorf("get signal quality stats: %w", err)
	}

	return &stats, nil
}

// BaseStationSignalQuality represents signal quality for a specific base station

// GetSignalQualityByBaseStation returns signal quality stats grouped by base station
func (r *MessageRepository) GetSignalQualityByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.BaseStationSignalQuality, error) {
	query := `
		SELECT
			bs_eui,
			AVG(rssi) as avg_rssi,
			AVG(snr) as avg_snr,
			COUNT(*) as message_count
		FROM messages
		WHERE tenant_id = $1
		AND received_at BETWEEN $2 AND $3
		GROUP BY bs_eui
		ORDER BY avg_rssi DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get signal quality by base station: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []mioty.BaseStationSignalQuality
	for rows.Next() {
		var bs mioty.BaseStationSignalQuality
		var bsEuiB []byte
		if err := rows.Scan(&bsEuiB, &bs.AvgRSSI, &bs.AvgSNR, &bs.MessageCount); err != nil {
			return nil, fmt.Errorf("scan signal quality: %w", err)
		}
		bs.EUI = euiFromBytes(bsEuiB)
		stats = append(stats, bs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate signal quality: %w", err)
	}

	return stats, nil
}

// GetBaseStationMessageStats retrieves aggregated message statistics for a base station.
// Includes messages where the BS is a secondary receiver in base_stations JSONB.
func (r *MessageRepository) GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error) {

	// Convert bsEui bytes to uint64
	var bsEuiUint uint64
	if len(bsEui) == 8 {
		bsEuiUint = binary.BigEndian.Uint64(bsEui)
	}

	bsContainment := fmt.Sprintf(`[{"bsEui":%d}]`, bsEuiUint)

	// Use CTE to extract per-BS signal values from JSONB for accurate stats
	baseQuery := fmt.Sprintf(`
		WITH matched AS (
			SELECT m.id, m.ep_eui, m.received_at,
				(elem->>'rssi')::double precision AS bs_rssi,
				(elem->>'snr')::double precision AS bs_snr
			FROM messages m,
				jsonb_array_elements(m.base_stations) AS elem
			WHERE m.tenant_id = $1
				AND (elem->>'bsEui') = '%d'
			UNION ALL
			SELECT id, ep_eui, received_at, rssi, snr
			FROM messages
			WHERE tenant_id = $1 AND bs_eui = $2
				AND (base_stations IS NULL OR jsonb_array_length(base_stations) = 0)
		)
		SELECT
			COUNT(*) as total_messages,
			COUNT(DISTINCT ep_eui) as total_endpoints,
			COALESCE(AVG(bs_rssi), 0) as avg_rssi,
			COALESCE(AVG(bs_snr), 0) as avg_snr,
			MAX(received_at) as last_message_at
		FROM matched
	`, bsEuiUint)

	args := []interface{}{tenantID, bsEui}
	if startTime != nil && endTime != nil {
		// Add time filter to both CTE branches
		baseQuery = fmt.Sprintf(`
		WITH matched AS (
			SELECT m.id, m.ep_eui, m.received_at,
				(elem->>'rssi')::double precision AS bs_rssi,
				(elem->>'snr')::double precision AS bs_snr
			FROM messages m,
				jsonb_array_elements(m.base_stations) AS elem
			WHERE m.tenant_id = $1
				AND (elem->>'bsEui') = '%d'
				AND m.received_at BETWEEN $3 AND $4
			UNION ALL
			SELECT id, ep_eui, received_at, rssi, snr
			FROM messages
			WHERE tenant_id = $1 AND bs_eui = $2
				AND (base_stations IS NULL OR jsonb_array_length(base_stations) = 0)
				AND received_at BETWEEN $3 AND $4
		)
		SELECT
			COUNT(*) as total_messages,
			COUNT(DISTINCT ep_eui) as total_endpoints,
			COALESCE(AVG(bs_rssi), 0) as avg_rssi,
			COALESCE(AVG(bs_snr), 0) as avg_snr,
			MAX(received_at) as last_message_at
		FROM matched
		`, bsEuiUint)
		args = append(args, *startTime, *endTime)
	}

	var stats mioty.BaseStationMessageStats
	var lastMsg sql.NullTime

	err := r.db.QueryRowContext(ctx, baseQuery, args...).Scan(
		&stats.TotalMessages,
		&stats.TotalEndpoints,
		&stats.AvgRSSI,
		&stats.AvgSNR,
		&lastMsg,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &stats, nil
		}
		return nil, fmt.Errorf("get base station message stats: %w", err)
	}

	if lastMsg.Valid {
		stats.LastMessageAt = &lastMsg.Time
	}

	// Get time-based counts (include JSONB secondary receivers)
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	periodQuery := `SELECT COUNT(*) FROM messages
		 WHERE tenant_id = $1
		   AND (bs_eui = $2 OR base_stations @> $3::jsonb)
		   AND received_at >= $4`

	// Messages today
	err = r.db.QueryRowContext(ctx, periodQuery,
		tenantID, bsEui, bsContainment, todayStart).Scan(&stats.MessagesToday)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get today's message count: %w", err)
	}

	// Messages this week
	err = r.db.QueryRowContext(ctx, periodQuery,
		tenantID, bsEui, bsContainment, weekStart).Scan(&stats.MessagesThisWeek)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get this week's message count: %w", err)
	}

	// Messages this month
	err = r.db.QueryRowContext(ctx, periodQuery,
		tenantID, bsEui, bsContainment, monthStart).Scan(&stats.MessagesThisMonth)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get this month's message count: %w", err)
	}

	return &stats, nil
}

// GetBaseStationEndpointCounts retrieves per-endpoint message counts for a base station.
// Includes messages where the BS is a secondary receiver in base_stations JSONB.
func (r *MessageRepository) GetBaseStationEndpointCounts(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time) (map[string]int64, error) {

	// Convert bsEui bytes to uint64
	var bsEuiUint uint64
	if len(bsEui) == 8 {
		bsEuiUint = binary.BigEndian.Uint64(bsEui)
	}

	bsContainment := fmt.Sprintf(`[{"bsEui":%d}]`, bsEuiUint)

	query := `
		SELECT
			ep_eui,
			COUNT(*) as message_count
		FROM messages
		WHERE tenant_id = $1 AND (bs_eui = $2 OR base_stations @> $3::jsonb)
	`

	args := []interface{}{tenantID, bsEuiUint, bsContainment}
	if startTime != nil && endTime != nil {
		query += " AND received_at BETWEEN $4 AND $5"
		args = append(args, *startTime, *endTime)
	}

	query += " GROUP BY ep_eui"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get base station endpoint counts: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in message repository: %v", err)
		}
	}()

	counts := make(map[string]int64)
	for rows.Next() {
		var epEui uint64
		var count int64
		if err := rows.Scan(&epEui, &count); err != nil {
			return nil, fmt.Errorf("scan endpoint count: %w", err)
		}
		// Convert uint64 to hex string
		counts[fmt.Sprintf("%016X", epEui)] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate endpoint counts: %w", err)
	}

	return counts, nil
}

// GetBaseStationLastSeen retrieves the last seen timestamp for a base station.
// Includes messages where the BS is a secondary receiver in base_stations JSONB.
func (r *MessageRepository) GetBaseStationLastSeen(ctx context.Context, tenantID int64, bsEui []byte) (*time.Time, error) {
	// Convert bsEui bytes to uint64
	var bsEuiUint uint64
	if len(bsEui) == 8 {
		bsEuiUint = binary.BigEndian.Uint64(bsEui)
	}

	bsContainment := fmt.Sprintf(`[{"bsEui":%d}]`, bsEuiUint)

	// Check messages first (include JSONB secondary receivers)
	var lastSeen sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(received_at) FROM messages WHERE tenant_id = $1 AND (bs_eui = $2 OR base_stations @> $3::jsonb)`,
		tenantID, bsEuiUint, bsContainment).Scan(&lastSeen)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get base station last seen: %w", err)
	}

	if lastSeen.Valid {
		return &lastSeen.Time, nil
	}

	// If no messages, check basestation_sessions for last connect time
	err = r.db.QueryRowContext(ctx,
		`SELECT MAX(connected_at) FROM basestation_sessions
		 WHERE tenant_id = $1 AND bs_eui = $2`,
		tenantID, bsEuiUint).Scan(&lastSeen)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get base station session last seen: %w", err)
	}

	if lastSeen.Valid {
		return &lastSeen.Time, nil
	}

	return nil, nil
}

// GetMessageCountsByEndpoint returns uncapped message counts grouped by endpoint EUI.
func (r *MessageRepository) GetMessageCountsByEndpoint(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error) {
	query := `
		SELECT ep_eui, COUNT(*) as message_count
		FROM messages
		WHERE tenant_id = $1 AND received_at BETWEEN $2 AND $3
		GROUP BY ep_eui`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get message counts by endpoint: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[string]int64)
	for rows.Next() {
		var euiUint uint64
		var count int64
		if err := rows.Scan(&euiUint, &count); err != nil {
			return nil, fmt.Errorf("scan endpoint message count: %w", err)
		}
		counts[fmt.Sprintf("%016x", euiUint)] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate endpoint message counts: %w", err)
	}

	return counts, nil
}

// GetWeeklyActivity returns message counts grouped by ISO week.
func (r *MessageRepository) GetWeeklyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.WeeklyActivity, error) {
	query := `
		SELECT DATE_TRUNC('week', received_at) AS week, COUNT(*) AS message_count
		FROM messages
		WHERE tenant_id = $1 AND received_at BETWEEN $2 AND $3
		GROUP BY week
		ORDER BY week`

	var activity []mioty.WeeklyActivity
	err := r.db.SelectContext(ctx, &activity, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get weekly activity: %w", err)
	}
	return activity, nil
}

// GetMonthlyActivity returns message counts grouped by month.
func (r *MessageRepository) GetMonthlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.MonthlyActivity, error) {
	query := `
		SELECT DATE_TRUNC('month', received_at) AS month, COUNT(*) AS message_count
		FROM messages
		WHERE tenant_id = $1 AND received_at BETWEEN $2 AND $3
		GROUP BY month
		ORDER BY month`

	var activity []mioty.MonthlyActivity
	err := r.db.SelectContext(ctx, &activity, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get monthly activity: %w", err)
	}
	return activity, nil
}

// GetMessageCountsByBaseStation returns uncapped message counts grouped by base station EUI.
func (r *MessageRepository) GetMessageCountsByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error) {
	query := `
		SELECT bs_eui, COUNT(*) as message_count
		FROM messages
		WHERE tenant_id = $1 AND received_at BETWEEN $2 AND $3
		GROUP BY bs_eui`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get message counts by base station: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[string]int64)
	for rows.Next() {
		var euiUint uint64
		var count int64
		if err := rows.Scan(&euiUint, &count); err != nil {
			return nil, fmt.Errorf("scan base station message count: %w", err)
		}
		counts[fmt.Sprintf("%016x", euiUint)] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate base station message counts: %w", err)
	}

	return counts, nil
}
