package mqtt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	cmdTestPrefix   = "mioty"
	cmdTestEpEUI    = uint64(0x70b3d59cd00009e6)
	cmdTestEpEUIHex = "70b3d59cd00009e6"
	cmdTestTenantID = int64(100)
	cmdTestQueID    = uint64(9007199254740991)
)

var errCmdTestCore = errors.New("core unavailable")

// mockDownlinkEnqueuer records EnqueueFromMQTT calls and the ref lookups,
// answering a lookup from queuedRefs.
type mockDownlinkEnqueuer struct {
	mu         sync.Mutex
	calls      []enqueueCall
	returnID   uint64
	returnErr  error
	queuedRefs map[string]bool
	refErr     error
	refLookups []storage.DownlinkCommandRef
}

func (m *mockDownlinkEnqueuer) CommandQueued(_ context.Context, command storage.DownlinkCommandRef) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refLookups = append(m.refLookups, command)
	return m.queuedRefs[command.Ref], m.refErr
}

type enqueueCall struct {
	Ctx      context.Context
	TenantID int64
	OrgID    *uuid.UUID
	Request  *mioty.DLDataQueue
	Command  storage.DownlinkCommand
}

func (m *mockDownlinkEnqueuer) EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, command storage.DownlinkCommand) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, enqueueCall{Ctx: ctx, TenantID: tenantID, OrgID: orgID, Request: req, Command: command})
	return m.returnID, m.returnErr
}

func (m *mockDownlinkEnqueuer) lastCall() enqueueCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

func (m *mockDownlinkEnqueuer) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// mockTenantLookup resolves org UUIDs to tenant IDs.
type mockTenantLookup struct {
	mapping   map[uuid.UUID]int64
	returnErr error
}

func (m *mockTenantLookup) LookupTenant(_ context.Context, orgUUID uuid.UUID) (int64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	if tid, ok := m.mapping[orgUUID]; ok {
		return tid, nil
	}
	return 0, nil
}

type publishedMessage struct {
	Topic   string
	QoS     byte
	Payload []byte
}

// mockCommandPublisher is a Publisher that records publishes and replays subscriptions.
type mockCommandPublisher struct {
	mu          sync.Mutex
	subscribers map[string]MessageHandler
	subCtx      context.Context // captured from Subscribe call
	published   []publishedMessage
}

func newMockCommandPublisher() *mockCommandPublisher {
	return &mockCommandPublisher{subscribers: make(map[string]MessageHandler)}
}

func (m *mockCommandPublisher) hasSubscribers() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.subscribers) > 0
}

func (m *mockCommandPublisher) Connect(_ context.Context) error { return nil }
func (m *mockCommandPublisher) Disconnect(_ context.Context)    {}
func (m *mockCommandPublisher) IsConnected() bool               { return true }
func (m *mockCommandPublisher) Publish(_ context.Context, topic string, qos byte, _ bool, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, _ := payload.([]byte)
	m.published = append(m.published, publishedMessage{Topic: topic, QoS: qos, Payload: body})
	return nil
}
func (m *mockCommandPublisher) Unsubscribe(_ context.Context, _ ...string) error { return nil }

func (m *mockCommandPublisher) Subscribe(ctx context.Context, topic string, _ byte, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[topic] = handler
	m.subCtx = ctx
	return nil
}

func (m *mockCommandPublisher) messages() []publishedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]publishedMessage(nil), m.published...)
}

// simulateMessage simulates a message arriving on the given topic.
// Uses the subscribe-time context (matching production client.go behavior).
func (m *mockCommandPublisher) simulateMessage(topic string, payload []byte) {
	m.mu.Lock()
	var handler MessageHandler
	ctx := m.subCtx
	if ctx == nil {
		ctx = testutil.TestContext()
	}
	for subTopic, h := range m.subscribers {
		if topicMatchesWildcard(subTopic, topic) {
			handler = h
			break
		}
	}
	m.mu.Unlock()

	if handler != nil {
		handler(ctx, topic, payload)
	}
}

// topicMatchesWildcard checks if a topic matches a wildcard pattern (simple implementation).
func topicMatchesWildcard(pattern, topic string) bool {
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")

	if len(patternParts) != len(topicParts) {
		return false
	}

	for i, p := range patternParts {
		if p == "+" {
			continue
		}
		if p != topicParts[i] {
			return false
		}
	}
	return true
}

