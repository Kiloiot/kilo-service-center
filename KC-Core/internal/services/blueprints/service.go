// Package blueprints provides blueprint service implementation for gRPC layer.
package blueprints

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	blueprintconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
)

// BlueprintTx is the transaction-scoped operation set used when a device
// model and its blueprint are created together, which must succeed or fail as
// a unit.
type BlueprintTx interface {
	CreateDeviceModel(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error)
	CreateBlueprint(ctx context.Context, params *models.BlueprintCreateParams) (*models.Blueprint, error)
}

// BlueprintTransactionRunner runs a device-model-plus-blueprint creation
// inside a transaction. Storage owns begin, commit and rollback; this service
// only says what happens between them.
type BlueprintTransactionRunner interface {
	Run(ctx context.Context, fn func(BlueprintTx) error) error
}

// ManufacturerStore persists manufacturers in the device catalog.
type ManufacturerStore interface {
	Create(ctx context.Context, params *models.ManufacturerCreateParams) (*models.Manufacturer, error)
	GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Manufacturer, error)
	List(ctx context.Context, params *models.ManufacturerListParams) ([]*models.Manufacturer, error)
	Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error)
	Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.ManufacturerUpdateParams) error
	Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error
}

// DeviceModelStore persists device models in the device catalog.
type DeviceModelStore interface {
	Create(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error)
	GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error)
	List(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModel, error)
	Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error)
	CountByManufacturer(ctx context.Context, tenantID int64, isSystem bool, manufacturerID uuid.UUID) (int64, error)
	Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.DeviceModelUpdateParams) error
	Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error
}

// BlueprintStore persists blueprints in the device catalog.
type BlueprintStore interface {
	Create(ctx context.Context, params *models.BlueprintCreateParams) (*models.Blueprint, error)
	GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Blueprint, error)
	GetDefaultForModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (*models.Blueprint, error)
	List(ctx context.Context, params *models.BlueprintListParams) ([]*models.Blueprint, error)
	Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error)
	CountByDeviceModel(ctx context.Context, tenantID int64, isSystem bool, deviceModelID uuid.UUID) (int64, error)
	Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.BlueprintUpdateParams) error
	SetDefault(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error
	UpdateRegistryInfo(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, repo, commitSHA, prURL string, verified bool) error
	Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error
}

// Service implements grpcservices.BlueprintService.
type Service struct {
	manufacturerRepo ManufacturerStore
	deviceModelRepo  DeviceModelStore
	blueprintRepo    BlueprintStore
	tenantID         int64 // Default tenant ID for single-tenant deployments
	registryCfg      *config.RegistryProviderConfig
	logger           logger.Logger
	txRunner         BlueprintTransactionRunner
	decoder          PayloadDecoder
}

// PayloadDecoder decodes an uplink payload against a blueprint specification
// for the preview endpoints.
type PayloadDecoder interface {
	Decode(ctx context.Context, bp *models.Blueprint, userData []byte, formatID uint8,
		calibration map[string]interface{}) (*blueprintconstants.DecodeResult, error)
}

// New creates a new blueprint service. Every collaborator is supplied here:
// the transaction runner and the decoder were previously attached afterwards,
// which left a window in which the service existed but could not do its work.
func New(
	manufacturerRepo ManufacturerStore,
	deviceModelRepo DeviceModelStore,
	blueprintRepo BlueprintStore,
	tenantID int64,
	registryCfg *config.RegistryProviderConfig,
	log logger.Logger,
	txRunner BlueprintTransactionRunner,
	decoder PayloadDecoder,
) *Service {
	return &Service{
		manufacturerRepo: manufacturerRepo,
		deviceModelRepo:  deviceModelRepo,
		blueprintRepo:    blueprintRepo,
		tenantID:         tenantID,
		registryCfg:      registryCfg,
		logger:           log,
		txRunner:         txRunner,
		decoder:          decoder,
	}
}

func (s *Service) tenantIDFromContext(ctx context.Context) (int64, error) {
	tenantID, err := pkgcontext.GetTenantID(ctx)
	if err == nil && tenantID > 0 {
		return tenantID, nil
	}
	if s.tenantID > 0 {
		return s.tenantID, nil
	}

	return 0, ErrTenantIDRequired
}

// Ensure Service implements grpcservices.BlueprintService
var _ grpcservices.BlueprintService = (*Service)(nil)
