package auth

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// requireActive refuses a deactivated user: only active users may obtain tokens.
func requireActive(user *models.User) error {
	if !user.IsActive {
		return ErrUserNotActive
	}
	return nil
}

// refuseInactive applies requireActive to an externally authenticated user
// before anything is written or issued for them.
func (s *ExternalAuthService) refuseInactive(ctx context.Context, user *models.User) error {
	if err := requireActive(user); err != nil {
		s.logger.WarnContext(ctx, logExternalAuthUserInactive, logger.FieldUserIDSnake, user.ID)
		return err
	}
	return nil
}
