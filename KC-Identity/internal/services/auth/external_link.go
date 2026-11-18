package auth

import (
	"context"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// linkExternalID records the provider identity on an existing account. An
// account deactivated since it was read is refused; any other failure is
// logged and sign-in goes on, as the link only speeds up the next lookup.
func (s *ExternalAuthService) linkExternalID(ctx context.Context, user *models.User, externalID string) error {
	err := s.userStore.LinkExternalID(ctx, user.ID, externalID)
	if errors.Is(err, storage.ErrUserInactive) {
		s.logger.WarnContext(ctx, logExternalAuthUserInactive, logger.FieldUserIDSnake, user.ID)
		return ErrUserNotActive
	}
	if err != nil {
		s.logger.WarnContext(ctx, logExternalAuthLinkFailed, logger.FieldError, err)
		return nil
	}
	user.ExternalID = &externalID
	return nil
}
