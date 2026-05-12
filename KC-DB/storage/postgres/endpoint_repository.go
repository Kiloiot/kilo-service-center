package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/hstore"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// PostgreSQL SQLSTATE codes for error classification (§E.1 Error Codes)
const (
	sqlstateUniqueViolation     = "23505"
	sqlstateForeignKeyViolation = "23503"
	sqlstateCheckViolation      = "23514"
	sqlstateInvalidTextRep      = "22P02"

	// PostgreSQL auto-names inline CHECK constraints as {table}_{column}_check
	constraintNwkKey = "endpoints_nwk_key_check"
	constraintAppKey = "endpoints_app_key_check"
	// PostgreSQL auto-names inline REFERENCES as {table}_{column}_fkey
	constraintDeviceModel = "endpoints_device_model_id_fkey"
)

// nullableByteParam converts nil or empty byte slices to SQL NULL.
// Prevents lib/pq from sending empty BYTEA for nullable columns with CHECK constraints.
func nullableByteParam(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

// hstoreToStringMap converts a postgres hstore to a Go map, omitting NULL values.
func hstoreToStringMap(h hstore.Hstore) map[string]string {
	out := make(map[string]string, len(h.Map))
	for k, v := range h.Map {
		if v.Valid {
			out[k] = v.String
		}
	}
	return out
}

// euiPtrFromBytes converts an 8-byte BYTEA column to *models.EUI. Returns nil
// for any other length so callers can leave the destination unset.
func euiPtrFromBytes(b []byte) *models.EUI {
	if len(b) != 8 {
		return nil
	}
	var eui models.EUI
	copy(eui[:], b)
	return &eui
}

// attachNullables groups the nullable scan targets for an endpoint's last-attach
// metrics. Pass field addresses to sql.Row.Scan, then call assignAttachFields.
type attachNullables struct {
	LastAttachRxTime     sql.NullInt64
	LastAttachRxDuration sql.NullInt64
	LastAttachSubpackets sql.NullString
	LastAttachedBsEui    sql.NullInt64
}

func assignAttachFields(ep *models.EndPoint, n attachNullables) {
	if n.LastAttachRxTime.Valid {
		v := n.LastAttachRxTime.Int64
		ep.LastAttachRxTime = &v
	}
	if n.LastAttachRxDuration.Valid {
		v := n.LastAttachRxDuration.Int64
		ep.LastAttachRxDuration = &v
	}
	if n.LastAttachSubpackets.Valid {
		v := n.LastAttachSubpackets.String
		ep.LastAttachSubpackets = &v
	}
	if n.LastAttachedBsEui.Valid {
		v := n.LastAttachedBsEui.Int64
		ep.LastAttachedBsEui = &v
	}
}

// detachNullables groups the nullable scan targets for an endpoint's detach
// and propagate state.
type detachNullables struct {
	LastDetachTime      sql.NullInt64
	LastDetachPacketCnt sql.NullInt64
	LastDetachSign      []byte
	LastPropagateTime   sql.NullInt64
	PropagateStatus     sql.NullString
	PropagatedAt        sql.NullTime
}

func assignDetachFields(ep *models.EndPoint, n detachNullables) {
	if n.LastDetachTime.Valid {
		v := n.LastDetachTime.Int64
		ep.LastDetachTime = &v
	}
	if n.LastDetachPacketCnt.Valid {
		v := n.LastDetachPacketCnt.Int64
		ep.LastDetachPacketCnt = &v
	}
	if len(n.LastDetachSign) > 0 {
		ep.LastDetachSign = n.LastDetachSign
	}
	if n.LastPropagateTime.Valid {
		v := n.LastPropagateTime.Int64
		ep.LastPropagateTime = &v
	}
	if n.PropagateStatus.Valid {
		v := n.PropagateStatus.String
		ep.PropagateStatus = &v
	}
	if n.PropagatedAt.Valid {
		v := n.PropagatedAt.Time
		ep.PropagatedAt = &v
	}
}

// radioNullables groups the nullable scan targets for the BSSCI §3.6.1/3.7.1
// radio metrics.
type radioNullables struct {
	LastSNR     sql.NullFloat64
	LastRSSI    sql.NullFloat64
	LastEqSNR   sql.NullFloat64
	LastProfile sql.NullString
}

func assignRadioMetrics(ep *models.EndPoint, n radioNullables) {
	if n.LastSNR.Valid {
		v := n.LastSNR.Float64
		ep.LastSNR = &v
	}
	if n.LastRSSI.Valid {
		v := n.LastRSSI.Float64
		ep.LastRSSI = &v
	}
	if n.LastEqSNR.Valid {
		v := n.LastEqSNR.Float64
		ep.LastEqSNR = &v
	}
	if n.LastProfile.Valid {
		v := n.LastProfile.String
		ep.LastProfile = &v
	}
}

// uplinkNullables groups the nullable scan targets for an endpoint's last-uplink
// telemetry. Direct columns (LastDlOpen, LastResponseExp, LastDlAck, PacketCnt)
// are scanned straight into the struct in callers and not represented here.
type uplinkNullables struct {
	LastUserData   []byte
	LastFormatID   sql.NullInt32
	LastMode       sql.NullString
	LastRxTime     sql.NullInt64
	LastRxDuration sql.NullInt64
}

func assignUplinkFields(ep *models.EndPoint, n uplinkNullables) {
	if len(n.LastUserData) > 0 {
		ep.LastUserData = n.LastUserData
	}
	if n.LastFormatID.Valid {
		v := n.LastFormatID.Int32
		ep.LastFormatID = &v
	}
	if n.LastMode.Valid {
		v := n.LastMode.String
		ep.LastMode = &v
	}
	if n.LastRxTime.Valid {
		v := n.LastRxTime.Int64
		ep.LastRxTime = &v
	}
	if n.LastRxDuration.Valid {
		v := n.LastRxDuration.Int64
		ep.LastRxDuration = &v
	}
}

// endpointBaseSelectColumns defines the standard column list for endpoint queries.
// Column order MUST match scanEndpointBaseRow field order.
const endpointBaseSelectColumns = `
	id, ep_eui, name, description, tenant_id, owner_tenant_id,
	nwk_key, app_key, crypto_mode,
	last_seen_at, frame_count, battery_level,
	tags, created_at, updated_at, sh_addr,
	last_attached_bs_eui, last_propagate_time, last_detach_time,
	last_detach_sign, last_detach_packet_cnt, propagate_status,
	ep_status, device_model_id`

// endpointListSelectColumns defines columns for paginated list queries (no key material).
// Column order MUST match scanEndpointListRow field order.
const endpointListSelectColumns = `
	id, ep_eui, name, description, tenant_id, owner_tenant_id,
	crypto_mode,
	last_seen_at, frame_count, battery_level,
	tags, created_at, updated_at, sh_addr,
	manufacturer, model, carrier_offset,
	propagated, propagated_at, propagation_count,
	last_attached_bs_eui, last_propagate_time, last_detach_time,
	last_detach_sign, last_detach_packet_cnt, propagate_status,
	ep_status, endpoint_class, device_model_id,
	bidi, pre_attach, type_eui, attach_cnt, last_packet_cnt,
	dual_chan, repetition, wide_carr_off, long_blk_dist`

// scanEndpointBaseRow scans a single row selected with endpointBaseSelectColumns into *models.EndPoint.
// Column order MUST match endpointBaseSelectColumns.
func scanEndpointBaseRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.EndPoint, error) {
	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var lastDetachSign []byte
	var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
	var propagateStatus sql.NullString

	err := scanner.Scan(
		&endpoint.ID,
		&endpoint.EUI,
		&endpoint.Name,
		&endpoint.Description,
		&endpoint.TenantID,
		&endpoint.OwnerTenantID,
		&endpoint.NwkSnKey,
		&endpoint.AppKey,
		&endpoint.CryptoMode,
		&endpoint.LastSeenAt,
		&endpoint.FrameCount,
		&endpoint.BatteryLevel,
		&tags,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
		&endpoint.ShAddr,
		// Detach fields (BSSCI §5.7)
		&lastAttachedBsEui, &lastPropagateTime, &lastDetachTime,
		&lastDetachSign, &lastDetachPacketCnt, &propagateStatus,
		&endpoint.EpStatus,
		&endpoint.DeviceModelID,
	)
	if err != nil {
		return nil, err
	}

	// Convert hstore to map[string]string
	endpoint.Tags = make(map[string]string)
	for k, v := range tags.Map {
		endpoint.Tags[k] = v.String
	}

	// Detach nullable fields (BSSCI §5.7)
	if len(lastDetachSign) > 0 {
		endpoint.LastDetachSign = lastDetachSign
	}
	if lastAttachedBsEui.Valid {
		val := lastAttachedBsEui.Int64
		endpoint.LastAttachedBsEui = &val
	}
	if lastPropagateTime.Valid {
		val := lastPropagateTime.Int64
		endpoint.LastPropagateTime = &val
	}
	if lastDetachTime.Valid {
		val := lastDetachTime.Int64
		endpoint.LastDetachTime = &val
	}
	if lastDetachPacketCnt.Valid {
		val := lastDetachPacketCnt.Int64
		endpoint.LastDetachPacketCnt = &val
	}
	if propagateStatus.Valid {
		val := propagateStatus.String
		endpoint.PropagateStatus = &val
	}

	return endpoint, nil
}

