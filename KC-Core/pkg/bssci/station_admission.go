package bssci

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// admitStation finds the registered base station a connect names and admits
// it only when the client certificate belongs to that station, in every
// edition, and its tenant settles. A refusal reads like an unregistered
// station, so it never reveals that the EUI is registered.
func (s *Server) admitStation(ctx context.Context, session *Session) (*basestation.BaseStation, bool) {
	euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()
	baseStation, err := s.connectionRegistry.GetBaseStationGlobal(ctx, euiBytes)
	if err != nil || baseStation == nil {
		s.logger.ErrorContext(ctx, LogBSSCIBaseStationNotFoundInDatabase,
			logger.FieldEui, session.BaseStationEUI,
			logger.FieldEuiHex, fmt.Sprintf("%X", euiBytes),
			logger.FieldError, err)
		return nil, false
	}
	claim := StationCertificateClaim{
		BaseStationEUI: session.BaseStationEUI,
		Certificate:    session.ClientCert,
		SubjectEUI:     session.certSubjectEUI,
	}
	if err := s.stationCertificates.BindStationCertificate(ctx, claim); err != nil {
		s.logger.WarnContext(ctx, LogBSSCIStationCertificateRefused,
			logger.FieldEui, session.BaseStationEUI,
			logger.FieldError, err)
		return nil, false
	}
	return baseStation, s.settleStationTenant(ctx, session, baseStation)
}

// settleStationTenant applies the organization policy: under organization
// enforcement the tenant the certificate resolved to must be the station's
// registered tenant; without it the session adopts the registered tenant and
// its default organization.
func (s *Server) settleStationTenant(ctx context.Context, session *Session, baseStation *basestation.BaseStation) bool {
	if s.config.OrgEnforcementEnabled {
		if session.ResolvedTenantID == baseStation.TenantID {
			return true
		}
		s.logger.WarnContext(ctx, LogBSSCIBaseStationNotFoundInDatabase,
			logger.FieldEui, session.BaseStationEUI,
			logger.FieldCertTenant, session.ResolvedTenantID)
		return false
	}
	if baseStation.TenantID > 0 {
		session.ResolvedTenantID = baseStation.TenantID
		if orgID, err := s.orgResolver.GetDefaultOrgForTenant(ctx, baseStation.TenantID); err == nil {
			session.OrganizationID = orgID
		}
	}
	return true
}
