package grpc

import (
	"context"
	"errors"
	"math"
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	healthstatus "github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/diagnostics"
	grpcservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/statistics"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/systemstatus"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

// SystemHandlers serves the release, statistics and system status RPCs.
type SystemHandlers struct {
	statisticsSvc   grpcservices.StatisticsService
	systemStatusSvc grpcservices.SystemStatusService
	scEui           uint64
	scVendor        string
	scModel         string
	scName          string
	scSwVersion     string
	edition         string
	capabilities    []grpcservices.Capability
	diagnostics     DiagnosticsBuilder
	serviceStart    time.Time
	log             logger.Logger
}

// DiagnosticsBuilder assembles the server-admin diagnostics bundle of a tenant.
type DiagnosticsBuilder interface {
	Build(ctx context.Context, tenantID int64) (*diagnostics.Bundle, error)
}

// SystemHandlerDeps wires SystemHandlers; nil services answer their RPCs
// as not configured.
type SystemHandlerDeps struct {
	Statistics   grpcservices.StatisticsService
	SystemStatus grpcservices.SystemStatusService
	SCEui        uint64
	SCVendor     string
	SCModel      string
	SCName       string
	SCSwVersion  string
	Edition      string
	Capabilities []grpcservices.Capability
	Diagnostics  DiagnosticsBuilder
	// StartedAt is the process start the uptime is measured from.
	StartedAt time.Time
}

// NewSystemHandlers builds the group; the process start is required because
// the reported uptime is measured from it.
func NewSystemHandlers(d SystemHandlerDeps, log logger.Logger) (*SystemHandlers, error) {
	if d.StartedAt.IsZero() {
		return nil, errors.New(errMsgServiceStartRequired)
	}
	return &SystemHandlers{
		statisticsSvc:   d.Statistics,
		systemStatusSvc: d.SystemStatus,
		scEui:           d.SCEui,
		scVendor:        d.SCVendor,
		scModel:         d.SCModel,
		scName:          d.SCName,
		scSwVersion:     d.SCSwVersion,
		edition:         d.Edition,
		capabilities:    d.Capabilities,
		diagnostics:     d.Diagnostics,
		serviceStart:    d.StartedAt,
		log:             log,
	}, nil
}

// GetReleaseInfo returns build and version metadata from the embedded release manifest.
func (s *SystemHandlers) GetReleaseInfo(ctx context.Context, _ *emptypb.Empty) (*pb.ReleaseInfo, error) {
	s.log.InfoContext(ctx, LogGetReleaseInfoCalled)

	// Load version info from embedded manifest
	info, err := version.Get()
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogReleaseManifestLoadFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenReleaseManifestFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenReleaseManifestFailed))
	}

	// Convert schema version to int32 with bounds checking (gosec G115)
	var schemaVersion int32
	if info.SchemaVersion > math.MaxInt32 {
		s.log.WarnContext(ctx, LogSchemaVersionExceedsInt32MaxClamping, logger.FieldActual, info.SchemaVersion)
		schemaVersion = math.MaxInt32
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
func (s *SystemHandlers) GetStatistics(ctx context.Context, req *pb.GetStatisticsRequest) (*pb.Statistics, error) {
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
		s.log.ErrorContext(ctx, LogGetStatisticsFailed, logger.FieldError, err)
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
func (s *SystemHandlers) GetSystemStatus(ctx context.Context, _ *emptypb.Empty) (*pb.SystemStatus, error) {
	s.log.InfoContext(ctx, systemstatus.LogSystemStatusCalled)

	versionValue := s.scSwVersion
	info, err := version.Get()
	if err != nil {
		s.log.WarnContext(ctx, systemstatus.LogSystemStatusManifestLoadFailed, logger.FieldError, err)
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
			s.log.WarnContext(ctx, systemstatus.LogSystemStatusHealthCheckFailed, logger.FieldError, svcErr)
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
		s.log.WarnContext(ctx, systemstatus.LogSystemStatusMetricsFetchFailed, logger.FieldError, err)
		return response, nil
	}

	response.ActiveBasestations = metrics.ActiveBasestations
	response.ActiveEndpoints = metrics.ActiveEndpoints
	response.MessagesProcessed = metrics.MessagesProcessed

	return response, nil
}

// ListCapabilities returns the allowlisted, non-secret feature toggles.
func (s *SystemHandlers) ListCapabilities(_ context.Context, _ *emptypb.Empty) (*pb.ListCapabilitiesResponse, error) {
	out := make([]*pb.Capability, len(s.capabilities))
	for i, c := range s.capabilities {
		out[i] = &pb.Capability{Name: c.Name, Enabled: c.Enabled}
	}
	return &pb.ListCapabilitiesResponse{Capabilities: out}, nil
}

// GetDiagnosticsBundle hands a server administrator the projected bundle of
// the calling tenant; the size limit maps to ResourceExhausted.
func (s *SystemHandlers) GetDiagnosticsBundle(ctx context.Context, _ *pb.GetDiagnosticsBundleRequest) (*pb.GetDiagnosticsBundleResponse, error) {
	if s.diagnostics == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	bundle, err := s.diagnostics.Build(ctx, tenantID)
	switch {
	case errors.Is(err, diagnostics.ErrBundleTooLarge):
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDiagnosticsTooLarge),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDiagnosticsTooLarge))
	case err != nil:
		s.log.ErrorContext(ctx, LogGetDiagnosticsBundleFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDiagnosticsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDiagnosticsFailed))
	}
	return &pb.GetDiagnosticsBundleResponse{
		Archive:     bundle.Archive,
		Filename:    bundle.Filename,
		ContentType: bundle.ContentType,
		SizeBytes:   int64(len(bundle.Archive)),
		GeneratedAt: timestamppb.New(bundle.GeneratedAt),
	}, nil
}
