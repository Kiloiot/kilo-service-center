package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// The base station list and detail carry the expiry of the certificate the
// service center issued, so the list's Certificate Expiry column can show it.
func TestBaseStationToProto_CarriesTheCertificateExpiry(t *testing.T) {
	expires := time.Date(2029, 9, 27, 12, 57, 9, 0, time.UTC)
	withCert := &models.BaseStation{TenantID: 1, Name: "tims base", TLSCertExpiresAt: &expires}
	withoutCert := &models.BaseStation{TenantID: 1, Name: "no certificate"}

	result := mustBaseStationProto(t, withCert)

	require.NotNil(t, result.CertificateExpiresAt)
	assert.True(t, expires.Equal(result.CertificateExpiresAt.AsTime()))
	assert.Nil(t, mustBaseStationProto(t, withoutCert).CertificateExpiresAt, "a station without a certificate leaves it unset")
}
