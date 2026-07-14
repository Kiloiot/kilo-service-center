// package: kilocenter.api.v1
// file: kilocenter.proto

import * as kilocenter_pb from "./kilocenter_pb";
import * as identity_pb from "./identity_pb";
import * as core_pb from "./core_pb";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import {grpc} from "@improbable-eng/grpc-web";

type KiloCenterServiceCreateEndPoint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateEndPointRequest;
  readonly responseType: typeof core_pb.EndPoint;
};

type KiloCenterServiceGetEndPoint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetEndPointRequest;
  readonly responseType: typeof core_pb.EndPoint;
};

type KiloCenterServiceUpdateEndPoint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateEndPointRequest;
  readonly responseType: typeof core_pb.EndPoint;
};

type KiloCenterServiceDeleteEndPoint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteEndPointRequest;
  readonly responseType: typeof google_protobuf_empty_pb.Empty;
};

type KiloCenterServiceListEndPoints = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEndPointsRequest;
  readonly responseType: typeof core_pb.ListEndPointsResponse;
};

type KiloCenterServiceAttachEndPoint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.AttachEndPointRequest;
  readonly responseType: typeof core_pb.AttachEndPointResponse;
};

type KiloCenterServiceDetachEndPoint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DetachEndPointRequest;
  readonly responseType: typeof core_pb.DetachEndPointResponse;
};

type KiloCenterServiceCreateBaseStation = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateBaseStationRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type KiloCenterServiceGetBaseStation = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type KiloCenterServiceUpdateBaseStation = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateBaseStationRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type KiloCenterServiceDeleteBaseStation = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteBaseStationRequest;
  readonly responseType: typeof google_protobuf_empty_pb.Empty;
};

type KiloCenterServiceListBaseStations = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBaseStationsRequest;
  readonly responseType: typeof core_pb.ListBaseStationsResponse;
};

type KiloCenterServiceGetBaseStationStats = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationStatsRequest;
  readonly responseType: typeof core_pb.GetBaseStationStatsResponse;
};

type KiloCenterServiceUpdateBaseStationEui = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateBaseStationEuiRequest;
  readonly responseType: typeof core_pb.BaseStation;
};

type KiloCenterServiceGetBaseStationAvailability = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationAvailabilityRequest;
  readonly responseType: typeof core_pb.GetBaseStationAvailabilityResponse;
};

type KiloCenterServiceGetBaseStationMessagesReceived = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationMessagesReceivedRequest;
  readonly responseType: typeof core_pb.GetBaseStationMessagesReceivedResponse;
};

type KiloCenterServiceGetMessage = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetMessageRequest;
  readonly responseType: typeof core_pb.Message;
};

type KiloCenterServiceSendDownlink = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SendDownlinkRequest;
  readonly responseType: typeof core_pb.SendDownlinkResponse;
};

type KiloCenterServiceRevokeDownlink = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.RevokeDownlinkRequest;
  readonly responseType: typeof core_pb.RevokeDownlinkResponse;
};

type KiloCenterServiceListDownlinkQueue = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListDownlinkQueueRequest;
  readonly responseType: typeof core_pb.ListDownlinkQueueResponse;
};

type KiloCenterServiceGetDownlinkResults = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDownlinkResultsRequest;
  readonly responseType: typeof core_pb.GetDownlinkResultsResponse;
};

type KiloCenterServiceSendULTransmit = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SendULTransmitRequest;
  readonly responseType: typeof core_pb.SendULTransmitResponse;
};

type KiloCenterServiceRequestBaseStationStatus = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.BaseStationStatusRequest;
  readonly responseType: typeof core_pb.BaseStationStatusResponse;
};

type KiloCenterServiceInitiatePing = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.InitiatePingRequest;
  readonly responseType: typeof core_pb.InitiatePingResponse;
};

type KiloCenterServiceGetDLRXStatus = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDLRXStatusRequest;
  readonly responseType: typeof core_pb.GetDLRXStatusResponse;
};

type KiloCenterServiceQueryDLRXStatus = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.QueryDLRXStatusRequest;
  readonly responseType: typeof core_pb.QueryDLRXStatusResponse;
};

type KiloCenterServiceGetDLRXStatusQueries = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDLRXStatusQueriesRequest;
  readonly responseType: typeof core_pb.GetDLRXStatusQueriesResponse;
};

type KiloCenterServiceGetSystemStatus = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof google_protobuf_empty_pb.Empty;
  readonly responseType: typeof core_pb.SystemStatus;
};

type KiloCenterServiceGetStatistics = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetStatisticsRequest;
  readonly responseType: typeof core_pb.Statistics;
};

type KiloCenterServiceGetReleaseInfo = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof google_protobuf_empty_pb.Empty;
  readonly responseType: typeof core_pb.ReleaseInfo;
};

type KiloCenterServiceLogin = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.LoginRequest;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type KiloCenterServiceRefreshTokens = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.RefreshTokensRequest;
  readonly responseType: typeof identity_pb.RefreshTokensResponse;
};

