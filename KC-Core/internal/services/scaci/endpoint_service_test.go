// Package scaciservices provides SCACI endpoint service tests.
//
// Tests verify unsigned integer validation per SCACI §3.6.1:
//   - shAddr: uint16 (0-65535)
//   - attachCnt: uint32 (0-4294967295)
//   - packetCnt: uint32 (0-4294967295)
package scaciservices

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// mockEndpointRepo implements interfaces.EndpointRepository for testing
type mockEndpointRepo struct {
	endpoints   map[uint64]*models.EndPoint
	createErr   error
	updateErr   error
	lastUpdates map[string]interface{} // Capture last UpdateFields call for verification
}

func newMockEndpointRepo() *mockEndpointRepo {
	return &mockEndpointRepo{
		endpoints: make(map[uint64]*models.EndPoint),
	}
}

// GetByEUI implements interfaces.EndpointRepository.GetByEUI
func (m *mockEndpointRepo) GetByEUI(_ context.Context, _ int64, eui []byte) (*models.EndPoint, error) {
	if len(eui) != 8 {
		return nil, storage.ErrNotFound
	}
	var euiKey uint64
	for i := 0; i < 8; i++ {
		euiKey = (euiKey << 8) | uint64(eui[i])
	}
	if ep, ok := m.endpoints[euiKey]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

// Create implements interfaces.EndpointRepository.Create
func (m *mockEndpointRepo) Create(_ context.Context, ep *models.EndPoint) error {
	if m.createErr != nil {
		return m.createErr
	}
	euiKey := ep.EUI.ToUint64()
	m.endpoints[euiKey] = ep
	return nil
}

// UpdateFields implements interfaces.EndpointRepository.UpdateFields
func (m *mockEndpointRepo) UpdateFields(_ context.Context, _ int64, _ int64, updates map[string]interface{}) error {
	m.lastUpdates = updates // Capture for test verification
	return m.updateErr
}

// Get implements interfaces.EndpointRepository.Get
func (m *mockEndpointRepo) Get(_ context.Context, _ models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}

// GetByTenant implements interfaces.EndpointRepository.GetByTenant
func (m *mockEndpointRepo) GetByTenant(_ context.Context, _ int64) ([]*models.EndPoint, error) {
	return nil, nil
}

// CountByTenant implements interfaces.EndpointRepository.CountByTenant
func (m *mockEndpointRepo) CountByTenant(_ context.Context, _ int64) (int64, error) {
	return int64(len(m.endpoints)), nil
}

// ListByTenantPaginated implements interfaces.EndpointRepository.ListByTenantPaginated
func (m *mockEndpointRepo) ListByTenantPaginated(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}

// GetByID implements interfaces.EndpointRepository.GetByID
func (m *mockEndpointRepo) GetByID(_ context.Context, _ int64, _ int64) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}

// Update implements interfaces.EndpointRepository.Update
func (m *mockEndpointRepo) Update(_ context.Context, _ *models.EndPoint) error {
	return nil
}

// UpdateLastSeen implements interfaces.EndpointRepository.UpdateLastSeen
func (m *mockEndpointRepo) UpdateLastSeen(_ context.Context, _ int64, _ models.EUI, _ uint32) error {
	return nil
}

// UpdateRadioMetricsSelective implements interfaces.EndpointRepository.UpdateRadioMetricsSelective
func (m *mockEndpointRepo) UpdateRadioMetricsSelective(_ context.Context, _ int64, _ models.EUI, _ interfaces.RadioMetricsUpdate) error {
	return nil
}

// GetEndpointWithKeysForDetachValidation implements interfaces.EndpointRepository.GetEndpointWithKeysForDetachValidation
func (m *mockEndpointRepo) GetEndpointWithKeysForDetachValidation(_ context.Context, _ models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}

