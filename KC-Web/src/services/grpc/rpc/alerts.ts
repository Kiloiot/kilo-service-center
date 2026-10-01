/**
 * Alert RPCs (CoreService): the tenant's unresolved warning-and-above events.
 */

import type { AlertDTO, AlertFilter, AlertSummaryDTO } from "@api-types/alerts";
import type { ListPage, PageRequest } from "@api-types/pagination";

import * as corePb from "@services/grpc/core_pb";

import { offsetPageToken, pageOf } from "../codec";
import { grpcTransport } from "../transport";

function mapAlert(alert: corePb.Alert): AlertDTO {
  return {
    id: alert.getId(),
    severity: alert.getSeverity(),
    category: alert.getCategory(),
    title: alert.getTitle(),
    description: alert.getDescription(),
    sourceName: alert.getSourceName(),
    timestamp: alert.getTimestamp()?.toDate(),
    status: alert.getStatus(),
  };
}

export async function listAlerts(
  page: PageRequest,
  filter: AlertFilter,
): Promise<ListPage<AlertDTO>> {
  const request = new corePb.ListAlertsRequest();
  request.setPageSize(page.pageSize);
  request.setPageToken(offsetPageToken(page));
  if (filter.severity) request.setSeverity(filter.severity);

  const response = await grpcTransport.callCore<
    corePb.ListAlertsRequest,
    corePb.ListAlertsResponse
  >((c) => c.listAlerts, request);
  return pageOf(
    response.getAlertsList().map(mapAlert),
    response.getTotalCount(),
    response.getNextPageToken(),
  );
}

export async function getAlertSummary(): Promise<AlertSummaryDTO | undefined> {
  const response = await grpcTransport.callCore<
    corePb.GetAlertSummaryRequest,
    corePb.GetAlertSummaryResponse
  >((c) => c.getAlertSummary, new corePb.GetAlertSummaryRequest());
  const summary = response.getSummary();
  if (!summary) return undefined;
  return {
    critical: summary.getCritical(),
    error: summary.getError(),
    warning: summary.getWarning(),
    recent: summary.getRecentList().map(mapAlert),
  };
}
