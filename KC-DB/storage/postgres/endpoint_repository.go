package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/hstore"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// EndPointRepository implements interfaces.EndpointRepository for PostgreSQL
type EndPointRepository struct {
	log   logger.Logger
	clock clock.Clock
	db    sqlx.ExtContext
	// conn opens the cascade transaction in UpdateWithEUI; nil when the
	// repository already runs inside a transaction.
	conn   *sqlx.DB
	cipher keycrypto.Cipher
}

// NewEndPointRepository creates a new endpoints table repository. The cipher
// encrypts nwk_key / app_key on write and decrypts them on read.
func NewEndPointRepository(db *sqlx.DB, cipher keycrypto.Cipher, clk clock.Clock, log logger.Logger) *EndPointRepository {
	return &EndPointRepository{
		log: log, clock: clk, db: db, conn: db, cipher: cipher}
}

// Create creates a new endpoint
func (r *EndPointRepository) Create(ctx context.Context, endpoint *models.EndPoint) error {
	return r.insertEndpoint(ctx, r.db, endpoint)
}

// CreateWithStatus creates the endpoint and records its attachment status, and
// the time of that decision, in one transaction.
func (r *EndPointRepository) CreateWithStatus(ctx context.Context, endpoint *models.EndPoint, status string) (err error) {
	tx, commit, rollback, err := r.beginCascade(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapBeginTransaction, err)
	}
	defer rollback(&err)
	if err := r.insertEndpoint(ctx, tx, endpoint); err != nil {
		return err
	}
	if _, err := r.transitionEndpointStatus(ctx, tx, endpoint.TenantID, endpoint.ID, status); err != nil {
		return err
	}
	if err := commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapCommitTransaction, err)
	}
	endpoint.EpStatus = status
	return nil
}

func (r *EndPointRepository) insertEndpoint(ctx context.Context, q sqlx.QueryerContext, endpoint *models.EndPoint) error {
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
			dual_chan, repetition, wide_carr_off, long_blk_dist,
			blueprint_snapshot
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
		RETURNING id, created_at, updated_at`

	// NwkSnKey validation is enforced at the handler/service layer.
	// Silent zero-fill removed to prevent invalid endpoints from being persisted.
	// AppKey: Column allows NULL (migration 001, no NOT NULL constraint); CHECK(length=16) applies only when non-null.
	// We let nil remain nil in DB rather than zero-fill, consistent with NwkSnKey treatment.

	tags := endpointTags(endpoint.Tags)
	class := mioty.EndpointClass(endpoint.Bidi)

	// Encrypt key material at rest. Empty input yields a nil param so the column
	// stays SQL NULL (pq converts a nil []byte to empty bytea, not NULL, so the
	// explicit interface{} nil is required).
	nwkKeyEnc, err := encryptKeyMaterial(r.cipher, endpoint.NwkSnKey)
	if err != nil {
		return err
	}
	appKeyEnc, err := encryptKeyMaterial(r.cipher, endpoint.AppKey)
	if err != nil {
		return err
	}
	var nwkKeyParam interface{}
	if nwkKeyEnc != nil {
		nwkKeyParam = nwkKeyEnc
	}
	var appKeyParam interface{}
	if appKeyEnc != nil {
		appKeyParam = appKeyEnc
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

	// JSONB: nil must be SQL NULL (empty bytea is invalid JSON).
	var blueprintSnapshotParam interface{}
	if len(endpoint.BlueprintSnapshot) > 0 {
		blueprintSnapshotParam = []byte(endpoint.BlueprintSnapshot)
	}

	err = q.QueryRowxContext(
		ctx, query,
		endpoint.EUI[:],
		endpoint.Name,
		endpoint.Description,
		endpoint.TenantID,
		endpoint.OwnerTenantID,
		nwkKeyParam,
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
		class,
		endpoint.Bidi,
		endpoint.PreAttach,
		typeEuiParam,
		attachCntParam,
		int64(endpoint.LastPacketCnt),
		endpoint.DualChan,
		endpoint.Repetition,
		endpoint.WideCarrOff,
		endpoint.LongBlkDist,
		blueprintSnapshotParam,
	).Scan(&endpoint.ID, &endpoint.CreatedAt, &endpoint.UpdatedAt)
	if err != nil {
		return WrapDuplicateError(err, resourceEndpoint)
	}

	endpoint.EPClass = class
	return nil
}

// ListAllEUIs returns every endpoint EUI across all tenants. Used to pre-warm
// the ingress disposition index at startup.
func (r *EndPointRepository) ListAllEUIs(ctx context.Context) ([]models.EUI, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ep_eui FROM endpoints`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListEndpointEUIs, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsEndpoints, logger.FieldError, err)
		}
	}()

	var euis []models.EUI
	for rows.Next() {
		var raw []byte
		if scanErr := rows.Scan(&raw); scanErr != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanEndpointEUI, scanErr)
		}
		var eui models.EUI
		copy(eui[:], raw)
		euis = append(euis, eui)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateEndpointEUIs, rowsErr)
	}
	return euis, nil
}