// scanEndpointListRow scans a single row from an endpoint list query into *models.EndPoint.
// Column order must match endpointListSelectColumns.
func scanEndpointListRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.EndPoint, error) {
	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var lastDetachSign []byte
	var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
	var propagateStatus sql.NullString
	var typeEUIBytes []byte
	var attachCnt sql.NullInt64

	err := scanner.Scan(
		&endpoint.ID,
		&endpoint.EUI,
		&endpoint.Name,
		&endpoint.Description,
		&endpoint.TenantID,
		&endpoint.OwnerTenantID,
		&endpoint.CryptoMode,
		&endpoint.LastSeenAt,
		&endpoint.FrameCount,
		&endpoint.BatteryLevel,
		&tags,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
		&endpoint.ShAddr,
		&endpoint.Manufacturer,
		&endpoint.Model,
		&endpoint.CarrierOffset,
		&endpoint.Propagated,
		&endpoint.PropagatedAt,
		&endpoint.PropagationCount,
		&lastAttachedBsEui, &lastPropagateTime, &lastDetachTime,
		&lastDetachSign, &lastDetachPacketCnt, &propagateStatus,
		&endpoint.EpStatus,
		&endpoint.EPClass,
		&endpoint.DeviceModelID,
		&endpoint.Bidi,
		&endpoint.PreAttach,
		&typeEUIBytes,
		&attachCnt,
		&endpoint.LastPacketCnt,
		&endpoint.DualChan,
		&endpoint.Repetition,
		&endpoint.WideCarrOff,
		&endpoint.LongBlkDist,
	)
	if err != nil {
		return nil, fmt.Errorf("scan endpoint: %w", err)
	}

	// Convert hstore to map[string]string
	endpoint.Tags = make(map[string]string)
	for k, v := range tags.Map {
		if v.Valid {
			endpoint.Tags[k] = v.String
		}
	}

	if len(typeEUIBytes) == 8 {
		var typeEUI models.EUI
		copy(typeEUI[:], typeEUIBytes)
		endpoint.TypeEUI = &typeEUI
	}

	if attachCnt.Valid {
		val := uint32(attachCnt.Int64) // #nosec G115 -- DB CHECK constraint ensures 0-4294967295
		endpoint.AttachCnt = &val
	}

	// Nullable fields (BSSCI §5.7)
	if len(lastDetachSign) > 0 {
		endpoint.LastDetachSign = lastDetachSign
	}
	if lastAttachedBsEui.Valid {
		val := lastAttachedBsEui.Int64
		endpoint.LastAttachedBsEui = &val
	}
	if lastPropagateTime.Valid {
		val := lastPropagateTime.Int64
		endpoint.LastPropagateTime = &val
	}
	if lastDetachTime.Valid {
		val := lastDetachTime.Int64
		endpoint.LastDetachTime = &val
	}
	if lastDetachPacketCnt.Valid {
		val := lastDetachPacketCnt.Int64
		endpoint.LastDetachPacketCnt = &val
	}
	if propagateStatus.Valid {
		val := propagateStatus.String
		endpoint.PropagateStatus = &val
	}

	return endpoint, nil
}

// endpointTenantLookupColumns lists columns returned by tenant-scoped endpoint
// lookups (GetByEUI on both EndPointRepository and transactionalEndPointRepository).
// Includes owner_tenant_id for cross-tenant roaming visibility and the full
// attach/detach radio metrics. Column order MUST match scanEndpointTenantLookupRow.
const endpointTenantLookupColumns = `
	id, ep_eui, name, description, tenant_id, owner_tenant_id,
	nwk_key, app_key, crypto_mode,
	last_seen_at, frame_count, battery_level,
	tags, created_at, updated_at, sh_addr,
	endpoint_class, bidi, pre_attach, type_eui,
	attach_cnt, last_packet_cnt,
	carrier_offset, dual_chan, repetition, wide_carr_off, long_blk_dist,
	last_attached_bs_eui, last_propagate_time, last_detach_time,
	last_detach_sign, last_detach_packet_cnt, propagate_status,
	last_attach_rx_time, last_attach_rx_duration,
	last_snr, last_rssi, last_eq_snr, last_profile, last_attach_subpackets,
	ep_status, device_model_id`

