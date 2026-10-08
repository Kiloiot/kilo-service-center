// Package main provides the KiloCenter MIOTY Service Center entrypoint.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kiloiot/kilo-service-center/KC-Core/cmd/kilocenter/builders"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/observability"
)

func main() {
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showVersion {
		versionInfo := getVersionInfo()
		fmt.Printf("KiloCenter MIOTY Server\n")
		fmt.Printf("Version: %s\n", versionInfo.Version)
		fmt.Printf("Build Time: %s\n", versionInfo.BuildTime)
		fmt.Printf("Git Commit: %s\n", versionInfo.GitCommit)
		fmt.Printf("Git Branch: %s\n", versionInfo.GitBranch)
		fmt.Printf("Schema Version: %d\n", versionInfo.SchemaVersion)
		os.Exit(0)
	}

	// Load configuration
	configFile := *configPath
	if configFile == "" {
		configFile = os.Getenv("KILOCENTER_CONFIG_FILE")
	}
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Initialize(cfg.General.LogLevel, cfg.General.LogFormat)
	log := logger.Get()

	// Load version info
	versionInfo := getVersionInfo()

	log.Info(LogStartingKiloCenterMIOTYServer,
		logger.FieldVersion, versionInfo.Version,
		logger.FieldBuildTime, versionInfo.BuildTime,
		logger.FieldGitCommit, versionInfo.GitCommit,
		logger.FieldGitBranch, versionInfo.GitBranch,
		logger.FieldSchemaVersion, versionInfo.SchemaVersion,
	)

	// Initialize observability (tracing + metrics)
	tracingShutdown, err := observability.InitTracing(context.Background(), observability.TracingConfig{ // context-root: process
		Enabled:    cfg.Monitoring.TracingEnabled,
		Endpoint:   cfg.Monitoring.TracingEndpoint,
		SampleRate: cfg.Monitoring.TracingSampleRate,
	}, builders.CoreServiceSourceName)
	if err != nil {
		log.Error(LogFailedInitTracing, logger.FieldError, err)
	} else {
		defer shutdownObservability(log, tracingShutdown, LogFailedShutdownTracing)
	}

	metricsShutdown, err := observability.InitMetrics(context.Background(), observability.MetricsConfig{ // context-root: process
		Enabled: cfg.Monitoring.MetricsEnabled,
		Port:    cfg.Monitoring.MetricsPort,
		Path:    cfg.Monitoring.MetricsPath,
	}, builders.CoreServiceSourceName)
	if err != nil {
		log.Error(LogFailedInitMetrics, logger.FieldError, err)
	} else {
		defer shutdownObservability(log, metricsShutdown, LogFailedShutdownMetrics)
	}

	// Create application context
	ctx, cancel := context.WithCancel(context.Background()) // context-root: process
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 1. Infrastructure (storage, migrations, MQTT, health, org resolver)
	infra, err := builders.BuildInfrastructure(ctx, cfg, versionInfo, nil)
	if err != nil {
		log.Fatal(LogFailedBuildInfrastructure, logger.Err(err))
	}
	defer runCleanups(infra.Cleanups)

	// 2. gRPC server (needs org resolver from infra)
	grpcServer, grpcConfig, err := builders.BuildGRPCServer(infra)
	if err != nil {
		log.Fatal(LogFailedBuildGRPCServer, logger.Err(err))
	}

	// Readiness includes the gRPC server, so the endpoint starts once it exists.
	go startHealthCheck(log, cfg.General.HealthCheckPort, infra.HealthService, infra.ReadinessService)

	// 3. Protocol servers (BSSCI + SCACI)
	protocol, err := builders.BuildProtocolServers(ctx, infra)
	if err != nil {
		log.Fatal(LogFailedBuildProtocol, logger.Err(err))
	}
	defer runCleanups(protocol.Cleanups)

	// 4. Core service (device management, protocol operations, analytics)
	coreResult := builders.BuildCoreService(ctx, infra, protocol, nil)
	defer runCleanups(coreResult.Cleanups)

	// 5. Register services + start gRPC and management HTTP servers
	builders.RegisterAndServe(grpcServer, grpcConfig, coreResult.Service, protocol, infra, cancel)

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		log.Info(LogReceivedShutdownSignal, logger.FieldSignal, sig)
	case <-ctx.Done():
		log.Info(LogContextCancelled)
	}

	// Graceful shutdown
	log.Info(LogShuttingDownGracefully)

	// Emit service stopped event before shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownEventTimeout) // context-root: shutdown
	defer shutdownCancel()
	emitServiceStopped(shutdownCtx, log, coreResult.SystemEventAdapter, infra.TenantID, versionInfo.Version)

	// Stop gRPC server first (stops accepting new requests)
	grpcServer.Stop()

	// Protocol + infrastructure cleanups run via deferred runCleanups
	log.Info(LogShutdownComplete)
}

// runCleanups executes cleanup functions in reverse order (LIFO).
func runCleanups(cleanups []func()) {
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}
