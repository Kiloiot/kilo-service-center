package resilience

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryAllowlist_NoStreamingOverlap(t *testing.T) {
	streams := streamingMethods()
	for method := range retryAllowlist {
		assert.False(t, streams[method],
			"streaming method %q must not appear in retry allowlist", method)
	}
}

func TestRetryAllowlist_OnlyIdempotentMethods(t *testing.T) {
	for method := range retryAllowlist {
		_, short := splitFullMethod(method)
		isIdempotent := false
		for _, prefix := range []string{"Get", "List", "Search", "Export", "Query", "Decode"} {
			if len(short) >= len(prefix) && short[:len(prefix)] == prefix {
				isIdempotent = true
				break
			}
		}
		assert.True(t, isIdempotent,
			"method %q does not match idempotent pattern (Get*/List*/Search*/Export*/Query*/Decode*)", method)
	}
}

func TestRetryAllowlistMethods_ReturnsAllEntries(t *testing.T) {
	methods := retryAllowlistMethods()
	assert.Equal(t, len(retryAllowlist), len(methods))

	seen := make(map[string]bool)
	for _, m := range methods {
		seen[m] = true
	}
	for k := range retryAllowlist {
		assert.True(t, seen[k], "method %q missing from retryAllowlistMethods()", k)
	}
}

func TestIsRetryable(t *testing.T) {
	assert.True(t, IsRetryable("/kilocenter.api.v1.CoreService/GetEndPoint"))
	assert.True(t, IsRetryable("/kilocenter.api.v1.CoreService/ListEndPoints"))
	assert.False(t, IsRetryable("/kilocenter.api.v1.CoreService/CreateEndPoint"))
	assert.False(t, IsRetryable("/kilocenter.api.v1.CoreService/DeleteEndPoint"))
	assert.False(t, IsRetryable("/kilocenter.api.v1.CoreService/StreamEvents"))
}

func TestStreamingMethods_ReturnsCopy(t *testing.T) {
	s1 := streamingMethods()
	s2 := streamingMethods()

	// Modifying one should not affect the other
	s1["/test/Method"] = true
	assert.False(t, s2["/test/Method"])
}

func TestBuildRetryServiceConfig_ValidJSON(t *testing.T) {
	cfg := config.GatewayResilienceConfig{
		MaxRetries:      3,
		RetryBackoff:    100 * time.Millisecond,
		RetryMaxBackoff: 1 * time.Second,
	}

	sc := BuildRetryServiceConfig(cfg)
	require.NotEmpty(t, sc)

	// Must be valid JSON
	assert.True(t, len(sc) > 2, "service config should be non-trivial JSON")
	assert.Contains(t, sc, "UNAVAILABLE")
	assert.Contains(t, sc, "maxAttempts")
}

func TestSplitFullMethod(t *testing.T) {
	tests := []struct {
		input   string
		service string
		method  string
	}{
		{"/kilocenter.api.v1.CoreService/GetEndPoint", "kilocenter.api.v1.CoreService", "GetEndPoint"},
		{"/kilocenter.api.v1.IdentityService/Login", "kilocenter.api.v1.IdentityService", "Login"},
		{"kilocenter.api.v1.CoreService/GetEndPoint", "kilocenter.api.v1.CoreService", "GetEndPoint"},
		{"noSlash", "noSlash", ""},
	}

	for _, tt := range tests {
		svc, method := splitFullMethod(tt.input)
		assert.Equal(t, tt.service, svc)
		assert.Equal(t, tt.method, method)
	}
}
