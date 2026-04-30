package bssciservices

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
)

// ============================================================================
// Mock Repository Implementations for Testing
// ============================================================================

// mockBaseStationRepo implements interfaces.BaseStationRepository with minimal in-memory behavior
type mockBaseStationRepo struct{}

func (m *mockBaseStationRepo) Create(_ context.Context, _ *models.BaseStation) error {
	return nil
}

func (m *mockBaseStationRepo) GetByID(_ context.Context, tenantID, id int64) (*models.BaseStation, error) {
	var eui models.EUI
	return &models.BaseStation{
		ID:       id,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *mockBaseStationRepo) GetByEUI(_ context.Context, tenantID int64, euiBytes []byte) (*models.BaseStation, error) {
	// Convert []byte to models.EUI [8]byte
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}

	return &models.BaseStation{
		ID:       1,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *mockBaseStationRepo) Update(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	return nil
}

func (m *mockBaseStationRepo) Delete(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *mockBaseStationRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	return []*models.BaseStation{}, 0, nil
}

func (m *mockBaseStationRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	return nil
}

func (m *mockBaseStationRepo) UpdateSessionInfo(_ context.Context, _ int64, _ []byte, _ string) error {
	return nil
}

func (m *mockBaseStationRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
	return &interfaces.BaseStationStatistics{}, nil
}

func (m *mockBaseStationRepo) GetPropagationState(_ context.Context, _ int64) (*models.BaseStationPropagationState, error) {
	return nil, nil // no-op for basic tests
}

func (m *mockBaseStationRepo) UpsertPropagationState(_ context.Context, _ *models.BaseStationPropagationState) error {
	return nil // no-op for basic tests
}

func (m *mockBaseStationRepo) UpdatePropagationStatus(_ context.Context, _ int64, _ string, _ *string) error {
	return nil // no-op for basic tests
}

func (m *mockBaseStationRepo) IncrementRetryCount(_ context.Context, _ int64, _ time.Time) error {
	return nil // no-op for basic tests
}

func (m *mockBaseStationRepo) UpdateEUI(_ context.Context, tenantID int64, _ []byte, newEui []byte) (*models.BaseStation, error) {
	// Return a mock base station with the new EUI
	var eui models.EUI
	if len(newEui) == 8 {
		copy(eui[:], newEui)
	}
	return &models.BaseStation{
		ID:       1,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *mockBaseStationRepo) GetByEUIGlobal(_ context.Context, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}
	return &models.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Test BS"}, nil
}

func (m *mockBaseStationRepo) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// mockBaseStationSessionRepo implements interfaces.BaseStationSessionRepository with in-memory session tracking
type mockBaseStationSessionRepo struct {
	nextID   int64
	sessions map[int64]*models.BaseStationSession
	mu       sync.Mutex
}

func newMockBaseStationSessionRepo() *mockBaseStationSessionRepo {
	return &mockBaseStationSessionRepo{
		nextID:   1,
		sessions: make(map[int64]*models.BaseStationSession),
	}
}

func (m *mockBaseStationSessionRepo) CreateSession(_ context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++

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

	m.sessions[id] = session
	return session, nil
}

func (m *mockBaseStationSessionRepo) GetSessionByID(_ context.Context, _ int64, sessionID int64) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		return session, nil
	}
	return nil, nil
}

func (m *mockBaseStationSessionRepo) GetActiveSessionByBaseStation(_ context.Context, _ int64, baseStationID int64) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		if session.BaseStationID == baseStationID && session.Status == models.SessionStatusActive {
			return session, nil
		}
	}
	return nil, nil
}

//nolint:revive // Method name matches interface requirement
func (m *mockBaseStationSessionRepo) GetSessionByBsUUID(_ context.Context, tenantID int64, snBsUUID [16]byte) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		// Enforce tenant isolation: only return sessions matching tenant ID
		if session.TenantID == tenantID && session.SnBsUuid == snBsUUID {
			return session, nil
		}
	}
	return nil, nil
}