// scanEndpointTenantLookupRow scans a row produced by endpointTenantLookupColumns
// into *models.EndPoint with full nullable resolution. Returns the raw scan
// error (callers map sql.ErrNoRows to their preferred sentinel).
func scanEndpointTenantLookupRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.EndPoint, error) {
	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var typeEUIBytes []byte
	var attachCnt sql.NullInt64
	var attach attachNullables
	var detach detachNullables
	var radio radioNullables

	err := scanner.Scan(
		&endpoint.ID,
		&endpoint.EUI,
		&endpoint.Name,
		&endpoint.Description,
		&endpoint.TenantID,
		&endpoint.OwnerTenantID,
		&endpoint.NwkSnKey,
		&endpoint.AppKey,
		&endpoint.CryptoMode,
		&endpoint.LastSeenAt,
		&endpoint.FrameCount,
		&endpoint.BatteryLevel,
		&tags,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
		&endpoint.ShAddr,
		&endpoint.EPClass,
		&endpoint.Bidi,
		&endpoint.PreAttach,
		&typeEUIBytes,
		&attachCnt,
		&endpoint.LastPacketCnt,
		&endpoint.CarrierOffset,
		&endpoint.DualChan,
		&endpoint.Repetition,
		&endpoint.WideCarrOff,
		&endpoint.LongBlkDist,
		// Detach fields (BSSCI §5.7)
		&attach.LastAttachedBsEui, &detach.LastPropagateTime, &detach.LastDetachTime,
		&detach.LastDetachSign, &detach.LastDetachPacketCnt, &detach.PropagateStatus,
		// Radio metrics (BSSCI §3.6.1/3.7.1)
		&attach.LastAttachRxTime, &attach.LastAttachRxDuration,
		&radio.LastSNR, &radio.LastRSSI, &radio.LastEqSNR, &radio.LastProfile, &attach.LastAttachSubpackets,
		&endpoint.EpStatus,
		&endpoint.DeviceModelID,
	)
	if err != nil {
		return nil, err
	}

	endpoint.Tags = hstoreToStringMap(tags)
	endpoint.TypeEUI = euiPtrFromBytes(typeEUIBytes)
	if attachCnt.Valid {
		val := uint32(attachCnt.Int64) // #nosec G115 -- DB CHECK constraint ensures 0-4294967295
		endpoint.AttachCnt = &val
	}
	assignAttachFields(endpoint, attach)
	assignDetachFields(endpoint, detach)
	assignRadioMetrics(endpoint, radio)

	return endpoint, nil
}

// endpointDetailColumns lists columns returned by full-detail endpoint lookups
// (GetByID on both EndPointRepository and transactionalEndPointRepository).
// Union of attach + detach + radio metric + UL/DL telemetry columns plus
// owner_tenant_id, ep_status and device_model_id. Column order MUST match
// scanEndpointDetailRow.
const endpointDetailColumns = `
	id, ep_eui, name, description, tenant_id, owner_tenant_id,
	nwk_key, app_key, crypto_mode,
	last_seen_at, frame_count, battery_level,
	tags, created_at, updated_at, sh_addr,
	manufacturer, model, carrier_offset, type_eui,
	propagated, propagated_at, propagation_count,
	bidi, pre_attach,
	dual_chan, repetition, wide_carr_off, long_blk_dist,
	attach_cnt, nonce, sign, last_attach_rx_time, last_attach_rx_duration,
	last_snr, last_rssi, last_eq_snr, last_profile, last_attach_subpackets,
	last_attached_bs_eui, last_propagate_time, last_detach_time,
	last_detach_sign, last_detach_packet_cnt, propagate_status,
	ep_status,
	last_packet_cnt,
	last_user_data, last_format_id, last_mode,
	last_rx_time, last_rx_duration, packet_cnt,
	last_dl_open, last_response_exp, last_dl_ack,
	endpoint_class,
	device_model_id`

// scanEndpointDetailRow scans a row produced by endpointDetailColumns into
// *models.EndPoint with full nullable resolution. Returns the raw scan error.
func scanEndpointDetailRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.EndPoint, error) {
	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var typeEUIBytes []byte
	var attachCnt sql.NullInt64
	var nonce, sign []byte
	var attach attachNullables
	var detach detachNullables
	var radio radioNullables
	var uplink uplinkNullables

	err := scanner.Scan(
		&endpoint.ID,
		&endpoint.EUI,
		&endpoint.Name,
		&endpoint.Description,
		&endpoint.TenantID,
		&endpoint.OwnerTenantID,
		&endpoint.NwkSnKey,
		&endpoint.AppKey,
		&endpoint.CryptoMode,
		&endpoint.LastSeenAt,
		&endpoint.FrameCount,
		&endpoint.BatteryLevel,
		&tags,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
		&endpoint.ShAddr,
		&endpoint.Manufacturer,
		&endpoint.Model,
		&endpoint.CarrierOffset,
		&typeEUIBytes,
		&endpoint.Propagated,
		&endpoint.PropagatedAt,
		&endpoint.PropagationCount,
		&endpoint.Bidi,
		&endpoint.PreAttach,
		// MIOTY config
		&endpoint.DualChan, &endpoint.Repetition, &endpoint.WideCarrOff, &endpoint.LongBlkDist,
		// Attach fields
		&attachCnt, &nonce, &sign, &attach.LastAttachRxTime, &attach.LastAttachRxDuration,
		// Radio metrics
		&radio.LastSNR, &radio.LastRSSI, &radio.LastEqSNR, &radio.LastProfile, &attach.LastAttachSubpackets,
		// Detach fields (BSSCI §5.7)
		&attach.LastAttachedBsEui, &detach.LastPropagateTime, &detach.LastDetachTime,
		&detach.LastDetachSign, &detach.LastDetachPacketCnt, &detach.PropagateStatus,
		// Attach status
		&endpoint.EpStatus,
		// UL deduplication
		&endpoint.LastPacketCnt,
		// UL data
		&uplink.LastUserData, &uplink.LastFormatID, &uplink.LastMode,
		// UL reception
		&uplink.LastRxTime, &uplink.LastRxDuration, &endpoint.PacketCnt,
		// Downlink control
		&endpoint.LastDlOpen, &endpoint.LastResponseExp, &endpoint.LastDlAck,
		// Legacy
		&endpoint.EPClass,
		// Blueprint device model.
		&endpoint.DeviceModelID,
	)
	if err != nil {
		return nil, err
	}

	endpoint.TypeEUI = euiPtrFromBytes(typeEUIBytes)
	endpoint.Tags = hstoreToStringMap(tags)
	if attachCnt.Valid {
		val := uint32(attachCnt.Int64) // #nosec G115 -- DB CHECK constraint ensures 0-4294967295
		endpoint.AttachCnt = &val
	}
	if len(nonce) > 0 {
		endpoint.Nonce = nonce
	}
	if len(sign) > 0 {
		endpoint.Sign = sign
	}
	assignAttachFields(endpoint, attach)
	assignDetachFields(endpoint, detach)
	assignRadioMetrics(endpoint, radio)
	assignUplinkFields(endpoint, uplink)

	return endpoint, nil
}

