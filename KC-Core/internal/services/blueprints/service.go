// Package blueprints provides blueprint service implementation for gRPC layer.
package blueprints

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	blueprintdecoder "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	blueprintconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
)

// fallbackModelSlug is used when slug generation strips every character.
const fallbackModelSlug = "model"

// Sentinel errors for blueprint operations.
var (
	ErrManufacturerNotFound = errors.New("manufacturer not found")
	ErrDeviceModelNotFound  = errors.New("device model not found")
	ErrBlueprintNotFound    = errors.New("blueprint not found")
	ErrTenantIDRequired     = errors.New("tenant ID required")
	// ErrOwnershipMismatch enforces a child's is_system to equal its parent's, else the per-model default index and model-code uniqueness break.
	ErrOwnershipMismatch    = errors.New("ownership mismatch: is_system must match parent")
	ErrInvalidTypeEUIFormat = errors.New("invalid type EUI format")
	ErrMissingTypeEUI       = errors.New("blueprint spec is missing required typeEui field")
	ErrSlugGenerationFailed = errors.New("slug generation exhausted all suffix attempts")
)

// Registry operation sentinel errors (mapped to gRPC tokens in handler)
var (
	// ErrRegistryDisabled is returned when registry submission is attempted but not enabled
	ErrRegistryDisabled = errors.New("registry provider is not enabled")

	// ErrAlreadySubmitted is returned when blueprint already has a registry PR URL
	ErrAlreadySubmitted = errors.New("blueprint already submitted to registry")

	// ErrBranchCreateFailed is returned when branch creation fails
	ErrBranchCreateFailed = errors.New("failed to create registry branch")

	// ErrCommitFailed is returned when commit operation fails
	ErrCommitFailed = errors.New("failed to commit to registry")

	// ErrSubmissionCreateFailed is returned when submission request creation fails
	ErrSubmissionCreateFailed = errors.New("failed to create submission request")

	// ErrRegistryAPIURLRequired is returned when enabled but api_url is empty
	ErrRegistryAPIURLRequired = errors.New("registry api_url is required when enabled")

	// ErrRegistryAuthFailed is returned when registry authentication fails (401)
	ErrRegistryAuthFailed = errors.New("registry authentication failed")

	// ErrRegistryPermissionDenied is returned when registry token lacks required permissions (403)
	ErrRegistryPermissionDenied = errors.New("registry token lacks required permissions")

	// ErrRegistryRateLimited is returned when registry rate limit is exceeded (429)
	ErrRegistryRateLimited = errors.New("registry rate limit exceeded")

	// ErrRegistryAPIError is returned for other registry API errors
	ErrRegistryAPIError = errors.New("registry API error")

	// ErrRegistryVersionAlreadyExists is returned when the blueprint version already exists in the registry
	ErrRegistryVersionAlreadyExists = errors.New("blueprint version already exists in registry")

	// ErrInvalidRegistryPathSegment is returned when a registry path segment is invalid (empty after sanitization)
	ErrInvalidRegistryPathSegment = errors.New("invalid registry path segment")

	// ErrInvalidModelCode is returned when a model code fails canonicalization validation
	ErrInvalidModelCode = errors.New("invalid model code: must contain only lowercase alphanumeric characters and hyphens")
)

// TxStarter provides transaction creation capability.
type TxStarter interface {
	BeginTx(ctx context.Context) (interfaces.Transaction, error)
}

// Service implements grpcservices.BlueprintService.
type Service struct {
	manufacturerRepo interfaces.ManufacturerRepository
	deviceModelRepo  interfaces.DeviceModelRepository
	blueprintRepo    interfaces.BlueprintRepository
	tenantID         int64 // Default tenant ID for single-tenant deployments
	registryCfg      *config.RegistryProviderConfig
	logger           logger.Logger
	txStarter        TxStarter
	decoder          *blueprintdecoder.DecoderService
}

// New creates a new blueprint service.
func New(
	manufacturerRepo interfaces.ManufacturerRepository,
	deviceModelRepo interfaces.DeviceModelRepository,
	blueprintRepo interfaces.BlueprintRepository,
	tenantID int64,
	registryCfg *config.RegistryProviderConfig,
	log logger.Logger,
) *Service {
	return &Service{
		manufacturerRepo: manufacturerRepo,
		deviceModelRepo:  deviceModelRepo,
		blueprintRepo:    blueprintRepo,
		tenantID:         tenantID,
		registryCfg:      registryCfg,
		logger:           log,
	}
}

// WithTxStarter sets the transaction starter for atomic operations.
func (s *Service) WithTxStarter(tx TxStarter) *Service {
	s.txStarter = tx
	return s
}

// WithDecoder sets the blueprint decoder for preview operations.
func (s *Service) WithDecoder(d *blueprintdecoder.DecoderService) *Service {
	s.decoder = d
	return s
}

