package builders

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// buildAuditRecorder builds the recorder every audited KC-Core action writes
// through, over the system event store; a wiring fault stops start-up.
func buildAuditRecorder(infra *Infrastructure, log logger.Logger) *audit.Recorder {
	auditEmitter, err := audit.NewEmitter(infra.Repos.SystemEvents, infra.Clock)
	if err != nil {
		log.Fatal(LogAuditEmitterCreateFailed, logger.Err(err))
	}
	auditDrops, err := audit.NewPrometheusDropCounter(prometheus.DefaultRegisterer)
	if err != nil {
		log.Fatal(LogAuditRecorderCreateFailed, logger.Err(err))
	}
	auditRecorder, err := audit.NewRecorder(auditEmitter, infra.LoggerIface, auditDrops)
	if err != nil {
		log.Fatal(LogAuditRecorderCreateFailed, logger.Err(err))
	}
	return auditRecorder
}
