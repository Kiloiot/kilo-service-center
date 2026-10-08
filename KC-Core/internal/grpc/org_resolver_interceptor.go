package grpc

import "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"

// OrgResolverInterceptorConfig is a type alias for the extracted interceptors.OrgResolverInterceptorConfig.
type OrgResolverInterceptorConfig = interceptors.OrgResolverInterceptorConfig

// NewOrgResolverInterceptor delegates to the extracted constructor.
func NewOrgResolverInterceptor(cfg OrgResolverInterceptorConfig) (*interceptors.OrgResolverInterceptor, error) {
	return interceptors.NewOrgResolverInterceptor(cfg)
}
