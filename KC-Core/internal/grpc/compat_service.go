package grpc

import (
	"context"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// KiloCenterServiceCompat is a backward-compatible shim that implements the
// unified KiloCenterServiceServer interface by delegating each core RPC to the
// CoreService. Identity RPCs (auth, users, orgs, memberships, API keys) fall
// through to UnimplementedKiloCenterServiceServer — they are served by
// KC-Identity via gateway routing.
type KiloCenterServiceCompat struct {
	pb.UnimplementedKiloCenterServiceServer
	core *CoreService
}

// NewKiloCenterServiceCompat creates a compat shim that fans out core RPCs to
// the CoreService implementation. Identity RPCs return Unimplemented.
func NewKiloCenterServiceCompat(core *CoreService) *KiloCenterServiceCompat {
	return &KiloCenterServiceCompat{core: core}
}

//revive:disable:exported Delegation methods are self-documenting one-liners.

// ---------------------------------------------------------------------------
// Core RPCs -- delegated to s.core
// ---------------------------------------------------------------------------

// Endpoints (7)

func (s *KiloCenterServiceCompat) CreateEndPoint(ctx context.Context, req *pb.CreateEndPointRequest) (*pb.EndPoint, error) {
	return s.core.CreateEndPoint(ctx, req)
}

func (s *KiloCenterServiceCompat) GetEndPoint(ctx context.Context, req *pb.GetEndPointRequest) (*pb.EndPoint, error) {
	return s.core.GetEndPoint(ctx, req)
}

func (s *KiloCenterServiceCompat) UpdateEndPoint(ctx context.Context, req *pb.UpdateEndPointRequest) (*pb.EndPoint, error) {
	return s.core.UpdateEndPoint(ctx, req)
}

func (s *KiloCenterServiceCompat) DeleteEndPoint(ctx context.Context, req *pb.DeleteEndPointRequest) (*emptypb.Empty, error) {
	return s.core.DeleteEndPoint(ctx, req)
}

func (s *KiloCenterServiceCompat) ListEndPoints(ctx context.Context, req *pb.ListEndPointsRequest) (*pb.ListEndPointsResponse, error) {
	return s.core.ListEndPoints(ctx, req)
}

func (s *KiloCenterServiceCompat) AttachEndPoint(ctx context.Context, req *pb.AttachEndPointRequest) (*pb.AttachEndPointResponse, error) {
	return s.core.AttachEndPoint(ctx, req)
}

func (s *KiloCenterServiceCompat) DetachEndPoint(ctx context.Context, req *pb.DetachEndPointRequest) (*pb.DetachEndPointResponse, error) {
	return s.core.DetachEndPoint(ctx, req)
}

// Base Stations (7)

func (s *KiloCenterServiceCompat) CreateBaseStation(ctx context.Context, req *pb.CreateBaseStationRequest) (*pb.BaseStation, error) {
	return s.core.CreateBaseStation(ctx, req)
}

func (s *KiloCenterServiceCompat) GetBaseStation(ctx context.Context, req *pb.GetBaseStationRequest) (*pb.BaseStation, error) {
	return s.core.GetBaseStation(ctx, req)
}

func (s *KiloCenterServiceCompat) UpdateBaseStation(ctx context.Context, req *pb.UpdateBaseStationRequest) (*pb.BaseStation, error) {
	return s.core.UpdateBaseStation(ctx, req)
}

func (s *KiloCenterServiceCompat) DeleteBaseStation(ctx context.Context, req *pb.DeleteBaseStationRequest) (*emptypb.Empty, error) {
	return s.core.DeleteBaseStation(ctx, req)
}

func (s *KiloCenterServiceCompat) ListBaseStations(ctx context.Context, req *pb.ListBaseStationsRequest) (*pb.ListBaseStationsResponse, error) {
	return s.core.ListBaseStations(ctx, req)
}

func (s *KiloCenterServiceCompat) GetBaseStationStats(ctx context.Context, req *pb.GetBaseStationStatsRequest) (*pb.GetBaseStationStatsResponse, error) {
	return s.core.GetBaseStationStats(ctx, req)
}

func (s *KiloCenterServiceCompat) UpdateBaseStationEui(ctx context.Context, req *pb.UpdateBaseStationEuiRequest) (*pb.BaseStation, error) {
	return s.core.UpdateBaseStationEui(ctx, req)
}

// Messages (1)

func (s *KiloCenterServiceCompat) GetMessage(ctx context.Context, req *pb.GetMessageRequest) (*pb.Message, error) {
	return s.core.GetMessage(ctx, req)
}

// Downlinks (4)

func (s *KiloCenterServiceCompat) SendDownlink(ctx context.Context, req *pb.SendDownlinkRequest) (*pb.SendDownlinkResponse, error) {
	return s.core.SendDownlink(ctx, req)
}

func (s *KiloCenterServiceCompat) RevokeDownlink(ctx context.Context, req *pb.RevokeDownlinkRequest) (*pb.RevokeDownlinkResponse, error) {
	return s.core.RevokeDownlink(ctx, req)
}

func (s *KiloCenterServiceCompat) ListDownlinkQueue(ctx context.Context, req *pb.ListDownlinkQueueRequest) (*pb.ListDownlinkQueueResponse, error) {
	return s.core.ListDownlinkQueue(ctx, req)
}

func (s *KiloCenterServiceCompat) GetDownlinkResults(ctx context.Context, req *pb.GetDownlinkResultsRequest) (*pb.GetDownlinkResultsResponse, error) {
	return s.core.GetDownlinkResults(ctx, req)
}

// BSSCI (6)

func (s *KiloCenterServiceCompat) SendULTransmit(ctx context.Context, req *pb.SendULTransmitRequest) (*pb.SendULTransmitResponse, error) {
	return s.core.SendULTransmit(ctx, req)
}

func (s *KiloCenterServiceCompat) RequestBaseStationStatus(ctx context.Context, req *pb.BaseStationStatusRequest) (*pb.BaseStationStatusResponse, error) {
	return s.core.RequestBaseStationStatus(ctx, req)
}

func (s *KiloCenterServiceCompat) InitiatePing(ctx context.Context, req *pb.InitiatePingRequest) (*pb.InitiatePingResponse, error) {
	return s.core.InitiatePing(ctx, req)
}

func (s *KiloCenterServiceCompat) GetDLRXStatus(ctx context.Context, req *pb.GetDLRXStatusRequest) (*pb.GetDLRXStatusResponse, error) {
	return s.core.GetDLRXStatus(ctx, req)
}

func (s *KiloCenterServiceCompat) QueryDLRXStatus(ctx context.Context, req *pb.QueryDLRXStatusRequest) (*pb.QueryDLRXStatusResponse, error) {
	return s.core.QueryDLRXStatus(ctx, req)
}

func (s *KiloCenterServiceCompat) GetDLRXStatusQueries(ctx context.Context, req *pb.GetDLRXStatusQueriesRequest) (*pb.GetDLRXStatusQueriesResponse, error) {
	return s.core.GetDLRXStatusQueries(ctx, req)
}

// System (3)

func (s *KiloCenterServiceCompat) GetSystemStatus(ctx context.Context, req *emptypb.Empty) (*pb.SystemStatus, error) {
	return s.core.GetSystemStatus(ctx, req)
}

func (s *KiloCenterServiceCompat) GetStatistics(ctx context.Context, req *pb.GetStatisticsRequest) (*pb.Statistics, error) {
	return s.core.GetStatistics(ctx, req)
}

func (s *KiloCenterServiceCompat) GetReleaseInfo(ctx context.Context, req *emptypb.Empty) (*pb.ReleaseInfo, error) {
	return s.core.GetReleaseInfo(ctx, req)
}

// Integrations (5)

func (s *KiloCenterServiceCompat) CreateIntegration(ctx context.Context, req *pb.CreateIntegrationRequest) (*pb.Integration, error) {
	return s.core.CreateIntegration(ctx, req)
}

func (s *KiloCenterServiceCompat) GetIntegration(ctx context.Context, req *pb.GetIntegrationRequest) (*pb.Integration, error) {
	return s.core.GetIntegration(ctx, req)
}

func (s *KiloCenterServiceCompat) UpdateIntegration(ctx context.Context, req *pb.UpdateIntegrationRequest) (*pb.Integration, error) {
	return s.core.UpdateIntegration(ctx, req)
}

func (s *KiloCenterServiceCompat) DeleteIntegration(ctx context.Context, req *pb.DeleteIntegrationRequest) (*emptypb.Empty, error) {
	return s.core.DeleteIntegration(ctx, req)
}

func (s *KiloCenterServiceCompat) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	return s.core.ListIntegrations(ctx, req)
}