// endpointDetachValidationColumns lists columns returned by detach validation
// lookups (GetEndpointWithKeysForDetachValidation variants). Includes the
// preshared_key + key material needed for detach signature validation per
// BSSCI §5.7. Column order MUST match scanEndpointDetachValidationRow.
const endpointDetachValidationColumns = `
	id, ep_eui, name, description, tenant_id, owner_tenant_id,
	nwk_key, app_key, sh_addr, bidi, pre_attach, type_eui,
	manufacturer, model, carrier_offset,
	propagated, propagated_at, propagation_count,
	dual_chan, repetition, wide_carr_off, long_blk_dist,
	attach_cnt, nonce, sign, preshared_key,
	last_attach_rx_time, last_attach_rx_duration,
	last_snr, last_rssi, last_eq_snr, last_profile,
	last_attach_subpackets,
	last_attached_bs_eui, last_propagate_time, last_detach_time,
	last_detach_sign, last_detach_packet_cnt, propagate_status,
	last_packet_cnt,
	last_user_data, last_format_id, last_mode,
	last_rx_time, last_rx_duration, packet_cnt,
	last_dl_open, last_response_exp, last_dl_ack,
	crypto_mode, endpoint_class,
	last_seen_at, frame_count, battery_level,
	tags, created_at, updated_at,
	device_model_id`

// scanEndpointDetachValidationRow scans a row produced by
// endpointDetachValidationColumns into *models.EndPoint with full nullable
// resolution. Returns the raw scan error.
func scanEndpointDetachValidationRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.EndPoint, error) {
	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var typeEUIBytes []byte
	var attach attachNullables
	var detach detachNullables
	var radio radioNullables
	var uplink uplinkNullables
	var lastSeenAt sql.NullTime
	var batteryLevel sql.NullFloat64

	err := scanner.Scan(
		&endpoint.ID,
		&endpoint.EUI,
		&endpoint.Name,
		&endpoint.Description,
		&endpoint.TenantID,
		&endpoint.OwnerTenantID,
		&endpoint.NwkSnKey,
		&endpoint.AppKey,
		&endpoint.ShAddr,
		&endpoint.Bidi,
		&endpoint.PreAttach,
		&typeEUIBytes,
		&endpoint.Manufacturer,
		&endpoint.Model,
		&endpoint.CarrierOffset,
		&endpoint.Propagated,
		&detach.PropagatedAt,
		&endpoint.PropagationCount,
		&endpoint.DualChan,
		&endpoint.Repetition,
		&endpoint.WideCarrOff,
		&endpoint.LongBlkDist,
		&endpoint.AttachCnt,
		&endpoint.Nonce,
		&endpoint.Sign,
		&endpoint.PresharedKey,
		&attach.LastAttachRxTime,
		&attach.LastAttachRxDuration,
		&radio.LastSNR,
		&radio.LastRSSI,
		&radio.LastEqSNR,
		&radio.LastProfile,
		&attach.LastAttachSubpackets,
		&attach.LastAttachedBsEui,
		&detach.LastPropagateTime,
		&detach.LastDetachTime,
		&endpoint.LastDetachSign,
		&detach.LastDetachPacketCnt,
		&detach.PropagateStatus,
		&endpoint.LastPacketCnt,
		&uplink.LastUserData,
		&uplink.LastFormatID,
		&uplink.LastMode,
		&uplink.LastRxTime,
		&uplink.LastRxDuration,
		&endpoint.PacketCnt,
		&endpoint.LastDlOpen,
		&endpoint.LastResponseExp,
		&endpoint.LastDlAck,
		&endpoint.CryptoMode,
		&endpoint.EPClass,
		&lastSeenAt,
		&endpoint.FrameCount,
		&batteryLevel,
		&tags,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
		&endpoint.DeviceModelID,
	)
	if err != nil {
		return nil, err
	}

	endpoint.Tags = hstoreToStringMap(tags)
	endpoint.TypeEUI = euiPtrFromBytes(typeEUIBytes)
	assignAttachFields(endpoint, attach)
	assignDetachFields(endpoint, detach)
	assignRadioMetrics(endpoint, radio)
	assignUplinkFields(endpoint, uplink)
	if lastSeenAt.Valid {
		endpoint.LastSeenAt = &lastSeenAt.Time
	}
	if batteryLevel.Valid {
		val := float32(batteryLevel.Float64)
		endpoint.BatteryLevel = &val
	}

	return endpoint, nil
}

// EndPointRepository implements interfaces.EndpointRepository for PostgreSQL
type EndPointRepository struct {
	db *sqlx.DB
}

// NewEndPointRepository creates a new endpoints table repository
func NewEndPointRepository(db *sqlx.DB) interfaces.EndpointRepository {
	return &EndPointRepository{db: db}
}

// Create creates a new endpoint
func (r *EndPointRepository) Create(ctx context.Context, endpoint *models.EndPoint) error {
	// Set owner_tenant_id to tenant_id on creation when omitted.
	if endpoint.OwnerTenantID == 0 {
		endpoint.OwnerTenantID = endpoint.TenantID
	}

	query := `
		INSERT INTO endpoints (
			ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			tags, sh_addr,
			manufacturer, model, carrier_offset,
			propagated, propagation_count, device_model_id,
			endpoint_class, bidi, pre_attach, type_eui,
			attach_cnt, last_packet_cnt,
			dual_chan, repetition, wide_carr_off, long_blk_dist
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
		RETURNING id, created_at, updated_at`

	// NwkSnKey validation is enforced at the handler/service layer.
	// Silent zero-fill removed to prevent invalid endpoints from being persisted.
	// AppKey: Column allows NULL (migration 001, no NOT NULL constraint); CHECK(length=16) applies only when non-null.
	// We let nil remain nil in DB rather than zero-fill, consistent with NwkSnKey treatment.

	// Convert map[string]string to hstore
	tagsHstore := make(map[string]sql.NullString)
	for k, v := range endpoint.Tags {
		tagsHstore[k] = sql.NullString{String: v, Valid: true}
	}
	tags := hstore.Hstore{Map: tagsHstore}

	// Handle AppKey nil→SQL NULL: pq driver converts nil []byte to empty bytea, not NULL.
	// Pass explicit nil interface{} to get SQL NULL for omitted app keys.
	var appKeyParam interface{}
	if endpoint.AppKey == nil {
		appKeyParam = nil // SQL NULL
	} else {
		appKeyParam = endpoint.AppKey // actual bytes
	}

	var typeEuiParam interface{}
	if endpoint.TypeEUI == nil {
		typeEuiParam = nil // SQL NULL
	} else {
		typeEuiParam = endpoint.TypeEUI[:] // 8-byte value
	}

	var attachCntParam interface{}
	if endpoint.AttachCnt == nil {
		attachCntParam = nil // SQL NULL
	} else {
		attachCntParam = int64(*endpoint.AttachCnt)
	}

	err := r.db.QueryRowContext(ctx, query,
		endpoint.EUI[:],
		endpoint.Name,
		endpoint.Description,
		endpoint.TenantID,
		endpoint.OwnerTenantID,
		endpoint.NwkSnKey,
		appKeyParam,
		endpoint.CryptoMode,
		tags,
		endpoint.ShAddr,
		endpoint.Manufacturer,
		endpoint.Model,
		endpoint.CarrierOffset,
		endpoint.Propagated,
		endpoint.PropagationCount,
		endpoint.DeviceModelID,
		endpoint.EPClass,
		endpoint.Bidi,
		endpoint.PreAttach,
		typeEuiParam,
		attachCntParam,
		int64(endpoint.LastPacketCnt),
		endpoint.DualChan,
		endpoint.Repetition,
		endpoint.WideCarrOff,
		endpoint.LongBlkDist,
	).Scan(&endpoint.ID, &endpoint.CreatedAt, &endpoint.UpdatedAt)

	if err != nil {
		return WrapDuplicateError(err, "endpoint")
	}

	return nil
}

