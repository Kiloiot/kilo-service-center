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
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
)

// BaseStationRepository implements the BaseStationRepository interface for PostgreSQL
type BaseStationRepository struct {
	db *sqlx.DB
}

// NewBaseStationRepository creates a new PostgreSQL Base Station repository
func NewBaseStationRepository(db *sqlx.DB) *BaseStationRepository {
	return &BaseStationRepository{db: db}
}

// Create creates a new Base Station
func (r *BaseStationRepository) Create(ctx context.Context, bs *models.BaseStation) error {
	query := `
		INSERT INTO basestations (
			tenant_id, bs_eui, name, description,
			connection_type, service_center_url,
			tls_ca_certificate, tls_certificate, tls_key,
			tls_auth_required, tls_hostname_verification,
			mqtt_broker_url, mqtt_client_id, mqtt_username, mqtt_password_encrypted, mqtt_topic_prefix,
			latitude, longitude, altitude, location_source, location_updated_at,
			config_file_content, config_file_uploaded_at,
			tags,
			created_at, updated_at
		) VALUES (
			:tenant_id, :bs_eui, :name, :description,
			:connection_type, :service_center_url,
			:tls_ca_certificate, :tls_certificate, :tls_key,
			:tls_auth_required, :tls_hostname_verification,
			:mqtt_broker_url, :mqtt_client_id, :mqtt_username, :mqtt_password_encrypted, :mqtt_topic_prefix,
			:latitude, :longitude, :altitude, :location_source, :location_updated_at,
			:config_file_content, :config_file_uploaded_at,
			:tags,
			NOW(), NOW()
		) RETURNING id, created_at, updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close statement in basestation creation: %v", err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, bs).Scan(&bs.ID, &bs.CreatedAt, &bs.UpdatedAt)
	if err != nil {
		return WrapDuplicateError(err, "base station")
	}

	return nil
}

// GetByID retrieves a Base Station by ID
func (r *BaseStationRepository) GetByID(ctx context.Context, tenantID, id int64) (*models.BaseStation, error) {
	var bs models.BaseStation
	query := `
		SELECT 
			id, tenant_id, bs_eui, name, description,
			connection_type, service_center_url,
			tls_ca_certificate, tls_certificate, tls_key,
			tls_auth_required, tls_hostname_verification,
			tls_cert_fingerprint, tls_cert_expires_at,
			mqtt_broker_url, mqtt_client_id, mqtt_username, mqtt_password_encrypted, mqtt_topic_prefix,
			is_online, last_seen_at, session_uuid, session_started_at,
			uptime, status_code, status_message,
			tags, bidi, ul_load, dl_load, last_modified_by, last_modified_at,
			vendor, model, sw_version,
			latitude, longitude, altitude, location_source, location_updated_at,
			config_file_content, config_file_uploaded_at,
			connection_status, last_error, retry_count, next_retry_at,
			created_at, updated_at
		FROM basestations
		WHERE tenant_id = $1 AND id = $2`

	err := r.db.GetContext(ctx, &bs, query, tenantID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("base station not found")
		}
		return nil, fmt.Errorf("get base station: %w", err)
	}

	return &bs, nil
}

// GetByEUI retrieves a Base Station by EUI
func (r *BaseStationRepository) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	var bs models.BaseStation
	query := `
		SELECT
			id, tenant_id, bs_eui, name, description,
			connection_type, service_center_url,
			tls_ca_certificate, tls_certificate, tls_key,
			tls_auth_required, tls_hostname_verification,
			tls_cert_fingerprint, tls_cert_expires_at,
			mqtt_broker_url, mqtt_client_id, mqtt_username, mqtt_password_encrypted, mqtt_topic_prefix,
			is_online, last_seen_at, session_uuid, session_started_at,
			uptime, status_code, status_message,
			tags, bidi, ul_load, dl_load, last_modified_by, last_modified_at,
			vendor, model, sw_version,
			latitude, longitude, altitude, location_source, location_updated_at,
			config_file_content, config_file_uploaded_at,
			connection_status, last_error, retry_count, next_retry_at,
			system_time, duty_cycle, uptime_seconds, temperature_celsius,
			cpu_load, memory_load, bs_config, last_status_at,
			created_at, updated_at
		FROM basestations
		WHERE tenant_id = $1 AND bs_eui = $2`

	err := r.db.GetContext(ctx, &bs, query, tenantID, eui)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("base station not found")
		}
		return nil, fmt.Errorf("get base station: %w", err)
	}

	return &bs, nil
}

// GetByEUIGlobal retrieves a Base Station by EUI across all tenants.
// Used during BSSCI connect handshake when tenant is not yet resolved.
func (r *BaseStationRepository) GetByEUIGlobal(ctx context.Context, eui []byte) (*models.BaseStation, error) {
	var bs models.BaseStation
	query := `
		SELECT
			id, tenant_id, bs_eui, name, description,
			connection_type, service_center_url,
			tls_ca_certificate, tls_certificate, tls_key,
			tls_auth_required, tls_hostname_verification,
			tls_cert_fingerprint, tls_cert_expires_at,
			mqtt_broker_url, mqtt_client_id, mqtt_username, mqtt_password_encrypted, mqtt_topic_prefix,
			is_online, last_seen_at, session_uuid, session_started_at,
			uptime, status_code, status_message,
			tags, bidi, ul_load, dl_load, last_modified_by, last_modified_at,
			vendor, model, sw_version,
			latitude, longitude, altitude, location_source, location_updated_at,
			config_file_content, config_file_uploaded_at,
			connection_status, last_error, retry_count, next_retry_at,
			system_time, duty_cycle, uptime_seconds, temperature_celsius,
			cpu_load, memory_load, bs_config, last_status_at,
			created_at, updated_at
		FROM basestations
		WHERE bs_eui = $1`

	err := r.db.GetContext(ctx, &bs, query, eui)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("base station not found")
		}
		return nil, fmt.Errorf("get base station: %w", err)
	}

	return &bs, nil
}

// ListAllLocations retrieves all base stations with non-null coordinates across all tenants.
func (r *BaseStationRepository) ListAllLocations(ctx context.Context) ([]*models.BaseStation, error) {
	query := `
		SELECT id, tenant_id, bs_eui, name,
		       latitude, longitude, altitude, location_source, is_online
		FROM basestations
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		ORDER BY tenant_id, name`

	var stations []*models.BaseStation
	if err := r.db.SelectContext(ctx, &stations, query); err != nil {
		return nil, fmt.Errorf("list all locations: %w", err)
	}
	return stations, nil
}

// Update updates an existing Base Station
func (r *BaseStationRepository) Update(ctx context.Context, tenantID, id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setClauses := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+3)
	args = append(args, tenantID, id)

	i := 3
	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, i))
		args = append(args, value)
		i++
	}

	// Always update updated_at
	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE basestations 
		SET %s 
		WHERE tenant_id = $1 AND id = $2`,
		strings.Join(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update base station: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("base station not found")
	}

	return nil
}

