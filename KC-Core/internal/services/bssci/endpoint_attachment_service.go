// Package bssciservices provides BSSCI-related services for KC-Core.
package bssciservices

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// AttachPropagateSender defines the interface for sending attach propagation.
type AttachPropagateSender interface {
	// SendAttachPropagateToAll sends attach to all connected base stations
	// Signature matches pkg/bssci/server.go:SendAttachPropagateToAll
	SendAttachPropagateToAll(endpointEUI uint64, nwkSnKey []byte, shortAddr uint16,
		bidirectional bool, lastPacketCnt uint32, dualChannel bool,
		repetition uint8, wideCarrOff bool, longBlkDist bool) []error

	// SendDetachPropagateToAll sends detach to all connected base stations
	SendDetachPropagateToAll(endpointEUI uint64) []error

	// GetConnectedSessionEUIs returns EUIs of connected base stations for event logging
	GetConnectedSessionEUIs() []string
}

// endpointAttachmentService implements endpoint lifecycle with EUI-based lookup.
type endpointAttachmentService struct {
	endpointRepo interfaces.EndpointRepository
	eventStore   interfaces.SystemEventStore
	bssciServer  AttachPropagateSender
	logger       logger.Logger
}

// NewEndpointAttachmentService creates a new endpoint attachment service.
func NewEndpointAttachmentService(
	endpointRepo interfaces.EndpointRepository,
	eventStore interfaces.SystemEventStore,
	bssciServer AttachPropagateSender,
	logger logger.Logger,
) grpcservices.EndpointAttachmentService {
	return &endpointAttachmentService{
		endpointRepo: endpointRepo,
		eventStore:   eventStore,
		bssciServer:  bssciServer,
		logger:       logger,
	}
}

// AttachEndPoint initiates attach propagation to all connected base stations.
func (s *endpointAttachmentService) AttachEndPoint(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
	// Parse EUI string to uint64 using existing validation
	euiU64, err := validation.ParseEUI(epEui)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint EUI: %w", err)
	}

	// Convert uint64 to models.EUI (8 bytes, big-endian)
	eui := models.EUI(mioty.EUI64(euiU64).ToBytes())

	// Get endpoint using repository
	endpoint, err := s.endpointRepo.GetByEUI(ctx, tenantID, eui[:])
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrEndpointNotFound
		}
		return nil, fmt.Errorf("endpoint attachment service: get endpoint: %w", err)
	}

	// Validate network key exists
	if len(endpoint.NwkSnKey) == 0 {
		return nil, ErrMissingNetworkKey
	}

	// Validate network key length per BSSCI section 5.8.1 (nwkSnKey must be Numeric[16])
	if len(endpoint.NwkSnKey) != 16 {
		return nil, ErrInvalidNetworkKeyLength
	}

	// Reject zero-filled network key (indicates missing/invalid provisioning)
	zeroKey := make([]byte, 16)
	if bytes.Equal(endpoint.NwkSnKey, zeroKey) {
		return nil, fmt.Errorf("%w: key is zero-filled", ErrMissingNetworkKey)
	}

	// Handle ShAddr: nil defaults to 0
	var shortAddr uint16
	if endpoint.ShAddr != nil {
		shortAddr = *endpoint.ShAddr
	}

	// Convert EUI to uint64
	endpointEUI := endpoint.EUI.ToUint64()

	// Get connected base stations for event logging
	connectedBSList := s.bssciServer.GetConnectedSessionEUIs()

	// Determine packet counter (0 for first attach, lastPacketCnt for subsequent)
	packetCounter := endpoint.LastPacketCnt

	// Generate operation ID
	operationID := fmt.Sprintf(bssci.OperationIDFormat, bssci.OperationIDPrefixAttach, endpoint.ID, time.Now().UnixNano())

	// Log start event for observability
	s.logStartEvent(ctx, operationID, bssci.OperationTypeAttach, endpoint.ID, tenantID, endpointEUI, connectedBSList)

	// Spawn async goroutine (fire-and-forget)
	go s.performAttachAsync(ctx, operationID, endpoint.ID, tenantID, endpoint, shortAddr, packetCounter, connectedBSList)

	return &grpcservices.EndpointOperationResult{
		OperationID: operationID,
		Status:      bssci.OperationStatusInitiated,
	}, nil
}

