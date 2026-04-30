// Package grpc provides gRPC service implementations for message listing RPCs.
package grpc

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage/mioty"
)

// Message listing handlers

// GetMessage returns a single message by ID.
func (s *CoreService) GetMessage(ctx context.Context, req *pb.GetMessageRequest) (*pb.Message, error) {
	if s.msgListingSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageIDRequired))
	}

	msg, err := s.msgListingSvc.GetMessage(ctx, tenantID, req.Id)
	if err != nil {
		s.log.ErrorContext(ctx, "get message failed", "message_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageNotFound))
	}

	return ulDataMessageToProto(msg), nil
}

// ListMessages returns messages for the tenant.
func (s *CoreService) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	if s.msgListingSvc == nil {
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

	filters, err := buildMessageFilters(req.EpEui, req.BsEui, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	messages, total, err := s.msgListingSvc.ListMessages(ctx, tenantID, filters, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list messages failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListMessagesFailed))
	}

	var pbMessages []*pb.Message
	for _, msg := range messages {
		pbMessages = append(pbMessages, ulDataMessageToProto(msg))
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
	return &pb.ListMessagesResponse{
		Messages:      pbMessages,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// StreamMessages streams messages for the tenant.
func (s *CoreService) StreamMessages(req *pb.StreamMessagesRequest, stream pb.KiloCenterService_StreamMessagesServer) error {
	if s.msgListingSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	ctx := stream.Context()
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return err
	}

	// StreamMessages only has ep_eui and bs_eui filters per proto
	filters, err := buildMessageFilters(req.EpEui, req.BsEui, nil, nil)
	if err != nil {
		return err
	}

	msgChan, err := s.msgListingSvc.StreamMessages(ctx, tenantID, filters)
	if err != nil {
		s.log.ErrorContext(ctx, "stream messages failed", "error", err)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStreamStartFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStreamStartFailed))
	}

	for msg := range msgChan {
		if err := stream.Send(ulDataMessageToProto(msg)); err != nil {
			s.log.ErrorContext(ctx, "stream send failed", "error", err)
			return err
		}
	}

	return nil
}

