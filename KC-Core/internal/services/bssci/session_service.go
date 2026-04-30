package bssciservices

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/basestation"
	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
)

type sessionService struct {
	sessionsByUUID   map[string]*bssci.Session               // REAL map from server.go:84
	bsSessionRepo    interfaces.BaseStationSessionRepository // Repository interface for session persistence
	bsRepo           interfaces.BaseStationRepository        // Base station repository
	systemEventStore interfaces.SystemEventStore             // System event store (injected separately, not from storage)
	tenantID         int64
	mu               sync.RWMutex
	logger           logger.Logger
}

// NewSessionService creates a new session service with repository-based persistence
func NewSessionService(
	bsSessionRepo interfaces.BaseStationSessionRepository,
	bsRepo interfaces.BaseStationRepository,
	systemEventStore interfaces.SystemEventStore,
	tenantID int64,
	log logger.Logger,
) bssci.SessionService {
	return &sessionService{
		sessionsByUUID:   make(map[string]*bssci.Session),
		bsSessionRepo:    bsSessionRepo,
		bsRepo:           bsRepo,
		systemEventStore: systemEventStore,
		tenantID:         tenantID,
		logger:           log,
	}
}

// resolvedTenantID returns the session's resolved tenant with fallback to service tenant.
// Named distinctly from the package-level resolvedTenant() to avoid shadowing.
func (s *sessionService) resolvedTenantID(session *bssci.Session) int64 {
	if session != nil && session.ResolvedTenantID > 0 {
		return session.ResolvedTenantID
	}
	return s.tenantID // CRITICAL: Use s.tenantID, not s.defaultTenantID
}

// ValidateVersion checks protocol version compatibility (BSSCI 2.1-2.2)
// Returns specific CatalogError tokens per BSSCI 2.1-2.2.
func (s *sessionService) ValidateVersion(version string) error {
	// Parse base station version
	bsMajor, bsMinor, _, cerr := bssci.ParseVersion(version)
	if cerr != nil {
		// Pass through specific error token from parseVersion
		// (errInvalidVersionFormat, errInvalidMajorVersion, errInvalidMinorVersion, errInvalidPatchVersion)
		return cerr
	}

	// Parse server version
	scMajor, scMinor, _, cerr := bssci.ParseVersion(mioty.MIOTYProtocolVersion)
	if cerr != nil {
		// This should never happen unless MIOTYProtocolVersion is misconfigured
		s.logger.Error(bssci.LogBSSCIInvalidServerProtocolVersion, "version", mioty.MIOTYProtocolVersion)
		return bssci.NewCatalogError(bssci.ErrVersionIncompatible, bssci.POSIX_EPROTO)
	}

	// Major version must match exactly (BSSCI 2.1)
	if bsMajor != scMajor {
		s.logger.Warn(bssci.LogBSSCIVersionIncompatible,
			"bsVersion", version,
			"bsMajor", bsMajor,
			"scMajor", scMajor)
		return bssci.NewCatalogError(bssci.ErrUnsupportedMajorVersion, bssci.POSIX_EPROTO)
	}

	// Minor version must match for compatibility (BSSCI 2.2)
	// Different minor versions should terminate the connection
	if bsMinor != scMinor {
		s.logger.Warn(bssci.LogBSSCIMinorVersionMismatch,
			"bsMinor", bsMinor,
			"scMinor", scMinor,
			"bsVersion", version,
			"scVersion", mioty.MIOTYProtocolVersion)
		return bssci.NewCatalogError(bssci.ErrUnsupportedMinorVersion, bssci.POSIX_EPROTO)
	}

	return nil
}

