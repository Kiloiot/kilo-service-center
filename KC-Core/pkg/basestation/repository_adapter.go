package basestation

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// columnServiceCenterURL is the base station column the read backfill writes.
const columnServiceCenterURL = "service_center_url"

// RecordStore covers the repository operations the adapter uses to
// persist and read base station records. Satisfied structurally by the KC-DB
// base station repository.
type RecordStore interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
	GetByEUIGlobal(ctx context.Context, eui []byte) (*models.BaseStation, error)
	Update(ctx context.Context, tenantID, id int64, updates map[string]interface{}) error
}

// RepositoryAdapter adapts the KC-DB base station repository to the Store interface
type RepositoryAdapter struct {
	repo                      RecordStore
	tenantID                  int64
	canonicalServiceCenterURL string
	logger                    logger.Logger
}

// NewRepositoryAdapter creates a new repository adapter
func NewRepositoryAdapter(repo RecordStore, tenantID int64, canonicalURL string, log logger.Logger) *RepositoryAdapter {
	return &RepositoryAdapter{
		repo:                      repo,
		tenantID:                  tenantID,
		canonicalServiceCenterURL: canonicalURL,
		logger:                    log,
	}
}

// effectiveTenant returns the tenant ID from context if available, otherwise falls back to the configured default
func (ra *RepositoryAdapter) effectiveTenant(ctx context.Context) int64 {
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil && tid > 0 {
		return tid
	}
	return ra.tenantID
}

// GetBaseStation implements BaseStationStore interface
func (ra *RepositoryAdapter) GetBaseStation(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	tenant := ra.effectiveTenant(ctx)
	euiBytes := eui[:]
	dbBaseStation, err := ra.repo.GetByEUI(ctx, tenant, euiBytes)
	if err != nil {
		return nil, fmt.Errorf(errFmtGetBasestation, err)
	}

	if dbBaseStation == nil {
		return nil, errBasestationNotFound
	}

	return ra.convertDBBaseStation(ctx, dbBaseStation), nil
}

// GetBaseStationGlobal retrieves a Base Station by EUI across all tenants.
// Used during BSSCI connect handshake before tenant is resolved.
func (ra *RepositoryAdapter) GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	euiBytes := eui[:]
	dbBaseStation, err := ra.repo.GetByEUIGlobal(ctx, euiBytes)
	if err != nil {
		return nil, fmt.Errorf(errFmtGetBasestation, err)
	}

	if dbBaseStation == nil {
		return nil, errBasestationNotFound
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

	// Backfill missing or non-canonical URLs of BSSCI stations on read; no known URL is stored as NULL.
	canonical := config.StoredServiceCenterURL(ra.canonicalServiceCenterURL)
	if dbBaseStation.ConnectionType == models.ConnectionTypeBSSCI && !sameServiceCenterURL(dbBaseStation.ServiceCenterURL, canonical) {
		bs.ServiceCenterURL = ra.canonicalServiceCenterURL
		ra.backfillServiceCenterURL(ctx, dbBaseStation, canonical)
	}

	return bs
}

// backfillServiceCenterURL stores the canonical URL; a failure leaves the read intact and is logged.
func (ra *RepositoryAdapter) backfillServiceCenterURL(ctx context.Context, dbBaseStation *models.BaseStation, canonical *string) {
	err := ra.repo.Update(ctx, dbBaseStation.TenantID, dbBaseStation.ID, map[string]interface{}{
		columnServiceCenterURL: canonical,
	})
	if err != nil {
		ra.logger.WarnContext(ctx, LogFailedToBackfillServiceCenterURL,
			logger.FieldEui, mioty.FormatEUIBytes(dbBaseStation.EUI[:]),
			logger.FieldTenantID, dbBaseStation.TenantID,
			logger.FieldError, err)
	}
}

// UpdateConnectionStatus implements BaseStationStore interface
func (ra *RepositoryAdapter) UpdateConnectionStatus(ctx context.Context, eui [8]byte, status *ConnectionStatus) error {
	tenant := ra.effectiveTenant(ctx)
	euiBytes := eui[:]

	dbBaseStation, err := ra.repo.GetByEUI(ctx, tenant, euiBytes)
	if err != nil {
		return fmt.Errorf(errFmtGetBasestation, err)
	}

	if dbBaseStation == nil {
		return errBasestationNotFound
	}

	// Update connection status
	updates := map[string]interface{}{
		fieldKeyIsOnline:       status.IsOnline,
		fieldKeyLastSeenAt:     status.LastSeen,
		fieldKeyConnectionType: string(status.ConnectionType),
	}

	if status.SessionID != "" {
		updates[fieldKeySessionUUID] = status.SessionID
	}
	if !status.SessionStartedAt.IsZero() {
		updates[fieldKeySessionStartedAt] = status.SessionStartedAt
	}

	return ra.repo.Update(ctx, tenant, dbBaseStation.ID, updates)
}

// DisconnectIfCurrent implements the conditional offline transition: the
// update applies only while the stored session_uuid still belongs to the
// disconnecting connection, so late cleanup from a replaced connection never
// marks the newer session offline.
func (ra *RepositoryAdapter) DisconnectIfCurrent(ctx context.Context, eui [8]byte, connectionID string, lastSeen time.Time) (bool, error) {
	tenant := ra.effectiveTenant(ctx)
	dbBaseStation, err := ra.repo.GetByEUI(ctx, tenant, eui[:])
	if err != nil {
		return false, fmt.Errorf(errFmtGetBasestation, err)
	}
	if dbBaseStation == nil {
		return false, nil
	}
	if dbBaseStation.SessionUUID == nil || *dbBaseStation.SessionUUID != connectionID {
		return false, nil
	}

	updates := map[string]interface{}{
		fieldKeyIsOnline:   false,
		fieldKeyLastSeenAt: lastSeen,
	}
	if err := ra.repo.Update(ctx, tenant, dbBaseStation.ID, updates); err != nil {
		return false, err
	}
	return true, nil
}

func sameServiceCenterURL(stored, canonical *string) bool {
	if stored == nil || canonical == nil {
		return stored == canonical
	}
	return *stored == *canonical
}
