package scaciservices

// Log messages owned by the SCACI domain services. Protocol-level messages
// shared with the SCACI transport live in KC-Core/pkg/scaci/log_messages.go;
// this catalog covers service-layer lookups and persistence guards.
const (
	LogInvalidEUILengthGetByEUI = "Invalid EUI length for GetByEUI"
	LogEndpointNotFoundGetByEUI = "Endpoint not found in GetByEUI"
	LogDetachPropagationRefused = "Detach propagation refused: the tenant has no endpoint with this EUI"
	LogHeldSessionLimitReached  = "Disconnected SCACI session holds the most operations allowed; it can no longer be resumed"
	LogSessionHeldForResume     = "Disconnected SCACI session held for its resume; its operations are recorded"
	LogHeldSessionDiscarded     = "Held SCACI session discarded: a newer session of its Application Center replaced it"
)
