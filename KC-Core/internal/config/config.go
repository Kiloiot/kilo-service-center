// Package config handles configuration management for KiloCenter.
// This is a thin wrapper delegating to the exported pkg/config loader.
package config

import (
	pkgconfig "github.com/kilocenter/KC-Core/pkg/config"
)

// Config represents the complete configuration structure.
// Uses embedded pkgconfig.Config for full type compatibility with exported loader.
type Config struct {
	pkgconfig.Config
}

// Load loads configuration from file and environment variables.
// Delegates to the exported pkgconfig.Load.
func Load(configPath string) (*Config, error) {
	cfg, err := pkgconfig.Load(configPath)
	if err != nil {
		return nil, err
	}
	return &Config{Config: *cfg}, nil
}
