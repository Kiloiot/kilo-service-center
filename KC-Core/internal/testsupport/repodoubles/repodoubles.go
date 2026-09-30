// Package repodoubles provides in-memory doubles for the narrow storage
// contracts the BSSCI services consume (contracts.go in
// internal/services/bssci), so both the service packages' own tests and the
// shared protocol fixture can use one implementation.
package repodoubles

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// testBaseStationName is the name every double reports for its station.
const testBaseStationName = "Test BS"

// BaseStationSessionRepo implements the services' BaseStationSessionStore
// contract with in-memory session tracking.
type BaseStationSessionRepo struct {
	NextID                int64
	Sessions              map[int64]*models.BaseStationSession
	FindErr               error // when set, FindResumableSession returns it (infra-failure tests)
	TerminateResumableErr error // when set, TerminateResumableSessions returns it (infra-failure tests)
	ActivateErr           error // when set, ActivateSessionIfResumable returns it (infra-failure tests)
	mu                    sync.Mutex
}

// NewBaseStationSessionRepo constructs the double.
func NewBaseStationSessionRepo() *BaseStationSessionRepo {
	return &BaseStationSessionRepo{
		NextID:   1,
		Sessions: make(map[int64]*models.BaseStationSession),
	}
}

// CreateSession implements the corresponding repository method for the double.
func (m *BaseStationSessionRepo) CreateSession(_ context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.NextID
	m.NextID++

	session := &models.BaseStationSession{
		ID:              id,
		BaseStationID:   req.BaseStationID,
		TenantID:        req.TenantID,
		SnBsUuid:        req.SnBsUuid,
		SnScUuid:        req.SnScUuid,
		SnBsOpId:        0,
		SnScOpId:        0,
		Status:          models.SessionStatusActive,
		ConnectionId:    req.ConnectionId,
		RemoteAddr:      req.RemoteAddr,
		CanResume:       req.CanResume,
		Encoding:        req.Encoding,        // BSSCI Section 1: persist encoding
		ProtocolVersion: req.ProtocolVersion, // BSSCI §4-4.5: persist negotiated version
		OrganizationID:  req.OrganizationID,
		StartedAt:       time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	m.Sessions[id] = session
	return session, nil
}

// GetActiveSessionByBaseStation implements the corresponding repository method for the double.
func (m *BaseStationSessionRepo) GetActiveSessionByBaseStation(_ context.Context, _ int64, baseStationID int64) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.Sessions {
		if session.BaseStationID == baseStationID && session.Status == models.SessionStatusActive {
			return session, nil
		}
	}
	return nil, nil
}

// GetSessionByScUUID nolint:revive // Method name matches interface requirement
func (m *BaseStationSessionRepo) GetSessionByScUUID(_ context.Context, tenantID int64, snScUUID [16]byte) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.Sessions {
		// Enforce tenant isolation: only return sessions matching tenant ID
		if session.TenantID == tenantID && session.SnScUuid == snScUUID {
			return session, nil
		}
	}
	return nil, nil
}

// UpdateSession implements the corresponding repository method for the double.
func (m *BaseStationSessionRepo) UpdateSession(_ context.Context, _ int64, sessionID int64, req *models.BaseStationSessionUpdateRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.Sessions[sessionID]; ok {
		if req.SnBsOpId != nil {
			session.SnBsOpId = *req.SnBsOpId
		}
		if req.SnScOpId != nil {
			session.SnScOpId = *req.SnScOpId
		}
		if req.Status != nil {
			session.Status = *req.Status
		}
		if req.LastPingAt != nil {
			session.LastPingAt = req.LastPingAt
		}
		if req.EndedAt != nil && req.ClearEndedAt {
			return ErrEndedAtConflict
		}
		if req.EndedAt != nil {
			session.EndedAt = req.EndedAt
		}
		if req.ClearEndedAt {
			session.EndedAt = nil
		}
		if req.CanResume != nil {
			session.CanResume = *req.CanResume
		}
		if req.ConnectionId != nil {
			session.ConnectionId = req.ConnectionId
		}
		if req.RemoteAddr != nil {
			session.RemoteAddr = req.RemoteAddr
		}
		if req.Encoding != nil {
			session.Encoding = *req.Encoding
		}
		if req.ProtocolVersion != nil {
			session.ProtocolVersion = req.ProtocolVersion
		}
		if req.OrganizationID != nil {
			session.OrganizationID = req.OrganizationID
		}
		session.UpdatedAt = time.Now()
	}
	return nil
}

