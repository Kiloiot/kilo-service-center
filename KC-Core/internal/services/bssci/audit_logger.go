// Package bssciservices implements BSSCI service layer components.
package bssciservices

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// DownlinkLookup reads the stored downlink an event describes.
type DownlinkLookup interface {
	GetDownlinkByQueueID(ctx context.Context, queID uint64, tenantID string) (*storage.DownlinkMessage, error)
}

// lookupWarner reports a downlink whose user data an event could not read.
type lookupWarner interface {
	WarnContext(ctx context.Context, msg string, fields ...interface{})
}

// DownlinkAuditLog records the downlink lifecycle in the owner tenant's
// events (BSSCI §3.12-§3.14, §3.10.1 dlAck): every event names the queue id,
// the endpoint, the base station involved and the user data. It implements
// bssci.AuditLogger and the result reporter's event recorder.
type DownlinkAuditLog struct {
	eventStore SystemEventRecorder
	downlinks  DownlinkLookup
	stations   bssci.StationDirectory
	clock      clock.Clock
	log        lookupWarner
}

// AuditLogDeps are the audit logger's collaborators; all are mandatory, so a
// wiring fault surfaces at startup instead of on the first audited action.
type AuditLogDeps struct {
	Events    SystemEventRecorder
	Downlinks DownlinkLookup
	Stations  bssci.StationDirectory
	Clock     clock.Clock
	Logger    lookupWarner
}

// NewAuditLogger creates the audit logger.
func NewAuditLogger(deps AuditLogDeps) (*DownlinkAuditLog, error) {
	switch {
	case deps.Events == nil:
		return nil, ErrNilEventStore
	case deps.Downlinks == nil:
		return nil, ErrNilAuditDownlinks
	case deps.Stations == nil:
		return nil, ErrNilAuditStations
	case deps.Clock == nil:
		return nil, ErrNilClock
	case deps.Logger == nil:
		return nil, ErrNilAuditLogger
	}
	return &DownlinkAuditLog{eventStore: deps.Events, downlinks: deps.Downlinks, stations: deps.Stations, clock: deps.Clock, log: deps.Logger}, nil
}

// RecordQueueAck records the base station accepting a downlink into its queue (dlDataQueRsp, BSSCI §3.12).
func (a *DownlinkAuditLog) RecordQueueAck(ctx context.Context, tenant string, session *bssci.Session,
	epEui uint64, queueID int64, opID int64,
) error {
	queID, tenantID, err := queueOwner(queueID, tenant)
	if err != nil {
		return err
	}
	station := session.BaseStationEUI
	return a.record(ctx, dlQueuedKind, downlinkFacts{
		tenantID: tenantID, queID: queID, epEUI: mioty.FormatEUI64(epEui), opID: &opID, station: &station,
	})
}

// RecordDLResult records the result a base station reported for a downlink
// (dlDataRes, BSSCI §3.14): for a sent downlink its transmit time and the
// packet counter of the uplink it followed.
func (a *DownlinkAuditLog) RecordDLResult(ctx context.Context, tenant string, session *bssci.Session,
	result *mioty.DLDataResult,
) error {
	if result.QueId > math.MaxInt64 {
		return fmt.Errorf(errFmtInvalidQueueID, result.QueId)
	}
	_, tenantID, err := queueOwner(int64(result.QueId), tenant)
	if err != nil {
		return err
	}
	station := session.BaseStationEUI
	return a.record(ctx, resultKind(result.Result), downlinkFacts{
		tenantID: tenantID, queID: result.QueId, epEUI: mioty.FormatEUI64(result.EpEui), station: &station,
		result: result.Result, txTime: result.TxTime, packetCnt: result.PacketCnt,
	})
}

// RecordQueueExpiry records a downlink the service center expired before any
// base station transmitted it; the event also refreshes the web downlink views.
func (a *DownlinkAuditLog) RecordQueueExpiry(ctx context.Context, downlink *storage.DownlinkMessage) error {
	queID, tenantID, err := queueOwner(downlink.QueID, downlink.TenantID)
	if err != nil {
		return err
	}
	return a.record(ctx, dlExpiredInQueueKind, downlinkFacts{
		tenantID: tenantID, queID: queID, epEUI: downlink.EPEUI, result: mioty.ResultExpired,
	})
}

// RecordDLRevokeResponse records a base station revoking a downlink (BSSCI §3.13).
func (a *DownlinkAuditLog) RecordDLRevokeResponse(ctx context.Context, tenant string, session *bssci.Session,
	epEui uint64, queueID int64, opID int64,
) error {
	queID, tenantID, err := queueOwner(queueID, tenant)
	if err != nil {
		return err
	}
	station := session.BaseStationEUI
	return a.record(ctx, dlRevokedKind, downlinkFacts{
		tenantID: tenantID, queID: queID, epEUI: mioty.FormatEUI64(epEui), opID: &opID, station: &station,
		result: mioty.DLDataResultRevoked,
	})
}

