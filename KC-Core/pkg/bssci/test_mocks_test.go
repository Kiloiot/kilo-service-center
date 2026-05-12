package bssci

import (
	"context"
	"fmt"
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// Compile-time interface assertions ensure mocks implement their contracts
var _ TenantResolver = (*mockTenantResolver)(nil)
var _ QueueSerializer = (*mockQueueSerializer)(nil)
var _ DetachSignatureValidator = (*mockDetachSignatureValidator)(nil)
var _ UplinkIngestService = (*mockUplinkIngestSvc)(nil)

// mockTenantResolver implements TenantResolver for testing
// Provides thread-safe queue ID → tenant ID mapping for test isolation
// Note: unexported type name (lowercase) ensures this is test-only
type mockTenantResolver struct {
	mu     sync.RWMutex
	queues map[int64]string
	logger logger.Logger
}

// NewMockTenantResolver creates a new mock tenant resolver for tests
func NewMockTenantResolver(log logger.Logger) TenantResolver {
	return &mockTenantResolver{
		queues: make(map[int64]string),
		logger: log,
	}
}

// RegisterQueueTenant registers a queue-to-tenant mapping
func (m *mockTenantResolver) RegisterQueueTenant(queueID int64, tenantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.queues == nil {
		m.queues = make(map[int64]string)
	}
	m.queues[queueID] = tenantID
}

// ResolveTenant resolves a tenant ID from a queue ID
func (m *mockTenantResolver) ResolveTenant(_ context.Context, queueID int64) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if tenantID, ok := m.queues[queueID]; ok {
		return tenantID, nil
	}
	return "", fmt.Errorf("queue %d not found", queueID)
}

// UnregisterQueueTenant removes a queue-to-tenant mapping
func (m *mockTenantResolver) UnregisterQueueTenant(queueID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.queues, queueID)
}

// mockQueueSerializer implements QueueSerializer for testing
// Returns minimal test payloads with command and opId fields
// Note: unexported type name (lowercase) ensures this is test-only
type mockQueueSerializer struct{}

// NewMockQueueSerializer creates a new mock queue serializer for tests
func NewMockQueueSerializer() QueueSerializer {
	return &mockQueueSerializer{}
}

// BuildDLDataQueueComplete constructs dlDataQueCmp response per BSSCI §5.12
func (m *mockQueueSerializer) BuildDLDataQueueComplete(opId int64) map[string]interface{} {
	return map[string]interface{}{
		"command": "dlDataQueCmp",
		"opId":    opId,
	}
}

// BuildDLDataResultResponse constructs dlDataResRsp response
func (m *mockQueueSerializer) BuildDLDataResultResponse(opId int64, _ *mioty.DLDataResult) map[string]interface{} {
	return map[string]interface{}{
		"command": "dlDataResRsp",
		"opId":    opId,
	}
}

// BuildDLDataResultComplete constructs dlDataResCmp response
func (m *mockQueueSerializer) BuildDLDataResultComplete(opId int64) map[string]interface{} {
	return map[string]interface{}{
		"command": "dlDataResCmp",
		"opId":    opId,
	}
}

// BuildDLDataRevokeComplete constructs dlDataRevCmp response
func (m *mockQueueSerializer) BuildDLDataRevokeComplete(opID int64) map[string]interface{} {
	return map[string]interface{}{
		"command": "dlDataRevCmp",
		"opId":    opID,
	}
}

// mockDetachSignatureValidator implements DetachSignatureValidator for testing
// Allows tests to configure validation behavior via validateFunc
// Note: unexported type name (lowercase) ensures this is test-only
type mockDetachSignatureValidator struct {
	validateFunc func(ctx context.Context, epEUI uint64, detachSign []byte) (*DetachValidationResult, error)
}

// NewMockDetachSignatureValidator creates a new mock detach validator for tests
// Pass nil for validateFunc to default to successful validation
func NewMockDetachSignatureValidator(validateFunc func(context.Context, uint64, []byte) (*DetachValidationResult, error)) DetachSignatureValidator {
	return &mockDetachSignatureValidator{
		validateFunc: validateFunc,
	}
}

// ValidateDetachSignature validates a detach signature using configured test behavior
func (m *mockDetachSignatureValidator) ValidateDetachSignature(ctx context.Context, epEUI uint64, detachSign []byte) (*DetachValidationResult, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, epEUI, detachSign)
	}
	// Default: successful validation with minimal metadata
	return &DetachValidationResult{
		Valid:            true,
		TenantID:         1,
		OwnerTenantID:    1,
		ValidationStatus: ValidationStatusValidated,
	}, nil
}

