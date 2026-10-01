package builders

import (
	"fmt"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
)

// attachmentDecisions is the one place an endpoint's attachment is decided
// and the owner's subscribers are told of it.
type attachmentDecisions struct {
	decider  bssci.AttachmentDecider
	notifier bssciservices.EndpointStatusNotifier
}

// buildAttachmentDecisions announces every decision to the owner's
// application centers as an epStat and, when MQTT is configured, as an MQTT
// event; the announcements run on the process's background work, which the
// composition root stops at shutdown.
func buildAttachmentDecisions(
	infra *Infrastructure,
	bssciInfra *BSSCIInfrastructure,
	epStatBroadcaster bssciservices.EPStatusBroadcaster,
	mqttEvents bssciservices.AttachmentEventPublisher,
	runner bssciservices.BackgroundRunner,
) (*attachmentDecisions, error) {
	epStat, err := bssciservices.NewEPStatNotifier(epStatBroadcaster, infra.LoggerIface)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}
	notifiers := []bssciservices.EndpointStatusNotifier{epStat}
	if mqttEvents != nil {
		mqttNotifier, err := bssciservices.NewMQTTAttachmentNotifier(mqttEvents, bssciInfra.OrgResolver, infra.LoggerIface)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
		}
		notifiers = append(notifiers, mqttNotifier)
	}
	notifier, err := bssciservices.NewEndpointStatusFanout(runner, notifiers...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}
	decider, err := bssciservices.NewEndpointAttachmentDecider(bssciInfra.EndpointRepo, notifier)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}
	return &attachmentDecisions{decider: decider, notifier: notifier}, nil
}
