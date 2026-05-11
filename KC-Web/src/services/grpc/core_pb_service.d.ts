// package: kilocenter.api.v1
// file: core.proto

import * as core_pb from "./core_pb";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import {grpc} from "@improbable-eng/grpc-web";

type CoreServiceCreateEndPoint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateEndPointRequest;
  readonly responseType: typeof core_pb.EndPoint;
};

type CoreServiceGetEndPoint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetEndPointRequest;
  readonly responseType: typeof core_pb.EndPoint;
};

type CoreServiceUpdateEndPoint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateEndPointRequest;
  readonly responseType: typeof core_pb.EndPoint;
};

type CoreServiceDeleteEndPoint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteEndPointRequest;
  readonly responseType: typeof google_protobuf_empty_pb.Empty;
};

type CoreServiceListEndPoints = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEndPointsRequest;
  readonly responseType: typeof core_pb.ListEndPointsResponse;
};

type CoreServiceAttachEndPoint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.AttachEndPointRequest;
  readonly responseType: typeof core_pb.AttachEndPointResponse;
};

type CoreServiceDetachEndPoint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DetachEndPointRequest;
  readonly responseType: typeof core_pb.DetachEndPointResponse;
};

type CoreServiceCreateBaseStation = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateBaseStationRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type CoreServiceGetBaseStation = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type CoreServiceUpdateBaseStation = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateBaseStationRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type CoreServiceDeleteBaseStation = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteBaseStationRequest;
  readonly responseType: typeof google_protobuf_empty_pb.Empty;
};

type CoreServiceListBaseStations = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBaseStationsRequest;
  readonly responseType: typeof core_pb.ListBaseStationsResponse;
};

type CoreServiceGetBaseStationStats = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationStatsRequest;
  readonly responseType: typeof core_pb.GetBaseStationStatsResponse;
};

type CoreServiceUpdateBaseStationEui = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateBaseStationEuiRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type CoreServiceGetMessage = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetMessageRequest;
  readonly responseType: typeof core_pb.Message;
};

type CoreServiceSendDownlink = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SendDownlinkRequest;
  readonly responseType: typeof core_pb.SendDownlinkResponse;
};

type CoreServiceRevokeDownlink = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.RevokeDownlinkRequest;
  readonly responseType: typeof core_pb.RevokeDownlinkResponse;
};

type CoreServiceListDownlinkQueue = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListDownlinkQueueRequest;
  readonly responseType: typeof core_pb.ListDownlinkQueueResponse;
};

type CoreServiceGetDownlinkResults = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDownlinkResultsRequest;
  readonly responseType: typeof core_pb.GetDownlinkResultsResponse;
};

type CoreServiceSendULTransmit = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SendULTransmitRequest;
  readonly responseType: typeof core_pb.SendULTransmitResponse;
};

type CoreServiceRequestBaseStationStatus = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.BaseStationStatusRequest;
  readonly responseType: typeof core_pb.BaseStationStatusResponse;
};

type CoreServiceInitiatePing = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.InitiatePingRequest;
  readonly responseType: typeof core_pb.InitiatePingResponse;
};

type CoreServiceGetDLRXStatus = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDLRXStatusRequest;
  readonly responseType: typeof core_pb.GetDLRXStatusResponse;
};

type CoreServiceQueryDLRXStatus = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.QueryDLRXStatusRequest;
  readonly responseType: typeof core_pb.QueryDLRXStatusResponse;
};

type CoreServiceGetDLRXStatusQueries = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDLRXStatusQueriesRequest;
  readonly responseType: typeof core_pb.GetDLRXStatusQueriesResponse;
};

type CoreServiceGetSystemStatus = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof google_protobuf_empty_pb.Empty;
  readonly responseType: typeof core_pb.SystemStatus;
};

type CoreServiceGetStatistics = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetStatisticsRequest;
  readonly responseType: typeof core_pb.Statistics;
};

type CoreServiceGetReleaseInfo = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof google_protobuf_empty_pb.Empty;
  readonly responseType: typeof core_pb.ReleaseInfo;
};

type CoreServiceCreateIntegration = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateIntegrationRequest;
  readonly responseType: typeof core_pb.Integration;
};

type CoreServiceGetIntegration = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetIntegrationRequest;
  readonly responseType: typeof core_pb.Integration;
};

type CoreServiceUpdateIntegration = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateIntegrationRequest;
  readonly responseType: typeof core_pb.Integration;
};

type CoreServiceDeleteIntegration = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteIntegrationRequest;
  readonly responseType: typeof google_protobuf_empty_pb.Empty;
};

type CoreServiceListIntegrations = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListIntegrationsRequest;
  readonly responseType: typeof core_pb.ListIntegrationsResponse;
};

type CoreServiceGetAnalyticsOverview = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetAnalyticsOverviewRequest;
  readonly responseType: typeof core_pb.GetAnalyticsOverviewResponse;
};

type CoreServiceGetActivityAnalytics = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetActivityAnalyticsRequest;
  readonly responseType: typeof core_pb.GetActivityAnalyticsResponse;
};

type CoreServiceGetSignalQualityAnalytics = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetSignalQualityAnalyticsRequest;
  readonly responseType: typeof core_pb.GetSignalQualityAnalyticsResponse;
};

type CoreServiceListEvents = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEventsRequest;
  readonly responseType: typeof core_pb.ListEventsResponse;
};

type CoreServiceListBaseStationActivity = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBaseStationActivityRequest;
  readonly responseType: typeof core_pb.ListBaseStationActivityResponse;
};

type CoreServiceListEndpointActivity = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEndpointActivityRequest;
  readonly responseType: typeof core_pb.ListEndpointActivityResponse;
};

type CoreServiceStreamEvents = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof core_pb.StreamEventsRequest;
  readonly responseType: typeof core_pb.Event;
};

type CoreServiceListAlerts = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListAlertsRequest;
  readonly responseType: typeof core_pb.ListAlertsResponse;
};

type CoreServiceGetAlertSummary = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetAlertSummaryRequest;
  readonly responseType: typeof core_pb.GetAlertSummaryResponse;
};

