// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/blueprints"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// BlueprintHandlers serves the manufacturer, device model and blueprint RPCs.
type BlueprintHandlers struct {
	blueprintSvc grpcservices.BlueprintService
	endpointSvc  grpcservices.EndpointService
	audit        AuditRecorder
	log          logger.Logger
}

// BlueprintHandlerDeps wires BlueprintHandlers; a nil Blueprints service
// answers its RPCs as not configured.
type BlueprintHandlerDeps struct {
	Blueprints grpcservices.BlueprintService
	Endpoints  grpcservices.EndpointService
}

// NewBlueprintHandlers builds the group; recorder records the catalog changes.
func NewBlueprintHandlers(d BlueprintHandlerDeps, recorder AuditRecorder, log logger.Logger) *BlueprintHandlers {
	return &BlueprintHandlers{
		blueprintSvc: d.Blueprints,
		endpointSvc:  d.Endpoints,
		audit:        recorder,
		log:          log,
	}
}

// Manufacturer handlers

// Device Model handlers

// Blueprint handlers

// CreateBlueprint creates a new blueprint.
func (s *BlueprintHandlers) CreateBlueprint(ctx context.Context, req *pb.CreateBlueprintRequest) (*pb.CreateBlueprintResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.DeviceModelId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelIDRequired))
	}
	if req.Version == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenVersionRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenVersionRequired))
	}
	if len(req.DecoderScript) == 0 || !json.Valid(req.DecoderScript) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInvalid))
	}

	deviceModelID, err := uuid.Parse(req.DeviceModelId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat))
	}

	// Proto uses Name/Description/DecoderScript; service uses TypeEUI/SpecJSON
	// Map DecoderScript to SpecJSON for DB compatibility
	if req.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	createReq := &grpcservices.BlueprintCreateRequest{
		DeviceModelID: deviceModelID,
		Version:       req.Version,
		TypeEUI:       nil,               // Proto doesn't have TypeEUI; can be set later
		SpecJSON:      req.DecoderScript, // Map DecoderScript to SpecJSON
		IsDefault:     req.IsDefault,
		IsSystem:      req.IsSystem,
	}

	blueprint, err := s.blueprintSvc.CreateBlueprint(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateBlueprintFailed, logger.FieldVersion, req.Version, logger.FieldError, err)
		switch {
		case errors.Is(err, blueprints.ErrOwnershipMismatch):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSystemOwnershipDeviceModelMismatch),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSystemOwnershipDeviceModelMismatch))
		case errors.Is(err, blueprints.ErrDeviceModelNotFound):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
		case errors.Is(err, blueprints.ErrMissingTypeEUI):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInvalid),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInvalid))
		case errors.Is(err, blueprints.ErrInvalidTypeEUIFormat):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
		case errors.Is(err, storage.ErrDuplicateKey):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNameExists),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNameExists))
		case errors.Is(err, blueprints.ErrTenantIDRequired):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateBlueprintFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateBlueprintFailed))
		}
	}

	s.recordCatalogChange(ctx, blueprintCreated, blueprint.ID, blueprint.Version)
	return &pb.CreateBlueprintResponse{
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// GetBlueprint returns a blueprint by ID.
func (s *BlueprintHandlers) GetBlueprint(ctx context.Context, req *pb.GetBlueprintRequest) (*pb.GetBlueprintResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	blueprint, err := s.blueprintSvc.GetBlueprint(ctx, id)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetBlueprintFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	}

	return &pb.GetBlueprintResponse{
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// UpdateBlueprint updates a blueprint.
func (s *BlueprintHandlers) UpdateBlueprint(ctx context.Context, req *pb.UpdateBlueprintRequest) (*pb.UpdateBlueprintResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	// System-catalog mutation is admin-only; resolve the target's ownership to decide.
	existing, err := s.blueprintSvc.GetBlueprint(ctx, id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	}
	if existing.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	// Proto uses Name/Description/DecoderScript; service uses Version/TypeEUI/SpecJSON
	updateReq := &grpcservices.BlueprintUpdateRequest{}
	if req.Version != "" {
		updateReq.Version = &req.Version
	}
	if len(req.DecoderScript) > 0 {
		updateReq.SpecJSON = req.DecoderScript // Map DecoderScript to SpecJSON
	}

	blueprint, err := s.blueprintSvc.UpdateBlueprint(ctx, id, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogUpdateBlueprintFailed, logger.FieldID, req.Id, logger.FieldError, err)
		switch {
		case errors.Is(err, blueprints.ErrBlueprintNotFound):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
		case errors.Is(err, blueprints.ErrMissingTypeEUI):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInvalid),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInvalid))
		case errors.Is(err, blueprints.ErrInvalidTypeEUIFormat):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateBlueprintFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBlueprintFailed))
		}
	}

	s.recordCatalogChange(ctx, blueprintUpdated, blueprint.ID, blueprint.Version)
	return &pb.UpdateBlueprintResponse{
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// DeleteBlueprint deletes a blueprint.
func (s *BlueprintHandlers) DeleteBlueprint(ctx context.Context, req *pb.DeleteBlueprintRequest) (*pb.DeleteBlueprintResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	// System-catalog mutation is admin-only; resolve the target's ownership to decide.
	existing, err := s.blueprintSvc.GetBlueprint(ctx, id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	}
	if existing.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	if err := s.blueprintSvc.DeleteBlueprint(ctx, id); err != nil {
		s.log.ErrorContext(ctx, LogDeleteBlueprintFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteBlueprintFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteBlueprintFailed))
	}

	s.recordCatalogChange(ctx, blueprintDeleted, existing.ID, existing.Version)
	return &pb.DeleteBlueprintResponse{Success: true}, nil
}

