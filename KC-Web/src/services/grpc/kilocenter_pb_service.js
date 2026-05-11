// package: kilocenter.api.v1
// file: kilocenter.proto

var kilocenter_pb = require("./kilocenter_pb");
var identity_pb = require("./identity_pb");
var core_pb = require("./core_pb");
var google_protobuf_empty_pb = require("google-protobuf/google/protobuf/empty_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var KiloCenterService = (function () {
  function KiloCenterService() {}
  KiloCenterService.serviceName = "kilocenter.api.v1.KiloCenterService";
  return KiloCenterService;
}());

KiloCenterService.CreateEndPoint = {
  methodName: "CreateEndPoint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateEndPointRequest,
  responseType: core_pb.EndPoint
};

KiloCenterService.GetEndPoint = {
  methodName: "GetEndPoint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetEndPointRequest,
  responseType: core_pb.EndPoint
};

KiloCenterService.UpdateEndPoint = {
  methodName: "UpdateEndPoint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateEndPointRequest,
  responseType: core_pb.EndPoint
};

KiloCenterService.DeleteEndPoint = {
  methodName: "DeleteEndPoint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteEndPointRequest,
  responseType: google_protobuf_empty_pb.Empty
};

KiloCenterService.ListEndPoints = {
  methodName: "ListEndPoints",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEndPointsRequest,
  responseType: core_pb.ListEndPointsResponse
};

KiloCenterService.AttachEndPoint = {
  methodName: "AttachEndPoint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.AttachEndPointRequest,
  responseType: core_pb.AttachEndPointResponse
};

KiloCenterService.DetachEndPoint = {
  methodName: "DetachEndPoint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DetachEndPointRequest,
  responseType: core_pb.DetachEndPointResponse
};

KiloCenterService.CreateBaseStation = {
  methodName: "CreateBaseStation",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateBaseStationRequest,
  responseType: core_pb.BaseStation
};

KiloCenterService.GetBaseStation = {
  methodName: "GetBaseStation",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationRequest,
  responseType: core_pb.BaseStation
};

KiloCenterService.UpdateBaseStation = {
  methodName: "UpdateBaseStation",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateBaseStationRequest,
  responseType: core_pb.BaseStation
};

KiloCenterService.DeleteBaseStation = {
  methodName: "DeleteBaseStation",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteBaseStationRequest,
  responseType: google_protobuf_empty_pb.Empty
};

KiloCenterService.ListBaseStations = {
  methodName: "ListBaseStations",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBaseStationsRequest,
  responseType: core_pb.ListBaseStationsResponse
};

KiloCenterService.GetBaseStationStats = {
  methodName: "GetBaseStationStats",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationStatsRequest,
  responseType: core_pb.GetBaseStationStatsResponse
};

KiloCenterService.UpdateBaseStationEui = {
  methodName: "UpdateBaseStationEui",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateBaseStationEuiRequest,
  responseType: core_pb.BaseStation
};

KiloCenterService.GetMessage = {
  methodName: "GetMessage",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetMessageRequest,
  responseType: core_pb.Message
};

KiloCenterService.SendDownlink = {
  methodName: "SendDownlink",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SendDownlinkRequest,
  responseType: core_pb.SendDownlinkResponse
};

KiloCenterService.RevokeDownlink = {
  methodName: "RevokeDownlink",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.RevokeDownlinkRequest,
  responseType: core_pb.RevokeDownlinkResponse
};

KiloCenterService.ListDownlinkQueue = {
  methodName: "ListDownlinkQueue",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListDownlinkQueueRequest,
  responseType: core_pb.ListDownlinkQueueResponse
};

KiloCenterService.GetDownlinkResults = {
  methodName: "GetDownlinkResults",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDownlinkResultsRequest,
  responseType: core_pb.GetDownlinkResultsResponse
};

KiloCenterService.SendULTransmit = {
  methodName: "SendULTransmit",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SendULTransmitRequest,
  responseType: core_pb.SendULTransmitResponse
};

