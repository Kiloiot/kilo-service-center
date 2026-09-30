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
  useCommissionBaseStationWithCerts,
  useDeleteBaseStation,
  useDownloadCertificate,
  useExportBaseStationMessages,
  useGenerateCertificate,
  useRetryCertificateGeneration,
  useUpdateBaseStation,
  useUpdateBaseStationEui,
} from "./useBaseStations";
export { useRequestBaseStationStatus } from "./useBaseStationStatusRequest";

// Endpoint hooks
export {
  useAttachEndpoint,
  useCreateEndpoint,
  useDeleteEndpoint,
  useDetachEndpoint,
  useEndpoint,
  useEndpointActivity,
  useEndpoints,
  useRevealEndpointKey,
  useUpdateEndpoint,
} from "./useEndpoints";

// Traffic hooks
export {
  useBaseStationTrafficSummary,
  useDownlinkQueue,
  useDownlinkResults,
  useEndpointTrafficSummary,
  useFlushDownlinkQueue,
  useRevokeDownlink,
  useSendDownlink,
  useUpdatePendingDownlink,
  useUplinks,
} from "./useTraffic";

// Log hooks
export { useErrorGroups, useEventLog } from "./useLogs";

// DL RX status hooks
export {
  useDlRxStatuses,
  useDlRxStatusQueries,
  useQueryDlRxStatus,
} from "./useDlRxStatus";

// Dashboard hooks
export { useDashboardAnalytics, useDashboardStats } from "./useDashboard";

// Filter hooks
export { useScopedFilters } from "./useFilters";

// Realtime hooks
export { useRealtimeUpdates } from "./useRealtime";

// Connection status hooks
export { useConnectionStatus } from "./useConnectionStatus";

// System status hooks
export { useSystemStatus } from "./useSystemStatus";
export { useVersionInfo } from "./useVersionInfo";

// Authentication hooks
export {
  useChangeOwnPassword,
  useExchangeAuthCode,
  useLoadAuthProfile,
  useLogin,
  useLogout,
  useRegisterAccount,
} from "./useAuth";

// User admin hooks
export {
  useChangePassword,
  useCreateUser,
  useDeleteUser,
  useUpdateUser,
  useUser,
  useUserOrganizations,
  useUsers,
} from "./useUsers";

// API Key hooks

// Auth settings hook
export { useAuthSettings } from "./useAuthSettings";
export { usePasswordRule } from "./usePasswordRule";

// Certificate hooks
export {
  useGenerateServerCertificates,
  useRenewServerCertificates,
  useServerCertificateStatus,
} from "./useCertificates";
