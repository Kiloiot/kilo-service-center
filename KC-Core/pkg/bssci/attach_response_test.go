package bssci

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const attachResponseTenant = int64(1)

// newOwnAttachFixture serves an attach of an endpoint the base station's own
// tenant owns and that has no short address yet.
func newOwnAttachFixture(t *testing.T) *attachFixture {
	t.Helper()
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], TestEpEui01)
	return newAttachFixture(t, &models.EndPoint{
		ID:       3101,
		EUI:      eui,
		TenantID: attachResponseTenant,
		NwkSnKey: testPresharedKey(),
	}, attachResponseTenant, nil)
}

func (f *attachFixture) attachResponse(t *testing.T) map[string]interface{} {
	t.Helper()
	code, message := f.conn.LastError()
	require.Zero(t, code, "the attach is not refused: %s", message)
	response := f.conn.LastMessage(mioty.CmdAttachResponse)
	require.NotNil(t, response, "the attach is answered with attRsp")
	return response
}

// BSSCI §3.6.2: attRsp carries shAddr only when the base station did not
// assign one in the att.
func TestAttachResponseOmitsShortAddressTheBaseStationAssigned(t *testing.T) {
	const stationShAddr = 0x1234
	f := newOwnAttachFixture(t)

	f.attachWith(t, map[string]interface{}{"shAddr": float64(stationShAddr)})

	response := f.attachResponse(t)
	assert.NotContains(t, response, "shAddr", "a station-assigned short address is not echoed")
	assert.Contains(t, response, wireFieldNwkSnKey, "the session key is always sent")
	require.Len(t, f.persistence.persisted(), 1)
	assert.Equal(t, uint16(stationShAddr), f.persistence.persisted()[0].ShAddr, "the station's short address is recorded")
}

func TestAttachResponseCarriesTheServiceCenterAssignedShortAddress(t *testing.T) {
	f := newOwnAttachFixture(t)

	f.attach(t)

	response := f.attachResponse(t)
	require.Contains(t, response, "shAddr", "the service center assigns the short address")
	assert.InDelta(t, float64(TestEpEui01&0xFFFF), response["shAddr"], 0)
	assert.Contains(t, response, wireFieldNwkSnKey)
}

// staleCounterPersistence reports that a concurrent attach consumed the counter.
type staleCounterPersistence struct {
	recordingAttachPersistence
}

func (*staleCounterPersistence) PersistAttachSession(context.Context, AttachSessionRecord) error {
	return fmt.Errorf("%w: stored 5", ErrAttachCounterStale)
}

// An att that loses the endpoint's row lock to an identical one is refused as
// a replay and never answered with attRsp.
func TestAttachThatLosesTheRowLockIsRefusedAsAReplay(t *testing.T) {
	f := newOwnAttachFixture(t)
	f.server.attachPersistence = &staleCounterPersistence{}

	f.attach(t)

	code, message := f.conn.LastError()
	assert.Equal(t, POSIX_EPROTO, code)
	assert.Equal(t, ResolveErrorMessage(errAttachCounterNotMonotonic), message)
	assert.False(t, f.conn.SeenCommand(mioty.CmdAttachResponse))
}