// HandleResume checks sessionsByUUID and validates counters
// EXACT resume logic from server.go:694-750
// Session parameter provides tenant context via session.ResolvedTenantID for DB queries
func (s *sessionService) HandleResume(session *bssci.Session, bsUUID []byte, scUUIDToMatch []byte, bsOpId, scOpId int64, bsEUI uint64) (*bssci.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Look for an existing session with this BS EUI and BS UUID
	// EXACT logic from server.go:696-722
	for scUUIDStr, sess := range s.sessionsByUUID {
		// Check if this is the right session:
		// 1. Same base station EUI
		// 2. Same base station UUID (if we stored it)
		// 3. If SC UUID was provided, it should match
		if sess.BaseStationEUI == bsEUI {
			if scUUIDToMatch != nil {
				// BS provided the SC UUID - must match exactly (server.go:703-708)
				if string(sess.SessionUUID) == string(scUUIDToMatch) {
					// Validate counters per BSSCI §5.2
					// BS counter must not move backwards (monotonicity)
					if bsOpId < sess.LastBsOpId {
						s.logger.Warn("Resume rejected: BS operation counter moved backwards (in-memory)",
							"bsEui", bsEUI,
							"providedBsOpId", bsOpId,
							"expectedBsOpId", sess.LastBsOpId,
							"scUuid", fmt.Sprintf("%x", scUUIDToMatch))
						continue // Try next session
					}
					// SC counter validation: SC is authoritative (BSSCI §3.2)
					// Accept if BS's scOpId >= sess.LastScOpId (stale or matches)
					if scOpId < sess.LastScOpId {
						s.logger.Warn("Resume rejected: BS claims SC counter beyond in-memory value",
							"bsEui", bsEUI,
							"providedScOpId", scOpId,
							"expectedScOpId", sess.LastScOpId,
							"scUuid", fmt.Sprintf("%x", scUUIDToMatch))
						continue // Try next session
					}
					if scOpId != sess.LastScOpId {
						s.logger.Warn("Resume accepted with stale BS counter (in-memory, SC authoritative)",
							"bsEui", bsEUI,
							"bsReportedScOpId", scOpId,
							"scAuthoritativeOpId", sess.LastScOpId,
							"scUuid", fmt.Sprintf("%x", scUUIDToMatch))
					}
					return sess, nil
				}
			} else if sess.BsUUID != nil && string(sess.BsUUID) == string(bsUUID) {
				// No SC UUID provided, but BS UUID matches (server.go:709-712)
				// Validate counters per BSSCI §5.2
				if bsOpId < sess.LastBsOpId {
					s.logger.Warn("Resume rejected: BS operation counter moved backwards (in-memory)",
						"bsEui", bsEUI,
						"providedBsOpId", bsOpId,
						"expectedBsOpId", sess.LastBsOpId,
						"bsUuid", fmt.Sprintf("%x", bsUUID))
					continue // Try next session
				}
				// SC counter validation: SC is authoritative (BSSCI §3.2)
				if scOpId < sess.LastScOpId {
					s.logger.Warn("Resume rejected: BS claims SC counter beyond in-memory value",
						"bsEui", bsEUI,
						"providedScOpId", scOpId,
						"expectedScOpId", sess.LastScOpId,
						"bsUuid", fmt.Sprintf("%x", bsUUID))
					continue // Try next session
				}
				if scOpId != sess.LastScOpId {
					s.logger.Warn("Resume accepted with stale BS counter (in-memory, SC authoritative)",
						"bsEui", bsEUI,
						"bsReportedScOpId", scOpId,
						"scAuthoritativeOpId", sess.LastScOpId,
						"bsUuid", fmt.Sprintf("%x", bsUUID))
				}
				return sess, nil
			} else if sess.BsUUID == nil {
				// Older session without BS UUID stored, use SC UUID as key (server.go:713-719)
				if scUUIDStr != "" {
					// Validate counters per BSSCI §5.2
					if bsOpId < sess.LastBsOpId {
						s.logger.Warn("Resume rejected: BS operation counter moved backwards (in-memory)",
							"bsEui", bsEUI,
							"providedBsOpId", bsOpId,
							"expectedBsOpId", sess.LastBsOpId)
						continue // Try next session
					}
					// SC counter validation: SC is authoritative (BSSCI §3.2)
					if scOpId < sess.LastScOpId {
						s.logger.Warn("Resume rejected: BS claims SC counter beyond in-memory value",
							"bsEui", bsEUI,
							"providedScOpId", scOpId,
							"expectedScOpId", sess.LastScOpId)
						continue // Try next session
					}
					if scOpId != sess.LastScOpId {
						s.logger.Warn("Resume accepted with stale BS counter (in-memory, SC authoritative)",
							"bsEui", bsEUI,
							"bsReportedScOpId", scOpId,
							"scAuthoritativeOpId", sess.LastScOpId)
					}
					return sess, nil
				}
			}
		}
	}

	// Database fallback: try to restore session from persistence layer
	// BSSCI §5.3/§5.3.1: session should be resumable across disconnects
	ctx := context.Background() // No request context available in HandleResume

	// Try GetSessionByScUuid first (primary lookup)
	if len(scUUIDToMatch) == 16 {
		var scUUIDArray [16]byte
		copy(scUUIDArray[:], scUUIDToMatch)

		dbSession, err := s.bsSessionRepo.GetSessionByScUUID(ctx, s.resolvedTenantID(session), scUUIDArray)
		if err == nil && dbSession != nil {
			// Validate operation counters per BSSCI §5.2
			// BS counter must not move backwards (equality allowed for idempotent resume)
			if bsOpId < dbSession.SnBsOpId {
				s.logger.WarnContext(ctx, "Resume rejected: BS operation counter moved backwards",
					"bsEui", bsEUI,
					"providedBsOpId", bsOpId,
					"expectedBsOpId", dbSession.SnBsOpId,
					"scUuid", fmt.Sprintf("%x", scUUIDArray))
				return nil, bssci.ErrResumeCounterMismatch
			}
			// SC counter validation: SC is authoritative for its own operation IDs (BSSCI §3.2)
			// BS may have a stale (less negative) counter from a crash before SC persisted.
			// Accept if BS's reported scOpId >= dbSession.SnScOpId (i.e., BS is stale or matches).
			// Reject only if BS claims a more negative value than SC has (impossible without tampering).
			if scOpId < dbSession.SnScOpId {
				// BS claims to have seen a more negative SC opId than SC ever issued - reject
				s.logger.WarnContext(ctx, "Resume rejected: BS claims SC counter beyond DB value",
					"bsEui", bsEUI,
					"providedScOpId", scOpId,
					"expectedScOpId", dbSession.SnScOpId,
					"scUuid", fmt.Sprintf("%x", scUUIDArray))
				return nil, bssci.ErrResumeCounterMismatch
			}
			if scOpId != dbSession.SnScOpId {
				// BS has a stale counter - log but accept (SC will continue from its authoritative value)
				s.logger.WarnContext(ctx, "Resume accepted with stale BS counter (SC is authoritative)",
					"bsEui", bsEUI,
					"bsReportedScOpId", scOpId,
					"scAuthoritativeOpId", dbSession.SnScOpId,
					"scUuid", fmt.Sprintf("%x", scUUIDArray))
			}

			// Counters valid - hydrate and return
			restoredSession := s.hydrateSessionFromDB(dbSession, bsEUI)

			// Re-populate in-memory map (requires lock upgrade)
			s.mu.RUnlock()
			s.mu.Lock()
			uuidKey := string(restoredSession.SessionUUID)
			s.sessionsByUUID[uuidKey] = restoredSession
			s.mu.Unlock()
			s.mu.RLock() // Restore read lock for deferred RUnlock

			return restoredSession, nil
		}
	}

	// Try GetSessionByBsUuid fallback (secondary lookup)
	if len(bsUUID) == 16 {
		var bsUUIDArray [16]byte
		copy(bsUUIDArray[:], bsUUID)

		dbSession, err := s.bsSessionRepo.GetSessionByBsUUID(ctx, s.resolvedTenantID(session), bsUUIDArray)
		if err == nil && dbSession != nil {
			// Validate operation counters per BSSCI §5.2
			// BS counter must not move backwards (equality allowed for idempotent resume)
			if bsOpId < dbSession.SnBsOpId {
				s.logger.WarnContext(ctx, "Resume rejected: BS operation counter moved backwards",
					"bsEui", bsEUI,
					"providedBsOpId", bsOpId,
					"expectedBsOpId", dbSession.SnBsOpId,
					"bsUuid", fmt.Sprintf("%x", bsUUIDArray))
				return nil, bssci.ErrResumeCounterMismatch
			}
			// SC counter validation: SC is authoritative for its own operation IDs (BSSCI §3.2)
			// BS may have a stale (less negative) counter from a crash before SC persisted.
			// Accept if BS's reported scOpId >= dbSession.SnScOpId (i.e., BS is stale or matches).
			// Reject only if BS claims a more negative value than SC has (impossible without tampering).
			if scOpId < dbSession.SnScOpId {
				// BS claims to have seen a more negative SC opId than SC ever issued - reject
				s.logger.WarnContext(ctx, "Resume rejected: BS claims SC counter beyond DB value",
					"bsEui", bsEUI,
					"providedScOpId", scOpId,
					"expectedScOpId", dbSession.SnScOpId,
					"bsUuid", fmt.Sprintf("%x", bsUUIDArray))
				return nil, bssci.ErrResumeCounterMismatch
			}
			if scOpId != dbSession.SnScOpId {
				// BS has a stale counter - log but accept (SC will continue from its authoritative value)
				s.logger.WarnContext(ctx, "Resume accepted with stale BS counter (SC is authoritative)",
					"bsEui", bsEUI,
					"bsReportedScOpId", scOpId,
					"scAuthoritativeOpId", dbSession.SnScOpId,
					"bsUuid", fmt.Sprintf("%x", bsUUIDArray))
			}

			// Counters valid - hydrate and return
			restoredSession := s.hydrateSessionFromDB(dbSession, bsEUI)

			// Re-populate in-memory map (requires lock upgrade)
			s.mu.RUnlock()
			s.mu.Lock()
			uuidKey := string(restoredSession.SessionUUID)
			s.sessionsByUUID[uuidKey] = restoredSession
			s.mu.Unlock()
			s.mu.RLock() // Restore read lock for deferred RUnlock

			return restoredSession, nil
		}
	}

	return nil, nil // No DB match - fallback to fresh session
}

