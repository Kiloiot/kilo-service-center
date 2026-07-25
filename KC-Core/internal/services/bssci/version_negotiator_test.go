package bssciservices

import (
	"fmt"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestNewVersionNegotiatorSetValidation verifies the constructor rejects
// empty, malformed, and duplicate supported-version sets.
func TestNewVersionNegotiatorSetValidation(t *testing.T) {
	tests := []struct {
		name      string
		supported []string
		expectErr bool
	}{
		{name: "canonical_set", supported: []string{mioty.MIOTYProtocolVersion}, expectErr: false},
		{name: "multiple_versions", supported: []string{"1.0.0", "1.1.2"}, expectErr: false},
		{name: "empty_set", supported: nil, expectErr: true},
		{name: "malformed_member", supported: []string{"1.0"}, expectErr: true},
		{name: "signed_component", supported: []string{"1.-1.0"}, expectErr: true},
		{name: "duplicate_member", supported: []string{"1.0.0", "1.0.0"}, expectErr: true},
		{name: "duplicate_after_normalization", supported: []string{"1.0.0", "01.0.0"}, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			neg, err := NewVersionNegotiator(tt.supported, &mockLogger{})
			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, neg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, neg)
			}
		})
	}

	t.Run("nil_logger", func(t *testing.T) {
		_, err := NewVersionNegotiator([]string{"1.0.0"}, nil)
		require.Error(t, err)
	})
}

// TestNegotiateSelection verifies BSSCI rev1 §4.1-§4.3 and §5.3.2: the
// selected version is always an exact member of the supported set, newer
// minors negotiate down, lower minors and different majors are rejected.
// Addresses BSSCI-2.1-01, BSSCI-2.2-01, BSSCI-2.2-02.
func TestNegotiateSelection(t *testing.T) {
	tests := []struct {
		name      string
		supported []string
		requested string
		selected  string
		errToken  string
	}{
		{name: "exact_match", supported: []string{"1.0.0"}, requested: "1.0.0", selected: "1.0.0"},
		{name: "patch_difference_ignored", supported: []string{"1.0.0"}, requested: "1.0.99", selected: "1.0.0"},
		{name: "newer_minor_negotiated_down", supported: []string{"1.0.0"}, requested: "1.1.0", selected: "1.0.0"},
		{name: "much_newer_minor_negotiated_down", supported: []string{"1.0.0"}, requested: "1.5.7", selected: "1.0.0"},
		{name: "requested_patch_never_echoed", supported: []string{"1.0.0"}, requested: "1.0.5", selected: "1.0.0"},
		{name: "highest_supported_minor_wins", supported: []string{"1.0.0", "1.1.2"}, requested: "1.2.0", selected: "1.1.2"},
		{name: "equal_minor_selects_highest_patch", supported: []string{"1.1.0", "1.1.2"}, requested: "1.1.9", selected: "1.1.2"},
		{name: "lower_major_rejected", supported: []string{"1.0.0"}, requested: "0.9.0", errToken: bssci.ErrUnsupportedMajorVersion},
		{name: "higher_major_rejected", supported: []string{"1.0.0"}, requested: "2.0.0", errToken: bssci.ErrUnsupportedMajorVersion},
		{name: "lower_minor_than_all_supported_rejected", supported: []string{"1.1.0"}, requested: "1.0.0", errToken: bssci.ErrUnsupportedMinorVersion},
		{name: "malformed_request_rejected", supported: []string{"1.0.0"}, requested: "1.0", errToken: bssci.ErrInvalidVersionFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			neg, err := NewVersionNegotiator(tt.supported, &mockLogger{})
			require.NoError(t, err)

			selected, negErr := neg.Negotiate(testutil.TestContext(), tt.requested)
			if tt.errToken != "" {
				require.Error(t, negErr)
				catErr, ok := negErr.(*bssci.CatalogError)
				require.True(t, ok, "Error should be CatalogError")
				assert.Equal(t, tt.errToken, catErr.Token)
				assert.Equal(t, bssci.POSIX_EPROTO, catErr.Posix)
				return
			}
			require.NoError(t, negErr)
			assert.Equal(t, tt.selected, selected)
			assert.Contains(t, tt.supported, selected,
				"selected version must be an exact member of the supported set")
		})
	}
}

// TestNegotiatePatchIgnored verifies BSSCI §4.3: patch differences never
// affect compatibility. Addresses BSSCI-2.3-01, BSSCI-2.3-02.
func TestNegotiatePatchIgnored(t *testing.T) {
	neg, err := NewVersionNegotiator([]string{mioty.MIOTYProtocolVersion}, &mockLogger{})
	require.NoError(t, err)

	scMajor, scMinor, _, cerr := bssci.ParseVersion(mioty.MIOTYProtocolVersion)
	require.Nil(t, cerr)

	for _, patch := range []int{0, 1, 5, 99} {
		requested := fmt.Sprintf("%d.%d.%d", scMajor, scMinor, patch)
		t.Run(fmt.Sprintf("patch_%d", patch), func(t *testing.T) {
			selected, negErr := neg.Negotiate(testutil.TestContext(), requested)
			require.NoError(t, negErr, "Patch diff should not cause incompatibility (BSSCI-2.3-01)")
			assert.Equal(t, mioty.MIOTYProtocolVersion, selected)
		})
	}
}

// TestParseVersionFormat verifies strict version parsing: exactly three
// unsigned ASCII-decimal components; whitespace, signs, missing or extra
// components, and overflow are rejected with component-specific tokens.
func TestParseVersionFormat(t *testing.T) {
	t.Run("valid_version", func(t *testing.T) {
		maj, minor, pat, cerr := bssci.ParseVersion("1.0.0")
		require.Nil(t, cerr)
		assert.Equal(t, 1, maj)
		assert.Equal(t, 0, minor)
		assert.Equal(t, 0, pat)
	})

	formatCases := map[string]string{
		"missing_component": "1.0",
		"extra_component":   "1.0.0.0",
		"empty_string":      "",
	}
	for name, input := range formatCases {
		t.Run(name, func(t *testing.T) {
			_, _, _, cerr := bssci.ParseVersion(input)
			require.NotNil(t, cerr, "Invalid format should return CatalogError")
			assert.Equal(t, bssci.ErrInvalidVersionFormat, cerr.Token)
			assert.Equal(t, bssci.POSIX_EPROTO, cerr.Posix)
		})
	}

	majorCases := map[string]string{
		"non_numeric_major": "v1.0.0",
		"signed_major":      "+1.0.0",
		"negative_major":    "-1.0.0",
		"whitespace_major":  " 1.0.0",
		"empty_major":       ".0.0",
		"overflow_major":    "99999999999999999999.0.0",
	}
	for name, input := range majorCases {
		t.Run(name, func(t *testing.T) {
			_, _, _, cerr := bssci.ParseVersion(input)
			require.NotNil(t, cerr)
			assert.Equal(t, bssci.ErrInvalidMajorVersion, cerr.Token)
			assert.Equal(t, bssci.POSIX_EPROTO, cerr.Posix)
		})
	}

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

	t.Run("trailing_whitespace_patch", func(t *testing.T) {
		_, _, _, cerr := bssci.ParseVersion("1.0.0 ")
		require.NotNil(t, cerr)
		assert.Equal(t, bssci.ErrInvalidPatchVersion, cerr.Token)
	})
}