// commandFixture drives handleMessage directly with recording collaborators.
type commandFixture struct {
	org      uuid.UUID
	pub      *mockCommandPublisher
	enqueuer *mockDownlinkEnqueuer
	lookup   *mockTenantLookup
	handler  *CommandHandler
}

func newCommandFixture() *commandFixture {
	org := uuid.New()
	f := &commandFixture{
		org:      org,
		pub:      newMockCommandPublisher(),
		enqueuer: &mockDownlinkEnqueuer{returnID: cmdTestQueID},
		lookup:   &mockTenantLookup{mapping: map[uuid.UUID]int64{org: cmdTestTenantID}},
	}
	f.handler = NewCommandHandler(f.pub, f.enqueuer, f.lookup, &MockLogger{}, cmdTestPrefix)
	return f
}

func (f *commandFixture) commandTopic() string {
	return fmt.Sprintf(TopicDeviceCommandDownFormat, cmdTestPrefix, f.org, cmdTestEpEUIHex)
}

func (f *commandFixture) send(payload string) {
	f.handler.handleMessage(testutil.TestContext(), f.commandTopic(), []byte(payload))
}

// outcome returns the single event the command produced and its decoded body.
func (f *commandFixture) outcome(t *testing.T, eventType string) map[string]interface{} {
	t.Helper()
	messages := f.pub.messages()
	require.Len(t, messages, 1, "every command produces exactly one outcome event")
	assert.Equal(t, DeviceEventTopic(cmdTestPrefix, f.org.String(), cmdTestEpEUIHex, eventType), messages[0].Topic,
		"the outcome goes to the queuing organization's topic")
	assert.Equal(t, byte(DownlinkEventsQoS), messages[0].QoS)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(messages[0].Payload, &body))
	return body
}

func (f *commandFixture) rejected(t *testing.T, code string) map[string]interface{} {
	t.Helper()
	body := f.outcome(t, DeviceEventDownlinkRejected)
	assert.Equal(t, code, body["code"])
	assert.NotEmpty(t, body["message"])
	assert.Equal(t, cmdTestEpEUIHex, body["epEui"])
	assert.Zero(t, f.enqueuer.callCount(), "a refused command is never queued")
	return body
}

func b64(payload []byte) string {
	return base64.StdEncoding.EncodeToString(payload)
}

func TestCommandHandler_ValidTopicParsesCorrectly(t *testing.T) {
	t.Parallel()

	f := newCommandFixture()
	ctx, cancel := context.WithCancel(testutil.TestContext())
	defer cancel()
	go f.handler.Start(ctx)
	require.Eventually(t, func() bool { return f.pub.hasSubscribers() }, testTimeout, testPollInterval)

	f.pub.simulateMessage(f.commandTopic(), []byte(`{"data":"`+b64([]byte("hello world"))+`","confirmed":true}`))
	require.Eventually(t, func() bool { return f.enqueuer.callCount() > 0 }, testTimeout, testPollInterval)

	call := f.enqueuer.lastCall()
	assert.Equal(t, cmdTestTenantID, call.TenantID)
	require.NotNil(t, call.OrgID)
	assert.Equal(t, f.org, *call.OrgID)
	assert.Equal(t, cmdTestEpEUI, call.Request.EpEui)
	assert.Equal(t, mioty.DownlinkUserData{[]byte("hello world")}, call.Request.UserData)
	require.NotNil(t, call.Request.ResponseExp)
	assert.True(t, *call.Request.ResponseExp)
}

// TestCommandHandler_LegacyCommandBehavesAsBefore pins the original contract:
// {"data","confirmed"} queues exactly the downlink it always queued, and the
// only difference is the new downlink_queued event.
func TestCommandHandler_LegacyCommandBehavesAsBefore(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()

	f.send(`{"data":"AQIDBA==","confirmed":false}`)

	require.Equal(t, 1, f.enqueuer.callCount())
	falseValue := false
	assert.Equal(t, &mioty.DLDataQueue{
		EpEui:       cmdTestEpEUI,
		UserData:    mioty.DownlinkUserData{{0x01, 0x02, 0x03, 0x04}},
		ResponseExp: &falseValue,
	}, f.enqueuer.lastCall().Request, "no optional flag is set unless the command names it")
	body := f.outcome(t, DeviceEventDownlinkQueued)
	assert.Equal(t, cmdTestEpEUIHex, body["epEui"])
	assert.Equal(t, float64(cmdTestQueID), body["queId"], "the queue id is exact for a JavaScript consumer")
	assert.NotContains(t, body, "ref", "no ref is echoed when the command carried none")
	assert.Empty(t, f.enqueuer.lastCall().Command.Ref, "a command without a ref queues a downlink without one")
}

