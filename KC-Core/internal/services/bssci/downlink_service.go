package bssciservices

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// DownlinkOutcomeWriter records how a downlink a base station held ended: the
// result it reported, its refusal of the queue, or its revocation. The first
// two return the row for the downlink's originators.
type DownlinkOutcomeWriter interface {
	UpdateDownlinkResult(ctx context.Context, tenantID int64, bsEUI uint64, result *mioty.DLDataResult) (*storage.DownlinkMessage, error)
	FailQueuedDownlink(ctx context.Context, queID int64, tenantID int64, bsEUI uint64, reason string) (*storage.DownlinkMessage, error)
	RevokeDownlink(ctx context.Context, revocation storage.DownlinkRevocation) (bool, error)
}

// StationResultReporter tells a downlink's originators the result a base
// station gave for it.
type StationResultReporter interface {
	ReportStationResult(ctx context.Context, downlink *storage.DownlinkMessage, result mioty.DLDataResult, station *bssci.Session)
}

// DownlinkServiceDeps are the downlink service's collaborators, all required.
type DownlinkServiceDeps struct {
	Logger     logger.Logger
	Tenants    bssci.TenantResolver
	Outcomes   DownlinkOutcomeWriter
	Holders    DownlinkHolderWriter
	Results    StationResultReporter
	Serializer bssci.QueueSerializer
	Clock      clock.Clock
}

type downlinkService struct {
	logger          logger.Logger
	tenantResolver  bssci.TenantResolver
	downlinks       DownlinkOutcomeWriter
	holders         DownlinkHolderWriter
	results         StationResultReporter
	queueSerializer bssci.QueueSerializer
	clock           clock.Clock
}

// NewDownlinkService creates the downlink service; every collaborator is
// mandatory, so a wiring fault surfaces at startup instead of on the first
// downlink result.
func NewDownlinkService(deps DownlinkServiceDeps) (bssci.DownlinkService, error) {
	switch {
	case deps.Logger == nil:
		return nil, ErrNilDownlinkServiceLogger
	case deps.Tenants == nil:
		return nil, ErrNilTenantResolver
	case deps.Outcomes == nil:
		return nil, ErrNilDownlinkWriter
	case deps.Holders == nil:
		return nil, ErrNilDownlinkHolderWriter
	case deps.Results == nil:
		return nil, ErrNilStationResultReporter
	case deps.Serializer == nil:
		return nil, ErrNilQueueSerializer
	case deps.Clock == nil:
		return nil, ErrNilDownlinkServiceClock
	}
	return &downlinkService{
		logger:          deps.Logger,
		tenantResolver:  deps.Tenants,
		downlinks:       deps.Outcomes,
		holders:         deps.Holders,
		results:         deps.Results,
		queueSerializer: deps.Serializer,
		clock:           deps.Clock,
	}, nil
}

// ProcessDLDataResult records the result a base station reported for a
// downlink (BSSCI §3.14) and reports it to the downlink's originators. A
// result for a downlink that already ended, expired or revoked while the
// station held it, leaves the outcome its originators were told; the
// station's operation completes either way.
func (d *downlinkService) ProcessDLDataResult(ctx context.Context, session *bssci.Session, result *mioty.DLDataResult) (map[string]interface{}, error) {
	queueID, ownerTenantID, err := d.resultOwner(ctx, result)
	if err != nil {
		return nil, err
	}
	d.logger.InfoContext(ctx, bssci.LogBSSCIReceivedDLDataResFromBaseStation, withTransmissionFields([]interface{}{
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldEpEui, result.EpEui,
		logger.FieldQueID, result.QueId,
		logger.FieldResult, result.Result,
		logger.FieldTenantIDCamel, ownerTenantID,
	}, result)...)

	// The endpoint, the owner tenant and the holding station are part of the match (BSSCI §3.14).
	downlink, err := d.downlinks.UpdateDownlinkResult(ctx, ownerTenantID, session.BaseStationEUI, result)
	switch {
	case errors.Is(err, storage.ErrDownlinkFinished):
		d.logger.InfoContext(ctx, bssci.LogBSSCIResultForFinishedDownlink,
			logger.FieldQueID, result.QueId, logger.FieldResult, result.Result)
	case err != nil:
		return nil, d.resultWriteFailure(ctx, result, err)
	default:
		d.logger.DebugContext(ctx, bssci.LogBSSCIUpdatedDownlinkResult, withTransmissionFields([]interface{}{
			logger.FieldQueID, result.QueId,
			logger.FieldResult, result.Result,
		}, result)...)
		d.results.ReportStationResult(ctx, downlink, *result, session)
	}
	d.tenantResolver.UnregisterQueueTenant(queueID)
	return d.queueSerializer.BuildDLDataResultResponse(result.OpId, result), nil
}

// resultOwner resolves the tenant owning the downlink a dlDataRes names and
// returns the queue id as the queue stores it.
func (d *downlinkService) resultOwner(ctx context.Context, result *mioty.DLDataResult) (int64, int64, error) {
	if result.QueId > math.MaxInt64 {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIQueueIDOutOfRange,
			logger.FieldQueID, result.QueId,
			logger.FieldMaxInt64, math.MaxInt64)
		return 0, 0, bssci.NewCatalogError(bssci.ErrQueueIDOutOfRange, bssci.POSIX_ERANGE)
	}
	queueID := int64(result.QueId)
	owner, err := d.tenantResolver.ResolveTenant(ctx, queueID)
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCICannotResolveTenantForDownlinkResult,
			logger.FieldQueID, result.QueId,
			logger.FieldError, err)
		return 0, 0, bssci.NewCatalogError(bssci.ErrCannotResolveTenantForQueue, bssci.POSIX_EPROTO)
	}
	ownerTenantID, err := d.parseOwner(ctx, owner)
	if err != nil {
		return 0, 0, err
	}
	return queueID, ownerTenantID, nil
}

