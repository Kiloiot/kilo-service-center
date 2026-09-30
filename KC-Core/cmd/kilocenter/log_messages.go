package main

// Composition-root log messages used during service startup and shutdown.
const (
	LogFailedBuildInfrastructure     = "Failed to build infrastructure"
	LogFailedBuildGRPCServer         = "Failed to build gRPC server"
	LogFailedBuildProtocol           = "Failed to build protocol servers"
	LogFailedInitTracing             = "Failed to init tracing"
	LogFailedInitMetrics             = "Failed to init metrics"
	LogFailedShutdownTracing         = "Failed to shut down tracing"
	LogFailedShutdownMetrics         = "Failed to shut down metrics"
	LogFailedEmitServiceStoppedEvent = "Failed to emit service stopped event"
	LogHealthCheckServerError        = "Health check server error: %v\n"
	LogFailedWriteHealthResponse     = "Failed to write health response"
)

// Health endpoint response bodies.
const (
	HealthResponseHealthy = `{"status":"healthy"}`
	HealthResponseAlive   = `{"status":"alive"}`
)

// Lifecycle log messages emitted by the main entrypoint.
const (
	LogContextCancelled              = "Context cancelled"
	LogReceivedShutdownSignal        = "Received shutdown signal"
	LogServiceStoppedEventEmitted    = "Service stopped event emitted"
	LogShutdownComplete              = "Shutdown complete"
	LogShuttingDownGracefully        = "Shutting down gracefully..."
	LogStartingKiloCenterMIOTYServer = "Starting KiloCenter MIOTY Server"
)