// Get retrieves an endpoint by EUI
func (r *EndPointRepository) Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error) {
	query := `SELECT ` + endpointBaseSelectColumns + ` FROM endpoints WHERE ep_eui = $1`

	endpoint, err := scanEndpointBaseRow(r.db.QueryRowContext(ctx, query, eui[:]))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	return endpoint, nil
}

// GetByEUI retrieves an endpoint by EUI for a specific tenant
func (r *EndPointRepository) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) {
	query := `SELECT ` + endpointTenantLookupColumns + ` FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`
	endpoint, err := scanEndpointTenantLookupRow(r.db.QueryRowContext(ctx, query, tenantID, eui))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}
	return endpoint, nil
}

// GetByTenant retrieves all endpoints for a tenant
// Returns full BSSCI §3.8.1/§5.8.1 attach propagate fields for reconciliation
func (r *EndPointRepository) GetByTenant(ctx context.Context, tenantID int64) ([]*models.EndPoint, error) {
	query := `
		SELECT
			id, ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			last_seen_at, frame_count, battery_level,
			tags, created_at, updated_at, sh_addr,
			last_attached_bs_eui, last_propagate_time, last_detach_time,
			last_detach_sign, last_detach_packet_cnt, propagate_status,
			bidi, dual_chan, repetition, wide_carr_off, long_blk_dist,
			last_packet_cnt, pre_attach,
			propagated, propagated_at, propagation_count,
			ep_status,
			device_model_id
		FROM endpoints
		WHERE tenant_id = $1
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in endpoint query: %v", err)
		}
	}()

	var endpoints []*models.EndPoint
	for rows.Next() {
		endpoint := &models.EndPoint{}
		var tags hstore.Hstore
		var lastDetachSign []byte
		var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
		var propagateStatus sql.NullString

		// BSSCI §3.8.1/§5.8.1 attach propagate fields (NOT NULL with defaults)
		var bidi, dualChan, repetition, wideCarrOff, longBlkDist, preAttach, propagated bool
		var lastPacketCnt int64 // BIGINT, bounds-check before uint32 cast
		var propagationCount int32
		var epStatus string // NOT NULL with DEFAULT 'detached'

		// NULLABLE columns only
		var propagatedAt sql.NullTime

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.EUI,
			&endpoint.Name,
			&endpoint.Description,
			&endpoint.TenantID,
			&endpoint.OwnerTenantID,
			&endpoint.NwkSnKey,
			&endpoint.AppKey,
			&endpoint.CryptoMode,
			&endpoint.LastSeenAt,
			&endpoint.FrameCount,
			&endpoint.BatteryLevel,
			&tags,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
			&endpoint.ShAddr,
			// Detach fields (BSSCI §5.7)
			&lastAttachedBsEui, &lastPropagateTime, &lastDetachTime,
			&lastDetachSign, &lastDetachPacketCnt, &propagateStatus,
			// BSSCI §3.8.1/§5.8.1 attach propagate fields
			&bidi, &dualChan, &repetition, &wideCarrOff, &longBlkDist,
			&lastPacketCnt, &preAttach,
			&propagated, &propagatedAt, &propagationCount,
			&epStatus,
			// Blueprint device model.
			&endpoint.DeviceModelID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}

		// Convert hstore to map[string]string
		endpoint.Tags = make(map[string]string)
		for k, v := range tags.Map {
			endpoint.Tags[k] = v.String
		}

		// Detach nullable fields (BSSCI §5.7)
		if len(lastDetachSign) > 0 {
			endpoint.LastDetachSign = lastDetachSign
		}
		if lastAttachedBsEui.Valid {
			val := lastAttachedBsEui.Int64
			endpoint.LastAttachedBsEui = &val
		}
		if lastPropagateTime.Valid {
			val := lastPropagateTime.Int64
			endpoint.LastPropagateTime = &val
		}
		if lastDetachTime.Valid {
			val := lastDetachTime.Int64
			endpoint.LastDetachTime = &val
		}
		if lastDetachPacketCnt.Valid {
			val := lastDetachPacketCnt.Int64
			endpoint.LastDetachPacketCnt = &val
		}
		if propagateStatus.Valid {
			val := propagateStatus.String
			endpoint.PropagateStatus = &val
		}

		// BSSCI §3.8.1/§5.8.1 attach propagate fields (direct assignment for NOT NULL)
		endpoint.Bidi = bidi
		endpoint.DualChan = dualChan
		endpoint.Repetition = repetition
		endpoint.WideCarrOff = wideCarrOff
		endpoint.LongBlkDist = longBlkDist
		endpoint.PreAttach = preAttach
		endpoint.Propagated = propagated
		endpoint.PropagationCount = propagationCount
		endpoint.EpStatus = epStatus

		// Bounds-check uint32 cast for lastPacketCnt
		if lastPacketCnt < 0 || lastPacketCnt > 4294967295 {
			return nil, fmt.Errorf("last_packet_cnt %d out of uint32 range for endpoint %d", lastPacketCnt, endpoint.ID)
		}
		endpoint.LastPacketCnt = uint32(lastPacketCnt)

		// NULLABLE assignment
		if propagatedAt.Valid {
			endpoint.PropagatedAt = &propagatedAt.Time
		}

		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant endpoints: %w", err)
	}

	return endpoints, nil
}

// CountByTenant returns the total count of endpoints for a tenant
func (r *EndPointRepository) CountByTenant(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM endpoints WHERE tenant_id = $1`

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count endpoints by tenant: %w", err)
	}

	return count, nil
}

// GetByID retrieves an endpoint by ID with tenant isolation
func (r *EndPointRepository) GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	query := `SELECT ` + endpointDetailColumns + ` FROM endpoints WHERE id = $1 AND tenant_id = $2`
	endpoint, err := scanEndpointDetailRow(r.db.QueryRowContext(ctx, query, id, tenantID))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint by ID: %w", err)
	}
	return endpoint, nil
}

