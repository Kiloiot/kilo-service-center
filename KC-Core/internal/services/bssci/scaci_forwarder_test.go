package bssciservices

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	repodoubles "github.com/Kiloiot/kilo-service-center/KC-Core/internal/testsupport/repodoubles"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // EPStatus constants
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// errTestSCACIServer is the fixture broadcast error for the SCACI forwarder tests.
var errTestSCACIServer = errors.New("SCACI server error")

// mockSCACIEPStatusBroadcaster implements scaciEPStatusBroadcaster for testing
type mockSCACIEPStatusBroadcaster struct {
	lastTenantID int64
	lastData     *scaci.EPStatusData
	err          error
	callCount    int
}

func (m *mockSCACIEPStatusBroadcaster) BroadcastEPStatus(_ context.Context, tenantID int64, data *scaci.EPStatusData) error {
	m.callCount++
	m.lastTenantID = tenantID
	m.lastData = data
	return m.err
}

// testLogger is a no-op logger for testing
type testLogger struct{}

func (l *testLogger) Debug(_ string, _ ...interface{})                           {}
func (l *testLogger) Info(_ string, _ ...interface{})                            {}
func (l *testLogger) Warn(_ string, _ ...interface{})                            {}
func (l *testLogger) Error(_ string, _ ...interface{})                           {}
func (l *testLogger) Fatal(_ string, _ ...interface{})                           {}
func (l *testLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (l *testLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (l *testLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (l *testLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (l *testLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (l *testLogger) WithField(_ string, _ interface{}) logger.Logger            { return l }
func (l *testLogger) WithFields(_ map[string]interface{}) logger.Logger          { return l }

// =============================================================================
// SCACIEPStatusAdapter Tests
// =============================================================================

func TestSCACIEPStatusAdapter_ImplementsInterface(_ *testing.T) {
	// Verify that NewSCACIEPStatusAdapter returns SCACIEPStatusAdapterWithSetter
	var _ = NewSCACIEPStatusAdapter(nil)
}

func TestSCACIEPStatusAdapter_SetSCACIServer_Stores(t *testing.T) {
	adapter := NewSCACIEPStatusAdapter(nil)
	mock := &mockSCACIEPStatusBroadcaster{}

	// Set the SCACI server
	adapter.SetSCACIServer(mock)

	// Call BroadcastEPStatus and verify it reaches the mock
	data := &bssci.EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusAttached,
	}

	err := adapter.BroadcastEPStatus(testutil.TestContext(), 42, data)
	if err != nil {
		t.Fatalf("BroadcastEPStatus failed: %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("expected 1 call to mock, got %d", mock.callCount)
	}
	if mock.lastTenantID != 42 {
		t.Errorf("expected tenantID 42, got %d", mock.lastTenantID)
	}
}

func TestSCACIEPStatusAdapter_SetSCACIServer_NilDoesNotPanic(t *testing.T) {
	adapter := NewSCACIEPStatusAdapter(nil)

	// Setting nil should not panic
	adapter.SetSCACIServer(nil)

	// BroadcastEPStatus should return nil (no-op) when server is nil
	data := &bssci.EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusAttached,
	}

	err := adapter.BroadcastEPStatus(testutil.TestContext(), 42, data)
	if err != nil {
		t.Fatalf("BroadcastEPStatus should return nil when server is nil, got: %v", err)
	}
}

func TestSCACIEPStatusAdapter_BroadcastEPStatus_NoServer_ReturnsNil(t *testing.T) {
	// Create adapter without setting SCACI server
	adapter := NewSCACIEPStatusAdapter(nil)

	data := &bssci.EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusDetached,
	}

	// Should return nil (not an error, just no-op)
	err := adapter.BroadcastEPStatus(testutil.TestContext(), 100, data)
	if err != nil {
		t.Fatalf("expected nil error when no server, got: %v", err)
	}
}

func TestSCACIEPStatusAdapter_BroadcastEPStatus_NilData_ReturnsNil(t *testing.T) {
	adapter := NewSCACIEPStatusAdapter(nil)
	mock := &mockSCACIEPStatusBroadcaster{}
	adapter.SetSCACIServer(mock)

	// Nil data should return nil without calling the server
	err := adapter.BroadcastEPStatus(testutil.TestContext(), 42, nil)
	if err != nil {
		t.Fatalf("expected nil error for nil data, got: %v", err)
	}

	if mock.callCount != 0 {
		t.Errorf("expected 0 calls to mock for nil data, got %d", mock.callCount)
	}
}

func TestSCACIEPStatusAdapter_BroadcastEPStatus_ConvertsTypes(t *testing.T) {
	adapter := NewSCACIEPStatusAdapter(nil)
	mock := &mockSCACIEPStatusBroadcaster{}
	adapter.SetSCACIServer(mock)

	// Create bssci.EPStatusData with all fields populated
	attachCnt := uint32(5)
	nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
	sign := mioty.Numeric4{0x0A, 0x0B, 0x0C, 0x0D}
	snr := 15.5
	rssi := -80.0
	eqSnr := 12.3
	subpackets := &mioty.Subpackets{
		SNR:       []float64{1.0, 2.0, 3.0},
		RSSI:      []float64{-70.0, -75.0, -80.0},
		Frequency: []int64{868100000, 868300000, 868500000},
		Phase:     []float64{0.5, 1.0, 1.5},
	}

	data := &bssci.EPStatusData{
		EpEui:      0xAABBCCDDEEFF1122,
		EpStatus:   pkgmioty.EPStatusAttached,
		AttachCnt:  &attachCnt,
		Nonce:      &nonce,
		Sign:       &sign,
		Snr:        &snr,
		Rssi:       &rssi,
		EqSnr:      &eqSnr,
		Subpackets: subpackets,
	}

	err := adapter.BroadcastEPStatus(testutil.TestContext(), 123, data)
	if err != nil {
		t.Fatalf("BroadcastEPStatus failed: %v", err)
	}

	// Verify all fields were converted correctly
	if mock.lastData == nil {
		t.Fatal("expected data to be passed to mock")
	}

	if mock.lastData.EpEui != 0xAABBCCDDEEFF1122 {
		t.Errorf("EpEui mismatch: got 0x%X", mock.lastData.EpEui)
	}
	if mock.lastData.EpStatus != pkgmioty.EPStatusAttached {
		t.Errorf("EpStatus mismatch: got %s", mock.lastData.EpStatus)
	}
	if mock.lastData.AttachCnt == nil || *mock.lastData.AttachCnt != 5 {
		t.Errorf("AttachCnt mismatch")
	}
	if mock.lastData.Nonce == nil || *mock.lastData.Nonce != nonce {
		t.Errorf("Nonce mismatch")
	}
	if mock.lastData.Sign == nil || *mock.lastData.Sign != sign {
		t.Errorf("Sign mismatch")
	}
	if mock.lastData.Snr == nil || *mock.lastData.Snr != 15.5 {
		t.Errorf("Snr mismatch")
	}
	if mock.lastData.Rssi == nil || *mock.lastData.Rssi != -80.0 {
		t.Errorf("Rssi mismatch")
	}
	if mock.lastData.EqSnr == nil || *mock.lastData.EqSnr != 12.3 {
		t.Errorf("EqSnr mismatch")
	}
	if mock.lastData.Subpackets == nil {
		t.Fatal("Subpackets should not be nil")
	}
	if len(mock.lastData.Subpackets.SNR) != 3 {
		t.Errorf("Subpackets.SNR length mismatch: got %d", len(mock.lastData.Subpackets.SNR))
	}
}

func TestSCACIEPStatusAdapter_BroadcastEPStatus_PropagatesError(t *testing.T) {
	// Use testLogger to avoid nil pointer dereference when error is logged
	adapter := NewSCACIEPStatusAdapter(&testLogger{})
	expectedErr := errTestSCACIServer
	mock := &mockSCACIEPStatusBroadcaster{err: expectedErr}
	adapter.SetSCACIServer(mock)

	data := &bssci.EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusAttached,
	}

	err := adapter.BroadcastEPStatus(testutil.TestContext(), 42, data)
	if err == nil {
		t.Fatal("expected error to be propagated")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestSCACIEPStatusAdapter_BroadcastEPStatus_AttachedStatus(t *testing.T) {
	adapter := NewSCACIEPStatusAdapter(nil)
	mock := &mockSCACIEPStatusBroadcaster{}
	adapter.SetSCACIServer(mock)

	data := &bssci.EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusAttached,
	}

	err := adapter.BroadcastEPStatus(testutil.TestContext(), 42, data)
	if err != nil {
		t.Fatalf("BroadcastEPStatus failed: %v", err)
	}

	if mock.lastData.EpStatus != pkgmioty.EPStatusAttached {
		t.Errorf("expected EPStatusAttached, got %s", mock.lastData.EpStatus)
	}
}

func TestSCACIEPStatusAdapter_BroadcastEPStatus_DetachedStatus(t *testing.T) {
	adapter := NewSCACIEPStatusAdapter(nil)
	mock := &mockSCACIEPStatusBroadcaster{}
	adapter.SetSCACIServer(mock)

	data := &bssci.EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusDetached,
	}

	err := adapter.BroadcastEPStatus(testutil.TestContext(), 42, data)
	if err != nil {
		t.Fatalf("BroadcastEPStatus failed: %v", err)
	}

	if mock.lastData.EpStatus != pkgmioty.EPStatusDetached {
		t.Errorf("expected EPStatusDetached, got %s", mock.lastData.EpStatus)
	}
}

// wiringDownlinkWriter and wiringQueueStore are the minimal queue fakes the
// bundle constructor needs; the forwarding path under test never touches them.
type wiringDownlinkWriter struct{ DownlinkQueueStores }

type wiringQueueStore struct{}

func (w *wiringQueueStore) GetTenantIDByQueueID(context.Context, uint64) (int64, error) {
	return 0, nil
}

// fakeSCACIServer records what the forwarders relay so the composition can be
// proven end to end: message in on the bundle side, message out on the SCACI
// side. The wiring these tests protect was silently broken for months because
// the setter was reached through a type assertion that could never succeed.
type fakeSCACIServer struct {
	ulData    []*mioty.ULDataMessage
	dlResults []*mioty.DLDataResult
	epStatus  []*scaci.EPStatusData
}

func (f *fakeSCACIServer) BroadcastULData(_ context.Context, _ int64, data *mioty.ULDataMessage) error {
	f.ulData = append(f.ulData, data)
	return nil
}

func (f *fakeSCACIServer) BroadcastDLDataResult(_ context.Context, _ scaci.ApplicationCenter, _ uint64, result *mioty.DLDataResult) error {
	f.dlResults = append(f.dlResults, result)
	return nil
}

func (f *fakeSCACIServer) BroadcastEPStatus(_ context.Context, _ int64, data *scaci.EPStatusData) error {
	f.epStatus = append(f.epStatus, data)
	return nil
}

// TestBSSCIToSCACIWiring mirrors the composition root: build the service
// bundle, wire a SCACI server through the typed setters exactly as
// buildSCACIServer does, and prove UL data, DL results and EPStatus all arrive.
func TestBSSCIToSCACIWiring(t *testing.T) {
	ctx := testutil.TestContext()
	log := logger.NewNop()

	var pendingOps map[bssci.SessionOpKey]*bssci.PendingOperation
	var pendingOpsMu sync.RWMutex
	pendingOps = make(map[bssci.SessionOpKey]*bssci.PendingOperation)

	bundle, err := NewBSSCIServices(
		repodoubles.NewBaseStationSessionRepo(),
		&repodoubles.BaseStationRepo{},
		&repodoubles.PendingOperationRepository{},
		&wiringDownlinkWriter{},
		&repodoubles.SystemEventStore{},
		&wiringQueueStore{},
		nil, // connection manager is not part of the forwarding path
		log,
		1,
		bssci.TestScEui01,
		&pendingOps,
		&pendingOpsMu,
		[]string{"1.0.0"},
		clock.SystemClock{},
		DownlinkResultsWithoutMQTT{},
		NewBackgroundWork(),
	)
	if err != nil {
		t.Fatalf("NewBSSCIServices: %v", err)
	}

	server := &fakeSCACIServer{}

	// The two lines under test: this is what the composition root does.
	bundle.Broadcaster.SetSCACIServer(server)
	bundle.EPStatusBroadcaster.SetSCACIServer(server)

	if err := bundle.Broadcaster.BroadcastULData(ctx, 1, &mioty.ULDataMessage{EpEui: 0x11}); err != nil {
		t.Fatalf("BroadcastULData: %v", err)
	}
	if err := bundle.Broadcaster.BroadcastDLDataResult(ctx, scaci.ApplicationCenter{TenantID: 1, AcEui: 0x70B3D59CD0000A01}, 7, &mioty.DLDataResult{QueId: 7}); err != nil {
		t.Fatalf("BroadcastDLDataResult: %v", err)
	}
	if err := bundle.EPStatusBroadcaster.BroadcastEPStatus(ctx, 1, &bssci.EPStatusData{EpEui: 0x22}); err != nil {
		t.Fatalf("BroadcastEPStatus: %v", err)
	}

	if len(server.ulData) != 1 || server.ulData[0].EpEui != 0x11 {
		t.Errorf("UL data did not reach the SCACI server: %+v", server.ulData)
	}
	if len(server.dlResults) != 1 || server.dlResults[0].QueId != 7 {
		t.Errorf("DL result did not reach the SCACI server: %+v", server.dlResults)
	}
	if len(server.epStatus) != 1 || server.epStatus[0].EpEui != 0x22 {
		t.Errorf("EPStatus did not reach the SCACI server: %+v", server.epStatus)
	}
}
