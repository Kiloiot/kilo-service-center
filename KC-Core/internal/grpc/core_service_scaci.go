// Package grpc provides gRPC service implementations.
// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"math"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
)

// SCACI Monitoring handlers

// ListScaciSessions returns SCACI sessions.
func (s *CoreService) ListScaciSessions(ctx context.Context, req *pb.ListScaciSessionsRequest) (*pb.ListScaciSessionsResponse, error) {
	if s.scaciMonitorSvc == nil {
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

	sessions, total, err := s.scaciMonitorSvc.ListSessions(ctx, tenantID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list SCACI sessions failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciListSessionsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciListSessionsFailed))
	}

	var pbSessions []*pb.ScaciSession
	for _, session := range sessions {
		pbSessions = append(pbSessions, scaciSessionToProto(session))
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
	return &pb.ListScaciSessionsResponse{
		Sessions:      pbSessions,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// GetScaciSession returns a specific SCACI session.
func (s *CoreService) GetScaciSession(ctx context.Context, req *pb.GetScaciSessionRequest) (*pb.GetScaciSessionResponse, error) {
	if s.scaciMonitorSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	session, err := s.scaciMonitorSvc.GetSession(ctx, tenantID, req.Id)
	if err != nil {
		s.log.ErrorContext(ctx, "get SCACI session failed", "session_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciSessionNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciSessionNotFound))
	}

	return &pb.GetScaciSessionResponse{Session: scaciSessionToProto(session)}, nil
}

// GetScaciStatistics returns SCACI operation statistics.
func (s *CoreService) GetScaciStatistics(ctx context.Context, _ *pb.GetScaciStatisticsRequest) (*pb.GetScaciStatisticsResponse, error) {
	if s.scaciMonitorSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := s.scaciMonitorSvc.GetStatistics(ctx, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "get SCACI statistics failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciStatisticsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciStatisticsFailed))
	}

	resp := &pb.GetScaciStatisticsResponse{
		Statistics: &pb.ScaciStatistics{
			TotalSessions:        stats.TotalSessions,
			ActiveSessions:       stats.ActiveSessions,
			TotalOperations:      stats.TotalOperations,
			SuccessfulOperations: stats.SuccessfulOperations,
			FailedOperations:     stats.FailedOperations,
			SuccessRate:          stats.SuccessRate,
		},
	}
	if stats.UptimeSince != nil {
		resp.Statistics.UptimeSince = timestamppb.New(*stats.UptimeSince)
	}
	return resp, nil
}

// ListScaciErrors returns SCACI errors.
func (s *CoreService) ListScaciErrors(ctx context.Context, req *pb.ListScaciErrorsRequest) (*pb.ListScaciErrorsResponse, error) {
	if s.scaciMonitorSvc == nil {
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

	errors, total, err := s.scaciMonitorSvc.ListErrors(ctx, tenantID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list SCACI errors failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciListErrorsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciListErrorsFailed))
	}

	var pbErrors []*pb.ScaciError
	for _, e := range errors {
		pbErrors = append(pbErrors, scaciErrorToProto(e))
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
	return &pb.ListScaciErrorsResponse{
		Errors:        pbErrors,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// ListScaciQueues returns SCACI queue status.
func (s *CoreService) ListScaciQueues(ctx context.Context, req *pb.ListScaciQueuesRequest) (*pb.ListScaciQueuesResponse, error) {
	if s.scaciMonitorSvc == nil {
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

	queues, total, err := s.scaciMonitorSvc.ListQueues(ctx, tenantID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list SCACI queues failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciListQueuesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciListQueuesFailed))
	}

	var pbQueues []*pb.ScaciQueueEntry
	for _, q := range queues {
		pbQueues = append(pbQueues, scaciQueueEntryToProto(q))
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
	return &pb.ListScaciQueuesResponse{
		QueueEntries:  pbQueues,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// GetScaciStatus returns overall SCACI status.
func (s *CoreService) GetScaciStatus(ctx context.Context, _ *pb.GetScaciStatusRequest) (*pb.GetScaciStatusResponse, error) {
	if s.scaciMonitorSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	scaciStatus, err := s.scaciMonitorSvc.GetStatus(ctx, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "get SCACI status failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciStatusFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciStatusFailed))
	}

	resp := &pb.GetScaciStatusResponse{
		Status: &pb.ScaciStatus{
			ServiceOnline:     scaciStatus.ServiceOnline,
			ActiveSessions:    scaciStatus.ActiveSessions,
			PendingOperations: scaciStatus.PendingOperations,
			ProtocolVersion:   scaciStatus.ProtocolVersion,
		},
	}
	if scaciStatus.UptimeSince != nil {
		resp.Status.UptimeSince = timestamppb.New(*scaciStatus.UptimeSince)
	}
	return resp, nil
}

// Helper functions

func scaciSessionToProto(s *grpcservices.ScaciSession) *pb.ScaciSession {
	if s == nil {
		return nil
	}
	pbSession := &pb.ScaciSession{
		Id:              s.ID,
		AcEui:           s.AcEUI,
		Status:          s.Status,
		CanResume:       s.CanResume,
		ProtocolVersion: s.ProtocolVersion,
		ConnectedAt:     timestamppb.New(s.ConnectedAt),
		OperationsCount: s.OperationsCount,
	}
	if s.LastActivityAt != nil {
		pbSession.LastActivityAt = timestamppb.New(*s.LastActivityAt)
	}
	return pbSession
}

func scaciErrorToProto(e *grpcservices.ScaciError) *pb.ScaciError {
	if e == nil {
		return nil
	}
	return &pb.ScaciError{
		Id:            e.ID,
		ErrorCode:     e.ErrorCode,
		ErrorMessage:  e.ErrorMessage,
		SessionId:     e.SessionID,
		OperationType: e.OperationType,
		OccurredAt:    timestamppb.New(e.OccurredAt),
	}
}

func scaciQueueEntryToProto(q *grpcservices.ScaciQueue) *pb.ScaciQueueEntry {
	if q == nil {
		return nil
	}
	pbEntry := &pb.ScaciQueueEntry{
		Id:            q.ID,
		EpEui:         q.EpEUI,
		OperationType: q.OperationType,
		Status:        q.Status,
		Payload:       q.Payload,
		QueuedAt:      timestamppb.New(q.QueuedAt),
	}
	if q.ProcessedAt != nil {
		pbEntry.ProcessedAt = timestamppb.New(*q.ProcessedAt)
	}
	return pbEntry
}
