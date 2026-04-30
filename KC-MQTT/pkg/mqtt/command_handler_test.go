package mqtt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/common/config"
	pkgcontext "github.com/kilocenter/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDownlinkEnqueuer records EnqueueFromMQTT calls.
type mockDownlinkEnqueuer struct {
	mu        sync.Mutex
	calls     []enqueueCall
	returnID  uint64
	returnErr error
}

type enqueueCall struct {
	Ctx       context.Context
	TenantID  int64
	OrgID     *uuid.UUID
	EpEUI     uint64
	Payload   []byte
	Confirmed bool
}

func (m *mockDownlinkEnqueuer) EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID,
	epEUI uint64, payload []byte, confirmed bool) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, enqueueCall{
		Ctx:       ctx,
		TenantID:  tenantID,
		OrgID:     orgID,
		EpEUI:     epEUI,
		Payload:   payload,
		Confirmed: confirmed,
	})
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

// mockCommandPublisher is a minimal Publisher for command handler tests.
type mockCommandPublisher struct {
	mu          sync.Mutex
	subscribers map[string]MessageHandler
	subCtx      context.Context // captured from Subscribe call
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
func (m *mockCommandPublisher) Publish(_ context.Context, _ string, _ byte, _ bool, _ interface{}) error {
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

// simulateMessage simulates a message arriving on the given topic.
// Uses the subscribe-time context (matching production client.go behavior).
func (m *mockCommandPublisher) simulateMessage(topic string, payload []byte) {
	m.mu.Lock()
	// Find matching handler (exact match or wildcard match)
	var handler MessageHandler
	ctx := m.subCtx
	if ctx == nil {
		ctx = context.Background()
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

func buildCommandPayload(data string, confirmed bool) []byte {
	cmd := commandPayload{Data: data, Confirmed: confirmed}
	b, _ := json.Marshal(cmd)
	return b
}

func TestCommandHandler_ValidTopicParsesCorrectly(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	epEUIHex := "70b3d59cd00009e6"
	topic := "mioty/" + orgUUID.String() + "/device/" + epEUIHex + "/command/down"

	enqueuer := &mockDownlinkEnqueuer{returnID: 42}
	lookup := &mockTenantLookup{mapping: map[uuid.UUID]int64{orgUUID: 100}}
	pub := newMockCommandPublisher()
	log := &MockLogger{}

	handler := NewCommandHandler(pub, enqueuer, lookup, log, "mioty")

	// Start in background, then simulate message
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handler.Start(ctx)

	// Wait for subscription to be registered before simulating
	require.Eventually(t, func() bool { return pub.hasSubscribers() }, testTimeout, testPollInterval)

	rawPayload := base64.StdEncoding.EncodeToString([]byte("hello world"))
	cmdPayload := buildCommandPayload(rawPayload, true)
	pub.simulateMessage(topic, cmdPayload)

	// Wait for enqueue call
	require.Eventually(t, func() bool { return enqueuer.callCount() > 0 }, testTimeout, testPollInterval)

	call := enqueuer.lastCall()
	assert.Equal(t, int64(100), call.TenantID)
	assert.NotNil(t, call.OrgID)
	assert.Equal(t, orgUUID, *call.OrgID)
	assert.Equal(t, uint64(0x70b3d59cd00009e6), call.EpEUI)
	assert.Equal(t, []byte("hello world"), call.Payload)
	assert.True(t, call.Confirmed)
}

func TestCommandHandler_InvalidTopicTooFewSegments(t *testing.T) {
	t.Parallel()

	enqueuer := &mockDownlinkEnqueuer{}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		logger:   log,
		prefix:   "mioty",
	}

	// Directly call handleMessage with a bad topic
	handler.handleMessage(context.Background(), "mioty/bad", []byte("{}"))

	assert.Equal(t, 0, enqueuer.callCount())
}

func TestCommandHandler_InvalidOrgUUID(t *testing.T) {
	t.Parallel()

	enqueuer := &mockDownlinkEnqueuer{}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		logger:   log,
		prefix:   "mioty",
	}

	handler.handleMessage(context.Background(), "mioty/not-a-uuid/device/70b3d59cd00009e6/command/down", []byte("{}"))

	assert.Equal(t, 0, enqueuer.callCount())
}

func TestCommandHandler_InvalidHexEUI(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	enqueuer := &mockDownlinkEnqueuer{}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		logger:   log,
		prefix:   "mioty",
	}

	topic := "mioty/" + orgUUID.String() + "/device/not-hex/command/down"
	handler.handleMessage(context.Background(), topic, buildCommandPayload(base64.StdEncoding.EncodeToString([]byte("test")), false))

	assert.Equal(t, 0, enqueuer.callCount())
}

func TestCommandHandler_EmptyPayloadRejected(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	enqueuer := &mockDownlinkEnqueuer{}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		logger:   log,
		prefix:   "mioty",
	}

	topic := "mioty/" + orgUUID.String() + "/device/70b3d59cd00009e6/command/down"
	handler.handleMessage(context.Background(), topic, []byte{})

	assert.Equal(t, 0, enqueuer.callCount())
}

