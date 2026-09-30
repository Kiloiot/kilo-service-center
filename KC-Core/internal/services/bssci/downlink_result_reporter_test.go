package bssciservices

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Identities of the result reporter tests.
const (
	reporterStationEUI  uint64 = 0x70B3D59CD00009E6
	reporterEndpointEUI uint64 = 0x70B3D59CD0000341
	reporterQueueID     int64  = 912
	reporterACQueueID   uint64 = 77
	reporterCommandRef         = "order-17"
)

var (
	// reporterOrg is the organization whose Application Center queued the
	// reported downlinks.
	reporterOrg    = uuid.MustParse("5a1e0d4c-2f3b-4c6d-8e9f-0a1b2c3d4e5f")
	reporterQueuer = scaci.ApplicationCenter{TenantID: 3, OrganizationID: reporterOrg, AcEui: 0x70B3D59CD0000A01}
)

type deliveredResult struct {
	queuer  scaci.ApplicationCenter
	acQueID uint64
	result  mioty.DLDataResult
}

// recordingApplicationCenters records every result delivered to Application Centers.
type recordingApplicationCenters struct {
	mu        sync.Mutex
	delivered []deliveredResult
}

func (r *recordingApplicationCenters) BroadcastDLDataResult(_ context.Context, queuer scaci.ApplicationCenter, acQueID uint64, result *mioty.DLDataResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.delivered = append(r.delivered, deliveredResult{queuer: queuer, acQueID: acQueID, result: *result})
	return nil
}

type publishedResult struct {
	org    string
	ref    string
	result mioty.DLDataResult
}

// recordingResultPublisher records every result published on MQTT.
type recordingResultPublisher struct {
	mu        sync.Mutex
	published []publishedResult
}

func (p *recordingResultPublisher) PublishDownlinkResult(_ context.Context, orgUUID, ref string, result *mioty.DLDataResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.published = append(p.published, publishedResult{org: orgUUID, ref: ref, result: *result})
	return nil
}

// reporterFixture is a reporter over recording collaborators; stop waits for
// the background delivery before the recordings are read.
type reporterFixture struct {
	reporter *DownlinkResultReporter
	acs      *recordingApplicationCenters
	mqtt     *recordingResultPublisher
	events   *mockEventStore
	work     *BackgroundWork
}

func newReporterFixture(t *testing.T) *reporterFixture {
	t.Helper()
	f := &reporterFixture{
		acs:    &recordingApplicationCenters{},
		mqtt:   &recordingResultPublisher{},
		events: &mockEventStore{},
		work:   NewBackgroundWork(),
	}
	reporter, err := NewDownlinkResultReporter(f.acs, f.mqtt,
		mustAuditLogger(t, f.events, queuedDownlinks{}, ownersStation), f.work, logger.NewNop())
	require.NoError(t, err)
	f.reporter = reporter
	return f
}

func (f *reporterFixture) stop(t *testing.T) {
	t.Helper()
	require.NoError(t, f.work.Stop(testutil.TestContext()))
}

// reportedRow is the organization's downlink; one with an Application Center
// queue id names the reporter's queuer as the Application Center that queued it.
func reportedRow(org uuid.UUID, acQueID *uint64) *storage.DownlinkMessage {
	row := &storage.DownlinkMessage{
		QueID: reporterQueueID, ACQueID: acQueID, EPEUI: mioty.FormatEUI64(reporterEndpointEUI),
		TenantID: "3", OrganizationID: &org, Ref: reporterCommandRef,
	}
	if acQueID != nil {
		acEUI := reporterQueuer.AcEui
		row.ACEUI = &acEUI
	}
	return row
}

func reporterStation() *bssci.Session {
	return &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: reporterStationEUI}}
}