KiloCenterService.RequestBaseStationStatus = {
  methodName: "RequestBaseStationStatus",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.BaseStationStatusRequest,
  responseType: core_pb.BaseStationStatusResponse
};

KiloCenterService.InitiatePing = {
  methodName: "InitiatePing",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.InitiatePingRequest,
  responseType: core_pb.InitiatePingResponse
};

KiloCenterService.GetDLRXStatus = {
  methodName: "GetDLRXStatus",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDLRXStatusRequest,
  responseType: core_pb.GetDLRXStatusResponse
};

KiloCenterService.QueryDLRXStatus = {
  methodName: "QueryDLRXStatus",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.QueryDLRXStatusRequest,
  responseType: core_pb.QueryDLRXStatusResponse
};

KiloCenterService.GetDLRXStatusQueries = {
  methodName: "GetDLRXStatusQueries",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDLRXStatusQueriesRequest,
  responseType: core_pb.GetDLRXStatusQueriesResponse
};

KiloCenterService.GetSystemStatus = {
  methodName: "GetSystemStatus",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: google_protobuf_empty_pb.Empty,
  responseType: core_pb.SystemStatus
};

KiloCenterService.GetStatistics = {
  methodName: "GetStatistics",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetStatisticsRequest,
  responseType: core_pb.Statistics
};

KiloCenterService.GetReleaseInfo = {
  methodName: "GetReleaseInfo",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: google_protobuf_empty_pb.Empty,
  responseType: core_pb.ReleaseInfo
};

KiloCenterService.Login = {
  methodName: "Login",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.LoginRequest,
  responseType: identity_pb.LoginResponse
};

KiloCenterService.RefreshTokens = {
  methodName: "RefreshTokens",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.RefreshTokensRequest,
  responseType: identity_pb.RefreshTokensResponse
};

KiloCenterService.GetProfile = {
  methodName: "GetProfile",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetProfileRequest,
  responseType: identity_pb.GetProfileResponse
};

KiloCenterService.GetAuthSettings = {
  methodName: "GetAuthSettings",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetAuthSettingsRequest,
  responseType: identity_pb.GetAuthSettingsResponse
};

KiloCenterService.Logout = {
  methodName: "Logout",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.LogoutRequest,
  responseType: identity_pb.LogoutResponse
};

KiloCenterService.ChangePassword = {
  methodName: "ChangePassword",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ChangePasswordRequest,
  responseType: identity_pb.ChangePasswordResponse
};

KiloCenterService.ExchangeOIDC = {
  methodName: "ExchangeOIDC",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ExchangeOIDCRequest,
  responseType: identity_pb.LoginResponse
};

KiloCenterService.ExchangeOAuth2 = {
  methodName: "ExchangeOAuth2",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ExchangeOAuth2Request,
  responseType: identity_pb.LoginResponse
};

KiloCenterService.RegisterAccount = {
  methodName: "RegisterAccount",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.RegisterAccountRequest,
  responseType: identity_pb.LoginResponse
};

KiloCenterService.CreateUser = {
  methodName: "CreateUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.CreateUserRequest,
  responseType: identity_pb.CreateUserResponse
};

KiloCenterService.GetUser = {
  methodName: "GetUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetUserRequest,
  responseType: identity_pb.GetUserResponse
};

KiloCenterService.UpdateUser = {
  methodName: "UpdateUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateUserRequest,
  responseType: identity_pb.UpdateUserResponse
};

KiloCenterService.DeleteUser = {
  methodName: "DeleteUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.DeleteUserRequest,
  responseType: identity_pb.DeleteUserResponse
};

KiloCenterService.ListUsers = {
  methodName: "ListUsers",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListUsersRequest,
  responseType: identity_pb.ListUsersResponse
};

KiloCenterService.UpdateUserPassword = {
  methodName: "UpdateUserPassword",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateUserPasswordRequest,
  responseType: identity_pb.UpdateUserPasswordResponse
};