// Compile-time interface assertion for SCACIEPStatusBroadcaster mock
var _ SCACIEPStatusBroadcaster = (*mockSCACIEPStatusBroadcaster)(nil)

// mockSCACIEPStatusBroadcaster implements SCACIEPStatusBroadcaster for testing
// Captures BroadcastEPStatus calls for verification in tests
// Thread-safe for concurrent access from goroutines
type mockSCACIEPStatusBroadcaster struct {
	mu          sync.Mutex
	calls       []epStatusCall
	returnError error
}

// epStatusCall captures a single BroadcastEPStatus invocation
type epStatusCall struct {
	TenantID int64
	Data     *EPStatusData
}

// NewMockSCACIEPStatusBroadcaster creates a new mock for testing
//
//nolint:revive // Returns concrete type intentionally for test helper methods
func NewMockSCACIEPStatusBroadcaster() *mockSCACIEPStatusBroadcaster {
	return &mockSCACIEPStatusBroadcaster{
		calls: make([]epStatusCall, 0),
	}
}

// SetError configures the mock to return an error on BroadcastEPStatus calls
func (m *mockSCACIEPStatusBroadcaster) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.returnError = err
}

// BroadcastEPStatus records the call and returns configured error
func (m *mockSCACIEPStatusBroadcaster) BroadcastEPStatus(_ context.Context, tenantID int64, data *EPStatusData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, epStatusCall{TenantID: tenantID, Data: data})
	return m.returnError
}

// CallCount returns the number of BroadcastEPStatus invocations
func (m *mockSCACIEPStatusBroadcaster) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// GetCall returns the captured call at the given index (0-based)
// Returns nil if index is out of bounds
func (m *mockSCACIEPStatusBroadcaster) GetCall(index int) *epStatusCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index < 0 || index >= len(m.calls) {
		return nil
	}
	return &m.calls[index]
}

// LastCall returns the most recent BroadcastEPStatus call, or nil if none
func (m *mockSCACIEPStatusBroadcaster) LastCall() *epStatusCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return nil
	}
	return &m.calls[len(m.calls)-1]
}

// Reset clears all recorded calls
func (m *mockSCACIEPStatusBroadcaster) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]epStatusCall, 0)
	m.returnError = nil
}

// syncMockSCACIEPStatusBroadcaster wraps mockSCACIEPStatusBroadcaster with WaitGroup
// for deterministic synchronization in handler-level tests.
// This avoids flaky time.Sleep-based tests per SCACI §3.13 EPStatus forwarding.
type syncMockSCACIEPStatusBroadcaster struct {
	*mockSCACIEPStatusBroadcaster
	wg sync.WaitGroup
}

// NewSyncMockSCACIEPStatusBroadcaster creates a sync-enabled mock broadcaster
//
//nolint:revive // Returns concrete type intentionally for test helper methods
func NewSyncMockSCACIEPStatusBroadcaster() *syncMockSCACIEPStatusBroadcaster {
	return &syncMockSCACIEPStatusBroadcaster{
		mockSCACIEPStatusBroadcaster: NewMockSCACIEPStatusBroadcaster(),
	}
}

// BroadcastEPStatus records the call and decrements WaitGroup for sync
func (m *syncMockSCACIEPStatusBroadcaster) BroadcastEPStatus(ctx context.Context, tenantID int64, data *EPStatusData) error {
	defer m.wg.Done()
	return m.mockSCACIEPStatusBroadcaster.BroadcastEPStatus(ctx, tenantID, data)
}

// ExpectCall prepares the WaitGroup to expect n broadcast calls
func (m *syncMockSCACIEPStatusBroadcaster) ExpectCall(n int) {
	m.wg.Add(n)
}

// Wait blocks until all expected broadcast calls complete
func (m *syncMockSCACIEPStatusBroadcaster) Wait() {
	m.wg.Wait()
}

// mockUplinkIngestSvc implements UplinkIngestService for testing.
// Returns a successful IngestResult with the endpoint's owner tenant.
type mockUplinkIngestSvc struct{}

func (m *mockUplinkIngestSvc) Ingest(_ context.Context, _ *UplinkPayload, _ UplinkIngestOptions) (*IngestResult, error) {
	return &IngestResult{
		OwnerTenantID: 0,
		OwnerOrgUUID:  uuid.Nil,
		IsDuplicate:   false,
		MessageID:     "test-msg-id",
	}, nil
}

// NewMockUplinkIngestSvc creates a mock UplinkIngestService for tests
func NewMockUplinkIngestSvc() UplinkIngestService {
	return &mockUplinkIngestSvc{}
}
