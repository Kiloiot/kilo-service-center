/**
 * useAuthSettings Hook
 *
 * Fetches auth provider configuration for UI components.
 * Used by UserMenu to determine logout behavior and password change visibility.
 */

import type { AuthSettingsAPI } from '@api-types/api';
import { useQuery } from '@tanstack/react-query';

import { apiService } from '@services/api';
import { TIMING } from '@constants/app';
import { queryKeys } from '@config/query-keys';

/**
 * Fetch auth settings from backend.
 * Settings are relatively static, so use long stale time.
 */
export function useAuthSettings() {
  return useQuery<AuthSettingsAPI>({
    queryKey: queryKeys.auth.settings(),
    queryFn: () => apiService.getAuthSettings(),
    staleTime: TIMING.LIST_REFRESH * 1000, // 30 seconds
  });
}
