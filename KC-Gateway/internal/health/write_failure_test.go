package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
)

var errClientGone = errors.New("client gone")

// refusingWriter is a response writer whose body writes fail.
type refusingWriter struct{ *httptest.ResponseRecorder }

func (w refusingWriter) Write([]byte) (int, error) { return 0, errClientGone }

// warnRecorder records warning messages and discards everything else.
type warnRecorder struct {
	logger.Logger
	warnings []string
}

func (r *warnRecorder) WarnContext(_ context.Context, msg string, _ ...interface{}) {
	r.warnings = append(r.warnings, msg)
}

func TestHealthEndpoints_LogAFailedResponseWrite(t *testing.T) {
	cfg := testResilienceConfig()
	endpoints := map[string]func(h *handler) http.HandlerFunc{
		routeHealth:      func(h *handler) http.HandlerFunc { return h.serveHealth },
		routeHealthReady: func(h *handler) http.HandlerFunc { return h.serveReady },
		routeHealthLive:  func(h *handler) http.HandlerFunc { return h.serveLive },
		routeHealthPing:  func(h *handler) http.HandlerFunc { return h.servePing },
	}
	for route, endpoint := range endpoints {
		t.Run(route, func(t *testing.T) {
			log := &warnRecorder{Logger: logger.NewNop()}
			h := newHandler(newIdleConn(t), newIdleConn(t),
				resilience.NewUpstreamBreaker("core", cfg),
				resilience.NewUpstreamBreaker("identity", cfg), log)

			endpoint(h)(refusingWriter{httptest.NewRecorder()}, httptest.NewRequest(http.MethodGet, route, nil))

			assert.Equal(t, []string{logHealthResponseWriteFailed}, log.warnings)
		})
	}
}
