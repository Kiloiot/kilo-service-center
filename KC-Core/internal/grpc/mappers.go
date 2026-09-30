package grpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	endpointpkg "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func buildBlueprintSnapshot(bp *models.Blueprint) (json.RawMessage, error) {
	return json.Marshal(models.BlueprintSnapshot{
		SpecJSON:          bp.SpecJSON,
		Version:           bp.Version,
		TypeEUI:           mioty.FormatEUIBytes(bp.TypeEUI),
		SourceBlueprintID: bp.ID.String(),
		IsSystem:          bp.IsSystem,
		AdoptedAt:         time.Now().UTC(),
	})
}

func snapshotSourceID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var snap models.BlueprintSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return ""
	}
	return snap.SourceBlueprintID
}

func snapshotTypeEUI(raw json.RawMessage) *models.EUI {
	if len(raw) == 0 {
		return nil
	}
	var snap models.BlueprintSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil || snap.TypeEUI == "" {
		return nil
	}
	b, err := hex.DecodeString(snap.TypeEUI)
	if err != nil || len(b) != 8 {
		return nil
	}
	var t models.EUI
	copy(t[:], b)
	return &t
}

func modelChanged(oldID, newID *uuid.UUID) bool {
	switch {
	case oldID == nil && newID == nil:
		return false
	case oldID == nil || newID == nil:
		return true
	default:
		return *oldID != *newID
	}
}

// applyBlueprintSnapshot materializes the blueprint snapshot onto the endpoint; model-membership mismatch is advisory (logged only).
// applyBlueprintSnapshot pins the blueprint onto the endpoint: device model, type
// EUI and the serialized snapshot.
func applyBlueprintSnapshot(ctx context.Context, log logger.Logger, blueprints grpcservices.BlueprintService, endpoint *models.EndPoint, blueprintID string) error {
	if blueprints == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	bpUUID, parseErr := uuid.Parse(blueprintID)
	if parseErr != nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBlueprintIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBlueprintIDFormat))
	}
	bp, fetchErr := blueprints.GetBlueprint(ctx, bpUUID)
	if fetchErr != nil || bp == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintNotFound))
	}
	if endpoint.DeviceModelID != nil && bp.DeviceModelID != *endpoint.DeviceModelID {
		log.WarnContext(ctx, LogSelectedBlueprintBelongsToADifferentDeviceModel,
			logger.FieldEndpointEui, endpoint.EUI.String(), logger.FieldBlueprintID, bp.ID,
			logger.FieldBlueprintModel, bp.DeviceModelID, logger.FieldEndpointModel, *endpoint.DeviceModelID)
	}
	raw, marshalErr := buildBlueprintSnapshot(bp)
	if marshalErr != nil {
		log.ErrorContext(ctx, LogFailedToBuildBlueprintSnapshot,
			logger.FieldBlueprintID, bp.ID, logger.FieldError, marshalErr)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintSnapshotBuildFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintSnapshotBuildFailed))
	}
	dm := bp.DeviceModelID
	endpoint.DeviceModelID = &dm
	endpoint.TypeEUI = snapshotTypeEUI(raw)
	endpoint.BlueprintSnapshot = raw
	return nil
}

// pinnedBlueprintBelongsToModel reports whether the pinned snapshot source is native to the model; dangling/absent counts as not belonging.
func pinnedBlueprintBelongsToModel(ctx context.Context, blueprints grpcservices.BlueprintService, sourceID string, modelID uuid.UUID) bool {
	if sourceID == "" || blueprints == nil {
		return false
	}
	id, err := uuid.Parse(sourceID)
	if err != nil {
		return false
	}
	bp, err := blueprints.GetBlueprint(ctx, id)
	if err != nil || bp == nil {
		return false
	}
	return bp.DeviceModelID == modelID
}

// endpointToProto converts an endpoint with its keys masked (only whether each
// is set); it is active when last seen within activityWindow before now.
func endpointToProto(endpoint *models.EndPoint, activityWindow time.Duration, now time.Time) *pb.EndPoint {
	// Extract optional fields
	var shAddr uint32
	if endpoint.ShAddr != nil {
		shAddr = uint32(*endpoint.ShAddr)
	}
	var attachCnt uint32
	if endpoint.AttachCnt != nil {
		attachCnt = *endpoint.AttachCnt
	}
	var typeEui []byte
	if endpoint.TypeEUI != nil {
		typeEui = endpoint.TypeEUI[:]
	}

	// Derive activity status from LastSeenAt and configured window
	status := endpointActivityInactive
	if endpoint.LastSeenAt != nil && activityWindow > 0 {
		if now.Sub(*endpoint.LastSeenAt) <= activityWindow {
			status = endpointActivityActive
		}
	}

	result := &pb.EndPoint{
		EpEui:        endpoint.EUI.String(),
		TenantId:     strconv.FormatInt(endpoint.TenantID, 10),
		Name:         endpoint.Name,
		Description:  endpoint.Description,
		EpClass:      endpoint.EPClass,
		NwkSnKeySet:  len(endpoint.NwkSnKey) > 0,
		AppKeySet:    len(endpoint.AppKey) > 0,
		Status:       status,
		AttachStatus: endpoint.EpStatus,
		Tags:         endpoint.Tags,
		CreatedAt:    timestamppb.New(endpoint.CreatedAt),
		UpdatedAt:    timestamppb.New(endpoint.UpdatedAt),
		// MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
		ShAddr:          shAddr,
		DualChan:        endpoint.DualChan,
		Repetition:      endpoint.Repetition,
		WideCarrOff:     endpoint.WideCarrOff,
		LongBlkDist:     endpoint.LongBlkDist,
		AttachCnt:       attachCnt,
		PreAttach:       endpoint.PreAttach,
		LastPacketCnt:   endpoint.LastPacketCnt,
		TypeEui:         typeEui,
		CarrierOffset:   int32(endpoint.CarrierOffset), //nolint:gosec // G115: CarrierOffset per BSSCI spec is small integer
		ReattachPending: endpointpkg.ReattachPending(endpoint),
	}

	if endpoint.DeviceModelID != nil {
		result.DeviceModelId = endpoint.DeviceModelID.String()
	}

	// Echo the selector back from the materialized snapshot; empty when following catalog default.
	if sourceID := snapshotSourceID(endpoint.BlueprintSnapshot); sourceID != "" {
		result.BlueprintId = sourceID
	}
	result.BlueprintSnapshot = endpoint.BlueprintSnapshot // read-back: full snapshot for reload/fallback render

	if endpoint.LastSeenAt != nil {
		result.LastSeenAt = timestamppb.New(*endpoint.LastSeenAt)
	}
	result.LastRssi = optionalDouble(endpoint.LastRSSI)
	result.LastSnr = optionalDouble(endpoint.LastSNR)
	result.LastEqSnr = optionalDouble(endpoint.LastEqSNR)

	return result
}

