package bssciservices

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
)

type statusService struct {
	pendingOps *map[bssci.SessionOpKey]*bssci.PendingOperation // INJECTED: shared map created in cmd/kilocenter/main.go
	mu         *sync.RWMutex                                   // INJECTED: shared mutex from main wiring
	repo       interfaces.PendingOperationRepository           // Repository for DB persistence
	logger     logger.Logger
}

// NewStatusService creates a new status service with injected references to the shared map and mutex.
//
// Parameters:
//   - pendingOps: Pointer to shared pendingOps map (SessionOpKey composite key)
//   - mu: Pointer to shared mutex for thread-safe map access
//   - repo: PendingOperationRepository for database persistence
//   - logger: Logger instance for event tracking
func NewStatusService(pendingOps *map[bssci.SessionOpKey]*bssci.PendingOperation, mu *sync.RWMutex, repo interfaces.PendingOperationRepository, log logger.Logger) bssci.StatusService {
	return &statusService{
		pendingOps: pendingOps,
		mu:         mu,
		repo:       repo,
		logger:     log,
	}
}

// RecordPendingOperation stores operation in map + DB using SessionOpKey composite key
// Persists to DB table bssci_pending_operations and mirrors into the shared cache
func (s *statusService) RecordPendingOperation(ctx context.Context, session *bssci.Session, opId int64, op *bssci.PendingOperation, dbSessionID int64) error {
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}

	// The database is authoritative: persist first, then mirror into the cache.
	// If the durable write fails the operation is not recorded anywhere, so a
	// caller that treats a persistence failure as fatal cannot send an
	// operation whose recovery record does not exist.
	operationData, err := json.Marshal(op.Message)
	if err != nil {
		s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToMarshalPendingOperation,
			"error", err,
			"opId", opId)
		return err
	}

	var metadataJSON json.RawMessage
	if op.Metadata != nil {
		metadataBytes, err := json.Marshal(op.Metadata)
		if err != nil {
			s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToMarshalPendingOperationMetadata,
				"error", err,
				"opId", opId)
			return err
		}
		metadataJSON = json.RawMessage(metadataBytes)
	}

	if err := s.repo.Create(ctx, &interfaces.PendingOperationRequest{
		SessionID:     dbSessionID, // Use DB session ID (not in-memory session.ID)
		OperationID:   opId,
		OperationType: op.OperationType,
		EndpointEUI:   op.Endpoint,
		OperationData: json.RawMessage(operationData),
		Metadata:      metadataJSON, // nil → SQL NULL
	}); err != nil {
		return err
	}

	s.mu.Lock()
	(*s.pendingOps)[key] = op
	s.mu.Unlock()
	return nil
}

// RecordPendingOperations durably records several operations in one repository
// transaction and mirrors them into the cache only after the transaction
// commits, so a multi-frame sequence never has partially persisted recovery
// state.
func (s *statusService) RecordPendingOperations(ctx context.Context, session *bssci.Session, ops []*bssci.PendingOperation, dbSessionID int64) error {
	reqs := make([]*interfaces.PendingOperationRequest, 0, len(ops))
	for _, op := range ops {
		operationData, err := json.Marshal(op.Message)
		if err != nil {
			s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToMarshalPendingOperation,
				"error", err,
				"opId", op.OperationID)
			return err
		}

		var metadataJSON json.RawMessage
		if op.Metadata != nil {
			metadataBytes, err := json.Marshal(op.Metadata)
			if err != nil {
				s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToMarshalPendingOperationMetadata,
					"error", err,
					"opId", op.OperationID)
				return err
			}
			metadataJSON = json.RawMessage(metadataBytes)
		}

		reqs = append(reqs, &interfaces.PendingOperationRequest{
			SessionID:     dbSessionID,
			OperationID:   op.OperationID,
			OperationType: op.OperationType,
			EndpointEUI:   op.Endpoint,
			OperationData: json.RawMessage(operationData),
			Metadata:      metadataJSON,
		})
	}

	if err := s.repo.CreateBatch(ctx, reqs); err != nil {
		return err
	}

	s.mu.Lock()
	for _, op := range ops {
		key := bssci.SessionOpKey{
			SessionID:   session.ID,
			OperationID: op.OperationID,
		}
		(*s.pendingOps)[key] = op
	}
	s.mu.Unlock()
	return nil
}

// RestorePendingOperation hydrates the cache from an already authoritative DB
// row without writing it back (session resume path).
func (s *statusService) RestorePendingOperation(session *bssci.Session, opId int64, op *bssci.PendingOperation) {
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}
	s.mu.Lock()
	(*s.pendingOps)[key] = op
	s.mu.Unlock()
}

