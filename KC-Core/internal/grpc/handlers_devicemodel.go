package grpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"

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

// CreateDeviceModel creates a new device model.
func (s *BlueprintHandlers) CreateDeviceModel(ctx context.Context, req *pb.CreateDeviceModelRequest) (*pb.CreateDeviceModelResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.ManufacturerId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerIDRequired))
	}
	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	manufacturerID, err := uuid.Parse(req.ManufacturerId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidManufacturerIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidManufacturerIDFormat))
	}

	// Convert hex TypeEUI string to bytes (proto uses string, service uses []byte)
	var typeEUI []byte
	if req.TypeEui != "" {
		var err error
		typeEUI, err = hex.DecodeString(req.TypeEui)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
		}
		if len(typeEUI) != 8 {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
		}
	}

	if req.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	createReq := &grpcservices.DeviceModelCreateRequest{
		ManufacturerID: manufacturerID,
		Name:           req.Name,
		Code:           req.Code,
		TypeEUI:        typeEUI,
		Description:    req.Description,
		IsSystem:       req.IsSystem,
	}

	model, err := s.blueprintSvc.CreateDeviceModel(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateDeviceModelFailed, logger.FieldName, req.Name, logger.FieldError, err)
		switch {
		case errors.Is(err, blueprints.ErrOwnershipMismatch):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSystemOwnershipManufacturerMismatch),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSystemOwnershipManufacturerMismatch))
		case errors.Is(err, blueprints.ErrManufacturerNotFound):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound))
		case errors.Is(err, blueprints.ErrInvalidModelCode):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidModelCode),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidModelCode))
		case errors.Is(err, storage.ErrDuplicateKey):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNameExists),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNameExists))
		case errors.Is(err, blueprints.ErrTenantIDRequired):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateDeviceModelFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateDeviceModelFailed))
		}
	}

	s.recordCatalogChange(ctx, deviceModelCreated, model.ID, model.Name)
	return &pb.CreateDeviceModelResponse{
		DeviceModel: deviceModelToProto(model),
	}, nil
}

// GetDeviceModel returns a device model by ID.
func (s *BlueprintHandlers) GetDeviceModel(ctx context.Context, req *pb.GetDeviceModelRequest) (*pb.GetDeviceModelResponse, error) {
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

	model, err := s.blueprintSvc.GetDeviceModel(ctx, id)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetDeviceModelFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
	}

	return &pb.GetDeviceModelResponse{
		DeviceModel: deviceModelToProto(model),
	}, nil
}

// UpdateDeviceModel updates a device model.
func (s *BlueprintHandlers) UpdateDeviceModel(ctx context.Context, req *pb.UpdateDeviceModelRequest) (*pb.UpdateDeviceModelResponse, error) {
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
	existing, err := s.blueprintSvc.GetDeviceModel(ctx, id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
	}
	if existing.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	updateReq := &grpcservices.DeviceModelUpdateRequest{}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}

	model, err := s.blueprintSvc.UpdateDeviceModel(ctx, id, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogUpdateDeviceModelFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateDeviceModelFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateDeviceModelFailed))
	}

	s.recordCatalogChange(ctx, deviceModelUpdated, model.ID, model.Name)
	return &pb.UpdateDeviceModelResponse{
		DeviceModel: deviceModelToProto(model),
	}, nil
}

// DeleteDeviceModel deletes a device model.
func (s *BlueprintHandlers) DeleteDeviceModel(ctx context.Context, req *pb.DeleteDeviceModelRequest) (*pb.DeleteDeviceModelResponse, error) {
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
	existing, err := s.blueprintSvc.GetDeviceModel(ctx, id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
	}
	if existing.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	if err := s.blueprintSvc.DeleteDeviceModel(ctx, id); err != nil {
		s.log.ErrorContext(ctx, LogDeleteDeviceModelFailed, logger.FieldID, req.Id, logger.FieldError, err)
		if errors.Is(err, storage.ErrForeignKeyViolation) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelHasBlueprints),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelHasBlueprints))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteDeviceModelFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteDeviceModelFailed))
	}

	s.recordCatalogChange(ctx, deviceModelDeleted, existing.ID, existing.Name)
	return &pb.DeleteDeviceModelResponse{Success: true}, nil
}