type KiloCenterServiceGetProfile = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetProfileRequest;
  readonly responseType: typeof identity_pb.GetProfileResponse;
};

type KiloCenterServiceGetAuthSettings = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetAuthSettingsRequest;
  readonly responseType: typeof identity_pb.GetAuthSettingsResponse;
};

type KiloCenterServiceLogout = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.LogoutRequest;
  readonly responseType: typeof identity_pb.LogoutResponse;
};

type KiloCenterServiceChangePassword = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ChangePasswordRequest;
  readonly responseType: typeof identity_pb.ChangePasswordResponse;
};

type KiloCenterServiceExchangeOIDC = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ExchangeOIDCRequest;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type KiloCenterServiceExchangeOAuth2 = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ExchangeOAuth2Request;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type KiloCenterServiceRegisterAccount = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.RegisterAccountRequest;
  readonly responseType: typeof identity_pb.LoginResponse;
};

type KiloCenterServiceCreateUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.CreateUserRequest;
  readonly responseType: typeof identity_pb.CreateUserResponse;
};

type KiloCenterServiceGetUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetUserRequest;
  readonly responseType: typeof identity_pb.GetUserResponse;
};

type KiloCenterServiceUpdateUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateUserRequest;
  readonly responseType: typeof identity_pb.UpdateUserResponse;
};

type KiloCenterServiceDeleteUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.DeleteUserRequest;
  readonly responseType: typeof identity_pb.DeleteUserResponse;
};

type KiloCenterServiceListUsers = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListUsersRequest;
  readonly responseType: typeof identity_pb.ListUsersResponse;
};

type KiloCenterServiceUpdateUserPassword = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateUserPasswordRequest;
  readonly responseType: typeof identity_pb.UpdateUserPasswordResponse;
};

type KiloCenterServiceCreateOrganization = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.CreateOrganizationRequest;
  readonly responseType: typeof identity_pb.CreateOrganizationResponse;
};

type KiloCenterServiceGetOrganization = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetOrganizationRequest;
  readonly responseType: typeof identity_pb.GetOrganizationResponse;
};

type KiloCenterServiceUpdateOrganization = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateOrganizationRequest;
  readonly responseType: typeof identity_pb.UpdateOrganizationResponse;
};

type KiloCenterServiceDeleteOrganization = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.DeleteOrganizationRequest;
  readonly responseType: typeof identity_pb.DeleteOrganizationResponse;
};

type KiloCenterServiceListOrganizations = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListOrganizationsRequest;
  readonly responseType: typeof identity_pb.ListOrganizationsResponse;
};

type KiloCenterServiceAddOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.AddOrganizationUserRequest;
  readonly responseType: typeof identity_pb.AddOrganizationUserResponse;
};

type KiloCenterServiceGetOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetOrganizationUserRequest;
  readonly responseType: typeof identity_pb.GetOrganizationUserResponse;
};

type KiloCenterServiceUpdateOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.UpdateOrganizationUserRequest;
  readonly responseType: typeof identity_pb.UpdateOrganizationUserResponse;
};

type KiloCenterServiceRemoveOrganizationUser = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.RemoveOrganizationUserRequest;
  readonly responseType: typeof identity_pb.RemoveOrganizationUserResponse;
};

type KiloCenterServiceListOrganizationUsers = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListOrganizationUsersRequest;
  readonly responseType: typeof identity_pb.ListOrganizationUsersResponse;
};

type KiloCenterServiceListUserOrganizations = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListUserOrganizationsRequest;
  readonly responseType: typeof identity_pb.ListUserOrganizationsResponse;
};

type KiloCenterServiceCreateApiKey = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.CreateApiKeyRequest;
  readonly responseType: typeof identity_pb.CreateApiKeyResponse;
};

type KiloCenterServiceGetApiKey = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.GetApiKeyRequest;
  readonly responseType: typeof identity_pb.GetApiKeyResponse;
};

type KiloCenterServiceDeleteApiKey = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.DeleteApiKeyRequest;
  readonly responseType: typeof identity_pb.DeleteApiKeyResponse;
};

type KiloCenterServiceListApiKeys = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof identity_pb.ListApiKeysRequest;
  readonly responseType: typeof identity_pb.ListApiKeysResponse;
};

type KiloCenterServiceCreateIntegration = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateIntegrationRequest;
  readonly responseType: typeof core_pb.Integration;
};

type KiloCenterServiceGetIntegration = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetIntegrationRequest;
  readonly responseType: typeof core_pb.Integration;
};

type KiloCenterServiceUpdateIntegration = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateIntegrationRequest;
  readonly responseType: typeof core_pb.Integration;
};

type KiloCenterServiceDeleteIntegration = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteIntegrationRequest;
  readonly responseType: typeof google_protobuf_empty_pb.Empty;
};

