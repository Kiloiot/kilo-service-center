package grpc

import (
	"context"
	"reflect"
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeEndpointSvc is a minimal fake for endpoint service that always returns success
type fakeEndpointSvc struct{}

func (f *fakeEndpointSvc) GetByEUI(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
	return &models.EndPoint{EUI: models.EUIFromString("0102030405060708")}, nil
}
func (f *fakeEndpointSvc) Create(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointSvc) Update(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointSvc) Delete(_ context.Context, _ []byte, _ int64) error { return nil }
func (f *fakeEndpointSvc) ListByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (f *fakeEndpointSvc) List(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointSvc) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (f *fakeEndpointSvc) CheckEUIGloballyUnique(_ context.Context, _ []byte) error { return nil }

// fakeLegacyStorage is a minimal fake that implements storage.Storage interface
type fakeLegacyStorage struct{}

func (f *fakeLegacyStorage) EnqueueDownlink(_ context.Context, _ *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	return &storage.DownlinkMessage{ID: 1, QueID: 12345}, nil
}
func (f *fakeLegacyStorage) CreateEndPoint(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) GetEndPoint(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) UpdateEndPoint(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) DeleteEndPoint(_ context.Context, _ []byte, _ int64) error { return nil }
func (f *fakeLegacyStorage) ListEndPointsByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (f *fakeLegacyStorage) ListEndPoints(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) CreateBaseStation(_ context.Context, _ *models.BaseStation) (*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) GetBaseStation(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) UpdateBaseStation(_ context.Context, _ *models.BaseStation) (*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) DeleteBaseStation(_ context.Context, _ []byte, _ int64) error { return nil }
func (f *fakeLegacyStorage) ListBaseStations(_ context.Context, _ int64, _, _ int) ([]*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) GetDownlinkQueue(_ context.Context, _, _ string) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) GetDownlinkResults(_ context.Context, _, _ string, _ *uuid.UUID, _ string, _, _ *time.Time, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return nil, 0, nil
}
func (f *fakeLegacyStorage) UpdateDownlinkStatus(_ context.Context, _, _ string, _ *uuid.UUID) error {
	return nil
}
func (f *fakeLegacyStorage) UpdateDownlinkBaseStation(_ context.Context, _ uint64, _ string, _ uint64) error {
	return nil
}
func (f *fakeLegacyStorage) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) GetDownlinkByPacketCnt(_ context.Context, _, _ string, _ uint32) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) RevokeDownlink(_ context.Context, _ int64, _ string) error { return nil }
func (f *fakeLegacyStorage) UpdateDownlinkResult(_ context.Context, _ int64, _ string, _ *int64, _ *uint32, _, _ []byte, _ string, _ *uuid.UUID) error {
	return nil
}
func (f *fakeLegacyStorage) CreateDLRXStatus(_ context.Context, _ *mioty.DLRXStatus) error {
	return nil
}
func (f *fakeLegacyStorage) GetDLRXStatusByEndpoint(_ context.Context, _ int64, _ []byte, _, _ int, _, _ *time.Time) ([]*mioty.DLRXStatus, int, error) {
	return nil, 0, nil
}
func (f *fakeLegacyStorage) GetAverageDLRXMetrics(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (float64, float64, int, error) {
	return 0, 0, 0, nil
}
func (f *fakeLegacyStorage) CreateDLRXStatusQuery(_ context.Context, _ int64, _ *uuid.UUID, _, _ []byte, _ int64) error {
	return nil
}
func (f *fakeLegacyStorage) MarkDLRXStatusReceived(_ context.Context, _ int64, _ []byte, _ []byte, _ int64) (bool, error) {
	return false, nil
}
func (f *fakeLegacyStorage) ExpireDLRXStatusQuery(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeLegacyStorage) GetDLRXStatusQueryHistory(_ context.Context, _ int64, _ []byte, _, _ int, _, _ *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	return nil, 0, nil
}
func (f *fakeLegacyStorage) GetDLRXStatusQueryStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (int64, int64, int64, error) {
	return 0, 0, 0, nil
}
func (f *fakeLegacyStorage) GetEndpointBaseStation(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (f *fakeLegacyStorage) GetBaseStationMessageStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) GetBaseStationEndpointCounts(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (map[string]int64, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) GetBaseStationLastSeen(_ context.Context, _ int64, _ []byte) (*time.Time, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) Ping(_ context.Context) error { return nil }
func (f *fakeLegacyStorage) Close() error                 { return nil }

// Roaming support methods (stubs for test fake)
func (f *fakeLegacyStorage) GetEndpointOwner(_ context.Context, _ []byte) (int64, error) {
	return 0, nil
}
func (f *fakeLegacyStorage) GetEndpointWithOwnership(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) IsRoamingEnabled(_ context.Context, _ int64) (bool, error) {
	return false, nil
}
func (f *fakeLegacyStorage) AreTenantsPartners(_ context.Context, _, _ int64) (bool, error) {
	return false, nil
}
func (f *fakeLegacyStorage) RecordRoamingEvent(_ context.Context, _ *models.RoamingEvent) error {
	return nil
}
func (f *fakeLegacyStorage) GetRoamingStatistics(_ context.Context, _ int64) (*models.RoamingStatistics, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) UpdateEndpointRoamingStatus(_ context.Context, _ []byte, _ int64) error {
	return nil
}
func (f *fakeLegacyStorage) GetRoamingEndpointsInSession(_ context.Context, _ int64) ([]models.RoamingEndpointInfo, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) AddRoamingEndpointToSession(_ context.Context, _ int64, _ string, _ int64) error {
	return nil
}
func (f *fakeLegacyStorage) RemoveRoamingEndpointFromSession(_ context.Context, _ int64, _ string) error {
	return nil
}
func (f *fakeLegacyStorage) GetTenantRoamingConfig(_ context.Context, _ int64) (*models.TenantRoamingConfig, error) {
	return nil, nil
}
func (f *fakeLegacyStorage) UpdateTenantRoamingConfig(_ context.Context, _ int64, _ *models.TenantRoamingConfig) error {
	return nil
}

// fakeSCACIQueuer implements SCACIDownlinkQueuer for SCACI-only path testing
// SCACI §3.10: Required for all SendDownlink tests
type fakeSCACIQueuer struct {
	lastPacketCnt []uint32           // Captures packet counter values sent to SCACI
	lastQueId     uint64             // Captures the QueId passed to QueueDownlinkInternal
	lastDlReq     *mioty.DLDataQueue // Captures the full request for inspection
}

func (f *fakeSCACIQueuer) QueueDownlinkInternal(_ context.Context, _ int64, _ *uuid.UUID, req *mioty.DLDataQueue) (*scaci.DLDataQueueResult, error) {
	if req != nil {
		f.lastPacketCnt = req.PacketCnt
		f.lastQueId = req.QueId
		f.lastDlReq = req
	}
	return &scaci.DLDataQueueResult{
		QueID:  req.QueId,
		BsEui:  0x0102030405060708,
		OpID:   -12345,
		Status: grpcerrors.StatusQueued,
	}, nil
}

// fakeSessionDirectory implements bssci.SessionDirectory
type fakeSessionDirectory struct{}

func (f *fakeSessionDirectory) GetConnectedSessions() []map[string]interface{} { return nil }
func (f *fakeSessionDirectory) GetSessionByEUI(_ uint64) interface{}           { return nil }
func (f *fakeSessionDirectory) SelectBidirectionalSession(_ int64, _ *uint64) (string, uint64, error) {
	return "", 0, nil
}
func (f *fakeSessionDirectory) FindSessionForEndpointAttachment(_ uint64) (string, error) {
	return "", nil
}

// fakeBasestationSvc implements grpcservices.BaseStationService
type fakeBasestationSvc struct{}

func (f *fakeBasestationSvc) Create(_ context.Context, _ *models.BaseStation) (*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeBasestationSvc) GetByEUI(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeBasestationSvc) Update(_ context.Context, _ *models.BaseStation) (*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeBasestationSvc) Delete(_ context.Context, _ []byte, _ int64) error { return nil }
func (f *fakeBasestationSvc) List(_ context.Context, _ int64, _, _ int) ([]*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeBasestationSvc) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}
func (f *fakeBasestationSvc) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// fakeMessageSvc implements grpcservices.MessageService
type fakeMessageSvc struct{}

func (f *fakeMessageSvc) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (f *fakeMessageSvc) GetDownlinkQueue(_ context.Context, _, _ string) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}
func (f *fakeMessageSvc) GetDownlinkResults(_ context.Context, _, _ string, _ *uuid.UUID, _ string, _, _ *time.Time, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return nil, 0, nil
}
func (f *fakeMessageSvc) GetDLRXStatusByEndpoint(_ context.Context, _ int64, _ []byte, _, _ int, _, _ *time.Time) ([]*mioty.DLRXStatus, int, error) {
	return nil, 0, nil
}
func (f *fakeMessageSvc) GetAverageDLRXMetrics(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (float64, float64, int, error) {
	return 0, 0, 0, nil
}
func (f *fakeMessageSvc) GetEndpointBaseStation(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// fakeDownlinkSvc implements bssci.DownlinkService (minimal for SendDownlink tests)
type fakeDownlinkSvc struct{}

func (f *fakeDownlinkSvc) EnqueueDownlink(_ context.Context, _ uint64, _ []byte, _ float32, _ int64) (int64, error) {
	return 0, nil
}
func (f *fakeDownlinkSvc) ProcessDLDataResult(_ context.Context, _ *bssci.Session, _ *mioty.DLDataResult) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeDownlinkSvc) UpdateDownlinkStatus(_ context.Context, _ uint64, _ string, _ string) error {
	return nil
}
func (f *fakeDownlinkSvc) ProcessRevokeResponse(_ context.Context, _ *bssci.Session, _ int64, _ int64, _ uint64) (map[string]interface{}, error) {
	return nil, nil
}

// fakeStatusSvc implements bssci.StatusService (minimal for SendDownlink tests)
type fakeStatusSvc struct{}

func (f *fakeStatusSvc) RecordPendingOperation(_ context.Context, _ *bssci.Session, _ int64, _ *bssci.PendingOperation, _ int64) error {
	return nil
}
func (f *fakeStatusSvc) GetPendingOperation(_ *bssci.Session, _ int64) (*bssci.PendingOperation, error) {
	return nil, nil
}
func (f *fakeStatusSvc) RemovePendingOperation(_ context.Context, _ *bssci.Session, _ int64) error {
	return nil
}
func (f *fakeStatusSvc) ExtractQueueMetadata(_ *bssci.Session, _ int64) (uint64, int64, string) {
	return 0, 0, ""
}
func (f *fakeStatusSvc) CleanupPendingOp(_ *bssci.Session, _ int64) {}

// fakeDownlinkCmd implements bssci.DownlinkCommander (minimal for SendDownlink tests)
type fakeDownlinkCmd struct {
	lastPacketCnt []int64 // Captures packet counter values sent to base station
}

func (f *fakeDownlinkCmd) SendDLDataQueue(_ string, _ uint64, _ [][]byte, _ int64, _ float32, _ bool, packetCnt []int64, _ uint8, _ bool, _ bool, _ bool, _ bool, _ int64) error {
	f.lastPacketCnt = packetCnt
	return nil
}
func (f *fakeDownlinkCmd) SendDLDataRevoke(_ string, _ uint64, _ uint64) error { return nil }
func (f *fakeDownlinkCmd) SendDLRXStatusQuery(_ string, _ uint64) error        { return nil }

// fakeULTransmit implements bssci.ULTransmitter (minimal for SendDownlink tests)
type fakeULTransmit struct{}

func (f *fakeULTransmit) SendULDataTransmit(_ string, _ uint64, _ []byte, _ uint16, _ uint32, _ []byte, _ string, _ uint8) (int64, error) {
	return 0, nil
}

// fakeStatusReq implements bssci.StatusRequester (minimal for SendDownlink tests)
type fakeStatusReq struct{}

func (f *fakeStatusReq) SendStatusRequest(_ interface{}) (int64, error) { return 0, nil }

// fakePingCmd implements bssci.PingCommander (minimal for SendDownlink tests)
type fakePingCmd struct{}

func (f *fakePingCmd) InitiatePing(_ context.Context, _ uint64, _ int64) (int64, error) {
	return 0, nil
}

// fakeDownlinkScheduler implements scheduler.DownlinkScheduler (minimal for SendDownlink tests)
type fakeDownlinkScheduler struct{}

func (f *fakeDownlinkScheduler) QueueDownlink(_ *mioty.DLDataQueue, _ int64) (uint64, uint64, error) {
	return 12345, 0x0102030405060708, nil
}

func (f *fakeDownlinkScheduler) RevokeDownlink(_ int64, _ uint64) (uint64, error) {
	return 0x0102030405060708, nil
}

// Compile-time interface assertions prevent future regressions when interfaces evolve
var _ grpcservices.EndpointService = (*fakeEndpointSvc)(nil)
var _ grpcservices.BaseStationService = (*fakeBasestationSvc)(nil)
var _ grpcservices.MessageService = (*fakeMessageSvc)(nil)
var _ bssci.DownlinkService = (*fakeDownlinkSvc)(nil)
var _ bssci.StatusService = (*fakeStatusSvc)(nil)
var _ bssci.DownlinkCommander = (*fakeDownlinkCmd)(nil)
var _ bssci.ULTransmitter = (*fakeULTransmit)(nil)
var _ bssci.StatusRequester = (*fakeStatusReq)(nil)
var _ bssci.PingCommander = (*fakePingCmd)(nil)
var _ scheduler.DownlinkScheduler = (*fakeDownlinkScheduler)(nil)
var _ DownlinkStore = (*fakeLegacyStorage)(nil)

// BSSCI §§5.11-5.12.3 packet counter validation (kilocenter_service.go:626-629):
// - Accepts values in uint32 range [0, 4294967295] (see TestSendDownlink_ValidPacketCounters)
// - Rejects negative values (see TestSendDownlink_NegativePacketCounter)
// - Rejects overflow values (see TestSendDownlink_OverflowPacketCounter)
// - Validates each element in arrays (see TestSendDownlink_MixedValidInvalid)

// TestSendDownlink_ValidPacketCounters verifies that valid packet counter values
// in the uint32 range [0, 4294967295] are accepted.
// BSSCI §§5.11-5.12.3 Gap 2: Valid values must pass validation
func TestSendDownlink_ValidPacketCounters(t *testing.T) {
	tests := []struct {
		name      string
		packetCnt []int64
	}{
		{name: "zero", packetCnt: []int64{0}},
		{name: "one", packetCnt: []int64{1}},
		{name: "typical", packetCnt: []int64{100}},
		{name: "max uint32", packetCnt: []int64{4294967295}},
		{name: "multiple valid", packetCnt: []int64{0, 1, 100, 4294967295}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SCACI §3.10: Create fakeSCACIQueuer to capture transmission (replaces scheduler)
			scaciQueuer := &fakeSCACIQueuer{}

			// Use single shared storage instance to mirror production state sharing
			storage := &fakeLegacyStorage{}

			svc, err := NewCoreService(CoreServiceDeps{
				EndpointSvc:       &fakeEndpointSvc{},
				BasestationSvc:    &fakeBasestationSvc{},
				MessageSvc:        &fakeMessageSvc{},
				DownlinkSvc:       &fakeDownlinkSvc{},
				StatusSvc:         &fakeStatusSvc{},
				DownlinkCmd:       &fakeDownlinkCmd{},
				DownlinkScheduler: &fakeDownlinkScheduler{},
				SessionDir:        &fakeSessionDirectory{},
				ULTransmit:        &fakeULTransmit{},
				StatusReq:         &fakeStatusReq{},
				PingCmd:           &fakePingCmd{},
				StatsStore:        storage,
				DownlinkStore:     storage,
				DLRXStorage:       storage,
				SCEui:             0x0000000000000001,
				SCVendor:          "Test",
				SCModel:           "TestCenter",
				SCName:            "test-instance",
				SCSwVersion:       "test-1.0",
			})
			if err != nil {
				t.Fatalf("NewCoreService() error = %v", err)
			}
			// SCACI §3.10: Wire SCACI queuer (required for SendDownlink)
			svc = svc.WithSCACIQueuer(scaciQueuer)

			ctx := testutil.TestContextWithTenant(1)
			payloads := make([][]byte, len(tt.packetCnt))
			for i := range payloads {
				payloads[i] = []byte("test")
			}

			req := &pb.SendDownlinkRequest{
				EpEui:        "0102030405060708",
				Payloads:     payloads,
				Priority:     1.0,
				Format:       0,
				CntDepend:    true,
				PacketCnt:    tt.packetCnt,
				ResponseExp:  false,
				ResponsePrio: false,
				DlWindReq:    false,
				ExpOnly:      false,
			}

			resp, err := svc.SendDownlink(ctx, req)
			if err != nil {
				t.Fatalf("SendDownlink() unexpected error for valid packet counter: %v", err)
			}
			if resp == nil {
				t.Fatal("SendDownlink() returned nil response for valid packet counter")
			}
			if resp.Id == "" {
				t.Error("SendDownlink() response missing Id")
			}

			// SCACI §3.10: Verify packet counters were transmitted to SCACI queuer as uint32
			expectedUint32 := make([]uint32, len(tt.packetCnt))
			for i, v := range tt.packetCnt {
				expectedUint32[i] = uint32(v)
			}
			if !reflect.DeepEqual(scaciQueuer.lastPacketCnt, expectedUint32) {
				t.Errorf("Packet counter transmission mismatch:\n  got:  %v\n  want: %v",
					scaciQueuer.lastPacketCnt, expectedUint32)
			}
		})
	}
}