func TestCommandHandler_ConfirmedFlagPropagated(t *testing.T) {
	t.Parallel()
	for _, confirmed := range []bool{false, true} {
		f := newCommandFixture()
		f.send(fmt.Sprintf(`{"data":"dGVzdA==","confirmed":%t}`, confirmed))
		require.Equal(t, 1, f.enqueuer.callCount())
		require.NotNil(t, f.enqueuer.lastCall().Request.ResponseExp)
		assert.Equal(t, confirmed, *f.enqueuer.lastCall().Request.ResponseExp)
	}
}

func TestCommandHandler_OptionalQueueFieldsReachTheCore(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()

	f.send(`{"data":"AQ==","prio":2.5,"format":7,"responsePrio":true,"dlWindReq":true,"expOnly":true,"dlRxStatQry":true}`)

	require.Equal(t, 1, f.enqueuer.callCount())
	req := f.enqueuer.lastCall().Request
	require.NotNil(t, req.Prio)
	assert.Equal(t, float32(2.5), *req.Prio)
	require.NotNil(t, req.Format)
	assert.Equal(t, uint8(7), *req.Format)
	for name, flag := range map[string]*bool{
		"responsePrio": req.ResponsePrio, "dlWindReq": req.DlWindReq, "expOnly": req.ExpOnly, "dlRxStatQry": req.DlRxStatQry,
	} {
		require.NotNil(t, flag, name)
		assert.True(t, *flag, name)
	}
	f.outcome(t, DeviceEventDownlinkQueued)
}

func TestCommandHandler_CounterDependentEntries(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()

	f.send(`{"entries":[{"packetCnt":10,"data":"AQ=="},{"packetCnt":11,"data":"Ag=="}]}`)

	require.Equal(t, 1, f.enqueuer.callCount())
	req := f.enqueuer.lastCall().Request
	assert.True(t, req.CntDepend)
	assert.Equal(t, []uint32{10, 11}, req.PacketCnt)
	assert.Equal(t, mioty.DownlinkUserData{{0x01}, {0x02}}, req.UserData)
	f.outcome(t, DeviceEventDownlinkQueued)
}

func TestCommandHandler_EmptyDataQueuesAPureAcknowledgement(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()

	f.send(`{"data":"","confirmed":false}`)

	require.Equal(t, 1, f.enqueuer.callCount(), "an empty payload is a pure acknowledgement downlink (BSSCI §3.12)")
	req := f.enqueuer.lastCall().Request
	require.Len(t, req.UserData, 1)
	assert.Empty(t, req.UserData[0])
	assert.False(t, req.CntDepend)
	f.outcome(t, DeviceEventDownlinkQueued)
}

func TestCommandHandler_PayloadAtTheRadioLimitIsQueued(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()

	f.send(`{"data":"` + b64(make([]byte, mioty.MaxDLUserDataBytes)) + `"}`)

	require.Equal(t, 1, f.enqueuer.callCount())
	f.outcome(t, DeviceEventDownlinkQueued)
}

func TestCommandHandler_RefEchoedOnEveryOutcome(t *testing.T) {
	t.Parallel()
	queued := newCommandFixture()
	queued.send(`{"data":"AQ==","ref":"order-17"}`)
	assert.Equal(t, "order-17", queued.outcome(t, DeviceEventDownlinkQueued)["ref"])
	assert.Equal(t, "order-17", queued.enqueuer.lastCall().Command.Ref, "the ref is queued with the downlink for its results")

	rejected := newCommandFixture()
	rejected.send(`{"data":"not base64!","ref":"order-18"}`)
	assert.Equal(t, "order-18", rejected.rejected(t, RejectCodeInvalidBase64)["ref"])

	wrongType := newCommandFixture()
	wrongType.send(`{"data":"AQ==","format":300,"ref":"order-19"}`)
	assert.Equal(t, "order-19", wrongType.rejected(t, RejectCodeInvalidField)["ref"],
		"the ref survives a field of the wrong type")
}

