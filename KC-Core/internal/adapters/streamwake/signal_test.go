package streamwake

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignal_NotificationsBeforeAWakeCloseItOnce(t *testing.T) {
	wake := NewSignal()
	woken := wake.Wake()

	wake.Notify()
	wake.Notify()

	_, open := <-woken
	assert.False(t, open, "the wake taken before the notifications is closed")
	select {
	case <-wake.Wake():
		t.Fatal("a wake taken after the notifications waits for the next one")
	default:
	}
}