// ListEndpointMessages returns messages for a specific endpoint.
func (s *CoreService) ListEndpointMessages(ctx context.Context, req *pb.ListEndpointMessagesRequest) (*pb.ListEndpointMessagesResponse, error) {
	if s.msgListingSvc == nil {
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

	epEui, err := parseEUI(req.EpEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	filters, err := buildMessageFilters(req.EpEui, "", nil, nil)
	if err != nil {
		return nil, err
	}

	messages, total, err := s.msgListingSvc.ListMessages(ctx, tenantID, filters, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list endpoint messages failed", "ep_eui", req.EpEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListMessagesFailed))
	}

	var pbMessages []*pb.Message
	for _, msg := range messages {
		pbMessages = append(pbMessages, ulDataMessageToProto(msg))
	}

	var nextToken string
	if offset+limit < int(total) {
		nextToken = generatePaginationToken(offset + limit)
	}

	if total > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	_ = epEui // Used for validation, filter uses string form

	totalCount := int32(total) //nolint:gosec // bounds checked above
	return &pb.ListEndpointMessagesResponse{
		Messages:      pbMessages,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// ListBaseStationMessages returns messages for a specific base station.
func (s *CoreService) ListBaseStationMessages(ctx context.Context, req *pb.ListBaseStationMessagesRequest) (*pb.ListBaseStationMessagesResponse, error) {
	if s.msgListingSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	filters, err := buildMessageFiltersWithDirection(req.EpEui, "", req.Direction, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	messages, total, err := s.msgListingSvc.ListBaseStationMessages(ctx, tenantID, bsEui, filters, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list base station messages failed", "bs_eui", req.BsEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListBSMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListBSMessagesFailed))
	}

	var pbMessages []*pb.BaseStationMessage
	for _, msg := range messages {
		pbMessages = append(pbMessages, ulDataMessageToBaseStationMessageProto(msg, bsEui))
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
	return &pb.ListBaseStationMessagesResponse{
		Messages:      pbMessages,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// GetBaseStationMessage returns a specific message.
func (s *CoreService) GetBaseStationMessage(ctx context.Context, req *pb.GetBaseStationMessageRequest) (*pb.GetBaseStationMessageResponse, error) {
	if s.msgListingSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}
	if req.MessageId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageIDRequired))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	msg, err := s.msgListingSvc.GetBaseStationMessage(ctx, tenantID, bsEui, req.MessageId)
	if err != nil {
		s.log.ErrorContext(ctx, "get base station message failed", "message_id", req.MessageId, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageNotFound))
	}

	return &pb.GetBaseStationMessageResponse{
		Message: ulDataMessageToBaseStationMessageProto(msg, bsEui),
	}, nil
}

// GetBaseStationMessageStats returns message statistics for a base station.
func (s *CoreService) GetBaseStationMessageStats(ctx context.Context, req *pb.GetBaseStationMessageStatsRequest) (*pb.GetBaseStationMessageStatsResponse, error) {
	if s.msgListingSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	startTime := timestampToTime(req.StartTime)
	endTime := timestampToTime(req.EndTime)

	stats, err := s.msgListingSvc.GetBaseStationMessageStats(ctx, tenantID, bsEui, startTime, endTime)
	if err != nil {
		s.log.ErrorContext(ctx, "get base station message stats failed", "bs_eui", req.BsEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageStatsFailed))
	}

	// Build nested stats object per proto definition
	pbStats := &pb.BaseStationMessageStats{
		BsEui:             req.BsEui,
		TotalMessages:     stats.TotalMessages,
		UniqueEndpoints:   stats.TotalEndpoints,
		AvgRssi:           stats.AvgRSSI,
		AvgSnr:            stats.AvgSNR,
		MessagesToday:     stats.MessagesToday,
		MessagesThisWeek:  stats.MessagesThisWeek,
		MessagesThisMonth: stats.MessagesThisMonth,
	}
	if stats.LastMessageAt != nil {
		pbStats.LastMessageAt = timestamppb.New(*stats.LastMessageAt)
	}

	return &pb.GetBaseStationMessageStatsResponse{
		Stats: pbStats,
	}, nil
}

// SearchBaseStationMessages searches messages for a base station.
func (s *CoreService) SearchBaseStationMessages(ctx context.Context, req *pb.SearchBaseStationMessagesRequest) (*pb.SearchBaseStationMessagesResponse, error) {
	if s.msgListingSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}
	if req.Query == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenQueryRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenQueryRequired))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	limit := clampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := parsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	filters, err := buildMessageFiltersWithDirection(req.EpEui, "", req.Direction, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	messages, total, err := s.msgListingSvc.SearchBaseStationMessages(ctx, tenantID, bsEui, req.Query, filters, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "search base station messages failed", "bs_eui", req.BsEui, "query", req.Query, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSearchMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSearchMessagesFailed))
	}

	var pbMessages []*pb.BaseStationMessage
	for _, msg := range messages {
		pbMessages = append(pbMessages, ulDataMessageToBaseStationMessageProto(msg, bsEui))
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
	return &pb.SearchBaseStationMessagesResponse{
		Messages:      pbMessages,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// ExportBaseStationMessages exports messages for a base station.
func (s *CoreService) ExportBaseStationMessages(ctx context.Context, req *pb.ExportBaseStationMessagesRequest) (*pb.ExportBaseStationMessagesResponse, error) {
	if s.msgListingSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}
	if req.Format == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFormatRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFormatRequired))
	}
	if !grpcerrors.IsValidExportFormat(req.Format) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenExportUnsupportedFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenExportUnsupportedFormat))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	filters, err := buildMessageFiltersWithDirection(req.EpEui, "", req.Direction, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	data, err := s.msgListingSvc.ExportBaseStationMessages(ctx, tenantID, bsEui, filters, req.Format)
	if err != nil {
		s.log.ErrorContext(ctx, "export base station messages failed", "bs_eui", req.BsEui, "format", req.Format, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenExportMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenExportMessagesFailed))
	}

	return &pb.ExportBaseStationMessagesResponse{
		Content:     data,
		Filename:    fmt.Sprintf("messages_%s.%s", req.BsEui, req.Format),
		ContentType: getContentTypeForFormat(req.Format),
	}, nil
}

