/**
 * Alert shapes: warning-and-above system events the tenant has not resolved.
 */

export interface AlertDTO {
  id: string;
  severity: string;
  category: string;
  title: string;
  description: string;
  sourceName: string;
  timestamp?: Date;
  status: string;
}

export interface AlertUI {
  id: string;
  severity: string;
  category: string;
  title: string;
  description: string;
  scope?: string;
  timestamp?: string;
  status: string;
}

export interface AlertSummaryDTO {
  critical: number;
  error: number;
  warning: number;
  recent: AlertDTO[];
}

export interface AlertSummaryUI {
  critical: number;
  error: number;
  warning: number;
  recent: AlertUI[];
}

export interface AlertFilter {
  severity?: string;
}
