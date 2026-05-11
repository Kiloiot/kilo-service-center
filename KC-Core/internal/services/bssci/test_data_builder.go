package bssciservices

import (
	"context"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// TestDataBuilder provides a fluent API for populating mock repositories with test data.
// This allows tests to seed the exact state needed for complex handler validation (attach,
// detach, ULData, DLDataResult) without bypassing the service layer.
//
// Usage:
//
//	builder := NewTestDataBuilder()
//	builder.WithEndpoint(0x1234567890ABCDEF, testKey, true)
//	builder.WithPendingDownlink(queId, epEui, payload)
//	mockStorage := builder.Build()
//	// Pass mockStorage to CreateTestServices via updated signature
type TestDataBuilder struct {
	endpoints      map[uint64]*models.EndPoint // keyed by EUI as uint64
	downlinks      map[uint64]*downlinkEntry   // keyed by queId
	uplinkMessages []mioty.ULDataMessage
	vmContext      string // macType for VM operations
	tenantID       int64  // default tenant for test data
}

// downlinkEntry holds downlink queue state for testing
type downlinkEntry struct {
	queId     uint64 //nolint:revive // queId matches MIOTY spec naming
	epEui     []byte // 8-byte EUI
	payload   []byte
	status    string
	txTime    *int64
	packetCnt *uint32
	tenantID  int64
}

// NewTestDataBuilder creates a new builder with empty state
func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		endpoints:      make(map[uint64]*models.EndPoint),
		downlinks:      make(map[uint64]*downlinkEntry),
		uplinkMessages: []mioty.ULDataMessage{},
		tenantID:       1, // default test tenant
	}
}

// WithTenantID sets the default tenant for all seeded data
func (b *TestDataBuilder) WithTenantID(tenantID int64) *TestDataBuilder {
	b.tenantID = tenantID
	return b
}

// WithEndpoint seeds an endpoint with basic provisioning data
func (b *TestDataBuilder) WithEndpoint(epEui uint64, nwkSnKey []byte, bidi bool) *TestDataBuilder {
	// Convert uint64 to models.EUI [8]byte
	var eui models.EUI
	for i := 0; i < 8; i++ {
		eui[7-i] = byte(epEui >> (uint(i) * 8)) //nolint:gosec // G115: i bounded 0-7, safe
	}

	b.endpoints[epEui] = &models.EndPoint{
		EUI:      eui,
		NwkSnKey: nwkSnKey,
		Bidi:     bidi,
		TenantID: b.tenantID,
		ShAddr:   nil, // not attached
	}
	return b
}

// WithAttachedEndpoint seeds an endpoint in attached state with runtime counters
// Parameters use unsigned types per SCACI §3.6.1 (uint16 shAddr, uint32 attachCnt/lastPacketCnt)
func (b *TestDataBuilder) WithAttachedEndpoint(epEui uint64, nwkSnKey []byte, shAddr uint16, attachCnt uint32, lastPacketCnt uint32) *TestDataBuilder {
	// Convert uint64 to models.EUI [8]byte
	var eui models.EUI
	for i := 0; i < 8; i++ {
		eui[7-i] = byte(epEui >> (uint(i) * 8)) //nolint:gosec // G115: i bounded 0-7, safe
	}

	shAddrPtr := shAddr
	attachCntPtr := attachCnt

	b.endpoints[epEui] = &models.EndPoint{
		EUI:           eui,
		NwkSnKey:      nwkSnKey,
		Bidi:          true, // attached endpoints must be bidi
		TenantID:      b.tenantID,
		ShAddr:        &shAddrPtr,
		AttachCnt:     &attachCntPtr,
		LastPacketCnt: lastPacketCnt,
	}
	return b
}

// WithPendingDownlink seeds a downlink queue entry awaiting transmission
func (b *TestDataBuilder) WithPendingDownlink(queId uint64, epEui uint64, payload []byte) *TestDataBuilder { //nolint:revive // queId matches MIOTY spec
	// Convert uint64 to []byte (8 bytes big-endian)
	euiBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		euiBytes[7-i] = byte(epEui >> (uint(i) * 8)) //nolint:gosec // G115: i bounded 0-7, safe
	}

	b.downlinks[queId] = &downlinkEntry{
		queId:    queId,
		epEui:    euiBytes,
		payload:  payload,
		status:   "pending",
		tenantID: b.tenantID,
	}
	return b
}

// WithDownlinkResult seeds a downlink queue entry with result metadata
func (b *TestDataBuilder) WithDownlinkResult(queId uint64, epEui uint64, result int, txTime *int64, packetCnt *uint32) *TestDataBuilder { //nolint:revive // queId matches MIOTY spec
	// Convert uint64 to []byte (8 bytes big-endian)
	euiBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		euiBytes[7-i] = byte(epEui >> (uint(i) * 8)) //nolint:gosec // G115: i bounded 0-7, safe
	}

	status := "pending"
	switch result {
	case 0:
		status = "transmitted"
	case 1:
		status = "expired"
	case 2:
		status = "discarded"
	}

	b.downlinks[queId] = &downlinkEntry{
		queId:     queId,
		epEui:     euiBytes,
		tenantID:  b.tenantID,
		status:    status,
		txTime:    txTime,
		packetCnt: packetCnt,
	}
	return b
}

