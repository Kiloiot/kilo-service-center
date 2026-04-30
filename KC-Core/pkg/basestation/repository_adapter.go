package basestation

import (
	"context"
	"fmt"
	"time"

	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
	pkgcontext "github.com/kilocenter/pkg/context"
)

// RepositoryAdapter adapts KC-DB BaseStationRepository to Store interface
type RepositoryAdapter struct {
	repo                      interfaces.BaseStationRepository
	tenantID                  int64
	canonicalServiceCenterURL string
}

// NewRepositoryAdapter creates a new repository adapter
func NewRepositoryAdapter(repo interfaces.BaseStationRepository, tenantID int64, canonicalURL string) *RepositoryAdapter {
	return &RepositoryAdapter{
		repo:                      repo,
		tenantID:                  tenantID,
		canonicalServiceCenterURL: canonicalURL,
	}
}

// effectiveTenant returns the tenant ID from context if available, otherwise falls back to the configured default
func (ra *RepositoryAdapter) effectiveTenant(ctx context.Context) int64 {
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil && tid > 0 {
		return tid
	}
	return ra.tenantID
}

// RegisterBaseStation implements Store interface
func (ra *RepositoryAdapter) RegisterBaseStation(ctx context.Context, reg *Registration) error {
	tenant := ra.effectiveTenant(ctx)

	// Convert Registration to models.BaseStation
	baseStation := &models.BaseStation{
		TenantID:       tenant,
		EUI:            models.EUI(reg.EUI[:]),
		Name:           reg.Name,
		ConnectionType: models.ConnectionType(reg.ConnectionType),
	}

	// Set optional fields
	// Unconditional override for BSSCI connections
	if reg.ConnectionType == ConnectionTypeBSSCI {
		baseStation.ServiceCenterURL = &ra.canonicalServiceCenterURL
	} else if reg.ServiceCenterURL != "" {
		baseStation.ServiceCenterURL = &reg.ServiceCenterURL
	}

	if reg.Vendor != "" {
		baseStation.Vendor = &reg.Vendor
	}

	if reg.Model != "" {
		baseStation.Model = &reg.Model
	}

	if reg.Version != "" {
		baseStation.Version = &reg.Version
	}

	// Set location if provided
	if reg.Location != nil {
		baseStation.Latitude = &reg.Location.Latitude
		baseStation.Longitude = &reg.Location.Longitude
		baseStation.Altitude = &reg.Location.Altitude
	}

	// Set connection status to online initially
	baseStation.IsOnline = true

	// Try to create, or update if already exists
	err := ra.repo.Create(ctx, baseStation)
	if err != nil {
		// If creation fails due to duplicate EUI, try to update
		existingBS, getErr := ra.repo.GetByEUI(ctx, tenant, baseStation.EUI[:])
		if getErr == nil && existingBS != nil {
			// Update existing basestation
			updates := map[string]interface{}{
				"name":            baseStation.Name,
				"connection_type": baseStation.ConnectionType,
				"is_online":       true,
				"vendor":          baseStation.Vendor,
				"model":           baseStation.Model,
				"sw_version":      baseStation.Version,
			}

			// Unconditional override for BSSCI connections in updates map
			if reg.ConnectionType == ConnectionTypeBSSCI {
				updates["service_center_url"] = ra.canonicalServiceCenterURL
			} else if baseStation.ServiceCenterURL != nil {
				updates["service_center_url"] = *baseStation.ServiceCenterURL
			}

			return ra.repo.Update(ctx, tenant, existingBS.ID, updates)
		}
		return fmt.Errorf("failed to register basestation: %w", err)
	}

	return nil
}

// GetBaseStation implements BaseStationStore interface
func (ra *RepositoryAdapter) GetBaseStation(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	tenant := ra.effectiveTenant(ctx)
	euiBytes := eui[:]
	dbBaseStation, err := ra.repo.GetByEUI(ctx, tenant, euiBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get basestation: %w", err)
	}

	if dbBaseStation == nil {
		return nil, fmt.Errorf("basestation not found")
	}

	return ra.convertDBBaseStation(ctx, dbBaseStation), nil
}

// GetBaseStationGlobal retrieves a Base Station by EUI across all tenants.
// Used during BSSCI connect handshake before tenant is resolved.
func (ra *RepositoryAdapter) GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	euiBytes := eui[:]
	dbBaseStation, err := ra.repo.GetByEUIGlobal(ctx, euiBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get basestation: %w", err)
	}

	if dbBaseStation == nil {
		return nil, fmt.Errorf("basestation not found")
	}

	return ra.convertDBBaseStation(ctx, dbBaseStation), nil
}