// DetachEndPoint initiates detach propagation to all connected base stations.
func (s *endpointAttachmentService) DetachEndPoint(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
	// Parse EUI string to uint64 using existing validation
	euiU64, err := validation.ParseEUI(epEui)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint EUI: %w", err)
	}

	// Convert uint64 to models.EUI (8 bytes, big-endian)
	eui := models.EUI(mioty.EUI64(euiU64).ToBytes())

	// Get endpoint using repository
	endpoint, err := s.endpointRepo.GetByEUI(ctx, tenantID, eui[:])
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrEndpointNotFound
		}
		return nil, fmt.Errorf("endpoint attachment service: get endpoint: %w", err)
	}

	// Convert EUI
	endpointEUI := endpoint.EUI.ToUint64()

	// Get connected base stations for event logging
	connectedBSList := s.bssciServer.GetConnectedSessionEUIs()

	// Generate operation ID
	operationID := fmt.Sprintf(bssci.OperationIDFormat, bssci.OperationIDPrefixDetach, endpoint.ID, time.Now().UnixNano())

	// Log start event for observability
	s.logStartEvent(ctx, operationID, bssci.OperationTypeDetach, endpoint.ID, tenantID, endpointEUI, connectedBSList)

	// Spawn async goroutine
	go s.performDetachAsync(ctx, operationID, endpoint.ID, tenantID, endpointEUI, connectedBSList)

	return &grpcservices.EndpointOperationResult{
		OperationID: operationID,
		Status:      bssci.OperationStatusInitiated,
	}, nil
}

// performAttachAsync executes attach propagation in background goroutine
func (s *endpointAttachmentService) performAttachAsync(
	ctx context.Context,
	operationID string,
	endpointID int64,
	tenantID int64,
	endpoint *models.EndPoint,
	shortAddr uint16,
	packetCounter uint32,
	connectedBSList []string,
) {
	// Extend context with timeout (parent context may be cancelled when handler returns)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Defensive check: Network key length should have been validated synchronously
	if len(endpoint.NwkSnKey) != 16 {
		s.logger.ErrorContext(ctx, "invalid network key bypassed synchronous validation",
			"length", len(endpoint.NwkSnKey),
			"epEui", fmt.Sprintf("%016x", endpoint.EUI.ToUint64()),
			"operationID", operationID)
		s.logFailureEvent(ctx, operationID, endpointID, tenantID, endpoint.EUI.ToUint64(),
			connectedBSList,
			fmt.Errorf("invalid network key length %d (expected 16)", len(endpoint.NwkSnKey)),
			bssci.EventTypeAttachPropagateFailed)
		return
	}

	// Convert repetition bool to uint8 (BSSCI server expects uint8)
	var repetition uint8
	if endpoint.Repetition {
		repetition = 1
	}

	// Call BSSCI server to send attach propagate to all connected base stations
	errs := s.bssciServer.SendAttachPropagateToAll(
		endpoint.EUI.ToUint64(),
		endpoint.NwkSnKey,
		shortAddr,
		endpoint.Bidi,        // bidirectional
		packetCounter,        // lastPacketCnt
		endpoint.DualChan,    // dualChannel
		repetition,           // repetition (converted to uint8)
		endpoint.WideCarrOff, // wideCarrOff
		endpoint.LongBlkDist, // longBlkDist
	)
	if len(errs) > 0 {
		// Log first error
		s.logFailureEvent(ctx, operationID, endpointID, tenantID, endpoint.EUI.ToUint64(),
			connectedBSList, errs[0], bssci.EventTypeAttachPropagateFailed)
	}
}

// performDetachAsync executes detach propagation in background goroutine
func (s *endpointAttachmentService) performDetachAsync(
	ctx context.Context,
	operationID string,
	endpointID int64,
	tenantID int64,
	epEui uint64,
	connectedBSList []string,
) {
	// Extend context with timeout (parent context may be cancelled when handler returns)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Call BSSCI server to send detach propagate to all connected base stations
	errs := s.bssciServer.SendDetachPropagateToAll(epEui)
	if len(errs) > 0 {
		// Log first error (combined errors would be noisy)
		s.logFailureEvent(ctx, operationID, endpointID, tenantID, epEui,
			connectedBSList, errs[0], bssci.EventTypeDetachPropagateFailed)
	}
}

