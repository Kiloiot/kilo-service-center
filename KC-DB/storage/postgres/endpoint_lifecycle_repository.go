package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq/hstore"
)

// endpointFieldSet accumulates a fixed-column SET clause and its bound values.
// Column names are compile-time literals passed by the builders below; values
// are always bound as query parameters. The WHERE clause reserves $1 for
// tenant_id and $2 for id, so bound values start at $3.
type endpointFieldSet struct {
	clauses []string
	args    []interface{}
}

func (s *endpointFieldSet) set(column string, value interface{}) {
	s.args = append(s.args, value)
	s.clauses = append(s.clauses, fmt.Sprintf("%s = $%d", column, len(s.args)))
}

// setNullTime writes a value or SQL NULL when the timestamp pointer is nil.
func (s *endpointFieldSet) setNullTime(column string, t *time.Time) {
	if t == nil {
		s.set(column, nil)
		return
	}
	s.set(column, *t)
}

// setNullBytes writes a value or SQL NULL when the byte slice is empty.
func (s *endpointFieldSet) setNullBytes(column string, b []byte) {
	if len(b) == 0 {
		s.set(column, nil)
		return
	}
	s.set(column, b)
}

func newEndpointFieldSet(tenantID, endpointID int64) *endpointFieldSet {
	return &endpointFieldSet{args: []interface{}{tenantID, endpointID}}
}

