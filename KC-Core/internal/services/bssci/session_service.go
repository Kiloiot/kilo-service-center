package bssciservices

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

type sessionService struct {
	sessionsByUUID   map[string]*bssci.Session               // REAL map
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

// HandleResume evaluates a session resume request per BSSCI §5.3.1 against the
// DB-authoritative resumable-session lookup and returns a typed outcome. Resume
// identity is snBsUuid scoped by tenant and base station EUI. The optional
// snBsOpId (minimum required known BS operation ID) and snScOpId (maximum known
// SC operation ID) constraints are asserted only when present. A lookup failure
// yields ResumeInfrastructureFailure (reject the connect, never a silent fresh
// session); incompatible counters or negotiated version yield ResumeInconsistent
// (terminate the stale session, then fresh); a clean match yields
// ResumeCompatible with the hydrated session, which is NOT published into any
// live registry here - the caller activates it after conCmp.
func (s *sessionService) HandleResume(ctx context.Context, session *bssci.Session, bsUUID []byte, bsOpId, scOpId *int64, bsEUI uint64) bssci.ResumeOutcome {
	if len(bsUUID) != 16 {
		return bssci.ResumeOutcome{Disposition: bssci.ResumeNoMatch}
	}

	// DB-authoritative lookup: only a disconnected, resumable session scoped
	// by tenant, base station EUI, and snBsUuid qualifies (BSSCI §5.3.1). The
	// in-memory registry is never consulted for resume - an active session
	// must never be handed out as resumable.
	var bsUUIDArray [16]byte
	copy(bsUUIDArray[:], bsUUID)
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, bsEUI)

	dbSession, err := s.bsSessionRepo.FindResumableSession(ctx, s.resolvedTenantID(session), euiBytes, bsUUIDArray)
	if err != nil {
		// A lookup failure must not degrade into a fresh session while the
		// old resumable state lingers; reject the connect instead.
		return bssci.ResumeOutcome{
			Disposition: bssci.ResumeInfrastructureFailure,
			Err:         fmt.Errorf("resumable session lookup failed: %w", err),
		}
	}
	if dbSession == nil {
		return bssci.ResumeOutcome{Disposition: bssci.ResumeNoMatch}
	}

	restoredSession := s.hydrateSessionFromDB(dbSession, bsEUI)

	if !s.resumeCountersConsistent(ctx, bsEUI, bsOpId, scOpId, dbSession.SnBsOpId, dbSession.SnScOpId) {
		return bssci.ResumeOutcome{
			Disposition: bssci.ResumeInconsistent,
			Previous:    restoredSession,
			Err:         bssci.ErrResumeCounterMismatch,
		}
	}

	// A resumable session's persisted negotiated version must be compatible
	// with the version selected for this connection (BSSCI rev1 §4.3: patch
	// differences are compatible; major/minor must match).
	if !resumeVersionCompatible(restoredSession.NegotiatedVersion, session.NegotiatedVersion) {
		s.logger.WarnContext(ctx, bssci.LogBSSCIResumeRejectedVersionIncompatible,
			"bsEui", bsEUI,
			"persistedVersion", restoredSession.NegotiatedVersion,
			"selectedVersion", session.NegotiatedVersion)
		return bssci.ResumeOutcome{
			Disposition: bssci.ResumeInconsistent,
			Previous:    restoredSession,
			Err:         bssci.ErrResumeCounterMismatch,
		}
	}

	// The hydrated session is NOT published into any live registry here; the
	// caller activates it after conCmp.
	return bssci.ResumeOutcome{Disposition: bssci.ResumeCompatible, Previous: restoredSession}
}

// resumeVersionCompatible reports whether a persisted negotiated version can
// resume under a newly selected version. An empty version on either side
// (legacy rows, or a caller that has not recorded the negotiated version)
// cannot assert incompatibility and is treated as compatible; otherwise the
// major and minor components must match (BSSCI rev1 §4.3 - patch differences
// are compatible).
func resumeVersionCompatible(persisted, selected string) bool {
	if persisted == "" || selected == "" || persisted == selected {
		return true
	}
	pMaj, pMin, pOK := majorMinor(persisted)
	sMaj, sMin, sOK := majorMinor(selected)
	if !pOK || !sOK {
		return false
	}
	return pMaj == sMaj && pMin == sMin
}

