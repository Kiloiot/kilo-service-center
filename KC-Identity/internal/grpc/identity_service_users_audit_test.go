package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
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

// errTestSessionsKept is a password change that stored the new password but
// could not end the old sessions, wrapped as the services return it.
var errTestSessionsKept = fmt.Errorf("revoke sessions: %w: %w", auth.ErrPasswordChangedSessionsKept, errTestRevokeUnavailable)

var errTestRevokeUnavailable = errors.New("refresh token store unavailable")

// A stored password is audited exactly once, also when ending the old
// sessions failed and the caller gets an error.
func TestPasswordChange_IsAuditedWheneverTheNewPasswordIsStored(t *testing.T) {
	actorID := uuid.New()
	entries := []struct {
		name      string
		failToken string
		change    func(*IdentityService, context.Context, error) error
	}{
		{name: "self-service change", failToken: grpcerrors.ErrTokenChangePasswordFailed, change: func(svc *IdentityService, ctx context.Context, outcome error) error {
			svc.WithAuthService(&mockAuthService{changePasswordFunc: func(context.Context, uuid.UUID, string, string) error { return outcome }})
			_, err := svc.ChangePassword(ctx, &pb.ChangePasswordRequest{CurrentPassword: testNewPassword, NewPassword: testNewPassword})
			return err
		}},
		{name: "admin reset", failToken: grpcerrors.ErrTokenUpdatePasswordFailed, change: func(svc *IdentityService, ctx context.Context, outcome error) error {
			svc.WithAdminUserService(&mockAdminUserService{
				getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
					return &models.User{ID: id, Email: testSubjectEmail, IsAdmin: id == actorID, IsActive: true}, nil
				},
				updatePasswordFunc: func(context.Context, uuid.UUID, string) error { return outcome },
			})
			_, err := svc.UpdateUserPassword(ctx, &pb.UpdateUserPasswordRequest{Id: uuid.NewString(), NewPassword: testNewPassword})
			return err
		}},
	}
	outcomes := []struct {
		name    string
		outcome error
		failed  bool
	}{
		{name: "stored and sessions ended"},
		{name: "stored but sessions kept", outcome: errTestSessionsKept, failed: true},
	}
	for _, entry := range entries {
		for _, oc := range outcomes {
			t.Run(entry.name+"/"+oc.name, func(t *testing.T) {
				svc, emitter := recordingService(t)
				ctx := pkgcontext.WithUserID(contextForTenant(testAuditCallerTenant), actorID.String())

				err := entry.change(svc, ctx, oc.outcome)

				require.Len(t, emitter.events, 1)
				assert.Equal(t, models.EventTypeUserPasswordChanged, emitter.events[0].EventType)
				if !oc.failed {
					require.NoError(t, err)
					return
				}
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, grpcerrors.GetGRPCCode(entry.failToken), st.Code())
				assert.Equal(t, grpcerrors.ResolveErrorMessage(entry.failToken), st.Message())
			})
		}
	}
}

func TestPasswordChange_IsNotAuditedWhenNothingWasStored(t *testing.T) {
	svc, emitter := recordingService(t)
	svc.WithAuthService(&mockAuthService{changePasswordFunc: func(context.Context, uuid.UUID, string, string) error {
		return auth.ErrInvalidCredentials
	}})
	ctx := pkgcontext.WithUserID(contextForTenant(testAuditCallerTenant), uuid.NewString())

	_, err := svc.ChangePassword(ctx, &pb.ChangePasswordRequest{CurrentPassword: testNewPassword, NewPassword: testNewPassword})

	require.Error(t, err)
	assert.Empty(t, emitter.events)
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
