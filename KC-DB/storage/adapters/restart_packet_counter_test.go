package adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	restartTestTenant   = int64(3)
	restartTestEndpoint = int64(11)
)

// restartingEndpoints records the counter restarts one transaction runs.
type restartingEndpoints struct {
	interfaces.EndpointRepository
	restarts   []int64
	restartErr error
	calls      *[]string
}

func (r *restartingEndpoints) RestartPacketCounter(_ context.Context, _ int64, endpointID int64) error {
	*r.calls = append(*r.calls, "restart")
	r.restarts = append(r.restarts, endpointID)
	return r.restartErr
}

// forgettingClassifier records the classifier resets one transaction runs.
type forgettingClassifier struct {
	forgotten []int64
	calls     *[]string
}

func (c *forgettingClassifier) ForgetEndpoint(_ context.Context, _ int64, endpointID int64) error {
	*c.calls = append(*c.calls, "forget")
	c.forgotten = append(c.forgotten, endpointID)
	return nil
}

// restartTx is an attach transaction over the two recorders.
type restartTx struct {
	txLifecycle
	endpoints  *restartingEndpoints
	classifier *forgettingClassifier
}

func (t *restartTx) EndPoints() interfaces.EndpointRepository { return t.endpoints }

func (t *restartTx) UplinkClassifier() interfaces.UplinkClassifierReset { return t.classifier }

func (t *restartTx) EndPointSessions() interfaces.EndPointSessionRepository { return nil }

func newRestartTx(restartErr error) (*restartTx, *[]string) {
	calls := &[]string{}
	return &restartTx{
		endpoints:  &restartingEndpoints{restartErr: restartErr, calls: calls},
		classifier: &forgettingClassifier{calls: calls},
	}, calls
}

// Radio protocol §3.6.5.3: the counter restart of an over-the-air attach
// zeroes the endpoint's counters and forgets its uplink classifier rows in the
// attach transaction, the classifier through the uplink store's own statement.
func TestRestartPacketCounterForgetsTheClassifierInTheSameTransaction(t *testing.T) {
	tx, calls := newRestartTx(nil)

	require.NoError(t, EndpointSessionOps{h: tx}.RestartPacketCounter(testutil.TestContext(), restartTestTenant, restartTestEndpoint))

	assert.Equal(t, []string{"restart", "forget"}, *calls)
	assert.Equal(t, []int64{restartTestEndpoint}, tx.classifier.forgotten)
}

// A restart of an endpoint the tenant does not hold forgets nothing.
func TestRestartPacketCounterOfAnUnknownEndpointForgetsNothing(t *testing.T) {
	tx, calls := newRestartTx(storage.ErrNotFound)

	err := EndpointSessionOps{h: tx}.RestartPacketCounter(testutil.TestContext(), restartTestTenant, restartTestEndpoint)

	require.ErrorIs(t, err, storage.ErrNotFound)
	assert.Equal(t, []string{"restart"}, *calls)
}
