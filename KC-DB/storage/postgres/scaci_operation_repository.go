package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// SCACIOperationRepository implements the SCACIOperationRepository interface for PostgreSQL
type SCACIOperationRepository struct {
	db  *sqlx.DB
	log logger.Logger
}

// Ensure SCACIOperationRepository implements the interface
var _ interfaces.SCACIOperationRepository = (*SCACIOperationRepository)(nil)

// NewSCACIOperationRepository creates a new PostgreSQL SCACI operation repository
func NewSCACIOperationRepository(db *sqlx.DB, log logger.Logger) *SCACIOperationRepository {
	return &SCACIOperationRepository{
		db:  db,
		log: log,
	}
}

// RecordOperation records a new SCACI operation
func (r *SCACIOperationRepository) RecordOperation(ctx context.Context, req *models.SCACIOperationRequest) (*models.SCACIOperation, error) {
	if req == nil {
		return nil, fmt.Errorf("operation request cannot be nil")
	}

	// Validate direction
	if req.Direction != string(models.OperationDirectionInbound) && req.Direction != string(models.OperationDirectionOutbound) {
		return nil, fmt.Errorf("invalid direction: %s (must be 'inbound' or 'outbound')", req.Direction)
	}

	// Serialize request data to JSONB
	var requestDataJSON []byte
	var err error
	if req.RequestData != nil {
		requestDataJSON, err = json.Marshal(req.RequestData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
	} else {
		requestDataJSON = []byte("{}")
	}

	query := `
		INSERT INTO scaci_operation_log (
			session_id, tenant_id, op_id, command, direction,
			state, request_data, initiated_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW()
		)
		RETURNING id, initiated_at, created_at, updated_at`

	operation := &models.SCACIOperation{
		SessionID:   req.SessionID,
		TenantID:    req.TenantID,
		OpId:        req.OpId,
		Command:     req.Command,
		Direction:   req.Direction,
		State:       string(models.OperationStatePending),
		RequestData: req.RequestData,
	}

	err = r.db.QueryRowContext(ctx, query,
		req.SessionID,
		req.TenantID,
		req.OpId,
		req.Command,
		req.Direction,
		string(models.OperationStatePending),
		requestDataJSON,
	).Scan(&operation.ID, &operation.InitiatedAt, &operation.CreatedAt, &operation.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to record operation: %w", err)
	}

	return operation, nil
}

// UpdateOperationState updates operation state
func (r *SCACIOperationRepository) UpdateOperationState(ctx context.Context, sessionID int64, opId int64, state models.OperationState, responseData map[string]interface{}) error {
	// Validate state
	validStates := map[models.OperationState]bool{
		models.OperationStatePending:               true,
		models.OperationStateAcknowledged:          true,
		models.OperationStateCompleted:             true,
		models.OperationStateCompletedWithWarnings: true,
		models.OperationStateFailed:                true,
	}
	if !validStates[state] {
		return fmt.Errorf("invalid operation state: %s", state)
	}

	// Serialize response data to JSONB
	var responseDataJSON []byte
	var err error
	if responseData != nil {
		responseDataJSON, err = json.Marshal(responseData)
		if err != nil {
			return fmt.Errorf("failed to marshal response data: %w", err)
		}
	}

	// Build query based on state transition
	var query string
	now := time.Now()

	switch state {
	case models.OperationStateAcknowledged:
		query = `
			UPDATE scaci_operation_log
			SET state = $1, response_data = $2, acknowledged_at = $3, updated_at = $4
			WHERE session_id = $5 AND op_id = $6 AND state = 'pending'`
		_, err = r.db.ExecContext(ctx, query, string(state), responseDataJSON, now, now, sessionID, opId)

	case models.OperationStateCompleted:
		query = `
			UPDATE scaci_operation_log
			SET state = $1, response_data = COALESCE($2, response_data), completed_at = $3, updated_at = $4
			WHERE session_id = $5 AND op_id = $6 AND state IN ('pending', 'acknowledged')`
		_, err = r.db.ExecContext(ctx, query, string(state), responseDataJSON, now, now, sessionID, opId)
	case models.OperationStateCompletedWithWarnings:
		query = `
			UPDATE scaci_operation_log
			SET state = $1, response_data = COALESCE($2, response_data), completed_at = $3, updated_at = $4
			WHERE session_id = $5 AND op_id = $6 AND state IN ('pending', 'acknowledged')`
		_, err = r.db.ExecContext(ctx, query, string(state), responseDataJSON, now, now, sessionID, opId)

	case models.OperationStateFailed:
		query = `
			UPDATE scaci_operation_log
			SET state = $1, response_data = COALESCE($2, response_data), completed_at = $3, updated_at = $4
			WHERE session_id = $5 AND op_id = $6 AND state != 'completed'`
		_, err = r.db.ExecContext(ctx, query, string(state), responseDataJSON, now, now, sessionID, opId)

	default:
		return fmt.Errorf("unsupported state transition to: %s", state)
	}

	if err != nil {
		return fmt.Errorf("failed to update operation state: %w", err)
	}

	return nil
}

// UpdateOperationStateWithError updates state with explicit error columns per SCACI §3.14
// Used when failing an operation with structured error metadata per SCACI §3.14
func (r *SCACIOperationRepository) UpdateOperationStateWithError(ctx context.Context, sessionID int64, opId int64,
	state models.OperationState, errorCode int, errorToken string, errorMessage string,
	responseData map[string]interface{}) error {

	// Only failed state makes sense with error details
	if state != models.OperationStateFailed {
		return fmt.Errorf("UpdateOperationStateWithError only valid for failed state, got: %s", state)
	}

	// Serialize response data to JSONB
	var responseDataJSON []byte
	var err error
	if responseData != nil {
		responseDataJSON, err = json.Marshal(responseData)
		if err != nil {
			return fmt.Errorf("failed to marshal response data: %w", err)
		}
	}

	now := time.Now()

	query := `
		UPDATE scaci_operation_log SET
			state = $1,
			error_code = $2,
			error_token = $3,
			error_message = $4,
			response_data = COALESCE(response_data, '{}'::jsonb) || COALESCE($5::jsonb, '{}'::jsonb),
			completed_at = $6,
			updated_at = $7
		WHERE session_id = $8 AND op_id = $9 AND state != 'completed'`

	_, err = r.db.ExecContext(ctx, query,
		string(state),
		errorCode,
		errorToken,
		errorMessage,
		responseDataJSON,
		now,
		now,
		sessionID,
		opId,
	)

	if err != nil {
		return fmt.Errorf("failed to update operation state with error: %w", err)
	}

	return nil
}

// CompleteFailedOperation sets completed_at on a failed operation without changing state.
// This is used by error handshake completion (errorAck) to mark the operation as "done"
// while preserving error evidence (state remains 'failed', error_code/error_token preserved).
func (r *SCACIOperationRepository) CompleteFailedOperation(ctx context.Context, sessionID int64, opId int64, responseData map[string]interface{}) error {
	// Serialize response data to JSONB
	var responseDataJSON []byte
	var err error
	if responseData != nil {
		responseDataJSON, err = json.Marshal(responseData)
		if err != nil {
			return fmt.Errorf("failed to marshal response data: %w", err)
		}
	}

	now := time.Now()

	// Only update if state is 'failed' - this preserves error evidence
	// while marking the operation as completed (via completed_at)
	query := `
		UPDATE scaci_operation_log SET
			response_data = COALESCE(response_data, '{}'::jsonb) || COALESCE($1::jsonb, '{}'::jsonb),
			completed_at = $2,
			updated_at = $3
		WHERE session_id = $4 AND op_id = $5 AND state = 'failed'`

	_, err = r.db.ExecContext(ctx, query,
		responseDataJSON,
		now,
		now,
		sessionID,
		opId,
	)

	if err != nil {
		return fmt.Errorf("failed to complete failed operation: %w", err)
	}

	return nil
}

// GetOperationByOpID retrieves an operation by session ID and operation ID
func (r *SCACIOperationRepository) GetOperationByOpID(ctx context.Context, sessionID int64, opId int64) (*models.SCACIOperation, error) {
	query := `
		SELECT
			id, session_id, tenant_id, op_id, command, direction,
			state, request_data, response_data, error_message,
			error_code, error_token,
			initiated_at, acknowledged_at, completed_at,
			created_at, updated_at
		FROM scaci_operation_log
		WHERE session_id = $1 AND op_id = $2
		ORDER BY initiated_at DESC
		LIMIT 1`

	return r.scanOperation(r.db.QueryRowContext(ctx, query, sessionID, opId))
}

// GetPendingOperations retrieves all pending operations for a session
func (r *SCACIOperationRepository) GetPendingOperations(ctx context.Context, sessionID int64) ([]*models.SCACIOperation, error) {
	query := `
		SELECT
			id, session_id, tenant_id, op_id, command, direction,
			state, request_data, response_data, error_message,
			error_code, error_token,
			initiated_at, acknowledged_at, completed_at,
			created_at, updated_at
		FROM scaci_operation_log
		WHERE session_id = $1 AND state IN ('pending', 'acknowledged')
		ORDER BY initiated_at ASC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending operations: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn("failed to close rows in SCACI operation query", "error", err)
		}
	}()

	operations := []*models.SCACIOperation{}
	for rows.Next() {
		operation, err := r.scanOperationFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operation: %w", err)
		}
		operations = append(operations, operation)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return operations, nil
}

// GetRecentOperations retrieves recent operations for a session
func (r *SCACIOperationRepository) GetRecentOperations(ctx context.Context, sessionID int64, limit int) ([]*models.SCACIOperation, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}

	query := `
		SELECT
			id, session_id, tenant_id, op_id, command, direction,
			state, request_data, response_data, error_message,
			error_code, error_token,
			initiated_at, acknowledged_at, completed_at,
			created_at, updated_at
		FROM scaci_operation_log
		WHERE session_id = $1
		ORDER BY initiated_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent operations: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn("failed to close rows in SCACI operation query", "error", err)
		}
	}()

	operations := []*models.SCACIOperation{}
	for rows.Next() {
		operation, err := r.scanOperationFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operation: %w", err)
		}
		operations = append(operations, operation)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return operations, nil
}

