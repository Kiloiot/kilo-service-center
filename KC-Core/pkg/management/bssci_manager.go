// Package management provides HTTP management interfaces for BSSCI operations
package management

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkggrpc "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// JSON response field keys for management endpoints.
const (
	responseKeySuccess = "success"
	responseKeyErrors  = "errors"
)

// SessionPropagator is the BSSCI server surface the management API drives:
// attach/detach propagation to one session or to every connected session, plus
// the connected-session listing.
type SessionPropagator interface {
	SendAttachPropagate(sessionID string, endpointEUI uint64, nwkSnKey []byte,
		shortAddr uint16, bidirectional bool, lastPacketCnt uint32, dualChannel bool,
		repetition uint8, wideCarrOff bool, longBlkDist bool) error
	SendAttachPropagateToAll(endpointEUI uint64, nwkSnKey []byte,
		shortAddr uint16, bidirectional bool, lastPacketCnt uint32, dualChannel bool,
		repetition uint8, wideCarrOff bool, longBlkDist bool) []error
	SendDetachPropagate(sessionID string, endpointEUI uint64) error
	SendDetachPropagateToAll(endpointEUI uint64) []error
	GetConnectedSessions() []map[string]interface{}
}

// BSSCIManager manages BSSCI operations for API access
type BSSCIManager struct {
	server SessionPropagator
	logger logger.Logger
}

const (
	routeAttachPropagate    = "/api/internal/attach-propagate"
	routeAttachPropagateAll = "/api/internal/attach-propagate-all"
	routeConnectedSessions  = "/api/internal/connected-sessions"
	routeDetachPropagate    = "/api/internal/detach-propagate"
	routeDetachPropagateAll = "/api/internal/detach-propagate-all"
)

// NewBSSCIManager creates a new BSSCI manager
// Internal management HTTP routes (loopback only).
func NewBSSCIManager(server SessionPropagator, log logger.Logger) *BSSCIManager {
	return &BSSCIManager{
		server: server,
		logger: log,
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
	mux.HandleFunc(routeAttachPropagate, m.handleAttachPropagate)
	mux.HandleFunc(routeAttachPropagateAll, m.handleAttachPropagateAll)
	mux.HandleFunc(routeConnectedSessions, m.handleGetConnectedSessions)

	// Add endpoint for detach propagate
	mux.HandleFunc(routeDetachPropagate, m.handleDetachPropagate)
	mux.HandleFunc(routeDetachPropagateAll, m.handleDetachPropagateAll)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	m.logger.Info(logMgmtServerStarting, logger.FieldAddress, addr)

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
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtFailedToBindAddress), logger.FieldAddress, addr, logger.FieldError, err)
		return fmt.Errorf("%s: %w", bssci.ResolveErrorMessage(bssci.ErrMgmtFailedToBindAddress), err)
	}

	m.logger.Info(logMgmtServerListening, logger.FieldAddress, addr)

	// Serve using the listener (this will block)
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		m.logger.Error(bssci.ResolveErrorMessage(bssci.ErrMgmtServerFailed), logger.FieldAddress, addr, logger.FieldError, err)
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
	m.logger.DebugContext(r.Context(), bssci.LogBSSCIAttachPropagateDebug,
		logger.FieldSessionID, sessionID,
		logger.FieldRepetition, req.Repetition,
		logger.FieldEpEui, req.EndpointEUI)

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
		m.logger.ErrorContext(r.Context(), bssci.ResolveErrorMessage(bssci.ErrMgmtAttachPropagateFailed),
			logger.FieldSessionID, sessionID,
			logger.FieldEpEui, req.EndpointEUI,
			logger.FieldError, err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtAttachPropagateFailed), http.StatusInternalServerError)
		return
	}

	w.Header().Set(pkggrpc.HeaderContentType, pkggrpc.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]bool{responseKeySuccess: true}); err != nil {
		m.logger.ErrorContext(r.Context(), bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), logger.FieldError, err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), http.StatusInternalServerError)
		return
	}
}

