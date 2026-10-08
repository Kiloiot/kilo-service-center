package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/roles"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// testEventTypeServiceStarted is the platform event type used by the
// RecordPlatformEvent fixtures.
const testEventTypeServiceStarted = "service.started"

// errFixtureDBConnectionLost simulates a repository outage.
var errFixtureDBConnectionLost = errors.New("database connection lost")

// stubRoleResolver answers GetUserRoles lookups with fixed results.
type stubRoleResolver struct {
	roles    authz.Roles
	err      error
	gotUser  uuid.UUID
	gotKey   uuid.UUID
	gotOrgID uuid.UUID
}

func (s *stubRoleResolver) Resolve(_ context.Context, userID, orgID uuid.UUID) (authz.Roles, error) {
	s.gotUser, s.gotOrgID = userID, orgID
	return s.roles, s.err
}

func (s *stubRoleResolver) ResolveServiceAccount(_ context.Context, keyID, orgID uuid.UUID) (authz.Roles, error) {
	s.gotKey, s.gotOrgID = keyID, orgID
	return s.roles, s.err
}

const (
	testRolesOrgID  = "fe7fe002-6880-4ea6-84ed-a69911dbdf8c"
	testRolesUserID = "be8a02c9-1470-4314-8a86-48712761aca9"
	testRolesKeyID  = "5b1f0c1e-3f4e-4d1a-9f2b-7c0d6e8a9b10"
)

func TestGetUserRoles_ResolvesAServiceAccountKeyThroughTheSameResolver(t *testing.T) {
	resolver := &stubRoleResolver{roles: authz.ServiceAccountRoles}
	svc := NewIdentityInternalService(nil, nil, resolver, nil, nil, logger.NewNop())

	resp, err := svc.GetUserRoles(testutil.TestContext(), &pb.GetUserRolesRequest{OrgId: testRolesOrgID, ServiceAccountId: testRolesKeyID})
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	got := resp.GetRoles()
	if !got.GetBaseStationManager() || !got.GetEndpointManager() || got.GetAdmin() || got.GetTenantManager() {
		t.Fatalf("roles = %+v, want base station and endpoint manager only", got)
	}
	if resolver.gotKey.String() != testRolesKeyID || resolver.gotOrgID.String() != testRolesOrgID || resolver.gotUser != uuid.Nil {
		t.Fatalf("resolved key %s user %s in %s", resolver.gotKey, resolver.gotUser, resolver.gotOrgID)
	}
}

func TestGetUserRoles_ReturnsTheResolvedRoles(t *testing.T) {
	resolver := &stubRoleResolver{roles: authz.Roles{BaseStationManager: true}}
	svc := NewIdentityInternalService(nil, nil, resolver, nil, nil, logger.NewNop())

	resp, err := svc.GetUserRoles(testutil.TestContext(), &pb.GetUserRolesRequest{OrgId: testRolesOrgID, UserId: testRolesUserID})
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	if !resp.GetRoles().GetBaseStationManager() || resp.GetRoles().GetEndpointManager() || resp.GetRoles().GetAdmin() {
		t.Fatalf("roles = %+v, want base station manager only", resp.GetRoles())
	}
	if resolver.gotOrgID.String() != testRolesOrgID || resolver.gotUser.String() != testRolesUserID {
		t.Fatalf("resolved for %s in %s", resolver.gotUser, resolver.gotOrgID)
	}
}

func TestGetUserRoles_WithoutAnOrganizationResolvesTheOrganizationIndependentRoles(t *testing.T) {
	resolver := &stubRoleResolver{roles: authz.AllRoles, gotOrgID: uuid.New()}
	svc := NewIdentityInternalService(nil, nil, resolver, nil, nil, logger.NewNop())

	resp, err := svc.GetUserRoles(testutil.TestContext(), &pb.GetUserRolesRequest{UserId: testRolesUserID})
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	if !resp.GetRoles().GetAdmin() || resolver.gotOrgID != uuid.Nil {
		t.Fatalf("roles %+v resolved in %s, want the organization-independent roles", resp.GetRoles(), resolver.gotOrgID)
	}
}