// Delete deletes a Base Station
func (r *BaseStationRepository) Delete(ctx context.Context, tenantID, id int64) error {
	query := `DELETE FROM basestations WHERE tenant_id = $1 AND id = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete base station: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("base station not found")
	}

	return nil
}

// List retrieves Base Stations based on filter criteria
func (r *BaseStationRepository) List(ctx context.Context, filter *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	// Build WHERE clauses
	whereClauses := []string{"tenant_id = $1"}
	args := []interface{}{filter.TenantID}
	argCount := 2

	if filter.ConnectionType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("connection_type = $%d", argCount))
		args = append(args, *filter.ConnectionType)
		argCount++
	}

	if filter.IsOnline != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_online = $%d", argCount))
		args = append(args, *filter.IsOnline)
		argCount++
	}

	if filter.Search != "" {
		searchClause := fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR encode(bs_eui, 'hex') ILIKE $%d)",
			argCount, argCount, argCount)
		whereClauses = append(whereClauses, searchClause)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern)
		argCount++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total matching records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM basestations WHERE %s", whereSQL)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count base stations: %w", err)
	}

	// Get paginated results
	var query string
	if filter.Limit > 0 {
		// Apply pagination when Limit is specified
		query = fmt.Sprintf(`
			SELECT
				id, tenant_id, bs_eui, name, description,
				connection_type, service_center_url,
				tls_ca_certificate, tls_certificate, tls_key,
				tls_auth_required, tls_hostname_verification,
				tls_cert_fingerprint, tls_cert_expires_at,
				mqtt_broker_url, mqtt_client_id, mqtt_username, mqtt_password_encrypted, mqtt_topic_prefix,
				is_online, last_seen_at, session_uuid, session_started_at,
				uptime, status_code, status_message,
				tags, bidi, ul_load, dl_load, last_modified_by, last_modified_at,
				vendor, model, sw_version,
				latitude, longitude, altitude,
				config_file_content, config_file_uploaded_at,
				connection_status, last_error, retry_count, next_retry_at,
				system_time, duty_cycle, uptime_seconds, temperature_celsius,
				cpu_load, memory_load, bs_config, last_status_at,
				created_at, updated_at
			FROM basestations
			WHERE %s
			ORDER BY name ASC, id ASC
			LIMIT $%d OFFSET $%d`,
			whereSQL, argCount, argCount+1)
		args = append(args, filter.Limit, filter.Offset)
	} else {
		// Fetch all matching records when Limit == 0
		query = fmt.Sprintf(`
			SELECT
				id, tenant_id, bs_eui, name, description,
				connection_type, service_center_url,
				tls_ca_certificate, tls_certificate, tls_key,
				tls_auth_required, tls_hostname_verification,
				tls_cert_fingerprint, tls_cert_expires_at,
				mqtt_broker_url, mqtt_client_id, mqtt_username, mqtt_password_encrypted, mqtt_topic_prefix,
				is_online, last_seen_at, session_uuid, session_started_at,
				uptime, status_code, status_message,
				tags, bidi, ul_load, dl_load, last_modified_by, last_modified_at,
				vendor, model, sw_version,
				latitude, longitude, altitude,
				config_file_content, config_file_uploaded_at,
				connection_status, last_error, retry_count, next_retry_at,
				system_time, duty_cycle, uptime_seconds, temperature_celsius,
				cpu_load, memory_load, bs_config, last_status_at,
				created_at, updated_at
			FROM basestations
			WHERE %s
			ORDER BY name ASC, id ASC`,
			whereSQL)
		// No LIMIT/OFFSET args appended
	}

	var baseStations []*models.BaseStation
	err = r.db.SelectContext(ctx, &baseStations, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list base stations: %w", err)
	}

	return baseStations, total, nil
}

// UpdateConnectionStatus updates the connection status of a Base Station
func (r *BaseStationRepository) UpdateConnectionStatus(ctx context.Context, tenantID, id int64, isOnline bool, lastError *string) error {
	updates := map[string]interface{}{
		"is_online":    isOnline,
		"last_seen_at": time.Now(),
	}

	if isOnline {
		updates["last_error"] = nil
		updates["retry_count"] = 0
		updates["next_retry_at"] = nil
	} else if lastError != nil {
		updates["last_error"] = *lastError
		// Increment retry count and calculate next retry with exponential backoff
		var retryCount int
		err := r.db.GetContext(ctx, &retryCount,
			"SELECT retry_count FROM basestations WHERE tenant_id = $1 AND id = $2",
			tenantID, id)
		if err == nil {
			retryCount++
			updates["retry_count"] = retryCount
			// Exponential backoff: 2^retry_count seconds, max 1 hour
			backoffSeconds := 1 << retryCount
			if backoffSeconds > 3600 {
				backoffSeconds = 3600
			}
			updates["next_retry_at"] = time.Now().Add(time.Duration(backoffSeconds) * time.Second)
		}
	}

	// Update connection status JSON
	statusJSON, _ := json.Marshal(map[string]interface{}{
		"is_online":  isOnline,
		"last_seen":  time.Now(),
		"last_error": lastError,
	})
	updates["connection_status"] = statusJSON

	return r.Update(ctx, tenantID, id, updates)
}

// UpdateSessionInfo updates the session information of a Base Station
func (r *BaseStationRepository) UpdateSessionInfo(ctx context.Context, tenantID int64, eui []byte, sessionUUID string) error {
	query := `
		UPDATE basestations 
		SET session_uuid = $1, 
		    session_started_at = NOW(),
		    is_online = true,
		    last_seen_at = NOW(),
		    updated_at = NOW()
		WHERE tenant_id = $2 AND bs_eui = $3`

	result, err := r.db.ExecContext(ctx, query, sessionUUID, tenantID, eui)
	if err != nil {
		return fmt.Errorf("update session info: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("base station not found")
	}

	return nil
}

// GetStatistics retrieves statistics for Base Stations
func (r *BaseStationRepository) GetStatistics(ctx context.Context, tenantID int64) (*interfaces.BaseStationStatistics, error) {
	var stats interfaces.BaseStationStatistics

	query := `
		SELECT 
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE is_online = true) as online_count,
			COUNT(*) FILTER (WHERE is_online = false) as offline_count,
			COUNT(*) FILTER (WHERE connection_type = 'bssci') as bssci_count,
			COUNT(*) FILTER (WHERE connection_type = 'mqtt') as mqtt_count
		FROM basestations
		WHERE tenant_id = $1`

	err := r.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get statistics: %w", err)
	}

	return &stats, nil
}

// euiToUint64 converts 8-byte EUI to uint64 for BIGINT columns in messages tables
func euiToUint64(eui []byte) uint64 {
	if len(eui) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(eui)
}

// euiToInt64 converts 8-byte EUI to int64 for endpoints.last_attached_bs_eui
func euiToInt64(eui []byte) int64 {
	if len(eui) != 8 {
		return 0
	}
	// #nosec G115 - EUI values are opaque identifiers; overflow to negative is acceptable for storage
	return int64(binary.BigEndian.Uint64(eui))
}

// UpdateEUI updates the Base Station EUI with transactional cascade to all dependent tables
func (r *BaseStationRepository) UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error) {
	// Validate EUI lengths
	if len(oldEui) != 8 || len(newEui) != 8 {
		return nil, fmt.Errorf("invalid EUI length: expected 8 bytes")
	}

	// Start transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1. Validate global uniqueness: reject if new_eui already exists (any tenant)
	var existsCount int
	err = tx.GetContext(ctx, &existsCount, "SELECT COUNT(*) FROM basestations WHERE bs_eui = $1", newEui)
	if err != nil {
		return nil, fmt.Errorf("check EUI uniqueness: %w", err)
	}
	if existsCount > 0 {
		return nil, storage.ErrAlreadyExists
	}

	// 2. Verify caller owns the base station with oldEui
	var bsID int64
	err = tx.GetContext(ctx, &bsID, "SELECT id FROM basestations WHERE tenant_id = $1 AND bs_eui = $2", tenantID, oldEui)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get base station: %w", err)
	}

	// Calculate integer representations for BIGINT columns
	oldUint64 := euiToUint64(oldEui)
	newUint64 := euiToUint64(newEui)
	oldInt64 := euiToInt64(oldEui)
	newInt64 := euiToInt64(newEui)

	// 3. Update all tables with bs_eui reference (transactional cascade)

	// BYTEA columns: basestations.bs_eui (primary)
	// Reset online status and clear session - EUI change invalidates any active BSSCI session
	_, err = tx.ExecContext(ctx, "UPDATE basestations SET bs_eui = $1, updated_at = NOW(), is_online = false, session_uuid = NULL WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update basestations: %w", err)
	}

	// BYTEA columns: downlink_queue.bs_eui, downlink_queue.tx_bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE downlink_queue SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update downlink_queue.bs_eui: %w", err)
	}
	_, err = tx.ExecContext(ctx, "UPDATE downlink_queue SET tx_bs_eui = $1 WHERE tx_bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update downlink_queue.tx_bs_eui: %w", err)
	}

	// BYTEA columns: dl_rx_status.bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update dl_rx_status: %w", err)
	}

	// BYTEA columns: dl_rx_status_queries.bs_eui_actual
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status_queries SET bs_eui_actual = $1 WHERE bs_eui_actual = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update dl_rx_status_queries: %w", err)
	}

	// BYTEA columns: roaming_events.from_bs_eui, roaming_events.to_bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE roaming_events SET from_bs_eui = $1 WHERE from_bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update roaming_events.from_bs_eui: %w", err)
	}
	_, err = tx.ExecContext(ctx, "UPDATE roaming_events SET to_bs_eui = $1 WHERE to_bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update roaming_events.to_bs_eui: %w", err)
	}

	// BYTEA columns: mioty_basestation_status.basestation_eui
	_, err = tx.ExecContext(ctx, "UPDATE mioty_basestation_status SET basestation_eui = $1 WHERE basestation_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("update mioty_basestation_status: %w", err)
	}

	// Schema-aware update: messages.bs_eui can be BYTEA (migration 001) or BIGINT (migration 047)
	var messagesColType string
	err = tx.GetContext(ctx, &messagesColType, `
		SELECT data_type
		FROM information_schema.columns
		WHERE table_name = 'messages' AND column_name = 'bs_eui'
	`)
	if err != nil {
		return nil, fmt.Errorf("detect messages column type: %w", err)
	}

	if messagesColType == "bytea" {
		// BYTEA columns: use raw bytes
		_, err = tx.ExecContext(ctx, "UPDATE messages SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
		if err != nil {
			return nil, fmt.Errorf("update messages (bytea): %w", err)
		}
		_, err = tx.ExecContext(ctx, "UPDATE messages_archive SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
		if err != nil {
			return nil, fmt.Errorf("update messages_archive (bytea): %w", err)
		}
	} else {
		// BIGINT columns (uint64 conversion): messages.bs_eui
		_, err = tx.ExecContext(ctx, "UPDATE messages SET bs_eui = $1 WHERE bs_eui = $2", newUint64, oldUint64)
		if err != nil {
			return nil, fmt.Errorf("update messages: %w", err)
		}
		// BIGINT columns (uint64 conversion): messages_archive.bs_eui
		_, err = tx.ExecContext(ctx, "UPDATE messages_archive SET bs_eui = $1 WHERE bs_eui = $2", newUint64, oldUint64)
		if err != nil {
			return nil, fmt.Errorf("update messages_archive: %w", err)
		}
	}

	// BIGINT columns (int64 conversion): endpoints.last_attached_bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE endpoints SET last_attached_bs_eui = $1 WHERE last_attached_bs_eui = $2", newInt64, oldInt64)
	if err != nil {
		return nil, fmt.Errorf("update endpoints: %w", err)
	}

	// 4. Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// 5. Return updated base station
	return r.GetByEUI(ctx, tenantID, newEui)
}

// GetDB returns the database connection for direct database operations
// This is used by the BSSCI server for session management operations
func (r *BaseStationRepository) GetDB() *sqlx.DB {
	return r.db
}

// ListWithStats retrieves base stations with message statistics
func (r *BaseStationRepository) ListWithStats(ctx context.Context, tenantID int64, limit, offset int) ([]*BaseStationWithStats, int64, error) {
	// Query base stations with message stats
	query := `
		SELECT
			b.id,
			b.bs_eui,
			b.name,
			b.description,
			b.is_online,
			b.last_seen_at,
			b.created_at,
			b.updated_at,
			COALESCE(m.message_count, 0) as message_count,
			COALESCE(m.unique_endpoints, 0) as unique_endpoints,
			COALESCE(m.avg_rssi, 0) as avg_rssi,
			COALESCE(m.avg_snr, 0) as avg_snr
		FROM basestations b
		LEFT JOIN (
			SELECT
				bs_eui,
				COUNT(*) as message_count,
				COUNT(DISTINCT ep_eui) as unique_endpoints,
				AVG(rssi) as avg_rssi,
				AVG(snr) as avg_snr
			FROM messages
			WHERE tenant_id = $1
			GROUP BY bs_eui
		) m ON ('x' || encode(b.bs_eui, 'hex'))::bit(64)::bigint = m.bs_eui
		WHERE b.tenant_id = $1
		ORDER BY b.last_seen_at DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`

	var results []*BaseStationWithStats
	err := r.db.SelectContext(ctx, &results, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query base stations with stats: %w", err)
	}

	// Get total count
	var totalCount int64
	countQuery := "SELECT COUNT(*) FROM basestations WHERE tenant_id = $1"
	err = r.db.GetContext(ctx, &totalCount, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("count base stations: %w", err)
	}

	return results, totalCount, nil
}

// GetPropagationState retrieves propagation state for a base station
func (r *BaseStationRepository) GetPropagationState(ctx context.Context, baseStationID int64) (*models.BaseStationPropagationState, error) {
	query := `SELECT * FROM basestation_propagation_state WHERE base_station_id = $1`
	var state models.BaseStationPropagationState
	err := r.db.GetContext(ctx, &state, query, baseStationID)
	if err == sql.ErrNoRows {
		return nil, nil // Not an error, just no state yet
	}
	if err != nil {
		return nil, fmt.Errorf("get propagation state: %w", err)
	}
	return &state, nil
}

// UpsertPropagationState inserts or updates propagation state for a base station
func (r *BaseStationRepository) UpsertPropagationState(ctx context.Context, state *models.BaseStationPropagationState) error {
	query := `
		INSERT INTO basestation_propagation_state
			(base_station_id, last_full_sync_at, last_endpoint_cursor, status, retry_count, next_retry_at, last_error, tenant_progress, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (base_station_id) DO UPDATE SET
			last_full_sync_at = EXCLUDED.last_full_sync_at,
			last_endpoint_cursor = EXCLUDED.last_endpoint_cursor,
			status = EXCLUDED.status,
			retry_count = EXCLUDED.retry_count,
			next_retry_at = EXCLUDED.next_retry_at,
			last_error = EXCLUDED.last_error,
			tenant_progress = EXCLUDED.tenant_progress,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query,
		state.BaseStationID, state.LastFullSyncAt, state.LastEndpointCursor,
		state.Status, state.RetryCount, state.NextRetryAt, state.LastError, state.TenantProgress)
	if err != nil {
		return fmt.Errorf("upsert propagation state: %w", err)
	}
	return nil
}

