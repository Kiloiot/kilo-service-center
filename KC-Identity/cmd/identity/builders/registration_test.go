package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

func TestSelfRegistrationEnabled_RequiresRegistrationAndLocalLogin(t *testing.T) {
	tests := []struct {
		name         string
		registration bool
		localLogin   bool
		want         bool
	}{
		{name: "both on", registration: true, localLogin: true, want: true},
		{name: "registration off", registration: false, localLogin: true, want: false},
		{name: "local login off", registration: true, localLogin: false, want: false},
		{name: "both off", registration: false, localLogin: false, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selfRegistrationEnabled(pkgconfig.AuthConfig{RegistrationEnabled: tt.registration, LocalLoginEnabled: tt.localLogin})
			assert.Equal(t, tt.want, got)
		})
	}
}