// ListBlueprints returns a list of blueprints.
func (s *BlueprintHandlers) ListBlueprints(ctx context.Context, req *pb.ListBlueprintsRequest) (*pb.ListBlueprintsResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	page, err := readPage(clampPageSize(req.PageSize), req.PageToken)
	if err != nil {
		return nil, err
	}

	var deviceModelID *uuid.UUID
	if req.DeviceModelId != "" {
		id, err := uuid.Parse(req.DeviceModelId)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat))
		}
		deviceModelID = &id
	}

	blueprints, total, err := s.blueprintSvc.ListBlueprints(ctx, req.IsSystem, deviceModelID, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListBlueprintsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListBlueprintsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListBlueprintsFailed))
	}

	var pbBlueprints []*pb.Blueprint
	for _, b := range blueprints {
		pbBlueprints = append(pbBlueprints, blueprintToProto(b))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListBlueprintsResponse{
		Blueprints:    pbBlueprints,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

// SetDefaultBlueprint sets a blueprint as the default for its device model.
func (s *BlueprintHandlers) SetDefaultBlueprint(ctx context.Context, req *pb.SetDefaultBlueprintRequest) (*pb.SetDefaultBlueprintResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	// System-catalog mutation is admin-only; resolve the target's ownership to decide.
	existing, err := s.blueprintSvc.GetBlueprint(ctx, id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	}
	if existing.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	if err := s.blueprintSvc.SetDefaultBlueprint(ctx, id); err != nil {
		s.log.ErrorContext(ctx, LogSetDefaultBlueprintFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSetDefaultBlueprintFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSetDefaultBlueprintFailed))
	}
	s.recordCatalogChange(ctx, blueprintUpdated, existing.ID, existing.Version)

	// Fetch the updated blueprint to include in the response
	blueprint, err := s.blueprintSvc.GetBlueprint(ctx, id)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetBlueprintAfterSetDefaultFailed, logger.FieldID, req.Id, logger.FieldError, err)
		// Still return success since the set operation succeeded
		return &pb.SetDefaultBlueprintResponse{Success: true}, nil
	}

	return &pb.SetDefaultBlueprintResponse{
		Success:   true,
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// SubmitBlueprintToRegistry submits a blueprint to an external registry.
func (s *BlueprintHandlers) SubmitBlueprintToRegistry(ctx context.Context, req *pb.SubmitBlueprintToRegistryRequest) (*pb.SubmitBlueprintToRegistryResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	// Validate contributor fields (handler validates request shape)
	if req.ContributorName == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenContributorNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenContributorNameRequired))
	}
	if req.ContributorEmail == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenContributorEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenContributorEmailRequired))
	}

	// Pass request fields to service
	result, err := s.blueprintSvc.SubmitToRegistry(ctx, id, &grpcservices.RegistrySubmitRequest{
		ContributorName:  req.ContributorName,
		ContributorEmail: req.ContributorEmail,
		Notes:            req.Notes,
	})
	if err != nil {
		s.log.ErrorContext(ctx, LogSubmitBlueprintToRegistryFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, s.mapRegistryError(err)
	}

	return &pb.SubmitBlueprintToRegistryResponse{
		Success:    true,
		PrUrl:      result.PRUrl,
		CommitSha:  result.CommitSHA,
		BranchName: result.BranchName,
	}, nil
}