//nolint:revive // Method name matches interface requirement
func (m *mockBaseStationSessionRepo) GetSessionByScUUID(_ context.Context, tenantID int64, snScUUID [16]byte) (*models.BaseStationSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		// Enforce tenant isolation: only return sessions matching tenant ID
		if session.TenantID == tenantID && session.SnScUuid == snScUUID {
			return session, nil
		}
	}
	return nil, nil
}

func (m *mockBaseStationSessionRepo) UpdateSession(_ context.Context, _ int64, sessionID int64, req *models.BaseStationSessionUpdateRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
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
		if req.EndedAt != nil {
			session.EndedAt = req.EndedAt
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

func (m *mockBaseStationSessionRepo) UpdateOperationIDs(_ context.Context, _ int64, sessionID int64, bsOpId, scOpId int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		session.SnBsOpId = bsOpId
		session.SnScOpId = scOpId
		session.UpdatedAt = time.Now()
	}
	return nil
}

func (m *mockBaseStationSessionRepo) UpdatePing(_ context.Context, _ int64, sessionID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		now := time.Now()
		session.LastPingAt = &now
		session.UpdatedAt = now
	}
	return nil
}

func (m *mockBaseStationSessionRepo) UpdateEncoding(_ context.Context, sessionID int64, encoding string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		session.Encoding = encoding
		session.UpdatedAt = time.Now()
	}
	return nil
}

func (m *mockBaseStationSessionRepo) TerminateSession(_ context.Context, _ int64, sessionID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		session.Status = models.SessionStatusTerminated
		now := time.Now()
		session.EndedAt = &now
		session.UpdatedAt = now
	}
	return nil
}

func (m *mockBaseStationSessionRepo) TerminateAllSessions(_ context.Context, _ int64, baseStationID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, session := range m.sessions {
		if session.BaseStationID == baseStationID && session.Status != models.SessionStatusTerminated {
			session.Status = models.SessionStatusTerminated
			session.EndedAt = &now
			session.UpdatedAt = now
		}
	}
	return nil
}

func (m *mockBaseStationSessionRepo) ListSessions(_ context.Context, _ *models.BaseStationSessionFilter) ([]*models.BaseStationSession, int64, error) {
	return []*models.BaseStationSession{}, 0, nil
}

func (m *mockBaseStationSessionRepo) GetSessionStatistics(_ context.Context, _ int64) (*interfaces.SessionStatistics, error) {
	return &interfaces.SessionStatistics{}, nil
}

//nolint:revive // Parameter name matches interface requirement
func (m *mockBaseStationSessionRepo) CheckSessionResumable(_ context.Context, _ int64, snBsUUID [16]byte, _ int64) (*interfaces.SessionResumptionInfo, error) {
	return &interfaces.SessionResumptionInfo{CanResume: false}, nil
}

func (m *mockBaseStationSessionRepo) CleanupExpiredSessions(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

// UpdateCountersAndTimestamp updates operation counters by session UUID
func (m *mockBaseStationSessionRepo) UpdateCountersAndTimestamp(_ context.Context, sessionUUID [16]byte, bsOpId, scOpId int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find session by UUID and update counters
	for _, session := range m.sessions {
		if session.SnScUuid == sessionUUID {
			session.SnBsOpId = bsOpId
			session.SnScOpId = scOpId
			session.UpdatedAt = time.Now()
			return nil
		}
	}
	return nil // Session not found - non-fatal for mock
}

// mockPendingOperationRepository implements interfaces.PendingOperationRepository for testing
type mockPendingOperationRepository struct{}

func (m *mockPendingOperationRepository) Create(_ context.Context, _ *interfaces.PendingOperationRequest) error {
	return nil
}

func (m *mockPendingOperationRepository) UpdateMetadata(_ context.Context, _ int64, _ int64, _ json.RawMessage) error {
	return nil
}

func (m *mockPendingOperationRepository) DeleteBySessionAndOperation(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *mockPendingOperationRepository) DeleteByOperation(_ context.Context, _ int64) error {
	return nil
}

func (m *mockPendingOperationRepository) DeleteBySession(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (m *mockPendingOperationRepository) GetBySession(_ context.Context, _ int64) ([]*interfaces.PendingOperation, error) {
	return []*interfaces.PendingOperation{}, nil
}

// mockSystemEventStore implements interfaces.SystemEventStore with no-op behavior
type mockSystemEventStore struct{}

func (m *mockSystemEventStore) CreateEvent(_ context.Context, _ *models.SystemEvent) error {
	return nil
}

func (m *mockSystemEventStore) GetEvents(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	return []*models.SystemEvent{}, nil
}

func (m *mockSystemEventStore) GetActiveAlerts(_ context.Context, _ interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	return []*models.SystemEvent{}, nil
}

func (m *mockSystemEventStore) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return &models.SystemEventStats{}, nil
}

func (m *mockSystemEventStore) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}

