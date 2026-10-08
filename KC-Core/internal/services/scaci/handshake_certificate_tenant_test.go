package scaciservices

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	refusalDefaultTenant     int64 = 1
	refusalCertificateTenant int64 = 12
)

const refusalSCEui uint64 = 0x70B3D59CD0000001

var errRefusalUnmapped = errors.New("certificate maps to no organization")

func newRefusalHandshake(resolver *mockOrgResolver) scaci.HandshakeService {
	return NewHandshakeService(&mockSCACISessionRepository{}, logger.NewNop(), resolver, refusalDefaultTenant, true,
		&mockCertificateVerifier{}, refusalSCEui, "", "", "", "", scaci.NewSessionFactory(clock.SystemClock{}))
}

func TestCertificateTenant_IsTheTenantTheCertificateResolvesTo(t *testing.T) {
	svc := newRefusalHandshake(&mockOrgResolver{
		resolveCertFunc: func(context.Context, *x509.Certificate) (uuid.UUID, int64, error) {
			return uuid.New(), refusalCertificateTenant, nil
		},
	})

	tenant, ok := svc.CertificateTenant(testutil.TestContext(), &x509.Certificate{Subject: pkix.Name{CommonName: "ac"}})

	assert.True(t, ok)
	assert.Equal(t, refusalCertificateTenant, tenant)
}

func TestCertificateTenant_NeverFallsBackToTheDefaultTenant(t *testing.T) {
	svc := newRefusalHandshake(&mockOrgResolver{
		resolveCertFunc: func(context.Context, *x509.Certificate) (uuid.UUID, int64, error) {
			return uuid.Nil, 0, errRefusalUnmapped
		},
	})

	for _, cert := range []*x509.Certificate{{}, nil} {
		tenant, ok := svc.CertificateTenant(testutil.TestContext(), cert)
		assert.False(t, ok, "a certificate of no organization resolves to no tenant")
		assert.Zero(t, tenant)
	}
}
