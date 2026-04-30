package roaming

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kilocenter/KC-DB/storage/models"
)

// Mock for EndpointOwnershipResolver
type MockOwnershipResolver struct {
	mock.Mock
}

func (m *MockOwnershipResolver) GetEndpointOwner(ctx context.Context, epEui []byte) (int64, error) {
	args := m.Called(ctx, epEui)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOwnershipResolver) GetEndpointWithOwnership(ctx context.Context, epEui []byte, servingTenantID int64) (*models.EndPoint, error) {
	args := m.Called(ctx, epEui, servingTenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EndPoint), args.Error(1)
}

func (m *MockOwnershipResolver) IsRoamingEnabled(ctx context.Context, tenantID int64) (bool, error) {
	args := m.Called(ctx, tenantID)
	return args.Bool(0), args.Error(1)
}

func (m *MockOwnershipResolver) AreTenantsPartners(ctx context.Context, tenant1, tenant2 int64) (bool, error) {
	args := m.Called(ctx, tenant1, tenant2)
	return args.Bool(0), args.Error(1)
}

// Mock for EventRecorder
type MockEventRecorder struct {
	mock.Mock
}

func (m *MockEventRecorder) RecordRoamingEvent(ctx context.Context, event *models.RoamingEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestDetector_DetectRoaming(t *testing.T) {
	tests := []struct {
		name            string
		epEui           []byte
		servingTenantID int64
		ownerTenantID   int64
		setupMocks      func(*MockOwnershipResolver)
		wantIsRoaming   bool
		wantOwnerTenant int64
		wantErr         bool
	}{
		{
			name:            "endpoint not roaming - same tenant",
			epEui:           []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			servingTenantID: 1,
			ownerTenantID:   1,
			setupMocks: func(m *MockOwnershipResolver) {
				m.On("GetEndpointOwner", mock.Anything, mock.Anything).Return(int64(1), nil)
			},
			wantIsRoaming:   false,
			wantOwnerTenant: 1,
			wantErr:         false,
		},
		{
			name:            "endpoint roaming - different tenant",
			epEui:           []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			servingTenantID: 2,
			ownerTenantID:   1,
			setupMocks: func(m *MockOwnershipResolver) {
				m.On("GetEndpointOwner", mock.Anything, mock.Anything).Return(int64(1), nil)
			},
			wantIsRoaming:   true,
			wantOwnerTenant: 1,
			wantErr:         false,
		},
		{
			name:            "endpoint not found - returns ErrEndpointNotFound",
			epEui:           []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			servingTenantID: 3,
			setupMocks: func(m *MockOwnershipResolver) {
				m.On("GetEndpointOwner", mock.Anything, mock.Anything).Return(int64(0), ErrEndpointNotFound)
			},
			wantIsRoaming:   false,
			wantOwnerTenant: 0,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResolver := new(MockOwnershipResolver)
			mockRecorder := new(MockEventRecorder)
			tt.setupMocks(mockResolver)

			config := &DetectorConfig{
				CacheEnabled: false, // Disable cache for deterministic tests
			}

			detector := NewDetector(config, mockResolver, mockRecorder)

			isRoaming, ownerTenantID, err := detector.DetectRoaming(
				context.Background(),
				tt.epEui,
				tt.servingTenantID,
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantIsRoaming, isRoaming)
				assert.Equal(t, tt.wantOwnerTenant, ownerTenantID)
			}

			mockResolver.AssertExpectations(t)
		})
	}
}

func TestDetector_ValidateRoamingAllowed(t *testing.T) {
	tests := []struct {
		name            string
		ownerTenantID   int64
		servingTenantID int64
		arePartners     bool
		wantErr         bool
		errorMsg        string
	}{
		{
			name:            "valid roaming - partners",
			ownerTenantID:   1,
			servingTenantID: 2,
			arePartners:     true,
			wantErr:         false,
		},
		{
			name:            "invalid roaming - not partners",
			ownerTenantID:   1,
			servingTenantID: 3,
			arePartners:     false,
			wantErr:         true,
			errorMsg:        "no roaming agreement between tenants",
		},
		{
			name:            "not roaming - same tenant",
			ownerTenantID:   1,
			servingTenantID: 1,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResolver := new(MockOwnershipResolver)
			mockRecorder := new(MockEventRecorder)

			if tt.ownerTenantID != tt.servingTenantID {
				// Mock IsRoamingEnabled for both tenants
				mockResolver.On("IsRoamingEnabled", mock.Anything, tt.ownerTenantID).Return(true, nil)
				mockResolver.On("IsRoamingEnabled", mock.Anything, tt.servingTenantID).Return(true, nil)
				// Mock AreTenantsPartners
				mockResolver.On("AreTenantsPartners", mock.Anything, tt.ownerTenantID, tt.servingTenantID).
					Return(tt.arePartners, nil)
			}

			config := &DetectorConfig{}
			detector := NewDetector(config, mockResolver, mockRecorder)

			err := detector.ValidateRoamingAllowed(context.Background(), tt.ownerTenantID, tt.servingTenantID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockResolver.AssertExpectations(t)
		})
	}
}

