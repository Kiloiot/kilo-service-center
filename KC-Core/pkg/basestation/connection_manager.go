// Package basestation manages base station connections via BSSCI and MQTT protocols.
package basestation

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-MQTT/pkg/mqtt"
)

// ConnectionType represents the type of connection protocol
type ConnectionType string

// Connection type constants define the protocol used for base station communication.
const (
	ConnectionTypeBSSCI ConnectionType = "bssci"
	ConnectionTypeMQTT  ConnectionType = "mqtt"
	placeholderLiveCert                = "established" // Placeholder for live connection certificates
)

// ConnectionStatus represents the current connection state
type ConnectionStatus struct {
	IsOnline         bool                   `json:"is_online"`
	LastSeen         time.Time              `json:"last_seen"`
	ConnectionType   ConnectionType         `json:"connection_type"`
	SessionID        string                 `json:"session_id,omitempty"`
	UptimeSeconds    int64                  `json:"uptime_seconds,omitempty"`
	ConnectionDetail map[string]interface{} `json:"connection_details,omitempty"`
	LastError        string                 `json:"last_error,omitempty"`
}

// Registration contains all information needed to register a Base Station
type Registration struct {
	EUI              [8]byte        `json:"eui"`
	Name             string         `json:"name"`
	ConnectionType   ConnectionType `json:"connection_type"`
	ServiceCenterURL string         `json:"service_center_url,omitempty"`

	// For BSSCI
	TLSCertificate []byte `json:"tls_certificate,omitempty"`
	TLSPrivateKey  []byte `json:"tls_private_key,omitempty"`
	TLSCACert      []byte `json:"tls_ca_cert,omitempty"`

	// For MQTT
	MQTTBrokerURL   string `json:"mqtt_broker_url,omitempty"`
	MQTTClientID    string `json:"mqtt_client_id,omitempty"`
	MQTTUsername    string `json:"mqtt_username,omitempty"`
	MQTTPassword    string `json:"mqtt_password,omitempty"`
	MQTTTopicPrefix string `json:"mqtt_topic_prefix,omitempty"`

	// Optional metadata
	Vendor   string    `json:"vendor,omitempty"`
	Model    string    `json:"model,omitempty"`
	Version  string    `json:"version,omitempty"`
	Location *Location `json:"location,omitempty"`
}

// Location represents geographic coordinates
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// ConnectionManager manages all Base Station connections
type ConnectionManager struct {
	mu            sync.RWMutex
	bssciHandler  *BSSCIHandler
	mqttHandler   *MQTTHandler
	store         Store
	eventRecorder EventRecorder
}

// Store interface for database operations
type Store interface {
	RegisterBaseStation(ctx context.Context, reg *Registration) error
	GetBaseStation(ctx context.Context, eui [8]byte) (*BaseStation, error)
	GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*BaseStation, error)
	UpdateConnectionStatus(ctx context.Context, eui [8]byte, status *ConnectionStatus) error
	ListBaseStations(ctx context.Context) ([]*BaseStation, error)
}

// EventRecorder interface for audit logging
type EventRecorder interface {
	RecordEvent(ctx context.Context, eui [8]byte, eventType string, data map[string]interface{}) error
}

