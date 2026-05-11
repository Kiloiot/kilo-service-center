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
	testOrigin             = "http://localhost:5173"
	testGRPCMethodPath     = "/test.Service/Method"
	testMethodPost         = "POST"
	testMethodOptions      = "OPTIONS"
	testMethodGet          = "GET"
	testGRPCWebHeaderValue = "1"
	testRootPath           = "/"
)

func TestGRPCWebCORSPreflight(t *testing.T) {
	_ = testutil.TestContext() // Use test context per policy

	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		allowAll       bool
		wantStatus     int
	}{
		{
			name:           "allowed origin",
			origin:         "http://localhost:5173",
			allowedOrigins: []string{"http://localhost:5173"},
			allowAll:       false,
			wantStatus:     http.StatusNoContent,
		},
		{
			name:           "disallowed origin",
			origin:         "http://evil.com",
			allowedOrigins: []string{"http://localhost:5173"},
			allowAll:       false,
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "allow all origins",
			origin:         "http://any.com",
			allowedOrigins: []string{},
			allowAll:       true,
			wantStatus:     http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := WebConfig{
				AllowedOrigins:   tt.allowedOrigins,
				AllowAllOrigins:  tt.allowAll,
				AllowCredentials: true,
				MaxAge:           config.GRPCWebDefaultMaxAgeSeconds,
				AllowedMethods:   config.GRPCWebDefaultAllowedMethods,
			}

			req := httptest.NewRequest(http.MethodOptions, "/", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handleCORSPreflight(w, req, cfg)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantStatus == http.StatusNoContent {
				assert.Equal(t, tt.origin, w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				assert.Equal(t, "Origin", w.Header().Get("Vary")) // Cache safety
			}
		})
	}
}

