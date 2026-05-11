package grpc

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	healthstatus "github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	grpcservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/statistics"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

// BSSCISessionCloser provides session termination for EUI changes.
// Used to close active BSSCI sessions when a base station's EUI is modified.
type BSSCISessionCloser interface {
	CloseSessionByEUI(ctx context.Context, eui uint64) error
}

// CoreService implements the CoreService gRPC service for device management,
// protocol operations, analytics, monitoring, certificates, blueprints, and integrations.
type CoreService struct {
	pb.UnimplementedCoreServiceServer
	endpointSvc       grpcservices.EndpointService
	basestationSvc    grpcservices.BaseStationService
	messageSvc        grpcservices.MessageService
	downlinkSvc       bssci.DownlinkService
	statusSvc         bssci.StatusService
	downlinkCmd       bssci.DownlinkCommander
	downlinkScheduler scheduler.DownlinkScheduler
	sessionDir        bssci.SessionDirectory
	ulTransmit        bssci.ULTransmitter
	statusReq         bssci.StatusRequester
	pingCmd           bssci.PingCommander
	log               logger.Logger
	statsStore        MessageStore
	downlinkStore     DownlinkStore // For EnqueueDownlink and UpdateDownlinkStatus operations
	// BSSCI §5.15: Narrow interface for DL RX status query telemetry
	dlrxStorage     DLRXStatusQueryStorage
	systemStatusSvc grpcservices.SystemStatusService
	// SCACI §3.10: Optional SCACI downlink queuer for gRPC→SCACI delegation
	// When set, SendDownlink delegates to SCACI handler core (single-source processing)
	scaciQueuer SCACIDownlinkQueuer
	// Service Center identity (BSSCI §5.3.2 / SCACI §3.3.2)
	scEui       uint64
	scVendor    string
	scModel     string
	scName      string
	scSwVersion string
	edition     string

	endpointAttachmentSvc  grpcservices.EndpointAttachmentService
	analyticsSvc           grpcservices.AnalyticsService
	eventSvc               grpcservices.EventService
	alertSvc               grpcservices.AlertService
	scaciMonitorSvc        grpcservices.ScaciMonitoringService
	certSvc                grpcservices.CertificateService
	blueprintSvc           grpcservices.BlueprintService
	msgListingSvc          grpcservices.MessageListingService
	integrationSvc         grpcservices.IntegrationService
	statisticsSvc          grpcservices.StatisticsService
	endpointStatsStore     EndpointStatsStore
	opStatusAdapter        OperationStatusAdapter
	activitySvc            grpcservices.ActivityService
	streamPollInterval     time.Duration
	serviceStart           time.Time
	protocolConfig         *config.ProtocolConfig
	endpointActivityWindow time.Duration

	// EUI update session termination (optional - set via WithBSSCISessionCloser)
	bssciSessionCloser BSSCISessionCloser
	// Base station event recording (optional - set via WithBSEventRecorder)
	bsEventRecorder basestation.EventRecorder
	// CRUD event recording (optional - set via WithEventWriter)
	eventWriter grpcservices.EventWriter

	// Server admin / org mapping (optional - set via WithAdminChecker / WithOrgMapper)
	adminChecker AdminChecker
	orgMapper    OrgMapper

	// Federation services (optional - only wired in CE/ECE mode)
	ceBootstrapSvc CEBootstrapHandler
	ceRegistrySvc  CERegistryHandler
}

// CoreServiceDeps bundles the required dependencies for NewCoreService.
type CoreServiceDeps struct {
	EndpointSvc       grpcservices.EndpointService
	BasestationSvc    grpcservices.BaseStationService
	MessageSvc        grpcservices.MessageService
	DownlinkSvc       bssci.DownlinkService
	StatusSvc         bssci.StatusService
	DownlinkCmd       bssci.DownlinkCommander
	DownlinkScheduler scheduler.DownlinkScheduler
	SessionDir        bssci.SessionDirectory
	ULTransmit        bssci.ULTransmitter
	StatusReq         bssci.StatusRequester
	PingCmd           bssci.PingCommander
	StatsStore        MessageStore
	DownlinkStore     DownlinkStore
	DLRXStorage       DLRXStatusQueryStorage
	SCEui             uint64
	SCVendor          string
	SCModel           string
	SCName            string
	SCSwVersion       string
	Edition           string
}

// NewCoreService creates a new service instance from a dependency bundle.
func NewCoreService(deps CoreServiceDeps) (*CoreService, error) {
	// Validate required collaborators
	if deps.EndpointSvc == nil {
		return nil, fmt.Errorf("endpointSvc cannot be nil")
	}
	if deps.BasestationSvc == nil {
		return nil, fmt.Errorf("basestationSvc cannot be nil")
	}
	if deps.MessageSvc == nil {
		return nil, fmt.Errorf("messageSvc cannot be nil")
	}
	if deps.DownlinkCmd == nil {
		return nil, fmt.Errorf("downlinkCmd cannot be nil")
	}
	if deps.DownlinkScheduler == nil {
		return nil, fmt.Errorf("downlinkScheduler cannot be nil")
	}
	if deps.SessionDir == nil {
		return nil, fmt.Errorf("sessionDir cannot be nil")
	}
	if deps.ULTransmit == nil {
		return nil, fmt.Errorf("ulTransmit cannot be nil")
	}
	if deps.StatusReq == nil {
		return nil, fmt.Errorf("statusReq cannot be nil")
	}
	if deps.PingCmd == nil {
		return nil, fmt.Errorf("pingCmd cannot be nil")
	}
	if deps.StatsStore == nil {
		return nil, fmt.Errorf("statsStore cannot be nil")
	}
	if deps.DownlinkStore == nil {
		return nil, fmt.Errorf("downlinkStore cannot be nil")
	}
	if deps.DLRXStorage == nil {
		return nil, fmt.Errorf("dlrxStorage cannot be nil")
	}
	if deps.DownlinkSvc == nil {
		return nil, fmt.Errorf("downlinkSvc cannot be nil")
	}
	if deps.StatusSvc == nil {
		return nil, fmt.Errorf("statusSvc cannot be nil")
	}

	// Create logger once and reuse
	log := logger.Get().WithField("component", "grpc-service")

	return &CoreService{
		endpointSvc:            deps.EndpointSvc,
		basestationSvc:         deps.BasestationSvc,
		messageSvc:             deps.MessageSvc,
		downlinkSvc:            deps.DownlinkSvc,
		statusSvc:              deps.StatusSvc,
		downlinkCmd:            deps.DownlinkCmd,
		downlinkScheduler:      deps.DownlinkScheduler,
		sessionDir:             deps.SessionDir,
		ulTransmit:             deps.ULTransmit,
		statusReq:              deps.StatusReq,
		pingCmd:                deps.PingCmd,
		statsStore:             deps.StatsStore,
		downlinkStore:          deps.DownlinkStore,
		dlrxStorage:            deps.DLRXStorage,
		scEui:                  deps.SCEui,
		scVendor:               deps.SCVendor,
		scModel:                deps.SCModel,
		scName:                 deps.SCName,
		scSwVersion:            deps.SCSwVersion,
		edition:                deps.Edition,
		log:                    log,
		serviceStart:           time.Now(),
		endpointActivityWindow: time.Duration(config.DefaultEndpointActivityWindowHours) * time.Hour,
	}, nil
}

// WithSCACIQueuer sets the SCACI downlink queuer for gRPC→SCACI delegation.
// When set, SendDownlink delegates to SCACI handler core instead of using direct scheduler.
// This ensures single-source processing for both socket and gRPC paths.
//
// Parameters:
//   - queuer: SCACI server implementing QueueDownlinkInternal
//
// Returns:
//   - *CoreService: The service instance for method chaining
//
// Usage:
//
//	svc, _ := NewCoreService(CoreServiceDeps{...})
//	svc.WithSCACIQueuer(scaciServer) // Enable SCACI delegation
func (s *CoreService) WithSCACIQueuer(queuer SCACIDownlinkQueuer) *CoreService {
	s.scaciQueuer = queuer
	if queuer != nil {
		s.log.Info("gRPC service configured with SCACI queuer delegation")
	}
	return s
}

// WithProtocolConfig sets the protocol configuration for BSSCI service center URL.
func (s *CoreService) WithProtocolConfig(cfg *config.ProtocolConfig) *CoreService {
	s.protocolConfig = cfg
	return s
}

// WithEndpointAttachmentService sets the endpoint attachment service for attach/detach operations.
func (s *CoreService) WithEndpointAttachmentService(svc grpcservices.EndpointAttachmentService) *CoreService {
	s.endpointAttachmentSvc = svc
	return s
}

// WithAnalyticsService sets the analytics service for analytics queries.
func (s *CoreService) WithAnalyticsService(svc grpcservices.AnalyticsService) *CoreService {
	s.analyticsSvc = svc
	return s
}

// WithEventService sets the event service for event listing.
func (s *CoreService) WithEventService(svc grpcservices.EventService) *CoreService {
	s.eventSvc = svc
	return s
}

// WithAlertService sets the alert service for alert queries.
func (s *CoreService) WithAlertService(svc grpcservices.AlertService) *CoreService {
	s.alertSvc = svc
	return s
}

// WithScaciMonitoringService sets the SCACI monitoring service.
func (s *CoreService) WithScaciMonitoringService(svc grpcservices.ScaciMonitoringService) *CoreService {
	s.scaciMonitorSvc = svc
	return s
}

// WithCertificateService sets the certificate service.
func (s *CoreService) WithCertificateService(svc grpcservices.CertificateService) *CoreService {
	s.certSvc = svc
	return s
}

// WithBlueprintService sets the blueprint service for device catalog management.
func (s *CoreService) WithBlueprintService(svc grpcservices.BlueprintService) *CoreService {
	s.blueprintSvc = svc
	return s
}

// WithMessageListingService sets the message listing service.
func (s *CoreService) WithMessageListingService(svc grpcservices.MessageListingService) *CoreService {
	s.msgListingSvc = svc
	return s
}

// WithStatisticsService sets the statistics aggregation service.
func (s *CoreService) WithStatisticsService(svc grpcservices.StatisticsService) *CoreService {
	s.statisticsSvc = svc
	return s
}

// WithIntegrationService sets the integration service for event sink management.
func (s *CoreService) WithIntegrationService(svc grpcservices.IntegrationService) *CoreService {
	s.integrationSvc = svc
	return s
}

// WithEndpointStatsStore sets the endpoint stats store for endpoint message statistics.
func (s *CoreService) WithEndpointStatsStore(store EndpointStatsStore) *CoreService {
	s.endpointStatsStore = store
	return s
}

// WithOperationStatusAdapter sets the operation status adapter for endpoint operations.
func (s *CoreService) WithOperationStatusAdapter(adapter OperationStatusAdapter) *CoreService {
	s.opStatusAdapter = adapter
	return s
}

// WithActivityService sets the activity service for unified activity feed.
func (s *CoreService) WithActivityService(svc grpcservices.ActivityService) *CoreService {
	s.activitySvc = svc
	return s
}

// WithStreamPollInterval sets the polling interval for streaming RPCs.
func (s *CoreService) WithStreamPollInterval(interval time.Duration) *CoreService {
	s.streamPollInterval = interval
	return s
}

// WithSystemStatusService sets the system status service for tenant-scoped metrics.
func (s *CoreService) WithSystemStatusService(svc grpcservices.SystemStatusService) *CoreService {
	s.systemStatusSvc = svc
	return s
}

