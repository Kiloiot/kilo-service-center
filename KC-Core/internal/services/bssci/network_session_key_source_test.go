package bssciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	sessionKeyTestTenant   = int64(5)
	sessionKeyTestEndpoint = int64(51)
	sessionKeyTestEUI      = uint64(0x70B3D56770111505)
	sessionKeyWaitBound    = 2 * time.Second
)

var errSessionKeyTestStore = errors.New("store unavailable")

func newSessionKeySource(t *testing.T, endpoints EndpointProvisioningReader, sessions ActiveEndpointSessionReader) bssci.NetworkSessionKeySource {
	t.Helper()
	source, err := NewNetworkSessionKeySource(endpoints, sessions)
	require.NoError(t, err)
	return source
}

func TestNewNetworkSessionKeySourceRefusesMissingCollaborators(t *testing.T) {
	_, err := NewNetworkSessionKeySource(nil, sessionKeySessions{})
	assert.ErrorIs(t, err, errNilProvisioningReader)
	_, err = NewNetworkSessionKeySource(sessionKeyEndpoints{}, nil)
	assert.ErrorIs(t, err, errNilActiveSessionReader)
}

// fixedSessionKeys serves every endpoint on its pre-shared key.
type fixedSessionKeys struct{}

func (fixedSessionKeys) NetworkSessionKey(_ context.Context, endpoint *models.EndPoint) ([]byte, error) {
	return endpoint.NwkSnKey, nil
}

type sessionKeyEndpoints struct {
	endpoint *models.EndPoint
	err      error
}

func (e sessionKeyEndpoints) GetByID(_ context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	if e.err != nil {
		return nil, e.err
	}
	if e.endpoint.ID != id || e.endpoint.TenantID != tenantID {
		return nil, storage.ErrNotFound
	}
	return e.endpoint, nil
}

type sessionKeySessions struct {
	session *models.EndPointSession
	err     error
}

func (s sessionKeySessions) GetActive(_ context.Context, endpointID string) (*models.EndPointSession, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.session == nil || strconv.FormatInt(s.session.EndPointID, 10) != endpointID {
		return nil, storage.ErrNotFound
	}
	return s.session, nil
}

// overTheAirEndpoint is an endpoint whose last over-the-air attach recorded
// the nonce and signature, with the session key that attach derived.
func overTheAirEndpoint(t *testing.T) (endpoint *models.EndPoint, sessionKey []byte) {
	t.Helper()
	endpoint = &models.EndPoint{
		ID:       sessionKeyTestEndpoint,
		TenantID: sessionKeyTestTenant,
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		Nonce:    []byte{0xA1, 0xB2, 0xC3, 0xD4},
		Sign:     []byte{0x11, 0x22, 0x33, 0x44},
	}
	binary.BigEndian.PutUint64(endpoint.EUI[:], sessionKeyTestEUI)
	sessionKey, err := bssci.DeriveSessionKey(sessionKeyTestEUI, endpoint.Nonce, endpoint.Sign, endpoint.NwkSnKey)
	require.NoError(t, err)
	return endpoint, sessionKey
}

func activeSession(key []byte) *models.EndPointSession {
	return &models.EndPointSession{EndPointID: sessionKeyTestEndpoint, TenantID: sessionKeyTestTenant, SessionKey: key}
}

