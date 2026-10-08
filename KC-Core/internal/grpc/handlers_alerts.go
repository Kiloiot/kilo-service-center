package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	alertsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/alerts"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// ListAlerts returns system alerts.
func (s *AnalyticsHandlers) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	if s.alertSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	page, err := readPage(clampPageSize(req.PageSize), req.PageToken)
	if err != nil {
		return nil, err
	}

	// Proto uses singular strings; convert to slices for service
	// Proto doesn't have start_time/end_time for alerts
	filters := &grpcservices.AlertFilters{}
	if req.Severity != "" {
		filters.Severity = []string{req.Severity}
	}
	if req.Status != "" {
		filters.Status = []string{req.Status}
	}

	alerts, total, err := s.alertSvc.List(ctx, tenantID, filters, page.limit, page.offset)
	if err != nil {
		return nil, s.listAlertsStatus(ctx, err)
	}

	var pbAlerts []*pb.Alert
	for _, a := range alerts {
		pbAlerts = append(pbAlerts, alertToProto(a))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListAlertsResponse{
		Alerts:        pbAlerts,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

// GetAlertSummary returns alert summary.
func (s *AnalyticsHandlers) GetAlertSummary(ctx context.Context, _ *pb.GetAlertSummaryRequest) (*pb.GetAlertSummaryResponse, error) {
	if s.alertSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	summary, err := s.alertSvc.GetSummary(ctx, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetAlertSummaryFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAlertSummaryFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAlertSummaryFailed))
	}

	// Convert recent alerts to proto
	var recentAlerts []*pb.Alert
	for _, a := range summary.Recent {
		recentAlerts = append(recentAlerts, alertToProto(a))
	}

	return &pb.GetAlertSummaryResponse{
		Summary: &pb.AlertSummary{
			Critical: summary.Critical,
			Error:    summary.Error,
			Warning:  summary.Warning,
			Recent:   recentAlerts,
		},
	}, nil
}

// listAlertsStatus answers a refused filter as invalid and any other failure
// as internal.
func (s *AnalyticsHandlers) listAlertsStatus(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, alertsservice.ErrInvalidAlertStatus):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidAlertStatus),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidAlertStatus))
	case errors.Is(err, alertsservice.ErrInvalidAlertSeverity):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidAlertSeverity),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidAlertSeverity))
	}
	s.log.ErrorContext(ctx, LogListAlertsFailed, logger.FieldError, err)
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListAlertsFailed),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListAlertsFailed))
}

func alertToProto(a *grpcservices.Alert) *pb.Alert {
	if a == nil {
		return nil
	}
	return &pb.Alert{
		Id:          a.ID,
		Severity:    a.Severity,
		Category:    a.Category,
		Title:       a.Title,
		Description: a.Description,
		SourceName:  a.SourceName,
		Timestamp:   timestamppb.New(a.Timestamp),
		Status:      a.Status,
	}
}