// slugRegex strips non-alphanumeric characters (except hyphens) for slug generation.
var slugRegex = regexp.MustCompile(`[^a-z0-9-]`)

// versionSegmentRegex allows only characters valid in semver path segments.
var versionSegmentRegex = regexp.MustCompile(`[^a-z0-9._-]`)

// modelCodeRegex matches characters NOT allowed in persisted model codes.
var modelCodeRegex = regexp.MustCompile(`[^a-z0-9-]`)

// generateSlug creates a URL-friendly slug from a model name.
func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", blueprintconstants.ModelCodeSeparator)
	slug = slugRegex.ReplaceAllString(slug, "")
	// Collapse consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > blueprintconstants.ModelCodeMaxLength {
		slug = slug[:blueprintconstants.ModelCodeMaxLength]
	}
	if slug == "" {
		slug = fallbackModelSlug
	}
	return slug
}

// canonicalizeModelCode validates and normalizes a model code for persistence.
// Applies ToLower + TrimSpace (lossless canonicalization), then strict-rejects if any
// illegal characters remain. Valid codes contain only [a-z0-9-], no leading/trailing
// hyphens, no consecutive dots, and fit within ModelCodeMaxLength.
func canonicalizeModelCode(code string) (string, error) {
	canonical := strings.ToLower(strings.TrimSpace(code))
	if canonical == "" || modelCodeRegex.MatchString(canonical) ||
		strings.Contains(canonical, "..") ||
		strings.HasPrefix(canonical, "-") || strings.HasSuffix(canonical, "-") {
		return "", ErrInvalidModelCode
	}
	if len(canonical) > blueprintconstants.ModelCodeMaxLength {
		return "", ErrInvalidModelCode
	}
	return canonical, nil
}

// extractTypeEUIFromSpec parses and validates typeEui from blueprint spec JSON.
// Returns 8-byte TypeEUI or error if missing/invalid. Per MIOTY spec section 2.2.3:
// "The version and typeEUI keys are mandatory in the structure's root."
func extractTypeEUIFromSpec(specJSON []byte) ([]byte, error) {
	var peek struct {
		TypeEUI string `json:"typeEui"`
	}
	if err := json.Unmarshal(specJSON, &peek); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	if peek.TypeEUI == "" {
		return nil, ErrMissingTypeEUI
	}
	b, err := hex.DecodeString(peek.TypeEUI)
	if err != nil || len(b) != blueprintconstants.TypeEUILength {
		return nil, ErrInvalidTypeEUIFormat
	}
	return b, nil
}

// CreateManufacturer creates a new manufacturer.
func (s *Service) CreateManufacturer(ctx context.Context, req *grpcservices.ManufacturerCreateRequest) (*models.Manufacturer, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var website *string
	if req.Website != "" {
		website = &req.Website
	}

	params := &models.ManufacturerCreateParams{
		TenantID: tenantID,
		IsSystem: req.IsSystem,
		Name:     req.Name,
		Website:  website,
	}

	manufacturer, err := s.manufacturerRepo.Create(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create manufacturer", "name", req.Name, "error", err)
		return nil, fmt.Errorf("create manufacturer: %w", err)
	}

	s.logger.InfoContext(ctx, "manufacturer created", "id", manufacturer.ID, "name", manufacturer.Name)
	return manufacturer, nil
}

// GetManufacturer retrieves a manufacturer by ID.
func (s *Service) GetManufacturer(ctx context.Context, id uuid.UUID) (*models.Manufacturer, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	manufacturer, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get manufacturer", "id", id, "error", err)
		return nil, fmt.Errorf("get manufacturer: %w", err)
	}
	return manufacturer, nil
}

// UpdateManufacturer updates an existing manufacturer.
func (s *Service) UpdateManufacturer(ctx context.Context, id uuid.UUID, req *grpcservices.ManufacturerUpdateRequest) (*models.Manufacturer, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		return nil, fmt.Errorf("get manufacturer: %w", err)
	}

	// Build update params
	params := &models.ManufacturerUpdateParams{}
	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Website != nil {
		params.Website = req.Website
	}

	if err := s.manufacturerRepo.Update(ctx, tenantID, existing.IsSystem, id, params); err != nil {
		s.logger.ErrorContext(ctx, "failed to update manufacturer", "id", id, "error", err)
		return nil, fmt.Errorf("update manufacturer: %w", err)
	}

	// Fetch updated manufacturer
	manufacturer, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get updated manufacturer: %w", err)
	}

	s.logger.InfoContext(ctx, "manufacturer updated", "id", id)
	return manufacturer, nil
}