func TestCommandHandler_OversizedRawPayloadRejected(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	enqueuer := &mockDownlinkEnqueuer{}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		logger:   log,
		prefix:   "mioty",
	}

	topic := "mioty/" + orgUUID.String() + "/device/70b3d59cd00009e6/command/down"
	oversized := make([]byte, config.MaxMessageSize+1)
	handler.handleMessage(context.Background(), topic, oversized)

	assert.Equal(t, 0, enqueuer.callCount())
}

func TestCommandHandler_DecodedEmptyPayloadRejected(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	enqueuer := &mockDownlinkEnqueuer{}
	lookup := &mockTenantLookup{mapping: map[uuid.UUID]int64{orgUUID: 1}}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		lookup:   lookup,
		logger:   log,
		prefix:   "mioty",
	}

	topic := "mioty/" + orgUUID.String() + "/device/70b3d59cd00009e6/command/down"
	// Base64 of empty string
	cmdPayload := buildCommandPayload("", false)
	handler.handleMessage(context.Background(), topic, cmdPayload)

	assert.Equal(t, 0, enqueuer.callCount())
}

func TestCommandHandler_DecodedOversizedPayloadRejected(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	enqueuer := &mockDownlinkEnqueuer{}
	lookup := &mockTenantLookup{mapping: map[uuid.UUID]int64{orgUUID: 1}}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		lookup:   lookup,
		logger:   log,
		prefix:   "mioty",
	}

	topic := "mioty/" + orgUUID.String() + "/device/70b3d59cd00009e6/command/down"
	// Create payload larger than MaxPayloadSize
	bigData := make([]byte, config.MaxPayloadSize+1)
	encoded := base64.StdEncoding.EncodeToString(bigData)
	cmdPayload := buildCommandPayload(encoded, false)
	handler.handleMessage(context.Background(), topic, cmdPayload)

	assert.Equal(t, 0, enqueuer.callCount())
}

func TestCommandHandler_ConfirmedFlagPropagated(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	enqueuer := &mockDownlinkEnqueuer{returnID: 1}
	lookup := &mockTenantLookup{mapping: map[uuid.UUID]int64{orgUUID: 1}}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		lookup:   lookup,
		logger:   log,
		prefix:   "mioty",
	}

	topic := "mioty/" + orgUUID.String() + "/device/70b3d59cd00009e6/command/down"

	// Test confirmed=false
	cmdPayload := buildCommandPayload(base64.StdEncoding.EncodeToString([]byte("test")), false)
	handler.handleMessage(context.Background(), topic, cmdPayload)
	require.Equal(t, 1, enqueuer.callCount())
	assert.False(t, enqueuer.lastCall().Confirmed)

	// Test confirmed=true
	cmdPayload = buildCommandPayload(base64.StdEncoding.EncodeToString([]byte("test2")), true)
	handler.handleMessage(context.Background(), topic, cmdPayload)
	require.Equal(t, 2, enqueuer.callCount())
	assert.True(t, enqueuer.lastCall().Confirmed)
}

