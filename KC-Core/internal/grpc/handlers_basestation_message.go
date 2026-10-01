package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// stationRequest is a base station message request's tenant, listing
// service and station.
type stationRequest struct {
	listing  grpcservices.MessageListingService
	tenantID int64
	bsEUI    []byte
}

// station reads the tenant and the base station a request names.
func (s *MessageHandlers) station(ctx context.Context, bsEUI string) (stationRequest, error) {
	listing, err := s.listing()
	if err != nil {
		return stationRequest{}, err
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return stationRequest{}, err
	}
	eui, err := requiredEUIBytes(bsEUI, grpcerrors.ErrTokenBasestationEUIRequired, grpcerrors.ErrTokenInvalidBasestationEUIFormat)
	if err != nil {
		return stationRequest{}, err
	}
	return stationRequest{listing: listing, tenantID: tenantID, bsEUI: eui}, nil
}

// stationMessagesToProto renders uplinks as the station's listing shows them.
func stationMessagesToProto(messages []*mioty.ULDataMessage, bsEUI []byte) []*pb.BaseStationMessage {
	var pbMessages []*pb.BaseStationMessage
	for _, msg := range messages {
		pbMessages = append(pbMessages, ulDataMessageToBaseStationMessageProto(msg, mioty.EUI64FromBytes(bsEUI)))
	}
	return pbMessages
}

// ListBaseStationMessages returns messages for a specific base station.
func (s *MessageHandlers) ListBaseStationMessages(ctx context.Context, req *pb.ListBaseStationMessagesRequest) (*pb.ListBaseStationMessagesResponse, error) {
	station, err := s.station(ctx, req.BsEui)
	if err != nil {
		return nil, err
	}
	page, err := readPage(clampPageSize(req.PageSize), req.PageToken)
	if err != nil {
		return nil, err
	}
	filters, err := buildMessageFilters(messageFilterRequest{
		epEUI: req.EpEui, direction: req.Direction, start: req.StartTime, end: req.EndTime,
		duplicate: req.Duplicate, dlOpen: req.DlOpen, profile: req.Profile, mode: req.Mode,
	})
	if err != nil {
		return nil, err
	}
	messages, total, err := station.listing.ListBaseStationMessages(ctx, station.tenantID, station.bsEUI, filters, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListBaseStationMessagesFailed, logger.FieldBsEuiSnake, req.BsEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListBSMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListBSMessagesFailed))
	}
	placed, err := page.respond(total)
	if err != nil {
		return nil, err
	}
	return &pb.ListBaseStationMessagesResponse{
		Messages: stationMessagesToProto(messages, station.bsEUI), NextPageToken: placed.nextToken, TotalCount: placed.totalCount,
	}, nil
}

// GetBaseStationMessage returns a specific message.
func (s *MessageHandlers) GetBaseStationMessage(ctx context.Context, req *pb.GetBaseStationMessageRequest) (*pb.GetBaseStationMessageResponse, error) {
	station, err := s.station(ctx, req.BsEui)
	if err != nil {
		return nil, err
	}
	if req.MessageId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageIDRequired))
	}
	msg, err := station.listing.GetBaseStationMessage(ctx, station.tenantID, station.bsEUI, req.MessageId)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetBaseStationMessageFailed, logger.FieldMessageIDSnake, req.MessageId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageNotFound))
	}
	return &pb.GetBaseStationMessageResponse{Message: ulDataMessageToBaseStationMessageProto(msg, mioty.EUI64FromBytes(station.bsEUI))}, nil
}

// GetBaseStationMessageStats returns message statistics for a base station.
func (s *MessageHandlers) GetBaseStationMessageStats(ctx context.Context, req *pb.GetBaseStationMessageStatsRequest) (*pb.GetBaseStationMessageStatsResponse, error) {
	station, err := s.station(ctx, req.BsEui)
	if err != nil {
		return nil, err
	}
	stats, err := station.listing.GetBaseStationMessageStats(ctx, station.tenantID, station.bsEUI, timestampToTime(req.StartTime), timestampToTime(req.EndTime))
	if err != nil {
		s.log.ErrorContext(ctx, LogGetBaseStationMessageStatsFailed, logger.FieldBsEuiSnake, req.BsEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageStatsFailed))
	}
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
	if stats.FirstMessageAt != nil {
		pbStats.FirstMessageAt = timestamppb.New(*stats.FirstMessageAt)
	}
	if stats.LastMessageAt != nil {
		pbStats.LastMessageAt = timestamppb.New(*stats.LastMessageAt)
	}
	return &pb.GetBaseStationMessageStatsResponse{Stats: pbStats}, nil
}

