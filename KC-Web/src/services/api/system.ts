/**
 * System version, status, analytics overview and capabilities in the shapes
 * the UI consumes.
 */

import type {
  AnalyticsOverviewAPI,
  SystemStatusResponse,
} from "@api-types/api";

import * as systemRpc from "@services/grpc/rpc/system";
import { SYSTEM_HEALTH_STATUS } from "@constants/app";

export const systemApi = {
  async getVersion(): Promise<{
    version: string;
    buildTime: string;
    gitCommit: string;
    gitBranch: string;
    buildUser: string;
    goVersion: string;
    schemaVersion: number;
    artifacts: Record<string, string>;
    isProduction: boolean;
    edition?: string;
    editionCode?: string;
    licenseId?: string;
    licenseUrl?: string;
    sourceUrl?: string;
    documentationUrl?: string;
    homepageUrl?: string;
    trademarkNotice?: string;
  }> {
    const response = await systemRpc.getReleaseInfo();
    return {
      version: response.version,
      buildTime: response.buildTime,
      gitCommit: response.gitCommit,
      gitBranch: response.gitBranch,
      buildUser: response.buildUser,
      goVersion: response.goVersion,
      schemaVersion: response.schemaVersion,
      artifacts: response.artifacts,
      isProduction: false,
      edition: response.edition,
      editionCode: response.editionCode,
      licenseId: response.licenseId,
      licenseUrl: response.licenseUrl,
      sourceUrl: response.sourceUrl,
      documentationUrl: response.documentationUrl,
      homepageUrl: response.homepageUrl,
      trademarkNotice: response.trademarkNotice,
    };
  },

  async getSystemStatus(): Promise<SystemStatusResponse> {
    const response = await systemRpc.getSystemStatus();

    // Compute healthy from services when present
    const healthy =
      response.services.length > 0
        ? response.services.every((s) => s.healthy)
        : response.status === SYSTEM_HEALTH_STATUS.HEALTHY;

    // Derive timestamp from latest checkedAt (already has fallback from client)
    const timestamp =
      response.services.length > 0
        ? response.services
            .reduce(
              (latest, s) => (s.checkedAt > latest ? s.checkedAt : latest),
              response.services[0].checkedAt,
            )
            .toISOString()
        : response.uptime
          ? response.uptime.toISOString()
          : "";

    return {
      services: response.services.map((s) => ({
        name: s.name,
        url: s.url,
        healthy: s.healthy,
        latencyMs: s.latencyMs,
        error: s.error,
        checkedAt: s.checkedAt.toISOString(),
      })),
      healthy,
      timestamp,
      startedAt: response.uptime?.toISOString(),
    };
  },

  async getAnalyticsOverview(): Promise<AnalyticsOverviewAPI> {
    const response = await systemRpc.getAnalyticsOverview();
    return {
      totalMessages: response.totalMessages,
      activeEndpoints: response.activeEndpoints,
      activeBaseStations: response.activeBaseStations,
      messagesLast24h: 0,
      messagesLastWeek: 0,
    };
  },

  /** The service center's feature toggles, by capability name. */
  async getCapabilities(): Promise<Record<string, boolean>> {
    const capabilities = await systemRpc.listCapabilities();
    return Object.fromEntries(capabilities.map((c) => [c.name, c.enabled]));
  },
};