// runEndpointFieldUpdate executes the accumulated SET clause against the
// endpoints table with tenant isolation and returns storage.ErrNotFound when
// no row matches.
func runEndpointFieldUpdate(ctx context.Context, exec endpointFieldExec, fs *endpointFieldSet) error {
	if len(fs.clauses) == 0 {
		return errTextNoUpdatesProvided
	}

	query := fmt.Sprintf(`
		UPDATE endpoints SET
			%s,
			updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2`,
		strings.Join(fs.clauses, ",\n			"))

	result, err := exec.ExecContext(ctx, query, fs.args...)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateEndpoint, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// buildEndpointRegistrationSet builds the SCACI Register field set. The network
// key must already be encrypted at rest.
func buildEndpointRegistrationSet(tenantID, endpointID int64, encryptedKey []byte, p models.EndpointRegistrationParams) *endpointFieldSet {
	fs := newEndpointFieldSet(tenantID, endpointID)
	fs.set("nwk_key", encryptedKey)
	fs.set("pre_attach", p.PreAttach)
	fs.set("bidi", p.Bidi)
	fs.set("endpoint_class", mioty.EndpointClass(p.Bidi))
	fs.set("sh_addr", int32(p.ShAddr))
	fs.set("attach_cnt", int64(p.AttachCnt))
	fs.set("packet_cnt", int64(p.PacketCnt))
	fs.set("last_packet_cnt", int64(p.PacketCnt))
	fs.set("dual_chan", p.DualChan)
	fs.set("repetition", p.Repetition)
	fs.set("wide_carr_off", p.WideCarrOff)
	fs.set("long_blk_dist", p.LongBlkDist)
	return fs
}

// buildEndpointAttachmentStateSet builds the attach and attach-propagate
// completion field set; nil optionals leave their column untouched.
func buildEndpointAttachmentStateSet(tenantID, endpointID int64, p models.EndpointAttachmentStateParams) *endpointFieldSet {
	fs := newEndpointFieldSet(tenantID, endpointID)
	if p.AttachCnt != nil {
		fs.set("attach_cnt", *p.AttachCnt)
	}
	if p.LastAttachRxTime != nil {
		fs.set("last_attach_rx_time", *p.LastAttachRxTime)
	}
	if p.LastAttachRxDuration != nil {
		fs.set("last_attach_rx_duration", *p.LastAttachRxDuration)
	}
	if p.Nonce != nil {
		fs.set("nonce", p.Nonce)
	}
	if p.Sign != nil {
		fs.set("sign", p.Sign)
	}
	if p.DualChan != nil {
		fs.set("dual_chan", *p.DualChan)
	}
	if p.Repetition != nil {
		fs.set("repetition", *p.Repetition)
	}
	if p.WideCarrOff != nil {
		fs.set("wide_carr_off", *p.WideCarrOff)
	}
	if p.LongBlkDist != nil {
		fs.set("long_blk_dist", *p.LongBlkDist)
	}
	if p.ShAddr != nil {
		fs.set("sh_addr", int32(*p.ShAddr))
	}
	if p.LastAttachSubpackets != nil {
		fs.set("last_attach_subpackets", *p.LastAttachSubpackets)
	}
	if p.LastAttachedBsEui != nil {
		fs.set("last_attached_bs_eui", p.LastAttachedBsEui)
	}
	if p.LastPropagateTime != nil {
		fs.set("last_propagate_time", *p.LastPropagateTime)
	}
	if p.PropagateStatus != nil {
		fs.set("propagate_status", *p.PropagateStatus)
	}
	if p.Propagated != nil {
		fs.set("propagated", *p.Propagated)
	}
	if p.PropagatedAt.Set {
		fs.setNullTime("propagated_at", p.PropagatedAt.Time)
	}
	return fs
}

// buildEndpointAttachSessionSet builds the attach-propagate radio parameter set.
func buildEndpointAttachSessionSet(tenantID, endpointID int64, p models.EndpointAttachSessionParams) *endpointFieldSet {
	fs := newEndpointFieldSet(tenantID, endpointID)
	fs.set("propagated_at", p.PropagatedAt)
	fs.set("sh_addr", int32(p.ShAddr))
	fs.set("bidi", p.Bidi)
	fs.set("last_packet_cnt", int64(p.LastPacketCnt))
	fs.set("dual_chan", p.DualChan)
	fs.set("repetition", p.Repetition)
	fs.set("wide_carr_off", p.WideCarrOff)
	fs.set("long_blk_dist", p.LongBlkDist)
	return fs
}

// buildEndpointDetachStateSet builds the detach field set; nil optionals leave
// their column untouched while the explicit-NULL fields distinguish clearing a
// column from leaving it.
func buildEndpointDetachStateSet(tenantID, endpointID int64, p models.EndpointDetachStateParams) *endpointFieldSet {
	fs := newEndpointFieldSet(tenantID, endpointID)
	if p.LastAttachedBsEui.Set {
		fs.setNullBytes("last_attached_bs_eui", p.LastAttachedBsEui.Value)
	}
	if p.LastPropagateTime != nil {
		fs.set("last_propagate_time", *p.LastPropagateTime)
	}
	if p.LastDetachTime != nil {
		fs.set("last_detach_time", *p.LastDetachTime)
	}
	if p.LastDetachSign != nil {
		fs.set("last_detach_sign", p.LastDetachSign)
	}
	if p.LastDetachPacketCnt != nil {
		fs.set("last_detach_packet_cnt", int64(*p.LastDetachPacketCnt))
	}
	if p.PropagateStatus != nil {
		fs.set("propagate_status", *p.PropagateStatus)
	}
	if p.Propagated != nil {
		fs.set("propagated", *p.Propagated)
	}
	if p.PropagatedAt.Set {
		fs.setNullTime("propagated_at", p.PropagatedAt.Time)
	}
	return fs
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
		return fmt.Errorf("%s: %w", errWrapUpdateEndpointLastSeen, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return errTextDeviceNotFound
	}

	return nil
}

// UpdateRadioMetricsSelective updates radio metrics with optional field support.
// Nil pointers in the update struct preserve existing database values.
func (r *EndPointRepository) UpdateRadioMetricsSelective(ctx context.Context, tenantID int64, eui models.EUI, update models.RadioMetricsUpdate) error {
	// Build dynamic SET clause
	setClauses := []string{
		"last_snr = $3",
		"last_rssi = $4",
		"last_attach_rx_time = $5",
		"last_seen_at = CURRENT_TIMESTAMP",
	}
	args := []interface{}{tenantID, eui[:], update.SNR, update.RSSI, update.RxTime}
	argIdx := 6

	if update.EqSNR != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_eq_snr = $%d", argIdx))
		args = append(args, *update.EqSNR)
		argIdx++
	}

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
		return fmt.Errorf("%s: %w", errWrapUpdateEndpointRadioMetrics, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// EndpointRegistrationUpdate writes the SCACI Register field set (§3.6.1),
// encrypting the network key at rest before the row is written.
func (r *EndPointRepository) EndpointRegistrationUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointRegistrationParams) error {
	encryptedKey, err := encryptKeyMaterial(r.cipher, p.NwkKey)
	if err != nil {
		return err
	}
	return runEndpointFieldUpdate(ctx, r.db, buildEndpointRegistrationSet(tenantID, endpointID, encryptedKey, p))
}

// EndpointAttachmentStateUpdate writes the attach and attach-propagate
// completion field set for the BSSCI server.
func (r *EndPointRepository) EndpointAttachmentStateUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointAttachmentStateParams) error {
	return runEndpointFieldUpdate(ctx, r.db, buildEndpointAttachmentStateSet(tenantID, endpointID, p))
}