func TestGetUserRoles_RejectsAndMapsFailures(t *testing.T) {
	cases := []struct {
		name     string
		req      *pb.GetUserRolesRequest
		resolver PrincipalRoleResolver
		token    string
	}{
		{name: "missing user", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID}, resolver: &stubRoleResolver{}, token: grpcerrors.ErrTokenUserIDRequired},
		{name: "malformed org", req: &pb.GetUserRolesRequest{OrgId: "not-a-uuid", UserId: testRolesUserID}, resolver: &stubRoleResolver{}, token: grpcerrors.ErrTokenInvalidOrgIDFormat},
		{name: "malformed user", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID, UserId: "not-a-uuid"}, resolver: &stubRoleResolver{}, token: grpcerrors.ErrTokenInvalidUserIDFormat},
		{name: "user and service account together", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID, UserId: testRolesUserID, ServiceAccountId: testRolesKeyID}, resolver: &stubRoleResolver{}, token: grpcerrors.ErrTokenInvalidRequest},
		{name: "malformed service account", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID, ServiceAccountId: "not-a-uuid"}, resolver: &stubRoleResolver{}, token: grpcerrors.ErrTokenInvalidAPIKeyIDFormat},
		{name: "unknown service account", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID, ServiceAccountId: testRolesKeyID}, resolver: &stubRoleResolver{err: roles.ErrAPIKeyNotFound}, token: grpcerrors.ErrTokenApiKeyNotFound},
		{name: "no resolver", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID, UserId: testRolesUserID}, resolver: nil, token: grpcerrors.ErrTokenServiceNotConfigured},
		{name: "unknown user", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID, UserId: testRolesUserID}, resolver: &stubRoleResolver{err: roles.ErrUserNotFound}, token: grpcerrors.ErrTokenUserNotFound},
		{name: "store failure", req: &pb.GetUserRolesRequest{OrgId: testRolesOrgID, UserId: testRolesUserID}, resolver: &stubRoleResolver{err: errFixtureDBConnectionLost}, token: grpcerrors.ErrTokenInternalError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewIdentityInternalService(nil, nil, tc.resolver, nil, nil, logger.NewNop())
			_, err := svc.GetUserRoles(testutil.TestContext(), tc.req)
			st, ok := status.FromError(err)
			if !ok || st.Code() != grpcerrors.GetGRPCCode(tc.token) || st.Message() != grpcerrors.ResolveErrorMessage(tc.token) {
				t.Fatalf("err = %v, want catalog token %s", err, tc.token)
			}
		})
	}
}

// mockUserLookup implements UserLookup for testing.
type mockUserLookup struct {
	user *models.User
	err  error
}

func (m *mockUserLookup) GetByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

