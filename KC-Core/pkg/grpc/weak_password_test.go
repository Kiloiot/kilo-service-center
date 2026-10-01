package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/passwordpolicy"
)

// The weak-password refusal states the enforced policy in its own words.
func TestWeakPasswordMessage_IsThePolicyDescription(t *testing.T) {
	assert.Equal(t, passwordpolicy.Rules.Describe(), ResolveErrorMessage(ErrTokenWeakPassword))
}