// Analytics (3)

func (s *KiloCenterServiceCompat) GetAnalyticsOverview(ctx context.Context, req *pb.GetAnalyticsOverviewRequest) (*pb.GetAnalyticsOverviewResponse, error) {
	return s.core.GetAnalyticsOverview(ctx, req)
}

func (s *KiloCenterServiceCompat) GetActivityAnalytics(ctx context.Context, req *pb.GetActivityAnalyticsRequest) (*pb.GetActivityAnalyticsResponse, error) {
	return s.core.GetActivityAnalytics(ctx, req)
}

func (s *KiloCenterServiceCompat) GetSignalQualityAnalytics(ctx context.Context, req *pb.GetSignalQualityAnalyticsRequest) (*pb.GetSignalQualityAnalyticsResponse, error) {
	return s.core.GetSignalQualityAnalytics(ctx, req)
}

// Events/Activity (2)

func (s *KiloCenterServiceCompat) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	return s.core.ListEvents(ctx, req)
}

func (s *KiloCenterServiceCompat) ListBaseStationActivity(ctx context.Context, req *pb.ListBaseStationActivityRequest) (*pb.ListBaseStationActivityResponse, error) {
	return s.core.ListBaseStationActivity(ctx, req)
}

func (s *KiloCenterServiceCompat) ListEndpointActivity(ctx context.Context, req *pb.ListEndpointActivityRequest) (*pb.ListEndpointActivityResponse, error) {
	return s.core.ListEndpointActivity(ctx, req)
}

