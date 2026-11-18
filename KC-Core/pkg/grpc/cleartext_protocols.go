package grpc

import "net/http"

// CleartextProtocols serves gRPC-web over HTTP/1.1 and native gRPC over HTTP/2
// with prior knowledge on a listener without TLS.
func CleartextProtocols() *http.Protocols {
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	return &protocols
}
