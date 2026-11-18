package management

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

const managementOwnerTenant = int64(12)

var errManagementOwnerUnknown = errors.New("endpoint not found")

func managementPropagationRoutes(m *BSSCIManager) map[string]struct {
	handler http.HandlerFunc
	body    map[string]interface{}
	tenants func(*fakeSessionPropagator) []int64
} {
	return map[string]struct {
		handler http.HandlerFunc
		body    map[string]interface{}
		tenants func(*fakeSessionPropagator) []int64
	}{
		"attach-propagate-all": {
			handler: m.handleAttachPropagateAll,
			body:    map[string]interface{}{"epEui": 1, "nwkSnKey": base64.StdEncoding.EncodeToString(make([]byte, 16))},
			tenants: func(f *fakeSessionPropagator) []int64 { return f.attachTenants },
		},
		"detach-propagate-all": {
			handler: m.handleDetachPropagateAll,
			body:    map[string]interface{}{"epEui": 1},
			tenants: func(f *fakeSessionPropagator) []int64 { return f.detachTenants },
		},
	}
}

func postPropagation(t *testing.T, handler http.HandlerFunc, body map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw)))
	require.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	return response
}

// The management fan-out reaches only the stations of the endpoint's owner.
func TestManagementPropagationIsScopedToTheEndpointOwner(t *testing.T) {
	for name := range managementPropagationRoutes(NewBSSCIManager(&fakeSessionPropagator{}, logger.NewNop())) {
		t.Run(name, func(t *testing.T) {
			server := &fakeSessionPropagator{owner: managementOwnerTenant}
			route := managementPropagationRoutes(NewBSSCIManager(server, logger.NewNop()))[name]

			response := postPropagation(t, route.handler, route.body)

			assert.Equal(t, true, response[responseKeySuccess])
			assert.Equal(t, []int64{managementOwnerTenant}, route.tenants(server))
		})
	}
}

// An endpoint whose owner is not known is propagated to no station.
func TestManagementPropagationOfAnUnownedEndpointReachesNoStation(t *testing.T) {
	for name := range managementPropagationRoutes(NewBSSCIManager(&fakeSessionPropagator{}, logger.NewNop())) {
		t.Run(name, func(t *testing.T) {
			server := &fakeSessionPropagator{ownerErr: errManagementOwnerUnknown}
			route := managementPropagationRoutes(NewBSSCIManager(server, logger.NewNop()))[name]

			response := postPropagation(t, route.handler, route.body)

			assert.Equal(t, false, response[responseKeySuccess])
			assert.Empty(t, route.tenants(server), "no fan-out is started")
		})
	}
}
