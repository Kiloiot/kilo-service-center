package interceptors

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	discardTestTenantClaim  = "tenant"
	discardTestAuthTenant   = int64(1)
	discardTestHeaderTenant = int64(2)
	discardTestOpaqueToken  = "kc_discard_test_key"
)

var (
	errDiscardTestStore = errors.New("store unavailable")
	errDiscardTestAdmin = errors.New("admin directory unavailable")
)

func tenantClaimToken(t *testing.T, claim string) jwt.Token {
	t.Helper()
	token := jwt.New()
	require.NoError(t, token.Set(discardTestTenantClaim, claim))
	return token
}

// A tenant claim is a decimal number or nothing: trailing characters must not
// yield the tenant their numeric prefix names.
func TestExtractTenantFromClaims_NonDecimalTenantResolvesToZero(t *testing.T) {
	assert.Equal(t, int64(42), extractTenantFromClaims(tenantClaimToken(t, "42"), discardTestTenantClaim))
	assert.Zero(t, extractTenantFromClaims(tenantClaimToken(t, "7x"), discardTestTenantClaim))
	assert.Zero(t, extractTenantFromClaims(tenantClaimToken(t, "tenant-7"), discardTestTenantClaim))
}

type lastUsedFailingAPIKeys struct{}

func (lastUsedFailingAPIKeys) LookupByHash(context.Context, string) (*APIKeyRecord, error) {
	return &APIKeyRecord{ID: uuid.New(), TenantID: discardTestAuthTenant, OrganizationID: uuid.New(), IsActive: true}, nil
}

func (lastUsedFailingAPIKeys) UpdateLastUsed(context.Context, uuid.UUID) error {
	return errDiscardTestStore
}

// A valid API key is admitted when its last-used time cannot be stored, and
// the failed update is logged.
func TestAuthenticateAPIKey_LastUsedFailureIsLoggedAndAdmitted(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	ai, err := NewAuthInterceptor(AuthConfig{Enabled: testAuthEnabled, HMACSecret: testHMACSecret, Logger: log})
	require.NoError(t, err)
	ai.WithAPIKeyAuthenticator(lastUsedFailingAPIKeys{})
	ctx := metadata.NewIncomingContext(testutil.TestContext(),
		metadata.Pairs(grpcconst.MetadataKeyAuthorization, grpcconst.BearerPrefix+discardTestOpaqueToken))

	authed, err := ai.authenticate(ctx, testNonExemptMethod)

	require.NoError(t, err)
	tenantID, err := pkgcontext.GetTenantID(authed)
	require.NoError(t, err)
	assert.Equal(t, discardTestAuthTenant, tenantID)
	logged := log.FilterMessage(LogAuthAPIKeyLastUsedUpdateFailed)
	require.Len(t, logged, 1)
	assert.Equal(t, errDiscardTestStore, logged[0].FieldMap()[logger.FieldError])
}

type fixedTenantResolver struct{ tenantID int64 }

func (r fixedTenantResolver) LookupTenant(context.Context, uuid.UUID) (int64, error) {
	return r.tenantID, nil
}

func (r fixedTenantResolver) ResolveCert(context.Context, *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, r.tenantID, nil
}

func (r fixedTenantResolver) GetDefaultOrgForTenant(context.Context, int64) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// failingAdminChecker claims admin while reporting a failure, so only the
// error decides the outcome.
type failingAdminChecker struct{}

func (failingAdminChecker) IsServerAdmin(context.Context, string) (bool, error) {
	return true, errDiscardTestAdmin
}

// An admin check that fails grants no cross-tenant access, and the failure
// is logged.
func TestOrgResolver_FailedAdminCheckGrantsNoCrossTenantAccess(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	oi, err := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver:     fixedTenantResolver{tenantID: discardTestHeaderTenant},
		Logger:       log,
		AdminChecker: failingAdminChecker{},
	})
	require.NoError(t, err)
	userID := uuid.New().String()
	ctx := pkgcontext.WithUserID(pkgcontext.WithTenantID(testutil.TestContext(), discardTestAuthTenant), userID)
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(
		grpcconst.MetadataKeyOrganizationID, uuid.New().String(),
		grpcconst.MetadataKeyUserID, userID))

	_, err = oi.resolveOrgContext(ctx, testNonExemptMethod)

	assert.Equal(t, codes.PermissionDenied, status.Code(err), "err: %v", err)
	logged := log.FilterMessage(LogGrpcOrgInterceptorAdminCheckFailed)
	require.Len(t, logged, 1)
	assert.Equal(t, errDiscardTestAdmin, logged[0].FieldMap()[logger.FieldError])
}

type failingEventWriter struct{}

func (failingEventWriter) CreateEvent(context.Context, *models.SystemEvent) error {
	return errDiscardTestStore
}

// A refusal whose security event cannot be stored is logged.
func TestRecordSecurityEvent_WriteFailureIsLogged(t *testing.T) {
	log := bsscitest.NewRecordingLogger()

	recordSecurityEvent(testutil.TestContext(), failingEventWriter{}, discardTestAuthTenant, log, securityEvent{
		method: testNonExemptMethod, eventType: models.EventTypeAuthInvalidToken,
	})

	logged := log.FilterMessage(LogSecurityEventRecordFailed)
	require.Len(t, logged, 1)
	fields := logged[0].FieldMap()
	assert.Equal(t, models.EventTypeAuthInvalidToken, fields[logger.FieldEventType])
	assert.Equal(t, errDiscardTestStore, fields[logger.FieldError])
}

type failingOrgResolver struct{}

func (failingOrgResolver) ResolveOrganization(context.Context, int64) (uuid.UUID, error) {
	return uuid.Nil, errDiscardTestStore
}

// A token whose tenant has no resolvable organization is admitted without
// organization context, and the failed lookup is logged.
func TestAuthenticateJWT_OrgResolutionFailureIsLogged(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	ai, err := NewAuthInterceptor(AuthConfig{
		Enabled: testAuthEnabled, HMACSecret: testHMACSecret, TenantClaim: discardTestTenantClaim, Logger: log,
	})
	require.NoError(t, err)
	ai.WithOrganizationResolver(failingOrgResolver{})
	signed, err := jwt.Sign(tenantClaimToken(t, "1"), jwt.WithKey(jwa.HS256, []byte(testHMACSecret)))
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(testutil.TestContext(),
		metadata.Pairs(grpcconst.MetadataKeyAuthorization, grpcconst.BearerPrefix+string(signed)))

	authed, err := ai.authenticate(ctx, testNonExemptMethod)

	require.NoError(t, err)
	_, orgErr := pkgcontext.GetOrganizationID(authed)
	assert.Error(t, orgErr)
	logged := log.FilterMessage(LogAuthOrganizationResolutionFailed)
	require.Len(t, logged, 1)
	assert.Equal(t, errDiscardTestStore, logged[0].FieldMap()[logger.FieldError])
}
