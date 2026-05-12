/**
 * Query Keys Factory
 *
 * Hierarchical query key definitions for React Query cache management.
 * Keys are used for cache invalidation via EVENT_INVALIDATION_MAP in realtime hooks.
 *
 * @example
 * useQuery({ queryKey: queryKeys.baseStations.list() })
 * queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.all })
 */

// Filter types for query key generation
export interface BaseStationFilters {
  search?: string;
  status?: string[];
  sort?: { field: string; direction: "asc" | "desc" };
}

export interface EndpointFilters {
  search?: string;
  attachState?: string[];
  bidirectional?: boolean;
  sort?: { field: string; direction: "asc" | "desc" };
}

/**
 * Query keys factory with hierarchical structure
 *
 * Pattern: entity.scope.details
 * - entity.all: invalidates all queries for that entity
 * - entity.list(filters): list with optional filters
 * - entity.detail(id): single item by ID
 */
export const queryKeys = {
  baseStations: {
    all: ["baseStations"] as const,
    list: (filters?: BaseStationFilters) =>
      [...queryKeys.baseStations.all, "list", filters] as const,
    detail: (eui: string) =>
      [...queryKeys.baseStations.all, "detail", eui] as const,
    messages: (
      eui: string,
      filter?: {
        direction?: "uplink" | "downlink";
        epEui?: string;
        startTime?: string;
        endTime?: string;
      },
      page?: number,
      pageSize?: number,
    ) =>
      [
        ...queryKeys.baseStations.all,
        "messages",
        eui,
        filter,
        page,
        pageSize,
      ] as const,
    // Unified activity feed
    activity: (
      eui: string,
      filter?: {
        startTime?: string;
        endTime?: string;
      },
      pageToken?: string,
      pageSize?: number,
    ) =>
      [
        ...queryKeys.baseStations.all,
        "activity",
        eui,
        filter,
        pageToken,
        pageSize,
      ] as const,
  },

  endpoints: {
    all: ["endpoints"] as const,
    list: (filters?: EndpointFilters) =>
      [...queryKeys.endpoints.all, "list", filters] as const,
    detail: (eui: string) =>
      [...queryKeys.endpoints.all, "detail", eui] as const,
    activity: (eui: string, pageToken?: string, pageSize?: number) =>
      [
        ...queryKeys.endpoints.all,
        "activity",
        eui,
        pageToken,
        pageSize,
      ] as const,
    downlinkQueue: (eui: string, pageSize?: number) =>
      [...queryKeys.endpoints.all, "downlinkQueue", eui, pageSize] as const,
    downlinkResults: (eui: string, statusFilter?: string, pageSize?: number) =>
      [
        ...queryKeys.endpoints.all,
        "downlinkResults",
        eui,
        statusFilter,
        pageSize,
      ] as const,
  },

  dashboard: {
    all: ["dashboard"] as const,
    stats: () => [...queryKeys.dashboard.all, "stats"] as const,
    analytics: () => [...queryKeys.dashboard.all, "analytics"] as const,
    alerts: () => [...queryKeys.dashboard.all, "alerts"] as const,
  },

  events: {
    all: ["events"] as const,
    list: (
      categories?: readonly string[],
      pageToken?: string,
      eventTypes?: readonly string[],
    ) =>
      [
        ...queryKeys.events.all,
        "list",
        categories,
        pageToken,
        eventTypes,
      ] as const,
  },

  certificates: {
    all: ["certificates"] as const,
    status: () => [...queryKeys.certificates.all, "status"] as const,
  },

  scaci: {
    all: ["scaci"] as const,
    sessions: () => [...queryKeys.scaci.all, "sessions"] as const,
    errors: () => [...queryKeys.scaci.all, "errors"] as const,
    statistics: () => [...queryKeys.scaci.all, "statistics"] as const,
    queues: () => [...queryKeys.scaci.all, "queues"] as const,
  },

  system: {
    all: ["system"] as const,
    version: () => [...queryKeys.system.all, "version"] as const,
    status: () => [...queryKeys.system.all, "status"] as const,
  },

  // Auth query keys
  auth: {
    all: ["auth"] as const,
    settings: () => [...queryKeys.auth.all, "settings"] as const,
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
    users: (orgId: string) =>
      [...queryKeys.organizations.all, "users", orgId] as const,
    userDetail: (orgId: string, userId: string) =>
      [...queryKeys.organizations.all, "users", orgId, userId] as const,
  },

  // User Organizations
  userOrganizations: (userId: string) => ["userOrganizations", userId] as const,

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
  blueprints: {
    all: ["blueprints"] as const,
    manufacturers: () =>
      [...queryKeys.blueprints.all, "manufacturers"] as const,
    deviceModels: (manufacturerId: string) =>
      [...queryKeys.blueprints.all, "deviceModels", manufacturerId] as const,
    list: (deviceModelId: string) =>
      [...queryKeys.blueprints.all, "list", deviceModelId] as const,
    detail: (id: string) =>
      [...queryKeys.blueprints.all, "detail", id] as const,
    deviceModelDetail: (id: string) =>
      [...queryKeys.blueprints.all, "deviceModel", id] as const,
  },
} as const;

// Type for query keys
export type QueryKeys = typeof queryKeys;
