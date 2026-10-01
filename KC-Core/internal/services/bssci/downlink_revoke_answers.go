package bssciservices

import (
	"context"
	"slices"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// DownlinkRevocationWriter records a base station's answer to a dlDataRev
// (BSSCI §3.13, §3.17): it ends expired a downlink the station was asked to
// drop when its lifetime ended, returning it for its originators, and ends
// revoked one an operator or Application Center revoked.
type DownlinkRevocationWriter interface {
	ExpireRevokedDownlink(ctx context.Context, revocation storage.DownlinkRevocation) (*storage.DownlinkMessage, bool, error)
	RevokeDownlink(ctx context.Context, revocation storage.DownlinkRevocation) (bool, error)
}

// RevokeAnswerer handles a base station's answers to the service center's
// dlDataRev (BSSCI §3.13, §3.17).
type RevokeAnswerer interface {
	ProcessRevokeResponse(ctx context.Context, session *bssci.Session, opId int64, queueID int64, endpointEUI uint64) (map[string]interface{}, bool, error)
	ProcessRevokeRefusal(ctx context.Context, session *bssci.Session, refusal bssci.RevokeRefusal) (bool, error)
}

// RevokeAnswerDeps are the revoke answers' collaborators, all required.
// NotHeldCodes are the POSIX codes of a dlDataRev refusal that say the
// station does not hold the downlink (protocol.downlink_expiry.revoke_not_held_codes).
type RevokeAnswerDeps struct {
	Logger       logger.Logger
	Tenants      bssci.TenantResolver
	Revocations  DownlinkRevocationWriter
	Expiries     StationExpiryReporter
	Serializer   bssci.QueueSerializer
	NotHeldCodes []int
}

// revokeAnswers handles a base station's answers to the service center's
// dlDataRev: the confirmation and the error refusal.
type revokeAnswers struct {
	logger          logger.Logger
	tenantResolver  bssci.TenantResolver
	revocations     DownlinkRevocationWriter
	expiries        StationExpiryReporter
	notHeldCodes    []int
	queueSerializer bssci.QueueSerializer
}

// NewRevokeAnswers creates the revoke answers; every collaborator is
// mandatory, so a wiring fault surfaces at startup instead of on the first
// revoke answer. A refusal decides by its code alone, so at least one code
// must say the station does not hold the downlink.
func NewRevokeAnswers(deps RevokeAnswerDeps) (RevokeAnswerer, error) {
	switch {
	case deps.Logger == nil:
		return nil, ErrNilDownlinkServiceLogger
	case deps.Tenants == nil:
		return nil, ErrNilTenantResolver
	case deps.Revocations == nil:
		return nil, ErrNilDownlinkRevocationWriter
	case deps.Expiries == nil:
		return nil, ErrNilStationResultReporter
	case deps.Serializer == nil:
		return nil, ErrNilQueueSerializer
	case len(deps.NotHeldCodes) == 0:
		return nil, ErrNoRevokeNotHeldCodes
	}
	return &revokeAnswers{
		logger:          deps.Logger,
		tenantResolver:  deps.Tenants,
		revocations:     deps.Revocations,
		expiries:        deps.Expiries,
		notHeldCodes:    deps.NotHeldCodes,
		queueSerializer: deps.Serializer,
	}, nil
}

// ProcessRevokeResponse records a base station's confirmation of a dlDataRev
// (BSSCI §3.13) and answers with dlDataRevCmp; revoked reports whether the
// downlink became revoked. A downlink the station was asked to drop when its
// lifetime ended ends expired and is reported so; one that already ended
// keeps its outcome and is reported to nobody again.
func (r *revokeAnswers) ProcessRevokeResponse(ctx context.Context, session *bssci.Session, opId int64, queueID int64, endpointEUI uint64) (map[string]interface{}, bool, error) {
	if queueID <= 0 {
		r.logger.ErrorContext(ctx, bssci.LogBSSCIInvalidQueueIDInRevokeResponse,
			logger.FieldQueueID, queueID,
			logger.FieldOpID, opId)
		return nil, false, bssci.NewCatalogError(bssci.ErrInvalidQueueID, bssci.POSIX_EINVAL)
	}
	revoked, err := r.revoke(ctx, session, queueID, endpointEUI)
	if err != nil {
		return nil, false, err
	}
	return r.queueSerializer.BuildDLDataRevokeComplete(opId), revoked, nil
}

// ProcessRevokeRefusal records a base station's error answer to a dlDataRev
// (BSSCI §3.17). A refusal whose POSIX code says the station does not hold
// the downlink means it will never be transmitted, and it ends as a confirmed
// revoke would (§3.13). Any other refusal proves nothing about the downlink,
// which stays in flight until a result, another answer or a new session of
// the station settles it; the expiry sweep asks the station again.
func (r *revokeAnswers) ProcessRevokeRefusal(ctx context.Context, session *bssci.Session, refusal bssci.RevokeRefusal) (bool, error) {
	r.logger.WarnContext(ctx, bssci.LogBSSCIRevokeRefusedByStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldEpEui, refusal.EndpointEUI,
		logger.FieldQueID, refusal.QueueID,
		logger.FieldCode, refusal.Code)
	if !slices.Contains(r.notHeldCodes, refusal.Code) {
		r.logger.WarnContext(ctx, bssci.LogBSSCIRevokeRefusalKeepsDownlinkInFlight,
			logger.FieldQueID, refusal.QueueID, logger.FieldBsEui, session.BaseStationEUI, logger.FieldCode, refusal.Code)
		return false, nil
	}
	return r.revoke(ctx, session, refusal.QueueID, refusal.EndpointEUI)
}

// revoke records under its owner tenant that the answering station no longer
// holds the downlink, persisted before the operation completes (BSSCI §3.13),
// and reports whether it ended revoked.
func (r *revokeAnswers) revoke(ctx context.Context, session *bssci.Session, queueID int64, endpointEUI uint64) (bool, error) {
	ownerTenantID, err := r.owner(ctx, queueID)
	if err != nil {
		return false, err
	}
	station := session.BaseStationEUI
	r.logger.InfoContext(ctx, bssci.LogBSSCIProcessingRevokeResponse,
		logger.FieldBsEui, station,
		logger.FieldEpEui, endpointEUI,
		logger.FieldQueID, queueID,
		logger.FieldTenantIDCamel, ownerTenantID)
	r.tenantResolver.UnregisterQueueTenant(queueID)

	revocation := storage.DownlinkRevocation{QueID: queueID, TenantID: ownerTenantID, Station: &station}
	expired, ended, err := r.revocations.ExpireRevokedDownlink(ctx, revocation)
	if err != nil {
		return false, r.writeFailure(ctx, queueID, err)
	}
	if ended {
		r.expiries.ReportExpiredAtStation(ctx, expired)
		return false, nil
	}
	revoked, err := r.revocations.RevokeDownlink(ctx, revocation)
	if err != nil {
		return false, r.writeFailure(ctx, queueID, err)
	}
	if !revoked {
		r.logger.InfoContext(ctx, bssci.LogBSSCIRevokeAnswerForDownlinkNotHeld, logger.FieldQueID, queueID, logger.FieldBsEui, station)
	}
	return revoked, nil
}

// owner resolves the tenant owning the downlink a dlDataRev named.
func (r *revokeAnswers) owner(ctx context.Context, queueID int64) (int64, error) {
	owner, err := r.tenantResolver.ResolveTenant(ctx, queueID)
	if err != nil {
		r.logger.ErrorContext(ctx, bssci.LogBSSCICannotResolveTenantForRevoke,
			logger.FieldQueueID, queueID,
			logger.FieldError, err)
		return 0, bssci.NewCatalogError(bssci.ErrCannotResolveTenantForQueue, bssci.POSIX_EIO)
	}
	return parseOwner(ctx, r.logger, owner)
}

// writeFailure logs a revoke answer that could not be recorded and names the
// error the base station is answered with.
func (r *revokeAnswers) writeFailure(ctx context.Context, queueID int64, err error) error {
	r.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDownlinkAsRevoked,
		logger.FieldQueID, queueID,
		logger.FieldError, err)
	return bssci.NewCatalogError(bssci.ErrDatabaseUpdateFailed, bssci.POSIX_EIO)
}
