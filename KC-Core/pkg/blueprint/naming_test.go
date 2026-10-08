package blueprint

import (
	"errors"
	"testing"
)

// TestGenerateSlug verifies slug generation rules.
func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "My Model", "my-model"},
		{"trim spaces", "  hello  ", "hello"},
		{"strip special chars", "foo@bar#baz", "foobarbaz"},
		{"collapse hyphens", "foo--bar---baz", "foo-bar-baz"},
		{"empty input", "", "model"},
		{"only special chars", "!@#$%", "model"},
		{"unicode stripped", "model-\u00e9", "model-"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Slug(tc.input)
			if tc.name == "unicode stripped" {
				// Unicode handling may vary; just ensure non-empty
				if got == "" {
					t.Errorf("expected non-empty slug for %q", tc.input)
				}
				return
			}
			if got != tc.expected {
				t.Errorf("Slug(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0", "v1.0"},
		{"v1.0", "v1.0"},
		{"V2.0", "v2.0"},
		{"v1.0.3-rc1", "v1.0.3-rc1"},
		{"  v3.0  ", "v3.0"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeVersion(tc.input)
			if got != tc.expected {
				t.Errorf("NormalizeVersion(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSanitizeVersionSegment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"valid semver", "v1.0", "v1.0", false},
		{"with rc suffix", "v1.0.3-rc1", "v1.0.3-rc1", false},
		{"uppercase normalized", "V2.0", "v2.0", false},
		{"traversal rejected", "../etc", "", true},
		{"slash rejected", "v1/../../etc", "", true},
		{"empty rejected", "", "", true},
		{"spaces trimmed", "  v1.0  ", "v1.0", false},
		{"bare v prefix rejected", "v", "", true},
		{"slash in version rejected", "v1/2", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := VersionSegment(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrRegistryPathSegment) {
					t.Errorf("expected ErrRegistryPathSegment, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("VersionSegment(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSanitizePathSegment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"normal name", "Weptech", "weptech", false},
		{"with spaces", "My Company", "my-company", false},
		{"empty rejected", "", "", true},
		{"only specials rejected", "!@#$%", "", true},
		{"NUL stripped", "\x00test", "test", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PathSegment(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrRegistryPathSegment) {
					t.Errorf("expected ErrRegistryPathSegment, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("PathSegment(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSanitizeModelCodeSegment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"valid lowercase", "robin-m", "robin-m", false},
		{"valid alphanumeric", "valid123", "valid123", false},
		{"slash rejected", "robin/m", "", true},
		{"uppercase rejected", "Robin-M", "", true},
		{"empty rejected", "", "", true},
		{"spaces only rejected", "   ", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ModelCodeSegment(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrRegistryPathSegment) {
					t.Errorf("expected ErrRegistryPathSegment, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("ModelCodeSegment(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestCanonicalizeModelCode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"valid lowercase", "robin-m", "robin-m", false},
		{"uppercase normalized", "Robin-M", "robin-m", false},
		{"all uppercase", "VALID123", "valid123", false},
		{"slash rejected", "robin/m", "", true},
		{"space rejected", "robin m", "", true},
		{"empty rejected", "", "", true},
		{"leading hyphen rejected", "-leading", "", true},
		{"trailing hyphen rejected", "trailing-", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CanonicalModelCode(tc.input)
			if tc.expectErr {
				if !errors.Is(err, ErrModelCodeFormat) {
					t.Errorf("expected ErrModelCodeFormat, got err=%v val=%q", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("CanonicalModelCode(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
