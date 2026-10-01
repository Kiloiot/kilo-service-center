package builders

import (
	"strings"
	"testing"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

const roleSourceTestIdentityAddress = "kc-identity:50052"

func TestBuildRoleSource_RequiresIdentity(t *testing.T) {
	_, err := buildRoleSource(&Infrastructure{Config: &pkgconfig.Config{}})
	if err == nil || err.Error() != errMsgRoleSourceNeedsIdentityAddress {
		t.Fatalf("without identity.address: %v, want the error naming identity.address", err)
	}

	withAddress := &pkgconfig.Config{Identity: pkgconfig.IdentityConfig{Address: roleSourceTestIdentityAddress}}
	_, err = buildRoleSource(&Infrastructure{Config: withAddress})
	if err == nil || !strings.Contains(err.Error(), roleSourceTestIdentityAddress) {
		t.Fatalf("with identity.address but no client: %v, want the error naming the unconnected address", err)
	}

	source, err := buildRoleSource(&Infrastructure{
		Config:                 &pkgconfig.Config{},
		IdentityInternalClient: pb.NewIdentityInternalServiceClient(nil),
	})
	if err != nil || source == nil {
		t.Fatalf("with KC-Identity: %v", err)
	}
}
