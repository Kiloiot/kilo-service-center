package adapters

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// BlueprintTx is the transaction-scoped operation set used when a device model
// and its blueprint are created together, which must succeed or fail as a unit.
type BlueprintTx interface {
	CreateDeviceModel(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error)
	CreateBlueprint(ctx context.Context, params *models.BlueprintCreateParams) (*models.Blueprint, error)
}

// BlueprintTransactionAdapter owns the transaction lifecycle for blueprint
// creation, so the blueprint service no longer opens or commits one itself.
type BlueprintTransactionAdapter struct {
	begin beginBlueprintTx
}

// NewBlueprintTransactionAdapter creates the adapter over the store.
func NewBlueprintTransactionAdapter(db *postgres.DB) *BlueprintTransactionAdapter {
	return &BlueprintTransactionAdapter{begin: blueprintBeginner(db)}
}

// Run executes fn inside a transaction; see runInTx.
func (a *BlueprintTransactionAdapter) Run(ctx context.Context, fn func(BlueprintTx) error) error {
	return runInTx(ctx, a.begin, func(h blueprintTxHandle) error { return fn(blueprintTxOps{h: h}) })
}
