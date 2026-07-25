package bssciservices

import (
	"context"
	"time"

	"encoding/binary"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// connectionRegistry adapts basestation.ConnectionManager to the narrow
// bssci.BaseStationConnectionRegistry contract the connect flow consumes.
type connectionRegistry struct {
	manager *basestation.ConnectionManager
	logger  logger.Logger
}

// NewConnectionRegistry captures the concrete connection manager so callers
// depend only on the registry contract.
func NewConnectionRegistry(manager *basestation.ConnectionManager, log logger.Logger) bssci.BaseStationConnectionRegistry {
	return &connectionRegistry{
		manager: manager,
		logger:  log,
	}
}

// GetBaseStationGlobal retrieves a base station by EUI across all tenants.
// Used during the BSSCI connect handshake before the tenant is resolved.
func (c *connectionRegistry) GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*basestation.BaseStation, error) {
	bs, err := c.manager.GetBaseStationGlobal(ctx, eui)
	if err != nil || bs == nil {
		c.logger.ErrorContext(ctx, bssci.LogBSSCIBaseStationNotFoundInDatabase,
			"euiHex", pkgmioty.FormatEUI64(binary.BigEndian.Uint64(eui[:])),
			"error", err)
		return nil, bssci.NewCatalogError(bssci.ErrBaseStationNotRegistered, bssci.POSIX_EPERM)
	}

	return bs, nil
}

// RegisterConnection publishes the session's live connection and marks the
// base station online.
func (c *connectionRegistry) RegisterConnection(ctx context.Context, session *bssci.Session, _ *basestation.BaseStation) error {
	status := &basestation.ConnectionStatus{
		IsOnline:       true,
		LastSeen:       time.Now(),
		ConnectionType: basestation.ConnectionTypeBSSCI,
		SessionID:      session.ID,
	}

	euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()

	if err := c.manager.UpdateConnectionStatus(ctx, euiBytes, status); err != nil {
		c.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateConnectionStatus,
			"error", err,
			"bsEui", session.BaseStationEUI)
		return err
	}

	return nil
}

// DisconnectBaseStationIfCurrent marks the base station offline only while the
// given connection is still its current one.
func (c *connectionRegistry) DisconnectBaseStationIfCurrent(ctx context.Context, eui [8]byte, connectionID string) error {
	if c.manager == nil {
		return nil
	}
	return c.manager.DisconnectBaseStationIfCurrent(ctx, eui, connectionID)
}

// UpdateLastSeen refreshes the base station's last-seen timestamp.
func (c *connectionRegistry) UpdateLastSeen(ctx context.Context, eui [8]byte) error {
	if c.manager == nil {
		return nil
	}
	return c.manager.UpdateLastSeen(ctx, eui)
}
