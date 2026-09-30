package certificates

import "errors"

// Domain sentinels for certificate lifecycle failures. The gRPC delivery
// layer maps each sentinel onto its catalog token; the service itself carries
// no transport vocabulary.
var (
	// ErrGeneratorNotFound reports a missing certificate generator script.
	ErrGeneratorNotFound = errors.New("certificate generator not found")
	// ErrServiceNotConfigured reports a certificate operation on an unconfigured service.
	ErrServiceNotConfigured = errors.New("certificate service not configured")
	// ErrInvalidBaseStationEUI reports a base station EUI that failed parsing.
	ErrInvalidBaseStationEUI = errors.New("invalid base station EUI format")
	// ErrInvalidValidityPeriod reports a validity period outside the accepted range.
	ErrInvalidValidityPeriod = errors.New("invalid certificate validity period")
	// ErrTenantRequired reports certificate issuance without a tenant context.
	ErrTenantRequired = errors.New("tenant context required for certificate issuance")
	// ErrBaseStationNotFound reports an EUI with no base station under the tenant.
	ErrBaseStationNotFound = errors.New("base station not found")
	// ErrDirectoryCreate reports a failure preparing the certificate directory.
	ErrDirectoryCreate = errors.New("certificate directory creation failed")
	// ErrCACertRead reports an unreadable CA certificate.
	ErrCACertRead = errors.New("CA certificate read failed")
	// ErrCACertCopy reports a CA certificate that could not be copied.
	ErrCACertCopy = errors.New("CA certificate copy failed")
	// ErrCAKeyRead reports an unreadable CA key.
	ErrCAKeyRead = errors.New("CA key read failed")
	// ErrCAKeyCopy reports a CA key that could not be copied.
	ErrCAKeyCopy = errors.New("CA key copy failed")
	// ErrGenerationFailed reports a failed certificate generation run.
	ErrGenerationFailed = errors.New("certificate generation failed")
	// ErrPersistenceFailed reports certificate material that could not be stored.
	ErrPersistenceFailed = errors.New("certificate persistence failed")
	// ErrTypeRequired reports a download without a certificate type.
	ErrTypeRequired = errors.New("certificate type required")
	// ErrNotFound reports a certificate that does not exist.
	ErrNotFound = errors.New("certificate not found")
	// ErrCertificateNotStored reports a base station whose certificate this service center holds no copy of.
	ErrCertificateNotStored = errors.New("certificate not stored")
	// ErrKeySuperseded reports a bundle whose private key is no longer the station's stored key.
	ErrKeySuperseded = errors.New("bundle private key superseded")
	// ErrKeyUnreadable reports a stored private key kept because it could not be decrypted.
	ErrKeyUnreadable = errors.New("stored private key unreadable")
	// ErrKeyDownloadNotRecorded reports a private key kept because its download could not be recorded.
	ErrKeyDownloadNotRecorded = errors.New("private key download not recorded")
	// ErrServerGenerationFailed reports a failed server certificate run.
	ErrServerGenerationFailed = errors.New("server certificate generation failed")
	// ErrNoCertificatesToRenew reports a renewal with no existing material.
	ErrNoCertificatesToRenew = errors.New("no certificates to renew")
)
