package bssciservices

import (
	"context"
	"fmt"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgendpoint "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// propagationService implements propagation.Service interface
type propagationService struct {
	endpointRepo EndpointDirectory
	sender       StationPropagator
	logger       logger.Logger
}

// NewPropagationService creates a new propagation service instance
func NewPropagationService(
	endpointRepo EndpointDirectory,
	sender StationPropagator,
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
	tenantID, err := pkgcontext.GetTenantID(ctx)
	if err != nil {
		return fmt.Errorf(errFmtPropagationTenant, endpointID, err)
	}

	// Fetch endpoint by ID
	endpoint, err := s.endpointRepo.GetByID(ctx, endpointID, tenantID)
	if err != nil {
		return fmt.Errorf(errFmtFetchEndpoint, endpointID, err)
	}

	// ATT-03: Filter sessions by tenant and propagate to each with telemetry
	var errs []error
	var propagatedCount int
	skippedByTenant := 0

	for _, sess := range activeSessions {
		if !s.shouldPropagate(endpoint.TenantID, sess.TenantID) {
			s.logger.InfoContext(ctx, pkgbssci.LogBSSCISkippingPropagationDueToTenantMismatch,
				logger.FieldEndpointTenant, endpoint.TenantID,
				logger.FieldSessionTenant, sess.TenantID,
				logger.FieldBsEuiSnake, sess.BaseStationEUI)
			skippedByTenant++
			continue
		}

		// Send propagate message using context enriched by caller
		if err := s.sender.SendAttachPropagateBySessionID(ctx, sess.ID, endpoint); err != nil {
			s.logger.WarnContext(ctx, pkgbssci.LogBSSCIFailedToPropagateToSession,
				logger.FieldEndpointID, endpoint.ID,
				logger.FieldSessionIDSnake, sess.ID,
				logger.FieldBsEuiSnake, sess.BaseStationEUI,
				logger.FieldError, err.Error())
			errs = append(errs, err)
		} else {
			propagatedCount++
		}
	}

	s.logger.InfoContext(ctx, pkgbssci.LogBSSCIEndpointPropagationCompleted,
		logger.FieldEndpointID, endpoint.ID,
		logger.FieldPropagatedCount, propagatedCount,
		logger.FieldTotalSessions, len(activeSessions),
		logger.FieldSkippedByTenant, skippedByTenant,
		logger.FieldErrors, len(errs))

	return aggregateErrors(errs)
}

// ReconcileBaseStation sends a connecting base station attPrp for the
// endpoints of its tenant the service center holds attached, bidirectional and
// unidirectional alike (BSSCI §3.8). A station that resumed its session kept
// the endpoints it held (BSSCI §1): it is sent attPrp only for the
// attachments it missed and detPrp for every endpoint detached while it was
// away (BSSCI §3.9).
// Context must already contain tenant/org values (enriched by caller)
func (s *propagationService) ReconcileBaseStation(
	ctx context.Context,
	session propagation.BaseStationSession,
	_ *models.BaseStation,
) error {
	attached, err := s.attachmentsToSend(ctx, session)
	if err != nil {
		return err
	}
	missed, err := s.detachedWhileAway(ctx, session)
	if err != nil {
		return err
	}
	total := len(attached) + len(missed)

	s.logger.InfoContext(ctx, pkgbssci.LogBSSCIStartingBaseStationReconciliation,
		logger.FieldSessionIDSnake, session.ID,
		logger.FieldBsEuiSnake, session.BaseStationEUI,
		logger.FieldTenantIDSnake, session.TenantID,
		logger.FieldTotalEndpoints, total)

	attachedSent, attachErrs := s.sendEach(ctx, session, attached, func(endpoint *models.EndPoint) error {
		return s.sender.SendAttachPropagateBySessionID(ctx, session.ID, endpoint)
	})
	detachedSent, detachErrs := s.sendEach(ctx, session, missed, func(endpoint *models.EndPoint) error {
		return s.sender.SendDetachPropagate(session.ID, endpoint.EUI.ToUint64())
	})
	errs := append(attachErrs, detachErrs...)

	s.logger.InfoContext(ctx, pkgbssci.LogBSSCIBaseStationReconciliationCompleted,
		logger.FieldSessionIDSnake, session.ID,
		logger.FieldBsEuiSnake, session.BaseStationEUI,
		logger.FieldReconciledCount, attachedSent+detachedSent,
		logger.FieldTotalEndpoints, total,
		logger.FieldErrors, len(errs))

	return aggregateErrors(errs)
}

// sendEach sends the station every endpoint through send and returns how many
// were sent and the failures, each of which it logs.
func (s *propagationService) sendEach(
	ctx context.Context,
	session propagation.BaseStationSession,
	endpoints []*models.EndPoint,
	send func(*models.EndPoint) error,
) (int, []error) {
	var errs []error
	for _, endpoint := range endpoints {
		if err := send(endpoint); err != nil {
			s.logger.WarnContext(ctx, pkgbssci.LogBSSCIReconciliationPropagateFailed,
				logger.FieldEndpointID, endpoint.ID,
				logger.FieldSessionIDSnake, session.ID,
				logger.FieldBsEuiSnake, session.BaseStationEUI,
				logger.FieldError, err.Error())
			errs = append(errs, err)
		}
	}
	return len(endpoints) - len(errs), errs
}

// attachmentsToSend returns the attached endpoints the station is sent
// attPrp for. A resumed station kept the attachments it held and the
// downlinks it queued for them, which an attPrp for a held endpoint makes it
// discard: it is sent only the endpoints attached after its connection was
// lost, or all of them when that time is unknown. A new session holds none.
func (s *propagationService) attachmentsToSend(ctx context.Context, station propagation.BaseStationSession) ([]*models.EndPoint, error) {
	if station.Resumed && station.DisconnectedAt != nil {
		missed, err := s.endpointRepo.GetByAttachmentChangedSince(ctx, station.TenantID, pkgbssci.EndpointStatusAttached, station.DisconnectedAt)
		if err != nil {
			return nil, fmt.Errorf(errFmtFetchAttachedEndpoints, station.TenantID, err)
		}
		return missed, nil
	}
	tenantEndpoints, err := s.endpointRepo.GetByTenant(ctx, station.TenantID)
	if err != nil {
		return nil, fmt.Errorf(errFmtFetchEndpointsForTenant, station.TenantID, err)
	}
	return attachedEndpoints(tenantEndpoints), nil
}

// detachedWhileAway returns the endpoints a resumed station may still hold
// although the service center detached them: those whose detachment was
// decided after its connection was lost, or every recorded detachment when
// that time is unknown. A new session holds none.
func (s *propagationService) detachedWhileAway(ctx context.Context, station propagation.BaseStationSession) ([]*models.EndPoint, error) {
	if !station.Resumed {
		return nil, nil
	}
	missed, err := s.endpointRepo.GetByAttachmentChangedSince(ctx, station.TenantID, pkgendpoint.EndpointStatusDetached, station.DisconnectedAt)
	if err != nil {
		return nil, fmt.Errorf(errFmtFetchDetachedEndpoints, station.TenantID, err)
	}
	return missed, nil
}

// attachedEndpoints keeps the endpoints the service center holds attached.
func attachedEndpoints(endpoints []*models.EndPoint) []*models.EndPoint {
	attached := make([]*models.EndPoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.EpStatus == pkgbssci.EndpointStatusAttached {
			attached = append(attached, endpoint)
		}
	}
	return attached
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
	return fmt.Errorf(errFmtAggregateFailures,
		pkgbssci.ResolveErrorMessage(pkgbssci.ErrPropagationBroadcastFailure),
		len(errs), errs[0])
}
