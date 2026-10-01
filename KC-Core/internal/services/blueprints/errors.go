package blueprints

import (
	"errors"

	blueprintconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"

	blueprintregistry "github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/blueprints"
)

// Sentinel errors for blueprint operations.
var (
	ErrManufacturerNotFound = errors.New("manufacturer not found")
	ErrDeviceModelNotFound  = errors.New("device model not found")
	ErrBlueprintNotFound    = errors.New("blueprint not found")
	ErrTenantIDRequired     = errors.New("tenant ID required")
	// ErrOwnershipMismatch enforces a child's is_system to equal its parent's, else the per-model default index and model-code uniqueness break.
	ErrOwnershipMismatch    = errors.New("ownership mismatch: is_system must match parent")
	ErrInvalidTypeEUIFormat = blueprintconstants.ErrSpecInvalidTypeEUI
	ErrMissingTypeEUI       = blueprintconstants.ErrSpecMissingTypeEUI
	ErrSlugGenerationFailed = errors.New("slug generation exhausted all suffix attempts")
	// ErrTxRunnerNotConfigured reports a combined create attempted without a transaction runner.
	ErrTxRunnerNotConfigured = errors.New("transaction starter not configured")
	// ErrDecoderNotConfigured reports a decode preview attempted without a payload decoder.
	ErrDecoderNotConfigured = errors.New("decoder service not configured")
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
	ErrRegistryAuthFailed = blueprintregistry.ErrRegistryAuthFailed

	// ErrRegistryPermissionDenied is returned when registry token lacks required permissions (403)
	ErrRegistryPermissionDenied = blueprintregistry.ErrRegistryPermissionDenied

	// ErrRegistryRateLimited is returned when registry rate limit is exceeded (429)
	ErrRegistryRateLimited = blueprintregistry.ErrRegistryRateLimited

	// ErrRegistryAPIError is returned for other registry API errors
	ErrRegistryAPIError = blueprintregistry.ErrRegistryAPIError

	// ErrRegistryVersionAlreadyExists is returned when the blueprint version already exists in the registry
	ErrRegistryVersionAlreadyExists = blueprintregistry.ErrRegistryVersionAlreadyExists

	// ErrInvalidRegistryPathSegment is returned when a registry path segment is invalid (empty after sanitization)
	ErrInvalidRegistryPathSegment = blueprintconstants.ErrRegistryPathSegment

	// ErrInvalidModelCode is returned when a model code fails canonicalization validation
	ErrInvalidModelCode = blueprintconstants.ErrModelCodeFormat
)

// Transaction lifecycle sentinels owned by this service. The composition root
// maps the storage adapter's lifecycle failures onto these, so the service and
// its callers never depend on the storage package's error identities.
var (
	ErrTxBegin  = errors.New("blueprint transaction begin")
	ErrTxCommit = errors.New("blueprint transaction commit")
)

// Error-wrap operation labels. Each names the repository, registry or codec
// operation whose failure is being wrapped, used as fmt.Errorf("%s: %w", errOp..., err).
const (
	errOpCreateManufacturer     = "create manufacturer"
	errOpGetManufacturer        = "get manufacturer"
	errOpUpdateManufacturer     = "update manufacturer"
	errOpGetUpdatedManufacturer = "get updated manufacturer"
	errOpDeleteManufacturer     = "delete manufacturer"
	errOpListManufacturers      = "list manufacturers"
	errOpCountManufacturers     = "count manufacturers"

	errOpCreateDeviceModel       = "create device model"
	errOpGetDeviceModel          = "get device model"
	errOpGetDeviceModelForTenant = "get device model for tenant"
	errOpUpdateDeviceModel       = "update device model"
	errOpGetUpdatedDeviceModel   = "get updated device model"
	errOpDeleteDeviceModel       = "delete device model"
	errOpListDeviceModels        = "list device models"
	errOpCountDeviceModels       = "count device models"

	errOpCreateBlueprint      = "create blueprint"
	errOpGetBlueprint         = "get blueprint"
	errOpCheckDefaultForModel = "check default blueprint for model"
	errOpUpdateBlueprint      = "update blueprint"
	errOpGetUpdatedBlueprint  = "get updated blueprint"
	errOpDeleteBlueprint      = "delete blueprint"
	errOpListBlueprints       = "list blueprints"
	errOpCountBlueprints      = "count blueprints"
	errOpSetDefaultBlueprint  = "set default blueprint"

	errOpCreateRegistryClient    = "create registry client"
	errOpPrepareContent          = "prepare content"
	errOpDecode                  = "decode"
	errOpMarshalDecodedData      = "marshal decoded data"
	errOpMarshalBlueprint        = "marshal blueprint"
	errOpResolveEffectiveTypeEUI = "resolve effective type EUI"
)
