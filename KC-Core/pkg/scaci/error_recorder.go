// Package scaci error_recorder.go implements SCACI §3.14 error persistence.
package scaci

import (
	"context"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// errorRecorderImpl implements the ErrorRecorder interface per SCACI §3.14.
// It persists error information to the operation log and emits system events.
type errorRecorderImpl struct {
	operationRepo interfaces.SCACIOperationRepository
	eventStore    interfaces.SystemEventStore
	log           logger.Logger
}

// NewErrorRecorder creates a new ErrorRecorder implementation.
func NewErrorRecorder(
	operationRepo interfaces.SCACIOperationRepository,
	eventStore interfaces.SystemEventStore,
	log logger.Logger,
) ErrorRecorder {
	return &errorRecorderImpl{
		operationRepo: operationRepo,
		eventStore:    eventStore,
		log:           log,
	}
}

// RecordOutboundError records an error sent by the Service Center to the Application Center.
// It resolves the error token to POSIX code and message, persists to operation log,
// and emits a system event.
func (r *errorRecorderImpl) RecordOutboundError(
	ctx context.Context,
	session *Session,
	opId int64,
	command string,
	errorToken string,
	defaultCode int,
	contextDetail string,
) (posixCode int, message string, err error) {
	// Resolve error token to POSIX code and message via catalog
	errDef := GetErrorDefinition(errorToken)
	// Use catalog code if non-zero, otherwise fallback to caller's defaultCode
	// This ensures wire never emits code=0 per SCACI §3.14.1
	posixCode = errDef.POSIXCode
	if posixCode == 0 {
		posixCode = defaultCode
	}
	message = errDef.Message

	// Persist to operation log with error columns
	if r.operationRepo != nil && session.ID > 0 {
		responseData := map[string]interface{}{
			MetadataKeyErrorToken:  errorToken,
			MetadataKeyErrorDetail: contextDetail,
		}
		updateErr := r.operationRepo.UpdateOperationStateWithError(
			ctx,
			session.ID,
			opId,
			models.OperationStateFailed,
			posixCode,
			errorToken,
			message,
			responseData,
		)
		if updateErr != nil {
			r.log.WarnContext(ctx, LogSCACIPersistOutboundErrorFailed,
				"opId", opId,
				"errorToken", errorToken,
				"error", updateErr)
		}
	}

	// Emit system event
	if r.eventStore != nil && session.TenantID > 0 {
		eventErr := r.eventStore.RecordSCACIError(
			ctx,
			session.TenantID,
			session.ID,
			command,
			opId,
			posixCode,
			message,
		)
		if eventErr != nil {
			r.log.WarnContext(ctx, LogSCACIRecordEventFailed,
				"opId", opId,
				"error", eventErr)
		}
	}

	return posixCode, message, nil
}

// RecordInboundError records an error received from the Application Center.
// This happens when AC cannot process an SC-initiated operation.
func (r *errorRecorderImpl) RecordInboundError(
	ctx context.Context,
	session *Session,
	opId int64,
	posixCode int,
	message string,
) error {
	// Persist to operation log - mark operation as failed with AC-provided error
	if r.operationRepo != nil && session.ID > 0 {
		responseData := map[string]interface{}{
			"inboundError":   true,
			"acPosixCode":    posixCode,
			"acErrorMessage": message,
		}
		// Use "inbound_error" as token for AC-originated errors
		updateErr := r.operationRepo.UpdateOperationStateWithError(
			ctx,
			session.ID,
			opId,
			models.OperationStateFailed,
			posixCode,
			"inbound_error", // Internal token for AC errors
			message,
			responseData,
		)
		if updateErr != nil {
			r.log.WarnContext(ctx, LogSCACIPersistInboundErrorFailed,
				"opId", opId,
				"error", updateErr)
			return updateErr
		}
	}

	// Emit system event for inbound error
	if r.eventStore != nil && session.TenantID > 0 {
		eventErr := r.eventStore.RecordSCACIError(
			ctx,
			session.TenantID,
			session.ID,
			"error", // command is "error" for inbound error messages
			opId,
			posixCode,
			message,
		)
		if eventErr != nil {
			r.log.WarnContext(ctx, LogSCACIRecordEventFailed,
				"opId", opId,
				"error", eventErr)
		}
	}

	return nil
}

// CompleteErrorHandshake marks an error operation as completed after receiving errorAck.
// This is called when we receive errorAck from AC, completing the error handshake.
// The operation state remains 'failed' but completed_at is set and errorAckReceived=true
// is added to response_data. This preserves error evidence (error_code, error_token).
func (r *errorRecorderImpl) CompleteErrorHandshake(
	ctx context.Context,
	session *Session,
	opId int64,
) error {
	if r.operationRepo == nil || session.ID <= 0 {
		return nil
	}

	// Mark as completed (set completed_at) while keeping state='failed'
	// This preserves error_code/error_token evidence per SCACI §3.14
	completionMeta := map[string]interface{}{
		"errorAckReceived": true,
	}
	return r.operationRepo.CompleteFailedOperation(
		ctx,
		session.ID,
		opId,
		completionMeta,
	)
}
