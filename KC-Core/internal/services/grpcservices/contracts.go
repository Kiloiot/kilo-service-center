package grpcservices

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// EndpointService handles endpoint CRUD for gRPC layer
type EndpointService interface {
	Create(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error)
	GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error)
	Update(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error)
	UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error)
	CheckEUIGloballyUnique(ctx context.Context, eui []byte) error
	Delete(ctx context.Context, eui []byte, tenantID int64) error
	List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error)
	ListByModelWithSnapshot(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) ([]*models.EndPoint, error)
}

// EndpointAttachmentService handles endpoint attach/detach propagation operations
type EndpointAttachmentService interface {
	// AttachEndPoint initiates attach propagation to all base stations
	// Returns operation_id and status for fire-and-forget tracking
	AttachEndPoint(ctx context.Context, epEui string, tenantID int64) (*EndpointOperationResult, error)

	// DetachEndPoint initiates detach propagation to all base stations
	// Returns operation_id and status for fire-and-forget tracking
	DetachEndPoint(ctx context.Context, epEui string, tenantID int64) (*EndpointOperationResult, error)
}

// EndpointOperationResult contains the result of attach/detach operations
type EndpointOperationResult struct {
	OperationID string
	Status      string // "initiated", "in_progress", "completed", "failed"
}

// BaseStationService handles base station CRUD for gRPC layer
type BaseStationService interface {
	Create(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error)
	GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error)
	Update(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error)
	UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error)
	Delete(ctx context.Context, eui []byte, tenantID int64) error
	List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error)
	ListAllLocations(ctx context.Context) ([]*models.BaseStation, error)
}

// MessageService handles message operations for gRPC layer
type MessageService interface {
	GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error)
	GetDownlinkQueue(ctx context.Context, epEui, tenantID string) ([]*storage.DownlinkMessage, error)
	GetDownlinkResults(ctx context.Context, epEui, tenantID string, orgID *uuid.UUID, statusFilter string,
		timeFrom, timeTo *time.Time, limit, offset int) ([]*storage.DownlinkMessage, int, error)
	GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte,
		limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error)
	GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte,
		startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error)
	GetEndpointBaseStation(ctx context.Context, epEui, tenantID string) (string, error)
}

// AnalyticsService handles analytics queries
type AnalyticsService interface {
	GetOverview(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*AnalyticsOverview, error)
	GetActivity(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*ActivityAnalytics, error)
	GetSignalQuality(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*SignalQualityAnalytics, error)
}

// EventService handles event listing operations
type EventService interface {
	List(ctx context.Context, tenantID int64, filters *EventFilters, limit, offset int) ([]*Event, int64, error)
	ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *EventFilters, limit, offset int) ([]*Event, int64, error)
	ListByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *EventFilters, limit, offset int) ([]*Event, int64, error)
	// Streaming methods
	Stream(ctx context.Context, tenantID int64, filters *EventFilters) (<-chan *Event, error)
	StreamByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *EventFilters) (<-chan *Event, error)
	StreamByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *EventFilters) (<-chan *Event, error)
}

// AlertService handles alert queries
type AlertService interface {
	List(ctx context.Context, tenantID int64, filters *AlertFilters, limit, offset int) ([]*Alert, int64, error)
	GetSummary(ctx context.Context, tenantID int64) (*AlertSummary, error)
}

// ScaciMonitoringService handles SCACI session and statistics monitoring
type ScaciMonitoringService interface {
	ListSessions(ctx context.Context, tenantID int64, limit, offset int) ([]*ScaciSession, int64, error)
	GetSession(ctx context.Context, tenantID int64, sessionID string) (*ScaciSession, error)
	GetStatistics(ctx context.Context, tenantID int64) (*ScaciStatistics, error)
	ListErrors(ctx context.Context, tenantID int64, limit, offset int) ([]*ScaciError, int64, error)
	ListQueues(ctx context.Context, tenantID int64, limit, offset int) ([]*ScaciQueue, int64, error)
	GetStatus(ctx context.Context, tenantID int64) (*ScaciStatus, error)
}

// CertificateService handles certificate operations
type CertificateService interface {
	GenerateCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error)
	DownloadCertificateByID(ctx context.Context, certType, certID string) ([]byte, string, error)
	GetStoredCertificate(ctx context.Context, tenantID int64, bsEui []byte, certType string) ([]byte, string, error)
	GenerateServerCertificates(ctx context.Context) error
	RenewServerCertificates(ctx context.Context) error
	GetServerCertificateStatus(ctx context.Context) (*CertificateStatus, error)
}