func TestCheckServerAdmin_AdminUser(t *testing.T) {
	mock := &mockUserLookup{user: &models.User{IsAdmin: true, IsActive: true}}
	svc := NewIdentityInternalService(nil, nil, nil, mock, nil, logger.NewNop())

	resp, err := svc.CheckServerAdmin(testutil.TestContext(), &pb.CheckServerAdminRequest{
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resp.IsAdmin {
		t.Error("expected is_admin to be true")
	}
}

func TestCheckServerAdmin_InactiveAdminIsNotAnAdmin(t *testing.T) {
	mock := &mockUserLookup{user: &models.User{IsAdmin: true, IsActive: false}}
	svc := NewIdentityInternalService(nil, nil, nil, mock, nil, logger.NewNop())

	resp, err := svc.CheckServerAdmin(testutil.TestContext(), &pb.CheckServerAdminRequest{UserId: testRolesUserID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.IsAdmin {
		t.Error("a deactivated administrator must not pass the admin check")
	}
}

func TestCheckServerAdmin_NonAdminUser(t *testing.T) {
	mock := &mockUserLookup{user: &models.User{IsAdmin: false}}
	svc := NewIdentityInternalService(nil, nil, nil, mock, nil, logger.NewNop())

	resp, err := svc.CheckServerAdmin(testutil.TestContext(), &pb.CheckServerAdminRequest{
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.IsAdmin {
		t.Error("expected is_admin to be false")
	}
}

func TestCheckServerAdmin_InvalidUUID(t *testing.T) {
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil, logger.NewNop())

	_, err := svc.CheckServerAdmin(testutil.TestContext(), &pb.CheckServerAdminRequest{
		UserId: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	st, _ := status.FromError(err)
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat)
	if st.Code() != expectedCode {
		t.Errorf("expected %v, got %v", expectedCode, st.Code())
	}
}

func TestCheckServerAdmin_EmptyUserID(t *testing.T) {
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil, logger.NewNop())

	_, err := svc.CheckServerAdmin(testutil.TestContext(), &pb.CheckServerAdminRequest{
		UserId: "",
	})
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestCheckServerAdmin_UserNotFound(t *testing.T) {
	mock := &mockUserLookup{err: admin.ErrUserNotFound}
	svc := NewIdentityInternalService(nil, nil, nil, mock, nil, logger.NewNop())

	_, err := svc.CheckServerAdmin(testutil.TestContext(), &pb.CheckServerAdminRequest{
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestCheckServerAdmin_NilUserSvc(t *testing.T) {
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil, logger.NewNop())

	_, err := svc.CheckServerAdmin(testutil.TestContext(), &pb.CheckServerAdminRequest{
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err == nil {
		t.Fatal("expected error when userSvc is nil")
	}
	st, _ := status.FromError(err)
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}

// mockEventWriter implements audit.EventWriter for testing.
type mockEventWriter struct {
	events []*models.SystemEvent
}

func (m *mockEventWriter) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestRecordPlatformEvent_WithEventWriter(t *testing.T) {
	writer := &mockEventWriter{}
	svc := NewIdentityInternalService(nil, nil, nil, nil, writer, logger.NewNop())

	resp, err := svc.RecordPlatformEvent(testutil.TestContext(), &pb.RecordPlatformEventRequest{
		TenantId:    1,
		EventType:   testEventTypeServiceStarted,
		Category:    "system",
		Severity:    "info",
		SourceType:  "system",
		SourceName:  "kc-gateway",
		Title:       "KC-Gateway started",
		Description: "Gateway started on host test-host",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if len(writer.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(writer.events))
	}

	event := writer.events[0]
	if event.TenantID != "1" {
		t.Errorf("expected tenant_id '1', got %q", event.TenantID)
	}
	if event.EventType != "service.started" {
		t.Errorf("expected event_type 'service.started', got %q", event.EventType)
	}
	if event.Category != "system" {
		t.Errorf("expected category 'system', got %q", event.Category)
	}
	if event.Title != "KC-Gateway started" {
		t.Errorf("expected title 'KC-Gateway started', got %q", event.Title)
	}
	if event.SourceName != "kc-gateway" {
		t.Errorf("expected source_name 'kc-gateway', got %q", event.SourceName)
	}
}

func TestRecordPlatformEvent_NilEventWriter(t *testing.T) {
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil, logger.NewNop())

	resp, err := svc.RecordPlatformEvent(testutil.TestContext(), &pb.RecordPlatformEventRequest{
		TenantId:    1,
		EventType:   testEventTypeServiceStarted,
		Category:    "system",
		Severity:    "info",
		SourceType:  "system",
		SourceName:  "kc-gateway",
		Title:       "KC-Gateway started",
		Description: "Gateway started",
	})
	if err != nil {
		t.Fatalf("expected no error with nil event writer, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
