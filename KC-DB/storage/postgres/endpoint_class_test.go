package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// Every writer stores the class its bidi flag implies, whatever class it was
// handed, and hands the stored class back on the endpoint it wrote.
func TestEndpointClassFollowsBidiOnEveryWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, updateStatusTestTenant, "TestTenant720")
	cleanupEndpointTestData(t, db, "ClassFollows%")
	defer cleanupEndpointTestData(t, db, "ClassFollows%")
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x22}
	renamed := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x23}

	storedClass := func(e models.EUI) string {
		t.Helper()
		stored, err := repo.GetByEUI(ctx, updateStatusTestTenant, e[:])
		require.NoError(t, err)
		return stored.EPClass
	}

	endpoint := &models.EndPoint{
		EUI: eui, Name: "ClassFollows-endpoint", TenantID: updateStatusTestTenant,
		Bidi: true, EPClass: mioty.EndpointClassUnidirectional,
		Tags:     make(map[string]string),
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}
	require.NoError(t, repo.Create(ctx, endpoint))
	assert.Equal(t, mioty.EndpointClassBidirectional, storedClass(eui), "create")
	assert.Equal(t, mioty.EndpointClassBidirectional, endpoint.EPClass, "create hands back the stored class")

	endpoint.Bidi, endpoint.EPClass = false, mioty.EndpointClassBidirectional
	require.NoError(t, repo.Update(ctx, endpoint))
	assert.Equal(t, mioty.EndpointClassUnidirectional, storedClass(eui), "update")
	assert.Equal(t, mioty.EndpointClassUnidirectional, endpoint.EPClass, "update hands back the stored class")

	endpoint.EUI, endpoint.Bidi, endpoint.EPClass = renamed, true, mioty.EndpointClassUnidirectional
	_, err := repo.UpdateWithEUI(ctx, updateStatusTestTenant, eui[:], endpoint)
	require.NoError(t, err)
	assert.Equal(t, mioty.EndpointClassBidirectional, storedClass(renamed), "update with a new EUI")
}

// A new endpoint stored with a status is stored with it and with the time of
// that decision, in one transaction; a create that fails stores neither.
func TestEndpointCreateWithStatusRecordsTheDecision(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, updateStatusTestTenant, "TestTenant720")
	cleanupEndpointTestData(t, db, "CreateWithStatus%")
	defer cleanupEndpointTestData(t, db, "CreateWithStatus%")
	decidedAt := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	repo := NewEndPointRepository(db, testsupport.TestCipher(), testutil.NewFakeClock(decidedAt), logger.Get())
	ctx := testutil.TestContext()
	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x24}
	newEndpoint := func() *models.EndPoint {
		return &models.EndPoint{
			EUI: eui, Name: "CreateWithStatus-endpoint", TenantID: updateStatusTestTenant, Bidi: true,
			Tags:     make(map[string]string),
			NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		}
	}

	created := newEndpoint()
	require.NoError(t, repo.CreateWithStatus(ctx, created, updateStatusTestAttached))
	assert.Equal(t, updateStatusTestAttached, created.EpStatus)

	stored, err := repo.GetByEUI(ctx, updateStatusTestTenant, eui[:])
	require.NoError(t, err)
	assert.Equal(t, updateStatusTestAttached, stored.EpStatus)
	decided, err := repo.GetByAttachmentChangedSince(ctx, updateStatusTestTenant, updateStatusTestAttached, nil)
	require.NoError(t, err)
	require.Len(t, decided, 1, "the attachment is recorded as a decision")
	assert.Equal(t, created.ID, decided[0].ID)

	err = repo.CreateWithStatus(ctx, newEndpoint(), updateStatusTestAttached)
	require.ErrorIs(t, err, kcerrors.ErrDuplicate)
	var rows int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM endpoints WHERE ep_eui = $1`, eui[:]).Scan(&rows))
	assert.Equal(t, 1, rows, "the failed create stored nothing")
}