// EndpointAttachSessionUpdate writes the attach-propagate radio parameters.
func (r *EndPointRepository) EndpointAttachSessionUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointAttachSessionParams) error {
	return runEndpointFieldUpdate(ctx, r.db, buildEndpointAttachSessionSet(tenantID, endpointID, p))
}

// sqlRestartPacketCounter zeroes the endpoint counters.
const sqlRestartPacketCounter = `
	UPDATE endpoints SET packet_cnt = 0, last_packet_cnt = 0, updated_at = $3
	WHERE tenant_id = $1 AND id = $2`

// RestartPacketCounter records the counter restart of an over-the-air attach (radio protocol §3.6.5.3).
func (r *EndPointRepository) RestartPacketCounter(ctx context.Context, tenantID int64, endpointID int64) error {
	result, err := r.db.ExecContext(ctx, sqlRestartPacketCounter, tenantID, endpointID, r.clock.Now())
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapRestartPacketCounter, err)
	}
	restarted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapRestartPacketCounter, err)
	}
	if restarted == 0 {
		return fmt.Errorf("%s: %w", errWrapRestartPacketCounter, storage.ErrNotFound)
	}
	return nil
}

// sqlLockAttachCounter reads the attach counter state under a row lock held to the end of the transaction.
const sqlLockAttachCounter = `SELECT attach_cnt, sign FROM endpoints WHERE tenant_id = $1 AND id = $2 FOR UPDATE`

// LockAttachCounter returns the attach counter the endpoint's next
// over-the-air attach must advance, nil when none binds it, and holds the
// endpoint row until the transaction ends.
func (r *EndPointRepository) LockAttachCounter(ctx context.Context, tenantID int64, endpointID int64) (*uint32, error) {
	var stored struct {
		AttachCnt sql.NullInt64 `db:"attach_cnt"`
		Sign      []byte        `db:"sign"`
	}
	err := sqlx.GetContext(ctx, r.db, &stored, sqlLockAttachCounter, tenantID, endpointID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%s: %w", errWrapLockAttachCounter, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapLockAttachCounter, err)
	}
	attachCnt, err := attachCounter(stored.AttachCnt, endpointID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapLockAttachCounter, err)
	}
	endpoint := models.EndPoint{Sign: stored.Sign, AttachCnt: attachCnt}
	return endpoint.OverTheAirAttachCounter(), nil
}

// EndpointDetachStateUpdate writes the detach field set for all detach paths.
func (r *EndPointRepository) EndpointDetachStateUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointDetachStateParams) error {
	return runEndpointFieldUpdate(ctx, r.db, buildEndpointDetachStateSet(tenantID, endpointID, p))
}

// sqlTransitionEndpointStatus sets ep_status only when it differs, so of two
// concurrent identical transitions the second waits on the row and matches
// none; only a change records its time.
const sqlTransitionEndpointStatus = `
	UPDATE endpoints SET ep_status = $3, attachment_changed_at = $4, updated_at = $4
	WHERE tenant_id = $1 AND id = $2 AND ep_status IS DISTINCT FROM $3`

// TransitionEndpointStatus sets the endpoint's ep_status and reports whether
// this call changed it.
func (r *EndPointRepository) TransitionEndpointStatus(ctx context.Context, tenantID int64, endpointID int64, status string) (bool, error) {
	return r.transitionEndpointStatus(ctx, r.db, tenantID, endpointID, status)
}

