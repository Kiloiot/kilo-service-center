package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

var errStoreUnavailable = errors.New("store unavailable")

// flakyOrgStore answers the first lookup and fails every later one.
type flakyOrgStore struct {
	OrganizationStore
	org     *models.Organization
	lookups int
}

func (s *flakyOrgStore) GetByID(context.Context, uuid.UUID, int64) (*models.Organization, error) {
	s.lookups++
	if s.lookups > 1 {
		return nil, errStoreUnavailable
	}
	return s.org, nil
}

func TestUpdateOrganization_WithoutChangesReturnsTheLookedUpOrganization(t *testing.T) {
	org := &models.Organization{OrgID: uuid.New()}
	store := &flakyOrgStore{org: org}
	svc := NewOrganizationAdminService(store, nil, logger.NewNop())

	got, err := svc.Update(testutil.TestContext(), org.OrgID, 1, &grpcservices.OrganizationUpdateRequest{})

	require.NoError(t, err)
	assert.Same(t, org, got)
	assert.Equal(t, 1, store.lookups)
}
