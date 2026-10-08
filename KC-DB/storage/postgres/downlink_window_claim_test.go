package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// TestDownlinkWindow_OneClaimPerTelegram pins radio §3.6.1: the window of a
// telegram is claimed once, by its owner only, and a released claim can be
// taken again.
func TestDownlinkWindow_OneClaimPerTelegram(t *testing.T) {
	_, db, _ := applicationQueueIDFixture(t)
	messages := &MessageRepository{db: db}
	telegram := uuid.NewString()
	_, err := db.Exec(`
		INSERT INTO messages (id, tenant_id, owner_tenant_id, command_type, op_id, ep_eui, bs_eui,
			rx_time, packet_cnt, snr, rssi, dl_open, response_exp, dl_ack, received_at)
		VALUES ($1, 321, 321, $2, 1, $3, $4, 1, 1, 1.0, -90.0, true, false, false, NOW())`,
		telegram, mioty.CmdULData, mioty.EUI64Bytes(0x70B3D59CD0000350), mioty.EUI64Bytes(releaseStation))
	require.NoError(t, err)

	foreign, err := messages.ClaimDownlinkWindow(t.Context(), 322, telegram)
	require.NoError(t, err)
	assert.False(t, foreign, "another tenant cannot claim the owner's window")

	first, err := messages.ClaimDownlinkWindow(t.Context(), 321, telegram)
	require.NoError(t, err)
	assert.True(t, first)
	second, err := messages.ClaimDownlinkWindow(t.Context(), 321, telegram)
	require.NoError(t, err)
	assert.False(t, second, "a claimed window is not claimed twice")

	require.NoError(t, messages.ReleaseDownlinkWindow(t.Context(), 321, telegram))
	again, err := messages.ClaimDownlinkWindow(t.Context(), 321, telegram)
	require.NoError(t, err)
	assert.True(t, again, "a released window can be claimed by the next reception")
}
