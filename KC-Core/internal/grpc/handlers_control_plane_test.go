package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/diagnostics"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/errorgroups"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	pendingTestQueID    = int64(9001)
	pendingTestEpEUI    = "70B3D59CD0000002"
	pendingTestPriority = float32(0.75)
	pendingTestFormat   = uint32(3)
	malformedEUI        = "not-an-eui"
	pendingTestFrom     = int64(1_700_000_000)
	capabilityTestName  = "scaci"
	bucketTestCount     = int64(2)
	capabilityTestOn    = true
)

// pendingMessageSvc records the patch and answers with the configured error.
type pendingMessageSvc struct {
	fakeMessageSvc
	calls     int
	lastOrgID *uuid.UUID
	lastEpEUI []byte
	lastQueID int64
	lastPatch storage.DownlinkPatch
	err       error
}

func (m *pendingMessageSvc) UpdatePendingDownlink(_ context.Context, _ int64, orgID *uuid.UUID, epEUI []byte, queID int64, patch storage.DownlinkPatch) (*storage.DownlinkMessage, error) {
	m.calls++
	m.lastOrgID, m.lastEpEUI, m.lastQueID, m.lastPatch = orgID, epEUI, queID, patch
	if m.err != nil {
		return nil, m.err
	}
	return &storage.DownlinkMessage{QueID: queID, EPEUI: pendingTestEpEUI, Status: mioty.DLQueueStatusPending, Priority: patch.Priority, Payload: patch.Payloads[0]}, nil
}

type bucketService struct {
	lastBucket string
	lastWindow grpcservices.ScaciWindow
}

func (b *bucketService) List(_ context.Context, _ int64, bucket string, window grpcservices.ScaciWindow, _, _ int) ([]*grpcservices.ErrorGroup, int64, error) {
	b.lastBucket, b.lastWindow = bucket, window
	if _, ok := map[string]bool{errorgroups.BucketControlPlane: true, errorgroups.BucketBaseStation: true, errorgroups.BucketEndpoint: true, errorgroups.BucketDownlink: true}[bucket]; !ok {
		return nil, 0, errorgroups.ErrInvalidBucket
	}
	if window.From != nil && window.To != nil && window.From.After(*window.To) {
		return nil, 0, errorgroups.ErrInvalidTimeRange
	}
	return []*grpcservices.ErrorGroup{{Bucket: bucket, EventType: "basestation_offline", Count: bucketTestCount}}, 1, nil
}

func TestUpdatePendingDownlink_ValidatesAndPatches(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	msgSvc := &pendingMessageSvc{}
	recorder, emitter := productionRecorder(t)
	svc.useDownlinks(downlinkFakes{messages: msgSvc, audit: recorder})

	resp, err := svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{
		EpEui: pendingTestEpEUI, QueId: pendingTestQueID, Payloads: [][]byte{{0xAA}}, Priority: pendingTestPriority, Format: pendingTestFormat, ResponseExp: true,
	})
	require.NoError(t, err)
	assert.Equal(t, pendingTestQueID, msgSvc.lastQueID)
	assert.Equal(t, pendingTestPriority, msgSvc.lastPatch.Priority)
	assert.Equal(t, uint8(pendingTestFormat), msgSvc.lastPatch.Format)
	assert.True(t, msgSvc.lastPatch.ResponseExp)
	assert.Equal(t, pendingTestQueID, resp.QueId)
	assert.Equal(t, pendingTestPriority, resp.Priority)
	require.Len(t, emitter.events, 1)
	assert.Equal(t, auditTestActor, emitter.events[0].UserID)

	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{QueId: pendingTestQueID})
	assertCode(t, err, grpcerrors.ErrTokenEndpointEUIRequired)

	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{EpEui: pendingTestEpEUI})
	assertCode(t, err, grpcerrors.ErrTokenQueueIDRequired)

	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{EpEui: pendingTestEpEUI, QueId: pendingTestQueID, CntDepend: true})
	assertCode(t, err, grpcerrors.ErrTokenDownlinkPayloadRequired)

	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{EpEui: pendingTestEpEUI, QueId: pendingTestQueID, Payloads: [][]byte{{1}, {2}}})
	assertCode(t, err, grpcerrors.ErrTokenDownlinkPayloadTooLarge)

	msgSvc.err = storage.ErrDownlinkNotPending
	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{EpEui: pendingTestEpEUI, QueId: pendingTestQueID, Payloads: [][]byte{{0xAA}}})
	assertCode(t, err, grpcerrors.ErrTokenDownlinkNotPending)

	msgSvc.err = storage.ErrDownlinkNotFound
	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{EpEui: pendingTestEpEUI, QueId: pendingTestQueID, Payloads: [][]byte{{0xAA}}})
	assertCode(t, err, grpcerrors.ErrTokenDownlinkNotFound)

	// The endpoint is part of the storage predicate, so the handler forwards
	// the parsed EUI and never decides ownership itself.
	msgSvc.err = nil
	callsBefore := msgSvc.calls
	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{EpEui: filterTestBsEUIHex, QueId: pendingTestQueID, Payloads: [][]byte{{0xAA}}})
	require.NoError(t, err)
	wantEUI, parseErr := validation.ParseEUIBytes(filterTestBsEUIHex)
	require.NoError(t, parseErr)
	assert.Equal(t, wantEUI, msgSvc.lastEpEUI)
	assert.Equal(t, callsBefore+1, msgSvc.calls)
	assert.Equal(t, mioty.FormatEUIBytes(wantEUI), emitter.events[len(emitter.events)-1].Details[bssci.EventKeyEpEui])
	assert.Equal(t, pendingTestQueID, emitter.events[len(emitter.events)-1].Details[bssci.EventKeyQueID])

	// A malformed EUI is rejected before storage is consulted.
	callsBefore = msgSvc.calls
	_, err = svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{EpEui: malformedEUI, QueId: pendingTestQueID, Payloads: [][]byte{{0xAA}}})
	assertCode(t, err, grpcerrors.ErrTokenInvalidEUIFormat)
	assert.Equal(t, callsBefore, msgSvc.calls)
}

