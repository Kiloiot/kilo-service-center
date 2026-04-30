// Package management provides HTTP management interfaces for BSSCI operations
package management

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
)

// BSSCIManager manages BSSCI operations for API access
type BSSCIManager struct {
	server       *bssci.Server
	messageStore interface{} // Will be *postgres.DB
	tenantID     int64
	logger       logger.Logger
}

// NewBSSCIManager creates a new BSSCI manager
func NewBSSCIManager(server *bssci.Server, messageStore interface{}, tenantID int64, log logger.Logger) *BSSCIManager {
	return &BSSCIManager{
		server:       server,
		messageStore: messageStore,
		tenantID:     tenantID,
		logger:       log,
	}
}

// AttachPropagateRequest represents a request to propagate endpoint attachment
type AttachPropagateRequest struct {
	EndpointEUI   uint64 `json:"epEui"`
	NwkSnKey      string `json:"nwkSnKey"` // Base64-encoded in JSON transport
	ShortAddr     uint16 `json:"shAddr"`
	Bidirectional bool   `json:"bidi"`
	LastPacketCnt uint32 `json:"lastPacketCnt"`
	DualChannel   bool   `json:"dualChan"`   // Match API field name
	Repetition    bool   `json:"repetition"` // Boolean per MIOTY BSSCI v1.0.0 spec
	WideCarrOff   bool   `json:"wideCarrOff"`
	LongBlkDist   bool   `json:"longBlkDist"`
}

// DetachPropagateRequest represents a request to propagate endpoint detachment
type DetachPropagateRequest struct {
	EndpointEUI uint64 `json:"epEui"`
	ShortAddr   uint16 `json:"shAddr"`
}

// StartHTTPServer starts an HTTP server for management operations
func (m *BSSCIManager) StartHTTPServer(port int) error {
	mux := http.NewServeMux()

	// Add endpoint for attach propagate
	mux.HandleFunc("/api/internal/attach-propagate", m.handleAttachPropagate)
	mux.HandleFunc("/api/internal/attach-propagate-all", m.handleAttachPropagateAll)
	mux.HandleFunc("/api/internal/connected-sessions", m.handleGetConnectedSessions)

	// Add endpoint for detach propagate
	mux.HandleFunc("/api/internal/detach-propagate", m.handleDetachPropagate)
	mux.HandleFunc("/api/internal/detach-propagate-all", m.handleDetachPropagateAll)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	m.logger.Info("Starting BSSCI management HTTP server", "address", addr)

	// Use http.Server for better control and error handling
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: HTTPReadHeaderTimeout,
		IdleTimeout:       HTTPIdleTimeout,
	}

	// Create a listener to test if the address is available
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtFailedToBindAddress), "address", addr, "error", err)
		return fmt.Errorf("%s: %w", bssci.ResolveErrorMessage(bssci.ErrMgmtFailedToBindAddress), err)
	}

	m.logger.Info("BSSCI management HTTP server listening", "address", addr)

	// Serve using the listener (this will block)
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtServerFailed), "address", addr, "error", err)
		return fmt.Errorf("%s: %w", bssci.ResolveErrorMessage(bssci.ErrMgmtServerFailed), err)
	}

	return nil
}

