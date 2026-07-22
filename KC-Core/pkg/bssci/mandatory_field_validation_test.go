package bssci

import (
	"context"
	"crypto/aes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/aead/cmac"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNwkSnKey is a deterministic 16-byte key for test signature computation
var testNwkSnKey = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

// computeTestSignature generates a valid CMAC signature for test data.
// Uses same algorithm as ValidateAttachSignature (crypto.go:13-54).
// IV format: [EUI64 | 0xFF | 0x00 | attachCnt(24-bit) | 0xFF | 0xFF]
func computeTestSignature(epEUI uint64, attachCnt uint32, nwkSnKey []byte) []byte {
	iv := make([]byte, 15)
	binary.BigEndian.PutUint64(iv[0:8], epEUI)
	iv[8] = 0xFF
	iv[9] = 0x00
	maskedCnt := attachCnt & 0xFFFFFF
	iv[10] = byte(maskedCnt >> 16)
	iv[11] = byte(maskedCnt >> 8)
	iv[12] = byte(maskedCnt)
	iv[13] = 0xFF
	iv[14] = 0xFF

	block, _ := aes.NewCipher(nwkSnKey)
	mac, _ := cmac.New(block)
	mac.Write(iv)
	return mac.Sum(nil)[:4]
}

// buildTestEndpointForValidation creates a minimal endpoint for validation tests.
// Uses fixed values for determinism. Matches TestEpEui01 and tenant 1.
// Uses testNwkSnKey for CMAC signature validation per crypto.go:13-54.
func buildTestEndpointForValidation() *models.EndPoint {
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], TestEpEui01)
	storedCnt := uint32(50) // Lower than test attachCnt (100) to pass monotonic check
	return &models.EndPoint{
		ID:        1001,
		EUI:       eui,
		TenantID:  1,
		AttachCnt: &storedCnt,
		NwkSnKey:  testNwkSnKey, // Must match key used in computeTestSignature
	}
}

// findErrorMessage scans sent messages for command=="error" and returns the first match.
// Returns nil if no error message was sent.
func findErrorMessage(messages []map[string]interface{}) map[string]interface{} {
	for _, msg := range messages {
		if cmd, ok := msg["command"].(string); ok && cmd == "error" {
			return msg
		}
	}
	return nil
}

// findAttachResponse returns the first attRsp frame if sent.
func findAttachResponse(messages []map[string]interface{}) map[string]interface{} {
	for _, msg := range messages {
		if cmd, ok := msg["command"].(string); ok && cmd == mioty.CmdAttachResponse {
			return msg
		}
	}
	return nil
}

// buildValidAttachData creates a valid attach payload with optional field override.
// Uses float64 for numeric fields to match MessagePack decoding (getNumericField).
// Uses int64 for attachCnt to match server.go:1996-1998 parsing.
// Uses []interface{} for nonce/sign to exercise validateByteArray branches.
// Includes all mandatory fields per message_metadata.go:195-272.
// Computes valid CMAC signature using testNwkSnKey per crypto.go:13-54.
func buildValidAttachData(overrideField string, overrideValue interface{}) map[string]interface{} {
	attachCnt := int64(100) // Must be > storedCnt (50) to pass monotonic check

	// Compute valid CMAC signature for this attachCnt using testNwkSnKey
	validSign := computeTestSignature(TestEpEui01, uint32(attachCnt), testNwkSnKey)

	data := map[string]interface{}{
		"epEui":       float64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"snr":         float64(10.5),
		"rssi":        float64(-85.0),
		"attachCnt":   attachCnt,
		"nonce":       []interface{}{float64(1), float64(2), float64(3), float64(4)},
		"sign":        []interface{}{float64(validSign[0]), float64(validSign[1]), float64(validSign[2]), float64(validSign[3])},
		"dualChan":    true,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}
	if overrideField != "" {
		data[overrideField] = overrideValue
	}
	return data
}

// buildAttachDataWithCnt builds attach data with a computed signature for the specified attachCnt.
// Used by TestAttachCounterRange where attachCnt is the variable under test.
func buildAttachDataWithCnt(attachCnt int64) map[string]interface{} {
	// Compute valid CMAC signature for THIS specific attachCnt
	validSign := computeTestSignature(TestEpEui01, uint32(attachCnt)&0xFFFFFF, testNwkSnKey)

	return map[string]interface{}{
		"epEui":       float64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"snr":         float64(10.5),
		"rssi":        float64(-85.0),
		"attachCnt":   attachCnt,
		"nonce":       []interface{}{float64(1), float64(2), float64(3), float64(4)},
		"sign":        []interface{}{float64(validSign[0]), float64(validSign[1]), float64(validSign[2]), float64(validSign[3])},
		"dualChan":    true,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}
}

// attachTestStorage provides a transaction-capable storage stub so handleAttach can
// persist attach metadata and emit attRsp without DisableAttachPersistence shortcuts.
type attachTestStorage struct {
	pendingRepo interfaces.PendingOperationRepository
	tx          *attachTestTx
}

func newAttachTestStorage() *attachTestStorage {
	return &attachTestStorage{
		pendingRepo: &stubPendingOperationRepo{},
		tx: &attachTestTx{
			epRepo:      &attachTestEndpointRepo{},
			sessionRepo: &attachTestSessionRepo{},
		},
	}
}