// logFailureEvent logs operation failure event
func (s *endpointAttachmentService) logFailureEvent(
	ctx context.Context,
	operationID string,
	endpointID int64,
	tenantID int64,
	epEui uint64,
	connectedBSList []string,
	err error,
	eventType string,
) {
	targetBS := describeTargetBaseStations(connectedBSList)

	eventData := map[string]interface{}{
		bssci.EventKeyOperationID:   operationID,
		bssci.EventKeyEndpointID:    endpointID,
		bssci.EventKeyEpEui:         fmt.Sprintf("%016x", epEui),
		bssci.EventKeyTargetBS:      targetBS,
		bssci.EventKeyTargetBSList:  connectedBSList,
		bssci.EventKeyTargetBSCount: len(connectedBSList),
		bssci.EventKeyError:         err.Error(),
	}

	// Set bsEui for UI compatibility
	if len(connectedBSList) > 0 {
		eventData[bssci.EventKeyBsEui] = connectedBSList[0]
		eventData[bssci.EventKeyBaseStationEui] = connectedBSList[0]
	} else {
		eventData[bssci.EventKeyBsEui] = bssci.BaseStationNone
		eventData[bssci.EventKeyBaseStationEui] = bssci.BaseStationNone
	}

	detailsJSON, _ := json.Marshal(eventData)

	now := time.Now()
	event := &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", tenantID),
		EventType:   eventType,
		Category:    bssci.EventCategoryEndpoint,
		Severity:    bssci.SeverityError,
		Title:       bssci.TitleOperationFailed,
		Description: fmt.Sprintf(bssci.DescriptionOperationFailedFormat, targetBS, err),
		Details:     detailsJSON,
		Status:      bssci.EventStatusNew,
		SourceType:  bssci.EventSourceTypeEndpoint,
		SourceName:  fmt.Sprintf("%016x", epEui),
		EndpointID:  &endpointID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if createErr := s.eventStore.CreateEvent(ctx, event); createErr != nil {
		s.logger.ErrorContext(ctx, "failed to log failure event", "error", createErr)
	}
}

// logStartEvent creates a start event for attach/detach operations
func (s *endpointAttachmentService) logStartEvent(
	ctx context.Context,
	operationID string,
	operationType string,
	endpointID int64,
	tenantID int64,
	epEui uint64,
	connectedBSList []string,
) {
	targetBS := describeTargetBaseStations(connectedBSList)

	eventData := map[string]interface{}{
		bssci.EventKeyOperationID:   operationID,
		bssci.EventKeyOperationType: operationType,
		bssci.EventKeyEndpointID:    endpointID,
		bssci.EventKeyEpEui:         fmt.Sprintf("%016x", epEui),
		bssci.EventKeyTargetBS:      targetBS,
		bssci.EventKeyTargetBSList:  connectedBSList,
		bssci.EventKeyTargetBSCount: len(connectedBSList),
	}

	// Set bsEui for UI compatibility
	if len(connectedBSList) > 0 {
		eventData[bssci.EventKeyBsEui] = connectedBSList[0]
		eventData[bssci.EventKeyBaseStationEui] = connectedBSList[0]
	} else {
		eventData[bssci.EventKeyBsEui] = bssci.BaseStationNone
		eventData[bssci.EventKeyBaseStationEui] = bssci.BaseStationNone
	}

	detailsJSON, _ := json.Marshal(eventData)

	// Select event type, title, and description based on operation type
	var eventType, title, description string
	switch operationType {
	case bssci.OperationTypeAttach:
		eventType = bssci.EventTypeAttachPropagateInitiated
		title = bssci.TitleAttachPropagateInitiated
		description = bssci.DescriptionAttachPropagateEndpoint
	case bssci.OperationTypeDetach:
		eventType = bssci.EventTypeDetachPropagateInitiated
		title = bssci.TitleDetachPropagateInitiated
		description = bssci.DescriptionDetachPropagateEndpoint
	default:
		s.logger.ErrorContext(ctx, bssci.LogBSSCIUnsupportedOperationTypeStartEvent, "operationType", operationType)
		return
	}

	now := time.Now()
	event := &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", tenantID),
		EventType:   eventType,
		Category:    bssci.EventCategoryEndpoint,
		Severity:    bssci.SeverityInfo,
		Title:       title,
		Description: description,
		Details:     detailsJSON,
		Status:      bssci.EventStatusNew,
		SourceType:  bssci.EventSourceTypeEndpoint,
		SourceName:  fmt.Sprintf("%016x", epEui),
		EndpointID:  &endpointID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.eventStore.CreateEvent(ctx, event); err != nil {
		s.logger.WarnContext(ctx, "failed to log start event", "error", err)
	}
}

