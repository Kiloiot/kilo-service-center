/**
 * Base Station Hooks
 *
 * React Query hooks for base station data fetching and mutations.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiService } from '@services/api';
import { formatEUIWithDashes } from '@utils/formatters';
import { CERT_VALIDITY_DAYS } from '@constants/app';
import type { BaseStationFilters } from '@config/query-keys';
import { queryKeys } from '@config/query-keys';

/**
 * Fetch all base stations with optional filters
 * Filters are passed to API even if backend ignores them (TypeScript aligned)
 */
export function useBaseStations(filters?: BaseStationFilters) {
  return useQuery({
    queryKey: queryKeys.baseStations.list(filters),
    queryFn: () => apiService.getBaseStations(filters),
  });
}

/**
 * Fetch a single base station by EUI
 */
export function useBaseStation(eui: string) {
  return useQuery({
    queryKey: queryKeys.baseStations.detail(eui),
    queryFn: () => apiService.getBaseStationDetails(eui),
    enabled: !!eui,
  });
}

/**
 * Commission (create) a new base station
 */
export function useCommissionBaseStation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: Parameters<typeof apiService.createBaseStation>[0]) =>
      apiService.createBaseStation(data),
    onSuccess: () => {
      // Invalidate base station list to refetch
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all });
    },
  });
}

/**
 * Commission result type for partial success handling.
 * Uses BS-first flow: creates base station before generating certificates.
 */
export interface CommissionResult {
  status: 'complete' | 'partial';
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
      const euiClean = data.eui.replace(/-/g, '');
      const euiDashed = formatEUIWithDashes(euiClean);

      // STEP 1: Create base station FIRST (must exist before cert generation)
      await apiService.createBaseStation({
        eui: euiClean,
        name: data.name,
        connection_type: 'bssci',
        latitude: data.latitude,
        longitude: data.longitude,
        altitude: data.altitude,
      });

      // STEP 2: Generate certificates AFTER BS exists
      // Certs are persisted to BS record server-side
      try {
        const certResponse = await apiService.generateCertificate({
          bsEui: euiDashed,
          validityDays: data.validityDays || CERT_VALIDITY_DAYS.THREE_YEARS,
        });

        // Complete success - BS created and certs generated
        return {
          status: 'complete',
          bsEui: certResponse.bsEui,
          certData: {
            serviceCenterUrl: certResponse.serviceCenterUrl,
            downloadUrls: {
              caCert: certResponse.downloadUrls.caCert,
              clientCert: certResponse.downloadUrls.clientCert,
              privateKey: certResponse.downloadUrls.privateKey,
            },
            expiryDate: certResponse.expiresAt,
          },
        };
      } catch {
        // Partial success - BS created but cert generation failed
        // Return retryToken so user can retry cert generation
        return {
          status: 'partial',
          bsEui: euiDashed,
          retryToken: euiDashed, // EUI for retry
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
 * Retry certificate generation for an existing base station
 */
export function useRetryCertificateGeneration() {
  return useMutation({
    mutationFn: async (data: { bsEui: string; validityDays?: number }) => {
      const certResponse = await apiService.generateCertificate({
        bsEui: data.bsEui,
        validityDays: data.validityDays || CERT_VALIDITY_DAYS.THREE_YEARS,
      });

      return {
        bsEui: certResponse.bsEui,
        serviceCenterUrl: certResponse.serviceCenterUrl,
        downloadUrls: {
          caCert: certResponse.downloadUrls.caCert,
          clientCert: certResponse.downloadUrls.clientCert,
          privateKey: certResponse.downloadUrls.privateKey,
        },
        expiryDate: certResponse.expiresAt,
      };
    },
  });
}

/**
 * Delete a base station
 */
export function useDeleteBaseStation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (eui: string) => apiService.deleteBaseStation(eui),
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
    }) => apiService.updateBaseStation(eui, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.detail(variables.eui) });
    },
  });
}

/**
 * Update base station EUI with cascade to all dependent tables.
 */
export function useUpdateBaseStationEui() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ eui, newEui }: { eui: string; newEui: string }) =>
      apiService.updateBaseStationEui(eui, newEui),
    onSuccess: (_, variables) => {
      // Invalidate both old and new EUI queries
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.detail(variables.eui) });
      queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.detail(variables.newEui) });
    },
  });
}

/**
 * Fetch unified base station activity (events + messages)
 */
export function useBaseStationActivity(
  eui: string,
  filter?: {
    startTime?: string;
    endTime?: string;
  },
  pageToken?: string,
  pageSize = 50
) {
  return useQuery({
    queryKey: queryKeys.baseStations.activity(eui, filter, pageToken, pageSize),
    queryFn: () => apiService.getBaseStationActivity(eui, filter, pageToken, pageSize),
    enabled: !!eui,
  });
}
