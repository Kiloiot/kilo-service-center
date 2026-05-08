// Package postgres provides PostgreSQL database implementation for KiloCenter
package postgres

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/crypto"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
)

// DB represents a PostgreSQL database connection
type DB struct {
	conn    *sql.DB
	sqlxDB  *sqlx.DB
	manager *ConnectionManager
	config  config.StorageConfig
	log     logger.Logger

	// Repository instances
	baseStationRepo          interfaces.BaseStationRepository
	basestationSessionRepo   interfaces.BaseStationSessionRepository
	basestationReceptionRepo interfaces.BaseStationReceptionRepository
	endPointRepo             interfaces.EndpointRepository
	endpointSessionRepo      interfaces.EndPointSessionRepository
	endpointKeyRepo          interfaces.EndPointKeyRepository
	downlinkQueueRepo        interfaces.DownlinkQueueRepository
	roamingRepo              interfaces.RoamingAgreementRepository
	organizationRepo         interfaces.OrganizationRepository
	systemEventsStore        *SystemEventStore
	messageRepo              *MessageRepository
	basestationStatusRepo    *BaseStationStatusRepository
	dlRxStatusRepo           *DLRXStatusRepository
	pendingOpRepo            interfaces.PendingOperationRepository
	scaciSessionRepo         *SCACISessionRepository
	scaciOperationRepo       *SCACIOperationRepository
	queueReader              *DownlinkQueueReader
	roamingImplRepo          *RoamingRepository // Production roaming methods
	userRepo                 interfaces.UserRepository
	apiKeyRepo               interfaces.APIKeyRepository
	integrationRepo          interfaces.IntegrationRepository
	operationStatusRepo      interfaces.OperationStatusRepository

	// Blueprint device catalog repositories (Migration 000102-000104)
	manufacturerRepo interfaces.ManufacturerRepository
	deviceModelRepo  interfaces.DeviceModelRepository
	blueprintRepo    interfaces.BlueprintRepository

	// Optional crypto for session key migration (set via SetKeyEncryptor)
	keyEncryptor *crypto.KeyEncryptor
}

// Ensure DB implements storage.Storage
var _ storage.Storage = (*DB)(nil)

// New creates a new PostgreSQL database connection with retry logic
func New(cfg config.StorageConfig) (*DB, error) {
	log := logger.Get().WithField("component", "postgres")

	// Create connection configuration
	connConfig := &Config{
		Host:            cfg.Host,
		Port:            cfg.Port,
		Database:        cfg.Database,
		Username:        cfg.Username,
		Password:        cfg.Password,
		SSLMode:         cfg.SSLMode,
		MaxOpenConns:    cfg.MaxConnections,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
	}

	// Create connection manager
	manager := NewConnectionManager(connConfig, log)

	// Connect with retry logic
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := manager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to establish database connection: %w", err)
	}

	// Get the established connection
	conn := manager.GetDB()

	// Test the connection
	if err := conn.PingContext(ctx); err != nil {
		_ = manager.Close() // Best effort close
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create sqlx wrapper for the connection
	sqlxDB := sqlx.NewDb(conn, "postgres")

	log.Info("PostgreSQL connection established",
		"host", cfg.Host,
		"database", cfg.Database,
		"max_open_conns", cfg.MaxConnections,
		"max_idle_conns", cfg.MaxIdleConns)

	db := &DB{
		conn:    conn,
		sqlxDB:  sqlxDB,
		manager: manager,
		config:  cfg,
		log:     log,
	}

	// Initialize repositories
	db.messageRepo = NewMessageRepository(sqlxDB)
	db.dlRxStatusRepo = NewDLRXStatusRepository(sqlxDB, log)
	db.scaciSessionRepo = NewSCACISessionRepository(sqlxDB)
	db.scaciOperationRepo = NewSCACIOperationRepository(sqlxDB, log)
	db.queueReader = NewDownlinkQueueReader(sqlxDB)

	return db, nil
}

// Ping checks the database connection with context
func (db *DB) Ping(ctx context.Context) error {
	return db.manager.HealthCheck(ctx)
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.manager.Close()
}

// SetKeyEncryptor sets the optional key encryptor for session key lazy migration
// This should be called after DB creation if lazy migration is desired
func (db *DB) SetKeyEncryptor(ke *crypto.KeyEncryptor) {
	db.keyEncryptor = ke
}

// Query executes a query that returns rows with retry logic
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	retryErr := db.manager.ExecuteWithRetry(ctx, func() error {
		rows, err = db.conn.QueryContext(ctx, query, args...)
		return err
	})

	if retryErr != nil {
		return nil, retryErr
	}
	return rows, nil
}

// QueryRow executes a query that returns at most one row
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning any rows with retry logic
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	retryErr := db.manager.ExecuteWithRetry(ctx, func() error {
		result, err = db.conn.ExecContext(ctx, query, args...)
		return err
	})

	if retryErr != nil {
		return nil, retryErr
	}
	return result, nil
}

// GetConnectionStats returns database connection pool statistics
func (db *DB) GetConnectionStats() sql.DBStats {
	return db.manager.Stats()
}

// WaitForConnection waits for the database to become available
func (db *DB) WaitForConnection(ctx context.Context, timeout time.Duration) error {
	return db.manager.WaitForConnection(ctx, timeout)
}

// GetDB returns the underlying database connection
// This should be used sparingly and only for operations that need direct access
func (db *DB) GetDB() *sql.DB {
	return db.conn
}

// GetSqlxDB returns the sqlx database connection for adapters
func (db *DB) GetSqlxDB() *sqlx.DB {
	return db.sqlxDB
}

// EndPoint operations

// CreateEndPoint creates a new end point
func (db *DB) CreateEndPoint(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	repo := NewEndPointRepository(db.sqlxDB)
	if err := repo.Create(ctx, endpoint); err != nil {
		return nil, fmt.Errorf("CreateEndPoint: %w", err)
	}
	return endpoint, nil
}

// GetEndPoint retrieves an end point by EUI
func (db *DB) GetEndPoint(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
	repo := NewEndPointRepository(db.sqlxDB)
	return repo.GetByEUI(ctx, tenantID, eui)
}

// UpdateEndPoint updates an end point
func (db *DB) UpdateEndPoint(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	repo := NewEndPointRepository(db.sqlxDB)
	if err := repo.Update(ctx, endpoint); err != nil {
		return nil, fmt.Errorf("UpdateEndPoint: %w", err)
	}
	return endpoint, nil
}

// UpdateEndPointWithEUI atomically cascades an EUI change and updates all fields
func (db *DB) UpdateEndPointWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error) {
	repo := NewEndPointRepository(db.sqlxDB)
	return repo.UpdateWithEUI(ctx, tenantID, oldEui, endpoint)
}

