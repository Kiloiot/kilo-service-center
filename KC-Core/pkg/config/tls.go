package config

import (
	"crypto/tls"
	"fmt"
)

// ParseTLSMinVersion converts config string to tls.VersionTLS* constant.
// Blank string defaults to TLS 1.2 for backward compatibility.
// Per MIOTY Security Guide and BSSCI §1 requirements.
//
// Supported values:
//   - "" (blank) → TLS 1.2 (default for existing installs)
//   - "1.2" → TLS 1.2
//   - "1.3" → TLS 1.3
//
// Returns error for unsupported versions.
func ParseTLSMinVersion(version string) (uint16, error) {
	switch version {
	case "", TLSVersionValue12:
		return tls.VersionTLS12, nil
	case TLSVersionValue13:
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf(errFmtUnsupportedTLSVersion, version)
	}
}
