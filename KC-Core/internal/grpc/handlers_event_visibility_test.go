package grpc

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/errorgroups"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func rolesCtx(roles authz.Roles) context.Context {
	return authz.WithRoles(ownerCtx(), roles)
}

func TestListEvents_NarrowsToTheCallersCategories(t *testing.T) {
	cases := []struct {
		name      string
		roles     authz.Roles
		requested []string
		want      []string
		reached   bool
	}{
		{
			name:    "administrator reads every category unfiltered",
			roles:   authz.AllRoles,
			want:    nil,
			reached: true,
		},
		{
			name:      "base station manager keeps base station categories only",
			roles:     authz.Roles{BaseStationManager: true},
			requested: []string{models.EventCategoryBaseStation, models.EventCategorySecurity, models.EventCategoryEndpoint},
			want:      []string{models.EventCategoryBaseStation},
			reached:   true,
		},
		{
			name:  "endpoint manager without a filter reads endpoint and shared protocol categories",
			roles: authz.Roles{EndpointManager: true},
			want: []string{
				models.EventCategoryEndpoint, models.EventCategoryMessage, models.EventCategoryProtocol,
				models.EventCategoryRoaming, models.EventCategorySCACI, models.EventCategorySession,
			},
			reached: true,
		},
		{
			name:      "security and system events stay hidden from managers",
			roles:     authz.Roles{BaseStationManager: true, EndpointManager: true, TenantManager: true},
			requested: []string{models.EventCategorySecurity, models.EventCategorySystem, models.EventCategoryAudit},
			reached:   false,
		},
		{
			name:    "tenant manager reads no event category",
			roles:   authz.Roles{TenantManager: true},
			reached: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newRetainedService(t, &retainedFakes{})
			events := &filterEventSvc{}
			svc.eventSvc = events

			resp, err := svc.ListEvents(rolesCtx(tc.roles), &pb.ListEventsRequest{Categories: tc.requested})

			require.NoError(t, err)
			if !tc.reached {
				assert.Nil(t, events.last, "no category is readable, so the event store is never queried")
				assert.Empty(t, resp.GetEvents())
				return
			}
			require.NotNil(t, events.last)
			assert.ElementsMatch(t, tc.want, events.last.Categories)
		})
	}
}

// eventStream collects nothing; StreamEvents ends before opening when no category is readable.
type eventStream struct {
	grpc.ServerStreamingServer[pb.Event]
	ctx context.Context
}

func (s eventStream) Context() context.Context { return s.ctx }

func (eventStream) SendHeader(metadata.MD) error { return nil }

type recordingStreamSvc struct {
	filterEventSvc
	streamed *grpcservices.EventFilters
}

func (r *recordingStreamSvc) Stream(_ context.Context, _ int64, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	r.streamed = filters
	ch := make(chan *grpcservices.Event)
	close(ch)
	return ch, nil
}

func TestStreamEvents_NarrowsToTheCallersCategories(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	events := &recordingStreamSvc{}
	svc.eventSvc = events

	err := svc.StreamEvents(&pb.StreamEventsRequest{Category: models.EventCategorySecurity},
		eventStream{ctx: rolesCtx(authz.Roles{EndpointManager: true})})
	require.NoError(t, err)
	assert.Nil(t, events.streamed, "a manager never subscribes to security events")

	err = svc.StreamEvents(&pb.StreamEventsRequest{}, eventStream{ctx: rolesCtx(authz.Roles{BaseStationManager: true})})
	require.NoError(t, err)
	require.NotNil(t, events.streamed)
	assert.NotContains(t, events.streamed.Categories, models.EventCategorySystem)
	assert.Contains(t, events.streamed.Categories, models.EventCategoryBSSCI)
}

// refusingBuckets answers every bucket as unreadable for the caller.
type refusingBuckets struct{}

func (refusingBuckets) List(context.Context, int64, string, grpcservices.ScaciWindow, int, int) ([]*grpcservices.ErrorGroup, int64, error) {
	return nil, 0, errorgroups.ErrBucketNotReadable
}

func TestListErrorGroups_UnreadableBucketIsAnInsufficientRole(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	svc.errorGroups = refusingBuckets{}

	_, err := svc.ListErrorGroups(rolesCtx(authz.Roles{EndpointManager: true}), &pb.ListErrorGroupsRequest{Bucket: errorgroups.BucketControlPlane})
	assertCode(t, err, grpcerrors.ErrTokenInsufficientRole)
}

const platformTenantID int64 = 1

// tenantEventStore answers List the way the event store does: tenant, category and type filters applied.
type tenantEventStore struct {
	filterEventSvc
	events []*grpcservices.Event
}

func (s *tenantEventStore) List(_ context.Context, tenantID int64, filters *grpcservices.EventFilters, _, _ int) ([]*grpcservices.Event, int64, error) {
	var out []*grpcservices.Event
	for _, e := range s.events {
		if e.TenantID != tenantID || !matchesAny(filters.Categories, e.Category) || !matchesAny(filters.EventTypes, e.EventType) {
			continue
		}
		out = append(out, e)
	}
	return out, int64(len(out)), nil
}

func matchesAny(allowed []string, value string) bool {
	return len(allowed) == 0 || slices.Contains(allowed, value)
}

var platformEventTypes = []string{
	models.EventTypeUserCreated, models.EventTypeUserDeleted, models.EventTypeUserRegistered,
	models.EventTypeUserPasswordChanged, models.EventTypeOrgDeleted,
	models.EventTypeAPIKeyDeleted, models.EventTypeCertificateServerRenewed,
}