// BlueprintService handles blueprint, manufacturer, and device model operations
type BlueprintService interface {
	// Manufacturers
	CreateManufacturer(ctx context.Context, req *ManufacturerCreateRequest) (*models.Manufacturer, error)
	GetManufacturer(ctx context.Context, id uuid.UUID) (*models.Manufacturer, error)
	UpdateManufacturer(ctx context.Context, id uuid.UUID, req *ManufacturerUpdateRequest) (*models.Manufacturer, error)
	DeleteManufacturer(ctx context.Context, id uuid.UUID) error
	ListManufacturers(ctx context.Context, isSystem bool, limit, offset int) ([]*models.Manufacturer, int64, error)

	// Device Models
	CreateDeviceModel(ctx context.Context, req *DeviceModelCreateRequest) (*models.DeviceModel, error)
	GetDeviceModel(ctx context.Context, id uuid.UUID) (*models.DeviceModel, error)
	GetDeviceModelForTenant(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error)
	UpdateDeviceModel(ctx context.Context, id uuid.UUID, req *DeviceModelUpdateRequest) (*models.DeviceModel, error)
	DeleteDeviceModel(ctx context.Context, id uuid.UUID) error
	ListDeviceModels(ctx context.Context, isSystem bool, manufacturerID *uuid.UUID, limit, offset int) ([]*models.DeviceModel, int64, error)

	// Blueprints
	CreateBlueprint(ctx context.Context, req *BlueprintCreateRequest) (*models.Blueprint, error)
	GetBlueprint(ctx context.Context, id uuid.UUID) (*models.Blueprint, error)
	UpdateBlueprint(ctx context.Context, id uuid.UUID, req *BlueprintUpdateRequest) (*models.Blueprint, error)
	DeleteBlueprint(ctx context.Context, id uuid.UUID) error
	ListBlueprints(ctx context.Context, isSystem bool, deviceModelID *uuid.UUID, limit, offset int) ([]*models.Blueprint, int64, error)
	SetDefaultBlueprint(ctx context.Context, id uuid.UUID) error
	// SubmitToRegistry submits a blueprint to an external registry.
	SubmitToRegistry(ctx context.Context, id uuid.UUID, req *RegistrySubmitRequest) (*RegistrySubmitResult, error)

	// CreateDeviceModelWithBlueprint creates a device model and default blueprint atomically.
	CreateDeviceModelWithBlueprint(ctx context.Context, req *DeviceModelWithBlueprintRequest) (*models.DeviceModel, *models.Blueprint, error)

	// DecodePreview runs the blueprint decoder on a payload for preview.
	DecodePreview(ctx context.Context, blueprintID uuid.UUID, payload []byte, formatID uint8) (*DecodePreviewResult, error)

	// DecodePreviewInline previews decoding against an unsaved inline spec.
	DecodePreviewInline(ctx context.Context, specJSON, payload []byte, formatID uint8) (*DecodePreviewResult, error)

	// ResolveEffectiveTypeEUI returns the TypeEUI from the device model's default blueprint.
	// Returns nil without error when no default blueprint exists or it has no TypeEUI.
	ResolveEffectiveTypeEUI(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.EUI, error)

	// GetDefaultForModel returns the device model's default blueprint (nil, nil when none).
	GetDefaultForModel(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error)
}

// MessageListingService handles message listing and streaming operations
type MessageListingService interface {
	// Single message by ID
	GetMessage(ctx context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error)

	// Endpoint messages
	ListMessages(ctx context.Context, tenantID int64, filters *MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	StreamMessages(ctx context.Context, tenantID int64, filters *MessageFilters) (<-chan *mioty.ULDataMessage, error)

	// Base station messages
	ListBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	GetBaseStationMessage(ctx context.Context, tenantID int64, bsEui []byte, messageID string) (*mioty.ULDataMessage, error)
	GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error)
	SearchBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, query string, filters *MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	ExportBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *MessageFilters, format string) ([]byte, error)
	StreamBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *MessageFilters) (<-chan *mioty.ULDataMessage, error)
}

