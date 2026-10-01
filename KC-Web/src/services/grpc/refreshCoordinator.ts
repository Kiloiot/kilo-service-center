/**
 * Access-token refresh that every tab of the app performs one at a time.
 * KC-Identity rotates refresh tokens and revokes the session when a spent one
 * is presented again, so two tabs must never send the same refresh token.
 * A refresh ends renewed, refused by the server, or unavailable; only a
 * refusal means the session is over.
 */

import { tokenExpiryMs } from "@utils/jwt";
import { logger } from "@utils/logger";
import {
  LOCK_NAMES,
  STORAGE_KEYS,
  TIMING_ACCESS_TOKEN_EXPIRY_SKEW_MS,
  TOKEN_REFRESH_OUTCOME,
} from "@constants/app";
import { SESSION_ERRORS } from "@constants/messages";

import { GrpcApiError } from "./errors";

/** How one refresh ended; see TOKEN_REFRESH_OUTCOME. */
export type RefreshOutcome =
  (typeof TOKEN_REFRESH_OUTCOME)[keyof typeof TOKEN_REFRESH_OUTCOME];

/** Tokens issued by one refresh-token rotation. */
export interface RotatedTokens {
  accessToken: string;
  refreshToken: string;
}

/** Spends a refresh token and returns the tokens that replace it. */
export type RotateRefreshToken = (
  refreshToken: string,
) => Promise<RotatedTokens>;

/** Token storage shared by every tab. */
export interface TokenStore {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

/** Runs a task while holding a named lock that excludes every other tab. */
export interface ExclusiveLock {
  run<T>(name: string, task: () => Promise<T>): Promise<T>;
}

/** The current time in epoch milliseconds. */
export type Clock = () => number;

export class TokenRefreshCoordinator {
  private readonly rotate: RotateRefreshToken;
  private readonly store: TokenStore;
  private readonly lock: ExclusiveLock;
  private readonly now: Clock;
  private inFlight: Promise<RefreshOutcome> | null = null;

  constructor(
    rotate: RotateRefreshToken,
    store: TokenStore,
    lock: ExclusiveLock,
    now: Clock = Date.now,
  ) {
    this.rotate = rotate;
    this.store = store;
    this.lock = lock;
    this.now = now;
  }

  /**
   * True when the stored access token is expired or expires within the skew,
   * so it must be renewed before it is sent. A token without an exp claim is
   * the server's to judge.
   */
  isAccessTokenExpiring(): boolean {
    const token = this.store.getItem(STORAGE_KEYS.AUTH_TOKEN);
    const expiry = token ? tokenExpiryMs(token) : null;
    return (
      expiry !== null &&
      expiry - this.now() <= TIMING_ACCESS_TOKEN_EXPIRY_SKEW_MS
    );
  }

  /**
   * Renews the stored tokens. A failure leaves them stored; the caller ends
   * the session only on a refusal.
   */
  refresh(): Promise<RefreshOutcome> {
    if (this.inFlight) return this.inFlight;

    const refreshToken = this.store.getItem(STORAGE_KEYS.REFRESH_TOKEN);
    if (!refreshToken) return Promise.resolve(this.refusedWithoutToken());

    this.inFlight = this.lock
      .run(LOCK_NAMES.TOKEN_REFRESH, () =>
        this.refreshHoldingLock(refreshToken),
      )
      .catch((error: unknown) => failed(refreshFailureOutcome(error), error))
      .finally(() => {
        this.inFlight = null;
      });
    return this.inFlight;
  }

  private async refreshHoldingLock(
    refreshToken: string,
  ): Promise<RefreshOutcome> {
    const stored = this.store.getItem(STORAGE_KEYS.REFRESH_TOKEN);
    if (!stored) return this.refusedWithoutToken();
    // Another tab rotated the token while this one waited for the lock.
    if (stored !== refreshToken) return TOKEN_REFRESH_OUTCOME.RENEWED;

    const tokens = await this.rotate(stored);
    this.store.setItem(STORAGE_KEYS.AUTH_TOKEN, tokens.accessToken);
    if (tokens.refreshToken) {
      this.store.setItem(STORAGE_KEYS.REFRESH_TOKEN, tokens.refreshToken);
    }
    return TOKEN_REFRESH_OUTCOME.RENEWED;
  }

  private refusedWithoutToken(): RefreshOutcome {
    return failed(
      TOKEN_REFRESH_OUTCOME.REFUSED,
      SESSION_ERRORS.REFRESH_TOKEN_MISSING,
    );
  }
}

/**
 * Only the server rejecting the refresh token is a refusal; a network
 * failure, an abort, a lock failure or any other status leaves the refresh
 * token valid for a later attempt.
 */
function refreshFailureOutcome(error: unknown): RefreshOutcome {
  return error instanceof GrpcApiError && error.isUnauthorized()
    ? TOKEN_REFRESH_OUTCOME.REFUSED
    : TOKEN_REFRESH_OUTCOME.UNAVAILABLE;
}

function failed(outcome: RefreshOutcome, cause: unknown): RefreshOutcome {
  logger.warn(SESSION_ERRORS.TOKEN_REFRESH_FAILED, outcome, cause);
  return outcome;
}
