package bssciservices

import (
	"context"
	"fmt"
	"sort"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// supportedVersion is one parsed member of the service center's supported
// protocol version set.
type supportedVersion struct {
	major, minor, patch int
	canonical           string
}

// versionNegotiator selects the BSSCI protocol version for a connection from
// a configured supported-version set per rev1 §4.1-§4.3 and the §5.3.2 conRsp
// arbitration rule: the base station requests its newest supported version and
// the service center answers with the version it will speak.
type versionNegotiator struct {
	logger    logger.Logger
	supported []supportedVersion // sorted descending by (major, minor, patch)
}

// NewVersionNegotiator builds a negotiator from the supported protocol version
// set. The set must be non-empty, well-formed ("major.minor.patch"), and free
// of duplicates; it is parsed, normalized, and sorted once at construction.
func NewVersionNegotiator(supported []string, log logger.Logger) (bssci.VersionNegotiator, error) {
	if log == nil {
		return nil, fmt.Errorf("version negotiator requires a logger")
	}
	if len(supported) == 0 {
		return nil, fmt.Errorf("version negotiator requires a non-empty supported version set")
	}

	parsed := make([]supportedVersion, 0, len(supported))
	seen := make(map[string]bool, len(supported))
	for _, raw := range supported {
		major, minor, patch, cerr := bssci.ParseVersion(raw)
		if cerr != nil {
			return nil, fmt.Errorf("malformed supported version %q: %s", raw, cerr.Token)
		}
		canonical := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		if seen[canonical] {
			return nil, fmt.Errorf("duplicate supported version %q", canonical)
		}
		seen[canonical] = true
		parsed = append(parsed, supportedVersion{major: major, minor: minor, patch: patch, canonical: canonical})
	}

	sort.Slice(parsed, func(i, j int) bool {
		a, b := parsed[i], parsed[j]
		if a.major != b.major {
			return a.major > b.major
		}
		if a.minor != b.minor {
			return a.minor > b.minor
		}
		return a.patch > b.patch
	})

	return &versionNegotiator{logger: log, supported: parsed}, nil
}

// Negotiate selects the highest supported version with the requested major and
// a supported minor not above the requested minor. The result is always an
// exact member of the supported set - the requested patch is ignored for
// compatibility (rev1 §4.3) and never echoed unless the service center
// implements it. A base station requesting a newer minor is negotiated down to
// the selected version, which the conRsp carries per §5.3.2; the base station
// agrees by completing the operation or rejects it with an error.
func (n *versionNegotiator) Negotiate(ctx context.Context, requested string) (string, error) {
	reqMajor, reqMinor, _, cerr := bssci.ParseVersion(requested)
	if cerr != nil {
		return "", cerr
	}

	majorMatched := false
	for _, v := range n.supported {
		if v.major != reqMajor {
			continue
		}
		majorMatched = true
		if v.minor > reqMinor {
			continue
		}
		if v.minor < reqMinor {
			n.logger.InfoContext(ctx, bssci.LogBSSCIMinorVersionNegotiatedDown,
				"requestedVersion", requested,
				"selectedVersion", v.canonical)
		}
		return v.canonical, nil
	}

	if !majorMatched {
		n.logger.WarnContext(ctx, bssci.LogBSSCIVersionIncompatible,
			"requestedVersion", requested)
		return "", bssci.NewCatalogError(bssci.ErrUnsupportedMajorVersion, bssci.POSIX_EPROTO)
	}

	// Same major, but every supported minor is above the requested minor: the
	// service center cannot offer a version the base station did not request
	// (BSSCI rev1 §4.2)
	n.logger.WarnContext(ctx, bssci.LogBSSCIMinorVersionMismatch,
		"requestedVersion", requested)
	return "", bssci.NewCatalogError(bssci.ErrUnsupportedMinorVersion, bssci.POSIX_EPROTO)
}
