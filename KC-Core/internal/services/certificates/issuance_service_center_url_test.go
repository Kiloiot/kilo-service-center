package certificates

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// A new station's certificate bundle carries the external BSSCI URL, and no
// URL at all when that URL names an address no base station can reach.
func TestGenerateCertificate_ServiceCenterURLFromExternalURL(t *testing.T) {
	tests := []struct {
		name        string
		externalURL string
		want        string
	}{
		{name: "reachable external URL", externalURL: "tls://bssci.example.com:5000", want: "tls://bssci.example.com:5000"},
		{name: "loopback external URL", externalURL: "tls://localhost:5000", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newIssuanceTestService(t, &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: 42}})
			svc.settings.protocol = &config.ProtocolConfig{BSCIExternalURL: tt.externalURL, BSCIHost: "0.0.0.0"}

			resp, err := svc.GenerateCertificate(testutil.TestContext(), &grpcservices.CertificateRequest{
				BsEUI:        "cafecafecafecafe",
				ValidityDays: 365,
				TenantID:     42,
			})
			if err != nil {
				t.Fatalf("issuance failed: %v", err)
			}
			if resp.ServiceCenterURL != tt.want {
				t.Errorf("ServiceCenterURL = %q, want %q", resp.ServiceCenterURL, tt.want)
			}
		})
	}
}