type CoreServiceListScaciSessions = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListScaciSessionsRequest;
  readonly responseType: typeof core_pb.ListScaciSessionsResponse;
};

type CoreServiceGetScaciSession = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetScaciSessionRequest;
  readonly responseType: typeof core_pb.GetScaciSessionResponse;
};

type CoreServiceGetScaciStatistics = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetScaciStatisticsRequest;
  readonly responseType: typeof core_pb.GetScaciStatisticsResponse;
};

type CoreServiceListScaciErrors = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListScaciErrorsRequest;
  readonly responseType: typeof core_pb.ListScaciErrorsResponse;
};

type CoreServiceListScaciQueues = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListScaciQueuesRequest;
  readonly responseType: typeof core_pb.ListScaciQueuesResponse;
};

type CoreServiceGetScaciStatus = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetScaciStatusRequest;
  readonly responseType: typeof core_pb.GetScaciStatusResponse;
};

type CoreServiceGenerateCertificate = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GenerateCertificateRequest;
  readonly responseType: typeof core_pb.GenerateCertificateResponse;
};

type CoreServiceDownloadCertificate = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DownloadCertificateRequest;
  readonly responseType: typeof core_pb.DownloadCertificateResponse;
};

type CoreServiceDownloadBaseStationCertificate = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DownloadBaseStationCertificateRequest;
  readonly responseType: typeof core_pb.DownloadCertificateResponse;
};

type CoreServiceGenerateServerCertificates = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GenerateServerCertificatesRequest;
  readonly responseType: typeof core_pb.GenerateServerCertificatesResponse;
};

type CoreServiceRenewServerCertificates = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.RenewServerCertificatesRequest;
  readonly responseType: typeof core_pb.RenewServerCertificatesResponse;
};

type CoreServiceGetServerCertificateStatus = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetServerCertificateStatusRequest;
  readonly responseType: typeof core_pb.GetServerCertificateStatusResponse;
};

type CoreServiceCreateManufacturer = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateManufacturerRequest;
  readonly responseType: typeof core_pb.CreateManufacturerResponse;
};

type CoreServiceGetManufacturer = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetManufacturerRequest;
  readonly responseType: typeof core_pb.GetManufacturerResponse;
};

type CoreServiceUpdateManufacturer = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateManufacturerRequest;
  readonly responseType: typeof core_pb.UpdateManufacturerResponse;
};

type CoreServiceDeleteManufacturer = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteManufacturerRequest;
  readonly responseType: typeof core_pb.DeleteManufacturerResponse;
};

type CoreServiceListManufacturers = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListManufacturersRequest;
  readonly responseType: typeof core_pb.ListManufacturersResponse;
};

type CoreServiceCreateDeviceModel = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateDeviceModelRequest;
  readonly responseType: typeof core_pb.CreateDeviceModelResponse;
};

type CoreServiceGetDeviceModel = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDeviceModelRequest;
  readonly responseType: typeof core_pb.GetDeviceModelResponse;
};

type CoreServiceUpdateDeviceModel = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateDeviceModelRequest;
  readonly responseType: typeof core_pb.UpdateDeviceModelResponse;
};

type CoreServiceDeleteDeviceModel = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteDeviceModelRequest;
  readonly responseType: typeof core_pb.DeleteDeviceModelResponse;
};

type CoreServiceListDeviceModels = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListDeviceModelsRequest;
  readonly responseType: typeof core_pb.ListDeviceModelsResponse;
};

type CoreServiceCreateBlueprint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateBlueprintRequest;
  readonly responseType: typeof core_pb.CreateBlueprintResponse;
};

type CoreServiceGetBlueprint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBlueprintRequest;
  readonly responseType: typeof core_pb.GetBlueprintResponse;
};

type CoreServiceUpdateBlueprint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateBlueprintRequest;
  readonly responseType: typeof core_pb.UpdateBlueprintResponse;
};

type CoreServiceDeleteBlueprint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteBlueprintRequest;
  readonly responseType: typeof core_pb.DeleteBlueprintResponse;
};

type CoreServiceListBlueprints = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBlueprintsRequest;
  readonly responseType: typeof core_pb.ListBlueprintsResponse;
};

type CoreServiceSetDefaultBlueprint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SetDefaultBlueprintRequest;
  readonly responseType: typeof core_pb.SetDefaultBlueprintResponse;
};

type CoreServiceSubmitBlueprintToRegistry = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SubmitBlueprintToRegistryRequest;
  readonly responseType: typeof core_pb.SubmitBlueprintToRegistryResponse;
};

type CoreServiceCreateDeviceModelWithBlueprint = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateDeviceModelWithBlueprintRequest;
  readonly responseType: typeof core_pb.CreateDeviceModelWithBlueprintResponse;
};

type CoreServiceDecodePreview = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DecodePreviewRequest;
  readonly responseType: typeof core_pb.DecodePreviewResponse;
};

type CoreServiceListMessages = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListMessagesRequest;
  readonly responseType: typeof core_pb.ListMessagesResponse;
};

type CoreServiceStreamMessages = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof core_pb.StreamMessagesRequest;
  readonly responseType: typeof core_pb.Message;
};

type CoreServiceListBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.ListBaseStationMessagesResponse;
};

type CoreServiceGetBaseStationMessage = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationMessageRequest;
  readonly responseType: typeof core_pb.GetBaseStationMessageResponse;
};

type CoreServiceGetBaseStationMessageStats = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationMessageStatsRequest;
  readonly responseType: typeof core_pb.GetBaseStationMessageStatsResponse;
};

type CoreServiceSearchBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SearchBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.SearchBaseStationMessagesResponse;
};

type CoreServiceExportBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ExportBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.ExportBaseStationMessagesResponse;
};

type CoreServiceStreamBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof core_pb.StreamBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.BaseStationMessage;
};

type CoreServiceListEndpointMessages = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEndpointMessagesRequest;
  readonly responseType: typeof core_pb.ListEndpointMessagesResponse;
};