// BaseStation represents a Base Station entity
type BaseStation struct {
	ID               int64
	TenantID         int64
	EUI              [8]byte
	Name             string
	Description      string
	ConnectionType   ConnectionType
	ServiceCenterURL string
	Vendor           string
	Model            string
	SoftwareVersion  string
	Location         *Location
	LastSeenAt       *time.Time
	Status           *ConnectionStatus
	Metadata         map[string]interface{}
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(store Store, eventRecorder EventRecorder, log logger.Logger) *ConnectionManager {
	return &ConnectionManager{
		store:         store,
		eventRecorder: eventRecorder,
		bssciHandler:  NewBSSCIHandler(),
		mqttHandler:   NewMQTTHandler(log),
	}
}

// RegisterBaseStation registers a new Base Station with the Service Center
func (cm *ConnectionManager) RegisterBaseStation(ctx context.Context, reg *Registration) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate registration
	if err := cm.validateRegistration(reg); err != nil {
		return fmt.Errorf("invalid registration: %w", err)
	}

	// Store in database
	if err := cm.store.RegisterBaseStation(ctx, reg); err != nil {
		return fmt.Errorf("failed to store base station: %w", err)
	}

	// Record registration event
	eventData := map[string]interface{}{
		"name":            reg.Name,
		"connection_type": reg.ConnectionType,
		"vendor":          reg.Vendor,
		"model":           reg.Model,
	}
	if err := cm.eventRecorder.RecordEvent(ctx, reg.EUI, models.EventTypeBSRegistered, eventData); err != nil {
		// Log error but don't fail registration
		fmt.Printf("Failed to record registration event: %v\n", err)
	}

	// Initialize connection based on type
	// For live connections (with placeholder certificate), skip initialization
	isLiveConnection := string(reg.TLSCertificate) == placeholderLiveCert && string(reg.TLSPrivateKey) == placeholderLiveCert

	switch reg.ConnectionType {
	case ConnectionTypeBSSCI:
		if !isLiveConnection {
			return cm.bssciHandler.InitializeConnection(ctx, reg)
		}
		// For live BSSCI connections, the connection is already established
		return nil
	case ConnectionTypeMQTT:
		return cm.mqttHandler.InitializeConnection(ctx, reg)
	default:
		return fmt.Errorf("unsupported connection type: %s", reg.ConnectionType)
	}
}

// GetBaseStation retrieves a Base Station by EUI (tenant-scoped via context)
func (cm *ConnectionManager) GetBaseStation(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	return cm.store.GetBaseStation(ctx, eui)
}

// GetBaseStationGlobal retrieves a Base Station by EUI across all tenants.
// Used during BSSCI connect handshake before tenant is resolved.
func (cm *ConnectionManager) GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	return cm.store.GetBaseStationGlobal(ctx, eui)
}

// GetConnectionStatus returns the current connection status for a Base Station
func (cm *ConnectionManager) GetConnectionStatus(ctx context.Context, eui [8]byte) (*ConnectionStatus, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	bs, err := cm.store.GetBaseStation(ctx, eui)
	if err != nil {
		return nil, fmt.Errorf("base station not found: %w", err)
	}

	switch bs.ConnectionType {
	case ConnectionTypeBSSCI:
		return cm.bssciHandler.GetStatus(eui)
	case ConnectionTypeMQTT:
		return cm.mqttHandler.GetStatus(eui)
	default:
		return nil, fmt.Errorf("unknown connection type: %s", bs.ConnectionType)
	}
}

// UpdateLastSeen updates the last seen timestamp for a Base Station
func (cm *ConnectionManager) UpdateLastSeen(ctx context.Context, eui [8]byte) error {
	// First get the base station to determine its connection type
	bs, err := cm.store.GetBaseStation(ctx, eui)
	if err != nil {
		return fmt.Errorf("failed to get base station: %w", err)
	}

	status := &ConnectionStatus{
		IsOnline:       true,
		LastSeen:       time.Now(),
		ConnectionType: bs.ConnectionType,
	}

	if err := cm.store.UpdateConnectionStatus(ctx, eui, status); err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	// Heartbeat events removed - they flood the system_events table with no audit value.
	// Activity is tracked via base_stations.last_seen_at, and operational events
	// (basestation_online, basestation_offline, connection_error) provide audit trail.
	return nil
}

// UpdateConnectionStatus updates the connection status for a Base Station
func (cm *ConnectionManager) UpdateConnectionStatus(ctx context.Context, eui [8]byte, status *ConnectionStatus) error {
	if err := cm.store.UpdateConnectionStatus(ctx, eui, status); err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}

	// Get base station details for the event
	bs, err := cm.store.GetBaseStation(ctx, eui)
	if err != nil {
		// Log error but continue with event recording
		fmt.Printf("Failed to get base station details: %v\n", err)
	}

	// Record status change event
	eventData := map[string]interface{}{
		"is_online":       status.IsOnline,
		"connection_type": status.ConnectionType,
		"session_id":      status.SessionID,
	}

	// Add base station name if available
	if bs != nil && bs.Name != "" {
		eventData["basestation_name"] = bs.Name
	}

	eventType := models.EventTypeBaseStationOffline
	if status.IsOnline {
		eventType = models.EventTypeBaseStationOnline
	}

	return cm.eventRecorder.RecordEvent(ctx, eui, eventType, eventData)
}

