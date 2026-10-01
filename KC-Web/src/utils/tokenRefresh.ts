/**
 * Proactive Token Refresh Scheduler
 *
 * Schedules access token refresh before expiry so API calls never hit 401.
 * Uses the JWT `exp` claim to compute when to refresh (60s before expiry).
 * A refresh the service cannot answer is retried with backoff and keeps the
 * session; only a refresh the server refuses clears it and goes to sign-in.
 */

import { sessionApi } from "@services/api";
import { exponentialBackoffMs } from "@utils/backoff";
import { tokenExpiryMs } from "@utils/jwt";
import { logger } from "@utils/logger";
import { redirectToSignIn } from "@utils/signInRedirect";
import { storageService } from "@utils/storage";
import {
  STORAGE_KEYS,
  TIMING_TOKEN_REFRESH_BUFFER_MS,
  TIMING_TOKEN_REFRESH_MIN_DELAY_MS,
  TIMING_TOKEN_REFRESH_RETRY_BASE_MS,
  TIMING_TOKEN_REFRESH_RETRY_MAX_MS,
  TOKEN_REFRESH_OUTCOME,
} from "@constants/app";
import { SESSION_ERRORS } from "@constants/messages";

let refreshTimer: ReturnType<typeof setTimeout> | null = null;
let retryAttempt = 0;

function getTokenExpiryMs(): number | null {
  const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
  return token ? tokenExpiryMs(token) : null;
}

function isRefreshDue(): boolean {
  const expiryMs = getTokenExpiryMs();
  return (
    expiryMs === null || expiryMs - Date.now() <= TIMING_TOKEN_REFRESH_BUFFER_MS
  );
}

function refreshIn(delayMs: number): void {
  clearTimer();
  refreshTimer = setTimeout(() => {
    void doRefresh();
  }, delayMs);
}

function clearTimer(): void {
  if (refreshTimer !== null) {
    clearTimeout(refreshTimer);
    refreshTimer = null;
  }
}

function retryWithBackoff(): void {
  const delayMs = exponentialBackoffMs(
    retryAttempt,
    TIMING_TOKEN_REFRESH_RETRY_BASE_MS,
    TIMING_TOKEN_REFRESH_RETRY_MAX_MS,
  );
  retryAttempt++;
  logger.warn(SESSION_ERRORS.PROACTIVE_REFRESH_RETRY, delayMs);
  refreshIn(delayMs);
}

function endSession(): void {
  logger.error(SESSION_ERRORS.PROACTIVE_REFRESH_REFUSED);
  stopRefresh();
  storageService.removeItem(STORAGE_KEYS.AUTH_TOKEN);
  storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
  storageService.removeItem(STORAGE_KEYS.USER_PROFILE);
  redirectToSignIn();
}

async function doRefresh(): Promise<void> {
  if (!storageService.getItem(STORAGE_KEYS.REFRESH_TOKEN)) return;

  // Another tab may have renewed the shared tokens since this timer was set.
  if (!isRefreshDue()) {
    scheduleRefresh();
    return;
  }

  switch (await sessionApi.refreshTokens()) {
    case TOKEN_REFRESH_OUTCOME.RENEWED:
      scheduleRefresh();
      return;
    case TOKEN_REFRESH_OUTCOME.UNAVAILABLE:
      retryWithBackoff();
      return;
    case TOKEN_REFRESH_OUTCOME.REFUSED:
      endSession();
  }
}

/** Schedule a refresh based on the current access token's exp claim. */
export function scheduleRefresh(): void {
  stopRefresh();

  const expiryMs = getTokenExpiryMs();
  if (!expiryMs) return;

  refreshIn(
    Math.max(
      expiryMs - Date.now() - TIMING_TOKEN_REFRESH_BUFFER_MS,
      TIMING_TOKEN_REFRESH_MIN_DELAY_MS,
    ),
  );
}

/** Cancel any pending scheduled refresh and forget its retries. */
export function stopRefresh(): void {
  clearTimer();
  retryAttempt = 0;
}