// DeleteManufacturer deletes a manufacturer.
func (s *Service) DeleteManufacturer(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrManufacturerNotFound
		}
		return fmt.Errorf("get manufacturer: %w", err)
	}

	if err := s.manufacturerRepo.Delete(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrManufacturerNotFound
		}
		s.logger.ErrorContext(ctx, "failed to delete manufacturer", "id", id, "error", err)
		return fmt.Errorf("delete manufacturer: %w", err)
	}

	s.logger.InfoContext(ctx, "manufacturer deleted", "id", id)
	return nil
}

// ListManufacturers returns a list of manufacturers.
func (s *Service) ListManufacturers(ctx context.Context, isSystem bool, limit, offset int) ([]*models.Manufacturer, int64, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	params := &models.ManufacturerListParams{
		TenantID: tenantID,
		IsSystem: isSystem,
		Limit:    limit,
		Offset:   offset,
	}

	manufacturers, err := s.manufacturerRepo.List(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list manufacturers", "error", err)
		return nil, 0, fmt.Errorf("list manufacturers: %w", err)
	}

	total, err := s.manufacturerRepo.Count(ctx, tenantID, isSystem)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to count manufacturers", "error", err)
		return nil, 0, fmt.Errorf("count manufacturers: %w", err)
	}

	return manufacturers, total, nil
}

// CreateDeviceModel creates a new device model.
func (s *Service) CreateDeviceModel(ctx context.Context, req *grpcservices.DeviceModelCreateRequest) (*models.DeviceModel, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	mfr, err := s.manufacturerRepo.GetByID(ctx, tenantID, req.ManufacturerID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		return nil, fmt.Errorf("get manufacturer: %w", err)
	}
	// Ownership homogeneity: a Custom model can't hang off a System manufacturer (or vice versa).
	if mfr.IsSystem != req.IsSystem {
		return nil, ErrOwnershipMismatch
	}

	// Canonicalize model code: generate slug from name if empty, validate if provided
	code := req.Code
	if code == "" {
		code = generateSlug(req.Name)
	} else {
		var err error
		code, err = canonicalizeModelCode(code)
		if err != nil {
			return nil, err
		}
	}

	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	var datasheetURL *string
	if req.DatasheetURL != "" {
		datasheetURL = &req.DatasheetURL
	}

	params := &models.DeviceModelCreateParams{
		TenantID:       tenantID,
		IsSystem:       req.IsSystem,
		ManufacturerID: req.ManufacturerID,
		Name:           req.Name,
		Code:           code,
		TypeEUI:        req.TypeEUI,
		Description:    description,
		DatasheetURL:   datasheetURL,
	}

	model, err := s.deviceModelRepo.Create(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create device model", "name", req.Name, "error", err)
		return nil, fmt.Errorf("create device model: %w", err)
	}

	s.logger.InfoContext(ctx, "device model created", "id", model.ID, "name", model.Name)
	return model, nil
}

// GetDeviceModel retrieves a device model by ID.
func (s *Service) GetDeviceModel(ctx context.Context, id uuid.UUID) (*models.DeviceModel, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get device model", "id", id, "error", err)
		return nil, fmt.Errorf("get device model: %w", err)
	}
	return model, nil
}

// GetDeviceModelForTenant retrieves a device model by ID using an explicit tenant ID
// instead of the service's default tenant. Used for request-context tenant validation.
func (s *Service) GetDeviceModelForTenant(
	ctx context.Context,
	tenantID int64,
	id uuid.UUID,
) (*models.DeviceModel, error) {
	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get device model for tenant",
			"tenantID", tenantID, "id", id, "error", err)
		return nil, fmt.Errorf("get device model for tenant: %w", err)
	}
	return model, nil
}

// UpdateDeviceModel updates an existing device model.
func (s *Service) UpdateDeviceModel(ctx context.Context, id uuid.UUID, req *grpcservices.DeviceModelUpdateRequest) (*models.DeviceModel, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		return nil, fmt.Errorf("get device model: %w", err)
	}

	// Build update params
	params := &models.DeviceModelUpdateParams{}
	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Code != nil {
		canonicalized, err := canonicalizeModelCode(*req.Code)
		if err != nil {
			return nil, err
		}
		params.Code = &canonicalized
	}
	if req.TypeEUI != nil {
		params.TypeEUI = req.TypeEUI
	}
	if req.Description != nil {
		params.Description = req.Description
	}
	if req.DatasheetURL != nil {
		params.DatasheetURL = req.DatasheetURL
	}

	if err := s.deviceModelRepo.Update(ctx, tenantID, existing.IsSystem, id, params); err != nil {
		s.logger.ErrorContext(ctx, "failed to update device model", "id", id, "error", err)
		return nil, fmt.Errorf("update device model: %w", err)
	}

	// Fetch updated model
	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get updated device model: %w", err)
	}

	s.logger.InfoContext(ctx, "device model updated", "id", id)
	return model, nil
}

