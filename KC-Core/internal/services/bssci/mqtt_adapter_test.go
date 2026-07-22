package bssciservices

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// mockDeviceEventPublisher records PublishDeviceEvent calls.
type mockDeviceEventPublisher struct {
	mu        sync.Mutex
	calls     []deviceEventCall
	returnErr error
}

type deviceEventCall struct {
	OrgUUID   string
	EpEUIHex  string
	EventType string
	Payload   []byte
}

func (m *mockDeviceEventPublisher) PublishDeviceEvent(_ context.Context, orgUUID string, epEUIHex string, eventType string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, deviceEventCall{
		OrgUUID:   orgUUID,
		EpEUIHex:  epEUIHex,
		EventType: eventType,
		Payload:   payload,
	})
	return m.returnErr
}

func (m *mockDeviceEventPublisher) lastCall() deviceEventCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

func (m *mockDeviceEventPublisher) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// Compile-time interface assertion
var _ mqtt.DeviceEventPublisher = (*mockDeviceEventPublisher)(nil)

func TestMQTTAdapter_PublishUplink_CorrectPayloadAndTopic(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishUplink(testutil.TestContext(), "org-uuid-123",
		0x70B3D59CD00009E6, 0xABCDEF1234567890,
		-80.5, 15.2, 1700000000000000000, 42, []byte("hello"), nil)

	require.NoError(t, err)
	require.Equal(t, 1, mock.callCount())

	call := mock.lastCall()
	assert.Equal(t, "org-uuid-123", call.OrgUUID)
	assert.Equal(t, "70b3d59cd00009e6", call.EpEUIHex)
	assert.Equal(t, mqtt.DeviceEventUp, call.EventType)

	// Verify JSON payload
	var payload map[string]interface{}
	err = json.Unmarshal(call.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, "abcdef1234567890", payload["bsEui"])
	assert.InDelta(t, -80.5, payload["rssi"], 0.01)
	assert.InDelta(t, 15.2, payload["snr"], 0.01)
	assert.Equal(t, float64(42), payload["cnt"])
}

func TestMQTTAdapter_PublishUplink_ZeroPaddedEUI(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishUplink(testutil.TestContext(), "org", 0x00000001, 0x00000002,
		0, 0, 0, 0, nil, nil)

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, "0000000000000001", call.EpEUIHex)

	var payload map[string]interface{}
	err = json.Unmarshal(call.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, "0000000000000002", payload["bsEui"])
}

func TestMQTTAdapter_PublishAttach_CorrectPayload(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishAttach(testutil.TestContext(), "org-uuid",
		0x70B3D59CD00009E6, 0xABCDEF1234567890)

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, "org-uuid", call.OrgUUID)
	assert.Equal(t, "70b3d59cd00009e6", call.EpEUIHex)
	assert.Equal(t, mqtt.DeviceEventAttach, call.EventType)

	var payload map[string]interface{}
	err = json.Unmarshal(call.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, "70b3d59cd00009e6", payload["epEui"])
	assert.Equal(t, "abcdef1234567890", payload["bsEui"])
	assert.Equal(t, "attach", payload["event"])
}

func TestMQTTAdapter_PublishDetach_CorrectPayload(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishDetach(testutil.TestContext(), "org-uuid",
		0x70B3D59CD00009E6, 0xABCDEF1234567890)

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, mqtt.DeviceEventDetach, call.EventType)

	var payload map[string]interface{}
	err = json.Unmarshal(call.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, "70b3d59cd00009e6", payload["epEui"])
	assert.Equal(t, "abcdef1234567890", payload["bsEui"])
	assert.Equal(t, "detach", payload["event"])
}

func TestMQTTAdapter_PublishDownlinkResult_CorrectPayload(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishDownlinkResult(testutil.TestContext(), "org-uuid",
		0x70B3D59CD00009E6, 12345, "success")

	require.NoError(t, err)
	call := mock.lastCall()
	assert.Equal(t, mqtt.DeviceEventDownlinkResult, call.EventType)

	var payload map[string]interface{}
	err = json.Unmarshal(call.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, "70b3d59cd00009e6", payload["epEui"])
	assert.Equal(t, float64(12345), payload["queId"])
	assert.Equal(t, "success", payload["result"])
}

func TestMQTTAdapter_PropagatesPublishError(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{returnErr: fmt.Errorf("broker down")}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishUplink(testutil.TestContext(), "org", 1, 2, 0, 0, 0, 0, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker down")
}