// ListByTenantPaginated retrieves paginated endpoints for a tenant with LIMIT/OFFSET
func (r *EndPointRepository) ListByTenantPaginated(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) {
	query := `SELECT` + endpointListSelectColumns + `
		FROM endpoints
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list endpoints paginated: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows in endpoint query: %v", err)
		}
	}()

	var endpoints []*models.EndPoint
	for rows.Next() {
		endpoint, err := scanEndpointListRow(rows)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate endpoints: %w", err)
	}

	return endpoints, nil
}

// Update updates an endpoint with tenant isolation
func (r *EndPointRepository) Update(ctx context.Context, endpoint *models.EndPoint) error {
	// Prepare nullable type_eui parameter
	var typeEuiParam interface{}
	if endpoint.TypeEUI != nil {
		typeEuiParam = endpoint.TypeEUI[:]
	}

	query := `
		UPDATE endpoints SET
			ep_eui = $3,
			name = $4,
			description = $5,
			nwk_key = $6,
			app_key = $7,
			crypto_mode = $8,
			battery_level = $9,
			tags = $10,
			sh_addr = $11,
			carrier_offset = $12,
			bidi = $13,
			endpoint_class = $14,
			ep_status = $15,
			pre_attach = $16,
			dual_chan = $17,
			repetition = $18,
			wide_carr_off = $19,
			long_blk_dist = $20,
			propagated = $21,
			propagated_at = $22,
			propagation_count = $23,
			device_model_id = $24,
			attach_cnt = $25,
			last_packet_cnt = $26,
			type_eui = $27
		WHERE id = $1 AND tenant_id = $2
		RETURNING updated_at`

	// Convert map[string]string to hstore
	tagsHstore := make(map[string]sql.NullString)
	for k, v := range endpoint.Tags {
		tagsHstore[k] = sql.NullString{String: v, Valid: true}
	}
	tags := hstore.Hstore{Map: tagsHstore}

	err := r.db.QueryRowContext(ctx, query,
		endpoint.ID,
		endpoint.TenantID,
		endpoint.EUI[:],
		endpoint.Name,
		endpoint.Description,
		nullableByteParam(endpoint.NwkSnKey),
		nullableByteParam(endpoint.AppKey),
		endpoint.CryptoMode,
		endpoint.BatteryLevel,
		tags,
		endpoint.ShAddr,
		endpoint.CarrierOffset,
		endpoint.Bidi,
		endpoint.EPClass,
		endpoint.EpStatus,
		endpoint.PreAttach,
		endpoint.DualChan,
		endpoint.Repetition,
		endpoint.WideCarrOff,
		endpoint.LongBlkDist,
		endpoint.Propagated,
		endpoint.PropagatedAt,
		endpoint.PropagationCount,
		endpoint.DeviceModelID,
		endpoint.AttachCnt,
		endpoint.LastPacketCnt,
		typeEuiParam,
	).Scan(&endpoint.UpdatedAt)

	if err == sql.ErrNoRows {
		return storage.ErrNotFound
	}
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case sqlstateUniqueViolation:
				return storage.ErrAlreadyExists
			case sqlstateForeignKeyViolation:
				if pqErr.Constraint == constraintDeviceModel {
					return storage.ErrForeignKeyViolation
				}
				return fmt.Errorf("foreign key violation (%s): %w", pqErr.Constraint, err)
			case sqlstateCheckViolation:
				switch pqErr.Constraint {
				case constraintNwkKey:
					return storage.ErrNwkKeyLength
				case constraintAppKey:
					return storage.ErrAppKeyLength
				default:
					return storage.ErrCheckViolation
				}
			case sqlstateInvalidTextRep:
				return storage.ErrInvalidInput
			}
		}
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	return nil
}

// CheckEUIUnique verifies no endpoint with the given EUI exists (global uniqueness)
func (r *EndPointRepository) CheckEUIUnique(ctx context.Context, eui []byte) error {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM endpoints WHERE ep_eui = $1", eui)
	if err != nil {
		return fmt.Errorf("check EUI uniqueness: %w", err)
	}
	if count > 0 {
		return storage.ErrAlreadyExists
	}
	return nil
}

// UpdateWithEUI atomically cascades an EUI change across all dependent tables and updates endpoint fields
func (r *EndPointRepository) UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error) {
	if len(oldEui) != 8 || len(endpoint.EUI) != 8 {
		return nil, fmt.Errorf("invalid EUI length: expected 8 bytes")
	}
	newEui := endpoint.EUI[:]

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Verify tenant ownership
	var epID int64
	err = tx.GetContext(ctx, &epID, "SELECT id FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2", tenantID, oldEui)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("verify endpoint ownership: %w", err)
	}

	// Convert map[string]string to hstore
	tagsHstore := make(map[string]sql.NullString)
	for k, v := range endpoint.Tags {
		tagsHstore[k] = sql.NullString{String: v, Valid: true}
	}
	tags := hstore.Hstore{Map: tagsHstore}

	// Prepare nullable type_eui parameter
	var typeEuiParam interface{}
	if endpoint.TypeEUI != nil {
		typeEuiParam = endpoint.TypeEUI[:]
	}

	// Update endpoints table: set new EUI + all fields
	updateQuery := `
		UPDATE endpoints SET
			ep_eui = $3,
			name = $4,
			description = $5,
			nwk_key = $6,
			app_key = $7,
			crypto_mode = $8,
			battery_level = $9,
			tags = $10,
			sh_addr = $11,
			carrier_offset = $12,
			bidi = $13,
			endpoint_class = $14,
			ep_status = $15,
			pre_attach = $16,
			dual_chan = $17,
			repetition = $18,
			wide_carr_off = $19,
			long_blk_dist = $20,
			propagated = $21,
			propagated_at = $22,
			propagation_count = $23,
			device_model_id = $24,
			attach_cnt = $25,
			last_packet_cnt = $26,
			type_eui = $27
		WHERE id = $1 AND tenant_id = $2
		RETURNING updated_at`

	err = tx.QueryRowContext(ctx, updateQuery,
		epID,
		tenantID,
		newEui,
		endpoint.Name,
		endpoint.Description,
		nullableByteParam(endpoint.NwkSnKey),
		nullableByteParam(endpoint.AppKey),
		endpoint.CryptoMode,
		endpoint.BatteryLevel,
		tags,
		endpoint.ShAddr,
		endpoint.CarrierOffset,
		endpoint.Bidi,
		endpoint.EPClass,
		endpoint.EpStatus,
		endpoint.PreAttach,
		endpoint.DualChan,
		endpoint.Repetition,
		endpoint.WideCarrOff,
		endpoint.LongBlkDist,
		endpoint.Propagated,
		endpoint.PropagatedAt,
		endpoint.PropagationCount,
		endpoint.DeviceModelID,
		endpoint.AttachCnt,
		endpoint.LastPacketCnt,
		typeEuiParam,
	).Scan(&endpoint.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case sqlstateUniqueViolation:
				return nil, storage.ErrAlreadyExists
			case sqlstateForeignKeyViolation:
				if pqErr.Constraint == constraintDeviceModel {
					return nil, storage.ErrForeignKeyViolation
				}
				return nil, fmt.Errorf("foreign key violation (%s): %w", pqErr.Constraint, err)
			case sqlstateCheckViolation:
				switch pqErr.Constraint {
				case constraintNwkKey:
					return nil, storage.ErrNwkKeyLength
				case constraintAppKey:
					return nil, storage.ErrAppKeyLength
				default:
					return nil, storage.ErrCheckViolation
				}
			case sqlstateInvalidTextRep:
				return nil, storage.ErrInvalidInput
			}
		}
		return nil, fmt.Errorf("update endpoints: %w", err)
	}

	// Cascade EUI to dependent tables

	// BYTEA columns: downlink_queue.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE downlink_queue SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update downlink_queue.ep_eui: %w", err)
	}

	// BYTEA columns: dl_rx_status.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update dl_rx_status.ep_eui: %w", err)
	}

	// BYTEA columns: dl_rx_status_queries.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status_queries SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update dl_rx_status_queries.ep_eui: %w", err)
	}

	// BIGINT columns: messages.ep_eui (uses uint64 representation)
	oldUint64 := euiToUint64(oldEui)
	newUint64 := euiToUint64(newEui)
	_, err = tx.ExecContext(ctx, "UPDATE messages SET ep_eui = $1 WHERE ep_eui = $2", newUint64, oldUint64)
	if err != nil {
		return nil, fmt.Errorf("update messages.ep_eui: %w", err)
	}

	// Schema-aware update: messages_archive.ep_eui can be BYTEA (migration 001) or BIGINT (if future migration aligns it)
	var archiveEpColType string
	err = tx.GetContext(ctx, &archiveEpColType, `
		SELECT data_type
		FROM information_schema.columns
		WHERE table_name = 'messages_archive' AND column_name = 'ep_eui'
	`)
	if err != nil {
		return nil, fmt.Errorf("detect messages_archive.ep_eui column type: %w", err)
	}
	if archiveEpColType == "bytea" {
		_, err = tx.ExecContext(ctx, "UPDATE messages_archive SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	} else {
		_, err = tx.ExecContext(ctx, "UPDATE messages_archive SET ep_eui = $1 WHERE ep_eui = $2", newUint64, oldUint64)
	}
	if err != nil {
		return nil, fmt.Errorf("update messages_archive.ep_eui: %w", err)
	}

	// BYTEA columns: mioty_messages.ep_eui (nullable)
	_, err = tx.ExecContext(ctx, "UPDATE mioty_messages SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update mioty_messages.ep_eui: %w", err)
	}

	// BYTEA columns: mioty_message_deduplication.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE mioty_message_deduplication SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update mioty_message_deduplication.ep_eui: %w", err)
	}

	// BYTEA columns: roaming_events.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE roaming_events SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update roaming_events.ep_eui: %w", err)
	}

	// BYTEA columns: bssci_pending_operations.endpoint_eui (nullable)
	_, err = tx.ExecContext(ctx, "UPDATE bssci_pending_operations SET endpoint_eui = $1 WHERE endpoint_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update bssci_pending_operations.endpoint_eui: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	endpoint.ID = epID
	return endpoint, nil
}

// DeleteByTenant deletes an endpoint by EUI with tenant isolation
// Returns storage.ErrNotFound when 0 rows affected
func (r *EndPointRepository) DeleteByTenant(ctx context.Context, tenantID int64, eui []byte) error {
	query := `DELETE FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, eui)
	if err != nil {
		return fmt.Errorf("failed to delete endpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateLastSeen updates the last seen timestamp and frame count with tenant isolation
func (r *EndPointRepository) UpdateLastSeen(ctx context.Context, tenantID int64, eui models.EUI, frameCount uint32) error {
	query := `
		UPDATE endpoints SET
			last_seen_at = CURRENT_TIMESTAMP,
			frame_count = $3
		WHERE ep_eui = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, eui[:], tenantID, frameCount)
	if err != nil {
		return fmt.Errorf("failed to update endpoint last seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device not found")
	}

	return nil
}

// UpdateRadioMetrics updates the radio metrics from attach or uplink operations
func (r *EndPointRepository) UpdateRadioMetrics(ctx context.Context, tenantID int64, eui models.EUI, snr, rssi, eqSnr float64, rxTime, rxDuration int64, profile string) error {
	query := `
		UPDATE endpoints SET
			last_snr = $3,
			last_rssi = $4,
			last_eq_snr = $5,
			last_attach_rx_time = $6,
			last_attach_rx_duration = $7,
			last_profile = $8,
			last_seen_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND ep_eui = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, eui[:], snr, rssi, eqSnr, rxTime, rxDuration, profile)
	if err != nil {
		return fmt.Errorf("failed to update endpoint radio metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateRadioMetricsSelective updates radio metrics with optional field support.
// Nil pointers in the update struct preserve existing database values.
func (r *EndPointRepository) UpdateRadioMetricsSelective(ctx context.Context, tenantID int64, eui models.EUI, update interfaces.RadioMetricsUpdate) error {
	// Build dynamic SET clause
	setClauses := []string{
		"last_snr = $3",
		"last_rssi = $4",
		"last_eq_snr = $5",
		"last_attach_rx_time = $6",
		"last_seen_at = CURRENT_TIMESTAMP",
	}
	args := []interface{}{tenantID, eui[:], update.SNR, update.RSSI, update.EqSNR, update.RxTime}
	argIdx := 7

	if update.RxDuration != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_attach_rx_duration = $%d", argIdx))
		args = append(args, *update.RxDuration)
		argIdx++
	}

	if update.Profile != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_profile = $%d", argIdx))
		args = append(args, *update.Profile)
	}

	query := fmt.Sprintf(`
		UPDATE endpoints SET
			%s
		WHERE tenant_id = $1 AND ep_eui = $2`,
		strings.Join(setClauses, ",\n			"))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update endpoint radio metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateDetachMetrics updates detach-specific telemetry plus shared radio metrics.
// Updates: last_detach_rx_time, last_detach_packet_cnt, last_detach_sign, last_snr, last_rssi, last_eq_snr, last_profile, last_seen_at
// Does NOT update: last_attach_rx_time, last_attach_rx_duration (preserves attach telemetry)
func (r *EndPointRepository) UpdateDetachMetrics(ctx context.Context, tenantID int64, eui models.EUI, update interfaces.DetachMetricsUpdate) error {
	// Build dynamic SET clause for detach-specific + shared metrics
	setClauses := []string{
		"last_detach_rx_time = $3",
		"last_detach_packet_cnt = $4",
		"last_detach_sign = $5",
		"last_snr = $6",
		"last_rssi = $7",
		"last_seen_at = CURRENT_TIMESTAMP",
	}
	args := []interface{}{tenantID, eui[:], update.RxTime, update.PacketCnt, update.Signature, update.SNR, update.RSSI}
	argIdx := 8

	// Optional EqSNR
	if update.EqSNR != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_eq_snr = $%d", argIdx))
		args = append(args, *update.EqSNR)
		argIdx++
	}

	// Optional Profile
	if update.Profile != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_profile = $%d", argIdx))
		args = append(args, *update.Profile)
	}

	query := fmt.Sprintf(`
		UPDATE endpoints SET
			%s
		WHERE tenant_id = $1 AND ep_eui = $2`,
		strings.Join(setClauses, ",\n			"))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update endpoint detach metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateFields updates endpoint fields using a map of field names to values
// This method is used by the BSSCI server to update endpoints with MIOTY data
func (r *EndPointRepository) UpdateFields(ctx context.Context, tenantID int64, endpointID int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	// Build dynamic UPDATE query
	setParts := make([]string, 0, len(updates))
	values := make([]interface{}, 0, len(updates)+2)
	paramIndex := 1

	// Add tenant_id and id parameters first
	values = append(values, tenantID, endpointID)

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, paramIndex+2))
		values = append(values, value)
		paramIndex++
	}

	query := fmt.Sprintf(`
		UPDATE endpoints SET
			%s,
			updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2`,
		strings.Join(setParts, ", "))

	result, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// StreamAllForPropagation fetches endpoints across all tenants for base station reconciliation
