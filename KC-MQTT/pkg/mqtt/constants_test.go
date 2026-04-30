package mqtt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceEventTopic_UplinkFormat(t *testing.T) {
	t.Parallel()
	topic := DeviceEventTopic("mioty", "550e8400-e29b-41d4-a716-446655440000", "0123456789abcdef", DeviceEventUp)
	assert.Equal(t, "mioty/550e8400-e29b-41d4-a716-446655440000/device/0123456789abcdef/event/up", topic)
}

func TestDeviceEventTopic_AttachFormat(t *testing.T) {
	t.Parallel()
	topic := DeviceEventTopic("mioty", "550e8400-e29b-41d4-a716-446655440000", "0123456789abcdef", DeviceEventAttach)
	assert.Equal(t, "mioty/550e8400-e29b-41d4-a716-446655440000/device/0123456789abcdef/event/attach", topic)
}

func TestDeviceEventTopic_DetachFormat(t *testing.T) {
	t.Parallel()
	topic := DeviceEventTopic("mioty", "550e8400-e29b-41d4-a716-446655440000", "0123456789abcdef", DeviceEventDetach)
	assert.Equal(t, "mioty/550e8400-e29b-41d4-a716-446655440000/device/0123456789abcdef/event/detach", topic)
}

func TestDeviceEventTopic_DownlinkResultFormat(t *testing.T) {
	t.Parallel()
	topic := DeviceEventTopic("mioty", "550e8400-e29b-41d4-a716-446655440000", "0123456789abcdef", DeviceEventDownlinkResult)
	assert.Equal(t, "mioty/550e8400-e29b-41d4-a716-446655440000/device/0123456789abcdef/event/downlink_result", topic)
}

func TestDeviceCommandDownWildcardTopic_Format(t *testing.T) {
	t.Parallel()
	topic := DeviceCommandDownWildcardTopic("mioty")
	assert.Equal(t, "mioty/+/device/+/command/down", topic)
}

func TestDeviceEventTopic_CustomPrefix(t *testing.T) {
	t.Parallel()
	topic := DeviceEventTopic("custom-prefix", "org-uuid", "deadbeef01234567", DeviceEventUp)
	assert.Equal(t, "custom-prefix/org-uuid/device/deadbeef01234567/event/up", topic)
}

func TestDeviceCommandDownWildcardTopic_CustomPrefix(t *testing.T) {
	t.Parallel()
	topic := DeviceCommandDownWildcardTopic("custom-prefix")
	assert.Equal(t, "custom-prefix/+/device/+/command/down", topic)
}
