package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

var (
	errOrgTestMembershipsDown = errors.New("membership store unavailable")
	errOrgTestSynthesisFailed = errors.New("default organization unavailable")
)

// fixedMemberships answers every membership lookup with the same result.
type fixedMemberships struct {
	list []*models.OrganizationMembershipWithOrg
	err  error
}

func (f fixedMemberships) ListUserMemberships(context.Context, uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	return f.list, f.err
}

// fixedSynthesizer answers every synthesis with the same result and counts calls.
type fixedSynthesizer struct {
	orgID uuid.UUID
	err   error
	calls *int
}

func (f fixedSynthesizer) SynthesizeMembership(context.Context, *models.User) (*grpcservices.OrganizationMembership, error) {
	*f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &grpcservices.OrganizationMembership{OrgID: f.orgID, Role: models.OrganizationRoleMember}, nil
}

func accessTokenOrg(t *testing.T, accessToken string) string {
	t.Helper()
	token, err := jwt.ParseInsecure([]byte(accessToken))
	require.NoError(t, err)
	var org string
	require.NoError(t, token.Get(svcTestTenantClaim, &org))
	return org
}

func TestRefreshTokens_ScopesTheRenewedAccessTokenToTheDefaultOrg(t *testing.T) {
	memberOrg, ceOrg := uuid.New(), uuid.New()
	tests := []struct {
		name          string
		memberships   fixedMemberships
		withCE        bool
		synthesisErr  error
		wantErr       error
		wantOrg       uuid.UUID
		wantSynthesis int
	}{
		{name: "first membership", memberships: fixedMemberships{list: []*models.OrganizationMembershipWithOrg{{OrgID: memberOrg}, {OrgID: uuid.New()}}}, withCE: true, wantOrg: memberOrg},
		{name: "membership lookup fails", memberships: fixedMemberships{err: errOrgTestMembershipsDown}, withCE: true, wantErr: ErrMembershipRequired},
		{name: "no membership and no default organization", wantErr: ErrMembershipRequired},
		{name: "no membership, default organization synthesized", withCE: true, wantOrg: ceOrg, wantSynthesis: 1},
		{name: "no membership, synthesis fails", withCE: true, synthesisErr: errOrgTestSynthesisFailed, wantErr: errOrgTestSynthesisFailed, wantSynthesis: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t, true)
			raw := f.seedRefreshToken(t, keepToken)
			svc := NewService(f.users, tt.memberships, f.tokens, f.issuer, svcTestLocalLogin, svcTestRefresh, logger.NewNop())
			var synthesized int
			if tt.withCE {
				svc.WithCEProvider(fixedSynthesizer{orgID: ceOrg, err: tt.synthesisErr, calls: &synthesized})
			}

			tokens, err := svc.RefreshTokens(testutil.TestContext(), raw)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantSynthesis, synthesized)
			if tt.wantErr != nil {
				assert.Nil(t, tokens)
				assert.False(t, f.stored(t, raw).IsReplaced(), "a refused refresh must not consume the token")
				return
			}
			assert.Equal(t, tt.wantOrg.String(), accessTokenOrg(t, tokens.AccessToken))
		})
	}
}
