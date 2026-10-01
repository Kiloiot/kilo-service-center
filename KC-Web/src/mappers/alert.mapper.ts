/**
 * Alert Mappers
 */

import type {
  AlertDTO,
  AlertSummaryDTO,
  AlertSummaryUI,
  AlertUI,
} from "@api-types/alerts";

export function mapAlert(dto: AlertDTO): AlertUI {
  return {
    id: dto.id,
    severity: dto.severity,
    category: dto.category,
    title: dto.title,
    description: dto.description,
    scope: dto.sourceName || undefined,
    timestamp: dto.timestamp?.toISOString(),
    status: dto.status,
  };
}

export function mapAlertSummary(dto: AlertSummaryDTO): AlertSummaryUI {
  return {
    critical: dto.critical,
    error: dto.error,
    warning: dto.warning,
    recent: dto.recent.map(mapAlert),
  };
}
