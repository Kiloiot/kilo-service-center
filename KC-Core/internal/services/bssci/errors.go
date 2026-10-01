package bssciservices

import "errors"

// Sentinel errors for the BSSCI domain services. Failure paths wrap the
// underlying cause with fmt.Errorf("%w: %w", sentinel, err) so callers keep
// errors.Is matching on both the sentinel and the cause while the rendered
// message text stays unchanged.
var (
	// errInvalidEndpointEUI reports an endpoint EUI string that failed parsing.
	errInvalidEndpointEUI = errors.New("invalid endpoint EUI")
	// errInvalidOwnerTenant reports a dlDataQue answer whose downlink owner is not a tenant id.
	errInvalidOwnerTenant = errors.New("invalid owner tenant of a queued downlink")
	// errRecordDownlinkHolder reports a failed record of the base station holding a downlink.
	errRecordDownlinkHolder = errors.New("failed to record the base station holding a downlink")
	// errConfirmDownlinkQueued reports a failed reserved-to-queued confirmation of a downlink.
	errConfirmDownlinkQueued = errors.New("failed to confirm a downlink queued")
	// errMarkEndpointAck reports a failed record of an endpoint's downlink acknowledgement.
	errMarkEndpointAck = errors.New("failed to record the endpoint acknowledgement of a downlink")
	// errUnidentifiedDownlink reports a queue row whose endpoint cannot be parsed.
	errUnidentifiedDownlink = errors.New("downlink result reporter: unidentified downlink")
	// errFailRejectedDownlink reports a failed update of a downlink its base station rejected.
	errFailRejectedDownlink = errors.New("failed to fail a rejected downlink")
	// errGetEndpoint reports an endpoint repository lookup failure during attach/detach.
	errGetEndpoint = errors.New("endpoint attachment service: get endpoint")
	// errRecordAttachment reports a failed write of the endpoint's attach decision.
	errRecordAttachment = errors.New("endpoint attachment service: record attachment")
	// errRecordDetachment reports a failed write of the endpoint's detach decision.
	errRecordDetachment = errors.New("endpoint attachment service: record detachment")
	// errUpdateEndpoint reports a failed endpoint field update inside the propagate transaction.
	errUpdateEndpoint = errors.New("failed to update endpoint")
	// errLoadEndpointSession reports a failed active endpoint-session load.
	errLoadEndpointSession = errors.New("failed to load endpoint session")
	// errLoadEndpointProvisioning reports a failed endpoint provisioning load for its session key.
	errLoadEndpointProvisioning = errors.New("failed to load endpoint provisioning")

	// ErrNilEventStore rejects an audit logger built without a system event store.
	ErrNilEventStore = errors.New("audit logger: system event store is nil")
	// ErrNilClock rejects an audit logger built without a clock.
	ErrNilClock = errors.New("audit logger: clock is nil")
	// ErrNilAuditDownlinks rejects an audit logger built without the downlink queue lookup.
	ErrNilAuditDownlinks = errors.New("audit logger: downlink lookup is nil")
	// ErrNilAuditStations rejects an audit logger built without the base station directory.
	ErrNilAuditStations = errors.New("audit logger: base station directory is nil")
	// ErrNilAuditLogger rejects an audit logger built without a logger.
	ErrNilAuditLogger = errors.New("audit logger: logger is nil")
	// ErrNilQueuer rejects an MQTT downlink adapter built without a queuer.
	ErrNilQueuer = errors.New("mqtt downlink adapter: downlink queuer is nil")
	// ErrNilCommandRefs rejects an MQTT downlink adapter built without the lookup of command refs.
	ErrNilCommandRefs = errors.New("mqtt downlink adapter: command ref lookup is nil")
	// ErrNilSCACIServer rejects a downlink queuer built without a SCACI server.
	ErrNilSCACIServer = errors.New("scaci downlink queuer: server is nil")
	// ErrNilDispatcherLogger rejects a downlink dispatcher built without a logger.
	ErrNilDispatcherLogger = errors.New("downlink dispatcher: logger is nil")
	// ErrNilDownlinkReserver rejects a downlink dispatcher built without a reserver.
	ErrNilDownlinkReserver = errors.New("downlink dispatcher: reserver is nil")
	// ErrNilDownlinkConfirmer rejects a downlink dispatcher built without a queue confirmer.
	ErrNilDownlinkConfirmer = errors.New("downlink dispatcher: queue confirmer is nil")
	// ErrNilDownlinkWindowClaimer rejects a downlink dispatcher built without a downlink window claimer.
	ErrNilDownlinkWindowClaimer = errors.New("downlink dispatcher: downlink window claimer is nil")
	// ErrNilReservationReclaimer rejects a downlink reclaimer built without the queue it releases.
	ErrNilReservationReclaimer = errors.New("downlink reclaimer: reservation reclaimer is nil")
	// ErrNilRequeueRecorder rejects a downlink reclaimer built without the recorder of returned downlinks.
	ErrNilRequeueRecorder = errors.New("downlink reclaimer: requeue recorder is nil")
	// ErrNilReclaimerLogger rejects a downlink reclaimer built without a logger.
	ErrNilReclaimerLogger = errors.New("downlink reclaimer: logger is nil")
	// ErrNilDiscardedRevocations rejects a downlink reclaimer built without the store that expires discarded revocations.
	ErrNilDiscardedRevocations = errors.New("downlink reclaimer: discarded revocations store is nil")
	// ErrNilRemovedStationHolds rejects a downlink reclaimer built without the store that ends a deleted station's downlinks.
	ErrNilRemovedStationHolds = errors.New("downlink reclaimer: removed station store is nil")
	// ErrNilDiscardedExpiryReporter rejects a downlink reclaimer built without the reporter of discarded downlinks.
	ErrNilDiscardedExpiryReporter = errors.New("downlink reclaimer: expiry reporter is nil")
	// ErrNilReclaimerQueueTenants rejects a downlink reclaimer built without the queue owner cache it clears.
	ErrNilReclaimerQueueTenants = errors.New("downlink reclaimer: queue tenant cache is nil")
	// errExpireDiscardedRevocations wraps a failed expiry of the overdue downlinks a fresh session discarded.
	errExpireDiscardedRevocations = errors.New("expire the overdue downlinks a base station discarded")
	// errReclaimReservations wraps a failed release of a base station's orphaned reservations.
	errReclaimReservations = errors.New("reclaim base station downlink reservations")
	// errReclaimDiscardedQueue wraps a failed release of the downlinks a fresh session discarded.
	errReclaimDiscardedQueue = errors.New("reclaim downlinks a base station discarded")
	// errReclaimEndpointQueue wraps a failed release of the endpoint downlinks an attach propagate discarded.
	errReclaimEndpointQueue = errors.New("reclaim endpoint downlinks a base station discarded on attach propagate")
	// ErrNilDownlinkSendFunc rejects a downlink dispatcher built without a send function.
	ErrNilDownlinkSendFunc = errors.New("downlink dispatcher: send function is nil")
	// ErrNilDispatcherClock rejects a downlink dispatcher built without a clock.
	ErrNilDispatcherClock = errors.New("downlink dispatcher: clock is nil")
	// ErrNilDownlinkServiceLogger rejects a downlink service built without a logger.
	ErrNilDownlinkServiceLogger = errors.New("downlink service: logger is nil")
	// ErrNilTenantResolver rejects a downlink service built without a queue tenant resolver.
	ErrNilTenantResolver = errors.New("downlink service: tenant resolver is nil")
	// ErrNilDownlinkWriter rejects a downlink service built without a queue writer.
	ErrNilDownlinkWriter = errors.New("downlink service: downlink queue writer is nil")
	// ErrNilRevokeAnswerer rejects a downlink service built without its revoke answers.
	ErrNilRevokeAnswerer = errors.New("downlink service: revoke answers are nil")
	// ErrNilDownlinkHolderWriter rejects a downlink service built without its queue holder writer.
	ErrNilDownlinkHolderWriter = errors.New("downlink service: downlink holder writer is nil")
	// ErrNilDownlinkServiceClock rejects a downlink service built without a clock.
	ErrNilDownlinkServiceClock = errors.New("downlink service: clock is nil")
	// ErrNilQueueSerializer rejects a downlink service built without a queue serializer.
	ErrNilQueueSerializer = errors.New("downlink service: queue serializer is nil")
	// ErrNilStationResultReporter rejects a downlink service built without its result reporter.
	ErrNilStationResultReporter = errors.New("downlink service: result reporter is nil")
	// ErrNilRevokeAnswersLogger rejects revoke answers built without a logger.
	ErrNilRevokeAnswersLogger = errors.New("revoke answer service: logger is nil")
	// ErrNilRevokeAnswersTenants rejects revoke answers built without a queue tenant resolver.
	ErrNilRevokeAnswersTenants = errors.New("revoke answer service: tenant resolver is nil")
	// ErrNilRevokeAnswersRevocations rejects revoke answers built without the writer of revoke answers.
	ErrNilRevokeAnswersRevocations = errors.New("revoke answer service: downlink revocation writer is nil")
	// ErrNilRevokeAnswersExpiries rejects revoke answers built without the reporter of downlinks expired at their station.
	ErrNilRevokeAnswersExpiries = errors.New("revoke answer service: expiry reporter is nil")
	// ErrNilRevokeAnswersSerializer rejects revoke answers built without a queue serializer.
	ErrNilRevokeAnswersSerializer = errors.New("revoke answer service: queue serializer is nil")
	// ErrNoRevokeNotHeldCodes rejects revoke answers that could never tell a station does not hold a downlink.
	ErrNoRevokeNotHeldCodes = errors.New("revoke answer service: no dlDataRev refusal code says the station does not hold the downlink")
	// ErrNilApplicationCenterResults rejects a result reporter built without its Application Center delivery.
	ErrNilApplicationCenterResults = errors.New("downlink result reporter: application center delivery is nil")
	// ErrNilDownlinkResultPublisher rejects a result reporter built without its MQTT publisher.
	ErrNilDownlinkResultPublisher = errors.New("downlink result reporter: mqtt publisher is nil")
	// ErrNilDownlinkResultEvents rejects a result reporter built without its event recorder.
	ErrNilDownlinkResultEvents = errors.New("downlink result reporter: event recorder is nil")
	// ErrNilLateResultEvents rejects a result reporter built without the recorder of results that contradict an expiry.
	ErrNilLateResultEvents = errors.New("downlink result reporter: late result recorder is nil")
	// ErrNilReporterRunner rejects a result reporter built without its background runner.
	ErrNilReporterRunner = errors.New("downlink result reporter: background runner is nil")
	// ErrNilReporterLogger rejects a result reporter built without a logger.
	ErrNilReporterLogger = errors.New("downlink result reporter: logger is nil")
	// ErrNilEndpointLocator rejects a serving station policy built without the endpoint locations.
	ErrNilEndpointLocator = errors.New("serving station policy: endpoint locations are nil")
	// ErrNilEndpointAckStore rejects an endpoint acknowledgement recorder built without the downlink queue.
	ErrNilEndpointAckStore = errors.New("endpoint acknowledgement recorder: downlink queue is nil")
	// ErrNilEndpointAckLogger rejects an endpoint acknowledgement recorder built without a logger.
	ErrNilEndpointAckLogger = errors.New("endpoint acknowledgement recorder: logger is nil")
	// ErrNilEndpointAckEvents rejects an endpoint acknowledgement recorder built without its event recorder.
	ErrNilEndpointAckEvents = errors.New("endpoint acknowledgement recorder: event recorder is nil")
	// ErrNilDownlinkAckRecorder rejects an uplink ingest service built without its acknowledgement recorder.
	ErrNilDownlinkAckRecorder = errors.New("uplink ingest service: downlink acknowledgement recorder is nil")
	// errNilAttachTransactionRunner rejects an attachment persister built without its transaction runner.
	errNilAttachTransactionRunner = errors.New("attachment persistence: transaction runner is nil")
	// errNilPrimaryStationLookup rejects an attachment persister built without its base station lookup.
	errNilPrimaryStationLookup = errors.New("attachment persistence: primary station lookup is nil")
	// errNilAttachPersistenceClock rejects an attachment persister built without a clock.
	errNilAttachPersistenceClock = errors.New("attachment persistence: clock is nil")
	// errNilAttachPersistenceLogger rejects an attachment persister built without a logger.
	errNilAttachPersistenceLogger = errors.New("attachment persistence: logger is nil")
	// ErrNilEndpointOwnerResolver rejects an uplink ingest service built without the endpoint owner rule.
	ErrNilEndpointOwnerResolver = errors.New("uplink ingest service: endpoint owner resolver is nil")
	// errNilAttachmentEndpoints rejects an attachment service built without its endpoint lookup.
	errNilAttachmentEndpoints = errors.New("endpoint attachment service: endpoint store is nil")
	// errNilAttachedEndpointCreator rejects an attachment service built without the creator of attached endpoints.
	errNilAttachedEndpointCreator = errors.New("endpoint attachment service: attached endpoint creator is nil")
	// errNilAttachmentDecider rejects an attachment service built without the attachment decider.
	errNilAttachmentDecider = errors.New("endpoint attachment service: attachment decider is nil")
	// errNilAttachmentPropagation rejects an attachment service built without its propagation.
	errNilAttachmentPropagation = errors.New("endpoint attachment service: propagation is nil")
	// errNilAttachPropagateSender rejects an attachment propagation built without the base stations it sends to.
	errNilAttachPropagateSender = errors.New("attachment propagation: base station sender is nil")
	// errNilAttachmentSessionKeys rejects an attachment propagation built without its network session key source.
	errNilAttachmentSessionKeys = errors.New("attachment propagation: network session key source is nil")
	// errNilAttachmentEvents rejects an attachment propagation built without its event recorder.
	errNilAttachmentEvents = errors.New("attachment propagation: event recorder is nil")
	// errNilAttachmentClock rejects an attachment propagation built without a clock.
	errNilAttachmentClock = errors.New("attachment propagation: clock is nil")
	// errNilAttachmentLogger rejects an attachment propagation built without a logger.
	errNilAttachmentLogger = errors.New("attachment propagation: logger is nil")
	// errAttachEndPoint reports an attach the attachment service could not take.
	errAttachEndPoint = errors.New("endpoint attachment service: attach")
	// errDetachEndPoint reports a detach the attachment service could not take.
	errDetachEndPoint = errors.New("endpoint attachment service: detach")
	// errCreateAttachedEndpoint reports a pre-attached endpoint that could not be stored.
	errCreateAttachedEndpoint = errors.New("endpoint attachment service: create attached endpoint")
	// errNilProvisioningReader rejects a network session key source built without the endpoint provisioning.
	errNilProvisioningReader = errors.New("network session key source: endpoint provisioning reader is nil")
	// errNilActiveSessionReader rejects a network session key source built without the endpoint sessions.
	errNilActiveSessionReader = errors.New("network session key source: active session reader is nil")
	// errNilStatusTransitioner rejects an attachment decider built without its status store.
	errNilStatusTransitioner = errors.New("attachment decider: status transitioner is nil")
	// errNilEndpointStatusNotifier rejects an attachment decider or notifier fan-out built without a notifier.
	errNilEndpointStatusNotifier = errors.New("attachment decider: endpoint status notifier is nil")
	// errNilBackgroundRunner rejects a service built without the runner of its background work.
	errNilBackgroundRunner = errors.New("background work runner is nil")
	// errNilEPStatusBroadcaster rejects an epStat notifier built without its broadcaster.
	errNilEPStatusBroadcaster = errors.New("epStat notifier: broadcaster is nil")
	// errNilAttachmentEventPublisher rejects an MQTT attachment notifier built without its publisher.
	errNilAttachmentEventPublisher = errors.New("MQTT attachment notifier: publisher is nil")
	// errNilOwnerOrganizations rejects an MQTT attachment notifier built without the owner organizations.
	errNilOwnerOrganizations = errors.New("MQTT attachment notifier: owner organizations are nil")
	// errNilEndpointStatusLogger rejects an endpoint status notifier built without a logger.
	errNilEndpointStatusLogger = errors.New("endpoint status notifier: logger is nil")
	// errNilEndpointOwnerLookup rejects an endpoint owner resolver built without its endpoint lookup.
	errNilEndpointOwnerLookup = errors.New("endpoint owner resolver: endpoint lookup is nil")
	// errUpdateEndpointSession reports a failed session update inside the propagate transaction.
	errUpdateEndpointSession = errors.New("failed to update endpoint session")
	// errCreateEndpointSession reports a failed session insert inside the propagate transaction.
	errCreateEndpointSession = errors.New("failed to create endpoint session")
	// errMarshalUplinkPayload reports an uplink event body that could not be serialized.
	errMarshalUplinkPayload = errors.New("mqtt adapter: failed to marshal uplink payload")
	// errMarshalAttachPayload reports an attach event body that could not be serialized.
	errMarshalAttachPayload = errors.New("mqtt adapter: failed to marshal attach payload")
	// errMarshalDetachPayload reports a detach event body that could not be serialized.
	errMarshalDetachPayload = errors.New("mqtt adapter: failed to marshal detach payload")
	// errMarshalDownlinkResultPayload reports a downlink-result event body that could not be serialized.
	errMarshalDownlinkResultPayload = errors.New("mqtt adapter: failed to marshal downlink result payload")
	// errRoamingDetectionFailed reports a roaming detector failure; ingest fails closed on it.
	errRoamingDetectionFailed = errors.New("roaming detection failed")
	// errRoamingNotAllowed reports a roaming uplink without a tenant partnership.
	errRoamingNotAllowed = errors.New("roaming not allowed")
	// errNilRoamingResolver rejects a roaming detector built without its ownership resolver.
	errNilRoamingResolver = errors.New("roaming detector: nil ownership resolver")
	// errResumableSessionLookup reports a resumable-session query failure during connect.
	errResumableSessionLookup = errors.New("resumable session lookup failed")
	// errTerminateStaleSession reports a stale session that could not be retired before activation.
	errTerminateStaleSession = errors.New("terminate stale session before activation")
	// errTerminateResumableSessions reports leftover resumable sessions that could not be retired before activation.
	errTerminateResumableSessions = errors.New("terminate leftover resumable sessions before activation")
	// errRemovePendingOperations reports a retired session's pending operations that could not be deleted.
	errRemovePendingOperations = errors.New("remove pending operations of stale session")
	// errCreateSessionInDatabase reports a failed base-station session insert.
	errCreateSessionInDatabase = errors.New("failed to create session in database")
	// errInitializeSessionCounters reports failed persistence of initial operation ID counters.
	errInitializeSessionCounters = errors.New("failed to initialize session counters")
	// errSessionNotPersistedDisconnect reports a disconnect mark on a session without a DB row.
	errSessionNotPersistedDisconnect = errors.New("cannot mark session disconnected: not persisted (DbSessionID=0)")
	// errMarkSessionDisconnected reports a failed disconnect-mark repository update.
	errMarkSessionDisconnected = errors.New("failed to mark session disconnected")
	// errSessionNotPersistedCounters reports a counter update on a session without a DB row.
	errSessionNotPersistedCounters = errors.New("cannot update counters: session not persisted (DbSessionID=0)")
	// errPersistSessionCounters reports failed persistence of operation ID counters.
	errPersistSessionCounters = errors.New("failed to persist session counters")
	// errSessionNotPersistedPing reports a ping-timestamp update on a session without a DB row.
	errSessionNotPersistedPing = errors.New("cannot update ping timestamp: session not persisted (DbSessionID=0)")
	// errPersistPingTimestamp reports failed persistence of the last ping timestamp.
	errPersistPingTimestamp = errors.New("failed to persist ping timestamp")
	// errSessionNotPersistedTerminate reports a terminate on a session without a DB row.
	errSessionNotPersistedTerminate = errors.New("cannot terminate session: not persisted (DbSessionID=0)")
	// errTerminateSessionFailed reports a failed session termination update.
	errTerminateSessionFailed = errors.New("failed to terminate session")
	// errQueueStoreUnavailable reports tenant resolution without a configured queue store.
	errQueueStoreUnavailable = errors.New("queue store not available for tenant resolution")
	// errEndpointLookupFailed reports a cross-tenant endpoint lookup failure during ingest.
	errEndpointLookupFailed = errors.New("endpoint lookup failed")
	// errPersistUplinkFailed reports a failed uplink message insert.
	errPersistUplinkFailed = errors.New("persist failed")
	// errNegotiatorNilLogger reports negotiator construction without a logger.
	errNegotiatorNilLogger = errors.New("version negotiator requires a logger")
	// errNegotiatorEmptySupportedSet reports negotiator construction with no supported versions.
	errNegotiatorEmptySupportedSet = errors.New("version negotiator requires a non-empty supported version set")
)

