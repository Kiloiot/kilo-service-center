package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/internal/services/blueprints"
	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestMapRegistryError_AllCases verifies all registry error mappings from service to gRPC tokens.
func TestMapRegistryError_AllCases(t *testing.T) {
	// Create a minimal CoreService just for testing mapRegistryError
	svc := &CoreService{}

	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "ErrRegistryDisabled",
			err:          blueprints.ErrRegistryDisabled,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryProviderDisabled),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryProviderDisabled),
		},
		{
			name:         "ErrRegistryAPIURLRequired",
			err:          blueprints.ErrRegistryAPIURLRequired,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryAPIURLRequired),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryAPIURLRequired),
		},
		{
			name:         "ErrAlreadySubmitted",
			err:          blueprints.ErrAlreadySubmitted,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintAlreadySubmitted),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintAlreadySubmitted),
		},
		{
			name:         "ErrBlueprintNotFound",
			err:          blueprints.ErrBlueprintNotFound,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound),
		},
		{
			name:         "ErrRegistryAuthFailed",
			err:          blueprints.ErrRegistryAuthFailed,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryAuthFailed),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryAuthFailed),
		},
		{
			name:         "ErrRegistryPermissionDenied",
			err:          blueprints.ErrRegistryPermissionDenied,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryPermissionDenied),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryPermissionDenied),
		},
		{
			name:         "ErrRegistryRateLimited",
			err:          blueprints.ErrRegistryRateLimited,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryRateLimited),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryRateLimited),
		},
		{
			name:         "ErrRegistryAPIError",
			err:          blueprints.ErrRegistryAPIError,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryAPIError),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryAPIError),
		},
		{
			name:         "ErrBranchCreateFailed",
			err:          blueprints.ErrBranchCreateFailed,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryBranchCreateFailed),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryBranchCreateFailed),
		},
		{
			name:         "ErrCommitFailed",
			err:          blueprints.ErrCommitFailed,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryCommitFailed),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryCommitFailed),
		},
		{
			name:         "ErrSubmissionCreateFailed",
			err:          blueprints.ErrSubmissionCreateFailed,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrySubmissionCreateFailed),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrySubmissionCreateFailed),
		},
		{
			name:         "ErrRegistryVersionAlreadyExists",
			err:          blueprints.ErrRegistryVersionAlreadyExists,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryVersionAlreadyExists),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryVersionAlreadyExists),
		},
		{
			name:         "ErrInvalidRegistryPathSegment",
			err:          blueprints.ErrInvalidRegistryPathSegment,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryInvalidPathSegment),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryInvalidPathSegment),
		},
		{
			name:         "ErrDeviceModelNotFound",
			err:          blueprints.ErrDeviceModelNotFound,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound),
		},
		{
			name:         "ErrManufacturerNotFound",
			err:          blueprints.ErrManufacturerNotFound,
			expectedCode: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
			expectedMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcErr := svc.mapRegistryError(tc.err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatalf("expected gRPC status error, got %v", grpcErr)
			}

			if st.Code() != tc.expectedCode {
				t.Errorf("code mismatch: expected %v, got %v", tc.expectedCode, st.Code())
			}

			if st.Message() != tc.expectedMsg {
				t.Errorf("message mismatch: expected %q, got %q", tc.expectedMsg, st.Message())
			}
		})
	}
}

// TestMapRegistryError_WrappedError tests that wrapped errors are correctly mapped.
func TestMapRegistryError_WrappedError(t *testing.T) {
	svc := &CoreService{}

	// Test with a wrapped error (as happens in actual service code)
	wrappedErr := blueprints.ErrRegistryAuthFailed

	grpcErr := svc.mapRegistryError(wrappedErr)
	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", grpcErr)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryAuthFailed)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}
}

func TestSubmitBlueprintToRegistry_MissingContributorName(t *testing.T) {
	ctx := testutil.TestContext()
	svc := &CoreService{
		blueprintSvc: &blueprints.Service{},
	}

	req := &pb.SubmitBlueprintToRegistryRequest{
		Id:               uuid.New().String(),
		ContributorName:  "",
		ContributorEmail: "contributor@example.com",
	}

	_, err := svc.SubmitBlueprintToRegistry(ctx, req)
	if err == nil {
		t.Fatalf("expected error for missing contributor name")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenContributorNameRequired)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}
	if st.Message() != grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenContributorNameRequired) {
		t.Errorf("message mismatch: expected %q, got %q", grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenContributorNameRequired), st.Message())
	}
}

