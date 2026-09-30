package bssciservices

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	pkgblueprint "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
)

// UplinkStore classifies one reception against the persisted packet counters
// and stores it in the same transaction as the message, the endpoint
// last-seen state and the delivery outbox rows.
type UplinkStore interface {
	Persist(ctx context.Context, req models.UplinkPersistRequest) (models.UplinkPersistOutcome, error)
}

// DLRXStatusReader supplies the per-base-station downlink RX metrics reported
// since the endpoint was last heard, attached to its next uplink (SCACI §3.8.1).
type DLRXStatusReader interface {
	GetDLRXStatusSinceLastHeard(ctx context.Context, tenantID int64, epEui []byte,
		bsEuis [][]byte) ([]*mioty.DLRXStatus, error)
}

// DownlinkAckRecorder records the downlink an uplink's dlAck acknowledges
// (BSSCI §3.10.1), whichever path delivered the uplink.
type DownlinkAckRecorder interface {
	RecordEndpointAck(ctx context.Context, ownerTenantID int64, epEUI uint64, packetCnt uint32) error
}

// UplinkWindows are the time windows of an uplink: how long its packet
// counter stays a duplicate of its first reception, and how long its delivery
// waits for the receptions of the other base stations (SCACI §3.8.1).
type UplinkWindows struct {
	Duplicate time.Duration
	Reception time.Duration
}

// UplinkIngestServiceImpl implements bssci.UplinkIngestService.
// It runs the shared ingest pipeline: tenant resolution, payload decoding and one
// transactional persist that classifies the reception and queues delivery.
type UplinkIngestServiceImpl struct {
	store             UplinkStore
	windows           UplinkWindows
	channels          []models.DeliveryChannel
	dlrxStatuses      DLRXStatusReader
	orgResolver       org.Resolver
	roamingSvc        bssci.RoamingService
	endpointRepo      EndpointResolver
	endpointOwners    bssci.EndpointOwnerResolver
	blueprintResolver bssci.BlueprintResolver
	blueprintDecoder  bssci.BlueprintDecoder
	downlinkAcks      DownlinkAckRecorder
	logger            logger.Logger
	tenantID          int64

	// syntheticFederationBsEUI is written into ULDataMessage.BaseStations for federation uplinks.
	// It is a reserved EUI that is never present in the base_stations table, ensuring tenant-visible
	// message records never reference a real CE base station.
	syntheticFederationBsEUI uint64
}

// NewUplinkIngestService constructs a new UplinkIngestServiceImpl.
// channels lists the delivery outbox rows every new message is queued for; the
// MQTT channel is dropped per message when the owner organization is unknown.
// Optional collaborators (dlrxStatuses, orgResolver, roamingSvc, blueprint
// resolver and decoder) may be nil; their steps are skipped when absent. The
// endpoint owner resolver and the downlink acknowledgement recorder are
// required.
func NewUplinkIngestService(
	store UplinkStore,
	windows UplinkWindows,
	channels []models.DeliveryChannel,
	dlrxStatuses DLRXStatusReader,
	orgResolver org.Resolver,
	roamingSvc bssci.RoamingService,
	endpointRepo EndpointResolver,
	endpointOwners bssci.EndpointOwnerResolver,
	blueprintResolver bssci.BlueprintResolver,
	blueprintDecoder bssci.BlueprintDecoder,
	downlinkAcks DownlinkAckRecorder,
	log logger.Logger,
	tenantID int64,
	syntheticFederationBsEUI uint64,
) (*UplinkIngestServiceImpl, error) {
	if endpointOwners == nil {
		return nil, ErrNilEndpointOwnerResolver
	}
	if downlinkAcks == nil {
		return nil, ErrNilDownlinkAckRecorder
	}
	return &UplinkIngestServiceImpl{
		store:                    store,
		windows:                  windows,
		channels:                 channels,
		dlrxStatuses:             dlrxStatuses,
		orgResolver:              orgResolver,
		roamingSvc:               roamingSvc,
		endpointRepo:             endpointRepo,
		endpointOwners:           endpointOwners,
		blueprintResolver:        blueprintResolver,
		blueprintDecoder:         blueprintDecoder,
		downlinkAcks:             downlinkAcks,
		logger:                   log,
		tenantID:                 tenantID,
		syntheticFederationBsEUI: syntheticFederationBsEUI,
	}, nil
}