// CleanupCompletedOperations removes old completed operations
func (r *SCACIOperationRepository) CleanupCompletedOperations(ctx context.Context, olderThan int64) (int64, error) {
	query := `
		DELETE FROM scaci_operation_log
			WHERE state IN ('completed', 'completed_with_warnings', 'failed')
			  AND completed_at < NOW() - INTERVAL '1 hour' * $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup completed operations: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return deleted, nil
}

// GetTenantOperationSummary retrieves aggregated operation statistics for a tenant
func (r *SCACIOperationRepository) GetTenantOperationSummary(ctx context.Context, tenantID int64, windowHours int) (*models.SCACIOperationSummary, error) {
	if windowHours <= 0 {
		windowHours = 24 // Default to 24-hour window
	}

	query := `
		SELECT
			command,
			direction,
			state,
			COUNT(*) as count
		FROM scaci_operation_log
		WHERE tenant_id = $1
		  AND created_at >= NOW() - INTERVAL '1 hour' * $2
		GROUP BY command, direction, state
		ORDER BY command, direction, state`

	rows, err := r.db.QueryContext(ctx, query, tenantID, windowHours)
	if err != nil {
		return nil, fmt.Errorf("failed to query operation summary: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn("failed to close rows in SCACI operation query", "error", err)
		}
	}()

	// Initialize maps
	commandCounts := make(map[string]int64)
	directionCounts := make(map[string]int64)
	stateCounts := make(map[string]int64)
	var totalOperations int64
	var errorCount int64

	for rows.Next() {
		var command, direction, state string
		var count int64

		if err := rows.Scan(&command, &direction, &state, &count); err != nil {
			return nil, fmt.Errorf("failed to scan operation summary row: %w", err)
		}

		// Aggregate counts
		commandCounts[command] += count
		directionCounts[direction] += count
		stateCounts[state] += count
		totalOperations += count

		// Track failed operations
		if state == string(models.OperationStateFailed) {
			errorCount += count
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Return zeroed summary if no operations found
	summary := &models.SCACIOperationSummary{
		TenantID:        tenantID,
		WindowHours:     windowHours,
		TotalOperations: totalOperations,
		CommandCounts:   commandCounts,
		DirectionCounts: directionCounts,
		StateCounts:     stateCounts,
		ErrorCount:      errorCount,
	}

	return summary, nil
}

// scanOperation is a helper to scan a single operation from a query row
func (r *SCACIOperationRepository) scanOperation(row *sql.Row) (*models.SCACIOperation, error) {
	var operation models.SCACIOperation
	var requestDataJSON, responseDataJSON []byte

	err := row.Scan(
		&operation.ID,
		&operation.SessionID,
		&operation.TenantID,
		&operation.OpId,
		&operation.Command,
		&operation.Direction,
		&operation.State,
		&requestDataJSON,
		&responseDataJSON,
		&operation.ErrorMessage,
		&operation.ErrorCode,
		&operation.ErrorToken,
		&operation.InitiatedAt,
		&operation.AcknowledgedAt,
		&operation.CompletedAt,
		&operation.CreatedAt,
		&operation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan operation: %w", err)
	}

	// Unmarshal JSON data
	if len(requestDataJSON) > 0 && string(requestDataJSON) != "{}" {
		if err := json.Unmarshal(requestDataJSON, &operation.RequestData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request data: %w", err)
		}
	}

	if len(responseDataJSON) > 0 && string(responseDataJSON) != "{}" {
		if err := json.Unmarshal(responseDataJSON, &operation.ResponseData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return &operation, nil
}

// scanOperationFromRows is a helper to scan an operation from sql.Rows
func (r *SCACIOperationRepository) scanOperationFromRows(rows *sql.Rows) (*models.SCACIOperation, error) {
	var operation models.SCACIOperation
	var requestDataJSON, responseDataJSON []byte

	err := rows.Scan(
		&operation.ID,
		&operation.SessionID,
		&operation.TenantID,
		&operation.OpId,
		&operation.Command,
		&operation.Direction,
		&operation.State,
		&requestDataJSON,
		&responseDataJSON,
		&operation.ErrorMessage,
		&operation.ErrorCode,
		&operation.ErrorToken,
		&operation.InitiatedAt,
		&operation.AcknowledgedAt,
		&operation.CompletedAt,
		&operation.CreatedAt,
		&operation.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan operation: %w", err)
	}

	// Unmarshal JSON data
	if len(requestDataJSON) > 0 && string(requestDataJSON) != "{}" {
		if err := json.Unmarshal(requestDataJSON, &operation.RequestData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request data: %w", err)
		}
	}

	if len(responseDataJSON) > 0 && string(responseDataJSON) != "{}" {
		if err := json.Unmarshal(responseDataJSON, &operation.ResponseData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return &operation, nil
}
