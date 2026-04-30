package resilience

import (
	"encoding/json"
	"fmt"

	"github.com/kilocenter/KC-Core/pkg/config"
)

// Service path prefixes for gRPC full method names.
const (
	coreServicePrefix             = "/kilocenter.api.v1.CoreService/"
	identityServicePrefix         = "/kilocenter.api.v1.IdentityService/"
	kiloCenterServicePrefix       = "/kilocenter.api.v1.KiloCenterService/"
	identityInternalServicePrefix = "/kilocenter.api.v1.IdentityInternalService/"
)

// retryAllowlist contains unary idempotent RPCs safe for automatic retry.
// Only Get*, List*, Search*, Export*, Query* methods are included.
// All streaming and mutating RPCs are explicitly excluded.
var retryAllowlist = map[string]bool{
	// CoreService — Get* (read-only)
	coreServicePrefix + "GetEndPoint":                true,
	coreServicePrefix + "GetBaseStation":             true,
	coreServicePrefix + "GetBaseStationStats":        true,
	coreServicePrefix + "GetMessage":                 true,
	coreServicePrefix + "GetDownlinkResults":         true,
	coreServicePrefix + "GetDLRXStatus":              true,
	coreServicePrefix + "GetDLRXStatusQueries":       true,
	coreServicePrefix + "GetSystemStatus":            true,
	coreServicePrefix + "GetStatistics":              true,
	coreServicePrefix + "GetReleaseInfo":             true,
	coreServicePrefix + "GetIntegration":             true,
	coreServicePrefix + "GetAnalyticsOverview":       true,
	coreServicePrefix + "GetActivityAnalytics":       true,
	coreServicePrefix + "GetSignalQualityAnalytics":  true,
	coreServicePrefix + "GetAlertSummary":            true,
	coreServicePrefix + "GetScaciSession":            true,
	coreServicePrefix + "GetScaciStatistics":         true,
	coreServicePrefix + "GetScaciStatus":             true,
	coreServicePrefix + "GetServerCertificateStatus": true,
	coreServicePrefix + "GetManufacturer":            true,
	coreServicePrefix + "GetDeviceModel":             true,
	coreServicePrefix + "GetBlueprint":               true,
	coreServicePrefix + "GetBaseStationMessage":      true,
	coreServicePrefix + "GetBaseStationMessageStats": true,
	coreServicePrefix + "GetEndPointStats":           true,
	coreServicePrefix + "GetEndPointOperations":      true,
	coreServicePrefix + "DecodePreview":              true,

	// CoreService — List* (read-only)
	coreServicePrefix + "ListEndPoints":           true,
	coreServicePrefix + "ListBaseStations":        true,
	coreServicePrefix + "ListDownlinkQueue":       true,
	coreServicePrefix + "ListIntegrations":        true,
	coreServicePrefix + "ListEvents":              true,
	coreServicePrefix + "ListBaseStationActivity": true,
	coreServicePrefix + "ListEndpointActivity":    true,
	coreServicePrefix + "ListAlerts":              true,
	coreServicePrefix + "ListScaciSessions":       true,
	coreServicePrefix + "ListScaciErrors":         true,
	coreServicePrefix + "ListScaciQueues":         true,
	coreServicePrefix + "ListManufacturers":       true,
	coreServicePrefix + "ListDeviceModels":        true,
	coreServicePrefix + "ListBlueprints":          true,
	coreServicePrefix + "ListMessages":            true,
	coreServicePrefix + "ListBaseStationMessages": true,
	coreServicePrefix + "ListEndpointMessages":    true,

	// CoreService — Search*, Export*, Query* (read-only)
	coreServicePrefix + "SearchBaseStationMessages": true,
	coreServicePrefix + "ExportBaseStationMessages": true,
	coreServicePrefix + "QueryDLRXStatus":           true,

	// IdentityService — Get* (read-only)
	identityServicePrefix + "GetProfile":          true,
	identityServicePrefix + "GetAuthSettings":     true,
	identityServicePrefix + "GetUser":             true,
	identityServicePrefix + "GetOrganization":     true,
	identityServicePrefix + "GetOrganizationUser": true,
	identityServicePrefix + "GetApiKey":           true,

	// IdentityService — List* (read-only)
	identityServicePrefix + "ListUsers":             true,
	identityServicePrefix + "ListOrganizations":     true,
	identityServicePrefix + "ListOrganizationUsers": true,
	identityServicePrefix + "ListApiKeys":           true,

	// KiloCenterService (compat) — mirrors CoreService + IdentityService read-only methods
	kiloCenterServicePrefix + "GetEndPoint":                true,
	kiloCenterServicePrefix + "GetBaseStation":             true,
	kiloCenterServicePrefix + "GetBaseStationStats":        true,
	kiloCenterServicePrefix + "GetMessage":                 true,
	kiloCenterServicePrefix + "GetDownlinkResults":         true,
	kiloCenterServicePrefix + "GetDLRXStatus":              true,
	kiloCenterServicePrefix + "GetDLRXStatusQueries":       true,
	kiloCenterServicePrefix + "GetSystemStatus":            true,
	kiloCenterServicePrefix + "GetStatistics":              true,
	kiloCenterServicePrefix + "GetReleaseInfo":             true,
	kiloCenterServicePrefix + "GetIntegration":             true,
	kiloCenterServicePrefix + "GetAnalyticsOverview":       true,
	kiloCenterServicePrefix + "GetActivityAnalytics":       true,
	kiloCenterServicePrefix + "GetSignalQualityAnalytics":  true,
	kiloCenterServicePrefix + "GetAlertSummary":            true,
	kiloCenterServicePrefix + "GetScaciSession":            true,
	kiloCenterServicePrefix + "GetScaciStatistics":         true,
	kiloCenterServicePrefix + "GetScaciStatus":             true,
	kiloCenterServicePrefix + "GetServerCertificateStatus": true,
	kiloCenterServicePrefix + "GetManufacturer":            true,
	kiloCenterServicePrefix + "GetDeviceModel":             true,
	kiloCenterServicePrefix + "GetBlueprint":               true,
	kiloCenterServicePrefix + "GetBaseStationMessage":      true,
	kiloCenterServicePrefix + "GetBaseStationMessageStats": true,
	kiloCenterServicePrefix + "GetEndPointStats":           true,
	kiloCenterServicePrefix + "GetEndPointOperations":      true,
	kiloCenterServicePrefix + "DecodePreview":              true,
	kiloCenterServicePrefix + "ListEndPoints":              true,
	kiloCenterServicePrefix + "ListBaseStations":           true,
	kiloCenterServicePrefix + "ListDownlinkQueue":          true,
	kiloCenterServicePrefix + "ListIntegrations":           true,
	kiloCenterServicePrefix + "ListEvents":                 true,
	kiloCenterServicePrefix + "ListBaseStationActivity":    true,
	kiloCenterServicePrefix + "ListEndpointActivity":       true,
	kiloCenterServicePrefix + "ListAlerts":                 true,
	kiloCenterServicePrefix + "ListScaciSessions":          true,
	kiloCenterServicePrefix + "ListScaciErrors":            true,
	kiloCenterServicePrefix + "ListScaciQueues":            true,
	kiloCenterServicePrefix + "ListManufacturers":          true,
	kiloCenterServicePrefix + "ListDeviceModels":           true,
	kiloCenterServicePrefix + "ListBlueprints":             true,
	kiloCenterServicePrefix + "ListMessages":               true,
	kiloCenterServicePrefix + "ListBaseStationMessages":    true,
	kiloCenterServicePrefix + "ListEndpointMessages":       true,
	kiloCenterServicePrefix + "SearchBaseStationMessages":  true,
	kiloCenterServicePrefix + "ExportBaseStationMessages":  true,
	kiloCenterServicePrefix + "QueryDLRXStatus":            true,
	kiloCenterServicePrefix + "GetProfile":                 true,
	kiloCenterServicePrefix + "GetAuthSettings":            true,
	kiloCenterServicePrefix + "GetUser":                    true,
	kiloCenterServicePrefix + "GetOrganization":            true,
	kiloCenterServicePrefix + "GetOrganizationUser":        true,
	kiloCenterServicePrefix + "GetApiKey":                  true,
	kiloCenterServicePrefix + "ListUsers":                  true,
	kiloCenterServicePrefix + "ListOrganizations":          true,
	kiloCenterServicePrefix + "ListOrganizationUsers":      true,
	kiloCenterServicePrefix + "ListApiKeys":                true,

	// IdentityInternalService — read-only
	identityInternalServicePrefix + "GetDefaultOrgForTenant": true,
}