func TestDetector_RecordEvents(t *testing.T) {
	tests := []struct {
		name      string
		isAttach  bool
		epEui     []byte
		setupMock func(*MockEventRecorder)
		wantErr   bool
	}{
		{
			name:     "record attach event",
			isAttach: true,
			epEui:    []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			setupMock: func(m *MockEventRecorder) {
				m.On("RecordRoamingEvent", mock.Anything, mock.MatchedBy(func(e *models.RoamingEvent) bool {
					return e.EventType == models.RoamingEventAttach
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "record detach event",
			isAttach: false,
			epEui:    []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
			setupMock: func(m *MockEventRecorder) {
				m.On("RecordRoamingEvent", mock.Anything, mock.MatchedBy(func(e *models.RoamingEvent) bool {
					return e.EventType == models.RoamingEventDetach
				})).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResolver := new(MockOwnershipResolver)
			mockRecorder := new(MockEventRecorder)
			tt.setupMock(mockRecorder)

			config := &DetectorConfig{
				EnableAuditTrail: true,
			}
			detector := NewDetector(config, mockResolver, mockRecorder)

			var err error
			if tt.isAttach {
				err = detector.RecordAttachEvent(context.Background(), tt.epEui, 1, 2, []byte{0xAA, 0xBB})
			} else {
				err = detector.RecordDetachEvent(context.Background(), tt.epEui, 1, 2, []byte{0xCC, 0xDD})
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRecorder.AssertExpectations(t)
		})
	}
}

func TestDetector_CacheIntegration(t *testing.T) {
	mockResolver := new(MockOwnershipResolver)
	mockRecorder := new(MockEventRecorder)

	// Setup mock to be called only once
	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	mockResolver.On("GetEndpointOwner", mock.Anything, epEui).Return(int64(1), nil).Once()

	config := &DetectorConfig{
		CacheEnabled: true,
		CacheTTL:     100 * time.Millisecond,
		CacheMaxSize: 100,
	}
	detector := NewDetector(config, mockResolver, mockRecorder)

	ctx := context.Background()

	// First call - should hit database
	isRoaming1, owner1, err1 := detector.DetectRoaming(ctx, epEui, 2)
	require.NoError(t, err1)
	assert.True(t, isRoaming1)
	assert.Equal(t, int64(1), owner1)

	// Second call - should hit cache
	isRoaming2, owner2, err2 := detector.DetectRoaming(ctx, epEui, 2)
	require.NoError(t, err2)
	assert.True(t, isRoaming2)
	assert.Equal(t, int64(1), owner2)

	// Mock should only be called once due to caching
	mockResolver.AssertExpectations(t)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Setup mock for another call after expiry
	mockResolver.On("GetEndpointOwner", mock.Anything, epEui).Return(int64(1), nil).Once()

	// Third call - cache expired, should hit database again
	isRoaming3, owner3, err3 := detector.DetectRoaming(ctx, epEui, 2)
	require.NoError(t, err3)
	assert.True(t, isRoaming3)
	assert.Equal(t, int64(1), owner3)

	mockResolver.AssertExpectations(t)
}

func TestDetector_Metrics(t *testing.T) {
	mockResolver := new(MockOwnershipResolver)
	mockRecorder := new(MockEventRecorder)

	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	mockResolver.On("GetEndpointOwner", mock.Anything, epEui).Return(int64(1), nil)

	config := &DetectorConfig{
		EnableMetrics: true,
		CacheEnabled:  true, // Enable cache to track cache misses
	}
	detector := NewDetector(config, mockResolver, mockRecorder)

	// Perform detection
	_, _, err := detector.DetectRoaming(context.Background(), epEui, 2)
	require.NoError(t, err)

	// Get metrics
	metrics := detector.GetMetrics()
	assert.Equal(t, uint64(1), metrics.GetRoamingDetected())
	assert.Equal(t, uint64(0), metrics.GetCacheHits())
	assert.Equal(t, uint64(1), metrics.GetCacheMisses())
}