// Streaming (3)

func (s *KiloCenterServiceCompat) StreamEvents(req *pb.StreamEventsRequest, stream grpc.ServerStreamingServer[pb.Event]) error {
	return s.core.StreamEvents(req, stream)
}

func (s *KiloCenterServiceCompat) StreamMessages(req *pb.StreamMessagesRequest, stream grpc.ServerStreamingServer[pb.Message]) error {
	return s.core.StreamMessages(req, stream)
}

func (s *KiloCenterServiceCompat) StreamBaseStationMessages(req *pb.StreamBaseStationMessagesRequest, stream grpc.ServerStreamingServer[pb.BaseStationMessage]) error {
	return s.core.StreamBaseStationMessages(req, stream)
}

// Alerts (2)

func (s *KiloCenterServiceCompat) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	return s.core.ListAlerts(ctx, req)
}

func (s *KiloCenterServiceCompat) GetAlertSummary(ctx context.Context, req *pb.GetAlertSummaryRequest) (*pb.GetAlertSummaryResponse, error) {
	return s.core.GetAlertSummary(ctx, req)
}

// SCACI (6)

func (s *KiloCenterServiceCompat) ListScaciSessions(ctx context.Context, req *pb.ListScaciSessionsRequest) (*pb.ListScaciSessionsResponse, error) {
	return s.core.ListScaciSessions(ctx, req)
}

func (s *KiloCenterServiceCompat) GetScaciSession(ctx context.Context, req *pb.GetScaciSessionRequest) (*pb.GetScaciSessionResponse, error) {
	return s.core.GetScaciSession(ctx, req)
}

func (s *KiloCenterServiceCompat) GetScaciStatistics(ctx context.Context, req *pb.GetScaciStatisticsRequest) (*pb.GetScaciStatisticsResponse, error) {
	return s.core.GetScaciStatistics(ctx, req)
}

func (s *KiloCenterServiceCompat) ListScaciErrors(ctx context.Context, req *pb.ListScaciErrorsRequest) (*pb.ListScaciErrorsResponse, error) {
	return s.core.ListScaciErrors(ctx, req)
}

func (s *KiloCenterServiceCompat) ListScaciQueues(ctx context.Context, req *pb.ListScaciQueuesRequest) (*pb.ListScaciQueuesResponse, error) {
	return s.core.ListScaciQueues(ctx, req)
}

func (s *KiloCenterServiceCompat) GetScaciStatus(ctx context.Context, req *pb.GetScaciStatusRequest) (*pb.GetScaciStatusResponse, error) {
	return s.core.GetScaciStatus(ctx, req)
}

// Certificates (6)

func (s *KiloCenterServiceCompat) GenerateCertificate(ctx context.Context, req *pb.GenerateCertificateRequest) (*pb.GenerateCertificateResponse, error) {
	return s.core.GenerateCertificate(ctx, req)
}

func (s *KiloCenterServiceCompat) DownloadCertificate(ctx context.Context, req *pb.DownloadCertificateRequest) (*pb.DownloadCertificateResponse, error) {
	return s.core.DownloadCertificate(ctx, req)
}

func (s *KiloCenterServiceCompat) DownloadBaseStationCertificate(ctx context.Context, req *pb.DownloadBaseStationCertificateRequest) (*pb.DownloadCertificateResponse, error) {
	return s.core.DownloadBaseStationCertificate(ctx, req)
}

func (s *KiloCenterServiceCompat) GenerateServerCertificates(ctx context.Context, req *pb.GenerateServerCertificatesRequest) (*pb.GenerateServerCertificatesResponse, error) {
	return s.core.GenerateServerCertificates(ctx, req)
}

