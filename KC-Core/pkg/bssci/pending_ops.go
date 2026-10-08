package bssci

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// pendingOperationJournal records the service-center operations a session has
// in flight so a resumed session can reissue them (BSSCI §3.3 / §5.2).
type pendingOperationJournal struct {
	status   StatusService
	sessions SessionService
	clock    clock.Clock
	log      logger.Logger
}

func newPendingOperationJournal(status StatusService, sessions SessionService, clk clock.Clock, log logger.Logger) *pendingOperationJournal {
	return &pendingOperationJournal{status: status, sessions: sessions, clock: clk, log: log}
}

// begin allocates the next service-center operation ID and persists the
// session counters that back it.
func (j *pendingOperationJournal) begin(ctx context.Context, session *Session) (int64, error) {
	opId := session.NextScOpID()
	if err := j.sessions.UpdateSessionCounters(ctx, session); err != nil {
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistSessionCounters), err)
	}
	return opId, nil
}

// persistPendingOperation stores a pending operation via StatusService (BSSCI §5.11-5.12.3 single writer)
// StatusService handles both DB persistence and in-memory map update using SessionOpKey composite key
func (j *pendingOperationJournal) persist(ctx context.Context, session *Session, opId int64, opType string, message map[string]interface{}, euiBytes []byte, metadata map[string]interface{}) error {
	// An SC operation without a persisted session has no recovery identity;
	// letting it on the wire would make it unrecoverable after a crash. This
	// is an inconsistent-session error, never a silent no-persistence mode.
	if session.DbSessionID == 0 {
		j.log.ErrorContext(ctx, LogBSSCIDatabaseNotAvailableForPendingOpPersistence,
			logger.FieldOpID, opId,
			logger.FieldOpType, opType)
		return NewCatalogError(errPendingOpSessionNotPersisted, POSIX_EPROTO)
	}

	pendingOp := j.build(opId, opType, message, euiBytes, metadata)

	// StatusService is the single path for pending operation persistence. A
	// failure is surfaced to the caller: an SC operation whose recovery record
	// was never durably written must not go on the wire.
	if err := j.status.RecordPendingOperation(ctx, session, opId, pendingOp, session.DbSessionID); err != nil {
		j.log.ErrorContext(ctx, LogBSSCIFailedToPersistPendingOperationMigrationNeeded,
			logger.FieldError, err,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldOpID, opId)
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistPendingOperation), err)
	}

	j.log.DebugContext(ctx, LogBSSCIPersistedPendingOperation,
		logger.FieldSessionID, session.DbSessionID,
		logger.FieldOpID, opId,
		logger.FieldOpType, opType)

	return nil
}

// buildPendingOperation assembles the recovery record for an SC-initiated
// operation, extracting the VM MACType and payload data from metadata when
// present.
func (j *pendingOperationJournal) build(opId int64, opType string, message map[string]interface{}, euiBytes []byte, metadata map[string]interface{}) *PendingOperation {
	var macType int
	var data []byte
	if metadata != nil {
		// Extract MACType
		if mt, ok := metadata["macType"].(int); ok {
			macType = mt
		} else if mt, ok := metadata["macType"].(uint8); ok {
			macType = int(mt)
		} else if mt, ok := metadata["macType"].(float64); ok {
			macType = int(mt)
		}

		// Extract Data (VM payload)
		if d, ok := metadata["data"].([]byte); ok {
			data = d
		} else if dataStr, ok := metadata["data"].(string); ok {
			// Handle base64-encoded data
			if decoded, err := base64.StdEncoding.DecodeString(dataStr); err == nil {
				data = decoded
			}
		} else if userDataStr, ok := metadata["userData"].(string); ok && opType == mioty.CmdULDataTransmit {
			// For mioty.CmdULDataTransmit operations, also check userData field
			if decoded, err := base64.StdEncoding.DecodeString(userDataStr); err == nil {
				data = decoded
			}
		}
	}

	return &PendingOperation{
		OperationID:   opId,
		OperationType: opType,
		Message:       message,
		Endpoint:      euiBytes,
		MACType:       macType,
		Data:          data,
		Metadata:      metadata,
		CreatedAt:     j.clock.Now(),
	}
}

