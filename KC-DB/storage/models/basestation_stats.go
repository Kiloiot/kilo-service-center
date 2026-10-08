package models

// BaseStationStatistics represents aggregated statistics for Base Stations
type BaseStationStatistics struct {
	TotalCount   int64 `db:"total_count" json:"total_count"`
	OnlineCount  int64 `db:"online_count" json:"online_count"`
	OfflineCount int64 `db:"offline_count" json:"offline_count"`
	BSSCICount   int64 `db:"bssci_count" json:"bssci_count"`
	MQTTCount    int64 `db:"mqtt_count" json:"mqtt_count"`
}

// SessionStatistics represents aggregated session statistics
type SessionStatistics struct {
	TotalSessions          int64   `json:"total_sessions"`
	ActiveSessions         int64   `json:"active_sessions"`
	TerminatedSessions     int64   `json:"terminated_sessions"`
	ResumableSessions      int64   `json:"resumable_sessions"`
	AverageSessionDuration float64 `json:"average_session_duration_hours"`
	TotalSessionTime       float64 `json:"total_session_time_hours"`
}

// SessionResumptionInfo provides information for session resumption decision
type SessionResumptionInfo struct {
	CanResume            bool   `json:"can_resume"`
	SessionID            int64  `json:"session_id"`
	LastKnownBsOpId      int64  `json:"last_known_bs_op_id"`
	LastKnownScOpId      int64  `json:"last_known_sc_op_id"`
	SessionAge           int64  `json:"session_age_hours"`
	ReasonIfNotResumable string `json:"reason_if_not_resumable,omitempty"`
}
