package bssci

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// sessionRegistry is the live-session map keyed by session ID with its own
// lock, so handlers never hold the server lifecycle lock for a lookup.
type sessionRegistry struct {
	mu   sync.RWMutex
	byID map[string]*Session
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{byID: make(map[string]*Session)}
}

func (r *sessionRegistry) get(id string) (*Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.byID[id]
	return session, ok
}

// remove unpublishes a session and reports whether it was the live one.
func (r *sessionRegistry) remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, live := r.byID[id]
	delete(r.byID, id)
	return live
}

// byEUI returns the first live session for the base station that passes
// accept, or nil.
func (r *sessionRegistry) byEUI(bsEui uint64, accept func(*Session) bool) *Session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, session := range r.byID {
		if session.BaseStationEUI == bsEui && (accept == nil || accept(session)) {
			return session
		}
	}
	return nil
}

// snapshot copies the live sessions out of the lock so callers can iterate
// and log without holding it.
func (r *sessionRegistry) snapshot() []*Session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Session, 0, len(r.byID))
	for _, session := range r.byID {
		out = append(out, session)
	}
	return out
}

// displace publishes session as the only live session for its base station
// and returns the sessions it evicted. Eviction and publication share one
// critical section so a by-EUI lookup never resolves to the connection being
// replaced.
func (r *sessionRegistry) displace(session *Session) []*Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	var displaced []*Session
	for id, live := range r.byID {
		if id != session.ID && live.BaseStationEUI == session.BaseStationEUI {
			displaced = append(displaced, live)
			delete(r.byID, id)
		}
	}
	r.byID[session.ID] = session
	return displaced
}

// GetConnectedSessionEUIs returns the EUIs of all connected sessions as hex strings.
// Used by EndpointAttachmentService for event logging.
func (s *Server) GetConnectedSessionEUIs() []string {
	live := s.sessions.snapshot()
	euis := make([]string, 0, len(live))
	for _, session := range live {
		// Connected is a time.Time - check if not zero (session is active)
		if !session.Connected.IsZero() && session.HandshakeComplete {
			euis = append(euis, mioty.FormatEUI64(session.BaseStationEUI))
		}
	}
	return euis
}

// Keys of the session snapshot GetConnectedSessions emits.
const (
	SessionKeyID                = "id"
	SessionKeyBaseStationEUI    = "baseStationEui"
	SessionKeyConnected         = "connected"
	SessionKeyLastSeen          = "lastSeen"
	SessionKeyVendor            = "vendor"
	SessionKeyModel             = "model"
	SessionKeyName              = "name"
	SessionKeyClientVersion     = "clientVersion"
	SessionKeyNegotiatedVersion = "negotiatedVersion"
	SessionKeyBidirectional     = "bidirectional"
	SessionKeyHandshakeComplete = "handshakeComplete"
	SessionKeyResolvedTenantID  = "resolvedTenantID"
	SessionKeyOrganizationID    = "organizationID"
)

// GetConnectedSessions returns information about all connected sessions
func (s *Server) GetConnectedSessions() []map[string]interface{} {
	live := s.sessions.snapshot()
	sessions := make([]map[string]interface{}, 0, len(live))
	for _, session := range live {
		sessions = append(sessions, map[string]interface{}{
			SessionKeyID:                session.ID,
			SessionKeyBaseStationEUI:    session.BaseStationEUI,
			SessionKeyConnected:         session.Connected,
			SessionKeyLastSeen:          session.LastSeen,
			SessionKeyVendor:            session.Vendor,
			SessionKeyModel:             session.Model,
			SessionKeyName:              session.Name,
			SessionKeyClientVersion:     session.ClientVersion,
			SessionKeyNegotiatedVersion: session.NegotiatedVersion,
			SessionKeyBidirectional:     session.Bidirectional,
			SessionKeyHandshakeComplete: session.HandshakeComplete,
			SessionKeyResolvedTenantID:  session.ResolvedTenantID,
			SessionKeyOrganizationID:    session.OrganizationID.String(),
		})
	}

	return sessions
}

// GetSessionByEUI returns a session for a given base station EUI
// Returns nil if no connected session exists for the given EUI
func (s *Server) GetSessionByEUI(bsEui uint64) interface{} {
	session := s.sessions.byEUI(bsEui, func(session *Session) bool { return session.HandshakeComplete })
	if session == nil {
		// A typed nil must not escape as a non-nil interface value.
		return nil
	}
	return session
}