// ListDeviceModels returns a list of device models.
func (s *BlueprintHandlers) ListDeviceModels(ctx context.Context, req *pb.ListDeviceModelsRequest) (*pb.ListDeviceModelsResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	page, err := readPage(clampPageSize(req.PageSize), req.PageToken)
	if err != nil {
		return nil, err
	}

	var manufacturerID *uuid.UUID
	if req.ManufacturerId != "" {
		id, err := uuid.Parse(req.ManufacturerId)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidManufacturerIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidManufacturerIDFormat))
		}
		manufacturerID = &id
	}

	models, total, err := s.blueprintSvc.ListDeviceModels(ctx, req.IsSystem, manufacturerID, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListDeviceModelsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListDeviceModelsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListDeviceModelsFailed))
	}

	var pbModels []*pb.DeviceModel
	for _, m := range models {
		pbModels = append(pbModels, deviceModelToProto(m))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListDeviceModelsResponse{
		DeviceModels:  pbModels,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

// CreateDeviceModelWithBlueprint creates a device model and default blueprint atomically.
func (s *BlueprintHandlers) CreateDeviceModelWithBlueprint(ctx context.Context, req *pb.CreateDeviceModelWithBlueprintRequest) (*pb.CreateDeviceModelWithBlueprintResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.ManufacturerId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerIDRequired))
	}
	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}
	if req.Version == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenVersionRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenVersionRequired))
	}
	if len(req.DecoderScript) == 0 || !json.Valid(req.DecoderScript) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInvalid))
	}

	manufacturerID, err := uuid.Parse(req.ManufacturerId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidManufacturerIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidManufacturerIDFormat))
	}

	if req.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	svcReq := &grpcservices.DeviceModelWithBlueprintRequest{
		ManufacturerID: manufacturerID,
		Name:           req.Name,
		Version:        req.Version,
		DecoderScript:  req.DecoderScript,
		IsSystem:       req.IsSystem,
	}

	model, blueprint, err := s.blueprintSvc.CreateDeviceModelWithBlueprint(ctx, svcReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateDeviceModelWithBlueprintFailed, logger.FieldName, req.Name, logger.FieldError, err)
		switch {
		case errors.Is(err, blueprints.ErrOwnershipMismatch):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSystemOwnershipManufacturerMismatch),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSystemOwnershipManufacturerMismatch))
		case errors.Is(err, blueprints.ErrManufacturerNotFound):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound))
		case errors.Is(err, blueprints.ErrMissingTypeEUI):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInvalid),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInvalid))
		case errors.Is(err, blueprints.ErrInvalidTypeEUIFormat):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
		case errors.Is(err, blueprints.ErrSlugGenerationFailed):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSlugGenerationFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSlugGenerationFailed))
		case errors.Is(err, blueprints.ErrTenantIDRequired):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAtomicModelBlueprintFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAtomicModelBlueprintFailed))
		}
	}

	s.recordCatalogChange(ctx, deviceModelCreated, model.ID, model.Name)
	s.recordCatalogChange(ctx, blueprintCreated, blueprint.ID, blueprint.Version)
	return &pb.CreateDeviceModelWithBlueprintResponse{
		DeviceModel: deviceModelToProto(model),
		Blueprint:   blueprintToProto(blueprint),
	}, nil
}

func deviceModelToProto(m *models.DeviceModel) *pb.DeviceModel {
	if m == nil {
		return nil
	}
	pbModel := &pb.DeviceModel{
		Id:             m.ID.String(),
		ManufacturerId: m.ManufacturerID.String(),
		Name:           m.Name,
		Code:           m.Code,
		TypeEui:        mioty.FormatEUIBytes(m.TypeEUI), // Convert bytes to hex string
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
		TenantId:       formatOptionalTenantID(m.TenantID),
		IsSystem:       m.IsSystem,
		BlueprintCount: int32(min(m.BlueprintCount, math.MaxInt32)), //nolint:gosec // count value bounded
	}
	if m.Description != nil {
		pbModel.Description = *m.Description
	}
	if m.DatasheetURL != nil {
		pbModel.DatasheetUrl = *m.DatasheetURL
	}
	return pbModel
}
