// Package scaciservices implements SCACI service layer components.
//
// status_service.go implements StatusService interface.
//
// Extracted Logic:
//   - Server uptime calculation (from handler_status.go:36)
//   - Base station counting (from handler_status.go:48)
//
// Dependencies (injected):
//   - interfaces.BaseStationRepository: BS state queries
//   - time.Time: Service start timestamp
//
// Error Handling:
//   - Returns Go errors (not SCACI tokens) - status queries are informational
//   - Transport layer handles errors by returning partial status
//
// Infrastructure Reuse:
//   - Uses existing BaseStationRepository.List() method
//   - No new database queries or schema changes
package scaciservices

import (
	"context"
	"time"

	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// statusService implements StatusService interface
//
// This service provides server status information per SCACI §3.5.
// It encapsulates:
//   - serviceStart timestamp for uptime calculation
//   - baseStationRepo for active BS counting
//   - endpointRepo for preferred BS lookup (§3.9.1 production readiness)
//
// Encapsulates status fields in a service to improve separation of concerns
// and allow status logic to evolve independently of the transport layer.
type statusService struct {
	baseStationRepo interfaces.BaseStationRepository
	endpointRepo    interfaces.EndpointRepository
	serviceStart    time.Time
}

// NewStatusService creates a new status service
//
// Parameters:
//   - baseStationRepo: Base station repository for counting active BSs
//   - endpointRepo: Endpoint repository for preferred BS lookup (§3.9.1)
//   - serviceStart: Timestamp when Service Center started (for uptime calculation)
//
// Returns:
//   - StatusService: Service instance implementing interface
func NewStatusService(
	baseStationRepo interfaces.BaseStationRepository,
	endpointRepo interfaces.EndpointRepository,
	serviceStart time.Time,
) scaci.StatusService {
	return &statusService{
		baseStationRepo: baseStationRepo,
		endpointRepo:    endpointRepo,
		serviceStart:    serviceStart,
	}
}

// GetUptime implements StatusService.GetUptime
//
// Extracted from handler_status.go:36
//
// Calculates time elapsed since Service Center startup.
// Used by Status handler to populate uptimeSec field per SCACI §3.5.2
//
// Returns:
//   - int64: Uptime in seconds
func (ss *statusService) GetUptime() int64 {
	return int64(time.Since(ss.serviceStart).Seconds())
}

// GetBaseStations implements StatusService.GetBaseStations
//
// Extracted from handler_status.go:42-55
//
// Queries base station repository for all base stations in tenant scope.
// Used by Status handler to populate baseStations array per SCACI §3.5.2
//
// The handler transforms these records into BaseStationStatus protocol format.
//
// Parameters:
//   - ctx: Request context with timeout
//   - tenantID: Tenant scope for BS filtering
//
// Returns:
//   - []*models.BaseStation: Slice of base stations (empty if none found or error)
//   - error: Database error or nil on success
func (ss *statusService) GetBaseStations(ctx context.Context, tenantID int64) ([]*models.BaseStation, error) {
	filter := &models.BaseStationFilter{
		TenantID: tenantID,
		Limit:    0, // Fetch all base stations (no pagination)
		Offset:   0,
	}

	baseStations, _, err := ss.baseStationRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return baseStations, nil
}

// GetBaseStation implements StatusService.GetBaseStation
//
// Encapsulates baseStationRepo.GetByEUI so handlers don't
// need direct access to the repository.
//
// Retrieves a single base station by EUI for tenant validation.
// Used by UL Data Transmit handler (handler_operations.go:422) to verify
// BS ownership before scheduling uplink transmission.
//
// Parameters:
//   - ctx: Request context with timeout
//   - tenantID: Tenant scope for BS filtering
//   - eui: Base station EUI (8-byte slice)
//
// Returns:
//   - *models.BaseStation: Base station record if found
//   - error: Database error or storage.ErrNotFound
func (ss *statusService) GetBaseStation(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	return ss.baseStationRepo.GetByEUI(ctx, tenantID, eui)
}

// GetPreferredBaseStation implements StatusService.GetPreferredBaseStation
//
// SCACI §3.9.1 Production Readiness: Retrieves last-attached base station EUI
// for an endpoint to support "Service Center preferred BS" selection.
//
// When UL Data Transmit is called without explicit bsEui, this method provides
// the endpoint's last-attached BS as the preferred routing target.
//
// Parameters:
//   - ctx: Request context with timeout
//   - tenantID: Tenant scope for endpoint filtering
//   - epEui: Endpoint EUI (8-byte slice)
//
// Returns:
//   - *uint64: Preferred BS EUI if endpoint has last_attached_bs_eui set, nil otherwise
//   - bool: true if preference found (non-NULL), false if endpoint not found or NULL
//   - error: Database error (nil on success or not found)
func (ss *statusService) GetPreferredBaseStation(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error) {
	return ss.endpointRepo.GetPreferredBsEui(ctx, tenantID, epEui)
}