func (s *attachTestStorage) EndPoints() interfaces.EndpointRepository          { return nil }
func (s *attachTestStorage) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (s *attachTestStorage) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (s *attachTestStorage) EndPointSessions() interfaces.EndPointSessionRepository       { return nil }
func (s *attachTestStorage) EndPointKeys() interfaces.EndPointKeyRepository               { return nil }
func (s *attachTestStorage) RoamingAgreements() interfaces.RoamingAgreementRepository     { return nil }
func (s *attachTestStorage) BaseStations() interfaces.BaseStationRepository               { return nil }
func (s *attachTestStorage) BaseStationSessions() interfaces.BaseStationSessionRepository { return nil }
func (s *attachTestStorage) DLRXStatus() interfaces.DLRXStatusRepository                  { return nil }
func (s *attachTestStorage) PendingOperations() interfaces.PendingOperationRepository {
	return s.pendingRepo
}
func (s *attachTestStorage) MIOTYMessages() interfaces.MIOTYMessageRepository   { return nil }
func (s *attachTestStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (s *attachTestStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *attachTestStorage) Users() interfaces.UserRepository                 { return nil }
func (s *attachTestStorage) APIKeys() interfaces.APIKeyRepository             { return nil }
func (s *attachTestStorage) Integrations() interfaces.IntegrationRepository   { return nil }
func (s *attachTestStorage) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (s *attachTestStorage) DeviceModels() interfaces.DeviceModelRepository   { return nil } // Blueprint catalog
func (s *attachTestStorage) Blueprints() interfaces.BlueprintRepository       { return nil } // Blueprint catalog
func (s *attachTestStorage) Organizations() interfaces.OrganizationRepository { return nil }
func (s *attachTestStorage) GetSqlxDB() *sqlx.DB                              { return nil }
func (s *attachTestStorage) SystemEvents() interfaces.SystemEventStore        { return nil }
func (s *attachTestStorage) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (s *attachTestStorage) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (s *attachTestStorage) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }
func (s *attachTestStorage) BeginTx(context.Context) (interfaces.Transaction, error) {
	return s.tx, nil
}
func (s *attachTestStorage) Ping(context.Context) error { return nil }
func (s *attachTestStorage) Close() error               { return nil }

// attachTestTx implements a minimal transaction used by handleAttach.
type attachTestTx struct {
	epRepo      *attachTestEndpointRepo
	sessionRepo *attachTestSessionRepo
}

func (t *attachTestTx) EndPoints() interfaces.EndpointRepository                         { return t.epRepo }
func (t *attachTestTx) DownlinkQueue() interfaces.DownlinkQueueRepository                { return nil }
func (t *attachTestTx) BaseStationReceptions() interfaces.BaseStationReceptionRepository { return nil }
func (t *attachTestTx) EndPointSessions() interfaces.EndPointSessionRepository           { return t.sessionRepo }
func (t *attachTestTx) EndPointKeys() interfaces.EndPointKeyRepository                   { return nil }
func (t *attachTestTx) RoamingAgreements() interfaces.RoamingAgreementRepository         { return nil }
func (t *attachTestTx) BaseStations() interfaces.BaseStationRepository                   { return nil }
func (t *attachTestTx) BaseStationSessions() interfaces.BaseStationSessionRepository     { return nil }
func (t *attachTestTx) PendingOperations() interfaces.PendingOperationRepository         { return nil }
func (t *attachTestTx) DLRXStatus() interfaces.DLRXStatusRepository                      { return nil }
func (t *attachTestTx) MIOTYMessages() interfaces.MIOTYMessageRepository                 { return nil }
func (t *attachTestTx) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository               { return nil }
func (t *attachTestTx) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (t *attachTestTx) Users() interfaces.UserRepository                 { return nil }
func (t *attachTestTx) APIKeys() interfaces.APIKeyRepository             { return nil }
func (t *attachTestTx) Integrations() interfaces.IntegrationRepository   { return nil }
func (t *attachTestTx) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (t *attachTestTx) DeviceModels() interfaces.DeviceModelRepository   { return nil } // Blueprint catalog
func (t *attachTestTx) Blueprints() interfaces.BlueprintRepository       { return nil } // Blueprint catalog
func (t *attachTestTx) Organizations() interfaces.OrganizationRepository { return nil }
func (t *attachTestTx) GetSqlxDB() *sqlx.DB                              { return nil }
func (t *attachTestTx) SystemEvents() interfaces.SystemEventStore        { return nil }
func (t *attachTestTx) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (t *attachTestTx) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (t *attachTestTx) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }
func (t *attachTestTx) Commit() error                                       { return nil }
func (t *attachTestTx) Rollback() error                                     { return nil }

type attachTestEndpointRepo struct {
	updates []map[string]interface{}
}

func (r *attachTestEndpointRepo) UpdateFields(_ context.Context, _ int64, _ int64, updates map[string]interface{}) error {
	r.updates = append(r.updates, updates)
	return nil
}

func (r *attachTestEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
	return nil
}

