// Package bssciservices implements BSSCI service layer components.
package bssciservices

import (
	"context"
	"fmt"
	"time"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
)

// auditLogger implements bssci.AuditLogger interface
//
// This service records downlink audit events to the system_events table per BSSCI §5.12 and §5.14.
// All events include tenant isolation, endpoint correlation, and base station context for compliance auditing.
type auditLogger struct {
	eventStore interfaces.SystemEventStore
}

// NewAuditLogger creates a new audit logger service
//
// Parameters:
//   - eventStore: System event store for persisting audit events
//
// The eventStore dependency should be reused from storage.SystemEvents() to avoid duplicate instantiation.
func NewAuditLogger(eventStore interfaces.SystemEventStore) bssci.AuditLogger {
	return &auditLogger{
		eventStore: eventStore,
	}
}

// RecordQueueAck logs successful downlink queue acknowledgment per BSSCI §5.12
//
// Creates a system event when the base station acknowledges a dlDataQue request.
// The event captures the queue ID, endpoint EUI, and base station details for audit trail.
//
// Event Structure:
//   - EventType: "dl_data_queue_acknowledged"
//   - Category: "downlink"
//   - Severity: "info"
//   - SourceType: "endpoint"
//
// Parameters:
//   - ctx: Request context for event creation
//   - tenant: Tenant ID string (formatted via formatTenantID helper)
//   - session: Active BSSCI session providing base station identification
//   - epEui: Endpoint EUI for correlation
//   - queueID: Queue ID assigned to the downlink
//   - opId: Operation ID for request/response correlation
//
// Returns error only on critical event store failures (logging is best-effort).
func (a *auditLogger) RecordQueueAck(ctx context.Context, tenant string, session *bssci.Session,
	epEui uint64, queueID int64, _ int64) error {

	event := &models.SystemEvent{
		TenantID:    tenant,
		EventType:   "dl_data_queue_acknowledged",
		Category:    models.EventCategoryMessage,
		Severity:    bssci.SeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleDLDataQueued, session.UserProvidedName),
		Description: fmt.Sprintf("Downlink data queued for endpoint %016X at base station %s (Queue ID: %d)", epEui, session.UserProvidedName, queueID),
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  fmt.Sprintf("%016X", epEui),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return a.eventStore.CreateEvent(ctx, event)
}

// RecordDLResult logs downlink transmission result per BSSCI §5.14
//
// Creates a system event when downlink transmission completes.
// The event type and severity are determined by the Result field per BSSCI §3.14.1:
//   - "sent": Success event (info severity)
//   - "expired": Expiration event (warning severity)
//   - "invalid": Failure event (error severity)
//
// Event Structure:
//   - EventType: "dl_data_sent", "dl_data_expired", or "dl_data_invalid"
//   - Category: "downlink"
//   - Severity: "info", "warning", or "error"
//   - SourceType: "endpoint"
//
// Parameters:
//   - ctx: Request context for event creation
//   - tenant: Tenant ID string
//   - session: Active BSSCI session providing base station identification
//   - result: DL Data Result message from base station (contains Result, QueId, EpEui)
//
// Returns error only on critical event store failures.
func (a *auditLogger) RecordDLResult(ctx context.Context, tenant string, session *bssci.Session,
	result *mioty.DLDataResult) error {

	var eventType, severity, title, description string

	// Map BSSCI result enum to event attributes per §3.14.1
	switch result.Result {
	case mioty.ResultSent:
		eventType = "dl_data_sent"
		severity = bssci.SeverityInfo
		title = fmt.Sprintf(models.EventTitleDLDataSent, session.UserProvidedName)
		description = fmt.Sprintf("Downlink data successfully sent to endpoint %016X via base station %s (Queue ID: %d)",
			result.EpEui, session.UserProvidedName, result.QueId)
	case mioty.ResultExpired:
		eventType = "dl_data_expired"
		severity = bssci.SeverityWarning
		title = fmt.Sprintf(models.EventTitleDLDataExpired, session.UserProvidedName)
		description = fmt.Sprintf("Downlink data expired for endpoint %016X at base station %s (Queue ID: %d)",
			result.EpEui, session.UserProvidedName, result.QueId)
	case mioty.ResultInvalid:
		eventType = "dl_data_invalid"
		severity = bssci.SeverityError
		title = fmt.Sprintf(models.EventTitleDLDataInvalid, session.UserProvidedName)
		description = fmt.Sprintf("Downlink data invalid for endpoint %016X at base station %s (Queue ID: %d)",
			result.EpEui, session.UserProvidedName, result.QueId)
	default:
		// Fallback for unexpected result values (should never happen after validation)
		eventType = "dl_data_unknown_result"
		severity = bssci.SeverityError
		title = fmt.Sprintf(models.EventTitleDLDataUnknownResult, session.UserProvidedName)
		description = fmt.Sprintf("Downlink reported unknown result '%s' for endpoint %016X at base station %s (Queue ID: %d)",
			result.Result, result.EpEui, session.UserProvidedName, result.QueId)
	}

	event := &models.SystemEvent{
		TenantID:    tenant,
		EventType:   eventType,
		Category:    models.EventCategoryMessage,
		Severity:    severity,
		Title:       title,
		Description: description,
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  fmt.Sprintf("%016X", result.EpEui),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return a.eventStore.CreateEvent(ctx, event)
}
