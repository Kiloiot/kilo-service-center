package bssciservices

import (
	"context"
	"fmt"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// propagationService implements propagation.Service interface
type propagationService struct {
	endpointRepo interfaces.EndpointRepository
	sender       pkgbssci.AttachPropagateSender
	logger       logger.Logger
}

// NewPropagationService creates a new propagation service instance
func NewPropagationService(
	endpointRepo interfaces.EndpointRepository,
	sender pkgbssci.AttachPropagateSender,
	logger logger.Logger,
) propagation.Service {
	return &propagationService{
		endpointRepo: endpointRepo,
		sender:       sender,
		logger:       logger,
	}
}

// TriggerEndpointPropagate fans out attach propagate to all eligible base stations
// Context must already contain tenant/org values (enriched by caller)
func (s *propagationService) TriggerEndpointPropagate(
	ctx context.Context,
	endpointID int64,
	activeSessions []propagation.BaseStationSession,
) error {
	// Extract tenantID from context (required for GetByID tenant isolation)
	tenantID, _ := pkgcontext.GetTenantID(ctx)

	// Fetch endpoint by ID
	endpoint, err := s.endpointRepo.GetByID(ctx, endpointID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to fetch endpoint %d: %w", endpointID, err)
	}

	// ATT-03: Filter sessions by tenant and propagate to each with telemetry
	var errs []error
	propagatedCount := 0
	skippedByTenant := 0

	for _, sess := range activeSessions {
		if !s.shouldPropagate(endpoint.TenantID, sess.TenantID) {
			s.logger.InfoContext(ctx, pkgbssci.LogBSSCISkippingPropagationDueToTenantMismatch,
				"endpoint_tenant", endpoint.TenantID,
				"session_tenant", sess.TenantID,
				"bs_eui", sess.BaseStationEUI)
			skippedByTenant++
			continue
		}

		// Send propagate message using context enriched by caller
		if err := s.sender.SendAttachPropagateBySessionID(ctx, sess.ID, endpoint); err != nil {
			s.logger.WarnContext(ctx, pkgbssci.LogBSSCIFailedToPropagateToSession,
				"endpoint_id", endpoint.ID,
				"session_id", sess.ID,
				"bs_eui", sess.BaseStationEUI,
				"error", err.Error())
			errs = append(errs, err)
		} else {
			propagatedCount++
		}
	}

	s.logger.InfoContext(ctx, pkgbssci.LogBSSCIEndpointPropagationCompleted,
		"endpoint_id", endpoint.ID,
		"propagated_count", propagatedCount,
		"total_sessions", len(activeSessions),
		"skipped_by_tenant", skippedByTenant,
		"errors", len(errs))

	return aggregateErrors(errs)
}

// ReconcileBaseStation replays all endpoints to newly connected base station
// Per BSSCI §3.8: Both bidirectional AND unidirectional endpoints need attPrp
// Context must already contain tenant/org values (enriched by caller)
func (s *propagationService) ReconcileBaseStation(
	ctx context.Context,
	session propagation.BaseStationSession,
	_ *models.BaseStation,
) error {
	// Use existing GetByTenant method
	endpoints, err := s.endpointRepo.GetByTenant(ctx, session.TenantID)
	if err != nil {
		return fmt.Errorf("failed to fetch endpoints for tenant %d: %w", session.TenantID, err)
	}

	// BSSCI §3.8: Propagate ALL endpoints to this BS
	// attPrp is idempotent - safe to send even if endpoint was propagated to another BS
	// This ensures multi-BS deployments work correctly (each BS needs attPrp)
	s.logger.InfoContext(ctx, pkgbssci.LogBSSCIStartingBaseStationReconciliation,
		"session_id", session.ID,
		"bs_eui", session.BaseStationEUI,
		"tenant_id", session.TenantID,
		"total_endpoints", len(endpoints))

	// BSSCI §3.8: Propagate each endpoint to this session (both bidi and unidirectional)
	var errs []error
	reconciledCount := 0

	for _, endpoint := range endpoints {
		// Use context as-is (already enriched by caller)
		if err := s.sender.SendAttachPropagateBySessionID(ctx, session.ID, endpoint); err != nil {
			s.logger.WarnContext(ctx, pkgbssci.LogBSSCIReconciliationPropagateFailed,
				"endpoint_id", endpoint.ID,
				"session_id", session.ID,
				"bs_eui", session.BaseStationEUI,
				"error", err.Error())
			errs = append(errs, err)
		} else {
			reconciledCount++
		}
	}

	s.logger.InfoContext(ctx, pkgbssci.LogBSSCIBaseStationReconciliationCompleted,
		"session_id", session.ID,
		"bs_eui", session.BaseStationEUI,
		"reconciled_count", reconciledCount,
		"total_endpoints", len(endpoints),
		"errors", len(errs))

	return aggregateErrors(errs)
}

// shouldPropagate determines if propagation should occur based on tenant ownership and roaming policies
// ATT-03: Roaming policy enforcement hook per BSSCI §5.8 multi-tenant requirements
func (s *propagationService) shouldPropagate(endpointTenant, sessionTenant int64) bool {
	// Same tenant - always allow
	if endpointTenant == sessionTenant {
		return true
	}

	// TODO: Cross-tenant propagation hook for roaming agreements.
	// Query roaming_agreements table to validate endpoint->session propagation.
	//         return s.roamingPolicy.AllowPropagate(endpointTenant, sessionTenant)
	return false
}

// aggregateErrors combines multiple errors into a single error.
// Returns nil if no errors occurred.
func aggregateErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s: %d failures, first: %w",
		pkgbssci.ResolveErrorMessage(pkgbssci.ErrPropagationBroadcastFailure),
		len(errs), errs[0])
}
