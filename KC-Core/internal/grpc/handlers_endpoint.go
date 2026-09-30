// Package grpc provides gRPC service implementations.
package grpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	endpointpkg "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/google/uuid"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// EndpointHandlers serves the endpoint RPCs.
type EndpointHandlers struct {
	endpointSvc            grpcservices.EndpointService
	blueprintSvc           grpcservices.BlueprintService
	endpointAttachmentSvc  grpcservices.EndpointAttachmentService
	endpointStatsStore     EndpointStatsStore
	registrationWindow     grpcservices.RegistrationWindow
	opStatusAdapter        OperationStatusAdapter
	audit                  AuditRecorder
	keyReveals             RequiredAuditRecorder
	endpointActivityWindow time.Duration
	clock                  clock.Clock
	servingStations        ServingStationLocator
	log                    logger.Logger
}

// EndpointHandlerDeps wires EndpointHandlers; Endpoints, Attachment, Clock
// and KeyReveals are required.
type EndpointHandlerDeps struct {
	// KeyReveals records every key reveal before the keys leave, refusing a reveal it cannot record.
	KeyReveals RequiredAuditRecorder
	Endpoints  grpcservices.EndpointService
	Blueprints grpcservices.BlueprintService
	Attachment grpcservices.EndpointAttachmentService
	Stats      EndpointStatsStore
	// Registrations starts the statistics at the endpoint's registration; required with Stats.
	Registrations grpcservices.RegistrationWindow
	Operations    OperationStatusAdapter
	// ActivityWindow marks an endpoint active when it was seen within it;
	// zero selects the configured default.
	ActivityWindow time.Duration
	// Clock measures the activity window.
	Clock clock.Clock
	// ServingStations names the station an endpoint's downlinks go to, shown on its detail.
	ServingStations ServingStationLocator
}

// NewEndpointHandlers validates the group and builds it.
func NewEndpointHandlers(d EndpointHandlerDeps, recorder AuditRecorder, log logger.Logger) (*EndpointHandlers, error) {
	if d.Endpoints == nil {
		return nil, errors.New(errMsgEndpointSvcCannotBeNil)
	}
	if d.Clock == nil {
		return nil, errors.New(errMsgEndpointClockCannotBeNil)
	}
	if d.Attachment == nil {
		return nil, errors.New(errMsgEndpointAttachmentCannotBeNil)
	}
	if d.Stats != nil && d.Registrations == nil {
		return nil, errors.New(errMsgEndpointStatsWindowCannotBeNil)
	}
	if d.KeyReveals == nil {
		return nil, audit.ErrNilRecorder
	}
	window := d.ActivityWindow
	if window <= 0 {
		window = time.Duration(config.DefaultEndpointActivityWindowHours) * time.Hour
	}
	return &EndpointHandlers{
		endpointSvc:            d.Endpoints,
		blueprintSvc:           d.Blueprints,
		endpointAttachmentSvc:  d.Attachment,
		endpointStatsStore:     d.Stats,
		registrationWindow:     d.Registrations,
		opStatusAdapter:        d.Operations,
		audit:                  recorder,
		keyReveals:             d.KeyReveals,
		endpointActivityWindow: window,
		clock:                  d.Clock,
		servingStations:        d.ServingStations,
		log:                    log,
	}, nil
}