// DecodePreview runs the blueprint decoder on a payload for preview.
func (s *BlueprintHandlers) DecodePreview(ctx context.Context, req *pb.DecodePreviewRequest) (*pb.DecodePreviewResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if len(req.Payload) == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewPayloadRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPreviewPayloadRequired))
	}
	if req.FormatId > math.MaxUint8 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewInvalidFormatID),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPreviewInvalidFormatID))
	}

	// Source is a oneof: a saved blueprint id, or an inline spec for previewing before save.
	var (
		result *grpcservices.DecodePreviewResult
		err    error
	)
	switch src := req.GetSource().(type) {
	case *pb.DecodePreviewRequest_BlueprintId:
		blueprintID, parseErr := uuid.Parse(src.BlueprintId)
		if parseErr != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBlueprintIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBlueprintIDFormat))
		}
		result, err = s.blueprintSvc.DecodePreview(ctx, blueprintID, req.Payload, uint8(req.FormatId)) //nolint:gosec // bounds checked above
	case *pb.DecodePreviewRequest_SpecJson:
		if len(src.SpecJson) == 0 {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSpecJSONEmpty),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSpecJSONEmpty))
		}
		result, err = s.blueprintSvc.DecodePreviewInline(ctx, src.SpecJson, req.Payload, uint8(req.FormatId)) //nolint:gosec // bounds checked above
	default:
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDecodeSourceRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDecodeSourceRequired))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogDecodePreviewFailed, logger.FieldError, err)
		if errors.Is(err, blueprints.ErrBlueprintNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewDecodeFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPreviewDecodeFailed))
	}

	return &pb.DecodePreviewResponse{
		Success:          result.Success,
		DecodedPayload:   result.DecodedPayload,
		ErrorCode:        result.ErrorCode,
		ErrorDetail:      result.ErrorDetail,
		FormatId:         uint32(result.FormatID),
		BlueprintVersion: result.BlueprintVersion,
	}, nil
}

// BulkAssignBlueprint re-materializes a blueprint onto snapshot-bearing endpoints; catalog-default followers are skipped.
func (s *BlueprintHandlers) BulkAssignBlueprint(ctx context.Context, req *pb.BulkAssignBlueprintRequest) (*pb.BulkAssignBlueprintResponse, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	blueprintID, err := s.validateBulkAssignPreconditions(req)
	if err != nil {
		return nil, err
	}

	// Validate before any mutation: no rollback, so reject a System blueprint up front.
	if req.SetAsDefault {
		if err := s.validateBulkAssignSetDefault(ctx, blueprintID); err != nil {
			return nil, err
		}
	}

	euis, err := s.resolveBulkAssignTargets(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	affected, err := s.rematerializeSnapshotOnEndpoints(ctx, tenantID, req.BlueprintId, euis)
	if err != nil {
		return nil, err
	}

	if req.SetAsDefault {
		// Eligibility (exists, not System) was validated before the mutation loop.
		if err := s.blueprintSvc.SetDefaultBlueprint(ctx, blueprintID); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSetDefaultBlueprintFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSetDefaultBlueprintFailed))
		}
	}

	return &pb.BulkAssignBlueprintResponse{AffectedCount: affected}, nil
}

func (s *BlueprintHandlers) validateBulkAssignPreconditions(req *pb.BulkAssignBlueprintRequest) (uuid.UUID, error) {
	if s.blueprintSvc == nil || s.endpointSvc == nil {
		return uuid.Nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	if req.BlueprintId == "" {
		return uuid.Nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintIDRequired))
	}
	blueprintID, parseErr := uuid.Parse(req.BlueprintId)
	if parseErr != nil {
		return uuid.Nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBlueprintIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBlueprintIDFormat))
	}
	return blueprintID, nil
}

// System blueprint can't be a tenant default; reject before any mutation (no rollback).
func (s *BlueprintHandlers) validateBulkAssignSetDefault(ctx context.Context, blueprintID uuid.UUID) error {
	bp, bpErr := s.blueprintSvc.GetBlueprint(ctx, blueprintID)
	if bpErr != nil || bp == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	}
	if bp.IsSystem {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSystemBlueprintDefaultNotAllowed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSystemBlueprintDefaultNotAllowed))
	}
	return nil
}

// Explicit ep_euis win over device_model_id (which expands to the tenant's snapshot-bearing endpoints).
func (s *BlueprintHandlers) resolveBulkAssignTargets(ctx context.Context, tenantID int64, req *pb.BulkAssignBlueprintRequest) ([]models.EUI, error) {
	if len(req.EpEuis) > 0 {
		euis := make([]models.EUI, 0, len(req.EpEuis))
		for _, e := range req.EpEuis {
			euis = append(euis, models.EUIFromString(e))
		}
		return euis, nil
	}
	if req.DeviceModelId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAssignTargetRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAssignTargetRequired))
	}
	modelID, parseErr := uuid.Parse(req.DeviceModelId)
	if parseErr != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat))
	}
	targets, listErr := s.endpointSvc.ListByModelWithSnapshot(ctx, tenantID, modelID)
	if listErr != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListEndpointsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListEndpointsFailed))
	}
	euis := make([]models.EUI, 0, len(targets))
	for _, ep := range targets {
		euis = append(euis, ep.EUI)
	}
	return euis, nil
}

