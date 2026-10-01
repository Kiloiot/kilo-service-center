package certificates

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	namesTestPublicHost = "bssci.kiloiot.io"
	namesTestLocalName  = "kilocenter.local"
	namesTestBridgeIP   = "172.17.0.3"
	namesTestExternal   = "tls://sc.example.net:5000"
	namesTestExtHost    = "sc.example.net"
	namesTestConfigured = "sc-backup.example.net"
)

// writeServerCert stages the service center's current server certificate.
func writeServerCert(t *testing.T, dir, commonName string, dnsNames []string, ips []net.IP) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(testCertValidity),
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	require.NoError(t, os.WriteFile(filepath.Join(dir, serverCertFileName), pemBytes, testFilePerm))
	for _, name := range []string{caCertFileName, caKeyFileName} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("staged"), testFilePerm))
	}
}

// renewalArgs renews the server certificate and returns the certgen arguments.
func renewalArgs(t *testing.T, protocol config.ProtocolConfig, configured []string) []string {
	t.Helper()
	dir := t.TempDir()
	writeServerCert(t, dir, namesTestPublicHost, []string{namesTestPublicHost, namesTestLocalName}, []net.IP{net.ParseIP(namesTestBridgeIP)})
	var args []string
	svc := &Service{
		clock:  clock.SystemClock{},
		logger: &mockLogger{},
		settings: settings{
			certsDir:           dir,
			serverValidityDays: 1,
			serverNames:        configured,
			protocol:           &protocol,
		},
		certGen: func(_ context.Context, _ string, genArgs ...string) (string, string, error) {
			args = genArgs
			return "", "", nil
		},
	}
	require.NoError(t, svc.RenewServerCertificates(testutil.TestContext()))
	return args
}

func flagValue(args []string, flag string) string {
	i := slices.Index(args, flag)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

// Renewing without a configured external URL keeps the names stations
// connect by: the current certificate's subject and alternative names.
func TestRenewServerCertificates_KeepsTheCurrentNames(t *testing.T) {
	args := renewalArgs(t, config.ProtocolConfig{}, nil)

	assert.Equal(t, namesTestPublicHost, flagValue(args, certGenFlagServer))
	altNames := strings.Split(flagValue(args, certGenFlagAltNames), altNameSeparator)
	assert.Contains(t, altNames, namesTestLocalName)
	assert.Contains(t, altNames, namesTestBridgeIP)
}

// A configured external URL names the certificate; the current and the
// configured names stay on it.
func TestRenewServerCertificates_ExternalURLNamesTheCertificate(t *testing.T) {
	args := renewalArgs(t, config.ProtocolConfig{BSCIExternalURL: namesTestExternal}, []string{namesTestConfigured})

	assert.Equal(t, namesTestExtHost, flagValue(args, certGenFlagServer))
	altNames := strings.Split(flagValue(args, certGenFlagAltNames), altNameSeparator)
	assert.ElementsMatch(t, []string{namesTestConfigured, namesTestPublicHost, namesTestLocalName, namesTestBridgeIP}, altNames)
}