// CountEvents returns total count matching filter (for pagination)
func (m *mockSystemEventStore) CountEvents(_ context.Context, _ interfaces.SystemEventFilter) (int64, error) {
	return 0, nil
}

// CountActiveAlerts returns total count of active alerts (for pagination)
func (m *mockSystemEventStore) CountActiveAlerts(_ context.Context, _ interfaces.AlertFilter) (int64, error) {
	return 0, nil
}

// mockMIOTYMessageRepository implements interfaces.MIOTYMessageRepository for testing
type mockMIOTYMessageRepository struct{}

func (m *mockMIOTYMessageRepository) CreateULDataMessage(_ context.Context, _ *mioty.ULDataMessage) error {
	return nil
}

func (m *mockMIOTYMessageRepository) CreateDetachMessage(_ context.Context, _ *mioty.DetachMessage, _ map[string]interface{}) error {
	return nil
}

func (m *mockMIOTYMessageRepository) CreateAttachMessage(_ context.Context, _ *mioty.AttachMessage, _ map[string]interface{}) error {
	return nil
}

func (m *mockMIOTYMessageRepository) GetULDataMessage(_ context.Context, _ string, _ int64) (*mioty.ULDataMessage, error) {
	return nil, nil
}

func (m *mockMIOTYMessageRepository) GetDetachMessage(_ context.Context, _ string, _ int64) (*mioty.DetachMessage, error) {
	return nil, nil
}

func (m *mockMIOTYMessageRepository) ListULDataMessages(_ context.Context, _ mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return []*mioty.ULDataMessage{}, 0, nil
}

func (m *mockMIOTYMessageRepository) GetMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return &mioty.MessageStats{}, nil
}

func (m *mockMIOTYMessageRepository) GetExtendedMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return &mioty.MessageStats{}, nil
}

func (m *mockMIOTYMessageRepository) GetMessageStatsByEndpoint(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return &mioty.MessageStats{}, nil
}

func (m *mockMIOTYMessageRepository) GetOverallStats(_ context.Context, _ int64) (*mioty.MessageStats, error) {
	return &mioty.MessageStats{}, nil
}

func (m *mockMIOTYMessageRepository) GetAnalyticsOverview(_ context.Context, _ int64, _, _ time.Time) (*mioty.AnalyticsOverviewStats, error) {
	return &mioty.AnalyticsOverviewStats{}, nil
}

func (m *mockMIOTYMessageRepository) GetHourlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.HourlyActivity, error) {
	return []mioty.HourlyActivity{}, nil
}

func (m *mockMIOTYMessageRepository) GetDailyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.DailyActivity, error) {
	return []mioty.DailyActivity{}, nil
}

func (m *mockMIOTYMessageRepository) GetTopEndpointsByActivity(_ context.Context, _ int64, _, _ time.Time, _ int) ([]mioty.EndpointActivity, error) {
	return []mioty.EndpointActivity{}, nil
}

func (m *mockMIOTYMessageRepository) GetSignalQualityStats(_ context.Context, _ int64, _, _ time.Time) (*mioty.SignalQualityStats, error) {
	return &mioty.SignalQualityStats{}, nil
}

func (m *mockMIOTYMessageRepository) GetSignalQualityByBaseStation(_ context.Context, _ int64, _, _ time.Time) ([]mioty.BaseStationSignalQuality, error) {
	return []mioty.BaseStationSignalQuality{}, nil
}

func (m *mockMIOTYMessageRepository) CreateAttachPropagateMessage(_ context.Context, _ *mioty.AttachPropagateMessage) error {
	return nil // no-op for basic tests
}