// unused interface methods
func (r *attachTestEndpointRepo) Create(context.Context, *models.EndPoint) error { return nil }
func (r *attachTestEndpointRepo) GetByID(context.Context, int64, int64) (*models.EndPoint, error) {
	return nil, nil
}
func (r *attachTestEndpointRepo) GetByEUI(context.Context, int64, []byte) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}
func (r *attachTestEndpointRepo) Get(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}
func (r *attachTestEndpointRepo) GetByTenant(context.Context, int64) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *attachTestEndpointRepo) CountByTenant(context.Context, int64) (int64, error) { return 0, nil }
func (r *attachTestEndpointRepo) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *attachTestEndpointRepo) Update(context.Context, *models.EndPoint) error { return nil }
func (r *attachTestEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	return nil
}
func (r *attachTestEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	return nil
}
func (r *attachTestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	return nil
}
func (r *attachTestEndpointRepo) StreamAllForPropagation(context.Context, int64, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *attachTestEndpointRepo) HasEndpointsSince(context.Context, time.Time) (bool, error) {
	return false, nil
}
func (r *attachTestEndpointRepo) GetEndpointWithKeysForDetachValidation(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}
func (r *attachTestEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil // No preference in tests
}
func (r *attachTestEndpointRepo) DeleteByTenant(context.Context, int64, []byte) error {
	return nil
}
func (r *attachTestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (r *attachTestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

type attachTestSessionRepo struct {
	_ map[string]*models.EndPointSession // future: track active sessions
}

func (r *attachTestSessionRepo) GetActive(_ context.Context, _ string) (*models.EndPointSession, error) {
	return nil, nil
}
func (r *attachTestSessionRepo) Create(_ context.Context, _ *models.EndPointSession) error {
	return nil
}
func (r *attachTestSessionRepo) Update(_ context.Context, _ *models.EndPointSession) error {
	return nil
}
func (r *attachTestSessionRepo) UpdateActivity(context.Context, string, bool) error { return nil }
func (r *attachTestSessionRepo) Terminate(context.Context, string) error            { return nil }
func (r *attachTestSessionRepo) Close(_ context.Context, _ string) error            { return nil }
func (r *attachTestSessionRepo) ListActive(context.Context, int64) ([]*models.EndPointSession, error) {
	return nil, nil
}
func (r *attachTestSessionRepo) ExpireOldSessions(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (r *attachTestSessionRepo) GetByID(context.Context, string) (*models.EndPointSession, error) {
	return nil, nil
}
func (r *attachTestSessionRepo) GetBySessionID(context.Context, string) (*models.EndPointSession, error) {
	return nil, nil
}
func (r *attachTestSessionRepo) GetByEndpointID(context.Context, int64) (*models.EndPointSession, error) {
	return nil, nil
}
func (r *attachTestSessionRepo) GetByEndPoint(context.Context, string, int, int) ([]*models.EndPointSession, error) {
	return nil, nil
}
func (r *attachTestSessionRepo) GetStats(context.Context, string) (*models.EndPointSessionStats, error) {
	return nil, nil
}
func (r *attachTestSessionRepo) UpdateSessionKey(context.Context, int64, string, []byte) error {
	return nil
}

// TestMandatoryFieldValidation verifies BSSCI §2.4-03 requirements

// validationMockConn is an alias for the shared TestConn from bssci testutil package
// This allows reusing the common mock connection implementation across test files
// Note: Field names are capitalized (SentMessages, FailWrites, Encoding)
type validationMockConn = testutil.TestConn

// TestStatusMandatoryFields verifies BSSCI §3.5.2 mandatory field validation
func TestStatusMandatoryFields(t *testing.T) {
	testLogger := logger.NewNop()

	tests := []struct {
		name         string
		data         map[string]interface{}
		missingField string
	}{
		{
			name: "missing code field",
			data: map[string]interface{}{
				"message": "operational",
				"time":    time.Now().UnixNano(),
			},
			missingField: "code",
		},
		{
			name: "missing message field",
			data: map[string]interface{}{
				"code": int64(0),
				"time": time.Now().UnixNano(),
			},
			missingField: "message",
		},
		{
			name: "missing time field",
			data: map[string]interface{}{
				"code":    int64(0),
				"message": "operational",
			},
			missingField: "time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)

			session := &Session{
				ID:                "test-session",
				BaseStationEUI:    TestBsEui01,
				Conn:              mockConn,
				Encoding:          "json",
				HandshakeComplete: true,
			}

			msg := &Message{
				Command: "statusRsp",
				OpId:    -1,
				Data:    tt.data,
			}

			_ = server.CallHandleMessage(session, msg, tt.data)

			// Handler sends error message (returns nil if send succeeds)
			require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
				"Should send error message for missing %s", tt.missingField)

			var errorMsg map[string]interface{}
			for _, sentMsg := range mockConn.SentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					errorMsg = sentMsg
					break
				}
			}

			require.NotNil(t, errorMsg, "Should send 'error' command message")

			code, hasPosix := errorMsg["code"]
			assert.True(t, hasPosix, "Error message must include code")
			assert.Equal(t, float64(71), code, "Missing field returns code 71")
		})
	}
}

// TestConnectMandatoryFields verifies BSSCI §5.3 mandatory field validation
func TestConnectMandatoryFields(t *testing.T) {
	testLogger := logger.NewNop()

	tests := []struct {
		name              string
		data              map[string]interface{}
		expectedPosixCode int
	}{
		{
			name: "missing bsEui",
			data: map[string]interface{}{
				"version": "1.0.0",
				"bidi":    true,
			},
			expectedPosixCode: 71, // POSIX_EPROTO - missing mandatory field is protocol violation per BSSCI §2.4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)

			server.SetConfig(&Config{
				ServiceCenterEUI: TestScEui01,
				Vendor:           "test-vendor",
				Model:            "test-model",
				Name:             "test-sc",
				SoftwareVersion:  "1.0.0",
			})

			session := &Session{
				ID:       "test-session",
				Conn:     mockConn,
				Encoding: "json",
			}

			msg := &Message{
				Command: "con",
				OpId:    0,
				Data:    tt.data,
			}

			_ = server.CallHandleConnect(session, msg, tt.data)

			// Verify error message sent
			var errorMsg map[string]interface{}
			for _, sentMsg := range mockConn.SentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					errorMsg = sentMsg
					break
				}
			}

			require.NotNil(t, errorMsg, "Should send error for missing mandatory field")

			code, hasCode := errorMsg["code"]
			assert.True(t, hasCode, "Error must include code")
			assert.Equal(t, float64(tt.expectedPosixCode), code, "Missing mandatory field returns expected POSIX code")
		})
	}
}

