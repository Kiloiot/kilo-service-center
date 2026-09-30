package grpc

import (
	"context"
	"errors"
	"math"
	"strconv"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ULTransmitHandlers serves SendULTransmit, the uplink transmission a
// service center asks a base station for (BSSCI §3.11).
type ULTransmitHandlers struct {
	sessions     BidirectionalSessionSelector
	transmitter  bssci.ULTransmitter
	baseStations BaseStationLookup
	log          logger.Logger
}

// ULTransmitHandlerDeps wires ULTransmitHandlers; every entry is required.
type ULTransmitHandlerDeps struct {
	Sessions     BidirectionalSessionSelector
	Transmitter  bssci.ULTransmitter
	BaseStations BaseStationLookup
}

// NewULTransmitHandlers validates the group and builds it.
func NewULTransmitHandlers(d ULTransmitHandlerDeps, log logger.Logger) (*ULTransmitHandlers, error) {
	switch {
	case d.Sessions == nil:
		return nil, errors.New(errMsgSessionDirCannotBeNil)
	case d.Transmitter == nil:
		return nil, errors.New(errMsgUlTransmitCannotBeNil)
	case d.BaseStations == nil:
		return nil, errors.New(errMsgBasestationSvcCannotBeNil)
	}
	return &ULTransmitHandlers{sessions: d.Sessions, transmitter: d.Transmitter, baseStations: d.BaseStations, log: log}, nil
}

// ulTransmitTarget is the endpoint a SendULTransmit request names and the
// base station it asks for, nil when the service center chooses.
type ulTransmitTarget struct {
	epEUI uint64
	bsEUI *uint64
}

// SendULTransmit initiates a Service Center to Base Station uplink data transmission
func (s *ULTransmitHandlers) SendULTransmit(ctx context.Context, req *pb.SendULTransmitRequest) (*pb.SendULTransmitResponse, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	s.log.InfoContext(ctx, LogSendULTransmitRequestReceived, logger.FieldEpEuiSnake, req.EpEui, logger.FieldTenantIDSnake, tenantID,
		logger.FieldPacketCntSnake, req.PacketCnt, logger.FieldUserDataLenSnake, len(req.UserData))
	target, err := parseULTransmitTarget(tenantID, req)
	if err != nil {
		return nil, err
	}
	if err := s.requireBaseStation(ctx, tenantID, target.bsEUI); err != nil {
		return nil, err
	}
	if err := checkULTransmitIdentity(req); err != nil {
		return nil, err
	}
	sessionID, bsEui, err := s.sessions.SelectBidirectionalSession(tenantID, target.bsEUI)
	if err != nil {
		return nil, sessionSelectionStatus(err)
	}
	shAddr, format, err := ulTransmitWidths(req)
	if err != nil {
		return nil, err
	}
	opID, err := s.transmitter.SendULDataTransmit(sessionID, target.epEUI, req.NwkSnKey, shAddr, req.PacketCnt, req.UserData, req.Profile, format)
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToSendULTransmit, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenULTransmitFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenULTransmitFailed))
	}
	s.log.InfoContext(ctx, LogULTransmitQueuedSuccessfully, logger.FieldTenantIDSnake, tenantID, logger.FieldEpEuiSnake, req.EpEui,
		logger.FieldBsEuiSnake, mioty.FormatEUI64(bsEui), logger.FieldPacketCntSnake, req.PacketCnt, logger.FieldOperationIDSnake, opID)
	return &pb.SendULTransmitResponse{Id: strconv.FormatInt(opID, 10), Status: grpcerrors.StatusQueued, Message: grpcerrors.MsgULTransmitQueued}, nil
}

// parseULTransmitTarget reads the endpoint and the base station a request
// names; a tenant the request states must be the caller's.
func parseULTransmitTarget(tenantID int64, req *pb.SendULTransmitRequest) (ulTransmitTarget, error) {
	var target ulTransmitTarget
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return target, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}
	epEUI, err := requiredEUIBytes(req.EpEui, grpcerrors.ErrTokenEndpointEUIRequired, grpcerrors.ErrTokenInvalidEndpointEUIFormat)
	if err != nil {
		return target, err
	}
	target.epEUI = mioty.EUI64FromBytes(epEUI)
	bsEUI, err := optionalEUIBytes(req.BsEui, grpcerrors.ErrTokenInvalidBasestationEUIFormat)
	if err != nil || bsEUI == nil {
		return target, err
	}
	station := mioty.EUI64FromBytes(bsEUI)
	target.bsEUI = &station
	return target, nil
}

// checkULTransmitIdentity refuses a network session key that is not one and
// the zero short address. Empty user data is allowed: a control telegram
// carries none (BSSCI §3.11.1).
func checkULTransmitIdentity(req *pb.SendULTransmitRequest) error {
	token := ""
	switch {
	case len(req.NwkSnKey) != endpointKeyLen:
		token = grpcerrors.ErrTokenNwkSnKeyLength
	case req.ShAddr == 0:
		token = grpcerrors.ErrTokenShortAddressZero
	default:
		return nil
	}
	return status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
}

// ulTransmitWidths narrows the short address to 16 bits and the format to 8.
func ulTransmitWidths(req *pb.SendULTransmitRequest) (uint16, uint8, error) {
	if req.ShAddr > math.MaxUint16 {
		return 0, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenShortAddressOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenShortAddressOverflow))
	}
	if req.Format > math.MaxUint8 {
		return 0, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFormatOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFormatOverflow))
	}
	return uint16(req.ShAddr), uint8(req.Format), nil
}

// requireBaseStation refuses a requested base station the tenant does not
// own before any session is chosen, so an offline station of another tenant
// discloses nothing.
func (s *ULTransmitHandlers) requireBaseStation(ctx context.Context, tenantID int64, bsEUI *uint64) error {
	if bsEUI == nil {
		return nil
	}
	_, err := s.baseStations.GetByEUI(ctx, mioty.EUI64Bytes(*bsEUI), tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToVerifyBaseStationOwnership, logger.FieldError, err)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationOwnershipFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationOwnershipFailed))
	}
	return nil
}

// sessionSelectionStatus maps a failed bidirectional session selection to
// the status the client receives.
func sessionSelectionStatus(err error) error {
	token := grpcerrors.ErrTokenHandshakeIncomplete
	switch {
	case errors.Is(err, bssci.ErrNoBidirectionalBaseStations):
		token = grpcerrors.ErrTokenNoBaseStationsConnected
	case errors.Is(err, bssci.ErrBaseStationUnavailable):
		token = grpcerrors.ErrTokenBaseStationNotFound
	case errors.Is(err, bssci.ErrBaseStationTenantMismatch):
		token = grpcerrors.ErrTokenTenantAccessDenied
	}
	return status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
}
