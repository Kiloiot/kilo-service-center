/**
 * System RPCs (CoreService): release info, status, the analytics overview
 * and capabilities.
 */

import type { Capability } from "@api-types/system";
import { Empty } from "google-protobuf/google/protobuf/empty_pb";

import * as corePb from "@services/grpc/core_pb";
import { HTTP_STATUS } from "@constants/app";
import { GRPC_CLIENT_ERRORS } from "@constants/messages";

import { optionalInt64String } from "../codec";
import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";

/**
 * Get release info / version
 */
export async function getReleaseInfo(): Promise<{
  version: string;
  buildTime: string;
  gitCommit: string;
  gitBranch: string;
  buildUser: string;
  goVersion: string;
  schemaVersion: number;
  artifacts: Record<string, string>;
  scEui?: string;
  scVendor?: string;
  scModel?: string;
  scName?: string;
  scSwVersion?: string;
  edition?: string;
  editionCode?: string;
  licenseId?: string;
  licenseUrl?: string;
  sourceUrl?: string;
  documentationUrl?: string;
  homepageUrl?: string;
  trademarkNotice?: string;
}> {
  const request = new Empty();

  const response = await grpcTransport.callCore<Empty, corePb.ReleaseInfo>(
    (c) => c.getReleaseInfo,
    request,
    { requireOrgUser: false },
  );

  return {
    version: response.getVersion(),
    buildTime: response.getBuildTime(),
    gitCommit: response.getGitCommit(),
    gitBranch: response.getGitBranch(),
    buildUser: response.getBuildUser(),
    goVersion: response.getGoVersion(),
    schemaVersion: response.getSchemaVersion(),
    artifacts: response
      .getArtifactsMap()
      .toObject()
      .reduce(
        (acc, [k, v]) => {
          acc[k] = v;
          return acc;
        },
        {} as Record<string, string>,
      ),
    scEui: optionalInt64String(response.getScEui()),
    scVendor: response.getScVendor() || undefined,
    scModel: response.getScModel() || undefined,
    scName: response.getScName() || undefined,
    scSwVersion: response.getScSwVersion() || undefined,
    edition: response.getEdition() || undefined,
    editionCode: response.getEditionCode() || undefined,
    licenseId: response.getLicenseId() || undefined,
    licenseUrl: response.getLicenseUrl() || undefined,
    sourceUrl: response.getSourceUrl() || undefined,
    documentationUrl: response.getDocumentationUrl() || undefined,
    homepageUrl: response.getHomepageUrl() || undefined,
    trademarkNotice: response.getTrademarkNotice() || undefined,
  };
}

/**
 * Get system status with service health information
 */
export async function getSystemStatus(): Promise<{
  version: string;
  status: string;
  uptime?: Date;
  activeEndpoints: number;
  activeBasestations: number;
  messagesProcessed: number;
  services: Array<{
    name: string;
    url: string;
    healthy: boolean;
    latencyMs: number;
    error: string;
    checkedAt: Date;
  }>;
}> {
  const request = new Empty();

  const response = await grpcTransport.callCore<Empty, corePb.SystemStatus>(
    (c) => c.getSystemStatus,
    request,
    { requireOrgUser: false },
  );

  // Fallback timestamp only for legacy servers that omit checked_at
  const fallbackTime = response.getUptime()?.toDate() ?? new Date();

  return {
    version: response.getVersion(),
    status: response.getStatus(),
    uptime: response.getUptime()?.toDate(),
    activeEndpoints: response.getActiveEndpoints(),
    activeBasestations: response.getActiveBasestations(),
    messagesProcessed: response.getMessagesProcessed(),
    services: response.getServicesList().map((s) => ({
      name: s.getName(),
      url: s.getUrl(),
      healthy: s.getHealthy(),
      latencyMs: s.getLatencyMs(),
      error: s.getError(),
      // Use proto checked_at if present; fallback only for legacy servers
      checkedAt: s.getCheckedAt()?.toDate() ?? fallbackTime,
    })),
  };
}

/**
 * Get analytics overview
 */
export async function getAnalyticsOverview(): Promise<{
  totalMessages: number;
  activeEndpoints: number;
  activeBaseStations: number;
  avgRssi: number;
  avgSnr: number;
  firstMessage?: Date;
  lastMessage?: Date;
}> {
  const request = new corePb.GetAnalyticsOverviewRequest();

  const response = await grpcTransport.callCore<
    corePb.GetAnalyticsOverviewRequest,
    corePb.GetAnalyticsOverviewResponse
  >((c) => c.getAnalyticsOverview, request);

  const overview = response.getOverview();
  if (!overview) {
    throw new GrpcApiError(
      HTTP_STATUS.INTERNAL_SERVER_ERROR,
      GRPC_CLIENT_ERRORS.INVALID_ANALYTICS_RESPONSE,
    );
  }

  return {
    totalMessages: overview.getTotalMessages(),
    activeEndpoints: overview.getActiveEndpoints(),
    activeBaseStations: overview.getActiveBaseStations(),
    avgRssi: overview.getAvgRssi(),
    avgSnr: overview.getAvgSnr(),
    firstMessage: overview.getFirstMessage()?.toDate(),
    lastMessage: overview.getLastMessage()?.toDate(),
  };
}

/**
 * The non-secret feature toggles of the service center (ListCapabilities).
 */
export async function listCapabilities(): Promise<Capability[]> {
  const response = await grpcTransport.callCore<
    Empty,
    corePb.ListCapabilitiesResponse
  >((c) => c.listCapabilities, new Empty());
  return response
    .getCapabilitiesList()
    .map((c) => ({ name: c.getName(), enabled: c.getEnabled() }));
}