// CheckEndPointEUIUnique verifies no endpoint with the given EUI exists globally
func (db *DB) CheckEndPointEUIUnique(ctx context.Context, eui []byte) error {
	repo := NewEndPointRepository(db.sqlxDB)
	return repo.CheckEUIUnique(ctx, eui)
}

// DeleteEndPoint deletes an end point
func (db *DB) DeleteEndPoint(ctx context.Context, eui []byte, tenantID int64) error {
	repo := NewEndPointRepository(db.sqlxDB)
	return repo.DeleteByTenant(ctx, tenantID, eui)
}

// ListEndPoints lists end points for a tenant
func (db *DB) ListEndPoints(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) {
	repo := NewEndPointRepository(db.sqlxDB)
	return repo.ListByTenantPaginated(ctx, tenantID, limit, offset)
}

// BaseStation operations

// CreateBaseStation creates a new base station
func (db *DB) CreateBaseStation(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	repo := NewBaseStationRepository(db.sqlxDB)
	if err := repo.Create(ctx, bs); err != nil {
		return nil, fmt.Errorf("CreateBaseStation: %w", err)
	}
	return bs, nil
}

// GetBaseStation retrieves a base station by EUI
func (db *DB) GetBaseStation(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	repo := NewBaseStationRepository(db.sqlxDB)
	return repo.GetByEUI(ctx, tenantID, eui)
}

// UpdateBaseStation updates a base station
func (db *DB) UpdateBaseStation(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	// Build updates map from the model
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

	repo := NewBaseStationRepository(db.sqlxDB)
	if err := repo.Update(ctx, bs.TenantID, bs.ID, updates); err != nil {
		return nil, fmt.Errorf("UpdateBaseStation: %w", err)
	}
	return bs, nil
}

// UpdateBaseStationEUI updates the EUI of a base station with cascade to all dependent tables.
// This is a transactional operation that updates the EUI across all tables that reference it.
func (db *DB) UpdateBaseStationEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error) {
	repo := NewBaseStationRepository(db.sqlxDB)
	return repo.UpdateEUI(ctx, tenantID, oldEui, newEui)
}

// DeleteBaseStation deletes a base station by EUI
func (db *DB) DeleteBaseStation(ctx context.Context, eui []byte, tenantID int64) error {
	// First get the base station to get its ID
	repo := NewBaseStationRepository(db.sqlxDB)
	bs, err := repo.GetByEUI(ctx, tenantID, eui)
	if err != nil {
		return fmt.Errorf("DeleteBaseStation: %w", err)
	}
	return repo.Delete(ctx, tenantID, bs.ID)
}

// ListBaseStations lists base stations for a tenant
func (db *DB) ListBaseStations(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error) {
	filter := &models.BaseStationFilter{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	}
	repo := NewBaseStationRepository(db.sqlxDB)
	stations, _, err := repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("ListBaseStations: %w", err)
	}
	return stations, nil
}

// ListAllBaseStationLocations returns all base stations with coordinates across all tenants.
func (db *DB) ListAllBaseStationLocations(ctx context.Context) ([]*models.BaseStation, error) {
	repo := NewBaseStationRepository(db.sqlxDB)
	stations, err := repo.ListAllLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllBaseStationLocations: %w", err)
	}
	return stations, nil
}

// Message operations are now handled by the MessageRepository
// Use CreateULDataMessage, GetULDataMessage, and ListULDataMessages methods

// Downlink operations

// EnqueueDownlink adds a downlink message to the queue
func (db *DB) EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	// Convert EUI string to bytes for database storage
	epEuiBytes, err := hex.DecodeString(downlink.EPEUI)
	if err != nil || len(epEuiBytes) != 8 {
		return nil, fmt.Errorf("invalid EP EUI: %s", downlink.EPEUI)
	}

	// Convert TenantID string to int64
	tenantID, err := strconv.ParseInt(downlink.TenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %s", downlink.TenantID)
	}

	query := `
		INSERT INTO downlink_queue (
			ep_eui, tenant_id, organization_id, payload,
			priority, status, attempts, max_attempts,
			que_id, cnt_depend, packet_cnt, format,
			response_exp, response_prio, dl_wind_req, exp_only,
			dl_rx_stat_qry, user_data, created_at, earliest_at, latest_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, NOW(), NOW(), NOW() + INTERVAL '24 hours'
		) RETURNING id, created_at, updated_at`

	var id int64
	var createdAt time.Time
	var updatedAt time.Time

	// Convert packet counter array to PostgreSQL array
	var packetCntArray interface{}
	if len(downlink.PacketCntArray) > 0 {
		packetCntArray = pq.Array(downlink.PacketCntArray)
	} else {
		packetCntArray = nil
	}

	// Set defaults for optional MIOTY fields
	format := 0
	if downlink.Format != 0 {
		format = int(downlink.Format)
	}

	responseExp := downlink.ResponseExp
	responsePrio := downlink.ResponsePrio
	dlWindReq := downlink.DlWindReq

	expOnly := downlink.ExpOnly
	dlRxStatQry := downlink.DlRxStatQry

	// Set default max attempts if not specified
	maxAttempts := downlink.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}

	// Marshal UserData through canonical DLDataQueue struct for counter-dependent messages
	var userDataJSON interface{}
	if downlink.CntDepend && len(downlink.UserData) > 0 {
		dlQueue := &mioty.DLDataQueue{
			UserData: downlink.UserData,
		}
		userData, err := json.Marshal(dlQueue)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user data: %w", err)
		}
		userDataJSON = userData
	} else {
		userDataJSON = nil
	}

	err = db.sqlxDB.QueryRowContext(
		ctx,
		query,
		epEuiBytes,
		tenantID,
		downlink.OrganizationID,
		downlink.Payload,
		downlink.Priority,
		downlink.Status,
		downlink.Attempts,
		maxAttempts,
		downlink.QueID,
		downlink.CntDepend,
		packetCntArray,
		format,
		responseExp,
		responsePrio,
		dlWindReq,
		expOnly,
		dlRxStatQry,
		userDataJSON,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to enqueue downlink: %w", err)
	}

	// Update the downlink object with database-generated values
	downlink.ID = id
	downlink.CreatedAt = createdAt
	downlink.UpdatedAt = createdAt

	return downlink, nil
}