// WithBSSCISessionCloser sets the BSSCI session closer for EUI change handling.
// When set, UpdateBaseStationEui will terminate active BSSCI sessions for the old EUI.
func (s *CoreService) WithBSSCISessionCloser(closer BSSCISessionCloser) *CoreService {
	s.bssciSessionCloser = closer
	return s
}

// WithBSEventRecorder sets the base station event recorder for EUI change events.
// Uses existing basestation.EventRecorder interface (defined in connection_manager.go).
func (s *CoreService) WithBSEventRecorder(recorder basestation.EventRecorder) *CoreService {
	s.bsEventRecorder = recorder
	return s
}

// WithEventWriter sets the event writer for system events.
func (s *CoreService) WithEventWriter(w grpcservices.EventWriter) *CoreService {
	s.eventWriter = w
	return s
}

// WithEndpointActivityWindow sets the endpoint activity threshold.
// Endpoints seen within this window are reported as "active" in the API.
func (s *CoreService) WithEndpointActivityWindow(window time.Duration) *CoreService {
	if window <= 0 {
		window = time.Duration(config.DefaultEndpointActivityWindowHours) * time.Hour
	}
	s.endpointActivityWindow = window
	return s
}

// CEBootstrapHandler is the canonical interface from pkg/federation.
type CEBootstrapHandler = federation.CEBootstrapHandler

// CERegistryHandler is the canonical interface from pkg/federation.
type CERegistryHandler = federation.CERegistryHandler

// WithAdminChecker sets the admin checker for server-admin authorization.
func (s *CoreService) WithAdminChecker(checker AdminChecker) *CoreService {
	s.adminChecker = checker
	return s
}

// WithOrgMapper sets the org mapper for tenant→org UUID resolution.
func (s *CoreService) WithOrgMapper(mapper OrgMapper) *CoreService {
	s.orgMapper = mapper
	return s
}

// WithCEBootstrapHandler sets the CE onboarding service.
func (s *CoreService) WithCEBootstrapHandler(h CEBootstrapHandler) *CoreService {
	s.ceBootstrapSvc = h
	return s
}

// WithCERegistryHandler sets the ECE CE registry service.
func (s *CoreService) WithCERegistryHandler(h CERegistryHandler) *CoreService {
	s.ceRegistrySvc = h
	return s
}

// EndPoint operations

// CreateEndPoint creates a new endpoint
func (s *CoreService) CreateEndPoint(ctx context.Context, req *pb.CreateEndPointRequest) (*pb.EndPoint, error) {
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
	if len(req.Endpoint.NwkSnKey) != 16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNwkSnKeyLength),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength))
	}
	if len(req.Endpoint.AppKey) > 0 && len(req.Endpoint.AppKey) != 16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAppKeyLength),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength))
	}

	// Now safe to log after validation
	s.log.InfoContext(ctx, "Creating endpoint", "eui", req.Endpoint.EpEui, "tenant_id", tenantID)

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
	// Status takes priority over AttachStatus for frontend compatibility
	epStatus := req.Endpoint.Status
	if epStatus == "" {
		epStatus = req.Endpoint.AttachStatus
	}
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
		EPClass:       req.Endpoint.EpClass,
		Bidi:          req.Endpoint.EpClass == epClassBidirectional,
		NwkSnKey:      req.Endpoint.NwkSnKey,
		AppKey:        appKey,
		EpStatus:      epStatus, // Status takes priority, AttachStatus as fallback
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
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
		}
		modelUUID, parseErr := uuid.Parse(req.Endpoint.DeviceModelId)
		if parseErr != nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat))
		}
		if _, lookupErr := s.blueprintSvc.GetDeviceModelForTenant(ctx, tenantID, modelUUID); lookupErr != nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
		}
		endpoint.DeviceModelID = &modelUUID

		// Resolve TypeEUI from device model's default blueprint (model is authoritative)
		effectiveTypeEui, resolveErr := s.blueprintSvc.ResolveEffectiveTypeEUI(ctx, tenantID, modelUUID)
		if resolveErr != nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed))
		}
		endpoint.TypeEUI = effectiveTypeEui
	}

	// Create endpoint in storage
	created, err := s.endpointSvc.Create(ctx, endpoint)
	if err != nil {
		// Check for duplicate error (global EUI uniqueness via migration 000080)
		if errors.Is(err, kcerrors.ErrDuplicate) {
			s.log.WarnContext(ctx, grpcerrors.LogEndpointAlreadyExists, "eui", req.Endpoint.EpEui, "tenant_id", tenantID)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointExists),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists))
		}
		s.log.ErrorContext(ctx, grpcerrors.LogEndpointCreateFailed, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateEndpointFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{"epEui": req.Endpoint.EpEui})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeEndpointCreated,
			Category:    models.EventCategoryEndpoint,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleEndpointCreated,
			Description: fmt.Sprintf(models.EventDescriptionEndpointCreated, req.Endpoint.EpEui),
			SourceType:  models.SourceTypeEndpoint,
			SourceName:  req.Endpoint.EpEui,
			EndpointID:  &created.ID,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	// Convert back to proto
	return s.endpointToProto(created), nil
}

// GetEndPoint retrieves an endpoint by EUI
func (s *CoreService) GetEndPoint(ctx context.Context, req *pb.GetEndPointRequest) (*pb.EndPoint, error) {
	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Getting endpoint", "eui", req.EpEui, "tenant_id", tenantID)

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
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get endpoint", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}

	return s.endpointToProto(endpoint), nil
}

// FieldMask path constants for UpdateBaseStation partial updates
const (
	bsFieldMaskName        = "name"
	bsFieldMaskDescription = "description"
	bsFieldMaskLatitude    = "latitude"
	bsFieldMaskLongitude   = "longitude"
	bsFieldMaskAltitude    = "altitude"
)

// FieldMask path constants for UpdateEndPoint partial updates
const (
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

	// EP Class values per MIOTY specification
	epClassBidirectional  = "A"
	epClassUnidirectional = "Z"
)

// fieldInMask delegates to the shared package for cross-service use.
func fieldInMask(mask *fieldmaskpb.FieldMask, field string) bool {
	return grpcerrors.FieldInMask(mask, field)
}

// UpdateEndPoint updates an endpoint
func (s *CoreService) UpdateEndPoint(ctx context.Context, req *pb.UpdateEndPointRequest) (*pb.EndPoint, error) {
	// Get authenticated tenant ID from context
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

	// Now safe to log after validation
	s.log.InfoContext(ctx, "Updating endpoint", "eui", req.Endpoint.EpEui, "tenant_id", tenantID)

	eui := models.EUIFromString(req.Endpoint.EpEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Validate tenant ID in request matches authenticated tenant (if provided)
	tenantStr := strconv.FormatInt(tenantID, 10)
	if req.Endpoint.TenantId != "" && req.Endpoint.TenantId != tenantStr {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Fetch existing endpoint to preserve fields not in update request
	endpoint, err := s.endpointSvc.GetByEUI(ctx, eui[:], tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get endpoint for update", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}

	// Selectively overwrite provided fields (preserve existing values otherwise)
	if req.Endpoint.Name != "" {
		endpoint.Name = req.Endpoint.Name
	}
	if req.Endpoint.Description != "" {
		endpoint.Description = req.Endpoint.Description
	}
	// Status takes priority over AttachStatus for frontend compatibility
	if req.Endpoint.Status != "" {
		endpoint.EpStatus = req.Endpoint.Status
	} else if req.Endpoint.AttachStatus != "" {
		endpoint.EpStatus = req.Endpoint.AttachStatus
	}
	if len(req.Endpoint.NwkSnKey) > 0 {
		endpoint.NwkSnKey = req.Endpoint.NwkSnKey
	}
	if len(req.Endpoint.AppKey) > 0 {
		endpoint.AppKey = req.Endpoint.AppKey
	}
	if len(req.Endpoint.Tags) > 0 {
		endpoint.Tags = req.Endpoint.Tags
	}
	if req.Endpoint.EpClass != "" {
		endpoint.EPClass = req.Endpoint.EpClass
		endpoint.Bidi = (req.Endpoint.EpClass == epClassBidirectional)
	}

	// FieldMask is required for partial updates
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMaskRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMaskRequired))
	}

	// MIOTY configuration fields - update only if explicitly included in mask
	// Note: Proto attach_status is UNMAPPED per spec - no model field exists
	if fieldInMask(mask, fieldMaskShAddr) {
		if req.Endpoint.ShAddr > math.MaxUint16 {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenShortAddressOverflow),
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

	// Boolean fields: update only when mask explicitly includes them
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

	// TypeEUI: update with explicit clear support when included in mask
	if fieldInMask(mask, fieldMaskTypeEUI) {
		if len(req.Endpoint.TypeEui) == 0 {
			endpoint.TypeEUI = nil
		} else if len(req.Endpoint.TypeEui) != 8 {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTypeEUIFormat))
		} else {
			var parsed models.EUI
			copy(parsed[:], req.Endpoint.TypeEui)
			endpoint.TypeEUI = &parsed
		}
	}

	// Device model association for blueprint decoding
	if fieldInMask(mask, fieldMaskDeviceModelID) {
		if req.Endpoint.DeviceModelId == "" {
			endpoint.DeviceModelID = nil
		} else {
			if s.blueprintSvc == nil {
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
			}
			modelUUID, parseErr := uuid.Parse(req.Endpoint.DeviceModelId)
			if parseErr != nil {
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidDeviceModelIDFormat),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidDeviceModelIDFormat))
			}
			if _, lookupErr := s.blueprintSvc.GetDeviceModelForTenant(ctx, tenantID, modelUUID); lookupErr != nil {
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
			}
			endpoint.DeviceModelID = &modelUUID

			// Resolve TypeEUI from device model's default blueprint (model is authoritative)
			effectiveTypeEui, resolveErr := s.blueprintSvc.ResolveEffectiveTypeEUI(ctx, tenantID, modelUUID)
			if resolveErr != nil {
				return nil, status.Error(
					grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed))
			}
			endpoint.TypeEUI = effectiveTypeEui
		}
	}

	// Post-field normalization: enforce model→typeEui precedence.
	// Covers updates that touch type_eui without touching device_model_id —
	// existing DeviceModelID still governs TypeEUI.
	if endpoint.DeviceModelID != nil {
		if s.blueprintSvc == nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
		}
		effectiveTypeEui, resolveErr := s.blueprintSvc.ResolveEffectiveTypeEUI(ctx, tenantID, *endpoint.DeviceModelID)
		if resolveErr != nil {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed))
		}
		endpoint.TypeEUI = effectiveTypeEui
	}

	// EUI change: atomic cascade + field update when new_ep_eui is set
	euiChanged := false
	if req.NewEpEui != "" {
		newEui := models.EUIFromString(req.NewEpEui)
		if newEui == (models.EUI{}) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
		}
		if newEui != endpoint.EUI {
			euiChanged = true
			// Uniqueness check (optimistic fast-path; repo also catches constraint race)
			if checkErr := s.endpointSvc.CheckEUIGloballyUnique(ctx, newEui[:]); checkErr != nil {
				if errors.Is(checkErr, storage.ErrAlreadyExists) {
					return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointExists),
						grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists))
				}
				s.log.ErrorContext(ctx, "Failed to check EUI uniqueness", "error", checkErr)
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateEndpointEUIFailed),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointEUIFailed))
			}
			oldEui := endpoint.EUI
			endpoint.EUI = newEui
			updated, updateErr := s.endpointSvc.UpdateWithEUI(ctx, tenantID, oldEui[:], endpoint)
			if updateErr != nil {
				if errors.Is(updateErr, storage.ErrAlreadyExists) {
					return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointExists),
						grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists))
				}
				if errors.Is(updateErr, storage.ErrForeignKeyViolation) {
					return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
						grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
				}
				if errors.Is(updateErr, storage.ErrNwkKeyLength) {
					return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNwkSnKeyLength),
						grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength))
				}
				if errors.Is(updateErr, storage.ErrAppKeyLength) {
					return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAppKeyLength),
						grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength))
				}
				s.log.ErrorContext(ctx, "Failed to update endpoint EUI", "error", updateErr)
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateEndpointEUIFailed),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointEUIFailed))
			}
			endpoint = updated
		}
	}

	// Persist non-EUI field changes (UpdateWithEUI already handles both atomically)
	if !euiChanged {
		updated, updateErr := s.endpointSvc.Update(ctx, endpoint)
		if errors.Is(updateErr, storage.ErrNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
		}
		if errors.Is(updateErr, storage.ErrAlreadyExists) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointExists),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists))
		}
		if errors.Is(updateErr, storage.ErrForeignKeyViolation) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeviceModelNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound))
		}
		if errors.Is(updateErr, storage.ErrNwkKeyLength) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNwkSnKeyLength),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength))
		}
		if errors.Is(updateErr, storage.ErrAppKeyLength) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAppKeyLength),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength))
		}
		if updateErr != nil {
			s.log.ErrorContext(ctx, "Failed to update endpoint", "error", updateErr)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateEndpointFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointFailed))
		}
		endpoint = updated
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{"epEui": req.Endpoint.EpEui})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeEndpointUpdated,
			Category:    models.EventCategoryEndpoint,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleEndpointUpdated,
			Description: fmt.Sprintf(models.EventDescriptionEndpointUpdated, req.Endpoint.EpEui),
			SourceType:  models.SourceTypeEndpoint,
			SourceName:  req.Endpoint.EpEui,
			EndpointID:  &endpoint.ID,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return s.endpointToProto(endpoint), nil
}

