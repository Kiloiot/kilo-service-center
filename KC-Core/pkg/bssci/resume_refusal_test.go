package bssci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Counters of the refused-resume scenario observed on a real station: the
// service center remembers session 3 at bsOpId 8806 / scOpId -1791 while the
// station, fresh from another service center, reports 6 / -4.
const (
	refusedResumeDbSessionID   = int64(3)
	refusedResumeTenantID      = int64(1)
	refusedResumeStationID     = int64(1)
	refusedResumePersistedBsOp = int64(8806)
	refusedResumePersistedScOp = int64(-1791)
	refusedResumeReportedBsOp  = int64(6)
	refusedResumeReportedScOp  = int64(-4)
	refusedResumeErrorCode     = 1
	refusedResumeErrorMessage  = "session resume with mismatching uuid not permitted"
)

var errTerminateUnavailable = errors.New("session store unavailable")

// failingTerminateSessionSvc cannot retire sessions.
type failingTerminateSessionSvc struct {
	SessionService
}

func (failingTerminateSessionSvc) TerminateSession(context.Context, *Session) error {
	return errTerminateUnavailable
}

// redial opens another connection from the same base station to the
// harness's service center.
func (h *interopHarness) redial() *interopHarness {
	h.t.Helper()
	client, srvSide := net.Pipe()
	done := make(chan struct{})
	h.server.wg.Add(1)
	go func() {
		defer close(done)
		h.server.handleConnection(srvSide)
	}()
	h.t.Cleanup(func() {
		_ = client.Close()
		<-done
	})
	return &interopHarness{t: h.t, conn: client, encoding: h.encoding, server: h.server, done: done}
}

// startRefusalServer starts a harness whose service center remembers the
// session the station's snBsUuid names, with one pending operation, as the
// database holds it after the station's last connection was lost, and
// records the events it raises. adjust, when set, runs after the seeding.
func startRefusalServer(t *testing.T, encoding string, adjust func(*Server)) (*interopHarness, *recordingEventStore) {
	t.Helper()
	events := &recordingEventStore{}
	h := startInteropServerWithSetup(t, encoding, func(server *Server) {
		server.events = newProtocolEventRecorder(events, server.clock, server.logger)
		server.sessionSvc.(*mockSessionService).StoreSessionByUUID(&Session{ProtocolSessionState: ProtocolSessionState{
			ID:               "previous-connection",
			BaseStationEUI:   TestBsEui01,
			SessionUUID:      refusedResumeSnBsUUID(),
			DbSessionID:      refusedResumeDbSessionID,
			ResolvedTenantID: refusedResumeTenantID,
			LastBsOpId:       refusedResumePersistedBsOp,
			LastScOpId:       refusedResumePersistedScOp,
		}})
		server.statusSvc.(*memoryStatusService).persistRows(refusedResumeDbSessionID, PersistedOperation{
			OperationID:   refusedResumePersistedScOp,
			OperationType: mioty.CmdStatus,
			OperationData: []byte(fmt.Sprintf(`{"command":%q,"opId":%d}`, mioty.CmdStatus, refusedResumePersistedScOp)),
		})
		if adjust != nil {
			adjust(server)
		}
	})
	return h, events
}

// assertSessionStillResumable checks the remembered session was neither
// retired nor stripped of its pending operations.
func assertSessionStillResumable(t *testing.T, server *Server) {
	t.Helper()
	terminated, disconnected := server.sessionSvc.(*mockSessionService).retiredRows()
	assert.Empty(t, terminated, "the remembered session is not retired")
	assert.Empty(t, disconnected, "a connection that never claimed the row does not touch it")
	statusSvc := server.statusSvc.(*memoryStatusService)
	assert.Empty(t, statusSvc.retiredPendingOperations())
	assert.Len(t, statusSvc.persistedRowsOf(refusedResumeDbSessionID), 1, "the pending operation survives")
}

// refusedResumeSnBsUUID matches the snBsUuid connectPayload sends.
func refusedResumeSnBsUUID() []byte {
	snBsUUID := make([]byte, len(mioty.SessionUUID{}))
	for i := range snBsUUID {
		snBsUUID[i] = byte(i + 1)
	}
	return snBsUUID
}

func refusedResumeConnect() map[string]interface{} {
	con := connectPayload(mioty.MIOTYProtocolVersion, TestBsEui01)
	con["snBsOpId"] = refusedResumeReportedBsOp
	con["snScOpId"] = refusedResumeReportedScOp
	return con
}

func conRspResumes(t *testing.T, conRsp map[string]interface{}) bool {
	t.Helper()
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp))
	resumed, ok := conRsp["snResume"].(bool)
	require.True(t, ok, "conRsp carries snResume")
	return resumed
}

// refuseConnectResponse answers the conRsp with the station's refusal and
// waits until the service center acknowledged it and closed the connection.
func (h *interopHarness) refuseConnectResponse() {
	h.t.Helper()
	h.writeFrame(map[string]interface{}{
		"command": mioty.CmdError, "opId": int64(0),
		"code": int64(refusedResumeErrorCode), "message": refusedResumeErrorMessage,
	})
	assert.Equal(h.t, mioty.CmdErrorAck, frameCommand(h.readFrame()))
	h.expectClosed()
	<-h.done
}

