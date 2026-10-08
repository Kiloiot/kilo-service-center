package blueprints

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// closeResponseBody closes a response body from a deferred call; the
// response has been consumed by then, so a failure is logged.
func closeResponseBody(ctx context.Context, log logger.Logger, resp *http.Response) {
	if err := resp.Body.Close(); err != nil {
		log.WarnContext(ctx, LogFailedCloseResponseBody, logger.FieldError, err)
	}
}

// errorBody reads an error response's body for diagnostics; an unreadable
// body is described by its read error instead.
func errorBody(resp *http.Response) []byte {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte(fmt.Sprintf(errFmtUnreadableResponseBody, err))
	}
	return body
}
