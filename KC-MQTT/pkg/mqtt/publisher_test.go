package mqtt

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPublisher records Publish calls for verification.
type mockPublisher struct {
	mu        sync.Mutex
	calls     []publishCall
	returnErr error
}

type publishCall struct {
	Topic    string
	QoS      byte
	Retained bool
	Payload  interface{}
}

func (m *mockPublisher) Connect(_ context.Context) error { return nil }
func (m *mockPublisher) Disconnect(_ context.Context)    {}
func (m *mockPublisher) IsConnected() bool               { return true }
func (m *mockPublisher) Subscribe(_ context.Context, _ string, _ byte, _ MessageHandler) error {
	return nil
}
func (m *mockPublisher) Unsubscribe(_ context.Context, _ ...string) error { return nil }

func (m *mockPublisher) Publish(_ context.Context, topic string, qos byte, retained bool, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, publishCall{Topic: topic, QoS: qos, Retained: retained, Payload: payload})
	return m.returnErr
}

func (m *mockPublisher) lastCall() publishCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

func TestPublishDeviceEvent_RejectsEmptyOrgUUID(t *testing.T) {
	t.Parallel()
	pub := &TopicPublisher{client: &mockPublisher{}, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "", "0123456789abcdef", DeviceEventUp, []byte("test"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrDeviceEventEmptyOrgUUID)
}

func TestPublishDeviceEvent_RejectsEmptyEUIHex(t *testing.T) {
	t.Parallel()
	pub := &TopicPublisher{client: &mockPublisher{}, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "org-uuid", "", DeviceEventUp, []byte("test"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrDeviceEventEmptyEUIHex)
}

func TestPublishDeviceEvent_UplinkUsesUplinkQoS(t *testing.T) {
	t.Parallel()
	mock := &mockPublisher{}
	pub := &TopicPublisher{client: mock, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "org-uuid", "0123456789abcdef", DeviceEventUp, []byte("test"))

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, byte(UplinkQoS), call.QoS)
	assert.Equal(t, "mioty/org-uuid/device/0123456789abcdef/event/up", call.Topic)
}

func TestPublishDeviceEvent_AttachUsesEventsQoS(t *testing.T) {
	t.Parallel()
	mock := &mockPublisher{}
	pub := &TopicPublisher{client: mock, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "org-uuid", "0123456789abcdef", DeviceEventAttach, []byte("test"))

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, byte(EventsQoS), call.QoS)
	assert.Equal(t, "mioty/org-uuid/device/0123456789abcdef/event/attach", call.Topic)
}

func TestPublishDeviceEvent_DetachUsesEventsQoS(t *testing.T) {
	t.Parallel()
	mock := &mockPublisher{}
	pub := &TopicPublisher{client: mock, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "org-uuid", "0123456789abcdef", DeviceEventDetach, []byte("test"))

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, byte(EventsQoS), call.QoS)
	assert.Contains(t, call.Topic, "/event/detach")
}

func TestPublishDeviceEvent_DownlinkResultUsesEventsQoS(t *testing.T) {
	t.Parallel()
	mock := &mockPublisher{}
	pub := &TopicPublisher{client: mock, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "org-uuid", "0123456789abcdef", DeviceEventDownlinkResult, []byte("test"))

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, byte(EventsQoS), call.QoS)
	assert.Contains(t, call.Topic, "/event/downlink_result")
}

func TestPublishDeviceEvent_PropagatesPublishError(t *testing.T) {
	t.Parallel()
	mock := &mockPublisher{returnErr: fmt.Errorf("broker unavailable")}
	pub := &TopicPublisher{client: mock, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "org-uuid", "0123456789abcdef", DeviceEventUp, []byte("test"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker unavailable")
}

func TestPublishDeviceEvent_NotRetained(t *testing.T) {
	t.Parallel()
	mock := &mockPublisher{}
	pub := &TopicPublisher{client: mock, prefix: "mioty"}

	err := pub.PublishDeviceEvent(context.Background(), "org-uuid", "0123456789abcdef", DeviceEventUp, []byte("test"))

	require.NoError(t, err)
	call := mock.lastCall()
	assert.False(t, call.Retained)
}
