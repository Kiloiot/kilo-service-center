package main

import "time"

// Lifecycle timing policy for the main entrypoint.
const (
	// healthServerReadHeaderTimeout bounds header reads on the health endpoint.
	healthServerReadHeaderTimeout = 10 * time.Second

	// healthServerIdleTimeout closes idle health endpoint connections.
	healthServerIdleTimeout = 120 * time.Second

	// shutdownEventTimeout bounds emitting the service-stopped event during
	// graceful shutdown.
	shutdownEventTimeout = 5 * time.Second
)
