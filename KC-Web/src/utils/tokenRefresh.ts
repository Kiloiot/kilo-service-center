/**
 * Proactive Token Refresh Scheduler
 *
 * Schedules access token refresh before expiry so API calls never hit 401.
 * Uses the JWT `exp` claim to compute when to refresh (60s before expiry).
 * Falls back to the reactive 401-retry path in the gRPC client if the
 * proactive refresh fails (e.g., offline).
 */

import { apiService } from '@services/api';
import { decodeJwtPayload } from '@utils/jwt';
import { logger } from '@utils/logger';
import { storageService } from '@utils/storage';
import { STORAGE_KEYS } from '@constants/app';

const REFRESH_BUFFER_MS = 60_000;
const MIN_SCHEDULE_MS = 5_000;

let refreshTimer: ReturnType<typeof setTimeout> | null = null;

function getTokenExpiryMs(): number | null {
  const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
  if (!token) return null;

  const payload = decodeJwtPayload(token);
  if (!payload || typeof payload.exp !== 'number') return null;

  return payload.exp * 1000;
}

async function doRefresh(): Promise<void> {
  const refreshToken = storageService.getItem(STORAGE_KEYS.REFRESH_TOKEN);
  if (!refreshToken) return;

  try {
    const result = await apiService.refreshTokens(refreshToken);
    storageService.setItem(STORAGE_KEYS.AUTH_TOKEN, result.accessToken);
    if (result.refreshToken) {
      storageService.setItem(STORAGE_KEYS.REFRESH_TOKEN, result.refreshToken);
    }
    scheduleRefresh();
  } catch {
    logger.error('Proactive token refresh failed, clearing session');
    storageService.removeItem(STORAGE_KEYS.AUTH_TOKEN);
    storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    storageService.removeItem(STORAGE_KEYS.USER_PROFILE);
    if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
      window.location.href = '/login';
    }
  }
}

/** Schedule a refresh based on the current access token's exp claim. */
export function scheduleRefresh(): void {
  stopRefresh();

  const expiryMs = getTokenExpiryMs();
  if (!expiryMs) return;

  const delayMs = Math.max(expiryMs - Date.now() - REFRESH_BUFFER_MS, MIN_SCHEDULE_MS);
  refreshTimer = setTimeout(() => {
    void doRefresh();
  }, delayMs);
}

/** Cancel any pending scheduled refresh. */
export function stopRefresh(): void {
  if (refreshTimer !== null) {
    clearTimeout(refreshTimer);
    refreshTimer = null;
  }
}
