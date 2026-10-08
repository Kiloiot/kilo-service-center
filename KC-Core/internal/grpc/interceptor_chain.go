package grpc

import (
	"errors"
	"fmt"

	"google.golang.org/grpc"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// interceptorChain holds the unary and stream interceptors in call order.
type interceptorChain struct {
	unary  []grpc.UnaryServerInterceptor
	stream []grpc.StreamServerInterceptor
}

func (c *interceptorChain) add(unary grpc.UnaryServerInterceptor, stream grpc.StreamServerInterceptor) {
	c.unary = append(c.unary, unary)
	c.stream = append(c.stream, stream)
}

// buildInterceptorChain assembles the interceptors in their fixed order:
// panic recovery, identity, authorization, then logging.
func buildInterceptorChain(cfg Config, log logger.Logger) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor, error) {
	chain := &interceptorChain{}

	// Panic recovery runs first so every later interceptor and handler is
	// covered: a panicking request returns the generic internal catalog error
	// instead of crashing the process.
	recoveryInterceptor := interceptors.NewRecoveryInterceptor(log)
	chain.add(recoveryInterceptor.UnaryInterceptor(), recoveryInterceptor.StreamInterceptor())

	addIdentity := addStandaloneIdentity
	if cfg.InternalTrustEnabled {
		addIdentity = addInternalTrustIdentity
	}
	if err := addIdentity(chain, cfg, log); err != nil {
		return nil, nil, err
	}
	if cfg.RoleSource == nil {
		return nil, nil, errors.New(errMsgRoleSourceRequired)
	}

	authzInterceptor := interceptors.NewAuthorizationInterceptor(cfg.RoleSource, NewMethodPolicy(), log).
		WithEventWriter(cfg.EventWriter).
		WithPlatformTenantID(cfg.PlatformTenantID)
	chain.add(authzInterceptor.UnaryInterceptor(), authzInterceptor.StreamInterceptor())

	chain.add(unaryInterceptor(log), streamInterceptor(log))
	return chain.unary, chain.stream, nil
}

// addInternalTrustIdentity trusts the identity headers KC-Gateway sets, in
// place of auth and org resolution. Community mode (no org enforcement)
// makes the org header optional for all methods.
func addInternalTrustIdentity(chain *interceptorChain, cfg Config, log logger.Logger) error {
	communityMode := cfg.FailClosedOrgResolver == nil
	if communityMode && cfg.DefaultOrgResolver == nil {
		return errors.New(errMsgDefaultOrgResolverRequired)
	}
	trustInterceptor := interceptors.NewInternalTrustInterceptor(log, communityMode).
		WithPeerSecret(cfg.PeerSecret).
		WithEventWriter(cfg.EventWriter).
		WithPlatformTenantID(cfg.PlatformTenantID).
		WithDefaultOrgResolver(cfg.DefaultOrgResolver)
	chain.add(trustInterceptor.UnaryInterceptor(), trustInterceptor.StreamInterceptor())
	if communityMode {
		log.Info(LogSecurityInternalOnlyCommunityMode)
		return nil
	}
	log.Info(LogSecurityInternalOnlyMode)
	return nil
}

// addStandaloneIdentity authenticates every request itself and then scopes it
// to an organization.
func addStandaloneIdentity(chain *interceptorChain, cfg Config, log logger.Logger) error {
	authInterceptor, err := NewAuthInterceptor(cfg.Auth)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToCreateAuthInterceptor, err)
	}
	if cfg.OrgResolver != nil {
		authInterceptor.WithOrganizationResolver(cfg.OrgResolver)
		log.Info(LogAuthInterceptorOrgResolutionConfigured)
	}
	if cfg.TenantResolver != nil {
		authInterceptor.WithTenantResolver(cfg.TenantResolver)
		log.Info(LogAuthInterceptorTenantResolutionForOrgUUID)
	}
	if cfg.APIKeyAuth != nil {
		authInterceptor.WithAPIKeyAuthenticator(cfg.APIKeyAuth)
		log.Info(LogAuthInterceptorAPIKeyBearerConfigured)
	}
	chain.add(authInterceptor.UnaryInterceptor(), authInterceptor.StreamInterceptor())
	return addOrganizationScope(chain, cfg, log)
}

// addOrganizationScope adds the fail-closed org resolver when one is
// configured; otherwise a community deployment runs every request under the
// default tenant and that tenant's default organization.
func addOrganizationScope(chain *interceptorChain, cfg Config, log logger.Logger) error {
	if cfg.FailClosedOrgResolver != nil {
		orgInterceptor, err := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
			Resolver:         cfg.FailClosedOrgResolver,
			Logger:           log,
			SkipMethods:      grpcerrors.OrgExemptMethodList(),
			EventWriter:      cfg.EventWriter,
			PlatformTenantID: cfg.PlatformTenantID,
			AdminChecker:     cfg.AdminChecker,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", errMsgFailedToCreateOrgResolverInterceptor, err)
		}
		chain.add(orgInterceptor.UnaryInterceptor(), orgInterceptor.StreamInterceptor())
		log.Info(LogFailClosedOrgResolverInterceptorConfigured)
		return nil
	}
	if cfg.DefaultTenantID <= 0 {
		return nil
	}
	if cfg.DefaultOrgResolver == nil {
		return errors.New(errMsgDefaultOrgResolverRequired)
	}
	community := newCommunityContext(cfg.DefaultTenantID, cfg.DefaultOrgResolver)
	chain.add(community.unaryInterceptor(), community.streamInterceptor())
	log.Info(LogCommunityModeInjectingDefaultTenant, logger.FieldTenantID, cfg.DefaultTenantID)
	return nil
}