// majorMinor parses the major and minor components of a "major.minor.patch"
// version string.
func majorMinor(v string) (string, string, bool) {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// resumeCountersConsistent applies the BSSCI §5.3.1 resume constraints
// against persisted counters. Absent constraints (nil) are not asserted;
// equality is always valid.
func (s *sessionService) resumeCountersConsistent(ctx context.Context, bsEUI uint64, bsOpId, scOpId *int64, knownBsOpId, knownScOpId int64) bool {
	if bsOpId != nil && *bsOpId > knownBsOpId {
		// The BS requires a minimum operation state the SC does not know
		s.logger.WarnContext(ctx, bssci.LogBSSCIResumeRejectedBsOpIDBeyondPersisted,
			"bsEui", bsEUI,
			"requiredBsOpId", *bsOpId,
			"persistedBsOpId", knownBsOpId)
		return false
	}
	if scOpId != nil && *scOpId < knownScOpId {
		// The BS claims a more negative SC operation ID than the SC issued
		s.logger.WarnContext(ctx, bssci.LogBSSCIResumeRejectedScOpIDBeyondIssued,
			"bsEui", bsEUI,
			"claimedScOpId", *scOpId,
			"issuedScOpId", knownScOpId)
		return false
	}
	if scOpId != nil && *scOpId != knownScOpId {
		s.logger.WarnContext(ctx, bssci.LogBSSCIResumeAcceptedStaleBsCounter,
			"bsEui", bsEUI,
			"bsReportedScOpId", *scOpId,
			"scAuthoritativeOpId", knownScOpId)
	}
	return true
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

	// Handle ProtocolVersion hydration (BSSCI §4-4.5). A legacy NULL stays
	// empty: resumeVersionCompatible treats it as "cannot assert
	// incompatibility" and the resume activation persists the newly selected
	// version, backfilling the row instead of fabricating a version here.
	var negotiatedVersion, clientVersion string
	if dbSession.ProtocolVersion != nil {
		negotiatedVersion = *dbSession.ProtocolVersion
		clientVersion = *dbSession.ProtocolVersion // Initially same as negotiated
	}

	return &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
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
		},
		// Transport fields (Conn, Connected, LastSeen, etc.) initialized by caller (handleConnect)
	}
}

// PersistSession writes to basestation_sessions table using repository interface
// Refactored from raw SQL to use BaseStationSessionRepository
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
			// A failure to retire the stale session must abort activation:
			// creating a new session on top would leave two active sessions.
			if err := s.bsSessionRepo.TerminateSession(ctx, s.resolvedTenantID(session), dbSession.ID); err != nil {
				s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToTerminateStaleSession,
					"error", err,
					"staleSessionID", dbSession.ID,
					"baseStationEui", session.BaseStationEUI)
				return fmt.Errorf("terminate stale session before activation: %w", err)
			}
			s.logger.InfoContext(ctx, bssci.LogBSSCITerminatedStaleSession,
				"staleSessionID", dbSession.ID,
				"baseStationEui", session.BaseStationEUI)
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
			s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDatabaseSession,
				"error", err,
				"sessionID", session.DbSessionID)
			return fmt.Errorf("failed to initialize session counters: %w", err)
		}

		// Convert bs.EUI ([8]byte) to uint64 for logging
		hexEUI := binary.BigEndian.Uint64(baseStation.EUI[:])
		s.logger.InfoContext(ctx, bssci.LogBSSCIDatabaseSessionCreated,
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
			s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDatabaseSession,
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

		s.logger.InfoContext(ctx, bssci.LogBSSCIDatabaseSessionUpdated,
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
				s.logger.WarnContext(ctx, bssci.LogBSSCIInvalidEncodingInDatabase,
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

		// Resume activation is atomic: active status, resumability, connection
		// metadata, encoding, negotiated version, organization, and a cleared
		// end timestamp land in one update
		negotiated := session.NegotiatedVersion
		activeStatus := models.SessionStatusActive
		canResume := true
		connectionID := session.ID
		updateReq := &models.BaseStationSessionUpdateRequest{
			Status:          &activeStatus,
			CanResume:       &canResume,
			ClearEndedAt:    true,
			SnBsOpId:        &session.LastBsOpId,
			SnScOpId:        &session.LastScOpId,
			ConnectionId:    &connectionID,
			RemoteAddr:      remoteAddr,
			Encoding:        &session.Encoding,
			ProtocolVersion: &negotiated, // BSSCI §4-4.5: persist negotiated version
		}
		if session.OrganizationID != uuid.Nil {
			orgID := session.OrganizationID
			updateReq.OrganizationID = &orgID
		}

		err = s.bsSessionRepo.UpdateSession(ctx, s.resolvedTenantID(session), dbSession.ID, updateReq)
		if err != nil {
			s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDatabaseSession,
				"error", err,
				"baseStationEui", session.BaseStationEUI)
			return err
		}

		session.DbSessionID = dbSession.ID
		s.logger.InfoContext(ctx, bssci.LogBSSCIDatabaseSessionUpdated,
			"dbSessionID", session.DbSessionID,
			"baseStationEui", session.BaseStationEUI)
	}

	return nil
}

