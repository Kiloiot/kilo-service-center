// package: kilocenter.api.v1
// file: core.proto

var core_pb = require("./core_pb");
var google_protobuf_empty_pb = require("google-protobuf/google/protobuf/empty_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var CoreService = (function () {
  function CoreService() {}
  CoreService.serviceName = "kilocenter.api.v1.CoreService";
  return CoreService;
}());

CoreService.CreateEndPoint = {
  methodName: "CreateEndPoint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateEndPointRequest,
  responseType: core_pb.EndPoint
};

CoreService.GetEndPoint = {
  methodName: "GetEndPoint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetEndPointRequest,
  responseType: core_pb.EndPoint
};

CoreService.UpdateEndPoint = {
  methodName: "UpdateEndPoint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateEndPointRequest,
  responseType: core_pb.EndPoint
};

CoreService.DeleteEndPoint = {
  methodName: "DeleteEndPoint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteEndPointRequest,
  responseType: google_protobuf_empty_pb.Empty
};

CoreService.ListEndPoints = {
  methodName: "ListEndPoints",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEndPointsRequest,
  responseType: core_pb.ListEndPointsResponse
};

CoreService.AttachEndPoint = {
  methodName: "AttachEndPoint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.AttachEndPointRequest,
  responseType: core_pb.AttachEndPointResponse
};

CoreService.DetachEndPoint = {
  methodName: "DetachEndPoint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DetachEndPointRequest,
  responseType: core_pb.DetachEndPointResponse
};

CoreService.CreateBaseStation = {
  methodName: "CreateBaseStation",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateBaseStationRequest,
  responseType: core_pb.BaseStation
};

CoreService.GetBaseStation = {
  methodName: "GetBaseStation",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationRequest,
  responseType: core_pb.BaseStation
};

CoreService.UpdateBaseStation = {
  methodName: "UpdateBaseStation",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateBaseStationRequest,
  responseType: core_pb.BaseStation
};

CoreService.DeleteBaseStation = {
  methodName: "DeleteBaseStation",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteBaseStationRequest,
  responseType: google_protobuf_empty_pb.Empty
};

CoreService.ListBaseStations = {
  methodName: "ListBaseStations",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBaseStationsRequest,
  responseType: core_pb.ListBaseStationsResponse
};

CoreService.GetBaseStationStats = {
  methodName: "GetBaseStationStats",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationStatsRequest,
  responseType: core_pb.GetBaseStationStatsResponse
};

CoreService.UpdateBaseStationEui = {
  methodName: "UpdateBaseStationEui",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateBaseStationEuiRequest,
  responseType: core_pb.BaseStation
};

CoreService.GetBaseStationAvailability = {
  methodName: "GetBaseStationAvailability",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationAvailabilityRequest,
  responseType: core_pb.GetBaseStationAvailabilityResponse
};

CoreService.GetBaseStationMessagesReceived = {
  methodName: "GetBaseStationMessagesReceived",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationMessagesReceivedRequest,
  responseType: core_pb.GetBaseStationMessagesReceivedResponse
};

CoreService.GetMessage = {
  methodName: "GetMessage",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetMessageRequest,
  responseType: core_pb.Message
};

CoreService.SendDownlink = {
  methodName: "SendDownlink",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SendDownlinkRequest,
  responseType: core_pb.SendDownlinkResponse
};

CoreService.RevokeDownlink = {
  methodName: "RevokeDownlink",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.RevokeDownlinkRequest,
  responseType: core_pb.RevokeDownlinkResponse
};

CoreService.ListDownlinkQueue = {
  methodName: "ListDownlinkQueue",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListDownlinkQueueRequest,
  responseType: core_pb.ListDownlinkQueueResponse
};

CoreService.GetDownlinkResults = {
  methodName: "GetDownlinkResults",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDownlinkResultsRequest,
  responseType: core_pb.GetDownlinkResultsResponse
};

