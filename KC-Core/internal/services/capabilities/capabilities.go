// Package capabilities names the non-secret feature toggles a service center
// exposes; nothing here serializes configuration.
package capabilities

import (
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

// Capability names; each maps to exactly one configuration toggle.
const (
	EnterpriseOrganizations = "enterprise_organizations"
	SCACI                   = "scaci"
	Roaming                 = "roaming"
	FederationRelay         = "federation_relay"
	MQTTPublishing          = "mqtt_publishing"
	LocalLogin              = "local_login"
	SelfRegistration        = "self_registration"
	OIDCLogin               = "oidc_login"
	OAuth2Login             = "oauth2_login"
	RegistrySubmission      = "blueprint_registry_submission"
	GRPCWeb                 = "grpc_web"
)

// FromConfig reads the allowlisted toggles; the order is the published order.
func FromConfig(cfg *config.Config) []grpcservices.Capability {
	return []grpcservices.Capability{
		{Name: EnterpriseOrganizations, Enabled: cfg.General.Edition == config.EditionECE},
		{Name: SCACI, Enabled: cfg.Protocol.SCACIEnabled},
		{Name: Roaming, Enabled: cfg.Protocol.Roaming.Enabled},
		{Name: FederationRelay, Enabled: cfg.Protocol.Federation.Enabled},
		{Name: MQTTPublishing, Enabled: cfg.MQTT.Enabled},
		{Name: LocalLogin, Enabled: cfg.Auth.LocalLoginEnabled},
		{Name: SelfRegistration, Enabled: cfg.Auth.RegistrationEnabled},
		{Name: OIDCLogin, Enabled: cfg.Auth.OIDC.Enabled},
		{Name: OAuth2Login, Enabled: cfg.Auth.OAuth2.Enabled},
		{Name: RegistrySubmission, Enabled: cfg.RegistryProvider.Enabled},
		{Name: GRPCWeb, Enabled: cfg.GRPC.Web.Enabled},
	}
}
