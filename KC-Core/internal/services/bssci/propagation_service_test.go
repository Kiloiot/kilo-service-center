package bssciservices

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Fixture broadcast errors aggregated by the propagation tests.
var (
	errTestBroadcastNetworkTimeout  = errors.New("session A: network timeout")
	errTestBroadcastInvalidEndpoint = errors.New("session B: invalid endpoint")
	errTestBroadcastDatabaseError   = errors.New("session C: database error")
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
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), endpointTenant)
	err := env.service.TriggerEndpointPropagate(ctx, endpoint.ID, sessions)

	require.NoError(t, err, "propagation should succeed")

	// ATT-02: Verify only same-tenant session received propagate
	require.Len(t, env.sender.calls, 1, "only one session should receive propagate")
	assert.Equal(t, "session-1", env.sender.calls[0].sessionID, "same-tenant session should receive")
}

// A reconnecting base station is sent attPrp for the endpoints the service
// center holds attached only, unidirectional ones included (BSSCI §3.8); one
// deregistered, detached or never attached stays off the station.
func TestReconcileResendsOnlyAttachedEndpoints(t *testing.T) {
	t.Parallel()

	const tenantID = int64(100)
	endpoints := []*models.EndPoint{
		{ID: 1001, TenantID: tenantID, Bidi: true, EpStatus: pkgbssci.EndpointStatusAttached},
		{ID: 1002, TenantID: tenantID, Bidi: false, EpStatus: pkgbssci.EndpointStatusAttached},
		{ID: 1003, TenantID: tenantID, Bidi: true, EpStatus: endpoint.EndpointStatusDetached},
		{ID: 1004, TenantID: tenantID, Bidi: false, EpStatus: endpoint.EndpointStatusDetached},
	}
	env := newPropagationTestEnv(t, endpoints...)
	session := propagation.BaseStationSession{ID: "session-1", BaseStationEUI: 0x0011223344556677, TenantID: tenantID}

	require.NoError(t, env.service.ReconcileBaseStation(testutil.TestContext(), session, nil))

	var resent []int64
	for _, call := range env.sender.calls {
		resent = append(resent, call.endpoint.ID)
	}
	assert.ElementsMatch(t, []int64{1001, 1002}, resent, "only the attached endpoints are propagated again")
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
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), endpointTenant)
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
		EpStatus: pkgbssci.EndpointStatusAttached,
	}

	env := newPropagationTestEnv(t, endpoint)

	session := propagation.BaseStationSession{
		ID:             "session-1",
		BaseStationEUI: 0x0011223344556677,
		TenantID:       tenantID,
	}

	ctx := testutil.TestContext()
	err := env.service.ReconcileBaseStation(ctx, session, nil)

	require.NoError(t, err, "reconciliation should succeed")

	// BSSCI §3.8: Unidirectional endpoints MUST be propagated
	require.Len(t, env.sender.calls, 1, "unidirectional endpoint should receive propagate")
	assert.Equal(t, "session-1", env.sender.calls[0].sessionID)
	assert.False(t, env.sender.calls[0].endpoint.Bidi, "bidi should be false in attPrp")
}

