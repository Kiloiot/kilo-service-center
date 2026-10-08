package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

var revealAppKey = []byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF, 0xB0}

// keyedEndpointSvc holds one endpoint of the owner tenant with the given keys.
func keyedEndpointSvc(appKey []byte) *mockEndpointSvcIsolation {
	return &mockEndpointSvcIsolation{
		getByEUIFunc: func(_ context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
			if tenantID != matrixOwnerTenant {
				return nil, storage.ErrNotFound
			}
			ep := &models.EndPoint{ID: 7, TenantID: tenantID, NwkSnKey: matrixNwkSnKey, AppKey: appKey}
			copy(ep.EUI[:], eui)
			return ep, nil
		},
	}
}

func revealCtx(tenantID int64) context.Context {
	return pkgcontext.WithUserID(testutil.TestContextWithTenant(tenantID), auditTestActor)
}

func TestEndpointReads_ReturnTheKeysMasked(t *testing.T) {
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), log: &mockLogger{}})

	ep, err := svc.GetEndPoint(revealCtx(matrixOwnerTenant), &pb.GetEndPointRequest{EpEui: matrixEndpointHex})

	require.NoError(t, err)
	assert.Empty(t, ep.GetNwkSnKey())
	assert.Empty(t, ep.GetAppKey())
	assert.True(t, ep.GetNwkSnKeySet(), "whether a key is set is known without its value")
	assert.True(t, ep.GetAppKeySet())

	unset := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(nil), log: &mockLogger{}})
	ep, err = unset.GetEndPoint(revealCtx(matrixOwnerTenant), &pb.GetEndPointRequest{EpEui: matrixEndpointHex})
	require.NoError(t, err)
	assert.False(t, ep.GetAppKeySet())
}

func TestEndpointKeyReveal_RecordsWhoRevealedWhichKey(t *testing.T) {
	recorder, capture := productionRecorder(t)
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: &mockLogger{}})
	both := []pb.EndpointKey{pb.EndpointKey_ENDPOINT_KEY_NWK_SN_KEY, pb.EndpointKey_ENDPOINT_KEY_APP_KEY}

	ep, err := svc.GetEndPoint(revealCtx(matrixOwnerTenant), &pb.GetEndPointRequest{EpEui: matrixEndpointHex, RevealKeys: both})

	require.NoError(t, err)
	assert.Equal(t, matrixNwkSnKey, ep.GetNwkSnKey())
	assert.Equal(t, revealAppKey, ep.GetAppKey())
	require.Len(t, capture.events, 1)
	event := capture.events[0]
	assert.Equal(t, models.EventTypeEndpointKeysRevealed, event.EventType)
	assert.Equal(t, auditTestActor, event.UserID, "the event names who revealed the keys")
	assert.Equal(t, matrixEndpointHex, event.Details[models.EventDetailKeyEpEui])
	assert.Equal(t, []string{fieldMaskNwkSnKey, fieldMaskAppKey}, event.Details[models.EventDetailKeyRevealedKeys])
	serialized := fmt.Sprintf("%v %s", event.Details, event.Description)
	for _, key := range [][]byte{matrixNwkSnKey, revealAppKey} {
		assert.NotContains(t, serialized, fmt.Sprintf("%x", key), "a revealed key never enters the event")
		assert.NotContains(t, serialized, fmt.Sprintf("%v", key))
	}
}

func TestEndpointKeyReveal_OnlyStoredKeysAreRevealed(t *testing.T) {
	recorder, capture := productionRecorder(t)
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(nil), audit: recorder, keyReveals: recorder, log: &mockLogger{}})

	ep, err := svc.GetEndPoint(revealCtx(matrixOwnerTenant), &pb.GetEndPointRequest{
		EpEui: matrixEndpointHex, RevealKeys: []pb.EndpointKey{pb.EndpointKey_ENDPOINT_KEY_APP_KEY},
	})

	require.NoError(t, err)
	assert.Empty(t, ep.GetAppKey())
	assert.Empty(t, capture.events, "nothing was revealed, so nothing is recorded")
}

func TestEndpointKeyReveal_RejectsAnUnknownKey(t *testing.T) {
	recorder, capture := productionRecorder(t)
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: &mockLogger{}})

	_, err := svc.GetEndPoint(revealCtx(matrixOwnerTenant), &pb.GetEndPointRequest{
		EpEui: matrixEndpointHex, RevealKeys: []pb.EndpointKey{pb.EndpointKey_ENDPOINT_KEY_NWK_SN_KEY, pb.EndpointKey_ENDPOINT_KEY_UNSPECIFIED},
	})

	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointKeyReveal), st.Message())
	assert.Empty(t, capture.events)
}