// StreamBaseStationMessages streams messages for a specific base station.
func (s *CoreService) StreamBaseStationMessages(req *pb.StreamBaseStationMessagesRequest, stream pb.KiloCenterService_StreamBaseStationMessagesServer) error {
	if s.msgListingSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	ctx := stream.Context()
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return err
	}

	if req.BsEui == "" {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	filters, err := buildMessageFiltersWithDirection(req.EpEui, "", req.Direction, nil, nil)
	if err != nil {
		return err
	}

	msgChan, err := s.msgListingSvc.StreamBaseStationMessages(ctx, tenantID, bsEui, filters)
	if err != nil {
		s.log.ErrorContext(ctx, "stream base station messages failed", "bs_eui", req.BsEui, "error", err)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStreamBSMessagesStartFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStreamBSMessagesStartFailed))
	}

	for msg := range msgChan {
		if err := stream.Send(ulDataMessageToBaseStationMessageProto(msg, bsEui)); err != nil {
			s.log.ErrorContext(ctx, "stream send failed", "error", err)
			return err
		}
	}

	return nil
}

// ListBaseStationActivity returns unified activity feed (events + messages) for a base station.
func (s *CoreService) ListBaseStationActivity(ctx context.Context, req *pb.ListBaseStationActivityRequest) (*pb.ListBaseStationActivityResponse, error) {
	if s.activitySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 || pageSize > MaxPageSize {
		pageSize = 50
	}

	// Build activity filters
	filters := &grpcservices.ActivityFilters{
		StartTime: timestampToTime(req.StartTime),
		EndTime:   timestampToTime(req.EndTime),
	}

	result, err := s.activitySvc.ListBaseStationActivity(ctx, tenantID, bsEui, filters, pageSize, req.PageToken)
	if err != nil {
		s.log.ErrorContext(ctx, "list base station activity failed", "bs_eui", req.BsEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListActivityFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListActivityFailed))
	}

	// Convert to proto response
	pbItems := make([]*pb.BaseStationActivityItem, 0, len(result.Items))
	for _, item := range result.Items {
		pbItem := &pb.BaseStationActivityItem{
			OccurredAt: timestamppb.New(item.OccurredAt),
		}

		if item.Type == grpcservices.ActivityItemTypeEvent && item.Event != nil {
			pbItem.Item = &pb.BaseStationActivityItem_Event{
				Event: &pb.Event{
					Id:          item.Event.ID,
					EventType:   item.Event.EventType,
					Category:    item.Event.Category,
					Severity:    item.Event.Severity,
					Title:       item.Event.Title,
					Description: item.Event.Description,
					Timestamp:   timestamppb.New(item.Event.Timestamp),
				},
			}
		} else if item.Type == grpcservices.ActivityItemTypeMessage && item.Message != nil {
			pbItem.Item = &pb.BaseStationActivityItem_Message{
				Message: ulDataMessageToBaseStationMessageProto(item.Message, bsEui),
			}
		}

		pbItems = append(pbItems, pbItem)
	}

	var totalCount int32
	maxInt32 := int64(^uint32(0) >> 1)
	if result.TotalCount > maxInt32 {
		totalCount = int32(maxInt32) // Cap at max int32
	} else {
		totalCount = int32(result.TotalCount) // #nosec G115 - checked above
	}

	return &pb.ListBaseStationActivityResponse{
		Items:         pbItems,
		NextPageToken: result.NextPageToken,
		TotalCount:    totalCount,
	}, nil
}

