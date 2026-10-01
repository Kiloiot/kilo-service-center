package grpc

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinks"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// downlinkRejections map the refusals of the downlink service, which the
// caller can correct, to the gRPC catalog.
var downlinkRejections = []struct {
	err   error
	token string
}{
	{downlinks.ErrPayloadRequired, grpcerrors.ErrTokenDownlinkPayloadRequired},
	{downlinks.ErrPacketCountersUnpaired, grpcerrors.ErrTokenDownlinkFormatInvalid},
	{downlinks.ErrTooManyPayloads, grpcerrors.ErrTokenDownlinkPayloadTooLarge},
	{downlinks.ErrPayloadTooLarge, grpcerrors.ErrTokenDownlinkPayloadTooLarge},
	{downlinks.ErrPriorityInvalid, grpcerrors.ErrTokenDownlinkPriorityInvalid},
	{downlinks.ErrFormatInvalid, grpcerrors.ErrTokenDownlinkFormatInvalid},
	{downlinks.ErrPacketCounterInvalid, grpcerrors.ErrTokenDownlinkFormatInvalid},
	{downlinks.ErrEndpointNotFound, grpcerrors.ErrTokenEndpointNotFound},
	{storage.ErrDownlinkNotFound, grpcerrors.ErrTokenDownlinkNotFound},
	{storage.ErrDownlinkNotPending, grpcerrors.ErrTokenDownlinkNotPending},
	{scheduler.ErrSchedulerQueueNotFound, grpcerrors.ErrTokenDownlinkRevokeNotFound},
	{scheduler.ErrSchedulerResourceMissing, grpcerrors.ErrTokenHandshakeIncomplete},
}

// commandFailure is how a downlink command reports a failure no refusal
// names: the log message and the catalog token the client receives.
type commandFailure struct {
	log   string
	token string
}

// downlinkStatus maps a failed downlink command to the status the client
// receives; refusals go unlogged, failures are logged.
func (s *DownlinkHandlers) downlinkStatus(ctx context.Context, err error, failure commandFailure) error {
	for _, rejection := range downlinkRejections {
		if errors.Is(err, rejection.err) {
			return status.Error(grpcerrors.GetGRPCCode(rejection.token), grpcerrors.ResolveErrorMessage(rejection.token))
		}
	}
	if errors.Is(err, downlinks.ErrEndpointLookup) {
		failure = commandFailure{log: LogFailedToGetEndpoint, token: grpcerrors.ErrTokenGetEndpointFailed}
	}
	s.log.ErrorContext(ctx, failure.log, logger.FieldError, err)
	if errors.Is(err, downlinks.ErrQueue) {
		return mapSCACIErrorToGRPC(err)
	}
	return status.Error(grpcerrors.GetGRPCCode(failure.token), grpcerrors.ResolveErrorMessage(failure.token))
}

// scaciQueueTokenToGRPC maps the catalog token carried by a scaci.DLDataQueueError
// onto the gRPC catalog token whose code and message the client receives. A
// persisted downlink is accepted whatever its dispatch outcome, so only
// validation and persistence tokens reach this mapping.
var scaciQueueTokenToGRPC = map[string]string{
	scaci.ErrDLPayloadTooLarge:        grpcerrors.ErrTokenDownlinkPayloadTooLarge,
	scaci.ErrCntDependMismatch:        grpcerrors.ErrTokenScaciCntDependMismatch,
	scaci.ErrCntDependPacketCntOmit:   grpcerrors.ErrTokenScaciCntDependPacketCntOmit,
	scaci.ErrNonCntDependMultiPayload: grpcerrors.ErrTokenScaciNonCntDependMultiPayload,
	scaci.ErrFailedPersistDownlink:    grpcerrors.ErrTokenScaciFailedPersistDownlink,
	scaci.ErrEndpointNotFound:         grpcerrors.ErrTokenScaciEndpointNotFound,
	scaci.ErrEndpointNotBidirectional: grpcerrors.ErrTokenScaciEndpointNotBidirectional,
}

// mapSCACIErrorToGRPC converts a failed internal dlDataQue into the gRPC
// status for its catalog token; anything without a token is an internal failure.
func mapSCACIErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}
	grpcToken := grpcerrors.ErrTokenScaciOperationFailed
	var queueErr *scaci.DLDataQueueError
	if errors.As(err, &queueErr) {
		if mapped, ok := scaciQueueTokenToGRPC[queueErr.Token]; ok {
			grpcToken = mapped
		}
	}
	return status.Error(grpcerrors.GetGRPCCode(grpcToken), grpcerrors.ResolveErrorMessage(grpcToken))
}

// downlinkToProto renders a downlink row with the result its originators see
// and the base station the row names.
func downlinkToProto(msg *storage.DownlinkMessage) *pb.DownlinkMessage {
	protoMsg := &pb.DownlinkMessage{
		Id:                    strconv.FormatInt(msg.ID, 10),
		QueId:                 msg.QueID,
		EpEui:                 msg.EPEUI,
		TenantId:              msg.TenantID,
		Payload:               msg.Payload,
		Payloads:              msg.UserData,
		Priority:              msg.Priority,
		Status:                string(msg.Status),
		CntDepend:             msg.CntDepend,
		PacketCnt:             msg.PacketCntArray,
		Format:                uint32(msg.Format),
		ResponseExp:           msg.ResponseExp,
		ResponsePrio:          msg.ResponsePrio,
		DlWindReq:             msg.DlWindReq,
		ExpOnly:               msg.ExpOnly,
		DlRxStatQry:           msg.DlRxStatQry,
		Result:                msg.Outcome(),
		TxTime:                msg.TxTime,
		TransmissionPacketCnt: msg.TransmissionPacketCnt,
		BsEui:                 baseStationEUI(msg.BsEui),
		ScheduledAt:           optionalTimestamp(msg.ScheduledAt),
		TransmittedAt:         optionalTimestamp(msg.SentAt),
		EndpointAckedAt:       optionalTimestamp(msg.EndpointAckedAt),
		AcceptedAt:            optionalTimestamp(msg.AcceptedAt),
	}
	if !msg.CreatedAt.IsZero() {
		protoMsg.CreatedAt = timestamppb.New(msg.CreatedAt)
	}
	return protoMsg
}

// downlinkQueueEntryToProto renders a queue row under its queue id; the BSSCI
// result name wins over the internal status once a result is known.
func downlinkQueueEntryToProto(msg *storage.DownlinkMessage) *pb.DownlinkMessage {
	protoMsg := downlinkToProto(msg)
	protoMsg.Id = strconv.FormatInt(msg.QueID, 10)
	if protoMsg.Result != "" {
		protoMsg.Status = protoMsg.Result
	}
	return protoMsg
}

// baseStationEUI renders a stored base station EUI; zero, no station, renders
// empty.
func baseStationEUI(eui uint64) string {
	if eui == 0 {
		return ""
	}
	return mioty.FormatEUI64(eui)
}

// requireOrganization reads the organization every downlink write and
// organization-scoped read runs under; the interceptors guarantee it, so a
// missing one is a wiring fault reported as a failed precondition.
func requireOrganization(ctx context.Context) (uuid.UUID, error) {
	orgID, err := pkgcontext.RequireOrganizationID(ctx)
	if err != nil {
		return uuid.Nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingOrgContext),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMissingOrgContext))
	}
	return orgID, nil
}
