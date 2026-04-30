/**
 * Hooks Barrel Export
 *
 * Domain-specific React Query hooks.
 * Import via '@hooks' alias.
 *
 * @example
 * import { useBaseStations, useEndpoints } from '@hooks';
 */

// Base Station hooks
export {
  useBaseStation,
  useBaseStationActivity,
  useBaseStations,
  useCommissionBaseStation,
  useCommissionBaseStationWithCerts,
  useDeleteBaseStation,
  useRetryCertificateGeneration,
  useUpdateBaseStation,
  useUpdateBaseStationEui,
} from './useBaseStations';

// Endpoint hooks
export {
  useAttachEndpoint,
  useCreateEndpoint,
  useDeleteEndpoint,
  useDetachEndpoint,
  useDownlinkQueue,
  useDownlinkResults,
  useEndpoint,
  useEndpointActivity,
  useEndpoints,
  useFlushDownlinkQueue,
  useRevokeDownlink,
  useSendDownlink,
  useUpdateEndpoint,
} from './useEndpoints';

// Dashboard hooks
export { useDashboardAnalytics, useDashboardEvents, useDashboardStats } from './useDashboard';

// Filter hooks
export { useBaseStationFilters, useEndpointFilters, useSavedViews } from './useFilters';

// Realtime hooks
export { useRealtimeConnection, useRealtimeInvalidation, useRealtimeUpdates } from './useRealtime';

// Connection status hooks
export { type UIConnectionStatus, useConnectionStatus } from './useConnectionStatus';

// System status hooks
export { useSystemStatus, type UseSystemStatusOptions } from './useSystemStatus';

// User admin hooks
export {
  useChangePassword,
  useCreateUser,
  useDeleteUser,
  useUpdateUser,
  useUser,
  useUserOrganizations,
  useUsers,
  useUsersForLookup,
} from './useUsers';

// API Key hooks
export { useApiKeys, useCreateApiKey, useDeleteApiKey } from './useApiKeys';

// Auth settings hook
export { useAuthSettings } from './useAuthSettings';

// Certificate hooks
export {
  useGenerateServerCertificates,
  useRenewServerCertificates,
  useServerCertificateStatus,
} from './useCertificates';
