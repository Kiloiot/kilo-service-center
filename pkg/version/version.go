// Package version reports the build identity of a running binary: the release
// manifest embedded at build time, or a development placeholder when the binary
// was built outside the release pipeline.
package version

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// DevLocalVersion marks a binary built outside the release pipeline. Builds
// carrying it are deliberately not treated as releases.
const DevLocalVersion = "dev-local"

// DevVersion marks a binary built with the bare "dev" version stamp.
const DevVersion = "dev"

// Community Edition manifest fallbacks for manifests older than the
// disclosure fields.
const (
	fallbackEdition         = "Community Edition"
	fallbackLicenseID       = "AGPL-3.0-or-later"
	fallbackLicenseURL      = "https://www.gnu.org/licenses/agpl-3.0.html"
	fallbackEditionCode     = "ce"
	fallbackSourceURL       = "https://github.com/Kiloiot/kilo-service-center"
	fallbackDocsURL         = "https://docs.kiloiot.io/"
	fallbackHomepageURL     = "https://kiloiot.io/mioty-service-center/"
	fallbackTrademarkNotice = "KiloCenter is a trademark of Tim Kravchunovsky."
)

// versionStringFmt formats version info for logging.
const (
	versionWithCommitFmt = "%s (built %s, commit %s)"
	versionFmt           = "%s (built %s)"
)

// dirtyVersionMarker appears in version stamps built from a modified tree.
const dirtyVersionMarker = "dirty"

// errInvalidManifestSchema reports an embedded manifest whose schemaVersion
// is missing or not positive.
var errInvalidManifestSchema = errors.New("invalid manifest: schemaVersion must be > 0")

// The committed file is a development stub (version "dev", no build identity);
// the release workflow stamps the real manifest here at build time.
//
//go:embed release-manifest.json
var manifestBytes []byte

// Info contains release metadata from manifest.json
type Info struct {
	Version         string            `json:"version"`
	BuildTime       string            `json:"buildTime"`
	GitCommit       string            `json:"gitCommit"`
	GitBranch       string            `json:"gitBranch"`
	BuildUser       string            `json:"buildUser"`
	GoVersion       string            `json:"goVersion"`
	Artifacts       map[string]string `json:"artifacts"`
	SchemaVersion   int               `json:"schemaVersion"`
	Edition         string            `json:"edition"`
	EditionCode     string            `json:"editionCode"`
	LicenseID       string            `json:"licenseId"`
	LicenseURL      string            `json:"licenseUrl"`
	SourceURL       string            `json:"sourceUrl"`
	DocsURL         string            `json:"docsUrl"`
	HomepageURL     string            `json:"homepageUrl"`
	TrademarkNotice string            `json:"trademarkNotice"`
}

var (
	once     sync.Once
	manifest *Info
	loadErr  error
)

// Get returns the embedded release manifest (singleton, thread-safe)
func Get() (*Info, error) {
	once.Do(func() {
		manifest = &Info{}
		loadErr = json.Unmarshal(manifestBytes, manifest)
		if loadErr != nil {
			manifest = nil
			return
		}

		// Validate required fields
		if manifest.Version == "" {
			manifest.Version = DevLocalVersion
		}
		if manifest.SchemaVersion <= 0 {
			loadErr = errInvalidManifestSchema
			manifest = nil
			return
		}

		// The development stub carries no build identity; the binary still
		// knows the toolchain that built it.
		if manifest.GoVersion == "" {
			manifest.GoVersion = runtime.Version()
		}

		// Apply dev fallbacks for disclosure fields absent in older manifests
		if manifest.Edition == "" {
			manifest.Edition = fallbackEdition
		}
		if manifest.LicenseID == "" {
			manifest.LicenseID = fallbackLicenseID
		}
		if manifest.LicenseURL == "" {
			manifest.LicenseURL = fallbackLicenseURL
		}
		if manifest.EditionCode == "" {
			manifest.EditionCode = fallbackEditionCode
		}
		if manifest.SourceURL == "" {
			manifest.SourceURL = fallbackSourceURL
		}
		if manifest.DocsURL == "" {
			manifest.DocsURL = fallbackDocsURL
		}
		if manifest.HomepageURL == "" {
			manifest.HomepageURL = fallbackHomepageURL
		}
		if manifest.TrademarkNotice == "" {
			manifest.TrademarkNotice = fallbackTrademarkNotice
		}
	})
	return manifest, loadErr
}

// String formats version info for logging
func (i *Info) String() string {
	if len(i.GitCommit) >= 7 {
		return fmt.Sprintf(versionWithCommitFmt, i.Version, i.BuildTime, i.GitCommit[:7])
	}
	return fmt.Sprintf(versionFmt, i.Version, i.BuildTime)
}

// IsProduction returns true if version is a semantic version tag (not dev/local)
func (i *Info) IsProduction() bool {
	return i.Version != DevVersion && i.Version != DevLocalVersion && !strings.Contains(i.Version, dirtyVersionMarker)
}
