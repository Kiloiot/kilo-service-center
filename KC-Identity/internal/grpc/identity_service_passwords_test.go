package grpc

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const testWeakPassword = "short"

// refusingPasswords refuses every password change for reason.
func refusingPasswords(actorID uuid.UUID, reason error) *IdentityService {
	return &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(actorID),
			createFunc: func(context.Context, *grpcservices.UserCreateRequest) (*models.User, error) {
				return nil, reason
			},
			updatePasswordFunc: func(context.Context, uuid.UUID, string) error { return reason },
		},
		authSvc: &mockAuthService{changePasswordFunc: func(context.Context, uuid.UUID, string, string) error {
			return reason
		}},
		audit: discardAudit{},
		log:   &mockLogger{},
	}
}

func assertRefusal(t *testing.T, err error, token string) {
	t.Helper()
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(token), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(token), st.Message())
}

// A weak password is refused with the password policy as the reason, on
// every path that sets a password.
func TestPasswords_WeakPasswordNamesThePolicy(t *testing.T) {
	actorID := uuid.New()
	svc := refusingPasswords(actorID, auth.ErrUserPasswordWeak)
	ctx := pkgcontext.WithUserID(contextForTenant(testAuditCallerTenant), actorID.String())

	_, err := svc.CreateUser(ctx, &pb.CreateUserRequest{Email: testSubjectEmail, Password: testWeakPassword})
	assertRefusal(t, err, grpcerrors.ErrTokenWeakPassword)
	_, err = svc.UpdateUserPassword(ctx, &pb.UpdateUserPasswordRequest{Id: uuid.NewString(), NewPassword: testWeakPassword})
	assertRefusal(t, err, grpcerrors.ErrTokenWeakPassword)
	_, err = svc.ChangePassword(ctx, &pb.ChangePasswordRequest{CurrentPassword: testNewPassword, NewPassword: testWeakPassword})
	assertRefusal(t, err, grpcerrors.ErrTokenWeakPassword)

	assert.Equal(t, codes.InvalidArgument, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenWeakPassword))
	assert.Contains(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenWeakPassword), strconv.Itoa(config.AuthPasswordMinLength))
}

// A wrong current password is refused as such, not as a failed session.
func TestPasswords_WrongCurrentPasswordIsNamed(t *testing.T) {
	actorID := uuid.New()
	svc := refusingPasswords(actorID, auth.ErrInvalidCredentials)
	ctx := pkgcontext.WithUserID(contextForTenant(testAuditCallerTenant), actorID.String())

	_, err := svc.ChangePassword(ctx, &pb.ChangePasswordRequest{CurrentPassword: testWeakPassword, NewPassword: testNewPassword})
	assertRefusal(t, err, grpcerrors.ErrTokenCurrentPasswordIncorrect)
	assert.NotEqual(t, codes.Unauthenticated, status.Code(err))
}