// GetEndPointStats retrieves the message statistics of an endpoint's current registration.
func (s *EndpointHandlers) GetEndPointStats(ctx context.Context, req *pb.GetEndPointStatsRequest) (*pb.GetEndPointStatsResponse, error) {
	if s.endpointStatsStore == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	epEui, err := validation.ParseEUI(req.EpEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	euiParsed := models.EUIFromString(req.EpEui)
	if euiParsed == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	since, registered, err := s.registrationWindow.Start(ctx, tenantID, euiParsed[:], nil)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetEndpointStatsFailed, logger.FieldEpEuiSnake, req.EpEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointStatsFailed))
	}
	if !registered {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}

	stats, err := s.endpointStatsStore.GetMessageStatsByEndpointSince(ctx, epEui, tenantID, *since)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetEndpointStatsFailed, logger.FieldEpEuiSnake, req.EpEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointStatsFailed))
	}

	endpoint, err := s.endpointSvc.GetByEUI(ctx, euiParsed[:], tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetEndpointFailed, logger.FieldEpEuiSnake, req.EpEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}

	resp := &pb.GetEndPointStatsResponse{
		EpEui:           req.EpEui,
		TotalCount:      stats.TotalCount,
		UniqueEndpoints: stats.UniqueEndpoints,
		AvgRssi:         stats.AvgRSSI,
		AvgSnr:          stats.AvgSNR,
		AttachStatus:    endpoint.EpStatus,
	}

	// Handle optional timestamp fields
	if stats.FirstSeen != nil {
		resp.FirstSeen = timestamppb.New(*stats.FirstSeen)
	}
	if stats.LastSeen != nil {
		resp.LastSeen = timestamppb.New(*stats.LastSeen)
	}
	if stats.ActiveDays != nil {
		if resp.ActiveDays, err = totalCountOf(int64(*stats.ActiveDays)); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// GetEndPointOperations retrieves recent operations for an endpoint.
func (s *EndpointHandlers) GetEndPointOperations(ctx context.Context, req *pb.GetEndPointOperationsRequest) (*pb.GetEndPointOperationsResponse, error) {
	if s.opStatusAdapter == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	euiParsed := models.EUIFromString(req.EpEui)
	if euiParsed == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Get endpoint to resolve ep_eui to endpoint ID
	endpoint, err := s.endpointSvc.GetByEUI(ctx, euiParsed[:], tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetEndpointFailed, logger.FieldEpEuiSnake, req.EpEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}

	// Set defaults for pagination
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	offset := int(req.Offset)

	// Get operations from adapter using endpoint's database ID
	operations, err := s.opStatusAdapter.GetEndpointOperations(ctx, endpoint.ID, tenantID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetEndpointOperationsFailed, logger.FieldEpEuiSnake, req.EpEui, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointOperationsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointOperationsFailed))
	}

	// Map to proto response
	pbOps := make([]*pb.EndPointOperation, 0, len(operations))
	for _, op := range operations {
		pbOp := &pb.EndPointOperation{
			Id:        op.ID,
			EventType: op.EventType,
			Category:  op.Category,
			Severity:  op.Severity,
			Title:     op.Title,
			CreatedAt: timestamppb.New(op.CreatedAt),
		}
		pbOps = append(pbOps, pbOp)
	}

	return &pb.GetEndPointOperationsResponse{
		Operations: pbOps,
	}, nil
}

// Endpoint activity status values derived from last-seen recency.
const (
	endpointActivityActive   = "active"
	endpointActivityInactive = "inactive"
)