CoreService.SendULTransmit = {
  methodName: "SendULTransmit",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SendULTransmitRequest,
  responseType: core_pb.SendULTransmitResponse
};

CoreService.RequestBaseStationStatus = {
  methodName: "RequestBaseStationStatus",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.BaseStationStatusRequest,
  responseType: core_pb.BaseStationStatusResponse
};

CoreService.InitiatePing = {
  methodName: "InitiatePing",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.InitiatePingRequest,
  responseType: core_pb.InitiatePingResponse
};

CoreService.GetDLRXStatus = {
  methodName: "GetDLRXStatus",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDLRXStatusRequest,
  responseType: core_pb.GetDLRXStatusResponse
};

CoreService.QueryDLRXStatus = {
  methodName: "QueryDLRXStatus",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.QueryDLRXStatusRequest,
  responseType: core_pb.QueryDLRXStatusResponse
};

CoreService.GetDLRXStatusQueries = {
  methodName: "GetDLRXStatusQueries",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDLRXStatusQueriesRequest,
  responseType: core_pb.GetDLRXStatusQueriesResponse
};

CoreService.GetSystemStatus = {
  methodName: "GetSystemStatus",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: google_protobuf_empty_pb.Empty,
  responseType: core_pb.SystemStatus
};

CoreService.GetStatistics = {
  methodName: "GetStatistics",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetStatisticsRequest,
  responseType: core_pb.Statistics
};

CoreService.GetReleaseInfo = {
  methodName: "GetReleaseInfo",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: google_protobuf_empty_pb.Empty,
  responseType: core_pb.ReleaseInfo
};

CoreService.CreateIntegration = {
  methodName: "CreateIntegration",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateIntegrationRequest,
  responseType: core_pb.Integration
};

CoreService.GetIntegration = {
  methodName: "GetIntegration",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetIntegrationRequest,
  responseType: core_pb.Integration
};

CoreService.UpdateIntegration = {
  methodName: "UpdateIntegration",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateIntegrationRequest,
  responseType: core_pb.Integration
};

CoreService.DeleteIntegration = {
  methodName: "DeleteIntegration",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteIntegrationRequest,
  responseType: google_protobuf_empty_pb.Empty
};

CoreService.ListIntegrations = {
  methodName: "ListIntegrations",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListIntegrationsRequest,
  responseType: core_pb.ListIntegrationsResponse
};

CoreService.GetAnalyticsOverview = {
  methodName: "GetAnalyticsOverview",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetAnalyticsOverviewRequest,
  responseType: core_pb.GetAnalyticsOverviewResponse
};

CoreService.GetActivityAnalytics = {
  methodName: "GetActivityAnalytics",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetActivityAnalyticsRequest,
  responseType: core_pb.GetActivityAnalyticsResponse
};

CoreService.GetSignalQualityAnalytics = {
  methodName: "GetSignalQualityAnalytics",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetSignalQualityAnalyticsRequest,
  responseType: core_pb.GetSignalQualityAnalyticsResponse
};

CoreService.ListEvents = {
  methodName: "ListEvents",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEventsRequest,
  responseType: core_pb.ListEventsResponse
};

CoreService.ListBaseStationActivity = {
  methodName: "ListBaseStationActivity",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBaseStationActivityRequest,
  responseType: core_pb.ListBaseStationActivityResponse
};

CoreService.ListEndpointActivity = {
  methodName: "ListEndpointActivity",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEndpointActivityRequest,
  responseType: core_pb.ListEndpointActivityResponse
};

CoreService.StreamEvents = {
  methodName: "StreamEvents",
  service: CoreService,
  requestStream: false,
  responseStream: true,
  requestType: core_pb.StreamEventsRequest,
  responseType: core_pb.Event
};

CoreService.ListAlerts = {
  methodName: "ListAlerts",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListAlertsRequest,
  responseType: core_pb.ListAlertsResponse
};

