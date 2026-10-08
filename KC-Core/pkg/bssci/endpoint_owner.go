package bssci

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// storedEUI is the stored form of an endpoint's EUI-64.
func storedEUI(epEUI uint64) models.EUI {
	return models.EUI(mioty.EUI64(epEUI).ToBytes())
}

// endpointOwnerContext scopes work to an endpoint's owner. It is rooted in the
// server context so the serving session's tenant and organization never leak
// into the owner's records.
func (s *Server) endpointOwnerContext(ownerTenantID int64) (context.Context, uuid.UUID) {
	ctx := pkgcontext.WithTenantID(s.safeCtx(), ownerTenantID)
	org, err := s.orgResolver.GetDefaultOrgForTenant(ctx, ownerTenantID)
	if err != nil {
		s.logger.WarnContext(ctx, LogBSSCIFailedToResolveOrganizationForEndpointOwner,
			logger.FieldTenantID, ownerTenantID,
			logger.FieldError, err)
		return ctx, uuid.Nil
	}
	if org != uuid.Nil {
		ctx = pkgcontext.WithOrganizationID(ctx, org)
	}
	return ctx, org
}
