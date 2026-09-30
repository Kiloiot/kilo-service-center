package bssciservices

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// AttachCounterTx serializes the over-the-air attaches of one endpoint and
// restarts the counters an attach restarts (radio protocol §3.6.5.3).
type AttachCounterTx interface {
	LockAttachCounter(ctx context.Context, tenantID, endpointID int64) (*uint32, error)
	RestartPacketCounter(ctx context.Context, tenantID, endpointID int64) error
}

// AttachStateTx records the endpoint fields of an attach or attach propagate.
type AttachStateTx interface {
	EndpointAttachmentStateUpdate(ctx context.Context, tenantID, endpointID int64, p models.EndpointAttachmentStateParams) error
	EndpointAttachSessionUpdate(ctx context.Context, tenantID, endpointID int64, p models.EndpointAttachSessionParams) error
}

// SessionRowTx reads and writes the endpoint's active session row.
type SessionRowTx interface {
	GetActiveSession(ctx context.Context, endpointID string) (*models.EndPointSession, error)
	UpdateSession(ctx context.Context, session *models.EndPointSession) error
	CreateSession(ctx context.Context, session *models.EndPointSession) error
}

// attachWriteTx is the part of an attach transaction that writes the endpoint.
type attachWriteTx interface {
	AttachCounterTx
	AttachStateTx
}

// EndpointSessionTx is the transaction-scoped operation set of one attach or
// attach propagate: the endpoint fields and the endpoint session move as a
// unit or not at all.
type EndpointSessionTx interface {
	AttachCounterTx
	AttachStateTx
	SessionRowTx
}

// Transaction lifecycle sentinels owned by this service. The composition root
// maps the storage adapter's lifecycle failures onto these, so the service and
// its callers never depend on the storage package's error identities.
var (
	ErrTxBegin  = errors.New("attach transaction begin")
	ErrTxCommit = errors.New("attach transaction commit")
)

// EndpointSessionTransactionRunner runs one attach operation inside a
// transaction. Storage owns begin, commit and rollback; this service only says
// what happens between them.
type EndpointSessionTransactionRunner interface {
	Run(ctx context.Context, fn func(EndpointSessionTx) error) error
}

// PrimaryStationLookup finds the base station an endpoint session records as
// its primary station.
type PrimaryStationLookup interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
}

// endpointAttachmentPersistence is the transactional owner of attach and
// attach-propagate endpoint-session persistence (BSSCI rev1 §5.7-§5.8 /
// classic §3.7-§3.8).
type endpointAttachmentPersistence struct {
	tx       EndpointSessionTransactionRunner
	stations PrimaryStationLookup
	clock    clock.Clock
	logger   logger.Logger
}

// NewEndpointAttachmentPersistence creates the persister over a transaction
// runner; the station lookup serves the out-of-transaction primary station.
func NewEndpointAttachmentPersistence(tx EndpointSessionTransactionRunner, stations PrimaryStationLookup, clk clock.Clock, log logger.Logger) (bssci.EndpointAttachmentPersistence, error) {
	switch {
	case tx == nil:
		return nil, errNilAttachTransactionRunner
	case stations == nil:
		return nil, errNilPrimaryStationLookup
	case clk == nil:
		return nil, errNilAttachPersistenceClock
	case log == nil:
		return nil, errNilAttachPersistenceLogger
	}
	return &endpointAttachmentPersistence{tx: tx, stations: stations, clock: clk, logger: log}, nil
}

// txLifecycleError maps a storage transaction lifecycle failure onto the
// service's own message, keeping the underlying sentinel matchable.
func txLifecycleError(err error, beginMsg, commitMsg string) error {
	switch {
	case errors.Is(err, ErrTxBegin):
		return fmt.Errorf("%s: %w", beginMsg, err)
	case errors.Is(err, ErrTxCommit):
		return fmt.Errorf("%s: %w", commitMsg, err)
	default:
		// Not a lifecycle failure: the callback already wrapped its own error
		// with the failing operation's context.
		return err // Preserves sentinel
	}
}

// primaryStationID is the id of the base station an endpoint session names as
// primary; nil when the station is unknown to the tenant.
func (p *endpointAttachmentPersistence) primaryStationID(ctx context.Context, tenantID int64, bsEUI []byte) *int64 {
	bs, err := p.stations.GetByEUI(ctx, tenantID, bsEUI)
	if err != nil || bs == nil {
		return nil
	}
	return &bs.ID
}