// UpdateDetachMetrics implements interfaces.EndpointRepository.UpdateDetachMetrics
func (m *mockEndpointRepo) UpdateDetachMetrics(_ context.Context, _ int64, _ models.EUI, _ interfaces.DetachMetricsUpdate) error {
	return nil
}

// UpdateRadioMetrics implements interfaces.EndpointRepository.UpdateRadioMetrics
func (m *mockEndpointRepo) UpdateRadioMetrics(_ context.Context, _ int64, _ models.EUI, _, _, _ float64, _, _ int64, _ string) error {
	return nil
}

// StreamAllForPropagation implements interfaces.EndpointRepository.StreamAllForPropagation
func (m *mockEndpointRepo) StreamAllForPropagation(_ context.Context, _ int64, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}

// HasEndpointsSince implements interfaces.EndpointRepository.HasEndpointsSince
func (m *mockEndpointRepo) HasEndpointsSince(_ context.Context, _ time.Time) (bool, error) {
	return false, nil
}

// GetPreferredBsEui implements interfaces.EndpointRepository.GetPreferredBsEui
func (m *mockEndpointRepo) GetPreferredBsEui(_ context.Context, _ int64, _ []byte) (*uint64, bool, error) {
	return nil, false, nil
}

// DeleteByTenant implements interfaces.EndpointRepository.DeleteByTenant
func (m *mockEndpointRepo) DeleteByTenant(_ context.Context, _ int64, _ []byte) error {
	return nil
}

// UpdateWithEUI implements interfaces.EndpointRepository.UpdateWithEUI
func (m *mockEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

// CheckEUIUnique implements interfaces.EndpointRepository.CheckEUIUnique
func (m *mockEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// Compile-time interface check
var _ interfaces.EndpointRepository = (*mockEndpointRepo)(nil)

// Test validation guards for SCACI §3.6.1 Register operation
func TestEndpointService_Register_ValidationGuards(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	validNwkKey := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	tests := []struct {
		name      string
		req       *scaci.Register
		wantErr   string
		wantNoErr bool
	}{
		{
			name: "missing EpEui returns error",
			req: &scaci.Register{
				EpEui:  0,
				NwkKey: validNwkKey,
			},
			wantErr: scaci.ErrMissingEpEui,
		},
		{
			name: "valid request succeeds",
			req: &scaci.Register{
				EpEui:     0x0102030405060709,
				NwkKey:    validNwkKey,
				ShAddr:    1234,    // uint16, valid
				AttachCnt: 1000000, // uint32, valid
				PacketCnt: 2000000, // uint32, valid
			},
			wantNoErr: true,
		},
		{
			name: "max uint16 shAddr succeeds",
			req: &scaci.Register{
				EpEui:  0x0102030405060710,
				NwkKey: validNwkKey,
				ShAddr: 65535, // Max uint16
			},
			wantNoErr: true,
		},
		{
			name: "max uint32 attachCnt succeeds",
			req: &scaci.Register{
				EpEui:     0x0102030405060711,
				NwkKey:    validNwkKey,
				AttachCnt: 4294967295, // Max uint32
			},
			wantNoErr: true,
		},
		{
			name: "max uint32 packetCnt succeeds",
			req: &scaci.Register{
				EpEui:     0x0102030405060712,
				NwkKey:    validNwkKey,
				PacketCnt: 4294967295, // Max uint32
			},
			wantNoErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errToken := svc.Register(testutil.TestContext(), tt.req, 1)

			if tt.wantNoErr {
				assert.Empty(t, errToken, "expected no error token")
			} else {
				assert.Equal(t, tt.wantErr, errToken, "expected specific error token")
			}
		})
	}
}

// Test that Register correctly stores unsigned values
func TestEndpointService_Register_StoresUnsignedValues(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	validNwkKey := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	req := &scaci.Register{
		EpEui:     0x1122334455667788,
		NwkKey:    validNwkKey,
		ShAddr:    65535,      // Max uint16
		AttachCnt: 4294967295, // Max uint32
		PacketCnt: 4294967295, // Max uint32
	}

	errToken := svc.Register(testutil.TestContext(), req, 1)
	assert.Empty(t, errToken, "Register should succeed with max values")

	// Verify the endpoint was created with correct values
	ep, ok := repo.endpoints[req.EpEui]
	assert.True(t, ok, "Endpoint should be stored in repository")
	if ok {
		assert.NotNil(t, ep, "Endpoint should not be nil")
	}
}

// Test Deregister operation
func TestEndpointService_Deregister_ValidationGuards(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	tests := []struct {
		name    string
		epEui   uint64
		wantErr string
	}{
		{
			name:    "missing EpEui returns error",
			epEui:   0,
			wantErr: scaci.ErrMissingEpEui,
		},
		{
			name:    "nonexistent endpoint returns not found",
			epEui:   0x1122334455667788,
			wantErr: scaci.ErrEndpointNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errToken := svc.Deregister(testutil.TestContext(), tt.epEui, 1)
			assert.Equal(t, tt.wantErr, errToken)
		})
	}
}

