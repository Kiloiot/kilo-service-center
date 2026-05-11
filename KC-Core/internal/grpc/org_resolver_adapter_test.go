package grpc

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// mockOrgRepository implements interfaces.OrganizationRepository for testing
type mockOrgRepository struct {
	getOrgByTenantIDFunc func(ctx context.Context, tenantID int64) (*models.Organization, error)
}

func (m *mockOrgRepository) GetTenantByOrgID(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil // Not used in these tests
}

func (m *mockOrgRepository) GetOrgByTenantID(ctx context.Context, tenantID int64) (*models.Organization, error) {
	if m.getOrgByTenantIDFunc != nil {
		return m.getOrgByTenantIDFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockOrgRepository) UpsertOrg(_ context.Context, _ *models.Organization) error {
	return nil
}

func (m *mockOrgRepository) Create(_ context.Context, _ *models.Organization) error {
	return nil
}

func (m *mockOrgRepository) GetByID(_ context.Context, _ uuid.UUID, _ int64) (*models.Organization, error) {
	return nil, nil
}

func (m *mockOrgRepository) Update(_ context.Context, _ uuid.UUID, _ int64, _ map[string]interface{}) error {
	return nil
}

func (m *mockOrgRepository) Delete(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}

func (m *mockOrgRepository) ListOrgMembers(_ context.Context, _ uuid.UUID, _ string) ([]*models.OrganizationMember, error) {
	return nil, nil
}

func (m *mockOrgRepository) CheckUserMembership(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockOrgRepository) AddMember(_ context.Context, _ *models.OrganizationMember) error {
	return nil
}

func (m *mockOrgRepository) RemoveMember(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockOrgRepository) UpdateMemberRole(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockOrgRepository) ListUserMemberships(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	return nil, nil
}

func (m *mockOrgRepository) GetOrgByExternalID(_ context.Context, _ string) (*models.Organization, error) {
	return nil, nil // Not used in these tests
}

func (m *mockOrgRepository) CheckBaseStationQuota(_ context.Context, _ uuid.UUID) error {
	return nil // Not used in these tests
}

func (m *mockOrgRepository) CheckEndpointQuota(_ context.Context, _ uuid.UUID) error {
	return nil // Not used in these tests
}

func (m *mockOrgRepository) CountOrgMembers(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil // Not used in these tests
}

func (m *mockOrgRepository) ListOrganizations(_ context.Context, _ *int64, _, _ int) ([]*models.Organization, int64, error) {
	return nil, 0, nil // Not used in these tests
}

func (m *mockOrgRepository) ListOrgMembersWithEmail(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*models.OrganizationMemberWithEmail, int64, error) {
	return nil, 0, nil // Not used in these tests
}

func (m *mockOrgRepository) GetOrgMemberWithEmail(_ context.Context, _, _ uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
	return nil, nil // Not used in these tests
}

func (m *mockOrgRepository) UpdateMemberPermissions(_ context.Context, _, _ uuid.UUID, _, _, _ bool) error {
	return nil // Not used in these tests
}

func (m *mockOrgRepository) CountOrgMembersByRole(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil // Not used in these tests
}

func (m *mockOrgRepository) ListUserMembershipsByTenant(_ context.Context, _ uuid.UUID, _ int64) ([]*models.OrganizationMembershipWithOrg, error) {
	return nil, nil // Not used in these tests
}

func (m *mockOrgRepository) GetByIDUnscoped(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
	return nil, nil // Not used in these tests
}

// logEntry represents a captured log call
type logEntry struct {
	level   string
	message string
	fields  map[string]interface{}
}

// capturingLogger implements logger.Logger and captures all log calls for verification
type capturingLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

func newCapturingLogger() *capturingLogger {
	return &capturingLogger{
		entries: make([]logEntry, 0),
	}
}

func (l *capturingLogger) captureWithContext(_ context.Context, level string, msg string, fields ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := logEntry{
		level:   level,
		message: msg,
		fields:  make(map[string]interface{}),
	}

	// Parse fields as key-value pairs
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			entry.fields[key] = fields[i+1]
		}
	}

	l.entries = append(l.entries, entry)
}