func (s *KiloCenterServiceCompat) RenewServerCertificates(ctx context.Context, req *pb.RenewServerCertificatesRequest) (*pb.RenewServerCertificatesResponse, error) {
	return s.core.RenewServerCertificates(ctx, req)
}

func (s *KiloCenterServiceCompat) GetServerCertificateStatus(ctx context.Context, req *pb.GetServerCertificateStatusRequest) (*pb.GetServerCertificateStatusResponse, error) {
	return s.core.GetServerCertificateStatus(ctx, req)
}

// Manufacturers (5)

func (s *KiloCenterServiceCompat) CreateManufacturer(ctx context.Context, req *pb.CreateManufacturerRequest) (*pb.CreateManufacturerResponse, error) {
	return s.core.CreateManufacturer(ctx, req)
}

func (s *KiloCenterServiceCompat) GetManufacturer(ctx context.Context, req *pb.GetManufacturerRequest) (*pb.GetManufacturerResponse, error) {
	return s.core.GetManufacturer(ctx, req)
}

func (s *KiloCenterServiceCompat) UpdateManufacturer(ctx context.Context, req *pb.UpdateManufacturerRequest) (*pb.UpdateManufacturerResponse, error) {
	return s.core.UpdateManufacturer(ctx, req)
}

func (s *KiloCenterServiceCompat) DeleteManufacturer(ctx context.Context, req *pb.DeleteManufacturerRequest) (*pb.DeleteManufacturerResponse, error) {
	return s.core.DeleteManufacturer(ctx, req)
}

func (s *KiloCenterServiceCompat) ListManufacturers(ctx context.Context, req *pb.ListManufacturersRequest) (*pb.ListManufacturersResponse, error) {
	return s.core.ListManufacturers(ctx, req)
}

// Device Models (5)

func (s *KiloCenterServiceCompat) CreateDeviceModel(ctx context.Context, req *pb.CreateDeviceModelRequest) (*pb.CreateDeviceModelResponse, error) {
	return s.core.CreateDeviceModel(ctx, req)
}

func (s *KiloCenterServiceCompat) GetDeviceModel(ctx context.Context, req *pb.GetDeviceModelRequest) (*pb.GetDeviceModelResponse, error) {
	return s.core.GetDeviceModel(ctx, req)
}

func (s *KiloCenterServiceCompat) UpdateDeviceModel(ctx context.Context, req *pb.UpdateDeviceModelRequest) (*pb.UpdateDeviceModelResponse, error) {
	return s.core.UpdateDeviceModel(ctx, req)
}

func (s *KiloCenterServiceCompat) DeleteDeviceModel(ctx context.Context, req *pb.DeleteDeviceModelRequest) (*pb.DeleteDeviceModelResponse, error) {
	return s.core.DeleteDeviceModel(ctx, req)
}

func (s *KiloCenterServiceCompat) ListDeviceModels(ctx context.Context, req *pb.ListDeviceModelsRequest) (*pb.ListDeviceModelsResponse, error) {
	return s.core.ListDeviceModels(ctx, req)
}

// Blueprints (9)

func (s *KiloCenterServiceCompat) CreateBlueprint(ctx context.Context, req *pb.CreateBlueprintRequest) (*pb.CreateBlueprintResponse, error) {
	return s.core.CreateBlueprint(ctx, req)
}

func (s *KiloCenterServiceCompat) GetBlueprint(ctx context.Context, req *pb.GetBlueprintRequest) (*pb.GetBlueprintResponse, error) {
	return s.core.GetBlueprint(ctx, req)
}

func (s *KiloCenterServiceCompat) UpdateBlueprint(ctx context.Context, req *pb.UpdateBlueprintRequest) (*pb.UpdateBlueprintResponse, error) {
	return s.core.UpdateBlueprint(ctx, req)
}

func (s *KiloCenterServiceCompat) DeleteBlueprint(ctx context.Context, req *pb.DeleteBlueprintRequest) (*pb.DeleteBlueprintResponse, error) {
	return s.core.DeleteBlueprint(ctx, req)
}

func (s *KiloCenterServiceCompat) ListBlueprints(ctx context.Context, req *pb.ListBlueprintsRequest) (*pb.ListBlueprintsResponse, error) {
	return s.core.ListBlueprints(ctx, req)
}

func (s *KiloCenterServiceCompat) SetDefaultBlueprint(ctx context.Context, req *pb.SetDefaultBlueprintRequest) (*pb.SetDefaultBlueprintResponse, error) {
	return s.core.SetDefaultBlueprint(ctx, req)
}

