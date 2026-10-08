/**
 * Base Station Hooks
 *
 * React Query hooks for base station data fetching and mutations.
 */

import type { ActivityFilter, BaseStationUI } from "@api-types/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { baseStationsApi, certificatesApi } from "@services/api";
import type { CommissionResultStatus } from "@constants/app";
import {
  CERT_VALIDITY_DAYS,
  COMMISSION_RESULT_STATUS,
  TIMING_BASE_STATION_STATUS_REFRESH_MS,
} from "@constants/app";
import { queryKeys } from "@config/query-keys";

/** Fetch all base stations; search and sort narrow the loaded list client-side. */
export function useBaseStations() {
  return useQuery({
    queryKey: queryKeys.baseStations.list(),
    queryFn: () => baseStationsApi.getBaseStations(),
  });
}

/**
 * Fetch a single base station by EUI. It is re-read at the status request
 * interval because the periodic statusRsp updates its metrics without an event.
 */
export function useBaseStation(eui: string) {
  return useQuery({
    queryKey: queryKeys.baseStations.detail(eui),
    queryFn: () => baseStationsApi.getBaseStationDetails(eui),
    enabled: !!eui,
    refetchInterval: TIMING_BASE_STATION_STATUS_REFRESH_MS,
  });
}

/**
 * Commission result type for partial success handling.
 * Uses BS-first flow: creates base station before generating certificates.
 */
export interface CommissionResult {
  status: CommissionResultStatus;
  bsEui: string;
  certData?: {
    serviceCenterUrl: string;
    downloadUrls: {
      caCert: string;
      clientCert: string;
      privateKey: string;
    };
    expiryDate?: string;
  };
  retryToken?: string; // EUI for cert retry if BS succeeded but certs failed
}

type IssuedCertificate = NonNullable<CommissionResult["certData"]>;

/** Issues a base station's certificate, valid for three years unless told otherwise. */
async function issueCertificate(
  bsEui: string,
  validityDays?: number,
): Promise<{ bsEui: string; certData: IssuedCertificate }> {
  const response = await certificatesApi.generateCertificate({
    bsEui,
    validityDays: validityDays || CERT_VALIDITY_DAYS.THREE_YEARS,
  });
  return {
    bsEui: response.bsEui,
    certData: {
      serviceCenterUrl: response.serviceCenterUrl,
      downloadUrls: {
        caCert: response.downloadUrls.caCert,
        clientCert: response.downloadUrls.clientCert,
        privateKey: response.downloadUrls.privateKey,
      },
      expiryDate: response.expiresAt,
    },
  };
}

/**
 * Commission a new base station with certificate generation
 *
 * FLOW: Creates BS first, then generates certs (which are persisted server-side).
 *
 * - If BS creation fails → no side effects, user can retry
 * - If cert generation fails after BS succeeds → returns partial success
 *   with retryToken so user can retry cert generation without creating duplicate BS
 */
export function useCommissionBaseStationWithCerts() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      eui: string;
      name: string;
      validityDays?: number;
      latitude?: number;
      longitude?: number;
      altitude?: number;
    }): Promise<CommissionResult> => {
      const euiClean = data.eui.replace(/-/g, "");

      // STEP 1: Create base station FIRST (must exist before cert generation)
      await baseStationsApi.createBaseStation({
        eui: euiClean,
        name: data.name,
        latitude: data.latitude,
        longitude: data.longitude,
        altitude: data.altitude,
      });

      // STEP 2: Generate certificates AFTER BS exists
      // Certs are persisted to BS record server-side
      try {
        const issued = await issueCertificate(euiClean, data.validityDays);
        return { status: COMMISSION_RESULT_STATUS.COMPLETE, ...issued };
      } catch {
        // Partial success - BS created but cert generation failed
        // Return retryToken so user can retry cert generation
        return {
          status: COMMISSION_RESULT_STATUS.PARTIAL,
          bsEui: euiClean,
          retryToken: euiClean,
        };
      }
    },
    onSuccess: () => {
      // Auto-refresh base station list
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all });
    },
  });
}

/**
 * Retry certificate generation for an existing base station; the station and
 * the certificate views re-read its new certificate.
 */
export function useRetryCertificateGeneration() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { bsEui: string; validityDays?: number }) => {
      const issued = await issueCertificate(data.bsEui, data.validityDays);
      return { bsEui: issued.bsEui, ...issued.certData };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.certificates.all });
    },
  });
}

/**
 * Delete a base station
 */
export function useDeleteBaseStation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (eui: string) => baseStationsApi.deleteBaseStation(eui),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all });
    },
  });
}

/**
 * Update a base station
 */
export function useUpdateBaseStation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      eui,
      data,
    }: {
      eui: string;
      data: {
        name?: string;
        latitude?: number | null;
        longitude?: number | null;
        altitude?: number | null;
      };
    }) => baseStationsApi.updateBaseStation(eui, data),
    // Every station view, its detail included, sits under baseStations.all.
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all }),
  });
}

type BaseStationChanges = Parameters<
  typeof baseStationsApi.updateBaseStation
>[1];

/** Moves a base station to a new EUI, then saves its other edits there; a failed save keeps the move. */
export function useUpdateBaseStationEui() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      eui,
      newEui,
      changes,
    }: {
      eui: string;
      newEui: string;
      changes?: BaseStationChanges;
    }): Promise<{ baseStation: BaseStationUI; changesSaved: boolean }> => {
      const moved = await baseStationsApi.updateBaseStationEui(eui, newEui);
      if (!changes) return { baseStation: moved, changesSaved: true };
      try {
        const updated = await baseStationsApi.updateBaseStation(
          newEui,
          changes,
        );
        return { baseStation: updated, changesSaved: true };
      } catch {
        return { baseStation: moved, changesSaved: false };
      }
    },
    // Both EUIs' details sit under baseStations.all.
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all }),
  });
}

/**
 * Fetch unified base station activity (events + messages)
 */
export function useBaseStationActivity(
  eui: string,
  filter: ActivityFilter,
  pageToken: string,
  pageSize: number,
) {
  return useQuery({
    queryKey: queryKeys.baseStations.activity(eui, filter, pageToken, pageSize),
    queryFn: () =>
      baseStationsApi.getBaseStationActivity(eui, filter, pageToken, pageSize),
    enabled: !!eui,
  });
}

export function useGenerateCertificate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (
      request: Parameters<typeof certificatesApi.generateCertificate>[0],
    ) => certificatesApi.generateCertificate(request),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all }),
  });
}

export function useDownloadCertificate() {
  return useMutation({
    mutationFn: ({
      certId,
      certType,
    }: {
      certId: string;
      certType: Parameters<typeof certificatesApi.downloadCertificate>[1];
    }) => certificatesApi.downloadCertificate(certId, certType),
  });
}

export function useExportBaseStationMessages() {
  return useMutation({
    mutationFn: ({
      bsEui,
      filter,
      format,
    }: {
      bsEui: string;
      filter: Parameters<typeof baseStationsApi.exportBaseStationMessages>[1];
      format: Parameters<typeof baseStationsApi.exportBaseStationMessages>[2];
    }) => baseStationsApi.exportBaseStationMessages(bsEui, filter, format),
  });
}