type CoreServiceGetEndPointStats = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetEndPointStatsRequest;
  readonly responseType: typeof core_pb.GetEndPointStatsResponse;
};

type CoreServiceGetEndPointOperations = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetEndPointOperationsRequest;
  readonly responseType: typeof core_pb.GetEndPointOperationsResponse;
};

type CoreServiceListAllBaseStationLocations = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListAllBaseStationLocationsRequest;
  readonly responseType: typeof core_pb.ListAllBaseStationLocationsResponse;
};

type CoreServiceGetCEStatus = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetCEStatusRequest;
  readonly responseType: typeof core_pb.GetCEStatusResponse;
};

type CoreServiceCompleteCEOnboarding = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CompleteCEOnboardingRequest;
  readonly responseType: typeof core_pb.CompleteCEOnboardingResponse;
};

type CoreServiceListCEInstances = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListCEInstancesRequest;
  readonly responseType: typeof core_pb.ListCEInstancesResponse;
};

type CoreServiceRevokeCEInstance = {
  readonly methodName: string;
  readonly service: typeof CoreService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.RevokeCEInstanceRequest;
  readonly responseType: typeof core_pb.RevokeCEInstanceResponse;
};

export class CoreService {
  static readonly serviceName: string;
  static readonly CreateEndPoint: CoreServiceCreateEndPoint;
  static readonly GetEndPoint: CoreServiceGetEndPoint;
  static readonly UpdateEndPoint: CoreServiceUpdateEndPoint;
  static readonly DeleteEndPoint: CoreServiceDeleteEndPoint;
  static readonly ListEndPoints: CoreServiceListEndPoints;
  static readonly AttachEndPoint: CoreServiceAttachEndPoint;
  static readonly DetachEndPoint: CoreServiceDetachEndPoint;
  static readonly CreateBaseStation: CoreServiceCreateBaseStation;
  static readonly GetBaseStation: CoreServiceGetBaseStation;
  static readonly UpdateBaseStation: CoreServiceUpdateBaseStation;
  static readonly DeleteBaseStation: CoreServiceDeleteBaseStation;
  static readonly ListBaseStations: CoreServiceListBaseStations;
  static readonly GetBaseStationStats: CoreServiceGetBaseStationStats;
  static readonly UpdateBaseStationEui: CoreServiceUpdateBaseStationEui;
  static readonly GetMessage: CoreServiceGetMessage;
  static readonly SendDownlink: CoreServiceSendDownlink;
  static readonly RevokeDownlink: CoreServiceRevokeDownlink;
  static readonly ListDownlinkQueue: CoreServiceListDownlinkQueue;
  static readonly GetDownlinkResults: CoreServiceGetDownlinkResults;
  static readonly SendULTransmit: CoreServiceSendULTransmit;
  static readonly RequestBaseStationStatus: CoreServiceRequestBaseStationStatus;
  static readonly InitiatePing: CoreServiceInitiatePing;
  static readonly GetDLRXStatus: CoreServiceGetDLRXStatus;
  static readonly QueryDLRXStatus: CoreServiceQueryDLRXStatus;
  static readonly GetDLRXStatusQueries: CoreServiceGetDLRXStatusQueries;
  static readonly GetSystemStatus: CoreServiceGetSystemStatus;
  static readonly GetStatistics: CoreServiceGetStatistics;
  static readonly GetReleaseInfo: CoreServiceGetReleaseInfo;
  static readonly CreateIntegration: CoreServiceCreateIntegration;
  static readonly GetIntegration: CoreServiceGetIntegration;
  static readonly UpdateIntegration: CoreServiceUpdateIntegration;
  static readonly DeleteIntegration: CoreServiceDeleteIntegration;
  static readonly ListIntegrations: CoreServiceListIntegrations;
  static readonly GetAnalyticsOverview: CoreServiceGetAnalyticsOverview;
  static readonly GetActivityAnalytics: CoreServiceGetActivityAnalytics;
  static readonly GetSignalQualityAnalytics: CoreServiceGetSignalQualityAnalytics;
  static readonly ListEvents: CoreServiceListEvents;
  static readonly ListBaseStationActivity: CoreServiceListBaseStationActivity;
  static readonly ListEndpointActivity: CoreServiceListEndpointActivity;
  static readonly StreamEvents: CoreServiceStreamEvents;
  static readonly ListAlerts: CoreServiceListAlerts;
  static readonly GetAlertSummary: CoreServiceGetAlertSummary;
  static readonly ListScaciSessions: CoreServiceListScaciSessions;
  static readonly GetScaciSession: CoreServiceGetScaciSession;
  static readonly GetScaciStatistics: CoreServiceGetScaciStatistics;
  static readonly ListScaciErrors: CoreServiceListScaciErrors;
  static readonly ListScaciQueues: CoreServiceListScaciQueues;
  static readonly GetScaciStatus: CoreServiceGetScaciStatus;
  static readonly GenerateCertificate: CoreServiceGenerateCertificate;
  static readonly DownloadCertificate: CoreServiceDownloadCertificate;
  static readonly DownloadBaseStationCertificate: CoreServiceDownloadBaseStationCertificate;
  static readonly GenerateServerCertificates: CoreServiceGenerateServerCertificates;
  static readonly RenewServerCertificates: CoreServiceRenewServerCertificates;
  static readonly GetServerCertificateStatus: CoreServiceGetServerCertificateStatus;
  static readonly CreateManufacturer: CoreServiceCreateManufacturer;
  static readonly GetManufacturer: CoreServiceGetManufacturer;
  static readonly UpdateManufacturer: CoreServiceUpdateManufacturer;
  static readonly DeleteManufacturer: CoreServiceDeleteManufacturer;
  static readonly ListManufacturers: CoreServiceListManufacturers;
  static readonly CreateDeviceModel: CoreServiceCreateDeviceModel;
  static readonly GetDeviceModel: CoreServiceGetDeviceModel;
  static readonly UpdateDeviceModel: CoreServiceUpdateDeviceModel;
  static readonly DeleteDeviceModel: CoreServiceDeleteDeviceModel;
  static readonly ListDeviceModels: CoreServiceListDeviceModels;
  static readonly CreateBlueprint: CoreServiceCreateBlueprint;
  static readonly GetBlueprint: CoreServiceGetBlueprint;
  static readonly UpdateBlueprint: CoreServiceUpdateBlueprint;
  static readonly DeleteBlueprint: CoreServiceDeleteBlueprint;
  static readonly ListBlueprints: CoreServiceListBlueprints;
  static readonly SetDefaultBlueprint: CoreServiceSetDefaultBlueprint;
  static readonly SubmitBlueprintToRegistry: CoreServiceSubmitBlueprintToRegistry;
  static readonly CreateDeviceModelWithBlueprint: CoreServiceCreateDeviceModelWithBlueprint;
  static readonly DecodePreview: CoreServiceDecodePreview;
  static readonly ListMessages: CoreServiceListMessages;
  static readonly StreamMessages: CoreServiceStreamMessages;
  static readonly ListBaseStationMessages: CoreServiceListBaseStationMessages;
  static readonly GetBaseStationMessage: CoreServiceGetBaseStationMessage;
  static readonly GetBaseStationMessageStats: CoreServiceGetBaseStationMessageStats;
  static readonly SearchBaseStationMessages: CoreServiceSearchBaseStationMessages;
  static readonly ExportBaseStationMessages: CoreServiceExportBaseStationMessages;
  static readonly StreamBaseStationMessages: CoreServiceStreamBaseStationMessages;
  static readonly ListEndpointMessages: CoreServiceListEndpointMessages;
  static readonly GetEndPointStats: CoreServiceGetEndPointStats;
  static readonly GetEndPointOperations: CoreServiceGetEndPointOperations;
  static readonly ListAllBaseStationLocations: CoreServiceListAllBaseStationLocations;
  static readonly GetCEStatus: CoreServiceGetCEStatus;
  static readonly CompleteCEOnboarding: CoreServiceCompleteCEOnboarding;
  static readonly ListCEInstances: CoreServiceListCEInstances;
  static readonly RevokeCEInstance: CoreServiceRevokeCEInstance;
}

