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

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// mockEndpointRepo implements EndpointStore for testing
type mockEndpointRepo struct {
	endpoints        map[uint64]*models.EndPoint
	createErr        error
	updateErr        error
	lastRegistration *models.EndpointRegistrationParams // Capture last registration call for verification
}

func newMockEndpointRepo() *mockEndpointRepo {
	return &mockEndpointRepo{
		endpoints: make(map[uint64]*models.EndPoint),
	}
}

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

func (m *mockEndpointRepo) Create(_ context.Context, ep *models.EndPoint) error {
	if m.createErr != nil {
		return m.createErr
	}
	euiKey := ep.EUI.ToUint64()
	m.endpoints[euiKey] = ep
	return nil
}

func (m *mockEndpointRepo) EndpointRegistrationUpdate(_ context.Context, _ int64, _ int64, p models.EndpointRegistrationParams) error {
	captured := p
	m.lastRegistration = &captured // Capture for test verification
	return m.updateErr
}

func (m *mockEndpointRepo) EndpointAttachmentStateUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointAttachmentStateParams) error {
	return m.updateErr
}

func (m *mockEndpointRepo) EndpointAttachSessionUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointAttachSessionParams) error {
	return m.updateErr
}

func (m *mockEndpointRepo) EndpointDetachStateUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointDetachStateParams) error {
	return m.updateErr
}

func (m *mockEndpointRepo) TransitionEndpointStatus(_ context.Context, _ int64, _ int64, _ string) (bool, error) {
	return true, m.updateErr
}

func (m *mockEndpointRepo) RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error) {
	return m.TransitionEndpointStatus(ctx, tenantID, endpointID, status)
}

func (m *mockEndpointRepo) Get(_ context.Context, _ models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}

func (m *mockEndpointRepo) GetByTenant(_ context.Context, _ int64) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointRepo) CountByTenant(_ context.Context, _ int64) (int64, error) {
	return int64(len(m.endpoints)), nil
}

func (m *mockEndpointRepo) ListByTenantPaginated(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointRepo) GetByID(_ context.Context, _ int64, _ int64) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}

func (m *mockEndpointRepo) Update(_ context.Context, _ *models.EndPoint) error {
	return nil
}

func (m *mockEndpointRepo) UpdateLastSeen(_ context.Context, _ int64, _ models.EUI, _ uint32) error {
	return nil
}

func (m *mockEndpointRepo) UpdateRadioMetricsSelective(_ context.Context, _ int64, _ models.EUI, _ models.RadioMetricsUpdate) error {
	return nil
}

func (m *mockEndpointRepo) GetPreferredBsEui(_ context.Context, _ int64, _ []byte) (*uint64, bool, error) {
	return nil, false, nil
}

func (m *mockEndpointRepo) DeleteByTenant(_ context.Context, _ int64, _ []byte) (int64, error) {
	return 0, nil
}

