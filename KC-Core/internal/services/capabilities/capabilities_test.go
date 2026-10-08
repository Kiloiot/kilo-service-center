package capabilities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

// on is the enabled state the test switches selected toggles to.
const on = true

func TestFromConfig_ReadsEachToggleOnce(t *testing.T) {
	cfg := &config.Config{}
	cfg.General.Edition = config.EditionECE
	cfg.Protocol.SCACIEnabled = on
	cfg.Protocol.Roaming.Enabled = on
	cfg.Auth.LocalLoginEnabled = on
	cfg.GRPC.Web.Enabled = on

	got := FromConfig(cfg)
	byName := map[string]bool{}
	for _, c := range got {
		_, dup := byName[c.Name]
		require.False(t, dup, "capability %s listed twice", c.Name)
		byName[c.Name] = c.Enabled
	}
	assert.True(t, byName[EnterpriseOrganizations])
	assert.True(t, byName[SCACI])
	assert.True(t, byName[Roaming])
	assert.True(t, byName[LocalLogin])
	assert.True(t, byName[GRPCWeb])
	assert.False(t, byName[FederationRelay])
	assert.False(t, byName[MQTTPublishing])
	assert.False(t, byName[SelfRegistration])
	assert.False(t, byName[OIDCLogin])
	assert.False(t, byName[OAuth2Login])
	assert.False(t, byName[RegistrySubmission])

	ce := &config.Config{}
	for _, c := range FromConfig(ce) {
		assert.False(t, c.Enabled, "an empty configuration enables nothing (%s)", c.Name)
	}
}
