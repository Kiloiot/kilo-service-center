package grpc

import (
	"strconv"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// exportFilenameFmt names exported message files: messages_<bsEui>.<format>.
const exportFilenameFmt = "messages_%s.%s"

// messageFilterRequest is what a message listing request narrows by.
type messageFilterRequest struct {
	epEUI, bsEUI, direction string
	start, end              *timestamppb.Timestamp
	duplicate, dlOpen       *bool
	profile, mode           string
}

// buildMessageFilters validates the listing predicates; an EUI that is no
// EUI is refused.
func buildMessageFilters(req messageFilterRequest) (*grpcservices.MessageFilters, error) {
	filters := &grpcservices.MessageFilters{
		StartTime: timestampToTime(req.start),
		EndTime:   timestampToTime(req.end),
		Direction: req.direction,
		Duplicate: req.duplicate,
		DlOpen:    req.dlOpen,
		Profile:   req.profile,
		Mode:      req.mode,
	}
	var err error
	if filters.EpEui, err = optionalEUIBytes(req.epEUI, grpcerrors.ErrTokenInvalidEndpointEUIFormat); err != nil {
		return nil, err
	}
	if filters.BsEui, err = optionalEUIBytes(req.bsEUI, grpcerrors.ErrTokenInvalidBasestationEUIFormat); err != nil {
		return nil, err
	}
	return filters, nil
}

// requiredEUIBytes parses an EUI the request must carry: missing is refused
// with requiredToken, malformed with invalidToken.
func requiredEUIBytes(text, requiredToken, invalidToken string) ([]byte, error) {
	if text == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(requiredToken), grpcerrors.ResolveErrorMessage(requiredToken))
	}
	return optionalEUIBytes(text, invalidToken)
}

// ulDataMessageToProto renders a stored uplink as the tenant-wide listing
// shows it: the primary reception's radio values.
func ulDataMessageToProto(msg *mioty.ULDataMessage) *pb.Message {
	if msg == nil {
		return nil
	}
	pbMsg := &pb.Message{
		Id:               msg.ID,
		EpEui:            mioty.FormatEUI64(msg.EpEui),
		BsEui:            mioty.FormatEUI64(msg.BsEui),
		TenantId:         strconv.FormatInt(msg.TenantID, 10),
		Payload:          msg.UserData,
		Rssi:             msg.RSSI,
		Snr:              msg.SNR,
		EqSnr:            valueOrZero(msg.EqSnr),
		PacketCounter:    msg.PacketCnt,
		DlOpen:           msg.DlOpen,
		ResExp:           msg.ResponseExp,
		DlAck:            msg.DlAck,
		ReceivedAt:       timestamppb.New(msg.ReceivedAt),
		OpId:             msg.OpId,
		Format:           userDataFormat(msg),
		UplinkMode:       valueOrZero(msg.Mode),
		DecodedPayload:   msg.DecodedPayload,
		DecodeStatus:     msg.DecodeStatus,
		DecodeErrorCode:  msg.DecodeErrorCode,
		BlueprintTypeEui: msg.BlueprintTypeEUI,
		BaseStations:     baseStationReceptionsToProto(msg.BaseStations),
		Duplicate:        valueOrZero(msg.Duplicate),
	}
	if msg.BlueprintVersionID != nil {
		pbMsg.BlueprintVersionId = msg.BlueprintVersionID.String()
	}
	return pbMsg
}

