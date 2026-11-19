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
//   - EndpointStore: Endpoint persistence
//   - DetachPropagator: BSSCI integration for detach propagation
//   - AttachmentDecider: Records and announces attach and detach decisions
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

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // FormatEUI64 helper
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
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
	endpointRepo     EndpointStore          // Endpoint persistence
	detachPropagator scaci.DetachPropagator // BSSCI detach propagation
	decider          AttachmentDecider      // Records and announces attach and detach decisions
	logger           logger.Logger
}

// NewEndpointService creates the endpoint service over the endpoint store,
// the BSSCI detach propagation and the attachment decider; each is required.
func NewEndpointService(
	endpointRepo EndpointStore,
	detachPropagator scaci.DetachPropagator,
	decider AttachmentDecider,
	log logger.Logger,
) (scaci.EndpointService, error) {
	switch {
	case endpointRepo == nil:
		return nil, errNilEndpointStore
	case detachPropagator == nil:
		return nil, errNilDetachPropagator
	case decider == nil:
		return nil, errNilAttachmentDecider
	case log == nil:
		return nil, errNilEndpointServiceLogger
	}
	return &endpointService{
		endpointRepo:     endpointRepo,
		detachPropagator: detachPropagator,
		decider:          decider,
		logger:           log,
	}, nil
}

// Register implements EndpointService.Register
//
// Extracted from handler_operations.go:41-178
//
// Flow:
//  1. Validate mandatory fields (epEui)
//  2. Check if endpoint exists (tenant-scoped lookup)
//  3. Create new endpoint if not found
//  4. Update MIOTY fields (bidi, preAttach, shAddr, etc.)
//  5. Log success
//
// Persistence:
//   - Create: endpointRepo.Create() for new endpoints
//   - Update: endpointRepo.EndpointRegistrationUpdate() for MIOTY parameters
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
	// Step 1: Validate mandatory fields per SCACI §3.6.1; the decoder already
	// refused a nwkKey that is not Numeric[16].
	if req.EpEui == 0 {
		return scaci.ErrMissingEpEui
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
		es.logger.ErrorContext(ctx, scaci.LogSCACIDatabaseErrorRegister,
			logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
			logger.FieldTenantIDCamel, tenantID,
			logger.FieldError, err)
		return scaci.ErrDatabaseError
	}

	// Step 5: Build the MIOTY registration field set from SCACI §3.6.1.
	// The repository encrypts nwk_key at rest before the row is written.
	registration := models.EndpointRegistrationParams{
		NwkKey:      req.NwkKey[:],
		PreAttach:   req.PreAttach,
		Bidi:        req.Bidi,
		ShAddr:      req.ShAddr,
		AttachCnt:   req.AttachCnt,
		PacketCnt:   req.PacketCnt,
		DualChan:    req.DualChan,
		Repetition:  req.Repetition,
		WideCarrOff: req.WideCarrOff,
		LongBlkDist: req.LongBlkDist,
	}

	// Step 6: Create new endpoint if not found
	if endpoint == nil {
		// SCACI §3.6 Register provides nwkKey; copy from request. AppKey stays
		// nil so the column is SQL NULL until a real application key is
		// provisioned by a separate operation - never an all-zero placeholder.
		newEndpoint := &models.EndPoint{
			EUI:      eui,
			Name:     fmt.Sprintf("EP-%s", mioty.FormatEUI64(req.EpEui)),
			TenantID: tenantID,
			Bidi:     req.Bidi,
			NwkSnKey: append([]byte(nil), req.NwkKey[:]...), // Copy network key from request
			AppKey:   nil,
			Tags:     make(map[string]string),
		}
		if err := es.endpointRepo.Create(dbCtx, newEndpoint); err != nil {
			es.logger.ErrorContext(ctx, scaci.LogSCACICreateEndpointFailed,
				logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
				logger.FieldTenantIDCamel, tenantID,
				logger.FieldError, err)
			return scaci.ErrFailedCreateEndpoint
		}
		endpoint = newEndpoint
		es.logger.InfoContext(ctx, scaci.LogSCACIEndpointCreated,
			logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
			logger.FieldTenantIDCamel, tenantID)
	}

	// Step 7: Apply MIOTY field updates (both create and update paths)
	if err := es.endpointRepo.EndpointRegistrationUpdate(dbCtx, tenantID, endpoint.ID, registration); err != nil {
		es.logger.ErrorContext(ctx, scaci.LogSCACIUpdateEndpointFailed,
			logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
			logger.FieldTenantIDCamel, tenantID,
			logger.FieldEndpointIDCamel, endpoint.ID,
			logger.FieldError, err)
		return scaci.ErrFailedUpdateEndpoint
	}

	es.logger.InfoContext(ctx, scaci.LogSCACIEndpointRegistered,
		logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
		logger.FieldTenantIDCamel, tenantID,
		logger.FieldBidi, req.Bidi,
		logger.FieldPreAttach, req.PreAttach)

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
			es.logger.WarnContext(ctx, scaci.LogSCACIEndpointNotFoundDeregister,
				logger.FieldEpEui, mioty.FormatEUI64(epEui),
				logger.FieldTenantIDCamel, tenantID)
			return scaci.ErrEndpointNotFound
		}
		es.logger.ErrorContext(ctx, scaci.LogSCACIDatabaseErrorDeregister,
			logger.FieldEpEui, mioty.FormatEUI64(epEui),
			logger.FieldTenantIDCamel, tenantID,
			logger.FieldError, err)
		return scaci.ErrDatabaseError
	}

	// Step 5: Detach the endpoint; the decider announces a change once (SCACI §3.13)
	_, err = es.decider.Decide(dbCtx, bssci.AttachmentDecision{
		TenantID:   tenantID,
		EndpointID: endpointRecord.ID,
		EpEUI:      epEui,
		Status:     endpoint.EndpointStatusDetached,
	})
	if err != nil {
		es.logger.ErrorContext(ctx, scaci.LogSCACIDetachEndpointFailed,
			logger.FieldEpEui, mioty.FormatEUI64(epEui),
			logger.FieldTenantIDCamel, tenantID,
			logger.FieldError, err)
		return scaci.ErrFailedUpdateEndpoint
	}

	es.logger.InfoContext(ctx, scaci.LogSCACIEndpointDeregistered,
		logger.FieldEpEui, mioty.FormatEUI64(epEui),
		logger.FieldTenantIDCamel, tenantID)

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
		es.logger.ErrorContext(ctx, LogInvalidEUILengthGetByEUI,
			logger.FieldLength, len(eui),
			logger.FieldTenantIDCamel, tenantID)
		return nil, scaci.ErrMissingEpEui
	}

	// Step 2: Create DB context with timeout
	dbCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Step 3: Call repository
	endpoint, err := es.endpointRepo.GetByEUI(dbCtx, tenantID, eui)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			es.logger.DebugContext(ctx, LogEndpointNotFoundGetByEUI,
				logger.FieldTenantIDCamel, tenantID)
			return nil, scaci.ErrEndpointNotFound
		}
		es.logger.ErrorContext(ctx, scaci.LogSCACIDatabaseErrorRegister,
			logger.FieldTenantIDCamel, tenantID,
			logger.FieldError, err)
		return nil, scaci.ErrDatabaseError
	}

	return endpoint, ""
}

