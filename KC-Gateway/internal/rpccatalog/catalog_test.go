package rpccatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
)

func TestServicePrefixAndFullMethod(t *testing.T) {
	assert.Equal(t, "/kilocenter.api.v1.IdentityService/", ServicePrefix(pb.IdentityService_ServiceDesc))
	assert.Equal(t, "/kilocenter.api.v1.CoreService/GetEndPoint", FullMethod(pb.CoreService_ServiceDesc, "GetEndPoint"))
}

func TestUnary_CoversEveryUnaryMethodAndNoStream(t *testing.T) {
	unary := Unary(Proxied()...)
	total := 0
	for _, desc := range Proxied() {
		total += len(desc.Methods)
		for _, method := range desc.Methods {
			assert.True(t, unary[FullMethod(desc, method.MethodName)], method.MethodName)
		}
		for _, stream := range desc.Streams {
			assert.False(t, unary[FullMethod(desc, stream.StreamName)], stream.StreamName)
		}
	}
	require.Len(t, unary, total)
	require.NotEmpty(t, pb.CoreService_ServiceDesc.Streams, "the stream exclusion must be exercised")
}

func TestStreaming_CoversEveryStreamAndNoUnaryMethod(t *testing.T) {
	streaming := Streaming(Proxied()...)
	total := 0
	for _, desc := range Proxied() {
		total += len(desc.Streams)
		for _, stream := range desc.Streams {
			assert.True(t, streaming[FullMethod(desc, stream.StreamName)], stream.StreamName)
		}
		for _, method := range desc.Methods {
			assert.False(t, streaming[FullMethod(desc, method.MethodName)], method.MethodName)
		}
	}
	require.Len(t, streaming, total)
	require.True(t, streaming[FullMethod(pb.CoreService_ServiceDesc, "StreamEvents")])
}

func TestProxied_LeavesOutTheInternalService(t *testing.T) {
	for _, desc := range Proxied() {
		assert.NotEqual(t, pb.IdentityInternalService_ServiceDesc.ServiceName, desc.ServiceName,
			"the internal identity service is never proxied, so no gateway policy may cover it")
	}
}
