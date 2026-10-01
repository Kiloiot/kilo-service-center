package grpc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// euiHexLength is the length of an EUI written as hex digits.
const euiHexLength = 16

// sessionLookupMessages answer a DL RX status query whose serving base
// station has no usable session.
var sessionLookupMessages = []struct {
	err error
	fmt string
}{
	{bssci.ErrSessionNotFound, grpcerrors.MsgFmtBSNotConnected},
	{bssci.ErrSessionNotReady, grpcerrors.MsgFmtBSHandshakeIncomplete},
	{bssci.ErrSessionNotBidirectional, grpcerrors.MsgFmtBSNotBidirectional},
}

// QueryDLRXStatus initiates a DL RX status query to an endpoint (BSSCI §3.16)
func (s *DLRXHandlers) QueryDLRXStatus(ctx context.Context, req *pb.QueryDLRXStatusRequest) (*pb.QueryDLRXStatusResponse, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	epEUI, err := parseEndpointHex(req.EpEui)
	if err != nil {
		return nil, err
	}
	// The station serving the tenant's endpoint answers the query, the same
	// station a downlink queued ahead of time goes to. Another tenant's
	// endpoint reads like one no station heard or attached yet.
	station, known, err := s.stations.ServingStation(ctx, tenantID, epEUI)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointSessionLookupFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointSessionLookupFailed))
	}
	if !known {
		return &pb.QueryDLRXStatusResponse{Message: grpcerrors.MsgEndpointNotAttached}, nil
	}
	bsEui := mioty.FormatEUI64(station)
	s.log.InfoContext(ctx, LogQueryDLRXStatusLookingForSession, logger.FieldTenantIDCamel, tenantID, logger.FieldEpEui, req.EpEui, logger.FieldBsEui, bsEui)
	// The lookup was scoped to the tenant's endpoint, so the station may belong to any tenant (roaming).
	sessionID, err := s.sessions.FindSessionForEndpointAttachment(station)
	if err != nil {
		s.log.WarnContext(ctx, LogFailedToFindBaseStationSessionForDLRXStatusQuery, logger.FieldTenantIDCamel, tenantID,
			logger.FieldBsEui, bsEui, logger.FieldEpEui, req.EpEui, logger.FieldError, err)
		return &pb.QueryDLRXStatusResponse{Message: sessionLookupMessage(err, bsEui)}, nil
	}
	s.log.InfoContext(ctx, LogQueryDLRXStatusFoundServingBaseStationSession, logger.FieldTenantIDCamel, tenantID,
		logger.FieldSessionIDCamel, sessionID, logger.FieldBsEui, bsEui)
	if err := s.command.SendDLRXStatusQuery(sessionID, epEUI); err != nil {
		return &pb.QueryDLRXStatusResponse{Message: fmt.Sprintf(grpcerrors.MsgFmtDLRXQuerySendFailed, err)}, nil
	}
	return &pb.QueryDLRXStatusResponse{QueryInitiated: true, Message: grpcerrors.MsgDLRXQuerySent}, nil
}

// sessionLookupMessage tells why the serving station's session cannot take
// the query.
func sessionLookupMessage(err error, bsEui string) string {
	for _, known := range sessionLookupMessages {
		if errors.Is(err, known.err) {
			return fmt.Sprintf(known.fmt, bsEui)
		}
	}
	return grpcerrors.MsgNoSuitableBSSession
}

// GetDLRXStatusQueries retrieves query tracking history for an endpoint (BSSCI §5.15 telemetry)
func (s *DLRXHandlers) GetDLRXStatusQueries(ctx context.Context, req *pb.GetDLRXStatusQueriesRequest) (*pb.GetDLRXStatusQueriesResponse, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	epEUI, err := parseEndpointHex(req.EpEui)
	if err != nil {
		return nil, err
	}
	limit, offset, err := dlrxQueriesPage(req)
	if err != nil {
		return nil, err
	}
	epEuiBytes := mioty.EUI64Bytes(epEUI)
	startTime, endTime := validTimestamp(req.StartTime), validTimestamp(req.EndTime)
	s.log.InfoContext(ctx, LogGetDLRXStatusQueriesCalled, logger.FieldTenantIDCamel, tenantID, logger.FieldEpEui, req.EpEui,
		logger.FieldLimit, limit, logger.FieldOffset, offset)
	queries, totalCount, err := s.queries.GetDLRXStatusQueryHistory(ctx, tenantID, epEuiBytes, limit, offset, startTime, endTime)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDLRxStatusQueryHistoryFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDLRxStatusQueryHistoryFailed))
	}
	pending, received, timeout, err := s.queries.GetDLRXStatusQueryStats(ctx, tenantID, epEuiBytes, startTime, endTime)
	if err != nil {
		s.log.WarnContext(ctx, LogFailedToGetDLRXStatusQueryStats, logger.FieldError, err,
			logger.FieldEpEui, mioty.FormatEUIBytes(epEuiBytes), logger.FieldTenantIDCamel, tenantID)
	}
	return &pb.GetDLRXStatusQueriesResponse{
		Queries:    dlrxQueriesToProto(queries),
		TotalCount: int64(totalCount),
		Stats:      &pb.DLRXStatusQueryStats{Pending: pending, Received: received, Timeout: timeout},
	}, nil
}

// dlrxQueriesPage validates the page and window of a query history request
// before it applies the default and the cap to the limit.
func dlrxQueriesPage(req *pb.GetDLRXStatusQueriesRequest) (limit, offset int, err error) {
	start, end := validTimestamp(req.StartTime), validTimestamp(req.EndTime)
	token := ""
	switch {
	case req.Limit < 0:
		token = grpcerrors.ErrTokenLimitNonNegative
	case req.Offset < 0:
		token = grpcerrors.ErrTokenOffsetNonNegative
	case start != nil && end != nil && start.After(*end):
		token = grpcerrors.ErrTokenInvalidTimeRange
	}
	if token != "" {
		return 0, 0, status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
	}
	limit = int(req.Limit)
	if limit == 0 {
		limit = DefaultPageSize
	}
	return min(limit, MaxPageSize), int(req.Offset), nil
}

// dlrxQueriesToProto renders the query history; the receive time and the
// organization are set only when known.
func dlrxQueriesToProto(queries []*mioty.DLRXStatusQuery) []*pb.DLRXStatusQuery {
	result := make([]*pb.DLRXStatusQuery, len(queries))
	for i, q := range queries {
		result[i] = &pb.DLRXStatusQuery{
			EpEui:       mioty.FormatEUIBytes(q.EpEui),
			BsEui:       mioty.FormatEUIBytes(q.BsEui),
			OpId:        q.OpId,
			Status:      q.Status,
			RequestedAt: timestamppb.New(q.RequestedAt),
			ReceivedAt:  optionalTimestamp(q.ReceivedAt),
		}
		if q.OrganizationID != nil {
			result[i].OrgUuid = q.OrganizationID.String()
		}
	}
	return result
}

// parseEndpointHex reads an endpoint EUI written as exactly 16 hex digits.
func parseEndpointHex(text string) (uint64, error) {
	token := ""
	switch {
	case text == "":
		token = grpcerrors.ErrTokenEndpointEUIRequired
	case len(text) != euiHexLength:
		token = grpcerrors.ErrTokenEUIHexLengthInvalid
	default:
		if _, err := hex.DecodeString(text); err != nil {
			token = grpcerrors.ErrTokenEUIHexCharsInvalid
		}
	}
	if token != "" {
		return 0, status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
	}
	eui, err := validation.ParseEUI(text)
	if err != nil {
		return 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}
	return eui, nil
}