CoreService.GetAlertSummary = {
  methodName: "GetAlertSummary",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetAlertSummaryRequest,
  responseType: core_pb.GetAlertSummaryResponse
};

CoreService.ListScaciSessions = {
  methodName: "ListScaciSessions",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListScaciSessionsRequest,
  responseType: core_pb.ListScaciSessionsResponse
};

CoreService.GetScaciSession = {
  methodName: "GetScaciSession",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetScaciSessionRequest,
  responseType: core_pb.GetScaciSessionResponse
};

CoreService.GetScaciStatistics = {
  methodName: "GetScaciStatistics",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetScaciStatisticsRequest,
  responseType: core_pb.GetScaciStatisticsResponse
};

CoreService.ListScaciErrors = {
  methodName: "ListScaciErrors",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListScaciErrorsRequest,
  responseType: core_pb.ListScaciErrorsResponse
};

CoreService.ListScaciQueues = {
  methodName: "ListScaciQueues",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListScaciQueuesRequest,
  responseType: core_pb.ListScaciQueuesResponse
};

CoreService.GetScaciStatus = {
  methodName: "GetScaciStatus",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetScaciStatusRequest,
  responseType: core_pb.GetScaciStatusResponse
};

CoreService.GenerateCertificate = {
  methodName: "GenerateCertificate",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GenerateCertificateRequest,
  responseType: core_pb.GenerateCertificateResponse
};

CoreService.DownloadCertificate = {
  methodName: "DownloadCertificate",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DownloadCertificateRequest,
  responseType: core_pb.DownloadCertificateResponse
};

CoreService.DownloadBaseStationCertificate = {
  methodName: "DownloadBaseStationCertificate",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DownloadBaseStationCertificateRequest,
  responseType: core_pb.DownloadCertificateResponse
};

CoreService.GenerateServerCertificates = {
  methodName: "GenerateServerCertificates",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GenerateServerCertificatesRequest,
  responseType: core_pb.GenerateServerCertificatesResponse
};

CoreService.RenewServerCertificates = {
  methodName: "RenewServerCertificates",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.RenewServerCertificatesRequest,
  responseType: core_pb.RenewServerCertificatesResponse
};

CoreService.GetServerCertificateStatus = {
  methodName: "GetServerCertificateStatus",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetServerCertificateStatusRequest,
  responseType: core_pb.GetServerCertificateStatusResponse
};

CoreService.CreateManufacturer = {
  methodName: "CreateManufacturer",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateManufacturerRequest,
  responseType: core_pb.CreateManufacturerResponse
};

CoreService.GetManufacturer = {
  methodName: "GetManufacturer",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetManufacturerRequest,
  responseType: core_pb.GetManufacturerResponse
};

CoreService.UpdateManufacturer = {
  methodName: "UpdateManufacturer",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateManufacturerRequest,
  responseType: core_pb.UpdateManufacturerResponse
};

CoreService.DeleteManufacturer = {
  methodName: "DeleteManufacturer",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteManufacturerRequest,
  responseType: core_pb.DeleteManufacturerResponse
};

CoreService.ListManufacturers = {
  methodName: "ListManufacturers",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListManufacturersRequest,
  responseType: core_pb.ListManufacturersResponse
};

CoreService.CreateDeviceModel = {
  methodName: "CreateDeviceModel",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateDeviceModelRequest,
  responseType: core_pb.CreateDeviceModelResponse
};

CoreService.GetDeviceModel = {
  methodName: "GetDeviceModel",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetDeviceModelRequest,
  responseType: core_pb.GetDeviceModelResponse
};

CoreService.UpdateDeviceModel = {
  methodName: "UpdateDeviceModel",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateDeviceModelRequest,
  responseType: core_pb.UpdateDeviceModelResponse
};

CoreService.DeleteDeviceModel = {
  methodName: "DeleteDeviceModel",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteDeviceModelRequest,
  responseType: core_pb.DeleteDeviceModelResponse
};