func TestEndpointKeyReveal_AnotherTenantsEndpointIsNotFound(t *testing.T) {
	recorder, capture := productionRecorder(t)
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: &mockLogger{}})

	ep, err := callThrough(revealCtx(matrixOtherTenant), authz.Roles{EndpointManager: true}, pb.CoreService_GetEndPoint_FullMethodName,
		func(ctx context.Context, _ interface{}) (interface{}, error) {
			return svc.GetEndPoint(ctx, &pb.GetEndPointRequest{EpEui: matrixEndpointHex, RevealKeys: revealNetworkKey})
		})

	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Nil(t, ep)
	assert.Empty(t, capture.events)
}

func TestEndpointKeyReveal_NeedsAnEndpointManager(t *testing.T) {
	for _, role := range matrixRoles {
		t.Run(role.name, func(t *testing.T) {
			recorder, capture := productionRecorder(t)
			svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: &mockLogger{}})
			ctx := revealCtx(matrixOwnerTenant)
			if role.roles == authz.ServiceAccountRoles {
				ctx = pkgcontext.WithServiceAccountID(testutil.TestContextWithTenant(matrixOwnerTenant), revealServiceAccount)
			}

			ep, err := callThrough(ctx, role.roles, pb.CoreService_GetEndPoint_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
				return svc.GetEndPoint(ctx, &pb.GetEndPointRequest{EpEui: matrixEndpointHex, RevealKeys: revealNetworkKey})
			})

			if !role.roles.Admin && !role.roles.EndpointManager {
				assert.Equal(t, codes.PermissionDenied, status.Code(err))
				assert.Nil(t, ep)
				assert.Empty(t, capture.events)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, matrixNwkSnKey, ep.(*pb.EndPoint).GetNwkSnKey())
			require.Len(t, capture.events, 1)
			if role.roles == authz.ServiceAccountRoles {
				assert.Equal(t, revealServiceAccount.String(), capture.events[0].Details[models.EventDetailKeyServiceAccount],
					"a service-account key is named as the actor")
			}
		})
	}
}

var revealServiceAccount = uuid.MustParse("5b1f0c1e-3f4e-4d1a-9f2b-7c0d6e8a9b10")

func keyUpdate(sent *pb.EndPoint, paths ...string) *pb.UpdateEndPointRequest {
	sent.EpEui = matrixEndpointHex
	return &pb.UpdateEndPointRequest{Endpoint: sent, UpdateMask: &fieldmaskpb.FieldMask{Paths: paths}}
}

func keysRemovedEvents(events []audit.Event) []audit.Event {
	var removed []audit.Event
	for _, ev := range events {
		if ev.EventType == models.EventTypeEndpointKeysRemoved {
			removed = append(removed, ev)
		}
	}
	return removed
}

func TestUpdateEndPoint_RecordsWhoRemovedAStoredKey(t *testing.T) {
	recorder, capture := productionRecorder(t)
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: &mockLogger{}})

	ep, err := svc.UpdateEndPoint(revealCtx(matrixOwnerTenant), keyUpdate(&pb.EndPoint{}, fieldMaskAppKey))

	require.NoError(t, err)
	assert.False(t, ep.GetAppKeySet())
	assert.True(t, ep.GetNwkSnKeySet())
	removed := keysRemovedEvents(capture.events)
	require.Len(t, removed, 1)
	event := removed[0]
	assert.Equal(t, models.EventCategoryAudit, event.Category, "readable by administrators only")
	assert.Equal(t, auditTestActor, event.UserID, "the event names who removed the key")
	assert.Equal(t, matrixEndpointHex, event.Details[models.EventDetailKeyEpEui])
	assert.Equal(t, []string{fieldMaskAppKey}, event.Details[models.EventDetailKeyRemovedKeys])
	serialized := fmt.Sprintf("%v %s", event.Details, event.Description)
	assert.NotContains(t, serialized, fmt.Sprintf("%x", revealAppKey), "a removed key never enters the event")
	assert.NotContains(t, serialized, fmt.Sprintf("%v", revealAppKey))
}

