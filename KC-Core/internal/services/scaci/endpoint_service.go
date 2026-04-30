// Package scaciservices implements the SCACI (Service Center Application Center Interface) protocol server.
//
// endpoint_service.go implements EndpointService interface.
//
// Extracted Logic:
//   - Register endpoint (from handler_operations.go:41-178)
//   - Deregister endpoint (from handler_operations.go:220-316)
//   - BSSCI detach propagation integration
//
// Dependencies (injected):
//   - interfaces.EndPointRepository: Endpoint persistence
//   - DetachPropagator: BSSCI integration for detach propagation
//   - logger.Logger: Structured logging
//
// Error Handling:
//   - Returns error tokens from errors_catalog.go
//   - NO Go errors returned from public methods
//   - Transport layer resolves tokens → POSIX codes
//
// Infrastructure Reuse:
//   - errors_catalog.go: All error tokens (err*)
//   - log_messages.go: All log constants (Log*)
//   - endpoint.DetachEndpoint: Shared detach helper
//   - All writes use canonical BSSCI paths
package scaciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/kilocenter/KC-Core/pkg/endpoint"
	"github.com/kilocenter/KC-Core/pkg/logger"
	pkgmioty "github.com/kilocenter/KC-Core/pkg/mioty" // FormatEUI64 helper
	"github.com/kilocenter/KC-Core/pkg/scaci"
	dbconfig "github.com/kilocenter/KC-DB/common/config"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// endpointService implements EndpointService interface
//
// This service manages SCACI endpoint lifecycle (register, deregister) per
// MIOTY §3.6-3.7. All database writes funnel through canonical repository methods.
//
// Tenant Isolation:
//   - All operations scoped to tenantID
//   - Endpoints cannot be accessed across tenant boundaries
//
// BSSCI Integration:
//   - Deregister triggers DetachPropagator.SendDetachPropagateToAll()
//   - This ensures all connected base stations clear endpoint state
type endpointService struct {
	endpointRepo     interfaces.EndpointRepository // Endpoint persistence
	detachPropagator scaci.DetachPropagator        // BSSCI detach propagation
	logger           logger.Logger
}

// NewEndpointService creates a new endpoint service
//
// Parameters:
//   - endpointRepo: Endpoint repository for database operations
//   - detachPropagator: BSSCI server for detach propagation (can be nil)
//   - logger: Structured logger
//
// Returns:
//   - EndpointService: Service instance implementing interface
func NewEndpointService(
	endpointRepo interfaces.EndpointRepository,
	detachPropagator scaci.DetachPropagator,
	log logger.Logger,
) scaci.EndpointService {
	return &endpointService{
		endpointRepo:     endpointRepo,
		detachPropagator: detachPropagator,
		logger:           log,
	}
}