// CreateEndPoint registers a new endpoint under the authenticated tenant.
func (s *EndpointHandlers) CreateEndPoint(ctx context.Context, req *pb.CreateEndPointRequest) (*pb.EndPoint, error) {
	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request BEFORE accessing fields to prevent nil pointer dereference
	if req.Endpoint == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointRequired))
	}
	if req.Endpoint.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}
	if err := networkKeyError(req.Endpoint.NwkSnKey); err != nil {
		return nil, err
	}
	if len(req.Endpoint.AppKey) > 0 && len(req.Endpoint.AppKey) != 16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAppKeyLength),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength))
	}
	bidi, err := bidiOfEpClass(req.Endpoint.EpClass)
	if err != nil {
		return nil, err
	}

	// Now safe to log after validation
	s.log.InfoContext(ctx, LogCreatingEndpoint, logger.FieldEui, req.Endpoint.EpEui, logger.FieldTenantIDSnake, tenantID)

	// Validate ShAddr range before narrowing cast
	if req.Endpoint.ShAddr > math.MaxUint16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenShortAddressOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenShortAddressOverflow))
	}

	eui := models.EUIFromString(req.Endpoint.EpEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Inline pointer creation for optional fields
	var shAddr *uint16
	if req.Endpoint.ShAddr > 0 && req.Endpoint.ShAddr <= math.MaxUint16 {
		addr := uint16(req.Endpoint.ShAddr) //nolint:gosec // Bounds checked above
		shAddr = &addr
	}

	var attachCnt *uint32
	cnt := req.Endpoint.AttachCnt
	attachCnt = &cnt

	var typeEui *models.EUI
	if len(req.Endpoint.TypeEui) > 0 {
		if len(req.Endpoint.TypeEui) != 8 {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
		}
		var parsed models.EUI
		copy(parsed[:], req.Endpoint.TypeEui)
		typeEui = &parsed
	}

	// Convert proto to canonical model (use tenant from context, ignore any in request)
	var appKey []byte
	if len(req.Endpoint.AppKey) > 0 {
		appKey = req.Endpoint.AppKey
	}

	endpoint := &models.EndPoint{
		EUI:           eui,
		TenantID:      tenantID,
		OwnerTenantID: tenantID,
		Name:          req.Endpoint.Name,
		Description:   req.Endpoint.Description,
		Bidi:          bidi,
		NwkSnKey:      req.Endpoint.NwkSnKey,
		AppKey:        appKey,
		EpStatus:      endpointpkg.EndpointStatusDetached,
		Tags:          req.Endpoint.Tags,
		// MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
		ShAddr:        shAddr,
		DualChan:      req.Endpoint.DualChan,
		Repetition:    req.Endpoint.Repetition,
		WideCarrOff:   req.Endpoint.WideCarrOff,
		LongBlkDist:   req.Endpoint.LongBlkDist,
		AttachCnt:     attachCnt,
		PreAttach:     req.Endpoint.PreAttach,
		TypeEUI:       typeEui,
		CarrierOffset: int(req.Endpoint.CarrierOffset),
		LastPacketCnt: req.Endpoint.LastPacketCnt,
	}

	// Validate and set device model association for blueprint decoding
	if req.Endpoint.DeviceModelId != "" {
		if s.blueprintSvc == nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured),
			)
		}
		modelUUID, parseErr := uuid.Parse(req.Endpoint.DeviceModelId)
		if parseErr != nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
			)
		}
		if _, lookupErr := s.blueprintSvc.GetDeviceModelForTenant(ctx, tenantID, modelUUID); lookupErr != nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound),
			)
		}
		endpoint.DeviceModelID = &modelUUID

		// Resolve TypeEUI from device model's default blueprint (model is authoritative)
		effectiveTypeEui, resolveErr := s.blueprintSvc.ResolveEffectiveTypeEUI(ctx, tenantID, modelUUID)
		if resolveErr != nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed),
			)
		}
		endpoint.TypeEUI = effectiveTypeEui
	}

	if req.Endpoint.BlueprintId != "" {
		if err := applyBlueprintSnapshot(ctx, s.log, s.blueprintSvc, endpoint, req.Endpoint.BlueprintId); err != nil {
			return nil, err
		}
	}

	created, err := s.storeNewEndpoint(ctx, endpoint, req.Endpoint.PreAttach)
	if err != nil {
		return nil, err
	}

	createdEui := created.EUI.String()
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		Category:    models.EventCategoryEndpoint,
		EventType:   models.EventTypeEndpointCreated,
		Title:       models.EventTitleEndpointCreated,
		Description: fmt.Sprintf(models.EventDescriptionEndpointCreated, createdEui),
		SourceType:  models.SourceTypeEndpoint,
		SourceName:  createdEui,
		EndpointID:  &created.ID,
		Details:     map[string]any{bssci.EventKeyEpEui: createdEui},
	})

	// Convert back to proto
	return endpointToProto(created, s.endpointActivityWindow, s.clock.Now()), nil
}

// GetEndPoint retrieves an endpoint by EUI
func (s *EndpointHandlers) GetEndPoint(ctx context.Context, req *pb.GetEndPointRequest) (*pb.EndPoint, error) {
	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogGettingEndpoint, logger.FieldEui, req.EpEui, logger.FieldTenantIDSnake, tenantID)

	// Validate request
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	eui := models.EUIFromString(req.EpEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Get endpoint from storage (use tenant from context)
	endpoint, err := s.endpointSvc.GetByEUI(ctx, eui[:], tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetEndpoint, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}

	resp := endpointToProto(endpoint, s.endpointActivityWindow, s.clock.Now())
	resp.ServingBsEui = s.servingStationEUI(ctx, tenantID, eui)
	revealed, err := revealEndpointKeys(resp, endpoint, req.GetRevealKeys())
	if err != nil {
		return nil, err
	}
	if len(revealed) == 0 {
		return resp, nil
	}
	if err := s.emitKeysRevealedEvent(ctx, tenantID, endpoint, revealed); err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	return resp, nil
}

// servingStationEUI names the station a downlink queued now would go to;
// empty when none serves the endpoint or the lookup fails, which is logged.
func (s *EndpointHandlers) servingStationEUI(ctx context.Context, tenantID int64, eui models.EUI) string {
	if s.servingStations == nil {
		return ""
	}
	bsEUI, known, err := s.servingStations.ServingStation(ctx, tenantID, eui.ToUint64())
	if err != nil {
		s.log.WarnContext(ctx, LogServingStationLookupFailed, logger.FieldEui, eui.String(), logger.FieldError, err)
		return ""
	}
	if !known {
		return ""
	}
	return mioty.FormatEUI64(bsEUI)
}

