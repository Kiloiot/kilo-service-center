package postgres

import (
	"context"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	uplinkOpTenant     = int64(510)
	uplinkOpMessage    = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
	uplinkOpOther      = "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e"
	uplinkOpFirst      = int64(-1)
	uplinkOpRetry      = int64(-2)
	uplinkOpContenders = 8
)

func uplinkOperationRequest(sessionID, opID int64) *models.SCACIOperationRequest {
	return &models.SCACIOperationRequest{
		SessionID: sessionID, TenantID: uplinkOpTenant, OpId: opID, Command: mioty.CmdULData,
		Direction: string(models.OperationDirectionOutbound), RequestData: map[string]interface{}{"packetCnt": 7},
	}
}

func uplinkOperationRows(t *testing.T, db *sqlx.DB, sessionID int64) int {
	t.Helper()
	var rows int
	require.NoError(t, db.Get(&rows, `SELECT count(*) FROM scaci_operation_log WHERE session_id = $1 AND command = 'ulData'`, sessionID))
	return rows
}

// secondApplicationCenterSession opens a session of another Application
// Center of the tenant.
func secondApplicationCenterSession(t *testing.T, db *sqlx.DB, tenantID int64) int64 {
	t.Helper()
	session, err := NewSCACISessionRepository(db, clock.SystemClock{}, logger.Get()).CreateSession(testutil.TestContext(), &models.SCACISessionCreateRequest{
		TenantID: tenantID,
		AcEUI:    [8]byte{0x02, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		SnAcUUID: [16]byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF, 0xD0},
		SnScUUID: [16]byte{0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7, 0xD8, 0xD9, 0xDA, 0xDB, 0xDC, 0xDD, 0xDE, 0xDF, 0xE0},
	})
	require.NoError(t, err)
	return session.ID
}

func newUplinkOperationRepo(t *testing.T) (*SCACIOperationRepository, *sqlx.DB, int64) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSCACIOperationTestDB(t)
	createSCACIOperationTestTenant(t, db, uplinkOpTenant, "UplinkDeliveryIdentity")
	sessionID := createSCACIOperationTestSession(t, db, uplinkOpTenant)
	return NewSCACIOperationRepository(db, logger.Get(), clock.SystemClock{}), db, sessionID
}

// A session holds one ulData per stored uplink: a repeat returns the first
// operation with its opId and state, while another uplink or another session
// is recorded anew (SCACI §3.2).
func TestEnsureUplinkOperation_OneOperationPerSessionAndUplink(t *testing.T) {
	repo, db, sessionID := newUplinkOperationRepo(t)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), testContextTimeout)
	defer cancel()

	first, recorded, err := repo.EnsureUplinkOperation(ctx, uplinkOperationRequest(sessionID, uplinkOpFirst), uplinkOpMessage)
	require.NoError(t, err)
	require.True(t, recorded)
	require.NoError(t, repo.UpdateOperationState(ctx, sessionID, uplinkOpFirst, models.OperationStateCompleted, nil))

	repeat, recorded, err := repo.EnsureUplinkOperation(ctx, uplinkOperationRequest(sessionID, uplinkOpRetry), uplinkOpMessage)
	require.NoError(t, err)
	assert.False(t, recorded, "the session already has the uplink")
	assert.Equal(t, first.ID, repeat.ID)
	assert.Equal(t, uplinkOpFirst, repeat.OpId, "the operation keeps its opId")
	assert.Equal(t, string(models.OperationStateCompleted), repeat.State, "and its state")
	assert.Equal(t, uplinkOpMessage, repeat.RequestData[models.OperationRequestKeySourceMessageID])

	_, recorded, err = repo.EnsureUplinkOperation(ctx, uplinkOperationRequest(sessionID, uplinkOpRetry), uplinkOpOther)
	require.NoError(t, err)
	assert.True(t, recorded, "another uplink is another operation")

	otherSession := secondApplicationCenterSession(t, db, uplinkOpTenant)
	_, recorded, err = repo.EnsureUplinkOperation(ctx, uplinkOperationRequest(otherSession, uplinkOpFirst), uplinkOpMessage)
	require.NoError(t, err)
	assert.True(t, recorded, "another session gets its own operation")
	assert.Equal(t, 2, uplinkOperationRows(t, db, sessionID))
}

// Service centers racing for the same session and uplink record it once.
func TestEnsureUplinkOperation_ConcurrentAttemptsRecordOnce(t *testing.T) {
	repo, db, sessionID := newUplinkOperationRepo(t)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), testContextTimeout)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan int64, uplinkOpContenders)
	fresh := make(chan bool, uplinkOpContenders)
	for i := range uplinkOpContenders {
		wg.Add(1)
		go func(opID int64) {
			defer wg.Done()
			op, recorded, err := repo.EnsureUplinkOperation(ctx, uplinkOperationRequest(sessionID, opID), uplinkOpMessage)
			assert.NoError(t, err)
			if err == nil {
				results <- op.OpId
				fresh <- recorded
			}
		}(-int64(i + 1))
	}
	wg.Wait()
	close(results)
	close(fresh)

	var opIDs []int64
	for opID := range results {
		opIDs = append(opIDs, opID)
	}
	recordedNow := 0
	for recorded := range fresh {
		if recorded {
			recordedNow++
		}
	}
	require.Len(t, opIDs, uplinkOpContenders)
	assert.Equal(t, 1, recordedNow, "one attempt records the operation")
	for _, opID := range opIDs {
		assert.Equal(t, opIDs[0], opID, "every attempt returns the one operation")
	}
	assert.Equal(t, 1, uplinkOperationRows(t, db, sessionID))
}

func TestEnsureUplinkOperation_RefusesAnIncompleteRequest(t *testing.T) {
	repo, _, sessionID := newUplinkOperationRepo(t)
	notULData := uplinkOperationRequest(sessionID, uplinkOpFirst)
	notULData.Command = mioty.CmdULDataResponse
	inbound := uplinkOperationRequest(sessionID, uplinkOpFirst)
	inbound.Direction = string(models.OperationDirectionInbound)

	for name, tc := range map[string]struct {
		req       *models.SCACIOperationRequest
		messageID string
	}{
		"no request":      {req: nil, messageID: uplinkOpMessage},
		"no uplink":       {req: uplinkOperationRequest(sessionID, uplinkOpFirst), messageID: ""},
		"no session":      {req: uplinkOperationRequest(0, uplinkOpFirst), messageID: uplinkOpMessage},
		"another command": {req: notULData, messageID: uplinkOpMessage},
		"inbound":         {req: inbound, messageID: uplinkOpMessage},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := repo.EnsureUplinkOperation(testutil.TestContext(), tc.req, tc.messageID)
			assert.ErrorIs(t, err, storage.ErrInvalidInput)
		})
	}
}