// describeTargetBaseStations generates human-readable description
func describeTargetBaseStations(list []string) string {
	switch len(list) {
	case 0:
		return "no base stations available"
	case 1:
		return fmt.Sprintf("base station %s", list[0])
	default:
		return fmt.Sprintf("%d base stations", len(list))
	}
}

// ============================================================================
// Sentinel Errors
// ============================================================================

var (
	// ErrEndpointNotFound indicates the requested endpoint does not exist
	ErrEndpointNotFound = errors.New("endpoint not found")

	// ErrMissingNetworkKey indicates endpoint has no network session key
	ErrMissingNetworkKey = errors.New("missing network session key")

	// ErrInvalidNetworkKeyLength indicates network key is not 16 bytes
	ErrInvalidNetworkKeyLength = errors.New("invalid network key length")
)

// endpointAttachmentPersistence is the transactional owner of attach and
// attach-propagate endpoint-session persistence (BSSCI rev1 §5.7-§5.8 /
// classic §3.7-§3.8).
type endpointAttachmentPersistence struct {
	storage         interfaces.Storage
	basestationRepo interfaces.BaseStationRepository
	logger          logger.Logger
}

// NewEndpointAttachmentPersistence creates the persister over the storage
// facade; the base station repository serves the out-of-transaction primary
// station lookup.
func NewEndpointAttachmentPersistence(storage interfaces.Storage, basestationRepo interfaces.BaseStationRepository, log logger.Logger) bssci.EndpointAttachmentPersistence {
	return &endpointAttachmentPersistence{
		storage:         storage,
		basestationRepo: basestationRepo,
		logger:          log,
	}
}

// PersistAttachSession runs the attach transaction: endpoint field update,
// endpoint-session upsert (with attach counter and short address), commit.
func (p *endpointAttachmentPersistence) PersistAttachSession(ctx context.Context, rec bssci.AttachSessionRecord) error {
	tx, txErr := p.storage.BeginTx(ctx)
	if txErr != nil {
		p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToBeginTransaction, "error", txErr)
		return txErr
	}

	var commitErr error
	defer func() {
		if commitErr != nil {
			_ = tx.Rollback()
		}
	}()

	if err := tx.EndPoints().UpdateFields(ctx, rec.TenantID, rec.EndpointID, rec.EndpointUpdates); err != nil {
		commitErr = err
		p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateEndpointAttachMetadata, "error", err)
		return err // Preserves sentinel: handlers match repository errors via errors.Is
	}

	endpointIDStr := fmt.Sprintf("%d", rec.EndpointID)
	activeSession, getErr := tx.EndPointSessions().GetActive(ctx, endpointIDStr)
	if getErr != nil {
		commitErr = getErr
		p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToLoadEndpointSession, "error", getErr)
		return getErr // Preserves sentinel: handlers match repository errors via errors.Is
	}

	now := time.Now().UTC()

	// Lookup base station ID for session enrichment (tenant-scoped)
	var primaryBsID *int64
	if p.basestationRepo != nil {
		bs, bsErr := p.basestationRepo.GetByEUI(ctx, rec.BSLookupTenantID, rec.BaseStationEUI)
		if bsErr == nil && bs != nil {
			primaryBsID = &bs.ID
		}
	}

	// ShAddr is uint16, model ShAddr is *int32 (INTEGER 0-65535)
	shAddrInt32 := int32(rec.ShAddr)

	if activeSession != nil {
		activeSession.SessionKey = rec.EncryptedKey
		//nolint:gosec // G115: AttachCnt is handler-validated to fit 24 bits
		activeSession.AttachCnt = int32(rec.AttachCnt)
		activeSession.LastActivityAt = now
		activeSession.ShAddr = &shAddrInt32
		activeSession.PrimaryBaseStationID = primaryBsID
		if err := tx.EndPointSessions().Update(ctx, activeSession); err != nil {
			commitErr = err
			p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateEndpointSession, "error", err)
			return err // Preserves sentinel: handlers match repository errors via errors.Is
		}
	} else {
		newSession := &models.EndPointSession{
			TenantID:   rec.TenantID,
			EndPointID: rec.EndpointID,
			SessionID:  uuid.New().String(),
			SessionKey: rec.EncryptedKey,
			//nolint:gosec // G115: AttachCnt is handler-validated to fit 24 bits
			AttachCnt:            int32(rec.AttachCnt),
			Status:               "active",
			StartedAt:            now,
			LastActivityAt:       now,
			ShAddr:               &shAddrInt32,
			PrimaryBaseStationID: primaryBsID,
		}
		if err := tx.EndPointSessions().Create(ctx, newSession); err != nil {
			commitErr = err
			p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToCreateEndpointSession, "error", err)
			return err // Preserves sentinel: handlers match repository errors via errors.Is
		}
	}

	if err := tx.Commit(); err != nil {
		commitErr = err
		p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToCommitAttachTransaction, "error", err)
		return err // Preserves sentinel: handlers match repository errors via errors.Is
	}
	commitErr = nil
	return nil
}