CoreService.ListDeviceModels = {
  methodName: "ListDeviceModels",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListDeviceModelsRequest,
  responseType: core_pb.ListDeviceModelsResponse
};

CoreService.CreateBlueprint = {
  methodName: "CreateBlueprint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateBlueprintRequest,
  responseType: core_pb.CreateBlueprintResponse
};

CoreService.GetBlueprint = {
  methodName: "GetBlueprint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBlueprintRequest,
  responseType: core_pb.GetBlueprintResponse
};

CoreService.UpdateBlueprint = {
  methodName: "UpdateBlueprint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.UpdateBlueprintRequest,
  responseType: core_pb.UpdateBlueprintResponse
};

CoreService.DeleteBlueprint = {
  methodName: "DeleteBlueprint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DeleteBlueprintRequest,
  responseType: core_pb.DeleteBlueprintResponse
};

CoreService.ListBlueprints = {
  methodName: "ListBlueprints",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBlueprintsRequest,
  responseType: core_pb.ListBlueprintsResponse
};

CoreService.SetDefaultBlueprint = {
  methodName: "SetDefaultBlueprint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SetDefaultBlueprintRequest,
  responseType: core_pb.SetDefaultBlueprintResponse
};

CoreService.SubmitBlueprintToRegistry = {
  methodName: "SubmitBlueprintToRegistry",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SubmitBlueprintToRegistryRequest,
  responseType: core_pb.SubmitBlueprintToRegistryResponse
};

CoreService.CreateDeviceModelWithBlueprint = {
  methodName: "CreateDeviceModelWithBlueprint",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CreateDeviceModelWithBlueprintRequest,
  responseType: core_pb.CreateDeviceModelWithBlueprintResponse
};

CoreService.DecodePreview = {
  methodName: "DecodePreview",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.DecodePreviewRequest,
  responseType: core_pb.DecodePreviewResponse
};

CoreService.ListMessages = {
  methodName: "ListMessages",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListMessagesRequest,
  responseType: core_pb.ListMessagesResponse
};

CoreService.StreamMessages = {
  methodName: "StreamMessages",
  service: CoreService,
  requestStream: false,
  responseStream: true,
  requestType: core_pb.StreamMessagesRequest,
  responseType: core_pb.Message
};

CoreService.ListBaseStationMessages = {
  methodName: "ListBaseStationMessages",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListBaseStationMessagesRequest,
  responseType: core_pb.ListBaseStationMessagesResponse
};

CoreService.GetBaseStationMessage = {
  methodName: "GetBaseStationMessage",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationMessageRequest,
  responseType: core_pb.GetBaseStationMessageResponse
};

CoreService.GetBaseStationMessageStats = {
  methodName: "GetBaseStationMessageStats",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetBaseStationMessageStatsRequest,
  responseType: core_pb.GetBaseStationMessageStatsResponse
};

CoreService.SearchBaseStationMessages = {
  methodName: "SearchBaseStationMessages",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.SearchBaseStationMessagesRequest,
  responseType: core_pb.SearchBaseStationMessagesResponse
};

CoreService.ExportBaseStationMessages = {
  methodName: "ExportBaseStationMessages",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ExportBaseStationMessagesRequest,
  responseType: core_pb.ExportBaseStationMessagesResponse
};

CoreService.StreamBaseStationMessages = {
  methodName: "StreamBaseStationMessages",
  service: CoreService,
  requestStream: false,
  responseStream: true,
  requestType: core_pb.StreamBaseStationMessagesRequest,
  responseType: core_pb.BaseStationMessage
};

CoreService.ListEndpointMessages = {
  methodName: "ListEndpointMessages",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListEndpointMessagesRequest,
  responseType: core_pb.ListEndpointMessagesResponse
};

