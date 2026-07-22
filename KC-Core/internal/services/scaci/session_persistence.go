package scaciservices

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

type sessionPersistence struct {
	sessionRepo interfaces.SCACISessionRepository // Already-injected repository interface
	logger      logger.Logger
}

// NewSessionPersistence creates a new session persistence service.
// Wraps existing repository with async persistence logic.
//
// Parameters:
//   - sessionRepo: Already-injected interfaces.SCACISessionRepository (no new KC-DB dependencies)
//   - logger: Logger instance for error tracking
func NewSessionPersistence(
	sessionRepo interfaces.SCACISessionRepository,
	log logger.Logger) scaci.SessionPersistence {
	return &sessionPersistence{
		sessionRepo: sessionRepo,
		logger:      log,
	}
}

// PersistConnectAsync handles async session creation/update after Connect handshake.
// Encapsulates the persistence pattern in a service (handlers orchestrate, services execute).
func (p *sessionPersistence) PersistConnectAsync(session *scaci.Session, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, negotiatedVersion string) {
	// CRITICAL: Deep-copy metadata BEFORE goroutine to prevent race with caller mutations.
	// This is the ONLY copy - do not call deepCopyMetadata again inside the goroutine.
	metadataCopy := deepCopyMetadata(session.Metadata)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), scaci.ConnectPersistTimeout)
		defer cancel()

		if session.Resumed && session.ID > 0 {
			// Update existing session (resumed connection)
			now := time.Now()
			status := "active"

			// Prepare TLS evidence pointers for resume per SCACI §1
			var tlsVer, cipherSt *string
			if tlsVersion != "" {
				tlsVer = &tlsVersion
			}
			if cipherSuite != "" {
				cipherSt = &cipherSuite
			}

			updateReq := &models.SCACISessionUpdateRequest{
				LastHeartbeat: &now,
				Status:        &status,
				TLSVersion:    tlsVer,       // Update TLS evidence on resume
				CipherSuite:   cipherSt,     // Update cipher suite on resume
				Metadata:      metadataCopy, // Use pre-copied metadata (no re-copy)
			}

			// Only persist opId counters if they're non-zero (preserve "opId 0 after connect")
			if session.AcOpIdCounter > 0 {
				updateReq.LastOpIDAc = &session.AcOpIdCounter
			}
			if session.ScOpIdCounter < 0 {
				updateReq.LastOpIDSc = &session.ScOpIdCounter
			}

			if err := p.sessionRepo.UpdateSession(ctx, session.TenantID, session.ID, updateReq); err != nil {
				p.logger.ErrorContext(ctx, scaci.LogSCACIUpdateSessionFailed, "error", err)
			}
		} else {
			// Create new session (fresh connection)
			var certFP, certSubj, remAddr, tlsVer, cipherSt *string
			if certFingerprint != "" {
				certFP = &certFingerprint
			}
			if certSubject != "" {
				certSubj = &certSubject
			}
			if remoteAddr != "" {
				remAddr = &remoteAddr
			}
			// TLS evidence per SCACI §1
			if tlsVersion != "" {
				tlsVer = &tlsVersion
			}
			if cipherSuite != "" {
				cipherSt = &cipherSuite
			}

			// Set organization ID if not nil
			var orgID *uuid.UUID
			if session.OrganizationID != uuid.Nil {
				orgID = &session.OrganizationID
			}

			// Set negotiated version - defaults to ProtocolVersionString if empty
			negVer := negotiatedVersion
			if negVer == "" {
				negVer = scaci.ProtocolVersionString
			}

			createReq := &models.SCACISessionCreateRequest{
				TenantID:               session.TenantID,
				OrganizationID:         orgID,
				AcEUI:                  session.GetAcEuiBytes(),
				SnAcUUID:               session.SnAcUUID,
				SnScUUID:               session.SnScUUID,
				CertificateFingerprint: certFP,
				ClientCertSubject:      certSubj,
				RemoteAddr:             remAddr,
				TLSVersion:             tlsVer,
				CipherSuite:            cipherSt,
				NegotiatedVersion:      negVer, // SCACI §§2.1-2.3: persist for resume validation
				CanResume:              true,
				Metadata:               metadataCopy, // Use pre-copied metadata (no re-copy)
			}

			if dbSession, err := p.sessionRepo.CreateSession(ctx, createReq); err != nil {
				p.logger.Error(scaci.LogSCACIPersistSessionFailed, "error", err)
			} else {
				// Store DB ID in runtime session
				// Note: This updates the session object passed by reference
				session.ID = dbSession.ID
			}
		}
	}()
}