// publishLiveSession makes session the only live session for its base station
// and tears down the connections it displaced.
func (s *Server) publishLiveSession(ctx context.Context, session *Session) {
	for _, stale := range s.sessions.displace(session) {
		s.logger.WarnContext(ctx, LogBSSCIDisplacedLiveSessionForBaseStation,
			logger.FieldEui, session.BaseStationEUI,
			logger.FieldDisplacedSessionID, stale.ID,
			logger.FieldSessionID, session.ID)
		s.sessionSvc.RemoveSession(stale)
		if stale.Conn != nil {
			if err := stale.Conn.Close(); err != nil {
				s.logger.WarnContext(ctx, LogBSSCIFailedToCloseDisplacedSessionConnection,
					logger.FieldError, err,
					logger.FieldDisplacedSessionID, stale.ID)
			}
		}
	}
}

// CloseSessionByEUI closes the live session of the base station with the
// given EUI and reports whether it held one. It terminates the DB session,
// removes the session from the live maps and closes the connection, so a
// station whose EUI changed keeps no session under its old identity and a
// deleted station keeps none at all.
func (s *Server) CloseSessionByEUI(ctx context.Context, eui uint64) bool {
	targetSession := s.sessions.byEUI(eui, nil)
	if targetSession == nil {
		return false
	}

	s.logger.InfoContext(ctx, LogBSSCIClosingRetiredStationSession,
		logger.FieldEui, eui,
		logger.FieldSessionID, targetSession.ID)

	// Terminate DB session record
	if targetSession.DbSessionID != 0 && s.sessionSvc != nil {
		if err := s.sessionSvc.TerminateSession(ctx, targetSession); err != nil {
			s.logger.WarnContext(ctx, LogBSSCIFailedToTerminateRetiredStationSession,
				logger.FieldError, err,
				logger.FieldSessionID, targetSession.DbSessionID)
		}
	}

	// Remove from SessionService's sessionsByUUID map to prevent stale resume
	if s.sessionSvc != nil {
		s.sessionSvc.RemoveSession(targetSession)
	}

	s.sessions.remove(targetSession.ID)

	// Close connection to trigger cleanup
	if targetSession.Conn != nil {
		if err := targetSession.Conn.Close(); err != nil {
			s.logger.WarnContext(ctx, LogBSSCIFailedToCloseRetiredStationConnection,
				logger.FieldError, err,
				logger.FieldSessionID, targetSession.ID)
		}
	}

	return true
}

// ConnectedStations lists the EUIs of the base stations with a completed
// handshake on this service center.
func (s *Server) ConnectedStations() []uint64 {
	live := s.sessions.snapshot()
	stations := make([]uint64, 0, len(live))
	for _, session := range live {
		if session.HandshakeComplete {
			stations = append(stations, session.BaseStationEUI)
		}
	}
	return stations
}

// ConnectedSessionsSnapshot returns lightweight snapshots of all connected base station sessions
// Used by propagation service to broadcast attach propagate messages to multiple base stations
// Implements SessionSnapshotProvider interface for SCACI integration
//
// BSSCI §5.8-5.8.3: Automatic endpoint propagation across multi-BS networks
func (s *Server) ConnectedSessionsSnapshot() []propagation.BaseStationSession {
	live := s.sessions.snapshot()
	snapshots := make([]propagation.BaseStationSession, 0, len(live))
	for _, session := range live {
		if !session.HandshakeComplete {
			continue
		}

		var orgIDStr *string
		if session.OrganizationID != uuid.Nil {
			orgStr := session.OrganizationID.String()
			orgIDStr = &orgStr
		}

		snapshots = append(snapshots, propagation.BaseStationSession{
			ID:                session.ID,
			BaseStationEUI:    session.BaseStationEUI,
			TenantID:          session.ResolvedTenantID,
			OrganizationID:    orgIDStr,
			HandshakeComplete: session.HandshakeComplete,
		})
	}

	return snapshots
}

