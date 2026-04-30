// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"strings"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
)

// Analytics handlers

// GetAnalyticsOverview returns analytics overview data.
func (s *CoreService) GetAnalyticsOverview(ctx context.Context, req *pb.GetAnalyticsOverviewRequest) (*pb.GetAnalyticsOverviewResponse, error) {
	if s.analyticsSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	startTime := timestampToTime(req.StartTime)
	endTime := timestampToTime(req.EndTime)

	overview, err := s.analyticsSvc.GetOverview(ctx, tenantID, startTime, endTime)
	if err != nil {
		s.log.ErrorContext(ctx, "get analytics overview failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAnalyticsOverviewFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAnalyticsOverviewFail))
	}

	return &pb.GetAnalyticsOverviewResponse{
		Overview: &pb.AnalyticsOverview{
			TotalMessages:      overview.TotalMessages,
			ActiveEndpoints:    overview.ActiveEndpoints,
			ActiveBaseStations: overview.OnlineBaseStations,
			AvgRssi:            overview.AverageRSSI,
			AvgSnr:             overview.AverageSNR,
		},
	}, nil
}

// GetActivityAnalytics returns activity analytics data.
func (s *CoreService) GetActivityAnalytics(ctx context.Context, req *pb.GetActivityAnalyticsRequest) (*pb.GetActivityAnalyticsResponse, error) {
	if s.analyticsSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	startTime := timestampToTime(req.StartTime)
	endTime := timestampToTime(req.EndTime)

	activity, err := s.analyticsSvc.GetActivity(ctx, tenantID, startTime, endTime)
	if err != nil {
		s.log.ErrorContext(ctx, "get activity analytics failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAnalyticsActivityFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAnalyticsActivityFail))
	}

	// Convert time slots from messages per hour
	var timeSlots []*pb.TimeSlotActivity
	for _, count := range activity.MessagesPerHour {
		timeSlots = append(timeSlots, &pb.TimeSlotActivity{
			MessageCount: count,
		})
	}

	return &pb.GetActivityAnalyticsResponse{
		Activity: &pb.ActivityAnalytics{
			TotalMessages:      int64(len(activity.MessagesPerHour)),
			UniqueEndpoints:    int64(len(activity.EndpointActivity)),
			UniqueBaseStations: int64(len(activity.TopBaseStations)),
			TimeSlots:          timeSlots,
		},
	}, nil
}

// GetSignalQualityAnalytics returns signal quality metrics.
func (s *CoreService) GetSignalQualityAnalytics(ctx context.Context, req *pb.GetSignalQualityAnalyticsRequest) (*pb.GetSignalQualityAnalyticsResponse, error) {
	if s.analyticsSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	startTime := timestampToTime(req.StartTime)
	endTime := timestampToTime(req.EndTime)

	quality, err := s.analyticsSvc.GetSignalQuality(ctx, tenantID, startTime, endTime)
	if err != nil {
		s.log.ErrorContext(ctx, "get signal quality analytics failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAnalyticsSignalFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAnalyticsSignalFail))
	}

	return &pb.GetSignalQualityAnalyticsResponse{
		SignalQuality: &pb.SignalQualityAnalytics{
			Overall: &pb.SignalQualityOverall{
				AvgRssi: quality.AverageRSSI,
				MinRssi: quality.RSSIRange[0],
				MaxRssi: quality.RSSIRange[1],
				AvgSnr:  quality.AverageSNR,
				MinSnr:  quality.SNRRange[0],
				MaxSnr:  quality.SNRRange[1],
			},
		},
	}, nil
}

// Events handlers

// ListEvents returns system events.
func (s *CoreService) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	if s.eventSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	// Build filters from request
	filters := &grpcservices.EventFilters{
		StartTime: timestampToTime(req.StartTime),
		EndTime:   timestampToTime(req.EndTime),
	}
	if len(req.GetCategories()) > 0 {
		filters.Categories = req.GetCategories()
	}
	if req.Severity != "" {
		filters.Severity = []string{req.Severity}
	}
	if len(req.GetEventTypes()) > 0 {
		filters.EventTypes = req.GetEventTypes()
	}

	events, total, err := s.eventSvc.List(ctx, tenantID, filters, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list events failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListEventsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListEventsFailed))
	}

	var pbEvents []*pb.Event
	for _, e := range events {
		pbEvents = append(pbEvents, eventToProto(e))
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
	return &pb.ListEventsResponse{
		Events:        pbEvents,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// Event streaming handlers

// StreamEvents streams events for the tenant.
func (s *CoreService) StreamEvents(req *pb.StreamEventsRequest, stream pb.KiloCenterService_StreamEventsServer) error {
	if s.eventSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	ctx := stream.Context()
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return err
	}

	// Build filters from request
	filters := &grpcservices.EventFilters{
		StartTime: timestampToTime(req.StartTime),
		EndTime:   timestampToTime(req.EndTime),
	}
	if req.Category != "" {
		filters.Categories = []string{req.Category}
	}
	if req.Severity != "" {
		filters.Severity = []string{req.Severity}
	}

	eventChan, err := s.eventSvc.Stream(ctx, tenantID, filters)
	if err != nil {
		s.log.ErrorContext(ctx, "stream events failed", "error", err)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStreamEventsStartFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStreamEventsStartFailed))
	}

	for event := range eventChan {
		if err := stream.Send(eventToProto(event)); err != nil {
			s.log.ErrorContext(ctx, "stream send failed", "error", err)
			return err
		}
	}
	return nil
}

// Alerts handlers

// ListAlerts returns system alerts.
func (s *CoreService) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	if s.alertSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
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

	alerts, total, err := s.alertSvc.List(ctx, tenantID, filters, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list alerts failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListAlertsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListAlertsFailed))
	}

	var pbAlerts []*pb.Alert
	for _, a := range alerts {
		pbAlerts = append(pbAlerts, alertToProto(a))
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
	return &pb.ListAlertsResponse{
		Alerts:        pbAlerts,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// GetAlertSummary returns alert summary.
func (s *CoreService) GetAlertSummary(ctx context.Context, _ *pb.GetAlertSummaryRequest) (*pb.GetAlertSummaryResponse, error) {
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
		s.log.ErrorContext(ctx, "get alert summary failed", "error", err)
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
			Warning:  summary.Warning,
			Info:     summary.Info,
			Recent:   recentAlerts,
		},
	}, nil
}

// Helper functions

// parseEUI parses a hex-encoded EUI string to bytes. Enforces exactly 8 bytes (64-bit EUI).
func parseEUI(euiStr string) ([]byte, error) {
	// Remove common prefixes
	euiStr = strings.TrimPrefix(euiStr, "0x")
	euiStr = strings.TrimPrefix(euiStr, "0X")

	// Decode hex
	eui, err := hex.DecodeString(euiStr)
	if err != nil {
		return nil, err
	}

	if len(eui) != 8 {
		return nil, errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEUIFormat))
	}

	return eui, nil
}

func eventToProto(e *grpcservices.Event) *pb.Event {
	if e == nil {
		return nil
	}
	return &pb.Event{
		Id:          e.ID,
		EventType:   e.EventType,
		Category:    e.Category,
		Severity:    e.Severity,
		Title:       e.Title,
		Description: e.Description,
		SourceName:  e.SourceName,
		Timestamp:   timestamppb.New(e.Timestamp),
		Data:        e.Data,
	}
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
