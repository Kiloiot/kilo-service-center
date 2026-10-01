package bssciservices

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Fixture errors returned by the downlink adapter test doubles.
var (
	errTestQueueFull        = errors.New("queue full")
	errTestSCACIUnavailable = errors.New("scaci unavailable")
)

// mockDownlinkQueuer implements DownlinkQueuer for adapter tests.
type mockDownlinkQueuer struct {
	mu        sync.Mutex
	calls     []queueDownlinkCall
	returnID  uint64
	returnErr error
}

type queueDownlinkCall struct {
	TenantID int64
	OrgID    *uuid.UUID
	Request  *mioty.DLDataQueue
	Command  storage.DownlinkCommand
}

func (m *mockDownlinkQueuer) QueueDownlink(_ context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, command storage.DownlinkCommand) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, queueDownlinkCall{
		TenantID: tenantID,
		OrgID:    orgID,
		Request:  req,
		Command:  command,
	})
	return m.returnID, m.returnErr
}

func (m *mockDownlinkQueuer) lastCall() queueDownlinkCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

func (m *mockDownlinkQueuer) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// knownRefs answers a ref lookup from the refs it holds and records each lookup.
type knownRefs struct {
	queued  map[string]bool
	err     error
	lookups []storage.DownlinkCommandRef
}

func (k *knownRefs) CommandQueued(_ context.Context, command storage.DownlinkCommandRef) (bool, error) {
	k.lookups = append(k.lookups, command)
	return k.queued[command.Ref], k.err
}

func newTestAdapter(t *testing.T, queuer DownlinkQueuer) mqtt.DownlinkEnqueuer {
	t.Helper()
	adapter, err := NewMQTTDownlinkAdapter(queuer, &knownRefs{})
	require.NoError(t, err)
	return adapter
}

func mustQueuer(t *testing.T, server SCACIDownlinkServer) DownlinkQueuer {
	t.Helper()
	queuer, err := NewSCACIDownlinkQueuer(server)
	require.NoError(t, err)
	return queuer
}

func TestNewMQTTDownlinkAdapter_RejectsMissingCollaborators(t *testing.T) {
	adapter, err := NewMQTTDownlinkAdapter(nil, &knownRefs{})
	require.ErrorIs(t, err, ErrNilQueuer)
	assert.Nil(t, adapter)
	adapter, err = NewMQTTDownlinkAdapter(mustQueuer(t, &mockSCACIDownlinkServer{}), nil)
	require.ErrorIs(t, err, ErrNilCommandRefs)
	assert.Nil(t, adapter)

	queuer, err := NewSCACIDownlinkQueuer(nil)
	require.ErrorIs(t, err, ErrNilSCACIServer)
	assert.Nil(t, queuer)
}

// Compile-time interface assertions
var (
	_ DownlinkQueuer        = (*mockDownlinkQueuer)(nil)
	_ mqtt.DownlinkEnqueuer = (*mqttDownlinkAdapter)(nil)
)

// TestMQTTDownlinkAdapter_QueuesTheRequestUnchanged pins that the adapter
// hands the command's dlDataQue request to the core as built, with no
// Application Center queue id and with the command's ref and deadline, in a
// single queueing call.
func TestMQTTDownlinkAdapter_QueuesTheRequestUnchanged(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 42}
	adapter := newTestAdapter(t, mock)
	orgID := uuid.New()
	confirmed := true
	req := &mioty.DLDataQueue{EpEui: 0x70B3D59CD00009E6, UserData: mioty.DownlinkUserData{[]byte("test")}, ResponseExp: &confirmed}

	expiresAt := dispatchTestNow
	command := storage.DownlinkCommand{Ref: "order-17", ExpiresAt: &expiresAt}
	queID, err := adapter.EnqueueFromMQTT(testutil.TestContext(), 42, &orgID, req, command)
	require.NoError(t, err)

	assert.Equal(t, uint64(42), queID)
	assert.Equal(t, 1, mock.callCount())
	call := mock.lastCall()
	assert.Same(t, req, call.Request)
	assert.Zero(t, call.Request.QueId)
	assert.Equal(t, int64(42), call.TenantID)
	assert.Equal(t, &orgID, call.OrgID)
	assert.Equal(t, command, call.Command)
}