func (m *mockMIOTYMessageRepository) CreateDetachPropagateMessage(_ context.Context, _ *mioty.DetachPropagateMessage) error {
	return nil // no-op for basic tests
}

func (m *mockMIOTYMessageRepository) UpdateULDataBaseStations(_ context.Context, _ int64, _ uint64, _ uint32, _ int64, _ []byte) error {
	return nil // no-op for tests - UPDATE on duplicate path
}

// GetBaseStationMessageStats retrieves aggregated message statistics for a base station
func (m *mockMIOTYMessageRepository) GetBaseStationMessageStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	return &mioty.BaseStationMessageStats{}, nil
}

func (m *mockMIOTYMessageRepository) GetMessageCountsByEndpoint(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (m *mockMIOTYMessageRepository) GetMessageCountsByBaseStation(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (m *mockMIOTYMessageRepository) GetWeeklyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.WeeklyActivity, error) {
	return nil, nil
}

func (m *mockMIOTYMessageRepository) GetMonthlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.MonthlyActivity, error) {
	return nil, nil
}

// mockMIOTYDownlinkRepository implements interfaces.MIOTYDownlinkRepository for testing
type mockMIOTYDownlinkRepository struct {
	enqueueResult *storage.DownlinkMessage
	enqueueErr    error
	statusErr     error
	resultErr     error
	revokeErr     error
}

func (m *mockMIOTYDownlinkRepository) EnqueueDownlink(_ context.Context, downlink *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	if m.enqueueErr != nil {
		return nil, m.enqueueErr
	}
	if m.enqueueResult != nil {
		return m.enqueueResult, nil
	}
	// Return input with ID assigned
	result := *downlink
	result.ID = 1
	result.QueID = 100
	return &result, nil
}

func (m *mockMIOTYDownlinkRepository) GetDownlinkQueue(_ context.Context, _ string, _ string) ([]*storage.DownlinkMessage, error) {
	return []*storage.DownlinkMessage{}, nil
}

func (m *mockMIOTYDownlinkRepository) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockMIOTYDownlinkRepository) GetDownlinkByPacketCnt(_ context.Context, _ string, _ string, _ uint32) (*storage.DownlinkMessage, error) {
	return nil, nil
}

// orgID parameter enables organization-scoped result queries
func (m *mockMIOTYDownlinkRepository) GetDownlinkResults(_ context.Context, _ string, _ string, _ *uuid.UUID, _ string, _, _ *time.Time, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return []*storage.DownlinkMessage{}, 0, nil
}

// orgID parameter enables organization-scoped status updates
func (m *mockMIOTYDownlinkRepository) UpdateDownlinkStatus(_ context.Context, _ string, _ string, _ *uuid.UUID) error {
	return m.statusErr
}

// orgID parameter enables organization-scoped result updates
func (m *mockMIOTYDownlinkRepository) UpdateDownlinkResult(_ context.Context, _ int64, _ string, _ *int64, _ *uint32, _ []byte, _ []byte, _ string, _ *uuid.UUID) error {
	return m.resultErr
}

func (m *mockMIOTYDownlinkRepository) UpdateDownlinkBaseStation(_ context.Context, _ uint64, _ string, _ uint64) error {
	return nil
}

func (m *mockMIOTYDownlinkRepository) RevokeDownlink(_ context.Context, _ int64, _ string) error {
	return m.revokeErr
}

// Transaction-only methods (return ErrNotImplemented in non-tx context)
// orgID parameter enables organization-scoped reservation
func (m *mockMIOTYDownlinkRepository) ReserveNextPendingDownlink(_ context.Context, _ int64, _ []byte, _ uint64, _ *uuid.UUID) (*storage.DownlinkMessage, error) {
	return nil, nil // No pending downlinks in basic mock
}

// orgID parameter enables organization-scoped queue marking
func (m *mockMIOTYDownlinkRepository) MarkReservedAsQueued(_ context.Context, _ uint64, _ int64, _ uint64, _ int64, _ *uint32, _ *uuid.UUID) error {
	return nil // No-op in basic mock
}

// mockDLRXStatusRepository implements interfaces.DLRXStatusRepository for testing
type mockDLRXStatusRepository struct{}

