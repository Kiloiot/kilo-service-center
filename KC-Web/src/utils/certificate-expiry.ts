/**
 * Certificate expiry classification shared by the certificates page, the
 * base station list and the base station certificate card.
 */

import type { BaseStationUI } from "@api-types/api";

import { formatDate } from "@utils/date-format";
import { calculateDaysUntilExpiry } from "@utils/formatters";
import {
  CERT_EXPIRY_THRESHOLD_DAYS,
  CERTIFICATE_EXPIRY_STATE,
  type CertificateExpiryState,
} from "@constants/app";
import { DATA_TABLE } from "@constants/messages";

export interface CertificateExpiry {
  expiresAt?: string;
  daysUntilExpiry: number | null;
  state: CertificateExpiryState;
}

/** A base station and the client certificate issued to it. */
export interface BaseStationCertificate extends CertificateExpiry {
  eui: string;
  name?: string;
  fingerprint?: string;
  lastHandshake?: string;
}

export function certificateExpiryState(
  daysUntilExpiry: number | null,
): CertificateExpiryState {
  if (daysUntilExpiry === null) return CERTIFICATE_EXPIRY_STATE.NOT_ISSUED;
  if (daysUntilExpiry < 0) return CERTIFICATE_EXPIRY_STATE.EXPIRED;
  if (daysUntilExpiry < CERT_EXPIRY_THRESHOLD_DAYS.CRITICAL) {
    return CERTIFICATE_EXPIRY_STATE.CRITICAL;
  }
  if (daysUntilExpiry < CERT_EXPIRY_THRESHOLD_DAYS.WARNING) {
    return CERTIFICATE_EXPIRY_STATE.EXPIRING;
  }
  return CERTIFICATE_EXPIRY_STATE.VALID;
}

export function certificateExpiry(expiresAt?: string): CertificateExpiry {
  const daysUntilExpiry = calculateDaysUntilExpiry(expiresAt);
  return {
    expiresAt,
    daysUntilExpiry,
    state: certificateExpiryState(daysUntilExpiry),
  };
}

/**
 * A base station's certificate: not issued with neither a pinned fingerprint
 * nor a stored expiry, pinned with its expiry not recorded when only the
 * fingerprint is stored, and otherwise classified by its expiry.
 */
export function stationCertificateExpiry(
  fingerprint?: string,
  expiresAt?: string,
): CertificateExpiry {
  const expiry = certificateExpiry(expiresAt);
  if (expiry.state === CERTIFICATE_EXPIRY_STATE.NOT_ISSUED && fingerprint) {
    return { ...expiry, state: CERTIFICATE_EXPIRY_STATE.EXPIRY_NOT_RECORDED };
  }
  return expiry;
}

/** A certificate's expiry date, or the empty placeholder while none is known. */
export function formatCertificateExpiresAt(expiry: CertificateExpiry): string {
  return expiry.expiresAt ? formatDate(expiry.expiresAt) : DATA_TABLE.NO_VALUE;
}

export function toBaseStationCertificate(
  station: BaseStationUI,
): BaseStationCertificate {
  return {
    eui: station.eui,
    name: station.name,
    fingerprint: station.certificateFingerprint,
    lastHandshake: station.lastHandshake,
    ...stationCertificateExpiry(
      station.certificateFingerprint,
      station.certificateExpiryDate,
    ),
  };
}
