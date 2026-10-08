package grpc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Test constants - no inline literals per governance.
const (
	testOrigin         = "http://localhost:5173"
	testGRPCMethodPath = "/test.Service/Method"
	testRootPath       = "/"

	// Multiplexer scenario toggles.
	testGRPCWebEnabled     = true
	testAllowAllOriginsOff = false
	testTLSDisabled        = false
)

// Wire values the gRPC-web transport must emit; asserted here, not exported.
const (
	testContentTypeGRPCWebPrefix         = "application/grpc-web"
	testContentTypeGRPCWeb               = "application/grpc-web+proto"
	testHeaderAccessControlRequestMethod = "Access-Control-Request-Method"
)

func TestGRPCWebAllowedHeaders(t *testing.T) {
	// Verify default headers include required metadata keys
	headers := grpcerrors.GRPCWebAllowedHeaders

	require.Contains(t, headers, grpcerrors.MetadataKeyAuthorization)
	require.Contains(t, headers, grpcerrors.MetadataKeyTenantID)
	require.Contains(t, headers, grpcerrors.MetadataKeyOrganizationID)
	require.Contains(t, headers, grpcerrors.MetadataKeyUserID)
}

func TestGRPCWebExposeHeaders(t *testing.T) {
	// Verify response headers include grpc status headers
	headers := grpcerrors.GRPCWebExposeHeaders

	require.Contains(t, headers, "grpc-status")
	require.Contains(t, headers, "grpc-message")
}

func TestMetadataConstantsNotDuplicated(t *testing.T) {
	// Verify the canonical constants exist in pkg/grpc
	// These should be the ONLY source of truth
	assert.Equal(t, "x-organization-id", grpcerrors.MetadataKeyOrganizationID)
	assert.Equal(t, "x-user-id", grpcerrors.MetadataKeyUserID)
	assert.Equal(t, "x-tenant-id", grpcerrors.MetadataKeyTenantID)
	assert.Equal(t, "authorization", grpcerrors.MetadataKeyAuthorization)
	assert.Equal(t, "Bearer ", grpcerrors.BearerPrefix)
}

func TestWebConfigDefaults(t *testing.T) {
	// Verify config defaults are correctly set
	assert.Equal(t, false, config.DefaultGRPCWebEnabled)
	assert.Equal(t, true, config.DefaultGRPCWebAllowCredentials)
	assert.Equal(t, 3600, config.GRPCWebDefaultMaxAgeSeconds)
	assert.Equal(t, []string{"POST", "OPTIONS"}, config.GRPCWebDefaultAllowedMethods)

	// Security: AllowAllOrigins MUST be false by default when AllowCredentials is true
	assert.Equal(t, false, config.DefaultGRPCWebAllowAllOrigins, "DefaultGRPCWebAllowAllOrigins must be false for security")
}

func TestHTTPTimeoutDefaults(t *testing.T) {
	// Verify HTTP timeout defaults are correctly set
	assert.Equal(t, "30s", config.HTTPDefaultReadTimeoutString)
	assert.Equal(t, "30s", config.HTTPDefaultWriteTimeoutString)
	assert.Equal(t, "120s", config.HTTPDefaultIdleTimeoutString)
}

// KC-Core's HTTP server routes gRPC-web and its CORS preflight through the
// shared multiplexer with the configured gRPC-web policy.
func TestCreateHTTPServer_ServesGRPCWebAndPreflight(t *testing.T) {
	readTimeout, err := time.ParseDuration(config.HTTPDefaultReadTimeoutString)
	require.NoError(t, err)
	writeTimeout, err := time.ParseDuration(config.HTTPDefaultWriteTimeoutString)
	require.NoError(t, err)
	idleTimeout, err := time.ParseDuration(config.HTTPDefaultIdleTimeoutString)
	require.NoError(t, err)
	server := createHTTPServer(grpc.NewServer(), Config{
		GRPCWeb: config.GRPCWebConfig{
			Enabled:          testGRPCWebEnabled,
			AllowAllOrigins:  testAllowAllOriginsOff,
			AllowedOrigins:   []string{testOrigin},
			AllowCredentials: config.DefaultGRPCWebAllowCredentials,
			AllowedMethods:   config.GRPCWebDefaultAllowedMethods,
			MaxAge:           config.GRPCWebDefaultMaxAgeSeconds,
		},
		EnableTLS:  testTLSDisabled,
		HTTPConfig: HTTPServerConfig{ReadTimeout: readTimeout, WriteTimeout: writeTimeout, IdleTimeout: idleTimeout},
	}, logger.Get())

	assert.Equal(t, writeTimeout, server.WriteTimeout)

	preflight := httptest.NewRequestWithContext(testutil.TestContext(), http.MethodOptions, testRootPath, nil)
	preflight.Header.Set(grpcerrors.HeaderOrigin, testOrigin)
	preflight.Header.Set(testHeaderAccessControlRequestMethod, http.MethodPost)
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, preflight)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, testOrigin, w.Header().Get(grpcerrors.HeaderAccessControlAllowOrigin))

	web := httptest.NewRequestWithContext(testutil.TestContext(), http.MethodPost, testGRPCMethodPath, nil)
	web.Header.Set(grpcerrors.HeaderContentType, testContentTypeGRPCWeb)
	web.Header.Set(grpcerrors.HeaderOrigin, testOrigin)
	w = httptest.NewRecorder()
	server.Handler.ServeHTTP(w, web)
	assert.True(t, strings.HasPrefix(w.Header().Get(grpcerrors.HeaderContentType), testContentTypeGRPCWebPrefix))
}