// Get retrieves an endpoint by EUI
func (r *EndPointRepository) Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error) {
	query := `SELECT ` + endpointBaseSelectColumns + ` FROM endpoints WHERE ep_eui = $1`

	endpoint, err := scanEndpointBaseRow(r.db.QueryRowxContext(ctx, query, eui[:]))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetEndpoint, err)
	}
	if err := decryptEndpointKeys(r.cipher, endpoint); err != nil {
		return nil, err
	}

	return endpoint, nil
}

// GetByEUI retrieves an endpoint by EUI for a specific tenant
func (r *EndPointRepository) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) {
	query := `SELECT ` + endpointTenantLookupColumns + ` FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`
	endpoint, err := scanEndpointTenantLookupRow(r.db.QueryRowxContext(ctx, query, tenantID, eui))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetEndpoint, err)
	}
	if err := decryptEndpointKeys(r.cipher, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// RegisteredAt is when the tenant registered the endpoint an EUI names, read
// without its session keys; storage.ErrNotFound when it has not.
func (r *EndPointRepository) RegisteredAt(ctx context.Context, tenantID int64, eui []byte) (time.Time, error) {
	var registeredAt time.Time
	err := r.db.QueryRowxContext(ctx, sqlEndpointRegisteredAt, tenantID, eui).Scan(&registeredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, storage.ErrNotFound
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", errWrapGetEndpointRegisteredAt, err)
	}
	return registeredAt, nil
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
		return nil, fmt.Errorf("%s: %w", errWrapQueryEndpoints, err)
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
		var lastDetachSign []byte
		var lastAttachedBsEui []byte
		var lastPropagateTime, lastDetachTime, lastDetachPacketCnt sql.NullInt64
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
			return nil, fmt.Errorf("%s: %w", errWrapFailedToScanEndpoint, err)
		}

		applyEndpointPostScan(endpoint, tags, lastDetachSign,
			lastAttachedBsEui, lastPropagateTime, lastDetachTime, lastDetachPacketCnt,
			propagateStatus)

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
		if lastPacketCnt < 0 || lastPacketCnt > math.MaxUint32 {
			return nil, fmt.Errorf(errFmtLastPacketCntOutOfUint32RangeFor, lastPacketCnt, endpoint.ID)
		}
		endpoint.LastPacketCnt = uint32(lastPacketCnt)

		// NULLABLE assignment
		if propagatedAt.Valid {
			endpoint.PropagatedAt = &propagatedAt.Time
		}

		if err := decryptEndpointKeys(r.cipher, endpoint); err != nil {
			return nil, err
		}

		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateTenantEndpoints, err)
	}

	return endpoints, nil
}