// SearchBaseStationMessages searches messages for a base station.
func (s *MessageHandlers) SearchBaseStationMessages(ctx context.Context, req *pb.SearchBaseStationMessagesRequest) (*pb.SearchBaseStationMessagesResponse, error) {
	station, err := s.station(ctx, req.BsEui)
	if err != nil {
		return nil, err
	}
	if req.Query == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenQueryRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenQueryRequired))
	}
	page, err := readPage(clampPageSize(req.PageSize), req.PageToken)
	if err != nil {
		return nil, err
	}
	filters, err := buildMessageFilters(messageFilterRequest{epEUI: req.EpEui, direction: req.Direction, start: req.StartTime, end: req.EndTime})
	if err != nil {
		return nil, err
	}
	messages, total, err := station.listing.SearchBaseStationMessages(ctx, station.tenantID, station.bsEUI, req.Query, filters, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogSearchBaseStationMessagesFailed, logger.FieldBsEuiSnake, req.BsEui, logger.FieldQuery, req.Query, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSearchMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSearchMessagesFailed))
	}
	placed, err := page.respond(total)
	if err != nil {
		return nil, err
	}
	return &pb.SearchBaseStationMessagesResponse{
		Messages: stationMessagesToProto(messages, station.bsEUI), NextPageToken: placed.nextToken, TotalCount: placed.totalCount,
	}, nil
}

// ExportBaseStationMessages exports messages for a base station.
func (s *MessageHandlers) ExportBaseStationMessages(ctx context.Context, req *pb.ExportBaseStationMessagesRequest) (*pb.ExportBaseStationMessagesResponse, error) {
	station, err := s.station(ctx, req.BsEui)
	if err != nil {
		return nil, err
	}
	if token := exportFormatRefusal(req.Format); token != "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
	}
	filters, err := buildMessageFilters(messageFilterRequest{epEUI: req.EpEui, direction: req.Direction, start: req.StartTime, end: req.EndTime})
	if err != nil {
		return nil, err
	}
	data, err := station.listing.ExportBaseStationMessages(ctx, station.tenantID, station.bsEUI, filters, req.Format)
	if err != nil {
		s.log.ErrorContext(ctx, LogExportBaseStationMessagesFailed, logger.FieldBsEuiSnake, req.BsEui, logger.FieldFormat, req.Format, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenExportMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenExportMessagesFailed))
	}
	return &pb.ExportBaseStationMessagesResponse{
		Content:     data,
		Filename:    fmt.Sprintf(exportFilenameFmt, req.BsEui, req.Format),
		ContentType: getContentTypeForFormat(req.Format),
	}, nil
}

// exportFormatRefusal is the token refusing an export format, empty for a supported one.
func exportFormatRefusal(format string) string {
	switch {
	case format == "":
		return grpcerrors.ErrTokenFormatRequired
	case !grpcerrors.IsValidExportFormat(format):
		return grpcerrors.ErrTokenExportUnsupportedFormat
	default:
		return ""
	}
}

// StreamBaseStationMessages streams messages for a specific base station.
func (s *MessageHandlers) StreamBaseStationMessages(req *pb.StreamBaseStationMessagesRequest, stream pb.KiloCenterService_StreamBaseStationMessagesServer) error {
	ctx := stream.Context()
	station, err := s.station(ctx, req.BsEui)
	if err != nil {
		return err
	}
	filters, err := buildMessageFilters(messageFilterRequest{epEUI: req.EpEui, direction: req.Direction})
	if err != nil {
		return err
	}
	msgChan, err := station.listing.StreamBaseStationMessages(ctx, station.tenantID, station.bsEUI, filters)
	if err != nil {
		s.log.ErrorContext(ctx, LogStreamBaseStationMessagesFailed, logger.FieldBsEuiSnake, req.BsEui, logger.FieldError, err)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStreamBSMessagesStartFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStreamBSMessagesStartFailed))
	}
	if err := openStream(ctx, s.log, stream); err != nil {
		return err
	}
	bsEUI := mioty.EUI64FromBytes(station.bsEUI)
	return s.forward(ctx, msgChan, func(msg *mioty.ULDataMessage) error {
		return stream.Send(ulDataMessageToBaseStationMessageProto(msg, bsEUI))
	})
}
