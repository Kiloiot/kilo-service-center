package scaci

import (
	"context"
	"crypto/x509"
	"net"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// recordSessionEvent files a lifecycle step of the session that happened at
// occurredAt under its tenant; reason says why a closed session ended.
func (s *Server) recordSessionEvent(session *Session, eventType, reason string, conn net.Conn, occurredAt time.Time) {
	s.fileSessionEvent(withSessionValues(s.persistContext(), session), &models.SCACISessionEvent{
		TenantID:   session.TenantID,
		Category:   models.EventCategorySCACI,
		EventType:  eventType,
		SessionID:  session.ID,
		AcEui:      mioty.FormatEUI64(session.AcEui),
		RemoteAddr: conn.RemoteAddr().String(),
		Reason:     reason,
		OccurredAt: occurredAt,
	})
}

// liveEventType is the event of a session whose connect operation completed.
func liveEventType(session *Session) string {
	if session.Resumed {
		return models.EventTypeSCACISessionResumed
	}
	return models.EventTypeSCACISessionOpened
}

// recordSuperseded files the close of every live session a new session of
// its application center took over (SCACI §1). The adopt that returned
// replaced saw each of them live, so each went live, and read the moment it
// opened, before the moment read here.
func (s *Server) recordSuperseded(replaced []sessionTarget) {
	supersededAt := s.clock.Now()
	for _, old := range replaced {
		if old.session.connected() {
			s.recordSessionEvent(old.session, models.EventTypeSCACISessionClosed, models.SCACISessionClosedSuperseded, old.conn, supersededAt)
		}
	}
}

// recordConnectRefused files a connect operation refused with errToken (SCACI
// §3.3) under the tenant of the session it would have opened or, before that
// session existed, of the client certificate. A certificate of no tenant makes
// it a server-level security event of the platform tenant, which only
// administrators read. acEui is zero when the connect could not be decoded.
func (s *Server) recordConnectRefused(conn net.Conn, session *Session, cert *x509.Certificate, acEui uint64, errToken string) {
	if s.sessionEvents == nil {
		return
	}
	event := &models.SCACISessionEvent{
		Category:     models.EventCategorySCACI,
		EventType:    models.EventTypeSCACIConnectRefused,
		RemoteAddr:   conn.RemoteAddr().String(),
		ErrorToken:   errToken,
		ErrorMessage: GetErrorDefinition(errToken).Message,
	}
	ctx := withSessionValues(s.persistContext(), session)
	if session != nil {
		event.TenantID, event.SessionID, acEui = session.TenantID, session.ID, session.AcEui
	} else if tenantID, ok := s.handshakeSvc.CertificateTenant(ctx, cert); ok {
		event.TenantID = tenantID
	} else {
		event.TenantID, event.Category = s.config.PlatformTenantID, models.EventCategorySecurity
	}
	if acEui != 0 {
		event.AcEui = mioty.FormatEUI64(acEui)
	}
	s.fileSessionEvent(ctx, event)
}

// fileSessionEvent stores the event; the event log is not on the protocol's
// critical path, so a failure is only logged.
func (s *Server) fileSessionEvent(ctx context.Context, event *models.SCACISessionEvent) {
	if s.sessionEvents == nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()
	if err := s.sessionEvents.RecordSessionEvent(writeCtx, event); err != nil {
		s.logger.WarnContext(ctx, LogSCACIRecordSessionEventFailed,
			logger.FieldEventType, event.EventType, logger.FieldError, err)
	}
}