// handleAttachPropagate handles attach propagate requests for a specific session
func (m *BSSCIManager) handleAttachPropagate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtSessionIDRequired), http.StatusBadRequest)
		return
	}

	var req AttachPropagateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidRequestBody), http.StatusBadRequest)
		return
	}

	// Decode base64-encoded network session key
	nwkSnKey, err := base64.StdEncoding.DecodeString(req.NwkSnKey)
	if err != nil {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidNwkSnKeyEncoding), http.StatusBadRequest)
		return
	}

	// BSSCI-3.8.1-01: Validate nwkSnKey is exactly 16 bytes
	if len(nwkSnKey) != 16 {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidNwkSnKeyLength), http.StatusBadRequest)
		return
	}

	// DEBUG: Log what we received for single session attach propagate
	m.logger.Debug(bssci.LogBSSCIAttachPropagateDebug,
		"sessionID", sessionID,
		"repetition", req.Repetition,
		"epEui", req.EndpointEUI)

	// Convert boolean repetition to uint8 (0 or 1)
	var repetitionValue uint8
	if req.Repetition {
		repetitionValue = 1
	}

	err = m.server.SendAttachPropagate(
		sessionID,
		req.EndpointEUI,
		nwkSnKey,
		req.ShortAddr,
		req.Bidirectional,
		req.LastPacketCnt,
		req.DualChannel,
		repetitionValue,
		req.WideCarrOff,
		req.LongBlkDist,
	)

	if err != nil {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtAttachPropagateFailed),
			"sessionID", sessionID,
			"epEui", req.EndpointEUI,
			"error", err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtAttachPropagateFailed), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), "error", err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), http.StatusInternalServerError)
		return
	}
}

// handleAttachPropagateAll handles attach propagate requests for all connected sessions
func (m *BSSCIManager) handleAttachPropagateAll(w http.ResponseWriter, r *http.Request) {
	m.logger.Debug("handleAttachPropagateAll called", "method", r.Method)

	if r.Method != http.MethodPost {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	m.logger.Debug("About to decode JSON request body")
	var req AttachPropagateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.logger.Error("Failed to decode JSON", "error", err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidRequestBody), http.StatusBadRequest)
		return
	}
	m.logger.Debug("Successfully decoded request",
		"epEui", req.EndpointEUI,
		"shAddr", req.ShortAddr,
		"bidi", req.Bidirectional)

	// Decode base64-encoded network session key
	nwkSnKey, err := base64.StdEncoding.DecodeString(req.NwkSnKey)
	if err != nil {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidNwkSnKeyEncoding), http.StatusBadRequest)
		return
	}

	// BSSCI-3.8.1-01: Validate nwkSnKey is exactly 16 bytes
	if len(nwkSnKey) != 16 {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidNwkSnKeyLength), http.StatusBadRequest)
		return
	}

	// Log the actual values we're passing to SendAttachPropagateToAll
	m.logger.Info("Received attach propagate request",
		"endpointEUI", req.EndpointEUI,
		"shortAddr", req.ShortAddr,
		"bidirectional", req.Bidirectional,
		"lastPacketCnt", req.LastPacketCnt,
		"dualChannel", req.DualChannel,
		"repetition", req.Repetition,
		"wideCarrOff", req.WideCarrOff,
		"longBlkDist", req.LongBlkDist)
	if req.Repetition {
		m.logger.Warn("Repetition is enabled - may affect DL performance", "endpointEUI", req.EndpointEUI)
	}

	// Convert boolean repetition to uint8 (0 or 1)
	var repetitionValue uint8
	if req.Repetition {
		repetitionValue = 1
	}

	// Run the attach propagate synchronously to return accurate result
	m.logger.Info("Starting SendAttachPropagateToAll", "endpointEUI", req.EndpointEUI)
	errors := m.server.SendAttachPropagateToAll(
		req.EndpointEUI,
		nwkSnKey,
		req.ShortAddr,
		req.Bidirectional,
		req.LastPacketCnt,
		req.DualChannel,
		repetitionValue,
		req.WideCarrOff,
		req.LongBlkDist,
	)

	var response map[string]interface{}
	if len(errors) > 0 {
		errorMessages := make([]string, len(errors))
		for i, err := range errors {
			errorMessages[i] = err.Error()
			m.logger.Error("Attach propagate failed",
				"endpointEUI", req.EndpointEUI,
				"error", err)
		}
		response = map[string]interface{}{
			"success": false,
			"errors":  errorMessages,
		}
	} else {
		m.logger.Info("Attach propagate sent successfully to all sessions",
			"endpointEUI", req.EndpointEUI,
			"sessionCount", len(m.server.GetConnectedSessions()))
		response = map[string]interface{}{
			"success": true,
			"errors":  []string{},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), "error", err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), http.StatusInternalServerError)
		return
	}
}

