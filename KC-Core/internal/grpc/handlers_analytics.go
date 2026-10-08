// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	analyticsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/analytics"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/errorgroups"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// AnalyticsHandlers serves the analytics, event and alert RPCs.
type AnalyticsHandlers struct {
	analyticsSvc grpcservices.AnalyticsService
	eventSvc     EventFeed
	alertSvc     grpcservices.AlertService
	errorGroups  grpcservices.ErrorGroupService
	log          logger.Logger
}

// AnalyticsHandlerDeps wires AnalyticsHandlers; a nil service answers its
// RPCs as not configured.
type AnalyticsHandlerDeps struct {
	Analytics   grpcservices.AnalyticsService
	Events      EventFeed
	Alerts      grpcservices.AlertService
	ErrorGroups grpcservices.ErrorGroupService
}

// NewAnalyticsHandlers builds the group.
func NewAnalyticsHandlers(d AnalyticsHandlerDeps, log logger.Logger) *AnalyticsHandlers {
	return &AnalyticsHandlers{analyticsSvc: d.Analytics, eventSvc: d.Events, alertSvc: d.Alerts, errorGroups: d.ErrorGroups, log: log}
}

// Analytics handlers

// GetAnalyticsOverview returns analytics overview data.
// Optional hex prefixes stripped from user-entered EUI strings.
func (s *AnalyticsHandlers) GetAnalyticsOverview(ctx context.Context, req *pb.GetAnalyticsOverviewRequest) (*pb.GetAnalyticsOverviewResponse, error) {
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
		s.log.ErrorContext(ctx, LogGetAnalyticsOverviewFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAnalyticsOverviewFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAnalyticsOverviewFail))
	}

	return &pb.GetAnalyticsOverviewResponse{
		Overview: &pb.AnalyticsOverview{
			TotalMessages:      overview.TotalMessages,
			ActiveEndpoints:    overview.ActiveEndpoints,
			ActiveBaseStations: overview.ActiveBaseStations,
			AvgRssi:            overview.AverageRSSI,
			AvgSnr:             overview.AverageSNR,
		},
	}, nil
}

// GetActivityAnalytics returns activity analytics data.
func (s *AnalyticsHandlers) GetActivityAnalytics(ctx context.Context, req *pb.GetActivityAnalyticsRequest) (*pb.GetActivityAnalyticsResponse, error) {
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

	activity, err := s.analyticsSvc.GetActivity(ctx, tenantID, startTime, endTime, req.Granularity)
	if errors.Is(err, analyticsservice.ErrUnsupportedGranularity) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUnsupportedGranularity),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUnsupportedGranularity))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogGetActivityAnalyticsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAnalyticsActivityFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAnalyticsActivityFail))
	}

	timeSlots := make([]*pb.TimeSlotActivity, len(activity.Slots))
	for i, slot := range activity.Slots {
		timeSlots[i] = &pb.TimeSlotActivity{
			Slot:          timestamppb.New(slot.Slot),
			MessageCount:  slot.MessageCount,
			EndpointCount: slot.EndpointCount,
		}
	}

	return &pb.GetActivityAnalyticsResponse{
		Activity: &pb.ActivityAnalytics{
			StartTime:          timestamppb.New(activity.StartTime),
			EndTime:            timestamppb.New(activity.EndTime),
			TotalMessages:      activity.TotalMessages,
			UniqueEndpoints:    activity.UniqueEndpoints,
			UniqueBaseStations: activity.UniqueBaseStations,
			TimeSlots:          timeSlots,
		},
	}, nil
}

// GetSignalQualityAnalytics returns signal quality metrics.
func (s *AnalyticsHandlers) GetSignalQualityAnalytics(ctx context.Context, req *pb.GetSignalQualityAnalyticsRequest) (*pb.GetSignalQualityAnalyticsResponse, error) {
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
		s.log.ErrorContext(ctx, LogGetSignalQualityAnalyticsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAnalyticsSignalFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAnalyticsSignalFail))
	}

	byStation := make([]*pb.BaseStationSignalQuality, len(quality.ByBaseStation))
	for i, station := range quality.ByBaseStation {
		byStation[i] = &pb.BaseStationSignalQuality{
			Eui:          station.EUI,
			AvgRssi:      station.AverageRSSI,
			AvgSnr:       station.AverageSNR,
			MessageCount: station.MessageCount,
		}
	}

	return &pb.GetSignalQualityAnalyticsResponse{
		SignalQuality: &pb.SignalQualityAnalytics{
			StartTime: timestamppb.New(quality.StartTime),
			EndTime:   timestamppb.New(quality.EndTime),
			Overall: &pb.SignalQualityOverall{
				AvgRssi:    quality.AverageRSSI,
				MinRssi:    quality.RSSIRange[0],
				MaxRssi:    quality.RSSIRange[1],
				MedianRssi: quality.MedianRSSI,
				AvgSnr:     quality.AverageSNR,
				MinSnr:     quality.SNRRange[0],
				MaxSnr:     quality.SNRRange[1],
				MedianSnr:  quality.MedianSNR,
			},
			ByBaseStation: byStation,
		},
	}, nil
}

