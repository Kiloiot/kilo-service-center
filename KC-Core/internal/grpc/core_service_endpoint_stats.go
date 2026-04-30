// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage/models"
)

// GetEndPointStats retrieves message statistics for an endpoint.
func (s *CoreService) GetEndPointStats(ctx context.Context, req *pb.GetEndPointStatsRequest) (*pb.GetEndPointStatsResponse, error) {
	if s.endpointStatsStore == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	// Parse EUI hex string to uint64
	epEuiClean := strings.ReplaceAll(req.EpEui, "-", "")
	epEuiClean = strings.ReplaceAll(epEuiClean, ":", "")
	epEui, err := strconv.ParseUint(epEuiClean, 16, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Get message stats from store
	stats, err := s.endpointStatsStore.GetMessageStatsByEndpoint(ctx, epEui, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "get endpoint stats failed", "ep_eui", req.EpEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointStatsFailed))
	}

	euiParsed := models.EUIFromString(req.EpEui)
	if euiParsed == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	endpoint, err := s.endpointSvc.GetByEUI(ctx, euiParsed[:], tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "get endpoint failed", "ep_eui", req.EpEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}

	resp := &pb.GetEndPointStatsResponse{
		EpEui:           req.EpEui,
		TotalCount:      stats.TotalCount,
		UniqueEndpoints: stats.UniqueEndpoints,
		AvgRssi:         stats.AvgRSSI,
		AvgSnr:          stats.AvgSNR,
		AttachStatus:    endpoint.EpStatus,
	}

	// Handle optional timestamp fields
	if stats.FirstSeen != nil {
		resp.FirstSeen = timestamppb.New(*stats.FirstSeen)
	}
	if stats.LastSeen != nil {
		resp.LastSeen = timestamppb.New(*stats.LastSeen)
	}
	if stats.ActiveDays != nil {
		resp.ActiveDays = int32(*stats.ActiveDays) //nolint:gosec // ActiveDays is bounded by days since epoch
	}

	return resp, nil
}

// GetEndPointOperations retrieves recent operations for an endpoint.
func (s *CoreService) GetEndPointOperations(ctx context.Context, req *pb.GetEndPointOperationsRequest) (*pb.GetEndPointOperationsResponse, error) {
	if s.opStatusAdapter == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	euiParsed := models.EUIFromString(req.EpEui)
	if euiParsed == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Get endpoint to resolve ep_eui to endpoint ID
	endpoint, err := s.endpointSvc.GetByEUI(ctx, euiParsed[:], tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "get endpoint failed", "ep_eui", req.EpEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}

	// Set defaults for pagination
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	offset := int(req.Offset)

	// Get operations from adapter using endpoint's database ID
	operations, err := s.opStatusAdapter.GetEndpointOperations(ctx, endpoint.ID, tenantID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "get endpoint operations failed", "ep_eui", req.EpEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointOperationsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointOperationsFailed))
	}

	// Map to proto response
	pbOps := make([]*pb.EndPointOperation, 0, len(operations))
	for _, op := range operations {
		pbOp := &pb.EndPointOperation{
			Id:        op.ID,
			EventType: op.EventType,
			Category:  op.Category,
			Severity:  op.Severity,
			Title:     op.Title,
			CreatedAt: timestamppb.New(op.CreatedAt),
		}
		pbOps = append(pbOps, pbOp)
	}

	return &pb.GetEndPointOperationsResponse{
		Operations: pbOps,
	}, nil
}