// StoreSessionByUUID adds session to sessionsByUUID map
// Real map storage
func (s *sessionService) StoreSessionByUUID(session *bssci.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	uuidKey := string(session.SessionUUID)
	s.sessionsByUUID[uuidKey] = session
}

// MarkHandshakeComplete sets HandshakeComplete=true
// Real handshake marker
func (s *sessionService) MarkHandshakeComplete(session *bssci.Session) {
	// BSSCI-3.3-03: Mark handshake as complete BEFORE reissuing any pending operations
	// This allows resumed operations to proceed without being blocked
	session.HandshakeComplete = true
}

// MarkDisconnected marks an active session disconnected and resumable after
// unexpected connection loss. The repository update is guarded by the
// session's connection ID: if a reconnect already replaced this connection,
// zero rows match and the newer session stays active.
func (s *sessionService) MarkDisconnected(ctx context.Context, session *bssci.Session) error {
	if session.DbSessionID == 0 {
		return fmt.Errorf("cannot mark session disconnected: not persisted (DbSessionID=0)")
	}
	err := s.bsSessionRepo.MarkDisconnected(ctx, s.resolvedTenantID(session), session.DbSessionID, session.ID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark session disconnected: %w", err)
	}
	return nil
}

// RemoveSession cleans sessionsByUUID map on disconnect
// Real cleanup
func (s *sessionService) RemoveSession(session *bssci.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Guard against empty UUID
	if len(session.SessionUUID) == 0 {
		return
	}

	uuidKey := string(session.SessionUUID)
	// Connection-identity guard: a reconnect may have already replaced this
	// UUID with a newer live session. Remove only when the stored entry is
	// still this exact connection, so connection A's cleanup cannot evict
	// connection B.
	if stored, ok := s.sessionsByUUID[uuidKey]; ok && stored.ID == session.ID {
		delete(s.sessionsByUUID, uuidKey)
	}
}

// UpdateEncoding persists the negotiated message encoding to the database
// Called when encoding is detected on first message per BSSCI Section 1
func (s *sessionService) UpdateEncoding(ctx context.Context, tenantID, sessionID int64, encoding string) error {
	return s.bsSessionRepo.UpdateEncoding(ctx, tenantID, sessionID, encoding)
}

// UpdateSessionCounters persists operation ID counters to database (BSSCI §5.2).
// Called immediately after successful SC-initiated operations to ensure resume correctness.
func (s *sessionService) UpdateSessionCounters(ctx context.Context, session *bssci.Session) error {
	if session.DbSessionID == 0 {
		return fmt.Errorf("cannot update counters: session not persisted (DbSessionID=0)")
	}

	// Uses the fixed atomic UpdateOperationIDs statement (sets updated_at and
	// errors when the session row is missing) so every counter-persistence
	// path shares one durable semantics: a zero-row update surfaces as an
	// error instead of silent success on an inconsistent session.
	err := s.bsSessionRepo.UpdateOperationIDs(ctx, s.resolvedTenantID(session), session.DbSessionID,
		session.LastBsOpId, session.LastScOpId)
	if err != nil {
		s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDatabaseSession,
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
		s.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToTerminateSession,
			"error", err,
			"sessionID", session.DbSessionID,
			"eui", session.BaseStationEUI)
		return fmt.Errorf("failed to terminate session: %w", err)
	}

	return nil
}