func (r *EndPointRepository) StreamAllForPropagation(ctx context.Context, cursorID int64, limit int) ([]*models.EndPoint, error) {
	query := `
		SELECT id, ep_eui, tenant_id, nwk_key, bidi, sh_addr, dual_chan,
		       repetition, wide_carr_off, long_blk_dist, propagated_at
		FROM endpoints
		WHERE propagated = TRUE
		  AND id > $1
		ORDER BY id
		LIMIT $2
	`
	var endpoints []*models.EndPoint
	err := r.db.SelectContext(ctx, &endpoints, query, cursorID, limit)
	if err != nil {
		return nil, fmt.Errorf("stream endpoints for propagation: %w", err)
	}
	return endpoints, nil
}

// HasEndpointsSince checks if any propagated endpoints exist after the given timestamp
func (r *EndPointRepository) HasEndpointsSince(ctx context.Context, since time.Time) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM endpoints
			WHERE propagated = TRUE
			  AND propagated_at > $1
		)
	`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, since)
	if err != nil {
		return false, fmt.Errorf("check endpoints since: %w", err)
	}
	return exists, nil
}

// GetEndpointWithKeysForDetachValidation retrieves an endpoint with all crypto material for detach signature validation
// Returns complete endpoint record including Sign, NwkSnKey, and PresharedKey fields.
func (r *EndPointRepository) GetEndpointWithKeysForDetachValidation(ctx context.Context, eui models.EUI) (*models.EndPoint, error) {
	query := `SELECT ` + endpointDetachValidationColumns + ` FROM endpoints WHERE ep_eui = $1 LIMIT 1`
	endpoint, err := scanEndpointDetachValidationRow(r.db.QueryRowContext(ctx, query, eui[:]))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get endpoint with keys for detach validation: %w", err)
	}
	return endpoint, nil
}

// ============================================================================
// Roaming support methods.
// ============================================================================

// GetEndpointWithOwnership retrieves an endpoint with ownership information
func (r *EndPointRepository) GetEndpointWithOwnership(ctx context.Context, eui models.EUI, _ int64) (*models.EndPoint, error) {
	query := `SELECT ` + endpointBaseSelectColumns + ` FROM endpoints WHERE ep_eui = $1`

	endpoint, err := scanEndpointBaseRow(r.db.QueryRowContext(ctx, query, eui[:]))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint with ownership: %w", err)
	}

	return endpoint, nil
}

// UpdateOwnerTenant updates the owner_tenant_id for an endpoint
func (r *EndPointRepository) UpdateOwnerTenant(ctx context.Context, eui models.EUI, ownerTenantID int64) error {
	query := `
		UPDATE endpoints
		SET owner_tenant_id = $2, updated_at = NOW()
		WHERE ep_eui = $1`

	result, err := r.db.ExecContext(ctx, query, eui[:], ownerTenantID)
	if err != nil {
		return fmt.Errorf("failed to update owner tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetRoamingEndpoints retrieves endpoints that are roaming (owner_tenant_id != tenant_id)
func (r *EndPointRepository) GetRoamingEndpoints(ctx context.Context, tenantID int64) ([]*models.EndPoint, error) {
	query := `
		SELECT
			id, ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			last_seen_at, frame_count, battery_level,
			tags, created_at, updated_at, sh_addr,
			device_model_id
		FROM endpoints
		WHERE tenant_id = $1 AND owner_tenant_id != tenant_id
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query roaming endpoints: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in endpoint query: %v", err)
		}
	}()

	var endpoints []*models.EndPoint
	for rows.Next() {
		endpoint := &models.EndPoint{}
		var tags hstore.Hstore

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.EUI,
			&endpoint.Name,
			&endpoint.Description,
			&endpoint.TenantID,
			&endpoint.OwnerTenantID,
			&endpoint.NwkSnKey,
			&endpoint.AppKey,
			&endpoint.CryptoMode,
			&endpoint.LastSeenAt,
			&endpoint.FrameCount,
			&endpoint.BatteryLevel,
			&tags,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
			&endpoint.ShAddr,
			// Blueprint device model.
			&endpoint.DeviceModelID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan roaming endpoint: %w", err)
		}

		// Convert hstore to map[string]string
		endpoint.Tags = make(map[string]string)
		for k, v := range tags.Map {
			endpoint.Tags[k] = v.String
		}

		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roaming endpoints: %w", err)
	}

	return endpoints, nil
}