// supportedEndpointUpdateMaskPaths enumerates every field path UpdateEndPoint
// honors. A path outside this set is rejected before any mutation.
var supportedEndpointUpdateMaskPaths = map[string]struct{}{
	fieldMaskName:          {},
	fieldMaskDescription:   {},
	fieldMaskEpClass:       {},
	fieldMaskStatus:        {},
	fieldMaskNwkSnKey:      {},
	fieldMaskAppKey:        {},
	fieldMaskTags:          {},
	fieldMaskDualChan:      {},
	fieldMaskRepetition:    {},
	fieldMaskWideCarrOff:   {},
	fieldMaskLongBlkDist:   {},
	fieldMaskShAddr:        {},
	fieldMaskAttachCnt:     {},
	fieldMaskPreAttach:     {},
	fieldMaskLastPacketCnt: {},
	fieldMaskCarrierOffset: {},
	fieldMaskDeviceModelID: {},
	fieldMaskTypeEUI:       {},
	fieldMaskBlueprintID:   {},
}

// fieldInMask delegates to the shared package for cross-service use.
func fieldInMask(mask *fieldmaskpb.FieldMask, field string) bool {
	return grpcerrors.FieldInMask(mask, field)
}

// UpdateEndPoint updates an endpoint
func (s *EndpointHandlers) UpdateEndPoint(ctx context.Context, req *pb.UpdateEndPointRequest) (*pb.EndPoint, error) {
	endpoint, tenantID, err := s.loadEndpointForUpdate(ctx, req)
	if err != nil {
		return nil, err
	}
	mask, err := requireEndpointUpdateMask(req)
	if err != nil {
		return nil, err
	}
	statusDecision, err := s.requestedStatusDecision(endpoint, req, mask)
	if err != nil {
		return nil, err
	}
	priorKeys := storedEndpointKeys(endpoint)
	if err := s.applyEndpointUpdate(ctx, tenantID, endpoint, req, mask); err != nil {
		return nil, err
	}
	if endpoint, err = s.saveEndpointUpdate(ctx, tenantID, endpoint, req); err != nil {
		return nil, err
	}
	if statusDecision != "" {
		if endpoint, err = s.decideEndpointStatus(ctx, tenantID, endpoint.EUI, statusDecision); err != nil {
			return nil, err
		}
	}

	s.emitEndpointUpdatedEvent(ctx, tenantID, endpoint)
	if removed := removedEndpointKeys(priorKeys, endpoint); len(removed) > 0 {
		s.emitKeysRemovedEvent(ctx, tenantID, endpoint, removed)
	}

	return endpointToProto(endpoint, s.endpointActivityWindow, s.clock.Now()), nil
}

// requireEndpointUpdateMask returns the request's field mask, which a partial
// update requires and which must name only supported paths, once each.
func requireEndpointUpdateMask(req *pb.UpdateEndPointRequest) (*fieldmaskpb.FieldMask, error) {
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMaskRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMaskRequired),
		)
	}
	if err := validateEndpointUpdateMask(mask); err != nil {
		return nil, err
	}
	return mask, nil
}

// applyEndpointUpdate applies the masked fields to the loaded endpoint,
// together with the device model, type EUI and blueprint snapshot they imply.
func (s *EndpointHandlers) applyEndpointUpdate(ctx context.Context, tenantID int64, endpoint *models.EndPoint, req *pb.UpdateEndPointRequest, mask *fieldmaskpb.FieldMask) error {
	// Captured before mask mutations — needed by the snapshot re-materialization trigger.
	priorDeviceModelID := endpoint.DeviceModelID
	priorSnapshotSourceID := snapshotSourceID(endpoint.BlueprintSnapshot)
	priorProfile := endpointpkg.StationProfileOf(endpoint)

	if err := applyEndpointMaskedFields(endpoint, req, mask); err != nil {
		return err
	}
	endpointpkg.RecordProfileChange(endpoint, priorProfile, s.clock.Now())
	if err := s.resolveEndpointDeviceModel(ctx, tenantID, endpoint, req, mask); err != nil {
		return err
	}
	if err := s.normalizeEndpointTypeEUIFromModel(ctx, tenantID, endpoint); err != nil {
		return err
	}
	if err := s.applyEndpointSnapshotTrigger(ctx, tenantID, endpoint, req, mask, priorDeviceModelID, priorSnapshotSourceID); err != nil {
		return err
	}
	// type_eui follows the active snapshot (overrides the model-default normalization above).
	if te := snapshotTypeEUI(endpoint.BlueprintSnapshot); te != nil {
		endpoint.TypeEUI = te
	}
	return nil
}