func platformTenantStore() *tenantEventStore {
	store := &tenantEventStore{events: []*grpcservices.Event{
		{ID: models.EventTypeEndpointCreated, TenantID: platformTenantID, Category: models.EventCategoryEndpoint, EventType: models.EventTypeEndpointCreated},
	}}
	for _, et := range platformEventTypes {
		store.events = append(store.events, &grpcservices.Event{ID: et, TenantID: platformTenantID, Category: models.EventCategoryAudit, EventType: et})
	}
	return store
}

func platformTenantCtx(roles authz.Roles) context.Context {
	return authz.WithRoles(testutil.TestContextWithTenantAndOrg(platformTenantID, retainedOwnerOrg), roles)
}

func listedEventTypes(t *testing.T, roles authz.Roles, req *pb.ListEventsRequest) []string {
	t.Helper()
	svc := newRetainedService(t, &retainedFakes{})
	svc.eventSvc = platformTenantStore()

	resp, err := svc.ListEvents(platformTenantCtx(roles), req)

	require.NoError(t, err)
	var got []string
	for _, e := range resp.GetEvents() {
		got = append(got, e.GetEventType())
	}
	return got
}

func TestListEvents_PlatformEventsReachOnlyServerAdministrators(t *testing.T) {
	members := map[string]authz.Roles{
		"tenant manager":       {TenantManager: true},
		"base station manager": {BaseStationManager: true},
		"endpoint manager":     {EndpointManager: true},
		"every manager role":   {TenantManager: true, BaseStationManager: true, EndpointManager: true},
	}
	requests := map[string]*pb.ListEventsRequest{
		"unfiltered":          {},
		"asking for user.*":   {EventTypes: []string{models.EventTypeUserCreated, models.EventTypeUserDeleted}},
		"asking for audit":    {Categories: []string{models.EventCategoryAudit}, EventTypes: []string{models.EventTypeOrgDeleted}},
		"asking for security": {Categories: []string{models.EventCategorySecurity, models.EventCategoryAudit}},
	}
	for member, roles := range members {
		for reqName, req := range requests {
			t.Run(member+"/"+reqName, func(t *testing.T) {
				for _, et := range listedEventTypes(t, roles, req) {
					assert.NotContains(t, platformEventTypes, et, "a platform-tenant member must not read platform events")
				}
			})
		}
	}

	t.Run("endpoint manager still reads the tenant's endpoint events", func(t *testing.T) {
		got := listedEventTypes(t, authz.Roles{EndpointManager: true}, &pb.ListEventsRequest{})
		assert.Equal(t, []string{models.EventTypeEndpointCreated}, got)
	})

	t.Run("server administrator reads every platform event", func(t *testing.T) {
		got := listedEventTypes(t, authz.AllRoles, &pb.ListEventsRequest{Categories: []string{models.EventCategoryAudit}})
		assert.ElementsMatch(t, platformEventTypes, got)
	})
}

// TestListEvents_SessionAndDownlinkEventsFollowTheirCategories: Application
// Center session events and refused connects of the tenant's own Application
// Center (scaci) and downlink acknowledgements (message) reach endpoint
// managers; a refused connect no tenant owns (security), a key reveal and a
// key removal (audit) reach administrators only.
func TestListEvents_SessionAndDownlinkEventsFollowTheirCategories(t *testing.T) {
	store := &tenantEventStore{events: []*grpcservices.Event{
		{ID: "opened", TenantID: matrixOwnerTenant, Category: models.EventCategorySCACI, EventType: models.EventTypeSCACISessionOpened},
		{ID: "closed", TenantID: matrixOwnerTenant, Category: models.EventCategorySCACI, EventType: models.EventTypeSCACISessionClosed},
		{ID: "acked", TenantID: matrixOwnerTenant, Category: models.EventCategoryMessage, EventType: models.EventTypeDLDataAcknowledged},
		{ID: "refused", TenantID: matrixOwnerTenant, Category: models.EventCategorySecurity, EventType: models.EventTypeSCACIConnectRefused},
		{ID: "refused-own", TenantID: matrixOwnerTenant, Category: models.EventCategorySCACI, EventType: models.EventTypeSCACIConnectRefused},
		{ID: "revealed", TenantID: matrixOwnerTenant, Category: models.EventCategoryAudit, EventType: models.EventTypeEndpointKeysRevealed},
		{ID: "removed", TenantID: matrixOwnerTenant, Category: models.EventCategoryAudit, EventType: models.EventTypeEndpointKeysRemoved},
	}}
	cases := []struct {
		name  string
		roles authz.Roles
		want  []string
	}{
		{name: "endpoint manager", roles: authz.Roles{EndpointManager: true}, want: []string{
			models.EventTypeSCACISessionOpened, models.EventTypeSCACISessionClosed, models.EventTypeDLDataAcknowledged,
			models.EventTypeSCACIConnectRefused,
		}},
		{name: "base station manager", roles: authz.Roles{BaseStationManager: true}},
		{name: "tenant manager", roles: authz.Roles{TenantManager: true}},
		{name: "administrator", roles: authz.AllRoles, want: []string{
			models.EventTypeSCACISessionOpened, models.EventTypeSCACISessionClosed, models.EventTypeDLDataAcknowledged,
			models.EventTypeSCACIConnectRefused, models.EventTypeSCACIConnectRefused,
			models.EventTypeEndpointKeysRevealed, models.EventTypeEndpointKeysRemoved,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newRetainedService(t, &retainedFakes{})
			svc.eventSvc = store
			resp, err := svc.ListEvents(authz.WithRoles(testutil.TestContextWithTenant(matrixOwnerTenant), tc.roles), &pb.ListEventsRequest{})
			require.NoError(t, err)
			var got []string
			for _, e := range resp.GetEvents() {
				got = append(got, e.GetEventType())
			}
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}