// validateRegistration validates the registration parameters
func (cm *ConnectionManager) validateRegistration(reg *Registration) error {
	if reg.Name == "" {
		return fmt.Errorf("name is required")
	}

	switch reg.ConnectionType {
	case ConnectionTypeBSSCI:
		if reg.ServiceCenterURL == "" {
			return fmt.Errorf("service center URL is required for BSSCI")
		}
		// For live BSSCI connections, certificates are already validated through TLS handshake
		// Only require certificates for pre-registration scenarios
		if len(reg.TLSCertificate) == 0 || len(reg.TLSPrivateKey) == 0 {
			// Allow placeholder for live connections
			if string(reg.TLSCertificate) != placeholderLiveCert || string(reg.TLSPrivateKey) != placeholderLiveCert {
				return fmt.Errorf("TLS certificate and private key are required for BSSCI")
			}
		}
	case ConnectionTypeMQTT:
		if reg.MQTTBrokerURL == "" {
			return fmt.Errorf("MQTT broker URL is required for MQTT")
		}
		if reg.MQTTClientID == "" {
			// Generate client ID from EUI if not provided
			reg.MQTTClientID = fmt.Sprintf("mioty-bs-%s", hex.EncodeToString(reg.EUI[:]))
		}
		if reg.MQTTTopicPrefix == "" {
			reg.MQTTTopicPrefix = "mioty/default"
		}
	default:
		return fmt.Errorf("invalid connection type: %s", reg.ConnectionType)
	}

	return nil
}

// BSSCIHandler manages BSSCI protocol connections
type BSSCIHandler struct {
	mu       sync.RWMutex
	sessions map[string]*BSSCISession
}

// BSSCISession represents an active BSSCI session
type BSSCISession struct {
	EUI       [8]byte
	SessionID string
	Conn      *tls.Conn
	StartedAt time.Time
	LastSeen  time.Time
}

// NewBSSCIHandler creates a new BSSCI handler
func NewBSSCIHandler() *BSSCIHandler {
	return &BSSCIHandler{
		sessions: make(map[string]*BSSCISession),
	}
}

