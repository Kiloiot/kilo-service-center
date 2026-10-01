package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	registeredAtTenant      = int64(710)
	registeredAtOtherTenant = int64(711)
)

// otherMasterKey opens no key the test cipher sealed.
var otherMasterKey = []byte("another-kilocenter-master-key-32")

// The registration time is read without the endpoint's session keys: a
// repository whose cipher cannot open them still answers it.
func TestRegisteredAt_ReadsTheRegistrationWithoutTheSessionKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	createTestTenant(t, db, registeredAtTenant, "RegisteredAtTenant")
	createTestTenant(t, db, registeredAtOtherTenant, "RegisteredAtOtherTenant")
	ctx := testutil.TestContext()
	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x10}
	require.NoError(t, NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get()).Create(ctx, &models.EndPoint{
		EUI: eui, Name: "RegisteredAt-EP", TenantID: registeredAtTenant, EPClass: "A",
		NwkSnKey: make([]byte, 16), AppKey: make([]byte, 16), Tags: map[string]string{},
	}))
	var createdAt time.Time
	require.NoError(t, db.Get(&createdAt, `SELECT created_at FROM endpoints WHERE ep_eui = $1`, eui[:]))

	otherCipher, err := keycrypto.NewCipher(otherMasterKey)
	require.NoError(t, err)
	repo := NewEndPointRepository(db, otherCipher, clock.SystemClock{}, logger.Get())
	_, err = repo.GetByEUI(ctx, registeredAtTenant, eui[:])
	require.Error(t, err, "the full lookup opens the session keys")

	registeredAt, err := repo.RegisteredAt(ctx, registeredAtTenant, eui[:])
	require.NoError(t, err)
	assert.True(t, createdAt.Equal(registeredAt))

	_, err = repo.RegisteredAt(ctx, registeredAtOtherTenant, eui[:])
	assert.ErrorIs(t, err, storage.ErrNotFound, "another tenant has no registration of the EUI")
}
