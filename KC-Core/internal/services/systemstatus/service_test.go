package systemstatus_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/systemstatus"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Metric seeds returned by the repository mocks.
const (
	testOnlineBaseStations  = 5
	testEndpointCount       = 100
	testMessagesProcessed   = 50000
	testIgnoredMetricValue  = 999
	testPartialMessageCount = 12345
	testNilStatsEndpoints   = 42
)

// errRepoFailure is the failure every erroring repository mock returns.
var errRepoFailure = errors.New("database error")

// mockLogger implements logger.Logger for testing
type mockLogger struct{}

func (m *mockLogger) InfoContext(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) WarnContext(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) ErrorContext(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) DebugContext(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) FatalContext(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) Info(_ string, _ ...any)                            {}
func (m *mockLogger) Warn(_ string, _ ...any)                            {}
func (m *mockLogger) Error(_ string, _ ...any)                           {}
func (m *mockLogger) Debug(_ string, _ ...any)                           {}
func (m *mockLogger) Fatal(_ string, _ ...any)                           {}
func (m *mockLogger) WithField(_ string, _ any) logger.Logger            { return m }
func (m *mockLogger) WithFields(_ map[string]any) logger.Logger          { return m }

// mockBSRepo implements BaseStationStatsFetcher
type mockBSRepo struct {
	stats *models.BaseStationStatistics
	err   error
}

func (m *mockBSRepo) GetStatistics(_ context.Context, _ int64) (*models.BaseStationStatistics, error) {
	return m.stats, m.err
}

// mockEPRepo implements EndpointCounter
type mockEPRepo struct {
	count int64
	err   error
}

func (m *mockEPRepo) CountByTenant(_ context.Context, _ int64) (int64, error) {
	return m.count, m.err
}

// mockMsgRepo implements MessageStatsFetcher
type mockMsgRepo struct {
	stats *mioty.MessageStats
	err   error
}

func (m *mockMsgRepo) GetOverallStats(_ context.Context, _ int64) (*mioty.MessageStats, error) {
	return m.stats, m.err
}

func TestGetStatus_WithMetrics(t *testing.T) {
	ctx := testutil.TestContextWithTenant(123)

	bsRepo := &mockBSRepo{stats: &models.BaseStationStatistics{OnlineCount: testOnlineBaseStations}}
	epRepo := &mockEPRepo{count: testEndpointCount}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: testMessagesProcessed}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{}, nil, nil)

	metrics, err := svc.GetStatus(ctx, 123)

	require.NoError(t, err)
	assert.Equal(t, int32(testOnlineBaseStations), metrics.ActiveBasestations)
	assert.Equal(t, int32(testEndpointCount), metrics.ActiveEndpoints)
	assert.Equal(t, int64(testMessagesProcessed), metrics.MessagesProcessed)
}

func TestGetStatus_ZeroTenant(t *testing.T) {
	ctx := testutil.TestContextWithTenant(0)

	// Repos should not be called for zero tenant
	bsRepo := &mockBSRepo{stats: &models.BaseStationStatistics{OnlineCount: testIgnoredMetricValue}}
	epRepo := &mockEPRepo{count: testIgnoredMetricValue}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: testIgnoredMetricValue}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{}, nil, nil)

	metrics, err := svc.GetStatus(ctx, 0)

	require.NoError(t, err)
	// Should return empty metrics, not the seeded values
	assert.Equal(t, int32(0), metrics.ActiveBasestations)
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

func TestGetStatus_NegativeTenant(t *testing.T) {
	ctx := testutil.TestContextWithTenant(-1)

	svc := systemstatus.New(nil, nil, nil, &mockLogger{}, nil, nil)

	metrics, err := svc.GetStatus(ctx, -1)

	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations)
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

func TestGetStatus_RepoErrors(t *testing.T) {
	// Test that repo errors log warnings but return partial results
	ctx := testutil.TestContextWithTenant(1)

	// Only msgRepo succeeds
	bsRepo := &mockBSRepo{err: errRepoFailure}
	epRepo := &mockEPRepo{err: errRepoFailure}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: testPartialMessageCount}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{}, nil, nil)

	metrics, err := svc.GetStatus(ctx, 1)

	// Should still succeed with partial data
	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations)                      // Failed, returns 0
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)                         // Failed, returns 0
	assert.Equal(t, int64(testPartialMessageCount), metrics.MessagesProcessed) // Success
}

func TestGetStatus_NilRepos(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := systemstatus.New(nil, nil, nil, &mockLogger{}, nil, nil)

	metrics, err := svc.GetStatus(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations)
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

func TestGetStatus_ClampToInt32(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	// Test values larger than MaxInt32
	bsRepo := &mockBSRepo{stats: &models.BaseStationStatistics{OnlineCount: math.MaxInt64}}
	epRepo := &mockEPRepo{count: math.MaxInt64}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: math.MaxInt64}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{}, nil, nil)

	metrics, err := svc.GetStatus(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, int32(math.MaxInt32), metrics.ActiveBasestations)
	assert.Equal(t, int32(math.MaxInt32), metrics.ActiveEndpoints)
	assert.Equal(t, int64(math.MaxInt64), metrics.MessagesProcessed) // int64, no clamping
}

func TestGetStatus_NilStatsResult(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	// Test nil stats return (not error)
	bsRepo := &mockBSRepo{stats: nil, err: nil}
	msgRepo := &mockMsgRepo{stats: nil, err: nil}
	epRepo := &mockEPRepo{count: testNilStatsEndpoints}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{}, nil, nil)

	metrics, err := svc.GetStatus(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations) // nil stats
	assert.Equal(t, int32(testNilStatsEndpoints), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed) // nil stats
}

const (
	testRowRunning  = "running"
	testRowDisabled = "disabled"
	testDisabledMsg = "turned off"
	testRowURL      = "tcp://running"
)

type fixedHealth struct{ response *health.Response }

func (f fixedHealth) CheckHealth(context.Context) *health.Response { return f.response }

func TestGetServiceStatuses_ListsRunningServicesOnly(t *testing.T) {
	checks := fixedHealth{response: &health.Response{Checks: map[string]*health.Check{
		testRowRunning:  {Status: health.StatusHealthy},
		testRowDisabled: {Status: health.StatusDisabled, Message: testDisabledMsg},
	}}}
	svc := systemstatus.New(nil, nil, nil, &mockLogger{}, checks, []systemstatus.EndpointURL{{Name: testRowRunning, URL: testRowURL}})

	services, err := svc.GetServiceStatuses(testutil.TestContext())
	require.NoError(t, err)
	require.Len(t, services, 1, "a disabled component is not listed as a failing service")
	assert.Equal(t, testRowRunning, services[0].Name)
	assert.True(t, services[0].Healthy)
	assert.Equal(t, testRowURL, services[0].URL)
}