KiloCenterService.CreateOrganization = {
  methodName: "CreateOrganization",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.CreateOrganizationRequest,
  responseType: identity_pb.CreateOrganizationResponse
};

KiloCenterService.GetOrganization = {
  methodName: "GetOrganization",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetOrganizationRequest,
  responseType: identity_pb.GetOrganizationResponse
};

KiloCenterService.UpdateOrganization = {
  methodName: "UpdateOrganization",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateOrganizationRequest,
  responseType: identity_pb.UpdateOrganizationResponse
};

KiloCenterService.DeleteOrganization = {
  methodName: "DeleteOrganization",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.DeleteOrganizationRequest,
  responseType: identity_pb.DeleteOrganizationResponse
};

KiloCenterService.ListOrganizations = {
  methodName: "ListOrganizations",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListOrganizationsRequest,
  responseType: identity_pb.ListOrganizationsResponse
};

KiloCenterService.AddOrganizationUser = {
  methodName: "AddOrganizationUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.AddOrganizationUserRequest,
  responseType: identity_pb.AddOrganizationUserResponse
};

KiloCenterService.GetOrganizationUser = {
  methodName: "GetOrganizationUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetOrganizationUserRequest,
  responseType: identity_pb.GetOrganizationUserResponse
};

KiloCenterService.UpdateOrganizationUser = {
  methodName: "UpdateOrganizationUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.UpdateOrganizationUserRequest,
  responseType: identity_pb.UpdateOrganizationUserResponse
};

KiloCenterService.RemoveOrganizationUser = {
  methodName: "RemoveOrganizationUser",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.RemoveOrganizationUserRequest,
  responseType: identity_pb.RemoveOrganizationUserResponse
};

KiloCenterService.ListOrganizationUsers = {
  methodName: "ListOrganizationUsers",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListOrganizationUsersRequest,
  responseType: identity_pb.ListOrganizationUsersResponse
};

KiloCenterService.ListUserOrganizations = {
  methodName: "ListUserOrganizations",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListUserOrganizationsRequest,
  responseType: identity_pb.ListUserOrganizationsResponse
};

KiloCenterService.CreateApiKey = {
  methodName: "CreateApiKey",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.CreateApiKeyRequest,
  responseType: identity_pb.CreateApiKeyResponse
};

KiloCenterService.GetApiKey = {
  methodName: "GetApiKey",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.GetApiKeyRequest,
  responseType: identity_pb.GetApiKeyResponse
};

KiloCenterService.DeleteApiKey = {
  methodName: "DeleteApiKey",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.DeleteApiKeyRequest,
  responseType: identity_pb.DeleteApiKeyResponse
};

KiloCenterService.ListApiKeys = {
  methodName: "ListApiKeys",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: identity_pb.ListApiKeysRequest,
  responseType: identity_pb.ListApiKeysResponse
};

KiloCenterService.CreateIntegration = {
  methodName: "CreateIntegration",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateIntegrationRequest,
  responseType: core_pb.Integration
};

KiloCenterService.GetIntegration = {
  methodName: "GetIntegration",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetIntegrationRequest,
  responseType: core_pb.Integration
};

KiloCenterService.UpdateIntegration = {
  methodName: "UpdateIntegration",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateIntegrationRequest,
  responseType: core_pb.Integration
};

KiloCenterService.DeleteIntegration = {
  methodName: "DeleteIntegration",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteIntegrationRequest,
  responseType: google_protobuf_empty_pb.Empty
};

KiloCenterService.ListIntegrations = {
  methodName: "ListIntegrations",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListIntegrationsRequest,
  responseType: core_pb.ListIntegrationsResponse
};

KiloCenterService.GetAnalyticsOverview = {
  methodName: "GetAnalyticsOverview",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetAnalyticsOverviewRequest,
  responseType: core_pb.GetAnalyticsOverviewResponse
};

