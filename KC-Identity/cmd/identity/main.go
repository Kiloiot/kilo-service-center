// Package main provides the KC-Identity service entrypoint.
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

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/cmd/identity/builders"
	"github.com/Kiloiot/kilo-service-center/pkg/observability"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

func main() {
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showVersion {
		info := getVersionInfo()
		fmt.Printf("KiloCenter Identity Service\n")
		fmt.Printf("Version: %s\n", info.Version)
		fmt.Printf("Build Time: %s\n", info.BuildTime)
		fmt.Printf("Git Commit: %s\n", info.GitCommit)
		fmt.Printf("Git Branch: %s\n", info.GitBranch)
		os.Exit(0)
	}

	// Load shared configuration (reuses KC-Core/pkg/config loader)
	configFile := *configPath
	if configFile == "" {
		configFile = os.Getenv("KILOCENTER_CONFIG_FILE")
	}
	cfg, err := pkgconfig.Load(configFile)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Initialize(cfg.General.LogLevel, cfg.General.LogFormat)
	log := logger.Get()

	versionInfo := getVersionInfo()
	log.Info("Starting KC-Identity Service",
		"version", versionInfo.Version,
		"grpc_port", cfg.GRPC.Port,
		"health_port", cfg.General.HealthCheckPort,
	)

	// Initialize observability (tracing + metrics)
	tracingShutdown, err := observability.InitTracing(context.Background(), observability.TracingConfig{
		Enabled:    cfg.Monitoring.TracingEnabled,
		Endpoint:   cfg.Monitoring.TracingEndpoint,
		SampleRate: cfg.Monitoring.TracingSampleRate,
	}, identityServiceName)
	if err != nil {
		log.Error("Failed to init tracing", "error", err)
	} else {
		defer func() { _ = tracingShutdown(context.Background()) }()
	}

	metricsShutdown, err := observability.InitMetrics(context.Background(), observability.MetricsConfig{
		Enabled: cfg.Monitoring.MetricsEnabled,
		Port:    cfg.Monitoring.MetricsPort,
		Path:    cfg.Monitoring.MetricsPath,
	}, identityServiceName)
	if err != nil {
		log.Error("Failed to init metrics", "error", err)
	} else {
		defer func() { _ = metricsShutdown(context.Background()) }()
	}

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 1. Infrastructure (storage, org resolver)
	infra, err := builders.BuildInfrastructure(ctx, cfg)
	if err != nil {
		log.Fatal("Failed to build infrastructure", "error", err)
	}
	defer runCleanups(infra.Cleanups)

	// Start health check endpoint
	go startHealthServer(ctx, cfg.General.HealthCheckPort, infra.DB)

	// 2. Identity service (auth, users, orgs, memberships, API keys)
	identityResult := builders.BuildIdentityService(infra)
	defer runCleanups(identityResult.Cleanups)

	// 3. Register services + start gRPC server
	grpcServer := builders.RegisterAndServe(cfg, infra, identityResult, cancel)

	// Emit service started event
	hostname, _ := os.Hostname()
	listenInfo := fmt.Sprintf("gRPC=%d", cfg.GRPC.Port)
	_ = infra.Storage.SystemEvents().CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(infra.TenantID, 10),
		EventType:   models.EventTypeServiceStarted,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStartedFmt, "KC-Identity", hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStartedFmt, "KC-Identity", versionInfo.Version, listenInfo),
		SourceType:  models.SourceTypeSystem,
		SourceName:  identityServiceName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	log.Info("Service started event emitted")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		log.Info("Context cancelled")
	}

	// Emit service stopped event before shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = infra.Storage.SystemEvents().CreateEvent(shutdownCtx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(infra.TenantID, 10),
		EventType:   models.EventTypeServiceStopped,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStoppedFmt, "KC-Identity", hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStoppedFmt, "KC-Identity", versionInfo.Version),
		SourceType:  models.SourceTypeSystem,
		SourceName:  identityServiceName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	log.Info("Service stopped event emitted")

	log.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
	log.Info("Shutdown complete")
}

// runCleanups executes cleanup functions in reverse order (LIFO).
func runCleanups(cleanups []func()) {
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}

// getVersionInfo returns release manifest or exits with fatal error.
func getVersionInfo() *version.Info {
	info, err := version.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to load release manifest: %v\n", err)
		os.Exit(1)
	}
	return info
}