// hydrateSessionFromDB creates a bssci.Session from a database record
// This helper centralizes field mapping: models.BaseStationSession → bssci.Session
// BSSCI §5.3.1: Restored session must preserve UUIDs and operation ID counters
func (s *sessionService) hydrateSessionFromDB(dbSession *models.BaseStationSession, bsEUI uint64) *bssci.Session {
	// Handle OrganizationID pointer → value conversion (nil-safe)
	var orgID uuid.UUID
	if dbSession.OrganizationID != nil {
		orgID = *dbSession.OrganizationID
	} else {
		orgID = uuid.Nil // Safe default when pointer is nil
	}

	// Handle ProtocolVersion hydration (BSSCI §4-4.5)
	var negotiatedVersion, clientVersion string
	if dbSession.ProtocolVersion != nil {
		negotiatedVersion = *dbSession.ProtocolVersion
		clientVersion = *dbSession.ProtocolVersion // Initially same as negotiated
	} else {
		// Fallback for old sessions without stored version
		negotiatedVersion = mioty.MIOTYProtocolVersion
		clientVersion = mioty.MIOTYProtocolVersion
	}

	return &bssci.Session{
		DbSessionID:       dbSession.ID,          // int64 → int64
		SessionUUID:       dbSession.SnScUuid[:], // [16]byte → []byte
		BsUUID:            dbSession.SnBsUuid[:], // [16]byte → []byte
		LastBsOpId:        dbSession.SnBsOpId,    // int64 → int64 (BSSCI §3.2)
		LastScOpId:        dbSession.SnScOpId,    // int64 → int64 (BSSCI §3.2)
		Encoding:          dbSession.Encoding,    // string → string (BSSCI §1)
		ClientVersion:     clientVersion,         // string → string (BSSCI §4-4.5)
		NegotiatedVersion: negotiatedVersion,     // string → string (BSSCI §4-4.5)
		OrganizationID:    orgID,                 // *uuid.UUID → uuid.UUID (nil-safe)
		ResolvedTenantID:  dbSession.TenantID,    // int64 → int64
		BaseStationEUI:    bsEUI,                 // From parameter (validated by caller)
		IsResumed:         true,                  // Mark as resumed session (BSSCI §5.3.2)
		// Lifecycle fields (ID, Conn, Connected, LastSeen, etc.) initialized by caller (handleConnect)
		// ActiveVMTypes, stopStatus channel created by caller as needed
	}
}