// DeleteEndPoint deletes an endpoint
func (s *CoreService) DeleteEndPoint(ctx context.Context, req *pb.DeleteEndPointRequest) (*emptypb.Empty, error) {
	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Deleting endpoint", "eui", req.EpEui, "tenant_id", tenantID)

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

	// Prefetch endpoint ID for event correlation before delete
	var epID *int64
	if s.eventWriter != nil {
		if ep, lookupErr := s.endpointSvc.GetByEUI(ctx, eui[:], tenantID); lookupErr == nil && ep != nil {
			epID = &ep.ID
		}
	}

	// Delete endpoint from storage
	err = s.endpointSvc.Delete(ctx, eui[:], tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to delete endpoint", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteEndpointFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{"epEui": req.EpEui})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeEndpointDeleted,
			Category:    models.EventCategoryEndpoint,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleEndpointDeleted,
			Description: fmt.Sprintf(models.EventDescriptionEndpointDeleted, req.EpEui),
			SourceType:  models.SourceTypeEndpoint,
			SourceName:  req.EpEui,
			EndpointID:  epID,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &emptypb.Empty{}, nil
}

// ListEndPoints lists endpoints for a tenant
func (s *CoreService) ListEndPoints(ctx context.Context, req *pb.ListEndPointsRequest) (*pb.ListEndPointsResponse, error) {
	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Listing endpoints", "tenant_id", tenantID, "page_size", req.PageSize, "page_token", req.PageToken)

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

	// List endpoints from storage (use tenant from context)
	endpoints, err := s.endpointSvc.List(ctx, tenantID, int(pageSize), offset)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to list endpoints", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListEndpointsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListEndpointsFailed))
	}

	// Convert to proto
	pbEndPoints := make([]*pb.EndPoint, len(endpoints))
	for i, endpoint := range endpoints {
		pbEndPoints[i] = s.endpointToProto(endpoint)
	}

	// Generate next page token
	nextPageToken := ""
	if len(endpoints) == int(pageSize) {
		nextPageToken = fmt.Sprintf("%d", offset+int(pageSize))
	}

	return &pb.ListEndPointsResponse{
		Endpoints:     pbEndPoints,
		NextPageToken: nextPageToken,
	}, nil
}

// AttachEndPoint initiates attach propagation for an endpoint.
func (s *CoreService) AttachEndPoint(ctx context.Context, req *pb.AttachEndPointRequest) (*pb.AttachEndPointResponse, error) {
	// Validate service is configured
	if s.endpointAttachmentSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	s.log.InfoContext(ctx, "Initiating endpoint attach", "ep_eui", req.EpEui, "tenant_id", tenantID)

	// Delegate to attachment service
	result, err := s.endpointAttachmentSvc.AttachEndPoint(ctx, req.EpEui, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to initiate attach", "ep_eui", req.EpEui, "error", err)
		// Map service errors to gRPC errors
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAttachFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAttachFailed))
	}

	return &pb.AttachEndPointResponse{
		OperationId: result.OperationID,
		Status:      result.Status,
	}, nil
}

// DetachEndPoint initiates detach propagation for an endpoint.
func (s *CoreService) DetachEndPoint(ctx context.Context, req *pb.DetachEndPointRequest) (*pb.DetachEndPointResponse, error) {
	// Validate service is configured
	if s.endpointAttachmentSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Extract tenant from authenticated context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	s.log.InfoContext(ctx, "Initiating endpoint detach", "ep_eui", req.EpEui, "tenant_id", tenantID)

	// Delegate to attachment service
	result, err := s.endpointAttachmentSvc.DetachEndPoint(ctx, req.EpEui, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to initiate detach", "ep_eui", req.EpEui, "error", err)
		// Map service errors to gRPC errors
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDetachFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDetachFailed))
	}

	return &pb.DetachEndPointResponse{
		OperationId: result.OperationID,
		Status:      result.Status,
	}, nil
}

// BaseStation operations

// CreateBaseStation creates a new base station
func (s *CoreService) CreateBaseStation(ctx context.Context, req *pb.CreateBaseStationRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request - nil checks BEFORE any access to req.Basestation.*
	if req.Basestation == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationRequired))
	}
	if req.Basestation.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	s.log.InfoContext(ctx, grpcerrors.LogBaseStationCreating, "eui", req.Basestation.BsEui, "tenant_id", tenantID)

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.Basestation.TenantId != "" && req.Basestation.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	eui := models.EUIFromString(req.Basestation.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	var tagsJSON *string
	if len(req.Basestation.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Basestation.Tags)
		tagsStr := string(tagsBytes)
		tagsJSON = &tagsStr
	}

	// Extract coordinate wrappers (nil = absent, non-nil = explicit value including 0.0)
	var lat, lon, alt *float64
	if req.Basestation.Latitude != nil {
		v := req.Basestation.Latitude.GetValue()
		lat = &v
	}
	if req.Basestation.Longitude != nil {
		v := req.Basestation.Longitude.GetValue()
		lon = &v
	}
	if req.Basestation.Altitude != nil {
		v := req.Basestation.Altitude.GetValue()
		alt = &v
	}

	// Validate lat/lon pair: both present or both absent
	if (lat != nil) != (lon != nil) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLatLonPairRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLatLonPairRequired))
	}

	var desc *string
	if req.Basestation.Description != "" {
		desc = &req.Basestation.Description
	}

	// Set location source and timestamp when coordinates are provided
	var locationSource *string
	var locationUpdatedAt *time.Time
	if lat != nil && lon != nil {
		src := models.LocationSourceManual
		locationSource = &src
		now := time.Now()
		locationUpdatedAt = &now
	}

	// Convert proto to canonical model using authenticated tenant ID
	baseStation := &models.BaseStation{
		EUI:               eui,
		TenantID:          tenantID,
		Name:              req.Basestation.Name,
		Description:       desc,
		Latitude:          lat,
		Longitude:         lon,
		Altitude:          alt,
		LocationSource:    locationSource,
		LocationUpdatedAt: locationUpdatedAt,
		Tags:              tagsJSON,
	}

	// Create base station in storage
	created, err := s.basestationSvc.Create(ctx, baseStation)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to create base station", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateBaseStationFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{"bsEui": req.Basestation.BsEui})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:      strconv.FormatInt(tenantID, 10),
			EventType:     models.EventTypeBSRegistered,
			Category:      models.EventCategoryBaseStation,
			Severity:      models.EventSeverityInfo,
			Title:         models.EventTitleBSRegistered,
			Description:   fmt.Sprintf(models.EventDescriptionBSRegistered, req.Basestation.BsEui),
			SourceType:    models.SourceTypeBaseStation,
			SourceName:    req.Basestation.BsEui,
			BasestationID: &created.ID,
			Details:       detailsJSON,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})
	}

	return s.baseStationToProto(created), nil
}

// GetBaseStation retrieves a base station by EUI
func (s *CoreService) GetBaseStation(ctx context.Context, req *pb.GetBaseStationRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Getting base station", "eui", req.BsEui, "tenant_id", tenantID)

	// Validate request
	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	eui := models.EUIFromString(req.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Get base station from storage using authenticated tenant ID
	baseStation, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get base station", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationFailed))
	}

	return s.baseStationToProto(baseStation), nil
}

// UpdateBaseStation updates a base station
func (s *CoreService) UpdateBaseStation(ctx context.Context, req *pb.UpdateBaseStationRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Basestation == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationRequired))
	}
	if req.Basestation.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	s.log.InfoContext(ctx, "Updating base station", "eui", req.Basestation.BsEui, "tenant_id", tenantID)

	// Validate EUI format
	eui := models.EUIFromString(req.Basestation.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.Basestation.TenantId != "" && req.Basestation.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Fetch existing base station to merge fields
	existing, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to fetch base station for update", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBaseStationFailed))
	}

	// FieldMask is required for partial updates
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMaskRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMaskRequired))
	}

	// Start with existing values, then overlay based on mask
	baseStation := existing

	// Name
	if fieldInMask(mask, bsFieldMaskName) {
		if req.Basestation.Name != "" {
			baseStation.Name = req.Basestation.Name
		}
	}

	// Description
	if fieldInMask(mask, bsFieldMaskDescription) {
		if req.Basestation.Description != "" {
			desc := req.Basestation.Description
			baseStation.Description = &desc
		} else {
			baseStation.Description = nil
		}
	}

	// Tags
	if len(req.Basestation.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Basestation.Tags)
		tagsStr := string(tagsBytes)
		baseStation.Tags = &tagsStr
	}

	// Coordinates with FieldMask support
	latMasked := fieldInMask(mask, bsFieldMaskLatitude)
	lonMasked := fieldInMask(mask, bsFieldMaskLongitude)
	altMasked := fieldInMask(mask, bsFieldMaskAltitude)

	// If one of lat/lon is masked, both must be
	if latMasked != lonMasked {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLatLonPairRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLatLonPairRequired))
	}

	if latMasked {
		if req.Basestation.Latitude != nil {
			v := req.Basestation.Latitude.GetValue()
			baseStation.Latitude = &v
		} else {
			baseStation.Latitude = nil
		}
	}
	if lonMasked {
		if req.Basestation.Longitude != nil {
			v := req.Basestation.Longitude.GetValue()
			baseStation.Longitude = &v
		} else {
			baseStation.Longitude = nil
		}
	}
	if altMasked {
		if req.Basestation.Altitude != nil {
			v := req.Basestation.Altitude.GetValue()
			baseStation.Altitude = &v
		} else {
			baseStation.Altitude = nil
		}
	}

	// Update location source/timestamp
	if latMasked && lonMasked {
		if baseStation.Latitude != nil && baseStation.Longitude != nil {
			src := models.LocationSourceManual
			baseStation.LocationSource = &src
			now := time.Now()
			baseStation.LocationUpdatedAt = &now
		} else {
			baseStation.LocationSource = nil
			baseStation.LocationUpdatedAt = nil
		}
	}

	// Update base station in storage
	updated, err := s.basestationSvc.Update(ctx, baseStation)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to update base station", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBaseStationFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{"bsEui": req.Basestation.BsEui})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:      strconv.FormatInt(tenantID, 10),
			EventType:     models.EventTypeBSUpdated,
			Category:      models.EventCategoryBaseStation,
			Severity:      models.EventSeverityInfo,
			Title:         models.EventTitleBSUpdated,
			Description:   fmt.Sprintf(models.EventDescriptionBSUpdated, req.Basestation.BsEui),
			SourceType:    models.SourceTypeBaseStation,
			SourceName:    req.Basestation.BsEui,
			BasestationID: &updated.ID,
			Details:       detailsJSON,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})
	}

	return s.baseStationToProto(updated), nil
}