// TestMultiBSPropagation verifies BSSCI §3.8: an attached endpoint another base
// station already confirmed is still propagated to a NEW base station.
func TestMultiBSPropagation(t *testing.T) {
	t.Parallel()

	const tenantID = int64(100)

	attachedStatus := pkgbssci.PropagateStatusAttached

	endpoint := &models.EndPoint{
		ID:              1001,
		TenantID:        tenantID,
		Bidi:            true,
		EpStatus:        pkgbssci.EndpointStatusAttached,
		PropagateStatus: &attachedStatus, // Already propagated to BS-A
	}

	env := newPropagationTestEnv(t, endpoint)

	// NEW base station BS-B connects
	sessionBSB := propagation.BaseStationSession{
		ID:             "session-bs-b",
		BaseStationEUI: 0x1122334455667788,
		// Different BS
		TenantID: tenantID,
	}

	ctx := testutil.TestContext()
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

// mockAttachPropagateSender captures the attPrp and detPrp each session is sent.
type mockAttachPropagateSender struct {
	calls    []propagateCall
	detaches []detachCall
}

type detachCall struct {
	sessionID   string
	endpointEUI uint64
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

func (m *mockAttachPropagateSender) SendDetachPropagate(sessionID string, endpointEUI uint64) error {
	m.detaches = append(m.detaches, detachCall{sessionID: sessionID, endpointEUI: endpointEUI})
	return nil
}

// fakeEndpointRepo provides endpoints for testing without database.
type fakeEndpointRepo struct {
	endpointsByID       map[int64]*models.EndPoint
	endpointsByTenant   map[int64][]*models.EndPoint
	attachmentChangedAt map[int64]time.Time
}

// attachmentChanged records when the service center last changed an endpoint's attachment.
func (f *fakeEndpointRepo) attachmentChanged(endpointID int64, at time.Time) {
	f.attachmentChangedAt[endpointID] = at
}

func (f *fakeEndpointRepo) GetByAttachmentChangedSince(_ context.Context, tenantID int64, status string, since *time.Time) ([]*models.EndPoint, error) {
	var changed []*models.EndPoint
	for _, ep := range f.endpointsByTenant[tenantID] {
		at, recorded := f.attachmentChangedAt[ep.ID]
		if ep.EpStatus != status || !recorded || (since != nil && !at.After(*since)) {
			continue
		}
		clone := *ep
		changed = append(changed, &clone)
	}
	return changed, nil
}

func newFakeEndpointRepo(endpoints ...*models.EndPoint) *fakeEndpointRepo {
	byID := make(map[int64]*models.EndPoint)
	byTenant := make(map[int64][]*models.EndPoint)

	for _, ep := range endpoints {
		byID[ep.ID] = ep
		byTenant[ep.TenantID] = append(byTenant[ep.TenantID], ep)
	}

	return &fakeEndpointRepo{
		endpointsByID:       byID,
		endpointsByTenant:   byTenant,
		attachmentChangedAt: make(map[int64]time.Time),
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

func (f *fakeEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, models.RadioMetricsUpdate) error {
	return nil
}

func (f *fakeEndpointRepo) EndpointRegistrationUpdate(context.Context, int64, int64, models.EndpointRegistrationParams) error {
	return nil
}

func (f *fakeEndpointRepo) EndpointAttachmentStateUpdate(context.Context, int64, int64, models.EndpointAttachmentStateParams) error {
	return nil
}

func (f *fakeEndpointRepo) EndpointAttachSessionUpdate(context.Context, int64, int64, models.EndpointAttachSessionParams) error {
	return nil
}

func (f *fakeEndpointRepo) EndpointDetachStateUpdate(context.Context, int64, int64, models.EndpointDetachStateParams) error {
	return nil
}

func (f *fakeEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}

func (f *fakeEndpointRepo) DeleteByTenant(context.Context, int64, []byte) (int64, error) {
	return 0, nil
}

func (f *fakeEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (f *fakeEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// TestAggregateErrors_BroadcastErrorAggregation verifies that aggregateErrors
// wraps multiple broadcast failures into a single error that carries the
// catalog message, the failure count, and the first underlying error reachable
// via errors.Is. Replaces the obsolete
// Test_SendAttachPropagateBySessionID_BroadcastErrorAggregation regression,
// which targeted a session-specific method that does not aggregate.
func TestAggregateErrors_BroadcastErrorAggregation(t *testing.T) {
	first := errTestBroadcastNetworkTimeout
	errs := []error{
		first,
		errTestBroadcastInvalidEndpoint,
		errTestBroadcastDatabaseError,
	}

	err := aggregateErrors(errs)
	require.Error(t, err)

	msg := err.Error()
	assert.Contains(t, msg, "3 failures", "error message must include failure count")
	assert.Contains(t, msg,
		pkgbssci.ResolveErrorMessage(pkgbssci.ErrPropagationBroadcastFailure),
		"error message must include the catalog-derived prefix")
	assert.True(t, errors.Is(err, first), "first underlying error must be reachable via errors.Is")
}

func TestAggregateErrors_NoErrors(t *testing.T) {
	assert.NoError(t, aggregateErrors(nil))
	assert.NoError(t, aggregateErrors([]error{}))
}