// GetDownlinkQueue retrieves pending downlink messages for a device
// Delegates to DownlinkQueueReader for DRY implementation
func (db *DB) GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error) {
	// Convert tenant ID
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %s", tenantID)
	}

	// Prepare optional endpoint filter
	var epFilter *[8]byte
	if deviceEUI != "" {
		epEuiBytes, err := hex.DecodeString(deviceEUI)
		if err != nil || len(epEuiBytes) != 8 {
			return nil, fmt.Errorf("invalid EP EUI: %s", deviceEUI)
		}
		var eui [8]byte
		copy(eui[:], epEuiBytes)
		epFilter = &eui
	}

	// Delegate to queue reader with limit=1000, offset=0
	return db.queueReader.ListTenantQueue(ctx, tid, epFilter, 1000, 0)
}

// UpdateDownlinkStatus updates the status of a downlink message.
// orgID filters by organization; nil means no org filter (backward compatible).
func (db *DB) UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error {
	// Parse the ID based on whether it's a queue ID or database ID
	var query string
	var result sql.Result
	var err error

	// Build organization filter clause for backward compatibility
	// nil orgID = no filter (existing callers), non-nil = filter on org (new SCACI path)
	orgFilter := ""
	var orgFilterArg interface{}
	if orgID != nil {
		orgFilter = " AND (organization_id = $3 OR organization_id IS NULL)"
		orgFilterArg = *orgID
	}

	// Check if this is a queue ID (longer) or database ID
	if queId, parseErr := strconv.ParseInt(id, 10, 64); parseErr == nil && queId > 1000000 {
		// This looks like a MIOTY queue ID
		query = fmt.Sprintf(`
			UPDATE downlink_queue
			SET status = $1,
			    updated_at = NOW(),
			    attempts = attempts + 1
			WHERE que_id = $2%s
			RETURNING id`, orgFilter)

		if orgID != nil {
			result, err = db.sqlxDB.ExecContext(ctx, query, status, queId, orgFilterArg)
		} else {
			result, err = db.sqlxDB.ExecContext(ctx, query, status, queId)
		}
	} else {
		// This is a database ID
		dbID, parseErr := strconv.ParseInt(id, 10, 64)
		if parseErr != nil {
			return fmt.Errorf("invalid downlink ID: %s", id)
		}

		query = fmt.Sprintf(`
			UPDATE downlink_queue
			SET status = $1,
			    updated_at = NOW(),
			    attempts = attempts + 1
			WHERE id = $2%s`, orgFilter)

		if orgID != nil {
			result, err = db.sqlxDB.ExecContext(ctx, query, status, dbID, orgFilterArg)
		} else {
			result, err = db.sqlxDB.ExecContext(ctx, query, status, dbID)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to update downlink status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("downlink message not found: %s", id)
	}

	return nil
}

// UpdateDownlinkResult updates the transmission result of a downlink message.
// Includes tenant and endpoint validation to prevent cross-tenant updates (BSSCI §3.14).
// orgID filters by organization; nil means no org filter (backward compatible).
func (db *DB) UpdateDownlinkResult(ctx context.Context, queId int64, result string, txTime *int64, packetCnt *uint32, bsEUI []byte, epEUI []byte, tenantID string, orgID *uuid.UUID) error {
	// Convert tenantID string to int64
	tenantIDInt, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	// Convert packet counter to int64 for database storage
	var transmissionPacketCnt *int64
	if packetCnt != nil {
		val := int64(*packetCnt)
		transmissionPacketCnt = &val
	}

	// Build organization filter clause for backward compatibility
	// nil orgID = no filter (existing callers), non-nil = filter on org (new SCACI path)
	orgFilter := ""
	args := []interface{}{result, txTime, bsEUI, transmissionPacketCnt, queId, epEUI, tenantIDInt}
	if orgID != nil {
		orgFilter = " AND (organization_id = $8 OR organization_id IS NULL)"
		args = append(args, *orgID)
	}

	// Update both legacy and transmission columns for backward compatibility
	// Will consolidate in future migration
	// CRITICAL: Include ep_eui and tenant_id in WHERE clause to prevent cross-tenant/endpoint updates
	query := fmt.Sprintf(`
		UPDATE downlink_queue
		SET result = $1,
		    tx_time = $2,
		    bs_eui = $3,
		    transmission_result = $1,
		    transmission_time = $2,
		    transmission_packet_cnt = $4,
		    status = CASE
		        WHEN $1 = 'sent' THEN 'transmitted'
		        WHEN $1 = 'expired' THEN 'expired'
		        WHEN $1 = 'invalid' THEN 'failed'
		        WHEN $1 = 'revoked' THEN 'revoked'
		        ELSE status
		    END,
		    transmitted_at = CASE
		        WHEN $1 = 'sent' THEN NOW()
		        ELSE transmitted_at
		    END,
		    updated_at = NOW()
		WHERE que_id = $5
		  AND ep_eui = $6
		  AND tenant_id = $7%s`, orgFilter)

	res, err := db.sqlxDB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update downlink result: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		// No rows updated means either:
		// 1. Queue ID doesn't exist
		// 2. Endpoint EUI doesn't match
		// 3. Tenant ID doesn't match (cross-tenant attempt)
		// 4. Organization ID doesn't match (if provided)
		// For security, we don't distinguish between these cases
		return storage.ErrDownlinkNotFound
	}

	return nil
}

// UpdateDownlinkBaseStation updates the base station ownership of a queued downlink
func (db *DB) UpdateDownlinkBaseStation(ctx context.Context, queId uint64, tenantID string, bsEUI uint64) error {
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %s", tenantID)
	}

	var euiBytes []byte
	if bsEUI != 0 {
		euiBytes = make([]byte, 8)
		binary.BigEndian.PutUint64(euiBytes, bsEUI)
	}
	// euiBytes is nil when bsEUI == 0, which sets bs_eui to NULL

	query := `
		UPDATE downlink_queue
		   SET bs_eui = $1,
		       updated_at = NOW()
		 WHERE que_id = $2
		   AND tenant_id = $3`
	res, err := db.sqlxDB.ExecContext(ctx, query, euiBytes, queId, tid)
	if err != nil {
		return fmt.Errorf("failed to set downlink owner: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		// No rows updated means either:
		// 1. Queue ID doesn't exist
		// 2. Tenant ID doesn't match
		// For security, we don't distinguish between these cases
		return storage.ErrDownlinkNotFound
	}

	return nil
}

// GetDownlinkByQueueID retrieves a downlink message by its queue ID
func (db *DB) GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error) {
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %s", tenantID)
	}

	var msg storage.DownlinkMessage
	var epEuiBytes []byte
	var bsEuiBytes []byte
	var msgTenantID int64
	var orgID *uuid.UUID

	query := `
		SELECT id, ep_eui, bs_eui, que_id, tenant_id, organization_id, status, dl_rx_stat_qry, created_at
		FROM downlink_queue
		WHERE que_id = $1 AND tenant_id = $2`

	err = db.sqlxDB.QueryRowContext(ctx, query, queId, tid).Scan(
		&msg.ID, &epEuiBytes, &bsEuiBytes, &msg.QueID, &msgTenantID, &orgID, &msg.Status, &msg.DlRxStatQry, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	msg.EPEUI = hex.EncodeToString(epEuiBytes)
	msg.TenantID = strconv.FormatInt(msgTenantID, 10)
	msg.OrganizationID = orgID

	if len(bsEuiBytes) == 8 {
		msg.BsEui = binary.BigEndian.Uint64(bsEuiBytes)
	}
	return &msg, nil
}