// TestCommandHandler_DeadlineReachesTheQueue: an RFC 3339 expiresAt, with or
// without fractional seconds and in any offset, is queued with the downlink
// as the instant it names; a command without one queues no deadline.
func TestCommandHandler_DeadlineReachesTheQueue(t *testing.T) {
	t.Parallel()
	for value, want := range map[string]time.Time{
		"2026-10-01T12:00:00Z":                time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		"2026-10-01T12:00:00.123456789Z":      time.Date(2026, 10, 1, 12, 0, 0, 123456789, time.UTC),
		"2026-10-01T14:00:00.000000001+02:00": time.Date(2026, 10, 1, 12, 0, 0, 1, time.UTC),
	} {
		f := newCommandFixture()
		f.send(`{"data":"AQ==","ref":"order-17","expiresAt":"` + value + `"}`)
		command := f.enqueuer.lastCall().Command
		require.NotNil(t, command.ExpiresAt, value)
		assert.True(t, want.Equal(*command.ExpiresAt), "%s queued as %v", value, *command.ExpiresAt)
		assert.Equal(t, "order-17", command.Ref)
		f.outcome(t, DeviceEventDownlinkQueued)
	}

	without := newCommandFixture()
	without.send(`{"data":"AQ=="}`)
	assert.Nil(t, without.enqueuer.lastCall().Command.ExpiresAt)
}

// TestCommandHandler_UnreadableDeadlineIsRefused: an expiresAt that is no RFC
// 3339 time, or no string, refuses the command as an invalid field with its
// ref echoed, and queues nothing.
func TestCommandHandler_UnreadableDeadlineIsRefused(t *testing.T) {
	t.Parallel()
	for _, value := range []string{`"tomorrow"`, `"2026-10-01 12:00:00"`, `"2026-10-01"`, `1790794743`} {
		f := newCommandFixture()
		f.send(`{"data":"AQ==","ref":"order-20","expiresAt":` + value + `}`)
		body := f.rejected(t, RejectCodeInvalidField)
		assert.Equal(t, "order-20", body["ref"], value)
		assert.Contains(t, body["message"], CommandFieldExpiresAt, value)
	}
}

// TestCommandHandler_RepeatedRefIsNotAnsweredAgain: a command whose ref
// already queued a downlink was answered when it was accepted, so its repeat
// publishes neither downlink_queued nor downlink_rejected, also when the
// queue recognizes it only while queueing, racing the first reception.
func TestCommandHandler_RepeatedRefIsNotAnsweredAgain(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()
	f.enqueuer.returnErr = fmt.Errorf("%w: core", ErrCommandAlreadyQueued)

	f.send(`{"data":"AQ==","ref":"order-17","expiresAt":"2026-10-01T12:00:00Z"}`)

	assert.Equal(t, 1, f.enqueuer.callCount(), "the repeat reaches the queue, which recognizes the ref")
	assert.Empty(t, f.pub.messages(), "nothing is published for a repeat")
}

// TestCommandHandler_AKnownRefIsRecognizedBeforeAnyCheck: a repeat of an
// accepted command publishes nothing and queues nothing, even when the
// endpoint was deleted or lost its downlink capability since and the core
// would refuse it, or its payload would now be refused; the ref is looked up
// in the organization's endpoint under its tenant.
func TestCommandHandler_AKnownRefIsRecognizedBeforeAnyCheck(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		payload string
		coreErr error
	}{
		"endpoint deleted": {`{"data":"AQ==","ref":"order-30"}`,
			fmt.Errorf("%w: %w", &DownlinkRefusal{Code: "scaci.error.endpoint_not_found", Message: "Endpoint not found"}, errCmdTestCore)},
		"endpoint lost bidi": {`{"data":"AQ==","ref":"order-30"}`,
			fmt.Errorf("%w: %w", &DownlinkRefusal{Code: "scaci.error.endpoint_not_bidirectional", Message: "not bidirectional"}, errCmdTestCore)},
		"payload refused now": {`{"data":"not base64!","ref":"order-30","expiresAt":"tomorrow"}`, nil},
	} {
		t.Run(name, func(t *testing.T) {
			f := newCommandFixture()
			f.enqueuer.queuedRefs = map[string]bool{"order-30": true}
			f.enqueuer.returnErr = tc.coreErr

			f.send(tc.payload)

			assert.Empty(t, f.pub.messages(), "nothing is published for a repeat")
			assert.Zero(t, f.enqueuer.callCount(), "nothing is queued")
			require.Len(t, f.enqueuer.refLookups, 1)
			assert.Equal(t, storage.DownlinkCommandRef{TenantID: cmdTestTenantID, OrganizationID: f.org, EpEUI: cmdTestEpEUI, Ref: "order-30"},
				f.enqueuer.refLookups[0])
		})
	}
}

