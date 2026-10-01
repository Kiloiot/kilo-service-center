package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Closing by EUI reports whether the station held a live session: only the
// connected station has one to close, and only once.
func TestCloseSessionByEUI_ReportsWhetherASessionWasClosed(t *testing.T) {
	server, session, _ := newPingServer(t, &stationEventLog{})

	assert.False(t, server.CloseSessionByEUI(testutil.TestContext(), uint64(TestBsEui02)), "a station without a session")
	assert.True(t, server.CloseSessionByEUI(testutil.TestContext(), session.BaseStationEUI), "the connected station")
	assert.Nil(t, server.GetSessionByEUI(session.BaseStationEUI), "its session is gone")
	assert.False(t, server.CloseSessionByEUI(testutil.TestContext(), session.BaseStationEUI), "nothing left to close")
}