// GetDownlinkByPacketCnt retrieves a downlink message by endpoint EUI and packet counter
// Implements SCACI §3.11 DL Data Revoke specification requirement
// Supports both counter-dependent (array match) and counter-independent (null/empty array) downlinks
func (db *DB) GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*storage.DownlinkMessage, error) {
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %s", tenantID)
	}

	// Decode endpoint EUI from hex string
	epEuiBytes, err := hex.DecodeString(epEui)
	if err != nil || len(epEuiBytes) != 8 {
		return nil, fmt.Errorf("invalid endpoint EUI: %s", epEui)
	}

	var msg storage.DownlinkMessage
	var epEuiBytesResult []byte
	var bsEuiBytes []byte
	var msgTenantID int64
	var packetCntArray pq.Int64Array

	// Query handles both counter-dependent and counter-independent downlinks
	// Counter-dependent: packetCnt must be in the array
	// Counter-independent: packet_cnt array is NULL or empty
	query := `
		SELECT id, ep_eui, bs_eui, que_id, tenant_id, status, cnt_depend, packet_cnt, created_at
		FROM downlink_queue
		WHERE ep_eui = $1
		  AND tenant_id = $2
		  AND status NOT IN ('transmitted', 'confirmed', 'expired', 'failed', 'revoked')
		  AND (
			  (cnt_depend = true AND $3 = ANY(packet_cnt))
			  OR
			  (cnt_depend = false AND (packet_cnt IS NULL OR cardinality(packet_cnt) = 0))
		  )
		ORDER BY created_at DESC
		LIMIT 1`

	err = db.sqlxDB.QueryRowContext(ctx, query, epEuiBytes, tid, int64(packetCnt)).Scan(
		&msg.ID, &epEuiBytesResult, &bsEuiBytes, &msg.QueID, &msgTenantID,
		&msg.Status, &msg.CntDepend, &packetCntArray, &msg.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query downlink by packet counter: %w", err)
	}

	msg.EPEUI = hex.EncodeToString(epEuiBytesResult)
	msg.TenantID = strconv.FormatInt(msgTenantID, 10)

	if len(bsEuiBytes) == 8 {
		msg.BsEui = binary.BigEndian.Uint64(bsEuiBytes)
	}

	// Convert packet counter array
	if len(packetCntArray) > 0 {
		msg.PacketCntArray = []int64(packetCntArray)
	}

	return &msg, nil
}

// RevokeDownlink marks a downlink message as revoked
func (db *DB) RevokeDownlink(ctx context.Context, queId int64, tenantID string) error {
	// Convert tenant ID
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %s", tenantID)
	}

	// Use constants instead of SQL literals (BSSCI §5.13)
	query := `
		UPDATE downlink_queue
		SET status = $3,
		    result = $4,
		    updated_at = NOW()
		WHERE que_id = $1 AND tenant_id = $2
		AND status IN ($5, $6, $7)`

	result, err := db.sqlxDB.ExecContext(ctx, query, queId, tid,
		bssci.DLQueueStatusRevoked,
		bssci.DLDataResultRevoked,
		bssci.DLQueueStatusPending,
		bssci.DLQueueStatusScheduled,
		bssci.DLQueueStatusQueued)
	if err != nil {
		return fmt.Errorf("failed to revoke downlink: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("downlink message not found or already transmitted: queue_id=%d", queId)
	}

	return nil
}