// SelectBidirectionalSession finds a suitable bidirectional base station session for UL transmit
//
// This helper is used by both SCACI (via ScheduleULDataTransmit interface) and gRPC
// to select an appropriate base station for uplink transmission.
//
// Tenant isolation: Only selects base stations that belong to the requesting tenant,
// preventing cross-tenant UL transmit operations (BSSCI §5.11 multi-tenant isolation).
//
// Parameters:
//   - tenantID: Tenant context for session selection (enforced via ResolvedTenantID comparison)
//   - targetBsEui: Optional specific base station EUI (nil = SC chooses automatically)
//
// Returns:
//   - sessionID: Internal session identifier
//   - actualBsEui: EUI of selected base station
//   - error: ErrBaseStationTenantMismatch if requested BS belongs to different tenant
func (s *Server) SelectBidirectionalSession(tenantID int64, targetBsEui *uint64) (sessionID string, actualBsEui uint64, err error) {
	live := s.sessions.snapshot()

	// If specific BS requested, find it and validate tenant ownership
	if targetBsEui != nil {
		for _, session := range live {
			sid := session.ID
			if session.BaseStationEUI == *targetBsEui &&
				session.HandshakeComplete &&
				session.Bidirectional {
				// Enforce tenant isolation: check resolved tenant (supports roaming/resumed sessions)
				if session.ResolvedTenantID != tenantID {
					// Log tenant mismatch attempt for security auditing
					ctx := s.sessionContext(session)
					s.logger.WarnContext(
						ctx, LogBSSCIBaseStationTenantMismatch,
						logger.FieldSessionID, sid,
						logger.FieldTenantID, tenantID,
						logger.FieldResolvedTenantID, session.ResolvedTenantID,
						logger.FieldRequestedBsEui, mioty.FormatEUI64(*targetBsEui),
					)
					return "", 0, ErrBaseStationTenantMismatch
				}
				return sid, session.BaseStationEUI, nil
			}
		}
		// Specific BS not found or not suitable - wrap sentinel for errors.Is() detection
		return "", 0, fmt.Errorf("%s %s: %w", ResolveErrorMessage(errBaseStationNotRegistered), mioty.FormatEUI64(*targetBsEui), ErrBaseStationUnavailable)
	}

	// Otherwise, find all suitable sessions within the requesting tenant and select deterministically
	// SCACI §3.9.1 Production Readiness: Go map iteration is non-deterministic.
	// Collect candidates and select by lowest EUI for reproducible behavior.
	var candidateSid string
	var candidateBsEui uint64
	for _, session := range live {
		sid := session.ID
		if session.HandshakeComplete &&
			session.Bidirectional &&
			session.ResolvedTenantID == tenantID {
			// First candidate or lower EUI wins (deterministic fallback selection)
			if candidateSid == "" || session.BaseStationEUI < candidateBsEui {
				candidateSid = sid
				candidateBsEui = session.BaseStationEUI
			}
		}
	}

	if candidateSid != "" {
		return candidateSid, candidateBsEui, nil
	}

	// No suitable sessions found for this tenant - return sentinel directly for errors.Is() detection
	return "", 0, ErrNoBidirectionalBaseStations
}

// FindSessionForEndpointAttachment implements SessionDirectory.FindSessionForEndpointAttachment
//
// This method supports roaming scenarios by locating a base station session without
// enforcing tenant ownership. The caller MUST validate endpoint ownership before calling.
//
// SECURITY: This method assumes the caller (e.g., QueryDLRXStatus, which locates the endpoint under the requesting tenant)
// has already validated that the requesting tenant owns the endpoint. It finds the base station
// session regardless of which tenant owns the base station, enabling roaming support:
// "All base stations on a server form a shared RF mesh. Any station accepts traffic from any endpoint."
//
// Validates:
//   - HandshakeComplete (BSSCI §3.3 requirement)
//   - Bidirectional capability (DL operation requirement)
//   - Session exists
//
// Returns:
//   - sessionID: Internal session identifier
//   - error: ErrSessionNotFound, ErrSessionNotReady, ErrSessionNotBidirectional
func (s *Server) FindSessionForEndpointAttachment(bsEui uint64) (sessionID string, err error) {
	// Search for session with matching BS EUI
	for _, session := range s.sessions.snapshot() {
		sid := session.ID
		if session.BaseStationEUI == bsEui {
			// Validate session handshake is complete per BSSCI §3.3
			if !session.HandshakeComplete {
				ctx := s.sessionContext(session)
				s.logger.WarnContext(ctx, LogBSSCISessionNotReady,
					logger.FieldSessionID, sid,
					logger.FieldBsEui, mioty.FormatEUI64(bsEui),
					logger.FieldReason, reasonHandshakeIncomplete)
				return "", fmt.Errorf("%s %s: %w",
					ResolveErrorMessage(errSessionNotReady), mioty.FormatEUI64(bsEui), ErrSessionNotReady)
			}

			// Validate session supports bidirectional operations
			if !session.Bidirectional {
				ctx := s.sessionContext(session)
				s.logger.WarnContext(ctx, LogBSSCISessionNotBidirectional,
					logger.FieldSessionID, sid,
					logger.FieldBsEui, mioty.FormatEUI64(bsEui),
					logger.FieldBidirectional, session.Bidirectional)
				return "", fmt.Errorf("%s %s: %w",
					ResolveErrorMessage(errSessionNotBidirectional), mioty.FormatEUI64(bsEui), ErrSessionNotBidirectional)
			}

			// Session is suitable for DL operations
			return sid, nil
		}
	}

	// Base station not found in connected sessions
	s.logger.DebugContext(s.ctx, LogBSSCISessionNotFound,
		logger.FieldBsEui, mioty.FormatEUI64(bsEui),
		logger.FieldReason, reasonNotConnected)
	return "", fmt.Errorf("%s %s: %w",
		ResolveErrorMessage(errSessionNotFound), mioty.FormatEUI64(bsEui), ErrSessionNotFound)
}
