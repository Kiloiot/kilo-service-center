package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// NullJSON handles NULL values for JSON columns
type NullJSON struct {
	Valid bool
	Data  json.RawMessage
}

// Scan implements the sql.Scanner interface for NullJSON
func (n *NullJSON) Scan(value interface{}) error {
	if value == nil {
		n.Valid = false
		n.Data = nil
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case []byte:
		n.Data = json.RawMessage(v)
	case string:
		n.Data = json.RawMessage(v)
	default:
		n.Data = nil
		n.Valid = false
	}
	return nil
}

// Value implements the driver.Valuer interface for NullJSON
func (n NullJSON) Value() (driver.Value, error) {
	if !n.Valid || n.Data == nil {
		return nil, nil
	}
	return []byte(n.Data), nil
}

// TenantProgressMap handles JSONB tenant_progress column as a native Go map
// Implements sql.Scanner and driver.Valuer for automatic PostgreSQL JSONB marshaling
//
// Structure: {"tenant_<id>": {"endpoints_processed": N, "last_updated": "timestamp", "errors": [...]}}
type TenantProgressMap map[string]interface{}

// Scan implements sql.Scanner for database retrieval
func (tp *TenantProgressMap) Scan(value interface{}) error {
	if value == nil {
		*tp = make(map[string]interface{})
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return json.Unmarshal(nil, tp) // Will return appropriate error
	}

	if len(data) == 0 {
		*tp = make(map[string]interface{})
		return nil
	}

	return json.Unmarshal(data, tp)
}

// Value implements driver.Valuer for database storage
func (tp TenantProgressMap) Value() (driver.Value, error) {
	if len(tp) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(tp)
}

// ConnectionType represents the type of connection for a Base Station
type ConnectionType string

// ConnectionType constants for Base Station connection types.
const (
	// ConnectionTypeBSSCI indicates BSSCI (TCP) connection.
	ConnectionTypeBSSCI ConnectionType = "bssci"
	// ConnectionTypeMQTT indicates MQTT connection.
	ConnectionTypeMQTT ConnectionType = "mqtt"
)

// Location source constants indicate how coordinates were obtained.
const (
	LocationSourceGPS    = "gps"
	LocationSourceManual = "manual"
)

// Geographic coordinate bounds for validation.
const (
	LatitudeMin  = -90.0
	LatitudeMax  = 90.0
	LongitudeMin = -180.0
	LongitudeMax = 180.0
)

