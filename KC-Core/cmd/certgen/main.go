// Package main provides certificate generation utilities for MIOTY TLS infrastructure.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var (
		certDir      = flag.String("dir", "certs", "Directory to store certificates")
		caOnly       = flag.Bool("ca-only", false, "Generate only CA certificate")
		serverOnly   = flag.Bool("server-only", false, "Generate only server certificate (CA must exist)")
		clientOnly   = flag.Bool("client-only", false, "Generate only client certificate (CA must exist)")
		serverName   = flag.String("server", "localhost", "Server name for certificate")
		clientName   = flag.String("client", "", "Client name for certificate (e.g., base station EUI)")
		validDays    = flag.Int("days", 365, "Certificate validity in days (for server/client certs)")
		caValidYears = flag.Int("ca-years", 20, "CA certificate validity in years")
	)
	flag.Parse()

	// Create certificate directory
	if err := os.MkdirAll(*certDir, 0750); err != nil {
		log.Fatalf("Failed to create certificate directory: %v", err)
	}

	caPath := filepath.Join(*certDir, "ca.crt")
	caKeyPath := filepath.Join(*certDir, "ca.key")

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
		fmt.Printf("CA certificate is valid until: %s\n", caCert.NotAfter.Format("2006-01-02"))

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
		fmt.Printf("Loaded CA certificate (valid until: %s)\n", caCert.NotAfter.Format("2006-01-02"))
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
		if err := saveCertificate(filepath.Join(*certDir, "client.crt"), clientCert); err != nil {
			log.Fatalf("Failed to save client certificate: %v", err)
		}
		if err := savePrivateKey(filepath.Join(*certDir, "client.key"), clientKey); err != nil {
			log.Fatalf("Failed to save client private key: %v", err)
		}
		fmt.Printf("Client certificate saved to %s\n", filepath.Join(*certDir, "client.crt"))
		fmt.Printf("Client certificate is valid until: %s\n", clientCert.NotAfter.Format("2006-01-02"))

		fmt.Println("\nClient certificate generated successfully!")
		fmt.Println("\nDeployment instructions:")
		fmt.Println("1. Deploy these files to the base station:")
		fmt.Printf("   - CA Certificate: %s\n", caPath)
		fmt.Printf("   - Client Certificate: %s\n", filepath.Join(*certDir, "client.crt"))
		fmt.Printf("   - Client Private Key: %s\n", filepath.Join(*certDir, "client.key"))
		fmt.Println("2. Configure the base station to:")
		fmt.Println("   - Trust the CA certificate for server verification")
		fmt.Println("   - Use the client certificate/key for mutual TLS authentication")
		return
	}

	// Generate server certificate
	fmt.Printf("Generating server certificate (valid for %d days)...\n", *validDays)
	serverCert, serverKey, err := generateServerCert(caCert, caKey, *serverName, *validDays)
	if err != nil {
		log.Fatalf("Failed to generate server certificate: %v", err)
	}

	// Save server certificate and key
	if err := saveCertificate(filepath.Join(*certDir, "server.crt"), serverCert); err != nil {
		log.Fatalf("Failed to save server certificate: %v", err)
	}
	if err := savePrivateKey(filepath.Join(*certDir, "server.key"), serverKey); err != nil {
		log.Fatalf("Failed to save server private key: %v", err)
	}
	fmt.Printf("Server certificate saved to %s\n", filepath.Join(*certDir, "server.crt"))
	fmt.Printf("Server certificate is valid until: %s\n", serverCert.NotAfter.Format("2006-01-02"))

	fmt.Println("\nCertificates generated successfully!")
	fmt.Println("\nDeployment instructions:")
	fmt.Println("1. Distribute the CA certificate to all base stations:")
	fmt.Printf("   - CA Certificate: %s\n", caPath)
	fmt.Println("2. Configure the BSSCI server with:")
	fmt.Printf("   - Server Certificate: %s\n", filepath.Join(*certDir, "server.crt"))
	fmt.Printf("   - Server Private Key: %s\n", filepath.Join(*certDir, "server.key"))
	fmt.Println("\nNote: Server certificates can be renewed without updating base stations,")
	fmt.Println("      as long as they are signed by the same CA.")
}

func generateCA(validYears int) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate RSA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate template with long validity
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:       []string{"KiloCenter"},
			OrganizationalUnit: []string{"MIOTY Certificate Authority"},
			Country:            []string{"US"},
			Province:           []string{"California"},
			Locality:           []string{"San Francisco"},
			CommonName:         "KiloCenter Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(validYears, 0, 0), // Long-lived CA
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Create the CA certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return cert, caKey, nil
}

func generateClientCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, clientName string, validDays int) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate RSA private key for client
	clientKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate client private key: %w", err)
	}

	// Use incrementing serial numbers for client certificates
	serialNumber := big.NewInt(time.Now().Unix())

	// Create client certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"KiloCenter"},
			OrganizationalUnit: []string{"MIOTY Base Station"},
			Country:            []string{"US"},
			Province:           []string{"California"},
			Locality:           []string{"San Francisco"},
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
		return nil, nil, fmt.Errorf("failed to create client certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse client certificate: %w", err)
	}

	return cert, clientKey, nil
}

func generateServerCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, serverName string, validDays int) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate RSA private key for server
	serverKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate server private key: %w", err)
	}

	// Create server certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:       []string{"KiloCenter"},
			OrganizationalUnit: []string{"MIOTY Service Center"},
			Country:            []string{"US"},
			Province:           []string{"California"},
			Locality:           []string{"San Francisco"},
			CommonName:         serverName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, validDays),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{serverName, "kilocenter.local"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(0, 0, 0, 0)},
	}

	// Add local network IP if possible
	if ips, err := getLocalIPs(); err == nil {
		template.IPAddresses = append(template.IPAddresses, ips...)
	}

	// Create the server certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse server certificate: %w", err)
	}

	return cert, serverKey, nil
}

func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	return ips, nil
}

func saveCertificate(filename string, cert *x509.Certificate) error {
	// Sanitize and validate path to prevent directory traversal (gosec G304)
	filename = filepath.Clean(filename)
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("failed to resolve filename: %w", err)
	}
	// Ensure path doesn't escape via absolute path or contain traversal
	if strings.Contains(absFilename, "..") {
		return fmt.Errorf("filename contains path traversal")
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer func() { _ = file.Close() }()

	err = pem.Encode(file, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
	if err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	return nil
}

func savePrivateKey(filename string, key *rsa.PrivateKey) error {
	// Sanitize and validate path to prevent directory traversal (gosec G304)
	filename = filepath.Clean(filename)
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("failed to resolve filename: %w", err)
	}
	// Ensure path doesn't escape via absolute path or contain traversal
	if strings.Contains(absFilename, "..") {
		return fmt.Errorf("filename contains path traversal")
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Set restrictive permissions on private key file
	if err := file.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set private key permissions: %w", err)
	}

	err = pem.Encode(file, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

func loadCA(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Sanitize and validate cert path (gosec G304)
	certPath = filepath.Clean(certPath)
	absCertPath, err := filepath.Abs(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve cert path: %w", err)
	}
	if strings.Contains(absCertPath, "..") {
		return nil, nil, fmt.Errorf("cert path contains path traversal")
	}

	// Sanitize and validate key path (gosec G304)
	keyPath = filepath.Clean(keyPath)
	absKeyPath, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve key path: %w", err)
	}
	if strings.Contains(absKeyPath, "..") {
		return nil, nil, fmt.Errorf("key path contains path traversal")
	}

	// Load CA certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Load CA private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CA private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to parse CA private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	return cert, key, nil
}
