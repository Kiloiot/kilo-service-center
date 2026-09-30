package postgres

import (
	"database/sql"
	"fmt"
	"math"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/lib/pq/hstore"
)

// Constraint names PostgreSQL assigns to the endpoints table.
const (
	// PostgreSQL auto-names inline CHECK constraints as {table}_{column}_check
	constraintNwkKey = "endpoints_nwk_key_check"
	constraintAppKey = "endpoints_app_key_check"
	// PostgreSQL auto-names inline REFERENCES as {table}_{column}_fkey
	constraintDeviceModel = "endpoints_device_model_id_fkey"
)

// nullableByteParam converts nil or empty byte slices to SQL NULL.
// Prevents lib/pq from sending empty BYTEA for nullable columns with CHECK constraints.
// resourceEndpoint names this repository's resource in duplicate errors.
const resourceEndpoint = "endpoint"

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
	LastAttachedBsEui    []byte // last_attached_bs_eui is BYTEA(8)
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
	ep.LastAttachedBsEui = mioty.OptionalEUI64FromBytes(n.LastAttachedBsEui)
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
	nwk_key, app_key, sign, crypto_mode,
	last_seen_at, frame_count, battery_level,
	tags, created_at, updated_at, sh_addr,
	last_attached_bs_eui, last_propagate_time, last_detach_time,
	last_detach_sign, last_detach_packet_cnt, propagate_status,
	ep_status, device_model_id,
	blueprint_snapshot`

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
	dual_chan, repetition, wide_carr_off, long_blk_dist,
	profile_changed_at`

// applyEndpointPostScan converts hstore tags and nullable detach fields
// onto the endpoint struct after scanning. Shared by base-row and list-row scanners.
func applyEndpointPostScan(
	endpoint *models.EndPoint,
	tags hstore.Hstore,
	lastDetachSign []byte,
	lastAttachedBsEui []byte,
	lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64,
	propagateStatus sql.NullString,
) {
	endpoint.Tags = make(map[string]string)
	for k, v := range tags.Map {
		endpoint.Tags[k] = v.String
	}
	if len(lastDetachSign) > 0 {
		endpoint.LastDetachSign = lastDetachSign
	}
	endpoint.LastAttachedBsEui = mioty.OptionalEUI64FromBytes(lastAttachedBsEui)
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
}

// scanEndpointBaseRow scans a single row selected with endpointBaseSelectColumns into *models.EndPoint.
// Column order MUST match endpointBaseSelectColumns.
func scanEndpointBaseRow(scanner interface {
	Scan(dest ...interface{}) error
},
) (*models.EndPoint, error) {
	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var lastDetachSign []byte
	var lastAttachedBsEui []byte
	var lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
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
		&endpoint.Sign,
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
		&endpoint.BlueprintSnapshot,
	)
	if err != nil {
		return nil, err
	}

	applyEndpointPostScan(endpoint, tags, lastDetachSign,
		lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt,
		propagateStatus)

	return endpoint, nil
}

// scanEndpointListRow scans a single row from an endpoint list query into *models.EndPoint.
// Column order must match endpointListSelectColumns.
func scanEndpointListRow(scanner interface {
	Scan(dest ...interface{}) error
},
) (*models.EndPoint, error) {
	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var lastDetachSign []byte
	var lastAttachedBsEui []byte
	var lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
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
		&endpoint.ProfileChangedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapScanEndpoint, err)
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

	if endpoint.AttachCnt, err = attachCounter(attachCnt, endpoint.ID); err != nil {
		return nil, err
	}

	// Nullable fields (BSSCI §5.7)
	if len(lastDetachSign) > 0 {
		endpoint.LastDetachSign = lastDetachSign
	}
	endpoint.LastAttachedBsEui = mioty.OptionalEUI64FromBytes(lastAttachedBsEui)
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
	nwk_key, app_key, sign, crypto_mode,
	last_seen_at, frame_count, battery_level,
	tags, created_at, updated_at, sh_addr,
	endpoint_class, bidi, pre_attach, type_eui,
	attach_cnt, last_packet_cnt,
	carrier_offset, dual_chan, repetition, wide_carr_off, long_blk_dist,
	last_attached_bs_eui, last_propagate_time, last_detach_time,
	last_detach_sign, last_detach_packet_cnt, propagate_status,
	last_attach_rx_time, last_attach_rx_duration,
	last_snr, last_rssi, last_eq_snr, last_profile, last_attach_subpackets,
	ep_status, device_model_id,
	propagated_at, profile_changed_at,
	blueprint_snapshot`

// scanEndpointTenantLookupRow scans a row produced by endpointTenantLookupColumns
// into *models.EndPoint with full nullable resolution. Returns the raw scan
// error (callers map sql.ErrNoRows to their preferred sentinel).
func scanEndpointTenantLookupRow(scanner interface {
	Scan(dest ...interface{}) error
},
) (*models.EndPoint, error) {
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
		&endpoint.Sign,
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
		&detach.PropagatedAt, &endpoint.ProfileChangedAt,
		&endpoint.BlueprintSnapshot,
	)
	if err != nil {
		return nil, err
	}

	endpoint.Tags = hstoreToStringMap(tags)
	endpoint.TypeEUI = euiPtrFromBytes(typeEUIBytes)
	if endpoint.AttachCnt, err = attachCounter(attachCnt, endpoint.ID); err != nil {
		return nil, err
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
	propagated, propagated_at, propagation_count, profile_changed_at,
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
	device_model_id,
	blueprint_snapshot`

// scanEndpointDetailRow scans a row produced by endpointDetailColumns into
// *models.EndPoint with full nullable resolution. Returns the raw scan error.
func scanEndpointDetailRow(scanner interface {
	Scan(dest ...interface{}) error
},
) (*models.EndPoint, error) {
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
		&endpoint.ProfileChangedAt,
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
		&endpoint.BlueprintSnapshot,
	)
	if err != nil {
		return nil, err
	}

	endpoint.TypeEUI = euiPtrFromBytes(typeEUIBytes)
	endpoint.Tags = hstoreToStringMap(tags)
	if endpoint.AttachCnt, err = attachCounter(attachCnt, endpoint.ID); err != nil {
		return nil, err
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

// attachCounter reads a stored attach_cnt, which its column check bounds to
// the 32-bit attach counter; nil when none is stored.
func attachCounter(stored sql.NullInt64, endpointID int64) (*uint32, error) {
	if !stored.Valid {
		return nil, nil
	}
	if stored.Int64 < 0 || stored.Int64 > math.MaxUint32 {
		return nil, fmt.Errorf(errFmtAttachCntOutOfUint32RangeFor, stored.Int64, endpointID)
	}
	attachCnt := uint32(stored.Int64)
	return &attachCnt, nil
}
