package bssciservices

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	auditOwnerTenant     = "42"
	auditOwnerTenantID   = int64(42)
	auditStationEUI      = uint64(0x70B3D59CD00009E6)
	auditStationLabel    = "tims base (70B3D59CD00009E6)"
	auditStationName     = "tims base"
	auditStationID       = int64(7)
	auditEndpointEUI     = uint64(0x70B3D56770111505)
	auditEndpointText    = "70B3D56770111505"
	auditQueueID         = int64(118)
	auditOpID            = int64(-3120)
	auditPacketCnt       = uint32(74)
	auditTxTime          = int64(1790606895123000000)
	auditTxTimeText      = "2026-09-28T14:48:15.123Z"
	auditPayloadHex      = "33"
	auditCounterPayloads = "32@74, 33@75"
)

var (
	auditTestNow        = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	errAuditQueueDown   = errors.New("queue down")
	errAuditStationGone = errors.New("no such station")
)

// mockEventStore keeps the last event the audit logger wrote.
type mockEventStore struct {
	lastEvent *models.SystemEvent
	err       error
}

func (m *mockEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	m.lastEvent = event
	return m.err
}

// queuedDownlinks answers the lookup with one downlink or an error.
type queuedDownlinks struct {
	downlink *storage.DownlinkMessage
	err      error
}

func (q queuedDownlinks) GetDownlinkByQueueID(context.Context, uint64, string) (*storage.DownlinkMessage, error) {
	return q.downlink, q.err
}

// ownerStations holds the base stations registered to the owner tenant.
type ownerStations map[int64]*models.BaseStation

func (o ownerStations) GetByEUI(_ context.Context, tenantID int64, _ []byte) (*models.BaseStation, error) {
	if station, ok := o[tenantID]; ok {
		return station, nil
	}
	return nil, errAuditStationGone
}

var ownersStation = ownerStations{auditOwnerTenantID: {ID: auditStationID, Name: auditStationName}}

func auditSession() *bssci.Session {
	return &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: auditStationEUI}}
}

func mustAuditLogger(t *testing.T, events SystemEventRecorder, downlinks DownlinkLookup, stations bssci.StationDirectory) *DownlinkAuditLog {
	t.Helper()
	audit, err := NewAuditLogger(AuditLogDeps{
		Events: events, Downlinks: downlinks, Stations: stations, Clock: testutil.NewFakeClock(auditTestNow), Logger: logger.NewNop(),
	})
	require.NoError(t, err)
	return audit
}

func eventDetails(t *testing.T, event *models.SystemEvent) map[string]interface{} {
	t.Helper()
	var details map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(event.Details))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&details))
	return details
}

// A sent downlink names the station, the payload and the packet counter; the
// transmit time is a detail only, since the server cannot render it in the
// viewer's time zone and the row's own time already shows when it happened.
func TestRecordDLResult_SentNamesStationPayloadAndTransmission(t *testing.T) {
	events := &mockEventStore{}
	downlinks := queuedDownlinks{downlink: &storage.DownlinkMessage{Payload: []byte{0x33}}}
	audit := mustAuditLogger(t, events, downlinks, ownersStation)
	txTime, packetCnt := auditTxTime, auditPacketCnt

	require.NoError(t, audit.RecordDLResult(testutil.TestContext(), auditOwnerTenant, auditSession(), &mioty.DLDataResult{
		EpEui: auditEndpointEUI, QueId: uint64(auditQueueID), Result: mioty.ResultSent, TxTime: &txTime, PacketCnt: &packetCnt,
	}))

	event := events.lastEvent
	require.NotNil(t, event)
	assert.Equal(t, models.EventTypeDLDataSent, event.EventType)
	assert.Equal(t, "Downlink sent", event.Title)
	assert.Equal(t, "Downlink 118 for endpoint 70B3D56770111505 sent by base station "+auditStationLabel+
		" after uplink packet counter 74, payload 33", event.Description)
	assert.NotContains(t, event.Description, auditTxTimeText)
	assert.Equal(t, auditEndpointText, event.SourceName)
	require.NotNil(t, event.BasestationID, "the owner's own station shows the event on its timeline")
	assert.Equal(t, auditStationID, *event.BasestationID)
	details := eventDetails(t, event)
	assert.Equal(t, "70B3D59CD00009E6", details[models.EventDetailKeyBsEui])
	assert.Equal(t, auditStationName, details[models.EventDetailKeyBaseStationName])
	assert.Equal(t, auditPayloadHex, details[models.EventDetailKeyUserData])
	assert.Equal(t, json.Number(strconv.FormatUint(uint64(auditPacketCnt), 10)), details[models.EventDetailKeyPacketCnt])
	assert.Equal(t, json.Number(strconv.FormatInt(auditTxTime, 10)), details[models.EventDetailKeyTxTime])
}

