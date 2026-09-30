package bssciservices

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// TestReportStationResult_NamesTheQueuerOfTheDownlinksOrganization: the
// result goes to the Application Center of the downlink's own tenant and
// organization, never to one of the same acEui elsewhere (SCACI §1).
func TestReportStationResult_NamesTheQueuerOfTheDownlinksOrganization(t *testing.T) {
	f := newReporterFixture(t)
	other := uuid.New()
	acQueID := reporterACQueueID

	f.reporter.ReportStationResult(testutil.TestContext(), reportedRow(other, &acQueID),
		mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultExpired}, reporterStation())
	f.stop(t)

	require.Len(t, f.acs.delivered, 1)
	assert.Equal(t, scaci.ApplicationCenter{TenantID: 3, OrganizationID: other, AcEui: reporterQueuer.AcEui}, f.acs.delivered[0].queuer)
}

// TestReportStationResult_RoutesWithoutTheOperationLog: the queue row names
// the Application Center that queued it, so a result reaches that
// Application Center under its queue id when the dlDataQue record was never
// written (SCACI §3.12).
func TestReportStationResult_RoutesWithoutTheOperationLog(t *testing.T) {
	acs := &recordingApplicationCenters{}
	work := NewBackgroundWork()
	reporter, err := NewDownlinkResultReporter(acs, DownlinkResultsWithoutMQTT{},
		mustAuditLogger(t, &mockEventStore{}, queuedDownlinks{}, ownersStation), work, logger.NewNop())
	require.NoError(t, err)
	acQueID := reporterACQueueID
	row := reportedRow(reporterOrg, &acQueID)
	invalid := mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultInvalid}

	reporter.ReportStationResult(testutil.TestContext(), row, invalid, reporterStation())
	require.NoError(t, work.Stop(testutil.TestContext()))

	assert.Equal(t, []deliveredResult{{queuer: reporterQueuer, acQueID: acQueID, result: invalid}}, acs.delivered)
}
