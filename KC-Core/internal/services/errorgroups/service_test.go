package errorgroups

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	testTenant   = int64(4)
	testLimit    = 25
	testOffset   = 50
	groupCount   = int64(3)
	sampleOpID   = "-12"
	sampleSource = "70B3D59CD0000001"
)

var now = time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)

// adminCtx is a caller the authorization interceptor admitted as an administrator.
func adminCtx() context.Context { return authz.WithRoles(testutil.TestContext(), authz.AllRoles) }

type fixedClock struct{}

func (fixedClock) Now() time.Time { return now }

var _ clock.Clock = fixedClock{}

type fakeReader struct {
	last models.ErrorGroupFilter
}

func (f *fakeReader) ListErrorGroups(_ context.Context, filter models.ErrorGroupFilter) ([]*models.EventErrorGroup, int64, error) {
	f.last = filter
	return []*models.EventErrorGroup{{EventType: models.EventTypeConnectionError, Code: "", Message: "lost", SourceName: sampleSource, FirstSeen: now.Add(-time.Hour), LastSeen: now, Count: groupCount, LastOpID: sampleOpID}}, 1, nil
}

func TestList_MapsBucketsOntoTypedSelections(t *testing.T) {
	reader := &fakeReader{}
	svc := New(reader, fixedClock{}, logger.NewNop())

	groups, total, err := svc.List(adminCtx(), testTenant, BucketControlPlane, grpcservices.ScaciWindow{}, testLimit, testOffset)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, groups, 1)
	assert.Equal(t, BucketControlPlane, groups[0].Bucket)
	assert.Equal(t, sampleSource, groups[0].SourceName)
	assert.Equal(t, groupCount, groups[0].Count)
	assert.Equal(t, sampleOpID, groups[0].LastOpID)
	assert.True(t, reader.last.IncludeSCACIFailures, "the control-plane bucket joins failed SCACI operations")
	assert.ElementsMatch(t, []string{models.EventSeverityError, models.EventSeverityCritical}, reader.last.Severities)
	assert.Contains(t, reader.last.Categories, models.EventCategorySCACI)
	assert.Equal(t, testLimit, reader.last.Limit)
	assert.Equal(t, testOffset, reader.last.Offset)
	assert.Equal(t, now.Add(-config.SCACIDashboardDefaultWindow), reader.last.From)
	assert.Equal(t, now, reader.last.To)

	_, _, err = svc.List(adminCtx(), testTenant, BucketDownlink, grpcservices.ScaciWindow{}, testLimit, 0)
	require.NoError(t, err)
	assert.False(t, reader.last.IncludeSCACIFailures)
	assert.Contains(t, reader.last.Severities, models.EventSeverityWarning, "expired downlinks are warnings")
	assert.NotEmpty(t, reader.last.EventTypePrefixes, "the downlink bucket is narrowed by event type")

	_, _, err = svc.List(adminCtx(), testTenant, BucketBaseStation, grpcservices.ScaciWindow{}, testLimit, 0)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{models.EventCategoryBaseStation, models.EventCategoryBSSCI}, reader.last.Categories)

	_, _, err = svc.List(adminCtx(), testTenant, BucketEndpoint, grpcservices.ScaciWindow{}, testLimit, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{models.EventCategoryEndpoint}, reader.last.Categories)
}

func TestList_RejectsUnknownBucketAndInvertedWindow(t *testing.T) {
	svc := New(&fakeReader{}, fixedClock{}, logger.NewNop())
	_, _, err := svc.List(adminCtx(), testTenant, "everything", grpcservices.ScaciWindow{}, testLimit, 0)
	assert.ErrorIs(t, err, ErrInvalidBucket)

	from := now
	to := now.Add(-time.Hour)
	_, _, err = svc.List(adminCtx(), testTenant, BucketEndpoint, grpcservices.ScaciWindow{From: &from, To: &to}, testLimit, 0)
	assert.ErrorIs(t, err, ErrInvalidTimeRange)
}

// TestList_BucketNeedsEveryCategoryReadable: a bucket opens only to roles that
// may read all of its event categories, and a refused bucket never reaches the store.
func TestList_BucketNeedsEveryCategoryReadable(t *testing.T) {
	endpointManager := authz.WithRoles(testutil.TestContext(), authz.Roles{EndpointManager: true})
	baseStationManager := authz.WithRoles(testutil.TestContext(), authz.Roles{BaseStationManager: true})
	cases := []struct {
		name     string
		ctx      context.Context
		bucket   string
		readable bool
	}{
		{name: "control plane holds security and system events", ctx: endpointManager, bucket: BucketControlPlane},
		{name: "endpoint manager reads downlink failures", ctx: endpointManager, bucket: BucketDownlink, readable: true},
		{name: "endpoint manager reads endpoint failures", ctx: endpointManager, bucket: BucketEndpoint, readable: true},
		{name: "endpoint manager cannot read base station failures", ctx: endpointManager, bucket: BucketBaseStation},
		{name: "base station manager reads base station failures", ctx: baseStationManager, bucket: BucketBaseStation, readable: true},
		{name: "no roles", ctx: testutil.TestContext(), bucket: BucketEndpoint},
		{name: "administrator reads the control plane", ctx: adminCtx(), bucket: BucketControlPlane, readable: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{}
			_, _, err := New(reader, fixedClock{}, logger.NewNop()).List(tc.ctx, testTenant, tc.bucket, grpcservices.ScaciWindow{}, testLimit, 0)
			if tc.readable {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrBucketNotReadable)
			assert.Zero(t, reader.last.TenantID, "a refused bucket never reaches the store")
		})
	}
}
