package bssciservices

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ConnectionStateManager is the connection-manager capability the connect
// flow consumes: global station lookup during the handshake, liveness
// updates, and connection-scoped disconnect.
type ConnectionStateManager interface {
	GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*basestation.BaseStation, error)
	UpdateLastSeen(ctx context.Context, eui [8]byte) error
	UpdateConnectionStatus(ctx context.Context, eui [8]byte, status *basestation.ConnectionStatus) error
	DisconnectBaseStationIfCurrent(ctx context.Context, eui [8]byte, connectionID string) error
}

// connectionRegistry adapts the connection manager to the narrow
// bssci.BaseStationConnectionRegistry contract the connect flow consumes.
type connectionRegistry struct {
	manager ConnectionStateManager
	logger  logger.Logger
}

// NewConnectionRegistry captures the connection manager so callers depend
// only on the registry contract.
func NewConnectionRegistry(manager ConnectionStateManager, log logger.Logger) bssci.BaseStationConnectionRegistry {
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
			logger.FieldEuiHex, mioty.FormatEUI64(binary.BigEndian.Uint64(eui[:])),
			logger.FieldError, err)
		return nil, bssci.NewCatalogError(bssci.ErrBaseStationNotRegistered, bssci.POSIX_EPERM)
	}

	return bs, nil
}

// RegisterConnection publishes the session's live connection, marks the base
// station online and records when the session's connect handshake completed.
func (c *connectionRegistry) RegisterConnection(ctx context.Context, session *bssci.Session, _ *basestation.BaseStation) error {
	activatedAt := time.Now()
	status := &basestation.ConnectionStatus{
		IsOnline:         true,
		LastSeen:         activatedAt,
		ConnectionType:   basestation.ConnectionTypeBSSCI,
		SessionID:        session.ID,
		SessionStartedAt: activatedAt,
	}

	euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()

	if err := c.manager.UpdateConnectionStatus(ctx, euiBytes, status); err != nil {
		c.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateConnectionStatus,
			logger.FieldError, err,
			logger.FieldBsEui, session.BaseStationEUI)
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
