// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/integrations"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// IntegrationHandlers serves the integration RPCs.
type IntegrationHandlers struct {
	integrationSvc grpcservices.IntegrationService
	audit          AuditRecorder
	log            logger.Logger
}

// IntegrationHandlerDeps wires IntegrationHandlers; a nil service answers
// its RPCs as not configured.
type IntegrationHandlerDeps struct {
	Integrations grpcservices.IntegrationService
}

// NewIntegrationHandlers builds the group.
func NewIntegrationHandlers(d IntegrationHandlerDeps, recorder AuditRecorder, log logger.Logger) *IntegrationHandlers {
	return &IntegrationHandlers{integrationSvc: d.Integrations, audit: recorder, log: log}
}

// integrationConfigValueMasked stands in for every stored integration setting
// on the way out: settings may hold credentials, so they are write-only.
const integrationConfigValueMasked = "configured"

// Integration audit event detail keys.
const (
	integrationDetailKeyID     = "integrationId"
	integrationDetailKeyName   = "name"
	integrationDetailKeyType   = "type"
	integrationDetailKeyStatus = "status"
)

// CreateIntegration creates a new integration.
func (s *IntegrationHandlers) CreateIntegration(ctx context.Context, req *pb.CreateIntegrationRequest) (*pb.Integration, error) {
	if s.integrationSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get org from context
	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingOrgContext),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMissingOrgContext))
	}

	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	if req.Config == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenConfigRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenConfigRequired))
	}

	// Convert proto Struct to JSON
	var config, eventFilter json.RawMessage
	if req.Config != nil {
		configBytes, err := req.Config.MarshalJSON()
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidRequest),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidRequest))
		}
		config = configBytes
	}
	if req.EventFilter != nil {
		filterBytes, err := req.EventFilter.MarshalJSON()
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidRequest),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidRequest))
		}
		eventFilter = filterBytes
	}

	createReq := &grpcservices.IntegrationCreateRequest{
		OrgID:          orgID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Config:         config,
		EventFilter:    eventFilter,
		DeliveryFormat: req.DeliveryFormat,
	}

	integration, err := s.integrationSvc.Create(ctx, tenantID, createReq)
	if errors.Is(err, integrations.ErrInvalidType) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIntegrationType),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIntegrationType))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateIntegrationFailed, logger.FieldName, req.Name, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateIntegrationFailed))
	}

	sourceID := orgID
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		EventType:   models.EventTypeIntegrationCreated,
		Title:       models.EventTitleIntegrationCreated,
		Description: fmt.Sprintf(models.EventDescriptionIntegrationCreatedFmt, req.Name),
		SourceID:    &sourceID,
		SourceName:  req.Name,
		Details: map[string]any{
			integrationDetailKeyID:     integration.ID,
			integrationDetailKeyName:   req.Name,
			integrationDetailKeyType:   req.Type,
			integrationDetailKeyStatus: integration.Status,
			"orgId":                    orgID.String(),
		},
	})

	return integrationToProto(integration), nil
}

// GetIntegration retrieves an integration by ID.
func (s *IntegrationHandlers) GetIntegration(ctx context.Context, req *pb.GetIntegrationRequest) (*pb.Integration, error) {
	if s.integrationSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	integration, err := s.integrationSvc.GetByID(ctx, tenantID, req.Id)
	if err != nil {
		if errors.Is(err, integrations.ErrIntegrationNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIntegrationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIntegrationNotFound))
		}
		s.log.ErrorContext(ctx, LogGetIntegrationFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetIntegrationFailed))
	}

	return integrationToProto(integration), nil
}

// UpdateIntegration updates an existing integration.
func (s *IntegrationHandlers) UpdateIntegration(ctx context.Context, req *pb.UpdateIntegrationRequest) (*pb.Integration, error) {
	if s.integrationSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	// Convert proto Struct to JSON
	var config, eventFilter json.RawMessage
	if req.Config != nil {
		configBytes, err := req.Config.MarshalJSON()
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidRequest),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidRequest))
		}
		config = configBytes
	}
	if req.EventFilter != nil {
		filterBytes, err := req.EventFilter.MarshalJSON()
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidRequest),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidRequest))
		}
		eventFilter = filterBytes
	}

	updateReq := &grpcservices.IntegrationUpdateRequest{
		Config:      config,
		EventFilter: eventFilter,
	}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}
	if req.Status != "" {
		updateReq.Status = &req.Status
	}

	integration, err := s.integrationSvc.Update(ctx, tenantID, req.Id, updateReq)
	if err != nil {
		if errors.Is(err, integrations.ErrIntegrationNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIntegrationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIntegrationNotFound))
		}
		if errors.Is(err, integrations.ErrInvalidStatus) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIntegrationStatus),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIntegrationStatus))
		}
		s.log.ErrorContext(ctx, LogUpdateIntegrationFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateIntegrationFailed))
	}

	updateEvent := audit.Event{
		TenantID:    tenantID,
		EventType:   models.EventTypeIntegrationUpdated,
		Title:       models.EventTitleIntegrationUpdated,
		Description: fmt.Sprintf(models.EventDescriptionIntegrationUpdatedFmt, integration.Name),
		SourceName:  integration.Name,
		Details: map[string]any{
			integrationDetailKeyID:     integration.ID,
			integrationDetailKeyName:   integration.Name,
			integrationDetailKeyType:   integration.Type,
			integrationDetailKeyStatus: integration.Status,
		},
	}
	if updateOrgID, orgErr := pkgcontext.GetOrganizationID(ctx); orgErr == nil {
		updateEvent.Details["orgId"] = updateOrgID.String()
		updateEvent.SourceID = &updateOrgID
	}
	s.audit.Record(ctx, updateEvent)

	return integrationToProto(integration), nil
}