// PersistSession writes to basestation_sessions table using repository interface
// Refactored from raw SQL (server.go:852-950) to use BaseStationSessionRepository
// TODO: Full implementation pending - repository methods need to support ON CONFLICT logic
func (s *sessionService) PersistSession(ctx context.Context, session *bssci.Session, baseStation *basestation.BaseStation, isResume bool, connectInfo json.RawMessage) error {
	// BSSCI §3.3.1: Connect info overwritten on each resume (user decision)
	// Only update connect_info when provided - avoid NULL overwrites
	if connectInfo != nil {
		session.ConnectInfo = connectInfo
	}
	// else leave session.ConnectInfo untouched - preserves DB-loaded value on resume without connect data

	// Get remote address from connection
	var remoteAddr *string
	if session.Conn != nil {
		if addr := session.Conn.RemoteAddr(); addr != nil {
			var addrStr string
			if tcpAddr, ok := addr.(*net.TCPAddr); ok {
				addrStr = tcpAddr.IP.String()
			} else {
				addrStr = addr.String()
			}
			remoteAddr = &addrStr
		}
	}

	if !isResume {
		// New session - terminate any stale session first per BSSCI §3 "new session starts, discarding state"
		dbSession, err := s.bsSessionRepo.GetActiveSessionByBaseStation(ctx, s.resolvedTenantID(session), baseStation.ID)
		if err == nil && dbSession != nil {
			// Terminate stale session before creating new one
			if err := s.bsSessionRepo.TerminateSession(ctx, s.resolvedTenantID(session), dbSession.ID); err != nil {
				s.logger.Error(bssci.LogBSSCIFailedToTerminateStaleSession,
					"error", err,
					"staleSessionID", dbSession.ID,
					"baseStationEui", session.BaseStationEUI)
				// Continue with new session creation despite termination error
			} else {
				s.logger.Info(bssci.LogBSSCITerminatedStaleSession,
					"staleSessionID", dbSession.ID,
					"baseStationEui", session.BaseStationEUI)
			}
		}

		// No existing session (or terminated) - create new one via repository

		// Convert UUIDs from []byte to [16]byte
		var snBsUUID, snScUUID [16]byte
		if len(session.BsUUID) == 16 {
			copy(snBsUUID[:], session.BsUUID)
		}
		if len(session.SessionUUID) == 16 {
			copy(snScUUID[:], session.SessionUUID)
		}

		// Assign to local variable for pointer safety (avoid loop variable address)
		connID := session.ID

		// Prepare organization ID pointer (nil when orgID is uuid.Nil to avoid NOT NULL violation)
		var orgIDPtr *uuid.UUID
		if session.OrganizationID != uuid.Nil {
			orgIDPtr = &session.OrganizationID
		}

		// Create local variable for ProtocolVersion pointer (pointer safety)
		negotiated := session.NegotiatedVersion

		// Prepare ConnectInfo for persistence
		var connectInfoNullJSON models.NullJSON
		if len(connectInfo) > 0 {
			connectInfoNullJSON = models.NullJSON{
				Valid: true,
				Data:  connectInfo,
			}
		} else {
			// BSSCI §5.3: Use empty object when info field absent (matches DB default)
			connectInfoNullJSON = models.NullJSON{
				Valid: true,
				Data:  json.RawMessage("{}"),
			}
		}

		// Create new session request
		req := &models.BaseStationSessionCreateRequest{
			BaseStationID:   baseStation.ID,
			TenantID:        s.resolvedTenantID(session),
			SnBsUuid:        snBsUUID,
			SnScUuid:        snScUUID,
			ConnectionId:    &connID,
			RemoteAddr:      remoteAddr,
			CanResume:       true,
			Encoding:        session.Encoding,    // BSSCI Section 1: persist negotiated encoding
			ProtocolVersion: &negotiated,         // BSSCI §4-4.5: persist negotiated protocol version
			ConnectInfo:     connectInfoNullJSON, // BSSCI §5.3: persist connect metadata
			OrganizationID:  orgIDPtr,
		}

		// Persist via repository
		dbSession, err = s.bsSessionRepo.CreateSession(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to create session in database: %w", err)
		}

		// Update in-memory session
		session.DbSessionID = dbSession.ID
		if dbSession.ConnectInfo.Valid {
			session.ConnectInfo = dbSession.ConnectInfo.Data
		}
		// Don't set HandshakeComplete here - it's set in handleConnectComplete:1076

		// Initialize per-session SC operation ID counter to 0.
		// First SC operation will use opId -1 after decrement.
		// BSSCI §5.2: Each session maintains independent operation sequencing.
		session.LastScOpId = 0
		session.LastBsOpId = 0

		// Persist initial counter values to support resume (BSSCI §3.3)
		zero := int64(0)
		updateReq := &models.BaseStationSessionUpdateRequest{
			SnBsOpId: &zero,
			SnScOpId: &zero,
		}
		if err = s.bsSessionRepo.UpdateSession(ctx, s.resolvedTenantID(session), dbSession.ID, updateReq); err != nil {
			s.logger.Error(bssci.LogBSSCIFailedToUpdateDatabaseSession,
				"error", err,
				"sessionID", session.DbSessionID)
			return fmt.Errorf("failed to initialize session counters: %w", err)
		}

		// Convert bs.EUI ([8]byte) to uint64 for logging
		hexEUI := binary.BigEndian.Uint64(baseStation.EUI[:])
		s.logger.Info(bssci.LogBSSCIDatabaseSessionCreated,
			"sessionID", session.DbSessionID,
			"bsEui", fmt.Sprintf("%016X", hexEUI))

	} else {
		// Resumed session - update existing record via repository
		var scUUID [16]byte
		if len(session.SessionUUID) == 16 {
			copy(scUUID[:], session.SessionUUID)
		}

		dbSession, err := s.bsSessionRepo.GetSessionByScUUID(ctx, s.resolvedTenantID(session), scUUID)
		if err != nil {
			s.logger.Error(bssci.LogBSSCIFailedToUpdateDatabaseSession,
				"error", err,
				"baseStationEui", session.BaseStationEUI)
			return err
		}

		// Restore per-session operation ID counters from DB (BSSCI §5.2).
		// Continue from last persisted state, not from what BS reports.
		// Server is authoritative for SC operation IDs.
		session.LastScOpId = dbSession.SnScOpId
		session.LastBsOpId = dbSession.SnBsOpId
		session.DbSessionID = dbSession.ID
		if dbSession.ConnectInfo.Valid {
			session.ConnectInfo = dbSession.ConnectInfo.Data
		}

		s.logger.Info(bssci.LogBSSCIDatabaseSessionUpdated,
			"dbSessionID", session.DbSessionID,
			"baseStationEui", session.BaseStationEUI,
			"restoredScOpId", session.LastScOpId,
			"restoredBsOpId", session.LastBsOpId)

		// Preserve negotiated encoding over stale DB value (BSSCI Section 1)
		// Only restore from DB if encoding hasn't been negotiated yet
		if session.Encoding == "" {
			// Validate encoding from DB is legal per BSSCI §1
			if dbSession.Encoding == bssci.EncodingJSON || dbSession.Encoding == bssci.EncodingMessagePack {
				session.Encoding = dbSession.Encoding
			} else {
				// Invalid encoding in DB - fall back to BSSCI spec default (MessagePack)
				s.logger.Warn(bssci.LogBSSCIInvalidEncodingInDatabase,
					"dbEncoding", dbSession.Encoding,
					"sessionID", dbSession.ID)
				session.Encoding = bssci.EncodingMessagePack
			}
		}

		// Restore org/tenant from DB if not already set
		if session.OrganizationID == uuid.Nil && dbSession.OrganizationID != nil {
			session.OrganizationID = *dbSession.OrganizationID
		}
		if session.ResolvedTenantID == 0 {
			session.ResolvedTenantID = dbSession.TenantID
		}

		// Build update request (persist negotiated protocol version)
		negotiated := session.NegotiatedVersion
		activeStatus := models.SessionStatusActive // Transition from terminated to active on resume
		updateReq := &models.BaseStationSessionUpdateRequest{
			Status:          &activeStatus, // Fix: Update status from terminated to active on resume
			SnBsOpId:        &session.LastBsOpId,
			SnScOpId:        &session.LastScOpId,
			RemoteAddr:      remoteAddr,
			Encoding:        &session.Encoding, // Update encoding if it changed
			ProtocolVersion: &negotiated,       // BSSCI §4-4.5: persist negotiated version
		}

		err = s.bsSessionRepo.UpdateSession(ctx, s.resolvedTenantID(session), dbSession.ID, updateReq)
		if err != nil {
			s.logger.Error(bssci.LogBSSCIFailedToUpdateDatabaseSession,
				"error", err,
				"baseStationEui", session.BaseStationEUI)
			return err
		}

		session.DbSessionID = dbSession.ID
		s.logger.Info(bssci.LogBSSCIDatabaseSessionUpdated,
			"dbSessionID", session.DbSessionID,
			"baseStationEui", session.BaseStationEUI)
	}

	return nil
}

