package auth

import "github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

// requireActive refuses a deactivated user: only active users may obtain tokens.
func requireActive(user *models.User) error {
	if !user.IsActive {
		return ErrUserNotActive
	}
	return nil
}
