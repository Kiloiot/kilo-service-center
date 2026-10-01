//go:build integration

package probe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// The isolated stack (scripts/conformance-stack.sh) publishes its addresses
// through these variables. They are deliberately distinct from the gate
// probes' BSSCI_ADDR/CORE_INTERNAL_ADDR, so the suite can never run against a
// developer's stack by accident.
const (
	envBSSCIAddr          = "CONFORMANCE_BSSCI_ADDR"
	envSCACIAddr          = "CONFORMANCE_SCACI_ADDR"
	envCoreAddr           = "CONFORMANCE_CORE_ADDR"
	envMQTTAddr           = "CONFORMANCE_MQTT_ADDR"
	envMQTTPrefix         = "CONFORMANCE_MQTT_PREFIX"
	envCertDir            = "CONFORMANCE_CERT_DIR"
	envDBDSN              = "CONFORMANCE_DB_DSN"
	envPrimaryOrg         = "CONFORMANCE_TENANT_PRIMARY_ORG"
	envSecondaryOrg       = "CONFORMANCE_TENANT_SECONDARY_ORG"
	envDuplicateWindowSec = "CONFORMANCE_DUPLICATE_WINDOW_SECONDS"
	envReceptionWindow    = "CONFORMANCE_RECEPTION_WINDOW"
	envUser               = "CONFORMANCE_USER_ID"
)

const (
	tenantPrimary   int64 = 1
	tenantSecondary int64 = 4
	caFile                = "ca.crt"
	acCertName            = "ac"
	serverName            = "localhost"
	apiTimeout            = 15 * time.Second
	dialTimeout           = 5 * time.Second
)

func stackEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s is not set: run the suite through scripts/conformance-stack.sh (make test-conformance)", name)
	}
	return v
}

// station is a simulated base station registered by TestSeedConformanceStack.
type station struct {
	eui    uint64
	cert   string
	tenant int64
}

var (
	stationA      = station{eui: 0x70B3D59CD000FF01, cert: "bs-ff01", tenant: tenantPrimary}
	stationB      = station{eui: 0x70B3D59CD000FF02, cert: "bs-ff02", tenant: tenantPrimary}
	stationRoamer = station{eui: 0x70B3D59CD000FF04, cert: "bs-ff04", tenant: tenantSecondary}
	// stationDeleted is registered and deleted by J9 itself; the seed leaves it out.
	stationDeleted = station{eui: 0x70B3D59CD000FF03, cert: "bs-ff03", tenant: tenantPrimary}
)

// TestSeedConformanceStack registers the simulated base stations. The stack
// script runs it once after KC-Core is up; it is idempotent.
func TestSeedConformanceStack(t *testing.T) {
	for _, st := range []station{stationA, stationB, stationRoamer} {
		_, err := coreClient(t).CreateBaseStation(apiCtx(t, st.tenant), &pb.CreateBaseStationRequest{
			Basestation: &pb.BaseStation{BsEui: euiHex(st.eui), Name: fmt.Sprintf("conformance-%s", st.cert)},
		})
		if status.Code(err) != codes.AlreadyExists {
			require.NoError(t, err, "register station %s", st.cert)
		}
	}
}

func tlsClientConfig(t *testing.T, certName string, minVersion uint16) *tls.Config {
	t.Helper()
	dir := stackEnv(t, envCertDir)
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, certName+".crt"), filepath.Join(dir, certName+".key"))
	require.NoError(t, err, "load client certificate %s", certName)
	caPEM, err := os.ReadFile(filepath.Join(dir, caFile)) //nolint:gosec // path from the stack script
	require.NoError(t, err, "read CA")
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPEM), "parse CA")
	return &tls.Config{Certificates: []tls.Certificate{pair}, RootCAs: pool, ServerName: serverName, MinVersion: minVersion}
}

func dialTLS(t *testing.T, addrEnv, certName string, minVersion uint16) net.Conn {
	t.Helper()
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", stackEnv(t, addrEnv),
		tlsClientConfig(t, certName, minVersion))
	require.NoError(t, err, "TLS dial %s", addrEnv)
	return conn
}

var (
	coreOnce sync.Once
	coreConn *grpc.ClientConn
	coreErr  error
)

// coreClient is KC-Core's internal gRPC API, reached with the trusted identity
// headers KC-Gateway would inject.
func coreClient(t *testing.T) pb.CoreServiceClient {
	t.Helper()
	coreOnce.Do(func() {
		coreConn, coreErr = grpc.NewClient(os.Getenv(envCoreAddr), grpc.WithTransportCredentials(insecure.NewCredentials()))
	})
	stackEnv(t, envCoreAddr)
	require.NoError(t, coreErr, "connect KC-Core gRPC")
	return pb.NewCoreServiceClient(coreConn)
}

func tenantOrg(t *testing.T, tenant int64) string {
	t.Helper()
	if tenant == tenantSecondary {
		return stackEnv(t, envSecondaryOrg)
	}
	return stackEnv(t, envPrimaryOrg)
}

func apiCtx(t *testing.T, tenant int64) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	t.Cleanup(cancel)
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, strconv.FormatInt(tenant, 10),
		grpcconst.MetadataKeyInternalOrgID, tenantOrg(t, tenant),
		grpcconst.MetadataKeyInternalUserID, stackEnv(t, envUser),
	))
}

func euiHex(eui uint64) string { return fmt.Sprintf("%016X", eui) }

// Endpoint EUIs are unique per test run so a suite can be repeated against
// the same stack: the run nonce fills the middle bits, a counter the low ones.
const (
	endpointEUIBase  uint64 = 0x70B3D5E000000000
	endpointRunShift        = 16
	endpointRunMask  uint64 = 0xFFFFFF
)

var (
	endpointRun = (uint64(time.Now().Unix()) & endpointRunMask) << endpointRunShift //nolint:gosec // positive by construction
	endpointSeq atomic.Uint64
)

func nextEndpointEUI() uint64 {
	return endpointEUIBase | endpointRun | endpointSeq.Add(1)
}
