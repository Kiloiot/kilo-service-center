// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	scacimonitoring "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci_monitoring"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// ScaciHandlers serves the SCACI monitoring RPCs.
type ScaciHandlers struct {
	scaciMonitorSvc grpcservices.ScaciMonitoringService
	log             logger.Logger
}

// ScaciHandlerDeps wires ScaciHandlers; a nil service answers its RPCs as
// not configured.
type ScaciHandlerDeps struct {
	Monitoring grpcservices.ScaciMonitoringService
}

// NewScaciHandlers builds the group.
func NewScaciHandlers(d ScaciHandlerDeps, log logger.Logger) *ScaciHandlers {
	return &ScaciHandlers{scaciMonitorSvc: d.Monitoring, log: log}
}

// SCACI Monitoring handlers

// ListScaciSessions returns SCACI sessions.
func (s *ScaciHandlers) ListScaciSessions(ctx context.Context, req *pb.ListScaciSessionsRequest) (*pb.ListScaciSessionsResponse, error) {
	if s.scaciMonitorSvc == nil {
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

	filter := grpcservices.ScaciSessionFilter{CanResume: scaciResumeFilter(req)}
	if req.Status != "" {
		filter.Status = &req.Status
	}
	sessions, total, err := s.scaciMonitorSvc.ListSessions(ctx, tenantID, filter, page.limit, page.offset)
	if errors.Is(err, scacimonitoring.ErrInvalidSessionStatus) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciInvalidSessionStatus),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciInvalidSessionStatus))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogListSCACISessionsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciListSessionsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciListSessionsFailed))
	}

	var pbSessions []*pb.ScaciSession
	for _, session := range sessions {
		pbSessions = append(pbSessions, scaciSessionToProto(session))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListScaciSessionsResponse{
		Sessions:      pbSessions,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

// GetScaciSession returns a specific SCACI session.
func (s *ScaciHandlers) GetScaciSession(ctx context.Context, req *pb.GetScaciSessionRequest) (*pb.GetScaciSessionResponse, error) {
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
	switch {
	case errors.Is(err, scacimonitoring.ErrSessionNotFound):
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciSessionNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciSessionNotFound))
	case errors.Is(err, scacimonitoring.ErrInvalidSessionID):
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	case err != nil:
		s.log.ErrorContext(ctx, LogGetSCACISessionFailed, logger.FieldSessionIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	return &pb.GetScaciSessionResponse{Session: scaciSessionToProto(session)}, nil
}

// GetScaciStatistics returns SCACI operation statistics.
func (s *ScaciHandlers) GetScaciStatistics(ctx context.Context, req *pb.GetScaciStatisticsRequest) (*pb.GetScaciStatisticsResponse, error) {
	if s.scaciMonitorSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := s.scaciMonitorSvc.GetStatistics(ctx, tenantID, scaciWindow(req.StartTime, req.EndTime))
	if errors.Is(err, scacimonitoring.ErrInvalidTimeRange) {
		return nil, invalidTimeRangeError()
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogGetSCACIStatisticsFailed, logger.FieldError, err)
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
func (s *ScaciHandlers) ListScaciErrors(ctx context.Context, req *pb.ListScaciErrorsRequest) (*pb.ListScaciErrorsResponse, error) {
	if s.scaciMonitorSvc == nil {
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

	groups, total, err := s.scaciMonitorSvc.ListErrors(ctx, tenantID, scaciWindow(req.StartTime, req.EndTime), page.limit, page.offset)
	if errors.Is(err, scacimonitoring.ErrInvalidTimeRange) {
		return nil, invalidTimeRangeError()
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogListSCACIErrorsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciListErrorsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciListErrorsFailed))
	}

	var pbErrors []*pb.ScaciError
	for _, e := range groups {
		pbErrors = append(pbErrors, scaciErrorToProto(e))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListScaciErrorsResponse{
		Errors:        pbErrors,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

// ListScaciQueues returns SCACI queue status.
func (s *ScaciHandlers) ListScaciQueues(ctx context.Context, req *pb.ListScaciQueuesRequest) (*pb.ListScaciQueuesResponse, error) {
	if s.scaciMonitorSvc == nil {
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

	orgID, err := requireOrganization(ctx)
	if err != nil {
		return nil, err
	}
	epFilter, err := scaciQueueEndpointFilter(req.EpEui)
	if err != nil {
		return nil, err
	}
	queues, total, err := s.scaciMonitorSvc.ListQueues(ctx, tenantID, orgID, epFilter, page.limit, page.offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListSCACIQueuesFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciListQueuesFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciListQueuesFailed))
	}

	var pbQueues []*pb.ScaciQueueEntry
	for _, q := range queues {
		pbQueues = append(pbQueues, scaciQueueEntryToProto(q))
	}

	placed, err := page.respond(int64(total))
	if err != nil {
		return nil, err
	}
	return &pb.ListScaciQueuesResponse{
		QueueEntries:  pbQueues,
		NextPageToken: placed.nextToken,
		TotalCount:    placed.totalCount,
	}, nil
}

// GetScaciStatus returns overall SCACI status.
func (s *ScaciHandlers) GetScaciStatus(ctx context.Context, req *pb.GetScaciStatusRequest) (*pb.GetScaciStatusResponse, error) {
	if s.scaciMonitorSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	orgID, err := requireOrganization(ctx)
	if err != nil {
		return nil, err
	}
	scaciStatus, err := s.scaciMonitorSvc.GetStatus(ctx, tenantID, orgID, scaciWindow(req.StartTime, req.EndTime))
	if errors.Is(err, scacimonitoring.ErrInvalidTimeRange) {
		return nil, invalidTimeRangeError()
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogGetSCACIStatusFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciStatusFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciStatusFailed))
	}

	resp := &pb.GetScaciStatusResponse{
		Status: &pb.ScaciStatus{
			ServiceOnline:     scaciStatus.ServiceOnline,
			ActiveSessions:    scaciStatus.ActiveSessions,
			PendingOperations: scaciStatus.PendingOperations,
			ProtocolVersion:   scaciStatus.ProtocolVersion,
			ScEui:             scaciStatus.SCEui,
			MissedPings:       scaciStatus.MissedPings,
			ReconnectAttempts: scaciStatus.ReconnectAttempts,
			LastConnectResult: scaciStatus.LastConnectResult,
		},
	}
	if scaciStatus.UptimeSince != nil {
		resp.Status.UptimeSince = timestamppb.New(*scaciStatus.UptimeSince)
	}
	if scaciStatus.LastPingAt != nil {
		resp.Status.LastPingAt = timestamppb.New(*scaciStatus.LastPingAt)
	}
	if scaciStatus.LastPingRTT != nil {
		resp.Status.LastPingRttMs = scaciStatus.LastPingRTT.Milliseconds()
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
		SnAcUuid:        s.SnAcUUID,
		SnScUuid:        s.SnScUUID,
		LastOpIdAc:      s.LastOpIDAc,
		LastOpIdSc:      s.LastOpIDSc,
	}
	if s.LastActivityAt != nil {
		pbSession.LastActivityAt = timestamppb.New(*s.LastActivityAt)
	}
	if s.DisconnectedAt != nil {
		pbSession.DisconnectedAt = timestamppb.New(*s.DisconnectedAt)
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
		ErrorToken:    e.ErrorToken,
		ErrorMessage:  e.ErrorMessage,
		SessionId:     e.SessionID,
		OperationType: e.OperationType,
		OccurredAt:    timestamppb.New(e.OccurredAt),
		FirstSeen:     timestamppb.New(e.FirstSeen),
		LastSeen:      timestamppb.New(e.LastSeen),
		Count:         e.Count,
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
		QueId:         q.QueID,
		Priority:      q.Priority,
	}
	if q.ProcessedAt != nil {
		pbEntry.ProcessedAt = timestamppb.New(*q.ProcessedAt)
	}
	return pbEntry
}

func scaciWindow(start, end *timestamppb.Timestamp) grpcservices.ScaciWindow {
	return grpcservices.ScaciWindow{From: timestampToTime(start), To: timestampToTime(end)}
}

func invalidTimeRangeError() error {
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTimeRange),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTimeRange))
}

// scaciQueueEndpointFilter turns an optional hex endpoint EUI into the
// big-endian filter the queue reader expects.
func scaciQueueEndpointFilter(epEuiHex string) (*[8]byte, error) {
	if epEuiHex == "" {
		return nil, nil
	}
	eui, err := validation.ParseEUI(epEuiHex)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}
	var filter [8]byte
	binary.BigEndian.PutUint64(filter[:], eui)
	return &filter, nil
}

// scaciResumeFilter prefers the explicit-presence filter; the legacy implicit
// field can only ever ask for resumable sessions.
func scaciResumeFilter(req *pb.ListScaciSessionsRequest) *bool {
	if req.CanResumeFilter != nil {
		return req.CanResumeFilter
	}
	if req.GetCanResume() { //nolint:staticcheck // deprecated field read for wire compatibility with legacy clients
		resumable := true
		return &resumable
	}
	return nil
}
