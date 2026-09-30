/**
 * Query Keys Factory
 *
 * Hierarchical query key definitions for React Query cache management.
 * Keys are used for cache invalidation by services/realtime/invalidation.ts.
 *
 * @example
 * useQuery({ queryKey: queryKeys.baseStations.list() })
 * queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all })
 */

import type { AlertFilter } from "@api-types/alerts";
import type {
  ActivityFilter,
  BlueprintScope,
  DeviceScope,
  DownlinkQueueFilter,
  DownlinkResultFilter,
  ErrorGroupFilter,
  EventLogFilter,
  UplinkFilter,
} from "@api-types/api";
import type { PageRequest } from "@api-types/pagination";

import { normalizeEui } from "@utils/eui";
import type { UplinkSource } from "@constants/app";

/**
 * Query keys factory with hierarchical structure
 *
 * Pattern: entity.scope.details
 * - entity.all: invalidates all queries for that entity
 * - entity.list(filters): list with optional filters
 * - entity.detail(id): single item by ID
 */
export const queryKeys = {
  // Device keys carry the canonical EUI (normalizeEui), so a page and a
  // realtime event name the same query whatever form each holds the EUI in.
  baseStations: {
    all: ["baseStations"] as const,
    lists: () => [...queryKeys.baseStations.all, "list"] as const,
    list: () => [...queryKeys.baseStations.lists()] as const,
    locations: () => [...queryKeys.baseStations.all, "locations"] as const,
    detail: (eui: string) =>
      [...queryKeys.baseStations.all, "detail", normalizeEui(eui)] as const,
    activityPrefix: (eui: string) =>
      [...queryKeys.baseStations.all, "activity", normalizeEui(eui)] as const,
    activity: (
      eui: string,
      filter?: ActivityFilter,
      pageToken?: string,
      pageSize?: number,
    ) =>
      [
        ...queryKeys.baseStations.activityPrefix(eui),
        filter,
        pageToken,
        pageSize,
      ] as const,
    availability: (eui: string, windowHours: number) =>
      [
        ...queryKeys.baseStations.all,
        "availability",
        normalizeEui(eui),
        windowHours,
      ] as const,
  },

  endpoints: {
    all: ["endpoints"] as const,
    lists: () => [...queryKeys.endpoints.all, "list"] as const,
    list: () => [...queryKeys.endpoints.lists()] as const,
    detail: (eui: string) =>
      [...queryKeys.endpoints.all, "detail", normalizeEui(eui)] as const,
    activityPrefix: (eui: string) =>
      [...queryKeys.endpoints.all, "activity", normalizeEui(eui)] as const,
    activity: (
      eui: string,
      filter?: ActivityFilter,
      pageToken?: string,
      pageSize?: number,
    ) =>
      [
        ...queryKeys.endpoints.activityPrefix(eui),
        filter,
        pageToken,
        pageSize,
      ] as const,
  },

  // Traffic: uplink-derived data under uplinks, queue and results under downlinks,
  // so one realtime event refreshes every scoped copy of a listing.
  traffic: {
    all: ["traffic"] as const,
    uplinks: ["traffic", "uplinks"] as const,
    uplinkList: (
      source: UplinkSource,
      filter: UplinkFilter,
      page: number,
      pageSize: number,
    ) =>
      [
        ...queryKeys.traffic.uplinks,
        "list",
        source,
        filter,
        page,
        pageSize,
      ] as const,
    uplinkSummary: (scope: DeviceScope) =>
      [...queryKeys.traffic.uplinks, "summary", scope] as const,
    downlinks: ["traffic", "downlinks"] as const,
    downlinkQueues: ["traffic", "downlinks", "queue"] as const,
    downlinkQueue: (
      filter: DownlinkQueueFilter,
      page: number,
      pageSize: number,
    ) => [...queryKeys.traffic.downlinkQueues, filter, page, pageSize] as const,
    downlinkResults: (
      filter: DownlinkResultFilter,
      page: number,
      pageSize: number,
    ) =>
      [
        ...queryKeys.traffic.downlinks,
        "results",
        filter,
        page,
        pageSize,
      ] as const,
  },

  dashboard: {
    all: ["dashboard"] as const,
    stats: () => [...queryKeys.dashboard.all, "stats"] as const,
    analytics: () => [...queryKeys.dashboard.all, "analytics"] as const,
    alertSummary: () => [...queryKeys.dashboard.all, "alertSummary"] as const,
    alerts: (page?: PageRequest, filter?: AlertFilter) =>
      [...queryKeys.dashboard.all, "alerts", page, filter] as const,
  },

  events: {
    all: ["events"] as const,
    log: (filter: EventLogFilter, page: number, pageSize: number) =>
      [...queryKeys.events.all, "log", filter, page, pageSize] as const,
    errorGroups: (filter: ErrorGroupFilter, page: number, pageSize: number) =>
      [...queryKeys.events.all, "errorGroups", filter, page, pageSize] as const,
  },

  certificates: {
    all: ["certificates"] as const,
    status: () => [...queryKeys.certificates.all, "status"] as const,
  },

  scaci: {
    all: ["scaci"] as const,
    status: () => [...queryKeys.scaci.all, "status"] as const,
    sessions: (page?: PageRequest) =>
      [...queryKeys.scaci.all, "sessions", page] as const,
  },

  system: {
    all: ["system"] as const,
    version: () => [...queryKeys.system.all, "version"] as const,
    status: () => [...queryKeys.system.all, "status"] as const,
    capabilities: () => [...queryKeys.system.all, "capabilities"] as const,
  },

  // Auth query keys
  auth: {
    all: ["auth"] as const,
    settings: () => [...queryKeys.auth.all, "settings"] as const,
    rolesAll: () => [...queryKeys.auth.all, "roles"] as const,
    roles: (organizationId: string | null) =>
      [...queryKeys.auth.rolesAll(), organizationId] as const,
  },

  // Users
  users: {
    all: ["users"] as const,
    list: (filters?: { limit?: number; offset?: number }) =>
      [...queryKeys.users.all, "list", filters] as const,
    detail: (id: string) => [...queryKeys.users.all, "detail", id] as const,
  },

  // Organizations
  organizations: {
    all: ["organizations"] as const,
    list: (filters?: { limit?: number; offset?: number; tenantId?: number }) =>
      [...queryKeys.organizations.all, "list", filters] as const,
    detail: (id: string) =>
      [...queryKeys.organizations.all, "detail", id] as const,
    usersAll: () => [...queryKeys.organizations.all, "users"] as const,
    users: (orgId: string) =>
      [...queryKeys.organizations.usersAll(), orgId] as const,
    userDetail: (orgId: string, userId: string) =>
      [...queryKeys.organizations.users(orgId), userId] as const,
  },

  // The organizations each user belongs to
  userOrganizations: {
    all: ["userOrganizations"] as const,
    list: (userId: string) =>
      [...queryKeys.userOrganizations.all, userId] as const,
  },

  // API Keys
  apiKeys: {
    all: ["apiKeys"] as const,
    list: (filters?: {
      pageSize?: number;
      pageToken?: string;
      userId?: string;
    }) => [...queryKeys.apiKeys.all, "list", filters] as const,
    detail: (id: string) => [...queryKeys.apiKeys.all, "detail", id] as const,
  },

  // Blueprints
  // Scope appended only when provided, so a scope-less invalidation stays a prefix of both scoped caches.
  blueprints: {
    all: ["blueprints"] as const,
    manufacturers: (scope?: BlueprintScope) =>
      scope
        ? ([...queryKeys.blueprints.all, "manufacturers", scope] as const)
        : ([...queryKeys.blueprints.all, "manufacturers"] as const),
    deviceModels: (manufacturerId: string, scope?: BlueprintScope) =>
      scope
        ? ([
            ...queryKeys.blueprints.all,
            "deviceModels",
            manufacturerId,
            scope,
          ] as const)
        : ([
            ...queryKeys.blueprints.all,
            "deviceModels",
            manufacturerId,
          ] as const),
    list: (deviceModelId: string, scope?: BlueprintScope) =>
      scope
        ? ([...queryKeys.blueprints.all, "list", deviceModelId, scope] as const)
        : ([...queryKeys.blueprints.all, "list", deviceModelId] as const),
    detail: (id: string) =>
      [...queryKeys.blueprints.all, "detail", id] as const,
    deviceModelDetail: (id: string) =>
      [...queryKeys.blueprints.all, "deviceModel", id] as const,
    manufacturerDetail: (id: string) =>
      [...queryKeys.blueprints.all, "manufacturer", id] as const,
    modelSnapshotCounts: () =>
      [...queryKeys.blueprints.all, "modelSnapshotCount"] as const,
    modelSnapshotCount: (deviceModelId: string) =>
      [...queryKeys.blueprints.modelSnapshotCounts(), deviceModelId] as const,
  },
} as const;

// Type for query keys
