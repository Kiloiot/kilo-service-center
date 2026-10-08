package builders

import (
	"fmt"
	"net"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Display names of the service-status rows shown on the dashboard.
const (
	statusNameCore       = "KC-Core"
	statusNamePostgreSQL = "PostgreSQL"
	statusNameBSSCI      = "BSSCI"
	statusNameSCACI      = "SCACI"
	statusNameGRPC       = "gRPC (internal)"
	statusNameIdentity   = "KC-Identity"
	statusNameMQTT       = "MQTT"
	statusNameArchival   = "Archival"

	selfHealthURLFmt = "http://%s%s"
)

// statusCheckerDeps carries the running components the infrastructure rows
// probe. MQTTConnected is nil when MQTT is disabled; Archival is the checker
// the composition root chose for the archival scheduler.
type statusCheckerDeps struct {
	DB            health.PostgreSQLHealthDB
	MQTTConnected func() bool
	Archival      health.Checker
	Log           logger.Logger
}

// statusRow pairs a dashboard row with the address its checker probes.
type statusRow struct {
	Name string
	URL  string
}

// statusBoard holds the checkers behind the dashboard rows. Every row reports
// on /health and the dashboard; only the rows this process needs to serve its
// API also decide /health/ready.
type statusBoard struct {
	status    *health.Service
	readiness *health.Service
	rows      []statusRow
}

func newStatusBoard(log logger.Logger, version string) *statusBoard {
	return &statusBoard{status: health.NewService(log, version), readiness: health.NewService(log, version)}
}

// track adds a row that reports without deciding readiness.
func (b *statusBoard) track(name, url string, checker health.Checker) {
	b.status.RegisterChecker(name, checker)
	b.rows = append(b.rows, statusRow{Name: name, URL: url})
}

// require adds a row the API cannot be served without; it also decides
// readiness.
func (b *statusBoard) require(name, url string, checker health.Checker) {
	b.track(name, url, checker)
	b.readiness.RegisterChecker(name, checker)
}

// registerStatusCheckers adds the rows of the infrastructure KC-Core runs on
// or is configured to reach, deriving every probe address from the
// component's own configuration. The listener rows are added by the builders
// that start those listeners.
func registerStatusCheckers(board *statusBoard, cfg *pkgconfig.Config, deps statusCheckerDeps) {
	timeout := cfg.Status.Timeout

	selfURL := fmt.Sprintf(selfHealthURLFmt,
		pkgconfig.ListenerProbeAddress(pkgconfig.ProbeLoopbackHost, cfg.General.HealthCheckPort), health.RouteHealthPing)
	board.track(statusNameCore, selfURL, health.NewHTTPChecker(selfURL, timeout, deps.Log))

	board.require(statusNamePostgreSQL, net.JoinHostPort(cfg.Storage.Host, strconv.Itoa(cfg.Storage.Port)),
		health.NewPostgreSQLChecker(deps.DB, deps.Log))

	if cfg.Identity.Address != "" {
		board.track(statusNameIdentity, cfg.Identity.Address, health.NewTCPChecker(cfg.Identity.Address, timeout, deps.Log))
	}
	if deps.MQTTConnected != nil {
		board.track(statusNameMQTT, pkgconfig.GetMQTTBrokerURL(cfg.MQTT), health.NewMQTTChecker(deps.MQTTConnected))
	}
	board.track(statusNameArchival, "", deps.Archival)
}
