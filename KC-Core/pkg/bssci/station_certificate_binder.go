package bssci

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// stationCertificates binds a connecting base station to the client
// certificate it presented: the certificate must not name another station,
// and it must be the certificate registered for the station.
type stationCertificates struct {
	directory RegisteredBaseStationDirectory
	logger    logger.Logger
}

// NewStationCertificateBinder builds the binding over the registered base
// stations.
func NewStationCertificateBinder(directory RegisteredBaseStationDirectory, log logger.Logger) (StationCertificateBinder, error) {
	switch {
	case directory == nil:
		return nil, errNilRegisteredStationDirectory
	case log == nil:
		return nil, errNilStationCertificateLogger
	}
	return &stationCertificates{directory: directory, logger: log}, nil
}

// BindStationCertificate refuses the claim unless the certificate belongs to
// the claimed station (BSSCI §1 mutual TLS). A certificate that names a
// station (a dashed-EUI common name) must name the claimed one; a
// certificate naming no station, such as an organization certificate, is
// bound by its fingerprint alone. The presented certificate's SHA-256
// fingerprint must equal the one registered for the station. A station
// registered before fingerprints were stored is pinned to its stored
// certificate after it matched, and a station registered with no
// certificate at all to the first certificate that names it. Once bound, the
// certificate's expiry is recorded while the station stores none.
func (b *stationCertificates) BindStationCertificate(ctx context.Context, claim StationCertificateClaim) error {
	station := mioty.FormatEUI64(claim.BaseStationEUI)
	if claim.SubjectEUI != nil && *claim.SubjectEUI != claim.BaseStationEUI {
		b.logger.WarnContext(ctx, LogBSSCICertSubjectEUIMismatch,
			logger.FieldCertEui, *claim.SubjectEUI,
			logger.FieldBsEui, claim.BaseStationEUI)
		return fmt.Errorf(errFmtStationCertNamesAnotherStation, station)
	}
	if claim.Certificate == nil {
		return fmt.Errorf(errFmtStationPresentedNoCertificate, station)
	}
	registered, err := b.directory.GetGlobal(ctx, claim.BaseStationEUI)
	if err != nil {
		return fmt.Errorf(errFmtRegisteredStationLookup, err)
	}
	if err := b.bind(ctx, claim, registered); err != nil {
		return err
	}
	b.recordExpiry(ctx, registered, claim.Certificate)
	return nil
}

// bind accepts the presented certificate when it matches the pinned
// fingerprint, or pins it while none is pinned.
func (b *stationCertificates) bind(ctx context.Context, claim StationCertificateClaim, registered RegisteredBaseStation) error {
	presented := crypto.CertFingerprintSHA256(claim.Certificate.Raw)
	if registered.TLSCertFingerprint != "" {
		return b.match(ctx, claim.BaseStationEUI, registered.TLSCertFingerprint, presented, errFmtStationCertMismatchFingerprint)
	}
	return b.pinFirst(ctx, claim, registered, presented)
}

// recordExpiry stores the bound certificate's expiry while the station has
// none; the expiry only feeds the certificate display, so a failed write
// never refuses the station.
func (b *stationCertificates) recordExpiry(ctx context.Context, registered RegisteredBaseStation, cert *x509.Certificate) {
	if registered.TLSCertExpiresAt != nil {
		return
	}
	recorded, err := b.directory.BackfillCertExpiryIfBlank(ctx, registered.TenantID, registered.ID, cert.NotAfter)
	if err != nil {
		b.logger.WarnContext(ctx, LogBSSCICertExpiryBackfillFailed, logger.FieldBsEui, registered.EUI, logger.FieldError, err)
		return
	}
	if recorded {
		b.logger.InfoContext(ctx, LogBSSCICertExpiryRecorded, logger.FieldBsEui, registered.EUI)
	}
}

// pinFirst pins the station's fingerprint while none is stored: to its
// stored certificate once the presented one matched it, or, with no stored
// certificate, to the presented certificate when it names the station.
func (b *stationCertificates) pinFirst(ctx context.Context, claim StationCertificateClaim, registered RegisteredBaseStation, presented string) error {
	station := mioty.FormatEUI64(claim.BaseStationEUI)
	switch {
	case registered.TLSCertificate != "":
		stored, err := crypto.CertFingerprintFromPEM([]byte(registered.TLSCertificate))
		if err != nil {
			return fmt.Errorf(errFmtStationStoredCertificateUnparsable, station, err)
		}
		if err := b.match(ctx, claim.BaseStationEUI, stored, presented, errFmtStationCertMismatchStored); err != nil {
			return err
		}
	case claim.SubjectEUI == nil:
		return fmt.Errorf(errFmtStationHasNoStoredCertificateIdentity, station)
	}
	pinned, err := b.directory.BackfillFingerprintIfBlank(ctx, registered.TenantID, registered.ID, presented)
	if err != nil {
		return fmt.Errorf(errFmtStationFingerprintBackfill, station, err)
	}
	if pinned {
		b.logger.InfoContext(ctx, LogBSSCICertFingerprintPinned, logger.FieldBsEui, claim.BaseStationEUI)
		return nil
	}
	// A concurrent connect pinned a fingerprint first: the presented
	// certificate must be the one it pinned.
	reloaded, err := b.directory.GetGlobal(ctx, claim.BaseStationEUI)
	if err != nil {
		return fmt.Errorf(errFmtRegisteredStationReload, err)
	}
	return b.match(ctx, claim.BaseStationEUI, reloaded.TLSCertFingerprint, presented, errFmtStationCertMismatchFingerprint)
}

// match refuses a presented fingerprint that is not the expected one.
func (b *stationCertificates) match(ctx context.Context, eui uint64, expected, presented, mismatch string) error {
	if expected == presented {
		return nil
	}
	b.logger.WarnContext(ctx, LogBSSCICertFingerprintMismatch, logger.FieldBsEui, eui)
	return fmt.Errorf(mismatch, mioty.FormatEUI64(eui))
}
