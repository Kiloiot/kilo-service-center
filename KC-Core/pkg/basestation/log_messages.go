package basestation

// Log message catalog for base-station connection management and event recording.
const (
	LogFailedToGetBaseStationDetails    = "Failed to get base station details"
	LogBaseStationEvent                 = "Base station event"
	LogFailedToPersistEvent             = "Failed to persist event"
	LogFailedToBackfillServiceCenterURL = "Failed to backfill base station service center URL"
)

// basestationNameUnknown is the placeholder used in event titles/descriptions
// when the base station name is not present in the event data map.
const basestationNameUnknown = "Unknown"

// Base station lifecycle event titles and descriptions.
const (
	titleFmtBSWentOffline        = `Base Station "%s" went offline`
	descriptionFmtBSDisconnected = `Base Station "%s" (EUI: %s) disconnected from the Service Center`
	titleFmtBSConnected          = `Base Station "%s" connected`
	descriptionFmtBSConnected    = `Base Station "%s" (EUI: %s) successfully connected to the Service Center`
	titleFmtBSGenericEvent       = "Base station %s: %s"
	descriptionFmtBSGenericEvent = "Event %s occurred for base station %s"
)

// Base station ping event titles and descriptions (BSSCI §5.4).
const (
	titleBSPingSent              = "Ping sent"
	descriptionFmtBSPingSent     = `The Service Center sent ping opId %v to Base Station "%s" (EUI: %s)`
	titleBSPingAnswered          = "Ping answered"
	descriptionFmtBSPingAnswered = `Base Station "%s" (EUI: %s) answered ping opId %v`
)

// Base station status answer event title and description (BSSCI §5.5).
const (
	titleBSStatusAnswered          = "Status answered"
	descriptionFmtBSStatusAnswered = `Base Station "%s" (EUI: %s) answered status request opId %v`
)