// TestMQTTDownlinkAdapter_CoreRefusalsCarryTheirCatalogCode pins that every
// refusal the core can name reaches the MQTT publisher with its catalog token
// and message: unknown endpoint, oversized payload, a deadline that passed
// before queueing (in the command's own code) and a nil queue result.
func TestMQTTDownlinkAdapter_CoreRefusalsCarryTheirCatalogCode(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		err     error
		code    string
		message string
	}{
		"unknown endpoint": {
			&scaci.DLDataQueueError{Token: scaci.ErrEndpointNotFound, POSIX: scaci.POSIX_ENOENT},
			scaci.ErrEndpointNotFound, scaci.GetErrorDefinition(scaci.ErrEndpointNotFound).Message,
		},
		"payload beyond the radio limit": {
			&scaci.DLDataQueueError{Token: scaci.ErrDLPayloadTooLarge, POSIX: scaci.POSIX_EINVAL},
			scaci.ErrDLPayloadTooLarge, scaci.GetErrorDefinition(scaci.ErrDLPayloadTooLarge).Message,
		},
		"deadline passed before the downlink was queued": {
			&scaci.DLDataQueueError{Token: scaci.ErrDownlinkDeadlineElapsed, POSIX: scaci.POSIX_ETIMEDOUT},
			mqtt.RejectCodeCommandExpired, mqtt.RejectMsgCommandExpired,
		},
		"no queue result": {
			bssci.NewCatalogError(bssci.ErrDLQueueNilResult, bssci.POSIX_EIO),
			bssci.ErrDLQueueNilResult, bssci.ResolveErrorMessage(bssci.ErrDLQueueNilResult),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			adapter := newTestAdapter(t, &mockDownlinkQueuer{returnErr: tc.err})
			orgID := uuid.New()

			queID, err := adapter.EnqueueFromMQTT(testutil.TestContext(), 1, &orgID, &mioty.DLDataQueue{EpEui: 0x1234}, storage.DownlinkCommand{})
			require.Error(t, err)
			assert.Zero(t, queID)
			var refusal *mqtt.DownlinkRefusal
			require.ErrorAs(t, err, &refusal, "the MQTT publisher learns why the core refused")
			assert.Equal(t, tc.code, refusal.Code)
			assert.Equal(t, tc.message, refusal.Message)
			assert.ErrorIs(t, err, tc.err, "the core error stays in the chain for the log")
		})
	}
}

// TestMQTTDownlinkAdapter_RepeatedRefIsNoRefusal: a command whose ref already
// queued a downlink reaches the handler as a repeat, never as a refusal it
// would publish.
func TestMQTTDownlinkAdapter_RepeatedRefIsNoRefusal(t *testing.T) {
	t.Parallel()
	repeat := &scaci.DLDataQueueError{Token: scaci.ErrDownlinkCommandRefQueued, POSIX: scaci.POSIX_EEXIST}
	adapter := newTestAdapter(t, &mockDownlinkQueuer{returnErr: repeat})
	orgID := uuid.New()

	queID, err := adapter.EnqueueFromMQTT(testutil.TestContext(), 1, &orgID, &mioty.DLDataQueue{EpEui: 0x1234}, storage.DownlinkCommand{Ref: "order-17"})
	require.ErrorIs(t, err, mqtt.ErrCommandAlreadyQueued)
	assert.Zero(t, queID)
	var refusal *mqtt.DownlinkRefusal
	assert.False(t, errors.As(err, &refusal), "a repeat is not reported as refused")
}