// saveEndpointUpdate persists the updated endpoint, through the EUI-change
// cascade when the request renames it.
func (s *EndpointHandlers) saveEndpointUpdate(ctx context.Context, tenantID int64, endpoint *models.EndPoint, req *pb.UpdateEndPointRequest) (*models.EndPoint, error) {
	current, euiChanged, err := s.changeEndpointEUI(ctx, tenantID, endpoint, req)
	if err != nil {
		return nil, err
	}
	if euiChanged {
		return current, nil
	}
	return s.persistEndpointUpdate(ctx, current)
}

func (s *EndpointHandlers) loadEndpointForUpdate(ctx context.Context, req *pb.UpdateEndPointRequest) (*models.EndPoint, int64, error) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if req.Endpoint == nil {
		return nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointRequired))
	}
	if req.Endpoint.EpEui == "" {
		return nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}
	s.log.InfoContext(ctx, LogUpdatingEndpoint, logger.FieldEui, req.Endpoint.EpEui, logger.FieldTenantIDSnake, tenantID)

	eui := models.EUIFromString(req.Endpoint.EpEui)
	if eui == (models.EUI{}) {
		return nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}
	tenantStr := strconv.FormatInt(tenantID, 10)
	if req.Endpoint.TenantId != "" && req.Endpoint.TenantId != tenantStr {
		return nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	endpoint, err := s.endpointSvc.GetByEUI(ctx, eui[:], tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetEndpointForUpdate, logger.FieldError, err)
		return nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}
	return endpoint, tenantID, nil
}

// validateEndpointUpdateMask rejects unsupported or duplicated field paths
// before any mutation, so a malformed mask never partially applies.
func validateEndpointUpdateMask(mask *fieldmaskpb.FieldMask) error {
	seen := make(map[string]struct{}, len(mask.GetPaths()))
	for _, path := range mask.GetPaths() {
		if _, ok := supportedEndpointUpdateMaskPaths[path]; !ok {
			return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUnknownFieldMaskPath),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUnknownFieldMaskPath))
		}
		if _, dup := seen[path]; dup {
			return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDuplicateFieldMaskPath),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDuplicateFieldMaskPath))
		}
		seen[path] = struct{}{}
	}
	return nil
}

// bidiOfEpClass reads the bidirectional flag an ep_class names; storage
// derives the stored class back from the flag.
func bidiOfEpClass(class string) (bool, error) {
	switch class {
	case mioty.EndpointClassBidirectional:
		return true, nil
	case mioty.EndpointClassUnidirectional:
		return false, nil
	default:
		return false, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEpClass),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEpClass))
	}
}

// applyMaskedIdentityFields mutates the name/description/ep_class/key/tag
// fields only when their path is present in the mask, enforcing per-field
// clear-ability and format rules.
func applyMaskedIdentityFields(endpoint *models.EndPoint, req *pb.UpdateEndPointRequest, mask *fieldmaskpb.FieldMask) error {
	if fieldInMask(mask, fieldMaskName) {
		if req.Endpoint.Name == "" {
			return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
		}
		endpoint.Name = req.Endpoint.Name
	}
	if fieldInMask(mask, fieldMaskDescription) {
		endpoint.Description = req.Endpoint.Description
	}
	if fieldInMask(mask, fieldMaskEpClass) {
		bidi, err := bidiOfEpClass(req.Endpoint.EpClass)
		if err != nil {
			return err
		}
		endpoint.Bidi = bidi
	}
	if fieldInMask(mask, fieldMaskNwkSnKey) {
		if err := networkKeyError(req.Endpoint.NwkSnKey); err != nil {
			return err
		}
		endpoint.NwkSnKey = req.Endpoint.NwkSnKey
	}
	if fieldInMask(mask, fieldMaskAppKey) {
		switch len(req.Endpoint.AppKey) {
		case 0:
			endpoint.AppKey = nil
		case endpointKeyLen:
			endpoint.AppKey = req.Endpoint.AppKey
		default:
			return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAppKeyLength),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength))
		}
	}
	if fieldInMask(mask, fieldMaskTags) {
		endpoint.Tags = req.Endpoint.Tags
	}
	return nil
}

