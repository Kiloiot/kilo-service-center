/**
 * Every base station on the server that has coordinates.
 */

import { useQuery } from "@tanstack/react-query";

import { baseStationsApi } from "@services/api";
import { queryKeys } from "@config/query-keys";

/** The key sits under the base station family, so station realtime events refresh it. */
export function useBaseStationLocations() {
  return useQuery({
    queryKey: queryKeys.baseStations.locations(),
    queryFn: () => baseStationsApi.listLocations(),
  });
}
