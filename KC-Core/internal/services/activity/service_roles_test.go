package activity

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const testTenant int64 = 1

var testEui = []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x09, 0xe6}

// categoryEventReader holds one event per category and applies the category filter like the store does.
type categoryEventReader struct {
	calls int
}

func (r *categoryEventReader) list(filters *grpcservices.EventFilters) ([]*grpcservices.Event, int64, error) {
	r.calls++
	all := []string{
		models.EventCategoryBaseStation, models.EventCategoryEndpoint, models.EventCategorySecurity,
		models.EventCategoryAudit, models.EventCategorySystem,
	}
	var out []*grpcservices.Event
	for _, c := range all {
		if len(filters.Categories) > 0 && !slices.Contains(filters.Categories, c) {
			continue
		}
		out = append(out, &grpcservices.Event{ID: c, TenantID: testTenant, Category: c})
	}
	return out, int64(len(out)), nil
}

func (r *categoryEventReader) ListByBaseStation(_ context.Context, _ int64, _ []byte, f *grpcservices.EventFilters, _, _ int) ([]*grpcservices.Event, int64, error) {
	return r.list(f)
}

func (r *categoryEventReader) ListByEndPoint(_ context.Context, _ int64, _ []byte, f *grpcservices.EventFilters, _, _ int) ([]*grpcservices.Event, int64, error) {
	return r.list(f)
}

type noMessages struct{}

func (noMessages) ListBaseStationMessages(context.Context, int64, []byte, *grpcservices.MessageFilters, int, int) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}

func (noMessages) ListMessages(context.Context, int64, *grpcservices.MessageFilters, int, int) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}

func eventCategories(result *grpcservices.ActivityListResult) []string {
	var got []string
	for _, item := range result.Items {
		got = append(got, item.Event.Category)
	}
	return got
}

func TestActivity_EventsFollowTheCallersReadableCategories(t *testing.T) {
	type feed func(*Service, context.Context) (*grpcservices.ActivityListResult, error)
	feeds := map[string]feed{
		"base station": func(s *Service, ctx context.Context) (*grpcservices.ActivityListResult, error) {
			return s.ListBaseStationActivity(ctx, testTenant, testEui, nil, 0, "")
		},
		"endpoint": func(s *Service, ctx context.Context) (*grpcservices.ActivityListResult, error) {
			return s.ListEndpointActivity(ctx, testTenant, testEui, nil, 0, "")
		},
	}
	cases := []struct {
		name    string
		roles   authz.Roles
		want    []string
		reached bool
	}{
		{name: "administrator", roles: authz.AllRoles, reached: true, want: []string{
			models.EventCategoryBaseStation, models.EventCategoryEndpoint, models.EventCategorySecurity,
			models.EventCategoryAudit, models.EventCategorySystem,
		}},
		{name: "base station manager", roles: authz.Roles{BaseStationManager: true}, reached: true, want: []string{models.EventCategoryBaseStation}},
		{name: "endpoint manager", roles: authz.Roles{EndpointManager: true}, reached: true, want: []string{models.EventCategoryEndpoint}},
		{name: "tenant manager", roles: authz.Roles{TenantManager: true}},
	}
	for feedName, list := range feeds {
		for _, tc := range cases {
			t.Run(feedName+"/"+tc.name, func(t *testing.T) {
				reader := &categoryEventReader{}
				svc := New(reader, noMessages{}, logger.NewNop())

				result, err := list(svc, authz.WithRoles(testutil.TestContextWithTenant(testTenant), tc.roles))

				require.NoError(t, err)
				assert.Equal(t, tc.reached, reader.calls > 0)
				assert.ElementsMatch(t, tc.want, eventCategories(result))
			})
		}
	}
}