func (l *capturingLogger) Debug(msg string, fields ...interface{}) {
	l.capture("debug", msg, fields...)
}
func (l *capturingLogger) Info(msg string, fields ...interface{}) { l.capture("info", msg, fields...) }
func (l *capturingLogger) Warn(msg string, fields ...interface{}) { l.capture("warn", msg, fields...) }
func (l *capturingLogger) Error(msg string, fields ...interface{}) {
	l.capture("error", msg, fields...)
}
func (l *capturingLogger) Fatal(msg string, fields ...interface{}) {
	l.capture("fatal", msg, fields...)
}
func (l *capturingLogger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	l.captureWithContext(ctx, "debug", msg, fields...)
}
func (l *capturingLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	l.captureWithContext(ctx, "info", msg, fields...)
}
func (l *capturingLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	l.captureWithContext(ctx, "warn", msg, fields...)
}
func (l *capturingLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	l.captureWithContext(ctx, "error", msg, fields...)
}
func (l *capturingLogger) FatalContext(ctx context.Context, msg string, fields ...interface{}) {
	l.captureWithContext(ctx, "fatal", msg, fields...)
}

func (l *capturingLogger) capture(level string, msg string, fields ...interface{}) {
	l.captureWithContext(context.Background(), level, msg, fields...)
}

func (l *capturingLogger) WithField(_ string, _ interface{}) logger.Logger {
	return l
}

func (l *capturingLogger) WithFields(_ map[string]interface{}) logger.Logger {
	return l
}

func (l *capturingLogger) getEntries() []logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]logEntry(nil), l.entries...) // Return copy
}

func (l *capturingLogger) hasEntry(level, message string) bool {
	entries := l.getEntries()
	for _, entry := range entries {
		if entry.level == level && entry.message == message {
			return true
		}
	}
	return false
}

func (l *capturingLogger) getEntry(level, message string) *logEntry {
	entries := l.getEntries()
	for _, entry := range entries {
		if entry.level == level && entry.message == message {
			return &entry
		}
	}
	return nil
}

// ============================================================================
// Test Cases for Organization Resolver Adapter
// ============================================================================

// TestOrgResolverAdapter_ResolveOrganization_Success verifies that successful
// organization resolution returns the correct UUID and logs debug information.
//
// Requirement: adapter must correctly invoke GetOrgByTenantID and return OrgID
func TestOrgResolverAdapter_ResolveOrganization_Success(t *testing.T) {
	// Setup: Create expected organization
	expectedOrgID := uuid.MustParse("12345678-1234-1234-1234-123456789012")
	expectedTenantID := int64(100)

	mockRepo := &mockOrgRepository{
		getOrgByTenantIDFunc: func(_ context.Context, tenantID int64) (*models.Organization, error) {
			assert.Equal(t, expectedTenantID, tenantID, "Should call repository with correct tenant ID")
			return &models.Organization{
				OrgID:    expectedOrgID,
				TenantID: tenantID,
				Name:     "Test Organization",
				State:    "active",
			}, nil
		},
	}

	mockLog := newCapturingLogger()
	adapter := NewOrgResolverAdapter(mockRepo, mockLog)

	// Execute: Resolve organization
	ctx := testutil.TestContextWithTenant(expectedTenantID)
	orgID, err := adapter.ResolveOrganization(ctx, expectedTenantID)

	// Verify: Returns correct UUID without error
	require.NoError(t, err)
	assert.Equal(t, expectedOrgID, orgID, "Should return organization UUID from repository")

	// Verify: Logs debug message with correct fields
	assert.True(t, mockLog.hasEntry("debug", "Resolved organization for tenant"),
		"Should log debug message for successful resolution")

	entry := mockLog.getEntry("debug", "Resolved organization for tenant")
	require.NotNil(t, entry, "Debug entry should exist")
	assert.Equal(t, expectedTenantID, entry.fields["tenant_id"], "Should log tenant_id")
	assert.Equal(t, expectedOrgID, entry.fields["org_id"], "Should log org_id")
}

