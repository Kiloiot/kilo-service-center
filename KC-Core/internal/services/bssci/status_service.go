package bssciservices

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"sync"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
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
// Real map from server.go:135, DB table: bssci_pending_operations
func (s *statusService) RecordPendingOperation(ctx context.Context, session *bssci.Session, opId int64, op *bssci.PendingOperation, dbSessionID int64) error {
	// 1. Update in-memory cache using SessionOpKey composite key
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}
	s.mu.Lock()
	(*s.pendingOps)[key] = op
	s.mu.Unlock()

	// 2. Convert in-memory types to repository types
	// Marshal Message map → operation_data JSON
	operationData, err := json.Marshal(op.Message)
	if err != nil {
		s.logger.Error("Failed to marshal pending operation",
			"error", err,
			"opId", opId)
		return err
	}

	// Marshal Metadata map → metadata JSON (nullable)
	var metadataJSON json.RawMessage
	if op.Metadata != nil {
		metadataBytes, err := json.Marshal(op.Metadata)
		if err != nil {
			s.logger.Error("Failed to marshal pending operation metadata",
				"error", err,
				"opId", opId)
			return err
		}
		metadataJSON = json.RawMessage(metadataBytes)
	}

	// 3. Persist via repository - USE CALLER'S CONTEXT
	return s.repo.Create(ctx, &interfaces.PendingOperationRequest{
		SessionID:     dbSessionID, // Use DB session ID (not in-memory session.ID)
		OperationID:   opId,
		OperationType: op.OperationType,
		EndpointEUI:   op.Endpoint,
		OperationData: json.RawMessage(operationData),
		Metadata:      metadataJSON, // nil → SQL NULL
	})
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

// RemovePendingOperation cleans map + DB using SessionOpKey composite key
func (s *statusService) RemovePendingOperation(ctx context.Context, session *bssci.Session, opId int64) error {
	// 1. Remove from cache using SessionOpKey
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}
	s.mu.Lock()
	delete(*s.pendingOps, key)
	s.mu.Unlock()

	// 2. Remove from DB via repository
	return s.repo.DeleteBySessionAndOperation(ctx, session.DbSessionID, opId)
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

// CleanupPendingOp removes pending operation from in-memory map only using SessionOpKey.
// Uses SessionOpKey composite key for multi-session support (BSSCI §5.11-5.12.3).
//
// This method provides controlled access to the pendingOps map for cleanup operations
// where DB persistence is not required or has already been handled separately.
//
// For complete cleanup (map + DB), use RemovePendingOperation instead.
//
// Parameters:
//   - session: Session containing SessionID for composite key construction
//   - opId: Operation ID to remove from map
//
// Thread Safety: Method handles mutex locking internally
func (s *statusService) CleanupPendingOp(session *bssci.Session, opId int64) {
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}

	s.mu.Lock()
	delete(*s.pendingOps, key)
	s.mu.Unlock()
}