// TestSendDownlink_NegativePacketCounter verifies that negative packet counter values
// are rejected with InvalidArgument error.
// BSSCI §§5.11-5.12.3 Gap 2: Negative values are outside uint32 range
func TestSendDownlink_NegativePacketCounter(t *testing.T) {
	expectedMessage := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkFormatInvalid)

	tests := []struct {
		name      string
		packetCnt []int64
		wantCode  codes.Code
	}{
		{
			name:      "negative one",
			packetCnt: []int64{-1},
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "large negative",
			packetCnt: []int64{-100},
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "very large negative",
			packetCnt: []int64{-999999},
			wantCode:  codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SCACI §3.10: Use SCACI-only path
			scaciQueuer := &fakeSCACIQueuer{}
			storage := &fakeLegacyStorage{}

			svc, err := NewCoreService(CoreServiceDeps{
				EndpointSvc:       &fakeEndpointSvc{},
				BasestationSvc:    &fakeBasestationSvc{},
				MessageSvc:        &fakeMessageSvc{},
				DownlinkSvc:       &fakeDownlinkSvc{},
				StatusSvc:         &fakeStatusSvc{},
				DownlinkCmd:       &fakeDownlinkCmd{},
				DownlinkScheduler: &fakeDownlinkScheduler{},
				SessionDir:        &fakeSessionDirectory{},
				ULTransmit:        &fakeULTransmit{},
				StatusReq:         &fakeStatusReq{},
				PingCmd:           &fakePingCmd{},
				StatsStore:        storage,
				DownlinkStore:     storage,
				DLRXStorage:       storage,
				SCEui:             0x0000000000000001,
				SCVendor:          "Test",
				SCModel:           "TestCenter",
				SCName:            "test-instance",
				SCSwVersion:       "test-1.0",
			})
			if err != nil {
				t.Fatalf("NewCoreService() error = %v", err)
			}
			svc = svc.WithSCACIQueuer(scaciQueuer)

			ctx := testutil.TestContextWithTenant(1)
			req := &pb.SendDownlinkRequest{
				EpEui:        "0102030405060708",
				Payloads:     [][]byte{[]byte("test")},
				Priority:     1.0,
				Format:       0,
				CntDepend:    true,
				PacketCnt:    tt.packetCnt,
				ResponseExp:  false,
				ResponsePrio: false,
				DlWindReq:    false,
				ExpOnly:      false,
			}

			_, err = svc.SendDownlink(ctx, req)
			if err == nil {
				t.Fatal("SendDownlink() expected error for negative packet counter, got nil")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("SendDownlink() code = %v, want %v", st.Code(), tt.wantCode)
			}
			if st.Message() != expectedMessage {
				t.Errorf("SendDownlink() message = %q, want %q", st.Message(), expectedMessage)
			}
		})
	}
}

