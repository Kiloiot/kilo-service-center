package bssci

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopPropagationService struct{}

func (noopPropagationService) TriggerEndpointPropagate(_ context.Context, _ int64, _ []propagation.BaseStationSession) error {
	return nil
}

func (noopPropagationService) ReconcileBaseStation(_ context.Context, _ propagation.BaseStationSession, _ *models.BaseStation) error {
	return nil
}

type noopDownlinkDispatcher struct{}

func (noopDownlinkDispatcher) DispatchIfAvailable(_ context.Context, _ int64, _ uuid.UUID, _ *Session, _ uint64, _, _ bool) (bool, error) {
	return false, nil
}

func (noopDownlinkDispatcher) DispatchQueue(_ context.Context, _ int64, _ uuid.UUID, _ *Session, _, _ uint64) (bool, error) {
	return false, nil
}

func newRuntimeDeps() RuntimeDependencies {
	return RuntimeDependencies{
		Propagation:        noopPropagationService{},
		DownlinkDispatcher: noopDownlinkDispatcher{},
	}
}

// TestConfigureRuntimeRejectsDoubleCall: the circular dependencies are wired
// exactly once by the composition root.
func TestConfigureRuntimeRejectsDoubleCall(t *testing.T) {
	server := newResumeReissueServer(t)
	require.NoError(t, server.ConfigureRuntime(newRuntimeDeps()))

	err := server.ConfigureRuntime(newRuntimeDeps())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already configured")
}

// TestConfigureRuntimeRejectedAfterStart: a serving instance cannot be
// reconfigured.
func TestConfigureRuntimeRejectedAfterStart(t *testing.T) {
	server := newResumeReissueServer(t)
	server.mu.Lock()
	server.started = true
	server.mu.Unlock()

	err := server.ConfigureRuntime(newRuntimeDeps())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "after Start")
}

// TestValidateRuntimeWiringReportsMissingDependency: an incompletely wired
// server is rejected at startup instead of failing as a nil dereference under
// traffic. The test harness intentionally omits several production-mandatory
// dependencies, so validation must fail and name one of them.
func TestValidateRuntimeWiringReportsMissingDependency(t *testing.T) {
	server := newResumeReissueServer(t)
	err := server.validateRuntimeWiring()
	require.Error(t, err, "a partially wired server must not pass wiring validation")
	assert.Contains(t, err.Error(), "is required")
}
