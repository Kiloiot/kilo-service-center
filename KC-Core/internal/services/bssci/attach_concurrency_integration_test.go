package bssciservices

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
)

const (
	concurrentAttachTenant = int64(910)
	concurrentAttachEUI    = uint64(0x70B3D5677011150B)
	concurrentAttachCnt    = uint32(5)
)

// storageAttachRunner drives the attach transaction through the storage
// adapter, as the composition root does.
type storageAttachRunner struct {
	adapter *adapters.EndpointSessionTransactionAdapter
}

func (r storageAttachRunner) Run(ctx context.Context, fn func(EndpointSessionTx) error) error {
	return r.adapter.Run(ctx, func(tx adapters.EndpointSessionOps) error { return fn(tx) })
}

func seedConcurrentAttachEndpoint(t *testing.T, db *postgres.DB, provisionedAttachCnt *uint32) int64 {
	t.Helper()
	ctx := testutil.TestContext()
	_, err := db.Exec(ctx, `
		INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES ($1, 'Concurrent Attach Tenant', 'concurrent attach test', 'active', NOW(), NOW())`, concurrentAttachTenant)
	require.NoError(t, err)
	endpoint := &models.EndPoint{
		Name:          "concurrent-attach",
		TenantID:      concurrentAttachTenant,
		OwnerTenantID: concurrentAttachTenant,
		NwkSnKey:      make([]byte, 16),
		EPClass:       mioty.EndpointClassBidirectional,
		AttachCnt:     provisionedAttachCnt,
	}
	binary.BigEndian.PutUint64(endpoint.EUI[:], concurrentAttachEUI)
	require.NoError(t, postgres.NewRepositories(db).Endpoints.Create(ctx, endpoint))
	return endpoint.ID
}

// overTheAirAttachRecord is the attach transaction of an att with the given
// counter; like every over-the-air attach it records the nonce and signature.
func overTheAirAttachRecord(endpointID int64, attachCnt uint32) bssci.AttachSessionRecord {
	counter := int64(attachCnt)
	return bssci.AttachSessionRecord{
		TenantID:   concurrentAttachTenant,
		EndpointID: endpointID,
		EndpointUpdates: models.EndpointAttachmentStateParams{
			AttachCnt: &counter,
			Nonce:     []byte{1, 2, 3, 4},
			Sign:      []byte{5, 6, 7, 8},
		},
		EncryptedKey: make([]byte, 16),
		AttachCnt:    attachCnt,
		ShAddr:       0x150B,
	}
}

// Radio spec §3.6.5.3: an endpoint that never attached over the air accepts
// any attach counter, the 0 provisioning stored included; once that attach is
// recorded, its counter is consumed.
func TestFirstOverTheAirAttachMayReuseTheProvisionedCounter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	provisioned := uint32(0)
	endpointID := seedConcurrentAttachEndpoint(t, db, &provisioned)
	persistence := newTestAttachmentPersistence(t, storageAttachRunner{adapter: adapters.NewEndpointSessionTransactionAdapter(db)})

	require.NoError(t, persistence.PersistAttachSession(testutil.TestContext(), overTheAirAttachRecord(endpointID, provisioned)),
		"the first over-the-air attach may use the provisioned counter")
	assert.ErrorIs(t, persistence.PersistAttachSession(testutil.TestContext(), overTheAirAttachRecord(endpointID, provisioned)),
		bssci.ErrAttachCounterStale, "a recorded over-the-air attach consumes its counter")
}

// Radio spec §3.7.1.2: when two base stations deliver the same att at once,
// exactly one attach is served; the other finds its counter already consumed.
func TestConcurrentIdenticalAttachesServeExactlyOne(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	endpointID := seedConcurrentAttachEndpoint(t, db, nil)

	persistence := newTestAttachmentPersistence(t, storageAttachRunner{adapter: adapters.NewEndpointSessionTransactionAdapter(db)})
	record := overTheAirAttachRecord(endpointID, concurrentAttachCnt)

	const stations = 2
	results := make([]error, stations)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range stations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i] = persistence.PersistAttachSession(testutil.TestContext(), record)
		}()
	}
	close(start)
	wg.Wait()

	served, refused := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			served++
		case assert.ErrorIs(t, err, bssci.ErrAttachCounterStale):
			refused++
		}
	}
	assert.Equal(t, 1, served, "exactly one of the identical attaches is served")
	assert.Equal(t, 1, refused, "the other finds its attach counter consumed")
}
