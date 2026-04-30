// Package main provides the KiloCenter MIOTY Service Center entrypoint.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kilocenter/KC-Core/cmd/kilocenter/builders"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/pkg/observability"
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

	log.Info("Starting KiloCenter MIOTY Server",
		"version", versionInfo.Version,
		"build_time", versionInfo.BuildTime,
		"git_commit", versionInfo.GitCommit,
		"git_branch", versionInfo.GitBranch,
		"schema_version", versionInfo.SchemaVersion,
	)

	// Initialize observability (tracing + metrics)
	tracingShutdown, err := observability.InitTracing(context.Background(), observability.TracingConfig{
		Enabled:    cfg.Monitoring.TracingEnabled,
		Endpoint:   cfg.Monitoring.TracingEndpoint,
		SampleRate: cfg.Monitoring.TracingSampleRate,
	}, "kc-core")
	if err != nil {
		log.Error(LogFailedInitTracing, "error", err)
	} else {
		defer func() { _ = tracingShutdown(context.Background()) }()
	}

	metricsShutdown, err := observability.InitMetrics(context.Background(), observability.MetricsConfig{
		Enabled: cfg.Monitoring.MetricsEnabled,
		Port:    cfg.Monitoring.MetricsPort,
		Path:    cfg.Monitoring.MetricsPath,
	}, "kc-core")
	if err != nil {
		log.Error(LogFailedInitMetrics, "error", err)
	} else {
		defer func() { _ = metricsShutdown(context.Background()) }()
	}

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
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

	// Start health check endpoint
	go startHealthCheck(cfg.General.HealthCheckPort, infra.HealthService)

	// 2. gRPC server (needs org resolver from infra)
	grpcServer, grpcConfig, err := builders.BuildGRPCServer(infra)
	if err != nil {
		log.Fatal(LogFailedBuildGRPCServer, logger.Err(err))
	}

	// 3. Protocol servers (BSSCI + SCACI)
	protocol, err := builders.BuildProtocolServers(ctx, infra)
	if err != nil {
		log.Fatal(LogFailedBuildProtocol, logger.Err(err))
	}
	defer runCleanups(protocol.Cleanups)

	// 4. Core service (device management, protocol operations, analytics)
	coreResult := builders.BuildCoreService(ctx, infra, protocol, nil)

	// 5. Register services + start gRPC and management HTTP servers
	builders.RegisterAndServe(grpcServer, grpcConfig, coreResult.Service, protocol, infra, cancel)

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		log.Info("Context cancelled")
	}

	// Graceful shutdown
	log.Info("Shutting down gracefully...")

	// Emit service stopped event before shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	hostname, _ := os.Hostname()
	_ = coreResult.SystemEventAdapter.CreateEvent(shutdownCtx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(infra.TenantID, 10),
		EventType:   models.EventTypeServiceStopped,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStoppedFmt, "KC-Core", hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStoppedFmt, "KC-Core", versionInfo.Version),
		SourceType:  models.SourceTypeSystem,
		SourceName:  "kc-core",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	log.Info("Service stopped event emitted")

	// Stop gRPC server first (stops accepting new requests)
	grpcServer.Stop()

	// Protocol + infrastructure cleanups run via deferred runCleanups
	log.Info("Shutdown complete")
}

// runCleanups executes cleanup functions in reverse order (LIFO).
func runCleanups(cleanups []func()) {
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}
