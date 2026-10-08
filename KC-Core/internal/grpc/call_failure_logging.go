package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// logCallFailure logs a failed RPC as an error only when the service failed;
// a request the service refused is the caller's mistake and a warning.
func logCallFailure(ctx context.Context, log logger.Logger, message, method string, err error) {
	if refusedRequest(status.Code(err)) {
		log.WarnContext(ctx, message, logger.FieldMethod, method, logger.FieldError, err)
		return
	}
	log.ErrorContext(ctx, message, logger.FieldMethod, method, logger.FieldError, err)
}

func refusedRequest(code codes.Code) bool {
	switch code {
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition,
		codes.OutOfRange, codes.Aborted:
		return true
	default:
		return false
	}
}
