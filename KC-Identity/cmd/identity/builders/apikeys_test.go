package builders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	identitygrpc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	testKeyTenant = int64(1)
	testKeyName   = "gateway uplink key"
)

var (
	testKeyOrg   = uuid.MustParse("4c310e39-b828-4fa4-aeba-26e9a40471de")
	testKeyAdmin = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

// serverAdmins knows one user, a server admin.
type serverAdmins struct{ grpcservices.AdminUserService }

func (serverAdmins) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	return &models.User{ID: id, IsAdmin: true, IsActive: true}, nil
}

// storedKeys keeps created keys in memory; it serves the admin service and,
// as the key repository, the validation the gateway asks for.
type storedKeys struct {
	interfaces.APIKeyRepository
	keys map[uuid.UUID]*models.APIKey
}

func (s *storedKeys) GetByHash(_ context.Context, hash string) (*models.APIKey, error) {
	for _, key := range s.keys {
		if key.KeyHash == hash {
			return key, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (s *storedKeys) UpdateLastUsed(context.Context, uuid.UUID) error { return nil }

func (s *storedKeys) Create(_ context.Context, key *models.APIKey) error {
	s.keys[key.ID] = key
	return nil
}

func (s *storedKeys) List(_ context.Context, _ int64, orgID uuid.UUID, _ *uuid.UUID, _, _ int) ([]*models.APIKey, error) {
	var keys []*models.APIKey
	for _, key := range s.keys {
		if key.OrgID == orgID {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *storedKeys) Count(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID) (int64, error) {
	keys, err := s.List(ctx, tenantID, orgID, userID, 0, 0)
	return int64(len(keys)), err
}

func (s *storedKeys) GetByIDAndOrg(_ context.Context, id, _ uuid.UUID) (*models.APIKey, error) {
	return s.keys[id], nil
}

func (s *storedKeys) DeleteByIDAndOrg(_ context.Context, id, _ uuid.UUID) error {
	delete(s.keys, id)
	return nil
}

// discardAudit stands in for the audit recorder; these tests do not inspect the trail.
type discardAudit struct{}

func (discardAudit) Record(context.Context, audit.Event) {}

func (discardAudit) RecordRequired(context.Context, audit.Event) error { return nil }

// communityKeyService builds the identity service the way the community
// composition root does for API keys, over the given key store.
func communityKeyService(t *testing.T, keys *storedKeys) *identitygrpc.IdentityService {
	t.Helper()
	svc, err := identitygrpc.NewIdentityService(logger.NewNop(), discardAudit{}, discardAudit{})
	require.NoError(t, err)
	return withAPIKeyAdministration(svc.WithAdminUserService(serverAdmins{}), keys, oneOrganization{}, logger.NewNop())
}

// oneOrganization is the installation's default organization.
type oneOrganization struct{}

func (oneOrganization) GetByIDUnscoped(_ context.Context, id uuid.UUID) (*models.Organization, error) {
	return &models.Organization{OrgID: id, TenantID: testKeyTenant}, nil
}

func adminCall() context.Context {
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), testKeyTenant)
	ctx = pkgcontext.WithOrganizationID(ctx, testKeyOrg)
	return pkgcontext.WithUserID(ctx, testKeyAdmin.String())
}

// The community edition has no organization administration, yet its server
// admin creates, lists and revokes API keys of the default organization.
func TestCommunityComposition_ManagesAPIKeys(t *testing.T) {
	keys := &storedKeys{keys: map[uuid.UUID]*models.APIKey{}}
	svc := communityKeyService(t, keys)

	created, err := svc.CreateApiKey(adminCall(), &pb.CreateApiKeyRequest{Name: testKeyName, KeyType: models.KeyTypeUser})
	require.NoError(t, err)
	assert.NotEmpty(t, created.GetRawKey())

	listed, err := svc.ListApiKeys(adminCall(), &pb.ListApiKeysRequest{})
	require.NoError(t, err)
	require.Len(t, listed.GetApiKeys(), 1)
	assert.Equal(t, testKeyName, listed.GetApiKeys()[0].GetName())

	_, err = svc.DeleteApiKey(adminCall(), &pb.DeleteApiKeyRequest{Id: created.GetApiKey().GetId()})
	require.NoError(t, err)
	assert.Empty(t, keys.keys)
}

// A delete without tenant context is refused before the key is touched, so
// its audit event can never be recorded against an unknown tenant.
func TestDeleteApiKey_WithoutTenantContextKeepsTheKey(t *testing.T) {
	keys := &storedKeys{keys: map[uuid.UUID]*models.APIKey{}}
	svc := communityKeyService(t, keys)
	created, err := svc.CreateApiKey(adminCall(), &pb.CreateApiKeyRequest{Name: testKeyName, KeyType: models.KeyTypeUser})
	require.NoError(t, err)

	noTenant := pkgcontext.WithUserID(pkgcontext.WithOrganizationID(testutil.TestContext(), testKeyOrg), testKeyAdmin.String())
	_, err = svc.DeleteApiKey(noTenant, &pb.DeleteApiKeyRequest{Id: created.GetApiKey().GetId()})

	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingTenantCtx), status.Code(err))
	assert.Len(t, keys.keys, 1)
}

var errEventStoreDown = errors.New("event store down")

// failingEventStore refuses every event write, as an unavailable database does.
type failingEventStore struct{}

func (failingEventStore) CreateEvent(context.Context, *models.SystemEvent) error {
	return errEventStoreDown
}

// capturedEventStore keeps every event written to it.
type capturedEventStore struct{ events []*models.SystemEvent }

func (s *capturedEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	s.events = append(s.events, event)
	return nil
}

// storeBackedKeyService administers keys with every audit event written to store.
func storeBackedKeyService(t *testing.T, store audit.EventWriter) *identitygrpc.IdentityService {
	t.Helper()
	emitter, err := audit.NewEmitter(store, clock.SystemClock{})
	require.NoError(t, err)
	recorder, err := audit.NewRecorder(emitter, logger.NewNop(), discardDrops{})
	require.NoError(t, err)
	svc, err := identitygrpc.NewIdentityService(logger.NewNop(), recorder, recorder)
	require.NoError(t, err)
	keys := &storedKeys{keys: map[uuid.UUID]*models.APIKey{}}
	return withAPIKeyAdministration(svc.WithAdminUserService(serverAdmins{}), keys, oneOrganization{}, logger.NewNop())
}

func TestCreateApiKey_RecordsWhoCreatedTheKeyNeverTheKey(t *testing.T) {
	store := &capturedEventStore{}
	svc := storeBackedKeyService(t, store)

	created, err := svc.CreateApiKey(adminCall(), &pb.CreateApiKeyRequest{Name: testKeyName, KeyType: models.KeyTypeServiceAccount})

	require.NoError(t, err)
	require.NotEmpty(t, created.GetRawKey())
	require.Len(t, store.events, 1)
	event := store.events[0]
	assert.Equal(t, models.EventTypeAPIKeyCreated, event.EventType)
	assert.Equal(t, testKeyAdmin.String(), event.UserID, "the event names who created the key")
	assert.Contains(t, string(event.Details), created.GetApiKey().GetId())
	assert.NotContains(t, string(event.Details)+event.Description, created.GetRawKey(), "the raw key never enters the event")
}

// A raw key is shown once; when its creation cannot be recorded it is not shown at all.
func TestCreateApiKey_WithholdsAKeyWhoseCreationCannotBeRecorded(t *testing.T) {
	svc := storeBackedKeyService(t, failingEventStore{})

	created, err := svc.CreateApiKey(adminCall(), &pb.CreateApiKeyRequest{Name: testKeyName, KeyType: models.KeyTypeServiceAccount})

	assert.Nil(t, created, "no raw key leaves without its audit record")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), status.Code(err))
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError), status.Convert(err).Message())
}

