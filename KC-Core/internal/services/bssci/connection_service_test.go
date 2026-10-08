package bssciservices

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const registryStationEUI uint64 = 0x70B3D59CD00009E6

type recordingConnectionManager struct {
	statuses []*basestation.ConnectionStatus
}

func (m *recordingConnectionManager) GetBaseStationGlobal(context.Context, [8]byte) (*basestation.BaseStation, error) {
	return nil, nil
}

func (m *recordingConnectionManager) UpdateLastSeen(context.Context, [8]byte) error { return nil }

func (m *recordingConnectionManager) UpdateConnectionStatus(_ context.Context, _ [8]byte, status *basestation.ConnectionStatus) error {
	m.statuses = append(m.statuses, status)
	return nil
}

func (m *recordingConnectionManager) DisconnectBaseStationIfCurrent(context.Context, [8]byte, string) error {
	return nil
}

func TestRegisterConnection_RecordsTheCompletedHandshake(t *testing.T) {
	manager := &recordingConnectionManager{}
	registry := NewConnectionRegistry(manager, logger.NewNop())
	session := &bssci.Session{}
	session.ID = "connection-7"
	session.BaseStationEUI = registryStationEUI
	before := time.Now()

	require.NoError(t, registry.RegisterConnection(testutil.TestContext(), session, nil))

	require.Len(t, manager.statuses, 1)
	status := manager.statuses[0]
	assert.True(t, status.IsOnline)
	assert.Equal(t, "connection-7", status.SessionID)
	assert.False(t, status.SessionStartedAt.Before(before), "the handshake time is the activation")
	assert.Equal(t, status.LastSeen, status.SessionStartedAt)
}
