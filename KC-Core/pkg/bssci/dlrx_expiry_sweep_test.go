package bssci

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expiryDLRXRepo captures the cutoff passed to the expiry sweep.
type expiryDLRXRepo struct {
	interfaces.DLRXStatusRepository
	cutoff    time.Time
	calls     int
	expired   int64
	expireErr error
}

func (r *expiryDLRXRepo) ExpireDLRXStatusQuery(_ context.Context, cutoff time.Time) (int64, error) {
	r.calls++
	r.cutoff = cutoff
	return r.expired, r.expireErr
}

// expiryStorage wires the capturing repository into the stub storage.
type expiryStorage struct {
	*stubStorage
	dlrx *expiryDLRXRepo
}

func (s *expiryStorage) DLRXStatus() interfaces.DLRXStatusRepository { return s.dlrx }

func newExpiryFixture(t *testing.T, queryTimeout time.Duration) (*Server, *expiryDLRXRepo) {
	t.Helper()
	log := newRecordingLogger()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(log, nil)
	dlrx := &expiryDLRXRepo{}
	storage := &expiryStorage{stubStorage: newStubStorage(), dlrx: dlrx}
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{MessageEncoding: EncodingJSON, DLRXQueryTimeout: queryTimeout}
	return server, dlrx
}

// TestSweepExpiredDLRXQueries_CutoffArithmetic: the sweep expires queries
// older than the configured timeout relative to the supplied clock.
func TestSweepExpiredDLRXQueries_CutoffArithmetic(t *testing.T) {
	server, dlrx := newExpiryFixture(t, 5*time.Minute)

	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	server.sweepExpiredDLRXQueries(now)

	require.Equal(t, 1, dlrx.calls)
	assert.Equal(t, now.Add(-5*time.Minute), dlrx.cutoff,
		"cutoff is the sweep clock minus the configured dlrx query timeout")
}

// TestSweepExpiredDLRXQueries_DefaultTimeout: an unset timeout falls back to
// the package default.
func TestSweepExpiredDLRXQueries_DefaultTimeout(t *testing.T) {
	server, dlrx := newExpiryFixture(t, 0)

	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	server.sweepExpiredDLRXQueries(now)

	require.Equal(t, 1, dlrx.calls)
	assert.Equal(t, now.Add(-defaultDLRXQueryTimeout), dlrx.cutoff)
}

// TestSweepExpiredDLRXQueries_StoreFailureIsNonFatal: a store failure is
// logged and the sweep returns without panicking, leaving the next tick to
// retry.
func TestSweepExpiredDLRXQueries_StoreFailureIsNonFatal(t *testing.T) {
	server, dlrx := newExpiryFixture(t, time.Minute)
	dlrx.expireErr = errors.New("db down")

	server.sweepExpiredDLRXQueries(time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC))

	require.Equal(t, 1, dlrx.calls)
}
