package bssci

import (
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SendAttachPropagateToAll sends attach propagate to every connected base
// station of the endpoint's tenant that completed its handshake (BSSCI §5.8,
// BSSCI-3.3-03). Cross-tenant roaming is not propagated.
func (s *Server) SendAttachPropagateToAll(endpointTenantID int64, endpointEUI uint64, nwkSnKey []byte,
	shortAddr uint16, bidirectional bool, lastPacketCnt uint32, dualChannel bool,
	repetition uint8, wideCarrOff bool, longBlkDist bool,
) []error {
	if endpointTenantID <= 0 {
		return s.propagationWithoutTenant(endpointEUI)
	}
	sessionIDs := s.tenantSessionIDs(endpointTenantID, noExcludedSession)
	if len(sessionIDs) == 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCINoConnectedBaseStationsForAttachPropagate,
			logger.FieldEndpointEUI, endpointEUI)
		return s.propagationWithoutStations(stationlessPropagation{
			endpointEUI:    endpointEUI,
			eventType:      EventTypeAttachOperationFailed,
			titleFormat:    titleAttachPropagateFailedFormat,
			description:    eventDescNoBSForAttachPropagate,
			details:        map[string]interface{}{models.EventDetailKeyShortAddr: shortAddr},
			eventFailedLog: LogBSSCIFailedToCreateAttachFailedEvent,
		})
	}

	var errs []error
	for _, sessionID := range sessionIDs {
		if err := s.SendAttachPropagate(sessionID, endpointEUI, nwkSnKey, shortAddr,
			bidirectional, lastPacketCnt, dualChannel, repetition, wideCarrOff, longBlkDist); err != nil {
			errs = append(errs, fmt.Errorf(errFmtSessionWrap, sessionID, err))
		}
	}
	return errs
}

// SendDetachPropagateToAll sends detach propagate to every connected base
// station of the endpoint's tenant that completed its handshake (BSSCI §5.9,
// BSSCI-3.3-03). Cross-tenant roaming is not propagated.
func (s *Server) SendDetachPropagateToAll(endpointTenantID int64, endpointEUI uint64) []error {
	if endpointTenantID <= 0 {
		return s.propagationWithoutTenant(endpointEUI)
	}
	sessionIDs := s.tenantSessionIDs(endpointTenantID, noExcludedSession)
	if len(sessionIDs) == 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCINoConnectedBaseStationsForDetachPropagate,
			logger.FieldEndpointEUI, endpointEUI)
		return s.propagationWithoutStations(stationlessPropagation{
			endpointEUI:    endpointEUI,
			eventType:      EventTypeDetachOperationFailed,
			titleFormat:    titleDetachPropagateFailedFormat,
			description:    eventDescNoBSForDetachPropagate,
			eventFailedLog: LogBSSCIFailedToCreateDetachFailedEvent,
		})
	}
	return s.sendDetachPropagateTo(sessionIDs, endpointEUI)
}

// EndpointOwnerTenant names the tenant that owns the endpoint.
func (s *Server) EndpointOwnerTenant(endpointEUI uint64) (int64, error) {
	owner, err := s.endpointOwners.ResolveOwner(s.safeCtx(), storedEUI(endpointEUI))
	if err != nil {
		return 0, fmt.Errorf(errFmtResolveEndpointOwnerTenant, mioty.FormatEUI64(endpointEUI), err)
	}
	return owner.TenantID, nil
}

// noExcludedSession names no session to leave out of a fan-out.
const noExcludedSession = ""

// tenantSessionIDs lists the sessions of the tenant that may receive a
// service center operation (BSSCI-3.3-03), leaving out the excluded one.
func (s *Server) tenantSessionIDs(tenantID int64, excluded string) []string {
	if tenantID <= 0 {
		return nil
	}
	live := s.sessions.snapshot()
	sessionIDs := make([]string, 0, len(live))
	for _, session := range live {
		if session.HandshakeComplete && session.ID != excluded && resolvedTenant(session, s.tenantID) == tenantID {
			sessionIDs = append(sessionIDs, session.ID)
		}
	}
	return sessionIDs
}

// propagationWithoutTenant refuses a propagation whose endpoint tenant is not
// known: its network key must reach no base station.
func (s *Server) propagationWithoutTenant(endpointEUI uint64) []error {
	s.logger.WarnContext(s.safeCtx(), LogBSSCIPropagationEndpointTenantUnknown,
		logger.FieldEpEui, mioty.FormatEUI64(endpointEUI))
	return []error{ErrEndpointTenantUnknown}
}

// sendDetachPropagateTo sends detach propagate for the endpoint to each session.
func (s *Server) sendDetachPropagateTo(sessionIDs []string, endpointEUI uint64) []error {
	var errs []error
	for _, sessionID := range sessionIDs {
		if err := s.SendDetachPropagate(sessionID, endpointEUI); err != nil {
			errs = append(errs, fmt.Errorf(errFmtSessionWrap, sessionID, err))
		}
	}
	return errs
}

// stationlessPropagation is a propagation no connected base station can
// receive, as its failure event describes it.
type stationlessPropagation struct {
	endpointEUI    uint64
	eventType      string
	titleFormat    string
	description    string
	details        map[string]interface{}
	eventFailedLog string
}

// propagationWithoutStations fails a propagation no connected base station
// can receive and records its failure event for the endpoint's owner.
func (s *Server) propagationWithoutStations(p stationlessPropagation) []error {
	s.recordStationlessPropagation(p)
	return []error{fmt.Errorf("%s", ResolveErrorMessage(errNoConnectedBaseStations))}
}

// recordStationlessPropagation records the failure event, which names the
// endpoint, for the tenant that owns it (multi-tenant roaming); an endpoint
// no tenant owns is only logged, so no tenant is shown another's EUI.
func (s *Server) recordStationlessPropagation(p stationlessPropagation) {
	owner, err := s.endpointOwners.ResolveOwner(s.safeCtx(), storedEUI(p.endpointEUI))
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIStationlessPropagationOwnerUnresolved,
			logger.FieldEpEui, mioty.FormatEUI64(p.endpointEUI),
			logger.FieldError, err)
		return
	}
	ownerCtx, _ := s.endpointOwnerContext(owner.TenantID)

	details := map[string]interface{}{
		models.EventDetailKeyEpEui:  mioty.FormatEUI64(p.endpointEUI),
		models.EventDetailKeyReason: eventReasonNoConnectedBS,
	}
	for key, value := range p.details {
		details[key] = value
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		s.logger.WarnContext(ownerCtx, p.eventFailedLog, logger.FieldError, err)
		return
	}

	if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", owner.TenantID),
		EventType:   p.eventType,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityError,
		Title:       fmt.Sprintf(p.titleFormat, mioty.FormatEUI64(p.endpointEUI)),
		Description: p.description,
		Details:     detailsJSON,
		Status:      EventStatusNew,
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  mioty.FormatEUI64(p.endpointEUI),
		CreatedAt:   s.clock.Now(),
	}); err != nil {
		s.logger.WarnContext(ownerCtx, p.eventFailedLog, logger.FieldError, err)
	}
}
