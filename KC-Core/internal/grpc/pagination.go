package grpc

import (
	grpchelpers "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// Pagination defaults — delegated to KC-Core/pkg/grpc for cross-service use.
const (
	DefaultPageSize           = grpchelpers.DefaultPageSize
	MaxPageSize               = grpchelpers.MaxPageSize
	DefaultHighVolumePageSize = grpchelpers.DefaultHighVolumePageSize
	MaxHighVolumePageSize     = grpchelpers.MaxHighVolumePageSize
)

// clampPageSize delegates to the shared package.
func clampPageSize(requested int32) int {
	return grpchelpers.ClampPageSize(requested)
}

// clampHighVolumePageSize delegates to the shared package.
func clampHighVolumePageSize(requested int32, defaultSize int32) int32 {
	return grpchelpers.ClampHighVolumePageSize(requested, defaultSize)
}