export type ServiceError = { message: string, code: number; metadata: grpc.Metadata }
export type Status = { details: string, code: number; metadata: grpc.Metadata }

interface UnaryResponse {
  cancel(): void;
}
interface ResponseStream<T> {
  cancel(): void;
  on(type: 'data', handler: (message: T) => void): ResponseStream<T>;
  on(type: 'end', handler: (status?: Status) => void): ResponseStream<T>;
  on(type: 'status', handler: (status: Status) => void): ResponseStream<T>;
}
interface RequestStream<T> {
  write(message: T): RequestStream<T>;
  end(): void;
  cancel(): void;
  on(type: 'end', handler: (status?: Status) => void): RequestStream<T>;
  on(type: 'status', handler: (status: Status) => void): RequestStream<T>;
}
interface BidirectionalStream<ReqT, ResT> {
  write(message: ReqT): BidirectionalStream<ReqT, ResT>;
  end(): void;
  cancel(): void;
  on(type: 'data', handler: (message: ResT) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'end', handler: (status?: Status) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'status', handler: (status: Status) => void): BidirectionalStream<ReqT, ResT>;
}

export class CoreServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  createEndPoint(
    requestMessage: core_pb.CreateEndPointRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.EndPoint|null) => void
  ): UnaryResponse;
  createEndPoint(
    requestMessage: core_pb.CreateEndPointRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.EndPoint|null) => void
  ): UnaryResponse;
  getEndPoint(
    requestMessage: core_pb.GetEndPointRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.EndPoint|null) => void
  ): UnaryResponse;
  getEndPoint(
    requestMessage: core_pb.GetEndPointRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.EndPoint|null) => void
  ): UnaryResponse;
  updateEndPoint(
    requestMessage: core_pb.UpdateEndPointRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.EndPoint|null) => void
  ): UnaryResponse;
  updateEndPoint(
    requestMessage: core_pb.UpdateEndPointRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.EndPoint|null) => void
  ): UnaryResponse;
  deleteEndPoint(
    requestMessage: core_pb.DeleteEndPointRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: google_protobuf_empty_pb.Empty|null) => void
  ): UnaryResponse;
  deleteEndPoint(
    requestMessage: core_pb.DeleteEndPointRequest,
    callback: (error: ServiceError|null, responseMessage: google_protobuf_empty_pb.Empty|null) => void
  ): UnaryResponse;
  listEndPoints(
    requestMessage: core_pb.ListEndPointsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEndPointsResponse|null) => void
  ): UnaryResponse;
  listEndPoints(
    requestMessage: core_pb.ListEndPointsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEndPointsResponse|null) => void
  ): UnaryResponse;
  attachEndPoint(
    requestMessage: core_pb.AttachEndPointRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.AttachEndPointResponse|null) => void
  ): UnaryResponse;
  attachEndPoint(
    requestMessage: core_pb.AttachEndPointRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.AttachEndPointResponse|null) => void
  ): UnaryResponse;
  detachEndPoint(
    requestMessage: core_pb.DetachEndPointRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.DetachEndPointResponse|null) => void
  ): UnaryResponse;
  detachEndPoint(
    requestMessage: core_pb.DetachEndPointRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.DetachEndPointResponse|null) => void
  ): UnaryResponse;
  createBaseStation(
    requestMessage: core_pb.CreateBaseStationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  createBaseStation(
    requestMessage: core_pb.CreateBaseStationRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  getBaseStation(
    requestMessage: core_pb.GetBaseStationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  getBaseStation(
    requestMessage: core_pb.GetBaseStationRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  updateBaseStation(
    requestMessage: core_pb.UpdateBaseStationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  updateBaseStation(
    requestMessage: core_pb.UpdateBaseStationRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  deleteBaseStation(
    requestMessage: core_pb.DeleteBaseStationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: google_protobuf_empty_pb.Empty|null) => void
  ): UnaryResponse;
  deleteBaseStation(
    requestMessage: core_pb.DeleteBaseStationRequest,
    callback: (error: ServiceError|null, responseMessage: google_protobuf_empty_pb.Empty|null) => void
  ): UnaryResponse;
  listBaseStations(
    requestMessage: core_pb.ListBaseStationsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBaseStationsResponse|null) => void
  ): UnaryResponse;
  listBaseStations(
    requestMessage: core_pb.ListBaseStationsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBaseStationsResponse|null) => void
  ): UnaryResponse;
  getBaseStationStats(
    requestMessage: core_pb.GetBaseStationStatsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationStatsResponse|null) => void
  ): UnaryResponse;
  getBaseStationStats(
    requestMessage: core_pb.GetBaseStationStatsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationStatsResponse|null) => void
  ): UnaryResponse;
  updateBaseStationEui(
    requestMessage: core_pb.UpdateBaseStationEuiRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  updateBaseStationEui(
    requestMessage: core_pb.UpdateBaseStationEuiRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStation|null) => void
  ): UnaryResponse;
  getMessage(
    requestMessage: core_pb.GetMessageRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.Message|null) => void
  ): UnaryResponse;
  getMessage(
    requestMessage: core_pb.GetMessageRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.Message|null) => void
  ): UnaryResponse;
  sendDownlink(
    requestMessage: core_pb.SendDownlinkRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.SendDownlinkResponse|null) => void
  ): UnaryResponse;
  sendDownlink(
    requestMessage: core_pb.SendDownlinkRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.SendDownlinkResponse|null) => void
  ): UnaryResponse;
  revokeDownlink(
    requestMessage: core_pb.RevokeDownlinkRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.RevokeDownlinkResponse|null) => void
  ): UnaryResponse;
  revokeDownlink(
    requestMessage: core_pb.RevokeDownlinkRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.RevokeDownlinkResponse|null) => void
  ): UnaryResponse;
  listDownlinkQueue(
    requestMessage: core_pb.ListDownlinkQueueRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListDownlinkQueueResponse|null) => void
  ): UnaryResponse;
  listDownlinkQueue(
    requestMessage: core_pb.ListDownlinkQueueRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListDownlinkQueueResponse|null) => void
  ): UnaryResponse;
  getDownlinkResults(
    requestMessage: core_pb.GetDownlinkResultsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDownlinkResultsResponse|null) => void
  ): UnaryResponse;
  getDownlinkResults(
    requestMessage: core_pb.GetDownlinkResultsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDownlinkResultsResponse|null) => void
  ): UnaryResponse;
  sendULTransmit(
    requestMessage: core_pb.SendULTransmitRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.SendULTransmitResponse|null) => void
  ): UnaryResponse;
  sendULTransmit(
    requestMessage: core_pb.SendULTransmitRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.SendULTransmitResponse|null) => void
  ): UnaryResponse;
  requestBaseStationStatus(
    requestMessage: core_pb.BaseStationStatusRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStationStatusResponse|null) => void
  ): UnaryResponse;
  requestBaseStationStatus(
    requestMessage: core_pb.BaseStationStatusRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.BaseStationStatusResponse|null) => void
  ): UnaryResponse;
  initiatePing(
    requestMessage: core_pb.InitiatePingRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.InitiatePingResponse|null) => void
  ): UnaryResponse;
  initiatePing(
    requestMessage: core_pb.InitiatePingRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.InitiatePingResponse|null) => void
  ): UnaryResponse;
  getDLRXStatus(
    requestMessage: core_pb.GetDLRXStatusRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDLRXStatusResponse|null) => void
  ): UnaryResponse;
  getDLRXStatus(
    requestMessage: core_pb.GetDLRXStatusRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDLRXStatusResponse|null) => void
  ): UnaryResponse;
  queryDLRXStatus(
    requestMessage: core_pb.QueryDLRXStatusRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.QueryDLRXStatusResponse|null) => void
  ): UnaryResponse;
  queryDLRXStatus(
    requestMessage: core_pb.QueryDLRXStatusRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.QueryDLRXStatusResponse|null) => void
  ): UnaryResponse;
  getDLRXStatusQueries(
    requestMessage: core_pb.GetDLRXStatusQueriesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDLRXStatusQueriesResponse|null) => void
  ): UnaryResponse;
  getDLRXStatusQueries(
    requestMessage: core_pb.GetDLRXStatusQueriesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDLRXStatusQueriesResponse|null) => void
  ): UnaryResponse;
  getSystemStatus(
    requestMessage: google_protobuf_empty_pb.Empty,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.SystemStatus|null) => void
  ): UnaryResponse;
  getSystemStatus(
    requestMessage: google_protobuf_empty_pb.Empty,
    callback: (error: ServiceError|null, responseMessage: core_pb.SystemStatus|null) => void
  ): UnaryResponse;
  getStatistics(
    requestMessage: core_pb.GetStatisticsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.Statistics|null) => void
  ): UnaryResponse;
  getStatistics(
    requestMessage: core_pb.GetStatisticsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.Statistics|null) => void
  ): UnaryResponse;
  getReleaseInfo(
    requestMessage: google_protobuf_empty_pb.Empty,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ReleaseInfo|null) => void
  ): UnaryResponse;
  getReleaseInfo(
    requestMessage: google_protobuf_empty_pb.Empty,
    callback: (error: ServiceError|null, responseMessage: core_pb.ReleaseInfo|null) => void
  ): UnaryResponse;
  createIntegration(
    requestMessage: core_pb.CreateIntegrationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.Integration|null) => void
  ): UnaryResponse;
  createIntegration(
    requestMessage: core_pb.CreateIntegrationRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.Integration|null) => void
  ): UnaryResponse;
  getIntegration(
    requestMessage: core_pb.GetIntegrationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.Integration|null) => void
  ): UnaryResponse;
  getIntegration(
    requestMessage: core_pb.GetIntegrationRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.Integration|null) => void
  ): UnaryResponse;
  updateIntegration(
    requestMessage: core_pb.UpdateIntegrationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.Integration|null) => void
  ): UnaryResponse;
  updateIntegration(
    requestMessage: core_pb.UpdateIntegrationRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.Integration|null) => void
  ): UnaryResponse;
  deleteIntegration(
    requestMessage: core_pb.DeleteIntegrationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: google_protobuf_empty_pb.Empty|null) => void
  ): UnaryResponse;
  deleteIntegration(
    requestMessage: core_pb.DeleteIntegrationRequest,
    callback: (error: ServiceError|null, responseMessage: google_protobuf_empty_pb.Empty|null) => void
  ): UnaryResponse;
  listIntegrations(
    requestMessage: core_pb.ListIntegrationsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListIntegrationsResponse|null) => void
  ): UnaryResponse;
  listIntegrations(
    requestMessage: core_pb.ListIntegrationsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListIntegrationsResponse|null) => void
  ): UnaryResponse;
  getAnalyticsOverview(
    requestMessage: core_pb.GetAnalyticsOverviewRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetAnalyticsOverviewResponse|null) => void
  ): UnaryResponse;
  getAnalyticsOverview(
    requestMessage: core_pb.GetAnalyticsOverviewRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetAnalyticsOverviewResponse|null) => void
  ): UnaryResponse;
  getActivityAnalytics(
    requestMessage: core_pb.GetActivityAnalyticsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetActivityAnalyticsResponse|null) => void
  ): UnaryResponse;
  getActivityAnalytics(
    requestMessage: core_pb.GetActivityAnalyticsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetActivityAnalyticsResponse|null) => void
  ): UnaryResponse;
  getSignalQualityAnalytics(
    requestMessage: core_pb.GetSignalQualityAnalyticsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetSignalQualityAnalyticsResponse|null) => void
  ): UnaryResponse;
  getSignalQualityAnalytics(
    requestMessage: core_pb.GetSignalQualityAnalyticsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetSignalQualityAnalyticsResponse|null) => void
  ): UnaryResponse;
  listEvents(
    requestMessage: core_pb.ListEventsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEventsResponse|null) => void
  ): UnaryResponse;
  listEvents(
    requestMessage: core_pb.ListEventsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEventsResponse|null) => void
  ): UnaryResponse;
  listBaseStationActivity(
    requestMessage: core_pb.ListBaseStationActivityRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBaseStationActivityResponse|null) => void
  ): UnaryResponse;
  listBaseStationActivity(
    requestMessage: core_pb.ListBaseStationActivityRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBaseStationActivityResponse|null) => void
  ): UnaryResponse;
  listEndpointActivity(
    requestMessage: core_pb.ListEndpointActivityRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEndpointActivityResponse|null) => void
  ): UnaryResponse;
  listEndpointActivity(
    requestMessage: core_pb.ListEndpointActivityRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEndpointActivityResponse|null) => void
  ): UnaryResponse;
  streamEvents(requestMessage: core_pb.StreamEventsRequest, metadata?: grpc.Metadata): ResponseStream<core_pb.Event>;
  listAlerts(
    requestMessage: core_pb.ListAlertsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListAlertsResponse|null) => void
  ): UnaryResponse;
  listAlerts(
    requestMessage: core_pb.ListAlertsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListAlertsResponse|null) => void
  ): UnaryResponse;
  getAlertSummary(
    requestMessage: core_pb.GetAlertSummaryRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetAlertSummaryResponse|null) => void
  ): UnaryResponse;
  getAlertSummary(
    requestMessage: core_pb.GetAlertSummaryRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetAlertSummaryResponse|null) => void
  ): UnaryResponse;
  listScaciSessions(
    requestMessage: core_pb.ListScaciSessionsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListScaciSessionsResponse|null) => void
  ): UnaryResponse;
  listScaciSessions(
    requestMessage: core_pb.ListScaciSessionsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListScaciSessionsResponse|null) => void
  ): UnaryResponse;
  getScaciSession(
    requestMessage: core_pb.GetScaciSessionRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetScaciSessionResponse|null) => void
  ): UnaryResponse;
  getScaciSession(
    requestMessage: core_pb.GetScaciSessionRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetScaciSessionResponse|null) => void
  ): UnaryResponse;
  getScaciStatistics(
    requestMessage: core_pb.GetScaciStatisticsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetScaciStatisticsResponse|null) => void
  ): UnaryResponse;
  getScaciStatistics(
    requestMessage: core_pb.GetScaciStatisticsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetScaciStatisticsResponse|null) => void
  ): UnaryResponse;
  listScaciErrors(
    requestMessage: core_pb.ListScaciErrorsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListScaciErrorsResponse|null) => void
  ): UnaryResponse;
  listScaciErrors(
    requestMessage: core_pb.ListScaciErrorsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListScaciErrorsResponse|null) => void
  ): UnaryResponse;
  listScaciQueues(
    requestMessage: core_pb.ListScaciQueuesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListScaciQueuesResponse|null) => void
  ): UnaryResponse;
  listScaciQueues(
    requestMessage: core_pb.ListScaciQueuesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListScaciQueuesResponse|null) => void
  ): UnaryResponse;
  getScaciStatus(
    requestMessage: core_pb.GetScaciStatusRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetScaciStatusResponse|null) => void
  ): UnaryResponse;
  getScaciStatus(
    requestMessage: core_pb.GetScaciStatusRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetScaciStatusResponse|null) => void
  ): UnaryResponse;
  generateCertificate(
    requestMessage: core_pb.GenerateCertificateRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GenerateCertificateResponse|null) => void
  ): UnaryResponse;
  generateCertificate(
    requestMessage: core_pb.GenerateCertificateRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GenerateCertificateResponse|null) => void
  ): UnaryResponse;
  downloadCertificate(
    requestMessage: core_pb.DownloadCertificateRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.DownloadCertificateResponse|null) => void
  ): UnaryResponse;
  downloadCertificate(
    requestMessage: core_pb.DownloadCertificateRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.DownloadCertificateResponse|null) => void
  ): UnaryResponse;
  downloadBaseStationCertificate(
    requestMessage: core_pb.DownloadBaseStationCertificateRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.DownloadCertificateResponse|null) => void
  ): UnaryResponse;
  downloadBaseStationCertificate(
    requestMessage: core_pb.DownloadBaseStationCertificateRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.DownloadCertificateResponse|null) => void
  ): UnaryResponse;
  generateServerCertificates(
    requestMessage: core_pb.GenerateServerCertificatesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GenerateServerCertificatesResponse|null) => void
  ): UnaryResponse;
  generateServerCertificates(
    requestMessage: core_pb.GenerateServerCertificatesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GenerateServerCertificatesResponse|null) => void
  ): UnaryResponse;
  renewServerCertificates(
    requestMessage: core_pb.RenewServerCertificatesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.RenewServerCertificatesResponse|null) => void
  ): UnaryResponse;
  renewServerCertificates(
    requestMessage: core_pb.RenewServerCertificatesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.RenewServerCertificatesResponse|null) => void
  ): UnaryResponse;
  getServerCertificateStatus(
    requestMessage: core_pb.GetServerCertificateStatusRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetServerCertificateStatusResponse|null) => void
  ): UnaryResponse;
  getServerCertificateStatus(
    requestMessage: core_pb.GetServerCertificateStatusRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetServerCertificateStatusResponse|null) => void
  ): UnaryResponse;
  createManufacturer(
    requestMessage: core_pb.CreateManufacturerRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateManufacturerResponse|null) => void
  ): UnaryResponse;
  createManufacturer(
    requestMessage: core_pb.CreateManufacturerRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateManufacturerResponse|null) => void
  ): UnaryResponse;
  getManufacturer(
    requestMessage: core_pb.GetManufacturerRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetManufacturerResponse|null) => void
  ): UnaryResponse;
  getManufacturer(
    requestMessage: core_pb.GetManufacturerRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetManufacturerResponse|null) => void
  ): UnaryResponse;
  updateManufacturer(
    requestMessage: core_pb.UpdateManufacturerRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.UpdateManufacturerResponse|null) => void
  ): UnaryResponse;
  updateManufacturer(
    requestMessage: core_pb.UpdateManufacturerRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.UpdateManufacturerResponse|null) => void
  ): UnaryResponse;
  deleteManufacturer(
    requestMessage: core_pb.DeleteManufacturerRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.DeleteManufacturerResponse|null) => void
  ): UnaryResponse;
  deleteManufacturer(
    requestMessage: core_pb.DeleteManufacturerRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.DeleteManufacturerResponse|null) => void
  ): UnaryResponse;
  listManufacturers(
    requestMessage: core_pb.ListManufacturersRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListManufacturersResponse|null) => void
  ): UnaryResponse;
  listManufacturers(
    requestMessage: core_pb.ListManufacturersRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListManufacturersResponse|null) => void
  ): UnaryResponse;
  createDeviceModel(
    requestMessage: core_pb.CreateDeviceModelRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateDeviceModelResponse|null) => void
  ): UnaryResponse;
  createDeviceModel(
    requestMessage: core_pb.CreateDeviceModelRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateDeviceModelResponse|null) => void
  ): UnaryResponse;
  getDeviceModel(
    requestMessage: core_pb.GetDeviceModelRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDeviceModelResponse|null) => void
  ): UnaryResponse;
  getDeviceModel(
    requestMessage: core_pb.GetDeviceModelRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetDeviceModelResponse|null) => void
  ): UnaryResponse;
  updateDeviceModel(
    requestMessage: core_pb.UpdateDeviceModelRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.UpdateDeviceModelResponse|null) => void
  ): UnaryResponse;
  updateDeviceModel(
    requestMessage: core_pb.UpdateDeviceModelRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.UpdateDeviceModelResponse|null) => void
  ): UnaryResponse;
  deleteDeviceModel(
    requestMessage: core_pb.DeleteDeviceModelRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.DeleteDeviceModelResponse|null) => void
  ): UnaryResponse;
  deleteDeviceModel(
    requestMessage: core_pb.DeleteDeviceModelRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.DeleteDeviceModelResponse|null) => void
  ): UnaryResponse;
  listDeviceModels(
    requestMessage: core_pb.ListDeviceModelsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListDeviceModelsResponse|null) => void
  ): UnaryResponse;
  listDeviceModels(
    requestMessage: core_pb.ListDeviceModelsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListDeviceModelsResponse|null) => void
  ): UnaryResponse;
  createBlueprint(
    requestMessage: core_pb.CreateBlueprintRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateBlueprintResponse|null) => void
  ): UnaryResponse;
  createBlueprint(
    requestMessage: core_pb.CreateBlueprintRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateBlueprintResponse|null) => void
  ): UnaryResponse;
  getBlueprint(
    requestMessage: core_pb.GetBlueprintRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBlueprintResponse|null) => void
  ): UnaryResponse;
  getBlueprint(
    requestMessage: core_pb.GetBlueprintRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBlueprintResponse|null) => void
  ): UnaryResponse;
  updateBlueprint(
    requestMessage: core_pb.UpdateBlueprintRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.UpdateBlueprintResponse|null) => void
  ): UnaryResponse;
  updateBlueprint(
    requestMessage: core_pb.UpdateBlueprintRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.UpdateBlueprintResponse|null) => void
  ): UnaryResponse;
  deleteBlueprint(
    requestMessage: core_pb.DeleteBlueprintRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.DeleteBlueprintResponse|null) => void
  ): UnaryResponse;
  deleteBlueprint(
    requestMessage: core_pb.DeleteBlueprintRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.DeleteBlueprintResponse|null) => void
  ): UnaryResponse;
  listBlueprints(
    requestMessage: core_pb.ListBlueprintsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBlueprintsResponse|null) => void
  ): UnaryResponse;
  listBlueprints(
    requestMessage: core_pb.ListBlueprintsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBlueprintsResponse|null) => void
  ): UnaryResponse;
  setDefaultBlueprint(
    requestMessage: core_pb.SetDefaultBlueprintRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.SetDefaultBlueprintResponse|null) => void
  ): UnaryResponse;
  setDefaultBlueprint(
    requestMessage: core_pb.SetDefaultBlueprintRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.SetDefaultBlueprintResponse|null) => void
  ): UnaryResponse;
  submitBlueprintToRegistry(
    requestMessage: core_pb.SubmitBlueprintToRegistryRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.SubmitBlueprintToRegistryResponse|null) => void
  ): UnaryResponse;
  submitBlueprintToRegistry(
    requestMessage: core_pb.SubmitBlueprintToRegistryRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.SubmitBlueprintToRegistryResponse|null) => void
  ): UnaryResponse;
  createDeviceModelWithBlueprint(
    requestMessage: core_pb.CreateDeviceModelWithBlueprintRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateDeviceModelWithBlueprintResponse|null) => void
  ): UnaryResponse;
  createDeviceModelWithBlueprint(
    requestMessage: core_pb.CreateDeviceModelWithBlueprintRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.CreateDeviceModelWithBlueprintResponse|null) => void
  ): UnaryResponse;
  decodePreview(
    requestMessage: core_pb.DecodePreviewRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.DecodePreviewResponse|null) => void
  ): UnaryResponse;
  decodePreview(
    requestMessage: core_pb.DecodePreviewRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.DecodePreviewResponse|null) => void
  ): UnaryResponse;
  listMessages(
    requestMessage: core_pb.ListMessagesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListMessagesResponse|null) => void
  ): UnaryResponse;
  listMessages(
    requestMessage: core_pb.ListMessagesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListMessagesResponse|null) => void
  ): UnaryResponse;
  streamMessages(requestMessage: core_pb.StreamMessagesRequest, metadata?: grpc.Metadata): ResponseStream<core_pb.Message>;
  listBaseStationMessages(
    requestMessage: core_pb.ListBaseStationMessagesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBaseStationMessagesResponse|null) => void
  ): UnaryResponse;
  listBaseStationMessages(
    requestMessage: core_pb.ListBaseStationMessagesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListBaseStationMessagesResponse|null) => void
  ): UnaryResponse;
  getBaseStationMessage(
    requestMessage: core_pb.GetBaseStationMessageRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationMessageResponse|null) => void
  ): UnaryResponse;
  getBaseStationMessage(
    requestMessage: core_pb.GetBaseStationMessageRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationMessageResponse|null) => void
  ): UnaryResponse;
  getBaseStationMessageStats(
    requestMessage: core_pb.GetBaseStationMessageStatsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationMessageStatsResponse|null) => void
  ): UnaryResponse;
  getBaseStationMessageStats(
    requestMessage: core_pb.GetBaseStationMessageStatsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationMessageStatsResponse|null) => void
  ): UnaryResponse;
  searchBaseStationMessages(
    requestMessage: core_pb.SearchBaseStationMessagesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.SearchBaseStationMessagesResponse|null) => void
  ): UnaryResponse;
  searchBaseStationMessages(
    requestMessage: core_pb.SearchBaseStationMessagesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.SearchBaseStationMessagesResponse|null) => void
  ): UnaryResponse;
  exportBaseStationMessages(
    requestMessage: core_pb.ExportBaseStationMessagesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ExportBaseStationMessagesResponse|null) => void
  ): UnaryResponse;
  exportBaseStationMessages(
    requestMessage: core_pb.ExportBaseStationMessagesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ExportBaseStationMessagesResponse|null) => void
  ): UnaryResponse;
  streamBaseStationMessages(requestMessage: core_pb.StreamBaseStationMessagesRequest, metadata?: grpc.Metadata): ResponseStream<core_pb.BaseStationMessage>;
  listEndpointMessages(
    requestMessage: core_pb.ListEndpointMessagesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEndpointMessagesResponse|null) => void
  ): UnaryResponse;
  listEndpointMessages(
    requestMessage: core_pb.ListEndpointMessagesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListEndpointMessagesResponse|null) => void
  ): UnaryResponse;
  getEndPointStats(
    requestMessage: core_pb.GetEndPointStatsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetEndPointStatsResponse|null) => void
  ): UnaryResponse;
  getEndPointStats(
    requestMessage: core_pb.GetEndPointStatsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetEndPointStatsResponse|null) => void
  ): UnaryResponse;
  getEndPointOperations(
    requestMessage: core_pb.GetEndPointOperationsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetEndPointOperationsResponse|null) => void
  ): UnaryResponse;
  getEndPointOperations(
    requestMessage: core_pb.GetEndPointOperationsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetEndPointOperationsResponse|null) => void
  ): UnaryResponse;
  listAllBaseStationLocations(
    requestMessage: core_pb.ListAllBaseStationLocationsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListAllBaseStationLocationsResponse|null) => void
  ): UnaryResponse;
  listAllBaseStationLocations(
    requestMessage: core_pb.ListAllBaseStationLocationsRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListAllBaseStationLocationsResponse|null) => void
  ): UnaryResponse;
  getCEStatus(
    requestMessage: core_pb.GetCEStatusRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetCEStatusResponse|null) => void
  ): UnaryResponse;
  getCEStatus(
    requestMessage: core_pb.GetCEStatusRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetCEStatusResponse|null) => void
  ): UnaryResponse;
  completeCEOnboarding(
    requestMessage: core_pb.CompleteCEOnboardingRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.CompleteCEOnboardingResponse|null) => void
  ): UnaryResponse;
  completeCEOnboarding(
    requestMessage: core_pb.CompleteCEOnboardingRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.CompleteCEOnboardingResponse|null) => void
  ): UnaryResponse;
  listCEInstances(
    requestMessage: core_pb.ListCEInstancesRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListCEInstancesResponse|null) => void
  ): UnaryResponse;
  listCEInstances(
    requestMessage: core_pb.ListCEInstancesRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.ListCEInstancesResponse|null) => void
  ): UnaryResponse;
  revokeCEInstance(
    requestMessage: core_pb.RevokeCEInstanceRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.RevokeCEInstanceResponse|null) => void
  ): UnaryResponse;
  revokeCEInstance(
    requestMessage: core_pb.RevokeCEInstanceRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.RevokeCEInstanceResponse|null) => void
  ): UnaryResponse;
}

