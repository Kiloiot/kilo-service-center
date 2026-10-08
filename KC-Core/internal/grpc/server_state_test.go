package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

const (
	testServerLoopback    = "127.0.0.1"
	testServerStopTimeout = time.Second
	testServerWait        = 2 * time.Second
	testServerPoll        = 10 * time.Millisecond
)

func TestServer_ListeningFollowsServing(t *testing.T) {
	srv, err := NewServer(Config{
		Log:        logger.NewNop(),
		Host:       testServerLoopback,
		Port:       ephemeralPort,
		RoleSource: fixedRoles(authz.AllRoles),
		HTTPConfig: HTTPServerConfig{WriteTimeout: testServerStopTimeout},
	})
	require.NoError(t, err)
	assert.False(t, srv.Listening(), "a built server is not serving yet")

	done := make(chan error, 1)
	go func() { done <- srv.Start() }()
	require.Eventually(t, srv.Listening, testServerWait, testServerPoll)

	srv.Stop()
	assert.False(t, srv.Listening(), "a stopped server no longer serves")
	require.NoError(t, <-done, "a deliberate Stop is a clean return from Start")
}
