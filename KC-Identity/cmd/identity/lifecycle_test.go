package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var (
	errEventStoreDown = errors.New("event store down")
	errClientGone     = errors.New("client gone")
)

const (
	testTenantID = int64(1)
	testVersion  = "1.2.3"
	testGRPCPort = 50052
)

// refusingEventWriter fails every lifecycle event write.
type refusingEventWriter struct{ attempts int }

func (w *refusingEventWriter) CreateEvent(context.Context, *models.SystemEvent) error {
	w.attempts++
	return errEventStoreDown
}

// refusingWriter is a response writer whose body writes fail.
type refusingWriter struct{ *httptest.ResponseRecorder }

func (w refusingWriter) Write([]byte) (int, error) { return 0, errClientGone }

// logRecorder records warning and info messages and discards everything else.
type logRecorder struct {
	logger.Logger
	warnings []string
	infos    []string
}

func (r *logRecorder) WarnContext(_ context.Context, msg string, _ ...interface{}) {
	r.warnings = append(r.warnings, msg)
}

func (r *logRecorder) InfoContext(_ context.Context, msg string, _ ...interface{}) {
	r.infos = append(r.infos, msg)
}

func TestLifecycleEvents_LogAFailedWriteInsteadOfReportingItRecorded(t *testing.T) {
	writer := &refusingEventWriter{}
	log := &logRecorder{Logger: logger.NewNop()}
	events := newLifecycleEvents(writer, log, testTenantID, testVersion)

	events.started(testutil.TestContext(), testGRPCPort)
	events.stopped()

	assert.Equal(t, 2, writer.attempts)
	assert.Equal(t, []string{LogLifecycleEventFailed, LogLifecycleEventFailed}, log.warnings)
	assert.Empty(t, log.infos, "a failed write is never reported as emitted")
}

func TestHealthEndpoints_LogAFailedResponseWrite(t *testing.T) {
	endpoints := map[string]func(h *healthHandler) http.HandlerFunc{
		routeHealth:      func(h *healthHandler) http.HandlerFunc { return h.ServeHTTP },
		routeHealthReady: func(h *healthHandler) http.HandlerFunc { return h.serveReady },
		routeHealthPing:  func(h *healthHandler) http.HandlerFunc { return h.servePing },
		routeHealthLive:  func(h *healthHandler) http.HandlerFunc { return h.serveLive },
	}
	for route, endpoint := range endpoints {
		t.Run(route, func(t *testing.T) {
			log := &logRecorder{Logger: logger.NewNop()}
			h := &healthHandler{log: log}

			endpoint(h)(refusingWriter{httptest.NewRecorder()}, httptest.NewRequest(http.MethodGet, route, nil))

			assert.Equal(t, []string{LogHealthResponseWriteFailed}, log.warnings)
		})
	}
}
