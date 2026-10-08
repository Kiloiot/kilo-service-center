package basestation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	adapterTestTenant = int64(1)
	knownURL          = "tls://bssci.example.com:5000"
)

type recordedStation struct {
	station   *models.BaseStation
	updates   []map[string]interface{}
	updateErr error
}

func (r *recordedStation) GetByEUI(context.Context, int64, []byte) (*models.BaseStation, error) {
	return r.station, nil
}

func (r *recordedStation) GetByEUIGlobal(context.Context, []byte) (*models.BaseStation, error) {
	return r.station, nil
}

func (r *recordedStation) Update(_ context.Context, _, _ int64, updates map[string]interface{}) error {
	r.updates = append(r.updates, updates)
	return r.updateErr
}

// warnRecorder records the messages logged as warnings.
type warnRecorder struct {
	logger.Logger
	warnings []string
}

func (w *warnRecorder) WarnContext(_ context.Context, msg string, _ ...interface{}) {
	w.warnings = append(w.warnings, msg)
}

func bssciStation(url *string) *models.BaseStation {
	return &models.BaseStation{TenantID: adapterTestTenant, ConnectionType: models.ConnectionTypeBSSCI, ServiceCenterURL: url}
}

func readStation(t *testing.T, store *recordedStation, canonical string) *BaseStation {
	t.Helper()
	station, err := NewRepositoryAdapter(store, adapterTestTenant, canonical, logger.NewNop()).GetBaseStation(testutil.TestContext(), [8]byte{1})
	require.NoError(t, err)
	return station
}

// With no URL a station can reach, the stored URL is NULL: an empty one is
// cleared, and a NULL one is left alone.
func TestGetBaseStation_StoresNoServiceCenterURLWhenNoneIsKnown(t *testing.T) {
	empty := ""
	withEmpty := &recordedStation{station: bssciStation(&empty)}
	readStation(t, withEmpty, "")
	require.Len(t, withEmpty.updates, 1)
	assert.Nil(t, withEmpty.updates[0][columnServiceCenterURL])

	withNull := &recordedStation{station: bssciStation(nil)}
	assert.Empty(t, readStation(t, withNull, "").ServiceCenterURL)
	assert.Empty(t, withNull.updates, "a NULL URL already says unknown")
}

func TestGetBaseStation_BackfillsTheKnownServiceCenterURL(t *testing.T) {
	store := &recordedStation{station: bssciStation(nil)}

	assert.Equal(t, knownURL, readStation(t, store, knownURL).ServiceCenterURL)
	require.Len(t, store.updates, 1)
	stored, ok := store.updates[0][columnServiceCenterURL].(*string)
	require.True(t, ok)
	assert.Equal(t, knownURL, *stored)
}

// A backfill the database refuses is logged, and the read still returns the station.
func TestGetBaseStation_LogsAFailedServiceCenterURLBackfill(t *testing.T) {
	store := &recordedStation{station: bssciStation(nil), updateErr: errors.New("update refused")}
	log := &warnRecorder{Logger: logger.NewNop()}

	station, err := NewRepositoryAdapter(store, adapterTestTenant, knownURL, log).GetBaseStation(testutil.TestContext(), [8]byte{1})

	require.NoError(t, err)
	assert.Equal(t, knownURL, station.ServiceCenterURL)
	assert.Equal(t, []string{LogFailedToBackfillServiceCenterURL}, log.warnings)
}

const adapterSessionID = "connection-1"

var adapterEUI = [8]byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x09, 0xe6}

func TestUpdateConnectionStatus_ActivationRecordsTheHandshakeTime(t *testing.T) {
	store := &recordedStation{station: bssciStation(nil)}
	handshake := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	err := NewRepositoryAdapter(store, adapterTestTenant, "", logger.NewNop()).UpdateConnectionStatus(testutil.TestContext(), adapterEUI, &ConnectionStatus{
		IsOnline: true, LastSeen: handshake, ConnectionType: ConnectionTypeBSSCI,
		SessionID: adapterSessionID, SessionStartedAt: handshake,
	})

	require.NoError(t, err)
	require.Len(t, store.updates, 1)
	assert.Equal(t, handshake, store.updates[0][fieldKeySessionStartedAt])
	assert.Equal(t, adapterSessionID, store.updates[0][fieldKeySessionUUID])
}

func TestUpdateConnectionStatus_LivenessKeepsTheHandshakeTime(t *testing.T) {
	store := &recordedStation{station: bssciStation(nil)}

	err := NewRepositoryAdapter(store, adapterTestTenant, "", logger.NewNop()).UpdateConnectionStatus(testutil.TestContext(), adapterEUI, &ConnectionStatus{
		IsOnline: true, LastSeen: time.Now(), ConnectionType: ConnectionTypeBSSCI,
	})

	require.NoError(t, err)
	require.Len(t, store.updates, 1)
	assert.NotContains(t, store.updates[0], fieldKeySessionStartedAt)
	assert.NotContains(t, store.updates[0], fieldKeySessionUUID)
}