KiloCenterService.GetActivityAnalytics = {
  methodName: "GetActivityAnalytics",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetActivityAnalyticsRequest,
  responseType: core_pb.GetActivityAnalyticsResponse
};

KiloCenterService.GetSignalQualityAnalytics = {
  methodName: "GetSignalQualityAnalytics",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetSignalQualityAnalyticsRequest,
  responseType: core_pb.GetSignalQualityAnalyticsResponse
};

KiloCenterService.ListEvents = {
  methodName: "ListEvents",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEventsRequest,
  responseType: core_pb.ListEventsResponse
};

KiloCenterService.ListBaseStationActivity = {
  methodName: "ListBaseStationActivity",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBaseStationActivityRequest,
  responseType: core_pb.ListBaseStationActivityResponse
};

KiloCenterService.ListEndpointActivity = {
  methodName: "ListEndpointActivity",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEndpointActivityRequest,
  responseType: core_pb.ListEndpointActivityResponse
};

KiloCenterService.StreamEvents = {
  methodName: "StreamEvents",
  service: KiloCenterService,
  requestStream: false,
  responseStream: true,
  requestType: core_pb.StreamEventsRequest,
  responseType: core_pb.Event
};

KiloCenterService.ListAlerts = {
  methodName: "ListAlerts",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListAlertsRequest,
  responseType: core_pb.ListAlertsResponse
};

KiloCenterService.GetAlertSummary = {
  methodName: "GetAlertSummary",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetAlertSummaryRequest,
  responseType: core_pb.GetAlertSummaryResponse
};

KiloCenterService.ListScaciSessions = {
  methodName: "ListScaciSessions",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListScaciSessionsRequest,
  responseType: core_pb.ListScaciSessionsResponse
};

KiloCenterService.GetScaciSession = {
  methodName: "GetScaciSession",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetScaciSessionRequest,
  responseType: core_pb.GetScaciSessionResponse
};

KiloCenterService.GetScaciStatistics = {
  methodName: "GetScaciStatistics",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetScaciStatisticsRequest,
  responseType: core_pb.GetScaciStatisticsResponse
};

KiloCenterService.ListScaciErrors = {
  methodName: "ListScaciErrors",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListScaciErrorsRequest,
  responseType: core_pb.ListScaciErrorsResponse
};

KiloCenterService.ListScaciQueues = {
  methodName: "ListScaciQueues",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListScaciQueuesRequest,
  responseType: core_pb.ListScaciQueuesResponse
};

KiloCenterService.GetScaciStatus = {
  methodName: "GetScaciStatus",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetScaciStatusRequest,
  responseType: core_pb.GetScaciStatusResponse
};

KiloCenterService.GenerateCertificate = {
  methodName: "GenerateCertificate",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GenerateCertificateRequest,
  responseType: core_pb.GenerateCertificateResponse
};

KiloCenterService.DownloadCertificate = {
  methodName: "DownloadCertificate",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DownloadCertificateRequest,
  responseType: core_pb.DownloadCertificateResponse
};

KiloCenterService.DownloadBaseStationCertificate = {
  methodName: "DownloadBaseStationCertificate",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DownloadBaseStationCertificateRequest,
  responseType: core_pb.DownloadCertificateResponse
};

KiloCenterService.GenerateServerCertificates = {
  methodName: "GenerateServerCertificates",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GenerateServerCertificatesRequest,
  responseType: core_pb.GenerateServerCertificatesResponse
};

KiloCenterService.RenewServerCertificates = {
  methodName: "RenewServerCertificates",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.RenewServerCertificatesRequest,
  responseType: core_pb.RenewServerCertificatesResponse
};

KiloCenterService.GetServerCertificateStatus = {
  methodName: "GetServerCertificateStatus",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetServerCertificateStatusRequest,
  responseType: core_pb.GetServerCertificateStatusResponse
};