// Attach implements EndpointService.Attach.
func (es *endpointService) Attach(ctx context.Context, ep *models.EndPoint) string {
	dbCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	_, err := es.decider.Decide(dbCtx, bssci.AttachmentDecision{
		TenantID:   ep.TenantID,
		EndpointID: ep.ID,
		EpEUI:      ep.EUI.ToUint64(),
		Status:     endpoint.EndpointStatusAttached,
	})
	if err != nil {
		es.logger.ErrorContext(ctx, scaci.LogSCACIUpdateEndpointFailed,
			logger.FieldTenantIDCamel, ep.TenantID,
			logger.FieldEndpointIDCamel, ep.ID,
			logger.FieldError, err)
		return scaci.ErrFailedUpdateEndpoint
	}
	return ""
}

// PropagateDetachToAll implements EndpointService.PropagateDetachToAll: it
// sends detPrp to every connected base station only for an endpoint the
// tenant owns, so an application center never detaches another tenant's
// endpoint.
func (es *endpointService) PropagateDetachToAll(ctx context.Context, tenantID int64, epEui uint64) []error {
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], epEui)

	dbCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	if _, err := es.endpointRepo.GetByEUI(dbCtx, tenantID, eui[:]); err != nil {
		es.logger.WarnContext(ctx, LogDetachPropagationRefused,
			logger.FieldEpEui, mioty.FormatEUI64(epEui),
			logger.FieldTenantIDCamel, tenantID,
			logger.FieldError, err)
		return []error{fmt.Errorf("%w: %w", errDetachPropagationEndpointLookup, err)}
	}

	return es.detachPropagator.SendDetachPropagateToAll(epEui)
}