// CountByTenant returns the total count of endpoints for a tenant
func (r *EndPointRepository) CountByTenant(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM endpoints WHERE tenant_id = $1`

	err := r.db.QueryRowxContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountEndpointsByTenant, err)
	}

	return count, nil
}

// GetByID retrieves an endpoint by ID with tenant isolation
func (r *EndPointRepository) GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	query := `SELECT ` + endpointDetailColumns + ` FROM endpoints WHERE id = $1 AND tenant_id = $2`
	endpoint, err := scanEndpointDetailRow(r.db.QueryRowxContext(ctx, query, id, tenantID))
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetEndpointByID, err)
	}
	if err := decryptEndpointKeys(r.cipher, endpoint); err != nil {
		return nil, err
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
		return nil, fmt.Errorf("%s: %w", errWrapListEndpointsPaginated, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsEndpoints, logger.FieldError, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapIterateEndpoints, err)
	}

	return endpoints, nil
}

// ListByModelWithSnapshot returns a tenant's snapshot-bearing endpoints for a device model.
func (r *EndPointRepository) ListByModelWithSnapshot(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) ([]*models.EndPoint, error) {
	query := `SELECT` + endpointListSelectColumns + `
		FROM endpoints
		WHERE tenant_id = $1 AND device_model_id = $2 AND blueprint_snapshot IS NOT NULL
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, deviceModelID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListEndpointsByModel, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsEndpoints, logger.FieldError, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapIterateEndpoints, err)
	}

	return endpoints, nil
}

// Update updates an endpoint with tenant isolation
func (r *EndPointRepository) Update(ctx context.Context, endpoint *models.EndPoint) error {
	err := r.updateEndpointFields(ctx, r.db, endpoint.ID, endpoint.TenantID, endpoint.EUI[:], endpoint)
	if err == sql.ErrNoRows {
		return storage.ErrNotFound
	}
	if err != nil {
		if classified := classifyEndpointPQError(err); classified != nil {
			return classified
		}
		return fmt.Errorf("%s: %w", errWrapUpdateEndpoint, err)
	}
	return nil
}

// classifyEndpointPQError maps the PostgreSQL constraint and syntax failures an
// endpoint write can raise onto storage sentinels; nil means the error is not a
// classified PostgreSQL failure.
func classifyEndpointPQError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return nil
	}
	switch pqErr.Code {
	case pqCodeUniqueViolation:
		return storage.ErrAlreadyExists
	case pqCodeForeignKeyViolation:
		if pqErr.Constraint == constraintDeviceModel {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf(errFmtForeignKeyViolation, pqErr.Constraint, err)
	case pqCodeCheckViolation:
		switch pqErr.Constraint {
		case constraintNwkKey:
			return storage.ErrNwkKeyLength
		case constraintAppKey:
			return storage.ErrAppKeyLength
		default:
			return storage.ErrCheckViolation
		}
	case pqCodeInvalidTextRep:
		return storage.ErrInvalidInput
	default:
		return nil
	}
}

// CheckEUIUnique verifies no endpoint with the given EUI exists (global uniqueness)
func (r *EndPointRepository) CheckEUIUnique(ctx context.Context, eui []byte) error {
	var count int
	err := sqlx.GetContext(ctx, r.db, &count, "SELECT COUNT(*) FROM endpoints WHERE ep_eui = $1", eui)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCheckEUIUniqueness, err)
	}
	if count > 0 {
		return storage.ErrAlreadyExists
	}
	return nil
}

// DeleteByTenant deletes the tenant's endpoint with the EUI and returns its
// id; storage.ErrNotFound when the tenant has no such endpoint.
func (r *EndPointRepository) DeleteByTenant(ctx context.Context, tenantID int64, eui []byte) (int64, error) {
	var removedID int64
	err := r.db.QueryRowxContext(ctx, sqlDeleteEndpointByTenant, tenantID, eui).Scan(&removedID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, storage.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapDeleteEndpoint, err)
	}
	return removedID, nil
}

// sqlDeleteEndpointByTenant deletes one tenant's endpoint by EUI.
const sqlDeleteEndpointByTenant = `DELETE FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2 RETURNING id`

const sqlEndpointRegisteredAt = `SELECT created_at FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`

// endpointFieldExec is the executor surface shared by the endpoints-table
// field updates so the non-transactional repository and the transaction-scoped
// repository run the same query.
type endpointFieldExec interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// ============================================================================
// Roaming support methods.
// ============================================================================