// Register implements EndpointService.Register
//
// Extracted from handler_operations.go:41-178
//
// Flow:
//  1. Validate mandatory fields (epEui, nwkKey length)
//  2. Check if endpoint exists (tenant-scoped lookup)
//  3. Create new endpoint if not found
//  4. Update MIOTY fields (bidi, preAttach, shAddr, etc.)
//  5. Log success
//
// Persistence:
//   - Create: endpointRepo.Create() for new endpoints
//   - Update: endpointRepo.UpdateFields() for MIOTY parameters
//   - All writes are tenant-scoped
//
// Parameters:
//   - ctx: Request context with timeout
//   - req: Decoded Register message from wire
//   - tenantID: Tenant scope for endpoint ownership
//
// Returns:
//   - string: Error token if registration fails, "" on success
func (es *endpointService) Register(
	ctx context.Context,
	req *scaci.Register,
	tenantID int64,
) string {
	// Step 1: Validate mandatory fields per SCACI §3.6.1
	if req.EpEui == 0 {
		return scaci.ErrMissingEpEui
	}
	if len(req.NwkKey) != 16 {
		return scaci.ErrInvalidNwkKeyLength
	}

	// Step 1b: Validate bounds per SCACI §3.6.1 before DB write
	// ShAddr is uint16 on wire (0-65535), fits in INTEGER column with CHECK constraint
	// AttachCnt/PacketCnt are uint32 on wire (0-4294967295), fit in BIGINT column with CHECK constraint
	// Note: Wire types (uint16/uint32) already constrain these values at the Go type level,
	// so explicit bounds checks are not needed here. DB CHECK constraints provide defense in depth.

	// Step 2: Convert EUI to models.EUI for repository calls
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], req.EpEui)

	// Step 3: Create DB context with timeout
	dbCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Step 4: Try to get existing endpoint (tenant-scoped)
	endpoint, err := es.endpointRepo.GetByEUI(dbCtx, tenantID, eui[:])

	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		es.logger.Error(scaci.LogSCACIDatabaseErrorRegister,
			"epEui", pkgmioty.FormatEUI64(req.EpEui),
			"tenantId", tenantID,
			"error", err)
		return scaci.ErrDatabaseError
	}

	// Step 5: Build updates map with exact DB column names
	// These are MIOTY protocol fields from SCACI §3.6.1
	// Cast to signed types for DB driver: INTEGER (int32) for shAddr, BIGINT (int64) for counters
	updates := map[string]interface{}{
		"nwk_key":         req.NwkKey[:],
		"pre_attach":      req.PreAttach,
		"bidi":            req.Bidi,
		"sh_addr":         int32(req.ShAddr),    // uint16 → int32 (safe, DB is INTEGER with CHECK 0-65535)
		"attach_cnt":      int64(req.AttachCnt), // uint32 → int64 (safe, DB is BIGINT with CHECK 0-4294967295)
		"packet_cnt":      int64(req.PacketCnt), // uint32 → int64 (safe, DB is BIGINT with CHECK 0-4294967295)
		"last_packet_cnt": int64(req.PacketCnt), // uint32 → int64 (safe, DB is BIGINT with CHECK 0-4294967295)
		"dual_chan":       req.DualChan,
		"repetition":      req.Repetition,
		"wide_carr_off":   req.WideCarrOff,
		"long_blk_dist":   req.LongBlkDist,
	}

	// Step 6: Create new endpoint if not found
	if endpoint == nil {
		// SCACI §3.6 Register provides nwkKey; copy from request
		// AppKey is initialized to zeros (will be provided by separate operation)
		newEndpoint := &models.EndPoint{
			EUI:      eui,
			Name:     fmt.Sprintf("EP-%016X", req.EpEui),
			TenantID: tenantID,
			NwkSnKey: append([]byte(nil), req.NwkKey[:]...), // Copy network key from request
			AppKey:   make([]byte, 16),                      // Initialize to zeros (valid 16 bytes)
			Tags:     make(map[string]string),
		}
		if err := es.endpointRepo.Create(dbCtx, newEndpoint); err != nil {
			es.logger.Error(scaci.LogSCACICreateEndpointFailed,
				"epEui", pkgmioty.FormatEUI64(req.EpEui),
				"tenantId", tenantID,
				"error", err)
			return scaci.ErrFailedCreateEndpoint
		}
		endpoint = newEndpoint
		es.logger.Info(scaci.LogSCACIEndpointCreated,
			"epEui", pkgmioty.FormatEUI64(req.EpEui),
			"tenantId", tenantID)
	}

	// Step 7: Apply MIOTY field updates (both create and update paths)
	if err := es.endpointRepo.UpdateFields(dbCtx, tenantID, endpoint.ID, updates); err != nil {
		es.logger.Error(scaci.LogSCACIUpdateEndpointFailed,
			"epEui", pkgmioty.FormatEUI64(req.EpEui),
			"tenantId", tenantID,
			"endpointId", endpoint.ID,
			"error", err)
		return scaci.ErrFailedUpdateEndpoint
	}

	es.logger.Info(scaci.LogSCACIEndpointRegistered,
		"epEui", pkgmioty.FormatEUI64(req.EpEui),
		"tenantId", tenantID,
		"bidi", req.Bidi,
		"preAttach", req.PreAttach)

	return ""
}

// Deregister implements EndpointService.Deregister
//
// Extracted from handler_operations.go:220-316
//
// Flow:
//  1. Validate epEui is non-zero
//  2. Look up endpoint (tenant-scoped)
//  3. Call shared detach helper (marks endpoint inactive)
//  4. Log success
//
// NOTE: BSSCI detach propagation is handled by handleDeregisterComplete, NOT here.
// This maintains spec-compliant three-way handshake semantics (wait for AC confirmation
// before broadcasting to all base stations).
//
// Parameters:
//   - ctx: Request context
//   - epEui: Endpoint EUI to deregister
//   - tenantID: Tenant scope for ownership validation
//
// Returns:
//   - string: Error token (errEndpointNotFound, errDatabaseError) or "" on success
func (es *endpointService) Deregister(
	ctx context.Context,
	epEui uint64,
	tenantID int64,
) string {
	// Step 1: Validate epEui is non-zero
	if epEui == 0 {
		return scaci.ErrMissingEpEui
	}

	// Step 2: Convert EUI to models.EUI
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], epEui)

	// Step 3: Create DB context with timeout
	dbCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Step 4: Look up endpoint with tenant scoping
	endpointRecord, err := es.endpointRepo.GetByEUI(dbCtx, tenantID, eui[:])
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			es.logger.Warn(scaci.LogSCACIEndpointNotFoundDeregister,
				"epEui", pkgmioty.FormatEUI64(epEui),
				"tenantId", tenantID)
			return scaci.ErrEndpointNotFound
		}
		es.logger.Error(scaci.LogSCACIDatabaseErrorDeregister,
			"epEui", pkgmioty.FormatEUI64(epEui),
			"tenantId", tenantID,
			"error", err)
		return scaci.ErrDatabaseError
	}

	// Step 5: Call shared detach helper (marks endpoint inactive, no telemetry)
	if err := endpoint.DetachEndpoint(dbCtx, es.endpointRepo, tenantID, endpointRecord.ID, nil); err != nil {
		es.logger.Error(scaci.LogSCACIDetachEndpointFailed,
			"epEui", pkgmioty.FormatEUI64(epEui),
			"tenantId", tenantID,
			"error", err)
		return scaci.ErrFailedUpdateEndpoint
	}

	es.logger.Info(scaci.LogSCACIEndpointDeregistered,
		"epEui", pkgmioty.FormatEUI64(epEui),
		"tenantId", tenantID)

	return ""
}