func applyMaskedTypeEUI(endpoint *models.EndPoint, req *pb.UpdateEndPointRequest) error {
	if len(req.Endpoint.TypeEui) == 0 {
		endpoint.TypeEUI = nil
		return nil
	}
	if len(req.Endpoint.TypeEui) != typeEUILen {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
	}
	var parsed models.EUI
	copy(parsed[:], req.Endpoint.TypeEui)
	endpoint.TypeEUI = &parsed
	return nil
}

// Proto attach_status is UNMAPPED per spec — no model field exists.
func applyEndpointMaskedFields(endpoint *models.EndPoint, req *pb.UpdateEndPointRequest, mask *fieldmaskpb.FieldMask) error {
	if err := applyMaskedIdentityFields(endpoint, req, mask); err != nil {
		return err
	}
	if fieldInMask(mask, fieldMaskShAddr) {
		if req.Endpoint.ShAddr > math.MaxUint16 {
			return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenShortAddressOverflow),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenShortAddressOverflow))
		}
		addr := uint16(req.Endpoint.ShAddr) //nolint:gosec // Bounds checked above
		endpoint.ShAddr = &addr
	}
	if fieldInMask(mask, fieldMaskAttachCnt) {
		cnt := req.Endpoint.AttachCnt
		endpoint.AttachCnt = &cnt
	}
	if fieldInMask(mask, fieldMaskLastPacketCnt) {
		endpoint.LastPacketCnt = req.Endpoint.LastPacketCnt
	}
	if fieldInMask(mask, fieldMaskCarrierOffset) {
		endpoint.CarrierOffset = int(req.Endpoint.CarrierOffset)
	}
	if fieldInMask(mask, fieldMaskDualChan) {
		endpoint.DualChan = req.Endpoint.DualChan
	}
	if fieldInMask(mask, fieldMaskRepetition) {
		endpoint.Repetition = req.Endpoint.Repetition
	}
	if fieldInMask(mask, fieldMaskWideCarrOff) {
		endpoint.WideCarrOff = req.Endpoint.WideCarrOff
	}
	if fieldInMask(mask, fieldMaskLongBlkDist) {
		endpoint.LongBlkDist = req.Endpoint.LongBlkDist
	}
	if fieldInMask(mask, fieldMaskPreAttach) {
		endpoint.PreAttach = req.Endpoint.PreAttach
	}
	if fieldInMask(mask, fieldMaskTypeEUI) {
		return applyMaskedTypeEUI(endpoint, req)
	}
	return nil
}

func (s *EndpointHandlers) resolveEndpointDeviceModel(ctx context.Context, tenantID int64, endpoint *models.EndPoint, req *pb.UpdateEndPointRequest, mask *fieldmaskpb.FieldMask) error {
	if !fieldInMask(mask, fieldMaskDeviceModelID) {
		return nil
	}
	if req.Endpoint.DeviceModelId == "" {
		endpoint.DeviceModelID = nil
		return nil
	}
	if s.blueprintSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	modelUUID, parseErr := uuid.Parse(req.Endpoint.DeviceModelId)
	if parseErr != nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat))
	}
	if _, lookupErr := s.blueprintSvc.GetDeviceModelForTenant(ctx, tenantID, modelUUID); lookupErr != nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
	}
	endpoint.DeviceModelID = &modelUUID
	return nil
}

// Model default is authoritative for type_eui.
func (s *EndpointHandlers) normalizeEndpointTypeEUIFromModel(ctx context.Context, tenantID int64, endpoint *models.EndPoint) error {
	if endpoint.DeviceModelID == nil {
		return nil
	}
	if s.blueprintSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	effectiveTypeEui, resolveErr := s.blueprintSvc.ResolveEffectiveTypeEUI(ctx, tenantID, *endpoint.DeviceModelID)
	if resolveErr != nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed))
	}
	endpoint.TypeEUI = effectiveTypeEui
	return nil
}