// Skips missing endpoints and catalog-default followers; returns the affected count.
func (s *BlueprintHandlers) rematerializeSnapshotOnEndpoints(ctx context.Context, tenantID int64, blueprintID string, euis []models.EUI) (int32, error) {
	var affected int32
	for _, eui := range euis {
		// Load the full endpoint (list columns are partial and would clobber on Update).
		ep, getErr := s.endpointSvc.GetByEUI(ctx, eui[:], tenantID)
		if getErr != nil || ep == nil {
			continue // best-effort per target: skip missing endpoints
		}
		if len(ep.BlueprintSnapshot) == 0 {
			continue // skip catalog-default followers
		}
		if err := applyBlueprintSnapshot(ctx, s.log, s.blueprintSvc, ep, blueprintID); err != nil {
			return 0, err // same blueprint for all targets — a fetch/validation failure is fatal
		}
		if _, updErr := s.endpointSvc.Update(ctx, ep); updErr != nil {
			return 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateEndpointFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointFailed))
		}
		affected++
	}
	return affected, nil
}

// mapRegistryError translates service errors to gRPC catalog tokens.
func (s *BlueprintHandlers) mapRegistryError(err error) error {
	switch {
	case errors.Is(err, blueprints.ErrRegistryDisabled):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryProviderDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryProviderDisabled))
	case errors.Is(err, blueprints.ErrRegistryAPIURLRequired):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryAPIURLRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryAPIURLRequired))
	case errors.Is(err, blueprints.ErrAlreadySubmitted):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintAlreadySubmitted),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintAlreadySubmitted))
	case errors.Is(err, blueprints.ErrBlueprintNotFound):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	case errors.Is(err, blueprints.ErrRegistryAuthFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryAuthFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryAuthFailed))
	case errors.Is(err, blueprints.ErrRegistryPermissionDenied):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryPermissionDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryPermissionDenied))
	case errors.Is(err, blueprints.ErrRegistryRateLimited):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryRateLimited),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryRateLimited))
	case errors.Is(err, blueprints.ErrRegistryAPIError):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryAPIError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryAPIError))
	case errors.Is(err, blueprints.ErrBranchCreateFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryBranchCreateFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryBranchCreateFailed))
	case errors.Is(err, blueprints.ErrCommitFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryCommitFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryCommitFailed))
	case errors.Is(err, blueprints.ErrSubmissionCreateFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrySubmissionCreateFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrySubmissionCreateFailed))
	case errors.Is(err, blueprints.ErrRegistryVersionAlreadyExists):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryVersionAlreadyExists),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryVersionAlreadyExists))
	case errors.Is(err, blueprints.ErrInvalidRegistryPathSegment):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistryInvalidPathSegment),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistryInvalidPathSegment))
	case errors.Is(err, blueprints.ErrDeviceModelNotFound):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
	case errors.Is(err, blueprints.ErrManufacturerNotFound):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound))
	default:
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSubmitBlueprintToRegistryFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSubmitBlueprintToRegistryFailed))
	}
}

// Helper functions

// formatOptionalTenantID renders a nullable tenant id ("" for System rows).
func formatOptionalTenantID(t *int64) string {
	if t == nil {
		return ""
	}
	return strconv.FormatInt(*t, 10)
}

func blueprintToProto(b *models.Blueprint) *pb.Blueprint {
	if b == nil {
		return nil
	}
	pbBp := &pb.Blueprint{
		Id:               b.ID.String(),
		DeviceModelId:    b.DeviceModelID.String(),
		Version:          b.Version,
		DecoderScript:    []byte(b.SpecJSON), // Map SpecJSON to DecoderScript (legacy)
		IsDefault:        b.IsDefault,
		CreatedAt:        timestamppb.New(b.CreatedAt),
		UpdatedAt:        timestamppb.New(b.UpdatedAt),
		TypeEui:          mioty.FormatEUIBytes(b.TypeEUI), // []byte→hex string
		SpecJson:         []byte(b.SpecJSON),              // json.RawMessage→bytes
		RegistryVerified: b.RegistryVerified,
		IsSystem:         b.IsSystem,
	}
	// Optional fields (pointers)
	if b.RegistryRepo != nil {
		pbBp.RegistryRepo = *b.RegistryRepo
	}
	if b.RegistryCommit != nil {
		pbBp.RegistryCommitSha = *b.RegistryCommit
	}
	if b.RegistryPRURL != nil {
		pbBp.RegistryPrUrl = *b.RegistryPRURL
	}
	return pbBp
}