KiloCenterService.CreateManufacturer = {
  methodName: "CreateManufacturer",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateManufacturerRequest,
  responseType: core_pb.CreateManufacturerResponse
};

KiloCenterService.GetManufacturer = {
  methodName: "GetManufacturer",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetManufacturerRequest,
  responseType: core_pb.GetManufacturerResponse
};

KiloCenterService.UpdateManufacturer = {
  methodName: "UpdateManufacturer",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateManufacturerRequest,
  responseType: core_pb.UpdateManufacturerResponse
};

KiloCenterService.DeleteManufacturer = {
  methodName: "DeleteManufacturer",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteManufacturerRequest,
  responseType: core_pb.DeleteManufacturerResponse
};

KiloCenterService.ListManufacturers = {
  methodName: "ListManufacturers",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListManufacturersRequest,
  responseType: core_pb.ListManufacturersResponse
};

KiloCenterService.CreateDeviceModel = {
  methodName: "CreateDeviceModel",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateDeviceModelRequest,
  responseType: core_pb.CreateDeviceModelResponse
};

KiloCenterService.GetDeviceModel = {
  methodName: "GetDeviceModel",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDeviceModelRequest,
  responseType: core_pb.GetDeviceModelResponse
};

KiloCenterService.UpdateDeviceModel = {
  methodName: "UpdateDeviceModel",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateDeviceModelRequest,
  responseType: core_pb.UpdateDeviceModelResponse
};

KiloCenterService.DeleteDeviceModel = {
  methodName: "DeleteDeviceModel",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteDeviceModelRequest,
  responseType: core_pb.DeleteDeviceModelResponse
};

KiloCenterService.ListDeviceModels = {
  methodName: "ListDeviceModels",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListDeviceModelsRequest,
  responseType: core_pb.ListDeviceModelsResponse
};

KiloCenterService.CreateBlueprint = {
  methodName: "CreateBlueprint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateBlueprintRequest,
  responseType: core_pb.CreateBlueprintResponse
};

KiloCenterService.GetBlueprint = {
  methodName: "GetBlueprint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBlueprintRequest,
  responseType: core_pb.GetBlueprintResponse
};

KiloCenterService.UpdateBlueprint = {
  methodName: "UpdateBlueprint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateBlueprintRequest,
  responseType: core_pb.UpdateBlueprintResponse
};

KiloCenterService.DeleteBlueprint = {
  methodName: "DeleteBlueprint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteBlueprintRequest,
  responseType: core_pb.DeleteBlueprintResponse
};

KiloCenterService.ListBlueprints = {
  methodName: "ListBlueprints",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBlueprintsRequest,
  responseType: core_pb.ListBlueprintsResponse
};

KiloCenterService.SetDefaultBlueprint = {
  methodName: "SetDefaultBlueprint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SetDefaultBlueprintRequest,
  responseType: core_pb.SetDefaultBlueprintResponse
};

KiloCenterService.SubmitBlueprintToRegistry = {
  methodName: "SubmitBlueprintToRegistry",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SubmitBlueprintToRegistryRequest,
  responseType: core_pb.SubmitBlueprintToRegistryResponse
};

KiloCenterService.CreateDeviceModelWithBlueprint = {
  methodName: "CreateDeviceModelWithBlueprint",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateDeviceModelWithBlueprintRequest,
  responseType: core_pb.CreateDeviceModelWithBlueprintResponse
};

KiloCenterService.DecodePreview = {
  methodName: "DecodePreview",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DecodePreviewRequest,
  responseType: core_pb.DecodePreviewResponse
};

KiloCenterService.ListMessages = {
  methodName: "ListMessages",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListMessagesRequest,
  responseType: core_pb.ListMessagesResponse
};

KiloCenterService.StreamMessages = {
  methodName: "StreamMessages",
  service: KiloCenterService,
  requestStream: false,
  responseStream: true,
  requestType: core_pb.StreamMessagesRequest,
  responseType: core_pb.Message
};