// StoreSessionByUUID adds session to sessionsByUUID map
// Real map storage from server.go:760-764
func (s *sessionService) StoreSessionByUUID(session *bssci.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	uuidKey := string(session.SessionUUID)
	s.sessionsByUUID[uuidKey] = session
}

// MarkHandshakeComplete sets HandshakeComplete=true
// Real handshake marker from server.go:1004
func (s *sessionService) MarkHandshakeComplete(session *bssci.Session) {
	// BSSCI-3.3-03: Mark handshake as complete BEFORE reissuing any pending operations
	// This allows resumed operations to proceed without being blocked
	session.HandshakeComplete = true
}

// RemoveSession cleans sessionsByUUID map on disconnect
// Real cleanup from server.go:438
func (s *sessionService) RemoveSession(session *bssci.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Guard against empty UUID
	if len(session.SessionUUID) == 0 {
		return
	}

	// Use same key format as StoreSessionByUUID (line 242)
	uuidKey := string(session.SessionUUID)
	delete(s.sessionsByUUID, uuidKey)
}

// UpdateEncoding persists the negotiated message encoding to the database
// Called when encoding is detected on first message per BSSCI Section 1
func (s *sessionService) UpdateEncoding(ctx context.Context, sessionID int64, encoding string) error {
	return s.bsSessionRepo.UpdateEncoding(ctx, sessionID, encoding)
}

