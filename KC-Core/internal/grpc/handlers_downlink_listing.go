package grpc

import (
	"context"
	"strconv"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ListDownlinkQueue lists the organization's in-flight downlinks.
func (s *DownlinkHandlers) ListDownlinkQueue(ctx context.Context, req *pb.ListDownlinkQueueRequest) (*pb.ListDownlinkQueueResponse, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	s.log.InfoContext(ctx, LogListingDownlinkQueue, logger.FieldTenantIDSnake, tenantID, logger.FieldEpEuiSnake, req.EpEui, logger.FieldPageSize, req.PageSize)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}
	orgID, err := requireOrganization(ctx)
	if err != nil {
		return nil, err
	}
	filter, err := downlinkQueueFilterFromRequest(req)
	if err != nil {
		return nil, err
	}
	filter.OrganizationID = &orgID
	page, err := readPage(int(clampHighVolumePageSize(req.PageSize, defaultDownlinkQueuePageSize)), req.PageToken)
	if err != nil {
		return nil, err
	}
	entries, total, err := s.queue.ListDownlinkQueue(ctx, tenantID, filter, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetDownlinkQueue, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkQueueListFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkQueueListFailed))
	}
	placed, err := page.respond(total)
	if err != nil {
		return nil, err
	}
	messages := make([]*pb.DownlinkMessage, len(entries))
	for i, entry := range entries {
		messages[i] = downlinkQueueEntryToProto(entry)
	}
	return &pb.ListDownlinkQueueResponse{Messages: messages, NextPageToken: placed.nextToken, TotalCount: placed.totalCount}, nil
}

// GetDownlinkResults lists the organization's finished downlinks with the
// result each ended with (BSSCI §3.14).
func (s *DownlinkHandlers) GetDownlinkResults(ctx context.Context, req *pb.GetDownlinkResultsRequest) (*pb.GetDownlinkResultsResponse, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	s.log.InfoContext(ctx, LogGettingDownlinkResults, logger.FieldTenantIDSnake, tenantID, logger.FieldEpEuiSnake, req.EpEui,
		logger.FieldStatusFilter, req.StatusFilter, logger.FieldPageSize, req.PageSize)
	page, err := readPage(clampDownlinkResultsPageSize(req.PageSize), req.PageToken)
	if err != nil {
		return nil, err
	}
	filter, err := downlinkResultFilterFromRequest(req)
	if err != nil {
		return nil, err
	}
	orgID, err := requireOrganization(ctx)
	if err != nil {
		return nil, err
	}
	results, total, err := s.results.GetDownlinkResults(ctx, tenantID, &orgID, filter, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetDownlinkResults, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkGetResultsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkGetResultsFailed))
	}
	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	messages := make([]*pb.DownlinkMessage, 0, len(results))
	for _, msg := range results {
		messages = append(messages, downlinkToProto(msg))
	}
	return &pb.GetDownlinkResultsResponse{Results: messages, NextPageToken: placed.nextToken, TotalCount: placed.totalCount}, nil
}

// downlinkQueueFilterFromRequest validates the optional queue predicates.
func downlinkQueueFilterFromRequest(req *pb.ListDownlinkQueueRequest) (storage.DownlinkQueueFilter, error) {
	filter := storage.DownlinkQueueFilter{Priority: req.Priority, QueID: req.QueId}
	var err error
	if filter.EpEUI, err = optionalEUIArray(req.EpEui, grpcerrors.ErrTokenInvalidEndpointEUIFormat); err != nil {
		return filter, err
	}
	if filter.BsEUI, err = optionalEUIArray(req.BsEui, grpcerrors.ErrTokenInvalidBasestationEUIFormat); err != nil {
		return filter, err
	}
	if req.Status != "" {
		queueStatus := mioty.DLQueueStatus(req.Status)
		if !queueStatus.Known() {
			return filter, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDownlinkQueueStatus),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDownlinkQueueStatus))
		}
		filter.Status = &queueStatus
	}
	return filter, nil
}

// downlinkResultFilterFromRequest validates the optional results predicates.
// The status filter is a BSSCI §3.14.1 result name, which selects the queue
// status the result is stored as, or a final queue status; an in-flight
// state belongs to the queue listing and is refused.
func downlinkResultFilterFromRequest(req *pb.GetDownlinkResultsRequest) (storage.DownlinkResultFilter, error) {
	filter := storage.DownlinkResultFilter{QueID: req.QueId, From: timestampToTime(req.TimeFrom), To: timestampToTime(req.TimeTo)}
	if req.StatusFilter != "" {
		stored, ok := mioty.QueueStatusForResult(req.StatusFilter)
		if !ok {
			stored = mioty.DLQueueStatus(req.StatusFilter)
		}
		if !stored.Terminal() {
			return filter, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDownlinkResult),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDownlinkResult))
		}
		filter.Status = string(stored)
	}
	var err error
	if filter.EpEUI, err = optionalEUIBytes(req.EpEui, grpcerrors.ErrTokenInvalidEndpointEUIFormat); err != nil {
		return filter, err
	}
	filter.BsEUI, err = optionalEUIBytes(req.BsEui, grpcerrors.ErrTokenInvalidBasestationEUIFormat)
	return filter, err
}

// optionalEUIBytes parses an optional EUI filter; empty is no filter, and a
// value that is no EUI is refused with invalidToken.
func optionalEUIBytes(text, invalidToken string) ([]byte, error) {
	if text == "" {
		return nil, nil
	}
	eui, err := validation.ParseEUIBytes(text)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(invalidToken), grpcerrors.ResolveErrorMessage(invalidToken))
	}
	return eui, nil
}

// optionalEUIArray is optionalEUIBytes for the filters that carry an EUI by value.
func optionalEUIArray(text, invalidToken string) (*[8]byte, error) {
	eui, err := optionalEUIBytes(text, invalidToken)
	if err != nil || eui == nil {
		return nil, err
	}
	return (*[8]byte)(eui), nil
}