// TestCommandHandler_ARefThatCannotBeLookedUpIsNotAnswered: when the queue
// cannot tell whether the ref was accepted, the command is neither queued nor
// refused, so an earlier acceptance is never contradicted; a command without
// a ref is not looked up.
func TestCommandHandler_ARefThatCannotBeLookedUpIsNotAnswered(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()
	f.enqueuer.refErr = errCmdTestCore

	f.send(`{"data":"AQ==","ref":"order-31"}`)

	assert.Empty(t, f.pub.messages())
	assert.Zero(t, f.enqueuer.callCount())

	unnamed := newCommandFixture()
	unnamed.enqueuer.refErr = errCmdTestCore
	unnamed.send(`{"data":"AQ=="}`)
	assert.Empty(t, unnamed.enqueuer.refLookups)
	unnamed.outcome(t, DeviceEventDownlinkQueued)
}

// TestCommandHandler_LogsNoUnboundedPublisherText: an unreadable expiresAt is
// logged cut to the length of an RFC 3339 time, quoted, without the parser's
// error text that repeats it; a repeated ref is logged at most as long as
// the queue stores one.
func TestCommandHandler_LogsNoUnboundedPublisherText(t *testing.T) {
	t.Parallel()
	log := bsscitest.NewRecordingLogger()
	f := newCommandFixture()
	f.handler = NewCommandHandler(f.pub, f.enqueuer, f.lookup, log, cmdTestPrefix)
	long := strings.Repeat("x", 4*len(time.RFC3339Nano))

	f.send(`{"data":"AQ==","expiresAt":"` + long + `"}`)

	entries := log.FilterMessage(LogCommandDownlinkRejected)
	require.Len(t, entries, 1)
	logged := fmt.Sprint(entries[0].FieldMap()[logger.FieldError])
	assert.Contains(t, logged, logger.FieldExpiresAt+`="`+long[:len(time.RFC3339Nano)]+`"`)
	assert.NotContains(t, logged, long[:len(time.RFC3339Nano)+1])
	assert.NotContains(t, logged, "cannot parse", "the parser's error text repeats the input")
}

// TestCommandHandler_LoggedRefCannotForgeALine: a repeated ref carrying a
// line break and terminal escapes is logged quoted, so it stays one field of
// one line.
func TestCommandHandler_LoggedRefCannotForgeALine(t *testing.T) {
	t.Parallel()
	log := bsscitest.NewRecordingLogger()
	f := newCommandFixture()
	f.handler = NewCommandHandler(f.pub, f.enqueuer, f.lookup, log, cmdTestPrefix)
	f.enqueuer.queuedRefs = map[string]bool{"order-40\nlevel=error msg=forged\x1b[31m": true}

	f.send(`{"data":"AQ==","ref":"order-40\nlevel=error msg=forged\u001b[31m"}`)

	entries := log.FilterMessage(LogCommandAlreadyQueued)
	require.Len(t, entries, 1)
	assert.Equal(t, `"order-40\nlevel=error msg=forged\x1b[31m"`, entries[0].FieldMap()[logger.FieldRef])
}

// TestCommandHandler_KnownRefWinsOverAFieldOfTheWrongType: a repeat whose
// ref decoded is recognized although another field no longer decodes; a new
// ref with such a field is refused once the lookup proved it new, and a ref
// that is itself of the wrong type is refused without a lookup.
func TestCommandHandler_KnownRefWinsOverAFieldOfTheWrongType(t *testing.T) {
	t.Parallel()
	known := newCommandFixture()
	known.enqueuer.queuedRefs = map[string]bool{"order-41": true}
	known.send(`{"data":"AQ==","ref":"order-41","prio":"high"}`)
	assert.Empty(t, known.pub.messages(), "a repeat is never refused")
	assert.Len(t, known.enqueuer.refLookups, 1)

	fresh := newCommandFixture()
	fresh.send(`{"data":"AQ==","ref":"order-42","prio":"high"}`)
	assert.Equal(t, "order-42", fresh.rejected(t, RejectCodeInvalidField)["ref"])
	assert.Len(t, fresh.enqueuer.refLookups, 1)

	badRef := newCommandFixture()
	badRef.send(`{"data":"AQ==","ref":42}`)
	badRef.rejected(t, RejectCodeInvalidField)
	assert.Empty(t, badRef.enqueuer.refLookups, "a ref that did not decode is never looked up")
}