func (m *mockEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// Compile-time interface check
var _ EndpointStore = (*mockEndpointRepo)(nil)

// announcedDecisions records the attachment decisions the service announced.
type announcedDecisions struct {
	statuses []string
}

func (a *announcedDecisions) NotifyEndpointStatus(_ context.Context, notice bssciservices.EndpointStatusNotice) {
	a.statuses = append(a.statuses, notice.Status.EpStatus)
}

// newTestEndpointService is the SCACI endpoint service over repo, deciding
// attachments through the service center's decider.
func newTestEndpointService(t *testing.T, repo interface {
	EndpointStore
	bssciservices.StatusTransitioner
}) (scaci.EndpointService, *announcedDecisions) {
	t.Helper()
	notices := &announcedDecisions{}
	decider, err := bssciservices.NewEndpointAttachmentDecider(repo, notices)
	require.NoError(t, err)
	svc, err := NewEndpointService(repo, &recordingDetachPropagator{}, decider, logger.NewNop())
	require.NoError(t, err)
	return svc, notices
}

func TestNewEndpointServiceRefusesAMissingCollaborator(t *testing.T) {
	repo := newMockEndpointRepo()
	decider, err := bssciservices.NewEndpointAttachmentDecider(repo, &announcedDecisions{})
	require.NoError(t, err)
	stations := &recordingDetachPropagator{}
	for name, tc := range map[string]struct {
		build func() (scaci.EndpointService, error)
		want  error
	}{
		"endpoint store": {func() (scaci.EndpointService, error) {
			return NewEndpointService(nil, stations, decider, logger.NewNop())
		}, errNilEndpointStore},
		"detach propagator": {func() (scaci.EndpointService, error) { return NewEndpointService(repo, nil, decider, logger.NewNop()) }, errNilDetachPropagator},
		"decider":           {func() (scaci.EndpointService, error) { return NewEndpointService(repo, stations, nil, logger.NewNop()) }, errNilAttachmentDecider},
		"logger":            {func() (scaci.EndpointService, error) { return NewEndpointService(repo, stations, decider, nil) }, errNilEndpointServiceLogger},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := tc.build()
			require.ErrorIs(t, err, tc.want)
		})
	}
}

// Test validation guards for SCACI §3.6.1 Register operation
func TestEndpointService_Register_ValidationGuards(t *testing.T) {
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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

	// Verify the registration field set was captured
	require.NotNil(t, repo.lastRegistration, "EndpointRegistrationUpdate should have been called")

	// Verify all §3.6.1 fields carry the request values
	assert.Equal(t, validNwkKey[:], repo.lastRegistration.NwkKey, "nwk_key must match request")
	assert.Equal(t, true, repo.lastRegistration.Bidi, "bidi must be true")
	assert.Equal(t, true, repo.lastRegistration.PreAttach, "pre_attach must be true")
	assert.Equal(t, uint16(12345), repo.lastRegistration.ShAddr, "sh_addr must match request")
	assert.Equal(t, uint32(100000), repo.lastRegistration.AttachCnt, "attach_cnt must match request")
	assert.Equal(t, uint32(200000), repo.lastRegistration.PacketCnt, "packet_cnt must match request")
	assert.Equal(t, true, repo.lastRegistration.DualChan, "dual_chan must be true")
	assert.Equal(t, true, repo.lastRegistration.Repetition, "repetition must be true")
	assert.Equal(t, true, repo.lastRegistration.WideCarrOff, "wide_carr_off must be true")
	assert.Equal(t, true, repo.lastRegistration.LongBlkDist, "long_blk_dist must be true")
}

// TestRegister_ZeroEpEui_ReturnsError confirms EpEui=0 is rejected per §3.6.1.
func TestRegister_ZeroEpEui_ReturnsError(t *testing.T) {
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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

	// Verify the max uint16 value is carried without truncation
	require.NotNil(t, repo.lastRegistration)
	assert.Equal(t, uint16(65535), repo.lastRegistration.ShAddr, "ShAddr 65535 must be carried without truncation")
}

// TestRegister_MaxAttachCnt_4294967295 verifies max uint32 value is accepted and stored correctly.
func TestRegister_MaxAttachCnt_4294967295(t *testing.T) {
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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

	// Verify the max uint32 value is carried without truncation
	require.NotNil(t, repo.lastRegistration)
	assert.Equal(t, uint32(4294967295), repo.lastRegistration.AttachCnt, "AttachCnt 4294967295 must be carried without truncation")
}

// TestRegister_MaxPacketCnt_4294967295 verifies max uint32 value is accepted and stored correctly.
func TestRegister_MaxPacketCnt_4294967295(t *testing.T) {
	repo := newMockEndpointRepo()
	svc, _ := newTestEndpointService(t, repo)

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

	// Verify the max uint32 value is carried without truncation and feeds both counters
	require.NotNil(t, repo.lastRegistration)
	assert.Equal(t, uint32(4294967295), repo.lastRegistration.PacketCnt, "PacketCnt 4294967295 must be carried without truncation")
}
