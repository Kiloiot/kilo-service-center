package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

func generateClientCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, clientName string, validDays int) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate RSA private key for client
	clientKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToGenerateClientPrivateKey, err)
	}

	// Use incrementing serial numbers for client certificates
	serialNumber := big.NewInt(time.Now().Unix())

	// Create client certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{certSubjectOrganization},
			OrganizationalUnit: []string{certSubjectOUBS},
			CommonName:         clientName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, validDays),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create the client certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToCreateClientCertificate, err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToParseClientCertificate, err)
	}

	return cert, clientKey, nil
}
