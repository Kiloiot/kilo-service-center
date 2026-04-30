package bssciservices

import (
	"context"
	"sync"

	"github.com/kilocenter/KC-DB/storage/mioty"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	pkgmioty "github.com/kilocenter/KC-Core/pkg/mioty" // FormatEUI64 helper
	"github.com/kilocenter/KC-Core/pkg/scaci"
)

// scaciBroadcaster interface is defined in pkg/bssci/server.go
// We reference it via the bssci package
type scaciBroadcaster interface {
	BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error
	BroadcastDLDataResult(ctx context.Context, tenantID int64, result *mioty.DLDataResult) error
}

// scaciEPStatusBroadcaster interface for EPStatus forwarding
// Used to convert between bssci.EPStatusData and scaci.EPStatusData
type scaciEPStatusBroadcaster interface {
	BroadcastEPStatus(ctx context.Context, tenantID int64, data *scaci.EPStatusData) error
}

type scaciForwarder struct {
	scaciServer scaciBroadcaster // Real interface from server.go:89
	mu          sync.RWMutex
	logger      logger.Logger
}

// NewSCACIForwarder creates a new SCACI forwarder
func NewSCACIForwarder(log logger.Logger) bssci.SCACIBroadcaster {
	return &scaciForwarder{
		logger: log,
	}
}

// SetSCACI sets the SCACI server reference (for wiring after SCACI construction)
func (f *scaciForwarder) SetSCACI(scaci scaciBroadcaster) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scaciServer = scaci
}

// SetSCACIServer wires the SCACI server for broadcasting
// Public wrapper around SetSCACI for main.go usage
func (f *scaciForwarder) SetSCACIServer(scaci scaciBroadcaster) {
	f.SetSCACI(scaci)
}

// BroadcastULData forwards uplink data to SCACI clients
// Delegates to real scaciBroadcaster.BroadcastULData (server.go:39)
func (f *scaciForwarder) BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error {
	f.mu.RLock()
	scaci := f.scaciServer
	f.mu.RUnlock()

	if scaci == nil {
		// No SCACI server available - this is normal if no Application Centers are connected
		return nil
	}

	// Delegate to real SCACI broadcaster (preserves exact behavior from server.go:1884)
	if err := scaci.BroadcastULData(ctx, tenantID, data); err != nil {
		f.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToForwardULDataToSCACI,
			"epEui", data.EpEui,
			"error", err)
		// Do not fail BSSCI operation if SCACI forwarding fails
		return err
	}

	return nil
}

// BroadcastDLDataResult forwards downlink results to SCACI clients
// Delegates to real scaciBroadcaster.BroadcastDLDataResult (server.go:40)
func (f *scaciForwarder) BroadcastDLDataResult(ctx context.Context, tenantID int64, result *mioty.DLDataResult) error {
	f.mu.RLock()
	scaci := f.scaciServer
	f.mu.RUnlock()

	if scaci == nil {
		// No SCACI server available - this is normal if no Application Centers are connected
		return nil
	}

	// Delegate to real SCACI broadcaster (preserves exact behavior from downlink_handlers.go:340)
	if err := scaci.BroadcastDLDataResult(ctx, tenantID, result); err != nil {
		f.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToBroadcastDLResultToSCACI,
			"tenantID", tenantID,
			"error", err)
		// Do not fail BSSCI operation if SCACI forwarding fails
		return err
	}

	return nil
}

// ============================================================================
// EPStatus Broadcaster Adapter (SCACI §3.13)
// ============================================================================

// scaciEPStatusAdapter adapts SCACI server to bssci.SCACIEPStatusBroadcaster interface
// Converts between bssci.EPStatusData and scaci.EPStatusData types
type scaciEPStatusAdapter struct {
	scaciServer scaciEPStatusBroadcaster
	mu          sync.RWMutex
	logger      logger.Logger
}

// SCACIEPStatusAdapterWithSetter extends bssci.SCACIEPStatusBroadcaster with server wiring
type SCACIEPStatusAdapterWithSetter interface {
	bssci.SCACIEPStatusBroadcaster
	SetSCACIServer(scaciSvr interface{})
}

// NewSCACIEPStatusAdapter creates a new adapter for EPStatus forwarding
func NewSCACIEPStatusAdapter(log logger.Logger) SCACIEPStatusAdapterWithSetter {
	return &scaciEPStatusAdapter{
		logger: log,
	}
}

// SetSCACIServer wires the SCACI server for EPStatus broadcasting
// Accepts interface{} to allow main.go type assertion without import cycles
func (a *scaciEPStatusAdapter) SetSCACIServer(scaciSvr interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if svr, ok := scaciSvr.(scaciEPStatusBroadcaster); ok {
		a.scaciServer = svr
	}
}

// BroadcastEPStatus implements bssci.SCACIEPStatusBroadcaster interface
// Converts bssci.EPStatusData to scaci.EPStatusData and forwards to SCACI server
func (a *scaciEPStatusAdapter) BroadcastEPStatus(ctx context.Context, tenantID int64, data *bssci.EPStatusData) error {
	a.mu.RLock()
	scaciSvr := a.scaciServer
	a.mu.RUnlock()

	if scaciSvr == nil {
		// No SCACI server available - this is normal if no Application Centers are connected
		return nil
	}

	if data == nil {
		return nil
	}

	// Convert bssci.EPStatusData to scaci.EPStatusData
	scaciData := &scaci.EPStatusData{
		EpEui:      data.EpEui,
		EpStatus:   data.EpStatus,
		AttachCnt:  data.AttachCnt,
		Nonce:      data.Nonce,
		Sign:       data.Sign,
		Snr:        data.Snr,
		Rssi:       data.Rssi,
		EqSnr:      data.EqSnr,
		Subpackets: data.Subpackets,
	}

	// Delegate to real SCACI broadcaster
	if err := scaciSvr.BroadcastEPStatus(ctx, tenantID, scaciData); err != nil {
		a.logger.ErrorContext(ctx, bssci.LogBSSCIEPStatusForwardFailed,
			"epEui", pkgmioty.FormatEUI64(data.EpEui),
			"error", err)
		return err
	}

	return nil
}
