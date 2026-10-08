package models

// Device catalog event types (category: endpoint, the category of the
// endpoint managers who maintain the catalog).
const (
	EventTypeManufacturerCreated = "manufacturer.created"
	EventTypeManufacturerUpdated = "manufacturer.updated"
	EventTypeManufacturerDeleted = "manufacturer.deleted"
	EventTypeDeviceModelCreated  = "device_model.created"
	EventTypeDeviceModelUpdated  = "device_model.updated"
	EventTypeDeviceModelDeleted  = "device_model.deleted"
	EventTypeBlueprintCreated    = "blueprint.created"
	EventTypeBlueprintUpdated    = "blueprint.updated"
	EventTypeBlueprintDeleted    = "blueprint.deleted"
)

// Device catalog event titles.
const (
	EventTitleManufacturerCreated = "Manufacturer created"
	EventTitleManufacturerUpdated = "Manufacturer updated"
	EventTitleManufacturerDeleted = "Manufacturer deleted"
	EventTitleDeviceModelCreated  = "Device model created"
	EventTitleDeviceModelUpdated  = "Device model updated"
	EventTitleDeviceModelDeleted  = "Device model deleted"
	EventTitleBlueprintCreated    = "Blueprint created"
	EventTitleBlueprintUpdated    = "Blueprint updated"
	EventTitleBlueprintDeleted    = "Blueprint deleted"
)

// Device catalog event description formats, completed with the entry's name
// (a blueprint's version).
const (
	EventDescriptionManufacturerCreatedFmt = "Manufacturer %s created"
	EventDescriptionManufacturerUpdatedFmt = "Manufacturer %s updated"
	EventDescriptionManufacturerDeletedFmt = "Manufacturer %s deleted"
	EventDescriptionDeviceModelCreatedFmt  = "Device model %s created"
	EventDescriptionDeviceModelUpdatedFmt  = "Device model %s updated"
	EventDescriptionDeviceModelDeletedFmt  = "Device model %s deleted"
	EventDescriptionBlueprintCreatedFmt    = "Blueprint version %s created"
	EventDescriptionBlueprintUpdatedFmt    = "Blueprint version %s updated"
	EventDescriptionBlueprintDeletedFmt    = "Blueprint version %s deleted"
)

// Detail keys of a device catalog event: the entry's id and its name.
const (
	EventDetailKeyCatalogID   = "catalogId"
	EventDetailKeyCatalogName = "name"
)