// Test GetByEUI operation
func TestEndpointService_GetByEUI_ValidationGuards(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	tests := []struct {
		name      string
		eui       []byte
		wantErr   string
		wantEpNil bool
	}{
		{
			name:      "invalid EUI length returns error",
			eui:       []byte{0x01, 0x02, 0x03}, // Only 3 bytes
			wantErr:   scaci.ErrMissingEpEui,
			wantEpNil: true,
		},
		{
			name:      "nonexistent endpoint returns not found",
			eui:       []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			wantErr:   scaci.ErrEndpointNotFound,
			wantEpNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep, errToken := svc.GetByEUI(testutil.TestContext(), 1, tt.eui)

			assert.Equal(t, tt.wantErr, errToken)
			if tt.wantEpNil {
				assert.Nil(t, ep)
			}
		})
	}
}

// TestRegister_AllFields_PersistsCorrectly verifies all §3.6.1 fields are persisted with exact DB column names.
// This is a regression guard for the endpoint_service field mapping.
func TestRegister_AllFields_PersistsCorrectly(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	validNwkKey := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	req := &scaci.Register{
		EpEui:       0x1122334455667788,
		NwkKey:      validNwkKey,
		Bidi:        true,
		PreAttach:   true,
		ShAddr:      12345,
		AttachCnt:   100000,
		PacketCnt:   200000,
		DualChan:    true,
		Repetition:  true, // bool per SCACI §3.6.1
		WideCarrOff: true,
		LongBlkDist: true,
	}

	errToken := svc.Register(testutil.TestContext(), req, 1)
	assert.Empty(t, errToken, "Register should succeed with all fields")

	// Verify UpdateFields was called with exact DB column names
	assert.NotNil(t, repo.lastUpdates, "UpdateFields should have been called")

	// Verify all §3.6.1 fields are present with correct column names
	expectedColumns := []string{
		"nwk_key",
		"pre_attach",
		"bidi",
		"sh_addr",
		"attach_cnt",
		"packet_cnt",
		"last_packet_cnt",
		"dual_chan",
		"repetition",
		"wide_carr_off",
		"long_blk_dist",
	}
	for _, col := range expectedColumns {
		_, exists := repo.lastUpdates[col]
		assert.True(t, exists, "UpdateFields must include column: %s", col)
	}

	// Verify specific values are persisted correctly
	assert.Equal(t, validNwkKey[:], repo.lastUpdates["nwk_key"], "nwk_key must match request")
	assert.Equal(t, true, repo.lastUpdates["bidi"], "bidi must be true")
	assert.Equal(t, true, repo.lastUpdates["pre_attach"], "pre_attach must be true")
	assert.Equal(t, int32(12345), repo.lastUpdates["sh_addr"], "sh_addr must be cast to int32")
	assert.Equal(t, int64(100000), repo.lastUpdates["attach_cnt"], "attach_cnt must be cast to int64")
	assert.Equal(t, int64(200000), repo.lastUpdates["packet_cnt"], "packet_cnt must be cast to int64")
	assert.Equal(t, int64(200000), repo.lastUpdates["last_packet_cnt"], "last_packet_cnt must match packet_cnt")
	assert.Equal(t, true, repo.lastUpdates["dual_chan"], "dual_chan must be true")
	assert.Equal(t, true, repo.lastUpdates["repetition"], "repetition must be true")
	assert.Equal(t, true, repo.lastUpdates["wide_carr_off"], "wide_carr_off must be true")
	assert.Equal(t, true, repo.lastUpdates["long_blk_dist"], "long_blk_dist must be true")
}

