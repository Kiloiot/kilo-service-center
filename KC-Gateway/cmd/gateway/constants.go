package main

import "time"

// defaultHealthPort is the health endpoint port used when the configuration
// provides none.
const defaultHealthPort = 8087

// Gateway lifecycle timing policy.
const (
	// breakerGaugeInterval is how often circuit breaker state is exported to
	// the Prometheus gauge.
	breakerGaugeInterval = 5 * time.Second
	// stopEventTimeout bounds the service-stopped lifecycle event RPC during
	// shutdown.
	stopEventTimeout = 5 * time.Second
	// shutdownTimeout bounds the HTTP server's graceful shutdown.
	shutdownTimeout = 10 * time.Second
)

// Status texts returned when the upstream circuit breaker rejects a call.
const (
	statusBreakerOpen       = "upstream circuit breaker is open"
	statusBreakerRecovering = "upstream circuit breaker is recovering"
)

const gatewayServiceName = "kc-gateway"

// gatewayDisplayName and gatewayVersion label KC-Gateway lifecycle events.
const (
	gatewayDisplayName = "KC-Gateway"
	gatewayVersion     = "1.0.0"
)

// Upstream names used for circuit breakers and their Prometheus label values.
const (
	upstreamCore     = "core"
	upstreamIdentity = "identity"
	upstreamLabel    = "upstream"
)

// listenInfoFmt summarizes the listening ports for the started event.
const listenInfoFmt = "gRPC-web :%d, Health :%d"

// breakerStateHelp documents the circuit-breaker state gauge encoding.
const breakerStateHelp = "Circuit breaker state (0=closed, 1=half-open, 2=open)"

// hopByHopHeaders are stripped before proxying: they are invalid in the
// HTTP/2 upstream connection.
var hopByHopHeaders = []string{"Connection", "Keep-Alive", "Proxy-Connection", "Transfer-Encoding", "Upgrade"}