type KiloCenterServiceListIntegrations = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListIntegrationsRequest;
  readonly responseType: typeof core_pb.ListIntegrationsResponse;
};

type KiloCenterServiceGetAnalyticsOverview = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetAnalyticsOverviewRequest;
  readonly responseType: typeof core_pb.GetAnalyticsOverviewResponse;
};

type KiloCenterServiceGetActivityAnalytics = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetActivityAnalyticsRequest;
  readonly responseType: typeof core_pb.GetActivityAnalyticsResponse;
};

type KiloCenterServiceGetSignalQualityAnalytics = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetSignalQualityAnalyticsRequest;
  readonly responseType: typeof core_pb.GetSignalQualityAnalyticsResponse;
};

type KiloCenterServiceListEvents = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEventsRequest;
  readonly responseType: typeof core_pb.ListEventsResponse;
};

type KiloCenterServiceListBaseStationActivity = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBaseStationActivityRequest;
  readonly responseType: typeof core_pb.ListBaseStationActivityResponse;
};

type KiloCenterServiceListEndpointActivity = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEndpointActivityRequest;
  readonly responseType: typeof core_pb.ListEndpointActivityResponse;
};

type KiloCenterServiceStreamEvents = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof core_pb.StreamEventsRequest;
  readonly responseType: typeof core_pb.Event;
};

type KiloCenterServiceListAlerts = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListAlertsRequest;
  readonly responseType: typeof core_pb.ListAlertsResponse;
};

type KiloCenterServiceGetAlertSummary = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetAlertSummaryRequest;
  readonly responseType: typeof core_pb.GetAlertSummaryResponse;
};

type KiloCenterServiceListScaciSessions = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListScaciSessionsRequest;
  readonly responseType: typeof core_pb.ListScaciSessionsResponse;
};

type KiloCenterServiceGetScaciSession = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetScaciSessionRequest;
  readonly responseType: typeof core_pb.GetScaciSessionResponse;
};

type KiloCenterServiceGetScaciStatistics = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetScaciStatisticsRequest;
  readonly responseType: typeof core_pb.GetScaciStatisticsResponse;
};

type KiloCenterServiceListScaciErrors = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListScaciErrorsRequest;
  readonly responseType: typeof core_pb.ListScaciErrorsResponse;
};

type KiloCenterServiceListScaciQueues = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListScaciQueuesRequest;
  readonly responseType: typeof core_pb.ListScaciQueuesResponse;
};

type KiloCenterServiceGetScaciStatus = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetScaciStatusRequest;
  readonly responseType: typeof core_pb.GetScaciStatusResponse;
};

type KiloCenterServiceGenerateCertificate = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GenerateCertificateRequest;
  readonly responseType: typeof core_pb.GenerateCertificateResponse;
};

type KiloCenterServiceDownloadCertificate = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DownloadCertificateRequest;
  readonly responseType: typeof core_pb.DownloadCertificateResponse;
};

type KiloCenterServiceDownloadBaseStationCertificate = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DownloadBaseStationCertificateRequest;
  readonly responseType: typeof core_pb.DownloadCertificateResponse;
};

type KiloCenterServiceGenerateServerCertificates = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GenerateServerCertificatesRequest;
  readonly responseType: typeof core_pb.GenerateServerCertificatesResponse;
};

type KiloCenterServiceRenewServerCertificates = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.RenewServerCertificatesRequest;
  readonly responseType: typeof core_pb.RenewServerCertificatesResponse;
};

type KiloCenterServiceGetServerCertificateStatus = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetServerCertificateStatusRequest;
  readonly responseType: typeof core_pb.GetServerCertificateStatusResponse;
};

type KiloCenterServiceCreateManufacturer = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateManufacturerRequest;
  readonly responseType: typeof core_pb.CreateManufacturerResponse;
};

type KiloCenterServiceGetManufacturer = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetManufacturerRequest;
  readonly responseType: typeof core_pb.GetManufacturerResponse;
};

type KiloCenterServiceUpdateManufacturer = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateManufacturerRequest;
  readonly responseType: typeof core_pb.UpdateManufacturerResponse;
};

type KiloCenterServiceDeleteManufacturer = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteManufacturerRequest;
  readonly responseType: typeof core_pb.DeleteManufacturerResponse;
};

type KiloCenterServiceListManufacturers = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListManufacturersRequest;
  readonly responseType: typeof core_pb.ListManufacturersResponse;
};

type KiloCenterServiceCreateDeviceModel = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateDeviceModelRequest;
  readonly responseType: typeof core_pb.CreateDeviceModelResponse;
};

type KiloCenterServiceGetDeviceModel = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetDeviceModelRequest;
  readonly responseType: typeof core_pb.GetDeviceModelResponse;
};

type KiloCenterServiceUpdateDeviceModel = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateDeviceModelRequest;
  readonly responseType: typeof core_pb.UpdateDeviceModelResponse;
};

