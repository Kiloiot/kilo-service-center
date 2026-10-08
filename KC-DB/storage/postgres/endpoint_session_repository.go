package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// Session keys (endpoint_sessions.session_key) are stored as AES-256-GCM
// envelopes via pkg/keycrypto. Reads are migration-tolerant: rows written
// before this format (raw GCM without the envelope magic) pass through
// unchanged so historical sessions still decrypt through their own path.

// EndPointSessionRepository implements interfaces.EndPointSessionRepository
type EndPointSessionRepository struct {
	db     sqlx.ExtContext
	cipher keycrypto.Cipher
	logger logger.Logger
}

// NewEndPointSessionRepository creates a new endpoint session repository. The
// cipher encrypts session_key on write and decrypts it on read.
func NewEndPointSessionRepository(
	db sqlx.ExtContext,
	cipher keycrypto.Cipher,
	log logger.Logger,
) *EndPointSessionRepository {
	return &EndPointSessionRepository{
		db:     db,
		cipher: cipher,
		logger: log,
	}
}

// Create creates a new endpoint session
func (r *EndPointSessionRepository) Create(ctx context.Context, session *models.EndPointSession) error {
	query := `
		INSERT INTO endpoint_sessions (
			endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at,
			sh_addr, last_packet_cnt, uplink_mode,
			dl_open, res_exp, dl_ack, repetition,
			primary_basestation_id, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at`

	// Default metadata if nil
	if session.Metadata == nil {
		session.Metadata = json.RawMessage("{}")
	}

	sessionKeyEnc, err := encryptKeyMaterial(r.cipher, session.SessionKey)
	if err != nil {
		return err
	}

	err = r.db.QueryRowxContext(
		ctx, query,
		session.EndPointID,
		session.TenantID,
		session.SessionID,
		nullableByteParam(sessionKeyEnc),
		session.AttachCnt,
		session.Status,
		session.StartedAt,
		session.LastActivityAt,
		session.ShAddr,
		session.PacketCnt,
		session.UplinkMode,
		session.DlOpen,
		session.ResExp,
		session.DlAck,
		session.Repetition,
		session.PrimaryBaseStationID,
		session.Metadata,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateDeviceSession, err)
	}

	return nil
}

// GetActive retrieves the active session for an endpoint
func (r *EndPointSessionRepository) GetActive(ctx context.Context, endpointID string) (*models.EndPointSession, error) {
	query := `
		SELECT
			id, endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at, ended_at,
			sh_addr, last_packet_cnt, uplink_mode,
			dl_open, res_exp, dl_ack, repetition,
			primary_basestation_id, uplink_count, downlink_count,
			metadata, created_at, updated_at
		FROM endpoint_sessions
		WHERE endpoint_id = $1 AND status = 'active'`

	session := &models.EndPointSession{}
	err := r.db.QueryRowxContext(ctx, query, endpointID).Scan(
		&session.ID,
		&session.EndPointID,
		&session.TenantID,
		&session.SessionID,
		&session.SessionKey,
		&session.AttachCnt,
		&session.Status,
		&session.StartedAt,
		&session.LastActivityAt,
		&session.EndedAt,
		&session.ShAddr,
		&session.PacketCnt,
		&session.UplinkMode,
		&session.DlOpen,
		&session.ResExp,
		&session.DlAck,
		&session.Repetition,
		&session.PrimaryBaseStationID,
		&session.UplinkCount,
		&session.DownlinkCount,
		&session.Metadata,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetActiveSession, err)
	}

	if err := decryptSessionKey(r.cipher, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetByID retrieves a session by ID
func (r *EndPointSessionRepository) GetByID(ctx context.Context, id string) (*models.EndPointSession, error) {
	query := `
		SELECT
			id, endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at, ended_at,
			sh_addr, last_packet_cnt, uplink_mode,
			dl_open, res_exp, dl_ack, repetition,
			primary_basestation_id, uplink_count, downlink_count,
			metadata, created_at, updated_at
		FROM endpoint_sessions
		WHERE id = $1`

	session := &models.EndPointSession{}
	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&session.ID,
		&session.EndPointID,
		&session.TenantID,
		&session.SessionID,
		&session.SessionKey,
		&session.AttachCnt,
		&session.Status,
		&session.StartedAt,
		&session.LastActivityAt,
		&session.EndedAt,
		&session.ShAddr,
		&session.PacketCnt,
		&session.UplinkMode,
		&session.DlOpen,
		&session.ResExp,
		&session.DlAck,
		&session.Repetition,
		&session.PrimaryBaseStationID,
		&session.UplinkCount,
		&session.DownlinkCount,
		&session.Metadata,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetSessionByID, err)
	}

	if err := decryptSessionKey(r.cipher, session); err != nil {
		return nil, err
	}

	return session, nil
}

// Update updates an endpoint session
func (r *EndPointSessionRepository) Update(ctx context.Context, session *models.EndPointSession) error {
	query := `
		UPDATE endpoint_sessions SET
			session_key = $2,
			attach_cnt = $3,
			sh_addr = $4,
			last_packet_cnt = $5,
			repetition = $6,
			status = $7,
			last_activity_at = $8,
			ended_at = $9,
			uplink_count = $10,
			downlink_count = $11,
			metadata = $12,
			uplink_mode = $13,
			dl_open = $14,
			res_exp = $15,
			dl_ack = $16,
			primary_basestation_id = $17,
			updated_at = NOW()
		WHERE id = $1 AND tenant_id = $18
		RETURNING updated_at`

	sessionKeyEnc, err := encryptKeyMaterial(r.cipher, session.SessionKey)
	if err != nil {
		return err
	}

	err = r.db.QueryRowxContext(
		ctx, query,
		session.ID,
		nullableByteParam(sessionKeyEnc),
		session.AttachCnt,
		session.ShAddr,
		session.PacketCnt, // Note: model field is PacketCnt, DB column is last_packet_cnt
		session.Repetition,
		session.Status,
		session.LastActivityAt,
		session.EndedAt,
		session.UplinkCount,
		session.DownlinkCount,
		session.Metadata,
		session.UplinkMode,
		session.DlOpen,
		session.ResExp,
		session.DlAck,
		session.PrimaryBaseStationID,
		session.TenantID, // WHERE clause for tenant isolation
	).Scan(&session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateDeviceSession, err)
	}

	return nil
}
