package bssciservices

import (
	"context"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
)

// certificateIdentityResolver is the CE composite implementation of
// bssci.CertificateIdentityResolver: a CN that parses as an EUI-64 (the CE
// issuance scheme, dashed uppercase) resolves against the registered base
// stations; any other CN form (org-<UUID>) is delegated to the deployment's
// organization resolver, so the ECE remote resolver keeps its existing
// contract untouched.
type certificateIdentityResolver struct {
	bsRepo      BaseStationStore
	orgResolver org.Resolver
	logger      logger.Logger
}

// NewCertificateIdentityResolver builds the CE composite resolver.
func NewCertificateIdentityResolver(
	bsRepo BaseStationStore,
	orgResolver org.Resolver,
	log logger.Logger,
) bssci.CertificateIdentityResolver {
	return &certificateIdentityResolver{
		bsRepo:      bsRepo,
		orgResolver: orgResolver,
		logger:      log,
	}
}

// ResolveCertificateIdentity resolves the identity asserted by the client
// certificate CN. EUI CNs are resolved by a global base-station lookup (the
// tenant is not yet authenticated at TLS accept) with the tenant's default
// organization; other CN forms delegate to the organization resolver.
func (r *certificateIdentityResolver) ResolveCertificateIdentity(ctx context.Context, cert *x509.Certificate) (bssci.CertificateIdentity, error) {
	cn := cert.Subject.CommonName

	if eui, euiErr := validation.ParseEUI(cn); euiErr == nil {
		if r.bsRepo == nil {
			return bssci.CertificateIdentity{}, fmt.Errorf(errFmtCertEUIRepoUnavailable, cn)
		}
		euiBytes := binary.BigEndian.AppendUint64(nil, eui)
		bs, err := r.bsRepo.GetByEUIGlobal(ctx, euiBytes)
		if err != nil || bs == nil {
			return bssci.CertificateIdentity{}, fmt.Errorf(errFmtCertEUINoRegisteredBS, cn, err)
		}

		identity := bssci.CertificateIdentity{TenantID: bs.TenantID, SubjectEUI: &eui}
		if r.orgResolver != nil {
			orgID, orgErr := r.orgResolver.GetDefaultOrgForTenant(ctx, bs.TenantID)
			if orgErr != nil {
				r.logger.WarnContext(ctx, bssci.LogBSSCICertIdentityDefaultOrgLookupFailed,
					logger.FieldTenantID, bs.TenantID,
					logger.FieldError, orgErr)
			} else {
				identity.OrganizationID = orgID
			}
		}
		return identity, nil
	}

	if r.orgResolver == nil {
		return bssci.CertificateIdentity{}, fmt.Errorf(errFmtCertCNNoOrgResolver, cn)
	}
	orgID, tenantID, err := r.orgResolver.ResolveCert(ctx, cert)
	if err != nil {
		return bssci.CertificateIdentity{}, err
	}
	return bssci.CertificateIdentity{OrganizationID: orgID, TenantID: tenantID}, nil
}

// registeredBaseStationDirectory adapts the base-station repository to the
// narrow bssci.RegisteredBaseStationDirectory read/backfill contract.
type registeredBaseStationDirectory struct {
	bsRepo RegisteredStationStore
}

// NewRegisteredBaseStationDirectory builds the repository-backed directory.
func NewRegisteredBaseStationDirectory(bsRepo RegisteredStationStore) bssci.RegisteredBaseStationDirectory {
	return &registeredBaseStationDirectory{bsRepo: bsRepo}
}

// GetGlobal returns the registration identity for an EUI across all tenants.
func (d *registeredBaseStationDirectory) GetGlobal(ctx context.Context, eui uint64) (bssci.RegisteredBaseStation, error) {
	euiBytes := binary.BigEndian.AppendUint64(nil, eui)
	bs, err := d.bsRepo.GetByEUIGlobal(ctx, euiBytes)
	if err != nil {
		return bssci.RegisteredBaseStation{}, err
	}
	if bs == nil {
		return bssci.RegisteredBaseStation{}, fmt.Errorf(errFmtBaseStationNotRegistered, mioty.FormatEUI64(eui))
	}

	registered := bssci.RegisteredBaseStation{
		ID:       bs.ID,
		TenantID: bs.TenantID,
		EUI:      eui,
		Name:     bs.Name,
	}
	if bs.TLSCertificate != nil {
		registered.TLSCertificate = *bs.TLSCertificate
	}
	if bs.TLSCertFingerprint != nil {
		registered.TLSCertFingerprint = *bs.TLSCertFingerprint
	}
	registered.TLSCertExpiresAt = bs.TLSCertExpiresAt
	return registered, nil
}

// BackfillFingerprintIfBlank persists the fingerprint only while the stored
// value is still blank (conditional SQL; see repository contract).
func (d *registeredBaseStationDirectory) BackfillFingerprintIfBlank(ctx context.Context, tenantID, id int64, fingerprint string) (bool, error) {
	return d.bsRepo.UpdateTLSFingerprintIfBlank(ctx, tenantID, id, fingerprint)
}

// BackfillCertExpiryIfBlank persists the certificate expiry only while none
// is stored (conditional SQL; see repository contract).
func (d *registeredBaseStationDirectory) BackfillCertExpiryIfBlank(ctx context.Context, tenantID, id int64, expiresAt time.Time) (bool, error) {
	return d.bsRepo.UpdateTLSCertExpiryIfBlank(ctx, tenantID, id, expiresAt)
}

// interface guards
var (
	_ bssci.CertificateIdentityResolver    = (*certificateIdentityResolver)(nil)
	_ bssci.RegisteredBaseStationDirectory = (*registeredBaseStationDirectory)(nil)
)