// Ingest executes the full uplink ingest pipeline for a single uplink payload.
// It returns an IngestResult with tenant context for use by the caller (e.g., downlink dispatch).
func (svc *UplinkIngestServiceImpl) Ingest(
	ctx context.Context,
	payload *bssci.UplinkPayload,
	opts bssci.UplinkIngestOptions,
) (*bssci.IngestResult, error) {
	// Step 1: Deduplication
	bsEUI := payload.BsEUI
	if opts.Source == bssci.UplinkSourceFederation && svc.syntheticFederationBsEUI != 0 {
		bsEUI = svc.syntheticFederationBsEUI
	}
	reception := mioty.BaseStationReception{
		BsEui:      bsEUI,
		RxTime:     payload.RxTime,
		RxDuration: payload.RxDuration,
		Snr:        payload.SNR,
		Rssi:       payload.RSSI,
		EqSnr:      payload.EqSNR,
		Profile:    payload.Profile,
		Mode:       payload.Mode,
		Subpackets: payload.Subpackets,
	}
	ulDataMsg := &mioty.ULDataMessage{
		CommandType:  mioty.CmdULData,
		OpId:         payload.OpID,
		EpEui:        payload.EpEUI,
		BsEui:        bsEUI,
		TenantID:     svc.tenantID,
		RxTime:       payload.RxTime,
		PacketCnt:    payload.PacketCnt,
		SNR:          payload.SNR,
		RSSI:         payload.RSSI,
		UserData:     payload.UserData,
		DlOpen:       payload.DLOpen,
		ResponseExp:  payload.ResponseExp,
		DlAck:        payload.DlAck,
		BaseStations: []mioty.BaseStationReception{reception},
		EqSnr:        payload.EqSNR,
		RxDuration:   payload.RxDuration,
		Profile:      payload.Profile,
		Mode:         payload.Mode,
		Format:       payload.Format,
		Subpackets:   payload.Subpackets,
	}
	var err error
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, payload.EpEUI)

	servingTenantID := svc.tenantID
	if opts.ServingTenantID > 0 {
		servingTenantID = opts.ServingTenantID
	}
	var ownerTenantID int64
	var isRoaming bool

	if svc.roamingSvc != nil {
		isRoaming, ownerTenantID, err = svc.roamingSvc.DetectAndValidateRoaming(ctx, epEuiBytes, servingTenantID)
		if err != nil {
			if errors.Is(err, roaming.ErrEndpointNotFound) {
				svc.logger.WarnContext(ctx, bssci.LogBSSCIEndpointNotFoundDuringIngestTenantResolution,
					logger.FieldEpEuiSnake, payload.EpEUI)
				// Disposition resolver should have prevented this; drop the packet
				return nil, fmt.Errorf(errFmtEndpointNotFoundDuringIngest, payload.EpEUI)
			}
			// Any other roaming error is fail-closed: do not fall back to serving tenant
			// as that could assign uplinks to the wrong tenant.
			svc.logger.ErrorContext(ctx, bssci.LogBSSCIRoamingDetectionFailedDuringIngest,
				logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldError, err)
			return nil, fmt.Errorf("%w: %w", errRoamingDetectionFailed, err)
		} else if isRoaming {
			svc.logger.InfoContext(ctx, bssci.LogBSSCIRoamingEndpointUplink,
				logger.FieldEpEuiSnake, payload.EpEUI,
				logger.FieldOwnerTenantSnake, ownerTenantID, logger.FieldServingTenantSnake, servingTenantID)
		}
	} else {
		var epEUI models.EUI
		copy(epEUI[:], epEuiBytes)
		owner, lookupErr := svc.endpointOwners.ResolveOwner(ctx, epEUI)
		if lookupErr != nil {
			if errors.Is(lookupErr, storage.ErrNotFound) {
				return nil, fmt.Errorf(errFmtEndpointNotFoundDuringIngest, payload.EpEUI)
			}
			return nil, fmt.Errorf("%w: %w", errEndpointLookupFailed, lookupErr)
		}
		ownerTenantID = owner.TenantID
	}

	if ownerTenantID <= 0 {
		return nil, fmt.Errorf(errFmtInvalidResolvedTenant, ownerTenantID, payload.EpEUI)
	}

	// Step 4: Build owner context
	ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)

	var ownerOrgUUID uuid.UUID
	if svc.orgResolver != nil {
		ownerOrgUUID, err = svc.orgResolver.GetDefaultOrgForTenant(ownerCtx, ownerTenantID)
		if err != nil {
			svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToResolveOrganizationForUplink,
				logger.FieldTenantIDSnake, ownerTenantID, logger.FieldError, err)
		}
	}
	if ownerOrgUUID != uuid.Nil {
		ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
	}

	// Set tenant and org on the message
	ulDataMsg.TenantID = ownerTenantID
	if ownerOrgUUID != uuid.Nil {
		orgStr := ownerOrgUUID.String()
		ulDataMsg.OrgUUID = &orgStr
	}

	// Step 5: Blueprint payload decoding
	if svc.blueprintResolver != nil && svc.blueprintDecoder != nil && len(ulDataMsg.UserData) > 0 {
		ulDataMsg.DecodeStatus = pkgblueprint.DecodeStatusSkipped

		epEUIBytesForBP := make([]byte, 8)
		binary.BigEndian.PutUint64(epEUIBytesForBP, payload.EpEUI)

		if svc.endpointRepo != nil {
			epModel, epErr := svc.endpointRepo.GetByEUI(ownerCtx, ownerTenantID, epEUIBytesForBP)
			if epErr == nil && epModel != nil {
				bp, bpErr := svc.blueprintResolver.ResolveBlueprintForEndpoint(ownerCtx, ownerTenantID, epModel, ulDataMsg.Format)
				if bpErr != nil {
					svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIBlueprintResolutionFailed,
						logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldError, bpErr)
				} else if bp != nil {
					ulDataMsg.DecodeStatus = pkgblueprint.DecodeStatusPending
					ulDataMsg.BlueprintTypeEUI = bp.TypeEUI
					ulDataMsg.BlueprintVersionID = &bp.ID

					calibration := svc.blueprintResolver.GetEndpointCalibration(ownerCtx, ownerTenantID, epModel)

					formatID := uint8(0)
					if ulDataMsg.Format != nil {
						formatID = *ulDataMsg.Format
					}

					decodeResult, decodeErr := svc.blueprintDecoder.Decode(ownerCtx, bp, ulDataMsg.UserData, formatID, calibration)
					if decodeErr != nil {
						svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIBlueprintDecodeError,
							logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldBlueprintID, bp.ID, logger.FieldError, decodeErr)
						ulDataMsg.DecodeStatus = pkgblueprint.DecodeStatusFailed
						ulDataMsg.DecodeErrorCode = pkgblueprint.ErrInternalDecodePanic
					} else if decodeResult != nil {
						if decodeResult.Success {
							decodedJSON, marshalErr := json.Marshal(decodeResult.DecodedData)
							if marshalErr != nil || !json.Valid(decodedJSON) {
								ulDataMsg.DecodedPayload = nil
								ulDataMsg.DecodeStatus = pkgblueprint.DecodeStatusFailed
								ulDataMsg.DecodeErrorCode = pkgblueprint.ErrInternalParseError
							} else {
								ulDataMsg.DecodedPayload = decodedJSON
								ulDataMsg.DecodeStatus = pkgblueprint.DecodeStatusSuccess
							}
						} else {
							ulDataMsg.DecodeStatus = pkgblueprint.DecodeStatusFailed
							ulDataMsg.DecodeErrorCode = decodeResult.ErrorCode
						}
					}
				}
			} else if epErr != nil && !errors.Is(epErr, storage.ErrNotFound) {
				svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToFetchEndpointForBlueprintDecode,
					logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldError, epErr)
			}
		}
	}

	// Step 6: Hydrate DL RX metrics into BaseStations
	if len(ulDataMsg.BaseStations) > 0 && svc.dlrxStatuses != nil {
		bsEuiBytes := make([][]byte, 0, len(ulDataMsg.BaseStations))
		for _, bs := range ulDataMsg.BaseStations {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, bs.BsEui)
			bsEuiBytes = append(bsEuiBytes, b)
		}
		epEuiBytesForDLRX := make([]byte, 8)
		binary.BigEndian.PutUint64(epEuiBytesForDLRX, ulDataMsg.EpEui)

		dlRxStatuses, dlRxErr := svc.dlrxStatuses.GetDLRXStatusSinceLastHeard(
			ownerCtx, ulDataMsg.TenantID, epEuiBytesForDLRX, bsEuiBytes,
		)
		if dlRxErr != nil {
			svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToFetchDLRXStatus,
				logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldError, dlRxErr)
		} else if len(dlRxStatuses) > 0 {
			dlRxMap := make(map[uint64]*mioty.DLRXStatus, len(dlRxStatuses))
			for _, dlRx := range dlRxStatuses {
				if len(dlRx.BsEui) >= 8 {
					key := binary.BigEndian.Uint64(dlRx.BsEui)
					dlRxMap[key] = dlRx
				}
			}
			for i := range ulDataMsg.BaseStations {
				bs := &ulDataMsg.BaseStations[i]
				if dlRx, ok := dlRxMap[bs.BsEui]; ok {
					snr := dlRx.DlRxSnr
					rssi := dlRx.DlRxRssi
					bs.DlRxSnr = &snr
					bs.DlRxRssi = &rssi
				}
			}
		}
	}

	// Step 7: Persist to database
	ulDataMsg.ID = uuid.New().String()
	channels := svc.deliveryChannels(ownerCtx, payload.EpEUI, ownerOrgUUID)
	outcome, err := svc.store.Persist(ownerCtx, models.UplinkPersistRequest{
		Message:         ulDataMsg,
		Window:          svc.windows.Duplicate,
		ReceptionWindow: svc.windows.Reception,
		Channels:        channels,
	})
	if err != nil {
		return nil, svc.persistFailure(ownerCtx, payload, err)
	}
	isDuplicate := outcome.Classification == models.UplinkDuplicate
	if isDuplicate {
		svc.logger.DebugContext(ownerCtx, bssci.LogBSSCIDuplicateUplinkReceived,
			logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldPacketCntSnake, payload.PacketCnt,
			logger.FieldDuplicateCount, outcome.DuplicateCount,
			logger.FieldTotalBaseStations, len(outcome.BaseStations))
	} else {
		svc.logger.InfoContext(ownerCtx, bssci.LogBSSCIUplinkFirstReception,
			logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldPacketCntSnake, payload.PacketCnt,
			logger.FieldSize, len(payload.UserData), logger.FieldRssi, payload.RSSI, logger.FieldSnr, payload.SNR)
	}
	svc.recordEndpointAck(ownerCtx, payload, ownerTenantID)
	return &bssci.IngestResult{
		IsDuplicate:   isDuplicate,
		OwnerTenantID: ownerTenantID,
		OwnerOrgUUID:  ownerOrgUUID,
		MessageID:     outcome.MessageID,
	}, nil
}

