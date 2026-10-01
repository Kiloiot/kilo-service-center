package grpc

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// headerSender is the part of a server stream that opens its response.
type headerSender interface {
	SendHeader(metadata.MD) error
}

// openStream sends the response header once the subscription exists: grpc-go
// defers it to the first message, and a gRPC-web client treats a stream as
// live only after the header, so a quiet stream would look disconnected.
func openStream(ctx context.Context, log logger.Logger, stream headerSender) error {
	if err := stream.SendHeader(metadata.MD{}); err != nil {
		log.ErrorContext(ctx, LogStreamOpenFailed, logger.FieldError, err)
		return err
	}
	return nil
}
