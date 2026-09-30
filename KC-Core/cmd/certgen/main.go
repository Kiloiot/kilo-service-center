// Package main provides certificate generation utilities for MIOTY TLS infrastructure.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// caMaxPathLen restricts the CA to signing leaf certificates only
// (no intermediate CAs).
const caMaxPathLen = 0

// Error message constants for certificate generation failures.
const (
	errMsgCertPathContainsPathTraversal    = "cert path contains path traversal"
	errMsgFailedToCloseFile                = "failed to close file"
	errMsgFailedToCreateCACertificate      = "failed to create CA certificate"
	errMsgFailedToCreateCertificateFile    = "failed to create certificate file"
	errMsgFailedToCreateClientCertificate  = "failed to create client certificate"
	errMsgFailedToCreatePrivateKeyFile     = "failed to create private key file"
	errMsgFailedToCreateServerCertificate  = "failed to create server certificate"
	errMsgFailedToGenerateCAPrivateKey     = "failed to generate CA private key"
	errMsgFailedToGenerateClientPrivateKey = "failed to generate client private key"
	errMsgFailedToGenerateServerPrivateKey = "failed to generate server private key"
	errMsgFailedToParseCACertificate       = "failed to parse CA certificate"
	errMsgFailedToParseCACertificatePEM    = "failed to parse CA certificate PEM"
	errMsgFailedToParseCAPrivateKey        = "failed to parse CA private key"
	errMsgFailedToParseCAPrivateKeyPEM     = "failed to parse CA private key PEM"
	errMsgFailedToParseClientCertificate   = "failed to parse client certificate"
	errMsgFailedToParseServerCertificate   = "failed to parse server certificate"
	errMsgFailedToReadCACertificate        = "failed to read CA certificate"
	errMsgFailedToReadCAPrivateKey         = "failed to read CA private key"
	errMsgFailedToResolveCertPath          = "failed to resolve cert path"
	errMsgFailedToResolveFilename          = "failed to resolve filename"
	errMsgFailedToResolveKeyPath           = "failed to resolve key path"
	errMsgFailedToSetPrivateKeyPermissions = "failed to set private key permissions"
	errMsgFailedToWriteCertificate         = "failed to write certificate"
	errMsgFailedToWritePrivateKey          = "failed to write private key"
	errMsgFilenameContainsPathTraversal    = "filename contains path traversal"
	errMsgKeyPathContainsPathTraversal     = "key path contains path traversal"
)

// Certificate subject identity shared by the CA and issued certificates.
const (
	certSubjectOrganization = "KiloCenter"
	certSubjectProvince     = "California"
	certSubjectLocality     = "San Francisco"
	certSubjectCountry      = "US"

	certSubjectOUCA  = "MIOTY Certificate Authority"
	certSubjectOUBS  = "MIOTY Base Station"
	certSubjectOUSC  = "MIOTY Service Center"
	certCommonNameCA = "KiloCenter Root CA"
	certLocalDNSName = "kilocenter.local"
)

// Certificate file names inside the output directory.
const (
	fileCACert     = "ca.crt"
	fileCAKey      = "ca.key"
	fileServerCert = "server.crt"
	fileServerKey  = "server.key"
	fileClientCert = "client.crt"
	fileClientKey  = "client.key"
)

// PEM block types.
const (
	pemTypeCertificate   = "CERTIFICATE"
	pemTypeRSAPrivateKey = "RSA PRIVATE KEY"
)

// rsaKeyBits sizes every generated RSA key.
const rsaKeyBits = 4096

// Serial numbers for the self-signed CA and the leaf certificates it issues.
const (
	serialCA   = 1
	serialLeaf = 2
)

// Output directory and private-key file permissions.
const (
	certDirPerm        = 0o750
	privateKeyFilePerm = 0o600
)

// altNameSeparator separates the names of the -san flag.
const altNameSeparator = ","

// loopbackIPv4 is always included in the server certificate SANs alongside
// the unspecified address.
var loopbackIPv4 = net.IPv4(127, 0, 0, 1)

