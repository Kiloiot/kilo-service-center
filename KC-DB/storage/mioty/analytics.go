package mioty

import "time"

// ULDataMessageFilter defines filters for querying UL Data messages
type ULDataMessageFilter struct {
	TenantID   int64
	EpEui      *uint64
	BsEui      *uint64
	StartTime  *time.Time
	EndTime    *time.Time
	SearchTerm *string // Optional search term for ep_eui, bs_eui, or user_data
	Direction  *string // Filter by direction: "uplink" | "downlink" | nil (all)
	Duplicate  *bool   // SCACI §3.8.1 multi-base-station reception flag
	DlOpen     *bool   // SCACI §3.8.1 dlOpen
	Profile    *string // SCACI §3.8.1 profile
	Mode       *string // SCACI §3.8.1 mode
	// StoredSince keeps the uplinks the database stored at or after it.
	StoredSince *time.Time
	Limit       int
	Offset      int
}

// MessageStats holds aggregate statistics for messages
type MessageStats struct {
	TotalCount      int64      `json:"total_count"`
	UniqueEndpoints int64      `json:"unique_endpoints"`
	AvgRSSI         float64    `json:"avg_rssi"`
	AvgSNR          float64    `json:"avg_snr"`
	MinRSSI         *float64   `json:"min_rssi,omitempty"`
	MaxRSSI         *float64   `json:"max_rssi,omitempty"`
	MinSNR          *float64   `json:"min_snr,omitempty"`
	MaxSNR          *float64   `json:"max_snr,omitempty"`
	FirstSeen       *time.Time `json:"first_seen,omitempty"`
	LastSeen        *time.Time `json:"last_seen,omitempty"`
	ActiveDays      *int       `json:"active_days,omitempty"`
}

// AnalyticsOverviewStats provides overview analytics
type AnalyticsOverviewStats struct {
	TotalMessages      int64      `db:"total_messages"`
	ActiveEndpoints    int64      `db:"active_endpoints"`
	ActiveBaseStations int64      `db:"active_basestations"`
	AvgRSSI            *float64   `db:"avg_rssi"`
	AvgSNR             *float64   `db:"avg_snr"`
	FirstMessage       *time.Time `db:"first_message"`
	LastMessage        *time.Time `db:"last_message"`
}

// SignalQualityStats represents overall signal quality statistics
type SignalQualityStats struct {
	AvgRSSI       float64 `db:"avg_rssi"`
	MinRSSI       float64 `db:"min_rssi"`
	MaxRSSI       float64 `db:"max_rssi"`
	MedianRSSI    float64 `db:"median_rssi"`
	AvgSNR        float64 `db:"avg_snr"`
	MinSNR        float64 `db:"min_snr"`
	MaxSNR        float64 `db:"max_snr"`
	MedianSNR     float64 `db:"median_snr"`
	TotalMessages int64   `db:"total_messages"`
}

// BaseStationSignalQuality is the signal quality of the uplinks one base
// station received.
type BaseStationSignalQuality struct {
	BsEui        uint64
	AvgRSSI      float64
	AvgSNR       float64
	MessageCount int64
}