// GetDownlinkResults retrieves downlink results with pagination and filtering.
// Implements enhanced pagination with accurate COUNT queries and SQL-based time filtering.
// Returns messages, total count, and error for proper pagination support.
// orgID filters by organization; nil means no org filter (backward compatible).
func (db *DB) GetDownlinkResults(ctx context.Context, deviceEUI string, tenantID string, orgID *uuid.UUID, statusFilter string, timeFrom, timeTo *time.Time, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	// Convert tenant ID
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid tenant ID: %s", tenantID)
	}

	var query, countQuery string
	var rows *sql.Rows
	var totalCount int
	argIndex := 3 // Start after epEui and tid

	// Build status condition based on filter
	statusCondition := fmt.Sprintf("status IN ('%s', '%s', '%s')",
		bssci.DLQueueStatusTransmitted, bssci.DLQueueStatusExpired, bssci.DLQueueStatusFailed)
	if statusFilter != "" {
		// Validate status filter against canonical constants
		switch statusFilter {
		case bssci.DLQueueStatusTransmitted, bssci.DLQueueStatusExpired, bssci.DLQueueStatusFailed:
			statusCondition = fmt.Sprintf("status = '%s'", statusFilter)
		default:
			return nil, 0, fmt.Errorf("invalid status filter: %s", statusFilter)
		}
	}

	// Build time filter conditions
	timeConditions := ""
	queryArgs := []interface{}{}

	// Default pagination limits
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	if deviceEUI != "" {
		// Convert EUI string to bytes
		epEuiBytes, parseErr := hex.DecodeString(deviceEUI)
		if parseErr != nil || len(epEuiBytes) != 8 {
			return nil, 0, fmt.Errorf("invalid EP EUI: %s", deviceEUI)
		}

		// Build base args
		queryArgs = append(queryArgs, epEuiBytes, tid)

		// Add organization filter if provided
		if orgID != nil {
			timeConditions += fmt.Sprintf(" AND (organization_id = $%d OR organization_id IS NULL)", argIndex)
			queryArgs = append(queryArgs, *orgID)
			argIndex++
		}

		// Add time filters if provided
		if timeFrom != nil {
			timeConditions += fmt.Sprintf(" AND transmitted_at >= $%d", argIndex)
			queryArgs = append(queryArgs, *timeFrom)
			argIndex++
		}
		if timeTo != nil {
			timeConditions += fmt.Sprintf(" AND transmitted_at <= $%d", argIndex)
			queryArgs = append(queryArgs, *timeTo)
			argIndex++
		}

		// First, get the total count for pagination
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*)
			FROM downlink_queue
			WHERE ep_eui = $1 AND tenant_id = $2
			AND %s%s`, statusCondition, timeConditions)

		err = db.sqlxDB.QueryRowContext(ctx, countQuery, queryArgs...).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count downlink results: %w", err)
		}

		// Query for specific device with result status (includes transmission_* columns)
		query = fmt.Sprintf(`
			SELECT
				id, ep_eui, tenant_id, organization_id, payload,
				priority, status, attempts, max_attempts,
				que_id, cnt_depend, packet_cnt, format,
				response_exp, response_prio, dl_wind_req, exp_only, dl_rx_stat_qry,
				COALESCE(transmission_result, result) as result,
				COALESCE(transmission_time, tx_time) as tx_time,
				transmission_packet_cnt,
				bs_eui,
				created_at, transmitted_at, acknowledged_at, user_data
			FROM downlink_queue
			WHERE ep_eui = $1 AND tenant_id = $2
			AND %s%s
			ORDER BY transmitted_at DESC NULLS LAST, created_at DESC
			LIMIT $%d OFFSET $%d`, statusCondition, timeConditions, argIndex, argIndex+1)

		queryArgs = append(queryArgs, limit, offset)
		rows, err = db.sqlxDB.QueryContext(ctx, query, queryArgs...)
	} else {
		// Build base args for all devices
		queryArgs = append(queryArgs, tid)
		argIndex = 2 // Start after tid

		// Add organization filter if provided
		if orgID != nil {
			timeConditions += fmt.Sprintf(" AND (organization_id = $%d OR organization_id IS NULL)", argIndex)
			queryArgs = append(queryArgs, *orgID)
			argIndex++
		}

		// Add time filters if provided
		if timeFrom != nil {
			timeConditions += fmt.Sprintf(" AND transmitted_at >= $%d", argIndex)
			queryArgs = append(queryArgs, *timeFrom)
			argIndex++
		}
		if timeTo != nil {
			timeConditions += fmt.Sprintf(" AND transmitted_at <= $%d", argIndex)
			queryArgs = append(queryArgs, *timeTo)
			argIndex++
		}

		// First, get the total count for pagination
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*)
			FROM downlink_queue
			WHERE tenant_id = $1
			AND %s%s`, statusCondition, timeConditions)

		err = db.sqlxDB.QueryRowContext(ctx, countQuery, queryArgs...).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count downlink results: %w", err)
		}

		// Query for all devices in tenant with result status (includes transmission_* columns)
		query = fmt.Sprintf(`
			SELECT
				id, ep_eui, tenant_id, organization_id, payload,
				priority, status, attempts, max_attempts,
				que_id, cnt_depend, packet_cnt, format,
				response_exp, response_prio, dl_wind_req, exp_only, dl_rx_stat_qry,
				COALESCE(transmission_result, result) as result,
				COALESCE(transmission_time, tx_time) as tx_time,
				transmission_packet_cnt,
				bs_eui,
				created_at, transmitted_at, acknowledged_at, user_data
			FROM downlink_queue
			WHERE tenant_id = $1
			AND %s%s
			ORDER BY transmitted_at DESC NULLS LAST, created_at DESC
			LIMIT $%d OFFSET $%d`, statusCondition, timeConditions, argIndex, argIndex+1)

		queryArgs = append(queryArgs, limit, offset)
		rows, err = db.sqlxDB.QueryContext(ctx, query, queryArgs...)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to query downlink results: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			db.log.Warn("rows close failed", "error", err, "operation", "GetDownlinkResults")
		}
	}()

	var messages []*storage.DownlinkMessage

	for rows.Next() {
		var msg storage.DownlinkMessage
		var epEuiBytes []byte
		var bsEuiBytes []byte
		var packetCntArray pq.Int64Array
		var result sql.NullString
		var txTime sql.NullInt64
		var transmissionPacketCnt sql.NullInt64
		var transmittedAt, acknowledgedAt sql.NullTime
		var userDataJSON []byte
		var orgID *uuid.UUID

		err := rows.Scan(
			&msg.ID,
			&epEuiBytes,
			&tid,
			&orgID,
			&msg.Payload,
			&msg.Priority,
			&msg.Status,
			&msg.Attempts,
			&msg.MaxAttempts,
			&msg.QueID,
			&msg.CntDepend,
			&packetCntArray,
			&msg.Format,
			&msg.ResponseExp,
			&msg.ResponsePrio,
			&msg.DlWindReq,
			&msg.ExpOnly,
			&msg.DlRxStatQry,
			&result,
			&txTime,
			&transmissionPacketCnt,
			&bsEuiBytes,
			&msg.CreatedAt,
			&transmittedAt,
			&acknowledgedAt,
			&userDataJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan downlink result: %w", err)
		}

		// Convert bytes to hex string
		msg.EPEUI = hex.EncodeToString(epEuiBytes)
		msg.TenantID = strconv.FormatInt(tid, 10)
		msg.OrganizationID = orgID

		// Handle nullable fields
		if result.Valid {
			msg.Result = result.String
		}
		if txTime.Valid {
			msg.TxTime = txTime.Int64
		}
		if transmissionPacketCnt.Valid {
			msg.TransmissionPacketCnt = transmissionPacketCnt.Int64
		}
		if len(bsEuiBytes) > 0 {
			msg.TxBSEUI = hex.EncodeToString(bsEuiBytes)
		}

		// Convert packet counter arrays
		if packetCntArray != nil {
			msg.PacketCntArray = []int64(packetCntArray)
		}

		// Handle nullable timestamps
		if transmittedAt.Valid {
			msg.SentAt = &transmittedAt.Time
		}
		if acknowledgedAt.Valid {
			msg.ScheduledAt = &acknowledgedAt.Time
		}

		// Unmarshal user_data through canonical DLDataQueue struct
		if len(userDataJSON) > 0 {
			var dlQueue mioty.DLDataQueue
			if err := json.Unmarshal(userDataJSON, &dlQueue); err == nil {
				msg.UserData = dlQueue.UserData
				// Populate single Payload field for backward compatibility
				if len(msg.UserData) > 0 && msg.Payload == nil {
					msg.Payload = msg.UserData[0]
				}
			}
		}

		messages = append(messages, &msg)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating downlink results: %w", err)
	}

	return messages, totalCount, nil
}

// Repository access methods

// BaseStations returns the BaseStation repository
func (db *DB) BaseStations() interfaces.BaseStationRepository {
	if db.baseStationRepo == nil {
		db.baseStationRepo = NewBaseStationRepository(db.sqlxDB)
	}
	return db.baseStationRepo
}

// EndPoints returns the EndPoint repository
func (db *DB) EndPoints() interfaces.EndpointRepository {
	if db.endPointRepo == nil {
		db.endPointRepo = NewEndPointRepository(db.sqlxDB)
	}
	return db.endPointRepo
}