// IntegrationService handles integration CRUD operations
type IntegrationService interface {
	Create(ctx context.Context, tenantID int64, req *IntegrationCreateRequest) (*models.Integration, error)
	GetByID(ctx context.Context, tenantID int64, id int64) (*models.Integration, error)
	Update(ctx context.Context, tenantID int64, id int64, req *IntegrationUpdateRequest) (*models.Integration, error)
	Delete(ctx context.Context, tenantID int64, id int64) error
	List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error)
}

// SystemStatusService handles system status aggregation for gRPC layer.
type SystemStatusService interface {
	GetStatus(ctx context.Context, tenantID int64) (*SystemStatusMetrics, error)
	GetServiceStatuses(ctx context.Context) ([]*ServiceStatusDTO, error)
}

// SystemStatusMetrics contains aggregated system statistics.
type SystemStatusMetrics struct {
	ActiveBasestations int32
	ActiveEndpoints    int32
	MessagesProcessed  int64
}

// ServiceStatusDTO represents health state of a monitored service.
type ServiceStatusDTO struct {
	Name      string
	URL       string
	Healthy   bool
	LatencyMs int32
	Error     string
	CheckedAt time.Time
}

// EventWriter writes system events to the event store.
// Structurally identical to pkg/grpc.EventWriter; Go structural typing ensures compatibility.
type EventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// AnalyticsOverview contains analytics overview data
type AnalyticsOverview struct {
	TotalEndpoints     int64
	ActiveEndpoints    int64
	TotalBaseStations  int64
	OnlineBaseStations int64
	TotalMessages      int64
	AverageRSSI        float64
	AverageSNR         float64
}

// ActivityAnalytics contains activity analytics data
type ActivityAnalytics struct {
	MessagesPerHour  map[string]int64
	EndpointActivity map[string]int64
	TopEndpoints     []EndpointActivity
	TopBaseStations  []BaseStationActivity
}

// EndpointActivity represents endpoint activity metrics
type EndpointActivity struct {
	EPEUI        string
	Name         string
	MessageCount int64
	LastSeen     *time.Time
}

// BaseStationActivity represents base station activity metrics
type BaseStationActivity struct {
	BSEUI        string
	Name         string
	MessageCount int64
	LastSeen     *time.Time
}

// SignalQualityAnalytics contains signal quality metrics
type SignalQualityAnalytics struct {
	AverageRSSI float64
	AverageSNR  float64
	RSSIRange   [2]float64 // [min, max]
	SNRRange    [2]float64 // [min, max]
	Quality     string     // "excellent", "good", "fair", "poor"
}

// EventFilters contains filters for event queries
type EventFilters struct {
	Categories []string
	Severity   []string
	EventTypes []string
	StartTime  *time.Time
	EndTime    *time.Time
}

// Event represents a system event
type Event struct {
	ID          string
	TenantID    int64
	Category    string
	EventType   string
	Severity    string
	Title       string
	Description string
	SourceName  string
	Timestamp   time.Time
	Data        []byte // JSON-encoded extra data (bs_eui, ep_eui, etc.)
}

// AlertFilters contains filters for alert queries
type AlertFilters struct {
	Severity  []string
	Status    []string
	StartTime *time.Time
	EndTime   *time.Time
}

// Alert represents a system alert
type Alert struct {
	ID          string
	TenantID    int64
	Severity    string
	Category    string
	Title       string
	Description string
	SourceName  string
	Timestamp   time.Time
	Status      string // new, acknowledged, resolved
}

// AlertSummary contains alert summary statistics
type AlertSummary struct {
	Critical int32
	Warning  int32
	Info     int32
	Recent   []*Alert
}

// ScaciSession represents a SCACI session
type ScaciSession struct {
	ID              string
	AcEUI           string // Application Center EUI (hex)
	Status          string // active, closed
	CanResume       bool
	ProtocolVersion string
	ConnectedAt     time.Time
	LastActivityAt  *time.Time
	OperationsCount int64
}

// ScaciStatistics contains SCACI operation statistics
type ScaciStatistics struct {
	TotalSessions        int64
	ActiveSessions       int64
	TotalOperations      int64
	SuccessfulOperations int64
	FailedOperations     int64
	SuccessRate          float64
	UptimeSince          *time.Time
}

// ScaciError represents a SCACI error
type ScaciError struct {
	ID            string
	ErrorCode     string
	ErrorMessage  string
	SessionID     string
	OperationType string
	OccurredAt    time.Time
}