type KiloCenterServiceDeleteDeviceModel = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteDeviceModelRequest;
  readonly responseType: typeof core_pb.DeleteDeviceModelResponse;
};

type KiloCenterServiceListDeviceModels = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListDeviceModelsRequest;
  readonly responseType: typeof core_pb.ListDeviceModelsResponse;
};

type KiloCenterServiceCreateBlueprint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateBlueprintRequest;
  readonly responseType: typeof core_pb.CreateBlueprintResponse;
};

type KiloCenterServiceGetBlueprint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBlueprintRequest;
  readonly responseType: typeof core_pb.GetBlueprintResponse;
};

type KiloCenterServiceUpdateBlueprint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.UpdateBlueprintRequest;
  readonly responseType: typeof core_pb.UpdateBlueprintResponse;
};

type KiloCenterServiceDeleteBlueprint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DeleteBlueprintRequest;
  readonly responseType: typeof core_pb.DeleteBlueprintResponse;
};

type KiloCenterServiceListBlueprints = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBlueprintsRequest;
  readonly responseType: typeof core_pb.ListBlueprintsResponse;
};

type KiloCenterServiceSetDefaultBlueprint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SetDefaultBlueprintRequest;
  readonly responseType: typeof core_pb.SetDefaultBlueprintResponse;
};

type KiloCenterServiceSubmitBlueprintToRegistry = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SubmitBlueprintToRegistryRequest;
  readonly responseType: typeof core_pb.SubmitBlueprintToRegistryResponse;
};

type KiloCenterServiceBulkAssignBlueprint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.BulkAssignBlueprintRequest;
  readonly responseType: typeof core_pb.BulkAssignBlueprintResponse;
};

type KiloCenterServiceCreateDeviceModelWithBlueprint = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CreateDeviceModelWithBlueprintRequest;
  readonly responseType: typeof core_pb.CreateDeviceModelWithBlueprintResponse;
};

type KiloCenterServiceDecodePreview = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.DecodePreviewRequest;
  readonly responseType: typeof core_pb.DecodePreviewResponse;
};

type KiloCenterServiceListMessages = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListMessagesRequest;
  readonly responseType: typeof core_pb.ListMessagesResponse;
};

type KiloCenterServiceStreamMessages = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof core_pb.StreamMessagesRequest;
  readonly responseType: typeof core_pb.Message;
};

type KiloCenterServiceListBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.ListBaseStationMessagesResponse;
};

type KiloCenterServiceGetBaseStationMessage = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationMessageRequest;
  readonly responseType: typeof core_pb.GetBaseStationMessageResponse;
};

type KiloCenterServiceGetBaseStationMessageStats = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetBaseStationMessageStatsRequest;
  readonly responseType: typeof core_pb.GetBaseStationMessageStatsResponse;
};

type KiloCenterServiceSearchBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.SearchBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.SearchBaseStationMessagesResponse;
};

type KiloCenterServiceExportBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ExportBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.ExportBaseStationMessagesResponse;
};

type KiloCenterServiceStreamBaseStationMessages = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof core_pb.StreamBaseStationMessagesRequest;
  readonly responseType: typeof core_pb.BaseStationMessage;
};

type KiloCenterServiceListEndpointMessages = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListEndpointMessagesRequest;
  readonly responseType: typeof core_pb.ListEndpointMessagesResponse;
};

type KiloCenterServiceGetEndPointStats = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetEndPointStatsRequest;
  readonly responseType: typeof core_pb.GetEndPointStatsResponse;
};

type KiloCenterServiceGetEndPointOperations = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetEndPointOperationsRequest;
  readonly responseType: typeof core_pb.GetEndPointOperationsResponse;
};

type KiloCenterServiceListAllBaseStationLocations = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListAllBaseStationLocationsRequest;
  readonly responseType: typeof core_pb.ListAllBaseStationLocationsResponse;
};

type KiloCenterServiceGetCEStatus = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.GetCEStatusRequest;
  readonly responseType: typeof core_pb.GetCEStatusResponse;
};

type KiloCenterServiceCompleteCEOnboarding = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.CompleteCEOnboardingRequest;
  readonly responseType: typeof core_pb.CompleteCEOnboardingResponse;
};

type KiloCenterServiceListCEInstances = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.ListCEInstancesRequest;
  readonly responseType: typeof core_pb.ListCEInstancesResponse;
};

type KiloCenterServiceRevokeCEInstance = {
  readonly methodName: string;
  readonly service: typeof KiloCenterService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof core_pb.RevokeCEInstanceRequest;
  readonly responseType: typeof core_pb.RevokeCEInstanceResponse;
};