// Organizations returns the Organization repository for Kilo Cloud multi-tenant integration
func (db *DB) Organizations() interfaces.OrganizationRepository {
	if db.organizationRepo == nil {
		db.organizationRepo = NewOrganizationRepository(db.sqlxDB)
	}
	return db.organizationRepo
}

// Users returns the User repository for authentication management
func (db *DB) Users() interfaces.UserRepository {
	if db.userRepo == nil {
		db.userRepo = NewUserRepository(db.sqlxDB)
	}
	return db.userRepo
}

// APIKeys returns the API Key repository for programmatic access management
func (db *DB) APIKeys() interfaces.APIKeyRepository {
	if db.apiKeyRepo == nil {
		db.apiKeyRepo = NewAPIKeyRepository(db.sqlxDB)
	}
	return db.apiKeyRepo
}

// Integrations returns the Integration repository for event sink management
func (db *DB) Integrations() interfaces.IntegrationRepository {
	if db.integrationRepo == nil {
		db.integrationRepo = NewIntegrationRepository(db.sqlxDB)
	}
	return db.integrationRepo
}

// OperationStatus returns the OperationStatus repository for endpoint operations
func (db *DB) OperationStatus() interfaces.OperationStatusRepository {
	if db.operationStatusRepo == nil {
		db.operationStatusRepo = NewOperationStatusRepository(db.sqlxDB)
	}
	return db.operationStatusRepo
}

// Manufacturers returns the Manufacturer repository for device catalog
// Blueprint feature (Migration 000102)
func (db *DB) Manufacturers() interfaces.ManufacturerRepository {
	if db.manufacturerRepo == nil {
		db.manufacturerRepo = NewManufacturerRepository(db.sqlxDB)
	}
	return db.manufacturerRepo
}

// DeviceModels returns the DeviceModel repository for device catalog
// Blueprint feature (Migration 000103)
func (db *DB) DeviceModels() interfaces.DeviceModelRepository {
	if db.deviceModelRepo == nil {
		db.deviceModelRepo = NewDeviceModelRepository(db.sqlxDB)
	}
	return db.deviceModelRepo
}

// Blueprints returns the Blueprint repository for payload decoding specifications
// Blueprint feature (Migration 000104)
func (db *DB) Blueprints() interfaces.BlueprintRepository {
	if db.blueprintRepo == nil {
		db.blueprintRepo = NewBlueprintRepository(db.sqlxDB)
	}
	return db.blueprintRepo
}

// SystemEvents returns the SystemEvent store for MIOTY event recording
func (db *DB) SystemEvents() interfaces.SystemEventStore {
	if db.systemEventsStore == nil {
		db.systemEventsStore = NewSystemEventStore(db.conn)
	}
	return db.systemEventsStore
}

// getMessageRepo returns the concrete MessageRepository for internal use
func (db *DB) getMessageRepo() *MessageRepository {
	if db.messageRepo == nil {
		db.messageRepo = NewMessageRepository(db.sqlxDB)
	}
	return db.messageRepo
}

// MIOTYMessages returns the MIOTY message repository interface
func (db *DB) MIOTYMessages() interfaces.MIOTYMessageRepository {
	return db.getMessageRepo() // MessageRepository implements interfaces.MIOTYMessageRepository
}

// getBaseStationStatusRepo returns the concrete BaseStationStatusRepository for internal use
func (db *DB) getBaseStationStatusRepo() *BaseStationStatusRepository {
	if db.basestationStatusRepo == nil {
		db.basestationStatusRepo = NewBaseStationStatusRepository(db.sqlxDB)
	}
	return db.basestationStatusRepo
}

// MIOTYBaseStationStatus returns the MIOTY base station status repository interface
func (db *DB) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return db.getBaseStationStatusRepo()
}

// getRoamingImplRepo returns the concrete RoamingRepository for internal use
func (db *DB) getRoamingImplRepo() *RoamingRepository {
	if db.roamingImplRepo == nil {
		db.roamingImplRepo = NewRoamingRepository(db.sqlxDB)
	}
	return db.roamingImplRepo
}

// MIOTYDownlinks returns the MIOTY downlink repository interface
// Wraps DB's downlink methods to implement the interface
func (db *DB) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository {
	return (*dbDownlinkWrapper)(db)
}

// dbDownlinkWrapper adapts *DB to interfaces.MIOTYDownlinkRepository
type dbDownlinkWrapper DB

func (w *dbDownlinkWrapper) EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	return (*DB)(w).EnqueueDownlink(ctx, downlink)
}
func (w *dbDownlinkWrapper) GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error) {
	return (*DB)(w).GetDownlinkQueue(ctx, deviceEUI, tenantID)
}
func (w *dbDownlinkWrapper) GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error) {
	return (*DB)(w).GetDownlinkByQueueID(ctx, queId, tenantID)
}
func (w *dbDownlinkWrapper) GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*storage.DownlinkMessage, error) {
	return (*DB)(w).GetDownlinkByPacketCnt(ctx, tenantID, epEui, packetCnt)
}
func (w *dbDownlinkWrapper) GetDownlinkResults(ctx context.Context, deviceEUI string, tenantID string, orgID *uuid.UUID, statusFilter string, timeFrom, timeTo *time.Time, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	return (*DB)(w).GetDownlinkResults(ctx, deviceEUI, tenantID, orgID, statusFilter, timeFrom, timeTo, limit, offset)
}
func (w *dbDownlinkWrapper) UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error {
	return (*DB)(w).UpdateDownlinkStatus(ctx, id, status, orgID)
}
func (w *dbDownlinkWrapper) UpdateDownlinkResult(ctx context.Context, queId int64, result string, txTime *int64, packetCnt *uint32, bsEUI []byte, epEUI []byte, tenantID string, orgID *uuid.UUID) error {
	return (*DB)(w).UpdateDownlinkResult(ctx, queId, result, txTime, packetCnt, bsEUI, epEUI, tenantID, orgID)
}
func (w *dbDownlinkWrapper) UpdateDownlinkBaseStation(ctx context.Context, queId uint64, tenantID string, bsEUI uint64) error {
	return (*DB)(w).UpdateDownlinkBaseStation(ctx, queId, tenantID, bsEUI)
}
func (w *dbDownlinkWrapper) RevokeDownlink(ctx context.Context, queId int64, tenantID string) error {
	return (*DB)(w).RevokeDownlink(ctx, queId, tenantID)
}

// ReserveNextPendingDownlink is tx-only; use Transaction.MIOTYDownlinks()
func (w *dbDownlinkWrapper) ReserveNextPendingDownlink(_ context.Context, _ int64, _ []byte, _ uint64, _ *uuid.UUID) (*storage.DownlinkMessage, error) {
	return nil, ErrNotImplemented
}