// persistPendingOperationBatch durably records several recovery records in one
// repository transaction (all-or-nothing) so a multi-frame sequence such as
// the dlRxStatQry/dlDataQue pair never has a partially persisted recovery
// state. The same inconsistent-session rule as persistPendingOperation
// applies.
func (j *pendingOperationJournal) persistBatch(ctx context.Context, session *Session, ops []*PendingOperation) error {
	if session.DbSessionID == 0 {
		j.log.ErrorContext(ctx, LogBSSCIDatabaseNotAvailableForPendingOpPersistence,
			logger.FieldOpCount, len(ops))
		return NewCatalogError(errPendingOpSessionNotPersisted, POSIX_EPROTO)
	}

	if err := j.status.RecordPendingOperations(ctx, session, ops, session.DbSessionID); err != nil {
		j.log.ErrorContext(ctx, LogBSSCIFailedToPersistPendingOperationMigrationNeeded,
			logger.FieldError, err,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldOpCount, len(ops))
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistPendingOperation), err)
	}

	return nil
}

// updatePendingOperationMetadata updates the metadata of an existing pending operation (Issue #3: accepts *Session for composite key)
func (j *pendingOperationJournal) updateMetadata(ctx context.Context, session *Session, opId int64, metadata map[string]interface{}) error {
	sessionID := session.DbSessionID
	if sessionID == 0 {
		j.log.WarnContext(ctx, LogBSSCIDatabaseNotAvailableForPendingOpUpdate)
		return nil
	}

	// Convert metadata to JSON for storage
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToMarshalMeta), err)
	}

	// StatusService owns pending-operation persistence and its cache: the DB
	// write happens first, the cache mirror only on success.
	if err := j.status.UpdatePendingOperationMetadata(ctx, session, opId, metadata, json.RawMessage(metadataJSON)); err != nil {
		return err
	}

	j.log.DebugContext(ctx, LogBSSCIUpdatedPendingOperationMetadata,
		logger.FieldSessionID, sessionID,
		logger.FieldOpID, opId,
		logger.FieldMetadata, metadata)

	return nil
}

// removePendingOperation removes a completed operation from database and memory (Issue #3: accepts *Session for composite key)
// BSSCI §§5.11-5.12.3 Gap 1: Dual-path for test compatibility
func (j *pendingOperationJournal) remove(ctx context.Context, session *Session, opId int64) error {
	// StatusService is the single path for pending operation persistence
	return j.status.RemovePendingOperation(ctx, session, opId)
}

