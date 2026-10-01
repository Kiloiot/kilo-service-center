/**
 * Blueprint Domain Hooks
 *
 * Centralized exports for all blueprint-related React Query hooks.
 */

// Manufacturer hooks
export {
  useCatalogManufacturers,
  useCreateManufacturer,
  useDeleteManufacturer,
  useManufacturer,
  useManufacturers,
  useUpdateManufacturer,
} from "./useManufacturers";

// Device Model hooks
export {
  useDeleteDeviceModel,
  useDeviceModel,
  useDeviceModels,
  useUpdateDeviceModel,
} from "./useDeviceModels";

// Blueprint hooks
export {
  useBlueprint,
  useBlueprints,
  useBulkAssignBlueprint,
  useCreateBlueprint,
  useModelSnapshotCount,
  useSetBlueprintDefault,
  useSubmitToRegistry,
} from "./useBlueprints";

// Catalog page state
export { useAddDeviceModelForm } from "./useAddDeviceModelForm";
export { useCatalogDialogs } from "./useCatalogDialogs";
export { useCatalogTree } from "./useCatalogTree";

// Blueprint detail page state
export { useBlueprintSpecEditor } from "./useBlueprintSpecEditor";
export { useDecodePlayground } from "./useDecodePlayground";
