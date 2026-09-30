package main

import "time"

// Entrypoint log messages used during service startup and shutdown.
const (
	LogServiceStarting        = "Starting KC-Identity Service"
	LogFailedInitTracing      = "Failed to init tracing"
	LogFailedInitMetrics      = "Failed to init metrics"
	LogFailedBuildInfra       = "Failed to build infrastructure"
	LogServiceStartedEvent    = "Service started event emitted"
	LogReceivedShutdownSignal = "Received shutdown signal"
	LogContextCancelled       = "Context cancelled"
	LogServiceStoppedEvent    = "Service stopped event emitted"
	LogShuttingDown           = "Shutting down gracefully..."
	LogShutdownComplete       = "Shutdown complete"
	LogHealthCheckServerError = "Health check server error"

	LogObservabilityShutdownFailed = "Failed to shut down tracing or metrics"
	LogHostnameUnavailable         = "Failed to resolve hostname for lifecycle events"
	LogLifecycleEventFailed        = "Failed to record lifecycle event"
	LogHealthResponseWriteFailed   = "Failed to write health response"
	LogHealthServerShutdownError   = "Health server shutdown error"
)

// Timing policies for the health endpoint and shutdown paths.
const (
	// healthServerReadHeaderTimeout bounds how long the health server waits for request headers.
	healthServerReadHeaderTimeout = 10 * time.Second
	// healthServerIdleTimeout closes idle keep-alive connections on the health server.
	healthServerIdleTimeout = 120 * time.Second
	// healthServerShutdownTimeout bounds the health server's graceful shutdown.
	healthServerShutdownTimeout = 5 * time.Second
	// shutdownEventTimeout bounds the service-stopped event write during shutdown.
	shutdownEventTimeout = 5 * time.Second
)
