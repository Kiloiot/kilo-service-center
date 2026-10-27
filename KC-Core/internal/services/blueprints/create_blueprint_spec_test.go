package blueprints

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// TC-CM-105: CreateBlueprint keeps its JSON-only check; the decoder's check is not applied here.
func TestCreateBlueprint_AcceptsASpecTheDecoderRefuses(t *testing.T) {
	modelID := uuid.New()
	spec := json.RawMessage(`{"version":"1.0","typeEui":"70B3D59CD0000095"}`)
	dmRepo := &mockDeviceModelRepository{
		getByIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
			return &models.DeviceModel{ID: modelID}, nil
		},
	}
	var stored *models.BlueprintCreateParams
	bpRepo := &mockBlueprintRepository{
		createFn: func(_ context.Context, p *models.BlueprintCreateParams) (*models.Blueprint, error) {
			stored = p
			return &models.Blueprint{ID: uuid.New()}, nil
		},
	}
	svc := New(&mockManufacturerRepository{}, dmRepo, bpRepo, 1, nil, &testLogger{}, &mockTxRunner{}, nil)

	if _, err := svc.CreateBlueprint(testutil.TestContext(), &grpcservices.BlueprintCreateRequest{
		DeviceModelID: modelID, Version: "1.0", SpecJSON: spec,
	}); err != nil {
		t.Fatalf("CreateBlueprint refused a specification without uplink and downlink: %v", err)
	}
	if stored == nil || string(stored.SpecJSON) != string(spec) {
		t.Fatalf("the specification was not stored as sent: %+v", stored)
	}
}
