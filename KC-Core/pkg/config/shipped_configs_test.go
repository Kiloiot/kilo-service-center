package config

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// shippedConfigFiles are the configuration files the repository ships for
// KC-Core, KC-Identity and KC-Gateway, relative to this package.
var shippedConfigFiles = []string{
	"../../config.yaml",
	"../../config.ce-test.yaml",
	"../../config/config.yaml",
	"../../../KC-Identity/config.yaml",
	"../../../KC-Gateway/config.yaml",
	"../../../config/config.yaml",
	"../../../config/config.docker.yaml",
	"../../../config/config.identity-docker.yaml",
	"../../../config/config.gateway-docker.yaml",
}

// TestShippedConfigFilesHaveNoUnknownKeys fails on any key the loader would
// silently ignore, so a shipped file never suggests a setting that has no
// effect.
func TestShippedConfigFilesHaveNoUnknownKeys(t *testing.T) {
	for _, path := range shippedConfigFiles {
		t.Run(filepath.Clean(path), func(t *testing.T) {
			v := viper.New()
			v.SetConfigFile(path)
			if err := v.ReadInConfig(); err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			var cfg Config
			if err := v.UnmarshalExact(&cfg); err != nil {
				t.Errorf("%s carries keys the loader ignores: %v", path, err)
			}
		})
	}
}
