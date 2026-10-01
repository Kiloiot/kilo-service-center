/**
 * The catalog page shows at most one dialog at a time; the open dialog and
 * its subject are one value instead of six independent flags.
 */

import { useCallback, useState } from "react";

import type { DeviceModelUI, ManufacturerUI } from "@api-types/api";

export type CatalogDialog =
  | { kind: "none" }
  | { kind: "addManufacturer" }
  | { kind: "editManufacturer"; manufacturer: ManufacturerUI }
  | { kind: "deleteManufacturer"; manufacturer: ManufacturerUI }
  | { kind: "editModel"; model: DeviceModelUI }
  | { kind: "deleteModel"; model: DeviceModelUI }
  | { kind: "addBlueprint"; model: DeviceModelUI };

const NONE: CatalogDialog = { kind: "none" };

export function useCatalogDialogs() {
  const [dialog, setDialog] = useState<CatalogDialog>(NONE);
  const open = useCallback((next: CatalogDialog) => setDialog(next), []);
  const close = useCallback(() => setDialog(NONE), []);
  return { dialog, open, close };
}