// UpdateServingTenant updates the serving tenant (tenant_id) while preserving owner
func (r *EndPointRepository) UpdateServingTenant(ctx context.Context, eui models.EUI, servingTenantID int64) error {
	query := `
		UPDATE endpoints
		SET tenant_id = $2, updated_at = NOW()
		WHERE ep_eui = $1`

	result, err := r.db.ExecContext(ctx, query, eui[:], servingTenantID)
	if err != nil {
		return fmt.Errorf("failed to update serving tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetPreferredBsEui returns the last_attached_bs_eui for an endpoint.
// Used by UL Data Transmit (SCACI §3.9.1) for "Service Center preferred BS" selection.
// Returns (nil, false, nil) if endpoint not found or column is NULL.
func (r *EndPointRepository) GetPreferredBsEui(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error) {
	query := `SELECT last_attached_bs_eui FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`

	var preferredBs sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, tenantID, epEui).Scan(&preferredBs)
	if err == sql.ErrNoRows {
		return nil, false, nil // No endpoint found
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to get preferred BS: %w", err)
	}
	if !preferredBs.Valid {
		return nil, false, nil // NULL value - no preference
	}
	if preferredBs.Int64 < 0 {
		return nil, false, fmt.Errorf("invalid negative BS EUI: %d", preferredBs.Int64)
	}
	val := uint64(preferredBs.Int64) //#nosec G115 - validated non-negative above
	return &val, true, nil
}
