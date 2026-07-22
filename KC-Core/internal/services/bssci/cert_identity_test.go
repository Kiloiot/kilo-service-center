package bssciservices

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// makeTestCert creates a real self-signed x509 certificate with the given CN.
func makeTestCert(t *testing.T, cn string) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

// certIdentityBSRepo overrides the global lookup of the fieldless base
// mock with configurable results.
type certIdentityBSRepo struct {
	mockBaseStationRepo
	baseStation *models.BaseStation
	getErr      error
}

func (r *certIdentityBSRepo) GetByEUIGlobal(_ context.Context, _ []byte) (*models.BaseStation, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.baseStation, nil
}

// certIdentityOrgResolver extends the ingest fake with delegation tracking.
type certIdentityOrgResolver struct {
	fakeOrgResolver
	resolveCertOrg    uuid.UUID
	resolveCertTenant int64
	resolveCertErr    error
	resolveCertCalls  int
}

func (r *certIdentityOrgResolver) ResolveCert(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
	r.resolveCertCalls++
	if r.resolveCertErr != nil {
		return uuid.Nil, 0, r.resolveCertErr
	}
	return r.resolveCertOrg, r.resolveCertTenant, nil
}

// TestCertIdentity_EUICN_ResolvesRegisteredStation: a dashed-EUI CN (the CE
// issuance scheme) resolves via the global station lookup and carries the
// subject EUI for connect-time binding.
func TestCertIdentity_EUICN_ResolvesRegisteredStation(t *testing.T) {
	const eui = uint64(0xCAFECAFECAFECAFE)
	orgID := uuid.New()
	repo := &certIdentityBSRepo{
		baseStation: &models.BaseStation{ID: 7, TenantID: 42, Name: "CE BS"},
	}
	orgResolver := &certIdentityOrgResolver{
		fakeOrgResolver: fakeOrgResolver{defaultOrgByTenant: map[int64]uuid.UUID{42: orgID}},
	}
	resolver := NewCertificateIdentityResolver(repo, orgResolver, &mockLoggerForDispatch{})

	identity, err := resolver.ResolveCertificateIdentity(testutil.TestContext(), makeTestCert(t, "CA-FE-CA-FE-CA-FE-CA-FE"))

	require.NoError(t, err)
	assert.Equal(t, int64(42), identity.TenantID)
	assert.Equal(t, orgID, identity.OrganizationID)
	require.NotNil(t, identity.SubjectEUI, "an EUI CN must carry the subject EUI")
	assert.Equal(t, eui, *identity.SubjectEUI)
	assert.Zero(t, orgResolver.resolveCertCalls, "EUI CNs never delegate to ResolveCert")
}

// TestCertIdentity_EUICN_UnregisteredStationRejected: an EUI CN with no
// registered station cannot resolve.
func TestCertIdentity_EUICN_UnregisteredStationRejected(t *testing.T) {
	repo := &certIdentityBSRepo{getErr: errors.New("not found")}
	resolver := NewCertificateIdentityResolver(repo, &certIdentityOrgResolver{}, &mockLoggerForDispatch{})

	_, err := resolver.ResolveCertificateIdentity(testutil.TestContext(), makeTestCert(t, "CA-FE-CA-FE-CA-FE-CA-FE"))

	require.Error(t, err, "an unregistered EUI CN must not resolve")
}

// TestCertIdentity_OrgCN_DelegatesToOrgResolver: a legacy org-<UUID> CN
// delegates to the deployment's organization resolver unchanged and carries
// no subject EUI.
func TestCertIdentity_OrgCN_DelegatesToOrgResolver(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &certIdentityOrgResolver{resolveCertOrg: orgID, resolveCertTenant: 9}
	resolver := NewCertificateIdentityResolver(&mockBaseStationRepo{}, orgResolver, &mockLoggerForDispatch{})

	identity, err := resolver.ResolveCertificateIdentity(testutil.TestContext(), makeTestCert(t, "org-"+orgID.String()))

	require.NoError(t, err)
	assert.Equal(t, 1, orgResolver.resolveCertCalls, "org CNs delegate to ResolveCert")
	assert.Equal(t, orgID, identity.OrganizationID)
	assert.Equal(t, int64(9), identity.TenantID)
	assert.Nil(t, identity.SubjectEUI, "org CNs carry no station identity")
}

// TestCertIdentity_OrgCN_DelegateFailurePropagates: a delegated resolution
// failure surfaces (strict mode closes the connection on it).
func TestCertIdentity_OrgCN_DelegateFailurePropagates(t *testing.T) {
	orgResolver := &certIdentityOrgResolver{resolveCertErr: errors.New("unknown org")}
	resolver := NewCertificateIdentityResolver(&mockBaseStationRepo{}, orgResolver, &mockLoggerForDispatch{})

	_, err := resolver.ResolveCertificateIdentity(testutil.TestContext(), makeTestCert(t, "org-unknown"))

	require.Error(t, err)
}
