// Package grpc provides gRPC handler tests for endpoint stats and operations.
package grpc

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/registrationscope"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var (
	errEndpointNotFound = errors.New("endpoint not found")
)

// Endpoint stats and operations test fixtures.
const (
	testEventTypeEndpointAttached = "endpoint.attached"
	testEventTypeUplinkReceived   = "uplink.received"

	testTotalMessageCount  = 100
	testOperationsPageSize = 10
	testPaginationPageSize = 50
	testPageSizeUnset      = 0
	testPageSizeOverMax    = 500
)

// ============================================================================
// Mock Endpoint Stats Store
// ============================================================================

type mockEndpointStatsStore struct {
	getMessageStatsByEndpointFunc func(ctx context.Context, epEui uint64, tenantID int64, since time.Time) (*mioty.MessageStats, error)
}

func (m *mockEndpointStatsStore) GetMessageStatsByEndpointSince(ctx context.Context, epEui uint64, tenantID int64, since time.Time) (*mioty.MessageStats, error) {
	if m.getMessageStatsByEndpointFunc != nil {
		return m.getMessageStatsByEndpointFunc(ctx, epEui, tenantID, since)
	}
	return nil, nil
}

// uplinkLedger counts the stored uplinks of an EUI received at or after since.
type uplinkLedger struct {
	received []time.Time
}

func (l uplinkLedger) GetMessageStatsByEndpointSince(_ context.Context, _ uint64, _ int64, since time.Time) (*mioty.MessageStats, error) {
	stats := &mioty.MessageStats{}
	for _, at := range l.received {
		if !at.Before(since) {
			stats.TotalCount++
		}
	}
	return stats, nil
}

type registeredAt struct {
	at  time.Time
	err error
}

func (r registeredAt) RegisteredAt(context.Context, int64, []byte) (time.Time, error) {
	return r.at, r.err
}

// ============================================================================
// Mock Operation Status Adapter
// ============================================================================

type mockOperationStatusAdapter struct {
	getEndpointOperationsFunc func(ctx context.Context, endpointID, tenantID int64, limit, offset int) ([]models.SystemEvent, error)
}

func (m *mockOperationStatusAdapter) GetEndpointOperations(ctx context.Context, endpointID, tenantID int64, limit, offset int) ([]models.SystemEvent, error) {
	if m.getEndpointOperationsFunc != nil {
		return m.getEndpointOperationsFunc(ctx, endpointID, tenantID, limit, offset)
	}
	return nil, nil
}

// ============================================================================
// Mock Endpoint Service
// ============================================================================

// mockEndpointSvcForStats uses canonical models.EndPoint types
type mockEndpointSvcForStats struct {
	getByEUIFunc func(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error)
}

func (m *mockEndpointSvcForStats) Create(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForStats) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
	if m.getByEUIFunc != nil {
		return m.getByEUIFunc(ctx, eui, tenantID)
	}
	return nil, nil
}

func (m *mockEndpointSvcForStats) Update(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForStats) Delete(_ context.Context, _ []byte, _ int64) (int64, error) {
	return 0, nil
}

func (m *mockEndpointSvcForStats) ListByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForStats) List(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForStats) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointSvcForStats) CreateWithStatus(ctx context.Context, ep *models.EndPoint, _ string) (*models.EndPoint, error) {
	return m.Create(ctx, ep)
}

func (m *mockEndpointSvcForStats) CheckEUIGloballyUnique(_ context.Context, _ []byte) error {
	return nil
}

// ============================================================================
// Test Helpers
// ============================================================================

// Note: testLogger is defined in kilocenter_service_integrations_test.go

func createTestEndpoint() *models.EndPoint {
	return &models.EndPoint{
		ID:       123,
		EUI:      models.EUIFromString("0123456789ABCDEF"),
		TenantID: 1,
		Name:     "Test Endpoint",
		EpStatus: "attached", // Model has EpStatus, not AttachStatus
	}
}

func createTestMessageStats() *mioty.MessageStats {
	now := time.Now()
	activeDays := 7
	return &mioty.MessageStats{
		TotalCount:      testTotalMessageCount,
		UniqueEndpoints: 1,
		AvgRSSI:         -75.5,
		AvgSNR:          10.5,
		FirstSeen:       &now,
		LastSeen:        &now,
		ActiveDays:      &activeDays,
	}
}

func createTestOperations() []models.SystemEvent {
	return []models.SystemEvent{
		{
			ID:        "op-1",
			EventType: testEventTypeEndpointAttached,
			Category:  "endpoint",
			Severity:  "info",
			Title:     "Endpoint attached",
			CreatedAt: time.Now(),
		},
		{
			ID:        "op-2",
			EventType: testEventTypeUplinkReceived,
			Category:  "message",
			Severity:  "info",
			Title:     "Uplink received",
			CreatedAt: time.Now(),
		},
	}
}