CoreService.GetEndPointStats = {
  methodName: "GetEndPointStats",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetEndPointStatsRequest,
  responseType: core_pb.GetEndPointStatsResponse
};

CoreService.GetEndPointOperations = {
  methodName: "GetEndPointOperations",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetEndPointOperationsRequest,
  responseType: core_pb.GetEndPointOperationsResponse
};

CoreService.ListAllBaseStationLocations = {
  methodName: "ListAllBaseStationLocations",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListAllBaseStationLocationsRequest,
  responseType: core_pb.ListAllBaseStationLocationsResponse
};

CoreService.GetCEStatus = {
  methodName: "GetCEStatus",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.GetCEStatusRequest,
  responseType: core_pb.GetCEStatusResponse
};

CoreService.CompleteCEOnboarding = {
  methodName: "CompleteCEOnboarding",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.CompleteCEOnboardingRequest,
  responseType: core_pb.CompleteCEOnboardingResponse
};

CoreService.ListCEInstances = {
  methodName: "ListCEInstances",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.ListCEInstancesRequest,
  responseType: core_pb.ListCEInstancesResponse
};

CoreService.RevokeCEInstance = {
  methodName: "RevokeCEInstance",
  service: CoreService,
  requestStream: false,
  responseStream: false,
  requestType: core_pb.RevokeCEInstanceRequest,
  responseType: core_pb.RevokeCEInstanceResponse
};

exports.CoreService = CoreService;

function CoreServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