// TestCommandHandler_RefIsBoundedLikeTheQueueStoresIt: a ref of
// storage.MaxDownlinkRefBytes is queued with the downlink; a longer one is
// refused before anything is queued, and still echoed for correlation.
func TestCommandHandler_RefIsBoundedLikeTheQueueStoresIt(t *testing.T) {
	t.Parallel()
	longest := strings.Repeat("r", storage.MaxDownlinkRefBytes)
	queued := newCommandFixture()
	queued.send(`{"data":"AQ==","ref":"` + longest + `"}`)
	assert.Equal(t, longest, queued.enqueuer.lastCall().Command.Ref)
	queued.outcome(t, DeviceEventDownlinkQueued)

	tooLong := longest + "r"
	refused := newCommandFixture()
	refused.send(`{"data":"AQ==","ref":"` + tooLong + `"}`)
	assert.Zero(t, refused.enqueuer.callCount(), "nothing is queued")
	body := refused.rejected(t, RejectCodeRefTooLong)
	assert.Equal(t, fmt.Sprintf(RejectMsgRefTooLongFmt, storage.MaxDownlinkRefBytes), body["message"])
	assert.Equal(t, tooLong, body["ref"])
}

func TestCommandHandler_ValidationRefusalsAreReported(t *testing.T) {
	t.Parallel()
	tooLarge := b64(make([]byte, mioty.MaxDLUserDataBytes+1))
	cases := []struct {
		name    string
		payload string
		code    string
	}{
		{"empty message", ``, RejectCodeEmptyPayload},
		{"not JSON", `{"data":`, RejectCodeInvalidJSON},
		{"JSON that is not an object", `["AQ=="]`, RejectCodeInvalidJSON},
		{"format beyond 8 bits", `{"data":"AQ==","format":300}`, RejectCodeInvalidField},
		{"prio that is not a number", `{"data":"AQ==","prio":"high"}`, RejectCodeInvalidField},
		{"prio beyond single precision", `{"data":"AQ==","prio":1e39}`, RejectCodeInvalidField},
		{"packet counter beyond 32 bits", `{"entries":[{"packetCnt":4294967296,"data":"AQ=="}]}`, RejectCodeInvalidField},
		{"no data and no entries", `{"confirmed":true}`, RejectCodeMissingData},
		{"data and entries", `{"data":"AQ==","entries":[{"packetCnt":1,"data":"AQ=="}]}`, RejectCodeDataWithEntries},
		{"empty entries", `{"entries":[]}`, RejectCodeEmptyEntries},
		{"entry without a counter", `{"entries":[{"data":"AQ=="}]}`, RejectCodeMissingPacketCnt},
		{"repeated counter", `{"entries":[{"packetCnt":3,"data":"AQ=="},{"packetCnt":3,"data":"Ag=="}]}`, RejectCodeDuplicatePacketCnt},
		{"data that is not base64", `{"data":"%%%"}`, RejectCodeInvalidBase64},
		{"entry that is not base64", `{"entries":[{"packetCnt":1,"data":"%%%"}]}`, RejectCodeInvalidBase64},
		{"data beyond the radio limit", `{"data":"` + tooLarge + `"}`, RejectCodePayloadTooLarge},
		{"entry beyond the radio limit", `{"entries":[{"packetCnt":1,"data":"` + tooLarge + `"}]}`, RejectCodePayloadTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newCommandFixture()
			f.send(tc.payload)
			f.rejected(t, tc.code)
		})
	}
}

func TestCommandHandler_OversizedMessageIsReported(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()

	f.handler.handleMessage(testutil.TestContext(), f.commandTopic(), make([]byte, config.MaxMessageSize+1))

	f.rejected(t, RejectCodeMessageTooLarge)
}

