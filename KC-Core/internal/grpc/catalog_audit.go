package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// catalogChangeKind is the audit wording of one kind of device catalog change.
type catalogChangeKind struct {
	eventType      string
	title          string
	descriptionFmt string
}

// Device catalog changes the catalog handlers record.
var (
	manufacturerCreated = catalogChangeKind{models.EventTypeManufacturerCreated, models.EventTitleManufacturerCreated, models.EventDescriptionManufacturerCreatedFmt}
	manufacturerUpdated = catalogChangeKind{models.EventTypeManufacturerUpdated, models.EventTitleManufacturerUpdated, models.EventDescriptionManufacturerUpdatedFmt}
	manufacturerDeleted = catalogChangeKind{models.EventTypeManufacturerDeleted, models.EventTitleManufacturerDeleted, models.EventDescriptionManufacturerDeletedFmt}
	deviceModelCreated  = catalogChangeKind{models.EventTypeDeviceModelCreated, models.EventTitleDeviceModelCreated, models.EventDescriptionDeviceModelCreatedFmt}
	deviceModelUpdated  = catalogChangeKind{models.EventTypeDeviceModelUpdated, models.EventTitleDeviceModelUpdated, models.EventDescriptionDeviceModelUpdatedFmt}
	deviceModelDeleted  = catalogChangeKind{models.EventTypeDeviceModelDeleted, models.EventTitleDeviceModelDeleted, models.EventDescriptionDeviceModelDeletedFmt}
	blueprintCreated    = catalogChangeKind{models.EventTypeBlueprintCreated, models.EventTitleBlueprintCreated, models.EventDescriptionBlueprintCreatedFmt}
	blueprintUpdated    = catalogChangeKind{models.EventTypeBlueprintUpdated, models.EventTitleBlueprintUpdated, models.EventDescriptionBlueprintUpdatedFmt}
	blueprintDeleted    = catalogChangeKind{models.EventTypeBlueprintDeleted, models.EventTitleBlueprintDeleted, models.EventDescriptionBlueprintDeletedFmt}
)

// recordCatalogChange records a committed device catalog change under the
// endpoint category, which the endpoint managers who read the catalog may read.
func (s *BlueprintHandlers) recordCatalogChange(ctx context.Context, kind catalogChangeKind, id uuid.UUID, name string) {
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		s.log.WarnContext(ctx, audit.LogEventDropped, logger.FieldEventType, kind.eventType, logger.FieldError, err)
		return
	}
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		Category:    models.EventCategoryEndpoint,
		EventType:   kind.eventType,
		Title:       kind.title,
		Description: fmt.Sprintf(kind.descriptionFmt, name),
		SourceID:    &id,
		SourceName:  name,
		Details: map[string]any{
			models.EventDetailKeyCatalogID:   id.String(),
			models.EventDetailKeyCatalogName: name,
		},
	})
}
