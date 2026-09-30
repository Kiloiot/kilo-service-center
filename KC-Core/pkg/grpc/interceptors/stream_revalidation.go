package interceptors

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// regrantFunc resolves the caller's current roles and refuses them unless they still satisfy the method.
type regrantFunc func(ctx context.Context, method string) (authz.Roles, error)

// revalidatingServerStream ends a stream on any role change, since its handler built its filters from the opening roles.
type revalidatingServerStream struct {
	contextServerStream
	method  string
	opened  authz.Roles
	regrant regrantFunc
	logger  logger.Logger
}

func newRevalidatingServerStream(base contextServerStream, method string, opened authz.Roles, regrant regrantFunc, log logger.Logger) *revalidatingServerStream {
	return &revalidatingServerStream{contextServerStream: base, method: method, opened: opened, regrant: regrant, logger: log}
}

func (s *revalidatingServerStream) SendMsg(m interface{}) error {
	roles, err := s.regrant(s.ctx, s.method)
	if err != nil {
		return err
	}
	if roles != s.opened {
		s.logger.InfoContext(s.ctx, grpcerrors.LogRBACStreamRolesChanged, logger.FieldMethod, s.method)
		return insufficientRoleError()
	}
	return s.ServerStream.SendMsg(m)
}