// TestSendDownlink_OverflowPacketCounter verifies that packet counter values > uint32 max
// are rejected with InvalidArgument error.
// BSSCI §§5.11-5.12.3 Gap 2: Values > 4294967295 overflow uint32
func TestSendDownlink_OverflowPacketCounter(t *testing.T) {
	expectedMessage := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkFormatInvalid)

	tests := []struct {
		name      string
		packetCnt []int64
		wantCode  codes.Code
	}{
		{
			name:      "uint32 max + 1",
			packetCnt: []int64{4294967296},
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "large overflow",
			packetCnt: []int64{5000000000},
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "very large overflow",
			packetCnt: []int64{9999999999},
			wantCode:  codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SCACI §3.10: Use SCACI-only path
			scaciQueuer := &fakeSCACIQueuer{}
			storage := &fakeLegacyStorage{}

			svc, err := NewCoreService(CoreServiceDeps{
				EndpointSvc:       &fakeEndpointSvc{},
				BasestationSvc:    &fakeBasestationSvc{},
				MessageSvc:        &fakeMessageSvc{},
				DownlinkSvc:       &fakeDownlinkSvc{},
				StatusSvc:         &fakeStatusSvc{},
				DownlinkCmd:       &fakeDownlinkCmd{},
				DownlinkScheduler: &fakeDownlinkScheduler{},
				SessionDir:        &fakeSessionDirectory{},
				ULTransmit:        &fakeULTransmit{},
				StatusReq:         &fakeStatusReq{},
				PingCmd:           &fakePingCmd{},
				StatsStore:        storage,
				DownlinkStore:     storage,
				DLRXStorage:       storage,
				SCEui:             0x0000000000000001,
				SCVendor:          "Test",
				SCModel:           "TestCenter",
				SCName:            "test-instance",
				SCSwVersion:       "test-1.0",
			})
			if err != nil {
				t.Fatalf("NewCoreService() error = %v", err)
			}
			svc = svc.WithSCACIQueuer(scaciQueuer)

			ctx := testutil.TestContextWithTenant(1)
			req := &pb.SendDownlinkRequest{
				EpEui:        "0102030405060708",
				Payloads:     [][]byte{[]byte("test")},
				Priority:     1.0,
				Format:       0,
				CntDepend:    true,
				PacketCnt:    tt.packetCnt,
				ResponseExp:  false,
				ResponsePrio: false,
				DlWindReq:    false,
				ExpOnly:      false,
			}

			_, err = svc.SendDownlink(ctx, req)
			if err == nil {
				t.Fatal("SendDownlink() expected error for overflow packet counter, got nil")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("SendDownlink() code = %v, want %v", st.Code(), tt.wantCode)
			}
			if st.Message() != expectedMessage {
				t.Errorf("SendDownlink() message = %q, want %q", st.Message(), expectedMessage)
			}
		})
	}
}

