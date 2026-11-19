package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Expected connected-session counts for the listing endpoint table tests.
const (
	wantNoSessions    = 0
	wantSingleSession = 1
)

// fakeSessionPropagator is a focused stand-in for the BSSCI server. Only the
// connected-session listing is exercised here; the propagation methods exist to
// satisfy SessionPropagator.
type fakeSessionPropagator struct {
	sessions []map[string]interface{}
}

func (f *fakeSessionPropagator) SendAttachPropagate(_ string, _ uint64, _ []byte,
	_ uint16, _ bool, _ uint32, _ bool, _ uint8, _ bool, _ bool) error {
	return nil
}

func (f *fakeSessionPropagator) SendAttachPropagateToAll(_ uint64, _ []byte,
	_ uint16, _ bool, _ uint32, _ bool, _ uint8, _ bool, _ bool) []error {
	return nil
}

func (f *fakeSessionPropagator) SendDetachPropagate(_ string, _ uint64) error { return nil }

func (f *fakeSessionPropagator) SendDetachPropagateToAll(_ uint64) []error { return nil }

func (f *fakeSessionPropagator) GetConnectedSessions() []map[string]interface{} {
	return f.sessions
}

// TestHandleGetConnectedSessions covers the handler's translation of the
// server's camelCase session map into the snake_case JSON contract, including
// the optional keys that are only present once the handshake has supplied them.
func TestHandleGetConnectedSessions(t *testing.T) {
	baseSession := func() map[string]interface{} {
		return map[string]interface{}{
			sessionKeyID:                "session-1",
			sessionKeyBaseStationEUI:    uint64(0x0102030405060708),
			sessionKeyName:              "test-bs",
			sessionKeyVendor:            "test-vendor",
			sessionKeyModel:             "test-model",
			sessionKeyConnected:         true,
			sessionKeyLastSeen:          "2026-01-01T00:00:00Z",
			sessionKeyClientVersion:     "1.0.0",
			sessionKeyNegotiatedVersion: "1.0.0",
			sessionKeyBidirectional:     true,
			sessionKeyHandshakeComplete: true,
			sessionKeyResolvedTenantID:  float64(7),
			sessionKeyOrganizationID:    "org-uuid",
		}
	}

	tests := []struct {
		name        string
		sessions    []map[string]interface{}
		wantCount   int
		wantFields  map[string]interface{}
		wantAbsent  []string
		wantPresent map[string]interface{}
	}{
		{
			name:      "no connected sessions yields an empty array",
			sessions:  []map[string]interface{}{},
			wantCount: wantNoSessions,
		},
		{
			name:      "mandatory fields are remapped to snake_case",
			sessions:  []map[string]interface{}{baseSession()},
			wantCount: wantSingleSession,
			// Response keys, deliberately spelled out: they are the HTTP contract,
			// not the server-side session map.
			wantFields: map[string]interface{}{
				"id":                 "session-1",
				"basestation_eui":    "0102030405060708",
				"name":               "test-bs",
				"vendor":             "test-vendor",
				"model":              "test-model",
				"client_version":     "1.0.0",
				"negotiated_version": "1.0.0",
				"bidirectional":      true,
				"handshake_complete": true,
				"resolved_tenant_id": float64(7),
				"organization_id":    "org-uuid",
			},
			wantAbsent: []string{"session_uuid", "sn_bs_uuid", "sn_sc_uuid", "geo_location", "connect_info"},
		},
		{
			name: "optional handshake metadata is included when present",
			sessions: []map[string]interface{}{func() map[string]interface{} {
				s := baseSession()
				s[sessionKeySessionUUID] = "sess-uuid"
				s[sessionKeySnBsUUID] = "bs-uuid"
				s[sessionKeySnScUUID] = "sc-uuid"
				s[sessionKeyGeoLocation] = "48.1,11.5"
				s[sessionKeyConnectInfo] = "info"
				return s
			}()},
			wantCount: wantSingleSession,
			wantPresent: map[string]interface{}{
				"session_uuid": "sess-uuid",
				"sn_bs_uuid":   "bs-uuid",
				"sn_sc_uuid":   "sc-uuid",
				"geo_location": "48.1,11.5",
				"connect_info": "info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewBSSCIManager(&fakeSessionPropagator{sessions: tt.sessions}, logger.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/api/internal/connected-sessions", nil)
			w := httptest.NewRecorder()

			mgr.handleGetConnectedSessions(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("unexpected status code: %d", w.Code)
			}

			var payload []map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if len(payload) != tt.wantCount {
				t.Fatalf("expected %d sessions, got %d", tt.wantCount, len(payload))
			}
			if tt.wantCount == 0 {
				return
			}

			got := payload[0]
			for key, want := range tt.wantFields {
				if got[key] != want {
					t.Errorf("field %q: expected %v, got %v", key, want, got[key])
				}
			}
			for key, want := range tt.wantPresent {
				if got[key] != want {
					t.Errorf("optional field %q: expected %v, got %v", key, want, got[key])
				}
			}
			for _, key := range tt.wantAbsent {
				if _, ok := got[key]; ok {
					t.Errorf("field %q should be absent when the server does not report it", key)
				}
			}
		})
	}
}

// TestHandleGetConnectedSessionsRejectsNonGET keeps the method guard covered.
func TestHandleGetConnectedSessionsRejectsNonGET(t *testing.T) {
	mgr := NewBSSCIManager(&fakeSessionPropagator{}, logger.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/internal/connected-sessions", nil)
	w := httptest.NewRecorder()

	mgr.handleGetConnectedSessions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
