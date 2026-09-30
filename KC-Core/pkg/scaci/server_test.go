// Package scaci provides tests for SCACI Send* helper command validation.
//
// Coverage:
//   - SendPingResponse command mismatch detection (§3.4.2)
//   - SendRegisterResponse command mismatch detection (§3.6.2)
//   - SendDeregisterResponse command mismatch detection (§3.7.2)
//   - SendULDataComplete command mismatch detection (§3.8.3)
//   - SendULDataTransmitResponse CommandType mismatch detection (§3.9.2)
//   - SendDLDataQueueResponse CommandType mismatch detection (§3.10.2)
//   - SendDLDataRevokeResponse command mismatch detection (§3.11.2)
//   - SendDLDataResultComplete command mismatch detection (§3.12.3)
package scaci

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger creates a logger for testing that captures log output.
func testLogger() logger.Logger {
	return bsscitest.NewRecordingLogger()
}

// ============================================================================
// Send Helper Command Mismatch Tests (SCACI §2.5)
//
// These tests verify that Send* helpers reject messages with wrong command
// fields, preventing misuse of BaseMessage-only response types.
// ============================================================================

// TestSendPingResponseCommandMismatch verifies §3.4.2 command validation.
func TestSendPingResponseCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &PingResponse{}
	msg.Command = "wrongCommand" // Not CmdPingResponse
	msg.OpId = 1

	err := s.SendPingResponse(conn, session, msg)
	require.Error(t, err, "SendPingResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendRegisterResponseCommandMismatch verifies §3.6.2 command validation.
func TestSendRegisterResponseCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &RegisterResponse{}
	msg.Command = "wrongCommand" // Not CmdRegisterResponse
	msg.OpId = 2

	err := s.SendRegisterResponse(conn, session, msg)
	require.Error(t, err, "SendRegisterResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDeregisterResponseCommandMismatch verifies §3.7.2 command validation.
func TestSendDeregisterResponseCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DeregisterResponse{}
	msg.Command = "wrongCommand" // Not CmdDeregisterResponse
	msg.OpId = 3

	err := s.SendDeregisterResponse(conn, session, msg)
	require.Error(t, err, "SendDeregisterResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendULDataCompleteCommandMismatch verifies §3.8.3 command validation.
func TestSendULDataCompleteCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &ULDataComplete{}
	msg.Command = "wrongCommand" // Not CmdULDataComplete
	msg.OpId = 4

	err := s.SendULDataComplete(conn, session, msg)
	require.Error(t, err, "SendULDataComplete should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendULDataTransmitResponseCommandMismatch verifies §3.9.2 CommandType validation.
// Note: ULDataTransmitResponse is a mioty type alias using CommandType (not Command).
func TestSendULDataTransmitResponseCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &ULDataTransmitResponse{}
	msg.CommandType = "wrongCommand" // Not CmdULDataTransmitResponse (uses CommandType!)
	msg.OpId = 5

	err := s.SendULDataTransmitResponse(conn, session, msg)
	require.Error(t, err, "SendULDataTransmitResponse should reject wrong CommandType")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDLDataQueueResponseCommandMismatch verifies §3.10.2 CommandType validation.
// Note: DLDataQueueResponse is a mioty type alias using CommandType (not Command).
func TestSendDLDataQueueResponseCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DLDataQueueResponse{}
	msg.CommandType = "wrongCommand" // Not CmdDLDataQueueResponse (uses CommandType!)
	msg.OpId = 6

	err := s.SendDLDataQueueResponse(conn, session, msg)
	require.Error(t, err, "SendDLDataQueueResponse should reject wrong CommandType")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDLDataRevokeResponseCommandMismatch verifies §3.11.2 command validation.
func TestSendDLDataRevokeResponseCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DLDataRevokeResponse{}
	msg.Command = "wrongCommand" // Not CmdDLDataRevokeResponse
	msg.OpId = 7

	err := s.SendDLDataRevokeResponse(conn, session, msg)
	require.Error(t, err, "SendDLDataRevokeResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDLDataResultCompleteCommandMismatch verifies §3.12.3 command validation.
func TestSendDLDataResultCompleteCommandMismatch(t *testing.T) {
	s := &Server{clock: clock.SystemClock{}, logger: testLogger(), commands: mustTestCommandRegistry()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DLDataResultComplete{}
	msg.Command = "wrongCommand" // Not CmdDLDataResultComplete
	msg.OpId = -8                // Negative for SC-originated

	err := s.SendDLDataResultComplete(conn, session, msg)
	require.Error(t, err, "SendDLDataResultComplete should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// ============================================================================
// Constructor Validation Tests (Fail-Fast Pattern per BSSCI server.go:641-644)
//
// These tests verify that NewServer rejects nil dependencies at construction
// time rather than allowing runtime nil pointer panics.
//
// Each test provides valid values for parameters checked BEFORE the one being
// tested, and nil for parameters checked AFTER, to isolate the validation.
// ============================================================================

// newServerDeps returns a dependency set with the first n entries of the
// constructor's presence order populated, so each test names the first gap.
func newServerDeps(n int) Dependencies {
	deps := Dependencies{Clock: clock.SystemClock{}}
	fill := []func(*Dependencies){
		func(d *Dependencies) { d.Registry = newTestRegistry(nil, nil) },
		func(d *Dependencies) { d.Operations = &mockOperationRepoStub{} },
		func(d *Dependencies) { d.Handshake = &MockHandshakeService{} },
		func(d *Dependencies) { d.Endpoints = &MockEndpointService{} },
		func(d *Dependencies) { d.UL = &MockULService{} },
		func(d *Dependencies) { d.DL = &MockDLService{} },
	}
	for k := 0; k < n && k < len(fill); k++ {
		fill[k](&deps)
	}
	return deps
}

func TestNewServer_RequiredDependencies(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	cases := []struct {
		name    string
		cfg     *Config
		log     logger.Logger
		deps    Dependencies
		message string
	}{
		{"nil cfg", nil, testLogger(), newServerDeps(0), "cfg is required"},
		{"nil logger", cfg, nil, newServerDeps(0), "logger is required"},
		{"nil sessionRegistry", cfg, testLogger(), newServerDeps(0), "sessionRegistry is required"},
		{"nil operationRepo", cfg, testLogger(), newServerDeps(1), "operationRepo is required"},
		{"nil handshakeSvc", cfg, testLogger(), newServerDeps(2), "handshakeSvc is required"},
		{"nil endpointSvc", cfg, testLogger(), newServerDeps(3), "endpointSvc is required"},
		{"nil ulSvc", cfg, testLogger(), newServerDeps(4), "ulSvc is required"},
		{"nil dlSvc", cfg, testLogger(), newServerDeps(5), "dlSvc is required"},
		{"nil statusSvc", cfg, testLogger(), newServerDeps(6), "statusSvc is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewServer(tc.cfg, tc.log, tc.deps)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.message)
		})
	}
}

type stubOrgDirectory struct{}

func (stubOrgDirectory) GetDefaultOrgForTenant(_ context.Context, _ int64) (uuid.UUID, error) {
	return uuid.Nil, nil
}

type stubSnapshotSource struct{}

func (stubSnapshotSource) ConnectedSessionsSnapshot() []propagation.BaseStationSession { return nil }

type stubEndpointPropagator struct{}

func (stubEndpointPropagator) TriggerEndpointPropagate(_ context.Context, _ int64, _ []propagation.BaseStationSession) error {
	return nil
}

// newServerThroughPersistence builds a dependency set complete up to and
// including sessionPersistence for the trailing nil-check tests.
func newServerThroughPersistence(orgResolver OrganizationDirectory, snapshot SessionSnapshotSource, propagator EndpointPropagator, recorder ErrorRecorder, events SessionEventStore) (*Server, error) {
	return newServerWithConfig(&Config{ListenAddr: ":5001", PlatformTenantID: platformTenant}, orgResolver, snapshot, propagator, recorder, events)
}

// newServerWithConfig builds a server over cfg with a dependency set
// complete up to and including sessionPersistence.
func newServerWithConfig(cfg *Config, orgResolver OrganizationDirectory, snapshot SessionSnapshotSource, propagator EndpointPropagator, recorder ErrorRecorder, events SessionEventStore) (*Server, error) {
	return NewServer(cfg, testLogger(), Dependencies{
		Registry:     newTestRegistry(nil, nil),
		Operations:   &mockOperationRepoStub{},
		Handshake:    &MockHandshakeService{},
		Endpoints:    &MockEndpointService{},
		UL:           &MockULService{},
		DL:           &MockDLService{},
		Status:       &MockStatusService{},
		Validator:    &MockSessionValidator{},
		Recorder:     &MockOperationRecorder{},
		Persistence:  &MockSessionPersistence{},
		OrgDirectory: orgResolver,
		Snapshots:    snapshot,
		Propagation:  propagator,
		Errors:       recorder,
		Clock:        clock.SystemClock{},

		SessionEvents: events,
	})
}

// TestNewServer_NilOrgResolver verifies constructor rejects nil orgResolver.
func TestNewServer_NilOrgResolver(t *testing.T) {
	_, err := newServerThroughPersistence(nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil orgResolver")
	assert.Contains(t, err.Error(), "orgResolver is required")
}

// TestNewServer_NilSessionSnapshotProvider verifies constructor rejects nil
// sessionSnapshotProvider.
func TestNewServer_NilSessionSnapshotProvider(t *testing.T) {
	_, err := newServerThroughPersistence(stubOrgDirectory{}, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil sessionSnapshotProvider")
	assert.Contains(t, err.Error(), "sessionSnapshotProvider is required")
}

// TestNewServer_NilPropagationSvc verifies constructor rejects nil propagationSvc.
func TestNewServer_NilPropagationSvc(t *testing.T) {
	_, err := newServerThroughPersistence(stubOrgDirectory{}, stubSnapshotSource{}, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil propagationSvc")
	assert.Contains(t, err.Error(), "propagationSvc is required")
}

// TestNewServer_NilErrorRecorder verifies constructor rejects nil errorRecorder.
func TestNewServer_NilErrorRecorder(t *testing.T) {
	_, err := newServerThroughPersistence(stubOrgDirectory{}, stubSnapshotSource{}, stubEndpointPropagator{}, nil, nil)
	require.Error(t, err, "NewServer should reject nil errorRecorder")
	assert.Contains(t, err.Error(), "errorRecorder is required")
}
