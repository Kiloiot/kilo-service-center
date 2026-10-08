package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// uplinkDeliveryKey is the stored uplink an outbound ulData record delivers.
const uplinkDeliveryKey = "(request_data->>'" + models.OperationRequestKeySourceMessageID + "')"

// uplinkDeliveryRecords is the predicate of uq_scaci_op_log_uplink_delivery
// (migration 000186), spelled the same so the conflict target infers it.
const uplinkDeliveryRecords = "command = '" + mioty.CmdULData + "' AND direction = '" +
	string(models.OperationDirectionOutbound) + "' AND " + uplinkDeliveryKey + " IS NOT NULL"

const (
	sqlEnsureUplinkOperation = `
		INSERT INTO scaci_operation_log (
			session_id, tenant_id, op_id, command, direction,
			state, request_data, initiated_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $8, $8
		)
		ON CONFLICT (session_id, ` + uplinkDeliveryKey + `) WHERE ` + uplinkDeliveryRecords + `
		DO NOTHING
		RETURNING id, initiated_at, created_at, updated_at`
	sqlUplinkOperation = sqlSelectSCACIOperations + `
		WHERE session_id = $1 AND ` + uplinkDeliveryRecords + ` AND ` + uplinkDeliveryKey + ` = $2`
)

// EnsureUplinkOperation records, under req.OpId, the outbound ulData operation
// that delivers the stored uplink sourceMessageID to the session of req,
// unless the session has one already: an operation keeps its opId and state
// however often the delivery is retried (SCACI §3.2). It returns the session's
// operation for the uplink and whether it recorded it now.
func (r *SCACIOperationRepository) EnsureUplinkOperation(ctx context.Context, req *models.SCACIOperationRequest, sourceMessageID string) (*models.SCACIOperation, bool, error) {
	if !isUplinkDeliveryRequest(req, sourceMessageID) {
		return nil, false, fmt.Errorf("%s: %w", errWrapUplinkOperationRequest, storage.ErrInvalidInput)
	}
	requestData := make(map[string]interface{}, len(req.RequestData)+1)
	for key, value := range req.RequestData {
		requestData[key] = value
	}
	requestData[models.OperationRequestKeySourceMessageID] = sourceMessageID
	requestDataJSON, err := json.Marshal(requestData)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", errWrapMarshalRequestData, err)
	}

	operation := &models.SCACIOperation{
		SessionID:   req.SessionID,
		TenantID:    req.TenantID,
		OpId:        req.OpId,
		Command:     req.Command,
		Direction:   req.Direction,
		State:       string(models.OperationStatePending),
		RequestData: requestData,
	}
	err = r.db.QueryRowContext(ctx, sqlEnsureUplinkOperation,
		req.SessionID, req.TenantID, req.OpId, req.Command, req.Direction,
		string(models.OperationStatePending), requestDataJSON, r.clock.Now(),
	).Scan(&operation.ID, &operation.InitiatedAt, &operation.CreatedAt, &operation.UpdatedAt)
	if err == nil {
		return operation, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("%s: %w", errWrapEnsureUplinkOperation, err)
	}
	existing, err := r.scanOperation(r.db.QueryRowContext(ctx, sqlUplinkOperation, req.SessionID, sourceMessageID))
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", errWrapGetUplinkOperation, err)
	}
	return existing, false, nil
}

// isUplinkDeliveryRequest reports a request for a session's outbound ulData
// that names the stored uplink it delivers.
func isUplinkDeliveryRequest(req *models.SCACIOperationRequest, sourceMessageID string) bool {
	return req != nil && req.SessionID > 0 && sourceMessageID != "" &&
		req.Command == mioty.CmdULData && req.Direction == string(models.OperationDirectionOutbound)
}
