package scaciservices

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SessionRows owns every write of a SCACI session's row: its creation, its
// resume, the loss of its connection, its heartbeat, its operation ID
// counters and the end of its resumability. The server, the session registry
// and the resume holder reach it through their own narrow ports.
type SessionRows struct {
	creations SessionCreationRunner
	lifecycle SessionLifecycleRows
	activity  SessionActivityRows
	// scEUI is this service center, the owner of the rows it creates or resumes.
	scEUI models.EUI
}

// Compile-time contracts: the ports SessionRows serves.
var (
	_ scaci.SessionPersistence    = (*SessionRows)(nil)
	_ scaci.SessionLifecycleStore = (*SessionRows)(nil)
	_ HeldSessionRows             = (*SessionRows)(nil)
)

// NewSessionRows builds the owner of the session rows of the service center
// serviceCenterEUI: fresh sessions are created through the creation
// transaction, existing rows are updated through the lifecycle and activity
// stores.
func NewSessionRows(creations SessionCreationRunner, lifecycle SessionLifecycleRows, activity SessionActivityRows, serviceCenterEUI uint64) (*SessionRows, error) {
	if creations == nil || lifecycle == nil || activity == nil {
		return nil, errMissingSessionPersistenceDependency
	}
	return &SessionRows{creations: creations, lifecycle: lifecycle, activity: activity,
		scEUI: models.EUI(mioty.EUI64(serviceCenterEUI).ToBytes())}, nil
}

// PersistResume records on the row of a resumed session the connection that
// resumed it (status, heartbeat, TLS evidence, metadata) unless the session
// stopped being resumable meanwhile. The operation ID counters are written
// only by the counter path, which never moves them back. Fresh sessions are
// persisted by PersistConnectSync, so this path never creates rows.
func (p *SessionRows) PersistResume(ctx context.Context, session *scaci.Session, tlsVersion, cipherSuite string) error {
	if !session.Resumed || session.ID <= 0 {
		return errResumeRequiresPersistedSession
	}

	return p.lifecycle.ResumeSession(ctx, session.TenantID, session.ID, &models.SCACISessionResume{
		TLSVersion:  optionalText(tlsVersion),
		CipherSuite: optionalText(cipherSuite),
		Metadata:    deepCopyMetadata(session.Metadata),
		ScEui:       p.scEUI,
	})
}

// PersistConnectSync creates the row of a fresh session and returns its ID,
// so the connect audit rows carry a real session ID (SCACI §3.3). The
// application center's earlier sessions are retired in the same transaction
// (SCACI §1). Resumed sessions use PersistResume.
func (p *SessionRows) PersistConnectSync(ctx context.Context, session *scaci.Session, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, negotiatedVersion string) (int64, error) {
	createCtx, cancel := context.WithTimeout(ctx, scaci.ConnectPersistTimeout)
	defer cancel()

	ac := session.ApplicationCenter().Key()
	createReq := &models.SCACISessionCreateRequest{
		TenantID:               ac.TenantID,
		OrganizationID:         ac.OrganizationID,
		AcEUI:                  ac.AcEUI,
		SnAcUUID:               session.SnAcUUID,
		SnScUUID:               session.SnScUUID,
		CertificateFingerprint: optionalText(certFingerprint),
		ClientCertSubject:      optionalText(certSubject),
		RemoteAddr:             optionalText(remoteAddr),
		TLSVersion:             optionalText(tlsVersion),
		CipherSuite:            optionalText(cipherSuite),
		NegotiatedVersion:      negotiatedOrDefault(negotiatedVersion), // SCACI §§2.1-2.3: persist for resume validation
		CanResume:              true,
		Metadata:               deepCopyMetadata(session.Metadata),
		ScEui:                  p.scEUI,
	}

	var id int64
	err := p.creations.Run(createCtx, func(tx SessionCreationTx) error {
		if err := tx.RetirePriorSessions(createCtx, ac); err != nil {
			return err
		}
		created, err := tx.CreateSession(createCtx, createReq)
		if err != nil {
			return err
		}
		id = created.ID
		return nil
	})
	return id, err
}

// optionalText is text for a nullable column: nil when empty.
func optionalText(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}

// negotiatedOrDefault is the negotiated version, or this service center's
// protocol version when none was negotiated.
func negotiatedOrDefault(negotiatedVersion string) string {
	if negotiatedVersion == "" {
		return scaci.ProtocolVersionString
	}
	return negotiatedVersion
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

// PersistHeartbeat records keepalive activity of a persisted session (SCACI
// §3.4); a session without a row has nothing to update.
func (p *SessionRows) PersistHeartbeat(ctx context.Context, session *scaci.Session) error {
	if session.ID <= 0 {
		return nil
	}
	return p.activity.UpdateHeartbeat(ctx, session.TenantID, session.ID)
}

// PersistDisconnect records the loss of the session's connection; the session
// stays resumable (SCACI §1), one a newer session replaced stays terminated.
func (p *SessionRows) PersistDisconnect(ctx context.Context, session *scaci.Session) error {
	return p.lifecycle.MarkSessionDisconnected(ctx, session.TenantID, session.ID)
}

// PersistOpIDs stores a snapshot of the session's operation ID counters
// (SCACI §3.2); the store never moves a counter back.
func (p *SessionRows) PersistOpIDs(ctx context.Context, session *scaci.Session, ids scaci.OpIDPair) error {
	return p.activity.UpdateOperationIDs(ctx, session.TenantID, session.ID, ids.AC, ids.SC)
}

// EndResumability terminates the session: its application center starts a
// new session when it reconnects (SCACI §1).
func (p *SessionRows) EndResumability(ctx context.Context, session *scaci.Session) error {
	return p.lifecycle.TerminateSession(ctx, session.TenantID, session.ID)
}
