package auth

import (
	"context"
	"io"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Response-body release log messages.
const (
	logResponseBodyDrainFailed = "HTTP response body drain failed"
	logResponseBodyCloseFailed = "HTTP response body close failed"
)

// closeResponseBody drains and closes an HTTP response body so its connection
// can be reused. The response has already been handled by then, so a failure
// is logged rather than returned.
func closeResponseBody(ctx context.Context, log logger.Logger, body io.ReadCloser) {
	if _, err := io.Copy(io.Discard, body); err != nil {
		log.WarnContext(ctx, logResponseBodyDrainFailed, logger.FieldError, err)
	}
	if err := body.Close(); err != nil {
		log.WarnContext(ctx, logResponseBodyCloseFailed, logger.FieldError, err)
	}
}