func (m *mockDLRXStatusRepository) CreateDLRXStatus(_ context.Context, _ *mioty.DLRXStatus) error {
	return nil
}

func (m *mockDLRXStatusRepository) GetDLRXStatusByEndpoint(_ context.Context, _ int64, _ []byte,
	_, _ int, _, _ *time.Time) ([]*mioty.DLRXStatus, int, error) {
	return nil, 0, nil
}

func (m *mockDLRXStatusRepository) GetLatestDLRXStatus(_ context.Context, _ int64, _ []byte) (*mioty.DLRXStatus, error) {
	return nil, nil
}

func (m *mockDLRXStatusRepository) GetLatestDLRXStatusPerEndpoint(_ context.Context, _ int64) ([]*mioty.DLRXStatus, error) {
	return nil, nil
}

func (m *mockDLRXStatusRepository) GetDLRXStatusByTimeRange(_ context.Context, _ int64, _, _ time.Time) ([]*mioty.DLRXStatus, error) {
	return nil, nil
}

func (m *mockDLRXStatusRepository) GetAverageDLRXMetrics(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (float64, float64, int, error) {
	return 0, 0, 0, nil
}

func (m *mockDLRXStatusRepository) DeleteOldDLRXStatus(_ context.Context, _ int64, _ int) (int64, error) {
	return 0, nil
}

func (m *mockDLRXStatusRepository) CreateDLRXStatusQuery(_ context.Context, _ int64, _ *uuid.UUID, _, _ []byte, _ int64) error {
	return nil // No-op for tests
}

func (m *mockDLRXStatusRepository) MarkDLRXStatusReceived(_ context.Context, _ int64, _ []byte, _ []byte, _ int64) (bool, error) {
	// Signature: (ctx, tenantID, epEui, bsEui, bsOpID)
	// Correlates by tenant+endpoint only; bsEui+bsOpID stored for audit
	return false, nil // No-op for tests
}

func (m *mockDLRXStatusRepository) ExpireDLRXStatusQuery(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil // No-op for tests
}

func (m *mockDLRXStatusRepository) GetDLRXStatusQueryHistory(_ context.Context, _ int64, _ []byte,
	_, _ int, _, _ *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	return nil, 0, nil // No-op for tests
}

func (m *mockDLRXStatusRepository) GetDLRXStatusQueryStats(_ context.Context, _ int64, _ []byte,
	_, _ *time.Time) (int64, int64, int64, error) {
	return 0, 0, 0, nil // No-op for tests
}

func (m *mockDLRXStatusRepository) GetLatestDLRXStatusByBaseStations(_ context.Context, _ int64, _ []byte, _ [][]byte) ([]*mioty.DLRXStatus, error) {
	return nil, nil // No-op for tests - DL RX hydration will skip if no data
}

// mockStorage implements interfaces.Storage for testing
type mockStorage struct {
	miotyMessages  *mockMIOTYMessageRepository
	miotyDownlinks *mockMIOTYDownlinkRepository
	dlRxStatus     *mockDLRXStatusRepository
	endpoints      map[int64]*models.EndPoint // For detach integration tests
	events         []*models.SystemEvent      // For event tracking in tests
	mu             sync.RWMutex               // Protects endpoints and events
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		miotyMessages:  &mockMIOTYMessageRepository{},
		miotyDownlinks: &mockMIOTYDownlinkRepository{},
		dlRxStatus:     &mockDLRXStatusRepository{},
		endpoints:      make(map[int64]*models.EndPoint),
		events:         make([]*models.SystemEvent, 0),
	}
}

// Repository accessors for interfaces.Storage
func (m *mockStorage) EndPoints() interfaces.EndpointRepository                         { return nil }
func (m *mockStorage) DownlinkQueue() interfaces.DownlinkQueueRepository                { return nil }
func (m *mockStorage) BaseStationReceptions() interfaces.BaseStationReceptionRepository { return nil }
func (m *mockStorage) EndPointSessions() interfaces.EndPointSessionRepository           { return nil }
func (m *mockStorage) EndPointKeys() interfaces.EndPointKeyRepository                   { return nil }
func (m *mockStorage) RoamingAgreements() interfaces.RoamingAgreementRepository         { return nil }
func (m *mockStorage) BaseStations() interfaces.BaseStationRepository                   { return nil }
func (m *mockStorage) BaseStationSessions() interfaces.BaseStationSessionRepository     { return nil }
func (m *mockStorage) DLRXStatus() interfaces.DLRXStatusRepository                      { return m.dlRxStatus }
func (m *mockStorage) PendingOperations() interfaces.PendingOperationRepository         { return nil }