// BaseStation represents a MIOTY Base Station (Gateway)
type BaseStation struct {
	ID          int64   `json:"id" db:"id"`
	TenantID    int64   `json:"tenant_id" db:"tenant_id"`
	EUI         EUI     `json:"bsEui" db:"bs_eui"`
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`

	// Connection Configuration
	ConnectionType          ConnectionType `json:"connection_type" db:"connection_type"`
	ServiceCenterURL        *string        `json:"service_center_url,omitempty" db:"service_center_url"`
	TLSCACertificate        *string        `json:"tls_ca_certificate,omitempty" db:"tls_ca_certificate"`
	TLSCertificate          *string        `json:"tls_certificate,omitempty" db:"tls_certificate"`
	TLSKey                  *string        `json:"-" db:"tls_key"` // Hidden in JSON for security
	TLSAuthRequired         bool           `json:"tls_auth_required" db:"tls_auth_required"`
	TLSHostnameVerification bool           `json:"tls_hostname_verification" db:"tls_hostname_verification"`
	TLSCertFingerprint      *string        `json:"tls_cert_fingerprint,omitempty" db:"tls_cert_fingerprint"`
	TLSCertExpiresAt        *time.Time     `json:"tls_cert_expires_at,omitempty" db:"tls_cert_expires_at"`

	// MQTT Configuration
	MQTTBrokerURL   *string `json:"mqtt_broker_url,omitempty" db:"mqtt_broker_url"`
	MQTTClientID    *string `json:"mqtt_client_id,omitempty" db:"mqtt_client_id"`
	MQTTUsername    *string `json:"mqtt_username,omitempty" db:"mqtt_username"`
	MQTTPassword    []byte  `json:"-" db:"mqtt_password_encrypted"` // Hidden in JSON
	MQTTTopicPrefix *string `json:"mqtt_topic_prefix,omitempty" db:"mqtt_topic_prefix"`

	// Status and Metrics
	IsOnline         bool       `json:"is_online" db:"is_online"`
	LastSeenAt       *time.Time `json:"last_seen_at,omitempty" db:"last_seen_at"`
	SessionUUID      *string    `json:"session_uuid,omitempty" db:"session_uuid"`
	SessionStartedAt *time.Time `json:"session_started_at,omitempty" db:"session_started_at"`
	Uptime           int64      `json:"uptime" db:"uptime"`
	StatusCode       int        `json:"status_code" db:"status_code"`
	StatusMessage    *string    `json:"status_message,omitempty" db:"status_message"`

	Tags *string `json:"tags,omitempty" db:"tags"`

	// MIOTY Bidirectional Capability
	Bidi           *bool      `json:"bidi,omitempty" db:"bidi"` // Hardware capability to transmit downlink
	ULLoad         *float32   `json:"ul_load,omitempty" db:"ul_load"`
	DLLoad         *float32   `json:"dl_load,omitempty" db:"dl_load"`
	LastModifiedBy *string    `json:"last_modified_by,omitempty" db:"last_modified_by"`
	LastModifiedAt *time.Time `json:"last_modified_at,omitempty" db:"last_modified_at"`

	// Hardware Info
	Vendor  *string `json:"vendor,omitempty" db:"vendor"`
	Model   *string `json:"model,omitempty" db:"model"`
	Version *string `json:"version,omitempty" db:"sw_version"`

	// Location
	Latitude          *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude         *float64   `json:"longitude,omitempty" db:"longitude"`
	Altitude          *float64   `json:"altitude,omitempty" db:"altitude"`
	LocationSource    *string    `json:"location_source,omitempty" db:"location_source"`
	LocationUpdatedAt *time.Time `json:"location_updated_at,omitempty" db:"location_updated_at"`

	// Configuration File
	ConfigFileContent    *string    `json:"config_file_content,omitempty" db:"config_file_content"`
	ConfigFileUploadedAt *time.Time `json:"config_file_uploaded_at,omitempty" db:"config_file_uploaded_at"`

	// Connection Status
	ConnectionStatus json.RawMessage `json:"connection_status,omitempty" db:"connection_status"`
	LastError        *string         `json:"last_error,omitempty" db:"last_error"`
	RetryCount       int             `json:"retry_count" db:"retry_count"`
	NextRetryAt      *time.Time      `json:"next_retry_at,omitempty" db:"next_retry_at"`

	// MIOTY Status Response Fields per BSSCI v1.0.0 Section 3.5.2
	SystemTime         *int64     `json:"system_time,omitempty" db:"system_time"`                 // Unix UTC system time, 64 bit, ns resolution
	DutyCycle          *float64   `json:"duty_cycle,omitempty" db:"duty_cycle"`                   // Fraction of TX time, sliding window over one hour
	UptimeSeconds      *int64     `json:"uptime_seconds,omitempty" db:"uptime_seconds"`           // System uptime in seconds
	TemperatureCelsius *float64   `json:"temperature_celsius,omitempty" db:"temperature_celsius"` // System temperature in degree Celsius
	CPULoad            *float64   `json:"cpu_load,omitempty" db:"cpu_load"`                       // CPU utilization, normalized to 1.0 for all cores
	MemoryLoad         *float64   `json:"memory_load,omitempty" db:"memory_load"`                 // Memory utilization, normalized to 1.0
	BSConfig           NullJSON   `json:"bs_config,omitempty" db:"bs_config"`                     // Configuration object from base station
	LastStatusAt       *time.Time `json:"last_status_at,omitempty" db:"last_status_at"`           // Timestamp when status response was last received

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// BaseStationCreateRequest represents the request to create a new Base Station
type BaseStationCreateRequest struct {
	EUI         string `json:"eui" validate:"required,len=16"` // Hex string
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description,omitempty"`

	// Connection Configuration
	ConnectionType          ConnectionType `json:"connection_type" validate:"required,oneof=bssci mqtt"`
	ServiceCenterURL        string         `json:"service_center_url,omitempty"`
	TLSCACertificate        string         `json:"tls_ca_certificate,omitempty"`
	TLSCertificate          string         `json:"tls_certificate,omitempty"`
	TLSKey                  string         `json:"tls_key,omitempty"`
	TLSAuthRequired         bool           `json:"tls_auth_required"`
	TLSHostnameVerification bool           `json:"tls_hostname_verification"`

	// MQTT Configuration
	MQTTBrokerURL string                 `json:"mqtt_broker_url,omitempty"`
	MQTTUsername  string                 `json:"mqtt_username,omitempty"`
	MQTTPassword  string                 `json:"mqtt_password,omitempty"`
	MQTTTopics    map[string]interface{} `json:"mqtt_topics,omitempty"`

	// Location
	Latitude  *float64 `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude *float64 `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	Altitude  *float64 `json:"altitude,omitempty"`

	// Configuration File Upload
	ConfigFileContent string `json:"config_file_content,omitempty"`
}

