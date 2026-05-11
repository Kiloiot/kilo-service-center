package postgres

import (
	"context"
	"encoding/json"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/jmoiron/sqlx"
)

// pendingOperationRepository implements interfaces.PendingOperationRepository
type pendingOperationRepository struct {
	db     *sqlx.DB
	logger logger.Logger
}

// NewPendingOperationRepository creates a new pending operation repository
func NewPendingOperationRepository(db *sqlx.DB, log logger.Logger) interfaces.PendingOperationRepository {
	return &pendingOperationRepository{
		db:     db,
		logger: log,
	}
}

// Create inserts or updates a pending operation (UPSERT pattern)
func (r *pendingOperationRepository) Create(ctx context.Context, req *interfaces.PendingOperationRequest) error {
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

// UpdateMetadata updates only the metadata field
func (r *pendingOperationRepository) UpdateMetadata(ctx context.Context, sessionID int64, operationID int64, metadata json.RawMessage) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE bssci_pending_operations
		SET metadata = $3, updated_at = NOW()
		WHERE basestation_session_id = $1 AND operation_id = $2
	`, sessionID, operationID, metadata)

	return err
}

// DeleteBySessionAndOperation removes by compound key
func (r *pendingOperationRepository) DeleteBySessionAndOperation(ctx context.Context, sessionID int64, operationID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM bssci_pending_operations
		WHERE basestation_session_id = $1 AND operation_id = $2
	`, sessionID, operationID)

	return err
}

// DeleteByOperation removes by operation ID only
func (r *pendingOperationRepository) DeleteByOperation(ctx context.Context, operationID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM bssci_pending_operations
		WHERE operation_id = $1
	`, operationID)

	return err
}

// DeleteBySession removes all pending operations for a session
// Used during session termination per BSSCI §3 "discarding state" requirement
func (r *pendingOperationRepository) DeleteBySession(ctx context.Context, sessionID int64) (int64, error) {
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
func (r *pendingOperationRepository) GetBySession(ctx context.Context, sessionID int64) ([]*interfaces.PendingOperation, error) {
	var ops []*interfaces.PendingOperation

	err := r.db.SelectContext(ctx, &ops, `
		SELECT id, basestation_session_id, operation_id, operation_type, endpoint_eui,
		       operation_data, metadata, created_at, updated_at
		FROM bssci_pending_operations
		WHERE basestation_session_id = $1
		ORDER BY created_at ASC
	`, sessionID)

	return ops, err
}
