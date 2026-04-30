package version

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Embed manifest (written to this directory by tools/release/generate_manifest.sh)
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
			manifest.Version = "dev-local"
		}
		if manifest.SchemaVersion <= 0 {
			loadErr = fmt.Errorf("invalid manifest: schemaVersion must be > 0")
			manifest = nil
			return
		}

		// Apply dev fallbacks for disclosure fields absent in older manifests
		if manifest.Edition == "" {
			manifest.Edition = "Community Edition"
		}
		if manifest.LicenseID == "" {
			manifest.LicenseID = "AGPL-3.0-or-later"
		}
		if manifest.LicenseURL == "" {
			manifest.LicenseURL = "https://www.gnu.org/licenses/agpl-3.0.html"
		}
		if manifest.EditionCode == "" {
			manifest.EditionCode = "ce"
		}
		if manifest.SourceURL == "" {
			manifest.SourceURL = "https://github.com/Kiloiot/KiloServiceCenter"
		}
		if manifest.DocsURL == "" {
			manifest.DocsURL = "https://docs.kiloiot.io/"
		}
		if manifest.HomepageURL == "" {
			manifest.HomepageURL = "https://kiloiot.io/mioty-service-center/"
		}
		if manifest.TrademarkNotice == "" {
			manifest.TrademarkNotice = "KiloCenter is a trademark of Tim Kravchunovsky."
		}
	})
	return manifest, loadErr
}

// String formats version info for logging
func (i *Info) String() string {
	if len(i.GitCommit) >= 7 {
		return fmt.Sprintf("%s (built %s, commit %s)", i.Version, i.BuildTime, i.GitCommit[:7])
	}
	return fmt.Sprintf("%s (built %s)", i.Version, i.BuildTime)
}

// IsProduction returns true if version is a semantic version tag (not dev/local)
func (i *Info) IsProduction() bool {
	return i.Version != "dev" && i.Version != "dev-local" && !strings.Contains(i.Version, "dirty")
}