// ulDataMessageToBaseStationMessageProto renders a stored uplink as one base
// station's listing shows it: the radio values of that station's reception
// (SCACI §3.8.1 baseStations) when it is one of them, else the primary ones.
func ulDataMessageToBaseStationMessageProto(msg *mioty.ULDataMessage, bsEui uint64) *pb.BaseStationMessage {
	if msg == nil {
		return nil
	}
	pbMsg := &pb.BaseStationMessage{
		Id:              msg.ID,
		BsEui:           mioty.FormatEUI64(msg.BsEui),
		EpEui:           mioty.FormatEUI64(msg.EpEui),
		Payload:         msg.UserData,
		Rssi:            msg.RSSI,
		Snr:             msg.SNR,
		EqSnr:           valueOrZero(msg.EqSnr),
		PacketCounter:   msg.PacketCnt,
		ReceivedAt:      timestamppb.New(msg.ReceivedAt),
		Direction:       mioty.DirectionUplink,
		DlOpen:          msg.DlOpen,
		DlAck:           msg.DlAck,
		ResExp:          msg.ResponseExp,
		OpId:            msg.OpId,
		Format:          userDataFormat(msg),
		UplinkMode:      valueOrZero(msg.Mode),
		BaseStations:    baseStationReceptionsToProto(msg.BaseStations),
		Duplicate:       valueOrZero(msg.Duplicate),
		DecodedPayload:  msg.DecodedPayload,
		DecodeStatus:    msg.DecodeStatus,
		DecodeErrorCode: msg.DecodeErrorCode,
	}
	if reception, ok := msg.ReceptionAt(bsEui); ok {
		pbMsg.BsEui = mioty.FormatEUI64(reception.BsEui)
		pbMsg.Rssi = reception.Rssi
		pbMsg.Snr = reception.Snr
		pbMsg.EqSnr = valueOrZero(reception.EqSnr)
	}
	return pbMsg
}

// baseStationReceptionsToProto renders the SCACI §3.8.1 receptions.
func baseStationReceptionsToProto(receptions []mioty.BaseStationReception) []*pb.BaseStationReceptionInfo {
	if len(receptions) == 0 {
		return nil
	}
	result := make([]*pb.BaseStationReceptionInfo, 0, len(receptions))
	for _, bs := range receptions {
		info := &pb.BaseStationReceptionInfo{
			BsEui:      mioty.FormatEUI64(bs.BsEui),
			RxTime:     bs.RxTime,
			Snr:        bs.Snr,
			Rssi:       bs.Rssi,
			EqSnr:      optionalDouble(bs.EqSnr),
			RxDuration: optionalInt64(bs.RxDuration),
			Profile:    optionalString(bs.Profile),
			Mode:       optionalString(bs.Mode),
			DlRxSnr:    optionalDouble(bs.DlRxSnr),
			DlRxRssi:   optionalDouble(bs.DlRxRssi),
		}
		if bs.Subpackets != nil {
			info.Subpackets = &pb.SubpacketInfo{
				Snr:       bs.Subpackets.SNR,
				Rssi:      bs.Subpackets.RSSI,
				Frequency: bs.Subpackets.Frequency,
				Phase:     bs.Subpackets.Phase,
			}
		}
		result = append(result, info)
	}
	return result
}

// valueOrZero is the value an optional field holds, its zero value when unset.
func valueOrZero[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}

// optionalInt64 renders an optional integer; nil stays absent.
func optionalInt64(v *int64) *wrapperspb.Int64Value {
	if v == nil {
		return nil
	}
	return wrapperspb.Int64(*v)
}

// optionalString renders an optional text; nil stays absent.
func optionalString(v *string) *wrapperspb.StringValue {
	if v == nil {
		return nil
	}
	return wrapperspb.String(*v)
}

// userDataFormat is the SCACI §3.8.1 format identifier, unset when the endpoint sent none.
func userDataFormat(msg *mioty.ULDataMessage) *uint32 {
	if msg.Format == nil {
		return nil
	}
	format := uint32(*msg.Format)
	return &format
}

// getContentTypeForFormat names the media type of an export format.
func getContentTypeForFormat(format string) string {
	switch format {
	case grpcerrors.ExportFormatCSV:
		return grpcerrors.ContentTypeCSV
	case grpcerrors.ExportFormatJSON:
		return grpcerrors.ContentTypeJSON
	default:
		return grpcerrors.ContentTypeOctetStream
	}
}
