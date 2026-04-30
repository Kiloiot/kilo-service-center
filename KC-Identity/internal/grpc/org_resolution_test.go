package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockMembershipLookup implements MembershipLookup for testing.
type mockMembershipLookup struct {
	role     string
	isActive bool
	err      error
}

func (m *mockMembershipLookup) GetMembership(_ context.Context, _, _ string) (string, bool, error) {
	return m.role, m.isActive, m.err
}

func TestGetUserMembership_ValidRequest(t *testing.T) {
	mock := &mockMembershipLookup{role: "admin", isActive: true}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	resp, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "fe7fe002-6880-4ea6-84ed-a69911dbdf8c",
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Role != "admin" {
		t.Errorf("expected role 'admin', got %q", resp.Role)
	}
	if !resp.IsActive {
		t.Error("expected is_active to be true")
	}
}

func TestGetUserMembership_InactiveUser(t *testing.T) {
	mock := &mockMembershipLookup{role: "viewer", isActive: false}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	resp, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "fe7fe002-6880-4ea6-84ed-a69911dbdf8c",
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Role != "viewer" {
		t.Errorf("expected role 'viewer', got %q", resp.Role)
	}
	if resp.IsActive {
		t.Error("expected is_active to be false")
	}
}

func TestGetUserMembership_EmptyOrgID(t *testing.T) {
	mock := &mockMembershipLookup{role: "admin", isActive: true}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	_, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "",
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err == nil {
		t.Fatal("expected error for empty org_id")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired)
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
	}
}

func TestGetUserMembership_EmptyUserID(t *testing.T) {
	mock := &mockMembershipLookup{role: "admin", isActive: true}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	_, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "fe7fe002-6880-4ea6-84ed-a69911dbdf8c",
		UserId: "",
	})
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserIDRequired)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserIDRequired)
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
	}
}

func TestGetUserMembership_InvalidOrgIDFormat(t *testing.T) {
	mock := &mockMembershipLookup{role: "admin", isActive: true}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	_, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "not-a-uuid",
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err == nil {
		t.Fatal("expected error for invalid org UUID format")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat)
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
	}
}

func TestGetUserMembership_InvalidUserIDFormat(t *testing.T) {
	mock := &mockMembershipLookup{role: "admin", isActive: true}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	_, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "fe7fe002-6880-4ea6-84ed-a69911dbdf8c",
		UserId: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected error for invalid user UUID format")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat)
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
	}
}

func TestGetUserMembership_NotFound(t *testing.T) {
	mock := &mockMembershipLookup{err: storage.ErrNotFound}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	_, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "fe7fe002-6880-4ea6-84ed-a69911dbdf8c",
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err == nil {
		t.Fatal("expected error for non-existent membership")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMembershipNotFound)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v (%s), got %v (%s)", expectedCode, codes.NotFound, st.Code(), st.Code())
	}
	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMembershipNotFound)
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
	}
}

func TestGetUserMembership_RepoFailure(t *testing.T) {
	mock := &mockMembershipLookup{err: fmt.Errorf("database connection lost")}
	svc := NewIdentityInternalService(nil, nil, mock, nil, nil)

	_, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "fe7fe002-6880-4ea6-84ed-a69911dbdf8c",
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err == nil {
		t.Fatal("expected error for repo failure")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetMemberFailed)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v (%s), got %v (%s)", expectedCode, codes.Internal, st.Code(), st.Code())
	}
	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetMemberFailed)
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
	}
}

func TestGetUserMembership_NilMembershipService(t *testing.T) {
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil)

	_, err := svc.GetUserMembership(context.Background(), &pb.GetUserMembershipRequest{
		OrgId:  "fe7fe002-6880-4ea6-84ed-a69911dbdf8c",
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err == nil {
		t.Fatal("expected error when membershipSvc is nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v (%s), got %v (%s)", expectedCode, codes.Unimplemented, st.Code(), st.Code())
	}
	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured)
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
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
	mock := &mockUserLookup{user: &models.User{IsAdmin: true}}
	svc := NewIdentityInternalService(nil, nil, nil, mock, nil)

	resp, err := svc.CheckServerAdmin(context.Background(), &pb.CheckServerAdminRequest{
		UserId: "be8a02c9-1470-4314-8a86-48712761aca9",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resp.IsAdmin {
		t.Error("expected is_admin to be true")
	}
}

func TestCheckServerAdmin_NonAdminUser(t *testing.T) {
	mock := &mockUserLookup{user: &models.User{IsAdmin: false}}
	svc := NewIdentityInternalService(nil, nil, nil, mock, nil)

	resp, err := svc.CheckServerAdmin(context.Background(), &pb.CheckServerAdminRequest{
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
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil)

	_, err := svc.CheckServerAdmin(context.Background(), &pb.CheckServerAdminRequest{
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
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil)

	_, err := svc.CheckServerAdmin(context.Background(), &pb.CheckServerAdminRequest{
		UserId: "",
	})
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestCheckServerAdmin_UserNotFound(t *testing.T) {
	mock := &mockUserLookup{err: storage.ErrNotFound}
	svc := NewIdentityInternalService(nil, nil, nil, mock, nil)

	_, err := svc.CheckServerAdmin(context.Background(), &pb.CheckServerAdminRequest{
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
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil)

	_, err := svc.CheckServerAdmin(context.Background(), &pb.CheckServerAdminRequest{
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

// mockEventWriter implements grpcerrors.EventWriter for testing.
type mockEventWriter struct {
	events []*models.SystemEvent
}

func (m *mockEventWriter) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestRecordPlatformEvent_WithEventWriter(t *testing.T) {
	writer := &mockEventWriter{}
	svc := NewIdentityInternalService(nil, nil, nil, nil, writer)

	resp, err := svc.RecordPlatformEvent(context.Background(), &pb.RecordPlatformEventRequest{
		TenantId:    1,
		EventType:   "service.started",
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
	svc := NewIdentityInternalService(nil, nil, nil, nil, nil)

	resp, err := svc.RecordPlatformEvent(context.Background(), &pb.RecordPlatformEventRequest{
		TenantId:    1,
		EventType:   "service.started",
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