// convertDBBaseStation converts a models.BaseStation to basestation.BaseStation
func (ra *RepositoryAdapter) convertDBBaseStation(ctx context.Context, dbBaseStation *models.BaseStation) *BaseStation {
	bs := &BaseStation{
		ID:             dbBaseStation.ID,
		TenantID:       dbBaseStation.TenantID,
		EUI:            [8]byte(dbBaseStation.EUI),
		Name:           dbBaseStation.Name,
		ConnectionType: ConnectionType(dbBaseStation.ConnectionType),
		Metadata:       make(map[string]interface{}),
	}

	if dbBaseStation.Description != nil {
		bs.Description = *dbBaseStation.Description
	}

	if dbBaseStation.ServiceCenterURL != nil {
		bs.ServiceCenterURL = *dbBaseStation.ServiceCenterURL
	}

	if dbBaseStation.Vendor != nil {
		bs.Vendor = *dbBaseStation.Vendor
	}

	if dbBaseStation.Model != nil {
		bs.Model = *dbBaseStation.Model
	}

	if dbBaseStation.Version != nil {
		bs.SoftwareVersion = *dbBaseStation.Version
	}

	if dbBaseStation.LastSeenAt != nil {
		bs.LastSeenAt = dbBaseStation.LastSeenAt
	}

	// Set location if available
	if dbBaseStation.Latitude != nil && dbBaseStation.Longitude != nil {
		bs.Location = &Location{
			Latitude:  *dbBaseStation.Latitude,
			Longitude: *dbBaseStation.Longitude,
		}
		if dbBaseStation.Altitude != nil {
			bs.Location.Altitude = *dbBaseStation.Altitude
		}
	}

	// Backfill NULL/non-canonical URLs for BSSCI stations on read
	if string(dbBaseStation.ConnectionType) == "bssci" &&
		(dbBaseStation.ServiceCenterURL == nil || *dbBaseStation.ServiceCenterURL != ra.canonicalServiceCenterURL) {
		bs.ServiceCenterURL = ra.canonicalServiceCenterURL
		_ = ra.repo.Update(ctx, dbBaseStation.TenantID, dbBaseStation.ID, map[string]interface{}{
			"service_center_url": ra.canonicalServiceCenterURL,
		})
	}

	return bs
}

// UpdateConnectionStatus implements BaseStationStore interface
func (ra *RepositoryAdapter) UpdateConnectionStatus(ctx context.Context, eui [8]byte, status *ConnectionStatus) error {
	tenant := ra.effectiveTenant(ctx)
	euiBytes := eui[:]

	dbBaseStation, err := ra.repo.GetByEUI(ctx, tenant, euiBytes)
	if err != nil {
		return fmt.Errorf("failed to get basestation: %w", err)
	}

	if dbBaseStation == nil {
		return fmt.Errorf("basestation not found")
	}

	// Update connection status
	updates := map[string]interface{}{
		"is_online":       status.IsOnline,
		"last_seen_at":    status.LastSeen,
		"connection_type": string(status.ConnectionType),
	}

	if status.SessionID != "" {
		updates["session_uuid"] = status.SessionID
	}

	return ra.repo.Update(ctx, tenant, dbBaseStation.ID, updates)
}

// ListBaseStations implements BaseStationStore interface
func (ra *RepositoryAdapter) ListBaseStations(ctx context.Context) ([]*BaseStation, error) {
	filter := &models.BaseStationFilter{
		TenantID: ra.effectiveTenant(ctx),
	}

	dbBaseStations, _, err := ra.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list basestations: %w", err)
	}

	var baseStations []*BaseStation
	for _, dbBS := range dbBaseStations {
		bs := &BaseStation{
			ID:             dbBS.ID,
			EUI:            [8]byte(dbBS.EUI),
			Name:           dbBS.Name,
			ConnectionType: ConnectionType(dbBS.ConnectionType),
			Metadata:       make(map[string]interface{}),
		}

		if dbBS.Description != nil {
			bs.Description = *dbBS.Description
		}

		if dbBS.ServiceCenterURL != nil {
			bs.ServiceCenterURL = *dbBS.ServiceCenterURL
		}

		if dbBS.Vendor != nil {
			bs.Vendor = *dbBS.Vendor
		}

		if dbBS.Model != nil {
			bs.Model = *dbBS.Model
		}

		if dbBS.Version != nil {
			bs.SoftwareVersion = *dbBS.Version
		}

		if dbBS.LastSeenAt != nil {
			bs.LastSeenAt = dbBS.LastSeenAt
		}

		// Set location if available
		if dbBS.Latitude != nil && dbBS.Longitude != nil {
			bs.Location = &Location{
				Latitude:  *dbBS.Latitude,
				Longitude: *dbBS.Longitude,
			}
			if dbBS.Altitude != nil {
				bs.Location.Altitude = *dbBS.Altitude
			}
		}

		// Create real connection status based on database fields
		status := &ConnectionStatus{
			IsOnline:       dbBS.IsOnline,
			ConnectionType: ConnectionType(dbBS.ConnectionType),
		}

		if dbBS.LastSeenAt != nil {
			status.LastSeen = *dbBS.LastSeenAt
		}

		if dbBS.SessionUUID != nil {
			status.SessionID = *dbBS.SessionUUID
		}

		// Calculate uptime if online and last seen is available
		if dbBS.IsOnline && dbBS.LastSeenAt != nil {
			status.UptimeSeconds = int64(time.Since(*dbBS.LastSeenAt).Seconds())
		}

		// Add connection details
		status.ConnectionDetail = make(map[string]interface{})
		if dbBS.ServiceCenterURL != nil {
			status.ConnectionDetail["service_center_url"] = *dbBS.ServiceCenterURL
		}
		if dbBS.Vendor != nil {
			status.ConnectionDetail["vendor"] = *dbBS.Vendor
		}
		if dbBS.Model != nil {
			status.ConnectionDetail["model"] = *dbBS.Model
		}

		bs.Status = status

		baseStations = append(baseStations, bs)
	}

	return baseStations, nil
}
