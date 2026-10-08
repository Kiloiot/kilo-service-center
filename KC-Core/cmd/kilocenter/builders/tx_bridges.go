package builders

import (
	"context"
	"errors"
	"fmt"

	blueprintservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/blueprints"
	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	scaciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
)

// The transaction bridges present the storage transaction adapters through the
// ports each service declares for itself, translating the storage lifecycle
// sentinels into the service-owned ones so the services never depend on the
// storage package's error identities. The transaction-scoped operation sets are
// structurally identical, so the adapter's transaction value satisfies the
// service port directly.

// blueprintTxRun runs a blueprint creation inside a storage transaction. It is
// the seam the bridge is built over so the lifecycle mapping can be exercised
// against injected begin, commit and rollback failures without a database. In
// production it is the transaction adapter's Run method.
type blueprintTxRun func(ctx context.Context, fn func(adapters.BlueprintTx) error) error

// endpointSessionTxRun is the same seam for an attach or attach-propagate
// transaction.
type endpointSessionTxRun func(ctx context.Context, fn func(adapters.EndpointSessionOps) error) error

// scaciSessionTxRun is the same seam for a SCACI session creation.
type scaciSessionTxRun func(ctx context.Context, fn func(adapters.SCACISessionTx) error) error

// blueprintTxBridge adapts the blueprint transaction runner to the blueprint
// service's transaction runner port.
type blueprintTxBridge struct {
	run blueprintTxRun
}

func (b blueprintTxBridge) Run(ctx context.Context, fn func(blueprintservices.BlueprintTx) error) error {
	err := b.run(ctx, func(tx adapters.BlueprintTx) error { return fn(tx) })
	return serviceTxError(err, blueprintservices.ErrTxBegin, blueprintservices.ErrTxCommit)
}

// endpointSessionTxBridge adapts the endpoint session transaction runner to
// the attachment persistence's transaction runner port.
type endpointSessionTxBridge struct {
	run endpointSessionTxRun
}

func (b endpointSessionTxBridge) Run(ctx context.Context, fn func(bssciservices.EndpointSessionTx) error) error {
	err := b.run(ctx, func(tx adapters.EndpointSessionOps) error { return fn(tx) })
	return serviceTxError(err, bssciservices.ErrTxBegin, bssciservices.ErrTxCommit)
}

// scaciSessionTxBridge adapts the SCACI session transaction runner to the
// session persistence's session creation port.
type scaciSessionTxBridge struct {
	run scaciSessionTxRun
}

func (b scaciSessionTxBridge) Run(ctx context.Context, fn func(scaciservices.SessionCreationTx) error) error {
	err := b.run(ctx, func(tx adapters.SCACISessionTx) error { return fn(tx) })
	return serviceTxError(err, scaciservices.ErrTxBegin, scaciservices.ErrTxCommit)
}

// serviceTxError maps the storage lifecycle sentinels onto a service's own,
// keeping every underlying cause matchable.
func serviceTxError(err, serviceBegin, serviceCommit error) error {
	switch {
	case errors.Is(err, adapters.ErrTxBegin):
		return fmt.Errorf("%w: %w", serviceBegin, err)
	case errors.Is(err, adapters.ErrTxCommit):
		return fmt.Errorf("%w: %w", serviceCommit, err)
	default:
		return err
	}
}

// Compile-time contracts: the bridges implement the service ports.
var (
	_ blueprintservices.BlueprintTransactionRunner   = blueprintTxBridge{}
	_ bssciservices.EndpointSessionTransactionRunner = endpointSessionTxBridge{}
	_ scaciservices.SessionCreationRunner            = scaciSessionTxBridge{}
)
