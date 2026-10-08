package interceptors

import (
	"context"
	"runtime/debug"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor converts a handler panic into the generic internal
// catalog error so a single defective request can never crash the process or
// leak its cause to the client. The panic value and stack are logged with the
// request context; the client sees only the catalog message.
type RecoveryInterceptor struct {
	log logger.Logger
}

// NewRecoveryInterceptor creates a panic-recovery interceptor.
func NewRecoveryInterceptor(log logger.Logger) *RecoveryInterceptor {
	if log == nil {
		log = logger.Get()
	}
	return &RecoveryInterceptor{log: log}
}

func (r *RecoveryInterceptor) recovered(ctx context.Context, method string, cause interface{}) error {
	r.log.ErrorContext(ctx, LogGrpcHandlerPanicRecovered,
		logger.FieldMethod, method,
		logger.FieldPanic, cause,
		logger.FieldStack, string(debug.Stack()))
	return status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInternalError),
		grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInternalError))
}

// UnaryInterceptor returns the unary panic-recovery interceptor.
func (r *RecoveryInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if cause := recover(); cause != nil {
				resp, err = nil, r.recovered(ctx, info.FullMethod, cause)
			}
		}()
		return handler(ctx, req)
	}
}

// StreamInterceptor returns the streaming panic-recovery interceptor.
func (r *RecoveryInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) (err error) {
		defer func() {
			if cause := recover(); cause != nil {
				err = r.recovered(ss.Context(), info.FullMethod, cause)
			}
		}()
		return handler(srv, ss)
	}
}
