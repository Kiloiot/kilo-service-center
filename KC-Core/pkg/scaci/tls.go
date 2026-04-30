// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/kilocenter/KC-Core/pkg/config"
)

// LoadTLSConfig loads TLS configuration for SCACI server per MIOTY SCACI v1.0.0 §1
//
// SCACI requires mutual TLS:
//   - Server presents certificate to authenticate to AC
//   - AC presents client certificate to authenticate to SC
//   - Both certificates validated against CA
//
// Per spec: "Digital certificates are required for the Service Center and the Application Center
// to establish the TLS connection."
//
// Parameters:
//   - tlsCfg: TLS configuration from config file
//
// Returns:
//   - *tls.Config: Configured TLS settings ready for net.Listen
//   - error: Configuration error if certificates invalid or missing
func LoadTLSConfig(tlsCfg config.TLSConfig) (*tls.Config, error) {
	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Load CA certificate for client validation
	caCert, err := os.ReadFile(tlsCfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	// SCACI §1: Validate CA pool is non-empty (defense-in-depth for mutual TLS)
	// AppendCertsFromPEM returns false if no valid certs were found in the PEM data
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("no valid CA certificates found in %s", tlsCfg.CAFile)
	}

	// Parse minimum TLS version using shared parser
	minVersion, err := config.ParseTLSMinVersion(tlsCfg.MinVersion)
	if err != nil {
		return nil, fmt.Errorf("SCACI TLS configuration: %w", err)
	}

	// Parse cipher suites if specified
	var cipherSuites []uint16
	if len(tlsCfg.AllowedCiphers) > 0 {
		cipherSuites, err = parseCipherSuites(tlsCfg.AllowedCiphers)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cipher suites: %w", err)
		}
	}

	//nolint:gosec // G402: MinVersion validated via config.ParseTLSMinVersion to ensure TLS 1.2+
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // Mutual TLS required per SCACI spec
		ClientCAs:    caCertPool,
		MinVersion:   minVersion,
		CipherSuites: cipherSuites,
		// Prefer server cipher suites for better security
		PreferServerCipherSuites: true,
	}, nil
}

// parseCipherSuites converts cipher suite names to TLS constants
//
// Parameters:
//   - names: Array of cipher suite names
//
// Returns:
//   - []uint16: TLS cipher suite constants
//   - error: Parse error if cipher name unrecognized
func parseCipherSuites(names []string) ([]uint16, error) {
	// Map of supported cipher suite names to TLS constants
	cipherMap := map[string]uint16{
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		// TLS 1.3 cipher suites
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	var result []uint16
	for _, name := range names {
		if cipher, ok := cipherMap[name]; ok {
			result = append(result, cipher)
		} else {
			return nil, fmt.Errorf("unsupported cipher suite: %s", name)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid cipher suites specified")
	}

	return result, nil
}

// TLSVersionName returns a human-readable TLS version string from the protocol version number.
// Per SCACI §1 TLS evidence persistence requirements.
//
// Parameters:
//   - version: TLS protocol version number from tls.ConnectionState().Version
//
// Returns:
//   - string: Human-readable version name (e.g., "TLS 1.2", "TLS 1.3")
func TLSVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}