func (s *KiloCenterServiceCompat) SubmitBlueprintToRegistry(ctx context.Context, req *pb.SubmitBlueprintToRegistryRequest) (*pb.SubmitBlueprintToRegistryResponse, error) {
	return s.core.SubmitBlueprintToRegistry(ctx, req)
}

func (s *KiloCenterServiceCompat) CreateDeviceModelWithBlueprint(ctx context.Context, req *pb.CreateDeviceModelWithBlueprintRequest) (*pb.CreateDeviceModelWithBlueprintResponse, error) {
	return s.core.CreateDeviceModelWithBlueprint(ctx, req)
}

func (s *KiloCenterServiceCompat) DecodePreview(ctx context.Context, req *pb.DecodePreviewRequest) (*pb.DecodePreviewResponse, error) {
	return s.core.DecodePreview(ctx, req)
}

// Messages listing/streaming (8)

func (s *KiloCenterServiceCompat) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	return s.core.ListMessages(ctx, req)
}

func (s *KiloCenterServiceCompat) ListBaseStationMessages(ctx context.Context, req *pb.ListBaseStationMessagesRequest) (*pb.ListBaseStationMessagesResponse, error) {
	return s.core.ListBaseStationMessages(ctx, req)
}

func (s *KiloCenterServiceCompat) GetBaseStationMessage(ctx context.Context, req *pb.GetBaseStationMessageRequest) (*pb.GetBaseStationMessageResponse, error) {
	return s.core.GetBaseStationMessage(ctx, req)
}

func (s *KiloCenterServiceCompat) GetBaseStationMessageStats(ctx context.Context, req *pb.GetBaseStationMessageStatsRequest) (*pb.GetBaseStationMessageStatsResponse, error) {
	return s.core.GetBaseStationMessageStats(ctx, req)
}

func (s *KiloCenterServiceCompat) SearchBaseStationMessages(ctx context.Context, req *pb.SearchBaseStationMessagesRequest) (*pb.SearchBaseStationMessagesResponse, error) {
	return s.core.SearchBaseStationMessages(ctx, req)
}

func (s *KiloCenterServiceCompat) ExportBaseStationMessages(ctx context.Context, req *pb.ExportBaseStationMessagesRequest) (*pb.ExportBaseStationMessagesResponse, error) {
	return s.core.ExportBaseStationMessages(ctx, req)
}

func (s *KiloCenterServiceCompat) ListEndpointMessages(ctx context.Context, req *pb.ListEndpointMessagesRequest) (*pb.ListEndpointMessagesResponse, error) {
	return s.core.ListEndpointMessages(ctx, req)
}

// Global Coverage (1)

func (s *KiloCenterServiceCompat) ListAllBaseStationLocations(ctx context.Context, req *pb.ListAllBaseStationLocationsRequest) (*pb.ListAllBaseStationLocationsResponse, error) {
	return s.core.ListAllBaseStationLocations(ctx, req)
}

// Endpoint Stats (2)

func (s *KiloCenterServiceCompat) GetEndPointStats(ctx context.Context, req *pb.GetEndPointStatsRequest) (*pb.GetEndPointStatsResponse, error) {
	return s.core.GetEndPointStats(ctx, req)
}

func (s *KiloCenterServiceCompat) GetEndPointOperations(ctx context.Context, req *pb.GetEndPointOperationsRequest) (*pb.GetEndPointOperationsResponse, error) {
	return s.core.GetEndPointOperations(ctx, req)
}

// CE Onboarding & ECE Registry (4)

func (s *KiloCenterServiceCompat) GetCEStatus(ctx context.Context, req *pb.GetCEStatusRequest) (*pb.GetCEStatusResponse, error) {
	return s.core.GetCEStatus(ctx, req)
}

func (s *KiloCenterServiceCompat) CompleteCEOnboarding(ctx context.Context, req *pb.CompleteCEOnboardingRequest) (*pb.CompleteCEOnboardingResponse, error) {
	return s.core.CompleteCEOnboarding(ctx, req)
}

func (s *KiloCenterServiceCompat) ListCEInstances(ctx context.Context, req *pb.ListCEInstancesRequest) (*pb.ListCEInstancesResponse, error) {
	return s.core.ListCEInstances(ctx, req)
}

func (s *KiloCenterServiceCompat) RevokeCEInstance(ctx context.Context, req *pb.RevokeCEInstanceRequest) (*pb.RevokeCEInstanceResponse, error) {
	return s.core.RevokeCEInstance(ctx, req)
}

//revive:enable:exported
