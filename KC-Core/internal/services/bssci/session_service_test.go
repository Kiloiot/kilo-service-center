package bssciservices

import (
	"context"
	"fmt"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateVersionMajorMinorCompatibility verifies BSSCI §2.1 and §2.2 compliance
// Addresses BSSCI-2.1-01, BSSCI-2.2-01, BSSCI-2.2-02
func TestValidateVersionMajorMinorCompatibility(t *testing.T) {
	svc := NewSessionService(nil, nil, nil, 0, &mockLogger{})

	// Parse server version to derive test cases
	serverMaj, serverMin, _, cerr := bssci.ParseVersion(mioty.MIOTYProtocolVersion)
	require.Nil(t, cerr, "Server version should parse correctly")

	serverVer := mioty.MIOTYProtocolVersion
	compatVer := fmt.Sprintf("%d.%d.%d", serverMaj, serverMin, 99)
	incompatMajor := fmt.Sprintf("%d.%d.%d", serverMaj+1, 0, 0)
	incompatMinor := fmt.Sprintf("%d.%d.%d", serverMaj, serverMin+1, 0)

	tests := []struct {
		name        string
		ver         string
		expectError bool
		errToken    string
	}{
		{
			name:        "exact_version_match",
			ver:         serverVer,
			expectError: false,
		},
		{
			name:        "patch_difference_compatible",
			ver:         compatVer,
			expectError: false,
		},
		{
			name:        "major_version_incompatible",
			ver:         incompatMajor,
			expectError: true,
			errToken:    bssci.ErrUnsupportedMajorVersion,
		},
		{
			name:        "minor_version_incompatible",
			ver:         incompatMinor,
			expectError: true,
			errToken:    bssci.ErrUnsupportedMinorVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateVersion(tt.ver)

			if tt.expectError {
				require.Error(t, err)
				catErr, ok := err.(*bssci.CatalogError)
				require.True(t, ok, "Error should be CatalogError")
				assert.Equal(t, tt.errToken, catErr.Token)
				assert.Equal(t, bssci.POSIX_EPROTO, catErr.Posix)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateVersionPatchIgnored verifies BSSCI §2.3 compliance
// Addresses BSSCI-2.3-01, BSSCI-2.3-02
func TestValidateVersionPatchIgnored(t *testing.T) {
	svc := NewSessionService(nil, nil, nil, 0, &mockLogger{})

	serverMaj, serverMin, _, cerr := bssci.ParseVersion(mioty.MIOTYProtocolVersion)
	require.Nil(t, cerr)

	for _, patch := range []int{0, 1, 5, 99} {
		ver := fmt.Sprintf("%d.%d.%d", serverMaj, serverMin, patch)
		t.Run(fmt.Sprintf("patch_%d", patch), func(t *testing.T) {
			err := svc.ValidateVersion(ver)
			assert.NoError(t, err, "Patch diff should not cause incompatibility (BSSCI-2.3-01)")
		})
	}
}

// TestParseVersionFormat verifies semantic version parsing
func TestParseVersionFormat(t *testing.T) {
	t.Run("valid_version", func(t *testing.T) {
		maj, minor, pat, cerr := bssci.ParseVersion("1.0.0")
		require.Nil(t, cerr)
		assert.Equal(t, 1, maj)
		assert.Equal(t, 0, minor)
		assert.Equal(t, 0, pat)
	})

	t.Run("invalid_format", func(t *testing.T) {
		_, _, _, cerr := bssci.ParseVersion("1.0")
		require.NotNil(t, cerr, "Invalid format should return CatalogError")
		assert.Equal(t, bssci.ErrInvalidVersionFormat, cerr.Token)
		assert.Equal(t, bssci.POSIX_EPROTO, cerr.Posix)
	})

	t.Run("invalid_major", func(t *testing.T) {
		_, _, _, cerr := bssci.ParseVersion("v1.0.0")
		require.NotNil(t, cerr)
		assert.Equal(t, bssci.ErrInvalidMajorVersion, cerr.Token)
		assert.Equal(t, bssci.POSIX_EPROTO, cerr.Posix)
	})

	t.Run("invalid_minor", func(t *testing.T) {
		_, _, _, cerr := bssci.ParseVersion("1.x.0")
		require.NotNil(t, cerr)
		assert.Equal(t, bssci.ErrInvalidMinorVersion, cerr.Token)
		assert.Equal(t, bssci.POSIX_EPROTO, cerr.Posix)
	})

	t.Run("invalid_patch", func(t *testing.T) {
		_, _, _, cerr := bssci.ParseVersion("1.0.beta")
		require.NotNil(t, cerr)
		assert.Equal(t, bssci.ErrInvalidPatchVersion, cerr.Token)
		assert.Equal(t, bssci.POSIX_EPROTO, cerr.Posix)
	})
}

type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...interface{})                           {}
func (m *mockLogger) Info(_ string, _ ...interface{})                            {}
func (m *mockLogger) Warn(_ string, _ ...interface{})                            {}
func (m *mockLogger) Error(_ string, _ ...interface{})                           {}
func (m *mockLogger) Fatal(_ string, _ ...interface{})                           {}
func (m *mockLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) WithField(_ string, _ interface{}) logger.Logger            { return m }
func (m *mockLogger) WithFields(_ map[string]interface{}) logger.Logger          { return m }