// loadPendingOperations loads pending operations from database for session resume
func (j *pendingOperationJournal) load(ctx context.Context, session *Session) ([]*PendingOperation, error) {
	sessionID := session.DbSessionID
	if sessionID == 0 || j.status == nil {
		j.log.DebugContext(ctx, LogBSSCIDatabaseNotAvailableForPendingOpsLoad)
		return nil, nil
	}

	// Retrieve the raw persisted rows through the pending-operation owner
	repoOps, err := j.status.PersistedOperations(ctx, sessionID)
	if err != nil {
		j.log.ErrorContext(ctx, LogBSSCIFailedToQueryPendingOperationsFromDatabase,
			logger.FieldError, err,
			logger.FieldSessionID, sessionID)
		return nil, err
	}

	var pendingOps []*PendingOperation
	for _, repoOp := range repoOps {
		opId := repoOp.OperationID
		opType := repoOp.OperationType
		endpointEui := repoOp.EndpointEUI
		operationDataJSON := repoOp.OperationData
		metadataJSON := repoOp.Metadata
		createdAt := repoOp.CreatedAt

		// Deserialize operation data with the strict frame decoder (UseNumber,
		// single object, trailing content rejected) so uint64 EUI values
		// survive resume exactly. A malformed persisted operation is an
		// infrastructure inconsistency: the caller rejects the resume rather
		// than silently losing protocol state.
		operationData, err := decodeJSONFrame(operationDataJSON)
		if err != nil {
			j.log.ErrorContext(ctx, LogBSSCIFailedToUnmarshalOperationData,
				logger.FieldError, err,
				logger.FieldOpID, opId)
			return nil, fmt.Errorf(errFmtTokenOpIDWrap, ResolveErrorMessage(errFailedToDecode), opId, err)
		}
		operationData = normalizeStrictDecodedMap(operationData)

		// Deserialize metadata under the same strict rules
		var metadata map[string]interface{}
		if len(metadataJSON) > 0 {
			metadata, err = decodeJSONFrame(metadataJSON)
			if err != nil {
				j.log.ErrorContext(ctx, LogBSSCIFailedToUnmarshalMetadata,
					logger.FieldError, err,
					logger.FieldOpID, opId)
				return nil, fmt.Errorf(errFmtTokenOpIDMetadata, ResolveErrorMessage(errFailedToDecode), opId, err)
			}
			metadata = normalizeStrictDecodedMap(metadata)
		} else {
			metadata = make(map[string]interface{})
		}

		// Normalize detach metadata on resume to prevent JSON round-trip type drift (BSSCI §5.7.1)
		if opType == mioty.CmdDetach && len(metadata) > 0 {
			if typedMeta := mapToDetachMetadata(metadata); typedMeta != nil {
				// Successfully reconstructed typed metadata, convert back to map with correct types
				normalizedMeta := detachMetadataToMap(typedMeta)
				// Preserve subpackets if present (not part of typed struct)
				if subpackets, ok := metadata["subpackets"]; ok {
					normalizedMeta["subpackets"] = subpackets
				}
				metadata = normalizedMeta
				j.log.DebugContext(ctx, LogBSSCINormalizedDetachMetadataOnResume,
					logger.FieldOpID, opId,
					logger.FieldEpEui, typedMeta.EpEui)
			} else {
				j.log.WarnContext(ctx, LogBSSCIFailedToNormalizeDetachMetadataOnResumeUsingRawData,
					logger.FieldOpID, opId)
			}
		}

		// Extract MAC type and data from metadata if available
		var macType int
		var data []byte
		if m, ok := metadata["macType"].(float64); ok {
			macType = int(m)
		}
		if d, ok := metadata["data"].([]byte); ok {
			data = d
		} else if dataStr, ok := metadata["data"].(string); ok {
			// Handle base64-encoded data
			if decoded, err := base64.StdEncoding.DecodeString(dataStr); err == nil {
				data = decoded
			}
		} else if userDataStr, ok := metadata["userData"].(string); ok && opType == mioty.CmdULDataTransmit {
			// For mioty.CmdULDataTransmit operations, also check userData field (legacy compatibility)
			if decoded, err := base64.StdEncoding.DecodeString(userDataStr); err == nil {
				data = decoded
				j.log.DebugContext(ctx, LogBSSCILoadedUserDataFromMetadataForULDataTx,
					logger.FieldOpID, opId,
					logger.FieldDataLen, len(data))
			}
		}

		pendingOp := &PendingOperation{
			OperationID:   opId,
			OperationType: opType,
			Message:       operationData,
			Endpoint:      endpointEui,
			MACType:       macType,
			Data:          data,
			Metadata:      metadata,
			CreatedAt:     createdAt,
		}

		pendingOps = append(pendingOps, pendingOp)
	}

	j.log.DebugContext(ctx, LogBSSCILoadedPendingOperationsFromDatabase,
		logger.FieldSessionID, sessionID,
		logger.FieldCount, len(pendingOps))

	return pendingOps, nil
}
