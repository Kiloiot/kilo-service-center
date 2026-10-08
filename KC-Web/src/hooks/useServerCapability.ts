/**
 * Whether one of the service center's feature toggles (ListCapabilities) is
 * on. Toggles change only with a service center restart, so the one query
 * never goes stale.
 */

import { useQuery } from "@tanstack/react-query";

import { systemApi } from "@services/api";
import type { SERVER_CAPABILITY } from "@constants/app";
import { queryKeys } from "@config/query-keys";

type ServerCapability =
  (typeof SERVER_CAPABILITY)[keyof typeof SERVER_CAPABILITY];

/** Every feature toggle, by capability name. */
function useServiceCapabilities() {
  return useQuery({
    queryKey: queryKeys.system.capabilities(),
    queryFn: () => systemApi.getCapabilities(),
    staleTime: Infinity,
  });
}

/** Whether the toggle is on; undefined until the capabilities have loaded. */
export function useServerCapability(
  name: ServerCapability,
): boolean | undefined {
  const { data } = useServiceCapabilities();
  return data ? (data[name] ?? false) : undefined;
}
