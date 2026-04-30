package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
)

func TestHandleGetConnectedSessionsIncludesVersionFields(t *testing.T) {
	testLogger := logger.NewNop()
	server := bssci.NewServerForTesting(testLogger)

	session := &bssci.Session{
		ID:                "session-1",
		BaseStationEUI:    0x0102030405060708,
		ClientVersion:     "1.0.0",
		NegotiatedVersion: "1.0.0",
		Name:              "test-bs",
		Vendor:            "test-vendor",
		Model:             "test-model",
	}
	server.AddSessionForTesting(session)

	mgr := NewBSSCIManager(server, nil, 1, testLogger)

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
	if len(payload) != 1 {
		t.Fatalf("expected 1 session, got %d", len(payload))
	}

	sessionJSON := payload[0]
	if sessionJSON["client_version"] != "1.0.0" {
		t.Fatalf("expected client_version to be '1.0.0', got %v", sessionJSON["client_version"])
	}
	if sessionJSON["negotiated_version"] != "1.0.0" {
		t.Fatalf("expected negotiated_version to be '1.0.0', got %v", sessionJSON["negotiated_version"])
	}
}
