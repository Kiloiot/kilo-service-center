package certificates

// Log messages for certificate lifecycle operations. These belong to the
// domain service; the gRPC delivery layer keeps its own request-level
// messages in the transport catalog.
const (
	LogCertBSNotFound             = "Base station not found for certificate retrieval"
	LogCertCACertCopied           = "Copied existing CA certificate"
	LogCertCACertCopyFailed       = "Failed to copy CA certificate"
	LogCertCACertReadFailed       = "Failed to read existing CA certificate"
	LogCertCAKeyCopyFailed        = "Failed to copy CA key"
	LogCertCAKeyRemoveFailed      = "Failed to remove the CA key copy from the certificate bundle"
	LogCertCAKeyReadFailed        = "Failed to read existing CA key"
	LogCertCertsPathInfo          = "KC-Core certificates path"
	LogCertConfigMissingPath      = "Certificate generator path not configured and default not found"
	LogCertConfigMissingPathHint  = "ensure service is started from kilocenter-modules/ directory or set certificates.certgen_path in config"
	LogCertDirCreateFailed        = "Failed to create certificate directory"
	LogCertDirRemoveFailed        = "Failed to remove certificate directory"
	LogCertDirectoryInfo          = "Certificate directory"
	LogCertDirectoryNotFound      = "Certificate directory not found"
	LogCertDirectoryNotFoundHint  = "Run certgen to generate certificates before starting"
	LogCertExpiredDirRemoveFailed = "Failed to remove expired certificate directory"
	LogCertFileReadFailed         = "Failed to read certificate file"
	LogCertGenerationExecuting    = "Executing certificate generation"
	LogCertGenerationFailed       = "Certificate generation failed"
	LogCertGenerationRequested    = "certificate generation requested"
	LogCertGenerationStderr       = "Certificate generation stderr"
	LogCertGenerationStdout       = "Certificate generation stdout"
	LogCertGenerationSuccess      = "Certificate generation successful"
	LogCertGeneratorPathInfo      = "Certificate generator path"
	LogCertInfoWriteFailed        = "Failed to write certificate info"
	LogCertInvalidEUI             = "Invalid EUI format"
	LogCertInvalidValidityDays    = "Invalid validity days"
	LogCertPEMBlockParseFailed    = "Failed to parse PEM block"
	LogCertParseFailed            = "Failed to parse certificate"
	LogCertPersistenceSkipped     = "Certificate persistence skipped"
	LogCertTempDirCreateFailed    = "Failed to create temp directory"
	LogCertBundleKeyRemoveFailed  = "Failed to remove the private key file of a certificate bundle"
	LogCertUnmarshalFailed        = "Failed to unmarshal certificate info"
	LogCertsDirCreateFailed       = "Failed to create certificates directory"
	LogDownloadCertFailed         = "download certificate failed"
	LogServerCertGenExecuting     = "Executing server certificate generation"
	LogServerCertGenFailed        = "Server certificate generation failed"
	LogServerCertGenRequested     = "server certificate generation requested"
	LogServerCertGenSuccess       = "Server certificate generation successful"
	LogServerCertRenewalRequested = "server certificate renewal requested"
)

// Certificate type identifiers shared with the delivery layer.
const (
	// CertTypeCA is the CA certificate type.
	CertTypeCA = "ca"
	// CertTypeClient is the client certificate type.
	CertTypeClient = "client"
	// CertTypeKey is the private key type.
	CertTypeKey = "key"
	// CertTypeServer is the server certificate type.
	CertTypeServer = "server"
)
