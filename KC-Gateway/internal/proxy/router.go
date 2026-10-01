package proxy

import (
	"strings"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	identityServicePrefix         = rpccatalog.ServicePrefix(pb.IdentityService_ServiceDesc)
	identityInternalServicePrefix = rpccatalog.ServicePrefix(pb.IdentityInternalService_ServiceDesc)
)

// statusUnknownService is the status text returned for methods the gateway
// refuses to route, matching gRPC's own unknown-service wording so
// internal-only services stay indistinguishable from absent ones.
const statusUnknownService = "unknown service"

// compatIdentityMethods is every KiloCenterService compat method whose name is
// an IdentityService RPC; those calls route to KC-Identity.
var compatIdentityMethods = func() map[string]bool {
	compat := make(map[string]bool, len(pb.KiloCenterService_ServiceDesc.Methods))
	for _, method := range pb.KiloCenterService_ServiceDesc.Methods {
		compat[method.MethodName] = true
	}
	methods := make(map[string]bool)
	for _, method := range pb.IdentityService_ServiceDesc.Methods {
		if compat[method.MethodName] {
			methods[rpccatalog.FullMethod(pb.KiloCenterService_ServiceDesc, method.MethodName)] = true
		}
	}
	return methods
}()

// SelectUpstream routes an external RPC to the upstream serving it: the
// connection it is proxied on, or the circuit breaker guarding that upstream.
// IdentityInternalService methods are explicitly denied (internal-only).
func SelectUpstream[U any](fullMethod string, core, identity U) (U, error) {
	// Hard deny: internal-only service must never be externally reachable
	if strings.HasPrefix(fullMethod, identityInternalServicePrefix) {
		var none U
		return none, status.Error(codes.Unimplemented, statusUnknownService)
	}

	// Route IdentityService RPCs to KC-Identity
	if strings.HasPrefix(fullMethod, identityServicePrefix) {
		return identity, nil
	}

	// Route compat identity methods to KC-Identity
	if compatIdentityMethods[fullMethod] {
		return identity, nil
	}

	// Everything else goes to KC-Core
	return core, nil
}
