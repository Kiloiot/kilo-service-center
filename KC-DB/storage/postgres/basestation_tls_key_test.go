package postgres

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	takeKeyTestOwner   = int64(471)
	takeKeyTestForeign = int64(472)
	takeKeyTestBsEUI   = uint64(0x70B3D59CD0000471)
	takeKeyTestBsName  = "take-tls-key"
	takeKeyTestSealed  = "sealed-private-key"
	columnTLSKey       = "tls_key"

	takeKeyTestConcurrency = 8
	takeKeyTestLockWait    = 3 * time.Second
)

var errTakeKeyTestRefused = errors.New("key refused")

// seedStationKey stores takeKeyTestSealed for a station of takeKeyTestOwner
// in a fresh database, with takeKeyTestForeign as a second tenant.
func seedStationKey(t *testing.T) (*BaseStationRepository, models.EUI) {
	t.Helper()
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, db, takeKeyTestOwner, "TakeKeyOwner")
	createTestTenant(t, db, takeKeyTestForeign, "TakeKeyForeign")
	repo := NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], eui64Bytes(takeKeyTestBsEUI))
	station := &models.BaseStation{
		EUI:              eui,
		TenantID:         takeKeyTestOwner,
		Name:             takeKeyTestBsName,
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}
	require.NoError(t, repo.Create(ctx, station))
	stored, err := repo.GetByEUI(ctx, takeKeyTestOwner, eui[:])
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, takeKeyTestOwner, stored.ID, map[string]interface{}{columnTLSKey: takeKeyTestSealed}))
	return repo, eui
}

// TestBaseStationTakeTLSKey_DoesNotWaitOnAKeyBeingTaken: a take while another
// holds the key is refused at once, so a taker that writes elsewhere while it
// holds the key is never starved of a connection by takers waiting on it.
func TestBaseStationTakeTLSKey_DoesNotWaitOnAKeyBeingTaken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	repo, eui := seedStationKey(t)
	ctx := testutil.TestContext()
	waitCtx, cancel := context.WithTimeout(ctx, takeKeyTestLockWait)
	t.Cleanup(cancel)

	var inner error
	var waited time.Duration
	err := repo.TakeTLSKey(ctx, takeKeyTestOwner, eui[:], func(string) error {
		start := time.Now()
		inner = repo.TakeTLSKey(waitCtx, takeKeyTestOwner, eui[:], func(string) error { return nil })
		waited = time.Since(start)
		return nil
	})

	require.NoError(t, err, "the first take hands the key out")
	require.Error(t, inner, "a take while the key is held gets nothing")
	assert.NotErrorIs(t, inner, context.DeadlineExceeded, "it is refused, not left waiting")
	assert.Less(t, waited, takeKeyTestLockWait/2)
}

// TestBaseStationTakeTLSKey_RecordsAnEventOfTheStationWhileTakingTheKey: the
// audit event naming the station is written on another connection while the
// key is held, so the key's row lock must not block rows that refer to it.
func TestBaseStationTakeTLSKey_RecordsAnEventOfTheStationWhileTakingTheKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	repo, eui := seedStationKey(t)
	ctx := testutil.TestContext()
	station, err := repo.GetByEUI(ctx, takeKeyTestOwner, eui[:])
	require.NoError(t, err)
	events := NewSystemEventStore(repo.db.DB, clock.SystemClock{}, logger.Get())
	recordCtx, cancel := context.WithTimeout(ctx, takeKeyTestLockWait)
	t.Cleanup(cancel)

	err = repo.TakeTLSKey(ctx, takeKeyTestOwner, eui[:], func(string) error {
		return events.CreateEvent(recordCtx, &models.SystemEvent{
			TenantID:      strconv.FormatInt(takeKeyTestOwner, 10),
			EventType:     models.EventTypeCertificatePrivateKeyDownloaded,
			Category:      models.EventCategoryAudit,
			Severity:      models.EventSeverityInfo,
			SourceType:    models.SourceTypeBaseStation,
			SourceName:    takeKeyTestBsName,
			BasestationID: &station.ID,
			Title:         models.EventTitleCertificatePrivateKeyDownloaded,
		})
	})

	require.NoError(t, err, "the event is recorded while the key is held, not blocked by its lock")
}

// TestBaseStationTakeTLSKey_HandsTheKeyOutOnce: another tenant takes nothing,
// a key the caller refuses stays, concurrent takes hand it out once, and the
// key is gone afterwards.
func TestBaseStationTakeTLSKey_HandsTheKeyOutOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	repo, eui := seedStationKey(t)
	ctx := testutil.TestContext()

	accept := func(got *[]string, mu *sync.Mutex) func(string) error {
		return func(sealed string) error {
			mu.Lock()
			defer mu.Unlock()
			*got = append(*got, sealed)
			return nil
		}
	}
	var mu sync.Mutex
	var taken []string

	err := repo.TakeTLSKey(ctx, takeKeyTestForeign, eui[:], accept(&taken, &mu))
	require.ErrorIs(t, err, storage.ErrNotFound, "another tenant takes nothing")

	err = repo.TakeTLSKey(ctx, takeKeyTestOwner, eui[:], func(string) error { return errTakeKeyTestRefused })
	require.ErrorIs(t, err, errTakeKeyTestRefused)
	kept, err := repo.GetByEUI(ctx, takeKeyTestOwner, eui[:])
	require.NoError(t, err)
	require.NotNil(t, kept.TLSKey, "a key the caller cannot open stays stored")

	var wg sync.WaitGroup
	for range takeKeyTestConcurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = repo.TakeTLSKey(ctx, takeKeyTestOwner, eui[:], accept(&taken, &mu))
		}()
	}
	wg.Wait()
	assert.Equal(t, []string{takeKeyTestSealed}, taken, "concurrent takes hand the key out exactly once")

	after, err := repo.GetByEUI(ctx, takeKeyTestOwner, eui[:])
	require.NoError(t, err)
	assert.Nil(t, after.TLSKey)
}
