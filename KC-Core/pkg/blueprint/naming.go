package blueprint

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Naming sentinels callers match with errors.Is; their messages come from the catalog.
var (
	ErrSpecInvalidTypeEUI  = errors.New(ResolveErrorMessage(ErrInvalidTypeEUIFormat))
	ErrSpecMissingTypeEUI  = errors.New(ResolveErrorMessage(ErrMissingTypeEUI))
	ErrRegistryPathSegment = errors.New(ResolveErrorMessage(ErrInvalidRegistryPathSegment))
	ErrModelCodeFormat     = errors.New(ResolveErrorMessage(ErrInvalidModelCode))
)

const (
	// fallbackModelSlug stands in when a name yields no slug characters.
	fallbackModelSlug = "model"
	errOpParseSpec    = "parse spec"
)

// slugRegex strips non-alphanumeric characters (except hyphens) for slug generation.
var slugRegex = regexp.MustCompile(`[^a-z0-9-]`)

// versionSegmentRegex allows only characters valid in semver path segments.
var versionSegmentRegex = regexp.MustCompile(`[^a-z0-9._-]`)

// modelCodeRegex matches characters NOT allowed in persisted model codes.
var modelCodeRegex = regexp.MustCompile(`[^a-z0-9-]`)

// Slug derives a lowercase hyphenated slug from a display name.
func Slug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", ModelCodeSeparator)
	slug = slugRegex.ReplaceAllString(slug, "")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > ModelCodeMaxLength {
		slug = slug[:ModelCodeMaxLength]
	}
	if slug == "" {
		slug = fallbackModelSlug
	}
	return slug
}

// CanonicalModelCode validates and lowercases a persisted model code.
func CanonicalModelCode(code string) (string, error) {
	canonical := strings.ToLower(strings.TrimSpace(code))
	if canonical == "" || modelCodeRegex.MatchString(canonical) ||
		strings.Contains(canonical, "..") ||
		strings.HasPrefix(canonical, "-") || strings.HasSuffix(canonical, "-") {
		return "", ErrModelCodeFormat
	}
	if len(canonical) > ModelCodeMaxLength {
		return "", ErrModelCodeFormat
	}
	return canonical, nil
}

// TypeEUIFromSpec reads the mandatory typeEui field of a blueprint spec.
func TypeEUIFromSpec(specJSON []byte) ([]byte, error) {
	var peek struct {
		TypeEUI string `json:"typeEui"`
	}
	if err := json.Unmarshal(specJSON, &peek); err != nil {
		return nil, fmt.Errorf("%s: %w", errOpParseSpec, err)
	}
	if peek.TypeEUI == "" {
		return nil, ErrSpecMissingTypeEUI
	}
	b, err := hex.DecodeString(peek.TypeEUI)
	if err != nil || len(b) != TypeEUILength {
		return nil, ErrSpecInvalidTypeEUI
	}
	return b, nil
}

// NormalizeVersion prefixes a version label the way the registry expects.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimLeft(v, "vV")
	return RegistryVersionPrefix + v
}

// VersionSegment validates a normalized version as a registry path segment.
func VersionSegment(v string) (string, error) {
	canonical := strings.ToLower(strings.TrimSpace(v))
	cleaned := versionSegmentRegex.ReplaceAllString(canonical, "")
	if cleaned == "" || cleaned == RegistryVersionPrefix || strings.Contains(cleaned, "..") {
		return "", ErrRegistryPathSegment
	}
	if cleaned != canonical {
		return "", ErrRegistryPathSegment
	}
	return cleaned, nil
}

// PathSegment turns a display name into a registry path segment.
func PathSegment(s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", ErrRegistryPathSegment
	}
	slug := Slug(s)
	if slug == fallbackModelSlug {
		cleaned := slugRegex.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "")
		cleaned = strings.Trim(cleaned, "-")
		if cleaned == "" {
			return "", ErrRegistryPathSegment
		}
	}
	if strings.Contains(slug, "..") {
		return "", ErrRegistryPathSegment
	}
	return slug, nil
}

// ModelCodeSegment validates a persisted model code as a registry path segment.
func ModelCodeSegment(code string) (string, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" || modelCodeRegex.MatchString(trimmed) || strings.Contains(trimmed, "..") {
		return "", ErrRegistryPathSegment
	}
	return trimmed, nil
}