// BaseStationUpdateRequest represents the request to update a Base Station
type BaseStationUpdateRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty"`

	// Connection Configuration
	ServiceCenterURL        *string `json:"service_center_url,omitempty"`
	TLSCACertificate        *string `json:"tls_ca_certificate,omitempty"`
	TLSCertificate          *string `json:"tls_certificate,omitempty"`
	TLSKey                  *string `json:"tls_key,omitempty"`
	TLSAuthRequired         *bool   `json:"tls_auth_required,omitempty"`
	TLSHostnameVerification *bool   `json:"tls_hostname_verification,omitempty"`

	// MQTT Configuration
	MQTTBrokerURL *string                 `json:"mqtt_broker_url,omitempty"`
	MQTTUsername  *string                 `json:"mqtt_username,omitempty"`
	MQTTPassword  *string                 `json:"mqtt_password,omitempty"`
	MQTTTopics    *map[string]interface{} `json:"mqtt_topics,omitempty"`

	// Location
	Latitude  *float64 `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude *float64 `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	Altitude  *float64 `json:"altitude,omitempty"`
}

// BaseStationPropagationState tracks automatic propagation reconciliation per base station
type BaseStationPropagationState struct {
	BaseStationID      int64             `db:"base_station_id" json:"baseStationId"`
	LastFullSyncAt     *time.Time        `db:"last_full_sync_at" json:"lastFullSyncAt"`
	LastEndpointCursor int64             `db:"last_endpoint_cursor" json:"lastEndpointCursor"`
	Status             string            `db:"status" json:"status"`
	RetryCount         int               `db:"retry_count" json:"retryCount"`
	NextRetryAt        *time.Time        `db:"next_retry_at" json:"nextRetryAt"`
	LastError          *string           `db:"last_error" json:"lastError"`
	TenantProgress     TenantProgressMap `db:"tenant_progress" json:"tenantProgress"` // JSONB with Scanner/Valuer
	UpdatedAt          time.Time         `db:"updated_at" json:"updatedAt"`
}

// BaseStationFilter represents filter criteria for listing Base Stations
type BaseStationFilter struct {
	TenantID       int64
	ConnectionType *ConnectionType
	IsOnline       *bool
	Search         string // Search in name, description, EUI
	Limit          int
	Offset         int
}

// Scan implements the sql.Scanner interface for ConnectionType
func (ct *ConnectionType) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*ct = ConnectionType(v)
	case []byte:
		*ct = ConnectionType(v)
	default:
		*ct = ConnectionType("")
	}
	return nil
}

// Value implements the driver.Valuer interface for ConnectionType
func (ct ConnectionType) Value() (driver.Value, error) {
	return string(ct), nil
}
