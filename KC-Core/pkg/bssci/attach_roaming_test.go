package bssci

import (
	"context"
	"encoding/binary"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Tenants of the roaming attach tests: the base station belongs to the
// serving tenant, the endpoint to its owner.
const (
	attachRoamingServingTenant = int64(1)
	attachRoamingOwnerTenant   = int64(2)
	attachRoamingAttachCnt     = uint32(5)
)

var errAttachRoamingLookup = errors.New("endpoint directory unavailable")

// attachRecordingRoaming counts the roaming attach events it recorded.
type attachRecordingRoaming struct {
	stubRoamingService
	attaches int
}

func (r *attachRecordingRoaming) RecordAttach(context.Context, []byte, []byte, int64) error {
	r.attaches++
	return nil
}

type attachFixture struct {
	server      *Server
	session     *Session
	conn        *bsscitest.TestConn
	endpoints   *fakeEndpointRepo
	persistence *recordingAttachPersistence
	events      *recordingEventStore
}

// newAttachRoamingFixture serves an attach on a base station of the serving
// tenant for an endpoint provisioned under the owner tenant.
func newAttachRoamingFixture(t *testing.T, roaming RoamingService) *attachFixture {
	t.Helper()
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], TestEpEui01)
	return newAttachFixture(t, &models.EndPoint{
		ID:       3001,
		EUI:      eui,
		TenantID: attachRoamingOwnerTenant,
		NwkSnKey: testPresharedKey(),
	}, attachRoamingServingTenant, roaming)
}

// newAttachFixture serves an attach for the endpoint on a base station of the
// serving tenant.
func newAttachFixture(t *testing.T, endpoint *models.EndPoint, servingTenant int64, roaming RoamingService) *attachFixture {
	t.Helper()
	endpoints := newFakeEndpointRepo(endpoint)

	log := logger.NewNop()
	eventStore := &recordingEventStore{}
	storage := newStubStorage()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(log, eventStore)
	server := NewTestServer(log, storage, eventStore, servingTenant,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{MessageEncoding: EncodingJSON}
	server.endpointRepo = endpoints
	persistence := &recordingAttachPersistence{}
	server.attachPersistence = persistence
	server.orgResolver = &fakeOrgResolver{tenantToOrg: make(map[int64]uuid.UUID), orgToTenant: make(map[uuid.UUID]int64)}
	if roaming != nil {
		server.roamingSvc = roaming
	}

	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "attach-session",
			BaseStationEUI:    TestBsEui01,
			ResolvedTenantID:  servingTenant,
			DbSessionID:       1,
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: true,
		},
		Conn: conn,
	}
	return &attachFixture{server: server, session: session, conn: conn, endpoints: endpoints, persistence: persistence, events: eventStore}
}

func (f *attachFixture) attach(t *testing.T) {
	t.Helper()
	f.attachWith(t, nil)
}

// attachWith sends a validly signed att, adding or overriding the given fields.
func (f *attachFixture) attachWith(t *testing.T, fields map[string]interface{}) {
	t.Helper()
	sign := generateAttachSignature(TestEpEui01, attachRoamingAttachCnt, testPresharedKey())
	signValues := make([]interface{}, len(sign))
	for i, b := range sign {
		signValues[i] = float64(b)
	}
	data := map[string]interface{}{
		"command":     mioty.CmdAttach,
		"opId":        int64(1),
		"epEui":       uint64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"attachCnt":   attachRoamingAttachCnt,
		"snr":         12.5,
		"rssi":        -75.5,
		"nonce":       []interface{}{float64(0), float64(0), float64(0), float64(0)},
		"sign":        signValues,
		"dualChan":    false,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}
	for key, value := range fields {
		data[key] = value
	}
	require.NoError(t, f.server.CallHandleMessage(f.session, &Message{Command: mioty.CmdAttach, OpId: 1, Data: data}, data))
}