// UpdateBaseStationEui updates the EUI of a base station with cascade to all dependent tables.
func (s *CoreService) UpdateBaseStationEui(ctx context.Context, req *pb.UpdateBaseStationEuiRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Updating base station EUI", "old_eui", req.BsEui, "new_eui", req.NewBsEui, "tenant_id", tenantID)

	// Validate request - current EUI is required
	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	// Validate request - new EUI is required
	if req.NewBsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNewBaseStationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNewBaseStationEUIRequired))
	}

	// Parse current EUI (accepts dashed or non-dashed format)
	oldEui := models.EUIFromString(req.BsEui)
	if oldEui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Parse new EUI (accepts dashed or non-dashed format)
	newEui := models.EUIFromString(req.NewBsEui)
	if newEui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Update EUI in storage with transactional cascade
	updated, err := s.basestationSvc.UpdateEUI(ctx, tenantID, oldEui[:], newEui[:])
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err == storage.ErrAlreadyExists {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationEUIExists),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationEUIExists))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to update base station EUI", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateBaseStationEUIFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBaseStationEUIFailed))
	}

	// Close active BSSCI session for the old EUI (if any)
	// EUI change invalidates any live session - base station must reconnect with new identity
	oldEuiUint64 := binary.BigEndian.Uint64(oldEui[:])
	if s.bssciSessionCloser != nil {
		if err := s.bssciSessionCloser.CloseSessionByEUI(ctx, oldEuiUint64); err != nil {
			s.log.WarnContext(ctx, "Failed to close BSSCI session for old EUI", "error", err, "eui", oldEuiUint64)
		}
	}

	// Record offline event for the new EUI (uses [8]byte per basestation.EventRecorder interface)
	if s.bsEventRecorder != nil {
		eventData := map[string]interface{}{
			"reason":            "EUI changed",
			bssci.EventKeyBsEui: fmt.Sprintf("%016x", oldEuiUint64),
		}
		if err := s.bsEventRecorder.RecordEvent(ctx, newEui, models.EventTypeBaseStationOffline, eventData); err != nil {
			s.log.WarnContext(ctx, "Failed to record offline event", "error", err)
		}
	}

	return s.baseStationToProto(updated), nil
}

// DeleteBaseStation deletes a base station
func (s *CoreService) DeleteBaseStation(ctx context.Context, req *pb.DeleteBaseStationRequest) (*emptypb.Empty, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Deleting base station", "eui", req.BsEui, "tenant_id", tenantID)

	// Validate request
	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	eui := models.EUIFromString(req.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Prefetch base station ID for event correlation before delete
	var bsID *int64
	if s.eventWriter != nil {
		if bs, lookupErr := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID); lookupErr == nil && bs != nil {
			bsID = &bs.ID
		}
	}

	// Delete base station from storage using authenticated tenant ID
	err = s.basestationSvc.Delete(ctx, eui[:], tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to delete base station", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteBaseStationFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{"bsEui": req.BsEui})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:      strconv.FormatInt(tenantID, 10),
			EventType:     models.EventTypeBSDeregistered,
			Category:      models.EventCategoryBaseStation,
			Severity:      models.EventSeverityInfo,
			Title:         models.EventTitleBSDeregistered,
			Description:   fmt.Sprintf(models.EventDescriptionBSDeleted, req.BsEui),
			SourceType:    models.SourceTypeBaseStation,
			SourceName:    req.BsEui,
			BasestationID: bsID,
			Details:       detailsJSON,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})
	}

	return &emptypb.Empty{}, nil
}

// ListBaseStations lists base stations for a tenant
func (s *CoreService) ListBaseStations(ctx context.Context, req *pb.ListBaseStationsRequest) (*pb.ListBaseStationsResponse, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Listing base stations", "tenant_id", tenantID, "page_size", req.PageSize, "page_token", req.PageToken)

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

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

	// List base stations from storage using authenticated tenant ID
	baseStations, err := s.basestationSvc.List(ctx, tenantID, int(pageSize), offset)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to list base stations", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListBaseStationsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListBaseStationsFailed))
	}

	// Convert to proto
	pbBaseStations := make([]*pb.BaseStation, len(baseStations))
	for i, baseStation := range baseStations {
		pbBaseStations[i] = s.baseStationToProto(baseStation)
	}

	// Generate next page token
	nextPageToken := ""
	if len(baseStations) == int(pageSize) {
		nextPageToken = fmt.Sprintf("%d", offset+int(pageSize))
	}

	return &pb.ListBaseStationsResponse{
		Basestations:  pbBaseStations,
		NextPageToken: nextPageToken,
	}, nil
}

// GetBaseStationStats retrieves aggregated message statistics for a base station.
func (s *CoreService) GetBaseStationStats(ctx context.Context, req *pb.GetBaseStationStatsRequest) (*pb.GetBaseStationStatsResponse, error) {
	// Get tenant ID from context (required by fail-closed org resolver interceptor)
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
	}

	s.log.InfoContext(ctx, "Getting base station stats", "bs_eui", req.BsEui, "tenant_id", tenantID)

	// Validate request
	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	eui := models.EUIFromString(req.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Get message stats from storage
	stats, err := s.statsStore.GetBaseStationMessageStats(ctx, tenantID, eui[:],
		timestampToTime(req.StartTime), timestampToTime(req.EndTime))
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get base station stats", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationStatsFailed))
	}

	// Get endpoint counts
	endpointCounts, err := s.statsStore.GetBaseStationEndpointCounts(ctx, tenantID, eui[:],
		timestampToTime(req.StartTime), timestampToTime(req.EndTime))
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get endpoint counts", "error", err)
		// Don't fail the entire request, just log the error
		endpointCounts = make(map[string]int64)
	}

	// Get last seen timestamp
	lastSeen, err := s.statsStore.GetBaseStationLastSeen(ctx, tenantID, eui[:])
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get last seen timestamp", "error", err)
	}

	// Get base station status from storage
	bs, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	var bsStatus string
	if err == nil && bs != nil {
		if bs.IsOnline {
			bsStatus = "online"
		} else {
			bsStatus = grpcerrors.StatusOffline
		}
	} else {
		bsStatus = grpcerrors.StatusOffline
	}

	// Convert to protobuf response
	response := &pb.GetBaseStationStatsResponse{
		BsEui:                 req.BsEui,
		TotalMessages:         stats.TotalMessages,
		TotalEndpoints:        stats.TotalEndpoints,
		MessagesToday:         stats.MessagesToday,
		MessagesThisWeek:      stats.MessagesThisWeek,
		MessagesThisMonth:     stats.MessagesThisMonth,
		AvgRssi:               stats.AvgRSSI,
		AvgSnr:                stats.AvgSNR,
		EndpointMessageCounts: endpointCounts,
		Status:                bsStatus,
	}

	// Add timestamps if available
	if stats.LastMessageAt != nil {
		response.LastMessageAt = timestamppb.New(*stats.LastMessageAt)
	}
	if lastSeen != nil {
		response.LastSeenAt = timestamppb.New(*lastSeen)
	}

	return response, nil
}

// Message operations

// SendDownlink sends a downlink message to an endpoint.
// When scaciQueuer is set, delegates to SCACI handler core for single-source processing.
// Otherwise falls back to direct scheduler mode (community compatibility).
func (s *CoreService) SendDownlink(ctx context.Context, req *pb.SendDownlinkRequest) (*pb.SendDownlinkResponse, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Sending downlink", "ep_eui", req.EpEui, "tenant_id", tenantID, "payload_count", len(req.Payloads))

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Validate request
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	// Validate payload count based on counter-dependency mode (BSSCI §3.12.1 + §5.12)
	// BSSCI §5.12: Allow empty payloads for pure acknowledgement downlinks in non-counter-dependent mode
	if req.CntDepend {
		// Counter-dependent mode requires at least one payload
		if len(req.Payloads) == 0 {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkPayloadRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkPayloadRequired))
		}
		if len(req.PacketCnt) != len(req.Payloads) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkFormatInvalid),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkFormatInvalid))
		}
	} else {
		// Non-counter-dependent mode allows 0 (ACK-only) or 1 (normal) payload
		if len(req.Payloads) > 1 {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkPayloadTooLarge),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkPayloadTooLarge))
		}
	}

	// Validate individual payload sizes (MIOTY radio protocol §4.3.2)
	for i, payload := range req.Payloads {
		if len(payload) > mioty.MaxDLUserDataBytes {
			s.log.WarnContext(ctx, "Downlink payload exceeds maximum size",
				"entry", i,
				"size", len(payload),
				"max", mioty.MaxDLUserDataBytes)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkPayloadTooLarge),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkPayloadTooLarge))
		}
	}

	// SCACI §3.10 enforcement: All gRPC downlinks route through SCACI handler core.
	// SCACI is the single-source for downlink processing.
	// Fail-fast check for scaciQueuer is enforced at startup in main.go.
	return s.sendDownlinkViaSCACI(ctx, req, tenantID)
}

