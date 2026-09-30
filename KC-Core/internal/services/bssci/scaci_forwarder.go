package bssciservices

import (
	"context"
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // FormatEUI64 helper
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
)

// SCACIServerBroadcaster is the SCACI server surface the forwarder relays
// uplink data and downlink results onto.
type SCACIServerBroadcaster interface {
	BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error
	ApplicationCenterResults
}

// SCACIEPStatusServerBroadcaster is the SCACI server surface EPStatus messages
// are relayed onto.
type SCACIEPStatusServerBroadcaster interface {
	BroadcastEPStatus(ctx context.Context, tenantID int64, data *scaci.EPStatusData) error
}

// SCACIForwarderWithSetter is the broadcaster the BSSCI side consumes plus the
// typed wiring point the composition root calls once SCACI exists. The setter
// is typed: a previous untyped variant was reached through a runtime type
// assertion that could never succeed, leaving forwarding silently disconnected.
type SCACIForwarderWithSetter interface {
	bssci.SCACIBroadcaster
	ApplicationCenterResults
	SetSCACIServer(scaci SCACIServerBroadcaster)
}

type scaciForwarder struct {
	scaciServer SCACIServerBroadcaster
	mu          sync.RWMutex
	logger      logger.Logger
}

// NewSCACIForwarder creates a new SCACI forwarder
func NewSCACIForwarder(log logger.Logger) SCACIForwarderWithSetter {
	return &scaciForwarder{
		logger: log,
	}
}

// SetSCACIServer wires the SCACI server for broadcasting. SCACI is constructed
// after the forwarder, so the reference is supplied once at that point.
func (f *scaciForwarder) SetSCACIServer(scaci SCACIServerBroadcaster) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scaciServer = scaci
}

// BroadcastULData forwards uplink data to SCACI clients
// Delegates to real scaciBroadcaster.BroadcastULData
func (f *scaciForwarder) BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error {
	f.mu.RLock()
	scaci := f.scaciServer
	f.mu.RUnlock()

	if scaci == nil {
		// No SCACI server available - this is normal if no Application Centers are connected
		return nil
	}

	// Delegate to real SCACI broadcaster (preserves exact behavior)
	if err := scaci.BroadcastULData(ctx, tenantID, data); err != nil {
		f.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToForwardULDataToSCACI,
			logger.FieldEpEui, data.EpEui,
			logger.FieldError, err)
		// Do not fail BSSCI operation if SCACI forwarding fails
		return err
	}

	return nil
}

// BroadcastDLDataResult forwards a downlink result to queuer, the Application
// Center that queued the downlink, under acQueID.
func (f *scaciForwarder) BroadcastDLDataResult(ctx context.Context, queuer scaci.ApplicationCenter, acQueID uint64, result *mioty.DLDataResult) error {
	f.mu.RLock()
	scaciServer := f.scaciServer
	f.mu.RUnlock()

	if scaciServer == nil {
		// No SCACI server available - this is normal if no Application Centers are connected
		return nil
	}

	if err := scaciServer.BroadcastDLDataResult(ctx, queuer, acQueID, result); err != nil {
		f.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToBroadcastDLResultToSCACI,
			logger.FieldTenantID, queuer.TenantID,
			logger.FieldError, err)
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
	scaciServer SCACIEPStatusServerBroadcaster
	mu          sync.RWMutex
	logger      logger.Logger
}

// SCACIEPStatusAdapterWithSetter extends EPStatusBroadcaster with the typed
// wiring point for the SCACI server.
type SCACIEPStatusAdapterWithSetter interface {
	EPStatusBroadcaster
	SetSCACIServer(scaciSvr SCACIEPStatusServerBroadcaster)
}

// NewSCACIEPStatusAdapter creates a new adapter for EPStatus forwarding
func NewSCACIEPStatusAdapter(log logger.Logger) SCACIEPStatusAdapterWithSetter {
	return &scaciEPStatusAdapter{
		logger: log,
	}
}

// SetSCACIServer wires the SCACI server for EPStatus broadcasting. The typed
// parameter replaces an interface{} whose failed assertion was silently
// ignored, leaving EPStatus forwarding disconnected with no error.
func (a *scaciEPStatusAdapter) SetSCACIServer(scaciSvr SCACIEPStatusServerBroadcaster) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.scaciServer = scaciSvr
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
			logger.FieldEpEui, mioty.FormatEUI64(data.EpEui),
			logger.FieldError, err)
		return err
	}

	return nil
}
