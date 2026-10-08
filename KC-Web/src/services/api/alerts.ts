/**
 * Alerts in the shapes the UI consumes.
 */

import type { AlertFilter, AlertSummaryUI, AlertUI } from "@api-types/alerts";
import type { ListPage, PageRequest } from "@api-types/pagination";
import { mapAlert, mapAlertSummary, mapListPage } from "@mappers";

import * as alertsRpc from "@services/grpc/rpc/alerts";

export const alertsApi = {
  async listAlerts(
    page: PageRequest,
    filter: AlertFilter,
  ): Promise<ListPage<AlertUI>> {
    return mapListPage(await alertsRpc.listAlerts(page, filter), mapAlert);
  },

  async getSummary(): Promise<AlertSummaryUI | null> {
    const summary = await alertsRpc.getAlertSummary();
    return summary ? mapAlertSummary(summary) : null;
  },
};