KiloCenterService.ListBaseStationMessages = {
  methodName: "ListBaseStationMessages",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBaseStationMessagesRequest,
  responseType: core_pb.ListBaseStationMessagesResponse
};

KiloCenterService.GetBaseStationMessage = {
  methodName: "GetBaseStationMessage",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationMessageRequest,
  responseType: core_pb.GetBaseStationMessageResponse
};

KiloCenterService.GetBaseStationMessageStats = {
  methodName: "GetBaseStationMessageStats",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationMessageStatsRequest,
  responseType: core_pb.GetBaseStationMessageStatsResponse
};

KiloCenterService.SearchBaseStationMessages = {
  methodName: "SearchBaseStationMessages",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SearchBaseStationMessagesRequest,
  responseType: core_pb.SearchBaseStationMessagesResponse
};

KiloCenterService.ExportBaseStationMessages = {
  methodName: "ExportBaseStationMessages",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ExportBaseStationMessagesRequest,
  responseType: core_pb.ExportBaseStationMessagesResponse
};

KiloCenterService.StreamBaseStationMessages = {
  methodName: "StreamBaseStationMessages",
  service: KiloCenterService,
  requestStream: false,
  responseStream: true,
  requestType: core_pb.StreamBaseStationMessagesRequest,
  responseType: core_pb.BaseStationMessage
};

KiloCenterService.ListEndpointMessages = {
  methodName: "ListEndpointMessages",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEndpointMessagesRequest,
  responseType: core_pb.ListEndpointMessagesResponse
};

KiloCenterService.GetEndPointStats = {
  methodName: "GetEndPointStats",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetEndPointStatsRequest,
  responseType: core_pb.GetEndPointStatsResponse
};

KiloCenterService.GetEndPointOperations = {
  methodName: "GetEndPointOperations",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetEndPointOperationsRequest,
  responseType: core_pb.GetEndPointOperationsResponse
};

KiloCenterService.ListAllBaseStationLocations = {
  methodName: "ListAllBaseStationLocations",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListAllBaseStationLocationsRequest,
  responseType: core_pb.ListAllBaseStationLocationsResponse
};

KiloCenterService.GetCEStatus = {
  methodName: "GetCEStatus",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetCEStatusRequest,
  responseType: core_pb.GetCEStatusResponse
};

KiloCenterService.CompleteCEOnboarding = {
  methodName: "CompleteCEOnboarding",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CompleteCEOnboardingRequest,
  responseType: core_pb.CompleteCEOnboardingResponse
};

KiloCenterService.ListCEInstances = {
  methodName: "ListCEInstances",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListCEInstancesRequest,
  responseType: core_pb.ListCEInstancesResponse
};

KiloCenterService.RevokeCEInstance = {
  methodName: "RevokeCEInstance",
  service: KiloCenterService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.RevokeCEInstanceRequest,
  responseType: core_pb.RevokeCEInstanceResponse
};

exports.KiloCenterService = KiloCenterService;

function KiloCenterServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

