package grpc

import (
	"context"
	"math"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// activityRequest is what both activity feeds are asked for.
type activityRequest struct {
	pageSize  int32
	pageToken string
	start     *timestamppb.Timestamp
	end       *timestamppb.Timestamp
}

func (r activityRequest) page() int {
	if r.pageSize <= 0 || r.pageSize > MaxPageSize {
		return DefaultActivityPageSize
	}
	return int(r.pageSize)
}

func (r activityRequest) filters() *grpcservices.ActivityFilters {
	return &grpcservices.ActivityFilters{StartTime: timestampToTime(r.start), EndTime: timestampToTime(r.end)}
}

// activityTotal caps a feed's total to the int32 the response carries.
func activityTotal(total int64) int32 {
	if total > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(total) // #nosec G115 - checked above
}

// deviceEUI validates the EUI of the device a feed is asked for.
func deviceEUI(eui, requiredToken, formatToken string) ([]byte, error) {
	if eui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(requiredToken), grpcerrors.ResolveErrorMessage(requiredToken))
	}
	parsed, err := validation.ParseEUIBytes(eui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(formatToken), grpcerrors.ResolveErrorMessage(formatToken))
	}
	return parsed, nil
}

// ListBaseStationActivity returns unified activity feed (events + messages) for a base station.
func (s *MessageHandlers) ListBaseStationActivity(ctx context.Context, req *pb.ListBaseStationActivityRequest) (*pb.ListBaseStationActivityResponse, error) {
	if s.activitySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	bsEui, err := deviceEUI(req.BsEui, grpcerrors.ErrTokenBasestationEUIRequired, grpcerrors.ErrTokenInvalidBasestationEUIFormat)
	if err != nil {
		return nil, err
	}
	ask := activityRequest{req.PageSize, req.PageToken, req.StartTime, req.EndTime}
	result, err := s.activitySvc.ListBaseStationActivity(ctx, tenantID, bsEui, ask.filters(), ask.page(), ask.pageToken)
	if err != nil {
		s.log.ErrorContext(ctx, LogListBaseStationActivityFailed, logger.FieldBsEuiSnake, req.BsEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListActivityFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListActivityFailed))
	}
	items := make([]*pb.BaseStationActivityItem, 0, len(result.Items))
	for _, item := range result.Items {
		pbItem := &pb.BaseStationActivityItem{OccurredAt: timestamppb.New(item.OccurredAt)}
		switch {
		case item.Type == grpcservices.ActivityItemTypeEvent && item.Event != nil:
			pbItem.Item = &pb.BaseStationActivityItem_Event{Event: eventToProto(item.Event)}
		case item.Type == grpcservices.ActivityItemTypeMessage && item.Message != nil:
			pbItem.Item = &pb.BaseStationActivityItem_Message{Message: ulDataMessageToBaseStationMessageProto(item.Message, mioty.EUI64FromBytes(bsEui))}
		}
		items = append(items, pbItem)
	}
	return &pb.ListBaseStationActivityResponse{
		Items: items, NextPageToken: result.NextPageToken, TotalCount: activityTotal(result.TotalCount),
	}, nil
}

// ListEndpointActivity returns unified activity feed (events + messages) for an endpoint.
func (s *MessageHandlers) ListEndpointActivity(ctx context.Context, req *pb.ListEndpointActivityRequest) (*pb.ListEndpointActivityResponse, error) {
	if s.activitySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	epEui, err := deviceEUI(req.EpEui, grpcerrors.ErrTokenEndpointEUIRequired, grpcerrors.ErrTokenInvalidEndpointEUIFormat)
	if err != nil {
		return nil, err
	}
	ask := activityRequest{req.PageSize, req.PageToken, req.StartTime, req.EndTime}
	result, err := s.activitySvc.ListEndpointActivity(ctx, tenantID, epEui, ask.filters(), ask.page(), ask.pageToken)
	if err != nil {
		s.log.ErrorContext(ctx, LogListEndpointActivityFailed, logger.FieldEpEuiSnake, req.EpEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListActivityFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListActivityFailed))
	}
	items := make([]*pb.EndpointActivityItem, 0, len(result.Items))
	for _, item := range result.Items {
		pbItem := &pb.EndpointActivityItem{OccurredAt: timestamppb.New(item.OccurredAt)}
		switch {
		case item.Type == grpcservices.ActivityItemTypeEvent && item.Event != nil:
			pbItem.Item = &pb.EndpointActivityItem_Event{Event: eventToProto(item.Event)}
		case item.Type == grpcservices.ActivityItemTypeMessage && item.Message != nil:
			pbItem.Item = &pb.EndpointActivityItem_Message{Message: ulDataMessageToProto(item.Message)}
		}
		items = append(items, pbItem)
	}
	return &pb.ListEndpointActivityResponse{
		Items: items, NextPageToken: result.NextPageToken, TotalCount: activityTotal(result.TotalCount),
	}, nil
}
