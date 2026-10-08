package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq/hstore"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// sqlUpdateEndpointFields rewrites the caller-editable columns of one tenant's
// endpoint and the time an edit last changed its station profile. ep_status
// and the propagation columns are server-owned attachment state that only the
// attachment transitions change; a nil blueprint snapshot or profile change
// time keeps the stored one.
const sqlUpdateEndpointFields = `
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
		pre_attach = $15,
		dual_chan = $16,
		repetition = $17,
		wide_carr_off = $18,
		long_blk_dist = $19,
		device_model_id = $20,
		attach_cnt = $21,
		last_packet_cnt = $22,
		type_eui = $23,
		blueprint_snapshot = COALESCE($24, blueprint_snapshot),
		profile_changed_at = COALESCE($25, profile_changed_at)
	WHERE id = $1 AND tenant_id = $2
	RETURNING updated_at`

// updateEndpointFields writes the caller-editable columns of endpoint row id
// under eui and hands back the stored class, which follows bidi. The row error
// is returned as the driver reported it.
func (r *EndPointRepository) updateEndpointFields(ctx context.Context, q sqlx.QueryerContext, id, tenantID int64, eui []byte, endpoint *models.EndPoint) error {
	nwkKeyEnc, err := encryptKeyMaterial(r.cipher, endpoint.NwkSnKey)
	if err != nil {
		return err
	}
	appKeyEnc, err := encryptKeyMaterial(r.cipher, endpoint.AppKey)
	if err != nil {
		return err
	}
	var typeEUI interface{}
	if endpoint.TypeEUI != nil {
		typeEUI = endpoint.TypeEUI[:]
	}
	var blueprintSnapshot interface{}
	if len(endpoint.BlueprintSnapshot) > 0 {
		blueprintSnapshot = []byte(endpoint.BlueprintSnapshot)
	}
	class := mioty.EndpointClass(endpoint.Bidi)

	err = q.QueryRowxContext(ctx, sqlUpdateEndpointFields,
		id, tenantID, eui,
		endpoint.Name, endpoint.Description,
		nullableByteParam(nwkKeyEnc), nullableByteParam(appKeyEnc),
		endpoint.CryptoMode, endpoint.BatteryLevel, endpointTags(endpoint.Tags),
		endpoint.ShAddr, endpoint.CarrierOffset,
		endpoint.Bidi, class, endpoint.PreAttach,
		endpoint.DualChan, endpoint.Repetition, endpoint.WideCarrOff, endpoint.LongBlkDist,
		endpoint.DeviceModelID, endpoint.AttachCnt, endpoint.LastPacketCnt,
		typeEUI, blueprintSnapshot, endpoint.ProfileChangedAt,
	).Scan(&endpoint.UpdatedAt)
	if err != nil {
		return err
	}
	endpoint.EPClass = class
	return nil
}

// endpointTags converts endpoint tags into the hstore column value.
func endpointTags(tags map[string]string) hstore.Hstore {
	values := make(map[string]sql.NullString, len(tags))
	for k, v := range tags {
		values[k] = sql.NullString{String: v, Valid: true}
	}
	return hstore.Hstore{Map: values}
}
