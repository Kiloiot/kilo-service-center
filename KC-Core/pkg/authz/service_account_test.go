package authz

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func TestServiceAccount_ManagesOnlyItsOwnOrganization(t *testing.T) {
	own := uuid.New()
	userID := uuid.New()
	past := time.Now().Add(-time.Hour)
	key := func(mutate func(*models.APIKey)) *models.APIKey {
		k := &models.APIKey{ID: uuid.New(), OrgID: own, KeyType: models.KeyTypeServiceAccount, IsActive: true}
		if mutate != nil {
			mutate(k)
		}
		return k
	}
	cases := []struct {
		name string
		key  *models.APIKey
		org  uuid.UUID
		want Roles
	}{
		{name: "own organization", key: key(nil), org: own, want: Roles{BaseStationManager: true, EndpointManager: true}},
		{name: "another organization", key: key(nil), org: uuid.New(), want: Roles{}},
		{name: "no organization", key: key(nil), org: uuid.Nil, want: Roles{}},
		{name: "revoked key", key: key(func(k *models.APIKey) { k.IsActive = false }), org: own, want: Roles{}},
		{name: "expired key", key: key(func(k *models.APIKey) { k.ExpiresAt = &past }), org: own, want: Roles{}},
		{name: "a user key is not a service account", key: key(func(k *models.APIKey) {
			k.KeyType = models.KeyTypeUser
			k.UserID = &userID
		}), org: own, want: Roles{}},
		{name: "no key", key: nil, org: own, want: Roles{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ServiceAccount(tc.key, tc.org)
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			if got.Admin || got.TenantManager {
				t.Fatal("a service account is never an administrator or tenant manager")
			}
		})
	}
}