// TestRollbackOnSendFailure verifies LastScOpId restored on send failure per BSSCI §3.2
func TestRollbackOnSendFailure(t *testing.T) {
	testLogger := logger.NewNop()
	mockConn := &validationMockConn{}
	mockConn.Reset()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver, mockStorage :=
		CreateTestServices(testLogger, nil)

	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)

	session := &Session{
		ID:                "test-session",
		BaseStationEUI:    TestBsEui01,
		Conn:              mockConn,
		Encoding:          "json",
		HandshakeComplete: true,
		LastScOpId:        -5,
	}

	server.RegisterSession(session)
	initialOpId := session.LastScOpId
	mockConn.FailWrites = true

	_, err := server.SendStatusRequest(session)

	require.Error(t, err, "SendStatusRequest should fail when write fails")
	assert.Equal(t, initialOpId, session.LastScOpId,
		"LastScOpId should be restored on send failure")
}

// TestAttachMandatoryFields verifies BSSCI §3.6.1 mandatory field validation for attach handler
func TestAttachMandatoryFields(t *testing.T) {
	testLogger := logger.NewNop()

	// Test cases covering 7 mandatory fields + 2 conditional length validations
	tests := []struct {
		name              string
		data              map[string]interface{}
		expectedErrorCode float64 // 71 = POSIX_EPROTO
		missingField      string
	}{
		{
			name: "missing epEui",
			data: map[string]interface{}{
				"rxTime":    time.Now().UnixNano(),
				"attachCnt": int64(1),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"nonce":     []interface{}{float64(0x01), float64(0x02), float64(0x03), float64(0x04)},
				"sign":      []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			},
			expectedErrorCode: 71,
			missingField:      "epEui",
		},
		{
			name: "missing rxTime",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"attachCnt": int64(1),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"nonce":     []interface{}{float64(0x01), float64(0x02), float64(0x03), float64(0x04)},
				"sign":      []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			},
			expectedErrorCode: 71,
			missingField:      "rxTime",
		},
		{
			name: "missing attachCnt",
			data: map[string]interface{}{
				"epEui":  int64(TestEpEui01),
				"rxTime": time.Now().UnixNano(),
				"snr":    float64(10.5),
				"rssi":   float64(-80.0),
				"nonce":  []interface{}{float64(0x01), float64(0x02), float64(0x03), float64(0x04)},
				"sign":   []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			},
			expectedErrorCode: 71,
			missingField:      "attachCnt",
		},
		{
			name: "invalid snr type",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"attachCnt": int64(1),
				"snr":       "not-a-number", // Invalid type
				"rssi":      float64(-80.0),
				"nonce":     []interface{}{float64(0x01), float64(0x02), float64(0x03), float64(0x04)},
				"sign":      []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			},
			expectedErrorCode: 71,
			missingField:      "snr",
		},
		{
			name: "invalid rssi type",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"attachCnt": int64(1),
				"snr":       float64(10.5),
				"rssi":      "not-a-number", // Invalid type
				"nonce":     []interface{}{float64(0x01), float64(0x02), float64(0x03), float64(0x04)},
				"sign":      []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			},
			expectedErrorCode: 71,
			missingField:      "rssi",
		},
		{
			name: "missing nonce",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"attachCnt": int64(1),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			},
			expectedErrorCode: 71,
			missingField:      "nonce",
		},
		{
			name: "nonce wrong length (3 bytes)",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"attachCnt": int64(1),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"nonce":     []interface{}{float64(0x01), float64(0x02), float64(0x03)}, // Only 3 bytes
				"sign":      []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			},
			expectedErrorCode: 71,
			missingField:      "nonce",
		},
		{
			name: "missing sign",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"attachCnt": int64(1),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"nonce":     []interface{}{float64(0x01), float64(0x02), float64(0x03), float64(0x04)},
			},
			expectedErrorCode: 71,
			missingField:      "sign",
		},
		{
			name: "sign wrong length (3 bytes)",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"attachCnt": int64(1),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"nonce":     []interface{}{float64(0x01), float64(0x02), float64(0x03), float64(0x04)},
				"sign":      []interface{}{float64(0xAA), float64(0xBB), float64(0xCC)}, // Only 3 bytes
			},
			expectedErrorCode: 71,
			missingField:      "sign",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)

			session := &Session{
				ID:                "test-session",
				BaseStationEUI:    TestBsEui01,
				Conn:              mockConn,
				Encoding:          "json",
				HandshakeComplete: true,
			}

			msg := &Message{
				Command: "att",
				OpId:    1,
				Data:    tt.data,
			}

			_ = server.CallHandleMessage(session, msg, tt.data)

			// Verify error message sent
			require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
				"Should send error for missing/invalid %s", tt.missingField)

			var errorMsg map[string]interface{}
			for _, sentMsg := range mockConn.SentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					errorMsg = sentMsg
					break
				}
			}

			require.NotNil(t, errorMsg, "Should send 'error' command message")

			code, hasCode := errorMsg["code"]
			assert.True(t, hasCode, "Error must include code")
			assert.Equal(t, tt.expectedErrorCode, code,
				"Missing/invalid %s must return POSIX 71 (EPROTO)", tt.missingField)
		})
	}
}

