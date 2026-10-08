package builders

import (
	"context"
	"slices"
	"testing"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// discardSCACI and discardEvents stand in for the SCACI broadcaster and the
// event store the delivery worker is wired with.
type (
	discardSCACI  struct{}
	discardEvents struct{}
)

func (discardSCACI) BroadcastULData(context.Context, int64, *mioty.ULDataMessage) error { return nil }

func (discardEvents) CreateEvent(context.Context, *models.SystemEvent) error { return nil }

// discardMQTT stands in for the MQTT publisher of a process that publishes.
type discardMQTT struct{ deliveryMQTT }

func deliveryTestInfrastructure(mqttEnabled bool) *Infrastructure {
	return &Infrastructure{
		Config: &pkgconfig.Config{
			MQTT: pkgconfig.MQTTConfig{Enabled: mqttEnabled},
			Protocol: pkgconfig.ProtocolConfig{Delivery: pkgconfig.DeliveryConfig{
				PollInterval: pkgconfig.DefaultProtocolDeliveryPollInterval,
				BatchSize:    pkgconfig.DefaultProtocolDeliveryBatchSize,
				RetryBackoff: pkgconfig.DefaultProtocolDeliveryRetryBackoff,
				MaxBackoff:   pkgconfig.DefaultProtocolDeliveryMaxBackoff,
			}},
		},
		Repos:       &postgres.Repositories{},
		Clock:       clock.SystemClock{},
		LoggerIface: logger.NewNop(),
	}
}

// TestBuildDeliveryWorker_RefusesAQueuedChannelWithoutASender covers the
// mismatch that parked every MQTT row: uplinks are queued for MQTT whenever
// the deployment enables it, so a worker without an MQTT publisher must not
// start.
func TestBuildDeliveryWorker_RefusesAQueuedChannelWithoutASender(t *testing.T) {
	infra := deliveryTestInfrastructure(testMQTTEnabled)
	if _, err := buildDeliveryWorker(infra, drainedChannels(infra), discardSCACI{}, nil, discardEvents{}); err == nil {
		t.Fatal("a worker for queued MQTT rows was built without an MQTT publisher")
	}
}

func TestBuildDeliveryWorker_DrainsEveryQueuedChannel(t *testing.T) {
	for _, mqttEnabled := range []bool{testMQTTEnabled, !testMQTTEnabled} {
		infra := deliveryTestInfrastructure(mqttEnabled)
		if _, err := buildDeliveryWorker(infra, drainedChannels(infra), discardSCACI{}, discardMQTT{}, discardEvents{}); err != nil {
			t.Fatalf("MQTT enabled=%v: %v", mqttEnabled, err)
		}
	}
}

// TestEndpointAckChannels_FollowConfigurationNotTheLocalClient covers the
// federation ingress, which stores acknowledgements without an MQTT client:
// they are queued for MQTT whenever the deployment enables it, never for an
// Application Center, and the worker drains them with the uplinks.
func TestEndpointAckChannels_FollowConfigurationNotTheLocalClient(t *testing.T) {
	enabled := deliveryTestInfrastructure(testMQTTEnabled)
	disabled := deliveryTestInfrastructure(!testMQTTEnabled)

	if got := endpointAckChannels(enabled); len(got) != 1 || got[0] != models.DeliveryChannelMQTTDownlinkAck {
		t.Fatalf("MQTT enabled without a local client: got %v", got)
	}
	if got := endpointAckChannels(disabled); len(got) != 0 {
		t.Fatalf("MQTT disabled: got %v", got)
	}
	want := []models.DeliveryChannel{models.DeliveryChannelSCACI, models.DeliveryChannelMQTT, models.DeliveryChannelMQTTDownlinkAck}
	if got := drainedChannels(enabled); !slices.Equal(got, want) {
		t.Fatalf("drained channels: got %v, want %v", got, want)
	}
}