// MIOTY-specific repositories (implemented for test support)
func (m *mockStorage) MIOTYMessages() interfaces.MIOTYMessageRepository {
	return m.miotyMessages
}

func (m *mockStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository {
	return m.miotyDownlinks
}

func (m *mockStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil // Not used in these tests
}
func (m *mockStorage) Users() interfaces.UserRepository               { return nil }
func (m *mockStorage) APIKeys() interfaces.APIKeyRepository           { return nil }
func (m *mockStorage) Integrations() interfaces.IntegrationRepository { return nil }

// Blueprint device catalog repositories (Migration 000102-000104)
func (m *mockStorage) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (m *mockStorage) DeviceModels() interfaces.DeviceModelRepository   { return nil }
func (m *mockStorage) Blueprints() interfaces.BlueprintRepository       { return nil }

// Additional accessors (not used in tests)
func (m *mockStorage) Organizations() interfaces.OrganizationRepository     { return nil }
func (m *mockStorage) GetSqlxDB() *sqlx.DB                                  { return nil }
func (m *mockStorage) SystemEvents() interfaces.SystemEventStore            { return nil }
func (m *mockStorage) SCACISessions() interfaces.SCACISessionRepository     { return nil }
func (m *mockStorage) SCACIOperations() interfaces.SCACIOperationRepository { return nil }
func (m *mockStorage) DownlinkQueueReader() interfaces.DownlinkQueueReader  { return nil }

// Transaction support (not implemented for tests)
func (m *mockStorage) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	return nil, nil
}

// Health check
func (m *mockStorage) Ping(_ context.Context) error {
	return nil
}

// Close (no-op for tests)
func (m *mockStorage) Close() error {
	return nil
}

// DownlinkStorage interface implementation (delegates to mockMIOTYDownlinkRepository)
// This allows mockStorage to be used where DownlinkStorage is expected
func (m *mockStorage) EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	return m.miotyDownlinks.EnqueueDownlink(ctx, downlink)
}

// orgID parameter enables organization-scoped status updates
func (m *mockStorage) UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error {
	return m.miotyDownlinks.UpdateDownlinkStatus(ctx, id, status, orgID)
}

// orgID parameter enables organization-scoped result updates
func (m *mockStorage) UpdateDownlinkResult(ctx context.Context, queId int64, result string, txTime *int64, packetCnt *uint32, bsEUI []byte, epEUI []byte, tenantID string, orgID *uuid.UUID) error {
	return m.miotyDownlinks.UpdateDownlinkResult(ctx, queId, result, txTime, packetCnt, bsEUI, epEUI, tenantID, orgID)
}

func (m *mockStorage) RevokeDownlink(ctx context.Context, queId int64, tenantID string) error {
	return m.miotyDownlinks.RevokeDownlink(ctx, queId, tenantID)
}

func (m *mockStorage) CreateDLRXStatusQuery(ctx context.Context, tenantID int64, orgUUID *uuid.UUID, epEui, bsEui []byte, opId int64) error {
	if m.dlRxStatus != nil {
		return m.dlRxStatus.CreateDLRXStatusQuery(ctx, tenantID, orgUUID, epEui, bsEui, opId)
	}
	return nil
}

func (m *mockStorage) MarkDLRXStatusReceived(ctx context.Context, tenantID int64, epEui []byte, bsEui []byte, bsOpID int64) (bool, error) {
	if m.dlRxStatus != nil {
		return m.dlRxStatus.MarkDLRXStatusReceived(ctx, tenantID, epEui, bsEui, bsOpID)
	}
	return false, nil
}

