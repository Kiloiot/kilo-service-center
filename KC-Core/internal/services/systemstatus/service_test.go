package systemstatus_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/systemstatus"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	stats *interfaces.BaseStationStatistics
	err   error
}

func (m *mockBSRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
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

	bsRepo := &mockBSRepo{stats: &interfaces.BaseStationStatistics{OnlineCount: 5}}
	epRepo := &mockEPRepo{count: 100}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: 50000}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{})

	metrics, err := svc.GetStatus(ctx, 123)

	require.NoError(t, err)
	assert.Equal(t, int32(5), metrics.ActiveBasestations)
	assert.Equal(t, int32(100), metrics.ActiveEndpoints)
	assert.Equal(t, int64(50000), metrics.MessagesProcessed)
}

func TestGetStatus_ZeroTenant(t *testing.T) {
	ctx := testutil.TestContextWithTenant(0)

	// Repos should not be called for zero tenant
	bsRepo := &mockBSRepo{stats: &interfaces.BaseStationStatistics{OnlineCount: 999}}
	epRepo := &mockEPRepo{count: 999}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: 999}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{})

	metrics, err := svc.GetStatus(ctx, 0)

	require.NoError(t, err)
	// Should return empty metrics, not 999
	assert.Equal(t, int32(0), metrics.ActiveBasestations)
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

func TestGetStatus_NegativeTenant(t *testing.T) {
	ctx := testutil.TestContextWithTenant(-1)

	svc := systemstatus.New(nil, nil, nil, &mockLogger{})

	metrics, err := svc.GetStatus(ctx, -1)

	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations)
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

func TestGetStatus_RepoErrors(t *testing.T) {
	// Test that repo errors log warnings but return partial results
	ctx := testutil.TestContextWithTenant(1)

	testErr := errors.New("database error")

	// Only msgRepo succeeds
	bsRepo := &mockBSRepo{err: testErr}
	epRepo := &mockEPRepo{err: testErr}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: 12345}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{})

	metrics, err := svc.GetStatus(ctx, 1)

	// Should still succeed with partial data
	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations)    // Failed, returns 0
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)       // Failed, returns 0
	assert.Equal(t, int64(12345), metrics.MessagesProcessed) // Success
}

func TestGetStatus_NilRepos(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := systemstatus.New(nil, nil, nil, &mockLogger{})

	metrics, err := svc.GetStatus(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations)
	assert.Equal(t, int32(0), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

func TestGetStatus_ClampToInt32(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	// Test values larger than MaxInt32
	bsRepo := &mockBSRepo{stats: &interfaces.BaseStationStatistics{OnlineCount: math.MaxInt64}}
	epRepo := &mockEPRepo{count: math.MaxInt64}
	msgRepo := &mockMsgRepo{stats: &mioty.MessageStats{TotalCount: math.MaxInt64}}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{})

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
	epRepo := &mockEPRepo{count: 42}

	svc := systemstatus.New(bsRepo, epRepo, msgRepo, &mockLogger{})

	metrics, err := svc.GetStatus(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, int32(0), metrics.ActiveBasestations) // nil stats
	assert.Equal(t, int32(42), metrics.ActiveEndpoints)
	assert.Equal(t, int64(0), metrics.MessagesProcessed) // nil stats
}
