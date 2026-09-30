package grpc

// Federation RPC handler contracts. They speak protobuf, so they live with
// the transport package; the domain-side federation contracts in
// pkg/federation stay protobuf-free.

import (
	"context"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
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