// DeleteDeviceModel deletes a device model.
func (s *Service) DeleteDeviceModel(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrDeviceModelNotFound
		}
		return fmt.Errorf("get device model: %w", err)
	}

	if err := s.deviceModelRepo.Delete(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrDeviceModelNotFound
		}
		s.logger.ErrorContext(ctx, "failed to delete device model", "id", id, "error", err)
		return fmt.Errorf("delete device model: %w", err)
	}

	s.logger.InfoContext(ctx, "device model deleted", "id", id)
	return nil
}

// ListDeviceModels returns a list of device models.
func (s *Service) ListDeviceModels(ctx context.Context, isSystem bool, manufacturerID *uuid.UUID, limit, offset int) ([]*models.DeviceModel, int64, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	params := &models.DeviceModelListParams{
		TenantID: tenantID,
		IsSystem: isSystem,
		Limit:    limit,
		Offset:   offset,
	}
	if manufacturerID != nil {
		params.ManufacturerID = manufacturerID
	}

	models, err := s.deviceModelRepo.List(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list device models", "error", err)
		return nil, 0, fmt.Errorf("list device models: %w", err)
	}

	// Use appropriate count method based on filter
	var total int64
	if manufacturerID != nil {
		total, err = s.deviceModelRepo.CountByManufacturer(ctx, tenantID, isSystem, *manufacturerID)
	} else {
		total, err = s.deviceModelRepo.Count(ctx, tenantID, isSystem)
	}
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to count device models", "error", err)
		return nil, 0, fmt.Errorf("count device models: %w", err)
	}

	return models, total, nil
}

// CreateBlueprint creates a new blueprint.
func (s *Service) CreateBlueprint(ctx context.Context, req *grpcservices.BlueprintCreateRequest) (*models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, req.DeviceModelID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		return nil, fmt.Errorf("get device model: %w", err)
	}
	// Ownership homogeneity: a Custom blueprint can't hang off a System model (or vice versa).
	if model.IsSystem != req.IsSystem {
		return nil, ErrOwnershipMismatch
	}

	// Extract typeEui from spec JSON if provided (per MIOTY spec section 2.2.3)
	typeEUI := req.TypeEUI
	if len(req.SpecJSON) > 0 {
		specTypeEUI, err := extractTypeEUIFromSpec(req.SpecJSON)
		if err != nil {
			return nil, err
		}
		// Cross-validate: if request also provides TypeEUI, it must match spec
		if len(typeEUI) > 0 && !bytes.Equal(typeEUI, specTypeEUI) {
			return nil, ErrInvalidTypeEUIFormat
		}
		typeEUI = specTypeEUI
	}
	if len(typeEUI) == 0 {
		typeEUI = model.TypeEUI
	}
	if len(typeEUI) > 0 && len(typeEUI) != blueprintconstants.TypeEUILength {
		return nil, ErrInvalidTypeEUIFormat
	}

	params := &models.BlueprintCreateParams{
		TenantID:      tenantID,
		IsSystem:      req.IsSystem,
		DeviceModelID: req.DeviceModelID,
		Version:       req.Version,
		TypeEUI:       typeEUI,
		SpecJSON:      req.SpecJSON,
		IsDefault:     req.IsDefault,
	}

	// Auto-default: when the model has no current default, this blueprint becomes it so snapshot-less endpoints still resolve.
	if !params.IsDefault {
		def, defErr := s.blueprintRepo.GetDefaultForModel(ctx, tenantID, req.DeviceModelID)
		if defErr != nil {
			return nil, fmt.Errorf("check default blueprint for model: %w", defErr)
		}
		if def == nil {
			params.IsDefault = true
		}
	}

	blueprint, err := s.blueprintRepo.Create(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create blueprint", "version", req.Version, "error", err)
		return nil, fmt.Errorf("create blueprint: %w", err)
	}

	s.logger.InfoContext(ctx, "blueprint created", "id", blueprint.ID, "version", blueprint.Version)
	return blueprint, nil
}

// GetBlueprint retrieves a blueprint by ID.
func (s *Service) GetBlueprint(ctx context.Context, id uuid.UUID) (*models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	blueprint, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get blueprint", "id", id, "error", err)
		return nil, fmt.Errorf("get blueprint: %w", err)
	}
	return blueprint, nil
}

