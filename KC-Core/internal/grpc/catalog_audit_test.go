package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var (
	catalogTestManufacturerID = uuid.MustParse("1b4e28ba-2fa1-11d2-883f-0016d3cca427")
	catalogTestModelID        = uuid.MustParse("6fa459ea-ee8a-3ca4-894e-db77e160355e")
	catalogTestBlueprintID    = uuid.MustParse("886313e1-3b8a-5372-9b90-0c9aee199e5d")
)

const (
	catalogTestManufacturer = "Acme Sensors"
	catalogTestModel        = "Acme T1"
	catalogTestVersion      = "1.2.0"
)

// customCatalog is a tenant's Custom catalog holding one entry of each kind.
type customCatalog struct {
	grpcservices.BlueprintService
}

func (customCatalog) CreateManufacturer(context.Context, *grpcservices.ManufacturerCreateRequest) (*models.Manufacturer, error) {
	return &models.Manufacturer{ID: catalogTestManufacturerID, Name: catalogTestManufacturer}, nil
}

func (customCatalog) GetDeviceModel(context.Context, uuid.UUID) (*models.DeviceModel, error) {
	return &models.DeviceModel{ID: catalogTestModelID, Name: catalogTestModel}, nil
}

func (customCatalog) DeleteDeviceModel(context.Context, uuid.UUID) error { return nil }

func (customCatalog) GetBlueprint(context.Context, uuid.UUID) (*models.Blueprint, error) {
	return &models.Blueprint{ID: catalogTestBlueprintID, Version: catalogTestVersion}, nil
}

func (customCatalog) UpdateBlueprint(context.Context, uuid.UUID, *grpcservices.BlueprintUpdateRequest) (*models.Blueprint, error) {
	return &models.Blueprint{ID: catalogTestBlueprintID, Version: catalogTestVersion}, nil
}

func catalogService(recorder AuditRecorder) *CoreService {
	return testCoreService(coreFields{blueprintSvc: customCatalog{}, audit: recorder, log: logger.NewNop()})
}

// onlyEvent is the single audit event a catalog change recorded.
func onlyEvent(t *testing.T, recorder *captureAuditRecorder) audit.Event {
	t.Helper()
	require.Len(t, recorder.events, 1)
	return recorder.events[0]
}

// Catalog changes are filed under the endpoint category, which the endpoint
// managers who read the catalog may read, so their open catalog views refresh.
func TestCatalogChanges_AreRecordedForEndpointManagers(t *testing.T) {
	cases := []struct {
		name      string
		change    func(*CoreService) error
		eventType string
		entryID   uuid.UUID
		entryName string
	}{
		{"manufacturer created", func(s *CoreService) error {
			_, err := s.CreateManufacturer(actorCtx(), &pb.CreateManufacturerRequest{Name: catalogTestManufacturer})
			return err
		}, models.EventTypeManufacturerCreated, catalogTestManufacturerID, catalogTestManufacturer},
		{"device model deleted", func(s *CoreService) error {
			_, err := s.DeleteDeviceModel(actorCtx(), &pb.DeleteDeviceModelRequest{Id: catalogTestModelID.String()})
			return err
		}, models.EventTypeDeviceModelDeleted, catalogTestModelID, catalogTestModel},
		{"blueprint updated", func(s *CoreService) error {
			_, err := s.UpdateBlueprint(actorCtx(), &pb.UpdateBlueprintRequest{Id: catalogTestBlueprintID.String(), Version: catalogTestVersion})
			return err
		}, models.EventTypeBlueprintUpdated, catalogTestBlueprintID, catalogTestVersion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &captureAuditRecorder{}
			require.NoError(t, tc.change(catalogService(recorder)))

			event := onlyEvent(t, recorder)
			assert.Equal(t, tc.eventType, event.EventType)
			assert.Equal(t, models.EventCategoryEndpoint, event.Category)
			assert.Equal(t, auditTestTenant, event.TenantID)
			assert.Equal(t, tc.entryName, event.SourceName)
			assert.Equal(t, tc.entryID.String(), event.Details[models.EventDetailKeyCatalogID])
		})
	}
}

// A refused change records nothing.
func TestCatalogChanges_RefusedChangeIsNotRecorded(t *testing.T) {
	recorder := &captureAuditRecorder{}
	_, err := catalogService(recorder).CreateManufacturer(actorCtx(), &pb.CreateManufacturerRequest{})
	require.Error(t, err)
	assert.Empty(t, recorder.events)
}

// Without a tenant the change cannot be filed; it is reported, not recorded.
func TestCatalogChanges_WithoutTenantAreNotRecorded(t *testing.T) {
	recorder := &captureAuditRecorder{}
	_, err := catalogService(recorder).CreateManufacturer(testutil.TestContext(), &pb.CreateManufacturerRequest{Name: catalogTestManufacturer})
	require.NoError(t, err)
	assert.Empty(t, recorder.events)
}
