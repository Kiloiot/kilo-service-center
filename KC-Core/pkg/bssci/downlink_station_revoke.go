package bssci

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// RevokeDownlink implements DownlinkScheduler.RevokeDownlink to cancel a queued downlink
// message before delivery (SCACI §3.11 DL Data Revoke).
//
// A downlink still pending in the queue is held by no base station, so it is
// revoked in the queue alone and bsEui is zero. A downlink a base station
// holds (reserved or queued) is revoked there with dlDataRev (BSSCI §3.13);
// the row becomes revoked when the base station confirms. A reference that
// names an organization or endpoint reaches only that owner's downlink.
//
// Returns:
//   - bsEui: EUI of base station asked to revoke; zero when revoked in the queue
//   - error: scheduler.ErrSchedulerQueueNotFound if entry doesn't exist or already
//     finished, scheduler.ErrSchedulerResourceMissing if the holding BS is disconnected
func (s *Server) RevokeDownlink(ctx context.Context, ref scheduler.DownlinkRef) (bsEui uint64, err error) {
	// Bound the caller's context and carry the tenant for query scoping.
	ctx, cancel := context.WithTimeout(pkgcontext.WithTenantID(ctx, ref.TenantID), dbconfig.DefaultQueryTimeout)
	defer cancel()

	if ref.QueID > math.MaxInt64 {
		return 0, scheduler.ErrSchedulerQueueNotFound
	}
	revocation := storage.DownlinkRevocation{
		QueID:          int64(ref.QueID),
		TenantID:       ref.TenantID,
		OrganizationID: ref.OrganizationID,
		EpEUI:          ref.EpEUI,
	}
	revoked, err := s.downlinkRevoke.RevokeDownlink(ctx, revocation)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errDatabaseError), err)
	}
	if revoked {
		s.recordQueueRevoked(ctx, revocation)
		return 0, nil
	}
	return s.revokeAtBaseStation(ctx, revocation)
}

// recordQueueRevoked files the downlink a revoke ended in the queue under its
// owner; the revoke stands when the event cannot be recorded.
func (s *Server) recordQueueRevoked(ctx context.Context, revocation storage.DownlinkRevocation) {
	downlink, err := s.downlinkQueueStore.GetDownlinkByRevocation(ctx, revocation)
	if err == nil {
		err = s.auditLogger.RecordQueueRevoked(ctx, downlink)
	}
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToRecordDLDataRevokedEvent,
			logger.FieldQueID, revocation.QueID,
			logger.FieldTenantID, revocation.TenantID,
			logger.FieldError, err)
	}
}

// revokeAtBaseStation sends dlDataRev to the base station holding the
// downlink; a downlink that has already finished cannot be revoked.
func (s *Server) revokeAtBaseStation(ctx context.Context, revocation storage.DownlinkRevocation) (uint64, error) {
	downlink, err := s.downlinkQueueStore.GetDownlinkByRevocation(ctx, revocation)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrNotFound) {
		return 0, scheduler.ErrSchedulerQueueNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errDatabaseError), err)
	}
	if downlink.Status.Terminal() {
		return 0, scheduler.ErrSchedulerQueueNotFound
	}

	return downlink.BsEui, s.revokeAtHolder(downlink, revocation.TenantID)
}

// RevokeHeldDownlink asks the base station holding a downlink that already
// ended, such as one that expired, to drop it (BSSCI §3.13); the station's
// confirmation leaves the downlink's outcome as it is.
//
// Returns:
//   - error: scheduler.ErrSchedulerResourceMissing if the holding base station
//     is disconnected, or the send failure
func (s *Server) RevokeHeldDownlink(_ context.Context, downlink *storage.DownlinkMessage) error {
	ownerTenantID, err := strconv.ParseInt(downlink.TenantID, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(ErrInvalidTenantIDFormat), err)
	}
	return s.revokeAtHolder(downlink, ownerTenantID)
}

// revokeAtHolder sends dlDataRev for the owner tenant's downlink to the base
// station holding it.
func (s *Server) revokeAtHolder(downlink *storage.DownlinkMessage, ownerTenantID int64) error {
	epEUI, err := validation.ParseEUI(downlink.EPEUI)
	if err != nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errInvalidEndpointEUIFormat))
	}
	queID, ok := downlink.WireQueueID()
	if !ok {
		return fmt.Errorf("%s", ResolveErrorMessage(errInvalidQueueID))
	}
	session := s.sessions.byEUI(downlink.BsEui, nil)
	if session == nil {
		return scheduler.ErrSchedulerResourceMissing
	}
	if err := s.SendDLDataRevoke(session.ID, epEUI, queID, ownerTenantID); err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlDataRev), err)
	}
	return nil
}
