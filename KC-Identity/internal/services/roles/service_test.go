package roles

import (
	"context"
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

type fakeUsers struct {
	user *models.User
	err  error
}

func (f fakeUsers) GetByID(context.Context, uuid.UUID) (*models.User, error) { return f.user, f.err }

type fakeMembers struct {
	member *models.OrganizationMemberWithEmail
	err    error
	asked  bool
}

func (f *fakeMembers) GetMember(context.Context, uuid.UUID, uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
	f.asked = true
	return f.member, f.err
}

var errTestStore = errors.New("store unavailable")

type fakeKeys struct {
	key *models.APIKey
	err error
}

func (f fakeKeys) GetByID(context.Context, uuid.UUID) (*models.APIKey, error) { return f.key, f.err }

func TestResolve(t *testing.T) {
	active := &models.User{IsActive: true}
	ownerMembership := &models.OrganizationMemberWithEmail{Status: models.OrganizationMemberStatusActive, IsOrgAdmin: true, IsBaseStationAdmin: true, IsEndpointAdmin: true}
	cases := []struct {
		name    string
		users   fakeUsers
		members *fakeMembers
		org     uuid.UUID
		want    authz.Roles
		wantErr error
	}{
		{name: "self-registered member without flags holds nothing", users: fakeUsers{user: active},
			members: &fakeMembers{member: &models.OrganizationMemberWithEmail{Status: models.OrganizationMemberStatusActive}}, org: uuid.New()},
		{name: "no membership leaves the user flags", users: fakeUsers{user: &models.User{IsActive: true, IsBaseStationManager: true}},
			members: &fakeMembers{err: storage.ErrRecordNotFound}, org: uuid.New(), want: authz.Roles{BaseStationManager: true}},
		{name: "an owner membership grants the organization roles", users: fakeUsers{user: active},
			members: &fakeMembers{member: ownerMembership}, org: uuid.New(),
			want: authz.Roles{TenantManager: true, BaseStationManager: true, EndpointManager: true}},
		{name: "without an organization the membership is not consulted", users: fakeUsers{user: active},
			members: &fakeMembers{member: ownerMembership}, org: uuid.Nil},
		{name: "unknown user", users: fakeUsers{err: storage.ErrRecordNotFound}, members: &fakeMembers{}, org: uuid.New(), wantErr: ErrUserNotFound},
		{name: "user store failure", users: fakeUsers{err: errTestStore}, members: &fakeMembers{}, org: uuid.New(), wantErr: errTestStore},
		{name: "membership store failure", users: fakeUsers{user: active}, members: &fakeMembers{err: errTestStore}, org: uuid.New(), wantErr: errTestStore},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := New(tc.users, tc.members, fakeKeys{}).Resolve(testutil.TestContext(), uuid.New(), tc.org)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("roles = %+v, want %+v", got, tc.want)
			}
			if tc.org == uuid.Nil && tc.members.asked {
				t.Fatal("membership was read without an organization")
			}
		})
	}
}

func TestResolveServiceAccount(t *testing.T) {
	own := uuid.New()
	serviceAccount := &models.APIKey{ID: uuid.New(), OrgID: own, KeyType: models.KeyTypeServiceAccount, IsActive: true}
	cases := []struct {
		name    string
		keys    fakeKeys
		org     uuid.UUID
		want    authz.Roles
		wantErr error
	}{
		{name: "own organization", keys: fakeKeys{key: serviceAccount}, org: own, want: authz.ServiceAccountRoles},
		{name: "another organization", keys: fakeKeys{key: serviceAccount}, org: uuid.New()},
		{name: "unknown key", keys: fakeKeys{err: storage.ErrRecordNotFound}, org: own, wantErr: ErrAPIKeyNotFound},
		{name: "key store failure", keys: fakeKeys{err: errTestStore}, org: own, wantErr: errTestStore},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := New(fakeUsers{}, &fakeMembers{}, tc.keys).ResolveServiceAccount(testutil.TestContext(), serviceAccount.ID, tc.org)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("roles = %+v, want %+v", got, tc.want)
			}
		})
	}
}
