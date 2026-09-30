package grpc

import (
	"context"
	"errors"
	"math"

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

// CreateManufacturer creates a device manufacturer catalog entry.
func (s *BlueprintHandlers) CreateManufacturer(ctx context.Context, req *pb.CreateManufacturerRequest) (*pb.CreateManufacturerResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	if req.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	createReq := &grpcservices.ManufacturerCreateRequest{
		Name:     req.Name,
		Website:  req.Website,
		IsSystem: req.IsSystem,
	}

	manufacturer, err := s.blueprintSvc.CreateManufacturer(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateManufacturerFailed, logger.FieldName, req.Name, logger.FieldError, err)
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

	s.recordCatalogChange(ctx, manufacturerCreated, manufacturer.ID, manufacturer.Name)
	return &pb.CreateManufacturerResponse{
		Manufacturer: manufacturerToProto(manufacturer),
	}, nil
}

// GetManufacturer returns a manufacturer by ID.
func (s *BlueprintHandlers) GetManufacturer(ctx context.Context, req *pb.GetManufacturerRequest) (*pb.GetManufacturerResponse, error) {
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
		s.log.ErrorContext(ctx, LogGetManufacturerFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound))
	}

	return &pb.GetManufacturerResponse{
		Manufacturer: manufacturerToProto(manufacturer),
	}, nil
}

// UpdateManufacturer updates a manufacturer.
func (s *BlueprintHandlers) UpdateManufacturer(ctx context.Context, req *pb.UpdateManufacturerRequest) (*pb.UpdateManufacturerResponse, error) {
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
	existing, err := s.blueprintSvc.GetManufacturer(ctx, id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound))
	}
	if existing.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
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
		s.log.ErrorContext(ctx, LogUpdateManufacturerFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateManufacturerFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateManufacturerFailed))
	}

	s.recordCatalogChange(ctx, manufacturerUpdated, manufacturer.ID, manufacturer.Name)
	return &pb.UpdateManufacturerResponse{
		Manufacturer: manufacturerToProto(manufacturer),
	}, nil
}

// DeleteManufacturer deletes a manufacturer.
func (s *BlueprintHandlers) DeleteManufacturer(ctx context.Context, req *pb.DeleteManufacturerRequest) (*pb.DeleteManufacturerResponse, error) {
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
	existing, err := s.blueprintSvc.GetManufacturer(ctx, id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerNotFound))
	}
	if existing.IsSystem {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	if err := s.blueprintSvc.DeleteManufacturer(ctx, id); err != nil {
		s.log.ErrorContext(ctx, LogDeleteManufacturerFailed, logger.FieldID, req.Id, logger.FieldError, err)
		if errors.Is(err, storage.ErrForeignKeyViolation) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenManufacturerHasModels),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenManufacturerHasModels))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteManufacturerFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteManufacturerFailed))
	}

	s.recordCatalogChange(ctx, manufacturerDeleted, existing.ID, existing.Name)
	return &pb.DeleteManufacturerResponse{Success: true}, nil
}

// ListManufacturers returns a list of manufacturers.
func (s *BlueprintHandlers) ListManufacturers(ctx context.Context, req *pb.ListManufacturersRequest) (*pb.ListManufacturersResponse, error) {
	if s.blueprintSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	page, err := readPage(clampPageSize(req.PageSize), req.PageToken)
	if err != nil {
		return nil, err
	}

	manufacturers, total, err := s.blueprintSvc.ListManufacturers(ctx, req.IsSystem, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListManufacturersFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListManufacturersFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListManufacturersFailed))
	}

	var pbManufacturers []*pb.Manufacturer
	for _, m := range manufacturers {
		pbManufacturers = append(pbManufacturers, manufacturerToProto(m))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListManufacturersResponse{
		Manufacturers: pbManufacturers,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

func manufacturerToProto(m *models.Manufacturer) *pb.Manufacturer {
	if m == nil {
		return nil
	}
	pbMfr := &pb.Manufacturer{
		Id:         m.ID.String(),
		Name:       m.Name,
		CreatedAt:  timestamppb.New(m.CreatedAt),
		UpdatedAt:  timestamppb.New(m.UpdatedAt),
		TenantId:   formatOptionalTenantID(m.TenantID),
		IsVerified: m.IsVerified,
		IsSystem:   m.IsSystem,
		ModelCount: int32(min(m.ModelCount, math.MaxInt32)), //nolint:gosec // count value bounded
	}
	if m.Website != nil {
		pbMfr.Website = *m.Website
	}
	return pbMfr
}