func TestMQTTDownlinkAdapter_UnnamedFailureStaysUnnamed(t *testing.T) {
	t.Parallel()
	adapter := newTestAdapter(t, &mockDownlinkQueuer{returnErr: errTestQueueFull})
	orgID := uuid.New()

	queID, err := adapter.EnqueueFromMQTT(testutil.TestContext(), 1, &orgID, &mioty.DLDataQueue{EpEui: 0x1234}, storage.DownlinkCommand{})
	require.ErrorIs(t, err, errTestQueueFull)
	assert.Zero(t, queID)
	var refusal *mqtt.DownlinkRefusal
	assert.False(t, errors.As(err, &refusal), "the command handler reports it as an enqueue failure")
}

// --- scaciDownlinkQueuer wrapper tests ---

// mockSCACIDownlinkServer implements SCACIDownlinkServer for wrapper tests.
type mockSCACIDownlinkServer struct {
	returnResult *scaci.DLDataQueueResult
	returnErr    error
	command      storage.DownlinkCommand
}

func (m *mockSCACIDownlinkServer) QueueDownlinkInternal(_ context.Context, _ int64, _ *uuid.UUID, _ *mioty.DLDataQueue, command storage.DownlinkCommand) (*scaci.DLDataQueueResult, error) {
	m.command = command
	return m.returnResult, m.returnErr
}

var _ SCACIDownlinkServer = (*mockSCACIDownlinkServer)(nil)

func TestSCACIDownlinkQueuer_SuccessReturnsQueID(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnResult: &scaci.DLDataQueueResult{QueID: 42},
	}
	queuer := mustQueuer(t, mock)
	command := storage.DownlinkCommand{Ref: "order-17"}
	id, err := queuer.QueueDownlink(testutil.TestContext(), 1, nil, &mioty.DLDataQueue{}, command)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), id)
	assert.Equal(t, command, mock.command, "the command reaches the core")
}

func TestSCACIDownlinkQueuer_ErrorPropagates(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnErr: errTestSCACIUnavailable,
	}
	queuer := mustQueuer(t, mock)
	id, err := queuer.QueueDownlink(testutil.TestContext(), 1, nil, &mioty.DLDataQueue{}, storage.DownlinkCommand{})
	assert.Error(t, err)
	assert.Equal(t, uint64(0), id)
}

func TestSCACIDownlinkQueuer_NilResultReturnsError(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnResult: nil,
		returnErr:    nil,
	}
	queuer := mustQueuer(t, mock)
	id, err := queuer.QueueDownlink(testutil.TestContext(), 1, nil, &mioty.DLDataQueue{}, storage.DownlinkCommand{})
	assert.Error(t, err)
	var catErr *bssci.CatalogError
	require.ErrorAs(t, err, &catErr)
	assert.Equal(t, bssci.ErrDLQueueNilResult, catErr.Token)
	assert.Equal(t, bssci.POSIX_EIO, catErr.Posix)
	assert.Equal(t, uint64(0), id)
}

// TestMQTTDownlinkAdapter_LooksUpTheCommandRef: the adapter answers whether
// the organization's endpoint already queued the ref from the queue itself.
func TestMQTTDownlinkAdapter_LooksUpTheCommandRef(t *testing.T) {
	refs := &knownRefs{queued: map[string]bool{"order-40": true}}
	adapter, err := NewMQTTDownlinkAdapter(mustQueuer(t, &mockSCACIDownlinkServer{}), refs)
	require.NoError(t, err)
	command := storage.DownlinkCommandRef{TenantID: 3, OrganizationID: uuid.New(), EpEUI: 0x70b3d59cd0000341, Ref: "order-40"}

	queued, err := adapter.CommandQueued(testutil.TestContext(), command)

	require.NoError(t, err)
	assert.True(t, queued)
	assert.Equal(t, []storage.DownlinkCommandRef{command}, refs.lookups)
}
