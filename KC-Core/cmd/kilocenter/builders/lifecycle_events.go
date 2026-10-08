package builders

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// serviceEventWriter stores the service lifecycle events.
type serviceEventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// ServiceHostname names this host in lifecycle events and the release info;
// the name only labels them, so a lookup failure is logged and left empty.
func ServiceHostname(ctx context.Context, log logger.Logger) string {
	hostname, err := os.Hostname()
	if err != nil {
		log.WarnContext(ctx, LogFailedResolveHostname, logger.FieldError, err)
		return ""
	}
	return hostname
}

// emitServiceStarted records the service-started event with the listening
// ports; a failure is logged because startup does not depend on it.
func emitServiceStarted(ctx context.Context, log logger.Logger, events serviceEventWriter, cfg *config.Config, infra *Infrastructure, hostname string) {
	details, err := json.Marshal(map[string]interface{}{
		"service":   CoreServiceSourceName,
		"version":   infra.VersionInfo.Version,
		"gitCommit": infra.VersionInfo.GitCommit,
		"host":      hostname,
		"ports": map[string]int{
			"grpc":  cfg.GRPC.Port,
			"bssci": cfg.Protocol.BSCIPort,
			"scaci": cfg.Protocol.SCACIPort,
		},
	})
	if err != nil {
		log.WarnContext(ctx, LogFailedEmitServiceStartedEvent, logger.FieldError, err)
		return
	}
	listenInfo := fmt.Sprintf(listenInfoFmt, cfg.GRPC.Port, cfg.Protocol.BSCIPort, cfg.Protocol.SCACIPort)
	if err := events.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(infra.TenantID, 10),
		EventType:   models.EventTypeServiceStarted,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStartedFmt, CoreServiceDisplayName, hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStartedFmt, CoreServiceDisplayName, infra.VersionInfo.Version, listenInfo),
		SourceType:  models.SourceTypeSystem,
		SourceName:  CoreServiceSourceName,
		Details:     details,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}); err != nil {
		log.WarnContext(ctx, LogFailedEmitServiceStartedEvent, logger.FieldError, err)
		return
	}
	log.InfoContext(ctx, LogServiceStartedEventEmitted)
}
