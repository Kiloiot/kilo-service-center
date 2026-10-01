package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// EndpointLocations reads where a tenant's endpoint was last heard or attached.
type EndpointLocations struct {
	db sqlx.ExtContext
}

// GetEndpointLocation reads where the tenant's endpoint was last heard and
// attached; roaming stations of any tenant count. An uplink stored without
// its reception list names its first receiver alone. storage.ErrNotFound when
// the tenant has no such endpoint; a known endpoint neither heard nor
// attached has an empty location.
func (r *EndpointLocations) GetEndpointLocation(ctx context.Context, tenantID int64, epEUI uint64) (storage.EndpointLocation, error) {
	var firstReceiver, receptionsJSON, attachedThrough []byte
	err := r.db.QueryRowxContext(ctx, sqlEndpointLocation,
		mioty.EUI64Bytes(epEUI), tenantID, models.SessionStatusActive, mioty.CmdULData).
		Scan(&firstReceiver, &receptionsJSON, &attachedThrough)
	if errors.Is(err, sql.ErrNoRows) {
		return storage.EndpointLocation{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.EndpointLocation{}, fmt.Errorf("%s: %w", errWrapGetEndpointLocation, err)
	}
	location := storage.EndpointLocation{}
	if len(attachedThrough) > 0 {
		location.AttachedThrough = mioty.EUI64FromBytes(attachedThrough)
	}
	if len(receptionsJSON) > 0 {
		if err := json.Unmarshal(receptionsJSON, &location.LatestReceptions); err != nil {
			return storage.EndpointLocation{}, fmt.Errorf("%s: %w", errWrapGetEndpointLocation, err)
		}
	}
	if len(location.LatestReceptions) == 0 && len(firstReceiver) > 0 {
		location.LatestReceptions = []mioty.BaseStationReception{{BsEui: mioty.EUI64FromBytes(firstReceiver)}}
	}
	return location, nil
}

// sqlEndpointLocation reads, for the endpoint under its owner tenant,
// the receptions of its latest uplink and the station its active session was
// last attached or propagated through. An uplink counts whether or not the
// session exists yet: a pre-provisioned endpoint is heard before the
// connect-time propagation reaches it.
const sqlEndpointLocation = `
	SELECT m.bs_eui, m.base_stations, b.bs_eui
	FROM endpoints e
	LEFT JOIN LATERAL (
		SELECT primary_basestation_id
		FROM endpoint_sessions
		WHERE endpoint_id = e.id AND status = $3
		ORDER BY last_activity_at DESC
		LIMIT 1
	) es ON TRUE
	LEFT JOIN basestations b ON b.id = es.primary_basestation_id
	LEFT JOIN LATERAL (
		SELECT bs_eui, base_stations
		FROM messages
		WHERE ep_eui = e.ep_eui AND owner_tenant_id = e.tenant_id AND command_type = $4
		ORDER BY rx_time DESC
		LIMIT 1
	) m ON TRUE
	WHERE e.ep_eui = $1 AND e.tenant_id = $2`
