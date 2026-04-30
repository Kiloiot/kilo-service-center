// Package federation defines the public contracts for CE/ECE federation services.
// These interfaces are the single canonical definitions used by both
// internal gRPC handlers and external ECE builder hooks.
package federation

import (
	"context"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
)

// CEBootstrapHandler provides CE onboarding RPC implementations.
type CEBootstrapHandler interface {
	GetCEStatus(ctx context.Context, req *pb.GetCEStatusRequest) (*pb.GetCEStatusResponse, error)
	CompleteCEOnboarding(ctx context.Context, req *pb.CompleteCEOnboardingRequest) (*pb.CompleteCEOnboardingResponse, error)
}

// CERegistryHandler provides ECE CE registry RPC implementations.
type CERegistryHandler interface {
	ListCEInstances(ctx context.Context, req *pb.ListCEInstancesRequest) (*pb.ListCEInstancesResponse, error)
	RevokeCEInstance(ctx context.Context, req *pb.RevokeCEInstanceRequest) (*pb.RevokeCEInstanceResponse, error)
}

// RelayController manages the lifecycle of the CE→ECE relay stream.
type RelayController interface {
	EnsureStarted(ctx context.Context) error
	IsConnected() bool
}

// RelayGate enables/disables relay routing at runtime.
type RelayGate interface {
	SetRelayEnabled(enabled bool)
}
