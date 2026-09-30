// Package main provides the KC-Identity service entrypoint.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/cmd/identity/builders"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

// identityDisplayName labels KC-Identity in event titles.
const identityDisplayName = "KC-Identity"

// listenInfoFmt summarizes the listening port for the started event.
const listenInfoFmt = "gRPC=%d"

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
	cfg, err := pkgconfig.LoadIdentity(configFile)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Initialize(cfg.General.LogLevel, cfg.General.LogFormat)
	log := logger.Get()

	versionInfo := getVersionInfo()
	log.Info(LogServiceStarting,
		logger.FieldVersion, versionInfo.Version,
		logger.FieldGrpcPortSnake, cfg.GRPC.Port,
		logger.FieldHealthPortSnake, cfg.General.HealthCheckPort,
	)

	shutdownObservability := initObservability(cfg, log)
	defer shutdownObservability()

	// Create application context
	ctx, cancel := context.WithCancel(context.Background()) // context-root: process
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 1. Infrastructure (storage, org resolver)
	infra, err := builders.BuildInfrastructure(ctx, cfg)
	if err != nil {
		log.Fatal(LogFailedBuildInfra, logger.FieldError, err)
	}
	defer runCleanups(infra.Cleanups)

	// Start health check endpoint
	go startHealthServer(ctx, cfg.General.HealthCheckPort, infra.DB)

	// 2. Identity service (auth, users, orgs, memberships, API keys)
	identityResult := builders.BuildIdentityService(ctx, infra)
	defer runCleanups(identityResult.Cleanups)

	// 3. Register services + start gRPC server
	grpcServer := builders.RegisterAndServe(cfg, infra, identityResult, cancel)

	events := newLifecycleEvents(infra.Repos.SystemEvents, log, infra.TenantID, versionInfo.Version)
	events.started(ctx, cfg.GRPC.Port)

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		log.Info(LogReceivedShutdownSignal, logger.FieldSignal, sig)
	case <-ctx.Done():
		log.Info(LogContextCancelled)
	}

	events.stopped()

	log.Info(LogShuttingDown)
	grpcServer.GracefulStop()
	log.Info(LogShutdownComplete)
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
