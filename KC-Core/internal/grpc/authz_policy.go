package grpc

import (
	"google.golang.org/grpc"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
)

// coreMethodRequirements is the role policy of every CoreService RPC. Public
// RPCs are admitted before this table is consulted.
var coreMethodRequirements = map[string]authz.Requirement{
	// Endpoints, their traffic, downlinks and key material.
	pb.CoreService_CreateEndPoint_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_GetEndPoint_FullMethodName:                    authz.EndpointManager,
	pb.CoreService_UpdateEndPoint_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_DeleteEndPoint_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_ListEndPoints_FullMethodName:                  authz.EndpointManager,
	pb.CoreService_AttachEndPoint_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_DetachEndPoint_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_GetEndPointStats_FullMethodName:               authz.EndpointManager,
	pb.CoreService_GetEndPointOperations_FullMethodName:          authz.EndpointManager,
	pb.CoreService_ListEndpointMessages_FullMethodName:           authz.EndpointManager,
	pb.CoreService_ListEndpointActivity_FullMethodName:           authz.EndpointManager,
	pb.CoreService_GetMessage_FullMethodName:                     authz.EndpointManager,
	pb.CoreService_ListMessages_FullMethodName:                   authz.EndpointManager,
	pb.CoreService_StreamMessages_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_SendDownlink_FullMethodName:                   authz.EndpointManager,
	pb.CoreService_RevokeDownlink_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_ListDownlinkQueue_FullMethodName:              authz.EndpointManager,
	pb.CoreService_GetDownlinkResults_FullMethodName:             authz.EndpointManager,
	pb.CoreService_UpdatePendingDownlink_FullMethodName:          authz.EndpointManager,
	pb.CoreService_SendULTransmit_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_GetDLRXStatus_FullMethodName:                  authz.EndpointManager,
	pb.CoreService_QueryDLRXStatus_FullMethodName:                authz.EndpointManager,
	pb.CoreService_GetDLRXStatusQueries_FullMethodName:           authz.EndpointManager,
	pb.CoreService_ListScaciSessions_FullMethodName:              authz.EndpointManager,
	pb.CoreService_GetScaciSession_FullMethodName:                authz.EndpointManager,
	pb.CoreService_GetScaciStatistics_FullMethodName:             authz.EndpointManager,
	pb.CoreService_ListScaciErrors_FullMethodName:                authz.EndpointManager,
	pb.CoreService_ListScaciQueues_FullMethodName:                authz.EndpointManager,
	pb.CoreService_GetScaciStatus_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_BulkAssignBlueprint_FullMethodName:            authz.EndpointManager,
	pb.CoreService_DecodePreview_FullMethodName:                  authz.EndpointManager,
	pb.CoreService_CreateManufacturer_FullMethodName:             authz.EndpointManager,
	pb.CoreService_GetManufacturer_FullMethodName:                authz.EndpointManager,
	pb.CoreService_UpdateManufacturer_FullMethodName:             authz.EndpointManager,
	pb.CoreService_DeleteManufacturer_FullMethodName:             authz.EndpointManager,
	pb.CoreService_ListManufacturers_FullMethodName:              authz.EndpointManager,
	pb.CoreService_CreateDeviceModel_FullMethodName:              authz.EndpointManager,
	pb.CoreService_GetDeviceModel_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_UpdateDeviceModel_FullMethodName:              authz.EndpointManager,
	pb.CoreService_DeleteDeviceModel_FullMethodName:              authz.EndpointManager,
	pb.CoreService_ListDeviceModels_FullMethodName:               authz.EndpointManager,
	pb.CoreService_CreateBlueprint_FullMethodName:                authz.EndpointManager,
	pb.CoreService_GetBlueprint_FullMethodName:                   authz.EndpointManager,
	pb.CoreService_UpdateBlueprint_FullMethodName:                authz.EndpointManager,
	pb.CoreService_DeleteBlueprint_FullMethodName:                authz.EndpointManager,
	pb.CoreService_ListBlueprints_FullMethodName:                 authz.EndpointManager,
	pb.CoreService_SetDefaultBlueprint_FullMethodName:            authz.EndpointManager,
	pb.CoreService_SubmitBlueprintToRegistry_FullMethodName:      authz.EndpointManager,
	pb.CoreService_CreateDeviceModelWithBlueprint_FullMethodName: authz.EndpointManager,

	// Base stations, their traffic and certificates.
	pb.CoreService_CreateBaseStation_FullMethodName:              authz.BaseStationManager,
	pb.CoreService_GetBaseStation_FullMethodName:                 authz.BaseStationManager,
	pb.CoreService_UpdateBaseStation_FullMethodName:              authz.BaseStationManager,
	pb.CoreService_DeleteBaseStation_FullMethodName:              authz.BaseStationManager,
	pb.CoreService_ListBaseStations_FullMethodName:               authz.BaseStationManager,
	pb.CoreService_GetBaseStationStats_FullMethodName:            authz.BaseStationManager,
	pb.CoreService_UpdateBaseStationEui_FullMethodName:           authz.BaseStationManager,
	pb.CoreService_GetBaseStationAvailability_FullMethodName:     authz.BaseStationManager,
	pb.CoreService_GetBaseStationMessagesReceived_FullMethodName: authz.BaseStationManager,
	pb.CoreService_RequestBaseStationStatus_FullMethodName:       authz.BaseStationManager,
	pb.CoreService_InitiatePing_FullMethodName:                   authz.BaseStationManager,
	pb.CoreService_ListBaseStationActivity_FullMethodName:        authz.BaseStationManager,
	pb.CoreService_ListBaseStationMessages_FullMethodName:        authz.BaseStationManager,
	pb.CoreService_GetBaseStationMessage_FullMethodName:          authz.BaseStationManager,
	pb.CoreService_GetBaseStationMessageStats_FullMethodName:     authz.BaseStationManager,
	pb.CoreService_SearchBaseStationMessages_FullMethodName:      authz.BaseStationManager,
	pb.CoreService_ExportBaseStationMessages_FullMethodName:      authz.BaseStationManager,
	pb.CoreService_StreamBaseStationMessages_FullMethodName:      authz.BaseStationManager,
	pb.CoreService_GenerateCertificate_FullMethodName:            authz.BaseStationManager,
	pb.CoreService_DownloadCertificate_FullMethodName:            authz.BaseStationManager,
	pb.CoreService_DownloadBaseStationCertificate_FullMethodName: authz.BaseStationManager,
	pb.CoreService_GetServerCertificateStatus_FullMethodName:     authz.BaseStationManager,

	// Tenant-wide counts, service health and the event feeds, which narrow to the readable categories.
	pb.CoreService_GetSystemStatus_FullMethodName:           authz.AnyRole,
	pb.CoreService_GetStatistics_FullMethodName:             authz.AnyRole,
	pb.CoreService_GetAnalyticsOverview_FullMethodName:      authz.AnyRole,
	pb.CoreService_GetActivityAnalytics_FullMethodName:      authz.AnyRole,
	pb.CoreService_GetSignalQualityAnalytics_FullMethodName: authz.AnyRole,
	pb.CoreService_ListCapabilities_FullMethodName:          authz.AnyRole,
	pb.CoreService_ListEvents_FullMethodName:                authz.AnyRole,
	pb.CoreService_StreamEvents_FullMethodName:              authz.AnyRole,
	pb.CoreService_ListErrorGroups_FullMethodName:           authz.AnyRole,

	// Installation-wide operations; integration settings may hold credentials.
	pb.CoreService_CreateIntegration_FullMethodName:           authz.AdminOnly,
	pb.CoreService_GetIntegration_FullMethodName:              authz.AdminOnly,
	pb.CoreService_UpdateIntegration_FullMethodName:           authz.AdminOnly,
	pb.CoreService_DeleteIntegration_FullMethodName:           authz.AdminOnly,
	pb.CoreService_ListIntegrations_FullMethodName:            authz.AdminOnly,
	pb.CoreService_GetDiagnosticsBundle_FullMethodName:        authz.AdminOnly,
	pb.CoreService_ListAlerts_FullMethodName:                  authz.AdminOnly,
	pb.CoreService_GetAlertSummary_FullMethodName:             authz.AdminOnly,
	pb.CoreService_GenerateServerCertificates_FullMethodName:  authz.AdminOnly,
	pb.CoreService_RenewServerCertificates_FullMethodName:     authz.AdminOnly,
	pb.CoreService_ListAllBaseStationLocations_FullMethodName: authz.AdminOnly,
	pb.CoreService_ListCEInstances_FullMethodName:             authz.AdminOnly,
	pb.CoreService_RevokeCEInstance_FullMethodName:            authz.AdminOnly,
}

