import type { BlueprintScope } from "@api-types/api";

import { BLUEPRINT_SCOPE } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";

/** The catalog a manufacturer, model or blueprint belongs to. */
export const catalogScopeOf = (entry: { isSystem: boolean }): BlueprintScope =>
  entry.isSystem ? BLUEPRINT_SCOPE.SYSTEM : BLUEPRINT_SCOPE.CUSTOM;

/** The display name of a catalog. */
export const catalogScopeLabel = (scope: BlueprintScope): string =>
  scope === BLUEPRINT_SCOPE.SYSTEM
    ? BLUEPRINT_LABELS.SCOPE_SYSTEM
    : BLUEPRINT_LABELS.SCOPE_CUSTOM;

/** An entry's name with its catalog, for lists that mix both catalogs. */
export const catalogEntryLabel = (entry: {
  name: string;
  isSystem: boolean;
}): string =>
  `${entry.name}${BLUEPRINT_LABELS.CATALOG_ENTRY_SEPARATOR}${catalogScopeLabel(catalogScopeOf(entry))}`;