// sendDownlinkViaSCACI delegates downlink queueing to SCACI handler core.
// This ensures gRPC path uses the same processing as socket path.
func (s *CoreService) sendDownlinkViaSCACI(ctx context.Context, req *pb.SendDownlinkRequest, tenantID int64) (*pb.SendDownlinkResponse, error) {
	// Parse endpoint EUI from hex string to uint64
	epEui, err := strconv.ParseUint(req.EpEui, 16, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEUIFormat))
	}

	// Validate Format fits in uint8
	if req.Format > math.MaxUint8 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkFormatInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkFormatInvalid))
	}

	// Convert PacketCnt from []int64 to []uint32 (MIOTY protocol requirement)
	var packetCnt []uint32
	if len(req.PacketCnt) > 0 {
		packetCnt = make([]uint32, len(req.PacketCnt))
		for i, cnt := range req.PacketCnt {
			if cnt < 0 || cnt > 4294967295 {
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkFormatInvalid),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkFormatInvalid))
			}
			packetCnt[i] = uint32(cnt)
		}
	}

	// Build MIOTY DLDataQueue request
	prio := req.Priority
	format := uint8(req.Format)

	// Generate unique QueId for gRPC path (SCACI §3.10).
	// The proto SendDownlinkRequest does not expose QueId - it's always generated server-side.
	// The duplicate guard in processDLDataQueueCore remains the arbiter for collision detection.
	queId := uint64(time.Now().UnixNano()) //nolint:gosec // G115: UnixNano() always positive

	dlReq := &mioty.DLDataQueue{
		EpEui:        epEui,
		QueId:        queId, // SCACI §3.10: Required for processDLDataQueueCore validation
		UserData:     req.Payloads,
		Prio:         &prio,
		CntDepend:    req.CntDepend,
		PacketCnt:    packetCnt,
		Format:       &format,
		ResponseExp:  &req.ResponseExp,
		ResponsePrio: &req.ResponsePrio,
		DlWindReq:    &req.DlWindReq,
		ExpOnly:      &req.ExpOnly,
		DlRxStatQry:  &req.DlRxStatQry, // SCACI §3.10: Propagate DL RX status query flag
	}

	// Extract organization ID from context (populated by fail-closed interceptor)
	var orgID *uuid.UUID
	if orgUUID, err := pkgcontext.GetOrganizationID(ctx); err == nil && orgUUID != uuid.Nil {
		orgID = &orgUUID
	}

	// Delegate to SCACI handler core (same path as socket dlDataQue)
	result, err := s.scaciQueuer.QueueDownlinkInternal(ctx, tenantID, orgID, dlReq)
	if err != nil {
		s.log.ErrorContext(ctx, "SCACI QueueDownlinkInternal failed",
			"error", err,
			"epEui", req.EpEui,
			"tenantId", tenantID)
		return nil, mapSCACIErrorToGRPC(err)
	}

	s.log.InfoContext(ctx, "Downlink queued via SCACI",
		"queueId", result.QueID,
		"bsEui", fmt.Sprintf("%016X", result.BsEui),
		"opId", result.OpID,
		"tenantId", tenantID,
		"epEui", req.EpEui)

	return &pb.SendDownlinkResponse{
		Id:     fmt.Sprintf("%d", result.QueID),
		Status: result.Status,
	}, nil
}

// mapSCACIErrorToGRPC converts SCACI error tokens to gRPC status codes.
// Uses error tokens from errors_catalog.go for catalog-backed responses.
func mapSCACIErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// InvalidArgument (400-class client errors)
	scaciToGRPCInvalid := map[string]string{
		scaci.ErrDLPayloadTooLarge:        grpcerrors.ErrTokenDownlinkPayloadTooLarge,
		scaci.ErrCntDependMismatch:        grpcerrors.ErrTokenScaciCntDependMismatch,
		scaci.ErrCntDependPacketCntOmit:   grpcerrors.ErrTokenScaciCntDependPacketCntOmit,
		scaci.ErrNonCntDependMultiPayload: grpcerrors.ErrTokenScaciNonCntDependMultiPayload,
		scaci.ErrQueueIDOutOfRange:        grpcerrors.ErrTokenScaciQueueIDOutOfRange,
		scaci.ErrFailedPersistDownlink:    grpcerrors.ErrTokenScaciFailedPersistDownlink,
	}
	for scaciToken, grpcToken := range scaciToGRPCInvalid {
		if strings.Contains(errStr, scaciToken) {
			return status.Error(grpcerrors.GetGRPCCode(grpcToken),
				grpcerrors.ResolveErrorMessage(grpcToken))
		}
	}

	// AlreadyExists (duplicate queue ID)
	if strings.Contains(errStr, scaci.ErrQueIDExists) {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciQueueIDExists),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciQueueIDExists))
	}

	// Unavailable (temporary infrastructure issues)
	scaciToGRPCUnavail := map[string]string{
		scaci.ErrSchedulerUnavailable:   grpcerrors.ErrTokenScaciSchedulerUnavailable,
		scaci.ErrBaseStationUnavailable: grpcerrors.ErrTokenScaciBaseStationUnavailable,
	}
	for scaciToken, grpcToken := range scaciToGRPCUnavail {
		if strings.Contains(errStr, scaciToken) {
			return status.Error(grpcerrors.GetGRPCCode(grpcToken),
				grpcerrors.ResolveErrorMessage(grpcToken))
		}
	}

	// NotFound (resource doesn't exist)
	scaciToGRPCNotFound := map[string]string{
		scaci.ErrEndpointNotFound: grpcerrors.ErrTokenScaciEndpointNotFound,
		scaci.ErrDownlinkNotFound: grpcerrors.ErrTokenScaciDownlinkNotFound,
	}
	for scaciToken, grpcToken := range scaciToGRPCNotFound {
		if strings.Contains(errStr, scaciToken) {
			return status.Error(grpcerrors.GetGRPCCode(grpcToken),
				grpcerrors.ResolveErrorMessage(grpcToken))
		}
	}

	// Default: Internal for unknown SCACI errors
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenScaciOperationFailed),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenScaciOperationFailed))
}

// ListDownlinkQueue lists the downlink queue for an endpoint
func (s *CoreService) ListDownlinkQueue(ctx context.Context, req *pb.ListDownlinkQueueRequest) (*pb.ListDownlinkQueueResponse, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Listing downlink queue", "tenant_id", tenantID, "ep_eui", req.EpEui, "page_size", req.PageSize)

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Default page size
	pageSize := clampHighVolumePageSize(req.PageSize, 50)

	// Query downlink messages from storage (empty epEui returns all for tenant)
	var messages []*storage.DownlinkMessage
	messages, err = s.messageSvc.GetDownlinkQueue(ctx, req.EpEui, strconv.FormatInt(tenantID, 10))

	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get downlink queue", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkQueueListFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkQueueListFailed))
	}

	// Apply pagination manually
	startIdx := 0
	if req.PageToken != "" {
		if idx, err := strconv.Atoi(req.PageToken); err == nil {
			startIdx = idx
		}
	}

	endIdx := startIdx + int(pageSize)
	if endIdx > len(messages) {
		endIdx = len(messages)
	}

	// Slice messages for pagination
	var paginatedMessages []*storage.DownlinkMessage
	if startIdx < len(messages) {
		paginatedMessages = messages[startIdx:endIdx]
	}

	// Convert storage messages to proto messages
	var protoMessages []*pb.DownlinkMessage
	for _, msg := range paginatedMessages {
		// Map status from database to MIOTY-compliant status
		status := msg.Status
		if msg.Result != "" {
			status = msg.Result // Use MIOTY result field if available
		}

		protoMsg := &pb.DownlinkMessage{
			Id:           fmt.Sprintf("%d", msg.QueID),
			QueId:        msg.QueID,
			EpEui:        msg.EPEUI,
			TenantId:     msg.TenantID,
			Payload:      msg.Payload,
			Priority:     msg.Priority,
			Status:       status,
			Payloads:     msg.UserData,
			CntDepend:    msg.CntDepend,
			PacketCnt:    msg.PacketCntArray,
			Format:       uint32(msg.Format),
			ResponseExp:  msg.ResponseExp,
			ResponsePrio: msg.ResponsePrio,
			DlWindReq:    msg.DlWindReq,
			ExpOnly:      msg.ExpOnly,
			DlRxStatQry:  msg.DlRxStatQry,
			Result:       msg.Result,
			TxTime:       msg.TxTime,
			BsEui:        fmt.Sprintf("%016X", msg.BsEui),
		}

		// Add timestamps if available
		if !msg.CreatedAt.IsZero() {
			protoMsg.CreatedAt = timestamppb.New(msg.CreatedAt)
		}
		if msg.ScheduledAt != nil {
			protoMsg.ScheduledAt = timestamppb.New(*msg.ScheduledAt)
		}
		if msg.SentAt != nil {
			protoMsg.TransmittedAt = timestamppb.New(*msg.SentAt)
		}

		protoMessages = append(protoMessages, protoMsg)
	}

	// Calculate next page token
	nextPageToken := ""
	if endIdx < len(messages) {
		nextPageToken = strconv.Itoa(endIdx)
	}

	// Validate TotalCount fits in int32
	if len(messages) > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	return &pb.ListDownlinkQueueResponse{
		Messages:      protoMessages,
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(messages)), //nolint:gosec // G115: validated above on line 775
	}, nil
}

// RevokeDownlink revokes a queued downlink message
func (s *CoreService) RevokeDownlink(ctx context.Context, req *pb.RevokeDownlinkRequest) (*pb.RevokeDownlinkResponse, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Revoking downlink", "ep_eui", req.EpEui, "queue_id", req.QueueId, "tenant_id", tenantID)

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Validate request
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}
	if req.QueueId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenQueueIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenQueueIDRequired))
	}

	euiParsed := models.EUIFromString(req.EpEui)
	if euiParsed == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}
	epEui := euiParsed.ToUint64()

	// Parse queue ID
	queId, err := strconv.ParseInt(req.QueueId, 10, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidQueueIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidQueueIDFormat))
	}

	// Check if endpoint exists and belongs to tenant
	_, err = s.endpointSvc.GetByEUI(ctx, euiParsed[:], tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get endpoint", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetEndpointFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetEndpointFailed))
	}

	// Get connected base station sessions
	sessions := s.sessionDir.GetConnectedSessions()
	if len(sessions) == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNoBaseStationsConnected),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNoBaseStationsConnected))
	}

	// Validate queId is non-negative before casting
	if queId < 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenQueueIDNonNegative),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenQueueIDNonNegative))
	}

	// Get the downlink message from database to find the owning base station
	dl, err := s.messageSvc.GetDownlinkByQueueID(ctx, uint64(queId), strconv.FormatInt(tenantID, 10)) //nolint:gosec // G115: validated above on line 844
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkRevokeNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkRevokeNotFound))
		}
		s.log.ErrorContext(ctx, "Failed to get downlink by queue ID", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkGetFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkGetFailed))
	}

	// Check if a base station owns this queue item
	if dl.BsEui == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNoBaseStationOwnsQueue),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNoBaseStationOwnsQueue))
	}

	// Find the session for the specific base station that owns this queue item
	var selectedSessionID string
	for _, session := range sessions {
		if id, ok := session["id"].(string); ok {
			bsEui, _ := session["baseStationEui"].(uint64)
			handshakeComplete, _ := session["handshakeComplete"].(bool)
			if bsEui == dl.BsEui && handshakeComplete {
				selectedSessionID = id
				break
			}
		}
	}

	if selectedSessionID == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenHandshakeIncomplete),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenHandshakeIncomplete))
	}

	// Send revoke command via BSSCI protocol (dlDataRev operation)
	err = s.downlinkCmd.SendDLDataRevoke(selectedSessionID, epEui, uint64(queId)) //nolint:gosec // G115: validated above on line 844
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to revoke downlink via BSSCI", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkRevokeFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkRevokeFailed))
	}

	// Database update occurs when handleDLDataRevokeResponse receives BS confirmation
	// This ensures we only mark as revoked after BS confirms (line 586-596 in downlink_handlers.go)

	return &pb.RevokeDownlinkResponse{
		Status:  grpcerrors.StatusRevokeInitiated,
		Message: fmt.Sprintf(grpcerrors.MsgRevokeInitiated, req.QueueId),
	}, nil
}

// Helper methods