// Explicit selection wins; else re-seed from the new model's default when the model changed and the pinned blueprint isn't native.
func (s *EndpointHandlers) applyEndpointSnapshotTrigger(ctx context.Context, tenantID int64, endpoint *models.EndPoint, req *pb.UpdateEndPointRequest, mask *fieldmaskpb.FieldMask, priorDeviceModelID *uuid.UUID, priorSnapshotSourceID string) error {
	if fieldInMask(mask, fieldMaskBlueprintID) && req.Endpoint.BlueprintId != "" && req.Endpoint.BlueprintId != priorSnapshotSourceID {
		return applyBlueprintSnapshot(ctx, s.log, s.blueprintSvc, endpoint, req.Endpoint.BlueprintId)
	}
	reSeed := modelChanged(priorDeviceModelID, endpoint.DeviceModelID) && endpoint.DeviceModelID != nil &&
		!pinnedBlueprintBelongsToModel(ctx, s.blueprintSvc, priorSnapshotSourceID, *endpoint.DeviceModelID)
	if !reSeed {
		return nil
	}
	def, defErr := s.blueprintSvc.GetDefaultForModel(ctx, tenantID, *endpoint.DeviceModelID)
	if defErr != nil {
		// Infra failure — do not silently continue an update that touches the decode source.
		s.log.ErrorContext(ctx, LogFailedToResolveNewModelDefaultBlueprintForReSeed,
			logger.FieldError, defErr, logger.FieldEndpointEui, endpoint.EUI.String())
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed))
	}
	if def == nil {
		s.log.WarnContext(ctx, LogDeviceModelChangedNoDefaultBlueprint,
			logger.FieldEndpointEui, endpoint.EUI.String())
		return nil
	}
	return applyBlueprintSnapshot(ctx, s.log, s.blueprintSvc, endpoint, def.ID.String())
}

// Maps storage errors common to both persist paths; nil when unmatched (caller applies its default).
func mapEndpointPersistError(err error) error {
	switch {
	case errors.Is(err, storage.ErrAlreadyExists):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointExists),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists))
	case errors.Is(err, storage.ErrForeignKeyViolation):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
	case errors.Is(err, storage.ErrNwkKeyLength):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNwkSnKeyLength),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength))
	case errors.Is(err, storage.ErrAppKeyLength):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAppKeyLength),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength))
	default:
		return nil
	}
}

// Atomic EUI-change cascade when new_ep_eui is set; reports whether the EUI changed.
func (s *EndpointHandlers) changeEndpointEUI(ctx context.Context, tenantID int64, endpoint *models.EndPoint, req *pb.UpdateEndPointRequest) (*models.EndPoint, bool, error) {
	if req.NewEpEui == "" {
		return endpoint, false, nil
	}
	newEui := models.EUIFromString(req.NewEpEui)
	if newEui == (models.EUI{}) {
		return nil, false, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}
	if newEui == endpoint.EUI {
		return endpoint, false, nil
	}
	// Uniqueness check (optimistic fast-path; repo also catches constraint race)
	if checkErr := s.endpointSvc.CheckEUIGloballyUnique(ctx, newEui[:]); checkErr != nil {
		if errors.Is(checkErr, storage.ErrAlreadyExists) {
			return nil, false, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointExists),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists))
		}
		s.log.ErrorContext(ctx, LogFailedToCheckEUIUniqueness, logger.FieldError, checkErr)
		return nil, false, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateEndpointEUIFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointEUIFailed))
	}
	oldEui := endpoint.EUI
	endpoint.EUI = newEui
	updated, updateErr := s.endpointSvc.UpdateWithEUI(ctx, tenantID, oldEui[:], endpoint)
	if updateErr != nil {
		if mapped := mapEndpointPersistError(updateErr); mapped != nil {
			return nil, false, mapped
		}
		s.log.ErrorContext(ctx, LogFailedToUpdateEndpointEUI, logger.FieldError, updateErr)
		return nil, false, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateEndpointEUIFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointEUIFailed))
	}
	return updated, true, nil
}

// Persists non-EUI changes; EUI changes are already persisted atomically by UpdateWithEUI.
func (s *EndpointHandlers) persistEndpointUpdate(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	updated, updateErr := s.endpointSvc.Update(ctx, endpoint)
	if errors.Is(updateErr, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if mapped := mapEndpointPersistError(updateErr); mapped != nil {
		return nil, mapped
	}
	if updateErr != nil {
		s.log.ErrorContext(ctx, LogFailedToUpdateEndpoint, logger.FieldError, updateErr)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointFailed))
	}
	return updated, nil
}

func (s *EndpointHandlers) emitEndpointUpdatedEvent(ctx context.Context, tenantID int64, endpoint *models.EndPoint) {
	epEui := endpoint.EUI.String()
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		Category:    models.EventCategoryEndpoint,
		EventType:   models.EventTypeEndpointUpdated,
		Title:       models.EventTitleEndpointUpdated,
		Description: fmt.Sprintf(models.EventDescriptionEndpointUpdated, epEui),
		SourceType:  models.SourceTypeEndpoint,
		SourceName:  epEui,
		EndpointID:  &endpoint.ID,
		Details:     map[string]any{bssci.EventKeyEpEui: epEui},
	})
}