// ListEndpointActivity returns unified activity feed (events + messages) for an endpoint.
func (s *CoreService) ListEndpointActivity(ctx context.Context, req *pb.ListEndpointActivityRequest) (*pb.ListEndpointActivityResponse, error) {
	if s.activitySvc == nil {
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

	epEui, err := parseEUI(req.EpEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 || pageSize > MaxPageSize {
		pageSize = 50
	}

	filters := &grpcservices.ActivityFilters{
		StartTime: timestampToTime(req.StartTime),
		EndTime:   timestampToTime(req.EndTime),
	}

	result, err := s.activitySvc.ListEndpointActivity(ctx, tenantID, epEui, filters, pageSize, req.PageToken)
	if err != nil {
		s.log.ErrorContext(ctx, "list endpoint activity failed", "ep_eui", req.EpEui, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListActivityFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListActivityFailed))
	}

	pbItems := make([]*pb.EndpointActivityItem, 0, len(result.Items))
	for _, item := range result.Items {
		pbItem := &pb.EndpointActivityItem{
			OccurredAt: timestamppb.New(item.OccurredAt),
		}

		if item.Type == grpcservices.ActivityItemTypeEvent && item.Event != nil {
			pbItem.Item = &pb.EndpointActivityItem_Event{
				Event: &pb.Event{
					Id:          item.Event.ID,
					EventType:   item.Event.EventType,
					Category:    item.Event.Category,
					Severity:    item.Event.Severity,
					Title:       item.Event.Title,
					Description: item.Event.Description,
					Timestamp:   timestamppb.New(item.Event.Timestamp),
				},
			}
		} else if item.Type == grpcservices.ActivityItemTypeMessage && item.Message != nil {
			pbItem.Item = &pb.EndpointActivityItem_Message{
				Message: ulDataMessageToProto(item.Message),
			}
		}

		pbItems = append(pbItems, pbItem)
	}

	var totalCount int32
	maxInt32 := int64(^uint32(0) >> 1)
	if result.TotalCount > maxInt32 {
		totalCount = int32(maxInt32)
	} else {
		totalCount = int32(result.TotalCount) // #nosec G115 - checked above
	}

	return &pb.ListEndpointActivityResponse{
		Items:         pbItems,
		NextPageToken: result.NextPageToken,
		TotalCount:    totalCount,
	}, nil
}

// Helper functions

// buildMessageFilters constructs message filters from request parameters.
// Returns errors for invalid EUI formats.
func buildMessageFilters(epEui, bsEui string, startTime, endTime *timestamppb.Timestamp) (*grpcservices.MessageFilters, error) {
	return buildMessageFiltersWithDirection(epEui, bsEui, "", startTime, endTime)
}

// buildMessageFiltersWithDirection constructs message filters including direction filter.
func buildMessageFiltersWithDirection(epEui, bsEui, direction string, startTime, endTime *timestamppb.Timestamp) (*grpcservices.MessageFilters, error) {
	filters := &grpcservices.MessageFilters{
		StartTime: timestampToTime(startTime),
		EndTime:   timestampToTime(endTime),
		Direction: direction,
	}
	if epEui != "" {
		eui, err := parseEUI(epEui)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
		}
		filters.EpEui = eui
	}
	if bsEui != "" {
		eui, err := parseEUI(bsEui)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
		}
		filters.BsEui = eui
	}
	return filters, nil
}

func ulDataMessageToProto(msg *mioty.ULDataMessage) *pb.Message {
	if msg == nil {
		return nil
	}
	pbMsg := &pb.Message{
		Id:            msg.ID,
		EpEui:         fmt.Sprintf("%016x", msg.EpEui),
		BsEui:         fmt.Sprintf("%016x", msg.BsEui),
		TenantId:      fmt.Sprintf("%d", msg.TenantID),
		Payload:       msg.UserData,
		Rssi:          msg.RSSI,
		Snr:           msg.SNR,
		PacketCounter: msg.PacketCnt,
		DlOpen:        msg.DlOpen,
		ResExp:        msg.ResponseExp,
		DlAck:         msg.DlAck,
		ReceivedAt:    timestamppb.New(msg.ReceivedAt),
	}
	// Handle optional EqSnr pointer
	if msg.EqSnr != nil {
		pbMsg.EqSnr = *msg.EqSnr
	}
	// Handle optional Mode pointer for uplink_mode
	if msg.Mode != nil {
		pbMsg.UplinkMode = *msg.Mode
	}
	// Blueprint decode fields
	if len(msg.DecodedPayload) > 0 {
		pbMsg.DecodedPayload = msg.DecodedPayload
	}
	if msg.DecodeStatus != "" {
		pbMsg.DecodeStatus = msg.DecodeStatus
	}
	if msg.DecodeErrorCode != "" {
		pbMsg.DecodeErrorCode = msg.DecodeErrorCode
	}
	if len(msg.BlueprintTypeEUI) > 0 {
		pbMsg.BlueprintTypeEui = msg.BlueprintTypeEUI
	}
	if msg.BlueprintVersionID != nil {
		pbMsg.BlueprintVersionId = msg.BlueprintVersionID.String()
	}
	// Multi-BS reception data per SCACI §3.8.1
	pbMsg.BaseStations = baseStationReceptionsToProto(msg.BaseStations)
	if msg.Duplicate != nil {
		pbMsg.Duplicate = *msg.Duplicate
	}
	return pbMsg
}