// handleAttachPropagateAll handles attach propagate requests for all connected sessions
func (m *BSSCIManager) handleAttachPropagateAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req AttachPropagateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.logger.ErrorContext(r.Context(), logMgmtDecodeFailed, logger.FieldError, err)
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

	// Log the actual values we're passing to SendAttachPropagateToAll
	m.logger.InfoContext(r.Context(), logMgmtAttachPropagateReceived,
		logger.FieldEndpointEUI, req.EndpointEUI,
		logger.FieldShortAddr, req.ShortAddr,
		logger.FieldBidirectional, req.Bidirectional,
		logger.FieldLastPacketCnt, req.LastPacketCnt,
		logger.FieldDualChannel, req.DualChannel,
		logger.FieldRepetition, req.Repetition,
		logger.FieldWideCarrOff, req.WideCarrOff,
		logger.FieldLongBlkDist, req.LongBlkDist)
	if req.Repetition {
		m.logger.WarnContext(r.Context(), logMgmtRepetitionEnabled, logger.FieldEndpointEUI, req.EndpointEUI)
	}

	// Convert boolean repetition to uint8 (0 or 1)
	var repetitionValue uint8
	if req.Repetition {
		repetitionValue = 1
	}

	// Run the attach propagate synchronously to return accurate result
	m.logger.InfoContext(r.Context(), logMgmtAttachPropagateAllStarting, logger.FieldEndpointEUI, req.EndpointEUI)
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
			m.logger.ErrorContext(r.Context(), logMgmtAttachPropagateFailed,
				logger.FieldEndpointEUI, req.EndpointEUI,
				logger.FieldError, err)
		}
		response = map[string]interface{}{
			responseKeySuccess: false,
			responseKeyErrors:  errorMessages,
		}
	} else {
		m.logger.InfoContext(r.Context(), logMgmtAttachPropagateAllOK,
			logger.FieldEndpointEUI, req.EndpointEUI,
			logger.FieldSessionCount, len(m.server.GetConnectedSessions()))
		response = map[string]interface{}{
			responseKeySuccess: true,
			responseKeyErrors:  []string{},
		}
	}

	w.Header().Set(pkggrpc.HeaderContentType, pkggrpc.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.ErrorContext(r.Context(), bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), logger.FieldError, err)
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
		bsEui, _ := session[sessionKeyBaseStationEUI].(uint64)
		sessionData := map[string]interface{}{
			"id":                 session[sessionKeyID],
			"basestation_eui":    mioty.FormatEUI64(bsEui),
			"name":               session[sessionKeyName],
			"vendor":             session[sessionKeyVendor],
			"model":              session[sessionKeyModel],
			"connected_at":       session[sessionKeyConnected],
			"last_seen":          session[sessionKeyLastSeen],
			"client_version":     session[sessionKeyClientVersion],
			"negotiated_version": session[sessionKeyNegotiatedVersion],
			"bidirectional":      session[sessionKeyBidirectional],
			"handshake_complete": session[sessionKeyHandshakeComplete],
			"bs_op_id":           session[sessionKeyBsOpID],
			"sc_op_id":           session[sessionKeyScOpID],
			"encoding":           session[sessionKeyEncoding],
			"can_resume":         session[sessionKeyCanResume],
			// ATT-02: Tenant/org fields for roaming-aware propagation
			"resolved_tenant_id": session[sessionKeyResolvedTenantID],
			"organization_id":    session[sessionKeyOrganizationID],
		}

		// Add optional session metadata (BSSCI §3.3, §5.3)
		if sessionUUID, ok := session[sessionKeySessionUUID]; ok {
			sessionData["session_uuid"] = sessionUUID
		}
		if snBsUUID, ok := session[sessionKeySnBsUUID]; ok {
			sessionData["sn_bs_uuid"] = snBsUUID
		}
		if snScUUID, ok := session[sessionKeySnScUUID]; ok {
			sessionData["sn_sc_uuid"] = snScUUID
		}
		if geoLocation, ok := session[sessionKeyGeoLocation]; ok {
			sessionData["geo_location"] = geoLocation
		}
		if connectInfo, ok := session[sessionKeyConnectInfo]; ok {
			sessionData["connect_info"] = connectInfo
		}

		sessionList = append(sessionList, sessionData)
	}

	w.Header().Set(pkggrpc.HeaderContentType, pkggrpc.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(sessionList); err != nil {
		m.logger.ErrorContext(r.Context(), bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), logger.FieldError, err)
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

	w.Header().Set(pkggrpc.HeaderContentType, pkggrpc.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]bool{responseKeySuccess: true}); err != nil {
		m.logger.ErrorContext(r.Context(), bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), logger.FieldError, err)
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
	m.logger.InfoContext(r.Context(), logMgmtDetachPropagateAllStarting, logger.FieldEndpointEUI, req.EndpointEUI)
	errors := m.server.SendDetachPropagateToAll(req.EndpointEUI)

	var response map[string]interface{}
	if len(errors) > 0 {
		errorMessages := make([]string, len(errors))
		for i, err := range errors {
			errorMessages[i] = err.Error()
			m.logger.ErrorContext(r.Context(), logMgmtDetachPropagateFailed,
				logger.FieldEndpointEUI, req.EndpointEUI,
				logger.FieldError, err)
		}
		response = map[string]interface{}{
			responseKeySuccess: false,
			responseKeyErrors:  errorMessages,
		}
	} else {
		m.logger.InfoContext(r.Context(), logMgmtDetachPropagateAllOK,
			logger.FieldEndpointEUI, req.EndpointEUI,
			logger.FieldSessionCount, len(m.server.GetConnectedSessions()))
		response = map[string]interface{}{
			responseKeySuccess: true,
			responseKeyErrors:  []string{},
		}
	}

	w.Header().Set(pkggrpc.HeaderContentType, pkggrpc.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.ErrorContext(r.Context(), bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), logger.FieldError, err)
		http.Error(w, bssci.ResolveErrorMessage(bssci.ErrMgmtJSONEncodeFailed), http.StatusInternalServerError)
		return
	}
}