// parseOwner reads the owner tenant the resolver names for a queue id.
func (d *downlinkService) parseOwner(ctx context.Context, owner string) (int64, error) {
	ownerTenantID, err := strconv.ParseInt(owner, 10, 64)
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIInvalidTenantIDFormat,
			logger.FieldTenantStr, owner,
			logger.FieldError, err)
		return 0, bssci.NewCatalogError(bssci.ErrInvalidTenantIDFormat, bssci.POSIX_EINVAL)
	}
	return ownerTenantID, nil
}

// resultWriteFailure logs a result that could not be recorded and names the
// error the base station is answered with.
func (d *downlinkService) resultWriteFailure(ctx context.Context, result *mioty.DLDataResult, err error) error {
	d.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDownlinkResult,
		logger.FieldQueID, result.QueId,
		logger.FieldResult, result.Result,
		logger.FieldError, err)
	if errors.Is(err, storage.ErrDownlinkNotFound) {
		return bssci.NewCatalogError(bssci.ErrQueueIDNotFound, bssci.POSIX_EPROTO)
	}
	return bssci.NewCatalogError(bssci.ErrDatabaseUpdateFailed, bssci.POSIX_EIO)
}

// withTransmissionFields appends the optional txTime and packetCnt a
// dlDataRes carries (BSSCI §3.14.1) as values, leaving out absent ones.
func withTransmissionFields(fields []interface{}, result *mioty.DLDataResult) []interface{} {
	if result.TxTime != nil {
		fields = append(fields, logger.FieldTxTime, *result.TxTime)
	}
	if result.PacketCnt != nil {
		fields = append(fields, logger.FieldPacketCnt, *result.PacketCnt)
	}
	return fields
}

// ProcessRevokeResponse records a base station's confirmation of a dlDataRev
// (BSSCI §3.13) and answers with dlDataRevCmp; revoked reports whether the
// downlink became revoked. A downlink that already ended, such as one the
// service center expired before asking its station to drop it, keeps its
// outcome, so the confirmation reports nothing a second time.
func (d *downlinkService) ProcessRevokeResponse(ctx context.Context, session *bssci.Session, opId int64, queueID int64, endpointEUI uint64) (map[string]interface{}, bool, error) {
	if queueID <= 0 {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIInvalidQueueIDInRevokeResponse,
			logger.FieldQueueID, queueID,
			logger.FieldOpID, opId)
		return nil, false, bssci.NewCatalogError(bssci.ErrInvalidQueueID, bssci.POSIX_EINVAL)
	}
	revoked, err := d.revoke(ctx, session, queueID, endpointEUI)
	if err != nil {
		return nil, false, err
	}
	return d.queueSerializer.BuildDLDataRevokeComplete(opId), revoked, nil
}

// ProcessRevokeRefusal records a base station's error answer to a dlDataRev
// (BSSCI §3.17): the station does not hold the downlink, so it will never be
// transmitted, and it ends revoked as a confirmed revoke would (§3.13). A
// downlink that already ended, expired by the sweep or finished by a late
// dlDataRes, keeps its outcome and is reported to nobody again.
func (d *downlinkService) ProcessRevokeRefusal(ctx context.Context, session *bssci.Session, refusal bssci.RevokeRefusal) (bool, error) {
	d.logger.WarnContext(ctx, bssci.LogBSSCIRevokeRefusedByStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldEpEui, refusal.EndpointEUI,
		logger.FieldQueID, refusal.QueueID,
		logger.FieldCode, refusal.Code,
		logger.FieldMessage, refusal.Message)
	return d.revoke(ctx, session, refusal.QueueID, refusal.EndpointEUI)
}

// revoke records the answering station's downlink revoked under its owner
// tenant, persisted before the operation completes (BSSCI §3.13), and
// reports whether the station still held it in flight.
func (d *downlinkService) revoke(ctx context.Context, session *bssci.Session, queueID int64, endpointEUI uint64) (bool, error) {
	owner, err := d.tenantResolver.ResolveTenant(ctx, queueID)
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCICannotResolveTenantForRevoke,
			logger.FieldQueueID, queueID,
			logger.FieldError, err)
		return false, bssci.NewCatalogError(bssci.ErrCannotResolveTenantForQueue, bssci.POSIX_EIO)
	}
	ownerTenantID, err := d.parseOwner(ctx, owner)
	if err != nil {
		return false, err
	}
	station := session.BaseStationEUI
	d.logger.InfoContext(ctx, bssci.LogBSSCIProcessingRevokeResponse,
		logger.FieldBsEui, station,
		logger.FieldEpEui, endpointEUI,
		logger.FieldQueID, queueID,
		logger.FieldTenantIDCamel, ownerTenantID)
	d.tenantResolver.UnregisterQueueTenant(queueID)

	revoked, err := d.downlinks.RevokeDownlink(ctx, storage.DownlinkRevocation{QueID: queueID, TenantID: ownerTenantID, Station: &station})
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDownlinkAsRevoked,
			logger.FieldQueID, queueID,
			logger.FieldError, err)
		return false, bssci.NewCatalogError(bssci.ErrDatabaseUpdateFailed, bssci.POSIX_EIO)
	}
	if !revoked {
		d.logger.InfoContext(ctx, bssci.LogBSSCIRevokeAnswerForDownlinkNotHeld, logger.FieldQueID, queueID, logger.FieldBsEui, station)
	}
	return revoked, nil
}