// PersistAttachSession runs the attach transaction: endpoint update and
// counter restart, endpoint-session upsert (attach counter, short address).
func (p *endpointAttachmentPersistence) PersistAttachSession(ctx context.Context, rec bssci.AttachSessionRecord) error {
	err := p.tx.Run(ctx, func(tx EndpointSessionTx) error {
		if err := p.attachEndpoint(ctx, tx, rec); err != nil {
			return err // Pass through: attachEndpoint names the failing step
		}

		endpointIDStr := fmt.Sprintf("%d", rec.EndpointID)
		activeSession, getErr := tx.GetActiveSession(ctx, endpointIDStr)
		if getErr != nil && !errors.Is(getErr, storage.ErrNotFound) {
			p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToLoadEndpointSession, logger.FieldError, getErr)
			return fmt.Errorf("%s: %w", bssci.LogBSSCIFailedToLoadEndpointSession, getErr)
		}

		now := p.clock.Now().UTC()
		primaryBsID := p.primaryStationID(ctx, rec.BSLookupTenantID, rec.BaseStationEUI)

		if activeSession != nil {
			rec.ApplyToSession(activeSession, now, primaryBsID)
			if err := tx.UpdateSession(ctx, activeSession); err != nil {
				p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateEndpointSession, logger.FieldError, err)
				return fmt.Errorf("%s: %w", bssci.LogBSSCIFailedToUpdateEndpointSession, err)
			}
			return nil
		}

		if err := tx.CreateSession(ctx, rec.NewSession(now, primaryBsID)); err != nil {
			p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToCreateEndpointSession, logger.FieldError, err)
			return fmt.Errorf("%s: %w", bssci.LogBSSCIFailedToCreateEndpointSession, err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrTxBegin) {
			p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToBeginTransaction, logger.FieldError, err)
		} else if errors.Is(err, ErrTxCommit) {
			p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToCommitAttachTransaction, logger.FieldError, err)
		}
		// txLifecycleError wraps with %w or passes the callback's
		// already-wrapped error through unchanged.
		return txLifecycleError(err, // Preserves sentinel
			bssci.LogBSSCIFailedToBeginTransaction,
			bssci.LogBSSCIFailedToCommitAttachTransaction)
	}
	return nil
}

// attachEndpoint writes the attach state and the counter restart an over-the-air attach implies (radio protocol §3.6.5.3).
// The endpoint row lock serializes concurrent attaches, so only one of two identical att frames is served (radio §3.7.1.2).
func (p *endpointAttachmentPersistence) attachEndpoint(ctx context.Context, tx attachWriteTx, rec bssci.AttachSessionRecord) error {
	stored, err := tx.LockAttachCounter(ctx, rec.TenantID, rec.EndpointID)
	if err != nil {
		p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToLockAttachCounter, logger.FieldError, err)
		return fmt.Errorf("%s: %w", bssci.LogBSSCIFailedToLockAttachCounter, err)
	}
	if !bssci.AttachCounterAdvances(stored, int64(rec.AttachCnt)) {
		return bssci.ErrAttachCounterStale
	}
	if err := tx.EndpointAttachmentStateUpdate(ctx, rec.TenantID, rec.EndpointID, rec.EndpointUpdates); err != nil {
		p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateEndpointAttachMetadata, logger.FieldError, err)
		return fmt.Errorf("%s: %w", bssci.LogBSSCIFailedToUpdateEndpointAttachMetadata, err)
	}
	if err := tx.RestartPacketCounter(ctx, rec.TenantID, rec.EndpointID); err != nil {
		p.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToRestartPacketCounter, logger.FieldError, err)
		return fmt.Errorf("%s: %w", bssci.LogBSSCIFailedToRestartPacketCounter, err)
	}
	return nil
}

// PersistAttachPropagateSession runs the attach-propagate transaction under
// the endpoint owner tenant; error strings propagate to the caller unchanged.
func (p *endpointAttachmentPersistence) PersistAttachPropagateSession(ctx context.Context, rec bssci.AttachPropagateSessionRecord) error {
	err := p.tx.Run(ctx, func(tx EndpointSessionTx) error {
		if err := tx.EndpointAttachSessionUpdate(ctx, rec.TenantID, rec.EndpointID, rec.EndpointUpdates); err != nil {
			return fmt.Errorf("%w: %w", errUpdateEndpoint, err)
		}

		endpointIDStr := fmt.Sprintf("%d", rec.EndpointID)
		activeSession, getErr := tx.GetActiveSession(ctx, endpointIDStr)
		if getErr != nil && !errors.Is(getErr, storage.ErrNotFound) {
			return fmt.Errorf("%w: %w", errLoadEndpointSession, getErr)
		}

		now := p.clock.Now().UTC()
		primaryBsID := p.primaryStationID(ctx, rec.TenantID, rec.BaseStationEUI)

		if activeSession != nil {
			rec.ApplyToSession(activeSession, now, primaryBsID)
			if err := tx.UpdateSession(ctx, activeSession); err != nil {
				return fmt.Errorf("%w: %w", errUpdateEndpointSession, err)
			}
			return nil
		}

		if err := tx.CreateSession(ctx, rec.NewSession(now, primaryBsID)); err != nil {
			return fmt.Errorf("%w: %w", errCreateEndpointSession, err)
		}
		return nil
	})
	// txLifecycleError wraps with %w or passes the callback's
	// already-wrapped error through unchanged.
	return txLifecycleError(err, // Preserves sentinel
		bssci.LogBSSCIFailedToBeginAttachPropagateTransaction,
		bssci.LogBSSCIFailedToCommitAttachPropagateTransaction)
}
