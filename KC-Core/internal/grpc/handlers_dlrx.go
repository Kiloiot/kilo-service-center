package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// DLRXHandlers serves the DL RX status RPCs.
type DLRXHandlers struct {
	queries  DLRXStatusQueryStorage
	statuses DLRXStatusReader
	stations ServingStationLocator
	command  bssci.DownlinkCommander
	sessions EndpointSessionFinder
	log      logger.Logger
}

// DLRXHandlerDeps wires DLRXHandlers; every entry is required.
type DLRXHandlerDeps struct {
	Queries   DLRXStatusQueryStorage
	Statuses  DLRXStatusReader
	Stations  ServingStationLocator
	Commander bssci.DownlinkCommander
	Sessions  EndpointSessionFinder
}

// NewDLRXHandlers validates the group and builds it.
func NewDLRXHandlers(d DLRXHandlerDeps, log logger.Logger) (*DLRXHandlers, error) {
	switch {
	case d.Queries == nil || d.Statuses == nil:
		return nil, errors.New(errMsgDlrxStorageCannotBeNil)
	case d.Stations == nil:
		return nil, errors.New(errMsgServingStationsCannotBeNil)
	case d.Commander == nil:
		return nil, errors.New(errMsgDownlinkCmdCannotBeNil)
	case d.Sessions == nil:
		return nil, errors.New(errMsgSessionDirCannotBeNil)
	}
	return &DLRXHandlers{
		queries:  d.Queries,
		statuses: d.Statuses,
		stations: d.Stations,
		command:  d.Commander,
		sessions: d.Sessions,
		log:      log,
	}, nil
}

// GetDLRXStatus retrieves DL RX status reports for an endpoint (BSSCI §3.15)
func (s *DLRXHandlers) GetDLRXStatus(ctx context.Context, req *pb.GetDLRXStatusRequest) (*pb.GetDLRXStatusResponse, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	epEUI, err := requiredEUIBytes(req.EpEui, grpcerrors.ErrTokenEndpointEUIRequired, grpcerrors.ErrTokenInvalidEndpointEUIFormat)
	if err != nil {
		return nil, err
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = DefaultDLRXStatusLimit
	}
	offset := int(req.Offset)
	startTime, endTime := validTimestamp(req.StartTime), validTimestamp(req.EndTime)
	s.log.InfoContext(ctx, LogGetDLRXStatusCalled, logger.FieldTenantIDCamel, tenantID, logger.FieldEpEui, req.EpEui,
		logger.FieldLimit, limit, logger.FieldOffset, offset)

	statuses, totalCount, err := s.statuses.GetDLRXStatusByEndpoint(ctx, tenantID, epEUI, limit, offset, startTime, endTime)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDLRxStatusFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDLRxStatusFailed))
	}
	avgSnr, avgRssi, _, err := s.statuses.GetAverageDLRXMetrics(ctx, tenantID, epEUI, startTime, endTime)
	if err != nil {
		s.log.WarnContext(ctx, LogFailedToGetAverageDLRXMetrics, logger.FieldError, err,
			logger.FieldEpEui, mioty.FormatEUIBytes(epEUI), logger.FieldTenantIDCamel, tenantID)
	}
	total, err := totalCountOf(int64(totalCount))
	if err != nil {
		return nil, err
	}
	return &pb.GetDLRXStatusResponse{
		Statuses:   s.plausibleStatuses(ctx, statuses),
		TotalCount: total,
		AvgSnr:     avgSnr,
		AvgRssi:    avgRssi,
	}, nil
}

// plausibleStatuses renders the reports whose dlRxSnr and dlRxRssi lie in
// the physically possible range (BSSCI §5.15); any other report is corrupt
// and skipped with a warning.
func (s *DLRXHandlers) plausibleStatuses(ctx context.Context, statuses []*mioty.DLRXStatus) []*pb.DLRXStatus {
	result := make([]*pb.DLRXStatus, 0, len(statuses))
	for _, st := range statuses {
		if st.DlRxSnr < mioty.DLRxSnrMinDB || st.DlRxSnr > mioty.DLRxSnrMaxDB {
			s.log.WarnContext(ctx, LogDlRxSnrValueOutOfPhysicsRangeSkippingStatus, logger.FieldEpEui, mioty.FormatEUIBytes(st.EpEui),
				logger.FieldDlRxSnr, st.DlRxSnr, logger.FieldValidRange, fmt.Sprintf(validRangeFmtDB, mioty.DLRxSnrMinDB, mioty.DLRxSnrMaxDB))
			continue
		}
		if st.DlRxRssi < mioty.DLRxRssiMinDBm || st.DlRxRssi > mioty.DLRxRssiMaxDBm {
			s.log.WarnContext(ctx, LogDlRxRssiValueOutOfPhysicsRangeSkippingStatus, logger.FieldEpEui, mioty.FormatEUIBytes(st.EpEui),
				logger.FieldDlRxRssi, st.DlRxRssi, logger.FieldValidRange, fmt.Sprintf(validRangeFmtDBm, mioty.DLRxRssiMinDBm, mioty.DLRxRssiMaxDBm))
			continue
		}
		result = append(result, &pb.DLRXStatus{
			EpEui:     mioty.FormatEUIBytes(st.EpEui),
			BsEui:     mioty.FormatEUIBytes(st.BsEui),
			RxTime:    st.RxTime,
			PacketCnt: st.PacketCnt,
			DlRxSnr:   st.DlRxSnr,
			DlRxRssi:  st.DlRxRssi,
			CreatedAt: timestamppb.New(st.CreatedAt),
		})
	}
	return result
}

// validTimestamp is the time an optional timestamp holds; an unset or
// invalid one does not bound the window.
func validTimestamp(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil || !ts.IsValid() {
		return nil
	}
	return timestampToTime(ts)
}
