package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
)

// columnLastError names the base station column holding the most recent
// connection error text.
const columnLastError = "last_error"

// BaseStationRepository implements the BaseStationRepository interface for PostgreSQL
type BaseStationRepository struct {
	log   logger.Logger
	clock clock.Clock
	db    *sqlx.DB
}

const updateFixedArgs = 2

// resourceBaseStation names this repository's resource in duplicate errors.
const resourceBaseStation = "base station"

// NewBaseStationRepository creates a new PostgreSQL Base Station repository
// updateFixedArgs counts the leading fixed parameters (tenant, id) of the
// dynamic update statement.
func NewBaseStationRepository(db *sqlx.DB, clk clock.Clock, log logger.Logger) *BaseStationRepository {
	return &BaseStationRepository{
		log: log, clock: clk, db: db}
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
		return fmt.Errorf("%s: %w", errWrapPrepareStatement, err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.log.Warn(logMsgCloseStmtBasestationCreate, logger.FieldError, err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, bs).Scan(&bs.ID, &bs.CreatedAt, &bs.UpdatedAt)
	if err != nil {
		return WrapDuplicateError(err, resourceBaseStation)
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
			return nil, errTextBaseStationNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStation, err)
	}

	return &bs, nil
}

// baseStationColumns are the columns a whole base station row is read with.
const baseStationColumns = `
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
			created_at, updated_at`

const (
	sqlGetBaseStationByEUI       = `SELECT` + baseStationColumns + ` FROM basestations WHERE tenant_id = $1 AND bs_eui = $2`
	sqlGetBaseStationByEUIGlobal = `SELECT` + baseStationColumns + ` FROM basestations WHERE bs_eui = $1`
	sqlDeleteBaseStationByEUI    = `DELETE FROM basestations WHERE tenant_id = $1 AND bs_eui = $2 RETURNING` + baseStationColumns
)

// GetByEUI retrieves a Base Station by EUI
func (r *BaseStationRepository) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	var bs models.BaseStation
	err := r.db.GetContext(ctx, &bs, sqlGetBaseStationByEUI, tenantID, eui)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", errWrapBaseStationNotFound, storage.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStation, err)
	}

	return &bs, nil
}

// GetByEUIGlobal retrieves a Base Station by EUI across all tenants.
// Used during BSSCI connect handshake when tenant is not yet resolved.
func (r *BaseStationRepository) GetByEUIGlobal(ctx context.Context, eui []byte) (*models.BaseStation, error) {
	var bs models.BaseStation
	err := r.db.GetContext(ctx, &bs, sqlGetBaseStationByEUIGlobal, eui)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errTextBaseStationNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStation, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapListAllLocations, err)
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
	args := make([]interface{}, 0, len(updates)+updateFixedArgs+1)
	args = append(args, tenantID, id)

	argIndex := len(args) + 1
	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
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
		return fmt.Errorf("%s: %w", errWrapUpdateBaseStation, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rows == 0 {
		return errTextBaseStationNotFound
	}

	return nil
}

// List retrieves Base Stations based on filter criteria
func (r *BaseStationRepository) List(ctx context.Context, filter *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	// Build WHERE clauses
	whereClauses := []string{"tenant_id = $1"}
	args := []interface{}{filter.TenantID}
	argCount := len(args) + 1

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
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountBaseStations, err)
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
		return nil, 0, fmt.Errorf("%s: %w", errWrapListBaseStations, err)
	}

	return baseStations, total, nil
}

// UpdateConnectionStatus updates the connection status of a Base Station
func (r *BaseStationRepository) UpdateConnectionStatus(ctx context.Context, tenantID, id int64, isOnline bool, lastError *string) error {
	updates := map[string]interface{}{
		"is_online":    isOnline,
		"last_seen_at": r.clock.Now(),
	}

	if isOnline {
		updates[columnLastError] = nil
		updates["retry_count"] = 0
		updates["next_retry_at"] = nil
	} else if lastError != nil {
		updates[columnLastError] = *lastError
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
			if backoffSeconds > maxConnectionRetryBackoffSeconds {
				backoffSeconds = maxConnectionRetryBackoffSeconds
			}
			updates["next_retry_at"] = r.clock.Now().Add(time.Duration(backoffSeconds) * time.Second)
		}
	}

	// Update connection status JSON
	statusJSON, err := json.Marshal(map[string]interface{}{
		"is_online":  isOnline,
		"last_seen":  r.clock.Now(),
		"last_error": lastError,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarshalConnectionStatus, err)
	}
	updates["connection_status"] = statusJSON

	return r.Update(ctx, tenantID, id, updates)
}

// GetStatistics retrieves statistics for Base Stations
func (r *BaseStationRepository) GetStatistics(ctx context.Context, tenantID int64) (*models.BaseStationStatistics, error) {
	var stats models.BaseStationStatistics

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
		return nil, fmt.Errorf("%s: %w", errWrapGetStatistics, err)
	}

	return &stats, nil
}

