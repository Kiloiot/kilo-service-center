// Package crypto provides certificate fingerprinting helpers.
package crypto

import "errors"

// Sentinel errors for certificate parsing failures.
var (
	errNoPEMBlock = errors.New("no PEM block in certificate data")
)

// Error format strings for wrapped failures; verbs are filled at the point of failure.
const (
	errFmtParseCertificate = "parse certificate: %w"
)