// UpdateOperationIDs implements the corresponding repository method for the double.
func (m *BaseStationSessionRepo) UpdateOperationIDs(_ context.Context, _ int64, sessionID int64, bsOpId, scOpId int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.Sessions[sessionID]; ok {
		session.SnBsOpId = bsOpId
		session.SnScOpId = scOpId
		session.UpdatedAt = time.Now()
	}
	return nil
}

// UpdateEncoding implements the corresponding repository method for the double.
func (m *BaseStationSessionRepo) UpdateEncoding(_ context.Context, _, sessionID int64, encoding string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.Sessions[sessionID]; ok {
		session.Encoding = encoding
		session.UpdatedAt = time.Now()
	}
	return nil
}

// TerminateSession implements the corresponding repository method for the double.
func (m *BaseStationSessionRepo) TerminateSession(_ context.Context, _ int64, sessionID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.Sessions[sessionID]; ok {
		session.Status = models.SessionStatusTerminated
		session.CanResume = false
		now := time.Now()
		session.EndedAt = &now
		session.UpdatedAt = now
	}
	return nil
}

// MarkDisconnected marks a session disconnected and resumable when the
// stored connection ID still matches (mirrors the conditional production
// update; zero matches is not an error)
func (m *BaseStationSessionRepo) MarkDisconnected(_ context.Context, tenantID, sessionID int64, connectionID string, endedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.Sessions[sessionID]
	if !ok || session.TenantID != tenantID {
		return nil
	}
	if session.ConnectionId == nil || *session.ConnectionId != connectionID {
		return nil
	}
	if session.Status != models.SessionStatusActive {
		return nil
	}
	session.Status = models.SessionStatusDisconnected
	session.CanResume = true
	ended := endedAt
	session.EndedAt = &ended
	session.UpdatedAt = time.Now()
	return nil
}

// ActivateSessionIfResumable mirrors the conditional production activation: the
// row is claimed only while it is still disconnected and resumable
func (m *BaseStationSessionRepo) ActivateSessionIfResumable(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) (bool, error) {
	m.mu.Lock()
	session, ok := m.Sessions[sessionID]
	claimable := ok && session.TenantID == tenantID && session.CanResumeSession()
	activateErr := m.ActivateErr
	m.mu.Unlock()

	if activateErr != nil {
		return false, activateErr
	}

	if !claimable {
		return false, nil
	}
	if err := m.UpdateSession(ctx, tenantID, sessionID, req); err != nil {
		return false, err
	}
	return true, nil
}

// TerminateResumableSessions mirrors the production sweep: the base station's
// disconnected, resumable rows are retired and their ids returned
func (m *BaseStationSessionRepo) TerminateResumableSessions(_ context.Context, tenantID, baseStationID int64) ([]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TerminateResumableErr != nil {
		return nil, m.TerminateResumableErr
	}

	now := time.Now()
	var retiredIDs []int64
	for _, session := range m.Sessions {
		if session.TenantID != tenantID || session.BaseStationID != baseStationID {
			continue
		}
		if !session.CanResumeSession() {
			continue
		}
		session.Status = models.SessionStatusTerminated
		session.CanResume = false
		if session.EndedAt == nil {
			ended := now
			session.EndedAt = &ended
		}
		session.UpdatedAt = now
		retiredIDs = append(retiredIDs, session.ID)
	}
	return retiredIDs, nil
}

