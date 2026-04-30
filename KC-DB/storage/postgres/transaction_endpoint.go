package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/lib/pq/hstore"
)

// transactionalEndPointRepository implements interfaces.EndPointRepository within a transaction
type transactionalEndPointRepository struct {
	tx *sql.Tx
	db *DB
}

// Create creates a new device within the transaction
func (r *transactionalEndPointRepository) Create(ctx context.Context, endpoint *models.EndPoint) error {
	// Set owner_tenant_id to tenant_id on creation if not specified (roaming support)
	if endpoint.OwnerTenantID == 0 {
		endpoint.OwnerTenantID = endpoint.TenantID
	}

	query := `
		INSERT INTO endpoints (
			ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			tags, sh_addr,
			manufacturer, model, carrier_offset,
			propagated, propagation_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at, owner_tenant_id`

	// Ensure keys are never nil (satisfy DB constraints)
	if endpoint.NwkSnKey == nil {
		endpoint.NwkSnKey = make([]byte, 16)
	}
	if endpoint.AppKey == nil {
		endpoint.AppKey = make([]byte, 16)
	}

	// Convert map[string]string to hstore
	tagsHstore := make(map[string]sql.NullString)
	for k, v := range endpoint.Tags {
		tagsHstore[k] = sql.NullString{String: v, Valid: true}
	}
	tags := hstore.Hstore{Map: tagsHstore}

	err := r.tx.QueryRowContext(ctx, query,
		endpoint.EUI[:],
		endpoint.Name,
		endpoint.Description,
		endpoint.TenantID,
		endpoint.OwnerTenantID,
		endpoint.NwkSnKey,
		endpoint.AppKey,
		endpoint.CryptoMode,
		tags,
		endpoint.ShAddr,
		endpoint.Manufacturer,
		endpoint.Model,
		endpoint.CarrierOffset,
		endpoint.Propagated,
		endpoint.PropagationCount,
	).Scan(&endpoint.ID, &endpoint.CreatedAt, &endpoint.UpdatedAt, &endpoint.OwnerTenantID)

	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	return nil
}