// UpdateBlueprint updates an existing blueprint.
func (s *Service) UpdateBlueprint(ctx context.Context, id uuid.UUID, req *grpcservices.BlueprintUpdateRequest) (*models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		return nil, fmt.Errorf("get blueprint: %w", err)
	}

	// Build update params
	params := &models.BlueprintUpdateParams{}
	if req.Version != nil {
		params.Version = req.Version
	}
	if req.TypeEUI != nil {
		params.TypeEUI = req.TypeEUI
	}
	// If SpecJSON is updated, extract and persist typeEui from it (per section 2.2.3)
	if len(req.SpecJSON) > 0 {
		params.SpecJSON = req.SpecJSON
		specTypeEUI, err := extractTypeEUIFromSpec(req.SpecJSON)
		if err != nil {
			return nil, err
		}
		// Cross-validate: if request also provides TypeEUI, it must match spec
		if req.TypeEUI != nil && !bytes.Equal(req.TypeEUI, specTypeEUI) {
			return nil, ErrInvalidTypeEUIFormat
		}
		params.TypeEUI = specTypeEUI
	}
	if req.IsDefault != nil {
		params.IsDefault = req.IsDefault
	}

	if err := s.blueprintRepo.Update(ctx, tenantID, existing.IsSystem, id, params); err != nil {
		s.logger.ErrorContext(ctx, "failed to update blueprint", "id", id, "error", err)
		return nil, fmt.Errorf("update blueprint: %w", err)
	}

	// Fetch updated blueprint
	blueprint, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get updated blueprint: %w", err)
	}

	s.logger.InfoContext(ctx, "blueprint updated", "id", id)
	return blueprint, nil
}

// DeleteBlueprint deletes a blueprint.
func (s *Service) DeleteBlueprint(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		return fmt.Errorf("get blueprint: %w", err)
	}

	if err := s.blueprintRepo.Delete(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		s.logger.ErrorContext(ctx, "failed to delete blueprint", "id", id, "error", err)
		return fmt.Errorf("delete blueprint: %w", err)
	}

	s.logger.InfoContext(ctx, "blueprint deleted", "id", id)
	return nil
}

// ListBlueprints returns a list of blueprints.
func (s *Service) ListBlueprints(ctx context.Context, isSystem bool, deviceModelID *uuid.UUID, limit, offset int) ([]*models.Blueprint, int64, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	params := &models.BlueprintListParams{
		TenantID: tenantID,
		IsSystem: isSystem,
		Limit:    limit,
		Offset:   offset,
	}
	if deviceModelID != nil {
		params.DeviceModelID = deviceModelID
	}

	blueprints, err := s.blueprintRepo.List(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list blueprints", "error", err)
		return nil, 0, fmt.Errorf("list blueprints: %w", err)
	}

	// Use appropriate count method based on filter
	var total int64
	if deviceModelID != nil {
		total, err = s.blueprintRepo.CountByDeviceModel(ctx, tenantID, isSystem, *deviceModelID)
	} else {
		total, err = s.blueprintRepo.Count(ctx, tenantID, isSystem)
	}
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to count blueprints", "error", err)
		return nil, 0, fmt.Errorf("count blueprints: %w", err)
	}

	return blueprints, total, nil
}

// SetDefaultBlueprint sets a blueprint as the default for its device model.
func (s *Service) SetDefaultBlueprint(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		return fmt.Errorf("get blueprint: %w", err)
	}

	if err := s.blueprintRepo.SetDefault(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		s.logger.ErrorContext(ctx, "failed to set default blueprint", "id", id, "error", err)
		return fmt.Errorf("set default blueprint: %w", err)
	}

	s.logger.InfoContext(ctx, "default blueprint set", "id", id)
	return nil
}

// normalizeVersion ensures a single lowercase "v" prefix on version strings.
// Examples: "1.0" → "v1.0", "v1.0" → "v1.0", "V2.0" → "v2.0"
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimLeft(v, "vV")
	return blueprintconstants.RegistryVersionPrefix + v
}

// sanitizeVersionSegment validates a version string for use in registry paths.
// Canonical transforms (ToLower + TrimSpace) are applied first, then strict validation:
// if regex stripping changes the canonical form, the input is rejected (no silent mutations).
func sanitizeVersionSegment(v string) (string, error) {
	canonical := strings.ToLower(strings.TrimSpace(v))
	cleaned := versionSegmentRegex.ReplaceAllString(canonical, "")
	if cleaned == "" || cleaned == blueprintconstants.RegistryVersionPrefix || strings.Contains(cleaned, "..") {
		return "", ErrInvalidRegistryPathSegment
	}
	if cleaned != canonical {
		return "", ErrInvalidRegistryPathSegment
	}
	return cleaned, nil
}

// sanitizePathSegment cleans a name string for use in registry directory paths.
// Uses generateSlug for manufacturer/model names. Returns error if the input has
// no meaningful alphanumeric content after slug generation.
func sanitizePathSegment(s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", ErrInvalidRegistryPathSegment
	}
	slug := generateSlug(s)
	// generateSlug returns fallbackModelSlug as fallback when all characters are stripped.
	// Reject if the slug is the fallback and the input had no alphanumeric content.
	if slug == fallbackModelSlug {
		cleaned := slugRegex.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "")
		cleaned = strings.Trim(cleaned, "-")
		if cleaned == "" {
			return "", ErrInvalidRegistryPathSegment
		}
	}
	if strings.Contains(slug, "..") {
		return "", ErrInvalidRegistryPathSegment
	}
	return slug, nil
}