// discardDrops stands in for the drop counter.
type discardDrops struct{}

func (discardDrops) Inc(string) {}

// gatewayHash is the hash the gateway's auth interceptor looks a presented key up by.
func gatewayHash(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// A key the community edition created validates for the gateway with the
// organization's tenant and its owner, reads as expired once its day has
// passed, and is unknown once deleted.
func TestCommunityAPIKey_ValidatesForTheGatewayUntilExpiredOrDeleted(t *testing.T) {
	keys := &storedKeys{keys: map[uuid.UUID]*models.APIKey{}}
	svc := communityKeyService(t, keys)
	validator := identitygrpc.NewIdentityInternalService(nil, newAPIKeyLookupAdapter(keys), nil, nil, nil, logger.NewNop())

	created, err := svc.CreateApiKey(adminCall(), &pb.CreateApiKeyRequest{Name: testKeyName, KeyType: models.KeyTypeUser})
	require.NoError(t, err)
	check := &pb.ValidateAPIKeyRequest{KeyHash: gatewayHash(created.GetRawKey())}

	valid, err := validator.ValidateAPIKey(testutil.TestContext(), check)
	require.NoError(t, err)
	assert.True(t, valid.GetIsActive())
	assert.False(t, valid.GetIsExpired())
	assert.Equal(t, testKeyTenant, valid.GetTenantId())
	assert.Equal(t, testKeyOrg.String(), valid.GetOrganizationId())
	assert.Equal(t, testKeyAdmin.String(), valid.GetUserId())

	past := time.Now().UTC().Add(-time.Minute)
	keys.keys[uuid.MustParse(created.GetApiKey().GetId())].ExpiresAt = &past
	expired, err := validator.ValidateAPIKey(testutil.TestContext(), check)
	require.NoError(t, err)
	assert.True(t, expired.GetIsExpired())

	_, err = svc.DeleteApiKey(adminCall(), &pb.DeleteApiKeyRequest{Id: created.GetApiKey().GetId()})
	require.NoError(t, err)
	_, err = validator.ValidateAPIKey(testutil.TestContext(), check)
	assert.Equal(t, codes.NotFound, status.Code(err))
}
