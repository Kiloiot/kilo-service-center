package bssciservices

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// failingResultWriter fails every downlink result update with one error.
type failingResultWriter struct {
	mockMIOTYDownlinksForDispatch
	err error
}

func (w *failingResultWriter) UpdateDownlinkResult(context.Context, int64, uint64, *mioty.DLDataResult) (*storage.DownlinkMessage, error) {
	return nil, w.err
}

// TestProcessDLDataResult_ReportsCatalogErrors pins the typed classification
// of every dlDataRes failure the base station is answered with.
func TestProcessDLDataResult_ReportsCatalogErrors(t *testing.T) {
	cases := []struct {
		name     string
		queueID  uint64
		tenant   string
		writeErr error
		want     *bssci.CatalogError
	}{
		{"queue id out of range", math.MaxInt64 + 1, "3", nil, bssci.NewCatalogError(bssci.ErrQueueIDOutOfRange, bssci.POSIX_ERANGE)},
		{"tenant unresolved", resultLogQueueID, "", nil, bssci.NewCatalogError(bssci.ErrCannotResolveTenantForQueue, bssci.POSIX_EPROTO)},
		{"invalid tenant id", resultLogQueueID, "tenant-3", nil, bssci.NewCatalogError(bssci.ErrInvalidTenantIDFormat, bssci.POSIX_EINVAL)},
		{"unknown queue id", resultLogQueueID, "3", fmt.Errorf("update result: %w", storage.ErrDownlinkNotFound), bssci.NewCatalogError(bssci.ErrQueueIDNotFound, bssci.POSIX_EPROTO)},
		{"database failure", resultLogQueueID, "3", assert.AnError, bssci.NewCatalogError(bssci.ErrDatabaseUpdateFailed, bssci.POSIX_EIO)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := NewTenantResolver(nil)
			if tc.tenant != "" {
				resolver.RegisterQueueTenant(int64(resultLogQueueID), tc.tenant)
			}
			svc, err := NewDownlinkService(DownlinkServiceDeps{
				Logger: logger.NewNop(), Tenants: resolver, Outcomes: &failingResultWriter{err: tc.writeErr}, Holders: &mockMIOTYDownlinksForDispatch{},
				Results: newReporterFixture(t).reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
			})
			require.NoError(t, err)

			_, err = svc.ProcessDLDataResult(testutil.TestContext(),
				&bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: 0x70b3d59cd00009e6}},
				&mioty.DLDataResult{EpEui: 0x70b3d59cd0000341, QueId: tc.queueID, Result: mioty.ResultExpired})

			var got *bssci.CatalogError
			require.True(t, errors.As(err, &got), "want a catalog error, got %v", err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestProcessDLDataResult_RefusedResultReportsNothing pins that a result the
// queue refuses, from a station that does not hold the downlink, reaches no
// originator and leaves the downlink's owner resolvable for its holder.
func TestProcessDLDataResult_RefusedResultReportsNothing(t *testing.T) {
	resolver := NewTenantResolver(nil)
	resolver.RegisterQueueTenant(int64(resultLogQueueID), "3")
	reports := newReporterFixture(t)
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: resolver, Outcomes: &failingResultWriter{err: storage.ErrDownlinkNotFound}, Holders: &mockMIOTYDownlinksForDispatch{},
		Results: reports.reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)

	_, err = svc.ProcessDLDataResult(testutil.TestContext(),
		&bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: 0x70b3d59cd00009e7}},
		&mioty.DLDataResult{EpEui: 0x70b3d59cd0000341, QueId: resultLogQueueID, Result: mioty.ResultSent})
	var got *bssci.CatalogError
	require.True(t, errors.As(err, &got), "want a catalog error, got %v", err)
	assert.Equal(t, bssci.NewCatalogError(bssci.ErrQueueIDNotFound, bssci.POSIX_EPROTO), got)

	reports.stop(t)
	assert.Empty(t, reports.acs.delivered, "no Application Center result")
	assert.Empty(t, reports.mqtt.published, "no MQTT result")
	assert.Nil(t, reports.events.lastEvent, "no result event")
	owner, err := resolver.ResolveTenant(testutil.TestContext(), int64(resultLogQueueID))
	require.NoError(t, err)
	assert.Equal(t, "3", owner)
}