func TestGRPCWebVaryHeader(t *testing.T) {
	_ = testutil.TestContext()

	cfg := WebConfig{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowAllOrigins:  false,
		AllowCredentials: true,
	}

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	handleCORSPreflight(w, req, cfg)

	// Vary: Origin MUST be set to prevent cache poisoning
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

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
	assert.Equal(t, true, config.DefaultGRPCWebEnabled)
	assert.Equal(t, true, config.DefaultGRPCWebAllowCredentials)
	assert.Equal(t, true, config.DefaultGRPCWebEnableWebsockets)
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

func TestHTTPMultiplexerRouting(t *testing.T) {
	// Parse HTTP timeouts from config constants - fail fast on parse errors
	readTimeout, err := time.ParseDuration(config.HTTPDefaultReadTimeoutString)
	require.NoError(t, err, "failed to parse HTTPDefaultReadTimeoutString")
	writeTimeout, err := time.ParseDuration(config.HTTPDefaultWriteTimeoutString)
	require.NoError(t, err, "failed to parse HTTPDefaultWriteTimeoutString")
	idleTimeout, err := time.ParseDuration(config.HTTPDefaultIdleTimeoutString)
	require.NoError(t, err, "failed to parse HTTPDefaultIdleTimeoutString")

	// Test 1: gRPC-web request routes to grpc-web wrapper
	t.Run("gRPC-web request routes to wrapper", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		webCfg := WebConfig{
			Enabled:          true,
			AllowAllOrigins:  false,
			AllowedOrigins:   []string{testOrigin},
			AllowCredentials: config.DefaultGRPCWebAllowCredentials,
			AllowedMethods:   config.GRPCWebDefaultAllowedMethods,
			MaxAge:           config.GRPCWebDefaultMaxAgeSeconds,
		}
		grpcWebWrapped := createGRPCWebWrapper(grpcServer, webCfg, logger.Get())

		cfg := Config{
			GRPCWeb:   webCfg,
			EnableTLS: false,
			HTTPConfig: HTTPServerConfig{
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
				IdleTimeout:  idleTimeout,
			},
		}
		httpServer := createHTTPServer(grpcServer, grpcWebWrapped, cfg, logger.Get())

		req := httptest.NewRequest(testMethodPost, testGRPCMethodPath, nil)
		req.Header.Set(grpcerrors.HeaderContentType, grpcerrors.ContentTypeGRPCWeb)
		req.Header.Set(grpcerrors.HeaderGRPCWeb, testGRPCWebHeaderValue)
		req.Header.Set(grpcerrors.HeaderOrigin, testOrigin)

		w := httptest.NewRecorder()
		httpServer.Handler.ServeHTTP(w, req)

		// Verify gRPC-web path with dual assertions for stability:
		// 1. grpc-status header present (wrapper always sets this)
		grpcStatus := w.Header().Get(grpcerrors.HeaderGRPCStatus)
		assert.NotEmpty(t, grpcStatus, "expected grpc-status header from gRPC-web wrapper")

		// 2. Content-Type starts with gRPC-web prefix
		respContentType := w.Header().Get(grpcerrors.HeaderContentType)
		assert.True(t, strings.HasPrefix(respContentType, grpcerrors.ContentTypeGRPCWebPrefix),
			"expected gRPC-web Content-Type prefix, got: %s", respContentType)
	})

	// Test 2: Native gRPC request routes to grpcServer.ServeHTTP
	t.Run("native gRPC request routes to grpc server", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		webCfg := WebConfig{
			Enabled:          true,
			AllowAllOrigins:  false,
			AllowedOrigins:   []string{testOrigin},
			AllowCredentials: config.DefaultGRPCWebAllowCredentials,
			AllowedMethods:   config.GRPCWebDefaultAllowedMethods,
		}
		grpcWebWrapped := createGRPCWebWrapper(grpcServer, webCfg, logger.Get())

		cfg := Config{
			GRPCWeb:   webCfg,
			EnableTLS: false,
			HTTPConfig: HTTPServerConfig{
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
				IdleTimeout:  idleTimeout,
			},
		}
		httpServer := createHTTPServer(grpcServer, grpcWebWrapped, cfg, logger.Get())

		req := httptest.NewRequest(testMethodPost, testGRPCMethodPath, nil)
		req.Header.Set(grpcerrors.HeaderContentType, grpcerrors.ContentTypeGRPC)
		req.ProtoMajor = 2
		req.ProtoMinor = 0

		w := httptest.NewRecorder()
		httpServer.Handler.ServeHTTP(w, req)

		// Verify native gRPC path: response Content-Type starts with application/grpc
		respContentType := w.Header().Get(grpcerrors.HeaderContentType)
		assert.True(t, strings.HasPrefix(respContentType, grpcerrors.ContentTypeGRPC),
			"expected gRPC Content-Type prefix, got: %s", respContentType)
	})

	// Test 3: OPTIONS preflight routes to handleCORSPreflight
	t.Run("OPTIONS preflight routes to CORS handler", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		webCfg := WebConfig{
			Enabled:          true,
			AllowAllOrigins:  false,
			AllowedOrigins:   []string{testOrigin},
			AllowCredentials: config.DefaultGRPCWebAllowCredentials,
			AllowedMethods:   config.GRPCWebDefaultAllowedMethods,
			MaxAge:           config.GRPCWebDefaultMaxAgeSeconds,
		}
		grpcWebWrapped := createGRPCWebWrapper(grpcServer, webCfg, logger.Get())

		cfg := Config{
			GRPCWeb:   webCfg,
			EnableTLS: false,
			HTTPConfig: HTTPServerConfig{
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
				IdleTimeout:  idleTimeout,
			},
		}
		httpServer := createHTTPServer(grpcServer, grpcWebWrapped, cfg, logger.Get())

		req := httptest.NewRequest(testMethodOptions, testRootPath, nil)
		req.Header.Set(grpcerrors.HeaderOrigin, testOrigin)
		req.Header.Set(grpcerrors.HeaderAccessControlRequestMethod, testMethodPost)

		w := httptest.NewRecorder()
		httpServer.Handler.ServeHTTP(w, req)

		// Verify CORS preflight path: status 204, CORS headers present
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, testOrigin, w.Header().Get(grpcerrors.HeaderAccessControlAllowOrigin))
		assert.Equal(t, grpcerrors.HeaderOrigin, w.Header().Get(grpcerrors.HeaderVary))
	})

	// Test 4: Non-matching request returns 404
	t.Run("GET request returns 404", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		webCfg := WebConfig{
			Enabled:          true,
			AllowAllOrigins:  false,
			AllowedOrigins:   []string{testOrigin},
			AllowCredentials: config.DefaultGRPCWebAllowCredentials,
		}
		grpcWebWrapped := createGRPCWebWrapper(grpcServer, webCfg, logger.Get())

		cfg := Config{
			GRPCWeb:   webCfg,
			EnableTLS: false,
			HTTPConfig: HTTPServerConfig{
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
				IdleTimeout:  idleTimeout,
			},
		}
		httpServer := createHTTPServer(grpcServer, grpcWebWrapped, cfg, logger.Get())

		req := httptest.NewRequest(testMethodGet, testRootPath, nil)

		w := httptest.NewRecorder()
		httpServer.Handler.ServeHTTP(w, req)

		// Verify 404 path: status 404, text/plain Content-Type
		assert.Equal(t, http.StatusNotFound, w.Code)
		respContentType := w.Header().Get(grpcerrors.HeaderContentType)
		assert.Equal(t, grpcerrors.ContentTypePlainText, respContentType)
	})
}
