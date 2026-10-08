package postgres

import (
	"context"
	"encoding/json"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// PendingOperationRepository persists BSSCI pending operations.
type PendingOperationRepository struct {
	db     *sqlx.DB
	logger logger.Logger
}

// NewPendingOperationRepository creates a new pending operation repository
func NewPendingOperationRepository(db *sqlx.DB, log logger.Logger) *PendingOperationRepository {
	return &PendingOperationRepository{
		db:     db,
		logger: log,
	}
}

// Create inserts or updates a pending operation (UPSERT pattern)
func (r *PendingOperationRepository) Create(ctx context.Context, req *models.PendingOperationRequest) error {
	var metadataArg interface{}
	if req.Metadata != nil {
		metadataArg = req.Metadata
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bssci_pending_operations
		(basestation_session_id, operation_id, operation_type, endpoint_eui, operation_data, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (basestation_session_id, operation_id)
		DO UPDATE SET
			operation_type = EXCLUDED.operation_type,
			endpoint_eui = EXCLUDED.endpoint_eui,
			operation_data = EXCLUDED.operation_data,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, req.SessionID, req.OperationID, req.OperationType, req.EndpointEUI, req.OperationData, metadataArg)

	return err
}

// CreateBatch inserts or updates several pending operations in one local
// transaction so a multi-frame sequence is recorded all-or-nothing. Requests
// are inserted in slice order so the id column preserves reissue order.
func (r *PendingOperationRepository) CreateBatch(ctx context.Context, reqs []*models.PendingOperationRequest) (err error) {
	if len(reqs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	for _, req := range reqs {
		var metadataArg interface{}
		if req.Metadata != nil {
			metadataArg = req.Metadata
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO bssci_pending_operations
			(basestation_session_id, operation_id, operation_type, endpoint_eui, operation_data, metadata)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (basestation_session_id, operation_id)
			DO UPDATE SET
				operation_type = EXCLUDED.operation_type,
				endpoint_eui = EXCLUDED.endpoint_eui,
				operation_data = EXCLUDED.operation_data,
				metadata = EXCLUDED.metadata,
				updated_at = NOW()
		`, req.SessionID, req.OperationID, req.OperationType, req.EndpointEUI, req.OperationData, metadataArg); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateMetadata updates only the metadata field
func (r *PendingOperationRepository) UpdateMetadata(ctx context.Context, sessionID int64, operationID int64, metadata json.RawMessage) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE bssci_pending_operations
		SET metadata = $3, updated_at = NOW()
		WHERE basestation_session_id = $1 AND operation_id = $2
	`, sessionID, operationID, metadata)

	return err
}

// DeleteBySessionAndOperation removes by compound key
func (r *PendingOperationRepository) DeleteBySessionAndOperation(ctx context.Context, sessionID int64, operationID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM bssci_pending_operations
		WHERE basestation_session_id = $1 AND operation_id = $2
	`, sessionID, operationID)

	return err
}

// DeleteBySession removes all pending operations for a session
// Used during session termination per BSSCI §3 "discarding state" requirement
func (r *PendingOperationRepository) DeleteBySession(ctx context.Context, sessionID int64) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM bssci_pending_operations
		WHERE basestation_session_id = $1
	`, sessionID)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// GetBySession retrieves all pending operations for a session
func (r *PendingOperationRepository) GetBySession(ctx context.Context, sessionID int64) ([]*models.PendingOperation, error) {
	var ops []*models.PendingOperation

	err := r.db.SelectContext(ctx, &ops, `
		SELECT id, basestation_session_id, operation_id, operation_type, endpoint_eui,
		       operation_data, metadata, created_at, updated_at
		FROM bssci_pending_operations
		WHERE basestation_session_id = $1
		ORDER BY created_at ASC, id ASC
	`, sessionID)

	return ops, err
}