CoreServiceClient.prototype.createEndPoint = function createEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CreateEndPoint, {
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

CoreServiceClient.prototype.getEndPoint = function getEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetEndPoint, {
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

CoreServiceClient.prototype.updateEndPoint = function updateEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.UpdateEndPoint, {
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

CoreServiceClient.prototype.deleteEndPoint = function deleteEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DeleteEndPoint, {
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

CoreServiceClient.prototype.listEndPoints = function listEndPoints(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListEndPoints, {
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

CoreServiceClient.prototype.attachEndPoint = function attachEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.AttachEndPoint, {
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

CoreServiceClient.prototype.detachEndPoint = function detachEndPoint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DetachEndPoint, {
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

CoreServiceClient.prototype.createBaseStation = function createBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CreateBaseStation, {
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

CoreServiceClient.prototype.getBaseStation = function getBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetBaseStation, {
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

CoreServiceClient.prototype.updateBaseStation = function updateBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.UpdateBaseStation, {
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

CoreServiceClient.prototype.deleteBaseStation = function deleteBaseStation(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DeleteBaseStation, {
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

CoreServiceClient.prototype.listBaseStations = function listBaseStations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListBaseStations, {
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

CoreServiceClient.prototype.getBaseStationStats = function getBaseStationStats(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetBaseStationStats, {
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

CoreServiceClient.prototype.updateBaseStationEui = function updateBaseStationEui(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.UpdateBaseStationEui, {
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

CoreServiceClient.prototype.getBaseStationAvailability = function getBaseStationAvailability(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetBaseStationAvailability, {
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

CoreServiceClient.prototype.getBaseStationMessagesReceived = function getBaseStationMessagesReceived(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetBaseStationMessagesReceived, {
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

CoreServiceClient.prototype.getMessage = function getMessage(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetMessage, {
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

CoreServiceClient.prototype.sendDownlink = function sendDownlink(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.SendDownlink, {
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

CoreServiceClient.prototype.revokeDownlink = function revokeDownlink(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.RevokeDownlink, {
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

CoreServiceClient.prototype.listDownlinkQueue = function listDownlinkQueue(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListDownlinkQueue, {
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

CoreServiceClient.prototype.getDownlinkResults = function getDownlinkResults(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetDownlinkResults, {
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

CoreServiceClient.prototype.sendULTransmit = function sendULTransmit(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.SendULTransmit, {
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

CoreServiceClient.prototype.requestBaseStationStatus = function requestBaseStationStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.RequestBaseStationStatus, {
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

CoreServiceClient.prototype.initiatePing = function initiatePing(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.InitiatePing, {
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

CoreServiceClient.prototype.getDLRXStatus = function getDLRXStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetDLRXStatus, {
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

CoreServiceClient.prototype.queryDLRXStatus = function queryDLRXStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.QueryDLRXStatus, {
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

CoreServiceClient.prototype.getDLRXStatusQueries = function getDLRXStatusQueries(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetDLRXStatusQueries, {
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

CoreServiceClient.prototype.getSystemStatus = function getSystemStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetSystemStatus, {
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

CoreServiceClient.prototype.getStatistics = function getStatistics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetStatistics, {
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

CoreServiceClient.prototype.getReleaseInfo = function getReleaseInfo(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetReleaseInfo, {
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

CoreServiceClient.prototype.createIntegration = function createIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CreateIntegration, {
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

CoreServiceClient.prototype.getIntegration = function getIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetIntegration, {
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

CoreServiceClient.prototype.updateIntegration = function updateIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.UpdateIntegration, {
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

CoreServiceClient.prototype.deleteIntegration = function deleteIntegration(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DeleteIntegration, {
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

CoreServiceClient.prototype.listIntegrations = function listIntegrations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListIntegrations, {
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

CoreServiceClient.prototype.getAnalyticsOverview = function getAnalyticsOverview(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetAnalyticsOverview, {
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

CoreServiceClient.prototype.getActivityAnalytics = function getActivityAnalytics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetActivityAnalytics, {
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

CoreServiceClient.prototype.getSignalQualityAnalytics = function getSignalQualityAnalytics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetSignalQualityAnalytics, {
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

CoreServiceClient.prototype.listEvents = function listEvents(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListEvents, {
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

CoreServiceClient.prototype.listBaseStationActivity = function listBaseStationActivity(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListBaseStationActivity, {
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

CoreServiceClient.prototype.listEndpointActivity = function listEndpointActivity(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListEndpointActivity, {
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

CoreServiceClient.prototype.streamEvents = function streamEvents(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(CoreService.StreamEvents, {
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

CoreServiceClient.prototype.listAlerts = function listAlerts(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListAlerts, {
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

CoreServiceClient.prototype.getAlertSummary = function getAlertSummary(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetAlertSummary, {
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

CoreServiceClient.prototype.listScaciSessions = function listScaciSessions(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListScaciSessions, {
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

CoreServiceClient.prototype.getScaciSession = function getScaciSession(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetScaciSession, {
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

CoreServiceClient.prototype.getScaciStatistics = function getScaciStatistics(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetScaciStatistics, {
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

CoreServiceClient.prototype.listScaciErrors = function listScaciErrors(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListScaciErrors, {
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

CoreServiceClient.prototype.listScaciQueues = function listScaciQueues(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListScaciQueues, {
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

CoreServiceClient.prototype.getScaciStatus = function getScaciStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetScaciStatus, {
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

CoreServiceClient.prototype.generateCertificate = function generateCertificate(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GenerateCertificate, {
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

CoreServiceClient.prototype.downloadCertificate = function downloadCertificate(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DownloadCertificate, {
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

CoreServiceClient.prototype.downloadBaseStationCertificate = function downloadBaseStationCertificate(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DownloadBaseStationCertificate, {
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

CoreServiceClient.prototype.generateServerCertificates = function generateServerCertificates(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GenerateServerCertificates, {
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

CoreServiceClient.prototype.renewServerCertificates = function renewServerCertificates(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.RenewServerCertificates, {
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

CoreServiceClient.prototype.getServerCertificateStatus = function getServerCertificateStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetServerCertificateStatus, {
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

CoreServiceClient.prototype.createManufacturer = function createManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CreateManufacturer, {
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

CoreServiceClient.prototype.getManufacturer = function getManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetManufacturer, {
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

CoreServiceClient.prototype.updateManufacturer = function updateManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.UpdateManufacturer, {
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

CoreServiceClient.prototype.deleteManufacturer = function deleteManufacturer(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DeleteManufacturer, {
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

CoreServiceClient.prototype.listManufacturers = function listManufacturers(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListManufacturers, {
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

CoreServiceClient.prototype.createDeviceModel = function createDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CreateDeviceModel, {
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

CoreServiceClient.prototype.getDeviceModel = function getDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetDeviceModel, {
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

CoreServiceClient.prototype.updateDeviceModel = function updateDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.UpdateDeviceModel, {
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

CoreServiceClient.prototype.deleteDeviceModel = function deleteDeviceModel(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DeleteDeviceModel, {
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

CoreServiceClient.prototype.listDeviceModels = function listDeviceModels(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListDeviceModels, {
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

CoreServiceClient.prototype.createBlueprint = function createBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CreateBlueprint, {
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

CoreServiceClient.prototype.getBlueprint = function getBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetBlueprint, {
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

CoreServiceClient.prototype.updateBlueprint = function updateBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.UpdateBlueprint, {
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

CoreServiceClient.prototype.deleteBlueprint = function deleteBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DeleteBlueprint, {
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

CoreServiceClient.prototype.listBlueprints = function listBlueprints(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListBlueprints, {
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

CoreServiceClient.prototype.setDefaultBlueprint = function setDefaultBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.SetDefaultBlueprint, {
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

CoreServiceClient.prototype.submitBlueprintToRegistry = function submitBlueprintToRegistry(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.SubmitBlueprintToRegistry, {
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

CoreServiceClient.prototype.createDeviceModelWithBlueprint = function createDeviceModelWithBlueprint(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CreateDeviceModelWithBlueprint, {
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

CoreServiceClient.prototype.decodePreview = function decodePreview(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.DecodePreview, {
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

CoreServiceClient.prototype.listMessages = function listMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListMessages, {
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

CoreServiceClient.prototype.streamMessages = function streamMessages(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(CoreService.StreamMessages, {
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

CoreServiceClient.prototype.listBaseStationMessages = function listBaseStationMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListBaseStationMessages, {
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

CoreServiceClient.prototype.getBaseStationMessage = function getBaseStationMessage(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetBaseStationMessage, {
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

CoreServiceClient.prototype.getBaseStationMessageStats = function getBaseStationMessageStats(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetBaseStationMessageStats, {
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

CoreServiceClient.prototype.searchBaseStationMessages = function searchBaseStationMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.SearchBaseStationMessages, {
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

CoreServiceClient.prototype.exportBaseStationMessages = function exportBaseStationMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ExportBaseStationMessages, {
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

CoreServiceClient.prototype.streamBaseStationMessages = function streamBaseStationMessages(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(CoreService.StreamBaseStationMessages, {
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

CoreServiceClient.prototype.listEndpointMessages = function listEndpointMessages(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListEndpointMessages, {
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

CoreServiceClient.prototype.getEndPointStats = function getEndPointStats(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetEndPointStats, {
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

CoreServiceClient.prototype.getEndPointOperations = function getEndPointOperations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetEndPointOperations, {
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

CoreServiceClient.prototype.listAllBaseStationLocations = function listAllBaseStationLocations(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListAllBaseStationLocations, {
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

CoreServiceClient.prototype.getCEStatus = function getCEStatus(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.GetCEStatus, {
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

CoreServiceClient.prototype.completeCEOnboarding = function completeCEOnboarding(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.CompleteCEOnboarding, {
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

CoreServiceClient.prototype.listCEInstances = function listCEInstances(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.ListCEInstances, {
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

CoreServiceClient.prototype.revokeCEInstance = function revokeCEInstance(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(CoreService.RevokeCEInstance, {
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

exports.CoreServiceClient = CoreServiceClient;