func newTestServiceWithEndpointStats(
	endpointSvc grpcservices.EndpointService,
	statsStore EndpointStatsStore,
	opAdapter OperationStatusAdapter,
) *CoreService {
	return newTestServiceWithRegistration(endpointSvc, statsStore, opAdapter, registeredAt{at: time.Unix(0, 0)})
}

func newTestServiceWithRegistration(
	endpointSvc grpcservices.EndpointService,
	statsStore EndpointStatsStore,
	opAdapter OperationStatusAdapter,
	registrations registrationscope.Registrations,
) *CoreService {
	return testCoreService(coreFields{
		log:                testLogger{},
		endpointSvc:        endpointSvc,
		endpointStatsStore: statsStore,
		registrationWindow: registrationscope.New(registrations),
		opStatusAdapter:    opAdapter,
	})
}

// ============================================================================
// GetEndPointStats Tests
// ============================================================================

func TestGetEndPointStats_Success(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	endpoint := createTestEndpoint()
	stats := createTestMessageStats()

	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return endpoint, nil
		},
	}

	statsStore := &mockEndpointStatsStore{
		getMessageStatsByEndpointFunc: func(_ context.Context, _ uint64, _ int64, _ time.Time) (*mioty.MessageStats, error) {
			return stats, nil
		},
	}

	svc := newTestServiceWithEndpointStats(endpointSvc, statsStore, nil)

	resp, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{
		EpEui: "01-23-45-67-89-AB-CD-EF",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "01-23-45-67-89-AB-CD-EF", resp.EpEui)
	assert.Equal(t, int64(testTotalMessageCount), resp.TotalCount)
	assert.Equal(t, -75.5, resp.AvgRssi)
	assert.Equal(t, 10.5, resp.AvgSnr)
	assert.Equal(t, "attached", resp.AttachStatus)
}

func TestGetEndPointStats_RefusesActiveDaysBeyondInt32(t *testing.T) {
	stats := createTestMessageStats()
	activeDays := math.MaxInt32 + 1
	stats.ActiveDays = &activeDays
	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return createTestEndpoint(), nil
		},
	}
	statsStore := &mockEndpointStatsStore{
		getMessageStatsByEndpointFunc: func(_ context.Context, _ uint64, _ int64, _ time.Time) (*mioty.MessageStats, error) {
			return stats, nil
		},
	}
	svc := newTestServiceWithEndpointStats(endpointSvc, statsStore, nil)

	_, err := svc.GetEndPointStats(testutil.TestContextWithTenant(1), &pb.GetEndPointStatsRequest{EpEui: "01-23-45-67-89-AB-CD-EF"})
	require.Error(t, err)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow), status.Code(err))
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow), status.Convert(err).Message())
}

func TestGetEndPointStats_ReRegisteredEUICountsOnlyTheCurrentRegistration(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)
	reRegistered := time.Date(2026, 9, 28, 12, 47, 16, 0, time.UTC)
	ledger := uplinkLedger{received: []time.Time{
		reRegistered.Add(-time.Hour),
		reRegistered.Add(time.Minute),
	}}
	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return createTestEndpoint(), nil
		},
	}

	svc := newTestServiceWithRegistration(endpointSvc, ledger, nil, registeredAt{at: reRegistered})
	resp, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{EpEui: "01-23-45-67-89-AB-CD-EF"})

	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.TotalCount, "the previous registration's uplink is not counted")
}