// assertServedForOwner checks the attach was answered and recorded under the
// endpoint's owner, with the base station looked up under its own tenant.
func (f *attachFixture) assertServedForOwner(t *testing.T) {
	t.Helper()
	code, message := f.conn.LastError()
	require.Zero(t, code, "the attach is not refused: %s", message)
	assert.True(t, f.conn.SeenCommand(mioty.CmdAttachResponse), "a roaming endpoint's attach is answered")
	require.Len(t, f.persistence.persisted(), 1)
	record := f.persistence.persisted()[0]
	assert.Equal(t, attachRoamingOwnerTenant, record.TenantID, "the attach is persisted under the endpoint's owner")
	assert.Equal(t, attachRoamingServingTenant, record.BSLookupTenantID, "the base station is looked up under its own tenant")
	assert.Equal(t, []int64{attachRoamingOwnerTenant}, f.endpoints.radioMetricsTenants, "radio metrics land with the owner")
}

// With roaming disabled every base station of the RF mesh serves every
// endpoint under its owning tenant, as uplinks and detaches already do.
func TestAttachOfForeignEndpointWithRoamingDisabled(t *testing.T) {
	f := newAttachRoamingFixture(t, nil)
	f.attach(t)
	f.assertServedForOwner(t)
}

// With roaming enabled an allowed roaming attach is served for the owner and
// recorded as a roaming attach.
func TestAttachOfForeignEndpointWithRoamingAllowed(t *testing.T) {
	roaming := &attachRecordingRoaming{stubRoamingService: stubRoamingService{isRoaming: true, owner: attachRoamingOwnerTenant}}
	f := newAttachRoamingFixture(t, roaming)
	f.attach(t)
	f.assertServedForOwner(t)
	assert.Equal(t, 1, roaming.attaches, "the roaming attach is recorded")
}

// With roaming enabled a roaming attach without an agreement is refused.
func TestAttachOfForeignEndpointWithRoamingRefused(t *testing.T) {
	f := newAttachRoamingFixture(t, &attachRecordingRoaming{stubRoamingService: stubRoamingService{err: errRoamingProbe}})
	f.attach(t)

	code, message := f.conn.LastError()
	assert.Equal(t, POSIX_EPERM, code)
	assert.Equal(t, ResolveErrorMessage(errRoamingNotAllowed), message)
	assert.Empty(t, f.persistence.persisted(), "a refused roaming attach is not persisted")
}

// An endpoint the directory cannot be asked about fails the attach closed
// instead of treating it as unknown.
func TestAttachFailsClosedWhenOwnerLookupFails(t *testing.T) {
	f := newAttachRoamingFixture(t, nil)
	f.endpoints.lookupErr = errAttachRoamingLookup
	f.attach(t)

	code, message := f.conn.LastError()
	assert.Equal(t, POSIX_EIO, code)
	assert.Equal(t, ResolveErrorMessage(errAttachOwnerLookupFailed), message)
	assert.Empty(t, f.persistence.persisted())
}

// The completion of a roaming attach is recorded for the endpoint's owner,
// never for the tenant of the base station that heard it.
func TestAttachCompletionOfForeignEndpointIsRecordedForOwner(t *testing.T) {
	f := newAttachRoamingFixture(t, nil)
	f.attach(t)
	f.assertServedForOwner(t)

	complete := map[string]interface{}{"command": mioty.CmdAttachComplete, "opId": int64(1)}
	require.NoError(t, f.server.CallHandleMessage(f.session, &Message{Command: mioty.CmdAttachComplete, OpId: 1, Data: complete}, complete))

	var attached *models.SystemEvent
	for _, event := range f.events.recorded() {
		if event.EventType == models.EventTypeEndpointAttached {
			attached = event
		}
	}
	require.NotNil(t, attached, "the completed attach is recorded")
	assert.Equal(t, strconv.FormatInt(attachRoamingOwnerTenant, 10), attached.TenantID, "the owner sees its endpoint attached")
}

func (r *recordingEventStore) recorded() []*models.SystemEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*models.SystemEvent(nil), r.created...)
}