// endpointToProto converts a models endpoint to proto
func (s *CoreService) endpointToProto(endpoint *models.EndPoint) *pb.EndPoint {
	// Extract optional fields
	var shAddr uint32
	if endpoint.ShAddr != nil {
		shAddr = uint32(*endpoint.ShAddr)
	}
	var attachCnt uint32
	if endpoint.AttachCnt != nil {
		attachCnt = *endpoint.AttachCnt
	}
	var typeEui []byte
	if endpoint.TypeEUI != nil {
		typeEui = endpoint.TypeEUI[:]
	}

	// Derive activity status from LastSeenAt and configured window
	status := "inactive"
	if endpoint.LastSeenAt != nil && s.endpointActivityWindow > 0 {
		if time.Since(*endpoint.LastSeenAt) <= s.endpointActivityWindow {
			status = "active"
		}
	}

	result := &pb.EndPoint{
		EpEui:        endpoint.EUI.String(),
		TenantId:     strconv.FormatInt(endpoint.TenantID, 10),
		Name:         endpoint.Name,
		Description:  endpoint.Description,
		EpClass:      endpoint.EPClass,
		NwkSnKey:     endpoint.NwkSnKey,
		AppKey:       endpoint.AppKey,
		Status:       status,
		AttachStatus: endpoint.EpStatus,
		Tags:         endpoint.Tags,
		CreatedAt:    timestamppb.New(endpoint.CreatedAt),
		UpdatedAt:    timestamppb.New(endpoint.UpdatedAt),
		// MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
		ShAddr:        shAddr,
		DualChan:      endpoint.DualChan,
		Repetition:    endpoint.Repetition,
		WideCarrOff:   endpoint.WideCarrOff,
		LongBlkDist:   endpoint.LongBlkDist,
		AttachCnt:     attachCnt,
		PreAttach:     endpoint.PreAttach,
		LastPacketCnt: endpoint.LastPacketCnt,
		TypeEui:       typeEui,
		CarrierOffset: int32(endpoint.CarrierOffset), //nolint:gosec // G115: CarrierOffset per BSSCI spec is small integer
	}

	if endpoint.DeviceModelID != nil {
		result.DeviceModelId = endpoint.DeviceModelID.String()
	}

	if endpoint.LastSeenAt != nil {
		result.LastSeenAt = timestamppb.New(*endpoint.LastSeenAt)
	}

	return result
}

// baseStationToProto converts a models base station to proto
func (s *CoreService) baseStationToProto(baseStation *models.BaseStation) *pb.BaseStation {
	var desc string
	if baseStation.Description != nil {
		desc = *baseStation.Description
	}

	// Unmarshal tags from JSON string to map
	var tags map[string]string
	if baseStation.Tags != nil && *baseStation.Tags != "" {
		_ = json.Unmarshal([]byte(*baseStation.Tags), &tags)
	}

	// Determine status - use online status if available
	status := grpcerrors.StatusOffline
	if baseStation.IsOnline {
		status = "online"
	}

	result := &pb.BaseStation{
		BsEui:       baseStation.EUI.String(),
		TenantId:    strconv.FormatInt(baseStation.TenantID, 10),
		Name:        baseStation.Name,
		Description: desc,
		Status:      status,
		Tags:        tags,
		CreatedAt:   timestamppb.New(baseStation.CreatedAt),
		UpdatedAt:   timestamppb.New(baseStation.UpdatedAt),
	}

	// Coordinate wrappers: nil model → nil proto (absent), non-nil → wrapped value
	if baseStation.Latitude != nil {
		result.Latitude = wrapperspb.Double(*baseStation.Latitude)
	}
	if baseStation.Longitude != nil {
		result.Longitude = wrapperspb.Double(*baseStation.Longitude)
	}
	if baseStation.Altitude != nil {
		result.Altitude = wrapperspb.Double(*baseStation.Altitude)
	}

	// Location metadata
	if baseStation.LocationSource != nil {
		result.LocationSource = *baseStation.LocationSource
	}
	if baseStation.LocationUpdatedAt != nil {
		result.LocationUpdatedAt = timestamppb.New(*baseStation.LocationUpdatedAt)
	}

	if baseStation.LastSeenAt != nil {
		result.LastSeenAt = timestamppb.New(*baseStation.LastSeenAt)
	}

	// Map MIOTY status fields with wrapper types (preserves nil = "Not available")
	// BSSCI v1.0.0 §5.5.2 statusRsp fields
	if baseStation.SystemTime != nil {
		result.SystemTime = wrapperspb.Int64(*baseStation.SystemTime)
	}
	if baseStation.DutyCycle != nil {
		result.DutyCycle = wrapperspb.Double(*baseStation.DutyCycle)
	}
	if baseStation.UptimeSeconds != nil {
		result.UptimeSeconds = wrapperspb.Int64(*baseStation.UptimeSeconds)
	}
	if baseStation.TemperatureCelsius != nil {
		result.TemperatureCelsius = wrapperspb.Double(*baseStation.TemperatureCelsius)
	}
	if baseStation.CPULoad != nil {
		result.CpuLoad = wrapperspb.Double(*baseStation.CPULoad)
	}
	if baseStation.MemoryLoad != nil {
		result.MemoryLoad = wrapperspb.Double(*baseStation.MemoryLoad)
	}
	if baseStation.BSConfig.Valid {
		var configMap map[string]interface{}
		if err := json.Unmarshal(baseStation.BSConfig.Data, &configMap); err == nil {
			if structVal, err := structpb.NewStruct(configMap); err == nil {
				result.BsConfig = structVal
			}
		}
	}
	if baseStation.LastStatusAt != nil {
		result.LastStatusAt = timestamppb.New(*baseStation.LastStatusAt)
	}

	if baseStation.ServiceCenterURL != nil {
		result.ServiceCenterUrl = *baseStation.ServiceCenterURL
	}

	return result
}

// ============================================================================
// UL Data Transmit Operations (BSSCI 3.11)
// ============================================================================

// SendULTransmit initiates a Service Center to Base Station uplink data transmission
func (s *CoreService) SendULTransmit(ctx context.Context, req *pb.SendULTransmitRequest) (*pb.SendULTransmitResponse, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "SendULTransmit request received",
		"ep_eui", req.EpEui,
		"tenant_id", tenantID,
		"packet_cnt", req.PacketCnt,
		"user_data_len", len(req.UserData))

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Validate required fields
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	// Parse endpoint EUI from hex string to uint64
	epEui, err := strconv.ParseUint(req.EpEui, 16, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Parse optional base station EUI if provided
	var targetBsEui *uint64
	if req.BsEui != "" {
		bsEuiParsed := models.EUIFromString(req.BsEui)
		if bsEuiParsed == (models.EUI{}) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
		}
		parsedBsEui := bsEuiParsed.ToUint64()
		targetBsEui = &parsedBsEui

		// Verify base station ownership before scheduling
		// This prevents information disclosure when the base station is offline
		_, err = s.basestationSvc.GetByEUI(ctx, bsEuiParsed[:], tenantID)
		if err == storage.ErrNotFound {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
		}
		if err != nil {
			s.log.ErrorContext(ctx, "Failed to verify base station ownership", "error", err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationOwnershipFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationOwnershipFailed))
		}
	}

	// Validate network session key length
	if len(req.NwkSnKey) != 16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNwkSnKeyLength),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength))
	}

	// Note: Empty userData is allowed for control telegrams (BSSCI §3.11.1)

	// Validate short address
	if req.ShAddr == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenShortAddressZero),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenShortAddressZero))
	}

	// Select bidirectional base station using shared helper
	sessionID, bsEui, err := s.sessionDir.SelectBidirectionalSession(tenantID, targetBsEui)
	if err != nil {
		// Map BSSCI sentinel errors to appropriate gRPC status codes
		// Note: These use err.Error() since they are domain errors from BSSCI package
		switch {
		case errors.Is(err, bssci.ErrNoBidirectionalBaseStations):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNoBaseStationsConnected),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNoBaseStationsConnected))
		case errors.Is(err, bssci.ErrBaseStationUnavailable):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
		case errors.Is(err, bssci.ErrBaseStationTenantMismatch):
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenHandshakeIncomplete),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenHandshakeIncomplete))
		}
	}

	// Validate ShAddr fits in uint16
	if req.ShAddr > math.MaxUint16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenShortAddressOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenShortAddressOverflow))
	}

	// Validate Format fits in uint8
	if req.Format > math.MaxUint8 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFormatOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFormatOverflow))
	}

	// Call the BSSCI server method and get the real operation ID
	opId, err := s.ulTransmit.SendULDataTransmit(
		sessionID, epEui, req.NwkSnKey, uint16(req.ShAddr),
		req.PacketCnt, req.UserData, req.Profile, uint8(req.Format))

	if err != nil {
		s.log.ErrorContext(ctx, "Failed to send UL transmit", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenULTransmitFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenULTransmitFailed))
	}

	// Log successful queuing
	s.log.InfoContext(ctx, "UL transmit queued successfully",
		"tenant_id", tenantID,
		"ep_eui", req.EpEui,
		"bs_eui", fmt.Sprintf("%016X", bsEui),
		"packet_cnt", req.PacketCnt,
		"operation_id", opId)

	// Return success response with real operation ID
	return &pb.SendULTransmitResponse{
		Id:      fmt.Sprintf("%d", opId), // Convert int64 to string
		Status:  grpcerrors.StatusQueued,
		Message: grpcerrors.MsgULTransmitQueued,
	}, nil
}

// ============================================================================
// Base Station Status Operations (BSSCI 3.5)
// ============================================================================

// RequestBaseStationStatus sends a status request to a base station
func (s *CoreService) RequestBaseStationStatus(ctx context.Context, req *pb.BaseStationStatusRequest) (*pb.BaseStationStatusResponse, error) {
	// Extract tenant from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "RequestBaseStationStatus request",
		"bs_eui", req.BsEui,
		"tenant_id", tenantID)

	bsEuiBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		bsEuiBytes[i] = byte(req.BsEui >> uint(8*(7-i))) // #nosec G115 - i is bounded 0-7, no overflow
	}

	// Verify tenant ownership first
	bs, err := s.basestationSvc.GetByEUI(ctx, bsEuiBytes, tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to verify base station ownership", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationOwnershipFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationOwnershipFailed))
	}

	// Find connected session
	sessionIface := s.sessionDir.GetSessionByEUI(req.BsEui)
	if sessionIface == nil {
		s.log.Warn("Base station not connected",
			"bs_eui", req.BsEui,
			"bs_name", bs.Name)
		return &pb.BaseStationStatusResponse{
			Success: false,
			Message: "Base station not connected",
			OpId:    0,
		}, nil
	}

	// Send status request - reuse existing logic
	opId, err := s.statusReq.SendStatusRequest(sessionIface)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogStatusRequestFailed,
			"bs_eui", req.BsEui,
			"error", err)
		return &pb.BaseStationStatusResponse{
			Success: false,
			Message: fmt.Sprintf(grpcerrors.MsgStatusRequestFailed, err),
			OpId:    0,
		}, nil
	}

	s.log.InfoContext(ctx, grpcerrors.MsgStatusRequestSent,
		"bs_eui", req.BsEui,
		"op_id", opId,
		"tenant_id", tenantID)

	return &pb.BaseStationStatusResponse{
		Success: true,
		Message: grpcerrors.MsgStatusRequestSent,
		OpId:    opId,
	}, nil
}

