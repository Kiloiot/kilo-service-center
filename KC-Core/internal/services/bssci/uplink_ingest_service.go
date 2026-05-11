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
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
)

// UplinkIngestServiceImpl implements bssci.UplinkIngestService.
// It runs the shared ingest pipeline: deduplication, tenant resolution, payload decoding,
// persistence, SCACI fan-out, and MQTT publishing.
type UplinkIngestServiceImpl struct {
	deduplicator      *bssci.MessageDeduplicator
	storage           interfaces.Storage
	orgResolver       org.Resolver
	roamingSvc        bssci.RoamingService
	endpointRepo      interfaces.EndpointRepository
	blueprintResolver bssci.BlueprintResolver
	blueprintDecoder  bssci.BlueprintDecoder
	broadcaster       bssci.SCACIBroadcaster
	mqttPublisher     bssci.MQTTEventPublisher
	logger            logger.Logger
	tenantID          int64

	// syntheticFederationBsEUI is written into ULDataMessage.BaseStations for federation uplinks.
	// It is a reserved EUI that is never present in the base_stations table, ensuring tenant-visible
	// message records never reference a real CE base station.
	syntheticFederationBsEUI uint64
}

// NewUplinkIngestService constructs a new UplinkIngestServiceImpl.
// All optional collaborators (broadcaster, mqttPublisher, blueprintResolver, etc.) may be nil;
// their associated steps are skipped gracefully when absent.
func NewUplinkIngestService(
	deduplicator *bssci.MessageDeduplicator,
	stor interfaces.Storage,
	orgResolver org.Resolver,
	roamingSvc bssci.RoamingService,
	endpointRepo interfaces.EndpointRepository,
	blueprintResolver bssci.BlueprintResolver,
	blueprintDecoder bssci.BlueprintDecoder,
	broadcaster bssci.SCACIBroadcaster,
	mqttPublisher bssci.MQTTEventPublisher,
	log logger.Logger,
	tenantID int64,
	syntheticFederationBsEUI uint64,
) *UplinkIngestServiceImpl {
	return &UplinkIngestServiceImpl{
		deduplicator:             deduplicator,
		storage:                  stor,
		orgResolver:              orgResolver,
		roamingSvc:               roamingSvc,
		endpointRepo:             endpointRepo,
		blueprintResolver:        blueprintResolver,
		blueprintDecoder:         blueprintDecoder,
		broadcaster:              broadcaster,
		mqttPublisher:            mqttPublisher,
		logger:                   log,
		tenantID:                 tenantID,
		syntheticFederationBsEUI: syntheticFederationBsEUI,
	}
}

// SetBlueprintResolver injects the blueprint resolver after service construction.
func (svc *UplinkIngestServiceImpl) SetBlueprintResolver(r bssci.BlueprintResolver) {
	svc.blueprintResolver = r
}

// SetBlueprintDecoder injects the blueprint decoder after service construction.
func (svc *UplinkIngestServiceImpl) SetBlueprintDecoder(d bssci.BlueprintDecoder) {
	svc.blueprintDecoder = d
}

// SetMQTTPublisher injects the MQTT publisher after service construction.
func (svc *UplinkIngestServiceImpl) SetMQTTPublisher(p bssci.MQTTEventPublisher) {
	svc.mqttPublisher = p
}