// TestRegister_ZeroEpEui_ReturnsError confirms EpEui=0 is rejected per §3.6.1.
func TestRegister_ZeroEpEui_ReturnsError(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	validNwkKey := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	req := &scaci.Register{
		EpEui:  0, // Zero EpEui - invalid per §3.6.1
		NwkKey: validNwkKey,
	}

	errToken := svc.Register(testutil.TestContext(), req, 1)
	assert.Equal(t, scaci.ErrMissingEpEui, errToken, "Zero EpEui must return ErrMissingEpEui")
}

// TestRegister_MaxShAddr_65535 verifies max uint16 value is accepted and stored correctly.
func TestRegister_MaxShAddr_65535(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	validNwkKey := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	req := &scaci.Register{
		EpEui:  0x1122334455667790,
		NwkKey: validNwkKey,
		ShAddr: 65535, // Max uint16
	}

	errToken := svc.Register(testutil.TestContext(), req, 1)
	assert.Empty(t, errToken, "Max uint16 ShAddr should succeed")

	// Verify stored as int32(65535) without truncation
	assert.NotNil(t, repo.lastUpdates)
	assert.Equal(t, int32(65535), repo.lastUpdates["sh_addr"], "ShAddr 65535 must be stored as int32(65535)")
}

// TestRegister_MaxAttachCnt_4294967295 verifies max uint32 value is accepted and stored correctly.
func TestRegister_MaxAttachCnt_4294967295(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	validNwkKey := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	req := &scaci.Register{
		EpEui:     0x1122334455667791,
		NwkKey:    validNwkKey,
		AttachCnt: 4294967295, // Max uint32
	}

	errToken := svc.Register(testutil.TestContext(), req, 1)
	assert.Empty(t, errToken, "Max uint32 AttachCnt should succeed")

	// Verify stored as int64(4294967295) without truncation
	assert.NotNil(t, repo.lastUpdates)
	assert.Equal(t, int64(4294967295), repo.lastUpdates["attach_cnt"], "AttachCnt 4294967295 must be stored as int64(4294967295)")
}

// TestRegister_MaxPacketCnt_4294967295 verifies max uint32 value is accepted and stored correctly.
func TestRegister_MaxPacketCnt_4294967295(t *testing.T) {
	log := logger.NewNop()
	repo := newMockEndpointRepo()
	svc := NewEndpointService(repo, nil, log)

	validNwkKey := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	req := &scaci.Register{
		EpEui:     0x1122334455667792,
		NwkKey:    validNwkKey,
		PacketCnt: 4294967295, // Max uint32
	}

	errToken := svc.Register(testutil.TestContext(), req, 1)
	assert.Empty(t, errToken, "Max uint32 PacketCnt should succeed")

	// Verify stored as int64(4294967295) without truncation
	assert.NotNil(t, repo.lastUpdates)
	assert.Equal(t, int64(4294967295), repo.lastUpdates["packet_cnt"], "PacketCnt 4294967295 must be stored as int64(4294967295)")
	assert.Equal(t, int64(4294967295), repo.lastUpdates["last_packet_cnt"], "last_packet_cnt must also be 4294967295")
}