// UpdatePropagationStatus updates only the status and error fields of propagation state
func (r *BaseStationRepository) UpdatePropagationStatus(ctx context.Context, baseStationID int64, status string, lastError *string) error {
	query := `UPDATE basestation_propagation_state SET status = $1, last_error = $2, updated_at = NOW() WHERE base_station_id = $3`
	_, err := r.db.ExecContext(ctx, query, status, lastError, baseStationID)
	if err != nil {
		return fmt.Errorf("update propagation status: %w", err)
	}
	return nil
}

// IncrementRetryCount increments the retry count and sets next retry time
func (r *BaseStationRepository) IncrementRetryCount(ctx context.Context, baseStationID int64, nextRetryAt time.Time) error {
	query := `UPDATE basestation_propagation_state SET retry_count = retry_count + 1, next_retry_at = $1, updated_at = NOW() WHERE base_station_id = $2`
	_, err := r.db.ExecContext(ctx, query, nextRetryAt, baseStationID)
	if err != nil {
		return fmt.Errorf("increment retry count: %w", err)
	}
	return nil
}

// BaseStationWithStats represents a base station with aggregated statistics
type BaseStationWithStats struct {
	ID              int64          `db:"id"`
	BsEui           []byte         `db:"bs_eui"`
	Name            string         `db:"name"`
	Description     sql.NullString `db:"description"`
	IsOnline        bool           `db:"is_online"`
	LastSeenAt      sql.NullTime   `db:"last_seen_at"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
	MessageCount    int            `db:"message_count"`
	UniqueEndpoints int            `db:"unique_endpoints"`
	AvgRSSI         float64        `db:"avg_rssi"`
	AvgSNR          float64        `db:"avg_snr"`
}
