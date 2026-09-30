package registrationscope

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

var registered = time.Date(2026, 9, 28, 12, 47, 16, 0, time.UTC)

type registrationsStub struct {
	at  time.Time
	err error
}

func (r registrationsStub) RegisteredAt(context.Context, int64, []byte) (time.Time, error) {
	return r.at, r.err
}

func start(t *testing.T, requested *time.Time) time.Time {
	t.Helper()
	got, ok, err := New(registrationsStub{at: registered}).Start(testutil.TestContext(), 1, []byte{1}, requested)
	require.NoError(t, err)
	require.True(t, ok)
	return *got
}

func TestStart_IsTheLaterOfTheRequestAndTheRegistration(t *testing.T) {
	earlier, later := registered.Add(-time.Hour), registered.Add(time.Hour)

	assert.Equal(t, registered, start(t, nil))
	assert.Equal(t, registered, start(t, &earlier))
	assert.Equal(t, later, start(t, &later))
}

func TestStart_OfAnEUITheTenantHasNotRegisteredIsNone(t *testing.T) {
	_, ok, err := New(registrationsStub{err: storage.ErrNotFound}).Start(testutil.TestContext(), 1, []byte{1}, nil)

	require.NoError(t, err)
	assert.False(t, ok)
}

func TestStart_WrapsAFailedRead(t *testing.T) {
	_, _, err := New(registrationsStub{err: errors.New("down")}).Start(testutil.TestContext(), 1, []byte{1}, nil)

	assert.ErrorIs(t, err, ErrReadRegistration)
}