// DeleteIntegration deletes an integration.
func (s *IntegrationHandlers) DeleteIntegration(ctx context.Context, req *pb.DeleteIntegrationRequest) (*emptypb.Empty, error) {
	if s.integrationSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	// Prefetch integration metadata before deletion for audit event.
	var integrationName, integrationType, integrationStatus string
	if existing, fetchErr := s.integrationSvc.GetByID(ctx, tenantID, req.Id); fetchErr == nil && existing != nil {
		integrationName = existing.Name
		integrationType = existing.Type
		integrationStatus = existing.Status
	}

	if err := s.integrationSvc.Delete(ctx, tenantID, req.Id); err != nil {
		if errors.Is(err, integrations.ErrIntegrationNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIntegrationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIntegrationNotFound))
		}
		s.log.ErrorContext(ctx, LogDeleteIntegrationFailed, logger.FieldID, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteIntegrationFailed))
	}

	sourceName := integrationName
	if sourceName == "" {
		sourceName = strconv.FormatInt(req.Id, 10)
	}
	deleteEvent := audit.Event{
		TenantID:    tenantID,
		EventType:   models.EventTypeIntegrationDeleted,
		Title:       models.EventTitleIntegrationDeleted,
		Description: fmt.Sprintf(models.EventDescriptionIntegrationDeletedFmt, sourceName),
		SourceName:  sourceName,
		Details: map[string]any{
			integrationDetailKeyID:     req.Id,
			integrationDetailKeyType:   integrationType,
			integrationDetailKeyStatus: integrationStatus,
		},
	}
	if delOrgID, orgErr := pkgcontext.GetOrganizationID(ctx); orgErr == nil {
		deleteEvent.Details["orgId"] = delOrgID.String()
		deleteEvent.SourceID = &delOrgID
	}
	s.audit.Record(ctx, deleteEvent)

	return &emptypb.Empty{}, nil
}

// ListIntegrations returns a paginated list of integrations.
func (s *IntegrationHandlers) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	if s.integrationSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	offset := int(req.Offset)

	integrations, totalCount, err := s.integrationSvc.List(ctx, tenantID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListIntegrationsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListIntegrationsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListIntegrationsFailed))
	}

	pbIntegrations := make([]*pb.Integration, 0, len(integrations))
	for _, integration := range integrations {
		pbIntegrations = append(pbIntegrations, integrationToProto(integration))
	}

	return &pb.ListIntegrationsResponse{
		Integrations: pbIntegrations,
		TotalCount:   totalCount,
	}, nil
}

// integrationToProto converts a models.Integration to pb.Integration
func integrationToProto(integration *models.Integration) *pb.Integration {
	pb := &pb.Integration{
		Id:             integration.ID,
		Name:           integration.Name,
		Type:           integration.Type,
		DeliveryFormat: integration.DeliveryFormat,
		Status:         integration.Status,
		CreatedAt:      timestamppb.New(integration.CreatedAt),
		UpdatedAt:      timestamppb.New(integration.UpdatedAt),
	}

	if integration.Description != nil {
		pb.Description = *integration.Description
	}

	// Convert config JSON to Struct, naming the configured settings without their values
	if integration.Config != nil {
		var configMap map[string]interface{}
		if err := json.Unmarshal(integration.Config, &configMap); err == nil {
			for key := range configMap {
				configMap[key] = integrationConfigValueMasked
			}
			if configStruct, err := structpb.NewStruct(configMap); err == nil {
				pb.Config = configStruct
			}
		}
	}

	// Convert event filter JSON to Struct
	if integration.EventFilter.Valid {
		var filterMap map[string]interface{}
		if err := json.Unmarshal(integration.EventFilter.Data, &filterMap); err == nil {
			if filterStruct, err := structpb.NewStruct(filterMap); err == nil {
				pb.EventFilter = filterStruct
			}
		}
	}

	return pb
}
