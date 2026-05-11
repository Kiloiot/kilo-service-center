package builders

import (
	"context"
	"fmt"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	orgresolver "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/orgresolver"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// defaultOrgResolverBuilder creates the CE single-tenant community resolver.
func defaultOrgResolverBuilder(
	ctx context.Context,
	ocfg *OrgResolverConfig,
	orgRepo interfaces.OrganizationRepository,
	log logger.Logger,
) (*OrgResolverResult, error) {
	log.Info("Community Edition: initializing single-tenant org resolver",
		"tenant_id", ocfg.TenantID)

	defaultOrg, lookupErr := orgRepo.GetOrgByTenantID(ctx, ocfg.TenantID)
	if lookupErr != nil || defaultOrg == nil {
		return nil, fmt.Errorf("CE requires default org for tenant %d: %w", ocfg.TenantID, lookupErr)
	}

	result := &OrgResolverResult{
		Resolver: orgresolver.NewCommunityResolver(ocfg.TenantID, defaultOrg.OrgID),
	}

	if ocfg.IdentityAddress != "" {
		identityConn, connErr := grpc.NewClient(ocfg.IdentityAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if connErr != nil {
			return nil, fmt.Errorf("failed to connect to KC-Identity at %s: %w", ocfg.IdentityAddress, connErr)
		}
		result.Cleanups = append(result.Cleanups, func() { _ = identityConn.Close() })
		result.IdentityInternalClient = pb.NewIdentityInternalServiceClient(identityConn)
	}

	return result, nil
}