// MarkReservedAsQueued is tx-only; use Transaction.MIOTYDownlinks()
func (w *dbDownlinkWrapper) MarkReservedAsQueued(_ context.Context, _ uint64, _ int64, _ uint64, _ int64, _ *uint32, _ *uuid.UUID) error {
	return ErrNotImplemented
}

// SCACISessions returns the SCACI Session repository
func (db *DB) SCACISessions() interfaces.SCACISessionRepository {
	if db.scaciSessionRepo == nil {
		db.scaciSessionRepo = NewSCACISessionRepository(db.sqlxDB)
	}
	return db.scaciSessionRepo
}

// BaseStationSessions returns the Base Station Session repository (BSSCI 3.3)
func (db *DB) BaseStationSessions() interfaces.BaseStationSessionRepository {
	if db.basestationSessionRepo == nil {
		db.basestationSessionRepo = NewBaseStationSessionRepository(db.sqlxDB)
	}
	return db.basestationSessionRepo
}

// BaseStationReceptions returns the Base Station Reception repository
func (db *DB) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	if db.basestationReceptionRepo == nil {
		db.basestationReceptionRepo = NewBaseStationReceptionRepository(db)
	}
	return db.basestationReceptionRepo
}

// SCACIOperations returns the SCACI Operation repository
func (db *DB) SCACIOperations() interfaces.SCACIOperationRepository {
	if db.scaciOperationRepo == nil {
		db.scaciOperationRepo = NewSCACIOperationRepository(db.sqlxDB, db.log)
	}
	return db.scaciOperationRepo
}

// PendingOperations returns the Pending Operation repository
func (db *DB) PendingOperations() interfaces.PendingOperationRepository {
	if db.pendingOpRepo == nil {
		db.pendingOpRepo = NewPendingOperationRepository(db.sqlxDB, db.log)
	}
	return db.pendingOpRepo
}

// DLRXStatus returns the DL RX Status repository
func (db *DB) DLRXStatus() interfaces.DLRXStatusRepository {
	if db.dlRxStatusRepo == nil {
		db.dlRxStatusRepo = NewDLRXStatusRepository(db.sqlxDB, db.log)
	}
	return db.dlRxStatusRepo
}

// EndPointSessions returns the Endpoint Session repository
func (db *DB) EndPointSessions() interfaces.EndPointSessionRepository {
	if db.endpointSessionRepo == nil {
		db.endpointSessionRepo = NewEndPointSessionRepositoryWithEncryption(db, db.keyEncryptor, db.log)
	}
	return db.endpointSessionRepo
}

// EndPointKeys returns the Endpoint Key repository
func (db *DB) EndPointKeys() interfaces.EndPointKeyRepository {
	if db.endpointKeyRepo == nil {
		db.endpointKeyRepo = NewEndPointKeyRepository(db)
	}
	return db.endpointKeyRepo
}

// DownlinkQueue returns the Downlink Queue repository
func (db *DB) DownlinkQueue() interfaces.DownlinkQueueRepository {
	if db.downlinkQueueRepo == nil {
		db.log.Warn("DownlinkQueue repository not supported")
		return nil
	}
	return db.downlinkQueueRepo
}

// DownlinkQueueReader returns the Downlink Queue Reader
func (db *DB) DownlinkQueueReader() interfaces.DownlinkQueueReader {
	if db.queueReader == nil {
		db.queueReader = NewDownlinkQueueReader(db.sqlxDB)
	}
	return db.queueReader
}

// RoamingAgreements returns the Roaming Agreement repository
func (db *DB) RoamingAgreements() interfaces.RoamingAgreementRepository {
	if db.roamingRepo == nil {
		db.log.Warn("RoamingAgreements repository not supported")
		return nil
	}
	return db.roamingRepo
}

// CreateULDataMessage stores a new MIOTY UL Data message
func (db *DB) CreateULDataMessage(ctx context.Context, msg *mioty.ULDataMessage) error {
	return db.getMessageRepo().CreateULDataMessage(ctx, msg)
}

// GetULDataMessage retrieves a single UL Data message by ID
func (db *DB) GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error) {
	return db.getMessageRepo().GetULDataMessage(ctx, id, tenantID)
}

// ListULDataMessages retrieves messages with filtering and pagination
func (db *DB) ListULDataMessages(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return db.getMessageRepo().ListULDataMessages(ctx, filter)
}

// DL RX Status operations (BSSCI §3.15-3.16)

