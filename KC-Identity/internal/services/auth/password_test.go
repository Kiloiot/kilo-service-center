package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr bool
	}{
		{"valid mixed", "secure1x", false},
		{"valid long mixed", "mypassword123", false},
		{"valid unicode letter with digit", "пароль1x", false},
		{"too short", "abc1", true},
		{"too long", strings.Repeat("a", 128) + "1x", true},
		{"letters only", "abcdefgh", true},
		{"digits only", "12345678", true},
		{"empty", "", true},
		{"exactly min length valid", "abcdef1x", false},
		{"exactly max length valid", strings.Repeat("a", 126) + "1x", false},
		{"special chars only no letter or digit", "!@#$%^&*", true},
		{"special chars with letter no digit", "!@#$abcd", true},
		{"special chars with digit no letter", "!@#$1234", true},
		{"special chars with letter and digit", "!@#$ab12", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.pass)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrUserPasswordWeak)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