// PersistAttachPropagateSession runs the attach-propagate transaction under
// the endpoint owner tenant; error strings propagate to the caller unchanged.
func (p *endpointAttachmentPersistence) PersistAttachPropagateSession(ctx context.Context, rec bssci.AttachPropagateSessionRecord) error {
	tx, txErr := p.storage.BeginTx(ctx)
	if txErr != nil {
		return fmt.Errorf("failed to begin attach propagate transaction: %w", txErr)
	}
	var commitErr error
	defer func() {
		if commitErr != nil {
			_ = tx.Rollback()
		}
	}()

	if err := tx.EndPoints().UpdateFields(ctx, rec.TenantID, rec.EndpointID, rec.EndpointUpdates); err != nil {
		commitErr = err
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	endpointIDStr := fmt.Sprintf("%d", rec.EndpointID)
	activeSession, getErr := tx.EndPointSessions().GetActive(ctx, endpointIDStr)
	if getErr != nil {
		commitErr = getErr
		return fmt.Errorf("failed to load endpoint session: %w", getErr)
	}

	now := time.Now().UTC()

	// Lookup base station ID for session enrichment (tenant-scoped to owner)
	var primaryBsID *int64
	if p.basestationRepo != nil {
		bs, bsErr := p.basestationRepo.GetByEUI(ctx, rec.TenantID, rec.BaseStationEUI)
		if bsErr == nil && bs != nil {
			primaryBsID = &bs.ID
		}
	}

	// ShAddr is uint16, model ShAddr is *int32 (INTEGER 0-65535)
	shAddrInt32 := int32(rec.ShAddr)

	if activeSession != nil {
		activeSession.SessionKey = rec.EncryptedKey
		activeSession.LastActivityAt = now
		activeSession.ShAddr = &shAddrInt32
		activeSession.PrimaryBaseStationID = primaryBsID
		if err := tx.EndPointSessions().Update(ctx, activeSession); err != nil {
			commitErr = err
			return fmt.Errorf("failed to update endpoint session: %w", err)
		}
	} else {
		newSession := &models.EndPointSession{
			TenantID:             rec.TenantID,
			EndPointID:           rec.EndpointID,
			SessionID:            uuid.New().String(),
			SessionKey:           rec.EncryptedKey,
			Status:               string(models.SessionStatusActive),
			UplinkMode:           "standard", // Default uplink mode per BSSCI §5.2
			StartedAt:            now,
			LastActivityAt:       now,
			ShAddr:               &shAddrInt32,
			PrimaryBaseStationID: primaryBsID,
		}
		if err := tx.EndPointSessions().Create(ctx, newSession); err != nil {
			commitErr = err
			return fmt.Errorf("failed to create endpoint session: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		commitErr = err
		return fmt.Errorf("failed to commit attach propagate transaction: %w", err)
	}
	commitErr = nil
	return nil
}