// handleGetConnectedSessions returns list of connected base station sessions
func (m *BSSCIManager) handleGetConnectedSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	sessions := m.server.GetConnectedSessions()

	// Convert to JSON-friendly format with enriched session metadata (BSSCI §5-5.3)
	sessionList := []map[string]interface{}{}
	for _, session := range sessions {
		sessionData := map[string]interface{}{
			"id":                 session["id"],
			"basestation_eui":    fmt.Sprintf("%016X", session["baseStationEui"]),
			"name":               session["name"],
			"vendor":             session["vendor"],
			"model":              session["model"],
			"connected_at":       session["connected"],
			"last_seen":          session["lastSeen"],
			"client_version":     session["clientVersion"],
			"negotiated_version": session["negotiatedVersion"],
			"bidirectional":      session["bidirectional"],
			"handshake_complete": session["handshakeComplete"],
			"bs_op_id":           session["bsOpId"],
			"sc_op_id":           session["scOpId"],
			"encoding":           session["encoding"],
			"can_resume":         session["canResume"],
			// ATT-02: Tenant/org fields for roaming-aware propagation
			"resolved_tenant_id": session["resolvedTenantID"],
			"organization_id":    session["organizationID"],
		}

		// Add optional session metadata (BSSCI §3.3, §5.3)
		if sessionUUID, ok := session["sessionUuid"]; ok {
			sessionData["session_uuid"] = sessionUUID
		}
		if snBsUUID, ok := session["snBsUuid"]; ok {
			sessionData["sn_bs_uuid"] = snBsUUID
		}
		if snScUUID, ok := session["snScUuid"]; ok {
			sessionData["sn_sc_uuid"] = snScUUID
		}
		if geoLocation, ok := session["geoLocation"]; ok {
			sessionData["geo_location"] = geoLocation
		}
		if connectInfo, ok := session["connectInfo"]; ok {
			sessionData["connect_info"] = connectInfo
		}

		sessionList = append(sessionList, sessionData)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sessionList); err != nil {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), "error", err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), http.StatusInternalServerError)
		return
	}
}

// handleDetachPropagate handles detach propagate requests for a specific session
func (m *BSSCIManager) handleDetachPropagate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtSessionIDRequired), http.StatusBadRequest)
		return
	}

	var req DetachPropagateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidRequestBody), http.StatusBadRequest)
		return
	}

	err := m.server.SendDetachPropagate(sessionID, req.EndpointEUI)
	if err != nil {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtDetachPropagateFailed), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), "error", err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), http.StatusInternalServerError)
		return
	}
}

// handleDetachPropagateAll handles detach propagate requests for all connected sessions
func (m *BSSCIManager) handleDetachPropagateAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req DetachPropagateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtInvalidRequestBody), http.StatusBadRequest)
		return
	}

	// Run the detach propagate synchronously to return accurate result
	m.logger.Info("Starting SendDetachPropagateToAll", "endpointEUI", req.EndpointEUI)
	errors := m.server.SendDetachPropagateToAll(req.EndpointEUI)

	var response map[string]interface{}
	if len(errors) > 0 {
		errorMessages := make([]string, len(errors))
		for i, err := range errors {
			errorMessages[i] = err.Error()
			m.logger.Error("Detach propagate failed",
				"endpointEUI", req.EndpointEUI,
				"error", err)
		}
		response = map[string]interface{}{
			"success": false,
			"errors":  errorMessages,
		}
	} else {
		m.logger.Info("Detach propagate sent successfully to all sessions",
			"endpointEUI", req.EndpointEUI,
			"sessionCount", len(m.server.GetConnectedSessions()))
		response = map[string]interface{}{
			"success": true,
			"errors":  []string{},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), "error", err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), http.StatusInternalServerError)
		return
	}
}