// TestCommandHandler_UnresolvedOrganizationIsReported: an organization that
// does not exist is reported; one that could not be looked up is reported
// only for a command without a ref, since a ref may name an accepted command.
func TestCommandHandler_UnresolvedOrganizationIsReported(t *testing.T) {
	t.Parallel()
	missing := newCommandFixture()
	missing.lookup.returnErr = fmt.Errorf("organization: %w", storage.ErrNotFound)
	missing.send(`{"data":"AQ==","ref":"order-20"}`)
	assert.Equal(t, "order-20", missing.rejected(t, RejectCodeOrgUnresolved)["ref"])

	unnamed := newCommandFixture()
	unnamed.lookup.returnErr = errCmdTestCore
	unnamed.send(`{"data":"AQ=="}`)
	unnamed.rejected(t, RejectCodeOrgUnresolved)
}

// TestCommandHandler_OrganizationLookupFailureNeverRefusesARef: a lookup
// that failed for another reason than a missing organization publishes
// nothing for a command with a ref; it may repeat an accepted command.
func TestCommandHandler_OrganizationLookupFailureNeverRefusesARef(t *testing.T) {
	t.Parallel()
	log := bsscitest.NewRecordingLogger()
	f := newCommandFixture()
	f.handler = NewCommandHandler(f.pub, f.enqueuer, f.lookup, log, cmdTestPrefix)
	f.lookup.returnErr = errCmdTestCore

	f.send(`{"data":"AQ==","ref":"order-21"}`)

	assert.Empty(t, f.pub.messages())
	assert.Zero(t, f.enqueuer.callCount())
	assert.Len(t, log.FilterMessage(LogCommandOrgLookupFailed), 1)
}

func TestCommandHandler_CoreRefusalIsReportedWithItsCode(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()
	f.enqueuer.returnErr = fmt.Errorf("%w: %w", &DownlinkRefusal{Code: "scaci.error.endpoint_not_found", Message: "Endpoint not found"}, errCmdTestCore)

	f.send(`{"data":"AQ==","ref":"order-21"}`)

	require.Equal(t, 1, f.enqueuer.callCount())
	messages := f.pub.messages()
	require.Len(t, messages, 1)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(messages[0].Payload, &body))
	assert.Equal(t, DeviceEventTopic(cmdTestPrefix, f.org.String(), cmdTestEpEUIHex, DeviceEventDownlinkRejected), messages[0].Topic)
	assert.Equal(t, "scaci.error.endpoint_not_found", body["code"])
	assert.Equal(t, "Endpoint not found", body["message"])
	assert.Equal(t, "order-21", body["ref"])
}

func TestCommandHandler_UnnamedCoreFailureIsReported(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()
	f.enqueuer.returnErr = errCmdTestCore

	f.send(`{"data":"AQ=="}`)

	messages := f.pub.messages()
	require.Len(t, messages, 1)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(messages[0].Payload, &body))
	assert.Equal(t, RejectCodeEnqueueFailed, body["code"])
	assert.Equal(t, RejectMsgEnqueueFailed, body["message"])
}

func TestCommandHandler_UppercaseTopicEUIRepliesOnTheCanonicalTopic(t *testing.T) {
	t.Parallel()
	f := newCommandFixture()

	topic := fmt.Sprintf(TopicDeviceCommandDownFormat, cmdTestPrefix, f.org, strings.ToUpper(cmdTestEpEUIHex))
	f.handler.handleMessage(testutil.TestContext(), topic, []byte(`{"data":"AQ=="}`))

	f.outcome(t, DeviceEventDownlinkQueued)
}

func TestCommandHandler_UnaddressableCommandsPublishNothing(t *testing.T) {
	t.Parallel()
	org := uuid.New().String()
	cases := map[string]string{
		"too few segments": cmdTestPrefix + "/bad",
		"invalid org":      fmt.Sprintf(TopicDeviceCommandDownFormat, cmdTestPrefix, "not-a-uuid", cmdTestEpEUIHex),
		"nil org":          fmt.Sprintf(TopicDeviceCommandDownFormat, cmdTestPrefix, uuid.Nil, cmdTestEpEUIHex),
		"invalid EUI":      fmt.Sprintf(TopicDeviceCommandDownFormat, cmdTestPrefix, org, "not-hex"),
	}
	for name, topic := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newCommandFixture()
			f.handler.handleMessage(testutil.TestContext(), topic, []byte(`{"data":"AQ=="}`))
			assert.Zero(t, f.enqueuer.callCount())
			assert.Empty(t, f.pub.messages(), "without an organization and endpoint there is no topic to answer on")
		})
	}
}

