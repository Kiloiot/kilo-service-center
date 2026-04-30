// Package config provides configuration tests for KiloCenter
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig_SCALogPingOperationsEnabled verifies that SCALogPingOperations
// defaults to true per SCACI §3.4 audit requirement.
func TestDefaultConfig_SCALogPingOperationsEnabled(t *testing.T) {
	// Create temp config file with minimal valid config (no SCACI overrides)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	minimalConfig := `
general:
  server_name: "TestServer"
storage:
  host: "localhost"
  port: 5432
  database: "test"
`
	err := os.WriteFile(configPath, []byte(minimalConfig), 0644)
	require.NoError(t, err)

	// Load config with defaults
	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify SCALogPingOperations defaults to true per SCACI §3.4
	assert.True(t, cfg.Protocol.SCALogPingOperations,
		"SCALogPingOperations should default to true for SCACI §3.4 audit requirement")
}

// TestDefaultConfig_SCALogStatusOperationsEnabled verifies that SCALogStatusOperations
// defaults to true per SCACI §3.5 audit requirement.
func TestDefaultConfig_SCALogStatusOperationsEnabled(t *testing.T) {
	// Create temp config file with minimal valid config (no SCACI overrides)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	minimalConfig := `
general:
  server_name: "TestServer"
storage:
  host: "localhost"
  port: 5432
  database: "test"
`
	err := os.WriteFile(configPath, []byte(minimalConfig), 0644)
	require.NoError(t, err)

	// Load config with defaults
	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify SCALogStatusOperations defaults to true per SCACI §3.5
	assert.True(t, cfg.Protocol.SCALogStatusOperations,
		"SCALogStatusOperations should default to true for SCACI §3.5 audit requirement")
}

// TestDefaultConfig_SCALogOperationsCanBeDisabled verifies that audit logging
// can be explicitly disabled via config for high-frequency deployments.
func TestDefaultConfig_SCALogOperationsCanBeDisabled(t *testing.T) {
	// Create temp config file with audit logging explicitly disabled
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configWithDisabledAudit := `
general:
  server_name: "TestServer"
storage:
  host: "localhost"
  port: 5432
  database: "test"
protocol:
  scaci_log_ping_operations: false
  scaci_log_status_operations: false
`
	err := os.WriteFile(configPath, []byte(configWithDisabledAudit), 0644)
	require.NoError(t, err)

	// Load config
	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify explicit false overrides defaults
	assert.False(t, cfg.Protocol.SCALogPingOperations,
		"SCALogPingOperations should be false when explicitly disabled")
	assert.False(t, cfg.Protocol.SCALogStatusOperations,
		"SCALogStatusOperations should be false when explicitly disabled")
}