// sanitizeModelCodeSegment validates a persisted model code for use in registry paths.
// Model codes are already validated at creation time, so only TrimSpace is applied.
// Rejects codes containing characters outside [a-z0-9-] or traversal sequences.
func sanitizeModelCodeSegment(code string) (string, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" || modelCodeRegex.MatchString(trimmed) || strings.Contains(trimmed, "..") {
		return "", ErrInvalidRegistryPathSegment
	}
	return trimmed, nil
}

// SubmitToRegistry submits a blueprint to an external registry.
func (s *Service) SubmitToRegistry(ctx context.Context, id uuid.UUID, req *grpcservices.RegistrySubmitRequest) (*grpcservices.RegistrySubmitResult, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Domain validation: Check if registry is enabled
	if s.registryCfg == nil || !s.registryCfg.Enabled {
		return nil, ErrRegistryDisabled
	}

	// Domain validation: api_url must be configured when enabled
	if s.registryCfg.APIURL == "" {
		return nil, ErrRegistryAPIURLRequired
	}

	// Get blueprint to verify it exists and check submission status
	bp, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		return nil, fmt.Errorf("get blueprint: %w", err)
	}

	// Domain validation: Check if already submitted (RegistryPRURL is *string)
	if bp.RegistryPRURL != nil && *bp.RegistryPRURL != "" {
		return nil, ErrAlreadySubmitted
	}

	// Resolve device model and manufacturer for structured directory naming
	deviceModel, err := s.deviceModelRepo.GetByID(ctx, tenantID, bp.DeviceModelID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		return nil, fmt.Errorf("get device model: %w", err)
	}

	manufacturer, err := s.manufacturerRepo.GetByID(ctx, tenantID, deviceModel.ManufacturerID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		return nil, fmt.Errorf("get manufacturer: %w", err)
	}

	// Build structured path segments
	mfrSlug, err := sanitizePathSegment(manufacturer.Name)
	if err != nil {
		return nil, err
	}

	// Prefer model code (URL-friendly slug), fall back to name
	var modelSlug string
	if deviceModel.Code != "" {
		modelSlug, err = sanitizeModelCodeSegment(deviceModel.Code)
	} else {
		modelSlug, err = sanitizePathSegment(deviceModel.Name)
	}
	if err != nil {
		return nil, err
	}

	normalizedVer := normalizeVersion(bp.Version)
	versionSlug, err := sanitizeVersionSegment(normalizedVer)
	if err != nil {
		return nil, err
	}

	// Generate structured file path: {manufacturer}/{model}/{version}.json
	filePath := fmt.Sprintf("%s%s/%s/%s%s", s.registryCfg.BlueprintPath, mfrSlug, modelSlug, versionSlug, s.registryCfg.FileExtension)

	// Generate branch name with short ID suffix for uniqueness
	shortID := id.String()[:8]
	branchName := fmt.Sprintf("%s%s/%s/%s/%s", s.registryCfg.BranchPrefix, mfrSlug, modelSlug, versionSlug, shortID)

	// Create registry client
	client, err := newRegistryClient(s.registryCfg, s.logger)
	if err != nil {
		return nil, fmt.Errorf("create registry client: %w", err)
	}

	// Preflight: check if file already exists in the base branch
	exists, err := client.fileExists(ctx, s.registryCfg.BaseBranch, filePath)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to check file existence", "path", filePath, "error", err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrRegistryAPIError, err)
	}
	if exists {
		return nil, ErrRegistryVersionAlreadyExists
	}

	// Get base branch SHA
	baseSHA, err := client.getBaseBranchSHA(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get base branch SHA", "error", err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrBranchCreateFailed, err)
	}

	// Create branch
	if err := client.createBranch(ctx, baseSHA, branchName); err != nil {
		s.logger.ErrorContext(ctx, "failed to create branch", "branch", branchName, "error", err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrBranchCreateFailed, err)
	}

	// Prepare blueprint content with enriched metadata
	blueprintContent, err := s.prepareBlueprintContent(bp, manufacturer.Name, deviceModel.Name, deviceModel.Code)
	if err != nil {
		return nil, fmt.Errorf("prepare content: %w", err)
	}

	authorName := req.ContributorName
	authorEmail := req.ContributorEmail

	// Commit the blueprint file using format constants
	commitMessage := fmt.Sprintf(blueprintconstants.RegistryCommitMsgFmt, manufacturer.Name, deviceModel.Name, normalizedVer)
	if req.Notes != "" {
		commitMessage = fmt.Sprintf("%s\n\n%s", commitMessage, req.Notes)
	}

	commitSHA, err := client.createOrUpdateFile(ctx, branchName, filePath, blueprintContent, commitMessage, authorName, authorEmail)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to commit file", "path", filePath, "error", err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrCommitFailed, err)
	}

	// Create submission request (pull request)
	prTitle := fmt.Sprintf(blueprintconstants.RegistryPRTitleFmt, manufacturer.Name, deviceModel.Name, normalizedVer)
	prBody := fmt.Sprintf(blueprintconstants.RegistryPRBodyFmt,
		manufacturer.Name, deviceModel.Name, normalizedVer,
		fmt.Sprintf("%016X", bp.TypeEUI), authorName, authorEmail, req.Notes)

	prURL, err := client.createSubmissionRequest(ctx, prTitle, prBody, branchName, s.registryCfg.BaseBranch)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create submission request", "error", err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrSubmissionCreateFailed, err)
	}

	// Update blueprint with registry PR URL
	repoPath := fmt.Sprintf("%s/%s", s.registryCfg.Owner, s.registryCfg.Repo)
	if err := s.blueprintRepo.UpdateRegistryInfo(ctx, tenantID, bp.IsSystem, id, repoPath, commitSHA, prURL, false); err != nil {
		s.logger.ErrorContext(ctx, "failed to update registry info", "id", id, "error", err)
		// Don't fail the operation - PR was created successfully
	}

	s.logger.InfoContext(ctx, "blueprint submitted to registry",
		"id", id,
		"pr_url", prURL,
		"commit_sha", commitSHA,
		"branch", branchName,
		"file_path", filePath,
	)

	return &grpcservices.RegistrySubmitResult{
		PRUrl:      prURL,
		CommitSHA:  commitSHA,
		BranchName: branchName,
		RepoPath:   repoPath,
	}, nil
}