// ScaciQueue represents a SCACI queue entry
type ScaciQueue struct {
	ID            string
	EpEUI         string
	OperationType string
	Status        string // pending, in_progress, completed, failed
	Payload       []byte
	QueuedAt      time.Time
	ProcessedAt   *time.Time
}

// ScaciStatus represents overall SCACI status
type ScaciStatus struct {
	ServiceOnline     bool
	ActiveSessions    int32
	PendingOperations int32
	UptimeSince       *time.Time
	ProtocolVersion   string
}

// CertificateRequest contains certificate generation request
type CertificateRequest struct {
	BsEUI           string // Base station EUI
	BaseStationName string // Human-readable base station name
	ValidityDays    int32  // Certificate validity in days
	TenantID        int64  // Tenant ID for multi-tenant cert persistence
}

// CertificateResponse contains generated certificate data
type CertificateResponse struct {
	BsEUI            string            // Base station EUI
	ServiceCenterURL string            // Service center URL for BSSCI
	DownloadURLs     map[string]string // URLs to download cert files (ca, client, key)
	ExpiresAt        *time.Time        // Certificate expiration time
}

// CertificateStatus contains server certificate status
type CertificateStatus struct {
	HasServerCert    bool
	ServerCertExpiry *time.Time
	HasCACert        bool
	CACertExpiry     *time.Time
	NeedsRenewal     bool
	Subject          string
	Issuer           string
}

// ManufacturerCreateRequest contains fields for creating a manufacturer
type ManufacturerCreateRequest struct {
	Name        string
	Description string
	Website     string
	LogoURL     string
	IsSystem    bool // admin-only: create a System catalog row (tenant_id NULL)
}

// ManufacturerUpdateRequest contains fields for updating a manufacturer
type ManufacturerUpdateRequest struct {
	Name         *string
	Description  *string
	Website      *string
	ContactEmail *string
	LogoURL      *string
}

// DeviceModelCreateRequest contains fields for creating a device model
// Maps to models.DeviceModelCreateParams: Code=URL-friendly slug, TypeEUI=8-byte MIOTY identifier
type DeviceModelCreateRequest struct {
	ManufacturerID uuid.UUID
	Name           string
	Code           string // URL-friendly slug (unique per manufacturer)
	TypeEUI        []byte // 8-byte MIOTY Type EUI (optional)
	Description    string
	DatasheetURL   string
	IsSystem       bool // admin-only: create a System catalog row (tenant_id NULL)
}

// DeviceModelUpdateRequest contains fields for updating a device model
type DeviceModelUpdateRequest struct {
	Name         *string
	Code         *string
	TypeEUI      []byte // 8-byte MIOTY Type EUI
	Description  *string
	DatasheetURL *string
}

// BlueprintCreateRequest contains fields for creating a blueprint
// Maps to models.BlueprintCreateParams: TypeEUI optional (falls back to device model), SpecJSON=payload decode spec
type BlueprintCreateRequest struct {
	DeviceModelID uuid.UUID
	Version       string          // Semantic version (e.g., "1.0.0")
	TypeEUI       []byte          // 8-byte MIOTY Type EUI (optional, resolved via device model)
	SpecJSON      json.RawMessage // Blueprint specification JSON
	IsDefault     bool
	IsSystem      bool // admin-only: create a System catalog row (tenant_id NULL)
}

// BlueprintUpdateRequest contains fields for updating a blueprint
type BlueprintUpdateRequest struct {
	Version   *string
	TypeEUI   []byte          // 8-byte MIOTY Type EUI
	SpecJSON  json.RawMessage // Blueprint specification JSON
	IsDefault *bool
}

// RegistrySubmitRequest contains fields for submitting a blueprint to an external registry.
type RegistrySubmitRequest struct {
	ContributorName  string // Required: Name of the contributor
	ContributorEmail string // Required: Email of the contributor
	Notes            string // Optional: Notes about the submission
}

// RegistrySubmitResult contains the result of a registry submission.
type RegistrySubmitResult struct {
	PRUrl      string // URL to the submission request (e.g., pull request)
	CommitSHA  string // SHA of the commit
	BranchName string // Name of the created branch
	RepoPath   string // Path to the blueprint in the repository
}

