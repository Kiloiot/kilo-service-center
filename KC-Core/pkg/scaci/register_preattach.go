package scaci

import (
	"context"
	"encoding/binary"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// preAttachRegisteredEndpoint attaches the registering tenant's endpoint when
// its registration asked for pre-attachment and propagates it to the
// connected base stations (SCACI §3.6, BSSCI §3.8).
func (s *Server) preAttachRegisteredEndpoint(session *Session, epEui uint64) {
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// An application center registers only its own tenant's endpoints (SCACI §3.6).
	tenantID := session.TenantID
	endpoint, errToken := s.endpointSvc.GetByEUI(s.sessionContext(session), tenantID, euiBytes)
	if errToken != "" || endpoint == nil || !endpoint.PreAttach {
		return
	}

	tenantCtx := s.tenantLifecycleContext(tenantID)
	// The endpoint is attached from here on, so every station that connects later is sent it (BSSCI §3.8).
	if attachErr := s.endpointSvc.Attach(tenantCtx, endpoint); attachErr != "" {
		return
	}

	s.logger.InfoContext(tenantCtx, LogSCACITriggeringAttachPropagation,
		logger.FieldEpEui, mioty.FormatEUI64(epEui),
		logger.FieldEndpointIDCamel, endpoint.ID,
		logger.FieldTenantIDCamel, tenantID)
	s.propagatePreAttachment(tenantCtx, endpoint.ID)
}

// tenantLifecycleContext carries the tenant and its default organization on
// the server lifecycle context, so work started with it stops with the server.
func (s *Server) tenantLifecycleContext(tenantID int64) context.Context {
	ctx := pkgcontext.WithTenantID(s.safeCtx(), tenantID)
	if s.orgResolver == nil {
		return ctx
	}
	orgID, err := s.orgResolver.GetDefaultOrgForTenant(ctx, tenantID)
	if err != nil || orgID == uuid.Nil {
		return ctx
	}
	return pkgcontext.WithOrganizationID(ctx, orgID)
}

// propagatePreAttachment sends the attached endpoint to the connected base
// stations in the background.
func (s *Server) propagatePreAttachment(tenantCtx context.Context, endpointID int64) {
	if s.propagationSvc == nil || s.sessionSnapshotProvider == nil {
		return
	}
	s.runTracked(func() {
		activeSessions := s.sessionSnapshotProvider.ConnectedSessionsSnapshot()
		if err := s.propagationSvc.TriggerEndpointPropagate(tenantCtx, endpointID, activeSessions); err != nil {
			s.logger.ErrorContext(tenantCtx, LogSCACIAttachPropagationErrors,
				logger.FieldEndpointID, endpointID,
				logger.FieldError, err)
		}
	})
}