func TestUpdateEndPoint_RecordsNoRemovalWhenNoStoredKeyIsCleared(t *testing.T) {
	replacement := []byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF, 0xD0}
	cases := []struct {
		name   string
		stored []byte
		req    *pb.UpdateEndPointRequest
	}{
		{"application key replaced", revealAppKey, keyUpdate(&pb.EndPoint{AppKey: replacement}, fieldMaskAppKey)},
		{"network key replaced", revealAppKey, keyUpdate(&pb.EndPoint{NwkSnKey: replacement}, fieldMaskNwkSnKey)},
		{"no application key to clear", nil, keyUpdate(&pb.EndPoint{}, fieldMaskAppKey)},
		{"keys untouched", revealAppKey, keyUpdate(&pb.EndPoint{Name: "renamed"}, fieldMaskName)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder, capture := productionRecorder(t)
			svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(tc.stored), audit: recorder, keyReveals: recorder, log: &mockLogger{}})

			_, err := svc.UpdateEndPoint(revealCtx(matrixOwnerTenant), tc.req)

			require.NoError(t, err)
			assert.Empty(t, keysRemovedEvents(capture.events))
		})
	}
}

func TestUpdateEndPoint_RefusesToRemoveTheNetworkKey(t *testing.T) {
	recorder, capture := productionRecorder(t)
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: &mockLogger{}})

	_, err := svc.UpdateEndPoint(revealCtx(matrixOwnerTenant), keyUpdate(&pb.EndPoint{}, fieldMaskNwkSnKey))

	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Empty(t, capture.events)
}

// failingEventStore refuses every event write, as an unavailable database does.
type failingEventStore struct{}

func (failingEventStore) CreateEvent(context.Context, *models.SystemEvent) error {
	return errAuditStore
}

// storeBackedRecorder records through the production emitter into store, logging to log.
func storeBackedRecorder(t *testing.T, store audit.EventWriter, log logger.Logger) *audit.Recorder {
	t.Helper()
	emitter, err := audit.NewEmitter(store, clock.SystemClock{})
	require.NoError(t, err)
	recorder, err := audit.NewRecorder(emitter, log, discardDrops{})
	require.NoError(t, err)
	return recorder
}

func TestEndpointKeyReveal_IsRefusedWhenItCannotBeRecorded(t *testing.T) {
	log := newCapturingLogger()
	recorder := storeBackedRecorder(t, failingEventStore{}, log)
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: log})
	both := []pb.EndpointKey{pb.EndpointKey_ENDPOINT_KEY_NWK_SN_KEY, pb.EndpointKey_ENDPOINT_KEY_APP_KEY}

	ep, err := svc.GetEndPoint(revealCtx(matrixOwnerTenant), &pb.GetEndPointRequest{EpEui: matrixEndpointHex, RevealKeys: both})

	assert.Nil(t, ep, "no key leaves without its audit record")
	st := status.Convert(err)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError), st.Message())
	written := fmt.Sprintf("%s %v", st.Message(), log.entries)
	for _, key := range [][]byte{matrixNwkSnKey, revealAppKey} {
		assert.NotContains(t, written, fmt.Sprintf("%x", key), "a withheld key never enters the error or the logs")
		assert.NotContains(t, written, fmt.Sprintf("%X", key))
		assert.NotContains(t, written, fmt.Sprintf("%v", key))
	}
}

func TestNewEndpointHandlers_RequiresAKeyRevealRecorder(t *testing.T) {
	_, err := NewEndpointHandlers(EndpointHandlerDeps{Endpoints: &fakeEndpointSvc{}, Attachment: &mockEndpointAttachmentSvc{}, Clock: clock.SystemClock{}}, &captureAuditRecorder{}, &mockLogger{})

	require.ErrorIs(t, err, audit.ErrNilRecorder, "handlers that could reveal a key unrecorded are not built")
}

func TestEndpointMaskedRead_DoesNotNeedTheAuditStore(t *testing.T) {
	recorder := storeBackedRecorder(t, failingEventStore{}, logger.NewNop())
	svc := testCoreService(coreFields{endpointSvc: keyedEndpointSvc(revealAppKey), audit: recorder, keyReveals: recorder, log: &mockLogger{}})

	ep, err := svc.GetEndPoint(revealCtx(matrixOwnerTenant), &pb.GetEndPointRequest{EpEui: matrixEndpointHex})

	require.NoError(t, err, "an ordinary read reveals nothing, so it records nothing")
	assert.Empty(t, ep.GetNwkSnKey())
	assert.Empty(t, ep.GetAppKey())
	assert.True(t, ep.GetNwkSnKeySet())
}
