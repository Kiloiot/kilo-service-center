package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func saveCertificate(filename string, cert *x509.Certificate) (err error) {
	// Sanitize and validate path to prevent directory traversal (gosec G304)
	filename = filepath.Clean(filename)
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToResolveFilename, err)
	}
	// Ensure path doesn't escape via absolute path or contain traversal
	if strings.Contains(absFilename, "..") {
		return errors.New(errMsgFilenameContainsPathTraversal)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToCreateCertificateFile, err)
	}
	defer closeWrittenFile(file, &err)

	err = pem.Encode(file, &pem.Block{
		Type:  pemTypeCertificate,
		Bytes: cert.Raw,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToWriteCertificate, err)
	}

	return nil
}

func savePrivateKey(filename string, key *rsa.PrivateKey) (err error) {
	// Sanitize and validate path to prevent directory traversal (gosec G304)
	filename = filepath.Clean(filename)
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToResolveFilename, err)
	}
	// Ensure path doesn't escape via absolute path or contain traversal
	if strings.Contains(absFilename, "..") {
		return errors.New(errMsgFilenameContainsPathTraversal)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToCreatePrivateKeyFile, err)
	}
	defer closeWrittenFile(file, &err)

	// Set restrictive permissions on private key file
	if err := file.Chmod(privateKeyFilePerm); err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToSetPrivateKeyPermissions, err)
	}

	err = pem.Encode(file, &pem.Block{
		Type:  pemTypeRSAPrivateKey,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToWritePrivateKey, err)
	}

	return nil
}

// closeWrittenFile closes a file from a deferred call; a written file's close
// can report a lost write, so its failure becomes the caller's error.
func closeWrittenFile(file *os.File, err *error) {
	if closeErr := file.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("%s: %w", errMsgFailedToCloseFile, closeErr)
	}
}

func loadCA(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Sanitize and validate cert path (gosec G304)
	certPath = filepath.Clean(certPath)
	absCertPath, err := filepath.Abs(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToResolveCertPath, err)
	}
	if strings.Contains(absCertPath, "..") {
		return nil, nil, errors.New(errMsgCertPathContainsPathTraversal)
	}

	// Sanitize and validate key path (gosec G304)
	keyPath = filepath.Clean(keyPath)
	absKeyPath, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToResolveKeyPath, err)
	}
	if strings.Contains(absKeyPath, "..") {
		return nil, nil, errors.New(errMsgKeyPathContainsPathTraversal)
	}

	// Load CA certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToReadCACertificate, err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, errors.New(errMsgFailedToParseCACertificatePEM)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToParseCACertificate, err)
	}

	// Load CA private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToReadCAPrivateKey, err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New(errMsgFailedToParseCAPrivateKeyPEM)
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToParseCAPrivateKey, err)
	}

	return cert, key, nil
}
