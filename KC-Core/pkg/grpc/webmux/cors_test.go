package webmux

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	testOrigin           = "http://localhost:5173"
	testForeignOrigin    = "http://evil.example"
	testAnyOrigin        = "http://any.example"
	testRootPath         = "/"
	testCustomHeader     = "x-custom"
	testCustomExpose     = "x-exposed"
	testListSeparator    = ", "
	testCredentialsOff   = false
	testAllowAllOriginOn = true
	testGRPCWebOn        = true
	testGRPCWebOff       = false
	testMaxAgeUnset      = 0
)

// testWebConfig enables gRPC-web for testOrigin with the shipped defaults.
func testWebConfig() config.GRPCWebConfig {
	return config.GRPCWebConfig{
		Enabled:          testGRPCWebOn,
		AllowedOrigins:   []string{testOrigin},
		AllowCredentials: config.DefaultGRPCWebAllowCredentials,
		AllowedMethods:   config.GRPCWebDefaultAllowedMethods,
		MaxAge:           config.GRPCWebDefaultMaxAgeSeconds,
	}
}

func servePreflight(t *testing.T, cfg config.GRPCWebConfig, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(testutil.TestContext(), http.MethodOptions, testRootPath, nil)
	req.Header.Set(grpcconst.HeaderOrigin, origin)
	w := httptest.NewRecorder()
	preflight{cfg: cfg}.ServeHTTP(w, req)
	return w
}

func TestPreflight_OriginPolicy(t *testing.T) {
	allowAll := testWebConfig()
	allowAll.AllowedOrigins = nil
	allowAll.AllowAllOrigins = testAllowAllOriginOn

	tests := []struct {
		name       string
		cfg        config.GRPCWebConfig
		origin     string
		wantStatus int
	}{
		{name: "allowed origin", cfg: testWebConfig(), origin: testOrigin, wantStatus: http.StatusNoContent},
		{name: "disallowed origin", cfg: testWebConfig(), origin: testForeignOrigin, wantStatus: http.StatusForbidden},
		{name: "allow all origins", cfg: allowAll, origin: testAnyOrigin, wantStatus: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := servePreflight(t, tt.cfg, tt.origin)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantStatus != http.StatusNoContent {
				assert.Empty(t, w.Header().Get(grpcconst.HeaderAccessControlAllowOrigin))
				return
			}
			assert.Equal(t, tt.origin, w.Header().Get(grpcconst.HeaderAccessControlAllowOrigin))
			assert.Equal(t, grpcconst.HeaderOrigin, w.Header().Get(grpcconst.HeaderVary))
		})
	}
}

func TestPreflight_DefaultHeaderPolicy(t *testing.T) {
	w := servePreflight(t, testWebConfig(), testOrigin)

	assert.Equal(t, strings.Join(config.GRPCWebDefaultAllowedMethods, testListSeparator),
		w.Header().Get(grpcconst.HeaderAccessControlAllowMethods))
	assert.Equal(t, strings.Join(grpcconst.GRPCWebAllowedHeaders, testListSeparator),
		w.Header().Get(grpcconst.HeaderAccessControlAllowHeaders))
	assert.Equal(t, strings.Join(grpcconst.GRPCWebExposeHeaders, testListSeparator),
		w.Header().Get(grpcconst.HeaderAccessControlExposeHeaders))
	assert.Equal(t, grpcconst.HeaderValueTrue, w.Header().Get(grpcconst.HeaderAccessControlAllowCredentials))
	assert.Equal(t, strconv.Itoa(config.GRPCWebDefaultMaxAgeSeconds), w.Header().Get(grpcconst.HeaderAccessControlMaxAge))
}

func TestPreflight_ConfiguredHeaderPolicy(t *testing.T) {
	cfg := testWebConfig()
	cfg.AllowedHeaders = []string{testCustomHeader}
	cfg.ExposeHeaders = []string{testCustomExpose}
	cfg.AllowedMethods = nil
	cfg.AllowCredentials = testCredentialsOff
	cfg.MaxAge = testMaxAgeUnset

	w := servePreflight(t, cfg, testOrigin)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, testCustomHeader, w.Header().Get(grpcconst.HeaderAccessControlAllowHeaders))
	assert.Equal(t, testCustomExpose, w.Header().Get(grpcconst.HeaderAccessControlExposeHeaders))
	assert.NotContains(t, w.Header(), grpcconst.HeaderAccessControlAllowMethods)
	assert.NotContains(t, w.Header(), grpcconst.HeaderAccessControlAllowCredentials)
	assert.NotContains(t, w.Header(), grpcconst.HeaderAccessControlMaxAge)
}