// TestReportStationResult_ReachesEveryOriginator: a result a base station
// gave reaches the Application Center under the queue id it assigned (SCACI
// §3.12.1), the MQTT topic of the organization that queued the downlink, and
// the owner tenant's events; only a sent result names the station.
func TestReportStationResult_ReachesEveryOriginator(t *testing.T) {
	org := reporterOrg
	acQueID := reporterACQueueID
	txTime := int64(1700000000000000000)
	packetCnt := uint32(12)
	for name, tc := range map[string]struct {
		result    mioty.DLDataResult
		eventType string
		names     bool
	}{
		"sent":    {mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultSent, TxTime: &txTime, PacketCnt: &packetCnt}, models.EventTypeDLDataSent, true},
		"invalid": {mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultInvalid}, models.EventTypeDLDataInvalid, false},
	} {
		t.Run(name, func(t *testing.T) {
			f := newReporterFixture(t)

			f.reporter.ReportStationResult(testutil.TestContext(), reportedRow(org, &acQueID), tc.result, reporterStation())
			f.stop(t)

			want := tc.result
			if tc.names {
				station := reporterStationEUI
				want.BsEui = &station
			}
			assert.Equal(t, []deliveredResult{{queuer: reporterQueuer, acQueID: acQueID, result: want}}, f.acs.delivered)
			assert.Equal(t, []publishedResult{{org: org.String(), ref: reporterCommandRef, result: want}}, f.mqtt.published)
			require.NotNil(t, f.events.lastEvent)
			assert.Equal(t, "3", f.events.lastEvent.TenantID)
			assert.Equal(t, tc.eventType, f.events.lastEvent.EventType)
		})
	}
}

// TestReportStationResult_ReachesNoApplicationCenterThatDidNotQueue pins
// SCACI §3.12.1: dlDataRes names the queue id the queuing Application Center
// assigned, so a downlink queued through gRPC or MQTT, which carries none, and
// one that names no Application Center that queued it are reported to no
// Application Center; MQTT still carries them.
func TestReportStationResult_ReachesNoApplicationCenterThatDidNotQueue(t *testing.T) {
	acQueID := reporterACQueueID
	unknownQueuer := reportedRow(reporterOrg, &acQueID)
	unknownQueuer.ACEUI = nil
	for name, row := range map[string]*storage.DownlinkMessage{
		"no application center queue id": reportedRow(reporterOrg, nil),
		"no queuing application center":  unknownQueuer,
	} {
		t.Run(name, func(t *testing.T) {
			f := newReporterFixture(t)

			f.reporter.ReportStationResult(testutil.TestContext(), row,
				mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultExpired}, reporterStation())
			f.stop(t)

			assert.Empty(t, f.acs.delivered)
			assert.Len(t, f.mqtt.published, 1)
		})
	}
}

// TestReportExpiredInQueue_ReachesEveryOriginator: a downlink the service
// center expired is reported expired under its service center queue id on
// MQTT and under the Application Center's id to the Application Center.
func TestReportExpiredInQueue_ReachesEveryOriginator(t *testing.T) {
	f := newReporterFixture(t)
	org := reporterOrg
	acQueID := reporterACQueueID

	require.NoError(t, f.reporter.ReportExpiredInQueue(testutil.TestContext(), reportedRow(org, &acQueID)))
	f.stop(t)

	expired := mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultExpired}
	assert.Equal(t, []deliveredResult{{queuer: reporterQueuer, acQueID: acQueID, result: expired}}, f.acs.delivered)
	assert.Equal(t, []publishedResult{{org: org.String(), ref: reporterCommandRef, result: expired}}, f.mqtt.published)
	require.NotNil(t, f.events.lastEvent)
	assert.Equal(t, models.EventTypeDLDataExpired, f.events.lastEvent.EventType)
	assert.Equal(t, "3", f.events.lastEvent.TenantID)
}