// methodPolicy answers role requirements for CoreService RPCs and for their
// KiloCenterService twins, which share the CoreService requirement by name.
type methodPolicy map[string]authz.Requirement

// NewMethodPolicy builds the role policy the authorization interceptor enforces.
func NewMethodPolicy() interceptors.MethodPolicy {
	policy := make(methodPolicy)
	for method, requirement := range coreMethodRequirements {
		policy[method] = requirement
	}
	compat := pb.KiloCenterService_ServiceDesc
	for _, name := range serviceMethodNames(compat) {
		if requirement, ok := coreMethodRequirements[fullMethodName(pb.CoreService_ServiceDesc, name)]; ok {
			policy[fullMethodName(compat, name)] = requirement
		}
	}
	return policy
}

func (p methodPolicy) Requirement(fullMethod string) (authz.Requirement, bool) {
	requirement, ok := p[fullMethod]
	return requirement, ok
}

// fullMethodSeparator joins the service and method names of a full gRPC method name.
const fullMethodSeparator = "/"

func fullMethodName(desc grpc.ServiceDesc, method string) string {
	return fullMethodSeparator + desc.ServiceName + fullMethodSeparator + method
}

// serviceMethodNames lists the unary and streaming RPC names of a service.
func serviceMethodNames(desc grpc.ServiceDesc) []string {
	names := make([]string, 0, len(desc.Methods)+len(desc.Streams))
	for _, m := range desc.Methods {
		names = append(names, m.MethodName)
	}
	for _, s := range desc.Streams {
		names = append(names, s.StreamName)
	}
	return names
}
