// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	"github.com/kilocenter/KC-Core/internal/services/integrations"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage/models"
	pkgcontext "github.com/kilocenter/pkg/context"
)

// CreateIntegration creates a new integration.
func (s *CoreService) CreateIntegration(ctx context.Context, req *pb.CreateIntegrationRequest) (*pb.Integration, error) {
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
	if err != nil {
		s.log.ErrorContext(ctx, "create integration failed", "name", req.Name, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateIntegrationFailed))
	}

	// Emit audit event for integration creation.
	if s.eventWriter != nil {
		sourceID := orgID
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"integrationId": integration.ID,
			"name":          req.Name,
			"type":          req.Type,
			"status":        integration.Status,
			"orgId":         orgID.String(),
		})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeIntegrationCreated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleIntegrationCreated,
			Description: fmt.Sprintf(models.EventDescriptionIntegrationCreatedFmt, req.Name),
			SourceType:  models.SourceTypeAPI,
			SourceID:    &sourceID,
			SourceName:  req.Name,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return integrationToProto(integration), nil
}

// GetIntegration retrieves an integration by ID.
func (s *CoreService) GetIntegration(ctx context.Context, req *pb.GetIntegrationRequest) (*pb.Integration, error) {
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
		s.log.ErrorContext(ctx, "get integration failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetIntegrationFailed))
	}

	return integrationToProto(integration), nil
}

// UpdateIntegration updates an existing integration.
func (s *CoreService) UpdateIntegration(ctx context.Context, req *pb.UpdateIntegrationRequest) (*pb.Integration, error) {
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
		s.log.ErrorContext(ctx, "update integration failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateIntegrationFailed))
	}

	// Emit audit event for integration update.
	if s.eventWriter != nil {
		detailsMap := map[string]interface{}{
			"integrationId": integration.ID,
			"name":          integration.Name,
			"type":          integration.Type,
			"status":        integration.Status,
		}
		evt := &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeIntegrationUpdated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleIntegrationUpdated,
			Description: fmt.Sprintf(models.EventDescriptionIntegrationUpdatedFmt, integration.Name),
			SourceType:  models.SourceTypeAPI,
			SourceName:  integration.Name,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if updateOrgID, orgErr := pkgcontext.GetOrganizationID(ctx); orgErr == nil {
			detailsMap["orgId"] = updateOrgID.String()
			evt.SourceID = &updateOrgID
		}
		evt.Details, _ = json.Marshal(detailsMap)
		_ = s.eventWriter.CreateEvent(ctx, evt)
	}

	return integrationToProto(integration), nil
}

// DeleteIntegration deletes an integration.
func (s *CoreService) DeleteIntegration(ctx context.Context, req *pb.DeleteIntegrationRequest) (*emptypb.Empty, error) {
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
		s.log.ErrorContext(ctx, "delete integration failed", "id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteIntegrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteIntegrationFailed))
	}

	// Emit audit event for integration deletion.
	if s.eventWriter != nil {
		sourceName := integrationName
		if sourceName == "" {
			sourceName = strconv.FormatInt(req.Id, 10)
		}
		detailsMap := map[string]interface{}{
			"integrationId": req.Id,
			"type":          integrationType,
			"status":        integrationStatus,
		}
		evt := &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeIntegrationDeleted,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleIntegrationDeleted,
			Description: fmt.Sprintf(models.EventDescriptionIntegrationDeletedFmt, sourceName),
			SourceType:  models.SourceTypeAPI,
			SourceName:  sourceName,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if delOrgID, orgErr := pkgcontext.GetOrganizationID(ctx); orgErr == nil {
			detailsMap["orgId"] = delOrgID.String()
			evt.SourceID = &delOrgID
		}
		evt.Details, _ = json.Marshal(detailsMap)
		_ = s.eventWriter.CreateEvent(ctx, evt)
	}

	return &emptypb.Empty{}, nil
}

// ListIntegrations returns a paginated list of integrations.
func (s *CoreService) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
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
		s.log.ErrorContext(ctx, "list integrations failed", "error", err)
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

	// Convert config JSON to Struct
	if integration.Config != nil {
		var configMap map[string]interface{}
		if err := json.Unmarshal(integration.Config, &configMap); err == nil {
			if configStruct, err := structpb.NewStruct(configMap); err == nil {
				pb.Config = configStruct
			}
		}
	}

	// Convert event filter JSON to Struct
	if integration.EventFilter != nil {
		var filterMap map[string]interface{}
		if err := json.Unmarshal(integration.EventFilter, &filterMap); err == nil {
			if filterStruct, err := structpb.NewStruct(filterMap); err == nil {
				pb.EventFilter = filterStruct
			}
		}
	}

	return pb
}