// InitializeConnection initializes a BSSCI connection
func (h *BSSCIHandler) InitializeConnection(_ context.Context, reg *Registration) error {
	// Parse certificates
	cert, err := tls.X509KeyPair(reg.TLSCertificate, reg.TLSPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse TLS certificate: %w", err)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if len(reg.TLSCACert) > 0 {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(reg.TLSCACert) {
			return fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Note: Actual connection establishment would happen here
	// This is a placeholder for the connection logic
	// In a real implementation, this would:
	// 1. Establish TLS connection to Service Center
	// 2. Perform BSSCI handshake
	// 3. Start message processing goroutine
	// 4. Handle reconnection logic

	sessionID := uuid.New().String()
	session := &BSSCISession{
		EUI:       reg.EUI,
		SessionID: sessionID,
		StartedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	h.mu.Lock()
	h.sessions[hex.EncodeToString(reg.EUI[:])] = session
	h.mu.Unlock()

	return nil
}

// GetStatus returns the connection status for a BSSCI session
func (h *BSSCIHandler) GetStatus(eui [8]byte) (*ConnectionStatus, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	session, exists := h.sessions[hex.EncodeToString(eui[:])]
	if !exists {
		return &ConnectionStatus{
			IsOnline:       false,
			ConnectionType: ConnectionTypeBSSCI,
		}, nil
	}

	return &ConnectionStatus{
		IsOnline:       true,
		LastSeen:       session.LastSeen,
		ConnectionType: ConnectionTypeBSSCI,
		SessionID:      session.SessionID,
		UptimeSeconds:  int64(time.Since(session.StartedAt).Seconds()),
	}, nil
}

// MQTTHandler manages MQTT protocol connections
type MQTTHandler struct {
	mu      sync.RWMutex
	clients map[string]mqtt.Publisher
	logger  logger.Logger
}

// NewMQTTHandler creates a new MQTT handler
func NewMQTTHandler(log logger.Logger) *MQTTHandler {
	return &MQTTHandler{
		clients: make(map[string]mqtt.Publisher),
		logger:  log,
	}
}

// InitializeConnection initializes an MQTT connection
func (h *MQTTHandler) InitializeConnection(ctx context.Context, reg *Registration) error {
	euiHex := hex.EncodeToString(reg.EUI[:])

	// Parse MQTT broker URL to extract host, port, and TLS settings
	brokerURL, err := url.Parse(reg.MQTTBrokerURL)
	if err != nil {
		return fmt.Errorf("invalid MQTT broker URL: %w", err)
	}

	host := brokerURL.Hostname()
	port := 1883 // default MQTT port
	if p := brokerURL.Port(); p != "" {
		if portNum, err := strconv.Atoi(p); err == nil {
			port = portNum
		}
	}

	// Determine TLS based on URL scheme (ssl://, mqtts://)
	// Note: Registration struct does not have MQTTUseTLS field, so only URL scheme is honored
	useTLS := brokerURL.Scheme == "ssl" || brokerURL.Scheme == "mqtts"

	// Build MQTT config from registration
	mqttConfig := &config.MQTTConfig{
		Host:                 host,
		Port:                 port,
		ClientID:             reg.MQTTClientID,
		Username:             reg.MQTTUsername,
		Password:             reg.MQTTPassword,
		TopicPrefix:          reg.MQTTTopicPrefix,
		KeepAlive:            60,
		MaxReconnectInterval: 120,
		CleanSession:         true,
		TLS:                  config.TLSConfig{Enabled: useTLS},
	}

	// Create MQTT client using KC-MQTT abstraction
	client, err := mqtt.NewClient(mqttConfig, h.logger)
	if err != nil {
		return fmt.Errorf("failed to create MQTT client: %w", err)
	}

	// Connect to broker
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	// Subscribe to relevant topics
	topic := fmt.Sprintf("%s/+/uplink", reg.MQTTTopicPrefix)
	if err := client.Subscribe(ctx, topic, mqtt.QoSAtLeastOnce, h.handleMessage); err != nil {
		h.logger.ErrorContext(ctx, "Failed to subscribe to topic", "topic", topic, "err", err)
		client.Disconnect(ctx)
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	h.mu.Lock()
	h.clients[euiHex] = client
	h.mu.Unlock()

	h.logger.InfoContext(ctx, "Base Station MQTT connection established", "eui", euiHex, "topic", topic)
	return nil
}

// GetStatus returns the connection status for an MQTT client
func (h *MQTTHandler) GetStatus(eui [8]byte) (*ConnectionStatus, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	euiHex := hex.EncodeToString(eui[:])
	client, exists := h.clients[euiHex]
	if !exists {
		return &ConnectionStatus{
			IsOnline:       false,
			ConnectionType: ConnectionTypeMQTT,
		}, nil
	}

	return &ConnectionStatus{
		IsOnline:       client.IsConnected(),
		LastSeen:       time.Now(), // Would track this properly in production
		ConnectionType: ConnectionTypeMQTT,
	}, nil
}

// handleMessage processes incoming MQTT messages using context-aware signature
func (h *MQTTHandler) handleMessage(ctx context.Context, topic string, payload []byte) {
	// Parse topic to extract endpoint EUI
	// Forward message to appropriate handler
	h.logger.DebugContext(ctx, "Received MQTT message", "topic", topic, "payload_size", len(payload))

	// In a real implementation, this would:
	// 1. Parse the MQTT topic to extract endpoint EUI
	// 2. Validate message format
	// 3. Forward to message processing pipeline
	// 4. Update statistics and last seen timestamps
}

// Disconnect gracefully disconnects all connections
func (cm *ConnectionManager) Disconnect(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Disconnect all MQTT clients
	for eui, client := range cm.mqttHandler.clients {
		client.Disconnect(ctx)
		delete(cm.mqttHandler.clients, eui)
	}

	// Close all BSSCI sessions
	for eui, session := range cm.bssciHandler.sessions {
		if session.Conn != nil {
			_ = session.Conn.Close()
		}
		delete(cm.bssciHandler.sessions, eui)
	}

	return nil
}