func TestSubmitBlueprintToRegistry_MissingContributorEmail(t *testing.T) {
	ctx := testutil.TestContext()
	svc := &CoreService{
		blueprintSvc: &blueprints.Service{},
	}

	req := &pb.SubmitBlueprintToRegistryRequest{
		Id:               uuid.New().String(),
		ContributorName:  "Contributor",
		ContributorEmail: "",
	}

	_, err := svc.SubmitBlueprintToRegistry(ctx, req)
	if err == nil {
		t.Fatalf("expected error for missing contributor email")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenContributorEmailRequired)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}
	if st.Message() != grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenContributorEmailRequired) {
		t.Errorf("message mismatch: expected %q, got %q", grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenContributorEmailRequired), st.Message())
	}
}

func TestCreateDeviceModelWithBlueprint_MissingName(t *testing.T) {
	ctx := testutil.TestContext()
	svc := &CoreService{
		blueprintSvc: &blueprints.Service{},
	}

	req := &pb.CreateDeviceModelWithBlueprintRequest{
		ManufacturerId: uuid.New().String(),
		Name:           "",
		Version:        "1.0.0",
		DecoderScript:  []byte(`{"codec":"test"}`),
	}

	_, err := svc.CreateDeviceModelWithBlueprint(ctx, req)
	if err == nil {
		t.Fatalf("expected error for missing name")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}
}

func TestCreateDeviceModelWithBlueprint_InvalidJSON(t *testing.T) {
	ctx := testutil.TestContext()
	svc := &CoreService{
		blueprintSvc: &blueprints.Service{},
	}

	req := &pb.CreateDeviceModelWithBlueprintRequest{
		ManufacturerId: uuid.New().String(),
		Name:           "Test Model",
		Version:        "1.0.0",
		DecoderScript:  []byte(`not valid json`),
	}

	_, err := svc.CreateDeviceModelWithBlueprint(ctx, req)
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInvalid)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}
}

func TestDecodePreview_MissingPayload(t *testing.T) {
	ctx := testutil.TestContext()
	svc := &CoreService{
		blueprintSvc: &blueprints.Service{},
	}

	req := &pb.DecodePreviewRequest{
		BlueprintId: uuid.New().String(),
		Payload:     nil,
		FormatId:    0,
	}

	_, err := svc.DecodePreview(ctx, req)
	if err == nil {
		t.Fatalf("expected error for missing payload")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewPayloadRequired)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}
}

func TestDecodePreview_InvalidFormatID(t *testing.T) {
	ctx := testutil.TestContext()
	svc := &CoreService{
		blueprintSvc: &blueprints.Service{},
	}

	req := &pb.DecodePreviewRequest{
		BlueprintId: uuid.New().String(),
		Payload:     []byte{0x01, 0x02},
		FormatId:    256,
	}

	_, err := svc.DecodePreview(ctx, req)
	if err == nil {
		t.Fatalf("expected error for invalid format ID")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewInvalidFormatID)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}
}

// mockBlueprintSvcForCreate implements grpcservices.BlueprintService by embedding
// a nil interface and overriding only CreateDeviceModel.
type mockBlueprintSvcForCreate struct {
	grpcservices.BlueprintService
	err error
}

func (m *mockBlueprintSvcForCreate) CreateDeviceModel(_ context.Context, _ *grpcservices.DeviceModelCreateRequest) (*models.DeviceModel, error) {
	return nil, m.err
}

func TestCreateDeviceModel_InvalidModelCode(t *testing.T) {
	ctx := testutil.TestContext()
	svc := &CoreService{
		blueprintSvc: &mockBlueprintSvcForCreate{err: blueprints.ErrInvalidModelCode},
		log:          &mockLogger{},
	}

	req := &pb.CreateDeviceModelRequest{
		ManufacturerId: uuid.New().String(),
		Name:           "Robin M",
		Code:           "Robin M",
	}

	_, err := svc.CreateDeviceModel(ctx, req)
	if err == nil {
		t.Fatalf("expected error for invalid model code")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidModelCode)
	if st.Code() != expectedCode {
		t.Errorf("code mismatch: expected %v, got %v", expectedCode, st.Code())
	}

	expectedMsg := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidModelCode)
	if st.Message() != expectedMsg {
		t.Errorf("message mismatch: expected %q, got %q", expectedMsg, st.Message())
	}
}
