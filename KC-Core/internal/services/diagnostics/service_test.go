package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

const (
	testTenant     = int64(3)
	otherTenant    = int64(4)
	testServerName = "sc-test"
	testPassword   = "hunter2"
	testSecret     = "jwt-secret-value"
	testDBHost     = "db.internal"
	testKeyPath    = "/etc/kc/bssci.key"
	scaciOn        = true
	ownBsEui       = uint64(0x70B3D59CD0000001)
	foreignBsEui   = uint64(0x70B3D59CD0000009)
	limitBytes     = 1 << 20
	limitEvents    = 50
	limitSessions  = 10
	schemaVersion  = 158
)

var now = time.Date(2026, 9, 2, 9, 0, 0, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return now }

var _ clock.Clock = fixedClock{}

type fakeEvents struct {
	lastTenant int64
	lastLimit  int
}

func (f *fakeEvents) List(_ context.Context, tenantID int64, _ *grpcservices.EventFilters, limit, _ int) ([]*grpcservices.Event, int64, error) {
	f.lastTenant, f.lastLimit = tenantID, limit
	return []*grpcservices.Event{{ID: "e1", TenantID: tenantID, EventType: models.EventTypeEndpointCreated, Data: []byte(`{"epEui":"AA"}`)}}, 1, nil
}

type fakeSCACI struct{ lastTenant int64 }

func (f *fakeSCACI) GetSessionStatistics(_ context.Context, tenantID int64) (*models.SCACISessionStatistics, error) {
	f.lastTenant = tenantID
	return &models.SCACISessionStatistics{TenantID: tenantID, TotalSessions: 2, ActiveSessions: 1}, nil
}

type fakeBSSCI struct{}

func (fakeBSSCI) GetConnectedSessions() []map[string]interface{} {
	return []map[string]interface{}{
		{bssci.SessionKeyBaseStationEUI: ownBsEui, bssci.SessionKeyConnected: true, bssci.SessionKeyHandshakeComplete: true, bssci.SessionKeyNegotiatedVersion: "1.0.0", bssci.SessionKeyVendor: "v", bssci.SessionKeyModel: "m", bssci.SessionKeyName: "roof", bssci.SessionKeyClientVersion: "c", bssci.SessionKeyResolvedTenantID: testTenant},
		{bssci.SessionKeyBaseStationEUI: foreignBsEui, bssci.SessionKeyConnected: true, bssci.SessionKeyHandshakeComplete: true, bssci.SessionKeyResolvedTenantID: otherTenant},
	}
}

func secretConfig() *config.Config {
	cfg := &config.Config{}
	cfg.General.ServerName = testServerName
	cfg.General.Edition = config.EditionECE
	cfg.Storage.Password = testPassword
	cfg.Storage.Host = testDBHost
	cfg.Auth.HMACSecret = testSecret
	cfg.MQTT.Password = testPassword
	cfg.Redis.Password = testPassword
	cfg.Protocol.BSCITLS.KeyFile = testKeyPath
	cfg.Protocol.SCACIEnabled = scaciOn
	return cfg
}

func newService(events *fakeEvents, maxBytes int) *Service {
	return New(Deps{
		Release: func() (*version.Info, error) {
			return &version.Info{Version: "1.68.0", SchemaVersion: schemaVersion}, nil
		},
		Config:     secretConfig(),
		Events:     events,
		SCACI:      &fakeSCACI{},
		BSSCI:      fakeBSSCI{},
		Limits:     Limits{MaxBundleBytes: maxBytes, MaxEvents: limitEvents, MaxSessions: limitSessions, Timeout: time.Second},
		Clock:      fixedClock{},
		Log:        logger.NewNop(),
		ServerName: testServerName,
	})
}

func readArchive(t *testing.T, archive []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)
	files := map[string][]byte{}
	for _, f := range reader.File {
		rc, err := f.Open()
		require.NoError(t, err)
		var buf bytes.Buffer
		_, err = buf.ReadFrom(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		files[f.Name] = buf.Bytes()
	}
	return files
}

func TestBuild_ContainsOnlyTheAllowlistedEntries(t *testing.T) {
	events := &fakeEvents{}
	bundle, err := newService(events, limitBytes).Build(testutil.TestContext(), testTenant)
	require.NoError(t, err)
	assert.Equal(t, config.DiagnosticsArchiveFilename, bundle.Filename)
	assert.Equal(t, config.DiagnosticsContentType, bundle.ContentType)
	assert.Equal(t, now, bundle.GeneratedAt)

	files := readArchive(t, bundle.Archive)
	assert.Len(t, files, len(Entries))
	for _, name := range Entries {
		_, ok := files[name]
		assert.True(t, ok, "entry %s missing", name)
		assert.False(t, strings.ContainsAny(name, `/\`) || strings.Contains(name, ".."), "entry %s must be a bare file name", name)
	}
	assert.Equal(t, testTenant, events.lastTenant, "events are the caller's tenant only")
	assert.Equal(t, limitEvents, events.lastLimit)

	var schema schemaEntry
	require.NoError(t, json.Unmarshal(files[EntrySchema], &schema))
	assert.Equal(t, schemaVersion, schema.SchemaVersion)
}

func TestBuild_ProjectsConfigWithoutSecrets(t *testing.T) {
	bundle, err := newService(&fakeEvents{}, limitBytes).Build(testutil.TestContext(), testTenant)
	require.NoError(t, err)
	files := readArchive(t, bundle.Archive)
	raw := string(files[EntryConfig])
	assert.NotContains(t, raw, testPassword)
	assert.NotContains(t, raw, testSecret)
	assert.NotContains(t, raw, "bssci.key")
	assert.Contains(t, raw, `"scaciEnabled": true`)
	assert.Contains(t, raw, testDBHost)
	for _, name := range Entries {
		assert.NotContains(t, string(files[name]), testPassword, "%s must not carry credentials", name)
	}
}

func TestBuild_KeepsOtherTenantsBaseStationsOutOfTheSessionList(t *testing.T) {
	bundle, err := newService(&fakeEvents{}, limitBytes).Build(testutil.TestContext(), testTenant)
	require.NoError(t, err)
	files := readArchive(t, bundle.Archive)
	var sessions bsSessionsEntry
	require.NoError(t, json.Unmarshal(files[EntryBSSCISessions], &sessions))
	assert.Equal(t, 2, sessions.TotalConnected)
	require.Len(t, sessions.Sessions, 1)
	assert.Equal(t, "70B3D59CD0000001", sessions.Sessions[0].BsEui)
	assert.NotContains(t, string(files[EntryBSSCISessions]), "70B3D59CD0000009", "another tenant's base station identifier never appears")
}

func TestBuild_RejectsOversizedBundles(t *testing.T) {
	_, err := newService(&fakeEvents{}, 1).Build(testutil.TestContext(), testTenant)
	assert.ErrorIs(t, err, ErrBundleTooLarge)
}