// RecordDownlinkAcknowledged records the endpoint acknowledging, in the uplink
// after packetCnt, the downlink queued under queueID (BSSCI §3.10.1 dlAck).
func (a *DownlinkAuditLog) RecordDownlinkAcknowledged(ctx context.Context, tenantID int64, epEUI uint64, queueID int64, packetCnt uint32) error {
	queID, err := wireQueueID(queueID)
	if err != nil {
		return err
	}
	return a.record(ctx, dlAcknowledgedKind, downlinkFacts{
		tenantID: tenantID, queID: queID, epEUI: mioty.FormatEUI64(epEUI), packetCnt: &packetCnt,
	})
}

// RecordQueueRevoked records a downlink revoked in the service center queue
// before any base station held it (SCACI §3.11).
func (a *DownlinkAuditLog) RecordQueueRevoked(ctx context.Context, downlink *storage.DownlinkMessage) error {
	return a.recordStored(ctx, dlRevokedInQueueKind, downlink, mioty.DLDataResultRevoked)
}

// RecordEnqueued records a downlink entering the service center queue (SCACI §3.10).
func (a *DownlinkAuditLog) RecordEnqueued(ctx context.Context, downlink *storage.DownlinkMessage) error {
	return a.recordStored(ctx, dlEnqueuedKind, downlink, "")
}

// RecordPendingUpdated records a pending downlink rewritten in the queue.
func (a *DownlinkAuditLog) RecordPendingUpdated(ctx context.Context, downlink *storage.DownlinkMessage) error {
	return a.recordStored(ctx, dlUpdatedKind, downlink, "")
}

// RecordRequeued records a downlink returned to the queue because the base
// station holding it let it go; it waits for the endpoint's next window.
func (a *DownlinkAuditLog) RecordRequeued(ctx context.Context, downlink storage.PendingDownlink, bsEUI uint64) error {
	return a.record(ctx, dlRequeuedKind, downlinkFacts{
		tenantID: downlink.TenantID, queID: downlink.QueID, epEUI: mioty.FormatEUI64(downlink.EpEUI), station: &bsEUI,
	})
}

// recordStored writes the event of a downlink the caller read from the queue.
func (a *DownlinkAuditLog) recordStored(ctx context.Context, kind downlinkEventKind, downlink *storage.DownlinkMessage, result string) error {
	queID, tenantID, err := queueOwner(downlink.QueID, downlink.TenantID)
	if err != nil {
		return err
	}
	return a.write(ctx, kind, downlinkFacts{tenantID: tenantID, queID: queID, epEUI: downlink.EPEUI, result: result}, downlink)
}

// record writes the event of a downlink it reads from the queue by its facts.
func (a *DownlinkAuditLog) record(ctx context.Context, kind downlinkEventKind, f downlinkFacts) error {
	return a.write(ctx, kind, f, a.lookup(ctx, f))
}

// write writes one downlink event, filed under the endpoint owner and, when
// the base station is the owner's own, on that station's timeline too.
func (a *DownlinkAuditLog) write(ctx context.Context, kind downlinkEventKind, f downlinkFacts, downlink *storage.DownlinkMessage) error {
	payload := formatDownlinkPayload(downlink)
	story := downlinkStory{queID: f.queID, endpoint: f.epEUI, payload: payload, result: f.result}
	if f.packetCnt != nil {
		story.packetCnt = *f.packetCnt
	}
	var station *bssci.StationIdentity
	if f.station != nil {
		identity := bssci.IdentifyStation(ctx, a.stations, f.tenantID, *f.station)
		station, story.station = &identity, identity.Label()
	}
	details, err := json.Marshal(downlinkDetails(f, station, payload))
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarshalAuditDetails, err)
	}
	now := a.clock.Now()
	event := &models.SystemEvent{
		TenantID: strconv.FormatInt(f.tenantID, 10), EventType: kind.eventType, Category: models.EventCategoryMessage,
		Severity: kind.severity, Title: kind.title, Description: kind.describe(story),
		SourceType: mioty.SourceTypeEndpoint, SourceName: f.epEUI, Details: details, CreatedAt: now, UpdatedAt: now,
	}
	if station != nil {
		event.BasestationID = station.ID
	}
	return a.eventStore.CreateEvent(ctx, event)
}

// lookup reads the downlink an event names; nil when it cannot be read, so
// the event still records what the station reported.
func (a *DownlinkAuditLog) lookup(ctx context.Context, f downlinkFacts) *storage.DownlinkMessage {
	downlink, err := a.downlinks.GetDownlinkByQueueID(ctx, f.queID, strconv.FormatInt(f.tenantID, 10))
	if err != nil {
		a.log.WarnContext(ctx, LogDownlinkEventPayloadUnread, logger.FieldQueID, f.queID, logger.FieldError, err)
		return nil
	}
	return downlink
}
