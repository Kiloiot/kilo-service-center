package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	testAuditPlatformTenant = int64(1)
	testAuditCallerTenant   = int64(9)
	testSubjectEmail        = "subject@example.com"
	testNewPassword         = "n3w-secret-password"
)

// recordingService is an identity service whose audit events are captured.
func recordingService(t *testing.T) (*IdentityService, *captureAudit) {
	t.Helper()
	emitter := &captureAudit{}
	recorder, err := audit.NewRecorder(emitter, &mockLogger{}, discardDrops{})
	require.NoError(t, err)
	svc, err := NewIdentityService(&mockLogger{}, recorder, recorder)
	require.NoError(t, err)
	return svc.WithPlatformTenantID(testAuditPlatformTenant), emitter
}

// A password change is audited for the self-service change and the admin
// reset alike: the platform tenant, the actor as user, the subject in the
// data, and never the password.
func TestPasswordChange_IsAuditedWithoutThePassword(t *testing.T) {
	actorID := uuid.New()
	subjectID := uuid.New()
	svc, emitter := recordingService(t)
	svc.WithAdminUserService(&mockAdminUserService{getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
		return &models.User{ID: id, Email: testSubjectEmail, IsAdmin: id == actorID, IsActive: true}, nil
	}}).WithAuthService(&mockAuthService{})
	ctx := pkgcontext.WithUserID(contextForTenant(testAuditCallerTenant), actorID.String())

	_, err := svc.UpdateUserPassword(ctx, &pb.UpdateUserPasswordRequest{Id: subjectID.String(), NewPassword: testNewPassword})
	require.NoError(t, err)
	_, err = svc.ChangePassword(ctx, &pb.ChangePasswordRequest{CurrentPassword: testNewPassword, NewPassword: testNewPassword})
	require.NoError(t, err)

	require.Len(t, emitter.events, 2)
	for i, subject := range []uuid.UUID{subjectID, actorID} {
		ev := emitter.events[i]
		assert.Equal(t, models.EventTypeUserPasswordChanged, ev.EventType)
		assert.Equal(t, testAuditPlatformTenant, ev.TenantID)
		assert.Equal(t, actorID.String(), ev.UserID)
		assert.Equal(t, subject.String(), ev.Details[auditKeyUserID])
		assert.NotContains(t, fmt.Sprint(ev), testNewPassword)
	}
}

const testRegisterPassword = "register-secret-1"

// A self-registration is audited under the platform tenant with the new user
// as both actor and subject, and never with a credential.
func TestRegisterAccount_IsAuditedAsTheNewUser(t *testing.T) {
	userID := uuid.New()
	svc, emitter := recordingService(t)
	svc.WithRegistrationService(&mockRegistrationService{registerAccountFunc: func(context.Context, *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
		return &grpcservices.AuthLoginResult{
			Tokens:  &grpcservices.AuthTokens{AccessToken: testRegisterPassword},
			Profile: &grpcservices.UserProfile{ID: userID, Email: testSubjectEmail},
		}, nil
	}})

	_, err := svc.RegisterAccount(testutil.TestContext(), &pb.RegisterAccountRequest{
		Email: testSubjectEmail, Password: testRegisterPassword, FirstName: "New", LastName: "User",
	})
	require.NoError(t, err)

	require.Len(t, emitter.events, 1)
	ev := emitter.events[0]
	assert.Equal(t, models.EventTypeUserRegistered, ev.EventType)
	assert.Equal(t, testAuditPlatformTenant, ev.TenantID)
	assert.Equal(t, userID.String(), ev.UserID)
	assert.Equal(t, userID.String(), ev.Details[auditKeyUserID])
	assert.NotContains(t, fmt.Sprint(ev), testRegisterPassword)
}
