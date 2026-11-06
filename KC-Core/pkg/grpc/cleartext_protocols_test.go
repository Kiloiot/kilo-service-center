package grpc

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testHTTP1Major = 1
	testHTTP2Major = 2
)

func protoMajorEcho() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strconv.Itoa(r.ProtoMajor)))
	})
}

func roundTripMajor(t *testing.T, url string, client *http.Protocols) int {
	t.Helper()
	req, err := http.NewRequestWithContext(testutil.TestContext(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := (&http.Client{Transport: &http.Transport{Protocols: client}}).Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.ProtoMajor
}

// A plaintext listener answers native gRPC clients over HTTP/2 with prior
// knowledge and browsers' gRPC-web over HTTP/1.1.
func TestCleartextProtocols_ServeHTTP2PriorKnowledgeAndHTTP1(t *testing.T) {
	server := httptest.NewUnstartedServer(protoMajorEcho())
	server.Config.Protocols = CleartextProtocols()
	server.Start()
	t.Cleanup(server.Close)

	var h2c http.Protocols
	h2c.SetUnencryptedHTTP2(true)
	assert.Equal(t, testHTTP2Major, roundTripMajor(t, server.URL, &h2c))

	var h1 http.Protocols
	h1.SetHTTP1(true)
	assert.Equal(t, testHTTP1Major, roundTripMajor(t, server.URL, &h1))
}
