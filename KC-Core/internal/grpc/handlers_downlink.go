package grpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinks"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
)

// DownlinkHandlers serves the downlink RPCs: queue, edit and revoke, and the
// queue and results listings.
type DownlinkHandlers struct {
	commands DownlinkCommands
	queue    DownlinkQueueLister
	results  DownlinkResultsReader
	log      logger.Logger
}

// DownlinkHandlerDeps wires DownlinkHandlers; every entry is required.
type DownlinkHandlerDeps struct {
	Commands DownlinkCommands
	Queue    DownlinkQueueLister
	Results  DownlinkResultsReader
}

// NewDownlinkHandlers validates the group and builds it.
func NewDownlinkHandlers(d DownlinkHandlerDeps, log logger.Logger) (*DownlinkHandlers, error) {
	if d.Commands == nil || d.Queue == nil || d.Results == nil {
		return nil, errors.New(errMsgDownlinkDepsCannotBeNil)
	}
	return &DownlinkHandlers{commands: d.Commands, queue: d.Queue, results: d.Results, log: log}, nil
}

// SendDownlink queues a downlink for an endpoint through the SCACI handler
// core, the path a socket dlDataQue takes (SCACI §3.10).
func (s *DownlinkHandlers) SendDownlink(ctx context.Context, req *pb.SendDownlinkRequest) (*pb.SendDownlinkResponse, error) {
	owner, err := downlinkOwner(ctx, req.TenantId, req.EpEui, grpcerrors.ErrTokenInvalidEUIFormat)
	if err != nil {
		return nil, err
	}
	s.log.InfoContext(ctx, LogSendingDownlink, logger.FieldEpEuiSnake, req.EpEui, logger.FieldTenantIDSnake, owner.TenantID, logger.FieldPayloadCount, len(req.Payloads))
	result, err := s.commands.Queue(ctx, owner, downlinkContent(req))
	if err != nil {
		return nil, s.downlinkStatus(ctx, err, commandFailure{log: LogSCACIQueueDownlinkInternalFailed, token: grpcerrors.ErrTokenScaciOperationFailed})
	}
	return &pb.SendDownlinkResponse{Id: strconv.FormatUint(result.QueID, 10), Status: string(result.Status)}, nil
}

// UpdatePendingDownlink rewrites the SCACI §3.10.1 fields of a downlink that
// no base station has taken yet.
func (s *DownlinkHandlers) UpdatePendingDownlink(ctx context.Context, req *pb.UpdatePendingDownlinkRequest) (*pb.DownlinkMessage, error) {
	owner, err := downlinkOwner(ctx, "", req.EpEui, grpcerrors.ErrTokenInvalidEUIFormat)
	if err != nil {
		return nil, err
	}
	if req.QueId <= 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenQueueIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenQueueIDRequired))
	}
	updated, err := s.commands.Update(ctx, downlinks.Target{Owner: owner, QueID: req.QueId}, downlinkContent(req))
	if err != nil {
		return nil, s.downlinkStatus(ctx, err, commandFailure{log: LogUpdatePendingDownlinkFailed, token: grpcerrors.ErrTokenDownlinkUpdateFailed})
	}
	return downlinkQueueEntryToProto(updated), nil
}

// RevokeDownlink revokes a queued downlink. The queue id must name a
// downlink of the requested endpoint in the caller's organization; any other
// id reads as not found, so a foreign id is indistinguishable from a missing
// one.
func (s *DownlinkHandlers) RevokeDownlink(ctx context.Context, req *pb.RevokeDownlinkRequest) (*pb.RevokeDownlinkResponse, error) {
	owner, err := downlinkOwner(ctx, req.TenantId, req.EpEui, grpcerrors.ErrTokenInvalidEndpointEUIFormat)
	if err != nil {
		return nil, err
	}
	s.log.InfoContext(ctx, LogRevokingDownlink, logger.FieldEpEuiSnake, req.EpEui, logger.FieldQueueIDSnake, req.QueueId, logger.FieldTenantIDSnake, owner.TenantID)
	queID, err := parseRevokeQueueID(req.QueueId)
	if err != nil {
		return nil, err
	}
	result, err := s.commands.Revoke(ctx, downlinks.Target{Owner: owner, QueID: queID})
	if err != nil {
		return nil, s.downlinkStatus(ctx, err, commandFailure{log: LogFailedToRevokeDownlinkViaBSSCI, token: grpcerrors.ErrTokenDownlinkRevokeFailed})
	}
	message := grpcerrors.MsgRevoked
	if result.Status == downlinks.RevokeStatusInitiated {
		message = grpcerrors.MsgRevokeInitiated
	}
	return &pb.RevokeDownlinkResponse{Status: result.Status, Message: fmt.Sprintf(message, req.QueueId)}, nil
}

// downlinkOwner reads who a downlink request acts for: the authenticated
// tenant, which a tenant the request states must match, its organization and
// the endpoint, refused with invalidEUIToken when it is no EUI.
func downlinkOwner(ctx context.Context, statedTenant, epEUI, invalidEUIToken string) (downlinks.Owner, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return downlinks.Owner{}, err
	}
	if statedTenant != "" && statedTenant != strconv.FormatInt(tenantID, 10) {
		return downlinks.Owner{}, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}
	if epEUI == "" {
		return downlinks.Owner{}, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}
	eui, err := validation.ParseEUI(epEUI)
	if err != nil {
		return downlinks.Owner{}, status.Error(grpcerrors.GetGRPCCode(invalidEUIToken), grpcerrors.ResolveErrorMessage(invalidEUIToken))
	}
	orgID, err := requireOrganization(ctx)
	if err != nil {
		return downlinks.Owner{}, err
	}
	return downlinks.Owner{TenantID: tenantID, OrganizationID: orgID, EpEUI: eui}, nil
}

// downlinkContentFields are the SCACI §3.10.1 fields a queue and an edit
// request both carry.
type downlinkContentFields interface {
	GetPayloads() [][]byte
	GetPriority() float32
	GetCntDepend() bool
	GetPacketCnt() []int64
	GetFormat() uint32
	GetResponseExp() bool
	GetResponsePrio() bool
	GetDlWindReq() bool
	GetExpOnly() bool
	GetDlRxStatQry() bool
}

// downlinkContent reads the downlink content a request carries.
func downlinkContent(req downlinkContentFields) downlinks.Content {
	return downlinks.Content{
		Payloads: req.GetPayloads(), Priority: req.GetPriority(), CntDepend: req.GetCntDepend(), PacketCnt: req.GetPacketCnt(),
		Format: req.GetFormat(), ResponseExp: req.GetResponseExp(), ResponsePrio: req.GetResponsePrio(),
		DlWindReq: req.GetDlWindReq(), ExpOnly: req.GetExpOnly(), DlRxStatQry: req.GetDlRxStatQry(),
	}
}

// parseRevokeQueueID reads the queue id a revoke names, a non-negative number.
func parseRevokeQueueID(text string) (int64, error) {
	token := ""
	queID, err := strconv.ParseInt(text, 10, 64)
	switch {
	case text == "":
		token = grpcerrors.ErrTokenQueueIDRequired
	case err != nil:
		token = grpcerrors.ErrTokenInvalidQueueIDFormat
	case queID < 0:
		token = grpcerrors.ErrTokenQueueIDNonNegative
	default:
		return queID, nil
	}
	return 0, status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
}
