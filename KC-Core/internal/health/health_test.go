package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testVersion     = "test"
	testRowListener = "listener"
	testRowDisabled = "disabled"
	testRowFailing  = "failing"
	testDisabledMsg = "turned off"
)

type fakeListener struct{ listening bool }

func (f *fakeListener) Listening() bool { return f.listening }

type fixedChecker struct{ status Status }

func (c fixedChecker) Check(context.Context) *Check { return &Check{Status: c.status} }

func TestListenerChecker_ReadsTheListenerState(t *testing.T) {
	listener := &fakeListener{}
	checker := NewListenerChecker(listener)

	check := checker.Check(testutil.TestContext())
	assert.Equal(t, StatusUnhealthy, check.Status)
	assert.Equal(t, msgNotListening, check.Message)

	listener.listening = true
	check = checker.Check(testutil.TestContext())
	assert.Equal(t, StatusHealthy, check.Status)
	assert.Equal(t, msgListening, check.Message)
}

func TestMQTTChecker_ReadsTheConnectionState(t *testing.T) {
	assert.Equal(t, StatusHealthy, NewMQTTChecker(func() bool { return true }).Check(testutil.TestContext()).Status)
	assert.Equal(t, StatusUnhealthy, NewMQTTChecker(func() bool { return false }).Check(testutil.TestContext()).Status)
}

func TestCheckHealth_DisabledRowsNeverLowerTheAggregate(t *testing.T) {
	svc := NewService(logger.NewNop(), testVersion)
	svc.RegisterChecker(testRowListener, NewListenerChecker(&fakeListener{listening: true}))
	svc.RegisterChecker(testRowDisabled, NewDisabledChecker(testDisabledMsg))

	resp := svc.CheckHealth(testutil.TestContext())
	assert.Equal(t, StatusHealthy, resp.Status)
	assert.Equal(t, StatusDisabled, resp.Checks[testRowDisabled].Status)
	assert.Equal(t, testDisabledMsg, resp.Checks[testRowDisabled].Message)

	svc.RegisterChecker(testRowFailing, fixedChecker{status: StatusUnhealthy})
	assert.Equal(t, StatusUnhealthy, svc.CheckHealth(testutil.TestContext()).Status, "an unhealthy row still decides the aggregate")
}