// TestReportExpiredInQueue_RefusesAnUnidentifiedRow: a row whose endpoint
// cannot be parsed is reported to nobody.
func TestReportExpiredInQueue_RefusesAnUnidentifiedRow(t *testing.T) {
	f := newReporterFixture(t)
	row := reportedRow(uuid.New(), nil)
	row.EPEUI = "not-an-eui"

	require.ErrorIs(t, f.reporter.ReportExpiredInQueue(testutil.TestContext(), row), errUnidentifiedDownlink)
	f.stop(t)

	assert.Empty(t, f.acs.delivered)
	assert.Empty(t, f.mqtt.published)
}

// TestReportStationResult_WithoutRefReportsNone: a downlink queued without an
// MQTT command ref (SCACI, gRPC) reports every result without one.
func TestReportStationResult_WithoutRefReportsNone(t *testing.T) {
	f := newReporterFixture(t)
	row := reportedRow(reporterOrg, nil)
	row.Ref = ""

	f.reporter.ReportStationResult(testutil.TestContext(), row,
		mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultInvalid}, reporterStation())
	f.stop(t)

	require.Len(t, f.mqtt.published, 1)
	assert.Empty(t, f.mqtt.published[0].ref)
}

// blockingApplicationCenters holds every delivered result until released.
type blockingApplicationCenters struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingApplicationCenters) BroadcastDLDataResult(context.Context, scaci.ApplicationCenter, uint64, *mioty.DLDataResult) error {
	close(b.started)
	<-b.release
	return nil
}

// shutdownSettleWindow is how long a Stop that should block is given to return early.
const shutdownSettleWindow = 50 * time.Millisecond

// TestDownlinkResultReporter_StopWaitsForTheDelivery: the composition root
// stops the runner at shutdown, so a result still being delivered to an
// Application Center finishes before the process exits.
func TestDownlinkResultReporter_StopWaitsForTheDelivery(t *testing.T) {
	acs := &blockingApplicationCenters{started: make(chan struct{}), release: make(chan struct{})}
	work := NewBackgroundWork()
	reporter, err := NewDownlinkResultReporter(acs, DownlinkResultsWithoutMQTT{},
		mustAuditLogger(t, &mockEventStore{}, queuedDownlinks{}, ownersStation), work, logger.NewNop())
	require.NoError(t, err)
	acQueID := reporterACQueueID

	reporter.ReportStationResult(testutil.TestContext(), reportedRow(reporterOrg, &acQueID),
		mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultInvalid}, reporterStation())
	<-acs.started

	stopped := make(chan struct{})
	go func() {
		assert.NoError(t, work.Stop(testutil.TestContext()))
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while a result was still being delivered")
	case <-time.After(shutdownSettleWindow):
	}

	close(acs.release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after the delivery finished")
	}
}

func TestNewDownlinkResultReporter_RejectsMissingCollaborators(t *testing.T) {
	acs := &recordingApplicationCenters{}
	mqtt := &recordingResultPublisher{}
	events := mustAuditLogger(t, &mockEventStore{}, queuedDownlinks{}, ownersStation)
	work := NewBackgroundWork()
	log := logger.NewNop()
	cases := map[string]struct {
		acs    ApplicationCenterResults
		mqtt   DownlinkResultPublisher
		events DownlinkResultEvents
		work   BackgroundRunner
		log    logger.Logger
		want   error
	}{
		"nil application centers": {mqtt: mqtt, events: events, work: work, log: log, want: ErrNilApplicationCenterResults},
		"nil mqtt publisher":      {acs: acs, events: events, work: work, log: log, want: ErrNilDownlinkResultPublisher},
		"nil event recorder":      {acs: acs, mqtt: mqtt, work: work, log: log, want: ErrNilDownlinkResultEvents},
		"nil runner":              {acs: acs, mqtt: mqtt, events: events, log: log, want: ErrNilReporterRunner},
		"nil logger":              {acs: acs, mqtt: mqtt, events: events, work: work, want: ErrNilReporterLogger},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			reporter, err := NewDownlinkResultReporter(tc.acs, tc.mqtt, tc.events, tc.work, tc.log)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, reporter)
		})
	}
}