// WithUplinkHistory seeds uplink message history for an endpoint
func (b *TestDataBuilder) WithUplinkHistory(epEui uint64, messages []mioty.ULDataMessage) *TestDataBuilder { //nolint:revive // epEui reserved for future filtering
	b.uplinkMessages = append(b.uplinkMessages, messages...)
	return b
}

// WithVMContext seeds Variable MAC context for VM operation tests
func (b *TestDataBuilder) WithVMContext(macType string) *TestDataBuilder {
	b.vmContext = macType
	return b
}

// Build constructs a mockStorage instance populated with seeded data
func (b *TestDataBuilder) Build() interfaces.Storage {
	endpointRepo := &mockEndpointRepository{
		endpoints: b.endpoints,
		mu:        &sync.RWMutex{},
	}

	downlinkRepo := &mockDownlinkQueueRepository{
		downlinks: b.downlinks,
		mu:        &sync.RWMutex{},
	}

	// Create base mockStorage
	baseStorage := newMockStorage()

	// Wrap with populated repositories
	return &mockStorageWithData{
		mockStorage:   baseStorage,
		endpointRepo:  endpointRepo,
		downlinkRepo:  downlinkRepo,
		miotyMessages: baseStorage.miotyMessages,
		miotyDownlinks: &mockMIOTYDownlinkRepositoryWithData{
			mockMIOTYDownlinkRepository: baseStorage.miotyDownlinks,
			downlinks:                   b.downlinks,
			mu:                          &sync.RWMutex{},
		},
		dlRxStatus: baseStorage.dlRxStatus,
	}
}

// mockStorageWithData extends mockStorage with populated repositories
type mockStorageWithData struct {
	*mockStorage
	endpointRepo   *mockEndpointRepository
	downlinkRepo   *mockDownlinkQueueRepository
	miotyMessages  *mockMIOTYMessageRepository
	miotyDownlinks *mockMIOTYDownlinkRepositoryWithData
	dlRxStatus     *mockDLRXStatusRepository
}

// Override EndPoints() to return populated repository
func (m *mockStorageWithData) EndPoints() interfaces.EndpointRepository {
	return m.endpointRepo
}

// Override DownlinkQueue() to return populated repository
func (m *mockStorageWithData) DownlinkQueue() interfaces.DownlinkQueueRepository {
	return m.downlinkRepo
}

// Override MIOTYDownlinks() to return populated repository
func (m *mockStorageWithData) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository {
	return m.miotyDownlinks
}

// mockEndpointRepository implements interfaces.EndpointRepository with in-memory storage
type mockEndpointRepository struct {
	endpoints map[uint64]*models.EndPoint // keyed by EUI as uint64
	mu        *sync.RWMutex
}

func (m *mockEndpointRepository) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) { //nolint:revive // ctx unused in mock
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert []byte to uint64 for lookup
	var euiInt uint64
	for i := 0; i < 8 && i < len(eui); i++ {
		euiInt = (euiInt << 8) | uint64(eui[i])
	}

	if ep, exists := m.endpoints[euiInt]; exists && ep.TenantID == tenantID {
		// Return copy to prevent test mutations
		epCopy := *ep
		return &epCopy, nil
	}
	return nil, nil // not found
}

func (m *mockEndpointRepository) Create(ctx context.Context, ep *models.EndPoint) error { //nolint:revive // ctx unused in mock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert EUI to uint64 for storage
	euiInt := ep.EUI.ToUint64()
	m.endpoints[euiInt] = ep
	return nil
}

func (m *mockEndpointRepository) Update(ctx context.Context, ep *models.EndPoint) error { //nolint:revive // ctx unused in mock
	m.mu.Lock()
	defer m.mu.Unlock()

	euiInt := ep.EUI.ToUint64()
	if existing, exists := m.endpoints[euiInt]; exists && existing.TenantID == ep.TenantID {
		m.endpoints[euiInt] = ep
		return nil
	}
	return nil // not found, no-op for tests
}

func (m *mockEndpointRepository) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) { //nolint:revive // stub
	return ep, nil
}

func (m *mockEndpointRepository) CheckEUIUnique(_ context.Context, _ []byte) error { //nolint:revive // stub
	return nil
}

func (m *mockEndpointRepository) Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error) { //nolint:revive // ctx unused in mock
	m.mu.RLock()
	defer m.mu.RUnlock()

	euiInt := eui.ToUint64()
	if ep, exists := m.endpoints[euiInt]; exists {
		epCopy := *ep
		return &epCopy, nil
	}
	return nil, nil
}

func (m *mockEndpointRepository) GetByTenant(ctx context.Context, tenantID int64) ([]*models.EndPoint, error) { //nolint:revive // ctx unused in mock
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.EndPoint
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID {
			epCopy := *ep
			result = append(result, &epCopy)
		}
	}
	return result, nil
}

