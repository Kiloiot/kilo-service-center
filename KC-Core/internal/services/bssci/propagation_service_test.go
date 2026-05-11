package bssciservices

import (
	"context"
	"testing"
	"time"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantFilteringInTriggerEndpointPropagate verifies ATT-02: endpoint propagation
// filters sessions by tenant and only sends to same-tenant sessions.
func TestTenantFilteringInTriggerEndpointPropagate(t *testing.T) {
	t.Parallel()

	const (
		endpointTenant = int64(100)
		sessionTenant1 = int64(100) // Same tenant
		sessionTenant2 = int64(200) // Different tenant (should be filtered)
	)

	endpoint := &models.EndPoint{
		ID:       1001,
		TenantID: endpointTenant,
		Bidi:     true,
	}

	env := newPropagationTestEnv(t, endpoint)

	// ATT-02: Sessions from different tenants
	sessions := []propagation.BaseStationSession{
		{
			ID:             "session-1",
			BaseStationEUI: 0x0011223344556677,
			TenantID:       sessionTenant1, // Same tenant - should receive
		},
		{
			ID:             "session-2",
			BaseStationEUI: 0x8899AABBCCDDEEFF,
			TenantID:       sessionTenant2, // Different tenant - should be filtered
		},
	}

	// ATT-02: Add tenant context (required by propagation service)
	ctx := pkgcontext.WithTenantID(context.Background(), endpointTenant)
	err := env.service.TriggerEndpointPropagate(ctx, endpoint.ID, sessions)

	require.NoError(t, err, "propagation should succeed")

	// ATT-02: Verify only same-tenant session received propagate
	require.Len(t, env.sender.calls, 1, "only one session should receive propagate")
	assert.Equal(t, "session-1", env.sender.calls[0].sessionID, "same-tenant session should receive")
}

// TestPropagateStatusFilteringInReconcile verifies BSSCI §3.8: reconciliation propagates
// ALL endpoints to every base station. attPrp is idempotent, so endpoints with any
// PropagateStatus (including "attached") are still propagated to support multi-BS deployments.
// Both bidirectional AND unidirectional endpoints need attPrp per spec.
func TestPropagateStatusFilteringInReconcile(t *testing.T) {
	t.Parallel()

	const tenantID = int64(100)

	// Endpoints with different propagate statuses
	attachedStatus := pkgbssci.PropagateStatusAttached
	receivedStatus := pkgbssci.PropagateStatusAttachReceived

	endpoints := []*models.EndPoint{
		{
			ID:              1001,
			TenantID:        tenantID,
			Bidi:            true,
			PropagateStatus: nil, // New endpoint - should be propagated
		},
		{
			ID:              1002,
			TenantID:        tenantID,
			Bidi:            true,
			PropagateStatus: &receivedStatus, // In progress - should be propagated
		},
		{
			ID:              1003,
			TenantID:        tenantID,
			Bidi:            true,
			PropagateStatus: &attachedStatus, // Already attached - STILL propagated (idempotent attPrp)
		},
		{
			ID:       1004,
			TenantID: tenantID,
			Bidi:     false, // Unidirectional - SHOULD be propagated per BSSCI §3.8
		},
	}

	env := newPropagationTestEnv(t, endpoints...)

	session := propagation.BaseStationSession{
		ID:             "session-1",
		BaseStationEUI: 0x0011223344556677,
		TenantID:       tenantID,
	}

	ctx := context.Background()
	err := env.service.ReconcileBaseStation(ctx, session, nil)

	require.NoError(t, err, "reconciliation should succeed")

	// BSSCI §3.8: ALL endpoints propagated (attPrp is idempotent for multi-BS deployments)
	// No endpoints are skipped - each BS needs to receive attPrp for every endpoint
	require.Len(t, env.sender.calls, 4, "all endpoints (including attached) should receive propagate")
}

// TestRoamingPolicyEnforcement verifies ATT-03: shouldPropagate blocks cross-tenant
// propagation pending roaming agreement implementation.
func TestRoamingPolicyEnforcement(t *testing.T) {
	t.Parallel()

	const (
		endpointTenant = int64(100)
		sessionTenant  = int64(200) // Different tenant
	)

	endpoint := &models.EndPoint{
		ID:       1001,
		TenantID: endpointTenant,
		Bidi:     true,
	}

	env := newPropagationTestEnv(t, endpoint)

	// ATT-03: Session from different tenant (roaming scenario)
	sessions := []propagation.BaseStationSession{
		{
			ID:             "session-roaming",
			BaseStationEUI: 0x8899AABBCCDDEEFF,
			TenantID:       sessionTenant, // Cross-tenant
		},
	}

	// ATT-03: Add endpoint tenant context
	ctx := pkgcontext.WithTenantID(context.Background(), endpointTenant)
	err := env.service.TriggerEndpointPropagate(ctx, endpoint.ID, sessions)

	// ATT-03: Should succeed but not send any propagates (blocked by roaming policy)
	require.NoError(t, err, "propagation should succeed without errors")
	assert.Empty(t, env.sender.calls, "cross-tenant propagation should be blocked")
}

// TestUnidirectionalEndpointPropagated verifies BSSCI §3.8: unidirectional (Class Z)
// endpoints are propagated to base stations. The spec states attPrp is "required for
// unidirectional End Points" for offline preattachment.
func TestUnidirectionalEndpointPropagated(t *testing.T) {
	t.Parallel()

	const tenantID = int64(100)

	endpoint := &models.EndPoint{
		ID:       1001,
		TenantID: tenantID,
		Bidi:     false, // Class Z unidirectional
	}

	env := newPropagationTestEnv(t, endpoint)

	session := propagation.BaseStationSession{
		ID:             "session-1",
		BaseStationEUI: 0x0011223344556677,
		TenantID:       tenantID,
	}

	ctx := context.Background()
	err := env.service.ReconcileBaseStation(ctx, session, nil)

	require.NoError(t, err, "reconciliation should succeed")

	// BSSCI §3.8: Unidirectional endpoints MUST be propagated
	require.Len(t, env.sender.calls, 1, "unidirectional endpoint should receive propagate")
	assert.Equal(t, "session-1", env.sender.calls[0].sessionID)
	assert.False(t, env.sender.calls[0].endpoint.Bidi, "bidi should be false in attPrp")
}

// TestMultiBSPropagation verifies BSSCI §3.8: endpoints with PropagateStatus="attached"
// are STILL propagated to a NEW base station. The global status doesn't prevent
// propagation to different BSs. attPrp is idempotent by design.
func TestMultiBSPropagation(t *testing.T) {
	t.Parallel()

	const tenantID = int64(100)

	attachedStatus := pkgbssci.PropagateStatusAttached

	// Endpoint already marked "attached" from previous BS-A reconciliation
	endpoint := &models.EndPoint{
		ID:              1001,
		TenantID:        tenantID,
		Bidi:            true,
		PropagateStatus: &attachedStatus, // Already propagated to BS-A
	}

	env := newPropagationTestEnv(t, endpoint)

	// NEW base station BS-B connects
	sessionBSB := propagation.BaseStationSession{
		ID:             "session-bs-b",
		BaseStationEUI: 0x1122334455667788, // Different BS
		TenantID:       tenantID,
	}

	ctx := context.Background()
	err := env.service.ReconcileBaseStation(ctx, sessionBSB, nil)

	require.NoError(t, err, "reconciliation should succeed")

	// BSSCI §3.8: BS-B MUST receive attPrp even though PropagateStatus is "attached"
	require.Len(t, env.sender.calls, 1, "new BS should receive attPrp for already-attached endpoint")
	assert.Equal(t, "session-bs-b", env.sender.calls[0].sessionID)
}

// --- Test Infrastructure ---

type propagationTestEnv struct {
	service *propagationService
	repo    *fakeEndpointRepo
	sender  *mockAttachPropagateSender
}

func newPropagationTestEnv(t *testing.T, endpoints ...*models.EndPoint) *propagationTestEnv {
	t.Helper()

	repo := newFakeEndpointRepo(endpoints...)
	sender := &mockAttachPropagateSender{}
	logger := logger.NewNop()

	service := NewPropagationService(repo, sender, logger).(*propagationService)

	return &propagationTestEnv{
		service: service,
		repo:    repo,
		sender:  sender,
	}
}

// mockAttachPropagateSender captures SendAttachPropagateBySessionID calls for verification.
type mockAttachPropagateSender struct {
	calls []propagateCall
}

type propagateCall struct {
	sessionID string
	endpoint  *models.EndPoint
}

func (m *mockAttachPropagateSender) SendAttachPropagateBySessionID(
	_ context.Context,
	sessionID string,
	endpoint *models.EndPoint,
) error {
	m.calls = append(m.calls, propagateCall{
		sessionID: sessionID,
		endpoint:  endpoint,
	})
	return nil
}

func (m *mockAttachPropagateSender) SendAttachPropagateToSession(
	_ context.Context,
	_ *pkgbssci.Session,
	_ *models.EndPoint,
) error {
	// Not used in propagation service tests (service uses BySessionID variant)
	return nil
}

// fakeEndpointRepo provides endpoints for testing without database.
type fakeEndpointRepo struct {
	endpointsByID     map[int64]*models.EndPoint
	endpointsByTenant map[int64][]*models.EndPoint
}

func newFakeEndpointRepo(endpoints ...*models.EndPoint) *fakeEndpointRepo {
	byID := make(map[int64]*models.EndPoint)
	byTenant := make(map[int64][]*models.EndPoint)

	for _, ep := range endpoints {
		byID[ep.ID] = ep
		byTenant[ep.TenantID] = append(byTenant[ep.TenantID], ep)
	}

	return &fakeEndpointRepo{
		endpointsByID:     byID,
		endpointsByTenant: byTenant,
	}
}

func (f *fakeEndpointRepo) GetByID(_ context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	if ep, ok := f.endpointsByID[id]; ok && ep.TenantID == tenantID {
		clone := *ep
		return &clone, nil
	}
	return nil, nil
}

func (f *fakeEndpointRepo) GetByTenant(_ context.Context, tenantID int64) ([]*models.EndPoint, error) {
	endpoints := f.endpointsByTenant[tenantID]
	// Return clones to prevent test mutations
	result := make([]*models.EndPoint, len(endpoints))
	for i, ep := range endpoints {
		clone := *ep
		result[i] = &clone
	}
	return result, nil
}

// Remaining interface methods not used in these tests
func (f *fakeEndpointRepo) Create(context.Context, *models.EndPoint) error { return nil }
func (f *fakeEndpointRepo) GetByEUI(context.Context, int64, []byte) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) Get(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) CountByTenant(context.Context, int64) (int64, error) { return 0, nil }
func (f *fakeEndpointRepo) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) Update(context.Context, *models.EndPoint) error { return nil }
func (f *fakeEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateFields(context.Context, int64, int64, map[string]interface{}) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
	return nil
}
func (f *fakeEndpointRepo) StreamAllForPropagation(context.Context, int64, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) HasEndpointsSince(context.Context, time.Time) (bool, error) {
	return false, nil
}
func (f *fakeEndpointRepo) GetEndpointWithKeysForDetachValidation(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}
func (f *fakeEndpointRepo) DeleteByTenant(context.Context, int64, []byte) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (f *fakeEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}
