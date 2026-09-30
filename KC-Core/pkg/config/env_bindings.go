package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// envBinding maps a config key to the environment variable that sets it.
type envBinding struct {
	key string
	env string
}

// registryProviderEnv binds the registry_provider keys explicitly: AutomaticEnv
// cannot tell "registry_provider.token" from "registry.provider.token" when
// mapping KILOCENTER_REGISTRY_PROVIDER_TOKEN.
var registryProviderEnv = []envBinding{
	{key: "registry_provider.token", env: "KILOCENTER_REGISTRY_PROVIDER_TOKEN"},
	{key: "registry_provider.owner", env: "KILOCENTER_REGISTRY_PROVIDER_OWNER"},
	{key: "registry_provider.repo", env: "KILOCENTER_REGISTRY_PROVIDER_REPO"},
	{key: "registry_provider.github_app_id", env: "KILOCENTER_REGISTRY_PROVIDER_GITHUB_APP_ID"},
	{key: "registry_provider.github_app_installation_id", env: "KILOCENTER_REGISTRY_PROVIDER_GITHUB_APP_INSTALLATION_ID"},
	{key: "registry_provider.github_app_private_key", env: "KILOCENTER_REGISTRY_PROVIDER_GITHUB_APP_PRIVATE_KEY"},
	{key: "registry_provider.blueprint_path", env: "KILOCENTER_REGISTRY_PROVIDER_BLUEPRINT_PATH"},
}

func bindRegistryProviderEnv(v *viper.Viper) error {
	for _, binding := range registryProviderEnv {
		if err := v.BindEnv(binding.key, binding.env); err != nil {
			return fmt.Errorf(errConfigBindEnvFmt, binding.env, err)
		}
	}
	return nil
}
