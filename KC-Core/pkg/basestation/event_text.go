package basestation

import (
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// eventText is how a base station event reads in the activity feeds.
type eventText struct {
	severity    string
	title       string
	description string
}

// eventDescriber words one event type from the station EUI and the event data.
type eventDescriber func(euiStr string, data map[string]interface{}) eventText

// eventDescribers words each base station event type the recorder is given.
var eventDescribers = map[string]eventDescriber{
	models.EventTypeBaseStationOffline: func(euiStr string, data map[string]interface{}) eventText {
		name := stationName(data)
		return eventText{models.EventSeverityWarning,
			fmt.Sprintf(titleFmtBSWentOffline, name), fmt.Sprintf(descriptionFmtBSDisconnected, name, euiStr)}
	},
	models.EventTypeBaseStationOnline: func(euiStr string, data map[string]interface{}) eventText {
		name := stationName(data)
		return eventText{models.EventSeverityInfo,
			fmt.Sprintf(titleFmtBSConnected, name), fmt.Sprintf(descriptionFmtBSConnected, name, euiStr)}
	},
	models.EventTypeBaseStationPingSent: func(euiStr string, data map[string]interface{}) eventText {
		return eventText{models.EventSeverityInfo,
			titleBSPingSent, fmt.Sprintf(descriptionFmtBSPingSent, data[models.EventDetailKeyOpID], stationName(data), euiStr)}
	},
	models.EventTypeBaseStationPingAnswered: func(euiStr string, data map[string]interface{}) eventText {
		return eventText{models.EventSeverityInfo,
			titleBSPingAnswered, fmt.Sprintf(descriptionFmtBSPingAnswered, stationName(data), euiStr, data[models.EventDetailKeyOpID])}
	},
	models.EventTypeBaseStationStatusAnswered: func(euiStr string, data map[string]interface{}) eventText {
		return eventText{models.EventSeverityInfo,
			titleBSStatusAnswered, fmt.Sprintf(descriptionFmtBSStatusAnswered, stationName(data), euiStr, data[models.EventDetailKeyOpID])}
	},
}

// describeEvent words an event; a type without its own wording gets a generic one.
func describeEvent(eventType, euiStr string, data map[string]interface{}) eventText {
	if describe, ok := eventDescribers[eventType]; ok {
		return describe(euiStr, data)
	}
	return eventText{models.EventSeverityInfo,
		fmt.Sprintf(titleFmtBSGenericEvent, euiStr, eventType), fmt.Sprintf(descriptionFmtBSGenericEvent, eventType, euiStr)}
}

// stationName is the station's registered name the event carries, else a placeholder.
func stationName(data map[string]interface{}) string {
	if name, ok := data[models.EventDetailKeyBaseStationName].(string); ok && name != "" {
		return name
	}
	return basestationNameUnknown
}