// GetPendingOperation retrieves operation from REAL map using SessionOpKey composite key
func (s *statusService) GetPendingOperation(session *bssci.Session, opId int64) (*bssci.PendingOperation, error) {
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	op, exists := (*s.pendingOps)[key]
	if !exists {
		return nil, bssci.NewCatalogError(bssci.ErrOperationNotFound, bssci.POSIX_ENOENT)
	}

	return op, nil
}

// RemovePendingOperation cleans DB + map using SessionOpKey composite key.
// The database is removed first: if the durable delete fails, the cache entry
// (and its live retry state) is preserved so the operation is not lost.
func (s *statusService) RemovePendingOperation(ctx context.Context, session *bssci.Session, opId int64) error {
	if err := s.repo.DeleteBySessionAndOperation(ctx, session.DbSessionID, opId); err != nil {
		return err
	}

	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}
	s.mu.Lock()
	delete(*s.pendingOps, key)
	s.mu.Unlock()
	return nil
}

// ExtractQueueMetadata extracts endpoint EUI, queue ID, and tenant ID from pending operation metadata.
// Returns tenantID for proper roaming tenant isolation (BSSCI §5.12).
// Uses SessionOpKey composite key for multi-session support (BSSCI §5.11-5.12.3).
//
// This helper centralizes the logic for retrieving downlink queue metadata from the
// pendingOps map. Used by downlink handlers to correlate responses with original requests.
//
// Parameters:
//   - session: Session containing SessionID for composite key construction
//   - opId: Operation ID to look up in pendingOps map
//
// Returns:
//   - endpointEUI: Endpoint EUI extracted from pending operation (0 if not found)
//   - queueID: Queue ID extracted from metadata (0 if not found)
//   - tenantID: Tenant ID string extracted from metadata (empty string if not found)
//
// Thread Safety: Method handles mutex locking internally
// UpdatePendingOperationMetadata persists new metadata for an existing pending
// row and mirrors it into the cache only after the DB write succeeds.
func (s *statusService) UpdatePendingOperationMetadata(ctx context.Context, session *bssci.Session, opId int64, metadata map[string]interface{}, metadataJSON json.RawMessage) error {
	if err := s.repo.UpdateMetadata(ctx, session.DbSessionID, opId, metadataJSON); err != nil {
		s.logger.WarnContext(ctx, bssci.LogBSSCIFailedToUpdatePendingOperationMetadata,
			"error", err,
			"sessionID", session.DbSessionID,
			"opId", opId)
		return err
	}

	key := bssci.SessionOpKey{SessionID: session.ID, OperationID: opId}
	s.mu.Lock()
	if pendingOp, ok := (*s.pendingOps)[key]; ok {
		pendingOp.Metadata = metadata
	}
	s.mu.Unlock()
	return nil
}

// PersistedOperations returns the raw persisted rows for resume hydration.
func (s *statusService) PersistedOperations(ctx context.Context, sessionID int64) ([]bssci.PersistedOperation, error) {
	rows, err := s.repo.GetBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	ops := make([]bssci.PersistedOperation, 0, len(rows))
	for _, row := range rows {
		ops = append(ops, bssci.PersistedOperation{
			OperationID:   row.OperationID,
			OperationType: row.OperationType,
			EndpointEUI:   row.EndpointEUI,
			OperationData: row.OperationData,
			Metadata:      row.Metadata,
			CreatedAt:     row.CreatedAt,
		})
	}
	return ops, nil
}

// DeletePendingOperations removes the session's persisted rows and, only on
// success, evicts its cached operations (keyed by the runtime session ID).
func (s *statusService) DeletePendingOperations(ctx context.Context, session *bssci.Session) (int64, error) {
	count, err := s.repo.DeleteBySession(ctx, session.DbSessionID)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	for key := range *s.pendingOps {
		if key.SessionID == session.ID {
			delete(*s.pendingOps, key)
		}
	}
	s.mu.Unlock()
	return count, nil
}

func (s *statusService) ExtractQueueMetadata(session *bssci.Session, opId int64) (endpointEUI uint64, queueID int64, tenantID string) {
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	pendingOp, exists := (*s.pendingOps)[key]
	if !exists {
		return 0, 0, ""
	}

	// Extract endpoint EUI from pending operation
	if len(pendingOp.Endpoint) == 8 {
		endpointEUI = binary.BigEndian.Uint64(pendingOp.Endpoint)
	}

	// Extract queue ID and tenant ID from metadata
	if pendingOp.Metadata != nil {
		// Extract queue ID (handle both int64 and float64 types from MessagePack)
		if qid, ok := pendingOp.Metadata["queId"].(int64); ok {
			queueID = qid
		} else if qid, ok := pendingOp.Metadata["queId"].(float64); ok {
			queueID = int64(qid)
		}

		// Extract tenant ID (stored as string at downlink_handlers.go:398)
		if tid, ok := pendingOp.Metadata["tenantID"].(string); ok {
			tenantID = tid
		}
	}

	return endpointEUI, queueID, tenantID
}