// Get retrieves an endpoint by EUI within the transaction
func (r *transactionalEndPointRepository) Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error) {
	query := `
		SELECT
			id, ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			last_seen_at, frame_count, battery_level,
			tags, created_at, updated_at, sh_addr,
			last_attached_bs_eui, last_propagate_time, last_detach_time,
			last_detach_sign, last_detach_packet_cnt, propagate_status
		FROM endpoints
		WHERE ep_eui = $1`

	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var lastDetachSign []byte
	var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
	var propagateStatus sql.NullString

	err := r.tx.QueryRowContext(ctx, query, eui[:]).Scan(
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
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	// Convert hstore to map[string]string
	endpoint.Tags = make(map[string]string)
	for k, v := range tags.Map {
		if v.Valid {
			endpoint.Tags[k] = v.String
		}
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

// GetByTenant retrieves all endpoints for a tenant within the transaction
func (r *transactionalEndPointRepository) GetByTenant(ctx context.Context, tenantID int64) ([]*models.EndPoint, error) {
	query := `
		SELECT
			id, ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			last_seen_at, frame_count, battery_level,
			tags, created_at, updated_at, sh_addr,
			last_attached_bs_eui, last_propagate_time, last_detach_time,
			last_detach_sign, last_detach_packet_cnt, propagate_status
		FROM endpoints
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.tx.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoints: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "GetByTenant")
		}
	}()

	var endpoints []*models.EndPoint
	for rows.Next() {
		endpoint := &models.EndPoint{}
		var tags hstore.Hstore
		var lastDetachSign []byte
		var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
		var propagateStatus sql.NullString

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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}

		// Convert hstore to map[string]string
		endpoint.Tags = make(map[string]string)
		for k, v := range tags.Map {
			if v.Valid {
				endpoint.Tags[k] = v.String
			}
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

		endpoints = append(endpoints, endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoints: %w", err)
	}

	return endpoints, nil
}

// Update updates an endpoint within the transaction
func (r *transactionalEndPointRepository) Update(ctx context.Context, endpoint *models.EndPoint) error {
	query := `
		UPDATE endpoints SET
			name = $2,
			description = $3,
			nwk_key = $4,
			app_key = $5,
			crypto_mode = $6,
			battery_level = $7,
			tags = $8,
			sh_addr = $9,
			manufacturer = $10,
			model = $11,
			carrier_offset = $12,
			bidi = $13,
			pre_attach = $14,
			dual_chan = $15,
			repetition = $16,
			wide_carr_off = $17,
			long_blk_dist = $18,
			propagated = $19,
			propagated_at = $20,
			propagation_count = $21,
			updated_at = NOW()
		WHERE ep_eui = $1
		RETURNING updated_at`

	// Convert map[string]string to hstore
	tagsHstore := make(map[string]sql.NullString)
	for k, v := range endpoint.Tags {
		tagsHstore[k] = sql.NullString{String: v, Valid: true}
	}
	tags := hstore.Hstore{Map: tagsHstore}

	err := r.tx.QueryRowContext(ctx, query,
		endpoint.EUI[:],
		endpoint.Name,
		endpoint.Description,
		endpoint.NwkSnKey,
		endpoint.AppKey,
		endpoint.CryptoMode,
		endpoint.BatteryLevel,
		tags,
		endpoint.ShAddr,
		endpoint.Manufacturer,
		endpoint.Model,
		endpoint.CarrierOffset,
		endpoint.Bidi,
		endpoint.PreAttach,
		endpoint.DualChan,
		endpoint.Repetition,
		endpoint.WideCarrOff,
		endpoint.LongBlkDist,
		endpoint.Propagated,
		endpoint.PropagatedAt,
		endpoint.PropagationCount,
	).Scan(&endpoint.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	return nil
}

// UpdateWithEUI delegates to the non-transactional EndPointRepository (EUI cascade manages its own transaction)
func (r *transactionalEndPointRepository) UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error) {
	return r.db.endPointRepo.UpdateWithEUI(ctx, tenantID, oldEui, endpoint)
}

// CheckEUIUnique delegates to the non-transactional EndPointRepository
func (r *transactionalEndPointRepository) CheckEUIUnique(ctx context.Context, eui []byte) error {
	return r.db.endPointRepo.CheckEUIUnique(ctx, eui)
}

// DeleteByTenant deletes an endpoint by EUI with tenant isolation within the transaction
// Returns storage.ErrNotFound when 0 rows affected
func (r *transactionalEndPointRepository) DeleteByTenant(ctx context.Context, tenantID int64, eui []byte) error {
	query := `DELETE FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`

	result, err := r.tx.ExecContext(ctx, query, tenantID, eui)
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

// UpdateLastSeen updates the last seen timestamp and frame count with tenant isolation within the transaction
func (r *transactionalEndPointRepository) UpdateLastSeen(ctx context.Context, tenantID int64, eui models.EUI, frameCount uint32) error {
	query := `
		UPDATE endpoints SET
			last_seen_at = NOW(),
			frame_count = $3,
			updated_at = NOW()
		WHERE ep_eui = $1 AND tenant_id = $2`

	result, err := r.tx.ExecContext(ctx, query, eui[:], tenantID, frameCount)
	if err != nil {
		return fmt.Errorf("failed to update endpoint last seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateRadioMetrics updates the radio metrics from attach or uplink operations within the transaction
func (r *transactionalEndPointRepository) UpdateRadioMetrics(ctx context.Context, tenantID int64, eui models.EUI, snr, rssi, eqSnr float64, rxTime, rxDuration int64, profile string) error {
	query := `
		UPDATE endpoints SET
			last_snr = $3,
			last_rssi = $4,
			last_eq_snr = $5,
			last_attach_rx_time = $6,
			last_attach_rx_duration = $7,
			last_profile = $8,
			last_seen_at = CURRENT_TIMESTAMP,
			updated_at = NOW()
		WHERE tenant_id = $1 AND ep_eui = $2`

	result, err := r.tx.ExecContext(ctx, query, tenantID, eui[:], snr, rssi, eqSnr, rxTime, rxDuration, profile)
	if err != nil {
		return fmt.Errorf("failed to update endpoint radio metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("endpoint not found")
	}

	return nil
}

// UpdateRadioMetricsSelective updates radio metrics with optional field support within the transaction.
// Nil pointers in the update struct preserve existing database values.
func (r *transactionalEndPointRepository) UpdateRadioMetricsSelective(ctx context.Context, tenantID int64, eui models.EUI, update interfaces.RadioMetricsUpdate) error {
	// Build dynamic SET clause
	setClauses := []string{
		"last_snr = $3",
		"last_rssi = $4",
		"last_eq_snr = $5",
		"last_attach_rx_time = $6",
		"last_seen_at = CURRENT_TIMESTAMP",
		"updated_at = NOW()",
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

	// #nosec G201 -- setClauses are constructed from safe column names with parameterized values
	query := fmt.Sprintf(`
		UPDATE endpoints SET
			%s
		WHERE tenant_id = $1 AND ep_eui = $2`,
		strings.Join(setClauses, ",\n			"))

	result, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update endpoint radio metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("endpoint not found")
	}

	return nil
}

// GetByEUI retrieves an endpoint by EUI for a specific tenant within the transaction
func (r *transactionalEndPointRepository) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) {
	query := `
		SELECT
			id, ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			last_seen_at, frame_count, battery_level,
			tags, created_at, updated_at, sh_addr,
			last_attached_bs_eui, last_propagate_time, last_detach_time,
			last_detach_sign, last_detach_packet_cnt, propagate_status,
			last_attach_rx_time, last_attach_rx_duration,
			last_snr, last_rssi, last_eq_snr, last_profile, last_attach_subpackets
		FROM endpoints
		WHERE tenant_id = $1 AND ep_eui = $2`

	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var lastDetachSign []byte
	var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
	var propagateStatus sql.NullString
	var lastAttachRxTime, lastAttachRxDuration sql.NullInt64
	var lastSNR, lastRSSI, lastEqSNR sql.NullFloat64
	var lastProfile, lastAttachSubpackets sql.NullString

	err := r.tx.QueryRowContext(ctx, query, tenantID, eui).Scan(
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
		// Radio metrics fields (BSSCI §3.6.1/3.7.1 attach/detach telemetry)
		&lastAttachRxTime, &lastAttachRxDuration,
		&lastSNR, &lastRSSI, &lastEqSNR, &lastProfile, &lastAttachSubpackets,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("endpoint not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
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

	// Radio metrics nullable fields (BSSCI §3.6.1/3.7.1 attach/detach telemetry)
	if lastAttachRxTime.Valid {
		val := lastAttachRxTime.Int64
		endpoint.LastAttachRxTime = &val
	}
	if lastAttachRxDuration.Valid {
		val := lastAttachRxDuration.Int64
		endpoint.LastAttachRxDuration = &val
	}
	if lastSNR.Valid {
		val := lastSNR.Float64
		endpoint.LastSNR = &val
	}
	if lastRSSI.Valid {
		val := lastRSSI.Float64
		endpoint.LastRSSI = &val
	}
	if lastEqSNR.Valid {
		val := lastEqSNR.Float64
		endpoint.LastEqSNR = &val
	}
	if lastProfile.Valid {
		val := lastProfile.String
		endpoint.LastProfile = &val
	}
	// BSSCI-ATTACH-024: Subpackets parity with GetByID
	if lastAttachSubpackets.Valid {
		val := lastAttachSubpackets.String
		endpoint.LastAttachSubpackets = &val
	}

	return endpoint, nil
}

// UpdateFields updates endpoint fields using a map of field names to values within the transaction
func (r *transactionalEndPointRepository) UpdateFields(ctx context.Context, tenantID int64, endpointID int64, updates map[string]interface{}) error {
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

	// #nosec G201 -- setParts are constructed from safe column names with parameterized values
	query := fmt.Sprintf(`
		UPDATE endpoints SET
			%s,
			updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2`,
		strings.Join(setParts, ", "))

	result, err := r.tx.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("endpoint not found")
	}

	return nil
}

// CountByTenant returns the total count of endpoints for a tenant within the transaction
func (r *transactionalEndPointRepository) CountByTenant(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM endpoints WHERE tenant_id = $1`

	err := r.tx.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count endpoints: %w", err)
	}

	return count, nil
}

// ListByTenantPaginated retrieves paginated endpoints for a tenant with LIMIT/OFFSET within the transaction
func (r *transactionalEndPointRepository) ListByTenantPaginated(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) {
	query := `SELECT` + endpointListSelectColumns + `
		FROM endpoints
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.tx.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query paginated endpoints: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "ListByTenantPaginated")
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
		return nil, fmt.Errorf("error iterating endpoints: %w", err)
	}

	return endpoints, nil
}

// GetByID retrieves an endpoint by ID with tenant isolation within the transaction
func (r *transactionalEndPointRepository) GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	query := `
		SELECT
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
			last_packet_cnt,
			last_user_data, last_format_id, last_mode,
			last_rx_time, last_rx_duration, packet_cnt,
			last_dl_open, last_response_exp, last_dl_ack,
			endpoint_class
		FROM endpoints
		WHERE id = $1 AND tenant_id = $2`

	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var typeEUIBytes []byte

	// Nullable scan targets
	var attachCnt sql.NullInt64 // uint32 stored as BIGINT with CHECK constraint 0-4294967295
	var lastFormatID sql.NullInt32
	var nonce, sign, lastUserData, lastDetachSign []byte
	var lastAttachRxTime, lastAttachRxDuration, lastRxTime, lastRxDuration sql.NullInt64
	var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
	var lastSNR, lastRSSI, lastEqSNR sql.NullFloat64
	var lastProfile, lastAttachSubpackets, lastMode, propagateStatus sql.NullString

	err := r.tx.QueryRowContext(ctx, query, id, tenantID).Scan(
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
		&attachCnt, &nonce, &sign, &lastAttachRxTime, &lastAttachRxDuration,
		// Radio metrics
		&lastSNR, &lastRSSI, &lastEqSNR, &lastProfile, &lastAttachSubpackets,
		// Detach fields (BSSCI §5.7)
		&lastAttachedBsEui, &lastPropagateTime, &lastDetachTime,
		&lastDetachSign, &lastDetachPacketCnt, &propagateStatus,
		// UL deduplication
		&endpoint.LastPacketCnt,
		// UL data
		&lastUserData, &lastFormatID, &lastMode,
		// UL reception
		&lastRxTime, &lastRxDuration, &endpoint.PacketCnt,
		// Downlink control
		&endpoint.LastDlOpen, &endpoint.LastResponseExp, &endpoint.LastDlAck,
		// Compatibility
		&endpoint.EPClass,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("endpoint not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint by ID: %w", err)
	}

	// Convert typeEUIBytes to *models.EUI
	if len(typeEUIBytes) == 8 {
		var typeEUI models.EUI
		copy(typeEUI[:], typeEUIBytes)
		endpoint.TypeEUI = &typeEUI
	}

	// Convert hstore to map
	if tags.Map != nil {
		endpoint.Tags = make(map[string]string)
		for k, v := range tags.Map {
			endpoint.Tags[k] = v.String
		}
	} else {
		endpoint.Tags = make(map[string]string)
	}

	// Convert nullable fields to pointers (AttachCnt: BIGINT → uint32)
	if attachCnt.Valid {
		val := uint32(attachCnt.Int64) // #nosec G115 -- DB CHECK constraint ensures 0-4294967295
		endpoint.AttachCnt = &val
	}
	if lastFormatID.Valid {
		val := lastFormatID.Int32
		endpoint.LastFormatID = &val
	}

	// Byte slices (direct assignment if non-empty)
	if len(nonce) > 0 {
		endpoint.Nonce = nonce
	}
	if len(sign) > 0 {
		endpoint.Sign = sign
	}
	if len(lastUserData) > 0 {
		endpoint.LastUserData = lastUserData
	}
	if len(lastDetachSign) > 0 {
		endpoint.LastDetachSign = lastDetachSign
	}

	// Nullable int64 fields
	if lastAttachRxTime.Valid {
		val := lastAttachRxTime.Int64
		endpoint.LastAttachRxTime = &val
	}
	if lastAttachRxDuration.Valid {
		val := lastAttachRxDuration.Int64
		endpoint.LastAttachRxDuration = &val
	}
	if lastRxTime.Valid {
		val := lastRxTime.Int64
		endpoint.LastRxTime = &val
	}
	if lastRxDuration.Valid {
		val := lastRxDuration.Int64
		endpoint.LastRxDuration = &val
	}
	// Detach nullable int64 fields (BSSCI §5.7)
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

	// Nullable float64 fields
	if lastSNR.Valid {
		val := lastSNR.Float64
		endpoint.LastSNR = &val
	}
	if lastRSSI.Valid {
		val := lastRSSI.Float64
		endpoint.LastRSSI = &val
	}
	if lastEqSNR.Valid {
		val := lastEqSNR.Float64
		endpoint.LastEqSNR = &val
	}

	// Nullable string fields
	if lastProfile.Valid {
		val := lastProfile.String
		endpoint.LastProfile = &val
	}
	if lastAttachSubpackets.Valid {
		val := lastAttachSubpackets.String
		endpoint.LastAttachSubpackets = &val
	}
	if lastMode.Valid {
		val := lastMode.String
		endpoint.LastMode = &val
	}
	// Detach nullable string fields (BSSCI §5.7)
	if propagateStatus.Valid {
		val := propagateStatus.String
		endpoint.PropagateStatus = &val
	}

	return endpoint, nil
}

// StreamAllForPropagation streams endpoints for propagation using the transaction
func (r *transactionalEndPointRepository) StreamAllForPropagation(ctx context.Context, cursorID int64, limit int) ([]*models.EndPoint, error) {
	query := `
		SELECT id, ep_eui, tenant_id, owner_tenant_id, nwk_key, bidi, sh_addr, dual_chan,
		       repetition, wide_carr_off, long_blk_dist, propagated_at, last_packet_cnt
		FROM endpoints
		WHERE propagated = TRUE AND id > $1
		ORDER BY id
		LIMIT $2
	`

	rows, err := r.tx.QueryContext(ctx, query, cursorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to stream endpoints for propagation: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "StreamAllForPropagation")
		}
	}()

	var endpoints []*models.EndPoint
	for rows.Next() {
		endpoint := &models.EndPoint{}
		var shAddr sql.NullInt32 // INTEGER for uint16 range per SCACI §3.6.1
		var propagatedAt sql.NullTime

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.EUI,
			&endpoint.TenantID,
			&endpoint.OwnerTenantID,
			&endpoint.NwkSnKey,
			&endpoint.Bidi,
			&shAddr,
			&endpoint.DualChan,
			&endpoint.Repetition,
			&endpoint.WideCarrOff,
			&endpoint.LongBlkDist,
			&propagatedAt,
			&endpoint.LastPacketCnt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}

		if shAddr.Valid {
			val := uint16(shAddr.Int32) // #nosec G115 -- DB CHECK constraint ensures 0-65535
			endpoint.ShAddr = &val
		}
		if propagatedAt.Valid {
			endpoint.PropagatedAt = &propagatedAt.Time
		}

		endpoints = append(endpoints, endpoint)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoints: %w", err)
	}

	return endpoints, nil
}

// HasEndpointsSince checks if there are endpoints updated since the given time using the transaction
func (r *transactionalEndPointRepository) HasEndpointsSince(ctx context.Context, since time.Time) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM endpoints
			WHERE propagated = TRUE
			  AND (propagated_at IS NULL OR propagated_at > $1)
			LIMIT 1
		)
	`

	var exists bool
	err := r.tx.QueryRowContext(ctx, query, since).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check endpoints since timestamp: %w", err)
	}

	return exists, nil
}

// UpdateDetachMetrics updates detach-related telemetry for an endpoint using the transaction
func (r *transactionalEndPointRepository) UpdateDetachMetrics(ctx context.Context, tenantID int64, eui models.EUI, update interfaces.DetachMetricsUpdate) error {
	// Build SET clauses dynamically based on provided fields
	setClauses := []string{
		"last_detach_rx_time = $3",
		"last_detach_packet_cnt = $4",
		"last_detach_sign = $5",
		"last_snr = $6",
		"last_rssi = $7",
		"last_seen_at = CURRENT_TIMESTAMP",
	}
	args := []interface{}{tenantID, eui[:], update.RxTime, update.PacketCnt, update.Signature, update.SNR, update.RSSI}

	// Add optional fields if provided
	argPos := 8
	if update.EqSNR != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_eq_snr = $%d", argPos))
		args = append(args, *update.EqSNR)
		argPos++
	}
	if update.Profile != nil {
		setClauses = append(setClauses, fmt.Sprintf("profile = $%d", argPos))
		args = append(args, *update.Profile)
	}

	// #nosec G201 -- setClauses are constructed from safe column names with parameterized values
	query := fmt.Sprintf(`
		UPDATE end_points
		SET %s
		WHERE tenant_id = $1 AND ep_eui = $2`,
		strings.Join(setClauses, ", "))

	result, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update detach metrics: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetEndpointWithKeysForDetachValidation retrieves an endpoint with all crypto material for detach signature validation within the transaction
// Returns complete endpoint record including Sign, NwkSnKey, and PresharedKey fields
func (r *transactionalEndPointRepository) GetEndpointWithKeysForDetachValidation(ctx context.Context, eui models.EUI) (*models.EndPoint, error) {
	query := `
		SELECT id, ep_eui, name, description, tenant_id, owner_tenant_id,
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
		       tags, created_at, updated_at
		FROM endpoints
		WHERE ep_eui = $1
		LIMIT 1`

	endpoint := &models.EndPoint{}
	var tags hstore.Hstore
	var typeEui, propagatedAt, lastAttachRxTime, lastAttachRxDuration sql.NullInt64
	var lastSNR, lastRSSI, lastEqSNR sql.NullFloat64
	var lastProfile, lastAttachSubpackets sql.NullString
	var lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
	var propagateStatus sql.NullString
	var lastFormatID sql.NullInt32
	var lastMode sql.NullString
	var lastRxTime, lastRxDuration sql.NullInt64
	var lastSeenAt sql.NullTime
	var batteryLevel sql.NullFloat64

	err := r.tx.QueryRowContext(ctx, query, eui[:]).Scan(
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
		&typeEui,
		&endpoint.Manufacturer,
		&endpoint.Model,
		&endpoint.CarrierOffset,
		&endpoint.Propagated,
		&propagatedAt,
		&endpoint.PropagationCount,
		&endpoint.DualChan,
		&endpoint.Repetition,
		&endpoint.WideCarrOff,
		&endpoint.LongBlkDist,
		&endpoint.AttachCnt,
		&endpoint.Nonce,
		&endpoint.Sign,
		&endpoint.PresharedKey,
		&lastAttachRxTime,
		&lastAttachRxDuration,
		&lastSNR,
		&lastRSSI,
		&lastEqSNR,
		&lastProfile,
		&lastAttachSubpackets,
		&lastAttachedBsEui,
		&lastPropagateTime,
		&lastDetachTime,
		&endpoint.LastDetachSign,
		&lastDetachPacketCnt,
		&propagateStatus,
		&endpoint.LastPacketCnt,
		&endpoint.LastUserData,
		&lastFormatID,
		&lastMode,
		&lastRxTime,
		&lastRxDuration,
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
	)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get endpoint with keys for detach validation: %w", err)
	}

	// Convert hstore to map[string]string
	endpoint.Tags = make(map[string]string)
	for k, v := range tags.Map {
		endpoint.Tags[k] = v.String
	}

	// Handle nullable fields
	if typeEui.Valid {
		euiVal := models.EUI{}
		// Convert int64 to EUI bytes
		for i := 7; i >= 0; i-- {
			euiVal[i] = byte(typeEui.Int64 & 0xFF)
			typeEui.Int64 >>= 8
		}
		endpoint.TypeEUI = &euiVal
	}
	if propagatedAt.Valid {
		t := time.Unix(0, propagatedAt.Int64)
		endpoint.PropagatedAt = &t
	}
	if lastAttachRxTime.Valid {
		val := lastAttachRxTime.Int64
		endpoint.LastAttachRxTime = &val
	}
	if lastAttachRxDuration.Valid {
		val := lastAttachRxDuration.Int64
		endpoint.LastAttachRxDuration = &val
	}
	if lastSNR.Valid {
		val := lastSNR.Float64
		endpoint.LastSNR = &val
	}
	if lastRSSI.Valid {
		val := lastRSSI.Float64
		endpoint.LastRSSI = &val
	}
	if lastEqSNR.Valid {
		val := lastEqSNR.Float64
		endpoint.LastEqSNR = &val
	}
	if lastProfile.Valid {
		val := lastProfile.String
		endpoint.LastProfile = &val
	}
	if lastAttachSubpackets.Valid {
		val := lastAttachSubpackets.String
		endpoint.LastAttachSubpackets = &val
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
	if lastFormatID.Valid {
		val := lastFormatID.Int32
		endpoint.LastFormatID = &val
	}
	if lastMode.Valid {
		val := lastMode.String
		endpoint.LastMode = &val
	}
	if lastRxTime.Valid {
		val := lastRxTime.Int64
		endpoint.LastRxTime = &val
	}
	if lastRxDuration.Valid {
		val := lastRxDuration.Int64
		endpoint.LastRxDuration = &val
	}
	if lastSeenAt.Valid {
		endpoint.LastSeenAt = &lastSeenAt.Time
	}
	if batteryLevel.Valid {
		val := float32(batteryLevel.Float64)
		endpoint.BatteryLevel = &val
	}

	return endpoint, nil
}

// GetPreferredBsEui returns the last_attached_bs_eui for an endpoint within the transaction.
// Used by UL Data Transmit (SCACI §3.9.1) for "Service Center preferred BS" selection.
// Returns (nil, false, nil) if endpoint not found or column is NULL.
func (r *transactionalEndPointRepository) GetPreferredBsEui(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error) {
	query := `SELECT last_attached_bs_eui FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`

	var preferredBs sql.NullInt64
	err := r.tx.QueryRowContext(ctx, query, tenantID, epEui).Scan(&preferredBs)
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