func main() {
	var (
		certDir      = flag.String("dir", "certs", "Directory to store certificates")
		caOnly       = flag.Bool("ca-only", false, "Generate only CA certificate")
		serverOnly   = flag.Bool("server-only", false, "Generate only server certificate (CA must exist)")
		clientOnly   = flag.Bool("client-only", false, "Generate only client certificate (CA must exist)")
		serverName   = flag.String("server", "localhost", "Server name for certificate")
		altNames     = flag.String("san", "", "Further server names and IP addresses, comma-separated")
		clientName   = flag.String("client", "", "Client name for certificate (e.g., base station EUI)")
		validDays    = flag.Int("days", 365, "Certificate validity in days (for server/client certs)")
		caValidYears = flag.Int("ca-years", 20, "CA certificate validity in years")
	)
	flag.Parse()

	// Create certificate directory
	if err := os.MkdirAll(*certDir, certDirPerm); err != nil {
		log.Fatalf("Failed to create certificate directory: %v", err)
	}

	caPath := filepath.Join(*certDir, fileCACert)
	caKeyPath := filepath.Join(*certDir, fileCAKey)

	var caCert *x509.Certificate
	var caKey *rsa.PrivateKey

	// Check if we need to generate or load CA
	if !*serverOnly && !*clientOnly {
		// Generate CA certificate with long validity (10 years by default)
		fmt.Printf("Generating CA certificate (valid for %d years)...\n", *caValidYears)
		var err error
		caCert, caKey, err = generateCA(*caValidYears)
		if err != nil {
			log.Fatalf("Failed to generate CA certificate: %v", err)
		}

		// Save CA certificate and key
		if err := saveCertificate(caPath, caCert); err != nil {
			log.Fatalf("Failed to save CA certificate: %v", err)
		}
		if err := savePrivateKey(caKeyPath, caKey); err != nil {
			log.Fatalf("Failed to save CA private key: %v", err)
		}
		fmt.Printf("CA certificate saved to %s\n", caPath)
		fmt.Printf("CA certificate is valid until: %s\n", caCert.NotAfter.Format(time.DateOnly))

		if *caOnly {
			fmt.Println("\nCA certificate generated successfully!")
			fmt.Println("This CA certificate should be distributed to all base stations.")
			fmt.Println("Base stations will trust any server certificate signed by this CA.")
			return
		}
	} else {
		// Load existing CA for server/client certificate generation
		fmt.Println("Loading existing CA certificate...")
		var err error
		caCert, caKey, err = loadCA(caPath, caKeyPath)
		if err != nil {
			log.Fatalf("Failed to load CA certificate: %v", err)
		}
		fmt.Printf("Loaded CA certificate (valid until: %s)\n", caCert.NotAfter.Format(time.DateOnly))
	}

	// Generate client certificate if requested
	if *clientOnly {
		if *clientName == "" {
			log.Fatal("Client name is required for client certificate generation")
		}
		fmt.Printf("Generating client certificate for %s (valid for %d days)...\n", *clientName, *validDays)
		clientCert, clientKey, err := generateClientCert(caCert, caKey, *clientName, *validDays)
		if err != nil {
			log.Fatalf("Failed to generate client certificate: %v", err)
		}

		// Save client certificate and key
		if err := saveCertificate(filepath.Join(*certDir, fileClientCert), clientCert); err != nil {
			log.Fatalf("Failed to save client certificate: %v", err)
		}
		if err := savePrivateKey(filepath.Join(*certDir, fileClientKey), clientKey); err != nil {
			log.Fatalf("Failed to save client private key: %v", err)
		}
		fmt.Printf("Client certificate saved to %s\n", filepath.Join(*certDir, fileClientCert))
		fmt.Printf("Client certificate is valid until: %s\n", clientCert.NotAfter.Format(time.DateOnly))

		fmt.Println("\nClient certificate generated successfully!")
		fmt.Println("\nDeployment instructions:")
		fmt.Println("1. Deploy these files to the base station:")
		fmt.Printf("   - CA Certificate: %s\n", caPath)
		fmt.Printf("   - Client Certificate: %s\n", filepath.Join(*certDir, fileClientCert))
		fmt.Printf("   - Client Private Key: %s\n", filepath.Join(*certDir, fileClientKey))
		fmt.Println("2. Configure the base station to:")
		fmt.Println("   - Trust the CA certificate for server verification")
		fmt.Println("   - Use the client certificate/key for mutual TLS authentication")
		return
	}

	// Generate server certificate
	fmt.Printf("Generating server certificate (valid for %d days)...\n", *validDays)
	serverCert, serverKey, err := generateServerCert(caCert, caKey, *serverName, splitAltNames(*altNames), *validDays)
	if err != nil {
		log.Fatalf("Failed to generate server certificate: %v", err)
	}

	// Save server certificate and key
	if err := saveCertificate(filepath.Join(*certDir, fileServerCert), serverCert); err != nil {
		log.Fatalf("Failed to save server certificate: %v", err)
	}
	if err := savePrivateKey(filepath.Join(*certDir, fileServerKey), serverKey); err != nil {
		log.Fatalf("Failed to save server private key: %v", err)
	}
	fmt.Printf("Server certificate saved to %s\n", filepath.Join(*certDir, fileServerCert))
	fmt.Printf("Server certificate is valid until: %s\n", serverCert.NotAfter.Format(time.DateOnly))

	fmt.Println("\nCertificates generated successfully!")
	fmt.Println("\nDeployment instructions:")
	fmt.Println("1. Distribute the CA certificate to all base stations:")
	fmt.Printf("   - CA Certificate: %s\n", caPath)
	fmt.Println("2. Configure the BSSCI server with:")
	fmt.Printf("   - Server Certificate: %s\n", filepath.Join(*certDir, fileServerCert))
	fmt.Printf("   - Server Private Key: %s\n", filepath.Join(*certDir, fileServerKey))
	fmt.Println("\nNote: Server certificates can be renewed without updating base stations,")
	fmt.Println("      as long as they are signed by the same CA.")
}

func generateCA(validYears int) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate RSA private key
	caKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToGenerateCAPrivateKey, err)
	}

	// Create CA certificate template with long validity
	template := x509.Certificate{
		SerialNumber: big.NewInt(serialCA),
		Subject: pkix.Name{
			Organization:       []string{certSubjectOrganization},
			OrganizationalUnit: []string{certSubjectOUCA},
			Country:            []string{certSubjectCountry},
			Province:           []string{certSubjectProvince},
			Locality:           []string{certSubjectLocality},
			CommonName:         certCommonNameCA,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(validYears, 0, 0), // Long-lived CA
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            caMaxPathLen,
		MaxPathLenZero:        true,
	}

	// Create the CA certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToCreateCACertificate, err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToParseCACertificate, err)
	}

	return cert, caKey, nil
}