// TestOrgResolverAdapter_ResolveOrganization_NotFound verifies that when an
// organization is not found for a tenant, the error is logged as a warning
// and propagated to the caller.
//
// Requirement: adapter must handle not-found gracefully with warning logs
func TestOrgResolverAdapter_ResolveOrganization_NotFound(t *testing.T) {
	// Setup: Create repository that returns not-found error
	expectedTenantID := int64(999)
	notFoundErr := fmt.Errorf("organization not found for tenant %d", expectedTenantID)

	mockRepo := &mockOrgRepository{
		getOrgByTenantIDFunc: func(_ context.Context, _ int64) (*models.Organization, error) {
			return nil, notFoundErr
		},
	}

	mockLog := newCapturingLogger()
	adapter := NewOrgResolverAdapter(mockRepo, mockLog)

	// Execute: Resolve non-existent organization
	ctx := testutil.TestContextWithTenant(expectedTenantID)
	orgID, err := adapter.ResolveOrganization(ctx, expectedTenantID)

	// Verify: Returns nil UUID with error
	require.Error(t, err)
	assert.Equal(t, uuid.Nil, orgID, "Should return nil UUID on error")
	assert.Equal(t, notFoundErr, err, "Should propagate repository error")

	// Verify: Logs warning message with error details
	assert.True(t, mockLog.hasEntry("warn", "Failed to resolve organization for tenant"),
		"Should log warning for not-found condition")

	entry := mockLog.getEntry("warn", "Failed to resolve organization for tenant")
	require.NotNil(t, entry, "Warning entry should exist")
	assert.Equal(t, expectedTenantID, entry.fields["tenant_id"], "Should log tenant_id")
	assert.Equal(t, notFoundErr, entry.fields["error"], "Should log error")
}

// TestOrgResolverAdapter_ResolveOrganization_DatabaseError verifies that
// database errors during resolution are logged with context and propagated.
//
// Requirement: adapter must handle infrastructure failures gracefully
func TestOrgResolverAdapter_ResolveOrganization_DatabaseError(t *testing.T) {
	// Setup: Create repository that returns database error
	expectedTenantID := int64(100)
	dbErr := fmt.Errorf("database connection failed")

	mockRepo := &mockOrgRepository{
		getOrgByTenantIDFunc: func(_ context.Context, _ int64) (*models.Organization, error) {
			return nil, dbErr
		},
	}

	mockLog := newCapturingLogger()
	adapter := NewOrgResolverAdapter(mockRepo, mockLog)

	// Execute: Resolve when database is unavailable
	ctx := testutil.TestContextWithTenant(expectedTenantID)
	orgID, err := adapter.ResolveOrganization(ctx, expectedTenantID)

	// Verify: Returns nil UUID with error
	require.Error(t, err)
	assert.Equal(t, uuid.Nil, orgID, "Should return nil UUID on database error")
	assert.Equal(t, dbErr, err, "Should propagate database error")

	// Verify: Logs warning with error context
	assert.True(t, mockLog.hasEntry("warn", "Failed to resolve organization for tenant"),
		"Should log warning for database errors")

	entry := mockLog.getEntry("warn", "Failed to resolve organization for tenant")
	require.NotNil(t, entry, "Warning entry should exist")
	assert.Equal(t, expectedTenantID, entry.fields["tenant_id"], "Should log tenant_id")
	assert.Equal(t, dbErr, entry.fields["error"], "Should log database error")
}

