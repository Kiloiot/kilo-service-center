package grpc

import (
	"errors"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// CoreService implements the CoreService gRPC service for device management,
// protocol operations, analytics, monitoring, certificates, blueprints, and integrations.

const componentGRPCService = "grpc-service"

// reasonEUIChanged annotates the offline event recorded when a base
// station's EUI is rewritten.
const reasonEUIChanged = "EUI changed"

// defaultDownlinkQueuePageSize bounds downlink queue listings when the
// request does not specify a page size.
const defaultDownlinkQueuePageSize = 50

// Inclusive SNR and RSSI physics ranges rendered for warnings.
const (
	validRangeFmtDB  = "[%.1f, %.1f] dB"
	validRangeFmtDBm = "[%.1f, %.1f] dBm"
)

// unimplementedCore sits one embedding level below the handler groups so
// their methods win promotion and every RPC no group serves still answers
// Unimplemented.
type unimplementedCore struct {
	pb.UnimplementedCoreServiceServer
}

// CoreService is the CoreService gRPC surface: a facade over the handler
// groups, each serving one slice of the API.
type CoreService struct {
	unimplementedCore
	*EndpointHandlers
	*BaseStationHandlers
	*DownlinkHandlers
	*ULTransmitHandlers
	*DLRXHandlers
	*MessageHandlers
	*AnalyticsHandlers
	*CertificateHandlers
	*BlueprintHandlers
	*IntegrationHandlers
	*ScaciHandlers
	*FederationHandlers
	*SystemHandlers
}

// CoreServiceDeps wires every handler group; Log and Audit are required.
type CoreServiceDeps struct {
	Log          logger.Logger
	Audit        AuditRecorder
	Endpoints    EndpointHandlerDeps
	BaseStations BaseStationHandlerDeps
	Downlinks    DownlinkHandlerDeps
	ULTransmit   ULTransmitHandlerDeps
	DLRX         DLRXHandlerDeps
	Messages     MessageHandlerDeps
	Analytics    AnalyticsHandlerDeps
	Certificates CertificateHandlerDeps
	Blueprints   BlueprintHandlerDeps
	Integrations IntegrationHandlerDeps
	Scaci        ScaciHandlerDeps
	Federation   FederationHandlerDeps
	System       SystemHandlerDeps
}

// NewCoreService builds the facade; the first group that rejects its
// dependencies fails construction.
func NewCoreService(deps CoreServiceDeps) (*CoreService, error) {
	if deps.Log == nil {
		return nil, errors.New(errMsgLoggerCannotBeNil)
	}
	if deps.Audit == nil {
		return nil, audit.ErrNilRecorder
	}
	log := deps.Log.WithField(logger.FieldComponent, componentGRPCService)
	endpoints, err := NewEndpointHandlers(deps.Endpoints, deps.Audit, log)
	if err != nil {
		return nil, err
	}
	baseStations, err := NewBaseStationHandlers(deps.BaseStations, deps.Audit, log)
	if err != nil {
		return nil, err
	}
	downlinks, err := NewDownlinkHandlers(deps.Downlinks, log)
	if err != nil {
		return nil, err
	}
	ulTransmit, err := NewULTransmitHandlers(deps.ULTransmit, log)
	if err != nil {
		return nil, err
	}
	dlrx, err := NewDLRXHandlers(deps.DLRX, log)
	if err != nil {
		return nil, err
	}
	system, err := NewSystemHandlers(deps.System, log)
	if err != nil {
		return nil, err
	}
	return &CoreService{
		EndpointHandlers:    endpoints,
		BaseStationHandlers: baseStations,
		DownlinkHandlers:    downlinks,
		ULTransmitHandlers:  ulTransmit,
		DLRXHandlers:        dlrx,
		MessageHandlers:     NewMessageHandlers(deps.Messages, log),
		AnalyticsHandlers:   NewAnalyticsHandlers(deps.Analytics, log),
		CertificateHandlers: NewCertificateHandlers(deps.Certificates, deps.Audit, log),
		BlueprintHandlers:   NewBlueprintHandlers(deps.Blueprints, deps.Audit, log),
		IntegrationHandlers: NewIntegrationHandlers(deps.Integrations, deps.Audit, log),
		ScaciHandlers:       NewScaciHandlers(deps.Scaci, log),
		FederationHandlers:  NewFederationHandlers(deps.Federation, log),
		SystemHandlers:      system,
	}, nil
}

const (
	bsFieldMaskName        = "name"
	bsFieldMaskDescription = "description"
	bsFieldMaskLatitude    = "latitude"
	bsFieldMaskLongitude   = "longitude"
	bsFieldMaskAltitude    = "altitude"
)

// FieldMask path constants for UpdateEndPoint partial updates
const (
	fieldMaskName          = "name"
	fieldMaskDescription   = "description"
	fieldMaskEpClass       = "ep_class"
	fieldMaskStatus        = "status"
	fieldMaskNwkSnKey      = "nwk_sn_key"
	fieldMaskAppKey        = "app_key"
	fieldMaskTags          = "tags"
	fieldMaskDualChan      = "dual_chan"
	fieldMaskRepetition    = "repetition"
	fieldMaskWideCarrOff   = "wide_carr_off"
	fieldMaskLongBlkDist   = "long_blk_dist"
	fieldMaskShAddr        = "sh_addr"
	fieldMaskAttachCnt     = "attach_cnt"
	fieldMaskPreAttach     = "pre_attach"
	fieldMaskLastPacketCnt = "last_packet_cnt"
	fieldMaskCarrierOffset = "carrier_offset"
	fieldMaskDeviceModelID = "device_model_id"
	fieldMaskTypeEUI       = "type_eui"
	fieldMaskBlueprintID   = "blueprint_id"

	// Session-key and Type-EUI byte lengths per MIOTY BSSCI v1.0.0.
	endpointKeyLen = 16
	typeEUILen     = 8
)

// BaseStation operations

// Message operations

// Helper methods

// ============================================================================
// UL Data Transmit Operations (BSSCI 3.11)
// ============================================================================

// ============================================================================
// Base Station Status Operations (BSSCI 3.5)
// ============================================================================
