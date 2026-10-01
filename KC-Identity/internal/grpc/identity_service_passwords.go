package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
)

// passwordRefusalTokens pairs each reason a password is refused with the
// catalog token that tells the user why.
var passwordRefusalTokens = map[error]string{
	auth.ErrUserPasswordWeak:   grpcerrors.ErrTokenWeakPassword,
	auth.ErrInvalidCredentials: grpcerrors.ErrTokenCurrentPasswordIncorrect,
}

// passwordRefusal answers a refused password with its reason, or with the
// fallback token for any other failure.
func passwordRefusal(err error, fallback string) error {
	token := fallback
	for reason, reasonToken := range passwordRefusalTokens {
		if errors.Is(err, reason) {
			token = reasonToken
			break
		}
	}
	return status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
}

// ChangePassword changes the current user's password.
func (s *IdentityService) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	userID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.CurrentPassword == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCurrentPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCurrentPasswordRequired))
	}
	if req.NewPassword == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNewPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNewPasswordRequired))
	}

	if err := s.authSvc.ChangePassword(ctx, userID, req.CurrentPassword, req.NewPassword); err != nil {
		s.log.ErrorContext(ctx, LogChangePasswordFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return nil, passwordRefusal(err, grpcerrors.ErrTokenChangePasswordFailed)
	}
	s.auditPasswordChange(ctx, userID)

	return &pb.ChangePasswordResponse{Success: true}, nil
}

// UpdateUserPassword updates a user's password (admin only).
func (s *IdentityService) UpdateUserPassword(ctx context.Context, req *pb.UpdateUserPasswordRequest) (*pb.UpdateUserPasswordResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}
	if req.NewPassword == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNewPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNewPasswordRequired))
	}

	userID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	if err := s.adminUserSvc.UpdatePassword(ctx, userID, req.NewPassword); err != nil {
		s.log.ErrorContext(ctx, LogUpdateUserPasswordFailed, logger.FieldUserIDSnake, req.Id, logger.FieldError, err)
		return nil, passwordRefusal(err, grpcerrors.ErrTokenUpdatePasswordFailed)
	}
	s.auditPasswordChange(ctx, userID)

	return &pb.UpdateUserPasswordResponse{Success: true}, nil
}

// auditPasswordChange records that subject's password changed, set by the
// user or reset by an admin, under the platform tenant like every account
// event; the password never enters the event.
func (s *IdentityService) auditPasswordChange(ctx context.Context, subject uuid.UUID) {
	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		EventType:   models.EventTypeUserPasswordChanged,
		Title:       models.EventTitleUserPasswordChanged,
		Description: fmt.Sprintf(models.EventDescriptionUserPasswordChanged, subject),
		SourceName:  subject.String(),
		Details:     map[string]any{auditKeyUserID: subject.String()},
	})
}