func (r *EndPointRepository) transitionEndpointStatus(ctx context.Context, q sqlx.ExecerContext, tenantID int64, endpointID int64, status string) (bool, error) {
	result, err := q.ExecContext(ctx, sqlTransitionEndpointStatus, tenantID, endpointID, status, r.clock.Now())
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapTransitionEndpointStatus, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}
	return changed > 0, nil
}

// sqlRestateEndpointStatus sets ep_status and records the time whether or not
// the status changes; the row lock orders concurrent calls, so of two
// identical ones only the first reports a change.
const sqlRestateEndpointStatus = `
	WITH prior AS (
		SELECT id, ep_status FROM endpoints WHERE tenant_id = $1 AND id = $2 FOR UPDATE
	)
	UPDATE endpoints e SET ep_status = $3, attachment_changed_at = $4, updated_at = $4
	FROM prior WHERE e.id = prior.id
	RETURNING prior.ep_status IS DISTINCT FROM $3`

// RestateEndpointStatus sets the endpoint's ep_status and records the time
// even when the status stays, for a decision that restates what the base
// stations hold; it reports whether this call changed the status.
func (r *EndPointRepository) RestateEndpointStatus(ctx context.Context, tenantID int64, endpointID int64, status string) (bool, error) {
	var changed bool
	err := r.db.QueryRowxContext(ctx, sqlRestateEndpointStatus, tenantID, endpointID, status, r.clock.Now()).Scan(&changed)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapRestateEndpointStatus, err)
	}
	return changed, nil
}

// sqlAttachmentChangedSince selects the tenant's endpoints whose status was changed
// to, or restated as, the given one after a moment; a NULL moment selects every
// recorded one.
const sqlAttachmentChangedSince = `SELECT ` + endpointTenantLookupColumns + ` FROM endpoints
	WHERE tenant_id = $1 AND ep_status = $2
	  AND attachment_changed_at > COALESCE($3::timestamptz, '-infinity'::timestamptz)
	ORDER BY id`

// GetByAttachmentChangedSince returns the tenant's endpoints whose status was
// changed to, or restated as, status after since, or every recorded one when
// since is nil.
func (r *EndPointRepository) GetByAttachmentChangedSince(ctx context.Context, tenantID int64, status string, since *time.Time) ([]*models.EndPoint, error) {
	rows, err := r.db.QueryxContext(ctx, sqlAttachmentChangedSince, tenantID, status, since)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapAttachmentChangedSince, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.log.Warn(logMsgCloseRowsEndpoints, logger.FieldError, closeErr)
		}
	}()
	var endpoints []*models.EndPoint
	for rows.Next() {
		endpoint, scanErr := scanEndpointTenantLookupRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("%s: %w", errWrapAttachmentChangedSince, scanErr)
		}
		if err := decryptEndpointKeys(r.cipher, endpoint); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapAttachmentChangedSince, err)
	}
	return endpoints, nil
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
		return nil, fmt.Errorf("%s: %w", errWrapFailedToQueryRoamingEndpoints, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsEndpoints, logger.FieldError, err)
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
			return nil, fmt.Errorf("%s: %w", errWrapScanRoamingEndpoint, err)
		}

		// Convert hstore to map[string]string
		endpoint.Tags = make(map[string]string)
		for k, v := range tags.Map {
			endpoint.Tags[k] = v.String
		}

		if err := decryptEndpointKeys(r.cipher, endpoint); err != nil {
			return nil, err
		}

		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateRoamingEndpoints, err)
	}

	return endpoints, nil
}

// GetPreferredBsEui returns the last_attached_bs_eui for an endpoint.
// Used by UL Data Transmit (SCACI §3.9.1) for "Service Center preferred BS" selection.
// Returns (nil, false, nil) if endpoint not found or column is NULL.
func (r *EndPointRepository) GetPreferredBsEui(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error) {
	query := `SELECT last_attached_bs_eui FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`

	var preferredBs []byte
	err := r.db.QueryRowxContext(ctx, query, tenantID, epEui).Scan(&preferredBs)
	if err == sql.ErrNoRows {
		return nil, false, nil // No endpoint found
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", errWrapGetPreferredBS, err)
	}
	preferred := mioty.OptionalEUI64FromBytes(preferredBs)
	return preferred, preferred != nil, nil
}
