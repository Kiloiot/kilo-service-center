package models

import "time"

// Application center session lifecycle event types (category: scaci).
const (
	// EventTypeSCACISessionOpened records an application center completing
	// the connect operation of a new session (SCACI §3.3).
	EventTypeSCACISessionOpened = "scaci.session_opened"
	// EventTypeSCACISessionResumed records an application center completing
	// the connect operation that resumes its session (SCACI §1, §3.3).
	EventTypeSCACISessionResumed = "scaci.session_resumed"
	// EventTypeSCACISessionClosed records a session losing its connection;
	// the session stays resumable (SCACI §1).
	EventTypeSCACISessionClosed = "scaci.session_closed"
	// EventTypeSCACIConnectRefused records a connect operation the service
	// center refused with an error (SCACI §3.3, §3.14).
	EventTypeSCACIConnectRefused = "scaci.connect_refused"
)

// SourceTypeApplicationCenter identifies events about an application center.
const SourceTypeApplicationCenter = "application_center"

// Why a session closed (EventDetailKeyReason of scaci.session_closed): its
// connection was lost and it stays resumable (SCACI §1), or a new session of
// the same application center took its place (SCACI §1).
const (
	SCACISessionClosedConnectionLost = "connection_lost"
	SCACISessionClosedSuperseded     = "superseded"
)

// Detail keys of the application center session events. A refused connect
// is one record per cause (tenant, application center, error token): count,
// firstSeen and lastSeen say how often and when it recurred.
const (
	EventDetailKeySessionID  = "sessionID"
	EventDetailKeyAcEui      = "acEui"
	EventDetailKeyRemoteAddr = "remoteAddr"
	EventDetailKeyErrorToken = "errorToken"
	EventDetailKeyCount      = "count"
	EventDetailKeyFirstSeen  = "firstSeen"
	EventDetailKeyLastSeen   = "lastSeen"
)

// Titles and descriptions of the application center session events.
const (
	// ApplicationCenterUnknown names the application center of a connect
	// refused before its EUI was decoded.
	ApplicationCenterUnknown = "unknown"

	EventTitleSCACISessionOpened           = "Application center %s connected"
	EventTitleSCACISessionResumed          = "Application center %s resumed its session"
	EventTitleSCACISessionClosed           = "Application center %s disconnected"
	EventTitleSCACIConnectRefused          = "Application center %s connect refused"
	EventDescriptionSCACISessionOpened     = "Session %d of application center %s is active"
	EventDescriptionSCACISessionResumed    = "Session %d of application center %s resumed on a new connection"
	EventDescriptionSCACISessionClosed     = "Session %d of application center %s lost its connection and stays resumable"
	EventDescriptionSCACISessionSuperseded = "Session %d of application center %s was replaced by a new session of the application center"
	EventDescriptionSCACIConnectRefused    = "Connect of application center %s refused: %s"
)

// SCACISessionEvent is one step in the lifecycle of an application center
// session (SCACI §1, §3.3), filed under the tenant the session belongs to.
type SCACISessionEvent struct {
	TenantID int64
	// Category is EventCategorySCACI for a tenant's session, and
	// EventCategorySecurity for a refused connect of no tenant, filed under
	// the platform tenant for administrators.
	Category  string
	EventType string // EventTypeSCACISession* or EventTypeSCACIConnectRefused
	// SessionID is zero for a connect refused before its session existed.
	SessionID int64
	// AcEui is the canonical application center EUI, empty when the connect
	// could not be decoded.
	AcEui      string
	RemoteAddr string
	// Reason says why a session closed (SCACISessionClosed*).
	Reason string
	// ErrorToken and ErrorMessage name the SCACI catalog error a refused
	// connect was answered with.
	ErrorToken   string
	ErrorMessage string
	// OccurredAt is when an opened, resumed or closed step happened, read
	// by the service center at that step; zero leaves it to the store. A
	// refused connect is a counted record the store stamps under its cause
	// lock, so its lastSeen only moves forward.
	OccurredAt time.Time
}