// TestULDataMandatoryFields verifies BSSCI §3.10.1 mandatory field validation for ULData handler
func TestULDataMandatoryFields(t *testing.T) {
	testLogger := logger.NewNop()

	// First test early-exit fields (epEui, packetCnt) that return errors without sending frames
	t.Run("early_exit_validations", func(t *testing.T) {
		earlyExitTests := []struct {
			name          string
			data          map[string]interface{}
			expectedError string
		}{
			{
				name: "missing epEui",
				data: map[string]interface{}{
					"packetCnt": int64(100),
					"rxTime":    time.Now().UnixNano(),
					"userData":  []interface{}{float64(0x01), float64(0x02)},
					"snr":       float64(10.5),
					"rssi":      float64(-80.0),
				},
				expectedError: "Missing epEui",
			},
			{
				name: "missing packetCnt",
				data: map[string]interface{}{
					"epEui":    int64(TestEpEui01),
					"rxTime":   time.Now().UnixNano(),
					"userData": []interface{}{float64(0x01), float64(0x02)},
					"snr":      float64(10.5),
					"rssi":     float64(-80.0),
				},
				expectedError: "Missing packetCnt",
			},
		}

		for _, tt := range earlyExitTests {
			t.Run(tt.name, func(t *testing.T) {
				mockConn := &validationMockConn{}
				mockConn.Reset()

				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver, mockStorage :=
					CreateTestServices(testLogger, nil)

				server := NewTestServer(testLogger, mockStorage, nil, 1,
					sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver)

				// Initialize deduplicator for ULData handler
				server.SetDeduplicator(NewMessageDeduplicator(5 * time.Minute))

				session := &Session{
					ID:             "test-session",
					BaseStationEUI: TestBsEui04,
					Conn:           mockConn,
					Encoding:       "json",
				}

				msg := &Message{
					Command: "ulData",
					OpId:    1,
				}

				// Call handler - should return error without sending frame
				err := server.CallHandleULData(session, msg, tt.data)
				require.Error(t, err, "Handler should return error for missing %s", tt.name)
				assert.Contains(t, err.Error(), tt.expectedError,
					"Error message should contain expected text")

				// Verify NO error frame was sent (early exit)
				assert.Empty(t, mockConn.SentMessages,
					"Should not send error frame for early-exit validation")
			})
		}
	})

	// Test cases covering mandatory fields that send error frames
	tests := []struct {
		name              string
		data              map[string]interface{}
		expectedErrorCode float64 // 71 = POSIX_EPROTO
		missingField      string
	}{
		{
			name: "missing rxTime",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"packetCnt": int64(100),
				"userData":  []interface{}{float64(0x01), float64(0x02)},
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
			},
			expectedErrorCode: 71,
			missingField:      "rxTime",
		},
		{
			name: "invalid snr type",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"packetCnt": int64(100),
				"userData":  []interface{}{float64(0x01), float64(0x02)},
				"snr":       "not-a-number", // Invalid type
				"rssi":      float64(-80.0),
			},
			expectedErrorCode: 71,
			missingField:      "snr",
		},
		{
			name: "invalid rssi type",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"rxTime":    time.Now().UnixNano(),
				"packetCnt": int64(100),
				"userData":  []interface{}{float64(0x01), float64(0x02)},
				"snr":       float64(10.5),
				"rssi":      "not-a-number", // Invalid type
			},
			expectedErrorCode: 71,
			missingField:      "rssi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)

			// Initialize deduplicator for ULData handler
			server.SetDeduplicator(NewMessageDeduplicator(5 * time.Minute))

			session := &Session{
				ID:                "test-session",
				BaseStationEUI:    TestBsEui01,
				Conn:              mockConn,
				Encoding:          "json",
				HandshakeComplete: true,
			}

			msg := &Message{
				Command: "ulData",
				OpId:    1,
				Data:    tt.data,
			}

			_ = server.CallHandleMessage(session, msg, tt.data)

			// Verify error message sent
			require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
				"Should send error for missing/invalid %s", tt.missingField)

			var errorMsg map[string]interface{}
			for _, sentMsg := range mockConn.SentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					errorMsg = sentMsg
					break
				}
			}

			require.NotNil(t, errorMsg, "Should send 'error' command message")

			code, hasCode := errorMsg["code"]
			assert.True(t, hasCode, "Error must include code")
			assert.Equal(t, tt.expectedErrorCode, code,
				"Missing/invalid %s must return POSIX 71 (EPROTO)", tt.missingField)
		})
	}
}

// TestDLDataResultMandatoryFields verifies BSSCI §3.14.1 mandatory field validation for DLDataResult handler
func TestDLDataResultMandatoryFields(t *testing.T) {
	testLogger := logger.NewNop()

	// Test cases covering 3 base mandatory fields + 4 conditional validations
	tests := []struct {
		name              string
		data              map[string]interface{}
		expectedErrorCode float64 // 71 = POSIX_EPROTO
		missingField      string
	}{
		{
			name: "missing epEui",
			data: map[string]interface{}{
				"queId":  int64(1000),
				"result": "sent",
				"txTime": int64(time.Now().UnixNano()),
			},
			expectedErrorCode: 71,
			missingField:      "epEui",
		},
		{
			name: "missing queId",
			data: map[string]interface{}{
				"epEui":  int64(TestEpEui01),
				"result": "sent",
				"txTime": int64(time.Now().UnixNano()),
			},
			expectedErrorCode: 71,
			missingField:      "queId",
		},
		{
			name: "missing result field",
			data: map[string]interface{}{
				"epEui":  int64(TestEpEui01),
				"queId":  int64(1000),
				"txTime": int64(time.Now().UnixNano()),
			},
			expectedErrorCode: 71,
			missingField:      "result",
		},
		{
			name: "invalid result enum value",
			data: map[string]interface{}{
				"epEui":  int64(TestEpEui01),
				"queId":  int64(1000),
				"result": "invalid-status", // Not sent/expired/invalid
				"txTime": int64(time.Now().UnixNano()),
			},
			expectedErrorCode: 71,
			missingField:      "result",
		},
		{
			name: "result=sent but missing txTime",
			data: map[string]interface{}{
				"epEui":     int64(TestEpEui01),
				"queId":     int64(1000),
				"result":    "sent",
				"packetCnt": int64(100),
			},
			expectedErrorCode: 71,
			missingField:      "txTime",
		},
		{
			name: "result=sent but missing packetCnt",
			data: map[string]interface{}{
				"epEui":  int64(TestEpEui01),
				"queId":  int64(1000),
				"result": "sent",
				"txTime": int64(time.Now().UnixNano()),
			},
			expectedErrorCode: 71,
			missingField:      "packetCnt",
		},
		{
			name: "result=expired but has txTime (should not be present)",
			data: map[string]interface{}{
				"epEui":  int64(TestEpEui01),
				"queId":  int64(1000),
				"result": "expired",
				"txTime": int64(time.Now().UnixNano()), // Should not be present
			},
			expectedErrorCode: 71,
			missingField:      "txTime/packetCnt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)

			session := &Session{
				ID:                "test-session",
				BaseStationEUI:    TestBsEui01,
				Conn:              mockConn,
				Encoding:          "json",
				HandshakeComplete: true,
			}

			msg := &Message{
				Command: "dlDataRes",
				OpId:    1,
				Data:    tt.data,
			}

			_ = server.CallHandleMessage(session, msg, tt.data)

			// Verify error message sent
			require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
				"Should send error for missing/invalid %s", tt.missingField)

			var errorMsg map[string]interface{}
			for _, sentMsg := range mockConn.SentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					errorMsg = sentMsg
					break
				}
			}

			require.NotNil(t, errorMsg, "Should send 'error' command message")

			code, hasCode := errorMsg["code"]
			assert.True(t, hasCode, "Error must include code")
			assert.Equal(t, tt.expectedErrorCode, code,
				"Missing/invalid %s must return POSIX 71 (EPROTO)", tt.missingField)
		})
	}
}

