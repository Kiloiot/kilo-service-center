package adapters

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// SCACISessionTx is what the creation of a fresh SCACI session does in one
// transaction: retire the application center's earlier sessions, then insert
// the new one (SCACI §1).
type SCACISessionTx interface {
	RetirePriorSessions(ctx context.Context, ac models.SCACIApplicationCenter) error
	CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error)
}

// SCACISessionTransactionAdapter owns the transaction lifecycle of a SCACI
// session creation, so the session persistence expresses only what happens
// inside one.
type SCACISessionTransactionAdapter struct {
	begin beginSCACISessionTx
}

// NewSCACISessionTransactionAdapter creates the adapter over the store.
func NewSCACISessionTransactionAdapter(db *postgres.DB) *SCACISessionTransactionAdapter {
	return &SCACISessionTransactionAdapter{begin: scaciSessionBeginner(db)}
}

// Run executes fn inside a transaction; see runInTx.
func (a *SCACISessionTransactionAdapter) Run(ctx context.Context, fn func(SCACISessionTx) error) error {
	return runInTx(ctx, a.begin, func(h scaciSessionTxHandle) error { return fn(scaciSessionTxOps{h: h}) })
}

// scaciSessionTxHandle is the transaction surface a SCACI session creation
// opens.
type scaciSessionTxHandle interface {
	SCACISessions() interfaces.SCACISessionRepository
	Commit() error
	Rollback() error
}

type beginSCACISessionTx func(ctx context.Context) (scaciSessionTxHandle, error)

func scaciSessionBeginner(db *postgres.DB) beginSCACISessionTx {
	return func(ctx context.Context) (scaciSessionTxHandle, error) {
		return db.BeginTx(ctx)
	}
}

// scaciSessionTxOps projects the handle down to the two writes a session
// creation performs.
type scaciSessionTxOps struct {
	h scaciSessionTxHandle
}

func (o scaciSessionTxOps) RetirePriorSessions(ctx context.Context, ac models.SCACIApplicationCenter) error {
	return o.h.SCACISessions().RetirePriorSessions(ctx, ac)
}

func (o scaciSessionTxOps) CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	return o.h.SCACISessions().CreateSession(ctx, req)
}

var _ SCACISessionTx = scaciSessionTxOps{}
