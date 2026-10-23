package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// pinOnCreate pins the selected blueprint of a new endpoint; a snapshot-only
// tenant's endpoint is pinned to its model's default blueprint or refused.
func (s *EndpointHandlers) pinOnCreate(ctx context.Context, tenantID int64, endpoint *models.EndPoint, in *pb.EndPoint) error {
	on, err := snapshotOnly(ctx, s.modes, s.log, tenantID)
	if err != nil {
		return err
	}
	if in.GetBlueprintId() != "" {
		return s.pinSelectedBlueprint(ctx, endpoint, in, on)
	}
	if !on {
		return nil
	}
	if endpoint.DeviceModelID == nil {
		return snapshotRequiredError()
	}
	return s.pinModelDefault(ctx, tenantID, endpoint, on)
}

// applyEndpointSnapshotTrigger re-pins an updated endpoint: an explicit
// selection wins; else a model change re-seeds from the new model's default
// when the pinned blueprint isn't native to it. A snapshot-only owner's
// endpoint ends with a snapshot of its model or the update is refused.
func (s *EndpointHandlers) applyEndpointSnapshotTrigger(ctx context.Context, tenantID int64, endpoint *models.EndPoint, req *pb.UpdateEndPointRequest, mask *fieldmaskpb.FieldMask, priorDeviceModelID *uuid.UUID, priorSnapshotSourceID string) error {
	explicit := fieldInMask(mask, fieldMaskBlueprintID) && req.Endpoint.BlueprintId != "" && req.Endpoint.BlueprintId != priorSnapshotSourceID
	if !explicit && !modelChanged(priorDeviceModelID, endpoint.DeviceModelID) {
		return nil
	}
	on, err := snapshotOnly(ctx, s.modes, s.log, endpoint.OwnerTenantID)
	if err != nil {
		return err
	}
	if explicit {
		return s.pinSelectedBlueprint(ctx, endpoint, req.Endpoint, on)
	}
	if endpoint.DeviceModelID == nil {
		if on {
			return snapshotRequiredError()
		}
		return nil
	}
	if pinnedBlueprintBelongsToModel(ctx, s.blueprintSvc, priorSnapshotSourceID, *endpoint.DeviceModelID) {
		return nil
	}
	return s.pinModelDefault(ctx, tenantID, endpoint, on)
}

// pinSelectedBlueprint pins the request's blueprint_id; a snapshot-only
// owner accepts only a blueprint of the request's device_model_id, an empty
// one refused, that is also the endpoint's model as the update mask leaves
// it, so an unmasked device_model_id cannot move the endpoint to another model.
func (s *EndpointHandlers) pinSelectedBlueprint(ctx context.Context, endpoint *models.EndPoint, in *pb.EndPoint, snapshotOnly bool) error {
	bp, err := loadBlueprint(ctx, s.blueprintSvc, in.GetBlueprintId())
	if err != nil {
		return err
	}
	if snapshotOnly && (!blueprintOfRequestedModel(bp, in.GetDeviceModelId()) || !blueprintOfModel(bp, endpoint.DeviceModelID)) {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotOfRequestedModel),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotOfRequestedModel))
	}
	return pinBlueprint(ctx, s.log, endpoint, bp, s.clock.Now())
}

// pinModelDefault pins the endpoint's model's default blueprint; without one
// a snapshot-only owner's change is refused and any other keeps its snapshot.
func (s *EndpointHandlers) pinModelDefault(ctx context.Context, tenantID int64, endpoint *models.EndPoint, snapshotOnly bool) error {
	if s.blueprintSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	def, defErr := s.blueprintSvc.GetDefaultForModel(ctx, tenantID, *endpoint.DeviceModelID)
	if defErr != nil {
		// Infra failure — do not silently continue a change that touches the decode source.
		s.log.ErrorContext(ctx, LogFailedToResolveNewModelDefaultBlueprintForReSeed,
			logger.FieldError, defErr, logger.FieldEndpointEui, endpoint.EUI.String())
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed))
	}
	if def == nil {
		if snapshotOnly {
			return snapshotRequiredError()
		}
		s.log.WarnContext(ctx, LogDeviceModelChangedNoDefaultBlueprint,
			logger.FieldEndpointEui, endpoint.EUI.String())
		return nil
	}
	return applyBlueprintSnapshot(ctx, s.log, s.blueprintSvc, endpoint, def.ID.String(), s.clock.Now())
}

func blueprintOfRequestedModel(bp *models.Blueprint, requestedModelID string) bool {
	id, err := uuid.Parse(requestedModelID)
	return err == nil && id == bp.DeviceModelID
}

func blueprintOfModel(bp *models.Blueprint, modelID *uuid.UUID) bool {
	return modelID != nil && *modelID == bp.DeviceModelID
}

func snapshotRequiredError() error {
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSnapshotRequired),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSnapshotRequired))
}
