package postgres

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestCreateSessionPersistsProtocolVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() {
		// Ignore close errors from sqlmock - it's expected since we don't set ExpectClose()
		_ = db.Close()
	}()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewBaseStationSessionRepository(sqlxDB)

	bsUUID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	scUUID := [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	protocolVersion := "1.0.0"

	rows := sqlmock.NewRows([]string{"id", "started_at", "created_at", "updated_at"}).
		AddRow(int64(1), time.Now(), time.Now(), time.Now())

	query := regexp.QuoteMeta(`
        INSERT INTO basestation_sessions (
            basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid,
            sn_bs_op_id, sn_sc_op_id, status,
            connection_id, remote_addr, can_resume,
            organization_id, encoding, protocol_version, connect_info, started_at, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, COALESCE($14, '{}'::jsonb), NOW(), NOW(), NOW()
        )
        RETURNING id, started_at, created_at, updated_at`)

	mock.ExpectQuery(query).
		WithArgs(
			int64(10),
			int64(20),
			bsUUID[:],
			scUUID[:],
			0,
			0,
			models.SessionStatusActive,
			(*string)(nil),
			(*string)(nil),
			true,
			(*uuid.UUID)(nil),
			bssci.EncodingMessagePack,
			&protocolVersion,
			(*map[string]interface{})(nil), // connect_info
		).
		WillReturnRows(rows)

	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:   10,
		TenantID:        20,
		SnBsUuid:        bsUUID,
		SnScUuid:        scUUID,
		CanResume:       true,
		Encoding:        bssci.EncodingMessagePack,
		ProtocolVersion: &protocolVersion,
	}

	if _, err := repo.CreateSession(context.Background(), req); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