// FindResumableSession mirrors the production lookup: tenant + base station
// EUI + snBsUuid with status=disconnected and can_resume=true
func (m *BaseStationSessionRepo) FindResumableSession(_ context.Context, tenantID int64, _ []byte, snBsUUID [16]byte) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FindErr != nil {
		return nil, m.FindErr
	}

	for _, session := range m.Sessions {
		if session.TenantID != tenantID {
			continue
		}
		if session.SnBsUuid != snBsUUID {
			continue
		}
		if !session.CanResumeSession() {
			continue
		}
		copied := *session
		return &copied, nil
	}
	return nil, storage.ErrNotFound
}

// SystemEventStore implements the services' SystemEventRecorder contract with
// no-op behavior.
type SystemEventStore struct{}

// CreateEvent implements the corresponding repository method for the double.
func (m *SystemEventStore) CreateEvent(_ context.Context, _ *models.SystemEvent) error {
	return nil
}

// DownlinkLookup is a queue double that holds no downlink.
type DownlinkLookup struct{}

// GetDownlinkByQueueID reports every queue id missing.
func (DownlinkLookup) GetDownlinkByQueueID(context.Context, uint64, string) (*storage.DownlinkMessage, error) {
	return nil, storage.ErrNotFound
}

// BaseStationRepo is an in-memory double for the services' BaseStationStore
// contract.
type BaseStationRepo struct{}

// GetByEUI implements the corresponding repository method for the double.
func (m *BaseStationRepo) GetByEUI(_ context.Context, tenantID int64, euiBytes []byte) (*models.BaseStation, error) {
	// Convert []byte to models.EUI [8]byte
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}

	return &models.BaseStation{
		ID:       1,
		TenantID: tenantID,
		EUI:      eui,
		Name:     testBaseStationName,
	}, nil
}

// GetByEUIGlobal implements the corresponding repository method for the double.
func (m *BaseStationRepo) GetByEUIGlobal(_ context.Context, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}
	return &models.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: testBaseStationName}, nil
}

// UpdateTLSFingerprintIfBlank implements the corresponding repository method for the double.
func (m *BaseStationRepo) UpdateTLSFingerprintIfBlank(_ context.Context, _, _ int64, _ string) (bool, error) {
	return true, nil
}

// UpdateTLSCertExpiryIfBlank implements the corresponding repository method for the double.
func (m *BaseStationRepo) UpdateTLSCertExpiryIfBlank(_ context.Context, _, _ int64, _ time.Time) (bool, error) {
	return true, nil
}

// PendingOperationRepository implements the services' PendingOperationStore
// contract with no-op behavior.
type PendingOperationRepository struct{}

// Create implements the corresponding repository method for the double.
func (m *PendingOperationRepository) Create(_ context.Context, _ *models.PendingOperationRequest) error {
	return nil
}

// CreateBatch implements the corresponding repository method for the double.
func (m *PendingOperationRepository) CreateBatch(_ context.Context, _ []*models.PendingOperationRequest) error {
	return nil
}

// UpdateMetadata implements the corresponding repository method for the double.
func (m *PendingOperationRepository) UpdateMetadata(_ context.Context, _ int64, _ int64, _ json.RawMessage) error {
	return nil
}

// DeleteBySessionAndOperation implements the corresponding repository method for the double.
func (m *PendingOperationRepository) DeleteBySessionAndOperation(_ context.Context, _ int64, _ int64) error {
	return nil
}