export class KiloCenterService {
  static readonly serviceName: string;
  static readonly CreateEndPoint: KiloCenterServiceCreateEndPoint;
  static readonly GetEndPoint: KiloCenterServiceGetEndPoint;
  static readonly UpdateEndPoint: KiloCenterServiceUpdateEndPoint;
  static readonly DeleteEndPoint: KiloCenterServiceDeleteEndPoint;
  static readonly ListEndPoints: KiloCenterServiceListEndPoints;
  static readonly AttachEndPoint: KiloCenterServiceAttachEndPoint;
  static readonly DetachEndPoint: KiloCenterServiceDetachEndPoint;
  static readonly CreateBaseStation: KiloCenterServiceCreateBaseStation;
  static readonly GetBaseStation: KiloCenterServiceGetBaseStation;
  static readonly UpdateBaseStation: KiloCenterServiceUpdateBaseStation;
  static readonly DeleteBaseStation: KiloCenterServiceDeleteBaseStation;
  static readonly ListBaseStations: KiloCenterServiceListBaseStations;
  static readonly GetBaseStationStats: KiloCenterServiceGetBaseStationStats;
  static readonly UpdateBaseStationEui: KiloCenterServiceUpdateBaseStationEui;
  static readonly GetBaseStationAvailability: KiloCenterServiceGetBaseStationAvailability;
  static readonly GetBaseStationMessagesReceived: KiloCenterServiceGetBaseStationMessagesReceived;
  static readonly GetMessage: KiloCenterServiceGetMessage;
  static readonly SendDownlink: KiloCenterServiceSendDownlink;
  static readonly RevokeDownlink: KiloCenterServiceRevokeDownlink;
  static readonly ListDownlinkQueue: KiloCenterServiceListDownlinkQueue;
  static readonly GetDownlinkResults: KiloCenterServiceGetDownlinkResults;
  static readonly SendULTransmit: KiloCenterServiceSendULTransmit;
  static readonly RequestBaseStationStatus: KiloCenterServiceRequestBaseStationStatus;
  static readonly InitiatePing: KiloCenterServiceInitiatePing;
  static readonly GetDLRXStatus: KiloCenterServiceGetDLRXStatus;
  static readonly QueryDLRXStatus: KiloCenterServiceQueryDLRXStatus;
  static readonly GetDLRXStatusQueries: KiloCenterServiceGetDLRXStatusQueries;
  static readonly GetSystemStatus: KiloCenterServiceGetSystemStatus;
  static readonly GetStatistics: KiloCenterServiceGetStatistics;
  static readonly GetReleaseInfo: KiloCenterServiceGetReleaseInfo;
  static readonly Login: KiloCenterServiceLogin;
  static readonly RefreshTokens: KiloCenterServiceRefreshTokens;
  static readonly GetProfile: KiloCenterServiceGetProfile;
  static readonly GetAuthSettings: KiloCenterServiceGetAuthSettings;
  static readonly Logout: KiloCenterServiceLogout;
  static readonly ChangePassword: KiloCenterServiceChangePassword;
  static readonly ExchangeOIDC: KiloCenterServiceExchangeOIDC;
  static readonly ExchangeOAuth2: KiloCenterServiceExchangeOAuth2;
  static readonly RegisterAccount: KiloCenterServiceRegisterAccount;
  static readonly CreateUser: KiloCenterServiceCreateUser;
  static readonly GetUser: KiloCenterServiceGetUser;
  static readonly UpdateUser: KiloCenterServiceUpdateUser;
  static readonly DeleteUser: KiloCenterServiceDeleteUser;
  static readonly ListUsers: KiloCenterServiceListUsers;
  static readonly UpdateUserPassword: KiloCenterServiceUpdateUserPassword;
  static readonly CreateOrganization: KiloCenterServiceCreateOrganization;
  static readonly GetOrganization: KiloCenterServiceGetOrganization;
  static readonly UpdateOrganization: KiloCenterServiceUpdateOrganization;
  static readonly DeleteOrganization: KiloCenterServiceDeleteOrganization;
  static readonly ListOrganizations: KiloCenterServiceListOrganizations;
  static readonly AddOrganizationUser: KiloCenterServiceAddOrganizationUser;
  static readonly GetOrganizationUser: KiloCenterServiceGetOrganizationUser;
  static readonly UpdateOrganizationUser: KiloCenterServiceUpdateOrganizationUser;
  static readonly RemoveOrganizationUser: KiloCenterServiceRemoveOrganizationUser;
  static readonly ListOrganizationUsers: KiloCenterServiceListOrganizationUsers;
  static readonly ListUserOrganizations: KiloCenterServiceListUserOrganizations;
  static readonly CreateApiKey: KiloCenterServiceCreateApiKey;
  static readonly GetApiKey: KiloCenterServiceGetApiKey;
  static readonly DeleteApiKey: KiloCenterServiceDeleteApiKey;
  static readonly ListApiKeys: KiloCenterServiceListApiKeys;
  static readonly CreateIntegration: KiloCenterServiceCreateIntegration;
  static readonly GetIntegration: KiloCenterServiceGetIntegration;
  static readonly UpdateIntegration: KiloCenterServiceUpdateIntegration;
  static readonly DeleteIntegration: KiloCenterServiceDeleteIntegration;
  static readonly ListIntegrations: KiloCenterServiceListIntegrations;
  static readonly GetAnalyticsOverview: KiloCenterServiceGetAnalyticsOverview;
  static readonly GetActivityAnalytics: KiloCenterServiceGetActivityAnalytics;
  static readonly GetSignalQualityAnalytics: KiloCenterServiceGetSignalQualityAnalytics;
  static readonly ListEvents: KiloCenterServiceListEvents;
  static readonly ListBaseStationActivity: KiloCenterServiceListBaseStationActivity;
  static readonly ListEndpointActivity: KiloCenterServiceListEndpointActivity;
  static readonly StreamEvents: KiloCenterServiceStreamEvents;
  static readonly ListAlerts: KiloCenterServiceListAlerts;
  static readonly GetAlertSummary: KiloCenterServiceGetAlertSummary;
  static readonly ListScaciSessions: KiloCenterServiceListScaciSessions;
  static readonly GetScaciSession: KiloCenterServiceGetScaciSession;
  static readonly GetScaciStatistics: KiloCenterServiceGetScaciStatistics;
  static readonly ListScaciErrors: KiloCenterServiceListScaciErrors;
  static readonly ListScaciQueues: KiloCenterServiceListScaciQueues;
  static readonly GetScaciStatus: KiloCenterServiceGetScaciStatus;
  static readonly GenerateCertificate: KiloCenterServiceGenerateCertificate;
  static readonly DownloadCertificate: KiloCenterServiceDownloadCertificate;
  static readonly DownloadBaseStationCertificate: KiloCenterServiceDownloadBaseStationCertificate;
  static readonly GenerateServerCertificates: KiloCenterServiceGenerateServerCertificates;
  static readonly RenewServerCertificates: KiloCenterServiceRenewServerCertificates;
  static readonly GetServerCertificateStatus: KiloCenterServiceGetServerCertificateStatus;
  static readonly CreateManufacturer: KiloCenterServiceCreateManufacturer;
  static readonly GetManufacturer: KiloCenterServiceGetManufacturer;
  static readonly UpdateManufacturer: KiloCenterServiceUpdateManufacturer;
  static readonly DeleteManufacturer: KiloCenterServiceDeleteManufacturer;
  static readonly ListManufacturers: KiloCenterServiceListManufacturers;
  static readonly CreateDeviceModel: KiloCenterServiceCreateDeviceModel;
  static readonly GetDeviceModel: KiloCenterServiceGetDeviceModel;
  static readonly UpdateDeviceModel: KiloCenterServiceUpdateDeviceModel;
  static readonly DeleteDeviceModel: KiloCenterServiceDeleteDeviceModel;
  static readonly ListDeviceModels: KiloCenterServiceListDeviceModels;
  static readonly CreateBlueprint: KiloCenterServiceCreateBlueprint;
  static readonly GetBlueprint: KiloCenterServiceGetBlueprint;
  static readonly UpdateBlueprint: KiloCenterServiceUpdateBlueprint;
  static readonly DeleteBlueprint: KiloCenterServiceDeleteBlueprint;
  static readonly ListBlueprints: KiloCenterServiceListBlueprints;
  static readonly SetDefaultBlueprint: KiloCenterServiceSetDefaultBlueprint;
  static readonly SubmitBlueprintToRegistry: KiloCenterServiceSubmitBlueprintToRegistry;
  static readonly BulkAssignBlueprint: KiloCenterServiceBulkAssignBlueprint;
  static readonly CreateDeviceModelWithBlueprint: KiloCenterServiceCreateDeviceModelWithBlueprint;
  static readonly DecodePreview: KiloCenterServiceDecodePreview;
  static readonly ListMessages: KiloCenterServiceListMessages;
  static readonly StreamMessages: KiloCenterServiceStreamMessages;
  static readonly ListBaseStationMessages: KiloCenterServiceListBaseStationMessages;
  static readonly GetBaseStationMessage: KiloCenterServiceGetBaseStationMessage;
  static readonly GetBaseStationMessageStats: KiloCenterServiceGetBaseStationMessageStats;
  static readonly SearchBaseStationMessages: KiloCenterServiceSearchBaseStationMessages;
  static readonly ExportBaseStationMessages: KiloCenterServiceExportBaseStationMessages;
  static readonly StreamBaseStationMessages: KiloCenterServiceStreamBaseStationMessages;
  static readonly ListEndpointMessages: KiloCenterServiceListEndpointMessages;
  static readonly GetEndPointStats: KiloCenterServiceGetEndPointStats;
  static readonly GetEndPointOperations: KiloCenterServiceGetEndPointOperations;
  static readonly ListAllBaseStationLocations: KiloCenterServiceListAllBaseStationLocations;
  static readonly GetCEStatus: KiloCenterServiceGetCEStatus;
  static readonly CompleteCEOnboarding: KiloCenterServiceCompleteCEOnboarding;
  static readonly ListCEInstances: KiloCenterServiceListCEInstances;
  static readonly RevokeCEInstance: KiloCenterServiceRevokeCEInstance;
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

export class KiloCenterServiceClient {
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
  getBaseStationAvailability(
    requestMessage: core_pb.GetBaseStationAvailabilityRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationAvailabilityResponse|null) => void
  ): UnaryResponse;
  getBaseStationAvailability(
    requestMessage: core_pb.GetBaseStationAvailabilityRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationAvailabilityResponse|null) => void
  ): UnaryResponse;
  getBaseStationMessagesReceived(
    requestMessage: core_pb.GetBaseStationMessagesReceivedRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationMessagesReceivedResponse|null) => void
  ): UnaryResponse;
  getBaseStationMessagesReceived(
    requestMessage: core_pb.GetBaseStationMessagesReceivedRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.GetBaseStationMessagesReceivedResponse|null) => void
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
  login(
    requestMessage: identity_pb.LoginRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  login(
    requestMessage: identity_pb.LoginRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  refreshTokens(
    requestMessage: identity_pb.RefreshTokensRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.RefreshTokensResponse|null) => void
  ): UnaryResponse;
  refreshTokens(
    requestMessage: identity_pb.RefreshTokensRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.RefreshTokensResponse|null) => void
  ): UnaryResponse;
  getProfile(
    requestMessage: identity_pb.GetProfileRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetProfileResponse|null) => void
  ): UnaryResponse;
  getProfile(
    requestMessage: identity_pb.GetProfileRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetProfileResponse|null) => void
  ): UnaryResponse;
  getAuthSettings(
    requestMessage: identity_pb.GetAuthSettingsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetAuthSettingsResponse|null) => void
  ): UnaryResponse;
  getAuthSettings(
    requestMessage: identity_pb.GetAuthSettingsRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetAuthSettingsResponse|null) => void
  ): UnaryResponse;
  logout(
    requestMessage: identity_pb.LogoutRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LogoutResponse|null) => void
  ): UnaryResponse;
  logout(
    requestMessage: identity_pb.LogoutRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LogoutResponse|null) => void
  ): UnaryResponse;
  changePassword(
    requestMessage: identity_pb.ChangePasswordRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ChangePasswordResponse|null) => void
  ): UnaryResponse;
  changePassword(
    requestMessage: identity_pb.ChangePasswordRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ChangePasswordResponse|null) => void
  ): UnaryResponse;
  exchangeOIDC(
    requestMessage: identity_pb.ExchangeOIDCRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  exchangeOIDC(
    requestMessage: identity_pb.ExchangeOIDCRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  exchangeOAuth2(
    requestMessage: identity_pb.ExchangeOAuth2Request,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  exchangeOAuth2(
    requestMessage: identity_pb.ExchangeOAuth2Request,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  registerAccount(
    requestMessage: identity_pb.RegisterAccountRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  registerAccount(
    requestMessage: identity_pb.RegisterAccountRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.LoginResponse|null) => void
  ): UnaryResponse;
  createUser(
    requestMessage: identity_pb.CreateUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.CreateUserResponse|null) => void
  ): UnaryResponse;
  createUser(
    requestMessage: identity_pb.CreateUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.CreateUserResponse|null) => void
  ): UnaryResponse;
  getUser(
    requestMessage: identity_pb.GetUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetUserResponse|null) => void
  ): UnaryResponse;
  getUser(
    requestMessage: identity_pb.GetUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetUserResponse|null) => void
  ): UnaryResponse;
  updateUser(
    requestMessage: identity_pb.UpdateUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateUserResponse|null) => void
  ): UnaryResponse;
  updateUser(
    requestMessage: identity_pb.UpdateUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateUserResponse|null) => void
  ): UnaryResponse;
  deleteUser(
    requestMessage: identity_pb.DeleteUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.DeleteUserResponse|null) => void
  ): UnaryResponse;
  deleteUser(
    requestMessage: identity_pb.DeleteUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.DeleteUserResponse|null) => void
  ): UnaryResponse;
  listUsers(
    requestMessage: identity_pb.ListUsersRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListUsersResponse|null) => void
  ): UnaryResponse;
  listUsers(
    requestMessage: identity_pb.ListUsersRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListUsersResponse|null) => void
  ): UnaryResponse;
  updateUserPassword(
    requestMessage: identity_pb.UpdateUserPasswordRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateUserPasswordResponse|null) => void
  ): UnaryResponse;
  updateUserPassword(
    requestMessage: identity_pb.UpdateUserPasswordRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateUserPasswordResponse|null) => void
  ): UnaryResponse;
  createOrganization(
    requestMessage: identity_pb.CreateOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.CreateOrganizationResponse|null) => void
  ): UnaryResponse;
  createOrganization(
    requestMessage: identity_pb.CreateOrganizationRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.CreateOrganizationResponse|null) => void
  ): UnaryResponse;
  getOrganization(
    requestMessage: identity_pb.GetOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetOrganizationResponse|null) => void
  ): UnaryResponse;
  getOrganization(
    requestMessage: identity_pb.GetOrganizationRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetOrganizationResponse|null) => void
  ): UnaryResponse;
  updateOrganization(
    requestMessage: identity_pb.UpdateOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateOrganizationResponse|null) => void
  ): UnaryResponse;
  updateOrganization(
    requestMessage: identity_pb.UpdateOrganizationRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateOrganizationResponse|null) => void
  ): UnaryResponse;
  deleteOrganization(
    requestMessage: identity_pb.DeleteOrganizationRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.DeleteOrganizationResponse|null) => void
  ): UnaryResponse;
  deleteOrganization(
    requestMessage: identity_pb.DeleteOrganizationRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.DeleteOrganizationResponse|null) => void
  ): UnaryResponse;
  listOrganizations(
    requestMessage: identity_pb.ListOrganizationsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListOrganizationsResponse|null) => void
  ): UnaryResponse;
  listOrganizations(
    requestMessage: identity_pb.ListOrganizationsRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListOrganizationsResponse|null) => void
  ): UnaryResponse;
  addOrganizationUser(
    requestMessage: identity_pb.AddOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.AddOrganizationUserResponse|null) => void
  ): UnaryResponse;
  addOrganizationUser(
    requestMessage: identity_pb.AddOrganizationUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.AddOrganizationUserResponse|null) => void
  ): UnaryResponse;
  getOrganizationUser(
    requestMessage: identity_pb.GetOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetOrganizationUserResponse|null) => void
  ): UnaryResponse;
  getOrganizationUser(
    requestMessage: identity_pb.GetOrganizationUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetOrganizationUserResponse|null) => void
  ): UnaryResponse;
  updateOrganizationUser(
    requestMessage: identity_pb.UpdateOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateOrganizationUserResponse|null) => void
  ): UnaryResponse;
  updateOrganizationUser(
    requestMessage: identity_pb.UpdateOrganizationUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.UpdateOrganizationUserResponse|null) => void
  ): UnaryResponse;
  removeOrganizationUser(
    requestMessage: identity_pb.RemoveOrganizationUserRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.RemoveOrganizationUserResponse|null) => void
  ): UnaryResponse;
  removeOrganizationUser(
    requestMessage: identity_pb.RemoveOrganizationUserRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.RemoveOrganizationUserResponse|null) => void
  ): UnaryResponse;
  listOrganizationUsers(
    requestMessage: identity_pb.ListOrganizationUsersRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListOrganizationUsersResponse|null) => void
  ): UnaryResponse;
  listOrganizationUsers(
    requestMessage: identity_pb.ListOrganizationUsersRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListOrganizationUsersResponse|null) => void
  ): UnaryResponse;
  listUserOrganizations(
    requestMessage: identity_pb.ListUserOrganizationsRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListUserOrganizationsResponse|null) => void
  ): UnaryResponse;
  listUserOrganizations(
    requestMessage: identity_pb.ListUserOrganizationsRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListUserOrganizationsResponse|null) => void
  ): UnaryResponse;
  createApiKey(
    requestMessage: identity_pb.CreateApiKeyRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.CreateApiKeyResponse|null) => void
  ): UnaryResponse;
  createApiKey(
    requestMessage: identity_pb.CreateApiKeyRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.CreateApiKeyResponse|null) => void
  ): UnaryResponse;
  getApiKey(
    requestMessage: identity_pb.GetApiKeyRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetApiKeyResponse|null) => void
  ): UnaryResponse;
  getApiKey(
    requestMessage: identity_pb.GetApiKeyRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.GetApiKeyResponse|null) => void
  ): UnaryResponse;
  deleteApiKey(
    requestMessage: identity_pb.DeleteApiKeyRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.DeleteApiKeyResponse|null) => void
  ): UnaryResponse;
  deleteApiKey(
    requestMessage: identity_pb.DeleteApiKeyRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.DeleteApiKeyResponse|null) => void
  ): UnaryResponse;
  listApiKeys(
    requestMessage: identity_pb.ListApiKeysRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListApiKeysResponse|null) => void
  ): UnaryResponse;
  listApiKeys(
    requestMessage: identity_pb.ListApiKeysRequest,
    callback: (error: ServiceError|null, responseMessage: identity_pb.ListApiKeysResponse|null) => void
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
  bulkAssignBlueprint(
    requestMessage: core_pb.BulkAssignBlueprintRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: core_pb.BulkAssignBlueprintResponse|null) => void
  ): UnaryResponse;
  bulkAssignBlueprint(
    requestMessage: core_pb.BulkAssignBlueprintRequest,
    callback: (error: ServiceError|null, responseMessage: core_pb.BulkAssignBlueprintResponse|null) => void
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