// TestOrgResolverAdapter_ResolveOrganization_LogsSuccess verifies that
// successful resolutions are logged at debug level with tenant and org context.
//
// Requirement: verify debug logging for audit trail
func TestOrgResolverAdapter_ResolveOrganization_LogsSuccess(t *testing.T) {
	// Setup: Create test organizations
	testCases := []struct {
		name     string
		tenantID int64
		orgID    uuid.UUID
		orgName  string
	}{
		{
			name:     "tenant_100",
			tenantID: 100,
			orgID:    uuid.MustParse("00000000-0000-0000-0000-000000000100"),
			orgName:  "Organization 100",
		},
		{
			name:     "tenant_200",
			tenantID: 200,
			orgID:    uuid.MustParse("00000000-0000-0000-0000-000000000200"),
			orgName:  "Organization 200",
		},
		{
			name:     "tenant_300",
			tenantID: 300,
			orgID:    uuid.MustParse("00000000-0000-0000-0000-000000000300"),
			orgName:  "Organization 300",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockOrgRepository{
				getOrgByTenantIDFunc: func(_ context.Context, tenantID int64) (*models.Organization, error) {
					return &models.Organization{
						OrgID:    tc.orgID,
						TenantID: tenantID,
						Name:     tc.orgName,
						State:    "active",
					}, nil
				},
			}

			mockLog := newCapturingLogger()
			adapter := NewOrgResolverAdapter(mockRepo, mockLog)

			// Execute: Resolve organization
			ctx := testutil.TestContextWithTenant(tc.tenantID)
			orgID, err := adapter.ResolveOrganization(ctx, tc.tenantID)

			// Verify: Success
			require.NoError(t, err)
			assert.Equal(t, tc.orgID, orgID)

			// Verify: Debug log contains correct context
			entry := mockLog.getEntry("debug", "Resolved organization for tenant")
			require.NotNil(t, entry, "Should have debug log entry")
			assert.Equal(t, tc.tenantID, entry.fields["tenant_id"])
			assert.Equal(t, tc.orgID, entry.fields["org_id"])
		})
	}
}

// TestOrgResolverAdapter_ResolveOrganization_EmptyTenantID verifies that
// the adapter handles edge case of tenant ID 0.
//
// Requirement: verify behavior with edge case inputs
func TestOrgResolverAdapter_ResolveOrganization_EmptyTenantID(t *testing.T) {
	// Setup: Create repository that returns error for tenant 0
	mockRepo := &mockOrgRepository{
		getOrgByTenantIDFunc: func(_ context.Context, tenantID int64) (*models.Organization, error) {
			if tenantID == 0 {
				return nil, fmt.Errorf("tenant ID cannot be zero")
			}
			return nil, nil
		},
	}

	mockLog := newCapturingLogger()
	adapter := NewOrgResolverAdapter(mockRepo, mockLog)

	// Execute: Attempt to resolve organization for tenant 0
	ctx := testutil.TestContextWithTenant(0)
	orgID, err := adapter.ResolveOrganization(ctx, 0)

	// Verify: Returns error
	require.Error(t, err)
	assert.Equal(t, uuid.Nil, orgID, "Should return nil UUID on error")

	// Verify: Error is logged
	assert.True(t, mockLog.hasEntry("warn", "Failed to resolve organization for tenant"),
		"Should log warning for tenant ID 0")
}

// TestOrgResolverAdapter_ResolveOrganization_MultipleCallsSameContext verifies
// that multiple resolution calls with the same tenant ID behave consistently.
//
// Requirement: verify stateless behavior for auth interceptor usage
func TestOrgResolverAdapter_ResolveOrganization_MultipleCallsSameContext(t *testing.T) {
	// Setup: Create adapter that tracks call count
	callCount := 0
	expectedOrgID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	expectedTenantID := int64(100)

	mockRepo := &mockOrgRepository{
		getOrgByTenantIDFunc: func(_ context.Context, tenantID int64) (*models.Organization, error) {
			callCount++
			return &models.Organization{
				OrgID:    expectedOrgID,
				TenantID: tenantID,
				Name:     "Test Organization",
				State:    "active",
			}, nil
		},
	}

	mockLog := newCapturingLogger()
	adapter := NewOrgResolverAdapter(mockRepo, mockLog)

	// Execute: Make multiple calls with same tenant
	ctx := testutil.TestContextWithTenant(expectedTenantID)

	for i := 0; i < 3; i++ {
		orgID, err := adapter.ResolveOrganization(ctx, expectedTenantID)

		// Verify: Each call succeeds with same result
		require.NoError(t, err)
		assert.Equal(t, expectedOrgID, orgID, "Should return consistent org ID across calls")
	}

	// Verify: Repository called for each resolution (adapter does not cache)
	assert.Equal(t, 3, callCount, "Adapter should call repository for each resolution")

	// Verify: Debug logs for all successful resolutions
	entries := mockLog.getEntries()
	debugCount := 0
	for _, entry := range entries {
		if entry.level == "debug" && entry.message == "Resolved organization for tenant" {
			debugCount++
		}
	}
	assert.Equal(t, 3, debugCount, "Should log debug message for each resolution")
}
