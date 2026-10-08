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
// result it reported or its refusal of the queue, returning the row for the
// downlink's originators.
type DownlinkOutcomeWriter interface {
	UpdateDownlinkResult(ctx context.Context, tenantID int64, bsEUI uint64, result *mioty.DLDataResult) (*storage.DownlinkMessage, error)
	FailQueuedDownlink(ctx context.Context, queID int64, tenantID int64, bsEUI uint64, reason string) (*storage.DownlinkMessage, error)
}

// StationResultReporter tells a downlink's originators the result a base
// station gave for it, and files a "sent" a station reported for a downlink
// already reported expired, which its originators are not told again.
type StationResultReporter interface {
	ReportStationResult(ctx context.Context, downlink *storage.DownlinkMessage, result mioty.DLDataResult, station *bssci.Session)
	RecordSentAfterExpiry(ctx context.Context, ownerTenantID int64, result mioty.DLDataResult, station *bssci.Session)
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

// downlinkAnswers answers every downlink operation a base station takes part
// in: its results and queue answers, and, through the revoke answers, its
// answers to a dlDataRev.
type downlinkAnswers struct {
	*downlinkService
	RevokeAnswerer
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

// NewDownlinkService creates the downlink service over the revoke answers;
// every collaborator is mandatory, so a wiring fault surfaces at startup
// instead of on the first downlink result.
func NewDownlinkService(deps DownlinkServiceDeps, revokes RevokeAnswerer) (bssci.DownlinkService, error) {
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
	case revokes == nil:
		return nil, ErrNilRevokeAnswerer
	}
	return &downlinkAnswers{
		downlinkService: &downlinkService{
			logger:          deps.Logger,
			tenantResolver:  deps.Tenants,
			downlinks:       deps.Outcomes,
			holders:         deps.Holders,
			results:         deps.Results,
			queueSerializer: deps.Serializer,
			clock:           deps.Clock,
		},
		RevokeAnswerer: revokes,
	}, nil
}

// ProcessDLDataResult records the result a base station reported for a
// downlink (BSSCI §3.14) and reports it to the downlink's originators. A
// result for a downlink that already ended, expired or revoked while the
// station held it, leaves the outcome its originators were told; the first
// "sent" its holder reports for one already reported expired contradicts that
// report, so it is logged as a warning and filed in the owner's events. The
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
	case errors.Is(err, storage.ErrDownlinkSentAfterExpiry):
		d.logger.WarnContext(ctx, bssci.LogBSSCISentResultForExpiredDownlink,
			logger.FieldQueID, result.QueId, logger.FieldBsEui, session.BaseStationEUI, logger.FieldTenantIDCamel, ownerTenantID)
		d.results.RecordSentAfterExpiry(ctx, ownerTenantID, *result, session)
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
	ownerTenantID, err := parseOwner(ctx, d.logger, owner)
	if err != nil {
		return 0, 0, err
	}
	return queueID, ownerTenantID, nil
}

// parseOwner reads the owner tenant the resolver names for a queue id.
func parseOwner(ctx context.Context, log logger.Logger, owner string) (int64, error) {
	ownerTenantID, err := strconv.ParseInt(owner, 10, 64)
	if err != nil {
		log.ErrorContext(ctx, bssci.LogBSSCIInvalidTenantIDFormat,
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
