package builders

import (
	"context"
	"testing"
	"time"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	pkggrpc "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
)

// TestEnterpriseSeamSignatures pins the constructors the enterprise repository
// calls into this package. Most of them have no community-tree caller, so this
// contract is what keeps their signatures from drifting - or from being
// deleted as unreachable - without the enterprise consumer noticing at its
// next build.
//
// Enterprise call sites:
//   - KC-ECE/internal/builders/core_ece.go           -> NewCEBootstrapHandler
//   - KC-ECE/internal/builders/infrastructure_ece.go -> NewLocalDBOrgResolver
//   - KC-ECE/cmd/federation-ingress/main.go          -> BuildFederationIngestDeps, NewOrgResolverConfig
func TestEnterpriseSeamSignatures(t *testing.T) {
	// Typed declarations pin the exact signatures; the assignments fail to
	// compile if a seam drifts.
	var newBootstrap func(*FederationContext) (pkggrpc.CEBootstrapHandler, error)
	var newResolver func(interfaces.OrgDirectoryRepository, logger.Logger, time.Duration, int) (org.Resolver, error)
	var buildIngest func(context.Context, *Infrastructure) (*bssciservices.UplinkIngestServiceImpl, error)
	var resolverConfig func(*pkgconfig.Config) *OrgResolverConfig
	newBootstrap = NewCEBootstrapHandler
	newResolver = NewLocalDBOrgResolver
	buildIngest = BuildFederationIngestDeps
	resolverConfig = NewOrgResolverConfig

	// The seam values must exist as callable functions at runtime, which also
	// roots them for reachability analysis.
	for i, seam := range []any{newBootstrap, newResolver, buildIngest, resolverConfig} {
		if seam == nil {
			t.Fatalf("enterprise seam %d is nil", i)
		}
	}
}

// TestEnterpriseSeamFields pins the fields the enterprise composition roots
// read or populate: the federation wirer builds its registry repository on
// FederationContext.Storage, and the federation ingress assembles an
// Infrastructure by hand before calling BuildFederationIngestDeps.
func TestEnterpriseSeamFields(t *testing.T) {
	var storage *postgres.DB
	fctx := FederationContext{Storage: storage}

	var (
		repos    *postgres.Repositories
		resolver org.Resolver
		client   mqtt.Publisher
		cfg      *pkgconfig.Config
	)
	infra := Infrastructure{Config: cfg, Storage: storage, Repos: repos, OrgResolverSvc: resolver, MQTTClient: client}
	if fctx.Storage != nil || infra.Repos != nil || infra.OrgResolverSvc != nil {
		t.Fatal("zero-value seam fields must stay nil")
	}
}

const testMQTTEnabled = true

// TestDeliveryChannels_FollowConfigurationNotTheLocalClient covers a process
// that stores uplinks without publishing them itself: MQTT rows are queued
// whenever the deployment enables MQTT.
func TestDeliveryChannels_FollowConfigurationNotTheLocalClient(t *testing.T) {
	enabled := &Infrastructure{Config: &pkgconfig.Config{MQTT: pkgconfig.MQTTConfig{Enabled: testMQTTEnabled}}}
	disabled := &Infrastructure{Config: &pkgconfig.Config{}}

	if got := deliveryChannels(enabled); len(got) != 2 || got[0] != models.DeliveryChannelSCACI || got[1] != models.DeliveryChannelMQTT {
		t.Fatalf("MQTT enabled without a local client: got %v", got)
	}
	if got := deliveryChannels(disabled); len(got) != 1 || got[0] != models.DeliveryChannelSCACI {
		t.Fatalf("MQTT disabled: got %v", got)
	}
}
