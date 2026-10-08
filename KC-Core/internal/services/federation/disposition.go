package federation

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	// binary used in Resolve and Load; context used in Resolve signature

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// EndpointEUIEnumerator lists every endpoint EUI for index pre-warming. It is
// a consumer-owned port implemented by the concrete PostgreSQL store.
type EndpointEUIEnumerator interface {
	ListAllEUIs(ctx context.Context) ([]models.EUI, error)
}

// EndpointReader retrieves a single endpoint by EUI for disposition classification.
type EndpointReader interface {
	Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error)
}

// DispositionResolver implements bssci.IngressDispositionResolver for CE mode.
//
// It maintains an in-memory EpEUI index populated at startup and updated on endpoint
// CRUD operations. Cache misses are confirmed via the endpoint repository before
// classifying the uplink as Relay or Drop.
//
// When relayEnabled is false (ECE mode, or CE before onboarding), unknown endpoints
// are always Dropped.
type DispositionResolver struct {
	endpointRepo EndpointReader
	enumerator   EndpointEUIEnumerator

	mu           sync.RWMutex
	relayEnabled bool
	euiIndex     map[uint64]struct{}
}

// NewDispositionResolver creates a resolver with an empty in-memory index.
// Call LoadFromDB to populate the index before handling uplinks; the
// enumerator may be nil, leaving the index to warm lazily.
func NewDispositionResolver(
	endpointRepo EndpointReader,
	enumerator EndpointEUIEnumerator,
	relayEnabled bool,
) *DispositionResolver {
	return &DispositionResolver{
		endpointRepo: endpointRepo,
		enumerator:   enumerator,
		relayEnabled: relayEnabled,
		euiIndex:     make(map[uint64]struct{}),
	}
}

// LoadFromDB pre-warms the in-memory EUI index from the enumerator. The
// replacement index is built completely before an atomic swap, so a failed
// enumeration leaves the existing index untouched and lazy warming intact.
func (r *DispositionResolver) LoadFromDB(ctx context.Context) error {
	if r.enumerator == nil {
		return nil
	}
	euis, err := r.enumerator.ListAllEUIs(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", pkgfederation.ErrEndpointIndexPrewarm, err)
	}
	replacement := make(map[uint64]struct{}, len(euis))
	for _, eui := range euis {
		replacement[eui.ToUint64()] = struct{}{}
	}
	r.mu.Lock()
	r.euiIndex = replacement
	r.mu.Unlock()
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

	// relayEnabled is written by SetRelayEnabled at runtime; read it under
	// the same lock.
	r.mu.RLock()
	relayEnabled := r.relayEnabled
	r.mu.RUnlock()
	if relayEnabled {
		return bssci.DispositionRelay, nil
	}
	return bssci.DispositionDrop, nil
}

// Add registers a newly created endpoint EUI in the index. Implements the
// grpcservices.EndpointIndex port called after successful endpoint CRUD.
func (r *DispositionResolver) Add(_ context.Context, eui models.EUI) {
	r.mu.Lock()
	r.euiIndex[eui.ToUint64()] = struct{}{}
	r.mu.Unlock()
}

// Remove removes a deleted endpoint EUI from the index. Implements the
// grpcservices.EndpointIndex port called after successful endpoint CRUD.
func (r *DispositionResolver) Remove(_ context.Context, eui models.EUI) {
	r.mu.Lock()
	delete(r.euiIndex, eui.ToUint64())
	r.mu.Unlock()
}

// SetRelayEnabled updates the relay enabled flag at runtime (e.g., after onboarding completes).
func (r *DispositionResolver) SetRelayEnabled(enabled bool) {
	r.mu.Lock()
	r.relayEnabled = enabled
	r.mu.Unlock()
}
