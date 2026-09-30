package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// The ownedCertStations fixture stores no certificate copy, like a station registered without issuance.
func TestDownloadBaseStationCertificate_CAIsServedWithoutAStoredCopy(t *testing.T) {
	svc := newCertificateHandlersOverRealService(t)

	resp, err := svc.DownloadBaseStationCertificate(contextForTenant(testOwnerTenant),
		&pb.DownloadBaseStationCertificateRequest{BsEui: testOwnedBsEui, CertType: certificates.CertTypeCA})
	require.NoError(t, err)
	assert.Equal(t, []byte(certTestStagedCA), resp.Content, "the station CA is the service center CA")
	assert.Equal(t, "basestation-"+testOwnedBsEui+"-ca-certificate.crt", resp.Filename)
	assert.Equal(t, grpcerrors.ContentTypePEM, resp.ContentType)
}

func TestDownloadBaseStationCertificate_ClientWithoutAStoredCopyIsNotStored(t *testing.T) {
	svc := newCertificateHandlersOverRealService(t)

	_, err := svc.DownloadBaseStationCertificate(contextForTenant(testOwnerTenant),
		&pb.DownloadBaseStationCertificateRequest{BsEui: testOwnedBsEui, CertType: certificates.CertTypeClient})
	require.Error(t, err)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertNotStored), status.Code(err))
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertNotStored), status.Convert(err).Message())
}

func TestDownloadBaseStationCertificate_ForeignStationIsNotFound(t *testing.T) {
	svc := newCertificateHandlersOverRealService(t)

	for _, certType := range []string{certificates.CertTypeCA, certificates.CertTypeClient} {
		_, err := svc.DownloadBaseStationCertificate(contextForTenant(testForeignTenant),
			&pb.DownloadBaseStationCertificateRequest{BsEui: testOwnedBsEui, CertType: certType})
		require.Error(t, err, "another tenant must not download the %s of this station", certType)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound), status.Convert(err).Message())
	}
}
