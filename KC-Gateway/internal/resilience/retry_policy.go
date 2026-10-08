package resilience

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/grpc"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

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

// retryableMethods lists, in a stable order, the unary methods of the given
// services that the retry policy admits.
func retryableMethods(descs ...grpc.ServiceDesc) []methodName {
	names := make([]methodName, 0)
	for _, desc := range descs {
		for _, method := range desc.Methods {
			if isRetryable(method.MethodName) {
				names = append(names, methodName{Service: desc.ServiceName, Method: method.MethodName})
			}
		}
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i].Service != names[j].Service {
			return names[i].Service < names[j].Service
		}
		return names[i].Method < names[j].Method
	})
	return names
}

func isRetryable(method string) bool {
	if retryableExactMethods[method] {
		return true
	}
	for _, prefix := range retryableMethodPrefixes {
		if strings.HasPrefix(method, prefix) {
			return true
		}
	}
	return false
}

// BuildRetryServiceConfig renders the gRPC service config that retries the
// policy's read-only methods of the given services on UNAVAILABLE.
func BuildRetryServiceConfig(cfg config.GatewayResilienceConfig, descs ...grpc.ServiceDesc) (string, error) {
	sc := serviceConfig{
		MethodConfig: []methodConfig{
			{
				Name: retryableMethods(descs...),
				RetryPolicy: &retryPolicy{
					MaxAttempts:          int(cfg.MaxRetries) + 1, //nolint:gosec // MaxRetries is a small config value, overflow impossible
					InitialBackoff:       fmt.Sprintf("%gs", cfg.RetryBackoff.Seconds()),
					MaxBackoff:           fmt.Sprintf("%gs", cfg.RetryMaxBackoff.Seconds()),
					BackoffMultiplier:    retryBackoffMultiplier,
					RetryableStatusCodes: []string{statusCodeUnavailable},
				},
			},
		},
	}
	data, err := json.Marshal(sc)
	if err != nil {
		return "", fmt.Errorf(errFmtEncodeRetryServiceConfig, err)
	}
	return string(data), nil
}
