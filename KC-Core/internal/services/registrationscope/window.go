// Package registrationscope starts an endpoint's views at its registration,
// so the history a deleted registration of the same EUI left stays with it.
package registrationscope

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// ErrReadRegistration reports that the registration an endpoint's view starts at could not be read.
var ErrReadRegistration = errors.New("read the endpoint registration")

// Registrations tells when the tenant registered the endpoint an EUI names;
// storage.ErrNotFound when it has none.
type Registrations interface {
	RegisteredAt(ctx context.Context, tenantID int64, epEui []byte) (time.Time, error)
}

// Window is the part of an endpoint's history its current registration owns.
type Window struct {
	registrations Registrations
}

// New builds the window over the tenant's endpoint registrations.
func New(registrations Registrations) *Window {
	return &Window{registrations: registrations}
}

// Start is where a view of the endpoint's history begins: the requested
// start, or the registration when that is later. registered is false when the
// tenant has no endpoint with that EUI, whose views are then empty.
func (w *Window) Start(ctx context.Context, tenantID int64, epEui []byte, requested *time.Time) (start *time.Time, registered bool, err error) {
	since, err := w.registrations.RegisteredAt(ctx, tenantID, epEui)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", ErrReadRegistration, err)
	}
	if requested != nil && requested.After(since) {
		return requested, true, nil
	}
	return &since, true, nil
}