func (m *mockStorage) ExpireDLRXStatusQuery(ctx context.Context, cutoff time.Time) (int64, error) {
	if m.dlRxStatus != nil {
		return m.dlRxStatus.ExpireDLRXStatusQuery(ctx, cutoff)
	}
	return 0, nil
}

// ============================================================================
// Test Helper Methods (not part of interfaces.Storage)
// ============================================================================

// SetEndpoint stores an endpoint in the mock for test assertions.
// Thread-safe for concurrent test access.
func (m *mockStorage) SetEndpoint(ep *models.EndPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.endpoints == nil {
		m.endpoints = make(map[int64]*models.EndPoint)
	}
	m.endpoints[ep.ID] = ep
}

// GetEvents returns a copy of all recorded events.
// Thread-safe for concurrent test access.
func (m *mockStorage) GetEvents() []*models.SystemEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to prevent test mutations
	result := make([]*models.SystemEvent, len(m.events))
	copy(result, m.events)
	return result
}

// AddEvent appends an event to the mock event store for test assertions.
// Thread-safe for concurrent test access.
func (m *mockStorage) AddEvent(event *models.SystemEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// ============================================================================
// Test Service Factory
// ============================================================================

// CreateTestServices creates minimal service instances for testing.
// Returns services that won't panic but don't provide full functionality.
//
// Usage in tests:
//
//	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver := bssciservices.CreateTestServices(logger, mockEventStore)
//	server := &Server{
//	    logger:          logger,
//	    sessionSvc:      sessionSvc,
//	    downlinkSvc:     downlinkSvc,
//	    statusSvc:       statusSvc,
//	    connectionSvc:   connectionSvc,
//	    broadcaster:     broadcaster,
//	    queueSerializer: queueSerializer,
//	    auditLogger:     auditLogger,
//	    tenantResolver:  tenantResolver,
//	    // ... other fields
//	}
//
// Note: Services are created with nil DB dependencies. Tests that need real
// persistence should provide their own mock implementations or use integration tests.
func CreateTestServices(log logger.Logger, eventStore interfaces.SystemEventStore) (
	bssci.SessionService,
	bssci.DownlinkService,
	bssci.StatusService,
	bssci.ConnectionService,
	bssci.SCACIBroadcaster,
	bssci.QueueSerializer,
	bssci.AuditLogger,
	bssci.TenantResolver,
	interfaces.Storage,
) {
	// Wrap zap logger for migrated services

	// SessionService with complete mock repositories
	// All repository interfaces fully implemented - no nil pointer panics
	// Order: bsSessionRepo, bsRepo, systemEventStore, tenantID, log
	sessionSvc := NewSessionService(
		newMockBaseStationSessionRepo(), // BaseStationSessionRepository
		&mockBaseStationRepo{},          // BaseStationRepository
		&mockSystemEventStore{},         // SystemEventStore
		1,                               // tenantID
		log,                             // logger
	)

	// StatusService - maintains in-memory pendingOps map
	testPendingOps := make(map[bssci.SessionOpKey]*bssci.PendingOperation)
	var testMu sync.RWMutex
	statusSvc := NewStatusService(&testPendingOps, &testMu, &mockPendingOperationRepository{}, log)

	// ConnectionService - stateless, only needs logger
	connectionSvc := NewConnectionService(log)

	// SCACIForwarder - initially unwired, safe to call (returns nil if no SCACI)
	broadcaster := NewSCACIForwarder(log)

	// QueueSerializer - stateless, no dependencies
	queueSerializer := NewQueueSerializer()

	// AuditLogger - accept eventStore from caller
	auditLogger := NewAuditLogger(eventStore)

	// TenantResolver - with nil queueStore for tests
	tenantResolver := NewTenantResolver(nil)

	// Create mock storage with MIOTY repositories
	mockStore := newMockStorage()

	// DownlinkService - with all dependencies, mock storage for tests
	downlinkSvc := NewDownlinkService(
		log,
		nil, // queueStore - nil for tests
		tenantResolver,
		nil,       // orgResolver - nil for tests
		mockStore, // interfaces.Storage with MIOTY repositories
		broadcaster,
		auditLogger,
		queueSerializer,
	)

	return sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStore
}
