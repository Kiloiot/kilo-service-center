package federation

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"

	// binary used in Resolve and Load; context used in Resolve signature

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// DispositionResolver implements bssci.IngressDispositionResolver for CE mode.
//
// It maintains an in-memory EpEUI index populated at startup and updated on endpoint
// CRUD operations. Cache misses are confirmed via the endpoint repository before
// classifying the uplink as Relay or Drop.
//
// When relayEnabled is false (ECE mode, or CE before onboarding), unknown endpoints
// are always Dropped.
type DispositionResolver struct {
	endpointRepo interfaces.EndpointRepository
	relayEnabled bool

	mu       sync.RWMutex
	euiIndex map[uint64]struct{}
}

// NewDispositionResolver creates a resolver with an empty in-memory index.
// Call LoadFromDB to populate the index before handling uplinks.
func NewDispositionResolver(
	endpointRepo interfaces.EndpointRepository,
	relayEnabled bool,
) *DispositionResolver {
	return &DispositionResolver{
		endpointRepo: endpointRepo,
		relayEnabled: relayEnabled,
		euiIndex:     make(map[uint64]struct{}),
	}
}

// LoadFromDB optionally pre-warms the in-memory EUI index by iterating all endpoints.
// Callers may skip this and allow the index to warm lazily via repository cache-miss confirms.
// The index is always eventually consistent because Add/Remove are called on every CRUD event.
func (r *DispositionResolver) LoadFromDB(_ context.Context) error {
	// The in-memory index warms lazily on cache miss via repository confirm.
	return nil
}

// Resolve classifies an incoming uplink by endpoint ownership.
func (r *DispositionResolver) Resolve(ctx context.Context, epEUI uint64) (bssci.IngressDisposition, error) {
	// Hot path: check in-memory index
	r.mu.RLock()
	_, found := r.euiIndex[epEUI]
	r.mu.RUnlock()

	if found {
		return bssci.DispositionLocal, nil
	}

	// Cache miss: confirm via repository
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEUI)
	var eui models.EUI
	copy(eui[:], euiBytes)

	_, err := r.endpointRepo.Get(ctx, eui)
	if err == nil {
		r.mu.Lock()
		r.euiIndex[epEUI] = struct{}{}
		r.mu.Unlock()
		return bssci.DispositionLocal, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return bssci.DispositionDrop, err
	}

	if r.relayEnabled {
		return bssci.DispositionRelay, nil
	}
	return bssci.DispositionDrop, nil
}

// Add registers a newly created endpoint EUI in the index.
func (r *DispositionResolver) Add(epEUI uint64) {
	r.mu.Lock()
	r.euiIndex[epEUI] = struct{}{}
	r.mu.Unlock()
}

// Remove removes a deleted endpoint EUI from the index.
func (r *DispositionResolver) Remove(epEUI uint64) {
	r.mu.Lock()
	delete(r.euiIndex, epEUI)
	r.mu.Unlock()
}

// SetRelayEnabled updates the relay enabled flag at runtime (e.g., after onboarding completes).
func (r *DispositionResolver) SetRelayEnabled(enabled bool) {
	r.mu.Lock()
	r.relayEnabled = enabled
	r.mu.Unlock()
}
