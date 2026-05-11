package bssci

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// LocalEndpointDispositionResolver classifies uplinks using the local endpoint repository.
// Endpoints found in any local tenant are served locally; unknown endpoints are dropped.
// CE mode replaces this with a relay-capable resolver in internal/services/federation/disposition.go.
type LocalEndpointDispositionResolver struct {
	endpointRepo interfaces.EndpointRepository
}

// NewLocalEndpointDispositionResolver creates a resolver that classifies endpoints as
// Local (found in any tenant) or Drop (unknown, no relay configured).
func NewLocalEndpointDispositionResolver(repo interfaces.EndpointRepository) *LocalEndpointDispositionResolver {
	return &LocalEndpointDispositionResolver{endpointRepo: repo}
}

// Resolve checks the local endpoint repository for the given EUI.
// Returns DispositionLocal if found in any tenant, DispositionDrop if not found,
// or an error if the repository query itself fails.
func (r *LocalEndpointDispositionResolver) Resolve(ctx context.Context, epEUI uint64) (IngressDisposition, error) {
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEUI)

	var eui models.EUI
	copy(eui[:], euiBytes)

	_, err := r.endpointRepo.Get(ctx, eui)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return DispositionDrop, nil
		}
		return DispositionDrop, err
	}
	return DispositionLocal, nil
}
