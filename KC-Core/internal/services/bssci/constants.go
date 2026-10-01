package bssciservices

import "time"

// Timing policies for the BSSCI domain services.
const (
	// propagateAsyncTimeout bounds background attach/detach propagation to all
	// connected base stations after the initiating handler has returned.
	propagateAsyncTimeout = 30 * time.Second
	// downlinkResultDeliveryTimeout bounds each background delivery of a
	// downlink result, to the Application Centers and to MQTT.
	downlinkResultDeliveryTimeout = 2 * time.Second
	// endpointKeyLength is the length of an endpoint's network session key,
	// Numeric[16] (BSSCI §5.8.1).
	endpointKeyLength = 16
)

// failureReasonStationErrorFmt records a base station's error answer (POSIX
// code and message, BSSCI §3.17.1) as a downlink's failure reason.
const failureReasonStationErrorFmt = "base station error %d: %s"
