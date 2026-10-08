package bssciservices

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// EndpointProvisioningReader reads an endpoint's provisioning, including the
// nonce and signature of its last over-the-air attach.
type EndpointProvisioningReader interface {
	GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error)
}

// ActiveEndpointSessionReader reads the endpoint session currently in force.
type ActiveEndpointSessionReader interface {
	GetActive(ctx context.Context, endpointID string) (*models.EndPointSession, error)
}

// networkSessionKeySource resolves an endpoint's current network session key
// from its active endpoint session and its provisioning.
type networkSessionKeySource struct {
	endpoints EndpointProvisioningReader
	sessions  ActiveEndpointSessionReader
}

// NewNetworkSessionKeySource creates the source attach propagation draws the
// network session key from.
func NewNetworkSessionKeySource(endpoints EndpointProvisioningReader, sessions ActiveEndpointSessionReader) (bssci.NetworkSessionKeySource, error) {
	if endpoints == nil {
		return nil, errNilProvisioningReader
	}
	if sessions == nil {
		return nil, errNilActiveSessionReader
	}
	return &networkSessionKeySource{endpoints: endpoints, sessions: sessions}, nil
}

// NetworkSessionKey returns the key of the endpoint's over-the-air session
// while that session is in force, otherwise its pre-shared key.
func (k *networkSessionKeySource) NetworkSessionKey(ctx context.Context, endpoint *models.EndPoint) ([]byte, error) {
	session, err := k.sessions.GetActive(ctx, strconv.FormatInt(endpoint.ID, 10))
	if errors.Is(err, storage.ErrNotFound) || (err == nil && session == nil) {
		return endpoint.NwkSnKey, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoadEndpointSession, err)
	}
	provisioned, err := k.endpoints.GetByID(ctx, endpoint.ID, endpoint.TenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoadEndpointProvisioning, err)
	}
	if provisioned == nil {
		return nil, fmt.Errorf("%w: %w", errLoadEndpointProvisioning, storage.ErrNotFound)
	}
	return bssci.CurrentNetworkSessionKey(provisioned, session.SessionKey), nil
}