// recordEndpointAck hands an uplink's dlAck to the downlink queue under the
// endpoint's owner; a failure never fails the uplink.
func (svc *UplinkIngestServiceImpl) recordEndpointAck(ctx context.Context, payload *bssci.UplinkPayload, ownerTenantID int64) {
	if !payload.DlAck {
		return
	}
	if err := svc.downlinkAcks.RecordEndpointAck(ctx, ownerTenantID, payload.EpEUI, payload.PacketCnt); err != nil {
		svc.logger.WarnContext(ctx, bssci.LogBSSCIFailedToRecordEndpointAck,
			logger.FieldEpEui, payload.EpEUI,
			logger.FieldPacketCnt, payload.PacketCnt,
			logger.FieldError, err)
	}
}

// classifierRefusal pairs a store refusal with the catalog error the base station receives.
type classifierRefusal struct {
	cause error
	token string
	posix int
	log   string
}

// classifierRefusals lists the store's refusals; any other persist failure is internal.
var classifierRefusals = []classifierRefusal{
	{storage.ErrPacketCounterCollision, bssci.ErrTokenPacketCounterCollision, bssci.POSIX_EEXIST, bssci.LogBSSCIUplinkPacketCounterCollision},
}

// persistFailure logs a failed persist and returns the error the base station is answered with.
func (svc *UplinkIngestServiceImpl) persistFailure(ctx context.Context, payload *bssci.UplinkPayload, err error) error {
	for _, refusal := range classifierRefusals {
		if errors.Is(err, refusal.cause) {
			svc.logger.WarnContext(ctx, refusal.log,
				logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldPacketCntSnake, payload.PacketCnt)
			return fmt.Errorf("%w: %w", bssci.NewCatalogError(refusal.token, refusal.posix), err)
		}
	}
	svc.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToPersistUplinkMessage,
		logger.FieldEpEuiSnake, payload.EpEUI, logger.FieldPacketCntSnake, payload.PacketCnt, logger.FieldError, err)
	return fmt.Errorf("%w: %w", errPersistUplinkFailed, err)
}

// deliveryChannels narrows the configured channels for one message: MQTT
// topics are organization-scoped, so a message whose owner organization
// could not be resolved is not queued for MQTT.
func (svc *UplinkIngestServiceImpl) deliveryChannels(ctx context.Context, epEUI uint64, ownerOrgUUID uuid.UUID) []models.DeliveryChannel {
	channels := make([]models.DeliveryChannel, 0, len(svc.channels))
	for _, channel := range svc.channels {
		if channel == models.DeliveryChannelMQTT && ownerOrgUUID == uuid.Nil {
			svc.logger.WarnContext(ctx, bssci.LogBSSCIMQTTUplinkPublishSkippedOrgUnresolved,
				logger.FieldEpEuiSnake, epEUI)
			continue
		}
		channels = append(channels, channel)
	}
	return channels
}