// A roaming endpoint's owner learns the relaying station's EUI, never another
// tenant's name for it, and the event stays off that station's timeline.
func TestRecordQueueAck_ForeignStationIsNamedByItsEUIOnly(t *testing.T) {
	events := &mockEventStore{}
	downlinks := queuedDownlinks{downlink: &storage.DownlinkMessage{Payload: []byte{0x33}}}
	audit := mustAuditLogger(t, events, downlinks, ownerStations{})

	require.NoError(t, audit.RecordQueueAck(testutil.TestContext(), auditOwnerTenant, auditSession(), auditEndpointEUI, auditQueueID, auditOpID))

	event := events.lastEvent
	require.NotNil(t, event)
	assert.Equal(t, "Downlink queued at base station", event.Title)
	assert.Equal(t, "Downlink 118 for endpoint 70B3D56770111505 queued at base station 70B3D59CD00009E6, payload 33", event.Description)
	assert.Nil(t, event.BasestationID)
	assert.NotContains(t, eventDetails(t, event), models.EventDetailKeyBaseStationName)
}

// A counter-dependent downlink lists each payload with its packet counter.
func TestRecordDLResult_CounterDependentPayloads(t *testing.T) {
	events := &mockEventStore{}
	downlinks := queuedDownlinks{downlink: &storage.DownlinkMessage{
		CntDepend: true, UserData: [][]byte{{0x32}, {0x33}}, PacketCntArray: []int64{74, 75},
	}}
	audit := mustAuditLogger(t, events, downlinks, ownersStation)

	require.NoError(t, audit.RecordDLResult(testutil.TestContext(), auditOwnerTenant, auditSession(), &mioty.DLDataResult{
		EpEui: auditEndpointEUI, QueId: uint64(auditQueueID), Result: mioty.ResultExpired,
	}))

	assert.Equal(t, "Downlink expired", events.lastEvent.Title)
	assert.Equal(t, models.EventSeverityWarning, events.lastEvent.Severity)
	assert.Contains(t, events.lastEvent.Description, "payload "+auditCounterPayloads)
}

func TestRecordDownlinkAcknowledged_RecordsTheEndpointAcknowledgement(t *testing.T) {
	events := &mockEventStore{}
	downlinks := queuedDownlinks{downlink: &storage.DownlinkMessage{Payload: []byte{0x33}}}
	audit := mustAuditLogger(t, events, downlinks, ownersStation)

	acknowledged := &storage.DownlinkMessage{QueID: auditQueueID, TenantID: auditOwnerTenant, EPEUI: mioty.FormatEUI64(auditEndpointEUI)}
	require.NoError(t, audit.RecordDownlinkAcknowledged(testutil.TestContext(), acknowledged, auditPacketCnt))

	event := events.lastEvent
	require.NotNil(t, event)
	assert.Equal(t, models.EventTypeDLDataAcknowledged, event.EventType)
	assert.Equal(t, "Downlink acknowledged by endpoint", event.Title)
	assert.Equal(t, "Endpoint 70B3D56770111505 acknowledged downlink 118 sent after uplink packet counter 74, payload 33", event.Description)
	assert.Equal(t, auditEndpointText, event.SourceName)
}

func TestRecordDLResult_UnknownResultIsAnError(t *testing.T) {
	events := &mockEventStore{}
	audit := mustAuditLogger(t, events, queuedDownlinks{err: errAuditQueueDown}, ownersStation)

	require.NoError(t, audit.RecordDLResult(testutil.TestContext(), auditOwnerTenant, auditSession(), &mioty.DLDataResult{
		EpEui: auditEndpointEUI, QueId: uint64(auditQueueID), Result: "lost",
	}))

	assert.Equal(t, models.EventTypeDLDataUnknownResult, events.lastEvent.EventType)
	assert.Equal(t, models.EventSeverityError, events.lastEvent.Severity)
	assert.Equal(t, "Base station "+auditStationLabel+" reported result 'lost' for downlink 118 of endpoint 70B3D56770111505, payload unavailable",
		events.lastEvent.Description, "an unreadable queue row still records the result")
}

func TestRecordQueueAck_StoreErrorIsReturned(t *testing.T) {
	events := &mockEventStore{err: errAuditQueueDown}
	audit := mustAuditLogger(t, events, queuedDownlinks{}, ownersStation)

	err := audit.RecordQueueAck(testutil.TestContext(), auditOwnerTenant, auditSession(), auditEndpointEUI, auditQueueID, auditOpID)

	require.ErrorIs(t, err, errAuditQueueDown)
}

func TestNewAuditLogger_RejectsMissingCollaborators(t *testing.T) {
	complete := AuditLogDeps{Events: &mockEventStore{}, Downlinks: queuedDownlinks{}, Stations: ownersStation,
		Clock: testutil.NewFakeClock(auditTestNow), Logger: logger.NewNop()}
	cases := map[error]func(d *AuditLogDeps){
		ErrNilEventStore:     func(d *AuditLogDeps) { d.Events = nil },
		ErrNilAuditDownlinks: func(d *AuditLogDeps) { d.Downlinks = nil },
		ErrNilAuditStations:  func(d *AuditLogDeps) { d.Stations = nil },
		ErrNilClock:          func(d *AuditLogDeps) { d.Clock = nil },
		ErrNilAuditLogger:    func(d *AuditLogDeps) { d.Logger = nil },
	}
	for want, strip := range cases {
		deps := complete
		strip(&deps)
		_, err := NewAuditLogger(deps)
		assert.ErrorIs(t, err, want)
	}
}