func TestListErrorGroups_PassesBucketAndMapsErrors(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	buckets := &bucketService{}
	svc.errorGroups = buckets

	resp, err := svc.ListErrorGroups(ownerCtx(), &pb.ListErrorGroupsRequest{Bucket: errorgroups.BucketBaseStation})
	require.NoError(t, err)
	require.Len(t, resp.Groups, 1)
	assert.Equal(t, errorgroups.BucketBaseStation, resp.Groups[0].Bucket)
	assert.Equal(t, int32(1), resp.TotalCount)
	assert.Equal(t, errorgroups.BucketBaseStation, buckets.lastBucket)

	_, err = svc.ListErrorGroups(ownerCtx(), &pb.ListErrorGroupsRequest{Bucket: "everything"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidErrorBucket)

	later := timestamppb.New(timestamppb.New(timestamppb.Now().AsTime()).AsTime())
	earlier := timestamppb.New(later.AsTime().Add(-1))
	_, err = svc.ListErrorGroups(ownerCtx(), &pb.ListErrorGroupsRequest{Bucket: errorgroups.BucketEndpoint, StartTime: later, EndTime: earlier})
	assertCode(t, err, grpcerrors.ErrTokenInvalidTimeRange)

	_, err = svc.ListErrorGroups(ownerCtx(), &pb.ListErrorGroupsRequest{Bucket: errorgroups.BucketEndpoint, PageToken: "not-a-token"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidPageToken)

	svc.errorGroups = nil
	_, err = svc.ListErrorGroups(ownerCtx(), &pb.ListErrorGroupsRequest{Bucket: errorgroups.BucketEndpoint})
	assertCode(t, err, grpcerrors.ErrTokenServiceNotConfigured)
}

func TestListCapabilities_ReturnsInjectedAllowlist(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	svc.capabilities = []grpcservices.Capability{{Name: capabilityTestName, Enabled: capabilityTestOn}}

	resp, err := svc.ListCapabilities(ownerCtx(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, resp.Capabilities, 1)
	assert.Equal(t, capabilityTestName, resp.Capabilities[0].Name)
	assert.True(t, resp.Capabilities[0].Enabled)
}

const (
	diagnosticsTestFilename = "bundle.zip"
	diagnosticsTestType     = "application/zip"
)

var diagnosticsTestArchive = []byte{0x50, 0x4b, 0x05, 0x06}

// fakeDiagnostics records the tenant it was asked for and answers with the configured bundle or error.
type fakeDiagnostics struct {
	lastTenant int64
	err        error
}

func (f *fakeDiagnostics) Build(_ context.Context, tenantID int64) (*diagnostics.Bundle, error) {
	f.lastTenant = tenantID
	if f.err != nil {
		return nil, f.err
	}
	return &diagnostics.Bundle{Archive: diagnosticsTestArchive, Filename: diagnosticsTestFilename, ContentType: diagnosticsTestType, GeneratedAt: time.Unix(pendingTestFrom, 0).UTC()}, nil
}

func adminCtx() context.Context { return pkgcontext.WithUserID(ownerCtx(), auditTestActor) }

func TestGetDiagnosticsBundle_ReturnsTheCallersBundle(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	builder := &fakeDiagnostics{}
	svc.diagnostics = builder

	resp, err := svc.GetDiagnosticsBundle(adminCtx(), &pb.GetDiagnosticsBundleRequest{})
	require.NoError(t, err)
	assert.Equal(t, retainedOwnerTenant, builder.lastTenant, "the bundle is built for the caller's tenant")
	assert.Equal(t, diagnosticsTestArchive, resp.Archive)
	assert.Equal(t, diagnosticsTestFilename, resp.Filename)
	assert.Equal(t, diagnosticsTestType, resp.ContentType)
	assert.Equal(t, int64(len(diagnosticsTestArchive)), resp.SizeBytes)
	assert.Equal(t, pendingTestFrom, resp.GeneratedAt.AsTime().Unix())
}

func TestGetDiagnosticsBundle_MapsSizeLimitAndFailures(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})

	svc.diagnostics = &fakeDiagnostics{err: diagnostics.ErrBundleTooLarge}
	_, err := svc.GetDiagnosticsBundle(adminCtx(), &pb.GetDiagnosticsBundleRequest{})
	assertCode(t, err, grpcerrors.ErrTokenDiagnosticsTooLarge)

	svc.diagnostics = &fakeDiagnostics{err: errAuditStore}
	_, err = svc.GetDiagnosticsBundle(adminCtx(), &pb.GetDiagnosticsBundleRequest{})
	assertCode(t, err, grpcerrors.ErrTokenDiagnosticsFailed)

	svc.diagnostics = nil
	_, err = svc.GetDiagnosticsBundle(adminCtx(), &pb.GetDiagnosticsBundleRequest{})
	assertCode(t, err, grpcerrors.ErrTokenServiceNotConfigured)
}
