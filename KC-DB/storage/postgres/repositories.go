package postgres

// Repositories is the eager set of every repository built over one
// connection. Composition roots hand out its fields; nothing is constructed
// lazily and nothing is looked up by name.
type Repositories struct {
	BaseStations        *BaseStationRepository
	BaseStationSessions *BaseStationSessionRepository
	BaseStationStatus   *BaseStationStatusRepository
	BaseStationMetrics  *BaseStationMetricsRepository
	Endpoints           *EndPointRepository
	EndpointSessions    *EndPointSessionRepository
	Organizations       *OrganizationRepository
	Tenants             *TenantRepository
	Users               *UserRepository
	RefreshTokens       *RefreshTokenRepository
	Registrations       *RegistrationRepository
	APIKeys             *APIKeyRepository
	Integrations        *IntegrationRepository
	OperationStatus     *OperationStatusRepository
	Manufacturers       *ManufacturerRepository
	DeviceModels        *DeviceModelRepository
	Blueprints          *BlueprintRepository
	SystemEvents        *SystemEventStore
	SCACIEvents         *SCACIEventStore
	Messages            *MessageRepository
	MessageAnalytics    *MessageAnalyticsRepository
	UplinkStore         *UplinkStore
	DeliveryOutbox      *MessageDeliveryOutboxRepository
	DLRXStatus          *DLRXStatusRepository
	PendingOperations   *PendingOperationRepository
	SCACISessions       *SCACISessionRepository
	SCACIOperations     *SCACIOperationRepository
	Downlinks           *MIOTYDownlinkRepository
	DownlinkQueueReader *DownlinkQueueReader
	Roaming             *RoamingRepository
	FederationOutbox    *FederationOutboxRepository
	CEInstallations     *CEInstallationRepository
}

// NewRepositories builds every repository over the connection's handles,
// cipher and clock.
func NewRepositories(db *DB) *Repositories {
	queueReader := NewDownlinkQueueReader(db.sqlxDB, db.log)
	systemEvents := NewSystemEventStore(db.conn, db.clock, db.log)
	return &Repositories{
		BaseStations:        NewBaseStationRepository(db.sqlxDB, db.clock, db.log),
		BaseStationSessions: NewBaseStationSessionRepository(db.sqlxDB, db.clock, db.log),
		BaseStationStatus:   NewBaseStationStatusRepository(db.sqlxDB, db.clock),
		BaseStationMetrics:  NewBaseStationMetricsRepository(db.sqlxDB),
		Endpoints:           NewEndPointRepository(db.sqlxDB, db.cipher, db.clock, db.log),
		EndpointSessions:    NewEndPointSessionRepository(db.sqlxDB, db.cipher, db.log),
		Organizations:       NewOrganizationRepository(db.sqlxDB, db.log),
		Tenants:             NewTenantRepository(db.sqlxDB),
		Users:               NewUserRepository(db.sqlxDB, db.clock),
		RefreshTokens:       NewRefreshTokenRepository(db.sqlxDB, db.clock),
		Registrations:       NewRegistrationRepository(db.sqlxDB, db.clock),
		APIKeys:             NewAPIKeyRepository(db.sqlxDB, db.clock),
		Integrations:        NewIntegrationRepository(db.sqlxDB, db.clock),
		OperationStatus:     NewOperationStatusRepository(db.sqlxDB, db.clock),
		Manufacturers:       NewManufacturerRepository(db.sqlxDB, db.log),
		DeviceModels:        NewDeviceModelRepository(db.sqlxDB, db.log),
		Blueprints:          NewBlueprintRepository(db.sqlxDB),
		SystemEvents:        systemEvents,
		SCACIEvents:         NewSCACIEventStore(db.conn, systemEvents, db.clock, db.log),
		Messages:            NewMessageRepository(db.sqlxDB, db.clock, db.log),
		MessageAnalytics:    NewMessageAnalyticsRepository(db.sqlxDB),
		UplinkStore:         NewUplinkStore(db.sqlxDB, db.clock, db.log),
		DeliveryOutbox:      NewMessageDeliveryOutboxRepository(db.sqlxDB, db.clock),
		DLRXStatus:          NewDLRXStatusRepository(db.sqlxDB, db.log),
		PendingOperations:   NewPendingOperationRepository(db.sqlxDB, db.log),
		SCACISessions:       NewSCACISessionRepository(db.sqlxDB, db.clock, db.log),
		SCACIOperations:     NewSCACIOperationRepository(db.sqlxDB, db.log, db.clock),
		Downlinks:           NewMIOTYDownlinkRepository(db.sqlxDB, queueReader, db.clock, db.log),
		DownlinkQueueReader: queueReader,
		Roaming:             NewRoamingRepository(db.sqlxDB),
		FederationOutbox:    NewFederationOutboxRepository(db.sqlxDB, db.clock),
		CEInstallations:     NewCEInstallationRepository(db.sqlxDB),
	}
}