// UpdateSessionCounters persists operation ID counters to database (BSSCI §5.2).
// Called immediately after successful SC-initiated operations to ensure resume correctness.
func (s *sessionService) UpdateSessionCounters(ctx context.Context, session *bssci.Session) error {
	if session.DbSessionID == 0 {
		return fmt.Errorf("cannot update counters: session not persisted (DbSessionID=0)")
	}

	updateReq := &models.BaseStationSessionUpdateRequest{
		SnBsOpId: &session.LastBsOpId,
		SnScOpId: &session.LastScOpId,
	}

	err := s.bsSessionRepo.UpdateSession(ctx, s.resolvedTenantID(session), session.DbSessionID, updateReq)
	if err != nil {
		s.logger.Error(bssci.LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", session.DbSessionID,
			"bsOpId", session.LastBsOpId,
			"scOpId", session.LastScOpId)
		return fmt.Errorf("failed to persist session counters: %w", err)
	}

	return nil
}

// UpdatePingTimestamp persists last ping timestamp to database (BSSCI §5.4)
// Called immediately after successful pingCmp reception to track base station health
func (s *sessionService) UpdatePingTimestamp(ctx context.Context, session *bssci.Session) error {
	if session.DbSessionID == 0 {
		return fmt.Errorf("cannot update ping timestamp: session not persisted (DbSessionID=0)")
	}

	now := time.Now()
	updateReq := &models.BaseStationSessionUpdateRequest{
		LastPingAt: &now,
	}

	err := s.bsSessionRepo.UpdateSession(ctx, s.resolvedTenantID(session), session.DbSessionID, updateReq)
	if err != nil {
		s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", session.DbSessionID,
			"timestamp", now)
		return fmt.Errorf("failed to persist ping timestamp: %w", err)
	}

	return nil
}

// TerminateSession marks a session as terminated in the database
// Called during disconnect cleanup per BSSCI §3 session lifecycle
func (s *sessionService) TerminateSession(ctx context.Context, session *bssci.Session) error {
	if session.DbSessionID == 0 {
		return fmt.Errorf("cannot terminate session: not persisted (DbSessionID=0)")
	}

	err := s.bsSessionRepo.TerminateSession(ctx, s.resolvedTenantID(session), session.DbSessionID)
	if err != nil {
		s.logger.Error(bssci.LogBSSCIFailedToTerminateSession,
			"error", err,
			"sessionID", session.DbSessionID,
			"eui", session.BaseStationEUI)
		return fmt.Errorf("failed to terminate session: %w", err)
	}

	return nil
}
