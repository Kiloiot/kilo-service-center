// Command migrate provides database migration management for KiloCenter.
// Delegates all migration operations to MigrationRunner from the postgres package.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

func main() {
	var (
		command = flag.String("command", "up", "Migration command: up, down, version, force VERSION")
		force   = flag.Int("force", -1, "Force migration to specific version")
		steps   = flag.Int("steps", 0, "Number of migrations to run (for down command)")
	)
	flag.Parse()

	runner := postgres.NewMigrationRunner(buildConfig())

	switch *command {
	case "up":
		version, err := runner.Run()
		if err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Printf("Migrations applied successfully (version %d)\n", version)

	case "down":
		if *steps > 0 {
			if err := runner.Steps(-*steps); err != nil {
				log.Fatalf("Failed to rollback %d migrations: %v", *steps, err)
			}
			fmt.Printf("Rolled back %d migrations\n", *steps)
		} else {
			if err := runner.Down(); err != nil {
				log.Fatalf("Failed to rollback all migrations: %v", err)
			}
			fmt.Println("All migrations rolled back")
		}

	case "version":
		version, dirty, err := runner.Version()
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

	case "force":
		if *force < 0 {
			log.Fatal("Force requires a version number")
		}
		if err := runner.Force(*force); err != nil {
			log.Fatalf("Failed to force version %d: %v", *force, err)
		}
		fmt.Printf("Forced to version %d\n", *force)

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}

func buildConfig() *postgres.Config {
	port, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	return &postgres.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     port,
		Database: getEnv("DB_NAME", "kilocenter"),
		Username: getEnv("DB_USER", "kilocenter"),
		Password: getEnv("DB_PASSWORD", "changeme"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
