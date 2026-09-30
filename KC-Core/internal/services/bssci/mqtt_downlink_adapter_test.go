package bssciservices

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
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
	Ref      string
}

func (m *mockDownlinkQueuer) QueueDownlink(_ context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, ref string) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, queueDownlinkCall{
		TenantID: tenantID,
		OrgID:    orgID,
		Request:  req,
		Ref:      ref,
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

func newTestAdapter(t *testing.T, queuer DownlinkQueuer) mqtt.DownlinkEnqueuer {
	t.Helper()
	adapter, err := NewMQTTDownlinkAdapter(queuer)
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
	adapter, err := NewMQTTDownlinkAdapter(nil)
	require.ErrorIs(t, err, ErrNilQueuer)
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
// Application Center queue id and with the command's ref, in a single
// queueing call.
func TestMQTTDownlinkAdapter_QueuesTheRequestUnchanged(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 42}
	adapter := newTestAdapter(t, mock)
	orgID := uuid.New()
	confirmed := true
	req := &mioty.DLDataQueue{EpEui: 0x70B3D59CD00009E6, UserData: mioty.DownlinkUserData{[]byte("test")}, ResponseExp: &confirmed}

	queID, err := adapter.EnqueueFromMQTT(testutil.TestContext(), 42, &orgID, req, "order-17")
	require.NoError(t, err)

	assert.Equal(t, uint64(42), queID)
	assert.Equal(t, 1, mock.callCount())
	call := mock.lastCall()
	assert.Same(t, req, call.Request)
	assert.Zero(t, call.Request.QueId)
	assert.Equal(t, int64(42), call.TenantID)
	assert.Equal(t, &orgID, call.OrgID)
	assert.Equal(t, "order-17", call.Ref)
}

// TestMQTTDownlinkAdapter_CoreRefusalsCarryTheirCatalogCode pins that every
// refusal the core can name reaches the MQTT publisher with its catalog token
// and message: unknown endpoint, oversized payload, and a nil queue result.
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

			queID, err := adapter.EnqueueFromMQTT(testutil.TestContext(), 1, &orgID, &mioty.DLDataQueue{EpEui: 0x1234}, "")
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

func TestMQTTDownlinkAdapter_UnnamedFailureStaysUnnamed(t *testing.T) {
	t.Parallel()
	adapter := newTestAdapter(t, &mockDownlinkQueuer{returnErr: errTestQueueFull})
	orgID := uuid.New()

	queID, err := adapter.EnqueueFromMQTT(testutil.TestContext(), 1, &orgID, &mioty.DLDataQueue{EpEui: 0x1234}, "")
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
	ref          string
}

func (m *mockSCACIDownlinkServer) QueueDownlinkInternal(_ context.Context, _ int64, _ *uuid.UUID, _ *mioty.DLDataQueue, ref string) (*scaci.DLDataQueueResult, error) {
	m.ref = ref
	return m.returnResult, m.returnErr
}

var _ SCACIDownlinkServer = (*mockSCACIDownlinkServer)(nil)

func TestSCACIDownlinkQueuer_SuccessReturnsQueID(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnResult: &scaci.DLDataQueueResult{QueID: 42},
	}
	queuer := mustQueuer(t, mock)
	id, err := queuer.QueueDownlink(testutil.TestContext(), 1, nil, &mioty.DLDataQueue{}, "order-17")
	require.NoError(t, err)
	assert.Equal(t, uint64(42), id)
	assert.Equal(t, "order-17", mock.ref, "the ref reaches the core")
}

func TestSCACIDownlinkQueuer_ErrorPropagates(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnErr: errTestSCACIUnavailable,
	}
	queuer := mustQueuer(t, mock)
	id, err := queuer.QueueDownlink(testutil.TestContext(), 1, nil, &mioty.DLDataQueue{}, "")
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
	id, err := queuer.QueueDownlink(testutil.TestContext(), 1, nil, &mioty.DLDataQueue{}, "")
	assert.Error(t, err)
	var catErr *bssci.CatalogError
	require.ErrorAs(t, err, &catErr)
	assert.Equal(t, bssci.ErrDLQueueNilResult, catErr.Token)
	assert.Equal(t, bssci.POSIX_EIO, catErr.Posix)
	assert.Equal(t, uint64(0), id)
}
