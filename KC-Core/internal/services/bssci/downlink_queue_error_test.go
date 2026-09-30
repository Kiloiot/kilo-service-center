package bssciservices

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

type failedDownlink struct {
	queID    int64
	tenantID int64
	station  uint64
	reason   string
}

// failureRecorder records every downlink the service fails and returns the
// row of a downlink the reporter's queuer queued in one organization.
type failureRecorder struct {
	mockMIOTYDownlinksForDispatch
	org     uuid.UUID
	acQueID uint64
	failed  []failedDownlink
}

func (r *failureRecorder) FailQueuedDownlink(_ context.Context, queID int64, tenantID int64, station uint64, reason string) (*storage.DownlinkMessage, error) {
	r.failed = append(r.failed, failedDownlink{queID: queID, tenantID: tenantID, station: station, reason: reason})
	acEUI := reporterQueuer.AcEui
	return &storage.DownlinkMessage{QueID: queID, ACQueID: &r.acQueID, ACEUI: &acEUI, EPEUI: mioty.FormatEUI64(reporterEndpointEUI),
		TenantID: strconv.FormatInt(tenantID, 10), OrganizationID: &r.org}, nil
}

// TestProcessQueueError_FailsTheDownlinkAndReportsItDiscarded pins the
// outcome of a dlDataQue the base station answered with error (BSSCI §3.17):
// the downlink fails with the station's error, and its originators learn it
// was discarded as invalid (SCACI §3.12): the Application Center that queued
// it, the MQTT downlink_result topic of its organization, and the events.
func TestProcessQueueError_FailsTheDownlinkAndReportsItDiscarded(t *testing.T) {
	writer := &failureRecorder{org: reporterOrg, acQueID: reporterACQueueID}
	f := newReporterFixture(t)
	resolver := NewTenantResolver(nil)
	resolver.RegisterQueueTenant(reporterQueueID, "3")
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: resolver, Outcomes: writer, Holders: &mockMIOTYDownlinksForDispatch{},
		Results: f.reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)

	err = svc.ProcessQueueError(testutil.TestContext(), reporterStation(),
		bssci.QueueRejection{QueueID: reporterQueueID, EndpointEUI: reporterEndpointEUI, OwnerTenant: "3", Code: 28, Message: "queue full"})
	require.NoError(t, err)
	f.stop(t)

	invalid := mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultInvalid}
	assert.Equal(t, []failedDownlink{{queID: reporterQueueID, tenantID: 3, station: reporterStationEUI, reason: "base station error 28: queue full"}}, writer.failed)
	assert.Equal(t, []deliveredResult{{queuer: reporterQueuer, acQueID: writer.acQueID, result: invalid}}, f.acs.delivered)
	assert.Equal(t, []publishedResult{{org: writer.org.String(), result: invalid}}, f.mqtt.published,
		"a station error on a dlDataQue produces an MQTT downlink_result")
	require.NotNil(t, f.events.lastEvent)
	assert.Equal(t, "3", f.events.lastEvent.TenantID)
	assert.Equal(t, models.EventTypeDLDataInvalid, f.events.lastEvent.EventType)
	_, stillMapped := resolver.ResolveTenant(testutil.TestContext(), reporterQueueID)
	assert.Error(t, stillMapped, "the queue-to-tenant mapping is released")
}

// TestProcessQueueError_RejectsAnInvalidQueueID: a rejection without a queue
// id names no downlink, so nothing is failed.
func TestProcessQueueError_RejectsAnInvalidQueueID(t *testing.T) {
	writer := &failureRecorder{}
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: NewTenantResolver(nil), Outcomes: writer, Holders: &mockMIOTYDownlinksForDispatch{},
		Results: newReporterFixture(t).reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)

	err = svc.ProcessQueueError(testutil.TestContext(), &bssci.Session{},
		bssci.QueueRejection{QueueID: 0, OwnerTenant: "3", Code: 28, Message: "queue full"})

	require.Error(t, err)
	assert.Empty(t, writer.failed)
}
