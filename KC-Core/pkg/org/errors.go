package org

import "errors"

// errCertificateNil reports a nil certificate passed to certificate-based resolution.
var errCertificateNil = errors.New("certificate is nil")

// Error format strings for wrapped resolution failures.
const (
	errFmtOrgNotFound           = "org %s not found: %w"
	errFmtNoDefaultOrgForTenant = "no default org for tenant %d: %w"
)
