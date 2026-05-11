package bssciservices

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}

func (m *mockDownlinkQueuer) QueueDownlink(_ context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, queueDownlinkCall{
		TenantID: tenantID,
		OrgID:    orgID,
		Request:  req,
	})
	return m.returnID, m.returnErr
}

func (m *mockDownlinkQueuer) lastCall() queueDownlinkCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

func (m *mockDownlinkQueuer) allQueueIDs() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]uint64, 0, len(m.calls))
	for _, call := range m.calls {
		if call.Request != nil {
			ids = append(ids, call.Request.QueId)
		}
	}
	return ids
}

// Compile-time interface assertions
var _ DownlinkQueuer = (*mockDownlinkQueuer)(nil)
var _ mqtt.DownlinkEnqueuer = (*mqttDownlinkAdapter)(nil)

func TestMQTTDownlinkAdapter_GeneratesNonZeroQueID(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 42}
	adapter := NewMQTTDownlinkAdapter(mock)

	orgID := uuid.New()
	_, err := adapter.EnqueueFromMQTT(context.Background(), 1, &orgID, 0x1234, []byte("test"), false)
	require.NoError(t, err)

	call := mock.lastCall()
	assert.Greater(t, call.Request.QueId, uint64(0))
}

func TestMQTTDownlinkAdapter_GeneratesUniqueQueIDsAcrossRapidCalls(t *testing.T) {
	t.Parallel()

	mock := &mockDownlinkQueuer{returnID: 1}
	adapter := NewMQTTDownlinkAdapter(mock)
	orgID := uuid.New()

	const calls = 64
	for i := 0; i < calls; i++ {
		_, err := adapter.EnqueueFromMQTT(context.Background(), 1, &orgID, 0x1234, []byte("test"), false)
		require.NoError(t, err)
	}

	ids := mock.allQueueIDs()
	require.Len(t, ids, calls)
	seen := make(map[uint64]struct{}, calls)
	for _, id := range ids {
		assert.Greater(t, id, uint64(0))
		_, exists := seen[id]
		assert.False(t, exists, "duplicate queue ID generated: %d", id)
		seen[id] = struct{}{}
	}
}

func TestMQTTDownlinkAdapter_WrapsPayloadAsSliceOfByteSlices(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 1}
	adapter := NewMQTTDownlinkAdapter(mock)

	orgID := uuid.New()
	payload := []byte("hello world")
	_, err := adapter.EnqueueFromMQTT(context.Background(), 1, &orgID, 0x1234, payload, false)
	require.NoError(t, err)

	call := mock.lastCall()
	require.Len(t, call.Request.UserData, 1)
	assert.Equal(t, payload, call.Request.UserData[0])
}

func TestMQTTDownlinkAdapter_ConfirmedTrue_SetsResponseExpTrue(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 1}
	adapter := NewMQTTDownlinkAdapter(mock)

	orgID := uuid.New()
	_, err := adapter.EnqueueFromMQTT(context.Background(), 1, &orgID, 0x1234, []byte("test"), true)
	require.NoError(t, err)

	call := mock.lastCall()
	require.NotNil(t, call.Request.ResponseExp)
	assert.True(t, *call.Request.ResponseExp)
}

func TestMQTTDownlinkAdapter_ConfirmedFalse_SetsResponseExpFalse(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 1}
	adapter := NewMQTTDownlinkAdapter(mock)

	orgID := uuid.New()
	_, err := adapter.EnqueueFromMQTT(context.Background(), 1, &orgID, 0x1234, []byte("test"), false)
	require.NoError(t, err)

	call := mock.lastCall()
	require.NotNil(t, call.Request.ResponseExp)
	assert.False(t, *call.Request.ResponseExp)
}

func TestMQTTDownlinkAdapter_ReturnsResultQueID(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 99999}
	adapter := NewMQTTDownlinkAdapter(mock)

	orgID := uuid.New()
	queID, err := adapter.EnqueueFromMQTT(context.Background(), 1, &orgID, 0x1234, []byte("test"), false)
	require.NoError(t, err)
	assert.Equal(t, uint64(99999), queID)
}

func TestMQTTDownlinkAdapter_PropagatesQueueError(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnErr: fmt.Errorf("queue full")}
	adapter := NewMQTTDownlinkAdapter(mock)

	orgID := uuid.New()
	queID, err := adapter.EnqueueFromMQTT(context.Background(), 1, &orgID, 0x1234, []byte("test"), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue full")
	assert.Equal(t, uint64(0), queID)
}

func TestMQTTDownlinkAdapter_PassesCorrectEpEUI(t *testing.T) {
	t.Parallel()
	mock := &mockDownlinkQueuer{returnID: 1}
	adapter := NewMQTTDownlinkAdapter(mock)

	orgID := uuid.New()
	epEUI := uint64(0x70B3D59CD00009E6)
	_, err := adapter.EnqueueFromMQTT(context.Background(), 42, &orgID, epEUI, []byte("test"), false)
	require.NoError(t, err)

	call := mock.lastCall()
	assert.Equal(t, epEUI, call.Request.EpEui)
	assert.Equal(t, int64(42), call.TenantID)
	assert.Equal(t, &orgID, call.OrgID)
}

// --- scaciDownlinkQueuer wrapper tests ---

// mockSCACIDownlinkServer implements SCACIDownlinkServer for wrapper tests.
type mockSCACIDownlinkServer struct {
	returnResult *scaci.DLDataQueueResult
	returnErr    error
}

func (m *mockSCACIDownlinkServer) QueueDownlinkInternal(_ context.Context, _ int64, _ *uuid.UUID, _ *mioty.DLDataQueue) (*scaci.DLDataQueueResult, error) {
	return m.returnResult, m.returnErr
}

var _ SCACIDownlinkServer = (*mockSCACIDownlinkServer)(nil)

func TestSCACIDownlinkQueuer_SuccessReturnsQueID(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnResult: &scaci.DLDataQueueResult{QueID: 42},
	}
	queuer := NewSCACIDownlinkQueuer(mock)
	id, err := queuer.QueueDownlink(context.Background(), 1, nil, &mioty.DLDataQueue{})
	require.NoError(t, err)
	assert.Equal(t, uint64(42), id)
}

func TestSCACIDownlinkQueuer_ErrorPropagates(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnErr: fmt.Errorf("scaci unavailable"),
	}
	queuer := NewSCACIDownlinkQueuer(mock)
	id, err := queuer.QueueDownlink(context.Background(), 1, nil, &mioty.DLDataQueue{})
	assert.Error(t, err)
	assert.Equal(t, uint64(0), id)
}

func TestSCACIDownlinkQueuer_NilResultReturnsError(t *testing.T) {
	t.Parallel()
	mock := &mockSCACIDownlinkServer{
		returnResult: nil,
		returnErr:    nil,
	}
	queuer := NewSCACIDownlinkQueuer(mock)
	id, err := queuer.QueueDownlink(context.Background(), 1, nil, &mioty.DLDataQueue{})
	assert.Error(t, err)
	var catErr *bssci.CatalogError
	require.ErrorAs(t, err, &catErr)
	assert.Equal(t, bssci.ErrDLQueueNilResult, catErr.Token)
	assert.Equal(t, bssci.POSIX_EIO, catErr.Posix)
	assert.Equal(t, uint64(0), id)
}
