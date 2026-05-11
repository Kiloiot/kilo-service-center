// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strconv"

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

// Manufacturer handlers

// CreateManufacturer creates a new manufacturer.
func (s *CoreService) CreateManufacturer(ctx context.Context, req *pb.CreateManufacturerRequest) (*pb.CreateManufacturerResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	createReq := &grpcservices.ManufacturerCreateRequest{
		Name:    req.Name,
		Website: req.Website,
	}

	manufacturer, err := s.blueprintSvc.CreateManufacturer(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, "create manufacturer failed", "name", req.Name, "error", err)
		switch {
		case errors.Is(err, storage.ErrDuplicateKey):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNameExists),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNameExists))
		case errors.Is(err, blueprints.ErrTenantIDRequired):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateManufacturerFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateManufacturerFailed))
		}
	}

	return &pb.CreateManufacturerResponse{
		Manufacturer: manufacturerToProto(manufacturer),
	}, nil
}

// GetManufacturer returns a manufacturer by ID.
func (s *CoreService) GetManufacturer(ctx context.Context, req *pb.GetManufacturerRequest) (*pb.GetManufacturerResponse, error) {
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

	manufacturer, err := s.blueprintSvc.GetManufacturer(ctx, id)
	if err != nil {
		s.log.ErrorContext(ctx, "get manufacturer failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound))
	}

	return &pb.GetManufacturerResponse{
		Manufacturer: manufacturerToProto(manufacturer),
	}, nil
}

// UpdateManufacturer updates a manufacturer.
func (s *CoreService) UpdateManufacturer(ctx context.Context, req *pb.UpdateManufacturerRequest) (*pb.UpdateManufacturerResponse, error) {
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

	updateReq := &grpcservices.ManufacturerUpdateRequest{}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}
	if req.Website != "" {
		updateReq.Website = &req.Website
	}
	if req.ContactEmail != "" {
		updateReq.ContactEmail = &req.ContactEmail
	}

	manufacturer, err := s.blueprintSvc.UpdateManufacturer(ctx, id, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, "update manufacturer failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateManufacturerFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateManufacturerFailed))
	}

	return &pb.UpdateManufacturerResponse{
		Manufacturer: manufacturerToProto(manufacturer),
	}, nil
}

// DeleteManufacturer deletes a manufacturer.
func (s *CoreService) DeleteManufacturer(ctx context.Context, req *pb.DeleteManufacturerRequest) (*pb.DeleteManufacturerResponse, error) {
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

	if err := s.blueprintSvc.DeleteManufacturer(ctx, id); err != nil {
		s.log.ErrorContext(ctx, "delete manufacturer failed", "id", req.Id, "error", err)
		if errors.Is(err, storage.ErrForeignKeyViolation) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerHasModels),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerHasModels))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteManufacturerFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteManufacturerFailed))
	}

	return &pb.DeleteManufacturerResponse{Success: true}, nil
}

// ListManufacturers returns a list of manufacturers.
func (s *CoreService) ListManufacturers(ctx context.Context, req *pb.ListManufacturersRequest) (*pb.ListManufacturersResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	manufacturers, total, err := s.blueprintSvc.ListManufacturers(ctx, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list manufacturers failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListManufacturersFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListManufacturersFailed))
	}

	var pbManufacturers []*pb.Manufacturer
	for _, m := range manufacturers {
		pbManufacturers = append(pbManufacturers, manufacturerToProto(m))
	}

	var nextToken string
	if offset+limit < int(total) {
		nextToken = generatePaginationToken(offset + limit)
	}

	if total > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	totalCount := int32(total) //nolint:gosec // bounds checked above
	return &pb.ListManufacturersResponse{
		Manufacturers: pbManufacturers,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// Device Model handlers

// CreateDeviceModel creates a new device model.
func (s *CoreService) CreateDeviceModel(ctx context.Context, req *pb.CreateDeviceModelRequest) (*pb.CreateDeviceModelResponse, error) {
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

	createReq := &grpcservices.DeviceModelCreateRequest{
		ManufacturerID: manufacturerID,
		Name:           req.Name,
		Code:           req.Code,
		TypeEUI:        typeEUI,
		Description:    req.Description,
	}

	model, err := s.blueprintSvc.CreateDeviceModel(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, "create device model failed", "name", req.Name, "error", err)
		switch {
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

	return &pb.CreateDeviceModelResponse{
		DeviceModel: deviceModelToProto(model),
	}, nil
}

// GetDeviceModel returns a device model by ID.
func (s *CoreService) GetDeviceModel(ctx context.Context, req *pb.GetDeviceModelRequest) (*pb.GetDeviceModelResponse, error) {
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
		s.log.ErrorContext(ctx, "get device model failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
	}

	return &pb.GetDeviceModelResponse{
		DeviceModel: deviceModelToProto(model),
	}, nil
}

// UpdateDeviceModel updates a device model.
func (s *CoreService) UpdateDeviceModel(ctx context.Context, req *pb.UpdateDeviceModelRequest) (*pb.UpdateDeviceModelResponse, error) {
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

	updateReq := &grpcservices.DeviceModelUpdateRequest{}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}

	model, err := s.blueprintSvc.UpdateDeviceModel(ctx, id, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, "update device model failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateDeviceModelFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateDeviceModelFailed))
	}

	return &pb.UpdateDeviceModelResponse{
		DeviceModel: deviceModelToProto(model),
	}, nil
}

// DeleteDeviceModel deletes a device model.
func (s *CoreService) DeleteDeviceModel(ctx context.Context, req *pb.DeleteDeviceModelRequest) (*pb.DeleteDeviceModelResponse, error) {
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

	if err := s.blueprintSvc.DeleteDeviceModel(ctx, id); err != nil {
		s.log.ErrorContext(ctx, "delete device model failed", "id", req.Id, "error", err)
		if errors.Is(err, storage.ErrForeignKeyViolation) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelHasBlueprints),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelHasBlueprints))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteDeviceModelFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteDeviceModelFailed))
	}

	return &pb.DeleteDeviceModelResponse{Success: true}, nil
}

