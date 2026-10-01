// Package systemstatus provides system status aggregation for the gRPC layer.
package systemstatus

import (
	"context"
	"math"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Log messages for system status assembly. Owned by the service; the gRPC
// delivery layer references them for its request-level logging too.
const (
	LogSystemStatusBSStatsFailed      = "Failed to fetch base station stats"
	LogSystemStatusEPCountFailed      = "Failed to fetch endpoint count"
	LogSystemStatusMsgStatsFailed     = "Failed to fetch message stats"
	LogSystemStatusMetricsFetchFailed = "Failed to fetch system status metrics"
	LogSystemStatusCalled             = "GetSystemStatus called"
	LogSystemStatusManifestLoadFailed = "Failed to load release manifest for system status"
	LogSystemStatusHealthCheckFailed  = "Failed to fetch service health statuses"
)

// BaseStationStatsFetcher is a narrow interface for base station statistics.
type BaseStationStatsFetcher interface {
	GetStatistics(ctx context.Context, tenantID int64) (*models.BaseStationStatistics, error)
}

// EndpointCounter is a narrow interface for endpoint counting.
type EndpointCounter interface {
	CountByTenant(ctx context.Context, tenantID int64) (int64, error)
}

// MessageStatsFetcher is a narrow interface for message statistics.
type MessageStatsFetcher interface {
	GetOverallStats(ctx context.Context, tenantID int64) (*mioty.MessageStats, error)
}

// EndpointURL holds the URL for a named endpoint.
type EndpointURL struct {
	Name string
	URL  string
}

// HealthChecker runs the aggregate health checks this service reports as
// per-service statuses. It is the only capability used from the health
// subsystem.
type HealthChecker interface {
	CheckHealth(ctx context.Context) *health.Response
}

// Service aggregates system status metrics from multiple repositories.
type Service struct {
	bsRepo       BaseStationStatsFetcher
	epRepo       EndpointCounter
	msgRepo      MessageStatsFetcher
	logger       logger.Logger
	healthSvc    HealthChecker
	endpointURLs []EndpointURL
}

// New creates a new SystemStatusService with narrow repository dependencies.
// healthChecker may be nil, in which case per-service statuses are not
// reported.
func New(
	bsRepo BaseStationStatsFetcher,
	epRepo EndpointCounter,
	msgRepo MessageStatsFetcher,
	log logger.Logger,
	healthChecker HealthChecker,
	endpointURLs []EndpointURL,
) *Service {
	return &Service{
		bsRepo:       bsRepo,
		epRepo:       epRepo,
		msgRepo:      msgRepo,
		logger:       log,
		healthSvc:    healthChecker,
		endpointURLs: endpointURLs,
	}
}

// GetStatus returns aggregated system metrics for a tenant.
// Returns empty metrics (not error) for invalid tenant ID or repository failures.
func (s *Service) GetStatus(ctx context.Context, tenantID int64) (*grpcservices.SystemStatusMetrics, error) {
	metrics := &grpcservices.SystemStatusMetrics{}

	// Skip if tenant ID is invalid
	if tenantID <= 0 {
		return metrics, nil
	}

	// Fetch base station stats (narrow: GetStatistics only)
	if s.bsRepo != nil {
		if bsStats, err := s.bsRepo.GetStatistics(ctx, tenantID); err != nil {
			s.logger.WarnContext(ctx, LogSystemStatusBSStatsFailed, logger.FieldError, err)
		} else if bsStats != nil {
			metrics.ActiveBasestations = clampToInt32(bsStats.OnlineCount)
		}
	}

	// Fetch endpoint count (narrow: CountByTenant only)
	if s.epRepo != nil {
		if epCount, err := s.epRepo.CountByTenant(ctx, tenantID); err != nil {
			s.logger.WarnContext(ctx, LogSystemStatusEPCountFailed, logger.FieldError, err)
		} else {
			metrics.ActiveEndpoints = clampToInt32(epCount)
		}
	}

	// Fetch message stats (narrow: GetOverallStats only)
	if s.msgRepo != nil {
		if msgStats, err := s.msgRepo.GetOverallStats(ctx, tenantID); err != nil {
			s.logger.WarnContext(ctx, LogSystemStatusMsgStatsFailed, logger.FieldError, err)
		} else if msgStats != nil {
			metrics.MessagesProcessed = msgStats.TotalCount
		}
	}

	return metrics, nil
}

// clampToInt32 safely clamps an int64 to int32 bounds.
func clampToInt32(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v) //nolint:gosec // bounds checked above
}

// GetServiceStatuses returns DTOs with service health information.
func (s *Service) GetServiceStatuses(ctx context.Context) ([]*grpcservices.ServiceStatusDTO, error) {
	if s.healthSvc == nil {
		return nil, nil
	}
	response := s.healthSvc.CheckHealth(ctx)

	services := make([]*grpcservices.ServiceStatusDTO, 0, len(response.Checks))
	for name, check := range response.Checks {
		// The dashboard lists running services; a disabled one has no health to show.
		if check.Status == health.StatusDisabled {
			continue
		}
		latencyMs := check.Duration.Milliseconds()
		// Cap latency at max int32 to prevent overflow (G115)
		if latencyMs > math.MaxInt32 {
			latencyMs = math.MaxInt32
		}
		dto := &grpcservices.ServiceStatusDTO{
			Name:      name,
			URL:       s.getEndpointURL(name),
			Healthy:   check.Status == health.StatusHealthy,
			LatencyMs: int32(latencyMs), // #nosec G115 - capped above
			CheckedAt: check.Timestamp,
		}
		if check.Status != health.StatusHealthy {
			dto.Error = check.Message
		}
		services = append(services, dto)
	}
	return services, nil
}

// getEndpointURL returns the URL for a named endpoint.
func (s *Service) getEndpointURL(name string) string {
	for _, ep := range s.endpointURLs {
		if ep.Name == name {
			return ep.URL
		}
	}
	return ""
}
