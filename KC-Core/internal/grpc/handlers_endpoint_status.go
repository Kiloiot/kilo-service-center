package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	endpointpkg "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// requestedStatusDecision returns the attachment status a status update asks
// the service center to decide, or "" when the update leaves it unchanged.
func (s *EndpointHandlers) requestedStatusDecision(endpoint *models.EndPoint, req *pb.UpdateEndPointRequest, mask *fieldmaskpb.FieldMask) (string, error) {
	if !fieldInMask(mask, fieldMaskStatus) {
		return "", nil
	}
	requested := req.Endpoint.Status
	if !endpointpkg.IsAttachmentDecision(requested) {
		return "", status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointStatus),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointStatus))
	}
	if requested == endpoint.EpStatus {
		return "", nil
	}
	return requested, nil
}

// decideEndpointStatus attaches or detaches the endpoint and returns it as it
// then stands.
func (s *EndpointHandlers) decideEndpointStatus(ctx context.Context, tenantID int64, eui models.EUI, decision string) (*models.EndPoint, error) {
	if _, err := s.decideAttachment(ctx, tenantID, eui.String(), decision); err != nil {
		return nil, err
	}
	decided, err := s.endpointSvc.GetByEUI(ctx, eui[:], tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetEndpointFailed, logger.FieldEpEuiSnake, eui.String(), logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}
	return decided, nil
}

// decideRequestedAttachment is AttachEndPoint and DetachEndPoint: the
// caller's tenant attaches or detaches the endpoint the request names.
func (s *EndpointHandlers) decideRequestedAttachment(ctx context.Context, epEui string, decision string) (*grpcservices.EndpointOperationResult, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if epEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}
	return s.decideAttachment(ctx, tenantID, epEui, decision)
}

// attachmentStep is how the attachment service takes one kind of decision
// and how its failure is logged and answered.
type attachmentStep struct {
	initiatingLog string
	failedLog     string
	failedToken   string
	decide        func(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error)
}

func (s *EndpointHandlers) attachmentStep(decision string) attachmentStep {
	if decision == endpointpkg.EndpointStatusAttached {
		return attachmentStep{LogInitiatingEndpointAttach, LogFailedToInitiateAttach, grpcerrors.ErrTokenAttachFailed, s.endpointAttachmentSvc.AttachEndPoint}
	}
	return attachmentStep{LogInitiatingEndpointDetach, LogFailedToInitiateDetach, grpcerrors.ErrTokenDetachFailed, s.endpointAttachmentSvc.DetachEndPoint}
}

// decideAttachment has the attachment service attach or detach the tenant's
// endpoint, which records and announces the decision and propagates it to
// the base stations, and answers its refusal as the RPCs do.
func (s *EndpointHandlers) decideAttachment(ctx context.Context, tenantID int64, epEui string, decision string) (*grpcservices.EndpointOperationResult, error) {
	step := s.attachmentStep(decision)
	s.log.InfoContext(ctx, step.initiatingLog, logger.FieldEpEuiSnake, epEui, logger.FieldTenantIDSnake, tenantID)
	result, err := step.decide(ctx, epEui, tenantID)
	if err == nil {
		return result, nil
	}
	s.log.ErrorContext(ctx, step.failedLog, logger.FieldEpEuiSnake, epEui, logger.FieldError, err)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	return nil, status.Error(grpcerrors.GetGRPCCode(step.failedToken), grpcerrors.ResolveErrorMessage(step.failedToken))
}

// storeNewEndpoint stores a new endpoint; one created with pre-attachment is
// stored attached together with its attachment, or not at all (BSSCI §3.8).
func (s *EndpointHandlers) storeNewEndpoint(ctx context.Context, endpoint *models.EndPoint, preAttach bool) (*models.EndPoint, error) {
	store := s.endpointSvc.Create
	if preAttach {
		store = s.endpointAttachmentSvc.CreateAttached
	}
	created, err := store(ctx, endpoint)
	if err == nil {
		return created, nil
	}
	if errors.Is(err, kcerrors.ErrDuplicate) {
		s.log.WarnContext(ctx, grpcerrors.LogEndpointAlreadyExists, logger.FieldEui, endpoint.EUI.String(), logger.FieldTenantIDSnake, endpoint.TenantID)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointExists),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists))
	}
	if errors.Is(err, grpcservices.ErrEndpointNotAttachable) {
		s.log.ErrorContext(ctx, LogFailedToInitiateAttach, logger.FieldEpEuiSnake, endpoint.EUI.String(), logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAttachFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAttachFailed))
	}
	s.log.ErrorContext(ctx, grpcerrors.LogEndpointCreateFailed, logger.FieldError, err)
	return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateEndpointFailed),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateEndpointFailed))
}