func TestCommandHandler_NilOrgUUIDRejected(t *testing.T) {
	t.Parallel()

	enqueuer := &mockDownlinkEnqueuer{}
	log := &MockLogger{}

	handler := &CommandHandler{
		enqueuer: enqueuer,
		logger:   log,
		prefix:   "mioty",
	}

	// uuid.Nil should be rejected
	topic := "mioty/" + uuid.Nil.String() + "/device/70b3d59cd00009e6/command/down"
	cmdPayload := buildCommandPayload(base64.StdEncoding.EncodeToString([]byte("test")), false)
	handler.handleMessage(context.Background(), topic, cmdPayload)

	assert.Equal(t, 0, enqueuer.callCount())
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
		h := NewCommandHandler(nil, &mockDownlinkEnqueuer{}, &mockTenantLookup{}, log, "mioty")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
		assert.Equal(t, ErrCommandHandlerMissingDeps, log.lastError())
	})

	t.Run("nil enqueuer logs error", func(t *testing.T) {
		t.Parallel()
		log := &recordingLogger{}
		h := NewCommandHandler(newMockCommandPublisher(), nil, &mockTenantLookup{}, log, "mioty")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
		assert.Equal(t, ErrCommandHandlerMissingDeps, log.lastError())
	})

	t.Run("nil lookup logs error", func(t *testing.T) {
		t.Parallel()
		log := &recordingLogger{}
		h := NewCommandHandler(newMockCommandPublisher(), &mockDownlinkEnqueuer{}, nil, log, "mioty")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
		assert.Equal(t, ErrCommandHandlerMissingDeps, log.lastError())
	})

	t.Run("nil logger silent return", func(t *testing.T) {
		t.Parallel()
		h := NewCommandHandler(newMockCommandPublisher(), &mockDownlinkEnqueuer{}, &mockTenantLookup{}, nil, "mioty")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.NotPanics(t, func() { h.Start(ctx) })
	})
}

// --- Context Propagation Tests ---

func TestCommandHandler_ContextPropagation(t *testing.T) {
	t.Parallel()

	orgUUID := uuid.New()
	enqueuer := &mockDownlinkEnqueuer{returnID: 1}
	lookup := &mockTenantLookup{mapping: map[uuid.UUID]int64{orgUUID: 42}}
	pub := newMockCommandPublisher()
	log := &MockLogger{}
	handler := NewCommandHandler(pub, enqueuer, lookup, log, "mioty")

	// Inject a marker value into the Start context
	type ctxKey struct{}
	startCtx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey{}, "test-marker"))
	defer cancel()

	go handler.Start(startCtx)
	require.Eventually(t, func() bool { return pub.hasSubscribers() }, testTimeout, testPollInterval)

	// Simulate a valid command message
	topic := "mioty/" + orgUUID.String() + "/device/70b3d59cd00009e6/command/down"
	cmdPayload := buildCommandPayload(base64.StdEncoding.EncodeToString([]byte("test")), false)
	pub.simulateMessage(topic, cmdPayload)

	require.Eventually(t, func() bool { return enqueuer.callCount() > 0 }, testTimeout, testPollInterval)

	call := enqueuer.lastCall()

	// Assert the context carries the start-time marker (subscribe-time ctx flowed through)
	assert.Equal(t, "test-marker", call.Ctx.Value(ctxKey{}))

	// Assert the handler enriched the context with tenant/org
	tenantID, err := pkgcontext.GetTenantID(call.Ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(42), tenantID)

	orgID, err := pkgcontext.GetOrganizationID(call.Ctx)
	require.NoError(t, err)
	assert.Equal(t, orgUUID, orgID)
}

// Test constants for timing
const (
	testTimeout      = 2 * 1e9  // 2 seconds in nanoseconds (time.Duration)
	testPollInterval = 10 * 1e6 // 10 milliseconds
)

// mockLogger for command handler tests (re-uses the same pattern from client_test.go)
// Note: MockLogger is already defined in client_test.go within the same package.
// We rely on it being available since both files are in package mqtt.

// Ensure mockDownlinkEnqueuer implements DownlinkEnqueuer
var _ DownlinkEnqueuer = (*mockDownlinkEnqueuer)(nil)

// Ensure mockTenantLookup implements TenantLookup
var _ TenantLookup = (*mockTenantLookup)(nil)

// Ensure mockCommandPublisher implements Publisher
var _ Publisher = (*mockCommandPublisher)(nil)