// DeviceModelWithBlueprintRequest contains fields for atomic model+blueprint creation.
type DeviceModelWithBlueprintRequest struct {
	ManufacturerID uuid.UUID
	Name           string          // Model name (slug auto-generated)
	Version        string          // Blueprint version (e.g., "1.0.0")
	DecoderScript  json.RawMessage // Blueprint specification JSON
	IsSystem       bool            // admin-only: create System catalog rows (tenant_id NULL)
}

// DecodePreviewResult contains the result of a decode preview operation.
type DecodePreviewResult struct {
	Success          bool
	DecodedPayload   json.RawMessage // JSON-encoded decoded fields
	ErrorCode        string          // Error token from decoder
	ErrorDetail      string          // Human-readable error detail
	FormatID         uint8
	BlueprintVersion string
}

// Message direction constants
const (
	// DirectionUplink indicates uplink message direction
	DirectionUplink = "uplink"
	// DirectionDownlink indicates downlink message direction
	DirectionDownlink = "downlink"
)

// MessageFilters contains filters for message queries
type MessageFilters struct {
	EpEui     []byte
	BsEui     []byte
	StartTime *time.Time
	EndTime   *time.Time
	MinRSSI   *float64
	MaxRSSI   *float64
	Direction string // Filter by direction: DirectionUplink | DirectionDownlink | "" (all)
}

// IntegrationCreateRequest contains fields for creating an integration
type IntegrationCreateRequest struct {
	OrgID          uuid.UUID       // Organization ID
	Name           string          // Integration name
	Description    string          // Optional description
	Type           string          // http, mqtt, database
	Config         json.RawMessage // Type-specific configuration
	EventFilter    json.RawMessage // Optional event filtering rules
	DeliveryFormat string          // json (default)
	CreatedBy      string          // User ID who created
}

// IntegrationUpdateRequest contains fields for updating an integration
type IntegrationUpdateRequest struct {
	Name        *string         // Optional name update
	Description *string         // Optional description update
	Config      json.RawMessage // Optional config update
	EventFilter json.RawMessage // Optional event filter update
	Status      *string         // Optional status update (active, paused, disabled)
	UpdatedBy   string          // User ID who updated
}

// ActivityService handles unified activity feed combining events and messages.
type ActivityService interface {
	// ListBaseStationActivity returns merged events and messages for a base station.
	// Items are sorted by timestamp descending with unified pagination.
	ListBaseStationActivity(ctx context.Context, tenantID int64, bsEui []byte, filters *ActivityFilters, pageSize int, pageToken string) (*ActivityListResult, error)

	// ListEndpointActivity returns merged events and messages for an endpoint.
	// Items are sorted by timestamp descending with unified pagination.
	ListEndpointActivity(ctx context.Context, tenantID int64, epEui []byte, filters *ActivityFilters, pageSize int, pageToken string) (*ActivityListResult, error)
}

// StatisticsService provides aggregated statistics across endpoints, base stations, and messages.
type StatisticsService interface {
	GetStatistics(ctx context.Context, tenantID int64, startTime, endTime *time.Time, granularity string) (*StatisticsResult, error)
}

// StatisticsResult contains aggregated statistics for a tenant.
type StatisticsResult struct {
	TotalMessages            int64
	TotalEndpoints           int64
	TotalBaseStations        int64
	MessageCounts            []TimeSeriesPoint
	EndpointMessageCounts    map[string]int64
	BaseStationMessageCounts map[string]int64
}

// TimeSeriesPoint represents a single data point in a time series.
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     int64
}

// ActivityFilters contains filters for activity queries
type ActivityFilters struct {
	StartTime *time.Time
	EndTime   *time.Time
}

// ActivityItemType identifies the type of activity item
type ActivityItemType string

const (
	// ActivityItemTypeEvent indicates an event activity item
	ActivityItemTypeEvent ActivityItemType = "event"
	// ActivityItemTypeMessage indicates a message activity item
	ActivityItemTypeMessage ActivityItemType = "message"
)

// ActivityItem represents a unified activity item (event or message)
type ActivityItem struct {
	Type       ActivityItemType
	OccurredAt time.Time
	Event      *Event               // Set if Type == ActivityItemTypeEvent
	Message    *mioty.ULDataMessage // Set if Type == ActivityItemTypeMessage
}

// ActivityListResult contains the result of activity listing
type ActivityListResult struct {
	Items         []*ActivityItem
	NextPageToken string
	TotalCount    int64
}