// isSentinelError checks if an error is a registry sentinel that should be propagated directly.
func isSentinelError(err error) bool {
	return errors.Is(err, ErrRegistryAuthFailed) ||
		errors.Is(err, ErrRegistryPermissionDenied) ||
		errors.Is(err, ErrRegistryRateLimited) ||
		errors.Is(err, ErrRegistryAPIError) ||
		errors.Is(err, ErrRegistryVersionAlreadyExists)
}

// CreateDeviceModelWithBlueprint creates a device model and default blueprint in a single transaction.
func (s *Service) CreateDeviceModelWithBlueprint(ctx context.Context, req *grpcservices.DeviceModelWithBlueprintRequest) (*models.DeviceModel, *models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	if s.txStarter == nil {
		return nil, nil, fmt.Errorf("transaction starter not configured")
	}

	if err := s.validateDeviceModelManufacturer(ctx, tenantID, req); err != nil {
		return nil, nil, err
	}

	// Extract typeEui from spec before transaction so it's available for model creation
	specTypeEUI, err := extractTypeEUIFromSpec(req.DecoderScript)
	if err != nil {
		return nil, nil, err
	}

	tx, err := s.txStarter.BeginTx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	model, err := s.createDeviceModelWithSlug(ctx, tx, tenantID, req, specTypeEUI)
	if err != nil {
		return nil, nil, err
	}

	bpParams := &models.BlueprintCreateParams{
		TenantID:      tenantID,
		IsSystem:      req.IsSystem,
		DeviceModelID: model.ID,
		Version:       req.Version,
		TypeEUI:       specTypeEUI,
		SpecJSON:      req.DecoderScript,
		IsDefault:     true,
	}

	blueprint, err := tx.Blueprints().Create(ctx, bpParams)
	if err != nil {
		return nil, nil, fmt.Errorf("create blueprint: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.InfoContext(ctx, "device model with blueprint created",
		"model_id", model.ID, "blueprint_id", blueprint.ID, "slug", model.Code)

	return model, blueprint, nil
}

func (s *Service) validateDeviceModelManufacturer(ctx context.Context, tenantID int64, req *grpcservices.DeviceModelWithBlueprintRequest) error {
	mfr, err := s.manufacturerRepo.GetByID(ctx, tenantID, req.ManufacturerID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrManufacturerNotFound
		}
		return fmt.Errorf("get manufacturer: %w", err)
	}
	// Ownership homogeneity: model+blueprint created here inherit req.IsSystem; parent must match.
	if mfr.IsSystem != req.IsSystem {
		return ErrOwnershipMismatch
	}
	return nil
}