// UpdateEUI updates the Base Station EUI with transactional cascade to all dependent tables
func (r *BaseStationRepository) UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (_ *models.BaseStation, err error) {
	// Validate EUI lengths
	if len(oldEui) != 8 || len(newEui) != 8 {
		return nil, errTextInvalidEUILengthExpected8Bytes
	}

	// Start transaction; rolling back after a successful commit is a no-op.
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapBeginTransaction, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	// 1. Validate global uniqueness: reject if new_eui already exists (any tenant)
	var existsCount int
	err = tx.GetContext(ctx, &existsCount, "SELECT COUNT(*) FROM basestations WHERE bs_eui = $1", newEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCheckEUIUniqueness, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStation, err)
	}

	// 3. Update all tables with bs_eui reference (transactional cascade)

	// BYTEA columns: basestations.bs_eui (primary)
	// Reset online status and clear session - EUI change invalidates any active BSSCI session
	_, err = tx.ExecContext(ctx, "UPDATE basestations SET bs_eui = $1, updated_at = NOW(), is_online = false, session_uuid = NULL WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateBasestations, err)
	}

	// BYTEA columns: downlink_queue.bs_eui, downlink_queue.tx_bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE downlink_queue SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDownlinkQueueBsEui, err)
	}
	_, err = tx.ExecContext(ctx, "UPDATE downlink_queue SET tx_bs_eui = $1 WHERE tx_bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDownlinkQueueTxBsEui, err)
	}

	// BYTEA columns: dl_rx_status.bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDlRxStatus, err)
	}

	// BYTEA columns: dl_rx_status_queries.bs_eui_actual
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status_queries SET bs_eui_actual = $1 WHERE bs_eui_actual = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDlRxStatusQueries, err)
	}

	// BYTEA columns: roaming_events.from_bs_eui, roaming_events.to_bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE roaming_events SET from_bs_eui = $1 WHERE from_bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateRoamingEventsFromBsEui, err)
	}
	_, err = tx.ExecContext(ctx, "UPDATE roaming_events SET to_bs_eui = $1 WHERE to_bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateRoamingEventsBsEui, err)
	}

	// BYTEA columns: mioty_basestation_status.basestation_eui
	_, err = tx.ExecContext(ctx, "UPDATE mioty_basestation_status SET basestation_eui = $1 WHERE basestation_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMiotyBasestationStatus, err)
	}

	// BYTEA columns: messages.bs_eui (8-byte big-endian per migration 000135)
	_, err = tx.ExecContext(ctx, "UPDATE messages SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMessages, err)
	}

	// BYTEA columns: messages_archive.bs_eui (rebuilt LIKE messages by migration 000139)
	_, err = tx.ExecContext(ctx, "UPDATE messages_archive SET bs_eui = $1 WHERE bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMessagesArchive, err)
	}

	// Preserved legacy archive (pre-000139) participates in identity
	// maintenance so its rows never carry a stale EUI
	if err := updateLegacyArchiveEUI(ctx, tx, legacyArchiveBsEUI, newEui, oldEui); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMessagesArchivePre000139, err)
	}

	// BYTEA columns: endpoints.last_attached_bs_eui
	_, err = tx.ExecContext(ctx, "UPDATE endpoints SET last_attached_bs_eui = $1 WHERE last_attached_bs_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateEndpoints, err)
	}

	// 4. Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCommitTransaction, err)
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
		) m ON b.bs_eui = m.bs_eui
		WHERE b.tenant_id = $1
		ORDER BY b.last_seen_at DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`

	var results []*BaseStationWithStats
	err := r.db.SelectContext(ctx, &results, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapQueryBaseStationsWithStats, err)
	}

	// Get total count
	var totalCount int64
	countQuery := "SELECT COUNT(*) FROM basestations WHERE tenant_id = $1"
	err = r.db.GetContext(ctx, &totalCount, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountBaseStations, err)
	}

	return results, totalCount, nil
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

// UpdateProfile persists the operator-editable base station fields: name,
// description, location and tags.
func (r *BaseStationRepository) UpdateProfile(ctx context.Context, bs *models.BaseStation) error {
	updates := map[string]interface{}{
		"name":                bs.Name,
		"description":         bs.Description,
		"latitude":            bs.Latitude,
		"longitude":           bs.Longitude,
		"altitude":            bs.Altitude,
		"location_source":     bs.LocationSource,
		"location_updated_at": bs.LocationUpdatedAt,
		"tags":                bs.Tags,
	}
	if err := r.Update(ctx, bs.TenantID, bs.ID, updates); err != nil {
		return fmt.Errorf("%s: %w", errWrapOpUpdateBaseStation, err)
	}
	return nil
}

// DeleteByEUI removes a tenant's base station addressed by EUI in one
// statement and returns the row it removed.
func (r *BaseStationRepository) DeleteByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	var bs models.BaseStation
	err := r.db.GetContext(ctx, &bs, sqlDeleteBaseStationByEUI, tenantID, eui)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%s: %w", errWrapBaseStationNotFound, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapOpDeleteBaseStation, err)
	}
	return &bs, nil
}