// GetByEUI implements EndpointService.GetByEUI
//
// Provides a service-layer wrapper around the repository's GetByEUI method
// for use by SCACI handlers that need to look up endpoint state.
//
// Flow:
//  1. Validate EUI is non-empty
//  2. Create DB context with timeout
//  3. Call repository GetByEUI
//  4. Map errors to SCACI error tokens
//
// Parameters:
//   - ctx: Request context
//   - tenantID: Tenant scope for endpoint lookup
//   - eui: Endpoint EUI (8-byte slice)
//
// Returns:
//   - *models.EndPoint: Endpoint record if found, nil if error
//   - string: Error token (errEndpointNotFound, errDatabaseError) or "" on success
func (es *endpointService) GetByEUI(
	ctx context.Context,
	tenantID int64,
	eui []byte,
) (*models.EndPoint, string) {
	// Step 1: Validate EUI length
	if len(eui) != 8 {
		es.logger.Error("Invalid EUI length for GetByEUI",
			"length", len(eui),
			"tenantId", tenantID)
		return nil, scaci.ErrMissingEpEui
	}

	// Step 2: Create DB context with timeout
	dbCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Step 3: Call repository
	endpoint, err := es.endpointRepo.GetByEUI(dbCtx, tenantID, eui)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			es.logger.Debug("Endpoint not found in GetByEUI",
				"tenantId", tenantID)
			return nil, scaci.ErrEndpointNotFound
		}
		es.logger.Error(scaci.LogSCACIDatabaseErrorRegister,
			"tenantId", tenantID,
			"error", err)
		return nil, scaci.ErrDatabaseError
	}

	return endpoint, ""
}

// GetGlobal implements EndpointService.GetGlobal
//
// Cross-tenant endpoint lookup for roaming support, matching BSSCI pattern
// at server.go:5197-5202. Used when tenant-scoped lookup fails.
//
// Parameters:
//   - ctx: Request context
//   - eui: Endpoint EUI (8-byte slice)
//
// Returns:
//   - *models.EndPoint: Endpoint record if found, nil if error
//   - string: Error token (ErrEndpointNotFound, ErrDatabaseError) or "" on success
func (es *endpointService) GetGlobal(
	ctx context.Context,
	eui []byte,
) (*models.EndPoint, string) {
	// Step 1: Validate EUI length (matching GetByEUI pattern)
	if len(eui) != 8 {
		es.logger.Error("Invalid EUI length for GetGlobal",
			"length", len(eui))
		return nil, scaci.ErrMissingEpEui
	}

	// Step 2: Create DB context with timeout (matching GetByEUI pattern)
	dbCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Step 3: Convert to models.EUI for repository call
	var euiModel models.EUI
	copy(euiModel[:], eui)

	// Step 4: Call repository's cross-tenant Get method (interfaces/repository.go:18)
	endpoint, err := es.endpointRepo.Get(dbCtx, euiModel)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			es.logger.Debug("Endpoint not found in GetGlobal (cross-tenant)")
			return nil, scaci.ErrEndpointNotFound
		}
		es.logger.Error(scaci.LogSCACIDatabaseErrorRegister,
			"error", err)
		return nil, scaci.ErrDatabaseError
	}

	return endpoint, ""
}

// PropagateDetachToAll implements EndpointService.PropagateDetachToAll
//
// Encapsulates detachPropagator so handlers don't need direct access to BSSCI server.
// Triggers BSSCI detach propagation to all connected base stations.
// Called by handleDeregisterComplete after AC confirms endpoint deregistration.
//
// Parameters:
//   - epEui: Endpoint EUI to detach from all base stations
//
// Returns:
//   - []error: Slice of errors (one per failed BS), empty if all succeeded or propagator nil
func (es *endpointService) PropagateDetachToAll(epEui uint64) []error {
	if es.detachPropagator == nil {
		es.logger.Debug("DetachPropagator not available, skipping propagation",
			"epEui", pkgmioty.FormatEUI64(epEui))
		return nil
	}

	return es.detachPropagator.SendDetachPropagateToAll(epEui)
}
