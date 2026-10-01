package grpc

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/registration"
)

// RegisterAccount handles self-service account registration.
func (s *IdentityService) RegisterAccount(ctx context.Context, req *pb.RegisterAccountRequest) (*pb.LoginResponse, error) {
	if s.registrationSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrationDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationDisabled))
	}

	if req.Email == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
	}
	if req.Password == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPasswordRequired))
	}
	if req.FirstName == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFirstNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFirstNameRequired))
	}
	if req.LastName == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLastNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLastNameRequired))
	}
	// CompanyName is optional in CE (validated by registration service)
	// In ECE, the registration service enforces this requirement

	svcReq := &grpcservices.RegisterAccountRequest{
		Email:       req.Email,
		Password:    req.Password,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		CompanyName: req.CompanyName,
	}

	result, err := s.registrationSvc.RegisterAccount(ctx, svcReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogRegistrationFailed, logger.Err(err))
		return nil, mapRegistrationError(err)
	}

	resp, err := buildLoginResponse(result)
	if err != nil {
		s.log.ErrorContext(ctx, LogBuildRegistrationResponseFailed, logger.Err(err))
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationFailed))
	}
	s.auditRegistration(ctx, result.Profile)
	return resp, nil
}

// auditRegistration records a user creating their own account under the
// platform tenant like every account event: the new user is the actor and the
// subject, and no credential enters the event.
func (s *IdentityService) auditRegistration(ctx context.Context, user *grpcservices.UserProfile) {
	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		EventType:   models.EventTypeUserRegistered,
		Title:       models.EventTitleUserRegistered,
		Description: fmt.Sprintf(models.EventDescriptionUserRegistered, user.Email),
		SourceName:  user.Email,
		UserID:      user.ID.String(),
		Details:     map[string]any{auditKeyUserID: user.ID.String(), userFieldEmail: user.Email},
	})
}

// mapRegistrationError maps registration domain sentinels to gRPC status
// codes via the catalog; matching uses errors.Is, never error text.
func mapRegistrationError(err error) error {
	token := grpcerrors.ErrTokenRegistrationFailed
	for sentinel, t := range registrationSentinelTokens {
		if errors.Is(err, sentinel) {
			token = t
			break
		}
	}
	return status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
}

// registrationSentinelTokens pairs every registration domain sentinel with
// its catalog token. An already-registered email maps to the generic
// invalid-argument token so account existence is not disclosed.
var registrationSentinelTokens = map[error]string{
	registration.ErrRegistrationDisabled: grpcerrors.ErrTokenRegistrationDisabled,
	registration.ErrEmailExists:          grpcerrors.ErrTokenInvalidArgument,
	registration.ErrWeakPassword:         grpcerrors.ErrTokenWeakPassword,
	registration.ErrEmailRequired:        grpcerrors.ErrTokenEmailRequired,
	registration.ErrFirstNameRequired:    grpcerrors.ErrTokenFirstNameRequired,
	registration.ErrLastNameRequired:     grpcerrors.ErrTokenLastNameRequired,
	registration.ErrCompanyNameRequired:  grpcerrors.ErrTokenCompanyNameRequired,
	registration.ErrEmailInvalidFormat:   grpcerrors.ErrTokenEmailInvalidFormat,
	registration.ErrFieldTooLong:         grpcerrors.ErrTokenFieldTooLong,
}
