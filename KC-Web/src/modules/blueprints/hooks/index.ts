/**
 * Blueprint Domain Hooks
 *
 * Centralized exports for all blueprint-related React Query hooks.
 */

// Manufacturer hooks
export {
  useCreateManufacturer,
  useDeleteManufacturer,
  useManufacturers,
  useUpdateManufacturer,
} from './useManufacturers';

// Device Model hooks
export {
  useCreateDeviceModel,
  useDeleteDeviceModel,
  useDeviceModels,
  useUpdateDeviceModel,
} from './useDeviceModels';

// Blueprint hooks
export {
  useBlueprint,
  useBlueprints,
  useCreateBlueprint,
  useDecodePreview,
  useDeleteBlueprint,
  useSetBlueprintDefault,
  useUpdateBlueprint,
} from './useBlueprints';