func TestNetworkSessionKeySource(t *testing.T) {
	endpoint, sessionKey := overTheAirEndpoint(t)

	t.Run("over-the-air session in force", func(t *testing.T) {
		source := newSessionKeySource(t, sessionKeyEndpoints{endpoint: endpoint}, sessionKeySessions{session: activeSession(sessionKey)})
		key, err := source.NetworkSessionKey(testutil.TestContext(), endpoint)
		require.NoError(t, err)
		assert.Equal(t, sessionKey, key)
	})

	t.Run("no session yet", func(t *testing.T) {
		source := newSessionKeySource(t, sessionKeyEndpoints{endpoint: endpoint}, sessionKeySessions{})
		key, err := source.NetworkSessionKey(testutil.TestContext(), endpoint)
		require.NoError(t, err)
		assert.Equal(t, endpoint.NwkSnKey, key)
	})

	t.Run("pre-attached session", func(t *testing.T) {
		source := newSessionKeySource(t, sessionKeyEndpoints{endpoint: endpoint}, sessionKeySessions{session: activeSession(endpoint.NwkSnKey)})
		key, err := source.NetworkSessionKey(testutil.TestContext(), endpoint)
		require.NoError(t, err)
		assert.Equal(t, endpoint.NwkSnKey, key)
	})

	t.Run("session store failure", func(t *testing.T) {
		source := newSessionKeySource(t, sessionKeyEndpoints{endpoint: endpoint}, sessionKeySessions{err: errSessionKeyTestStore})
		_, err := source.NetworkSessionKey(testutil.TestContext(), endpoint)
		require.ErrorIs(t, err, errLoadEndpointSession)
		require.ErrorIs(t, err, errSessionKeyTestStore)
	})

	t.Run("provisioning read failure", func(t *testing.T) {
		source := newSessionKeySource(t, sessionKeyEndpoints{err: errSessionKeyTestStore}, sessionKeySessions{session: activeSession(sessionKey)})
		_, err := source.NetworkSessionKey(testutil.TestContext(), endpoint)
		require.ErrorIs(t, err, errLoadEndpointProvisioning)
		require.ErrorIs(t, err, errSessionKeyTestStore)
	})
}

// recordingPropagateSender records the key each attach propagation carried.
type recordingPropagateSender struct {
	keys chan []byte
}

func (r *recordingPropagateSender) SendAttachPropagateToAll(_ uint64, nwkSnKey []byte, _ uint16, _ bool, _ uint32, _ bool, _ uint8, _ bool, _ bool) []error {
	r.keys <- nwkSnKey
	return nil
}

func (r *recordingPropagateSender) SendDetachPropagateToAll(uint64) []error { return nil }

func (r *recordingPropagateSender) GetConnectedSessionEUIs() []string { return nil }

type sessionKeyEndpointLookup struct {
	endpoint *models.EndPoint
}

func (l sessionKeyEndpointLookup) GetByEUI(context.Context, int64, []byte) (*models.EndPoint, error) {
	clone := *l.endpoint
	return &clone, nil
}

func (l sessionKeyEndpointLookup) EndpointDetachStateUpdate(context.Context, int64, int64, models.EndpointDetachStateParams) error {
	return nil
}

func (l sessionKeyEndpointLookup) TransitionEndpointStatus(context.Context, int64, int64, string) (bool, error) {
	return true, nil
}

func (l sessionKeyEndpointLookup) RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error) {
	return l.TransitionEndpointStatus(ctx, tenantID, endpointID, status)
}

// An attach started from the API propagates an over-the-air endpoint's
// session key, not its pre-shared key.
func TestAttachEndPointPropagatesTheCurrentSessionKey(t *testing.T) {
	endpoint, sessionKey := overTheAirEndpoint(t)
	sender := &recordingPropagateSender{keys: make(chan []byte, 1)}
	source := newSessionKeySource(t, sessionKeyEndpoints{endpoint: endpoint}, sessionKeySessions{session: activeSession(sessionKey)})
	svc := newAttachmentFixture(t, sessionKeyEndpointLookup{endpoint: endpoint}, noEndpointCreation{}, sender, source,
		&failureEventStore{recorded: make(chan error, 1)}).svc

	_, err := svc.AttachEndPoint(testutil.TestContext(), "70B3D56770111505", sessionKeyTestTenant)
	require.NoError(t, err)

	select {
	case key := <-sender.keys:
		assert.Equal(t, sessionKey, key)
	case <-time.After(sessionKeyWaitBound):
		t.Fatal("no attach propagation was sent")
	}
}
