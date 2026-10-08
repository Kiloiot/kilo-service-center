package builders

import (
	"database/sql"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testLoopbackAny        = "127.0.0.1:0"
	testUnreachableDBDSN   = "postgres://kilocenter@127.0.0.1:1/kilocenter?sslmode=disable&connect_timeout=1"
	testStatusProbeTimeout = 2 * time.Second
	testStorageHost        = "db.internal"
	testStoragePort        = 5432
	testBrokerHost         = "broker.internal"
	testBrokerPort         = 1883
	testVersion            = "test"
	testArchivalDisabled   = "archival off"
)

type fakeArchivalStatus struct{ running bool }

func (f fakeArchivalStatus) GetStatus() map[string]interface{} {
	return map[string]interface{}{"running": f.running}
}

type fakeListener struct{ listening bool }

func (f *fakeListener) Listening() bool { return f.listening }

func openListener(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", testLoopbackAny)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

func listenerPort(t *testing.T, ln net.Listener) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return port
}

func closedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", testLoopbackAny)
	require.NoError(t, err)
	port := listenerPort(t, ln)
	require.NoError(t, ln.Close())
	return port
}

func unreachableDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", testUnreachableDBDSN)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func statusRowsByName(rows []statusRow) map[string]string {
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Name] = row.URL
	}
	return out
}

func infraConfig(t *testing.T) *pkgconfig.Config {
	t.Helper()
	cfg := &pkgconfig.Config{}
	cfg.Status.Timeout = testStatusProbeTimeout
	cfg.General.HealthCheckPort = closedPort(t)
	cfg.Storage.Host = testStorageHost
	cfg.Storage.Port = testStoragePort
	cfg.MQTT.Host = testBrokerHost
	cfg.MQTT.Port = testBrokerPort
	return cfg
}

func TestRegisterStatusCheckers_InfrastructureRows(t *testing.T) {
	identityListener := openListener(t)
	cfg := infraConfig(t)
	cfg.Identity.Address = identityListener.Addr().String()

	board := newStatusBoard(logger.Get(), testVersion)
	registerStatusCheckers(board, cfg, statusCheckerDeps{
		DB:            unreachableDB(t),
		MQTTConnected: func() bool { return true },
		Archival:      &archivalHealthChecker{scheduler: fakeArchivalStatus{running: true}, log: logger.Get()},
		Log:           logger.Get(),
	})

	resp := board.status.CheckHealth(testutil.TestContext())
	healthy := map[string]bool{
		statusNameCore:       false,
		statusNamePostgreSQL: false,
		statusNameIdentity:   true,
		statusNameMQTT:       true,
		statusNameArchival:   true,
	}
	require.Len(t, resp.Checks, len(healthy))
	for name, want := range healthy {
		check, ok := resp.Checks[name]
		require.True(t, ok, "row %q missing", name)
		assert.Equal(t, want, check.Status == health.StatusHealthy, "row %q: %s", name, check.Message)
	}

	urls := statusRowsByName(board.rows)
	assert.Equal(t, cfg.Identity.Address, urls[statusNameIdentity])
	assert.Equal(t, "http://localhost:"+strconv.Itoa(cfg.General.HealthCheckPort)+health.RouteHealthPing, urls[statusNameCore])
	assert.Equal(t, testStorageHost+":"+strconv.Itoa(testStoragePort), urls[statusNamePostgreSQL])
	assert.Equal(t, pkgconfig.GetMQTTBrokerURL(cfg.MQTT), urls[statusNameMQTT])
	assert.Equal(t, "", urls[statusNameArchival])
}

func TestRegisterStatusCheckers_SkipsComponentsThatAreNotConfigured(t *testing.T) {
	cfg := infraConfig(t)
	cfg.Identity.Address = ""

	board := newStatusBoard(logger.Get(), testVersion)
	registerStatusCheckers(board, cfg, statusCheckerDeps{
		DB:       unreachableDB(t),
		Archival: health.NewDisabledChecker(testArchivalDisabled),
		Log:      logger.Get(),
	})

	resp := board.status.CheckHealth(testutil.TestContext())
	want := []string{statusNameCore, statusNamePostgreSQL, statusNameArchival}
	assert.ElementsMatch(t, want, keysOf(resp.Checks))
	assert.ElementsMatch(t, want, keysOf(statusRowsByName(board.rows)))
	assert.Equal(t, health.StatusDisabled, resp.Checks[statusNameArchival].Status, "disabled archival reports disabled, not unhealthy")
}

// Readiness answers only for what this process needs to serve its API:
// dependencies, background jobs and its own protocol listeners never gate it.
func TestReadiness_IsNotDecidedByDependencyRows(t *testing.T) {
	cfg := infraConfig(t)
	cfg.Identity.Address = testLoopbackAny

	board := newStatusBoard(logger.Get(), testVersion)
	registerStatusCheckers(board, cfg, statusCheckerDeps{
		DB:            unreachableDB(t),
		MQTTConnected: func() bool { return false },
		Archival:      health.NewDisabledChecker(testArchivalDisabled),
		Log:           logger.Get(),
	})
	board.track(statusNameBSSCI, "", health.NewListenerChecker(&fakeListener{listening: false}))
	board.require(statusNameGRPC, "", health.NewListenerChecker(&fakeListener{listening: true}))

	readinessRows := keysOf(board.readiness.CheckHealth(testutil.TestContext()).Checks)
	for _, dependency := range []string{statusNameArchival, statusNameMQTT, statusNameIdentity, statusNameBSSCI, statusNameCore} {
		assert.NotContains(t, readinessRows, dependency, "readiness must not depend on the %s row", dependency)
	}
	assert.ElementsMatch(t, []string{statusNamePostgreSQL, statusNameGRPC}, readinessRows)
	assert.ElementsMatch(t, []string{statusNameCore, statusNamePostgreSQL, statusNameIdentity, statusNameMQTT, statusNameArchival, statusNameBSSCI, statusNameGRPC},
		keysOf(board.status.CheckHealth(testutil.TestContext()).Checks), "/health keeps reporting every row")
}

func TestReadiness_FollowsTheGRPCServerState(t *testing.T) {
	board := newStatusBoard(logger.Get(), testVersion)
	grpcState := &fakeListener{listening: false}
	board.require(statusNameGRPC, "", health.NewListenerChecker(grpcState))

	assert.Equal(t, health.StatusUnhealthy, board.readiness.CheckHealth(testutil.TestContext()).Status, "not ready before the gRPC server serves")
	grpcState.listening = true
	assert.Equal(t, health.StatusHealthy, board.readiness.CheckHealth(testutil.TestContext()).Status)
}

func keysOf[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
