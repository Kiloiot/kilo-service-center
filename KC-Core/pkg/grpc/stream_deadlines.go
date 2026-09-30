package grpc

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// deadlineWarner reports a stream whose deadlines could not be cleared.
type deadlineWarner interface {
	WarnContext(ctx context.Context, msg string, fields ...interface{})
}

// ExemptStreamsFromHTTPDeadlines clears the HTTP server's read and write
// deadlines for the streaming RPCs isStream names, so the server timeouts
// bound unary calls and never cut a stream a client keeps open.
func ExemptStreamsFromHTTPDeadlines(next http.Handler, isStream func(fullMethod string) bool, log deadlineWarner) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isStream(r.URL.Path) {
			clearDeadlines(w, r, log)
		}
		next.ServeHTTP(w, r)
	})
}

func clearDeadlines(w http.ResponseWriter, r *http.Request, log deadlineWarner) {
	controller := http.NewResponseController(w)
	for _, clear := range []func(time.Time) error{controller.SetReadDeadline, controller.SetWriteDeadline} {
		if err := clear(time.Time{}); err != nil {
			log.WarnContext(r.Context(), LogStreamDeadlineNotCleared, logger.FieldMethod, r.URL.Path, logger.FieldError, err)
		}
	}
}

// StreamingMethods returns the full names of the streaming methods of the
// services registered on a server.
func StreamingMethods(services map[string]grpc.ServiceInfo) map[string]bool {
	methods := make(map[string]bool)
	for service, info := range services {
		for _, method := range info.Methods {
			if method.IsClientStream || method.IsServerStream {
				methods[fullMethodSeparator+service+fullMethodSeparator+method.Name] = true
			}
		}
	}
	return methods
}
