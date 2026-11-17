package nettransport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
)

// TLSFiles names the PEM files a mutual-TLS listener needs.
type TLSFiles struct {
	Cert string
	Key  string
	CA   string
}

// Exist reports whether every file is present on disk.
func (f TLSFiles) Exist() bool {
	for _, path := range []string{f.Cert, f.Key, f.CA} {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

// TLSOptions tunes the server-side TLS policy. CipherSuites names Go cipher
// suite constants; empty selects the default modern set.
type TLSOptions struct {
	MinVersion   string
	CipherSuites []string
	// DeniedCertFingerprints are SHA-256 fingerprints of client certificates refused at the handshake even though the CA signed them.
	DeniedCertFingerprints []string
}

// defaultCipherSuites is the modern AEAD set both protocol servers offer when
// the operator names none.
var defaultCipherSuites = []uint16{
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_AES_128_GCM_SHA256,
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
}

// cipherSuiteByName resolves the operator-facing suite names to their constants.
var cipherSuiteByName = map[string]uint16{
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_AES_128_GCM_SHA256":                  tls.TLS_AES_128_GCM_SHA256,
	"TLS_AES_256_GCM_SHA384":                  tls.TLS_AES_256_GCM_SHA384,
	"TLS_CHACHA20_POLY1305_SHA256":            tls.TLS_CHACHA20_POLY1305_SHA256,
}

// BuildServerTLSConfig loads the key pair and CA pool and returns a config
// that requires and verifies a client certificate, as both MIOTY interfaces
// mandate. The key pair is read per handshake, so a renewed server
// certificate needs no restart.
func BuildServerTLSConfig(files TLSFiles, opts TLSOptions) (*tls.Config, error) {
	keyPair, err := newKeyPairReloader(files.Cert, files.Key)
	if err != nil {
		return nil, err
	}
	caPEM, err := os.ReadFile(files.CA)
	if err != nil {
		return nil, fmt.Errorf(errFmtReadCACertificate, err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf(errFmtNoValidCACertificates, files.CA)
	}
	minVersion, err := config.ParseTLSMinVersion(opts.MinVersion)
	if err != nil {
		return nil, fmt.Errorf(errFmtTLSMinVersion, err)
	}
	suites, err := resolveCipherSuites(opts.CipherSuites)
	if err != nil {
		return nil, err
	}
	denied, err := deniedFingerprints(opts.DeniedCertFingerprints)
	if err != nil {
		return nil, err
	}
	//nolint:gosec // G402: MinVersion is validated by config.ParseTLSMinVersion
	cfg := &tls.Config{
		GetCertificate: keyPair.GetCertificate,
		ClientAuth:     tls.RequireAndVerifyClientCert,
		ClientCAs:      caPool,
		MinVersion:     minVersion,
		CipherSuites:   suites,
	}
	if len(denied) > 0 {
		cfg.VerifyConnection = refuseDenied(denied)
	}
	return cfg, nil
}

var fingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// deniedFingerprints accepts the forms operators copy from openssl (colons, upper case) and refuses anything else, so a typo cannot silently deny nothing.
func deniedFingerprints(values []string) (map[string]struct{}, error) {
	denied := make(map[string]struct{}, len(values))
	for _, value := range values {
		fingerprint := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), ":", ""))
		if !fingerprintPattern.MatchString(fingerprint) {
			return nil, fmt.Errorf(errFmtInvalidCertFingerprint, ErrInvalidCertFingerprint, value)
		}
		denied[fingerprint] = struct{}{}
	}
	return denied, nil
}

// refuseDenied runs on every handshake, resumed ones included, after the chain is verified.
func refuseDenied(denied map[string]struct{}) func(tls.ConnectionState) error {
	return func(state tls.ConnectionState) error {
		if len(state.PeerCertificates) == 0 {
			return nil
		}
		if _, ok := denied[crypto.CertFingerprintSHA256(state.PeerCertificates[0].Raw)]; ok {
			return ErrDeniedClientCertificate
		}
		return nil
	}
}

func resolveCipherSuites(names []string) ([]uint16, error) {
	if len(names) == 0 {
		return defaultCipherSuites, nil
	}
	suites := make([]uint16, 0, len(names))
	for _, name := range names {
		suite, ok := cipherSuiteByName[name]
		if !ok {
			return nil, fmt.Errorf(errFmtUnsupportedCipher, ErrUnsupportedCipherSuite, name)
		}
		suites = append(suites, suite)
	}
	return suites, nil
}