// TestSendDownlink_MixedValidInvalid verifies that when a packet counter array contains
// both valid and invalid values, the entire request is rejected.
// BSSCI §§5.11-5.12.3 Gap 2: Validation fails fast on first invalid counter
func TestSendDownlink_MixedValidInvalid(t *testing.T) {
	expectedMessage := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkFormatInvalid)

	tests := []struct {
		name      string
		packetCnt []int64
		wantCode  codes.Code
	}{
		{
			name:      "valid then negative",
			packetCnt: []int64{0, -1},
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "valid then overflow",
			packetCnt: []int64{100, 4294967296},
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "multiple valid then negative",
			packetCnt: []int64{0, 100, 1000, -5, 4294967295},
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "multiple valid then overflow",
			packetCnt: []int64{0, 100, 5000000000, 4294967295},
			wantCode:  codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SCACI §3.10: Use SCACI-only path
			scaciQueuer := &fakeSCACIQueuer{}
			storage := &fakeLegacyStorage{}

			svc, err := NewCoreService(CoreServiceDeps{
				EndpointSvc:       &fakeEndpointSvc{},
				BasestationSvc:    &fakeBasestationSvc{},
				MessageSvc:        &fakeMessageSvc{},
				DownlinkSvc:       &fakeDownlinkSvc{},
				StatusSvc:         &fakeStatusSvc{},
				DownlinkCmd:       &fakeDownlinkCmd{},
				DownlinkScheduler: &fakeDownlinkScheduler{},
				SessionDir:        &fakeSessionDirectory{},
				ULTransmit:        &fakeULTransmit{},
				StatusReq:         &fakeStatusReq{},
				PingCmd:           &fakePingCmd{},
				StatsStore:        storage,
				DownlinkStore:     storage,
				DLRXStorage:       storage,
				SCEui:             0x0000000000000001,
				SCVendor:          "Test",
				SCModel:           "TestCenter",
				SCName:            "test-instance",
				SCSwVersion:       "test-1.0",
			})
			if err != nil {
				t.Fatalf("NewCoreService() error = %v", err)
			}
			svc = svc.WithSCACIQueuer(scaciQueuer)

			ctx := testutil.TestContextWithTenant(1)
			payloads := make([][]byte, len(tt.packetCnt))
			for i := range payloads {
				payloads[i] = []byte("test")
			}

			req := &pb.SendDownlinkRequest{
				EpEui:        "0102030405060708",
				Payloads:     payloads,
				Priority:     1.0,
				Format:       0,
				CntDepend:    true,
				PacketCnt:    tt.packetCnt,
				ResponseExp:  false,
				ResponsePrio: false,
				DlWindReq:    false,
				ExpOnly:      false,
			}

			_, err = svc.SendDownlink(ctx, req)
			if err == nil {
				t.Fatal("SendDownlink() expected error for mixed valid/invalid counters, got nil")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("SendDownlink() code = %v, want %v", st.Code(), tt.wantCode)
			}
			if st.Message() != expectedMessage {
				t.Errorf("SendDownlink() message = %q, want %q", st.Message(), expectedMessage)
			}
		})
	}
}