// TestDetachMandatoryFields verifies BSSCI §5.7.1 mandatory field validation for detach operations
func TestDetachMandatoryFields(t *testing.T) {
	testLogger := logger.NewNop()

	// POSIX error codes per BSSCI §4
	const (
		POSIX_EPROTO = int64(71) //nolint:revive // test constants mirror POSIX names
	)

	baseDetachData := map[string]interface{}{
		"command":   "det",
		"opId":      int64(1),
		"epEui":     TestEpEui01,
		"rxTime":    time.Now().UnixNano(),
		"packetCnt": uint32(42),
		"snr":       float64(10.5),
		"rssi":      float64(-80.0),
		"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
	}

	tests := []struct {
		name              string
		data              map[string]interface{}
		missingField      string
		expectedErrorCode int64
	}{
		{
			name: "missing epEui field",
			data: func() map[string]interface{} {
				d := make(map[string]interface{})
				for k, v := range baseDetachData {
					if k != "epEui" {
						d[k] = v
					}
				}
				return d
			}(),
			missingField:      "epEui",
			expectedErrorCode: POSIX_EPROTO,
		},
		{
			name: "missing rxTime field",
			data: func() map[string]interface{} {
				d := make(map[string]interface{})
				for k, v := range baseDetachData {
					if k != "rxTime" {
						d[k] = v
					}
				}
				return d
			}(),
			missingField:      "rxTime",
			expectedErrorCode: POSIX_EPROTO,
		},
		{
			name: "missing packetCnt field",
			data: func() map[string]interface{} {
				d := make(map[string]interface{})
				for k, v := range baseDetachData {
					if k != "packetCnt" {
						d[k] = v
					}
				}
				return d
			}(),
			missingField:      "packetCnt",
			expectedErrorCode: POSIX_EPROTO,
		},
		{
			name: "missing snr field",
			data: func() map[string]interface{} {
				d := make(map[string]interface{})
				for k, v := range baseDetachData {
					if k != "snr" {
						d[k] = v
					}
				}
				return d
			}(),
			missingField:      "snr",
			expectedErrorCode: POSIX_EPROTO,
		},
		{
			name: "missing rssi field",
			data: func() map[string]interface{} {
				d := make(map[string]interface{})
				for k, v := range baseDetachData {
					if k != "rssi" {
						d[k] = v
					}
				}
				return d
			}(),
			missingField:      "rssi",
			expectedErrorCode: POSIX_EPROTO,
		},
		{
			name: "missing signature field",
			data: func() map[string]interface{} {
				d := make(map[string]interface{})
				for k, v := range baseDetachData {
					if k != "sign" {
						d[k] = v
					}
				}
				return d
			}(),
			missingField:      "sign",
			expectedErrorCode: POSIX_EPROTO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{Encoding: "msgpack"}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)
			server.SetConfig(&Config{
				DetachSignatureValidationEnabled: false, // Disable to test mandatory fields only
			})
			server.RegisterHandlers()

			session := &Session{
				ID:                "test-detach-validation",
				BaseStationEUI:    TestBsEui01,
				Conn:              mockConn,
				Encoding:          "msgpack",
				HandshakeComplete: true,
				BsOpId:            0,
				ScOpId:            0,
			}

			msg := &Message{
				Command: "det",
				OpId:    1,
				Data:    tt.data,
			}

			_ = server.CallHandleMessage(session, msg, tt.data)

			// Verify error message sent for missing mandatory field
			require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
				"Should send error for missing %s", tt.missingField)

			var errorMsg map[string]interface{}
			for _, sentMsg := range mockConn.SentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					errorMsg = sentMsg
					break
				}
			}

			require.NotNil(t, errorMsg, "Should send 'error' command message for missing mandatory field")

			code, hasCode := errorMsg["code"]
			assert.True(t, hasCode, "Error must include code")
			codeVal, err := coerceInt64(code)
			require.NoError(t, err, "code must be numeric, got %T", code)
			assert.Equal(t, int64(tt.expectedErrorCode), codeVal,
				"Missing %s must return POSIX 71 (EPROTO) per BSSCI §4", tt.missingField)
		})
	}
}

