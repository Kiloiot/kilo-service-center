// Package grpc provides gRPC service implementations for message listing RPCs.
package grpc

import (
	"context"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// MessageHandlers serves the message listing, streaming and activity RPCs.
type MessageHandlers struct {
	msgListingSvc grpcservices.MessageListingService
	activitySvc   grpcservices.ActivityService
	log           logger.Logger
}

// MessageHandlerDeps wires MessageHandlers; a nil service answers its RPCs
// as not configured.
type MessageHandlerDeps struct {
	Listing  grpcservices.MessageListingService
	Activity grpcservices.ActivityService
}

// NewMessageHandlers builds the group.
func NewMessageHandlers(d MessageHandlerDeps, log logger.Logger) *MessageHandlers {
	return &MessageHandlers{msgListingSvc: d.Listing, activitySvc: d.Activity, log: log}
}

// listing is the message listing service, or the status of an RPC served
// without one.
func (s *MessageHandlers) listing() (grpcservices.MessageListingService, error) {
	if s.msgListingSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	return s.msgListingSvc, nil
}

// GetMessage returns a single message by ID.
func (s *MessageHandlers) GetMessage(ctx context.Context, req *pb.GetMessageRequest) (*pb.Message, error) {
	listing, err := s.listing()
	if err != nil {
		return nil, err
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageIDRequired))
	}
	msg, err := listing.GetMessage(ctx, tenantID, req.Id)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetMessageFailed, logger.FieldMessageIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageNotFound))
	}
	return ulDataMessageToProto(msg), nil
}

// ListMessages returns messages for the tenant.
func (s *MessageHandlers) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	messages, placed, err := s.listUplinks(ctx, req.PageSize, req.PageToken, LogListMessagesFailed, messageFilterRequest{
		epEUI: req.EpEui, bsEUI: req.BsEui, start: req.StartTime, end: req.EndTime,
		duplicate: req.Duplicate, dlOpen: req.DlOpen, profile: req.Profile, mode: req.Mode,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ListMessagesResponse{Messages: messages, NextPageToken: placed.nextToken, TotalCount: placed.totalCount}, nil
}

// ListEndpointMessages returns messages for a specific endpoint.
func (s *MessageHandlers) ListEndpointMessages(ctx context.Context, req *pb.ListEndpointMessagesRequest) (*pb.ListEndpointMessagesResponse, error) {
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}
	messages, placed, err := s.listUplinks(ctx, req.PageSize, req.PageToken, LogListEndpointMessagesFailed, messageFilterRequest{epEUI: req.EpEui})
	if err != nil {
		return nil, err
	}
	return &pb.ListEndpointMessagesResponse{Messages: messages, NextPageToken: placed.nextToken, TotalCount: placed.totalCount}, nil
}

// listUplinks serves one page of the tenant-wide uplink listing.
func (s *MessageHandlers) listUplinks(ctx context.Context, pageSize int32, pageToken, failureLog string, filter messageFilterRequest) ([]*pb.Message, pageResponse, error) {
	listing, err := s.listing()
	if err != nil {
		return nil, pageResponse{}, err
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, pageResponse{}, err
	}
	page, err := readPage(clampPageSize(pageSize), pageToken)
	if err != nil {
		return nil, pageResponse{}, err
	}
	filters, err := buildMessageFilters(filter)
	if err != nil {
		return nil, pageResponse{}, err
	}
	messages, total, err := listing.ListMessages(ctx, tenantID, filters, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, failureLog, logger.FieldEpEuiSnake, filter.epEUI, logger.FieldError, err)
		return nil, pageResponse{}, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListMessagesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListMessagesFailed))
	}
	placed, err := page.respond(total)
	if err != nil {
		return nil, pageResponse{}, err
	}
	var pbMessages []*pb.Message
	for _, msg := range messages {
		pbMessages = append(pbMessages, ulDataMessageToProto(msg))
	}
	return pbMessages, placed, nil
}

// StreamMessages streams messages for the tenant.
func (s *MessageHandlers) StreamMessages(req *pb.StreamMessagesRequest, stream pb.KiloCenterService_StreamMessagesServer) error {
	listing, err := s.listing()
	if err != nil {
		return err
	}
	ctx := stream.Context()
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return err
	}
	filters, err := buildMessageFilters(messageFilterRequest{epEUI: req.EpEui, bsEUI: req.BsEui})
	if err != nil {
		return err
	}
	msgChan, err := listing.StreamMessages(ctx, tenantID, filters)
	if err != nil {
		s.log.ErrorContext(ctx, LogStreamMessagesFailed, logger.FieldError, err)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStreamStartFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStreamStartFailed))
	}
	if err := openStream(ctx, s.log, stream); err != nil {
		return err
	}
	return s.forward(ctx, msgChan, func(msg *mioty.ULDataMessage) error { return stream.Send(ulDataMessageToProto(msg)) })
}

// forward sends every streamed uplink until the stream ends or a send fails.
func (s *MessageHandlers) forward(ctx context.Context, msgChan <-chan *mioty.ULDataMessage, send func(*mioty.ULDataMessage) error) error {
	for msg := range msgChan {
		if err := send(msg); err != nil {
			s.log.ErrorContext(ctx, LogStreamSendFailed, logger.FieldError, err)
			return err
		}
	}
	return nil
}
