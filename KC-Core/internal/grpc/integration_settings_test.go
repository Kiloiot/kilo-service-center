package grpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	integrationTestSecret      = "s3cret-password"
	integrationTestURLKey      = "url"
	integrationTestPasswordKey = "password"
)

// TestIntegrationSettings_AreWriteOnly: a stored setting may be a credential,
// so every read names the configured settings without their values.
func TestIntegrationSettings_AreWriteOnly(t *testing.T) {
	stored := createTestIntegration()
	stored.Config = json.RawMessage(`{"` + integrationTestURLKey + `": "https://example.com/webhook", "` +
		integrationTestPasswordKey + `": "` + integrationTestSecret + `"}`)
	svc := createTestIntegrationService()
	svc.integrationSvc = &mockIntegrationService{
		getByIDFunc: func(context.Context, int64, int64) (*models.Integration, error) { return stored, nil },
		listFunc: func(context.Context, int64, int, int) ([]*models.Integration, int64, error) {
			return []*models.Integration{stored}, 1, nil
		},
	}
	ctx := testutil.TestContextWithTenant(stored.TenantID)

	got, err := svc.GetIntegration(ctx, &pb.GetIntegrationRequest{Id: stored.ID})
	require.NoError(t, err)
	list, err := svc.ListIntegrations(ctx, &pb.ListIntegrationsRequest{})
	require.NoError(t, err)
	require.Len(t, list.GetIntegrations(), 1)

	for _, integration := range []*pb.Integration{got, list.GetIntegrations()[0]} {
		fields := integration.GetConfig().AsMap()
		keys := make([]string, 0, len(fields))
		for key, value := range fields {
			keys = append(keys, key)
			assert.Equal(t, integrationConfigValueMasked, value, "setting %s must not be returned", key)
		}
		assert.ElementsMatch(t, []string{integrationTestURLKey, integrationTestPasswordKey}, keys, "the configured settings stay named")
		raw, marshalErr := json.Marshal(integration.GetConfig())
		require.NoError(t, marshalErr)
		assert.NotContains(t, string(raw), integrationTestSecret)
	}
}
