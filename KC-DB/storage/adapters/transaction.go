package adapters

import (
	"context"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// Transaction lifecycle sentinels. Callers map these onto their own error
// vocabulary while errors.Is still reaches the underlying storage sentinel.
var (
	// ErrTxBegin reports that a transaction could not be opened.
	ErrTxBegin = errors.New("begin transaction")
	// ErrTxCommit reports that a transaction could not be committed. The work
	// inside the transaction succeeded; only the commit failed.
	ErrTxCommit = errors.New("commit transaction")
	// ErrTxRollback reports that an open transaction could not be rolled back,
	// which can leave its row locks held until the connection is released.
	ErrTxRollback = errors.New("rollback transaction")
)

// Each adapter drives its own transaction handle: the repository accessors its
// one operation needs, plus the lifecycle. The begin functions are functions
// rather than interfaces because *postgres.DB.BeginTx returns the concrete
// transaction type, and Go has no covariant returns - an interface declaring a
// handle return would not be satisfied by it. Wrapping the call site converts
// once, and lets the commit and rollback paths be exercised without a database.

// endpointSessionTxHandle is the transaction surface an attach operation opens:
// the repository accessors it reaches through, plus the lifecycle. The flattened
// EndpointSessionOps a caller receives are projected from it.
type endpointSessionTxHandle interface {
	EndPoints() interfaces.EndpointRepository
	UplinkClassifier() interfaces.UplinkClassifierReset
	EndPointSessions() interfaces.EndPointSessionRepository
	Commit() error
	Rollback() error
}

type beginEndpointSessionTx func(ctx context.Context) (endpointSessionTxHandle, error)

func endpointSessionBeginner(db *postgres.DB) beginEndpointSessionTx {
	return func(ctx context.Context) (endpointSessionTxHandle, error) {
		return db.BeginTx(ctx)
	}
}

// EndpointSessionOps projects the handle's repository accessors down to the
// exact operations an attach or attach-propagate transaction performs, so a
// caller reaches only those, never a whole repository, and cannot commit or
// roll back out of band.
type EndpointSessionOps struct {
	h endpointSessionTxHandle
}

// EndpointAttachmentStateUpdate records the endpoint fields of an attach.
func (o EndpointSessionOps) EndpointAttachmentStateUpdate(ctx context.Context, tenantID, endpointID int64,
	p models.EndpointAttachmentStateParams) error {
	return o.h.EndPoints().EndpointAttachmentStateUpdate(ctx, tenantID, endpointID, p)
}

// EndpointAttachSessionUpdate records the endpoint fields of an attach propagate.
func (o EndpointSessionOps) EndpointAttachSessionUpdate(ctx context.Context, tenantID, endpointID int64,
	p models.EndpointAttachSessionParams) error {
	return o.h.EndPoints().EndpointAttachSessionUpdate(ctx, tenantID, endpointID, p)
}

// RestartPacketCounter zeroes the endpoint's counters and forgets its uplink
// classifier rows in the same transaction (radio protocol §3.6.5.3).
func (o EndpointSessionOps) RestartPacketCounter(ctx context.Context, tenantID, endpointID int64) error {
	if err := o.h.EndPoints().RestartPacketCounter(ctx, tenantID, endpointID); err != nil {
		return err
	}
	return o.h.UplinkClassifier().ForgetEndpoint(ctx, tenantID, endpointID)
}

// LockAttachCounter locks the endpoint row and reads its stored attach counter.
func (o EndpointSessionOps) LockAttachCounter(ctx context.Context, tenantID, endpointID int64) (*uint32, error) {
	return o.h.EndPoints().LockAttachCounter(ctx, tenantID, endpointID)
}

// GetActiveSession reads the endpoint session in force.
func (o EndpointSessionOps) GetActiveSession(ctx context.Context, endpointID string) (*models.EndPointSession, error) {
	return o.h.EndPointSessions().GetActive(ctx, endpointID)
}

// UpdateSession rewrites the endpoint session in force.
func (o EndpointSessionOps) UpdateSession(ctx context.Context, session *models.EndPointSession) error {
	return o.h.EndPointSessions().Update(ctx, session)
}

// CreateSession starts a new endpoint session.
func (o EndpointSessionOps) CreateSession(ctx context.Context, session *models.EndPointSession) error {
	return o.h.EndPointSessions().Create(ctx, session)
}

// blueprintTxHandle is the transaction surface a blueprint creation opens.
type blueprintTxHandle interface {
	DeviceModels() interfaces.DeviceModelRepository
	Blueprints() interfaces.BlueprintRepository
	WithSavepoint(ctx context.Context, fn func() error) error
	Commit() error
	Rollback() error
}

type beginBlueprintTx func(ctx context.Context) (blueprintTxHandle, error)

func blueprintBeginner(db *postgres.DB) beginBlueprintTx {
	return func(ctx context.Context) (blueprintTxHandle, error) {
		return db.BeginTx(ctx)
	}
}

// blueprintTxOps projects the handle's repository accessors down to the two
// creates a device-model-plus-blueprint transaction performs.
type blueprintTxOps struct {
	h blueprintTxHandle
}

func (o blueprintTxOps) CreateDeviceModel(ctx context.Context,
	params *models.DeviceModelCreateParams) (*models.DeviceModel, error) {
	// Each candidate insert runs inside a private savepoint: PostgreSQL aborts
	// the whole transaction on a failed statement, so without the savepoint a
	// duplicate-code collision would poison every retry with 25P02. The
	// service retries with a suffixed slug inside the same transaction.
	var model *models.DeviceModel
	err := o.h.WithSavepoint(ctx, func() error {
		created, createErr := o.h.DeviceModels().Create(ctx, params)
		if createErr != nil {
			return createErr
		}
		model = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return model, nil
}

func (o blueprintTxOps) CreateBlueprint(ctx context.Context,
	params *models.BlueprintCreateParams) (*models.Blueprint, error) {
	return o.h.Blueprints().Create(ctx, params)
}

// downlinkTxHandle is the transaction surface of a downlink reservation.
type downlinkTxHandle interface {
	MIOTYDownlinks() interfaces.MIOTYDownlinkTxRepository
	Commit() error
	Rollback() error
}

type beginDownlinkTx func(ctx context.Context) (downlinkTxHandle, error)

func downlinkBeginner(db *postgres.DB) beginDownlinkTx {
	return func(ctx context.Context) (downlinkTxHandle, error) {
		return db.BeginTx(ctx)
	}
}

// Compile-time contract: the flattened projection satisfies the adapter port.
var _ BlueprintTx = blueprintTxOps{}