// CreateDLRXStatus persists a new DL RX status report
func (db *DB) CreateDLRXStatus(ctx context.Context, status *mioty.DLRXStatus) error {
	if db.dlRxStatusRepo == nil {
		return fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.CreateDLRXStatus(ctx, status)
}

// GetDLRXStatusByEndpoint retrieves DL RX status records for a specific endpoint
func (db *DB) GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error) {
	if db.dlRxStatusRepo == nil {
		return nil, 0, fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.GetDLRXStatusByEndpoint(ctx, tenantID, epEui, limit, offset, startTime, endTime)
}

// GetAverageDLRXMetrics calculates average SNR and RSSI for an endpoint
func (db *DB) GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error) {
	if db.dlRxStatusRepo == nil {
		return 0, 0, 0, fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.GetAverageDLRXMetrics(ctx, tenantID, epEui, startTime, endTime)
}

// CreateDLRXStatusQuery tracks a dlRxStatQry request for correlation (BSSCI §5.15 audit trail)
func (db *DB) CreateDLRXStatusQuery(ctx context.Context, tenantID int64, orgUUID *uuid.UUID, epEui, bsEui []byte, opId int64) error {
	if db.dlRxStatusRepo == nil {
		return fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.CreateDLRXStatusQuery(ctx, tenantID, orgUUID, epEui, bsEui, opId)
}

// MarkDLRXStatusReceived marks a query as received when dlRxStat arrives
// Correlates by tenant+endpoint only, selecting oldest pending (BSSCI §5.15)
// bsEui and bsOpID stored for audit only, NOT used in correlation WHERE clause
func (db *DB) MarkDLRXStatusReceived(ctx context.Context, tenantID int64, epEui []byte, bsEui []byte, bsOpID int64) (bool, error) {
	if db.dlRxStatusRepo == nil {
		return false, fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.MarkDLRXStatusReceived(ctx, tenantID, epEui, bsEui, bsOpID)
}

// ExpireDLRXStatusQuery marks queries older than cutoff as 'timeout'
func (db *DB) ExpireDLRXStatusQuery(ctx context.Context, cutoff time.Time) (int64, error) {
	if db.dlRxStatusRepo == nil {
		return 0, fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.ExpireDLRXStatusQuery(ctx, cutoff)
}

// GetDLRXStatusQueryHistory retrieves query tracking records for an endpoint (BSSCI §5.15 telemetry)
func (db *DB) GetDLRXStatusQueryHistory(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	if db.dlRxStatusRepo == nil {
		return nil, 0, fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.GetDLRXStatusQueryHistory(ctx, tenantID, epEui, limit, offset, startTime, endTime)
}

// GetDLRXStatusQueryStats calculates query status distribution for an endpoint (BSSCI §5.15 telemetry)
func (db *DB) GetDLRXStatusQueryStats(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (pending, received, timeout int64, err error) {
	if db.dlRxStatusRepo == nil {
		return 0, 0, 0, fmt.Errorf("DL RX status repository not initialized")
	}
	return db.dlRxStatusRepo.GetDLRXStatusQueryStats(ctx, tenantID, epEui, startTime, endTime)
}

// GetEndpointBaseStation retrieves the base station currently serving an endpoint
func (db *DB) GetEndpointBaseStation(ctx context.Context, epEui string, tenantID string) (string, error) {
	// Convert EUI from hex string to bytes
	epEuiBytes, err := hex.DecodeString(epEui)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint EUI: %w", err)
	}

	// Convert tenant ID to integer
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid tenant ID: %w", err)
	}

	query := `
		SELECT b.bs_eui
		FROM endpoints e
		JOIN endpoint_sessions es ON es.endpoint_id = e.id
		JOIN basestations b ON b.id = es.primary_basestation_id
		WHERE e.ep_eui = $1 AND e.tenant_id = $2 AND es.status = 'active'
		LIMIT 1
	`

	var bsEuiBytes []byte
	err = db.sqlxDB.QueryRowContext(ctx, query, epEuiBytes, tid).Scan(&bsEuiBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", storage.ErrNotFound
		}
		return "", fmt.Errorf("failed to get endpoint base station: %w", err)
	}

	return hex.EncodeToString(bsEuiBytes), nil
}

// GetBaseStationMessageStats retrieves aggregated message statistics for a base station
// Implements MessageStore interface
func (db *DB) GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error) {
	if db.messageRepo == nil {
		return nil, fmt.Errorf("message repository not initialized")
	}
	return db.getMessageRepo().GetBaseStationMessageStats(ctx, tenantID, bsEui, startTime, endTime)
}

// GetBaseStationEndpointCounts retrieves per-endpoint message counts for a base station
// Implements MessageStore interface
func (db *DB) GetBaseStationEndpointCounts(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time) (map[string]int64, error) {
	if db.messageRepo == nil {
		return nil, fmt.Errorf("message repository not initialized")
	}
	return db.getMessageRepo().GetBaseStationEndpointCounts(ctx, tenantID, bsEui, startTime, endTime)
}

// GetBaseStationLastSeen retrieves the last seen timestamp for a base station
// Implements MessageStore interface
func (db *DB) GetBaseStationLastSeen(ctx context.Context, tenantID int64, bsEui []byte) (*time.Time, error) {
	if db.messageRepo == nil {
		return nil, fmt.Errorf("message repository not initialized")
	}
	return db.getMessageRepo().GetBaseStationLastSeen(ctx, tenantID, bsEui)
}

// AddRoamingEndpointToSession adds a roaming endpoint to a base station session
func (db *DB) AddRoamingEndpointToSession(ctx context.Context, sessionID int64, epEui string, ownerTenantID int64) error {
	return db.getRoamingImplRepo().AddRoamingEndpointToSession(ctx, sessionID, epEui, ownerTenantID)
}

// RemoveRoamingEndpointFromSession removes a roaming endpoint from a base station session
func (db *DB) RemoveRoamingEndpointFromSession(ctx context.Context, sessionID int64, epEuiHex string) error {
	return db.getRoamingImplRepo().RemoveRoamingEndpointFromSession(ctx, sessionID, epEuiHex)
}

// GetEndpointOwner returns the owner tenant ID for an endpoint
func (db *DB) GetEndpointOwner(ctx context.Context, epEui []byte) (int64, error) {
	return db.getRoamingImplRepo().GetEndpointOwner(ctx, epEui)
}

// GetEndpointWithOwnership returns endpoint with ownership information
func (db *DB) GetEndpointWithOwnership(ctx context.Context, epEui []byte, servingTenantID int64) (*models.EndPoint, error) {
	return db.getRoamingImplRepo().GetEndpointWithOwnership(ctx, epEui, servingTenantID)
}

// IsRoamingEnabled checks if roaming is enabled for a tenant
func (db *DB) IsRoamingEnabled(ctx context.Context, tenantID int64) (bool, error) {
	return db.getRoamingImplRepo().IsRoamingEnabled(ctx, tenantID)
}

// AreTenantsPartners checks if two tenants have a roaming agreement
func (db *DB) AreTenantsPartners(ctx context.Context, tenant1, tenant2 int64) (bool, error) {
	return db.getRoamingImplRepo().AreTenantsPartners(ctx, tenant1, tenant2)
}

// RecordRoamingEvent records a roaming event
func (db *DB) RecordRoamingEvent(ctx context.Context, event *models.RoamingEvent) error {
	return db.getRoamingImplRepo().RecordRoamingEvent(ctx, event)
}

// GetRoamingStatistics retrieves roaming statistics for a tenant
func (db *DB) GetRoamingStatistics(ctx context.Context, tenantID int64) (*models.RoamingStatistics, error) {
	return db.getRoamingImplRepo().GetRoamingStatistics(ctx, tenantID)
}

// UpdateEndpointRoamingStatus updates roaming status for an endpoint
func (db *DB) UpdateEndpointRoamingStatus(ctx context.Context, epEui []byte, servingTenantID int64) error {
	return db.getRoamingImplRepo().UpdateEndpointRoamingStatus(ctx, epEui, servingTenantID)
}

// GetRoamingEndpointsInSession retrieves roaming endpoints in a session
func (db *DB) GetRoamingEndpointsInSession(ctx context.Context, sessionID int64) ([]models.RoamingEndpointInfo, error) {
	return db.getRoamingImplRepo().GetRoamingEndpointsInSession(ctx, sessionID)
}

// GetTenantRoamingConfig retrieves roaming configuration for a tenant
func (db *DB) GetTenantRoamingConfig(ctx context.Context, tenantID int64) (*models.TenantRoamingConfig, error) {
	return db.getRoamingImplRepo().GetTenantRoamingConfig(ctx, tenantID)
}

// UpdateTenantRoamingConfig updates roaming configuration for a tenant
func (db *DB) UpdateTenantRoamingConfig(ctx context.Context, tenantID int64, config *models.TenantRoamingConfig) error {
	return db.getRoamingImplRepo().UpdateTenantRoamingConfig(ctx, tenantID, config)
}
