package grpc

import (
	"context"

	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// requireAdmin admits a caller the authorization interceptor resolved as an
// administrator; it guards operations whose requirement depends on the target,
// such as changes to the system catalog.
func requireAdmin(ctx context.Context) error {
	if authz.FromContext(ctx).Admin {
		return nil
	}
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
}