// Error format strings for failures that interleave dynamic context with the
// message text. Each is used with fmt.Errorf and preserves the exact wire text.
const (
	errFmtCertEUIRepoUnavailable       = "certificate EUI CN %q: base station repository unavailable"
	errFmtCertEUINoRegisteredBS        = "certificate EUI CN %q: no registered base station: %w"
	errFmtCertCNNoOrgResolver          = "certificate CN %q: no organization resolver configured"
	errFmtBaseStationNotRegistered     = "base station %s not registered"
	errFmtCannotResolveTenantForQueue  = "cannot resolve tenant for queue %d: %w"
	errFmtInvalidQueueID               = "invalid queue ID: %d"
	errFmtKeyZeroFilled                = "%w: key is zero-filled"
	errFmtNetworkKeyLength             = "invalid network key length %d (expected 16)"
	errFmtFetchEndpoint                = "failed to fetch endpoint %d: %w"
	errFmtPropagationTenant            = "no tenant to propagate endpoint %d under: %w"
	errFmtEncodeEventDetails           = "encode event details: %w"
	errFmtFetchEndpointsForTenant      = "failed to fetch endpoints for tenant %d: %w"
	errFmtFetchDetachedEndpoints       = "failed to fetch the endpoints detached for tenant %d: %w"
	errFmtFetchAttachedEndpoints       = "failed to fetch the endpoints attached for tenant %d: %w"
	errFmtResolveEndpointOwner         = "failed to resolve the owner of endpoint %s: %w"
	errFmtUnknownAttachmentDecision    = "unknown attachment decision %q"
	errFmtAggregateFailures            = "%s: %d failures, first: %w"
	errFmtEndpointNotFoundDuringIngest = "endpoint not found during ingest: ep_eui=%d"
	errFmtInvalidResolvedTenant        = "tenant resolution yielded invalid tenant id %d for ep_eui=%d"
	errFmtResumeActivation             = "resume activation for session %d: %w"
	errFmtMalformedSupportedVersion    = "malformed supported version %q: %s"
	errFmtDuplicateSupportedVersion    = "duplicate supported version %q"
)