func (m *mockEndpointRepository) CountByTenant(ctx context.Context, tenantID int64) (int64, error) { //nolint:revive // ctx unused in mock
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *mockEndpointRepository) ListByTenantPaginated(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) { //nolint:revive // ctx unused in mock
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []*models.EndPoint
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID {
			epCopy := *ep
			all = append(all, &epCopy)
		}
	}

	// Apply pagination
	start := offset
	if start > len(all) {
		return []*models.EndPoint{}, nil
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], nil
}

func (m *mockEndpointRepository) GetByID(_ context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ep := range m.endpoints {
		if ep.ID == id && ep.TenantID == tenantID {
			epCopy := *ep
			return &epCopy, nil
		}
	}
	return nil, nil
}

func (m *mockEndpointRepository) DeleteByTenant(_ context.Context, _ int64, _ []byte) error {
	return nil // no-op for basic tests
}

func (m *mockEndpointRepository) UpdateLastSeen(_ context.Context, _ int64, _ models.EUI, _ uint32) error {
	return nil // no-op for basic tests
}

func (m *mockEndpointRepository) UpdateRadioMetrics(_ context.Context, _ int64, _ models.EUI, _, _, _ float64, _, _ int64, _ string) error {
	return nil // no-op for basic tests
}

func (m *mockEndpointRepository) UpdateRadioMetricsSelective(_ context.Context, _ int64, _ models.EUI, _ interfaces.RadioMetricsUpdate) error {
	return nil // no-op for basic tests
}

func (m *mockEndpointRepository) UpdateDetachMetrics(_ context.Context, _ int64, _ models.EUI, _ interfaces.DetachMetricsUpdate) error {
	return nil // no-op for basic tests
}

func (m *mockEndpointRepository) UpdateFields(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	return nil // no-op for basic tests
}

func (m *mockEndpointRepository) StreamAllForPropagation(_ context.Context, _ int64, _ int) ([]*models.EndPoint, error) {
	return nil, nil // no-op for basic tests
}

func (m *mockEndpointRepository) HasEndpointsSince(_ context.Context, _ time.Time) (bool, error) {
	return false, nil // no-op for basic tests
}

func (m *mockEndpointRepository) GetEndpointWithKeysForDetachValidation(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	euiInt := eui.ToUint64()
	if ep, exists := m.endpoints[euiInt]; exists {
		epCopy := *ep
		return &epCopy, nil
	}
	return nil, storage.ErrNotFound
}

func (m *mockEndpointRepository) GetPreferredBsEui(_ context.Context, _ int64, _ []byte) (*uint64, bool, error) {
	return nil, false, nil // no-op for basic tests
}

// mockDownlinkQueueRepository implements interfaces.DownlinkQueueRepository with in-memory storage
type mockDownlinkQueueRepository struct {
	downlinks map[uint64]*downlinkEntry
	mu        *sync.RWMutex
}

// Stub methods to satisfy interfaces.DownlinkQueueRepository
func (m *mockDownlinkQueueRepository) Enqueue(_ context.Context, _ *models.DownlinkMessage) error {
	return nil // no-op for tests
}

func (m *mockDownlinkQueueRepository) GetPending(_ context.Context, _ models.EUI) ([]*models.DownlinkMessage, error) {
	return []*models.DownlinkMessage{}, nil
}

func (m *mockDownlinkQueueRepository) GetNext(_ context.Context, _ models.EUI) (*models.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockDownlinkQueueRepository) UpdateStatus(_ context.Context, _ int64, _ string, _ *models.EUI) error {
	return nil
}

func (m *mockDownlinkQueueRepository) MarkTransmitted(_ context.Context, _ int64, _ models.EUI) error {
	return nil
}

func (m *mockDownlinkQueueRepository) MarkConfirmed(_ context.Context, _ int64) error {
	return nil
}

func (m *mockDownlinkQueueRepository) MarkFailed(_ context.Context, _ int64, _ string) error {
	return nil
}

func (m *mockDownlinkQueueRepository) ExpireOldMessages(_ context.Context) (int64, error) {
	return 0, nil
}

func (m *mockDownlinkQueueRepository) GetByTenant(_ context.Context, _ int64, _ string) ([]*models.DownlinkMessage, error) {
	return []*models.DownlinkMessage{}, nil
}

// mockMIOTYDownlinkRepositoryWithData extends the base mock with seeded downlink data
type mockMIOTYDownlinkRepositoryWithData struct {
	*mockMIOTYDownlinkRepository
	downlinks map[uint64]*downlinkEntry
	mu        *sync.RWMutex
}

// Override UpdateDownlinkResult to use seeded data
// orgID parameter enables organization-scoped result updates
func (m *mockMIOTYDownlinkRepositoryWithData) UpdateDownlinkResult(_ context.Context, queId int64, result string, txTime *int64, packetCnt *uint32, _ []byte, _ []byte, _ string, _ *uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dl, exists := m.downlinks[uint64(queId)]; exists { //nolint:gosec // G115: queId positive, safe conversion
		dl.status = result
		dl.txTime = txTime
		dl.packetCnt = packetCnt
	}
	return nil
}
