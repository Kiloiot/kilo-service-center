/**
 * Event Mappers
 *
 * Transforms system event data from API to UI format.
 * Handles BSSCI, SCACI, and system events.
 */

import type { CertificateAPI, CertificateUI, EventAPI, EventUI } from '@api-types/api';

import { formatRelativeDuration } from '@utils/formatters';

/**
 * Transform a system event from API to UI format
 */
export function mapEvent(api: EventAPI): EventUI {
  const timestamp = api.timestamp || api.createdAt;
  return {
    id: api.id,
    type: api.severity, // Use severity field from backend
    severity: api.severity,
    message: api.message || api.description || api.title,
    time: formatRelativeDuration(timestamp),
    timestamp, // Keep ISO timestamp for date grouping
    title: api.title, // Keep title for display
    category: api.category,
    eventType: api.event_type,
    sourceName: api.source_name,
    metadata: api.metadata,
  };
}

/**
 * Transform an array of events
 */
export function mapEventList(events: EventAPI[]): EventUI[] {
  return events.map(mapEvent);
}

/**
 * Calculate certificate status based on expiry date
 */
export function deriveCertificateStatus(expiryDate: string): {
  status: 'valid' | 'expiring' | 'expired';
  daysUntilExpiry: number;
} {
  const expiry = new Date(expiryDate);
  const now = new Date();
  const daysUntilExpiry = Math.floor((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

  let status: 'valid' | 'expiring' | 'expired';
  if (daysUntilExpiry < 0) {
    status = 'expired';
  } else if (daysUntilExpiry < 30) {
    status = 'expiring';
  } else {
    status = 'valid';
  }

  return { status, daysUntilExpiry };
}

/**
 * Transform a certificate from API to UI format
 */
export function mapCertificate(api: CertificateAPI): CertificateUI {
  const { status, daysUntilExpiry } = deriveCertificateStatus(api.expiryDate);

  return {
    name: api.name,
    status,
    expiryDate: api.expiryDate,
    daysUntilExpiry,
    issuer: api.issuer,
    subject: api.subject,
  };
}

/**
 * Transform an array of certificates
 */
export function mapCertificateList(certificates: CertificateAPI[]): CertificateUI[] {
  return certificates.map(mapCertificate);
}