func ulDataMessageToBaseStationMessageProto(msg *mioty.ULDataMessage, targetBsEuiBytes []byte) *pb.BaseStationMessage {
	if msg == nil {
		return nil
	}
	pbMsg := &pb.BaseStationMessage{
		Id:            msg.ID,
		BsEui:         fmt.Sprintf("%016x", msg.BsEui),
		EpEui:         fmt.Sprintf("%016x", msg.EpEui),
		Payload:       msg.UserData,
		Rssi:          msg.RSSI,
		Snr:           msg.SNR,
		PacketCounter: msg.PacketCnt,
		ReceivedAt:    timestamppb.New(msg.ReceivedAt),
		Direction:     mioty.DirectionUplink, // ULDataMessage is always uplink
	}
	// Handle optional EqSnr pointer
	if msg.EqSnr != nil {
		pbMsg.EqSnr = *msg.EqSnr
	}
	// Handle optional Mode pointer for uplink_mode
	if msg.Mode != nil {
		pbMsg.UplinkMode = *msg.Mode
	}
	// Multi-BS reception data per SCACI §3.8.1
	pbMsg.BaseStations = baseStationReceptionsToProto(msg.BaseStations)
	if msg.Duplicate != nil {
		pbMsg.Duplicate = *msg.Duplicate
	}
	// Blueprint decode fields
	if len(msg.DecodedPayload) > 0 {
		pbMsg.DecodedPayload = msg.DecodedPayload
	}
	if msg.DecodeStatus != "" {
		pbMsg.DecodeStatus = msg.DecodeStatus
	}
	if msg.DecodeErrorCode != "" {
		pbMsg.DecodeErrorCode = msg.DecodeErrorCode
	}
	// When BaseStations is populated, project per-BS signal values for the target BS
	if len(targetBsEuiBytes) == 8 && len(msg.BaseStations) > 0 {
		targetBsEui := binary.BigEndian.Uint64(targetBsEuiBytes)
		for _, bs := range msg.BaseStations {
			if bs.BsEui == targetBsEui {
				pbMsg.BsEui = fmt.Sprintf("%016x", targetBsEui)
				pbMsg.Rssi = bs.Rssi
				pbMsg.Snr = bs.Snr
				pbMsg.EqSnr = 0
				if bs.EqSnr != nil {
					pbMsg.EqSnr = *bs.EqSnr
				}
				break
			}
		}
	}
	return pbMsg
}

// baseStationReceptionsToProto converts BaseStationReception slice to proto BaseStationReceptionInfo
func baseStationReceptionsToProto(receptions []mioty.BaseStationReception) []*pb.BaseStationReceptionInfo {
	if len(receptions) == 0 {
		return nil
	}
	result := make([]*pb.BaseStationReceptionInfo, 0, len(receptions))
	for _, bs := range receptions {
		info := &pb.BaseStationReceptionInfo{
			BsEui:  fmt.Sprintf("%016x", bs.BsEui),
			RxTime: bs.RxTime,
			Snr:    bs.Snr,
			Rssi:   bs.Rssi,
		}
		if bs.EqSnr != nil {
			info.EqSnr = wrapperspb.Double(*bs.EqSnr)
		}
		if bs.RxDuration != nil {
			info.RxDuration = wrapperspb.Int64(*bs.RxDuration)
		}
		if bs.Profile != nil {
			info.Profile = wrapperspb.String(*bs.Profile)
		}
		if bs.Mode != nil {
			info.Mode = wrapperspb.String(*bs.Mode)
		}
		if bs.DlRxSnr != nil {
			info.DlRxSnr = wrapperspb.Double(*bs.DlRxSnr)
		}
		if bs.DlRxRssi != nil {
			info.DlRxRssi = wrapperspb.Double(*bs.DlRxRssi)
		}
		if bs.Subpackets != nil {
			info.Subpackets = &pb.SubpacketInfo{
				Snr:       bs.Subpackets.SNR,
				Rssi:      bs.Subpackets.RSSI,
				Frequency: bs.Subpackets.Frequency,
				Phase:     bs.Subpackets.Phase,
			}
		}
		result = append(result, info)
	}
	return result
}

// Format helpers

func getContentTypeForFormat(format string) string {
	switch format {
	case grpcerrors.ExportFormatCSV:
		return grpcerrors.ContentTypeCSV
	case grpcerrors.ExportFormatJSON:
		return grpcerrors.ContentTypeJSON
	default:
		return grpcerrors.ContentTypeOctetStream
	}
}

// Pagination helpers — delegate to KC-Core/pkg/grpc for cross-service use.

func parsePaginationToken(token string, offset *int) (bool, error) {
	return grpcerrors.ParsePaginationToken(token, offset)
}

func generatePaginationToken(offset int) string {
	return grpcerrors.GeneratePaginationToken(offset)
}