KiloCenterServiceClient.prototype.createEndPoint = function createEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateEndPoint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getEndPoint = function getEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetEndPoint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateEndPoint = function updateEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateEndPoint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteEndPoint = function deleteEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteEndPoint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listEndPoints = function listEndPoints(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListEndPoints, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.attachEndPoint = function attachEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.AttachEndPoint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.detachEndPoint = function detachEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DetachEndPoint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createBaseStation = function createBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateBaseStation, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getBaseStation = function getBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetBaseStation, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateBaseStation = function updateBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateBaseStation, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteBaseStation = function deleteBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteBaseStation, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listBaseStations = function listBaseStations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListBaseStations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getBaseStationStats = function getBaseStationStats(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetBaseStationStats, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateBaseStationEui = function updateBaseStationEui(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateBaseStationEui, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getMessage = function getMessage(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetMessage, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.sendDownlink = function sendDownlink(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.SendDownlink, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.revokeDownlink = function revokeDownlink(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.RevokeDownlink, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listDownlinkQueue = function listDownlinkQueue(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListDownlinkQueue, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getDownlinkResults = function getDownlinkResults(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetDownlinkResults, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.sendULTransmit = function sendULTransmit(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.SendULTransmit, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.requestBaseStationStatus = function requestBaseStationStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.RequestBaseStationStatus, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.initiatePing = function initiatePing(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.InitiatePing, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getDLRXStatus = function getDLRXStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetDLRXStatus, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.queryDLRXStatus = function queryDLRXStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.QueryDLRXStatus, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getDLRXStatusQueries = function getDLRXStatusQueries(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetDLRXStatusQueries, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getSystemStatus = function getSystemStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetSystemStatus, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getStatistics = function getStatistics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetStatistics, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getReleaseInfo = function getReleaseInfo(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetReleaseInfo, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.login = function login(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.Login, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.refreshTokens = function refreshTokens(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.RefreshTokens, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getProfile = function getProfile(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetProfile, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getAuthSettings = function getAuthSettings(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetAuthSettings, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.logout = function logout(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.Logout, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.changePassword = function changePassword(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ChangePassword, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.exchangeOIDC = function exchangeOIDC(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ExchangeOIDC, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.exchangeOAuth2 = function exchangeOAuth2(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ExchangeOAuth2, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.registerAccount = function registerAccount(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.RegisterAccount, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createUser = function createUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getUser = function getUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateUser = function updateUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteUser = function deleteUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listUsers = function listUsers(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListUsers, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateUserPassword = function updateUserPassword(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateUserPassword, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createOrganization = function createOrganization(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateOrganization, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getOrganization = function getOrganization(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetOrganization, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateOrganization = function updateOrganization(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateOrganization, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteOrganization = function deleteOrganization(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteOrganization, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listOrganizations = function listOrganizations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListOrganizations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.addOrganizationUser = function addOrganizationUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.AddOrganizationUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getOrganizationUser = function getOrganizationUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetOrganizationUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateOrganizationUser = function updateOrganizationUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateOrganizationUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.removeOrganizationUser = function removeOrganizationUser(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.RemoveOrganizationUser, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listOrganizationUsers = function listOrganizationUsers(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListOrganizationUsers, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listUserOrganizations = function listUserOrganizations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListUserOrganizations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createApiKey = function createApiKey(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateApiKey, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getApiKey = function getApiKey(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetApiKey, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteApiKey = function deleteApiKey(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteApiKey, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listApiKeys = function listApiKeys(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListApiKeys, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createIntegration = function createIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateIntegration, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getIntegration = function getIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetIntegration, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateIntegration = function updateIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateIntegration, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteIntegration = function deleteIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteIntegration, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listIntegrations = function listIntegrations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListIntegrations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getAnalyticsOverview = function getAnalyticsOverview(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetAnalyticsOverview, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getActivityAnalytics = function getActivityAnalytics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetActivityAnalytics, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getSignalQualityAnalytics = function getSignalQualityAnalytics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetSignalQualityAnalytics, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listEvents = function listEvents(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListEvents, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listBaseStationActivity = function listBaseStationActivity(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListBaseStationActivity, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listEndpointActivity = function listEndpointActivity(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListEndpointActivity, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.streamEvents = function streamEvents(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(KiloCenterService.StreamEvents, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onMessage: function (responseMessage) {
      listeners.data.forEach(function (handler) {
        handler(responseMessage);
      });
    },
    onEnd: function (status, statusMessage, trailers) {
      listeners.status.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners.end.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners = null;
    }
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    cancel: function () {
      listeners = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listAlerts = function listAlerts(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListAlerts, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getAlertSummary = function getAlertSummary(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetAlertSummary, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listScaciSessions = function listScaciSessions(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListScaciSessions, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getScaciSession = function getScaciSession(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetScaciSession, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getScaciStatistics = function getScaciStatistics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetScaciStatistics, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listScaciErrors = function listScaciErrors(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListScaciErrors, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listScaciQueues = function listScaciQueues(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListScaciQueues, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getScaciStatus = function getScaciStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetScaciStatus, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.generateCertificate = function generateCertificate(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GenerateCertificate, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.downloadCertificate = function downloadCertificate(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DownloadCertificate, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.downloadBaseStationCertificate = function downloadBaseStationCertificate(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DownloadBaseStationCertificate, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.generateServerCertificates = function generateServerCertificates(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GenerateServerCertificates, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.renewServerCertificates = function renewServerCertificates(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.RenewServerCertificates, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getServerCertificateStatus = function getServerCertificateStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetServerCertificateStatus, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createManufacturer = function createManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateManufacturer, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getManufacturer = function getManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetManufacturer, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateManufacturer = function updateManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateManufacturer, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteManufacturer = function deleteManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteManufacturer, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listManufacturers = function listManufacturers(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListManufacturers, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createDeviceModel = function createDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateDeviceModel, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getDeviceModel = function getDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetDeviceModel, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateDeviceModel = function updateDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateDeviceModel, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteDeviceModel = function deleteDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteDeviceModel, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listDeviceModels = function listDeviceModels(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListDeviceModels, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createBlueprint = function createBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateBlueprint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getBlueprint = function getBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetBlueprint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.updateBlueprint = function updateBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.UpdateBlueprint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.deleteBlueprint = function deleteBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DeleteBlueprint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listBlueprints = function listBlueprints(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListBlueprints, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.setDefaultBlueprint = function setDefaultBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.SetDefaultBlueprint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.submitBlueprintToRegistry = function submitBlueprintToRegistry(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.SubmitBlueprintToRegistry, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.createDeviceModelWithBlueprint = function createDeviceModelWithBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CreateDeviceModelWithBlueprint, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.decodePreview = function decodePreview(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.DecodePreview, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listMessages = function listMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.streamMessages = function streamMessages(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(KiloCenterService.StreamMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onMessage: function (responseMessage) {
      listeners.data.forEach(function (handler) {
        handler(responseMessage);
      });
    },
    onEnd: function (status, statusMessage, trailers) {
      listeners.status.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners.end.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners = null;
    }
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    cancel: function () {
      listeners = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listBaseStationMessages = function listBaseStationMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListBaseStationMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getBaseStationMessage = function getBaseStationMessage(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetBaseStationMessage, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getBaseStationMessageStats = function getBaseStationMessageStats(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetBaseStationMessageStats, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.searchBaseStationMessages = function searchBaseStationMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.SearchBaseStationMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.exportBaseStationMessages = function exportBaseStationMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ExportBaseStationMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.streamBaseStationMessages = function streamBaseStationMessages(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(KiloCenterService.StreamBaseStationMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onMessage: function (responseMessage) {
      listeners.data.forEach(function (handler) {
        handler(responseMessage);
      });
    },
    onEnd: function (status, statusMessage, trailers) {
      listeners.status.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners.end.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners = null;
    }
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    cancel: function () {
      listeners = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listEndpointMessages = function listEndpointMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListEndpointMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getEndPointStats = function getEndPointStats(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetEndPointStats, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getEndPointOperations = function getEndPointOperations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetEndPointOperations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listAllBaseStationLocations = function listAllBaseStationLocations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListAllBaseStationLocations, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.getCEStatus = function getCEStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.GetCEStatus, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.completeCEOnboarding = function completeCEOnboarding(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.CompleteCEOnboarding, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.listCEInstances = function listCEInstances(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.ListCEInstances, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

KiloCenterServiceClient.prototype.revokeCEInstance = function revokeCEInstance(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(KiloCenterService.RevokeCEInstance, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

exports.KiloCenterServiceClient = KiloCenterServiceClient;