// InitiatePing sends a ping request to a base station (BSSCI §5.4)
func (s *CoreService) InitiatePing(ctx context.Context, req *pb.InitiatePingRequest) (*pb.InitiatePingResponse, error) {
	// Extract tenant from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "InitiatePing request",
		"bs_eui", req.BsEui,
		"tenant_id", tenantID)

	bsEuiBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		bsEuiBytes[i] = byte(req.BsEui >> uint(8*(7-i))) // #nosec G115 - i is bounded 0-7, no overflow
	}

	// Verify tenant ownership
	_, err = s.basestationSvc.GetByEUI(ctx, bsEuiBytes, tenantID)
	if err == storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to verify base station ownership", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationOwnershipFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationOwnershipFailed))
	}

	// Call BSSCI server InitiatePing (defense-in-depth tenant validation)
	opId, err := s.pingCmd.InitiatePing(ctx, req.BsEui, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogPingInitiateFailed,
			"bs_eui", req.BsEui,
			"error", err)

		// Branch on CatalogError token for proper gRPC status codes
		var catalogErr *bssci.CatalogError
		if errors.As(err, &catalogErr) {
			switch catalogErr.Token {
			case "bssci.error.session_not_found":
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBSSessionNotFound),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBSSessionNotFound))
			case "bssci.error.cannot_send_ping":
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenHandshakeIncomplete),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenHandshakeIncomplete))
			}
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	s.log.InfoContext(ctx, grpcerrors.MsgPingRequestSent,
		"bs_eui", req.BsEui,
		"op_id", opId,
		"tenant_id", tenantID)

	return &pb.InitiatePingResponse{
		Success: true,
		Message: grpcerrors.MsgPingRequestSent,
		OpId:    opId,
	}, nil
}

// GetDownlinkResults retrieves downlink transmission results per BSSCI §3.14
func (s *CoreService) GetDownlinkResults(ctx context.Context, req *pb.GetDownlinkResultsRequest) (*pb.GetDownlinkResultsResponse, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "Getting downlink results",
		"tenant_id", tenantID,
		"ep_eui", req.EpEui,
		"status_filter", req.StatusFilter,
		"page_size", req.PageSize)

	// Validate and normalize request
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}

	// Parse page token as offset (treat non-numeric or negative as 0)
	offset := 0
	if req.PageToken != "" {
		parsedOffset, err := strconv.Atoi(req.PageToken)
		if err == nil && parsedOffset > 0 {
			offset = parsedOffset
		}
		// If parsing fails or is negative, offset remains 0
	}

	// Translate BSSCI terms to internal status for database query
	// Single translation point for BSSCI §3.14.1
	statusFilter := req.StatusFilter
	switch statusFilter {
	case "sent":
		statusFilter = "transmitted"
	case "invalid":
		statusFilter = "failed"
	}
	// "expired" remains the same

	// Parse time filters from the request
	var timeFrom, timeTo *time.Time
	if req.TimeFrom != nil && req.TimeFrom.IsValid() {
		t := req.TimeFrom.AsTime()
		timeFrom = &t
	}
	if req.TimeTo != nil && req.TimeTo.IsValid() {
		t := req.TimeTo.AsTime()
		timeTo = &t
	}

	// Extract organization ID from context for audit tracking
	var orgID *uuid.UUID
	if orgUUID, orgErr := pkgcontext.GetOrganizationID(ctx); orgErr == nil && orgUUID != uuid.Nil {
		orgID = &orgUUID
	}

	results, totalCount, err := s.messageSvc.GetDownlinkResults(ctx, req.EpEui, strconv.FormatInt(tenantID, 10), orgID, statusFilter, timeFrom, timeTo, int(pageSize), offset)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get downlink results", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkGetResultsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkGetResultsFailed))
	}

	// Convert results to protobuf messages (pagination already applied in database)
	var pbMessages []*pb.DownlinkMessage

	for _, msg := range results {

		pbMsg := &pb.DownlinkMessage{
			Id:           strconv.FormatInt(msg.ID, 10),
			EpEui:        msg.EPEUI,
			TenantId:     msg.TenantID,
			Payload:      msg.Payload,
			Priority:     msg.Priority,
			Status:       msg.Status,
			QueId:        msg.QueID,
			CntDepend:    msg.CntDepend,
			PacketCnt:    msg.PacketCntArray,
			Format:       uint32(msg.Format),
			ResponseExp:  msg.ResponseExp,
			ResponsePrio: msg.ResponsePrio,
			DlWindReq:    msg.DlWindReq,
			ExpOnly:      msg.ExpOnly,
			DlRxStatQry:  msg.DlRxStatQry,
			Payloads:     msg.UserData,
		}

		// Include transmission result fields (BSSCI §3.14.1)
		if msg.Result != "" {
			pbMsg.Result = msg.Result
		}
		if msg.TxTime != 0 {
			pbMsg.TxTime = msg.TxTime
		}
		if msg.TransmissionPacketCnt != 0 {
			pbMsg.TransmissionPacketCnt = msg.TransmissionPacketCnt
		}
		if msg.TxBSEUI != "" {
			pbMsg.BsEui = msg.TxBSEUI
		}

		// Add timestamps
		pbMsg.CreatedAt = timestamppb.New(msg.CreatedAt)
		if msg.ScheduledAt != nil {
			pbMsg.ScheduledAt = timestamppb.New(*msg.ScheduledAt)
		}
		if msg.SentAt != nil {
			pbMsg.TransmittedAt = timestamppb.New(*msg.SentAt)
		}

		pbMessages = append(pbMessages, pbMsg)
	}

	// Generate next page token only if there are more results after the current page
	nextPageToken := ""
	if offset+int(pageSize) < totalCount {
		nextPageToken = strconv.Itoa(offset + int(pageSize))
	}

	// Validate TotalCount fits in int32
	if totalCount > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	return &pb.GetDownlinkResultsResponse{
		Results:       pbMessages,
		NextPageToken: nextPageToken,
		TotalCount:    int32(totalCount), //nolint:gosec // G115: validated above on line 1237
	}, nil
}

// GetDLRXStatus retrieves DL RX status reports for an endpoint (BSSCI §3.15)
func (s *CoreService) GetDLRXStatus(ctx context.Context, req *pb.GetDLRXStatusRequest) (*pb.GetDLRXStatusResponse, error) {
	// Extract tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate endpoint EUI
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	// Parse EUI
	epEui, err := strconv.ParseUint(req.EpEui, 16, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Convert to bytes
	epEuiBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		epEuiBytes[7-i] = byte(epEui >> (i * 8))
	}

	// Set pagination defaults
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int(req.Offset)

	// Parse optional time filters
	var startTime, endTime *time.Time
	if req.StartTime != nil && req.StartTime.IsValid() {
		t := req.StartTime.AsTime()
		startTime = &t
	}
	if req.EndTime != nil && req.EndTime.IsValid() {
		t := req.EndTime.AsTime()
		endTime = &t
	}

	s.log.InfoContext(ctx, "GetDLRXStatus called",
		"tenantId", tenantID,
		"epEui", req.EpEui,
		"limit", limit,
		"offset", offset)

	// Query DL RX status from storage
	statuses, totalCount, err := s.messageSvc.GetDLRXStatusByEndpoint(ctx, tenantID, epEuiBytes, limit, offset, startTime, endTime)
	if err != nil && err != storage.ErrNotFound {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDLRxStatusFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDLRxStatusFailed))
	}

	// Get average metrics for the same time window
	avgSnr, avgRssi, _, err := s.messageSvc.GetAverageDLRXMetrics(ctx, tenantID, epEuiBytes, startTime, endTime)
	if err != nil {
		s.log.WarnContext(ctx, "Failed to get average DL RX metrics",
			"error", err,
			"epEui", hex.EncodeToString(epEuiBytes),
			"tenantId", tenantID)
		// Continue with zero values for avgSnr and avgRssi
	}

	// Convert to proto format
	protoStatuses := make([]*pb.DLRXStatus, 0, len(statuses))
	for _, st := range statuses {
		// Validate dlRxSnr and dlRxRssi against physics limits (BSSCI §5.15).
		if st.DlRxSnr < mioty.DLRxSnrMinDB || st.DlRxSnr > mioty.DLRxSnrMaxDB {
			s.log.WarnContext(ctx, "DlRxSnr value out of physics range, skipping status",
				"epEui", hex.EncodeToString(st.EpEui),
				"dlRxSnr", st.DlRxSnr,
				"validRange", fmt.Sprintf("[%.1f, %.1f] dB", mioty.DLRxSnrMinDB, mioty.DLRxSnrMaxDB))
			continue // Skip corrupted record
		}
		if st.DlRxRssi < mioty.DLRxRssiMinDBm || st.DlRxRssi > mioty.DLRxRssiMaxDBm {
			s.log.WarnContext(ctx, "DlRxRssi value out of physics range, skipping status",
				"epEui", hex.EncodeToString(st.EpEui),
				"dlRxRssi", st.DlRxRssi,
				"validRange", fmt.Sprintf("[%.1f, %.1f] dBm", mioty.DLRxRssiMinDBm, mioty.DLRxRssiMaxDBm))
			continue // Skip corrupted record
		}

		protoStatuses = append(protoStatuses, &pb.DLRXStatus{
			EpEui:     hex.EncodeToString(st.EpEui),
			BsEui:     hex.EncodeToString(st.BsEui),
			RxTime:    st.RxTime,
			PacketCnt: st.PacketCnt,
			DlRxSnr:   st.DlRxSnr,
			DlRxRssi:  st.DlRxRssi,
			CreatedAt: timestamppb.New(st.CreatedAt),
		})
	}

	// Validate TotalCount fits in int32
	if totalCount > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	return &pb.GetDLRXStatusResponse{
		Statuses:   protoStatuses,
		TotalCount: int32(totalCount), //nolint:gosec // G115: validated above on line 1328
		AvgSnr:     avgSnr,
		AvgRssi:    avgRssi,
	}, nil
}