func refusedResumeEvents(events *recordingEventStore) []*models.SystemEvent {
	events.mu.Lock()
	defer events.mu.Unlock()
	var refused []*models.SystemEvent
	for _, event := range events.created {
		if event.EventType == models.EventTypeSessionResumeRefused {
			refused = append(refused, event)
		}
	}
	return refused
}

// A station that refuses a resuming conRsp holds a session the service center
// cannot know (BSSCI §1: without agreement "a new session is started"), so
// its next connect must start a new session instead of offering the refused
// resume again forever.
func TestStationRefusedResumeStartsNewSessionOnReconnect(t *testing.T) {
	for _, encoding := range []string{EncodingJSON, EncodingMessagePack} {
		t.Run(encoding, func(t *testing.T) {
			h, events := startRefusalServer(t, encoding, nil)

			h.writeFrame(refusedResumeConnect())
			require.True(t, conRspResumes(t, h.readFrame()), "the remembered session is offered first")
			h.refuseConnectResponse()

			again := h.redial()
			again.writeFrame(refusedResumeConnect())
			assert.False(t, conRspResumes(t, again.readFrame()), "the reconnect starts a new session")

			terminated, _ := h.server.sessionSvc.(*mockSessionService).retiredRows()
			assert.Equal(t, []int64{refusedResumeDbSessionID}, terminated, "the refused session is no longer resumable")
			statusSvc := h.server.statusSvc.(*memoryStatusService)
			assert.Equal(t, []int64{refusedResumeDbSessionID}, statusSvc.retiredPendingOperations(),
				"the refused session's pending operations are deleted")
			assert.Empty(t, statusSvc.persistedRowsOf(refusedResumeDbSessionID))
			assertRefusedResumeEvent(t, refusedResumeEvents(events))
		})
	}
}

func assertRefusedResumeEvent(t *testing.T, refused []*models.SystemEvent) {
	t.Helper()
	require.Len(t, refused, 1, "the refused resume is recorded once")
	event := refused[0]
	assert.Equal(t, "1", event.TenantID, "the event belongs to the station's tenant")
	assert.Equal(t, models.EventCategoryBSSCI, event.Category)
	assert.Equal(t, mioty.SourceTypeBaseStation, event.SourceType)
	assert.Equal(t, mioty.FormatEUI64(TestBsEui01), event.SourceName)
	require.NotNil(t, event.BasestationID)
	assert.Equal(t, refusedResumeStationID, *event.BasestationID)

	var details map[string]interface{}
	require.NoError(t, json.Unmarshal(event.Details, &details))
	assert.Equal(t, mioty.FormatEUI64(TestBsEui01), details[models.EventDetailKeyBsEui])
	assert.EqualValues(t, refusedResumeErrorCode, details[models.EventDetailKeyFailureCode])
	assert.Equal(t, refusedResumeErrorMessage, details[models.EventDetailKeyMessage])
}

// A refused new session has no resume to retire: nothing is terminated and
// no refused-resume event is raised.
func TestStationRefusedNewSessionRetiresNothing(t *testing.T) {
	h, events := startRefusalServer(t, EncodingJSON, nil)

	unknownSnBsUUID := make([]interface{}, len(mioty.SessionUUID{}))
	for i := range unknownSnBsUUID {
		unknownSnBsUUID[i] = int64(0xFF - i)
	}
	con := connectPayload(mioty.MIOTYProtocolVersion, TestBsEui01)
	con["snBsUuid"] = unknownSnBsUUID
	h.writeFrame(con)
	require.False(t, conRspResumes(t, h.readFrame()), "an unknown snBsUuid starts a new session")
	h.refuseConnectResponse()

	assertSessionStillResumable(t, h.server)
	assert.Empty(t, refusedResumeEvents(events))
}

// A resume the station never answers - its connection is lost after the
// conRsp, or conCmp does not arrive in time - is not a refusal: the session
// stays resumable with its pending operations and is offered again.
func TestUnansweredResumeKeepsSessionResumable(t *testing.T) {
	for name, abandon := range map[string]func(*interopHarness){
		"connection lost": func(h *interopHarness) {
			require.NoError(t, h.conn.Close())
			<-h.done
		},
		"conCmp timeout": func(h *interopHarness) {
			h.expectClosed()
			<-h.done
		},
	} {
		t.Run(name, func(t *testing.T) {
			h, events := startRefusalServer(t, EncodingJSON, func(server *Server) {
				server.config.OperationAckTimeout = interopShortTimeout
			})

			h.writeFrame(refusedResumeConnect())
			require.True(t, conRspResumes(t, h.readFrame()))
			abandon(h)

			assertSessionStillResumable(t, h.server)
			assert.Empty(t, refusedResumeEvents(events))

			again := h.redial()
			again.writeFrame(refusedResumeConnect())
			assert.True(t, conRspResumes(t, again.readFrame()), "the reconnect is offered the same session again")
		})
	}
}

// When the refused session cannot be retired it stays resumable, so the
// refusal is not reported as settled.
func TestStationRefusedResumeRetirementFailureRaisesNoEvent(t *testing.T) {
	h, events := startRefusalServer(t, EncodingJSON, func(server *Server) {
		server.sessionSvc = failingTerminateSessionSvc{SessionService: server.sessionSvc}
	})

	h.writeFrame(refusedResumeConnect())
	require.True(t, conRspResumes(t, h.readFrame()))
	h.refuseConnectResponse()

	assert.Empty(t, refusedResumeEvents(events))
}
