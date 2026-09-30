package blueprints

// Log messages for manufacturer catalog operations.
const (
	LogManufacturerCreateFailed = "failed to create manufacturer"
	LogManufacturerCreated      = "manufacturer created"
	LogManufacturerGetFailed    = "failed to get manufacturer"
	LogManufacturerUpdateFailed = "failed to update manufacturer"
	LogManufacturerUpdated      = "manufacturer updated"
	LogManufacturerDeleteFailed = "failed to delete manufacturer"
	LogManufacturerDeleted      = "manufacturer deleted"
	LogManufacturerListFailed   = "failed to list manufacturers"
	LogManufacturerCountFailed  = "failed to count manufacturers"
)

// Log messages for device model catalog operations.
const (
	LogDeviceModelCreateFailed       = "failed to create device model"
	LogDeviceModelCreated            = "device model created"
	LogDeviceModelGetFailed          = "failed to get device model"
	LogDeviceModelGetForTenantFailed = "failed to get device model for tenant"
	LogDeviceModelUpdateFailed       = "failed to update device model"
	LogDeviceModelUpdated            = "device model updated"
	LogDeviceModelDeleteFailed       = "failed to delete device model"
	LogDeviceModelDeleted            = "device model deleted"
	LogDeviceModelListFailed         = "failed to list device models"
	LogDeviceModelCountFailed        = "failed to count device models"
)

// Log messages for blueprint catalog operations.
const (
	LogBlueprintCreateFailed           = "failed to create blueprint"
	LogBlueprintCreated                = "blueprint created"
	LogBlueprintGetFailed              = "failed to get blueprint"
	LogBlueprintUpdateFailed           = "failed to update blueprint"
	LogBlueprintUpdated                = "blueprint updated"
	LogBlueprintDeleteFailed           = "failed to delete blueprint"
	LogBlueprintDeleted                = "blueprint deleted"
	LogBlueprintListFailed             = "failed to list blueprints"
	LogBlueprintCountFailed            = "failed to count blueprints"
	LogBlueprintSetDefaultFailed       = "failed to set default blueprint"
	LogBlueprintDefaultSet             = "default blueprint set"
	LogDeviceModelWithBlueprintCreated = "device model with blueprint created"
)

// Log messages for registry submission operations.
const (
	LogRegistryFileCheckFailed        = "failed to check file existence"
	LogRegistryBaseSHAFailed          = "failed to get base branch SHA"
	LogRegistryBranchCreateFailed     = "failed to create branch"
	LogRegistryCommitFailed           = "failed to commit file"
	LogRegistrySubmissionCreateFailed = "failed to create submission request"
	LogRegistryInfoUpdateFailed       = "failed to update registry info"
	LogBlueprintSubmitted             = "blueprint submitted to registry"
)
