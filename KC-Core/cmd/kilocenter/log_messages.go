package main

// Composition-root log messages used during service startup and shutdown.
const (
	LogFailedBuildInfrastructure = "Failed to build infrastructure"
	LogFailedBuildGRPCServer     = "Failed to build gRPC server"
	LogFailedBuildProtocol       = "Failed to build protocol servers"
	LogFailedInitTracing         = "Failed to init tracing"
	LogFailedInitMetrics         = "Failed to init metrics"
	LogHealthCheckServerError    = "Health check server error: %v\n"
)

// Health endpoint response bodies.
const (
	HealthResponseHealthy = `{"status":"healthy"}`
	HealthResponseAlive   = `{"status":"alive"}`
	HealthResponseFormat  = `{"status":"%s"}`
)
