// Package sessionreconcile hands the protocol sessions a previous process of
// this service center left live back to the resumable disconnected state when
// the service center starts: no connection holds them any more, and their
// peers may still resume them (BSSCI §3.3, SCACI §1).
package sessionreconcile

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// AbandonedSessionStore returns a service center's sessions left live by a
// previous process to the resumable disconnected state.
type AbandonedSessionStore interface {
	DisconnectAbandonedSessions(ctx context.Context, scEUI models.EUI) ([]int64, error)
}

// Reconciler reconciles one protocol's session rows. It touches only rows
// this service center owns.
type Reconciler struct {
	store             AbandonedSessionStore
	scEUI             models.EUI
	reconciledMessage string
	logger            logger.Logger
}

// New builds the reconciler for the service center serviceCenterEUI; it logs
// reconciledMessage when it reconciled sessions.
func New(store AbandonedSessionStore, serviceCenterEUI uint64, reconciledMessage string, log logger.Logger) (*Reconciler, error) {
	if store == nil {
		return nil, ErrNilAbandonedSessionStore
	}
	if log == nil {
		return nil, ErrNilReconcilerLogger
	}
	return &Reconciler{
		store:             store,
		scEUI:             models.EUI(mioty.EUI64(serviceCenterEUI).ToBytes()),
		reconciledMessage: reconciledMessage,
		logger:            log,
	}, nil
}

// ReconcileAbandonedSessions returns this service center's sessions left
// live by a previous process to the resumable disconnected state.
func (r *Reconciler) ReconcileAbandonedSessions(ctx context.Context) error {
	reconciled, err := r.store.DisconnectAbandonedSessions(ctx, r.scEUI)
	if err != nil {
		return fmt.Errorf("%w: %w", errReconcileAbandonedSessions, err)
	}
	if len(reconciled) > 0 {
		r.logger.InfoContext(ctx, r.reconciledMessage, logger.FieldCount, len(reconciled))
	}
	return nil
}
