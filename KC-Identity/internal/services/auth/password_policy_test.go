package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/passwordpolicy"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testPolicyLetter = "a"
	testPolicyDigit  = "1"
)

// The policy the settings publish is the one ValidatePassword enforces.
func TestAuthSettings_PublishTheEnforcedPasswordPolicy(t *testing.T) {
	svc := &Service{}
	settings, err := svc.GetAuthSettings(testutil.TestContext())
	require.NoError(t, err)
	policy := settings.PasswordPolicy

	assert.Equal(t, passwordpolicy.Rules, policy)
	require.True(t, policy.RequiresLetter)
	require.True(t, policy.RequiresDigit)
	shortest := strings.Repeat(testPolicyLetter, int(policy.MinLength)-1) + testPolicyDigit
	longest := strings.Repeat(testPolicyLetter, int(policy.MaxLength)-1) + testPolicyDigit
	assert.NoError(t, ValidatePassword(shortest))
	assert.NoError(t, ValidatePassword(longest))
	assert.ErrorIs(t, ValidatePassword(shortest[1:]), ErrUserPasswordWeak)
	assert.ErrorIs(t, ValidatePassword(longest+testPolicyDigit), ErrUserPasswordWeak)
}
