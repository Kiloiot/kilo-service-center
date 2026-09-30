package bssciservices

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// errTestBrokerDown is the fixture publish error for the MQTT adapter tests.
var errTestBrokerDown = errors.New("broker down")

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

	err := adapter.PublishUplink(testutil.TestContext(), "org-uuid-123", &mioty.ULDataMessage{
		EpEui: 0x70B3D59CD00009E6, BsEui: 0xABCDEF1234567890,
		RSSI: -80.5, SNR: 15.2, RxTime: 1700000000000000000, PacketCnt: 42, UserData: []byte("hello"),
	})

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

func TestMQTTAdapter_PublishUplink_CarriesTheDownlinkWindowFlags(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishUplink(testutil.TestContext(), "org-uuid-123", &mioty.ULDataMessage{
		EpEui: 0x70B3D59CD00009E6, BsEui: 0xABCDEF1234567890, PacketCnt: 43,
		DlOpen: true, ResponseExp: true, DlAck: false,
	})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(mock.lastCall().Payload, &payload))
	assert.Equal(t, true, payload["dlOpen"], "a subscriber learns the endpoint opened a downlink window")
	assert.Equal(t, true, payload["responseExp"])
	assert.Equal(t, false, payload["dlAck"])
	assert.Equal(t, float64(43), payload["cnt"], "the published fields stay unchanged")
}

func TestMQTTAdapter_PublishUplink_ZeroPaddedEUI(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishUplink(testutil.TestContext(), "org", &mioty.ULDataMessage{EpEui: 0x00000001, BsEui: 0x00000002})

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

	heardBy := uint64(0xABCDEF1234567890)
	err := adapter.PublishAttach(testutil.TestContext(), "org-uuid",
		0x70B3D59CD00009E6, &heardBy)

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

// An attach or detach decided in the service center was heard by no base
// station, so its event names none.
func TestMQTTAdapter_ServiceCenterDecisionNamesNoStation(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	for _, publish := range []func(context.Context, string, uint64, *uint64) error{adapter.PublishAttach, adapter.PublishDetach} {
		require.NoError(t, publish(testutil.TestContext(), "org-uuid", 0x70B3D59CD00009E6, nil))
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(mock.lastCall().Payload, &payload))
		assert.Equal(t, "70b3d59cd00009e6", payload["epEui"])
		assert.NotContains(t, payload, "bsEui")
	}
}

func TestMQTTAdapter_PublishDetach_CorrectPayload(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	heardBy := uint64(0xABCDEF1234567890)
	err := adapter.PublishDetach(testutil.TestContext(), "org-uuid",
		0x70B3D59CD00009E6, &heardBy)

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

	err := adapter.PublishDownlinkResult(testutil.TestContext(), "org-uuid", &mioty.DLDataResult{
		EpEui: 0x70B3D59CD00009E6, QueId: 12345, Result: "success",
	})

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

func TestMQTTAdapter_PublishDownlinkResult_SentCarriesTheTransmission(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)
	txTime := int64(1790507999020943879)
	packetCnt := uint32(44)
	bsEui := uint64(0x70B3D59CD00009E6)

	err := adapter.PublishDownlinkResult(testutil.TestContext(), "org-uuid", &mioty.DLDataResult{
		EpEui: 0x70B3D5677011150A, QueId: 9007199254740991, Result: mioty.ResultSent,
		TxTime: &txTime, PacketCnt: &packetCnt, BsEui: &bsEui,
	})
	require.NoError(t, err)

	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(mock.lastCall().Payload, &payload))
	assert.JSONEq(t, `"70b3d59cd00009e6"`, string(payload["bsEui"]))
	assert.Equal(t, "1790507999020943879", string(payload["txTime"]))
	assert.Equal(t, "44", string(payload["packetCnt"]))
	assert.Equal(t, "9007199254740991", string(payload["queId"]))
}

func TestMQTTAdapter_PublishDownlinkResult_NotSentOmitsTheTransmission(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishDownlinkResult(testutil.TestContext(), "org-uuid", &mioty.DLDataResult{
		EpEui: 0x70B3D5677011150A, QueId: 7, Result: mioty.DLDataResultExpired,
	})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(mock.lastCall().Payload, &payload))
	assert.Equal(t, mioty.DLDataResultExpired, payload["result"])
	for _, key := range []string{"bsEui", "txTime", "packetCnt"} {
		assert.NotContains(t, payload, key, "only a sent result reports its transmission")
	}
}

func TestMQTTAdapter_PropagatesPublishError(t *testing.T) {
	t.Parallel()
	mock := &mockDeviceEventPublisher{returnErr: errTestBrokerDown}
	adapter := NewMQTTAdapter(mock)

	err := adapter.PublishUplink(testutil.TestContext(), "org", &mioty.ULDataMessage{EpEui: 1, BsEui: 2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker down")
}
