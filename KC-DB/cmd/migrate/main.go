// Command migrate provides database migration management for KiloCenter.
// Delegates all migration operations to MigrationRunner from the postgres package.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// Migration subcommands.
const (
	commandUp      = "up"
	commandDown    = "down"
	commandVersion = "version"
	commandForce   = "force"
)

// Development database connection defaults (mirrors docker-compose.dev.yml).
const (
	defaultDBPort     = "5432"
	defaultDBHost     = "localhost"
	defaultDBName     = "kilocenter"
	defaultDBUser     = "kilocenter"
	defaultDBPassword = "changeme"
	defaultDBSSLMode  = "disable"
)

const errFmtInvalidDBPort = "invalid DB_PORT: %w"

func main() {
	var (
		command = flag.String("command", "up", "Migration command: up, down, version, force VERSION")
		force   = flag.Int("force", -1, "Force migration to specific version")
		steps   = flag.Int("steps", 0, "Number of migrations to run (for down command)")
		to      = flag.Uint("to", 0, "Highest version to apply (for up command); never migrates down")
	)
	flag.Parse()

	config, err := buildConfig()
	if err != nil {
		log.Fatal(err)
	}
	runner, err := postgres.NewMigrationRunner(config, postgres.UpgradeGates())
	if err != nil {
		log.Fatalf("Failed to create migration runner: %v", err)
	}
	ctx := context.Background() // context-root: process

	switch *command {
	case commandUp:
		version, err := runUp(ctx, runner, *to)
		if err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Printf("Migrations applied successfully (version %d)\n", version)

	case commandDown:
		if *steps > 0 {
			if err := runner.Steps(ctx, -*steps); err != nil {
				log.Fatalf("Failed to rollback %d migrations: %v", *steps, err)
			}
			fmt.Printf("Rolled back %d migrations\n", *steps)
		} else {
			if err := runner.Down(ctx); err != nil {
				log.Fatalf("Failed to rollback all migrations: %v", err)
			}
			fmt.Println("All migrations rolled back")
		}

	case commandVersion:
		version, dirty, err := runner.Version(ctx)
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		if version == 0 {
			fmt.Println("No migrations applied yet")
			return
		}
		if dirty {
			fmt.Printf("Current version: %d (dirty)\n", version)
		} else {
			fmt.Printf("Current version: %d\n", version)
		}

	case commandForce:
		if *force < 0 {
			log.Fatal("Force requires a version number")
		}
		if err := runner.Force(ctx, *force); err != nil {
			log.Fatalf("Failed to force version %d: %v", *force, err)
		}
		fmt.Printf("Forced to version %d\n", *force)

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}

func runUp(ctx context.Context, runner *postgres.MigrationRunner, to uint) (uint, error) {
	if to > 0 {
		return runner.RunTo(ctx, to)
	}
	return runner.Run(ctx)
}

func buildConfig() (*postgres.Config, error) {
	port, err := strconv.Atoi(getEnv("DB_PORT", defaultDBPort))
	if err != nil {
		return nil, fmt.Errorf(errFmtInvalidDBPort, err)
	}
	return &postgres.Config{
		Host:     getEnv("DB_HOST", defaultDBHost),
		Port:     port,
		Database: getEnv("DB_NAME", defaultDBName),
		Username: getEnv("DB_USER", defaultDBUser),
		Password: getEnv("DB_PASSWORD", defaultDBPassword),
		SSLMode:  getEnv("DB_SSLMODE", defaultDBSSLMode),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
