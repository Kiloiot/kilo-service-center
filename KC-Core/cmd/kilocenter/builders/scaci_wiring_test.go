package builders

import (
	"context"
	"errors"
	"testing"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errBindFailed = errors.New("bind failed")
)

// fakeSCACIRuntime implements the scaciRuntime seam, recording lifecycle and
// broadcast traffic.
type fakeSCACIRuntime struct {
	startErr   error
	startCalls int
	ulData     int
	dlResults  int
	epStatuses int
}

func (f *fakeSCACIRuntime) Start() error {
	f.startCalls++
	return f.startErr
}

func (f *fakeSCACIRuntime) BroadcastULData(context.Context, int64, *mioty.ULDataMessage) error {
	f.ulData++
	return nil
}

func (f *fakeSCACIRuntime) BroadcastDLDataResult(context.Context, scaci.ApplicationCenter, uint64, *mioty.DLDataResult) error {
	f.dlResults++
	return nil
}

func (f *fakeSCACIRuntime) BroadcastEPStatus(context.Context, int64, *scaci.EPStatusData) error {
	f.epStatuses++
	return nil
}

func newWiringBundle() *bssciservices.BSSCIServiceBundle {
	return &bssciservices.BSSCIServiceBundle{
		Broadcaster:         bssciservices.NewSCACIForwarder(logger.NewNop()),
		EPStatusBroadcaster: bssciservices.NewSCACIEPStatusAdapter(logger.NewNop()),
	}
}

// broadcastAll pushes one message of each forwarded kind through the bundle's
// forwarders and reports how much of it reached the fake server.
func broadcastAll(t *testing.T, bundle *bssciservices.BSSCIServiceBundle) {
	t.Helper()
	ctx := testutil.TestContext()
	require.NoError(t, bundle.Broadcaster.BroadcastULData(ctx, 1, &mioty.ULDataMessage{}))
	require.NoError(t, bundle.Broadcaster.BroadcastDLDataResult(ctx, scaci.ApplicationCenter{TenantID: 1, AcEui: 0x70B3D59CD0000A01}, 1, &mioty.DLDataResult{}))
	require.NoError(t, bundle.EPStatusBroadcaster.BroadcastEPStatus(ctx, 1, &bssci.EPStatusData{EpEui: 1, EpStatus: "attached"}))
}

func TestStartAndWireSCACI_MissingBroadcaster(t *testing.T) {
	bundle := newWiringBundle()
	bundle.Broadcaster = nil
	server := &fakeSCACIRuntime{}
	err := startAndWireSCACI(bundle, server, true)
	require.Error(t, err)
	assert.Zero(t, server.startCalls, "Start must not run with a missing broadcaster")
}

func TestStartAndWireSCACI_MissingEPStatusBroadcaster(t *testing.T) {
	bundle := newWiringBundle()
	bundle.EPStatusBroadcaster = nil
	server := &fakeSCACIRuntime{}
	err := startAndWireSCACI(bundle, server, true)
	require.Error(t, err)
	assert.Zero(t, server.startCalls, "Start must not run with a missing EPStatus broadcaster")
}

func TestStartAndWireSCACI_NilBundle(t *testing.T) {
	server := &fakeSCACIRuntime{}
	require.Error(t, startAndWireSCACI(nil, server, true))
	assert.Zero(t, server.startCalls)
}

func TestStartAndWireSCACI_NilServer(t *testing.T) {
	require.Error(t, startAndWireSCACI(newWiringBundle(), nil, true))
}

func TestStartAndWireSCACI_StartFailureLeavesForwardingUnwired(t *testing.T) {
	bundle := newWiringBundle()
	server := &fakeSCACIRuntime{startErr: errBindFailed}
	err := startAndWireSCACI(bundle, server, true)
	require.Error(t, err)
	assert.Equal(t, 1, server.startCalls)

	broadcastAll(t, bundle)
	assert.Zero(t, server.ulData, "no setter may run after a failed Start")
	assert.Zero(t, server.dlResults)
	assert.Zero(t, server.epStatuses)
}

