/**
 * On-demand base station status request.
 */

import { useMutation } from "@tanstack/react-query";

import { baseStationsApi } from "@services/api";

/**
 * Asks a base station for its status. The station answers later; its answer
 * streams in as basestation_status_answered, which refreshes the station.
 */
export function useRequestBaseStationStatus() {
  return useMutation({
    mutationFn: (eui: string) => baseStationsApi.requestStatus(eui),
  });
}