func TestGetEndPointStats_OfAnUnregisteredEUIIsNotFound(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithRegistration(&mockEndpointSvcForStats{}, uplinkLedger{}, nil, registeredAt{err: storage.ErrNotFound})
	_, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{EpEui: "01-23-45-67-89-AB-CD-EF"})

	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetEndPointStats_AFailedRegistrationReadFailsTheCard(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithRegistration(&mockEndpointSvcForStats{}, uplinkLedger{}, nil, registeredAt{err: errEndpointNotFound})
	_, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{EpEui: "01-23-45-67-89-AB-CD-EF"})

	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetEndPointStats_MissingEpEui(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithEndpointStats(nil, &mockEndpointStatsStore{}, nil)

	resp, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{
		EpEui: "",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
}

func TestGetEndPointStats_InvalidEUIFormat(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithEndpointStats(nil, &mockEndpointStatsStore{}, nil)

	resp, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{
		EpEui: "invalid-eui-format",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
}

func TestGetEndPointStats_EndpointNotFound(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return nil, errEndpointNotFound
		},
	}

	statsStore := &mockEndpointStatsStore{
		getMessageStatsByEndpointFunc: func(_ context.Context, _ uint64, _ int64, _ time.Time) (*mioty.MessageStats, error) {
			return createTestMessageStats(), nil
		},
	}

	svc := newTestServiceWithEndpointStats(endpointSvc, statsStore, nil)

	resp, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{
		EpEui: "01-23-45-67-89-AB-CD-EF",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetEndPointStats_ServiceNotConfigured(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithEndpointStats(nil, nil, nil)

	resp, err := svc.GetEndPointStats(ctx, &pb.GetEndPointStatsRequest{
		EpEui: "01-23-45-67-89-AB-CD-EF",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
}

// ============================================================================
// GetEndPointOperations Tests
// ============================================================================

func TestGetEndPointOperations_Success(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	endpoint := createTestEndpoint()
	operations := createTestOperations()

	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return endpoint, nil
		},
	}

	opAdapter := &mockOperationStatusAdapter{
		getEndpointOperationsFunc: func(_ context.Context, endpointID, _ int64, _, _ int) ([]models.SystemEvent, error) {
			assert.Equal(t, int64(123), endpointID) // Should use endpoint.ID, not EUI
			return operations, nil
		},
	}

	svc := newTestServiceWithEndpointStats(endpointSvc, nil, opAdapter)

	resp, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui:    "01-23-45-67-89-AB-CD-EF",
		PageSize: testOperationsPageSize,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Operations, 2)
	assert.Equal(t, "op-1", resp.Operations[0].Id)
	assert.Equal(t, testEventTypeEndpointAttached, resp.Operations[0].EventType)
}

func TestGetEndPointOperations_MissingEpEui(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithEndpointStats(nil, nil, &mockOperationStatusAdapter{})

	resp, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui: "",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
}

func TestGetEndPointOperations_InvalidEUIFormat(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithEndpointStats(nil, nil, &mockOperationStatusAdapter{})

	resp, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui: "not-a-valid-hex-eui",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
}

func TestGetEndPointOperations_EndpointNotFound(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return nil, errEndpointNotFound
		},
	}

	svc := newTestServiceWithEndpointStats(endpointSvc, nil, &mockOperationStatusAdapter{})

	resp, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui: "01-23-45-67-89-AB-CD-EF",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
}

func TestGetEndPointOperations_ServiceNotConfigured(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	svc := newTestServiceWithEndpointStats(nil, nil, nil)

	resp, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui: "01-23-45-67-89-AB-CD-EF",
	})

	require.Error(t, err)
	require.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
}

func TestGetEndPointOperations_Pagination(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	endpoint := createTestEndpoint()

	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return endpoint, nil
		},
	}

	var capturedLimit, capturedOffset int
	opAdapter := &mockOperationStatusAdapter{
		getEndpointOperationsFunc: func(_ context.Context, _, _ int64, limit, offset int) ([]models.SystemEvent, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []models.SystemEvent{}, nil
		},
	}

	svc := newTestServiceWithEndpointStats(endpointSvc, nil, opAdapter)

	_, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui:    "01-23-45-67-89-AB-CD-EF",
		PageSize: testPaginationPageSize,
		Offset:   10,
	})

	require.NoError(t, err)
	assert.Equal(t, 50, capturedLimit)
	assert.Equal(t, 10, capturedOffset)
}

func TestGetEndPointOperations_DefaultPagination(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	endpoint := createTestEndpoint()

	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return endpoint, nil
		},
	}

	var capturedLimit int
	opAdapter := &mockOperationStatusAdapter{
		getEndpointOperationsFunc: func(_ context.Context, _, _ int64, limit, _ int) ([]models.SystemEvent, error) {
			capturedLimit = limit
			return []models.SystemEvent{}, nil
		},
	}

	svc := newTestServiceWithEndpointStats(endpointSvc, nil, opAdapter)

	_, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui:    "01-23-45-67-89-AB-CD-EF",
		PageSize: testPageSizeUnset, // Should default to DefaultPageSize
	})

	require.NoError(t, err)
	assert.Equal(t, 20, capturedLimit)
}

func TestGetEndPointOperations_MaxPagination(t *testing.T) {
	ctx := testutil.TestContextWithTenant(1)

	endpoint := createTestEndpoint()

	endpointSvc := &mockEndpointSvcForStats{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
			return endpoint, nil
		},
	}

	var capturedLimit int
	opAdapter := &mockOperationStatusAdapter{
		getEndpointOperationsFunc: func(_ context.Context, _, _ int64, limit, _ int) ([]models.SystemEvent, error) {
			capturedLimit = limit
			return []models.SystemEvent{}, nil
		},
	}

	svc := newTestServiceWithEndpointStats(endpointSvc, nil, opAdapter)

	_, err := svc.GetEndPointOperations(ctx, &pb.GetEndPointOperationsRequest{
		EpEui:    "01-23-45-67-89-AB-CD-EF",
		PageSize: testPageSizeOverMax, // Should be capped to MaxPageSize
	})

	require.NoError(t, err)
	assert.Equal(t, MaxPageSize, capturedLimit)
}