// Retries with a suffixed slug on code collisions; ErrSlugGenerationFailed when all attempts fail.
func (s *Service) createDeviceModelWithSlug(ctx context.Context, tx interfaces.Transaction, tenantID int64, req *grpcservices.DeviceModelWithBlueprintRequest, specTypeEUI []byte) (*models.DeviceModel, error) {
	baseSlug := generateSlug(req.Name)
	slug := baseSlug
	for attempt := 0; attempt < blueprintconstants.ModelCodeMaxSuffixAttempts; attempt++ {
		if attempt > 0 {
			slug = fmt.Sprintf("%s"+blueprintconstants.ModelCodeSuffixFormat, baseSlug, attempt+1)
			if len(slug) > blueprintconstants.ModelCodeMaxLength {
				slug = slug[:blueprintconstants.ModelCodeMaxLength]
			}
		}
		modelParams := &models.DeviceModelCreateParams{
			TenantID:       tenantID,
			IsSystem:       req.IsSystem,
			ManufacturerID: req.ManufacturerID,
			Name:           req.Name,
			Code:           slug,
			TypeEUI:        specTypeEUI,
		}
		model, createErr := tx.DeviceModels().Create(ctx, modelParams)
		if createErr != nil {
			if errors.Is(createErr, storage.ErrDuplicateKey) {
				continue
			}
			return nil, fmt.Errorf("create device model: %w", createErr)
		}
		return model, nil
	}
	return nil, ErrSlugGenerationFailed
}

// DecodePreview runs the blueprint decoder on a payload for preview.
func (s *Service) DecodePreview(ctx context.Context, blueprintID uuid.UUID, payload []byte, formatID uint8) (*grpcservices.DecodePreviewResult, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bp, err := s.blueprintRepo.GetByID(ctx, tenantID, blueprintID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		return nil, fmt.Errorf("get blueprint: %w", err)
	}

	return s.runDecodePreview(ctx, bp, payload, formatID)
}

// DecodePreviewInline previews decoding against an unsaved inline spec (no catalog fetch).
func (s *Service) DecodePreviewInline(ctx context.Context, specJSON, payload []byte, formatID uint8) (*grpcservices.DecodePreviewResult, error) {
	return s.runDecodePreview(ctx, &models.Blueprint{SpecJSON: specJSON}, payload, formatID)
}

func (s *Service) runDecodePreview(ctx context.Context, bp *models.Blueprint, payload []byte, formatID uint8) (*grpcservices.DecodePreviewResult, error) {
	if s.decoder == nil {
		return nil, fmt.Errorf("decoder service not configured")
	}

	result, err := s.decoder.Decode(ctx, bp, payload, formatID, nil)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if !result.Success {
		return &grpcservices.DecodePreviewResult{
			Success:          false,
			ErrorCode:        result.ErrorCode,
			ErrorDetail:      result.ErrorDetail,
			FormatID:         formatID,
			BlueprintVersion: bp.Version,
		}, nil
	}

	decoded, err := json.Marshal(result.DecodedData)
	if err != nil {
		return nil, fmt.Errorf("marshal decoded data: %w", err)
	}

	return &grpcservices.DecodePreviewResult{
		Success:          true,
		DecodedPayload:   decoded,
		FormatID:         formatID,
		BlueprintVersion: bp.Version,
	}, nil
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

// prepareBlueprintContent prepares the blueprint content for submission.
// Returns base64-encoded JSON content as a string (API expects base64-encoded content).
// Includes enriched metadata: manufacturer name, device model name and code.
func (s *Service) prepareBlueprintContent(bp *models.Blueprint, manufacturerName, deviceModelName, deviceModelCode string) (string, error) {
	// Create submission format with enriched metadata
	content := map[string]interface{}{
		"id":                bp.ID.String(),
		"device_model_id":   bp.DeviceModelID.String(),
		"version":           bp.Version,
		"type_eui":          fmt.Sprintf("%016X", bp.TypeEUI),
		"spec":              json.RawMessage(bp.SpecJSON),
		"is_default":        bp.IsDefault,
		"manufacturer_name": manufacturerName,
		"device_model_name": deviceModelName,
		"device_model_code": deviceModelCode,
		"created_at":        bp.CreatedAt,
		"updated_at":        bp.UpdatedAt,
	}

	jsonContent, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal blueprint: %w", err)
	}

	// Return base64-encoded string (not []byte to avoid double-encoding by JSON marshal)
	return base64.StdEncoding.EncodeToString(jsonContent), nil
}

// GetDefaultForModel returns the device model's default blueprint (nil, nil when none).
func (s *Service) GetDefaultForModel(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error) {
	return s.blueprintRepo.GetDefaultForModel(ctx, tenantID, modelID)
}

// ResolveEffectiveTypeEUI returns the default blueprint's 8-byte TypeEUI for a model, or nil when there's no default or valid TypeEUI (no fallback to DeviceModel.TypeEUI).
func (s *Service) ResolveEffectiveTypeEUI(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.EUI, error) {
	bp, err := s.blueprintRepo.GetDefaultForModel(ctx, tenantID, modelID)
	if err != nil {
		return nil, fmt.Errorf("resolve effective type EUI: %w", err)
	}
	if bp == nil {
		return nil, nil
	}
	if len(bp.TypeEUI) == 8 {
		var eui models.EUI
		copy(eui[:], bp.TypeEUI)
		return &eui, nil
	}
	return nil, nil
}

// Ensure Service implements grpcservices.BlueprintService
var _ grpcservices.BlueprintService = (*Service)(nil)
