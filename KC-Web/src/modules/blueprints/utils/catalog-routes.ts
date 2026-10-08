import type { BlueprintScope } from "@api-types/api";

import {
  BLUEPRINT_QUERY_PARAMS,
  BLUEPRINT_SCOPE,
  ROUTES,
} from "@constants/app";

/** The catalog a page's query names; the Custom catalog when it names none. */
export const scopeFromQuery = (params: URLSearchParams): BlueprintScope =>
  params.get(BLUEPRINT_QUERY_PARAMS.SCOPE) === BLUEPRINT_SCOPE.SYSTEM
    ? BLUEPRINT_SCOPE.SYSTEM
    : BLUEPRINT_SCOPE.CUSTOM;

/** The catalog page with the given catalog open. */
export const catalogPath = (scope: BlueprintScope): string =>
  `${ROUTES.BLUEPRINTS}?${new URLSearchParams({ [BLUEPRINT_QUERY_PARAMS.SCOPE]: scope })}`;

/** The Add Model page for a catalog, optionally for one of its manufacturers. */
export const addModelPath = (
  scope: BlueprintScope,
  manufacturerId?: string,
): string => {
  const query = new URLSearchParams({ [BLUEPRINT_QUERY_PARAMS.SCOPE]: scope });
  if (manufacturerId) {
    query.set(BLUEPRINT_QUERY_PARAMS.MANUFACTURER, manufacturerId);
  }
  return `${ROUTES.BLUEPRINT_MODEL_NEW}?${query}`;
};
