package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	testAdminHomeTenant   = int64(42)
	testTargetOrgTenant   = int64(100)
	testMembershipChanges = 3
)

type captureAudit struct{ events []audit.Event }

func (c *captureAudit) EmitAudit(_ context.Context, ev audit.Event) error {
	c.events = append(c.events, ev)
	return nil
}

// discardAudit stands in for the recorder where a test does not inspect the audit trail.
type discardAudit struct{}

func (discardAudit) Record(context.Context, audit.Event) {}

type discardDrops struct{}

func (discardDrops) Inc(string) {}

// A server admin acting on another tenant's organization: every membership
// audit event belongs to the organization's tenant, not the admin's own.
func TestMembershipAudit_FiledUnderTheTargetOrganizationsTenant(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	emitter := &captureAudit{}
	recorder, err := audit.NewRecorder(emitter, &mockLogger{}, discardDrops{})
	require.NoError(t, err)
	svc := &IdentityService{
		roles:        fixedRoles(authz.AllRoles),
		adminUserSvc: &mockAdminUserService{getByIDFunc: adminGetByIDFunc(callerID)},
		orgSvc: &mockOrgService{getByIDUnscopedFunc: func(_ context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{OrgID: id, TenantID: testTargetOrgTenant}, nil
		}},
		membershipSvc: &mockMembershipService{getMembershipFunc: func(context.Context, uuid.UUID, uuid.UUID) (*grpcservices.OrganizationMember, error) {
			return &grpcservices.OrganizationMember{Role: models.OrganizationRoleMember}, nil
		}},
		audit: recorder,
		log:   &mockLogger{},
	}
	ctx := pkgcontext.WithUserID(contextForTenant(testAdminHomeTenant), callerID.String())

	_, err = svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId: orgID.String(), UserId: uuid.New().String(), Role: models.OrganizationRoleMember,
	})
	require.NoError(t, err)
	_, err = svc.UpdateOrganizationUser(ctx, &pb.UpdateOrganizationUserRequest{
		OrgId: orgID.String(), UserId: uuid.New().String(), Role: models.OrganizationRoleMember,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{orgUserFieldRole}},
	})
	require.NoError(t, err)
	_, err = svc.RemoveOrganizationUser(ctx, &pb.RemoveOrganizationUserRequest{OrgId: orgID.String(), UserId: uuid.New().String()})
	require.NoError(t, err)

	require.Len(t, emitter.events, testMembershipChanges)
	for _, ev := range emitter.events {
		assert.Equal(t, testTargetOrgTenant, ev.TenantID, "%s is filed under the organization's tenant", ev.EventType)
		assert.Equal(t, callerID.String(), ev.UserID, "the acting admin stays on the record")
	}
}

// User accounts are server-wide: their lifecycle is filed under the platform
// tenant whether or not the call carries a tenant, and names the acting admin.
func TestUserAudit_FiledUnderThePlatformTenant(t *testing.T) {
	const platformTenant = int64(1)
	callerID := uuid.New()
	subjectID := uuid.New()
	emitter := &captureAudit{}
	recorder, err := audit.NewRecorder(emitter, &mockLogger{}, discardDrops{})
	require.NoError(t, err)
	getByID := adminGetByIDFunc(callerID)
	base, err := NewIdentityService(&mockLogger{}, recorder, recorder)
	require.NoError(t, err)
	svc := base.
		WithAdminUserService(&mockAdminUserService{
			getByIDFunc: getByID,
			createFunc: func(context.Context, *grpcservices.UserCreateRequest) (*models.User, error) {
				return &models.User{ID: subjectID, Email: "subject@example.com"}, nil
			},
			updateFunc: func(context.Context, uuid.UUID, *grpcservices.UserUpdateRequest) (*models.User, error) {
				return &models.User{ID: subjectID, Email: "subject@example.com"}, nil
			},
		}).
		WithPlatformTenantID(platformTenant)
	withoutTenant := pkgcontext.WithUserID(testutil.TestContext(), callerID.String())

	_, err = svc.CreateUser(withoutTenant, &pb.CreateUserRequest{Email: "subject@example.com", Password: "secret-password"})
	require.NoError(t, err)
	_, err = svc.UpdateUser(pkgcontext.WithUserID(contextForTenant(testAdminHomeTenant), callerID.String()),
		&pb.UpdateUserRequest{Id: subjectID.String(), Note: "renamed", UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{userFieldNote}}})
	require.NoError(t, err)
	_, err = svc.DeleteUser(withoutTenant, &pb.DeleteUserRequest{Id: subjectID.String()})
	require.NoError(t, err)

	require.Len(t, emitter.events, 3, "every user change is on the audit trail")
	for _, ev := range emitter.events {
		assert.Equal(t, platformTenant, ev.TenantID, "%s is a server-level event", ev.EventType)
		assert.Equal(t, callerID.String(), ev.UserID, "%s names the acting admin", ev.EventType)
		assert.Equal(t, subjectID.String(), ev.Details[auditKeyUserID], "%s names the user it changed", ev.EventType)
	}
}

func TestNewIdentityService_RequiresAnAuditRecorder(t *testing.T) {
	recorder, err := audit.NewRecorder(&captureAudit{}, &mockLogger{}, discardDrops{})
	require.NoError(t, err)

	_, err = NewIdentityService(&mockLogger{}, nil, recorder)
	require.ErrorIs(t, err, audit.ErrNilRecorder, "an identity service that could drop every audit event unreported is not built")
	_, err = NewIdentityService(&mockLogger{}, recorder, nil)
	require.ErrorIs(t, err, audit.ErrNilRecorder, "nor one that could hand out a raw key unrecorded")
}
