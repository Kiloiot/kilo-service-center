package basestation

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// eventKeyBsEui is the canonical key for base station EUI in event details
// Matches bssci.EventKeyBsEui but defined locally to avoid import cycle
const eventKeyBsEui = "bsEui"

// PersistentEventRecorder implements EventRecorder and persists events to database
type PersistentEventRecorder struct {
	logger     logger.Logger
	eventStore interfaces.SystemEventStore
	tenantID   string
}

// NewPersistentEventRecorder creates a new persistent event recorder
func NewPersistentEventRecorder(log logger.Logger, eventStore interfaces.SystemEventStore, tenantID string) *PersistentEventRecorder {
	return &PersistentEventRecorder{
		logger:     log,
		eventStore: eventStore,
		tenantID:   tenantID,
	}
}

// RecordEvent records an event to both logger and database
func (r *PersistentEventRecorder) RecordEvent(ctx context.Context, eui [8]byte, eventType string, data map[string]interface{}) error {
	euiStr := hex.EncodeToString(eui[:])

	// Derive tenant from context (session-scoped), fall back to configured default
	tenantID := r.tenantID
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil && tid > 0 {
		tenantID = fmt.Sprintf("%d", tid)
	}

	// Determine severity based on event type
	severity := models.EventSeverityInfo
	var title string
	description := ""

	switch eventType {
	case models.EventTypeBaseStationOffline, "status_offline":
		severity = models.EventSeverityWarning
		basestationName := "Unknown"
		if name, ok := data["basestation_name"].(string); ok && name != "" {
			basestationName = name
		}
		title = fmt.Sprintf("Base Station \"%s\" went offline", basestationName)
		description = fmt.Sprintf("Base Station \"%s\" (EUI: %s) disconnected from the Service Center", basestationName, euiStr)

	case models.EventTypeBaseStationOnline, "status_online":
		severity = models.EventSeverityInfo
		basestationName := "Unknown"
		if name, ok := data["basestation_name"].(string); ok && name != "" {
			basestationName = name
		}
		title = fmt.Sprintf("Base Station \"%s\" connected", basestationName)
		description = fmt.Sprintf("Base Station \"%s\" (EUI: %s) successfully connected to the Service Center", basestationName, euiStr)

	case models.EventTypeBSRegistered:
		severity = models.EventSeverityInfo
		basestationName := "Unknown"
		if name, ok := data["name"].(string); ok && name != "" {
			basestationName = name
		}
		title = fmt.Sprintf("%s: %s", models.EventTitleBSRegistered, basestationName)
		description = fmt.Sprintf("Base Station \"%s\" (EUI: %s) registered with the Service Center", basestationName, euiStr)

	case models.EventTypeConnectionError:
		severity = models.EventSeverityCritical
		title = "Base station connection error"
		description = fmt.Sprintf("Base station %s encountered a connection error", euiStr)

	default:
		title = fmt.Sprintf("Base station %s: %s", euiStr, eventType)
		description = fmt.Sprintf("Event %s occurred for base station %s", eventType, euiStr)
	}

	r.logger.Info("Base station event",
		"eui", euiStr,
		"event_type", eventType,
		"severity", severity,
		"data", data,
	)

	event := &models.SystemEvent{
		TenantID:    tenantID,
		EventType:   eventType,
		Category:    models.EventCategoryBaseStation,
		Severity:    severity,
		Title:       title,
		Description: description,
		SourceType:  models.SourceTypeBaseStation,
		SourceName:  euiStr,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Always include bsEui in details for frontend compatibility
	eventData := make(map[string]interface{})
	for k, v := range data {
		eventData[k] = v
	}
	eventData[eventKeyBsEui] = euiStr // Always add bsEui for frontend

	jsonData, _ := json.Marshal(eventData)
	event.Details = jsonData

	err := r.eventStore.CreateEvent(ctx, event)
	if err != nil {
		r.logger.Error("Failed to persist event",
			"eui", euiStr,
			"error", err,
		)
	}

	return nil
}
