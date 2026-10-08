package bssci

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// unregisteredStationRegistry knows no base station and counts offline
// transitions, which fail for an unknown station exactly as the production
// store does.
type unregisteredStationRegistry struct {
	interopConnectionService
	offlineTransitions atomic.Int32
}

func (*unregisteredStationRegistry) GetBaseStationGlobal(context.Context, [8]byte) (*basestation.BaseStation, error) {
	return nil, storage.ErrNotFound
}

func (r *unregisteredStationRegistry) DisconnectBaseStationIfCurrent(context.Context, [8]byte, string) error {
	r.offlineTransitions.Add(1)
	return storage.ErrNotFound
}

// offlineCountingRegistry registers every base station and counts offline
// transitions.
type offlineCountingRegistry struct {
	interopConnectionService
	offlineTransitions atomic.Int32
}

func (r *offlineCountingRegistry) DisconnectBaseStationIfCurrent(context.Context, [8]byte, string) error {
	r.offlineTransitions.Add(1)
	return nil
}

// activateInteropSession completes con/conCmp and waits for the first status
// request, which proves the connection is the live one.
func activateInteropSession(t *testing.T, h *interopHarness) {
	t.Helper()
	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	require.Equal(t, mioty.CmdStatus, frameCommand(h.readFrame()))
}

// The live connection took its base station online, so losing it takes the
// station offline exactly once.
func TestLiveConnectionTeardownTakesStationOffline(t *testing.T) {
	registry := &offlineCountingRegistry{}
	h := startInteropServerWithRegistry(t, EncodingJSON, registry, func(cfg *Config) {
		cfg.StatusRequestInitialDelay = interopShortTimeout
		cfg.StatusRequestInterval = interopIODeadline
	})
	activateInteropSession(t, h)

	require.NoError(t, h.conn.Close())
	<-h.done

	assert.Equal(t, int32(1), registry.offlineTransitions.Load())
}

// A rejected connect never made the base station online, so its teardown
// has no offline status to record.
func TestRejectedConnectRecordsNoOfflineTransition(t *testing.T) {
	registry := &unregisteredStationRegistry{}
	h := startInteropServerWithRegistry(t, EncodingJSON, registry, nil)

	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	rejection := h.readFrame()
	require.Equal(t, mioty.CmdError, frameCommand(rejection), "an unregistered station is rejected")

	h.writeFrame(map[string]interface{}{"command": mioty.CmdErrorAck, "opId": int64(0)})
	h.expectClosed()
	<-h.done

	assert.Zero(t, registry.offlineTransitions.Load(), "teardown must not take an unregistered station offline")
}
