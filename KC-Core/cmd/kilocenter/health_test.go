package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

const (
	testVersion     = "test"
	testRowDatabase = "database"
	testRowBSSCI    = "bssci"
)

type fixedChecker struct{ status health.Status }

func (c fixedChecker) Check(context.Context) *health.Check { return &health.Check{Status: c.status} }

func serve(mux *http.ServeMux, route string) int {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
	return rec.Code
}

func TestHealthMux_ReadinessAnswersOnlyForTheReadinessRows(t *testing.T) {
	status := health.NewService(logger.NewNop(), testVersion)
	readiness := health.NewService(logger.NewNop(), testVersion)
	database := fixedChecker{status: health.StatusHealthy}
	status.RegisterChecker(testRowDatabase, database)
	readiness.RegisterChecker(testRowDatabase, database)
	status.RegisterChecker(testRowBSSCI, fixedChecker{status: health.StatusUnhealthy})

	mux := newHealthMux(logger.NewNop(), status, readiness)
	assert.Equal(t, http.StatusServiceUnavailable, serve(mux, health.RouteHealth), "/health reports the failing listener")
	assert.Equal(t, http.StatusOK, serve(mux, health.RouteHealthReady), "a listener waiting for certificates does not take the process out of rotation")
	assert.Equal(t, http.StatusOK, serve(mux, health.RouteHealthLive))
	assert.Equal(t, http.StatusOK, serve(mux, health.RouteHealthPing))

	readiness.RegisterChecker(testRowDatabase, fixedChecker{status: health.StatusUnhealthy})
	assert.Equal(t, http.StatusServiceUnavailable, serve(mux, health.RouteHealthReady), "a failing readiness row makes the process unready")
}
