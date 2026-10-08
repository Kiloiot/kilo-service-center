package bssci

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// servedEndpoint is a tenant's endpoint whose serving station is decided once
// per connect.
type servedEndpoint struct {
	tenantID int64
	epEUI    uint64
}

// settleConnectedStation brings a station that completed its connect up to
// date: the attachments it lacks first, then the pending downlinks it served
// when it connected, so each dlDataQue reaches it behind the attPrp for its
// endpoint.
func (s *Server) settleConnectedStation(ctx context.Context, session *Session, served []storage.PendingDownlink) {
	s.reconcileConnectedStation(ctx, session)
	for _, downlink := range served {
		s.dispatchAheadOfWindow(ctx, session, downlink)
	}
}

// reconcileConnectedStation sends the station the endpoints the service
// center holds attached, and a resumed one those detached while it was away
// (BSSCI §5.8.3).
func (s *Server) reconcileConnectedStation(ctx context.Context, session *Session) {
	station := propagation.BaseStationSession{
		ID:                session.ID,
		BaseStationEUI:    session.BaseStationEUI,
		TenantID:          resolvedTenant(session, s.tenantID),
		HandshakeComplete: true,
		Resumed:           session.IsResumed,
		DisconnectedAt:    session.DisconnectedAt,
	}
	if err := s.propagationSvc.ReconcileBaseStation(ctx, station, nil); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIBaseStationReconciliationFailed,
			logger.FieldBsEuiSnake, session.BaseStationEUI,
			logger.FieldError, err)
	}
}

// servedDownlinks returns the pending downlinks of the endpoints a
// bidirectional station serves as it completes its connect, which it is sent
// ahead of their next downlink window (BSSCI §5.12: a queue operation may
// schedule downlink data a priori). They are chosen before the station can
// report an uplink: a downlink window an uplink opens later belongs to the
// in-window dispatch, which fills it once.
func (s *Server) servedDownlinks(ctx context.Context, session *Session) []storage.PendingDownlink {
	if !session.Bidirectional {
		return nil
	}
	pending, err := s.pendingDownlinks.ListPendingDownlinks(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToListPendingDownlinks,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
		return nil
	}
	decided := make(map[servedEndpoint]bool)
	var served []storage.PendingDownlink
	for _, downlink := range pending {
		endpoint := servedEndpoint{tenantID: downlink.TenantID, epEUI: downlink.EpEUI}
		serves, known := decided[endpoint]
		if !known {
			serves = s.stationServes(ctx, session, endpoint)
			decided[endpoint] = serves
		}
		if serves {
			served = append(served, downlink)
		}
	}
	return served
}

// stationServes reports whether the serving-station policy picks the station
// for the tenant's endpoint.
func (s *Server) stationServes(ctx context.Context, session *Session, endpoint servedEndpoint) bool {
	bsEUI, known, err := s.servingStations.ServingStation(ctx, endpoint.tenantID, endpoint.epEUI)
	if err != nil {
		s.logger.WarnContext(ctx, LogBSSCIServingStationLookupFailedOnConnect,
			logger.FieldEpEui, endpoint.epEUI,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
		return false
	}
	return known && bsEUI == session.BaseStationEUI
}

// dispatchAheadOfWindow reserves and sends one pending downlink under the
// tenant and organization it was enqueued for, through the dispatcher's exact
// reservation, so a row another dispatch took meanwhile is skipped.
func (s *Server) dispatchAheadOfWindow(ctx context.Context, session *Session, downlink storage.PendingDownlink) {
	ownerCtx := pkgcontext.WithOrganizationID(pkgcontext.WithTenantID(ctx, downlink.TenantID), downlink.OrganizationID)
	if _, err := s.downlinkDispatcher.DispatchQueue(ownerCtx, downlink.TenantID, downlink.OrganizationID,
		session, downlink.QueID, downlink.EpEUI); err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToDispatchOnConnect,
			logger.FieldQueID, downlink.QueID,
			logger.FieldEpEui, downlink.EpEUI,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
	}
}
