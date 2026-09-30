package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// generateServerCert issues the server certificate for serverName, the local
// name and loopback, plus altNames; it never names the host's interfaces.
func generateServerCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, serverName string, altNames []string, validDays int) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate RSA private key for server
	serverKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToGenerateServerPrivateKey, err)
	}

	// Create server certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(serialLeaf),
		Subject: pkix.Name{
			Organization:       []string{certSubjectOrganization},
			OrganizationalUnit: []string{certSubjectOUSC},
			Country:            []string{certSubjectCountry},
			Province:           []string{certSubjectProvince},
			Locality:           []string{certSubjectLocality},
			CommonName:         serverName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, validDays),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	template.DNSNames, template.IPAddresses = subjectAltNames(append([]string{serverName, certLocalDNSName}, altNames...))

	// Create the server certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToCreateServerCertificate, err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToParseServerCertificate, err)
	}

	return cert, serverKey, nil
}

// splitAltNames reads the -san list; blanks are dropped.
func splitAltNames(list string) []string {
	var names []string
	for _, name := range strings.Split(list, altNameSeparator) {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// subjectAltNames sorts names into DNS names and IP addresses, once each,
// after the loopback and unspecified addresses every server certificate carries.
func subjectAltNames(names []string) ([]string, []net.IP) {
	ips := []net.IP{loopbackIPv4, net.IPv4zero}
	var dnsNames []string
	seen := map[string]bool{loopbackIPv4.String(): true, net.IPv4zero.String(): true}
	for _, name := range names {
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		if ip := net.ParseIP(name); ip != nil {
			ips = append(ips, ip)
			continue
		}
		dnsNames = append(dnsNames, name)
	}
	return dnsNames, ips
}
