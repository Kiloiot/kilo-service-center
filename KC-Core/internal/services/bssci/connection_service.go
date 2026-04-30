package bssciservices

import (
	"context"
	"time"

	"github.com/kilocenter/KC-Core/pkg/basestation"
	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
)

type connectionService struct {
	logger logger.Logger
}

// NewConnectionService creates a new connection service
func NewConnectionService(log logger.Logger) bssci.ConnectionService {
	return &connectionService{
		logger: log,
	}
}

// GetBaseStation retrieves basestation via ConnectionManager (tenant-scoped via context)
func (c *connectionService) GetBaseStation(ctx context.Context, eui [8]byte, mgr *basestation.ConnectionManager) (*basestation.BaseStation, error) {
	bs, err := mgr.GetBaseStation(ctx, eui)
	if err != nil || bs == nil {
		c.logger.Error(bssci.LogBSSCIBaseStationNotFoundInDatabase,
			"euiHex", formatEUI(eui),
			"error", err)
		return nil, bssci.NewCatalogError(bssci.ErrBaseStationNotRegistered, bssci.POSIX_EPERM)
	}

	return bs, nil
}

// GetBaseStationGlobal retrieves basestation by EUI across all tenants.
// Used during BSSCI connect handshake before tenant is resolved.
func (c *connectionService) GetBaseStationGlobal(ctx context.Context, eui [8]byte, mgr *basestation.ConnectionManager) (*basestation.BaseStation, error) {
	bs, err := mgr.GetBaseStationGlobal(ctx, eui)
	if err != nil || bs == nil {
		c.logger.Error(bssci.LogBSSCIBaseStationNotFoundInDatabase,
			"euiHex", formatEUI(eui),
			"error", err)
		return nil, bssci.NewCatalogError(bssci.ErrBaseStationNotRegistered, bssci.POSIX_EPERM)
	}

	return bs, nil
}

// RegisterConnection updates basestation connection status via REAL ConnectionManager
// Real path from server.go:967-979
func (c *connectionService) RegisterConnection(ctx context.Context, session *bssci.Session, _ *basestation.BaseStation, mgr *basestation.ConnectionManager) error {
	// Build connection status matching the real ConnectionStatus struct
	status := &basestation.ConnectionStatus{
		IsOnline:       true,
		LastSeen:       time.Now(),
		ConnectionType: basestation.ConnectionTypeBSSCI,
		SessionID:      session.ID,
	}

	// Convert EUI to [8]byte for UpdateConnectionStatus
	var euiBytes [8]byte
	for i := 7; i >= 0; i-- {
		euiBytes[i] = byte(session.BaseStationEUI >> (8 * (7 - i)))
	}

	err := mgr.UpdateConnectionStatus(ctx, euiBytes, status)
	if err != nil {
		c.logger.Error("Failed to update basestation connection status",
			"error", err,
			"bsEui", session.BaseStationEUI)
		return err
	}

	return nil
}

// formatEUI formats an 8-byte EUI as a hex string
func formatEUI(eui [8]byte) string {
	return string([]byte{
		hexDigit(eui[0] >> 4), hexDigit(eui[0] & 0x0F),
		hexDigit(eui[1] >> 4), hexDigit(eui[1] & 0x0F),
		hexDigit(eui[2] >> 4), hexDigit(eui[2] & 0x0F),
		hexDigit(eui[3] >> 4), hexDigit(eui[3] & 0x0F),
		hexDigit(eui[4] >> 4), hexDigit(eui[4] & 0x0F),
		hexDigit(eui[5] >> 4), hexDigit(eui[5] & 0x0F),
		hexDigit(eui[6] >> 4), hexDigit(eui[6] & 0x0F),
		hexDigit(eui[7] >> 4), hexDigit(eui[7] & 0x0F),
	})
}

func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}