// ListDeviceModels returns a list of device models.
func (s *CoreService) ListDeviceModels(ctx context.Context, req *pb.ListDeviceModelsRequest) (*pb.ListDeviceModelsResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
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

	models, total, err := s.blueprintSvc.ListDeviceModels(ctx, manufacturerID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list device models failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListDeviceModelsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListDeviceModelsFailed))
	}

	var pbModels []*pb.DeviceModel
	for _, m := range models {
		pbModels = append(pbModels, deviceModelToProto(m))
	}

	var nextToken string
	if offset+limit < int(total) {
		nextToken = generatePaginationToken(offset + limit)
	}

	if total > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	totalCount := int32(total) //nolint:gosec // bounds checked above
	return &pb.ListDeviceModelsResponse{
		DeviceModels:  pbModels,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// Blueprint handlers

// CreateBlueprint creates a new blueprint.
func (s *CoreService) CreateBlueprint(ctx context.Context, req *pb.CreateBlueprintRequest) (*pb.CreateBlueprintResponse, error) {
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
	createReq := &grpcservices.BlueprintCreateRequest{
		DeviceModelID: deviceModelID,
		Version:       req.Version,
		TypeEUI:       nil,               // Proto doesn't have TypeEUI; can be set later
		SpecJSON:      req.DecoderScript, // Map DecoderScript to SpecJSON
		IsDefault:     req.IsDefault,
	}

	blueprint, err := s.blueprintSvc.CreateBlueprint(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, "create blueprint failed", "version", req.Version, "error", err)
		switch {
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

	return &pb.CreateBlueprintResponse{
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// GetBlueprint returns a blueprint by ID.
func (s *CoreService) GetBlueprint(ctx context.Context, req *pb.GetBlueprintRequest) (*pb.GetBlueprintResponse, error) {
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
		s.log.ErrorContext(ctx, "get blueprint failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	}

	return &pb.GetBlueprintResponse{
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// UpdateBlueprint updates a blueprint.
func (s *CoreService) UpdateBlueprint(ctx context.Context, req *pb.UpdateBlueprintRequest) (*pb.UpdateBlueprintResponse, error) {
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
		s.log.ErrorContext(ctx, "update blueprint failed", "id", req.Id, "error", err)
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

	return &pb.UpdateBlueprintResponse{
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// DeleteBlueprint deletes a blueprint.
func (s *CoreService) DeleteBlueprint(ctx context.Context, req *pb.DeleteBlueprintRequest) (*pb.DeleteBlueprintResponse, error) {
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

	if err := s.blueprintSvc.DeleteBlueprint(ctx, id); err != nil {
		s.log.ErrorContext(ctx, "delete blueprint failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteBlueprintFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteBlueprintFailed))
	}

	return &pb.DeleteBlueprintResponse{Success: true}, nil
}

// ListBlueprints returns a list of blueprints.
func (s *CoreService) ListBlueprints(ctx context.Context, req *pb.ListBlueprintsRequest) (*pb.ListBlueprintsResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
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

	blueprints, total, err := s.blueprintSvc.ListBlueprints(ctx, deviceModelID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list blueprints failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListBlueprintsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListBlueprintsFailed))
	}

	var pbBlueprints []*pb.Blueprint
	for _, b := range blueprints {
		pbBlueprints = append(pbBlueprints, blueprintToProto(b))
	}

	var nextToken string
	if offset+limit < int(total) {
		nextToken = generatePaginationToken(offset + limit)
	}

	if total > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	totalCount := int32(total) //nolint:gosec // bounds checked above
	return &pb.ListBlueprintsResponse{
		Blueprints:    pbBlueprints,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// SetDefaultBlueprint sets a blueprint as the default for its device model.
func (s *CoreService) SetDefaultBlueprint(ctx context.Context, req *pb.SetDefaultBlueprintRequest) (*pb.SetDefaultBlueprintResponse, error) {
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

	if err := s.blueprintSvc.SetDefaultBlueprint(ctx, id); err != nil {
		s.log.ErrorContext(ctx, "set default blueprint failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSetDefaultBlueprintFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSetDefaultBlueprintFailed))
	}

	// Fetch the updated blueprint to include in the response
	blueprint, err := s.blueprintSvc.GetBlueprint(ctx, id)
	if err != nil {
		s.log.ErrorContext(ctx, "get blueprint after set default failed", "id", req.Id, "error", err)
		// Still return success since the set operation succeeded
		return &pb.SetDefaultBlueprintResponse{Success: true}, nil
	}

	return &pb.SetDefaultBlueprintResponse{
		Success:   true,
		Blueprint: blueprintToProto(blueprint),
	}, nil
}

// SubmitBlueprintToRegistry submits a blueprint to an external registry.
func (s *CoreService) SubmitBlueprintToRegistry(ctx context.Context, req *pb.SubmitBlueprintToRegistryRequest) (*pb.SubmitBlueprintToRegistryResponse, error) {
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
		s.log.ErrorContext(ctx, "submit blueprint to registry failed", "id", req.Id, "error", err)
		return nil, s.mapRegistryError(err)
	}

	return &pb.SubmitBlueprintToRegistryResponse{
		Success:    true,
		PrUrl:      result.PRUrl,
		CommitSha:  result.CommitSHA,
		BranchName: result.BranchName,
	}, nil
}

// CreateDeviceModelWithBlueprint creates a device model and default blueprint atomically.
func (s *CoreService) CreateDeviceModelWithBlueprint(ctx context.Context, req *pb.CreateDeviceModelWithBlueprintRequest) (*pb.CreateDeviceModelWithBlueprintResponse, error) {
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

	svcReq := &grpcservices.DeviceModelWithBlueprintRequest{
		ManufacturerID: manufacturerID,
		Name:           req.Name,
		Version:        req.Version,
		DecoderScript:  req.DecoderScript,
	}

	model, blueprint, err := s.blueprintSvc.CreateDeviceModelWithBlueprint(ctx, svcReq)
	if err != nil {
		s.log.ErrorContext(ctx, "create device model with blueprint failed", "name", req.Name, "error", err)
		switch {
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

	return &pb.CreateDeviceModelWithBlueprintResponse{
		DeviceModel: deviceModelToProto(model),
		Blueprint:   blueprintToProto(blueprint),
	}, nil
}

// DecodePreview runs the blueprint decoder on a payload for preview.
func (s *CoreService) DecodePreview(ctx context.Context, req *pb.DecodePreviewRequest) (*pb.DecodePreviewResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.BlueprintId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintIDRequired))
	}
	if len(req.Payload) == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewPayloadRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPreviewPayloadRequired))
	}

	blueprintID, err := uuid.Parse(req.BlueprintId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBlueprintIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBlueprintIDFormat))
	}

	if req.FormatId > 255 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewInvalidFormatID),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPreviewInvalidFormatID))
	}

	result, err := s.blueprintSvc.DecodePreview(ctx, blueprintID, req.Payload, uint8(req.FormatId)) //nolint:gosec // bounds checked above
	if err != nil {
		s.log.ErrorContext(ctx, "decode preview failed", "blueprint_id", req.BlueprintId, "error", err)
		switch {
		case errors.Is(err, blueprints.ErrBlueprintNotFound):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPreviewDecodeFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPreviewDecodeFailed))
		}
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

// mapRegistryError translates service errors to gRPC catalog tokens.
func (s *CoreService) mapRegistryError(err error) error {
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

func manufacturerToProto(m *models.Manufacturer) *pb.Manufacturer {
	if m == nil {
		return nil
	}
	pbMfr := &pb.Manufacturer{
		Id:         m.ID.String(),
		Name:       m.Name,
		CreatedAt:  timestamppb.New(m.CreatedAt),
		UpdatedAt:  timestamppb.New(m.UpdatedAt),
		TenantId:   strconv.FormatInt(m.TenantID, 10), // int64→string
		IsVerified: m.IsVerified,
		ModelCount: int32(min(m.ModelCount, math.MaxInt32)), //nolint:gosec // count value bounded
	}
	if m.Website != nil {
		pbMfr.Website = *m.Website
	}
	return pbMfr
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
		TypeEui:        hex.EncodeToString(m.TypeEUI), // Convert bytes to hex string
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
		TenantId:       strconv.FormatInt(m.TenantID, 10),           // int64→string
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
		TypeEui:          hex.EncodeToString(b.TypeEUI), // []byte→hex string
		SpecJson:         []byte(b.SpecJSON),            // json.RawMessage→bytes
		RegistryVerified: b.RegistryVerified,
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