// recordingLogger captures log messages for assertion.
type recordingLogger struct {
	MockLogger
	mu     sync.Mutex
	errors []string
}

func (r *recordingLogger) ErrorContext(_ context.Context, msg string, _ ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, msg)
}

func (r *recordingLogger) lastError() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.errors) == 0 {
		return ""
	}
	return r.errors[len(r.errors)-1]
}

// --- Nil Guard Tests ---

func TestCommandHandler_NilDependencies_EarlyReturn(t *testing.T) {
	t.Parallel()

	t.Run("nil client logs error", func(t *testing.T) {
		t.Parallel()
		log := &recordingLogger{}
		h := NewCommandHandler(nil, &mockDownlinkEnqueuer{}, &mockTenantLookup{}, log, cmdTestPrefix)
		ctx, cancel := context.WithCancel(testutil.TestContext())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
		assert.Equal(t, ErrCommandHandlerMissingDeps, log.lastError())
	})

	t.Run("nil enqueuer logs error", func(t *testing.T) {
		t.Parallel()
		log := &recordingLogger{}
		h := NewCommandHandler(newMockCommandPublisher(), nil, &mockTenantLookup{}, log, cmdTestPrefix)
		ctx, cancel := context.WithCancel(testutil.TestContext())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
		assert.Equal(t, ErrCommandHandlerMissingDeps, log.lastError())
	})

	t.Run("nil lookup logs error", func(t *testing.T) {
		t.Parallel()
		log := &recordingLogger{}
		h := NewCommandHandler(newMockCommandPublisher(), &mockDownlinkEnqueuer{}, nil, log, cmdTestPrefix)
		ctx, cancel := context.WithCancel(testutil.TestContext())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
		assert.Equal(t, ErrCommandHandlerMissingDeps, log.lastError())
	})

	t.Run("nil logger silent return", func(t *testing.T) {
		t.Parallel()
		h := NewCommandHandler(newMockCommandPublisher(), &mockDownlinkEnqueuer{}, &mockTenantLookup{}, nil, cmdTestPrefix)
		ctx, cancel := context.WithCancel(testutil.TestContext())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
	})
}

// --- Context Propagation Tests ---

func TestCommandHandler_ContextPropagation(t *testing.T) {
	t.Parallel()

	f := newCommandFixture()

	// Inject a marker value into the Start context
	type ctxKey struct{}
	startCtx, cancel := context.WithCancel(context.WithValue(testutil.TestContext(), ctxKey{}, "test-marker"))
	defer cancel()

	go f.handler.Start(startCtx)
	require.Eventually(t, func() bool { return f.pub.hasSubscribers() }, testTimeout, testPollInterval)

	f.pub.simulateMessage(f.commandTopic(), []byte(`{"data":"dGVzdA==","confirmed":false}`))
	require.Eventually(t, func() bool { return f.enqueuer.callCount() > 0 }, testTimeout, testPollInterval)

	call := f.enqueuer.lastCall()

	// Assert the context carries the start-time marker (subscribe-time ctx flowed through)
	assert.Equal(t, "test-marker", call.Ctx.Value(ctxKey{}))

	// Assert the handler enriched the context with tenant/org
	tenantID, err := pkgcontext.GetTenantID(call.Ctx)
	require.NoError(t, err)
	assert.Equal(t, cmdTestTenantID, tenantID)

	orgID, err := pkgcontext.GetOrganizationID(call.Ctx)
	require.NoError(t, err)
	assert.Equal(t, f.org, orgID)
}

// Test constants for timing
const (
	testTimeout      = 2 * 1e9  // 2 seconds in nanoseconds (time.Duration)
	testPollInterval = 10 * 1e6 // 10 milliseconds
)

// Ensure mockDownlinkEnqueuer implements DownlinkEnqueuer
var _ DownlinkEnqueuer = (*mockDownlinkEnqueuer)(nil)

// Ensure mockTenantLookup implements TenantLookup
var _ TenantLookup = (*mockTenantLookup)(nil)

// Ensure mockCommandPublisher implements Publisher
var _ Publisher = (*mockCommandPublisher)(nil)
