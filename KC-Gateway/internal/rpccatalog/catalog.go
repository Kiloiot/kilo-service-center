// Package rpccatalog derives the gateway's method sets from the generated
// service descriptors so routing, retry and timeout policies share one source.
package rpccatalog

import (
	"google.golang.org/grpc"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
)

const methodSeparator = "/"

// Proxied lists every service the gateway forwards to an upstream. The
// internal IdentityInternalService is never forwarded, so no per-method policy
// covers it.
func Proxied() []grpc.ServiceDesc {
	return []grpc.ServiceDesc{
		pb.CoreService_ServiceDesc,
		pb.IdentityService_ServiceDesc,
		pb.KiloCenterService_ServiceDesc,
	}
}

// ServicePrefix returns the "/package.Service/" prefix shared by every full
// method name of desc.
func ServicePrefix(desc grpc.ServiceDesc) string {
	return methodSeparator + desc.ServiceName + methodSeparator
}

// FullMethod returns the full method name of one method of desc.
func FullMethod(desc grpc.ServiceDesc, method string) string {
	return ServicePrefix(desc) + method
}

// Streaming returns the full names of every streaming method of the given services.
func Streaming(descs ...grpc.ServiceDesc) map[string]bool {
	methods := make(map[string]bool)
	for _, desc := range descs {
		for _, stream := range desc.Streams {
			methods[FullMethod(desc, stream.StreamName)] = true
		}
	}
	return methods
}

// Unary returns the full names of every unary method of the given services.
func Unary(descs ...grpc.ServiceDesc) map[string]bool {
	methods := make(map[string]bool)
	for _, desc := range descs {
		for _, method := range desc.Methods {
			methods[FullMethod(desc, method.MethodName)] = true
		}
	}
	return methods
}