// Ingest executes the full uplink ingest pipeline for a single uplink payload.
// It returns an IngestResult with tenant context for use by the caller (e.g., downlink dispatch).
func (svc *UplinkIngestServiceImpl) Ingest(
	ctx context.Context,
	payload *bssci.UplinkPayload,
	opts bssci.UplinkIngestOptions,
) (*bssci.IngestResult, error) {
	// Step 1: Deduplication
	isDuplicate, record, err := svc.deduplicator.CheckAndRecord(
		payload.EpEUI,
		payload.PacketCnt,
		payload.BsEUI,
		payload.UserData,
		payload.RSSI,
		payload.SNR,
		payload.EqSNR,
		payload.RxTime,
		payload.RxDuration,
		payload.Profile,
		payload.Mode,
		payload.Subpackets,
	)
	if err != nil {
		svc.logger.ErrorContext(ctx, bssci.LogBSSCIUplinkDeduplicationError,
			"ep_eui", payload.EpEUI, "packet_cnt", payload.PacketCnt, "error", err)
		return nil, fmt.Errorf("deduplication failed: %w", err)
	}

	if isDuplicate {
		svc.logger.DebugContext(ctx, bssci.LogBSSCIDuplicateUplinkReceived,
			"ep_eui", payload.EpEUI, "packet_cnt", payload.PacketCnt,
			"duplicate_count", record.DuplicateCount,
			"total_base_stations", len(record.BaseStations))
	} else {
		svc.logger.InfoContext(ctx, bssci.LogBSSCIUplinkFirstReception,
			"ep_eui", payload.EpEUI, "packet_cnt", payload.PacketCnt,
			"size", len(payload.UserData), "rssi", payload.RSSI, "snr", payload.SNR)
	}

	// Step 2: Build ULDataMessage
	isDup := record.DuplicateCount > 0
	bsEUI := payload.BsEUI
	if opts.Source == bssci.UplinkSourceFederation && svc.syntheticFederationBsEUI != 0 {
		// Federation uplinks must not expose the real CE BS EUI in tenant-visible fields.
		bsEUI = svc.syntheticFederationBsEUI
	}

	ulDataMsg := &mioty.ULDataMessage{
		CommandType:  mioty.CmdULData,
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
		BaseStations: record.ConvertToBaseStationReceptions(),
		Duplicate:    &isDup,
		EqSnr:        payload.EqSNR,
		RxDuration:   payload.RxDuration,
		Profile:      payload.Profile,
		Mode:         payload.Mode,
		Format:       payload.Format,
		Subpackets:   payload.Subpackets,
	}

	// Step 3: Tenant resolution (roaming-aware)
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, payload.EpEUI)

	servingTenantID := svc.tenantID
	var ownerTenantID int64
	var isRoaming bool

	if svc.roamingSvc != nil {
		isRoaming, ownerTenantID, err = svc.roamingSvc.DetectAndValidateRoaming(ctx, epEuiBytes, servingTenantID)
		if err != nil {
			if errors.Is(err, roaming.ErrEndpointNotFound) {
				svc.logger.WarnContext(ctx, bssci.LogBSSCIEndpointNotFoundDuringIngestTenantResolution,
					"ep_eui", payload.EpEUI)
				// Disposition resolver should have prevented this; drop the packet
				return nil, fmt.Errorf("endpoint not found during ingest: ep_eui=%d", payload.EpEUI)
			}
			// Any other roaming error is fail-closed: do not fall back to serving tenant
			// as that could assign uplinks to the wrong tenant.
			svc.logger.ErrorContext(ctx, bssci.LogBSSCIRoamingDetectionFailedDuringIngest,
				"ep_eui", payload.EpEUI, "error", err)
			return nil, fmt.Errorf("roaming detection failed: %w", err)
		} else if isRoaming {
			svc.logger.InfoContext(ctx, bssci.LogBSSCIRoamingEndpointUplink,
				"ep_eui", payload.EpEUI,
				"owner_tenant", ownerTenantID, "serving_tenant", servingTenantID)
		}
	} else {
		// No roaming service: perform a direct cross-tenant endpoint lookup
		var epEUI models.EUI
		copy(epEUI[:], epEuiBytes)
		ep, lookupErr := svc.endpointRepo.Get(ctx, epEUI)
		if lookupErr != nil {
			if errors.Is(lookupErr, storage.ErrNotFound) {
				return nil, fmt.Errorf("endpoint not found during ingest: ep_eui=%d", payload.EpEUI)
			}
			return nil, fmt.Errorf("endpoint lookup failed: %w", lookupErr)
		}
		ownerTenantID = ep.TenantID
	}

	if ownerTenantID <= 0 {
		return nil, fmt.Errorf("tenant resolution yielded invalid tenant id %d for ep_eui=%d", ownerTenantID, payload.EpEUI)
	}

	// Step 4: Build owner context
	ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)

	var ownerOrgUUID uuid.UUID
	if svc.orgResolver != nil {
		ownerOrgUUID, err = svc.orgResolver.GetDefaultOrgForTenant(ownerCtx, ownerTenantID)
		if err != nil {
			svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToResolveOrganizationForUplink,
				"tenant_id", ownerTenantID, "error", err)
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
						"ep_eui", payload.EpEUI, "error", bpErr)
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
							"ep_eui", payload.EpEUI, "blueprint_id", bp.ID, "error", decodeErr)
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
					"ep_eui", payload.EpEUI, "error", epErr)
			}
		}
	}

	// Step 6: Hydrate DL RX metrics into BaseStations
	if len(ulDataMsg.BaseStations) > 0 && svc.storage != nil && svc.storage.DLRXStatus() != nil {
		bsEuiBytes := make([][]byte, 0, len(ulDataMsg.BaseStations))
		for _, bs := range ulDataMsg.BaseStations {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, bs.BsEui)
			bsEuiBytes = append(bsEuiBytes, b)
		}
		epEuiBytesForDLRX := make([]byte, 8)
		binary.BigEndian.PutUint64(epEuiBytesForDLRX, ulDataMsg.EpEui)

		dlRxStatuses, dlRxErr := svc.storage.DLRXStatus().GetLatestDLRXStatusByBaseStations(
			ownerCtx, ulDataMsg.TenantID, epEuiBytesForDLRX, bsEuiBytes)
		if dlRxErr != nil {
			svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToFetchDLRXStatus,
				"ep_eui", payload.EpEUI, "error", dlRxErr)
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
	messageID := uuid.New().String()
	ulDataMsg.ID = messageID
	if svc.storage != nil && svc.storage.MIOTYMessages() != nil {
		if !isDuplicate {
			if err := svc.storage.MIOTYMessages().CreateULDataMessage(ownerCtx, ulDataMsg); err != nil {
				svc.logger.ErrorContext(ownerCtx, bssci.LogBSSCIFailedToPersistUplinkMessage,
					"ep_eui", payload.EpEUI, "packet_cnt", payload.PacketCnt, "error", err)
				return nil, fmt.Errorf("persist failed: %w", err)
			}
		} else if len(ulDataMsg.BaseStations) > 0 {
			rxTimeMin := time.Now().Add(-svc.deduplicator.WindowDuration()).UnixNano()
			baseStationsJSON, marshalErr := json.Marshal(ulDataMsg.BaseStations)
			if marshalErr != nil {
				svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToMarshalBaseStationsForDuplicateUpdate,
					"ep_eui", payload.EpEUI, "error", marshalErr)
			} else if updateErr := svc.storage.MIOTYMessages().UpdateULDataBaseStations(
				ownerCtx, ulDataMsg.TenantID, ulDataMsg.EpEui, uint32(ulDataMsg.PacketCnt),
				rxTimeMin, baseStationsJSON); updateErr != nil {
				svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToUpdateBaseStationsForDuplicate,
					"ep_eui", payload.EpEUI, "error", updateErr)
			}
		}
	}

	// Step 8: SCACI fan-out
	if svc.broadcaster != nil {
		if broadcastErr := svc.broadcaster.BroadcastULData(ownerCtx, ownerTenantID, ulDataMsg); broadcastErr != nil {
			svc.logger.ErrorContext(ownerCtx, bssci.LogBSSCIFailedToBroadcastUplinkToSCACI,
				"ep_eui", payload.EpEUI, "error", broadcastErr)
		}
	}

	// Step 9: MQTT publish
	if svc.mqttPublisher != nil && ownerOrgUUID != uuid.Nil {
		orgStr := ownerOrgUUID.String()
		go func() {
			if pubErr := svc.mqttPublisher.PublishUplink(
				ownerCtx, orgStr,
				payload.EpEUI, payload.BsEUI,
				payload.RSSI, payload.SNR,
				payload.RxTime, payload.PacketCnt,
				payload.UserData,
			); pubErr != nil {
				svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIFailedToPublishUplinkToMQTT,
					"ep_eui", payload.EpEUI, "error", pubErr)
			}
		}()
	} else if svc.mqttPublisher != nil {
		svc.logger.WarnContext(ownerCtx, bssci.LogBSSCIMQTTUplinkPublishSkippedOrgUnresolved,
			"ep_eui", payload.EpEUI)
	}

	resultMessageID := ""
	if !isDuplicate {
		resultMessageID = messageID
	}

	return &bssci.IngestResult{
		OwnerTenantID: ownerTenantID,
		OwnerOrgUUID:  ownerOrgUUID,
		IsDuplicate:   isDuplicate,
		MessageID:     resultMessageID,
	}, nil
}