// DeleteEndPoint deletes an endpoint
func (s *EndpointHandlers) DeleteEndPoint(ctx context.Context, req *pb.DeleteEndPointRequest) (*emptypb.Empty, error) {
	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogDeletingEndpoint, logger.FieldEui, req.EpEui, logger.FieldTenantIDSnake, tenantID)

	// Validate request
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	eui := models.EUIFromString(req.EpEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	removedID, err := s.endpointSvc.Delete(ctx, eui[:], tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToDeleteEndpoint, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteEndpointFailed))
	}

	removedEui := eui.String()
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		Category:    models.EventCategoryEndpoint,
		EventType:   models.EventTypeEndpointDeleted,
		Title:       models.EventTitleEndpointDeleted,
		Description: fmt.Sprintf(models.EventDescriptionEndpointDeleted, removedEui),
		SourceType:  models.SourceTypeEndpoint,
		SourceName:  removedEui,
		EndpointID:  &removedID,
		Details:     map[string]any{bssci.EventKeyEpEui: removedEui},
	})

	return &emptypb.Empty{}, nil
}

// ListEndPoints lists endpoints for a tenant
func (s *EndpointHandlers) ListEndPoints(ctx context.Context, req *pb.ListEndPointsRequest) (*pb.ListEndPointsResponse, error) {
	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogListingEndpoints, logger.FieldTenantIDSnake, tenantID, logger.FieldPageSize, req.PageSize, logger.FieldPageToken, req.PageToken)

	// Default page size
	pageSize := clampHighVolumePageSize(req.PageSize, DefaultHighVolumePageSize)

	// Parse page token (simple offset-based pagination)
	offset := 0
	if req.PageToken != "" {
		_, err := fmt.Sscanf(req.PageToken, "%d", &offset)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	// device_model_id filters to snapshot-bearing endpoints of one model (bulk coverage preview, unpaginated).
	var endpoints []*models.EndPoint
	if req.DeviceModelId != "" {
		modelID, parseErr := uuid.Parse(req.DeviceModelId)
		if parseErr != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat))
		}
		endpoints, err = s.endpointSvc.ListByModelWithSnapshot(ctx, tenantID, modelID)
	} else {
		endpoints, err = s.endpointSvc.List(ctx, tenantID, int(pageSize), offset)
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToListEndpoints, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListEndpointsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListEndpointsFailed))
	}

	// Convert to proto
	now := s.clock.Now()
	pbEndPoints := make([]*pb.EndPoint, len(endpoints))
	for i, endpoint := range endpoints {
		pbEndPoints[i] = endpointToProto(endpoint, s.endpointActivityWindow, now)
	}

	// Generate next page token (model-filtered path is unpaginated)
	nextPageToken := ""
	if req.DeviceModelId == "" && len(endpoints) == int(pageSize) {
		nextPageToken = fmt.Sprintf("%d", offset+int(pageSize))
	}

	return &pb.ListEndPointsResponse{
		Endpoints:     pbEndPoints,
		NextPageToken: nextPageToken,
	}, nil
}

// AttachEndPoint attaches an endpoint and propagates the attachment.
func (s *EndpointHandlers) AttachEndPoint(ctx context.Context, req *pb.AttachEndPointRequest) (*pb.AttachEndPointResponse, error) {
	result, err := s.decideRequestedAttachment(ctx, req.EpEui, endpointpkg.EndpointStatusAttached)
	if err != nil {
		return nil, err
	}
	return &pb.AttachEndPointResponse{OperationId: result.OperationID, Status: result.Status}, nil
}

// DetachEndPoint detaches an endpoint and propagates the detachment.
func (s *EndpointHandlers) DetachEndPoint(ctx context.Context, req *pb.DetachEndPointRequest) (*pb.DetachEndPointResponse, error) {
	result, err := s.decideRequestedAttachment(ctx, req.EpEui, endpointpkg.EndpointStatusDetached)
	if err != nil {
		return nil, err
	}
	return &pb.DetachEndPointResponse{OperationId: result.OperationID, Status: result.Status}, nil
}

// networkKeyError refuses a network session key a base station could not use:
// not 16 bytes, or all zeros.
func networkKeyError(key []byte) error {
	if len(key) != endpointKeyLen {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNwkSnKeyLength),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength))
	}
	if bytes.Equal(key, make([]byte, endpointKeyLen)) {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNwkSnKeyZero),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyZero))
	}
	return nil
}
