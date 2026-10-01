package bssciservices

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// EPStatusBroadcaster sends an epStat to the tenant's application centers
// (SCACI §3.13).
type EPStatusBroadcaster interface {
	BroadcastEPStatus(ctx context.Context, tenantID int64, data *bssci.EPStatusData) error
}

// AttachmentEventPublisher publishes an endpoint's attach and detach events
// to its organization's MQTT subscribers; heardBy is the base station of an
// over-the-air attach or detach, nil for a decision taken in the service
// center.
type AttachmentEventPublisher interface {
	PublishAttach(ctx context.Context, orgUUID string, epEUI uint64, heardBy *uint64) error
	PublishDetach(ctx context.Context, orgUUID string, epEUI uint64, heardBy *uint64) error
}

// OwnerOrganizations names the organization an owner tenant's MQTT topics
// belong to.
type OwnerOrganizations interface {
	GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error)
}

type endpointStatusFanout struct {
	runner    BackgroundRunner
	notifiers []EndpointStatusNotifier
}

// NewEndpointStatusFanout tells every notifier about each decision in the
// background, scoped to the endpoint owner and detached from the request
// that took the decision.
func NewEndpointStatusFanout(runner BackgroundRunner, notifiers ...EndpointStatusNotifier) (EndpointStatusNotifier, error) {
	if runner == nil {
		return nil, errNilBackgroundRunner
	}
	for _, notifier := range notifiers {
		if notifier == nil {
			return nil, errNilEndpointStatusNotifier
		}
	}
	return &endpointStatusFanout{runner: runner, notifiers: notifiers}, nil
}

func (f *endpointStatusFanout) NotifyEndpointStatus(ctx context.Context, notice EndpointStatusNotice) {
	ownerCtx := pkgcontext.WithTenantID(context.WithoutCancel(ctx), notice.TenantID)
	for _, notifier := range f.notifiers {
		f.runner.Go(ownerCtx, func(ctx context.Context) {
			notifier.NotifyEndpointStatus(ctx, notice)
		})
	}
}

type epStatNotifier struct {
	broadcaster EPStatusBroadcaster
	logger      logger.Logger
}

// NewEPStatNotifier sends each decision to the owner's application centers
// as an epStat (SCACI §3.13).
func NewEPStatNotifier(broadcaster EPStatusBroadcaster, log logger.Logger) (EndpointStatusNotifier, error) {
	if broadcaster == nil {
		return nil, errNilEPStatusBroadcaster
	}
	if log == nil {
		return nil, errNilEndpointStatusLogger
	}
	return &epStatNotifier{broadcaster: broadcaster, logger: log}, nil
}

func (n *epStatNotifier) NotifyEndpointStatus(ctx context.Context, notice EndpointStatusNotice) {
	if err := n.broadcaster.BroadcastEPStatus(ctx, notice.TenantID, notice.Status); err != nil {
		n.logger.WarnContext(ctx, LogEndpointStatusEPStatNotDelivered,
			logger.FieldEpEui, mioty.FormatEUI64(notice.Status.EpEui),
			logger.FieldError, err)
	}
}

type mqttAttachmentNotifier struct {
	publisher AttachmentEventPublisher
	orgs      OwnerOrganizations
	logger    logger.Logger
}

// NewMQTTAttachmentNotifier publishes each decision as the endpoint's MQTT
// event/attach or event/detach under its owner's organization.
func NewMQTTAttachmentNotifier(publisher AttachmentEventPublisher, orgs OwnerOrganizations, log logger.Logger) (EndpointStatusNotifier, error) {
	if publisher == nil {
		return nil, errNilAttachmentEventPublisher
	}
	if orgs == nil {
		return nil, errNilOwnerOrganizations
	}
	if log == nil {
		return nil, errNilEndpointStatusLogger
	}
	return &mqttAttachmentNotifier{publisher: publisher, orgs: orgs, logger: log}, nil
}

func (n *mqttAttachmentNotifier) NotifyEndpointStatus(ctx context.Context, notice EndpointStatusNotice) {
	epEUI := notice.Status.EpEui
	org, err := n.orgs.GetDefaultOrgForTenant(ctx, notice.TenantID)
	if err != nil || org == uuid.Nil {
		n.logger.WarnContext(ctx, LogEndpointStatusMQTTOrgUnresolved,
			logger.FieldEpEui, mioty.FormatEUI64(epEUI),
			logger.FieldTenantID, notice.TenantID,
			logger.FieldError, err)
		return
	}
	publish := n.publisher.PublishDetach
	if notice.Status.EpStatus == bssci.EndpointStatusAttached {
		publish = n.publisher.PublishAttach
	}
	if err := publish(pkgcontext.WithOrganizationID(ctx, org), org.String(), epEUI, notice.HeardBy); err != nil {
		n.logger.WarnContext(ctx, LogEndpointStatusMQTTNotPublished,
			logger.FieldEpEui, mioty.FormatEUI64(epEUI),
			logger.FieldError, err)
	}
}