// Events handlers

// ListEvents returns system events.
func (s *AnalyticsHandlers) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	if s.eventSvc == nil {
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

	categories, unrestricted := authz.VisibleEventCategories(authz.FromContext(ctx), req.GetCategories())
	if !unrestricted && len(categories) == 0 {
		return &pb.ListEventsResponse{}, nil
	}

	filters := &grpcservices.EventFilters{
		StartTime:  timestampToTime(req.StartTime),
		EndTime:    timestampToTime(req.EndTime),
		OpID:       req.OpId,
		EpEUI:      req.EpEui,
		BsEUI:      req.BsEui,
		Outcome:    req.Outcome,
		Search:     req.Search,
		Categories: categories,
	}
	if req.Severity != "" {
		filters.Severity = []string{req.Severity}
	}
	if len(req.GetEventTypes()) > 0 {
		filters.EventTypes = req.GetEventTypes()
	}

	events, total, err := s.eventSvc.List(ctx, tenantID, filters, page.limit, page.offset)
	if errors.Is(err, eventsservice.ErrInvalidOutcome) || errors.Is(err, eventsservice.ErrOutcomeSeverityMismatch) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEventOutcome),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEventOutcome))
	}
	if errors.Is(err, eventsservice.ErrInvalidEndpointEUI) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogListEventsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListEventsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListEventsFailed))
	}

	var pbEvents []*pb.Event
	for _, e := range events {
		pbEvents = append(pbEvents, eventToProto(e))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListEventsResponse{
		Events:        pbEvents,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

// Event streaming handlers

// StreamEvents streams events for the tenant.
func (s *AnalyticsHandlers) StreamEvents(req *pb.StreamEventsRequest, stream pb.KiloCenterService_StreamEventsServer) error {
	if s.eventSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	ctx := stream.Context()
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return err
	}

	var requested []string
	if req.Category != "" {
		requested = []string{req.Category}
	}
	categories, unrestricted := authz.VisibleEventCategories(authz.FromContext(ctx), requested)
	if !unrestricted && len(categories) == 0 {
		return nil
	}

	filters := &grpcservices.EventFilters{
		StartTime:  timestampToTime(req.StartTime),
		EndTime:    timestampToTime(req.EndTime),
		Categories: categories,
	}
	if req.Severity != "" {
		filters.Severity = []string{req.Severity}
	}

	eventChan, err := s.eventSvc.Stream(ctx, tenantID, filters)
	if err != nil {
		s.log.ErrorContext(ctx, LogStreamEventsFailed, logger.FieldError, err)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStreamEventsStartFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStreamEventsStartFailed))
	}

	if err := openStream(ctx, s.log, stream); err != nil {
		return err
	}
	for event := range eventChan {
		if err := stream.Send(eventToProto(event)); err != nil {
			s.log.ErrorContext(ctx, LogStreamSendFailed, logger.FieldError, err)
			return err
		}
	}
	return nil
}

// Helper functions

// ListErrorGroups returns one failure bucket, grouped and paged.
func (s *AnalyticsHandlers) ListErrorGroups(ctx context.Context, req *pb.ListErrorGroupsRequest) (*pb.ListErrorGroupsResponse, error) {
	if s.errorGroups == nil {
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
	groups, total, err := s.errorGroups.List(ctx, tenantID, req.Bucket, scaciWindow(req.StartTime, req.EndTime), page.limit, page.offset)
	switch {
	case errors.Is(err, errorgroups.ErrBucketNotReadable):
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInsufficientRole),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInsufficientRole))
	case errors.Is(err, errorgroups.ErrInvalidBucket):
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidErrorBucket),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidErrorBucket))
	case errors.Is(err, errorgroups.ErrInvalidTimeRange):
		return nil, invalidTimeRangeError()
	case err != nil:
		s.log.ErrorContext(ctx, LogListErrorGroupsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListEventsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListEventsFailed))
	}
	pbGroups := make([]*pb.ErrorGroup, len(groups))
	for i, g := range groups {
		pbGroups[i] = &pb.ErrorGroup{
			Bucket:     g.Bucket,
			EventType:  g.EventType,
			Code:       g.Code,
			Message:    g.Message,
			SourceName: g.SourceName,
			FirstSeen:  timestamppb.New(g.FirstSeen),
			LastSeen:   timestamppb.New(g.LastSeen),
			Count:      g.Count,
			LastOpId:   g.LastOpID,
		}
	}
	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListErrorGroupsResponse{
		Groups:        pbGroups,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
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
		UserId:      e.UserID,
		UserEmail:   e.UserEmail,
	}
}

// EventFeed is the event surface the gRPC layer serves: a filtered page and a
// live stream.
type EventFeed interface {
	List(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error)
	Stream(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error)
}
