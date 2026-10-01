package bssciservices

// Log messages owned by the BSSCI domain services. Message text is stable so
// operators can grep the log stream by exact string; protocol-level messages
// live in the KC-Core/pkg/bssci catalog and are reused directly.
const (
	LogInvalidNetworkKeyBypassedValidation = "invalid network key bypassed synchronous validation"
	LogFailedToLogFailureEvent             = "failed to log failure event"
	LogFailedToLogStartEvent               = "failed to log start event"
	// LogDownlinkResultOwnerUnparsable is logged when a downlink row names an owner tenant that is not a tenant id.
	LogDownlinkResultOwnerUnparsable    = "downlink result not delivered: the owner tenant of the downlink is not a tenant id"
	LogEndpointStatusEPStatNotDelivered = "epStat for an attachment decision was not delivered to every application center"
	LogEndpointStatusMQTTOrgUnresolved  = "MQTT event for an attachment decision skipped: organization unresolved"
	LogEndpointStatusMQTTNotPublished   = "MQTT event for an attachment decision was not published"
	// LogDownlinkEventPayloadUnread is logged when a downlink event is recorded without the queued user data.
	LogDownlinkEventPayloadUnread = "downlink event recorded without its user data: the queue row could not be read"
	// LogDownlinkRequeueEventFailed is logged when a downlink returned to the queue could not be announced.
	LogDownlinkRequeueEventFailed = "downlink returned to the queue without its event: the event could not be recorded"
	// LogExpiredDownlinkUnreported is logged when a downlink its station dropped is expired but cannot be identified to its originators.
	LogExpiredDownlinkUnreported = "downlink expired at its base station but not reported: its endpoint or queue id cannot be parsed"
	// LogDeletedStationDownlinksUnsettled is logged when the downlinks a deleted base station held cannot be ended at its deletion.
	LogDeletedStationDownlinksUnsettled = "downlinks of a deleted base station not ended at its deletion: the expiry sweep ends them"
	// LogDeletedStationDownlinksExpired is logged when the downlinks a deleted base station held end expired.
	LogDeletedStationDownlinksExpired = "downlinks a deleted base station held ended expired"
	// LogSessionRowRemovedWithStation is logged when a closing session finds its row deleted with its base station.
	LogSessionRowRemovedWithStation = "session row already removed with its base station: nothing to persist"
	// LogSentAfterExpiryEventFailed is logged when a "sent" contradicting a reported expiry cannot be filed in the events.
	LogSentAfterExpiryEventFailed = "sent result for an expired downlink not filed in the events"
)

// Target base-station descriptions used in attachment operation events.
const (
	descNoBaseStations       = "no base stations available"
	descFmtSingleBaseStation = "base station %s"
	descFmtBaseStationCount  = "%d base stations"
)

// Downlink lifecycle event descriptions recorded by the audit logger; each
// names the queue id, the endpoint, the base station and the user data.
const (
	descriptionDLDataQueuedFormat          = "Downlink %d for endpoint %s queued at base station %s, payload %s"
	descriptionDLDataSentFormat            = "Downlink %d for endpoint %s sent by base station %s after uplink packet counter %d, payload %s"
	descriptionDLDataSentAfterExpiryFormat = "Downlink %d for endpoint %s reported sent by base station %s after uplink packet counter %d, after it was reported expired; the expiry stands, payload %s"
	descriptionDLDataExpiredFormat         = "Downlink %d for endpoint %s expired at base station %s, payload %s"
	descriptionDLDataInvalidFormat         = "Downlink %d for endpoint %s refused as invalid by base station %s, payload %s"
	descriptionDLDataUnknownResultFormat   = "Base station %s reported result '%s' for downlink %d of endpoint %s, payload %s"
	descriptionDLDataExpiredInQueueFormat  = "Downlink %d for endpoint %s expired in the service center queue before any downlink window, payload %s"
	descriptionDLDataRevokedFormat         = "Downlink %d for endpoint %s revoked at base station %s, payload %s"
	descriptionDLDataAcknowledgedFormat    = "Endpoint %s acknowledged downlink %d sent after uplink packet counter %d, payload %s"
	descriptionDLDataRevokedInQueueFormat  = "Downlink %d for endpoint %s revoked in the service center queue before any base station held it, payload %s"
	descriptionDLDataEnqueuedFormat        = "Downlink %d for endpoint %s queued at the service center, payload %s"
	descriptionDLDataUpdatedFormat         = "Pending downlink %d for endpoint %s updated in the service center queue, payload %s"
	descriptionDLDataRequeuedFormat        = "Downlink %d for endpoint %s returned to the service center queue: base station %s no longer holds it, payload %s"
	errWrapMarshalAuditDetails             = "marshal audit event details"
	// payloadUnavailable stands in for user data the queue could not be read for.
	payloadUnavailable = "unavailable"
	// payloadEmpty names a downlink without user data.
	payloadEmpty = "empty"
	// payloadAtCounterFormat renders one counter-dependent entry as hex@packetCnt.
	payloadAtCounterFormat = "%s@%d"
	payloadEntrySeparator  = ", "
)
