package authz

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func TestEffective_CombinesUserFlagsWithAnActiveMembership(t *testing.T) {
	active := models.OrganizationMemberStatusActive
	cases := []struct {
		name       string
		user       *models.User
		membership *models.OrganizationMember
		want       Roles
	}{
		{name: "no user", user: nil, want: Roles{}},
		{name: "self-registered user without flags", user: &models.User{IsActive: true}, want: Roles{}},
		{name: "inactive administrator holds nothing", user: &models.User{IsAdmin: true}, want: Roles{}},
		{name: "administrator holds every role", user: &models.User{IsActive: true, IsAdmin: true}, want: AllRoles},
		{
			name: "user flags alone",
			user: &models.User{IsActive: true, IsEndpointManager: true},
			want: Roles{EndpointManager: true},
		},
		{
			name:       "membership flags add organization roles",
			user:       &models.User{IsActive: true, IsEndpointManager: true},
			membership: &models.OrganizationMember{Status: active, IsOrgAdmin: true, IsBaseStationAdmin: true},
			want:       Roles{TenantManager: true, BaseStationManager: true, EndpointManager: true},
		},
		{
			name:       "an inactive membership grants nothing",
			user:       &models.User{IsActive: true},
			membership: &models.OrganizationMember{Status: models.OrganizationMemberStatusInvited, IsOrgAdmin: true, IsBaseStationAdmin: true, IsEndpointAdmin: true},
			want:       Roles{},
		},
		{
			name:       "membership flags never make an administrator",
			user:       &models.User{IsActive: true},
			membership: &models.OrganizationMember{Status: active, IsOrgAdmin: true, IsBaseStationAdmin: true, IsEndpointAdmin: true},
			want:       Roles{TenantManager: true, BaseStationManager: true, EndpointManager: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Effective(tc.user, tc.membership); got != tc.want {
				t.Fatalf("Effective() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestRequirement_AdministratorSatisfiesEveryRequirement(t *testing.T) {
	for _, q := range []Requirement{AdminOnly, TenantManager, BaseStationManager, EndpointManager, AnyManager, AnyRole} {
		if !q.GrantedTo(AllRoles) {
			t.Fatalf("requirement %b refused an administrator", q)
		}
		if q.GrantedTo(Roles{}) {
			t.Fatalf("requirement %b admitted a caller without roles", q)
		}
	}
	if AdminOnly.GrantedTo(Roles{TenantManager: true, BaseStationManager: true, EndpointManager: true}) {
		t.Fatal("manager roles satisfied an administrator-only requirement")
	}
	if !AnyManager.GrantedTo(Roles{EndpointManager: true}) || AnyManager.GrantedTo(Roles{TenantManager: true}) {
		t.Fatal("AnyManager must admit base station and endpoint managers only")
	}
}

func TestEventCategories_EveryStoredCategoryHasAnOwner(t *testing.T) {
	for _, category := range []string{
		models.EventCategorySecurity, models.EventCategoryEndpoint, models.EventCategoryBaseStation,
		models.EventCategoryMessage, models.EventCategorySystem, models.EventCategoryRoaming,
		models.EventCategoryError, models.EventCategoryAudit, models.EventCategoryProtocol,
		models.EventCategorySCACI, models.EventCategoryBSSCI, models.EventCategorySession,
	} {
		if !models.IsValidEventCategory(category) {
			t.Fatalf("%q is not a stored category", category)
		}
		if _, ok := categoryRequirements[category]; !ok {
			t.Fatalf("category %q has no reader requirement", category)
		}
	}
	for _, adminOnly := range []string{models.EventCategorySecurity, models.EventCategorySystem, models.EventCategoryAudit, models.EventCategoryError} {
		if CanReadCategory(Roles{TenantManager: true, BaseStationManager: true, EndpointManager: true}, adminOnly) {
			t.Fatalf("a manager may read %q", adminOnly)
		}
	}
}

func TestVisibleEventCategories(t *testing.T) {
	if cats, unrestricted := VisibleEventCategories(AllRoles, nil); !unrestricted || cats != nil {
		t.Fatalf("administrator without a filter: %v, %v", cats, unrestricted)
	}
	if cats, unrestricted := VisibleEventCategories(AllRoles, []string{models.EventCategorySecurity}); unrestricted || len(cats) != 1 {
		t.Fatalf("administrator with a filter keeps it: %v, %v", cats, unrestricted)
	}
	if cats, unrestricted := VisibleEventCategories(Roles{}, nil); unrestricted || len(cats) != 0 {
		t.Fatalf("a caller without roles reads nothing: %v, %v", cats, unrestricted)
	}
	if cats, _ := VisibleEventCategories(Roles{EndpointManager: true}, []string{"unknown"}); len(cats) != 0 {
		t.Fatalf("an unknown category is hidden from managers: %v", cats)
	}
}

func TestRolesTravelOnTheContext(t *testing.T) {
	ctx := testutil.TestContext()
	if FromContext(ctx).Any() {
		t.Fatal("a context without roles must hold none")
	}
	want := Roles{BaseStationManager: true}
	if got := FromContext(WithRoles(ctx, want)); got != want {
		t.Fatalf("FromContext = %+v, want %+v", got, want)
	}
}