// DeleteBySession implements the corresponding repository method for the double.
func (m *PendingOperationRepository) DeleteBySession(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

// GetBySession implements the corresponding repository method for the double.
func (m *PendingOperationRepository) GetBySession(_ context.Context, _ int64) ([]*models.PendingOperation, error) {
	return []*models.PendingOperation{}, nil
}

// Storage satisfies the TestStore accessor set the BSSCI test server fans out
// into narrow views, and the services' DownlinkQueueWriter contract. The
// accessors all return nil: the test server keeps whatever store view a test
// wired directly, and the flows these fixtures exercise never touch the rest.
type Storage struct{}

// NewStorage constructs the double.
func NewStorage() *Storage {
	return &Storage{}
}

// EndPoints implements the corresponding repository accessor for the double.
func (m *Storage) EndPoints() interfaces.EndpointRepository { return nil }

// DLRXStatus implements the corresponding repository accessor for the double.
func (m *Storage) DLRXStatus() interfaces.DLRXStatusRepository { return nil }

// MIOTYMessages implements the corresponding repository accessor for the double.
func (m *Storage) MIOTYMessages() interfaces.MIOTYMessageRepository { return nil }

// MIOTYDownlinks implements the corresponding repository accessor for the double.
func (m *Storage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }

// MIOTYBaseStationStatus implements the corresponding repository accessor for the double.
func (m *Storage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository { return nil }

// EnqueueDownlink implements the DownlinkQueueWriter contract: the input is
// returned with an ID assigned, as the production queue would.
func (m *Storage) EnqueueDownlink(_ context.Context, downlink *storage.DownlinkMessage, _ time.Duration) (*storage.DownlinkMessage, error) {
	result := *downlink
	result.ID = 1
	result.QueID = 100
	return &result, nil
}

// UpdateDownlinkStatus implements the DownlinkQueueWriter contract for the double.
func (m *Storage) UpdateDownlinkStatus(_ context.Context, _ string, _ mioty.DLQueueStatus, _ *uuid.UUID) error {
	return nil
}

// UpdateDownlinkResult implements the DownlinkQueueWriter contract for the double.
func (m *Storage) UpdateDownlinkResult(_ context.Context, tenantID int64, _ uint64, result *mioty.DLDataResult) (*storage.DownlinkMessage, error) {
	if result.QueId > math.MaxInt64 {
		return nil, storage.ErrDownlinkNotFound
	}
	return reportedDownlink(tenantID, int64(result.QueId), result.EpEui), nil
}

// RevokeDownlink implements the DownlinkQueueWriter contract for the double.
func (m *Storage) RevokeDownlink(context.Context, storage.DownlinkRevocation) (bool, error) {
	return true, nil
}

// FailQueuedDownlink implements the DownlinkQueueWriter contract for the double.
func (m *Storage) FailQueuedDownlink(_ context.Context, queID int64, tenantID int64, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return reportedDownlink(tenantID, queID, 0), nil
}

// UpdateDownlinkBaseStation implements the DownlinkHolderWriter contract for the double.
func (m *Storage) UpdateDownlinkBaseStation(context.Context, uint64, int64, uint64) error {
	return nil
}

// MarkReservedAsQueued implements the DownlinkHolderWriter contract for the double.
func (m *Storage) MarkReservedAsQueued(context.Context, uint64, int64, uint64, int64, *uint32, *uuid.UUID) error {
	return nil
}

// reportedDownlink is the queue row the double returns for a finished downlink.
func reportedDownlink(tenantID, queID int64, epEUI uint64) *storage.DownlinkMessage {
	return &storage.DownlinkMessage{QueID: queID, EPEUI: mioty.FormatEUI64(epEUI), TenantID: strconv.FormatInt(tenantID, 10)}
}

// TestStore is the accessor set the BSSCI test server fans out into narrow
// views. It exists so test scaffolding does not depend on a storage aggregate
// that production code no longer uses.
type TestStore interface {
	EndPoints() interfaces.EndpointRepository
	MIOTYMessages() interfaces.MIOTYMessageRepository
	DLRXStatus() interfaces.DLRXStatusRepository
	MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository
	MIOTYDownlinks() interfaces.MIOTYDownlinkRepository
}