// PersistConnectSync creates session synchronously, returns DB ID for operation logging.
// Used for fresh connects only - ensures session.ID is assigned BEFORE operation logging
// so that Connect audit rows have real session IDs (SCACI §3.3-04 audit trail).
//
// Resumed sessions continue using PersistConnectAsync (they already have session.ID > 0).
func (p *sessionPersistence) PersistConnectSync(ctx context.Context, session *scaci.Session, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, negotiatedVersion string) (int64, error) {
	createCtx, cancel := context.WithTimeout(ctx, scaci.ConnectPersistTimeout)
	defer cancel()

	// Reuse exact model construction from PersistConnectAsync for single source of truth
	var certFP, certSubj, remAddr, tlsVer, cipherSt *string
	if certFingerprint != "" {
		certFP = &certFingerprint
	}
	if certSubject != "" {
		certSubj = &certSubject
	}
	if remoteAddr != "" {
		remAddr = &remoteAddr
	}
	// TLS evidence per SCACI §1
	if tlsVersion != "" {
		tlsVer = &tlsVersion
	}
	if cipherSuite != "" {
		cipherSt = &cipherSuite
	}

	// Set organization ID if not nil
	var orgID *uuid.UUID
	if session.OrganizationID != uuid.Nil {
		orgID = &session.OrganizationID
	}

	// Set negotiated version - defaults to ProtocolVersionString if empty
	negVer := negotiatedVersion
	if negVer == "" {
		negVer = scaci.ProtocolVersionString
	}

	createReq := &models.SCACISessionCreateRequest{
		TenantID:               session.TenantID,
		OrganizationID:         orgID,
		AcEUI:                  session.GetAcEuiBytes(),
		SnAcUUID:               session.SnAcUUID,
		SnScUUID:               session.SnScUUID,
		CertificateFingerprint: certFP,
		ClientCertSubject:      certSubj,
		RemoteAddr:             remAddr,
		TLSVersion:             tlsVer,
		CipherSuite:            cipherSt,
		NegotiatedVersion:      negVer, // SCACI §§2.1-2.3: persist for resume validation
		CanResume:              true,
		Metadata:               deepCopyMetadata(session.Metadata), // Same helper for consistency
	}

	dbSession, err := p.sessionRepo.CreateSession(createCtx, createReq)
	if err != nil {
		return 0, err
	}
	return dbSession.ID, nil
}

// deepCopyMetadata recursively copies a map[string]interface{} structure.
//
// Deep-copied types (recursive):
//   - map[string]interface{}: Creates new map, recursively copies values
//   - []interface{}: Creates new slice, recursively copies elements
//
// Shallow/pass-through types (NOT deep-copied):
//   - Primitives (string, int, int64, float64, bool): Copied by value (safe)
//   - Typed maps (map[string]string, map[int]string): Passed by reference (shallow)
//   - Typed slices ([]string, []int): Passed by reference (shallow)
//   - nil: Returns nil
//
// IMPORTANT: Typed maps/slices (e.g., map[string]string, []string) are NOT
// recursively copied - they pass through as references. This is acceptable
// because SCACI metadata from MessagePack decoding uses map[string]interface{}
// and []interface{} exclusively. If typed collections appear in metadata,
// callers must not mutate them after calling deepCopyMetadata.
//
// This function is used to create an independent copy of session metadata
// before async persistence to prevent race conditions with caller mutations.
func deepCopyMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return nil
	}
	result := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue recursively copies interface{} values.
// See deepCopyMetadata for type handling documentation.
func deepCopyValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		// Recursively copy untyped maps
		return deepCopyMetadata(val)
	case []interface{}:
		// Recursively copy untyped slices
		copied := make([]interface{}, len(val))
		for i, item := range val {
			copied[i] = deepCopyValue(item)
		}
		return copied
	default:
		// Primitives: copied by value (safe)
		// Typed maps/slices: passed by reference (shallow - see doc comment)
		return val
	}
}

// PersistHeartbeatAsync updates session heartbeat timestamp in database.
// Heartbeat persistence for ping operations per SCACI §3.4.
//
// Context handling:
//   - ctx: Carries session metadata from s.sessionContext(session)
//   - logCtx: Decoupled from cancellation via WithoutCancel (preserves tenant/org metadata)
//   - hbCtx: Derived from logCtx with timeout for DB call
func (p *sessionPersistence) PersistHeartbeatAsync(ctx context.Context, session *scaci.Session) {
	if session == nil || session.ID == 0 {
		return
	}

	// Capture primitives before goroutine to prevent race/dangling pointer
	sessionID := session.ID
	tenantID := session.TenantID

	// Decouple from caller cancellation while preserving tenant/org metadata (Go 1.21+)
	logCtx := context.WithoutCancel(ctx)

	go func() {
		// Derive timeout context from logCtx to preserve tenant/org metadata
		hbCtx, cancel := context.WithTimeout(logCtx, scaci.ConnectPersistTimeout)
		defer cancel()

		if err := p.sessionRepo.UpdateHeartbeat(hbCtx, tenantID, sessionID); err != nil {
			p.logger.WarnContext(logCtx, scaci.LogSCACIPersistHeartbeatFailed,
				"sessionID", sessionID,
				"tenantID", tenantID,
				"error", err)
		}
	}()
}
