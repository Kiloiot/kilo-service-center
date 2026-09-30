package adapters

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// EndpointSessionTransactionAdapter owns the transaction lifecycle for endpoint
// attach operations, so the service layer expresses only what happens inside
// one.
type EndpointSessionTransactionAdapter struct {
	begin beginEndpointSessionTx
}

// NewEndpointSessionTransactionAdapter creates the adapter over the store.
func NewEndpointSessionTransactionAdapter(db *postgres.DB) *EndpointSessionTransactionAdapter {
	return &EndpointSessionTransactionAdapter{begin: endpointSessionBeginner(db)}
}

// Run executes fn inside a transaction; see runInTx.
func (a *EndpointSessionTransactionAdapter) Run(ctx context.Context, fn func(EndpointSessionOps) error) error {
	return runInTx(ctx, a.begin, func(h endpointSessionTxHandle) error { return fn(EndpointSessionOps{h: h}) })
}
