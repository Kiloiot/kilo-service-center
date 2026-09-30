package main

import (
	"crypto/x509"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCAYears     = 1
	testServerDays  = 1
	testPublicName  = "bssci.kiloiot.io"
	testExtraName   = "sc.example.net"
	testExtraIPText = "172.17.0.3"
)

// The server certificate names what the caller asks for, never the host's
// interface addresses: those change, and publishing them leaks the host's
// internal network layout.
func TestGenerateServerCert_NamesOnlyWhatIsAskedFor(t *testing.T) {
	caCert, caKey, err := generateCA(testCAYears)
	require.NoError(t, err)

	cert, _, err := generateServerCert(caCert, caKey, testPublicName, nil, testServerDays)
	require.NoError(t, err)

	assert.Equal(t, testPublicName, cert.Subject.CommonName)
	assert.ElementsMatch(t, []string{testPublicName, certLocalDNSName}, cert.DNSNames)
	assert.True(t, ipsEqual([]net.IP{loopbackIPv4, net.IPv4zero}, cert.IPAddresses), "got %v", cert.IPAddresses)
}

func TestGenerateServerCert_AddsTheRequestedNamesOnce(t *testing.T) {
	caCert, caKey, err := generateCA(testCAYears)
	require.NoError(t, err)

	names := splitAltNames(" " + testExtraName + ",," + testExtraIPText + "," + testPublicName + ",127.0.0.1")
	cert, _, err := generateServerCert(caCert, caKey, testPublicName, names, testServerDays)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{testPublicName, certLocalDNSName, testExtraName}, cert.DNSNames)
	want := []net.IP{loopbackIPv4, net.IPv4zero, net.ParseIP(testExtraIPText)}
	assert.True(t, ipsEqual(want, cert.IPAddresses), "got %v", cert.IPAddresses)
}

func ipsEqual(want, got []net.IP) bool {
	if len(want) != len(got) {
		return false
	}
	for i := range want {
		if !want[i].Equal(got[i]) {
			return false
		}
	}
	return true
}

// A certificate names the product and the holder only; it claims no country,
// province or locality, which the service center cannot know for its operator.
func TestGeneratedCertificates_ClaimNoLocation(t *testing.T) {
	caCert, caKey, err := generateCA(testCAYears)
	require.NoError(t, err)
	serverCert, _, err := generateServerCert(caCert, caKey, testPublicName, nil, testServerDays)
	require.NoError(t, err)
	clientCert, _, err := generateClientCert(caCert, caKey, testPublicName, testServerDays)
	require.NoError(t, err)

	for _, cert := range []*x509.Certificate{caCert, serverCert, clientCert} {
		assert.Empty(t, cert.Subject.Country, cert.Subject.CommonName)
		assert.Empty(t, cert.Subject.Province, cert.Subject.CommonName)
		assert.Empty(t, cert.Subject.Locality, cert.Subject.CommonName)
		assert.Equal(t, []string{certSubjectOrganization}, cert.Subject.Organization)
	}
}
