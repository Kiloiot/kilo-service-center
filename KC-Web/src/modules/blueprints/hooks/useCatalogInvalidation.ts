import { useCallback } from "react";

import { useQueryClient } from "@tanstack/react-query";

import { queryKeys } from "@config/query-keys";

/**
 * Re-reads every catalog view: manufacturers, models and blueprints show
 * each other's names, counts and defaults, so a change to one is stale in all.
 */
export function useCatalogInvalidation(): () => Promise<void> {
  const queryClient = useQueryClient();
  return useCallback(
    () => queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.all }),
    [queryClient],
  );
}