// QueryDLRXStatus initiates a DL RX status query to an endpoint (BSSCI §3.16)
func (s *CoreService) QueryDLRXStatus(ctx context.Context, req *pb.QueryDLRXStatusRequest) (*pb.QueryDLRXStatusResponse, error) {
	// Extract tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate endpoint EUI
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	// Validate EUI is exactly 16 hex characters (8 bytes per MIOTY spec)
	if len(req.EpEui) != 16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEUIHexLengthInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEUIHexLengthInvalid))
	}

	// Validate EUI contains only valid hex characters
	if _, err := hex.DecodeString(req.EpEui); err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEUIHexCharsInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEUIHexCharsInvalid))
	}

	// Convert tenant ID to string for storage method
	tenantIDStr := fmt.Sprintf("%d", tenantID)

	// Get base station serving this endpoint
	bsEui, err := s.messageSvc.GetEndpointBaseStation(ctx, req.EpEui, tenantIDStr)
	if err != nil {
		if err == storage.ErrNotFound {
			return &pb.QueryDLRXStatusResponse{
				QueryInitiated: false,
				Message:        "Endpoint not currently attached to any base station",
			}, nil
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointSessionLookupFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointSessionLookupFailed))
	}

	// Parse base station EUI
	bsEuiUint64, err := strconv.ParseUint(bsEui, 16, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Parse endpoint EUI
	epEuiUint64, err := strconv.ParseUint(req.EpEui, 16, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	s.log.InfoContext(ctx, "QueryDLRXStatus looking for session",
		"tenantId", tenantID,
		"epEui", req.EpEui,
		"bsEui", bsEui)

	// Find session for the base station serving this endpoint.
	// GetEndpointBaseStation already validated endpoint ownership (filtered by tenant_id),
	// so we can safely route to the returned BS regardless of which tenant owns it (roaming support).
	sessionID, err := s.sessionDir.FindSessionForEndpointAttachment(bsEuiUint64)
	if err != nil {
		// Log the failure reason before mapping to user message
		s.log.WarnContext(ctx, "Failed to find base station session for DL RX status query",
			"tenantId", tenantID,
			"bsEui", bsEui,
			"epEui", req.EpEui,
			"error", err)

		// Map session lookup errors to user-friendly messages
		if errors.Is(err, bssci.ErrSessionNotFound) {
			return &pb.QueryDLRXStatusResponse{
				QueryInitiated: false,
				Message:        fmt.Sprintf("Base station %s is not currently connected", bsEui),
			}, nil
		}
		if errors.Is(err, bssci.ErrSessionNotReady) {
			return &pb.QueryDLRXStatusResponse{
				QueryInitiated: false,
				Message:        fmt.Sprintf("Base station %s handshake not complete", bsEui),
			}, nil
		}
		if errors.Is(err, bssci.ErrSessionNotBidirectional) {
			return &pb.QueryDLRXStatusResponse{
				QueryInitiated: false,
				Message:        fmt.Sprintf("Base station %s does not support bidirectional operations", bsEui),
			}, nil
		}

		// Preserve existing message for operations teams
		return &pb.QueryDLRXStatusResponse{
			QueryInitiated: false,
			Message:        "No suitable base station session available",
		}, nil
	}

	s.log.InfoContext(ctx, "QueryDLRXStatus found serving base station session",
		"tenantId", tenantID,
		"sessionId", sessionID,
		"bsEui", bsEui)

	// Send query through the validated session
	err = s.downlinkCmd.SendDLRXStatusQuery(sessionID, epEuiUint64)
	if err != nil {
		return &pb.QueryDLRXStatusResponse{
			QueryInitiated: false,
			Message:        fmt.Sprintf("Failed to send query: %v", err),
		}, nil
	}

	return &pb.QueryDLRXStatusResponse{
		QueryInitiated: true,
		Message:        "DL RX status query sent to base station",
	}, nil
}

// GetDLRXStatusQueries retrieves query tracking history for an endpoint (BSSCI §5.15 telemetry)
func (s *CoreService) GetDLRXStatusQueries(ctx context.Context, req *pb.GetDLRXStatusQueriesRequest) (*pb.GetDLRXStatusQueriesResponse, error) {
	// Extract tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate endpoint EUI
	if req.EpEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired))
	}

	// Validate EUI is exactly 16 hex characters (8 bytes per MIOTY spec)
	if len(req.EpEui) != 16 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEUIHexLengthInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEUIHexLengthInvalid))
	}

	// Parse EUI
	epEui, err := strconv.ParseUint(req.EpEui, 16, 64)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat))
	}

	// Convert to bytes
	epEuiBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		epEuiBytes[7-i] = byte(epEui >> (i * 8))
	}

	// ===== VALIDATE RAW INPUTS BEFORE DEFAULTING =====

	// Reject negative limit (must validate before defaulting to 20)
	if req.Limit < 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLimitNonNegative),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLimitNonNegative))
	}

	// Reject negative offset
	if req.Offset < 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOffsetNonNegative),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOffsetNonNegative))
	}

	// Validate time range ordering (if both provided)
	if req.StartTime != nil && req.EndTime != nil {
		if req.StartTime.IsValid() && req.EndTime.IsValid() {
			if req.StartTime.AsTime().After(req.EndTime.AsTime()) {
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTimeRange),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTimeRange))
			}
		}
	}

	// ===== NOW APPLY DEFAULTS AND BOUNDS =====

	// Set pagination defaults
	limit := int(req.Limit)
	if limit == 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	offset := int(req.Offset) // Already validated >= 0

	// Parse optional time filters
	var startTime, endTime *time.Time
	if req.StartTime != nil && req.StartTime.IsValid() {
		t := req.StartTime.AsTime()
		startTime = &t
	}
	if req.EndTime != nil && req.EndTime.IsValid() {
		t := req.EndTime.AsTime()
		endTime = &t
	}

	s.log.InfoContext(ctx, "GetDLRXStatusQueries called",
		"tenantId", tenantID,
		"epEui", req.EpEui,
		"limit", limit,
		"offset", offset)

	// Query history from narrow DLRX storage interface (BSSCI §5.15 telemetry)
	queries, totalCount, err := s.dlrxStorage.GetDLRXStatusQueryHistory(ctx, tenantID, epEuiBytes, limit, offset, startTime, endTime)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDLRxStatusQueryHistoryFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDLRxStatusQueryHistoryFailed))
	}

	// Query stats (always computed per user requirement)
	pendingCount, receivedCount, timeoutCount, err := s.dlrxStorage.GetDLRXStatusQueryStats(ctx, tenantID, epEuiBytes, startTime, endTime)
	if err != nil {
		s.log.WarnContext(ctx, "Failed to get DL RX status query stats",
			"error", err,
			"epEui", hex.EncodeToString(epEuiBytes),
			"tenantId", tenantID)
		// Continue with zero values for stats
	}

	// Convert to proto format
	protoQueries := make([]*pb.DLRXStatusQuery, len(queries))
	for i, q := range queries {
		protoQuery := &pb.DLRXStatusQuery{
			EpEui:       strings.ToUpper(hex.EncodeToString(q.EpEui)),
			BsEui:       strings.ToUpper(hex.EncodeToString(q.BsEui)),
			OpId:        q.OpId,
			Status:      q.Status,
			RequestedAt: timestamppb.New(q.RequestedAt),
		}
		// ReceivedAt is optional (nil if status != "received")
		if q.ReceivedAt != nil {
			protoQuery.ReceivedAt = timestamppb.New(*q.ReceivedAt)
		}
		// OrgUuid is optional (empty string if not set)
		if q.OrganizationID != nil {
			protoQuery.OrgUuid = q.OrganizationID.String()
		}
		protoQueries[i] = protoQuery
	}

	return &pb.GetDLRXStatusQueriesResponse{
		Queries:    protoQueries,
		TotalCount: int64(totalCount),
		Stats: &pb.DLRXStatusQueryStats{
			Pending:  pendingCount,
			Received: receivedCount,
			Timeout:  timeoutCount,
		},
	}, nil
}

// GetReleaseInfo returns build and version metadata from the embedded release manifest.
func (s *CoreService) GetReleaseInfo(ctx context.Context, _ *emptypb.Empty) (*pb.ReleaseInfo, error) {
	s.log.InfoContext(ctx, "GetReleaseInfo called")

	// Load version info from embedded manifest
	info, err := version.Get()
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogReleaseManifestLoadFailed, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenReleaseManifestFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenReleaseManifestFailed))
	}

	// Convert schema version to int32 with bounds checking (gosec G115)
	var schemaVersion int32
	if info.SchemaVersion > 2147483647 {
		s.log.WarnContext(ctx, "Schema version exceeds int32 max, clamping", "actual", info.SchemaVersion)
		schemaVersion = 2147483647
	} else {
		schemaVersion = int32(info.SchemaVersion) // #nosec G115 -- bounds checked above
	}

	// Convert to proto format
	return &pb.ReleaseInfo{
		Version:          info.Version,
		BuildTime:        info.BuildTime,
		GitCommit:        info.GitCommit,
		GitBranch:        info.GitBranch,
		BuildUser:        info.BuildUser,
		GoVersion:        info.GoVersion,
		SchemaVersion:    schemaVersion,
		Artifacts:        info.Artifacts,
		ScEui:            s.scEui,
		ScVendor:         s.scVendor,
		ScModel:          s.scModel,
		ScName:           s.scName,
		ScSwVersion:      s.scSwVersion,
		Edition:          config.EditionLabel(s.edition),
		EditionCode:      s.edition,
		LicenseId:        info.LicenseID,
		LicenseUrl:       info.LicenseURL,
		SourceUrl:        info.SourceURL,
		DocumentationUrl: info.DocsURL,
		HomepageUrl:      info.HomepageURL,
		TrademarkNotice:  info.TrademarkNotice,
	}, nil
}

// GetStatistics returns aggregated statistics across endpoints, base stations, and messages.
func (s *CoreService) GetStatistics(ctx context.Context, req *pb.GetStatisticsRequest) (*pb.Statistics, error) {
	if s.statisticsSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	startTime := timestampToTime(req.StartTime)
	endTime := timestampToTime(req.EndTime)

	stats, err := s.statisticsSvc.GetStatistics(ctx, tenantID, startTime, endTime, req.Granularity)
	if err != nil {
		if errors.Is(err, statistics.ErrUnsupportedGranularity) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUnsupportedGranularity),
				err.Error())
		}
		s.log.ErrorContext(ctx, "get statistics failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMessageStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMessageStatsFailed))
	}

	// Convert time series
	pbCounts := make([]*pb.TimeSeriesData, len(stats.MessageCounts))
	for i, mc := range stats.MessageCounts {
		pbCounts[i] = &pb.TimeSeriesData{
			Timestamp: timestamppb.New(mc.Timestamp),
			Value:     mc.Value,
		}
	}

	return &pb.Statistics{
		TotalMessages:            stats.TotalMessages,
		TotalEndpoints:           stats.TotalEndpoints,
		TotalBasestations:        stats.TotalBaseStations,
		MessageCounts:            pbCounts,
		EndpointMessageCounts:    stats.EndpointMessageCounts,
		BasestationMessageCounts: stats.BaseStationMessageCounts,
	}, nil
}

// GetSystemStatus returns service health and tenant-scoped metrics when available.
func (s *CoreService) GetSystemStatus(ctx context.Context, _ *emptypb.Empty) (*pb.SystemStatus, error) {
	s.log.InfoContext(ctx, grpcerrors.LogSystemStatusCalled)

	versionValue := s.scSwVersion
	info, err := version.Get()
	if err != nil {
		s.log.WarnContext(ctx, grpcerrors.LogSystemStatusManifestLoadFailed, "error", err)
	} else if info.Version != "" {
		versionValue = info.Version
	}

	response := &pb.SystemStatus{
		Version: versionValue,
		Status:  string(healthstatus.StatusHealthy),
		Uptime:  timestamppb.New(s.serviceStart),
	}

	// Fetch service health statuses (tenant-agnostic - always populate)
	if s.systemStatusSvc != nil {
		dtos, svcErr := s.systemStatusSvc.GetServiceStatuses(ctx)
		if svcErr != nil {
			s.log.WarnContext(ctx, grpcerrors.LogSystemStatusHealthCheckFailed, "error", svcErr)
		} else if len(dtos) > 0 {
			services := make([]*pb.ServiceStatus, 0, len(dtos))
			for _, dto := range dtos {
				svc := &pb.ServiceStatus{
					Name:      dto.Name,
					Url:       dto.URL,
					Healthy:   dto.Healthy,
					LatencyMs: dto.LatencyMs,
					Error:     dto.Error,
					CheckedAt: timestamppb.New(dto.CheckedAt),
				}
				services = append(services, svc)
			}
			response.Services = services
		}
	}

	// Tenant-scoped metrics (only if tenant context exists)
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil || tenantID <= 0 || s.systemStatusSvc == nil {
		return response, nil
	}

	metrics, err := s.systemStatusSvc.GetStatus(ctx, tenantID)
	if err != nil {
		s.log.WarnContext(ctx, grpcerrors.LogSystemStatusMetricsFetchFailed, "error", err)
		return response, nil
	}

	response.ActiveBasestations = metrics.ActiveBasestations
	response.ActiveEndpoints = metrics.ActiveEndpoints
	response.MessagesProcessed = metrics.MessagesProcessed

	return response, nil
}

// timestampToTime converts a protobuf timestamp to a Go time pointer
// Returns nil if the timestamp is nil
func timestampToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