// Error token constants for attach field validation tests
// Note: Nonce/sign validation happens in the message normalizer (message_normalizer.go),
// which returns "Invalid field type" for out-of-range values. The handler's validateByteArray
// (server.go:4020-4077) only runs if normalization succeeds.
const (
	tokenInvalidFieldType      = "bssci.error.invalid_field_type"       // Normalizer error for nonce/sign
	tokenInvalidAttachCntRange = "bssci.error.invalid_attach_cnt_range" // Handler error for attachCnt 24-bit
)

// TestAttachNonceValidation verifies BSSCI §5.6.1 nonce array element validation (0-255 range)
// Tests numericToByte at message_normalizer.go:442-510 which catches invalid byte values
// before the handler's validateByteArray runs
func TestAttachNonceValidation(t *testing.T) {
	testLogger := logger.NewNop()

	tests := []struct {
		name        string
		nonce       interface{}
		expectError bool
	}{
		{"value_exceeds_255", []interface{}{float64(1), float64(2), float64(256), float64(4)}, true},
		{"negative_value", []interface{}{float64(1), float64(-1), float64(3), float64(4)}, true},
		{"fractional_value", []interface{}{float64(1), float64(2), 12.5, float64(4)}, true},
		{"non_numeric_element", []interface{}{float64(1), "bad", float64(3), float64(4)}, true},
		{"mixed_one_bad", []interface{}{float64(0), float64(255), float64(300), float64(128)}, true},
		{"valid_boundary_0", []interface{}{float64(0), float64(0), float64(0), float64(0)}, false},
		{"valid_boundary_255", []interface{}{float64(255), float64(255), float64(255), float64(255)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			storage := newAttachTestStorage()
			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, _ :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, storage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)
			server.config = &Config{
				MessageEncoding: EncodingJSON,
			}
			server.storage = storage

			// Seed endpoint so valid cases proceed past lookup and signature check
			endpoint := buildTestEndpointForValidation()
			server.endpointRepo = newFakeEndpointRepo(endpoint)
			server.RegisterHandlers()

			session := &Session{
				ID:                "test-session",
				BaseStationEUI:    TestBsEui01,
				ResolvedTenantID:  1, // Must match endpoint.TenantID
				DbSessionID:       1, // Required for pending operation persistence
				Conn:              mockConn,
				Encoding:          "json",
				HandshakeComplete: true,
			}

			data := buildValidAttachData("nonce", tt.nonce)
			msg := &Message{
				Command: "att",
				OpId:    1,
				Data:    data,
			}

			_ = server.CallHandleMessage(session, msg, data)

			if tt.expectError {
				require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
					"Should send error for invalid nonce")
				errorMsg := findErrorMessage(mockConn.SentMessages)
				require.NotNil(t, errorMsg, "Should send error command")
				// Assert POSIX code 71 (EPROTO) per BSSCI §4
				assert.Equal(t, float64(71), errorMsg["code"], "POSIX_EPROTO")
				// Normalizer catches invalid byte values and returns "Invalid field type"
				msgContent, ok := errorMsg["message"].(string)
				require.True(t, ok, "message field should be string")
				assert.Equal(t, ResolveErrorMessage(tokenInvalidFieldType), msgContent,
					"Normalizer should return Invalid field type for invalid byte values")
			} else {
				// Valid format MUST NOT trigger ANY error frame - hard fail
				errorMsg := findErrorMessage(mockConn.SentMessages)
				require.Nil(t, errorMsg, "Valid nonce %s with seeded endpoint MUST NOT trigger error frame, got: code=%v message=%v",
					tt.name, errorMsg["code"], errorMsg["message"])
				attRsp := findAttachResponse(mockConn.SentMessages)
				require.NotNil(t, attRsp, "Valid nonce %s should emit attRsp", tt.name)
			}
		})
	}
}

// TestAttachSignValidation verifies BSSCI §5.6.1 sign array element FORMAT validation (0-255 range).
// Tests numericToByte at message_normalizer.go:442-510 which catches invalid byte values.
// NOTE: This test only validates NORMALIZER behavior. Valid format cases will still fail
// CMAC signature validation since we override the sign with arbitrary test values.
// The valid format assertion verifies the normalizer doesn't reject valid byte formats.
func TestAttachSignValidation(t *testing.T) {
	testLogger := logger.NewNop()

	tests := []struct {
		name        string
		sign        interface{}
		expectError bool
	}{
		// Invalid format cases - normalizer should reject with tokenInvalidFieldType
		{"value_exceeds_255", []interface{}{float64(1), float64(2), float64(256), float64(4)}, true},
		{"negative_value", []interface{}{float64(1), float64(-1), float64(3), float64(4)}, true},
		{"fractional_value", []interface{}{float64(1), float64(2), 12.5, float64(4)}, true},
		{"non_numeric_element", []interface{}{float64(1), "bad", float64(3), float64(4)}, true},
		// Valid format cases - normalizer accepts, but CMAC validation will fail (expected)
		// These test format validation only, NOT signature validation
		{"valid_boundary_0", []interface{}{float64(0), float64(0), float64(0), float64(0)}, false},
		{"valid_boundary_255", []interface{}{float64(255), float64(255), float64(255), float64(255)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			storage := newAttachTestStorage()
			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, _ :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, storage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)
			server.config = &Config{
				MessageEncoding: EncodingJSON,
			}
			server.storage = storage

			// Seed endpoint for lookup
			endpoint := buildTestEndpointForValidation()
			server.endpointRepo = newFakeEndpointRepo(endpoint)
			server.RegisterHandlers()

			session := &Session{
				ID:                "test-session",
				BaseStationEUI:    TestBsEui01,
				ResolvedTenantID:  1,
				DbSessionID:       1,
				Conn:              mockConn,
				Encoding:          "json",
				HandshakeComplete: true,
			}

			data := buildValidAttachData("sign", tt.sign)
			msg := &Message{
				Command: "att",
				OpId:    1,
				Data:    data,
			}

			_ = server.CallHandleMessage(session, msg, data)

			if tt.expectError {
				require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
					"Should send error for invalid sign format")
				errorMsg := findErrorMessage(mockConn.SentMessages)
				require.NotNil(t, errorMsg, "Should send error command")
				// Assert POSIX code 71 (EPROTO) per BSSCI §4
				assert.Equal(t, float64(71), errorMsg["code"], "POSIX_EPROTO")
				// Normalizer catches invalid byte values and returns "Invalid field type"
				msgContent, ok := errorMsg["message"].(string)
				require.True(t, ok, "message field should be string")
				require.Equal(t, ResolveErrorMessage(tokenInvalidFieldType), msgContent,
					"Normalizer MUST return Invalid field type for invalid byte values")
			} else {
				// Valid format - normalizer passes. CMAC will fail since we override sign.
				// Assert the error is NOT a format validation error (CMAC error is expected).
				errorMsg := findErrorMessage(mockConn.SentMessages)
				require.NotNil(t, errorMsg, "Valid sign format will fail CMAC, error expected")
				msgContent, _ := errorMsg["message"].(string)
				require.NotEqual(t, ResolveErrorMessage(tokenInvalidFieldType), msgContent,
					"Valid sign FORMAT must pass normalizer - CMAC error expected, not format error")
			}
		})
	}
}

