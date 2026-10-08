package builders

import (
	"context"
	"fmt"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// defaultOrgResolverBuilder creates the CE single-tenant community resolver.
func defaultOrgResolverBuilder(
	ctx context.Context,
	ocfg *OrgResolverConfig,
	orgRepo interfaces.OrgDirectoryRepository,
	log logger.Logger,
) (*OrgResolverResult, error) {
	log.Info(LogCommunityEditionInitializingSingleTenantOrgResolver,
		logger.FieldTenantIDSnake, ocfg.TenantID)

	defaultOrg, lookupErr := orgRepo.GetOrgByTenantID(ctx, ocfg.TenantID)
	if lookupErr != nil || defaultOrg == nil {
		return nil, fmt.Errorf(errFmtCERequiresDefaultOrg, ocfg.TenantID, lookupErr)
	}

	result := &OrgResolverResult{
		Resolver: org.NewCommunityResolver(ocfg.TenantID, defaultOrg.OrgID),
	}

	if ocfg.IdentityAddress != "" {
		identityConn, connErr := grpc.NewClient(ocfg.IdentityAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if connErr != nil {
			return nil, fmt.Errorf(errFmtFailedToConnectKCIdentity, ocfg.IdentityAddress, connErr)
		}
		result.Cleanups = append(result.Cleanups, func() {
			if err := identityConn.Close(); err != nil {
				log.Error(LogFailedCloseIdentityConnection, logger.Err(err))
			}
		})
		result.IdentityInternalClient = pb.NewIdentityInternalServiceClient(identityConn)
	}

	return result, nil
}
