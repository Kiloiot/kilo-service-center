package bssciservices

import (
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// downlinkFacts are what one downlink lifecycle event states.
type downlinkFacts struct {
	tenantID  int64
	queID     uint64
	epEUI     string
	opID      *int64
	station   *uint64
	result    string
	txTime    *int64
	packetCnt *uint32
}

// downlinkStory is what an event description tells about a downlink.
type downlinkStory struct {
	queID     uint64
	endpoint  string
	station   string
	payload   string
	result    string
	packetCnt uint32
}

// downlinkEventKind is the fixed part of one kind of downlink event.
type downlinkEventKind struct {
	eventType string
	title     string
	severity  string
	describe  func(downlinkStory) string
}

var (
	dlQueuedKind = downlinkEventKind{models.EventTypeDLDataQueueAcknowledged, models.EventTitleDLDataQueued, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataQueuedFormat, d.queID, d.endpoint, d.station, d.payload)
		}}
	dlSentKind = downlinkEventKind{models.EventTypeDLDataSent, models.EventTitleDLDataSent, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataSentFormat, d.queID, d.endpoint, d.station, d.packetCnt, d.payload)
		}}
	dlSentAfterExpiryKind = downlinkEventKind{models.EventTypeDLDataSentAfterExpiry, models.EventTitleDLDataSentAfterExpiry, models.EventSeverityWarning,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataSentAfterExpiryFormat, d.queID, d.endpoint, d.station, d.packetCnt, d.payload)
		}}
	dlExpiredKind = downlinkEventKind{models.EventTypeDLDataExpired, models.EventTitleDLDataExpired, models.EventSeverityWarning,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataExpiredFormat, d.queID, d.endpoint, d.station, d.payload)
		}}
	dlInvalidKind = downlinkEventKind{models.EventTypeDLDataInvalid, models.EventTitleDLDataInvalid, models.EventSeverityError,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataInvalidFormat, d.queID, d.endpoint, d.station, d.payload)
		}}
	dlUnknownKind = downlinkEventKind{models.EventTypeDLDataUnknownResult, models.EventTitleDLDataUnknownResult, models.EventSeverityError,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataUnknownResultFormat, d.station, d.result, d.queID, d.endpoint, d.payload)
		}}
	dlExpiredInQueueKind = downlinkEventKind{models.EventTypeDLDataExpired, models.EventTitleDLDataExpired, models.EventSeverityWarning,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataExpiredInQueueFormat, d.queID, d.endpoint, d.payload)
		}}
	dlRevokedKind = downlinkEventKind{models.EventTypeDLDataRevoked, models.EventTitleDLDataRevoked, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataRevokedFormat, d.queID, d.endpoint, d.station, d.payload)
		}}
	dlAcknowledgedKind = downlinkEventKind{models.EventTypeDLDataAcknowledged, models.EventTitleDLDataAcknowledged, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataAcknowledgedFormat, d.endpoint, d.queID, d.packetCnt, d.payload)
		}}
	dlRevokedInQueueKind = downlinkEventKind{models.EventTypeDLDataRevoked, models.EventTitleDLDataRevoked, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataRevokedInQueueFormat, d.queID, d.endpoint, d.payload)
		}}
	dlEnqueuedKind = downlinkEventKind{models.EventTypeDLDataEnqueued, models.EventTitleDLDataEnqueued, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataEnqueuedFormat, d.queID, d.endpoint, d.payload)
		}}
	dlUpdatedKind = downlinkEventKind{models.EventTypeDLDataUpdated, models.EventTitleDLDataUpdated, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataUpdatedFormat, d.queID, d.endpoint, d.payload)
		}}
	dlRequeuedKind = downlinkEventKind{models.EventTypeDLDataRequeued, models.EventTitleDLDataRequeued, models.EventSeverityInfo,
		func(d downlinkStory) string {
			return fmt.Sprintf(descriptionDLDataRequeuedFormat, d.queID, d.endpoint, d.station, d.payload)
		}}
)

// resultKinds maps a base station's dlDataRes result onto its event (BSSCI §3.14.1).
var resultKinds = map[string]downlinkEventKind{
	mioty.ResultSent:    dlSentKind,
	mioty.ResultExpired: dlExpiredKind,
	mioty.ResultInvalid: dlInvalidKind,
}

// resultKind is the event of a result; a value outside the specification is recorded as unknown.
func resultKind(result string) downlinkEventKind {
	if kind, ok := resultKinds[result]; ok {
		return kind
	}
	return dlUnknownKind
}

// formatDownlinkPayload renders the user data in hex, each entry of a
// counter-dependent downlink with the packet counter it was queued for.
func formatDownlinkPayload(downlink *storage.DownlinkMessage) string {
	if downlink == nil {
		return payloadUnavailable
	}
	if !downlink.CntDepend || len(downlink.UserData) != len(downlink.PacketCntArray) {
		return payloadText(downlink.Payload)
	}
	entries := make([]string, len(downlink.UserData))
	for i, entry := range downlink.UserData {
		entries[i] = fmt.Sprintf(payloadAtCounterFormat, payloadText(entry), downlink.PacketCntArray[i])
	}
	return strings.Join(entries, payloadEntrySeparator)
}

// payloadText names a downlink without user data instead of rendering nothing.
func payloadText(payload []byte) string {
	if len(payload) == 0 {
		return payloadEmpty
	}
	return mioty.FormatEUIBytes(payload)
}

// downlinkDetails are the event details of a downlink event.
func downlinkDetails(f downlinkFacts, station *bssci.StationIdentity, payload string) map[string]interface{} {
	details := map[string]interface{}{
		models.EventDetailKeyEpEui:    f.epEUI,
		models.EventDetailKeyQueID:    f.queID,
		models.EventDetailKeyUserData: payload,
	}
	if f.opID != nil {
		details[models.EventDetailKeyOpID] = *f.opID
	}
	if f.result != "" {
		details[models.EventDetailKeyResult] = f.result
	}
	if f.packetCnt != nil {
		details[models.EventDetailKeyPacketCnt] = *f.packetCnt
	}
	if f.txTime != nil {
		details[models.EventDetailKeyTxTime] = *f.txTime
	}
	if station != nil {
		details[models.EventDetailKeyBsEui] = station.EUI
		if station.Name != "" {
			details[models.EventDetailKeyBaseStationName] = station.Name
		}
	}
	return details
}
