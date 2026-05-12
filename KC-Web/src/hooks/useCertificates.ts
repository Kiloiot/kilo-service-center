/**
 * Certificate Hooks
 *
 * React Query hooks for server certificate status and management.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiService } from "@services/api";
import { queryKeys } from "@config/query-keys";

/**
 * Fetch current server certificate status (server cert + CA cert).
 */
export function useServerCertificateStatus() {
  return useQuery({
    queryKey: queryKeys.certificates.status(),
    queryFn: () => apiService.getCertificateStatus(),
  });
}

/**
 * Generate new server certificates. Invalidates certificate status on success.
 */
export function useGenerateServerCertificates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => apiService.generateServerCertificates(),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: queryKeys.certificates.status(),
      }),
  });
}

/**
 * Renew existing server certificates. Invalidates certificate status on success.
 */
export function useRenewServerCertificates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => apiService.renewServerCertificates(),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: queryKeys.certificates.status(),
      }),
  });
}