func TestStartAndWireSCACI_SuccessWiresAllForwarding(t *testing.T) {
	bundle := newWiringBundle()
	server := &fakeSCACIRuntime{}
	require.NoError(t, startAndWireSCACI(bundle, server, true))
	assert.Equal(t, 1, server.startCalls)

	broadcastAll(t, bundle)
	assert.Equal(t, 1, server.ulData, "UL data must traverse the forwarder into the SCACI server")
	assert.Equal(t, 1, server.dlResults, "DL results must traverse the forwarder into the SCACI server")
	assert.Equal(t, 1, server.epStatuses, "EPStatus must traverse the adapter into the SCACI server")
}

// TestStartAndWireSCACI_ListenerDisabledStillWiresForwarding verifies that
// with the socket listener disabled the application core stays fully wired:
// no Start call, but every forwarding setter is applied so gRPC/MQTT
// queueing and BSSCI forwarding keep working.
func TestStartAndWireSCACI_ListenerDisabledStillWiresForwarding(t *testing.T) {
	bundle := newWiringBundle()
	server := &fakeSCACIRuntime{}
	require.NoError(t, startAndWireSCACI(bundle, server, false))
	assert.Zero(t, server.startCalls, "the listener must not start when disabled")

	broadcastAll(t, bundle)
	assert.Equal(t, 1, server.ulData)
	assert.Equal(t, 1, server.dlResults)
	assert.Equal(t, 1, server.epStatuses)
}

func TestRoamingDetectorConfig_CarriesEveryConfiguredSetting(t *testing.T) {
	configured := pkgconfig.RoamingConfig{
		CacheEnabled:     !pkgconfig.DefaultProtocolRoamingCacheEnabled,
		CacheTTL:         2 * pkgconfig.DefaultProtocolRoamingCacheTTL,
		CacheMaxSize:     2 * pkgconfig.DefaultProtocolRoamingCacheMaxSize,
		EnableAuditTrail: !pkgconfig.DefaultProtocolRoamingEnableAuditTrail,
	}
	assert.Equal(t, roaming.DetectorConfig{
		CacheEnabled:     configured.CacheEnabled,
		CacheTTL:         configured.CacheTTL,
		CacheMaxSize:     configured.CacheMaxSize,
		EnableAuditTrail: configured.EnableAuditTrail,
	}, roamingDetectorConfig(configured))
}

// presentResolver is a non-nil org resolver; the guard never calls it.
type presentResolver struct{ org.Resolver }

const (
	testSCACIListenerEnabled  = true
	testSCACIListenerDisabled = false
	testFlagOn                = true
	testFlagOff               = false
)

func TestGuardSCACIOrgResolution(t *testing.T) {
	cases := []struct {
		name        string
		listener    bool
		strict      bool
		certMapping bool
		resolver    org.Resolver
		wantErr     string
	}{
		{name: "listener disabled ignores strict mode without mapping or resolver", listener: testSCACIListenerDisabled, strict: testFlagOn, certMapping: testFlagOff},
		{name: "listener enabled strict mode needs a resolver", listener: testSCACIListenerEnabled, strict: testFlagOn, certMapping: testFlagOn, wantErr: errMsgStrictOrgResolutionNilOrgResolver},
		{name: "listener enabled strict mode fully configured", listener: testSCACIListenerEnabled, strict: testFlagOn, certMapping: testFlagOn, resolver: presentResolver{}},
		{name: "listener enabled lenient mode", listener: testSCACIListenerEnabled, strict: testFlagOff, certMapping: testFlagOff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			protocol := &pkgconfig.ProtocolConfig{
				SCACIEnabled:           tc.listener,
				StrictOrgResolution:    tc.strict,
				SCACICertTenantMapping: tc.certMapping,
			}
			err := guardSCACIOrgResolution(protocol, tc.resolver, logger.NewNop())
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.wantErr)
		})
	}
}
