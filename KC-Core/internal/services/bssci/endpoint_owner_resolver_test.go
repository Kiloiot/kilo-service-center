package bssciservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	ownerRuleServingTenant = int64(7)
	ownerRuleOwnerTenant   = int64(3)
	ownerRuleEUI           = uint64(0x70B3D5677011150A)
)

type ownerLookupByEUI map[models.EUI]*models.EndPoint

func (l ownerLookupByEUI) Get(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	endpoint, ok := l[eui]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return endpoint, nil
}

func TestEndpointOwnerResolverNamesTheOwnerTenant(t *testing.T) {
	eui := models.EUI(mioty.EUI64(ownerRuleEUI).ToBytes())
	endpoint := &models.EndPoint{ID: 11, EUI: eui, TenantID: ownerRuleServingTenant, OwnerTenantID: ownerRuleOwnerTenant}
	resolver, err := NewEndpointOwnerResolver(ownerLookupByEUI{eui: endpoint})
	require.NoError(t, err)

	owner, err := resolver.ResolveOwner(testutil.TestContext(), eui)

	require.NoError(t, err)
	assert.Equal(t, ownerRuleOwnerTenant, owner.TenantID, "the owner is the endpoint's owner_tenant_id")
	assert.Same(t, endpoint, owner.Endpoint)
}

func TestEndpointOwnerResolverReportsAnEndpointNoTenantOwns(t *testing.T) {
	resolver, err := NewEndpointOwnerResolver(ownerLookupByEUI{})
	require.NoError(t, err)

	_, err = resolver.ResolveOwner(testutil.TestContext(), models.EUI(mioty.EUI64(ownerRuleEUI).ToBytes()))

	require.ErrorIs(t, err, storage.ErrNotFound)
}

func TestNewEndpointOwnerResolverRefusesAMissingLookup(t *testing.T) {
	_, err := NewEndpointOwnerResolver(nil)

	require.ErrorIs(t, err, errNilEndpointOwnerLookup)
}

func TestNewUplinkIngestServiceRefusesAMissingOwnerRule(t *testing.T) {
	_, err := NewUplinkIngestService(nil, UplinkWindows{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 0, 0)

	require.ErrorIs(t, err, ErrNilEndpointOwnerResolver)
}
