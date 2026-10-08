package bssciservices

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// EndpointOwnerLookup reads an endpoint by its globally unique EUI, whichever
// tenant it is provisioned under.
type EndpointOwnerLookup interface {
	Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error)
}

type endpointOwnerResolver struct {
	endpoints EndpointOwnerLookup
}

// NewEndpointOwnerResolver builds the one rule that names an endpoint's owner:
// the endpoint's owner_tenant_id.
func NewEndpointOwnerResolver(endpoints EndpointOwnerLookup) (bssci.EndpointOwnerResolver, error) {
	if endpoints == nil {
		return nil, errNilEndpointOwnerLookup
	}
	return endpointOwnerResolver{endpoints: endpoints}, nil
}

func (r endpointOwnerResolver) ResolveOwner(ctx context.Context, eui models.EUI) (bssci.EndpointOwner, error) {
	endpoint, err := r.endpoints.Get(ctx, eui)
	if err != nil {
		return bssci.EndpointOwner{}, fmt.Errorf(errFmtResolveEndpointOwner, eui, err)
	}
	return bssci.EndpointOwner{TenantID: endpoint.OwnerTenantID, Endpoint: endpoint}, nil
}
