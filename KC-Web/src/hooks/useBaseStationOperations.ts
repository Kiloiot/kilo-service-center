/**
 * Base station operation hooks: availability, ping, certificate download and
 * the per-station certificate list.
 */

import type { BaseStationUI } from "@api-types/api";
import { useMutation, useQuery } from "@tanstack/react-query";

import {
  baseStationOperationsApi,
  baseStationsApi,
  certificatesApi,
} from "@services/api";
import { toBaseStationCertificate } from "@utils/certificate-expiry";
import { windowEndingNow } from "@utils/time-window";
import { BS_AVAILABILITY, type PublicCertificateType } from "@constants/app";
import { queryKeys } from "@config/query-keys";

import { useFileDownload } from "./useFileDownload";

export function useBaseStationAvailability(bsEui: string) {
  return useQuery({
    queryKey: queryKeys.baseStations.availability(
      bsEui,
      BS_AVAILABILITY.WINDOW_HOURS,
    ),
    queryFn: () =>
      baseStationOperationsApi.getAvailability(
        bsEui,
        windowEndingNow(BS_AVAILABILITY.WINDOW_HOURS),
        BS_AVAILABILITY.BUCKET_SECONDS,
      ),
    enabled: !!bsEui,
  });
}

/** Pings a base station; the ping sent and answered events refresh the station and its activity. */
export function useInitiatePing() {
  return useMutation({
    mutationFn: (bsEui: string) => baseStationOperationsApi.ping(bsEui),
  });
}

export function useBaseStationCertificateDownload() {
  return useFileDownload(
    ({ bsEui, certType }: { bsEui: string; certType: PublicCertificateType }) =>
      certificatesApi.downloadBaseStationCertificate(bsEui, certType),
  );
}

const selectCertificates = (stations: BaseStationUI[]) =>
  stations.map(toBaseStationCertificate);

/** Every base station with its certificate expiry, sharing the base station list cache. */
export function useBaseStationCertificates() {
  return useQuery({
    queryKey: queryKeys.baseStations.list(),
    queryFn: () => baseStationsApi.getBaseStations(),
    select: selectCertificates,
  });
}