// optionalDouble renders a stored measurement; nil, never measured, stays absent.
func optionalDouble(v *float64) *wrapperspb.DoubleValue {
	if v == nil {
		return nil
	}
	return wrapperspb.Double(*v)
}

// baseStationToProto converts a models base station to proto; stored tags
// that do not decode fail the conversion.
func baseStationToProto(baseStation *models.BaseStation) (*pb.BaseStation, error) {
	var desc string
	if baseStation.Description != nil {
		desc = *baseStation.Description
	}

	tags, err := decodeBaseStationTags(baseStation.Tags)
	if err != nil {
		return nil, err
	}

	// Determine status - use online status if available
	status := grpcerrors.StatusOffline
	if baseStation.IsOnline {
		status = grpcerrors.StatusOnline
	}

	result := &pb.BaseStation{
		BsEui:       baseStation.EUI.String(),
		TenantId:    strconv.FormatInt(baseStation.TenantID, 10),
		Name:        baseStation.Name,
		Description: desc,
		Status:      status,
		Tags:        tags,
		CreatedAt:   timestamppb.New(baseStation.CreatedAt),
		UpdatedAt:   timestamppb.New(baseStation.UpdatedAt),
	}

	// Coordinate wrappers: nil model → nil proto (absent), non-nil → wrapped value
	if baseStation.Latitude != nil {
		result.Latitude = wrapperspb.Double(*baseStation.Latitude)
	}
	if baseStation.Longitude != nil {
		result.Longitude = wrapperspb.Double(*baseStation.Longitude)
	}
	if baseStation.Altitude != nil {
		result.Altitude = wrapperspb.Double(*baseStation.Altitude)
	}

	// Location metadata
	if baseStation.LocationSource != nil {
		result.LocationSource = *baseStation.LocationSource
	}
	if baseStation.LocationUpdatedAt != nil {
		result.LocationUpdatedAt = timestamppb.New(*baseStation.LocationUpdatedAt)
	}

	if baseStation.LastSeenAt != nil {
		result.LastSeenAt = timestamppb.New(*baseStation.LastSeenAt)
	}

	applyStatusReport(result, baseStation)
	if baseStation.TLSCertExpiresAt != nil {
		result.CertificateExpiresAt = timestamppb.New(*baseStation.TLSCertExpiresAt)
	}
	if baseStation.TLSCertFingerprint != nil {
		result.TlsCertFingerprint = *baseStation.TLSCertFingerprint
	}
	if baseStation.SessionStartedAt != nil {
		result.SessionStartedAt = timestamppb.New(*baseStation.SessionStartedAt)
	}

	if baseStation.ServiceCenterURL != nil {
		result.ServiceCenterUrl = *baseStation.ServiceCenterURL
	}

	return result, nil
}

// applyStatusReport maps the last statusRsp fields (BSSCI v1.0.0 §5.5.2);
// wrapper types keep an unreported field unset ("Not available").
func applyStatusReport(result *pb.BaseStation, baseStation *models.BaseStation) {
	if baseStation.SystemTime != nil {
		result.SystemTime = wrapperspb.Int64(*baseStation.SystemTime)
	}
	if baseStation.DutyCycle != nil {
		result.DutyCycle = wrapperspb.Double(*baseStation.DutyCycle)
	}
	if baseStation.UptimeSeconds != nil {
		result.UptimeSeconds = wrapperspb.Int64(*baseStation.UptimeSeconds)
	}
	if baseStation.TemperatureCelsius != nil {
		result.TemperatureCelsius = wrapperspb.Double(*baseStation.TemperatureCelsius)
	}
	if baseStation.CPULoad != nil {
		result.CpuLoad = wrapperspb.Double(*baseStation.CPULoad)
	}
	if baseStation.MemoryLoad != nil {
		result.MemoryLoad = wrapperspb.Double(*baseStation.MemoryLoad)
	}
	if baseStation.BSConfig.Valid {
		var configMap map[string]interface{}
		if err := json.Unmarshal(baseStation.BSConfig.Data, &configMap); err == nil {
			if structVal, err := structpb.NewStruct(configMap); err == nil {
				result.BsConfig = structVal
			}
		}
	}
	if baseStation.LastStatusAt != nil {
		result.LastStatusAt = timestamppb.New(*baseStation.LastStatusAt)
	}
}

// timestampToTime converts a protobuf timestamp to a Go time pointer
// Returns nil if the timestamp is nil
func timestampToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

// optionalTimestamp converts an optional time to a protobuf timestamp, nil
// when the time is unset.
func optionalTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