// TestAttachCounterRange verifies BSSCI §5.6.1 attachCnt 24-bit boundary validation (0-0xFFFFFF)
// Tests the normalizer Validator at message_metadata.go:216-220 for 24-bit range
// and coerceUint32 at message_normalizer.go:369-391 for negative values.
// Valid cases use computed CMAC signatures via buildAttachDataWithCnt().
func TestAttachCounterRange(t *testing.T) {
	testLogger := logger.NewNop()

	// tokenInvalidMsgFormat is returned when the normalizer's Validator returns false
	// The normalizer returns "Invalid message format" on Validator failure
	const tokenInvalidMsgFormat = "bssci.error.invalid_message_format"

	tests := []struct {
		name        string
		attachCnt   interface{}
		expectError bool
		errorToken  string // Which error token to expect
	}{
		// Negative values fail coerceUint32 -> "Invalid field type"
		{"negative", int64(-1), true, tokenInvalidFieldType},
		// Values > 16777215 fail the Validator -> "Invalid message format"
		{"exceeds_24bit", int64(0x1000000), true, tokenInvalidMsgFormat},
		{"just_over", int64(16777216), true, tokenInvalidMsgFormat},
		// Valid values pass through - use values > storedCnt (50) to pass replay protection
		{"valid_boundary_51", int64(51), false, ""},
		{"valid_max", int64(0xFFFFFF), false, ""},
		{"valid_mid", int64(8388608), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &validationMockConn{}
			mockConn.Reset()

			storage := newAttachTestStorage()
			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, _ :=
				CreateTestServices(testLogger, nil)

			server := NewTestServer(testLogger, storage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)
			server.config = &Config{
				MessageEncoding: EncodingJSON,
			}
			server.storage = storage

			// Seed endpoint for lookup
			endpoint := buildTestEndpointForValidation()
			server.endpointRepo = newFakeEndpointRepo(endpoint)
			server.RegisterHandlers()

			session := &Session{
				ID:                "test-session",
				BaseStationEUI:    TestBsEui01,
				ResolvedTenantID:  1,
				DbSessionID:       1,
				Conn:              mockConn,
				Encoding:          "json",
				HandshakeComplete: true,
			}

			var data map[string]interface{}
			if !tt.expectError {
				// Valid cases: compute CMAC signature for the specific attachCnt
				attachCntVal := tt.attachCnt.(int64)
				data = buildAttachDataWithCnt(attachCntVal)
			} else {
				// Invalid cases: use buildValidAttachData with override
				data = buildValidAttachData("attachCnt", tt.attachCnt)
			}
			msg := &Message{
				Command: "att",
				OpId:    1,
				Data:    data,
			}

			_ = server.CallHandleMessage(session, msg, data)

			if tt.expectError {
				require.GreaterOrEqual(t, len(mockConn.SentMessages), 1,
					"Should send error for invalid attachCnt")
				errorMsg := findErrorMessage(mockConn.SentMessages)
				require.NotNil(t, errorMsg, "Should send error command")
				// Assert POSIX code 71 (EPROTO) per BSSCI §4
				assert.Equal(t, float64(71), errorMsg["code"], "POSIX_EPROTO")
				// Assert error message matches expected token
				msgContent, ok := errorMsg["message"].(string)
				require.True(t, ok, "message field should be string")
				assert.Equal(t, ResolveErrorMessage(tt.errorToken), msgContent,
					"Error message must match expected catalog token")
			} else {
				// Valid format with computed CMAC MUST NOT trigger ANY error frame - hard fail
				errorMsg := findErrorMessage(mockConn.SentMessages)
				require.Nil(t, errorMsg, "Valid attachCnt %s with seeded endpoint MUST NOT trigger error frame, got: code=%v message=%v",
					tt.name, errorMsg["code"], errorMsg["message"])
				attRsp := findAttachResponse(mockConn.SentMessages)
				require.NotNil(t, attRsp, "Valid attachCnt %s should emit attRsp", tt.name)
			}
		})
	}
}