// IsRetryable returns true if the given full gRPC method name is safe to retry.
func IsRetryable(fullMethod string) bool {
	return retryAllowlist[fullMethod]
}

// retryAllowlistMethods returns a copy of all retryable method names.
func retryAllowlistMethods() []string {
	methods := make([]string, 0, len(retryAllowlist))
	for m := range retryAllowlist {
		methods = append(methods, m)
	}
	return methods
}

// streamingMethodSet lists all server-streaming RPCs across all services.
// gRPC retry is unary-only; streaming methods must never appear in the retry allowlist.
var streamingMethodSet = map[string]bool{
	coreServicePrefix + "StreamEvents":                    true,
	coreServicePrefix + "StreamMessages":                  true,
	coreServicePrefix + "StreamBaseStationMessages":       true,
	kiloCenterServicePrefix + "StreamEvents":              true,
	kiloCenterServicePrefix + "StreamMessages":            true,
	kiloCenterServicePrefix + "StreamBaseStationMessages": true,
}

// streamingMethods returns the set of streaming method names.
func streamingMethods() map[string]bool {
	out := make(map[string]bool, len(streamingMethodSet))
	for k, v := range streamingMethodSet {
		out[k] = v
	}
	return out
}

// BuildRetryServiceConfig builds a gRPC service config JSON string with retry
// policy for the methods in the retry allowlist.
func BuildRetryServiceConfig(cfg config.GatewayResilienceConfig) string {
	type retryPolicy struct {
		MaxAttempts          int      `json:"maxAttempts"`
		InitialBackoff       string   `json:"initialBackoff"`
		MaxBackoff           string   `json:"maxBackoff"`
		BackoffMultiplier    float64  `json:"backoffMultiplier"`
		RetryableStatusCodes []string `json:"retryableStatusCodes"`
	}

	type methodName struct {
		Service string `json:"service,omitempty"`
		Method  string `json:"method,omitempty"`
	}

	type methodConfig struct {
		Name        []methodName `json:"name"`
		RetryPolicy *retryPolicy `json:"retryPolicy,omitempty"`
	}

	type serviceConfig struct {
		MethodConfig []methodConfig `json:"methodConfig"`
	}

	// Build per-method entries for each retryable method
	names := make([]methodName, 0, len(retryAllowlist))
	for fullMethod := range retryAllowlist {
		svc, method := splitFullMethod(fullMethod)
		names = append(names, methodName{Service: svc, Method: method})
	}

	sc := serviceConfig{
		MethodConfig: []methodConfig{
			{
				Name: names,
				RetryPolicy: &retryPolicy{
					MaxAttempts:          int(cfg.MaxRetries) + 1, //nolint:gosec // MaxRetries is a small config value, overflow impossible
					InitialBackoff:       fmt.Sprintf("%gs", cfg.RetryBackoff.Seconds()),
					MaxBackoff:           fmt.Sprintf("%gs", cfg.RetryMaxBackoff.Seconds()),
					BackoffMultiplier:    2.0,
					RetryableStatusCodes: []string{"UNAVAILABLE"},
				},
			},
		},
	}

	data, _ := json.Marshal(sc)
	return string(data)
}

// splitFullMethod splits "/service.Name/Method" into ("service.Name", "Method").
func splitFullMethod(fullMethod string) (string, string) {
	// fullMethod format: "/package.Service/Method"
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	for i := 0; i < len(fullMethod); i++ {
		if fullMethod[i] == '/' {
			return fullMethod[:i], fullMethod[i+1:]
		}
	}
	return fullMethod, ""
}
